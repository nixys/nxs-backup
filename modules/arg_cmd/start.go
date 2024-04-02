package arg_cmd

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/nightlyone/lockfile"
	appctx "github.com/nixys/nxs-go-appctx/v2"

	"nxs-backup/ctx"
	"nxs-backup/modules/backup"
	"nxs-backup/modules/logger"
)

func Start(appCtx *appctx.AppContext) error {
	var errs *multierror.Error

	cc := appCtx.CustomCtx().(*ctx.Ctx)

	cc.LogCh <- logger.Log("", "").Info("Backup starting.")

	// Crate lockfile
	lock, err := lockfile.New(path.Join(os.TempDir(), "nxs-backup.lck"))
	if cc.Cfg.WaitingTimeout != 0 {
		now := time.Now()
		waitTill := now.Add(time.Minute * cc.Cfg.WaitingTimeout)
		for waitTill.After(time.Now()) {
			if err = lock.TryLock(); err != nil {
				time.Sleep(time.Second * 5)
			} else {
				break
			}
		}
	} else {
		err = lock.TryLock()
	}
	if err != nil {
		err = fmt.Errorf("Can't start nxs-backup. Another nxs-backup process already running. ")
		cc.LogCh <- logger.Log("", "").Error(err)
		return err
	}
	defer func() { _ = lock.Unlock() }()

	jobNameArg := cc.CmdParams.(*ctx.StartCmd).JobName

	if jobNameArg == "external" || jobNameArg == "all" {
		if len(cc.ExternalJobs) > 0 {
			cc.LogCh <- logger.Log("", "").Info("Starting backup external jobs.")
			for _, job := range cc.ExternalJobs {
				if err := backup.Perform(cc.LogCh, job); err != nil {
					errs = multierror.Append(errs, err)
				}
			}
		} else {
			cc.LogCh <- logger.Log("", "").Info("No external jobs.")
		}
	}
	if jobNameArg == "databases" || jobNameArg == "all" {
		if len(cc.DBsJobs) > 0 {
			cc.LogCh <- logger.Log("", "").Info("Starting backup databases jobs.")
			for _, job := range cc.DBsJobs {
				if err := backup.Perform(cc.LogCh, job); err != nil {
					errs = multierror.Append(errs, err)
				}
			}
		} else {
			cc.LogCh <- logger.Log("", "").Info("No databases jobs.")
		}
	}
	if jobNameArg == "files" || jobNameArg == "all" {
		if len(cc.FilesJobs) > 0 {
			cc.LogCh <- logger.Log("", "").Info("Starting backup files jobs.")
			for _, job := range cc.FilesJobs {
				if err := backup.Perform(cc.LogCh, job); err != nil {
					errs = multierror.Append(errs, err)
				}
			}
		} else {
			cc.LogCh <- logger.Log("", "").Info("No files jobs.")
		}
	}

	for _, job := range cc.Jobs {
		if job.GetName() == jobNameArg {
			if err := backup.Perform(cc.LogCh, job); err != nil {
				errs = multierror.Append(errs, err)
			}
		}
	}

	if errs.ErrorOrNil() != nil {
		return fmt.Errorf("Some of backups failed with next errors:\n%v", errs)
	}

	cc.LogCh <- logger.Log("", "").Infof("Backup finished.\n")
	return nil
}
