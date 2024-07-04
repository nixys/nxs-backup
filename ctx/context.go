package ctx

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/docker/go-units"
	"github.com/hashicorp/go-multierror"
	appctx "github.com/nixys/nxs-go-appctx/v3"
	"github.com/sirupsen/logrus"

	"github.com/nixys/nxs-backup/interfaces"
	"github.com/nixys/nxs-backup/misc"
	"github.com/nixys/nxs-backup/modules/cmd_handler/api_server"
	"github.com/nixys/nxs-backup/modules/cmd_handler/generate_config"
	"github.com/nixys/nxs-backup/modules/cmd_handler/self_update"
	"github.com/nixys/nxs-backup/modules/cmd_handler/start_backup"
	"github.com/nixys/nxs-backup/modules/cmd_handler/test_config"
	"github.com/nixys/nxs-backup/modules/logger"
	"github.com/nixys/nxs-backup/modules/metrics"
)

type rateType string

const (
	disk rateType = "disk"
	net  rateType = "net"
)

// Ctx defines application custom context
type Ctx struct {
	Cmd       interfaces.Handler
	Log       *logrus.Logger
	Done      chan error
	EventCh   chan logger.LogRecord
	EventsWG  *sync.WaitGroup
	Notifiers []interfaces.Notifier
}

type app struct {
	waitTimeout time.Duration
	jobs        map[string]interfaces.Job
	fileJobs    interfaces.Jobs
	dbJobs      interfaces.Jobs
	extJobs     interfaces.Jobs
	initErrs    *multierror.Error
	metricsData *metrics.Data
	serverBind  string
}

func AppCtxInit() (any, error) {
	c := &Ctx{
		EventsWG: &sync.WaitGroup{},
		EventCh:  make(chan logger.LogRecord),
		Done:     make(chan error),
	}

	ra, err := ReadArgs()
	if err != nil {
		return nil, err
	}

	l, _ := appctx.DefaultLogInit(os.Stderr, logrus.InfoLevel, &logrus.TextFormatter{})
	c.Log = l

	switch ra.Cmd {
	case "update":
		c.Cmd = self_update.Init(
			self_update.Opts{
				Version: ra.CmdParams.(*UpdateCmd).Version,
				Done:    c.Done,
			},
		)
	case "generate":
		if _, err = readConfig(ra.ConfigPath); err != nil {
			printInitError("Failed to read configuration file: %v\n", err)
			return nil, err
		}
		cp := ra.CmdParams.(*GenerateCmd)
		c.Cmd = generate_config.Init(
			generate_config.Opts{
				Done:     c.Done,
				CfgPath:  ra.ConfigPath,
				JobType:  cp.Type,
				OutPath:  cp.OutPath,
				Arg:      ra.Arg,
				Storages: cp.Storages,
			},
		)
	case "testCfg":
		a, err := appInit(c, ra.ConfigPath)
		if err != nil {
			return nil, err
		}
		c.Cmd = test_config.Init(
			test_config.Opts{
				InitErr:  a.initErrs.ErrorOrNil(),
				Done:     c.Done,
				FileJobs: a.fileJobs,
				DBJobs:   a.dbJobs,
				ExtJobs:  a.extJobs,
			},
		)
	case "start":
		a, err := appInit(c, ra.ConfigPath)
		if err != nil {
			return nil, err
		}
		c.Cmd = start_backup.Init(
			start_backup.Opts{
				InitErr:     a.initErrs.ErrorOrNil(),
				Done:        c.Done,
				EvCh:        c.EventCh,
				WaitPrev:    a.waitTimeout,
				JobName:     ra.CmdParams.(*StartCmd).JobName,
				Jobs:        a.jobs,
				FileJobs:    a.fileJobs,
				DBJobs:      a.dbJobs,
				ExtJobs:     a.extJobs,
				MetricsData: a.metricsData,
			},
		)
	case "server":
		a, err := appInit(c, ra.ConfigPath)
		if err != nil {
			return nil, err
		}
		if a.metricsData == nil {
			err = fmt.Errorf("server metrics disabled by config")
			printInitError("Init err:\n%s", err)
			return nil, err
		}
		c.Cmd, err = api_server.Init(
			api_server.Opts{
				Bind:           a.serverBind,
				MetricFilePath: a.metricsData.MetricFilePath(),
				Log:            c.Log,
				Done:           c.Done,
			},
		)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func printInitError(ft string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, ft, err)
}

func appInit(c *Ctx, cfgPath string) (app, error) {

	a := app{
		jobs: make(map[string]interfaces.Job),
	}

	conf, err := readConfig(cfgPath)
	if err != nil {
		printInitError("Failed to read configuration file: %v\n", err)
		return a, err
	}

	a.waitTimeout = conf.WaitingTimeout
	a.serverBind = conf.Server.Bind

	a.metricsData = metrics.InitData(
		metrics.DataOpts{
			Project:     conf.ProjectName,
			Server:      conf.ServerName,
			MetricsFile: conf.Server.Metrics.FilePath,
			Enabled:     false,
		},
	)

	if conf.Server.Metrics.Enabled {
		nva := 0.0
		ver, _ := semver.NewVersion(misc.VERSION)
		newVer, _, _ := misc.CheckNewVersionAvailable(strconv.FormatUint(ver.Major(), 10))
		if newVer != "" {
			nva = 1
		}
		a.metricsData.NewVersionAvailable = nva
		a.metricsData.Enabled = true
	}

	if err = logInit(c, conf.LogFile, conf.LogLevel); err != nil {
		printInitError("Failed to init log file: %v\n", err)
		return a, err
	}

	// Notifications init
	if err = notifiersInit(c, conf); err != nil {
		a.initErrs = multierror.Append(a.initErrs, err.(*multierror.Error).WrappedErrors()...)
	}
	// Init app
	storages, err := storagesInit(conf)
	if err != nil {
		a.initErrs = multierror.Append(a.initErrs, err.(*multierror.Error).WrappedErrors()...)
	}

	jobs, err := jobsInit(
		jobsOpts{
			jobs:        conf.Jobs,
			storages:    storages,
			metricsData: a.metricsData,
			mainLim:     conf.Limits,
		},
	)
	if err != nil {
		a.initErrs = multierror.Append(a.initErrs, err.(*multierror.Error).WrappedErrors()...)
	}

	for _, job := range jobs {
		switch job.GetType() {
		case "desc_files", "inc_files":
			a.fileJobs = append(a.fileJobs, job)
		case "mysql", "mysql_xtrabackup", "postgresql", "postgresql_basebackup", "mongodb", "redis":
			a.dbJobs = append(a.dbJobs, job)
		case "external":
			a.extJobs = append(a.extJobs, job)
		}
		a.jobs[job.GetName()] = job
	}

	return a, nil
}

func logInit(c *Ctx, file, level string) error {
	var (
		f   *os.File
		l   logrus.Level
		err error
	)

	switch file {
	case "stdout":
		f = os.Stdout
	case "stderr":
		f = os.Stderr
	default:
		if err = os.MkdirAll(path.Dir(file), os.ModePerm); err != nil {
			return err
		}
		if f, err = os.OpenFile(file, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600); err != nil {
			return err
		}
	}

	// Validate log level
	if l, err = logrus.ParseLevel(level); err != nil {
		return fmt.Errorf("log init: %w", err)
	}

	c.Log, err = appctx.DefaultLogInit(f, l, &logger.LogFormatter{})
	return err
}

func getRateLimit(rate rateType, newLim, baseLim *limitsConf) (rl int64, err error) {
	noLim := "0"
	lim := &limitsConf{
		NetRate:  &noLim,
		DiskRate: &noLim,
	}

	if baseLim != nil {
		if baseLim.DiskRate != nil {
			lim.DiskRate = baseLim.DiskRate
		}
		if baseLim.NetRate != nil {
			lim.NetRate = baseLim.NetRate
		}
	}
	if newLim != nil {
		if newLim.DiskRate != nil {
			lim.DiskRate = newLim.DiskRate
		}
		if newLim.NetRate != nil {
			lim.NetRate = newLim.NetRate
		}
	}

	switch rate {
	case disk:
		rl, err = units.FromHumanSize(*lim.DiskRate)
	case net:
		rl, err = units.FromHumanSize(*lim.NetRate)
	}
	if err != nil {
		return 0, fmt.Errorf("Failed to parse rate limit: %w. ", err)
	}

	return
}
