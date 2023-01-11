package local

import (
	"errors"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"nxs-backup/interfaces"
	"nxs-backup/misc"
	"nxs-backup/modules/logger"
	. "nxs-backup/modules/storage"
)

type Local struct {
	backupPath string
	Retention
}

func Init() *Local {
	return &Local{}
}

func (l *Local) IsLocal() int { return 1 }

func (l *Local) SetBackupPath(path string) {
	l.backupPath = path
}

func (l *Local) SetRetention(r Retention) {
	l.Retention = r
}

func (l *Local) DeliveryBackup(logCh chan logger.LogRecord, jobName, tmpBackupFile, ofs, bakType string) (err error) {
	var (
		bakDstPath, mtdDstPath string
		links                  map[string]string
	)

	if bakType == misc.IncBackupType {
		bakDstPath, mtdDstPath, links, err = GetIncBackupDstAndLinks(tmpBackupFile, ofs, l.backupPath)
	} else {
		bakDstPath, links, err = GetDescBackupDstAndLinks(tmpBackupFile, ofs, l.backupPath, l.Retention)
	}
	if err != nil {
		logCh <- logger.Log(jobName, "local").Errorf("Unable to get destination path and links: '%s'", err)
		return
	}

	if mtdDstPath != "" {
		if err = l.deliveryBackupMetadata(logCh, jobName, tmpBackupFile, mtdDstPath); err != nil {
			return
		}
	}

	err = os.MkdirAll(path.Dir(bakDstPath), os.ModePerm)
	if err != nil {
		logCh <- logger.Log(jobName, "local").Errorf("Unable to create directory: '%s'", err)
		return err
	}

	bakDst, err := os.Create(bakDstPath)
	if err != nil {
		return
	}
	defer func() { _ = bakDst.Close() }()

	bakSrc, err := os.Open(tmpBackupFile)
	if err != nil {
		return
	}
	defer func() { _ = bakSrc.Close() }()

	_, err = io.Copy(bakDst, bakSrc)
	if err != nil {
		logCh <- logger.Log(jobName, "local").Errorf("Unable to make copy: %s", err)
		return
	}
	logCh <- logger.Log(jobName, "local").Infof("Successfully copied temp backup to %s", bakDstPath)

	for dst, src := range links {
		err = os.MkdirAll(path.Dir(dst), os.ModePerm)
		if err != nil {
			logCh <- logger.Log(jobName, "local").Errorf("Unable to create directory: '%s'", err)
			return err
		}
		_ = os.Remove(dst)
		if err = os.Symlink(src, dst); err != nil {
			return err
		}
		logCh <- logger.Log(jobName, "local").Infof("Successfully created symlink %s", dst)
	}

	return
}

func (l *Local) deliveryBackupMetadata(logCh chan logger.LogRecord, jobName, tmpBackupFile, mtdDstPath string) error {
	mtdSrcPath := tmpBackupFile + ".inc"

	err := os.MkdirAll(path.Dir(mtdDstPath), os.ModePerm)
	if err != nil {
		logCh <- logger.Log(jobName, "local").Errorf("Unable to create directory: '%s'", err)
		return err
	}

	_ = os.Remove(mtdDstPath)
	mtdDst, err := os.Create(mtdDstPath)
	if err != nil {
		return err
	}
	defer func() { _ = mtdDst.Close() }()

	mtdSrc, err := os.Open(mtdSrcPath)
	if err != nil {
		return err
	}
	defer func() { _ = mtdSrc.Close() }()

	_, err = io.Copy(mtdDst, mtdSrc)
	if err != nil {
		logCh <- logger.Log(jobName, "local").Errorf("Unable to make copy: %s", err)
		return err
	}
	logCh <- logger.Log(jobName, "local").Infof("Successfully copied metadata to %s", mtdDstPath)

	return nil
}

func (l *Local) DeleteOldBackups(logCh chan logger.LogRecord, ofsPartsList []string, jobName, bakType string, full bool) (err error) {

	var errs *multierror.Error

	for _, ofsPart := range ofsPartsList {
		if bakType == misc.IncBackupType {
			err = l.deleteIncBackup(logCh, jobName, ofsPart, full)
		} else {
			err = l.deleteDescBackup(logCh, jobName, ofsPart)
		}
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	return errs.ErrorOrNil()
}

func (l *Local) deleteDescBackup(logCh chan logger.LogRecord, jobName, ofsPart string) error {
	var errs *multierror.Error
	curDate := time.Now()

	for _, period := range []string{"daily", "weekly", "monthly"} {
		bakDir := path.Join(l.backupPath, ofsPart, period)
		files, err := ioutil.ReadDir(bakDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			logCh <- logger.Log(jobName, "local").Errorf("Failed to read files in directory '%s' with next error: %s", bakDir, err)
			return err
		}

		for _, file := range files {
			fileDate := file.ModTime()
			var retentionDate time.Time

			switch period {
			case "daily":
				retentionDate = fileDate.AddDate(0, 0, l.Retention.Days)
			case "weekly":
				retentionDate = fileDate.AddDate(0, 0, l.Retention.Weeks*7)
			case "monthly":
				retentionDate = fileDate.AddDate(0, l.Retention.Months, 0)
			}

			retentionDate = retentionDate.Truncate(24 * time.Hour)
			if curDate.After(retentionDate) {
				err = os.Remove(path.Join(bakDir, file.Name()))
				if err != nil {
					logCh <- logger.Log(jobName, "local").Errorf("Failed to delete file '%s' in directory '%s' with next error: %s",
						file.Name(), bakDir, err)
					errs = multierror.Append(errs, err)
				} else {
					logCh <- logger.Log(jobName, "local").Infof("Deleted old backup file '%s' in directory '%s'", file.Name(), bakDir)
				}
			}
		}
	}

	return errs.ErrorOrNil()
}

func (l *Local) deleteIncBackup(logCh chan logger.LogRecord, jobName, ofsPart string, full bool) error {
	var errs *multierror.Error

	if full {
		backupDir := path.Join(l.backupPath, ofsPart)
		if err := os.RemoveAll(backupDir); err != nil {
			logCh <- logger.Log(jobName, "local").Errorf("Failed to delete '%s' with next error: %s", backupDir, err)
			errs = multierror.Append(errs, err)
		}
	} else {
		intMoy, _ := strconv.Atoi(misc.GetDateTimeNow("moy"))
		lastMonth := intMoy - l.Months

		var year string
		if lastMonth > 0 {
			year = misc.GetDateTimeNow("year")
		} else {
			year = misc.GetDateTimeNow("previous_year")
			lastMonth += 12
		}

		backupDir := path.Join(l.backupPath, ofsPart, year)

		dirs, err := os.ReadDir(backupDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			} else {
				logCh <- logger.Log(jobName, "local").Errorf("Failed to get access to directory '%s' with next error: %v", backupDir, err)
				return err
			}
		}
		rx := regexp.MustCompile(`month_\d\d`)
		for _, dir := range dirs {
			dirName := dir.Name()
			if rx.MatchString(dirName) {
				dirParts := strings.Split(dirName, "_")
				dirMonth, _ := strconv.Atoi(dirParts[1])
				if dirMonth < lastMonth {
					if err = os.RemoveAll(path.Join(backupDir, dirName)); err != nil {
						logCh <- logger.Log(jobName, "local").Errorf("Failed to delete '%s' in dir '%s' with next error: %s",
							dirName, backupDir, err)
						errs = multierror.Append(errs, err)
					} else {
						logCh <- logger.Log(jobName, "local").Infof("Deleted old backup '%s' in directory '%s'", dirName, backupDir)
					}
				}
			}
		}
	}

	return errs.ErrorOrNil()
}

func (l *Local) GetFileReader(ofsPath string) (io.Reader, error) {
	fp, err := filepath.EvalSymlinks(path.Join(l.backupPath, ofsPath))
	if err != nil {
		return nil, err
	}
	return os.Open(fp)
}

func (l *Local) Close() error {
	return nil
}

func (l *Local) Clone() interfaces.Storage {
	cl := *l
	return &cl
}

func (l *Local) GetName() string {
	return "local"
}
