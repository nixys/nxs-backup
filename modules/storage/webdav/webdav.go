package webdav

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/nixys/nxs-backup/interfaces"
	"github.com/nixys/nxs-backup/misc"
	"github.com/nixys/nxs-backup/modules/backend/files"
	"github.com/nixys/nxs-backup/modules/backend/webdav"
	"github.com/nixys/nxs-backup/modules/logger"
	. "github.com/nixys/nxs-backup/modules/storage"
)

type WebDav struct {
	client        *webdav.Client
	name          string
	backupPath    string
	rateLimit     int64
	rotateEnabled bool
	Retention
}

type Opts struct {
	URL               string
	Username          string
	Password          string
	OAuthToken        string
	ConnectionTimeout time.Duration
}

func Init(name string, params Opts, rl int64) (*WebDav, error) {

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

	return &WebDav{
		name:      name,
		client:    client,
		rateLimit: rl,
	}, nil
}

func (wd *WebDav) Configure(p Params) {
	wd.backupPath = path.Join("/", p.BackupPath)
	wd.rateLimit = p.RateLimit
	wd.rotateEnabled = p.RotateEnabled
	wd.Retention = p.Retention
}

func (wd *WebDav) IsLocal() int { return 0 }

func (wd *WebDav) DeliveryBackup(logCh chan logger.LogRecord, jobName, tmpBackupFile, ofs, bakType string) (err error) {

	var (
		bakDstPath, mtdDstPath string
		links                  map[string]string
	)

	if bakType == string(misc.IncFiles) {
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

func (wd *WebDav) copy(logCh chan logger.LogRecord, jobName, srcPath, dstPath string) (err error) {

	// Make remote directories
	remDir := path.Dir(dstPath)
	if err = wd.mkDir(remDir); err != nil {
		logCh <- logger.Log(jobName, wd.name).Errorf("Unable to create remote directory '%s': '%s'", remDir, err)
		return
	}

	srcFile, err := files.GetLimitedFileReader(srcPath, wd.rateLimit)
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

func (wd *WebDav) DeleteOldBackups(logCh chan logger.LogRecord, ofsPart string, job interfaces.Job, full bool) error {
	if !wd.rotateEnabled {
		logCh <- logger.Log(job.GetName(), wd.name).Debugf("Backup rotate skipped by config.")
		return nil
	}

	if job.GetType() == misc.IncFiles {
		return wd.deleteIncBackup(logCh, job.GetName(), ofsPart, full)
	} else {
		return wd.deleteDescBackup(logCh, job.GetName(), ofsPart, job.IsBackupSafety())
	}
}

func (wd *WebDav) deleteDescBackup(logCh chan logger.LogRecord, jobName, ofsPart string, safety bool) error {
	var errs *multierror.Error

	for _, p := range RetentionPeriodsList {
		retentionCount, retentionDate := GetRetention(p, wd.Retention)
		if retentionCount == 0 && retentionDate.IsZero() {
			continue
		}

		bakDir := path.Join(wd.backupPath, ofsPart, p.String())
		wdFiles, err := wd.client.Ls(bakDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			logCh <- logger.Log(jobName, wd.name).Errorf("Failed to read files in remote directory '%s' with next error: %s", bakDir, err)
			return err
		}

		if wd.Retention.UseCount {
			sort.Slice(wdFiles, func(i, j int) bool {
				return wdFiles[i].ModTime().Before(wdFiles[j].ModTime())
			})

			if !safety {
				retentionCount--
			}
			if retentionCount <= len(wdFiles) {
				wdFiles = wdFiles[:len(wdFiles)-retentionCount]
			} else {
				wdFiles = wdFiles[:0]
			}
		} else {
			i := 0
			for _, file := range wdFiles {
				if file.ModTime().Before(retentionDate) {
					wdFiles[i] = file
					i++
				}
			}
			wdFiles = wdFiles[:i]
		}

		for _, file := range wdFiles {
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

	return errs.ErrorOrNil()
}

func (wd *WebDav) deleteIncBackup(logCh chan logger.LogRecord, jobName, ofsPart string, full bool) error {
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
					} else {
						logCh <- logger.Log(jobName, wd.name).Infof("Deleted old backup '%s' in directory '%s'", dirName, backupDir)
					}
				}
			}
		}
	}

	return errs.ErrorOrNil()
}

func (wd *WebDav) mkDir(dstPath string) error {

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

func (wd *WebDav) getInfo(dstPath string) (os.FileInfo, error) {

	dir := path.Dir(dstPath)
	base := path.Base(dstPath)

	wdfl, err := wd.client.Ls(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range wdfl {
		if file.Name() == base {
			return file, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (wd *WebDav) GetFileReader(ofsPath string) (io.Reader, error) {
	f, err := wd.client.Read(path.Join(wd.backupPath, ofsPath))
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

func (wd *WebDav) ListBackups(ofsPath string) ([]string, error) {
	bPath := path.Join(wd.backupPath, ofsPath)

	fl, err := wd.client.Ls(bPath)
	if err != nil {
		return nil, err
	}

	return wd.listPaths(bPath, fl)
}

func (wd *WebDav) listPaths(base string, fList []fs.FileInfo) ([]string, error) {
	var paths []string

	for _, file := range fList {
		if !file.IsDir() {
			fp := path.Join(base, file.Name())
			paths = append(paths, fp)
		} else {
			subDir := path.Join(base, file.Name())
			subDirFiles, err := wd.client.Ls(subDir)
			if err != nil {
				return nil, err
			}
			subPaths, err := wd.listPaths(subDir, subDirFiles)
			if err != nil {
				return nil, err
			}
			paths = append(paths, subPaths...)
		}
	}

	return paths, nil
}

func (wd *WebDav) Close() error {
	return nil
}

func (wd *WebDav) Clone() interfaces.Storage {
	cl := *wd
	return &cl
}

func (wd *WebDav) GetName() string {
	return wd.name
}
