package ctx

import (
	"fmt"
	"os"
	"path"
	"sync"

	appctx "github.com/nixys/nxs-go-appctx/v2"

	"nxs-backup/interfaces"
	"nxs-backup/modules/backend/notifier"
	"nxs-backup/modules/logger"
)

// Ctx defines application custom context
type Ctx struct {
	ConfigPath   string
	CmdParams    interface{}
	Storages     interfaces.Storages
	Jobs         interfaces.Jobs
	FilesJobs    interfaces.Jobs
	DBsJobs      interfaces.Jobs
	ExternalJobs interfaces.Jobs
	LogCh        chan logger.LogRecord
	Mailer       notifier.Mailer
	Alerter      notifier.AlertServer
	WG           *sync.WaitGroup

	Cfg confOpts
}

// Init initiates application custom context
func (c *Ctx) Init(opts appctx.CustomContextFuncOpts) (appctx.CfgData, error) {

	// Set application context
	arg := opts.Args.(*ArgsParams)
	c.CmdParams = arg.CmdParams
	c.ConfigPath = arg.ConfigPath

	// Read config file
	conf, err := confRead(opts.Config)
	if err != nil {
		fmt.Printf("Failed to read configuration file with next errors:\n%v", err)
		os.Exit(1)
	}
	c.Cfg = conf

	if conf.LogFile != "stdout" && conf.LogFile != "stderr" {
		if err = os.MkdirAll(path.Dir(conf.LogFile), os.ModePerm); err != nil {
			fmt.Printf("Failed to create logfile dir: %v", err)
			os.Exit(1)
		}
	}

	storages, err := storagesInit(conf)
	if err != nil {
		fmt.Printf("Failed init storages with next errors:\n%v", err)
		os.Exit(1)
	}
	for _, s := range storages {
		c.Storages = append(c.Storages, s)
	}

	c.Jobs, err = jobsInit(conf.Jobs, storages)
	if err != nil {
		fmt.Printf("Failed init jobs with next errors:\n%v", err)
		os.Exit(1)
	}
	for _, job := range c.Jobs {
		switch job.GetType() {
		case "desc_files", "inc_files":
			c.FilesJobs = append(c.FilesJobs, job)
		case "mysql", "mysql_xtrabackup", "postgresql", "postgresql_basebackup", "mongodb", "redis":
			c.DBsJobs = append(c.DBsJobs, job)
		case "external":
			c.ExternalJobs = append(c.ExternalJobs, job)
		}
	}

	c.LogCh = make(chan logger.LogRecord)

	c.Mailer, err = mailerInit(conf)
	if err != nil {
		fmt.Printf("Failed init mail notifications with next errors:\n%v", err)
		os.Exit(1)
	}
	c.Alerter, err = alerterInit(conf)
	if err != nil {
		fmt.Printf("Failed init nxs-alert notifications with next errors:\n%v", err)
		os.Exit(1)
	}
	c.WG = new(sync.WaitGroup)

	return appctx.CfgData{
		LogFile:  conf.LogFile,
		LogLevel: conf.LogLevel,
		PidFile:  conf.PidFile,
	}, nil
}

// Reload reloads application custom context
func (c *Ctx) Reload(opts appctx.CustomContextFuncOpts) (appctx.CfgData, error) {

	opts.Log.Debug("reloading context")

	_ = c.Jobs.Close()
	_ = c.Storages.Close()

	return c.Init(opts)
}

// Free frees application custom context
func (c *Ctx) Free(opts appctx.CustomContextFuncOpts) int {

	opts.Log.Debug("freeing context")

	_ = c.Jobs.Close()
	_ = c.Storages.Close()

	return 0
}
