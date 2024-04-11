package ctx

import (
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/hashicorp/go-multierror"
	appctx "github.com/nixys/nxs-go-appctx/v3"
	"github.com/sirupsen/logrus"

	"github.com/nixys/nxs-backup/interfaces"
	"github.com/nixys/nxs-backup/modules/cmd_handler"
	"github.com/nixys/nxs-backup/modules/logger"
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

func AppCtxInit() (any, error) {
	var (
		errs     *multierror.Error
		fileJobs interfaces.Jobs
		dbJobs   interfaces.Jobs
		extJobs  interfaces.Jobs
	)

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

	if ra.Cmd == "update" {
		c.Cmd = cmd_handler.InitSelfUpdate(ra.CmdParams.(*UpdateCmd).Version, c.Done)
		return c, nil
	}

	conf, err := confRead(ra.ConfigPath)
	if err != nil {
		printInitError("Failed to read configuration file: %v\n", err)
		return nil, err
	}

	if ra.Cmd == "generate" {
		cp := ra.CmdParams.(*GenerateCmd)
		c.Cmd = cmd_handler.InitGenerateConfig(
			c.Done,
			ra.ConfigPath,
			cp.Type,
			cp.OutPath,
			cp.Storages,
			ra.Arg,
		)
		return c, nil
	}

	c.Log, err = logInit(conf.LogFile, conf.LogLevel)
	if err != nil {
		printInitError("Failed to init log file: %v\n", err)
		return nil, err
	}

	// Init app
	storages, err := storagesInit(conf)
	if err != nil {
		errs = multierror.Append(errs, err.(*multierror.Error).WrappedErrors()...)
	}

	jobs, err := jobsInit(conf, storages)
	if err != nil {
		errs = multierror.Append(errs, err.(*multierror.Error).WrappedErrors()...)
	}

	jobsMap := make(map[string]interfaces.Job)

	for _, job := range jobs {
		switch job.GetType() {
		case "desc_files", "inc_files":
			fileJobs = append(fileJobs, job)
		case "mysql", "mysql_xtrabackup", "postgresql", "postgresql_basebackup", "mongodb", "redis":
			dbJobs = append(dbJobs, job)
		case "external":
			extJobs = append(extJobs, job)
		}
		jobsMap[job.GetName()] = job
	}

	// Notifications init

	c.Notifiers, err = notifiersInit(conf)
	if err != nil {
		errs = multierror.Append(errs, err.(*multierror.Error).WrappedErrors()...)
	}

	if ra.Cmd == "testCfg" {
		c.Cmd = cmd_handler.InitTestConfig(errs.ErrorOrNil(), c.Done, fileJobs, dbJobs, extJobs)
	} else if ra.Cmd == "start" {
		cp := ra.CmdParams.(*StartCmd)
		c.Cmd = cmd_handler.InitStartBackup(
			errs.ErrorOrNil(),
			c.Done,
			c.EventCh,
			conf.WaitingTimeout,
			cp.JobName,
			jobsMap,
			fileJobs,
			dbJobs,
			extJobs,
		)
	}

	return c, nil
}

func printInitError(ft string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, ft, err)
}

func logInit(file, level string) (*logrus.Logger, error) {
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
			return nil, err
		}
		if f, err = os.OpenFile(file, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600); err != nil {
			return nil, err
		}
	}

	// Validate log level
	if l, err = logrus.ParseLevel(level); err != nil {
		return nil, fmt.Errorf("log init: %w", err)
	}

	return appctx.DefaultLogInit(f, l, &logger.LogFormatter{})
}
