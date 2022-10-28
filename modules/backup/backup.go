package backup

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/go-multierror"

	"nxs-backup/interfaces"
	"nxs-backup/misc"
	"nxs-backup/modules/logger"
)

func Perform(logCh chan logger.LogRecord, job interfaces.Job) error {
	var errs *multierror.Error
	var tmpDirPath string

	if job.GetStoragesCount() == 0 {
		logCh <- logger.Log(job.GetName(), "").Warn("There are no configured storages for job.")
		return nil
	}

	if !job.IsBackupSafety() {
		if err := job.DeleteOldBackups(logCh, ""); err != nil {
			errs = multierror.Append(errs, err)
		}
	} else {
		defer func() {
			err := job.DeleteOldBackups(logCh, "")
			if err != nil {
				errs = multierror.Append(errs, err)
			}
		}()
	}

	if !job.NeedToMakeBackup() {
		logCh <- logger.Log(job.GetName(), "").Infof("According to the backup plan today new backups are not created for job %s", job.GetName())
		return nil
	}

	logCh <- logger.Log(job.GetName(), "").Info("Starting")

	if jobTmpDir := job.GetTempDir(); jobTmpDir != "" {
		tmpDirPath = path.Join(jobTmpDir, fmt.Sprintf("%s_%s", job.GetType(), misc.GetDateTimeNow("")))
		err := os.MkdirAll(tmpDirPath, os.ModePerm)
		if err != nil {
			logCh <- logger.Log(job.GetName(), "").Errorf("Job `%s` failed. Unable to create tmp dir with next error: %s", job.GetName(), err)
			errs = multierror.Append(errs, err)
			return errs.ErrorOrNil()
		}
	}

	if err := job.DoBackup(logCh, tmpDirPath); err != nil {
		errs = multierror.Append(errs, err)
	}

	_ = job.CleanupTmpData()
	_ = filepath.Walk(tmpDirPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// try to delete empty dirs
			if info.IsDir() {
				_ = os.Remove(path)
			}
			return nil
		})
	// cleanup tmp dir
	_ = os.Remove(tmpDirPath)

	logCh <- logger.Log(job.GetName(), "").Info("Finished")

	return errs.ErrorOrNil()
}
