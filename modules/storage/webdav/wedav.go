package webdav

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"

	"nxs-backup/interfaces"
	"nxs-backup/misc"
	"nxs-backup/modules/backend/webdav"
	"nxs-backup/modules/logger"
	. "nxs-backup/modules/storage"
)

type webDav struct {
	client     *webdav.Client
	backupPath string
	name       string
	Retention
}

type Params struct {
	URL               string
	Username          string
	Password          string
	OAuthToken        string
	ConnectionTimeout time.Duration
}

func Init(name string, params Params) (*webDav, error) {

	client, err := webdav.Init(webdav.Params{
		URL:               params.URL,
		Username:          params.Username,
		Password:          params.Password,
		OAuthToken:        params.OAuthToken,
		ConnectionTimeout: params.ConnectionTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to init '%s' WebDav storage. Error: %v ", name, err)
	}

	return &webDav{
		name:   name,
		client: client,
	}, nil
}

func (wd *webDav) IsLocal() int { return 0 }

func (wd *webDav) SetBackupPath(path string) {
	wd.backupPath = path
}

func (wd *webDav) SetRetention(r Retention) {
	wd.Retention = r
}

func (wd *webDav) DeliveryBackup(logCh chan logger.LogRecord, jobName, tmpBackupFile, ofs, bakType string) (err error) {

	var (
		bakDstPath, mtdDstPath string
		links                  map[string]string
	)

	if bakType == misc.IncBackupType {
		bakDstPath, mtdDstPath, links, err = GetIncBackupDstAndLinks(tmpBackupFile, ofs, wd.backupPath)
	} else {
		bakDstPath, links, err = GetDescBackupDstAndLinks(tmpBackupFile, ofs, wd.backupPath, wd.Retention)
	}
	if err != nil {
		logCh <- logger.Log(jobName, wd.name).Errorf("Unable to get destination path and links: '%s'", err)
		return
	}

	if mtdDstPath != "" {
		if err = wd.copy(logCh, jobName, tmpBackupFile+".inc", bakDstPath); err != nil {
			logCh <- logger.Log(jobName, wd.name).Errorf("Unable to upload tmp backup")
			return
		}
	}

	if err = wd.copy(logCh, jobName, tmpBackupFile, bakDstPath); err != nil {
		logCh <- logger.Log(jobName, wd.name).Errorf("Unable to upload tmp backup")
		return
	}

	for dst, src := range links {
		remDir := path.Dir(dst)
		err = wd.mkDir(path.Dir(dst))
		if err != nil {
			logCh <- logger.Log(jobName, wd.name).Errorf("Unable to create remote directory '%s': '%s'", remDir, err)
			return
		}
		err = wd.client.Copy(src, dst)
		if err != nil {
			logCh <- logger.Log(jobName, wd.name).Errorf("Unable to make copy: %s", err)
			return
		}
	}

	return
}

func (wd *webDav) copy(logCh chan logger.LogRecord, jobName, srcPath, dstPath string) (err error) {

	// Make remote directories
	remDir := path.Dir(dstPath)
	if err = wd.mkDir(remDir); err != nil {
		logCh <- logger.Log(jobName, wd.name).Errorf("Unable to create remote directory '%s': '%s'", remDir, err)
		return
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		logCh <- logger.Log(jobName, wd.name).Errorf("Unable to open '%s'", err)
		return
	}
	defer func() { _ = srcFile.Close() }()

	err = wd.client.Upload(dstPath, srcFile)
	if err != nil {
		logCh <- logger.Log(jobName, wd.name).Errorf("Unable to upload file: %s", err)
	} else {
		logCh <- logger.Log(jobName, wd.name).Infof("File %s successfull uploaded", dstPath)
	}

	return err
}

func (wd *webDav) DeleteOldBackups(logCh chan logger.LogRecord, ofsPartsList []string, jobName, bakType string, full bool) (err error) {

	var errs *multierror.Error

	for _, ofsPart := range ofsPartsList {
		if bakType == misc.IncBackupType {
			err = wd.deleteIncBackup(logCh, jobName, ofsPart, full)
		} else {
			err = wd.deleteDescBackup(logCh, jobName, ofsPart)
		}
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	return errs.ErrorOrNil()
}

func (wd *webDav) deleteDescBackup(logCh chan logger.LogRecord, jobName, ofsPart string) error {

	var errs *multierror.Error
	curDate := time.Now()

	for _, period := range []string{"daily", "weekly", "monthly"} {
		bakDir := path.Join(wd.backupPath, ofsPart, period)
		files, err := wd.client.Ls(bakDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			logCh <- logger.Log(jobName, wd.name).Errorf("Failed to read files in remote directory '%s' with next error: %s", bakDir, err)
			return err
		}

		for _, file := range files {
			fileDate := file.ModTime()
			var retentionDate time.Time

			switch period {
			case "daily":
				retentionDate = fileDate.AddDate(0, 0, wd.Retention.Days)
			case "weekly":
				retentionDate = fileDate.AddDate(0, 0, wd.Retention.Weeks*7)
			case "monthly":
				retentionDate = fileDate.AddDate(0, wd.Retention.Months, 0)
			}

			retentionDate = retentionDate.Truncate(24 * time.Hour)
			if curDate.After(retentionDate) {
				err = wd.client.Rm(path.Join(bakDir, file.Name()))
				if err != nil {
					logCh <- logger.Log(jobName, wd.name).Errorf("Failed to delete file '%s' in remote directory '%s' with next error: %s",
						file.Name(), bakDir, err)
					errs = multierror.Append(errs, err)
				} else {
					logCh <- logger.Log(jobName, wd.name).Infof("Deleted old backup file '%s' in remote directory '%s'", file.Name(), bakDir)
				}
			}
		}
	}

	return errs.ErrorOrNil()
}

func (wd *webDav) deleteIncBackup(logCh chan logger.LogRecord, jobName, ofsPart string, full bool) error {
	var errs *multierror.Error

	if full {
		backupDir := path.Join(wd.backupPath, ofsPart)

		err := wd.client.Rm(backupDir)
		if err != nil {
			logCh <- logger.Log(jobName, wd.name).Errorf("Failed to delete '%s' with next error: %s", backupDir, err)
			errs = multierror.Append(errs, err)
		}
	} else {
		intMoy, _ := strconv.Atoi(misc.GetDateTimeNow("moy"))
		lastMonth := intMoy - wd.Months

		var year string
		if lastMonth > 0 {
			year = misc.GetDateTimeNow("year")
		} else {
			year = misc.GetDateTimeNow("previous_year")
			lastMonth += 12
		}

		backupDir := path.Join(wd.backupPath, ofsPart, year)

		dirs, err := wd.client.Ls(backupDir)
		if err != nil {
			logCh <- logger.Log(jobName, wd.name).Errorf("Failed to get access to directory '%s' with next error: %v", backupDir, err)
			return err
		}
		rx := regexp.MustCompile(`month_\d\d`)
		for _, dir := range dirs {
			dirName := dir.Name()
			if rx.MatchString(dirName) {
				dirParts := strings.Split(dirName, "_")
				dirMonth, _ := strconv.Atoi(dirParts[1])
				if dirMonth < lastMonth {
					if err = wd.client.Rm(path.Join(backupDir, dirName)); err != nil {
						logCh <- logger.Log(jobName, wd.name).Errorf("Failed to delete '%s' in dir '%s' with next error: %s",
							dirName, backupDir, err)
						errs = multierror.Append(errs, err)
					}
				}
			}
		}
	}

	return errs.ErrorOrNil()
}

func (wd *webDav) mkDir(dstPath string) error {

	dstPath = path.Clean(dstPath)
	if dstPath == "." || dstPath == "/" {
		return nil
	}
	fi, err := wd.getInfo(dstPath)
	if err == nil {
		if fi.IsDir() {
			return nil
		}
		return fmt.Errorf("%s is a file not a directory", dstPath)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("mkdir %q failed: %w", dstPath, err)
	}

	dir := path.Dir(dstPath)
	err = wd.mkDir(dir)
	if err != nil {
		return err
	}
	err = wd.client.Mkdir(dstPath)
	if err != nil {
		return err
	}

	return nil
}

func (wd *webDav) getInfo(dstPath string) (os.FileInfo, error) {

	dir := path.Dir(dstPath)
	base := path.Base(dstPath)

	files, err := wd.client.Ls(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.Name() == base {
			return file, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (wd *webDav) GetFileReader(ofsPath string) (io.Reader, error) {
	f, err := wd.client.Read(ofsPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var buf []byte
	buf, err = ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf), err
}

func (wd *webDav) Close() error {
	return nil
}

func (wd *webDav) Clone() interfaces.Storage {
	cl := *wd
	return &cl
}

func (wd *webDav) GetName() string {
	return wd.name
}
