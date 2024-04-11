package interfaces

import (
	"io"
	"os"
	"path"

	"github.com/hashicorp/go-multierror"

	"github.com/nixys/nxs-backup/misc"
	"github.com/nixys/nxs-backup/modules/logger"
	"github.com/nixys/nxs-backup/modules/storage"
)

type Storage interface {
	IsLocal() int
	SetBackupPath(path string)
	SetRetention(r storage.Retention)
	DeliveryBackup(logCh chan logger.LogRecord, jobName, tmpBackupPath, ofs, bakType string) error
	DeleteOldBackups(logCh chan logger.LogRecord, ofsPart string, job Job, full bool) error
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
	var errs *multierror.Error

	for _, st := range s {
		if ofsPath != "" {
			err := st.DeleteOldBackups(logCh, ofsPath, j, true)
			if err != nil {
				errs = multierror.Append(errs, err)
			}
		} else {
			for _, ofsPart := range j.GetTargetOfsList() {
				err := st.DeleteOldBackups(logCh, ofsPart, j, false)
				if err != nil {
					errs = multierror.Append(errs, err)
				}
			}
		}
	}
	return errs.ErrorOrNil()
}

func (s Storages) Delivery(logCh chan logger.LogRecord, job Job) error {

	errs := new(multierror.Error)

	for ofs, dumpObj := range job.GetDumpObjects() {
		if !dumpObj.Delivered {
			for _, st := range s {
				if err := st.DeliveryBackup(logCh, job.GetName(), dumpObj.TmpFile, ofs, job.GetType()); err != nil {
					errs = multierror.Append(errs, err)
				}
			}
			if errs.Len() < len(s) {
				job.SetDumpObjectDelivered(ofs)
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
