package smb

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hirochachacha/go-smb2"

	"nxs-backup/interfaces"
	"nxs-backup/misc"
	"nxs-backup/modules/logger"
	. "nxs-backup/modules/storage"
)

type SMB struct {
	session    *smb2.Session
	share      *smb2.Share
	backupPath string
	name       string
	Retention
}

type Params struct {
	Host              string
	Port              int
	User              string
	Password          string
	Domain            string
	Share             string
	ConnectionTimeout time.Duration
}

func Init(sName string, params Params) (s *SMB, err error) {
	s = &SMB{
		name: sName,
	}

	conn, err := net.DialTimeout(
		"tcp",
		fmt.Sprintf(
			"%s:%d",
			params.Host,
			params.Port,
		),
		params.ConnectionTimeout*time.Second,
	)
	if err != nil {
		return s, fmt.Errorf("Failed to init '%s' SMB storage. Error: %v ", sName, err)
	}

	s.session, err = (&smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     params.User,
			Password: params.Password,
			Domain:   params.Domain,
		},
	}).Dial(conn)
	if err != nil {
		return s, err
	}

	s.share, err = s.session.Mount(params.Share)
	if err != nil {
		return s, fmt.Errorf("Failed to init '%s' SMB storage. Error: %v ", sName, err)
	}

	return
}

func (s *SMB) IsLocal() int { return 0 }

func (s *SMB) SetBackupPath(path string) {
	s.backupPath = strings.TrimPrefix(path, "/")
}

func (s *SMB) SetRetention(r Retention) {
	s.Retention = r
}

func (s *SMB) DeliveryBackup(logCh chan logger.LogRecord, jobName, tmpBackupFile, ofs, bakType string) (err error) {

	var (
		bakDstPath, mtdDstPath string
		links                  map[string]string
	)

	if bakType == misc.IncBackupType {
		bakDstPath, mtdDstPath, links, err = GetIncBackupDstAndLinks(tmpBackupFile, ofs, s.backupPath)
	} else {
		bakDstPath, links, err = GetDescBackupDstAndLinks(tmpBackupFile, ofs, s.backupPath, s.Retention)
	}
	if err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to get destination path and links: '%s'", err)
		return
	}

	if mtdDstPath != "" {
		if err = s.copy(logCh, jobName, tmpBackupFile+".inc", bakDstPath); err != nil {
			logCh <- logger.Log(jobName, s.name).Errorf("Unable to upload tmp backup")
			return
		}
	}

	if err = s.copy(logCh, jobName, tmpBackupFile, bakDstPath); err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to upload tmp backup")
		return
	}

	for dst, src := range links {
		remDir := path.Dir(dst)
		err = s.share.MkdirAll(remDir, os.ModeDir)
		if err != nil {
			logCh <- logger.Log(jobName, s.name).Errorf("Unable to create remote directory '%s': '%s'", remDir, err)
			return err
		}
		err = s.share.Symlink(src, dst)
		if err != nil {
			logCh <- logger.Log(jobName, s.name).Errorf("Unable to make symlink: %s", err)
			return err
		}
	}

	return nil
}

func (s *SMB) copy(logCh chan logger.LogRecord, jobName, srcPath, dstPath string) (err error) {
	// Make remote directories
	remDir := path.Dir(dstPath)
	if err = s.share.MkdirAll(remDir, os.ModeDir); err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to create remote directory '%s': '%s'", remDir, err)
		return
	}

	dstFile, err := s.share.Create(dstPath)
	if err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to create remote file: %s", err)
		return
	}
	defer func() { _ = dstFile.Close() }()

	srcFile, err := os.Open(srcPath)
	if err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to open '%s'", err)
		return
	}
	defer func() { _ = srcFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to make copy: %s", err)
	} else {
		logCh <- logger.Log(jobName, s.name).Infof("File %s successfully uploaded", dstPath)
	}
	return
}

func (s *SMB) DeleteOldBackups(logCh chan logger.LogRecord, ofsPartsList []string, jobName, bakType string, full bool) (err error) {

	var errs *multierror.Error

	for _, ofsPart := range ofsPartsList {
		if bakType == misc.IncBackupType {
			err = s.deleteIncBackup(logCh, jobName, ofsPart, full)
		} else {
			err = s.deleteDescBackup(logCh, jobName, ofsPart)
		}
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	return errs.ErrorOrNil()
}

func (s *SMB) deleteDescBackup(logCh chan logger.LogRecord, jobName, ofsPart string) error {
	var errs *multierror.Error
	curDate := time.Now()

	for _, period := range []string{"daily", "weekly", "monthly"} {
		bakDir := path.Join(s.backupPath, ofsPart, period)
		files, err := s.share.ReadDir(bakDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			logCh <- logger.Log(jobName, s.name).Errorf("Failed to read files in remote directory '%s' with next error: %s", bakDir, err)
			return err
		}

		for _, file := range files {

			fileDate := file.ModTime()
			var retentionDate time.Time

			switch period {
			case "daily":
				retentionDate = fileDate.AddDate(0, 0, s.Retention.Days)
			case "weekly":
				retentionDate = fileDate.AddDate(0, 0, s.Retention.Weeks*7)
			case "monthly":
				retentionDate = fileDate.AddDate(0, s.Retention.Months, 0)
			}

			retentionDate = retentionDate.Truncate(24 * time.Hour)
			if curDate.After(retentionDate) {
				err = s.share.Remove(path.Join(bakDir, file.Name()))
				if err != nil {
					logCh <- logger.Log(jobName, s.name).Errorf("Failed to delete file '%s' in remote directory '%s' with next error: %s",
						file.Name(), bakDir, err)
					errs = multierror.Append(errs, err)
				} else {
					logCh <- logger.Log(jobName, s.name).Infof("Deleted old backup file '%s' in remote directory '%s'", file.Name(), bakDir)
				}
			}
		}
	}

	return errs.ErrorOrNil()
}

func (s *SMB) deleteIncBackup(logCh chan logger.LogRecord, jobName, ofsPart string, full bool) error {
	var errs *multierror.Error

	if full {
		backupDir := path.Join(s.backupPath, ofsPart)

		err := s.share.RemoveAll(backupDir)
		if err != nil {
			logCh <- logger.Log(jobName, s.name).Errorf("Failed to delete '%s' with next error: %s", backupDir, err)
			errs = multierror.Append(errs, err)
		}
	} else {
		intMoy, _ := strconv.Atoi(misc.GetDateTimeNow("moy"))
		lastMonth := intMoy - s.Months

		var year string
		if lastMonth > 0 {
			year = misc.GetDateTimeNow("year")
		} else {
			year = misc.GetDateTimeNow("previous_year")
			lastMonth += 12
		}

		backupDir := path.Join(s.backupPath, ofsPart, year)

		dirs, err := s.share.ReadDir(backupDir)
		if err != nil {
			logCh <- logger.Log(jobName, s.name).Errorf("Failed to get access to directory '%s' with next error: %v", backupDir, err)
			return err
		}
		rx := regexp.MustCompile(`month_\d\d`)
		for _, dir := range dirs {
			dirName := dir.Name()
			if rx.MatchString(dirName) {
				dirParts := strings.Split(dirName, "_")
				dirMonth, _ := strconv.Atoi(dirParts[1])
				if dirMonth < lastMonth {
					if err = s.share.RemoveAll(path.Join(backupDir, dirName)); err != nil {
						logCh <- logger.Log(jobName, s.name).Errorf("Failed to delete '%s' in dir '%s' with next error: %s",
							dirName, backupDir, err)
						errs = multierror.Append(errs, err)
					} else {
						logCh <- logger.Log(jobName, s.name).Infof("Deleted old backup '%s' in directory '%s'", dirName, backupDir)
					}
				}
			}
		}
	}

	return errs.ErrorOrNil()
}

func (s *SMB) GetFileReader(ofsPath string) (io.Reader, error) {
	f, err := s.share.Open(ofsPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var buf []byte
	buf, err = io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf), err
}

func (s *SMB) Close() error {
	_ = s.share.Umount()
	return s.session.Logoff()
}

func (s *SMB) Clone() interfaces.Storage {
	cl := *s
	return &cl
}

func (s *SMB) GetName() string {
	return s.name
}
