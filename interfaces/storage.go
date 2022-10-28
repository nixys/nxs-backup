package interfaces

import (
	"io"
	"os"
	"path"

	"github.com/hashicorp/go-multierror"

	"nxs-backup/misc"
	"nxs-backup/modules/logger"
	"nxs-backup/modules/storage"
)

type Storage interface {
	IsLocal() int
	SetBackupPath(path string)
	SetRetention(r storage.Retention)
	DeliveryBackup(logCh chan logger.LogRecord, jobName, tmpBackupPath, ofs, bakType string) error
	DeleteOldBackups(logCh chan logger.LogRecord, ofsPartsList []string, jobName, bakType string, full bool) error
	GetFileReader(path string) (io.Reader, error)
	Close() error
	Clone() Storage
	GetName() string
}

type Storages []Storage

func (s Storages) Len() int           { return len(s) }
func (s Storages) Less(i, j int) bool { return s[i].IsLocal() < s[j].IsLocal() }
func (s Storages) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s Storages) DeleteOldBackups(logCh chan logger.LogRecord, j Job, ofsPath string) error {
	var err error
	var errs *multierror.Error

	for _, st := range s {
		if ofsPath != "" {
			err = st.DeleteOldBackups(logCh, []string{ofsPath}, j.GetName(), j.GetType(), true)
		} else {
			err = st.DeleteOldBackups(logCh, j.GetTargetOfsList(), j.GetName(), j.GetType(), false)
		}
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs.ErrorOrNil()
}

func (s Storages) Delivery(logCh chan logger.LogRecord, job Job) error {

	var errs *multierror.Error

	for ofs, dumpObj := range job.GetDumpObjects() {
		if !dumpObj.Delivered {
			var errsDelivery []error
			for _, st := range s {
				if err := st.DeliveryBackup(logCh, job.GetName(), dumpObj.TmpFile, ofs, job.GetType()); err != nil {
					errsDelivery = append(errsDelivery, err)
				}
			}
			if len(errsDelivery) == 0 {
				job.SetDumpObjectDelivered(ofs)
			} else {
				errs = multierror.Append(errs, errsDelivery...)
			}
		}
	}

	return errs.ErrorOrNil()
}

func (s Storages) CleanupTmpData(job Job) error {
	var errs *multierror.Error

	for _, dumpObj := range job.GetDumpObjects() {

		tmpBakFile := dumpObj.TmpFile
		if job.GetType() == misc.IncBackupType {
			// cleanup tmp metadata files
			_ = os.Remove(path.Join(tmpBakFile + ".inc"))
			initFile := path.Join(tmpBakFile + ".init")
			if _, err := os.Stat(initFile); err == nil {
				_ = os.Remove(initFile)
			}
		}

		// cleanup tmp backup file
		if err := os.Remove(tmpBakFile); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs.ErrorOrNil()
}

func (s Storages) Close() error {
	for _, st := range s {
		_ = st.Close()
	}
	return nil
}
