package nfs

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
	"github.com/sirupsen/logrus"
	"github.com/vmware/go-nfs-client/nfs"
	"github.com/vmware/go-nfs-client/nfs/rpc"

	"github.com/nixys/nxs-backup/interfaces"
	"github.com/nixys/nxs-backup/misc"
	"github.com/nixys/nxs-backup/modules/logger"
	. "github.com/nixys/nxs-backup/modules/storage"
)

type NFS struct {
	target     *nfs.Target
	backupPath string
	name       string
	Retention
}

type Params struct {
	Host   string
	Target string
	UID    uint32
	GID    uint32
}

func Init(name string, params Params) (*NFS, error) {
	mount, err := nfs.DialMount(params.Host)
	if err != nil {
		return nil, fmt.Errorf("Failed to init '%s' NFS storage. Dial MOUNT service error: %v ", name, err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	auth := rpc.NewAuthUnix(hostname, params.UID, params.GID)

	target, err := mount.Mount(params.Target, auth.Auth())
	if err != nil {
		return nil, fmt.Errorf("Failed to init '%s' NFS storage. Mount volume error: %v ", name, err)
	}

	if _, err = target.FSInfo(); err != nil {
		return nil, fmt.Errorf("Failed to init '%s' NFS storage. Get target status error: %v ", name, err)
	}

	if _, err = target.ReadDirPlus("/"); err != nil {
		return nil, fmt.Errorf("Failed to init '%s' NFS storage. Get files error: %v ", name, err)
	}

	return &NFS{
		name:   name,
		target: target,
	}, nil
}

func (n *NFS) IsLocal() int { return 0 }

func (n *NFS) SetBackupPath(path string) {
	n.backupPath = path
}

func (n *NFS) SetRetention(r Retention) {
	n.Retention = r
}

func (n *NFS) DeliveryBackup(logCh chan logger.LogRecord, jobName, tmpBackupFile, ofs, bakType string) error {
	var bakRemPaths, mtdRemPaths []string

	if bakType == misc.IncBackupType {
		bakRemPaths, mtdRemPaths = GetIncBackupDstList(tmpBackupFile, ofs, n.backupPath)
	} else {
		bakRemPaths = GetDescBackupDstList(tmpBackupFile, ofs, n.backupPath, n.Retention)
	}

	if len(mtdRemPaths) > 0 {
		for _, dstPath := range mtdRemPaths {
			if err := n.copy(logCh, jobName, dstPath, tmpBackupFile+".inc"); err != nil {
				return err
			}
		}
	}

	for _, dstPath := range bakRemPaths {
		if err := n.copy(logCh, jobName, dstPath, tmpBackupFile); err != nil {
			return err
		}
	}

	return nil
}

func (n *NFS) copy(logCh chan logger.LogRecord, jobName, dst, src string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		logCh <- logger.LogRecord{
			Level:       logrus.ErrorLevel,
			StorageName: n.name,
			JobName:     jobName,
			Message:     fmt.Sprintf("Unable to open file: '%s'", err),
		}
		return err
	}
	defer func() { _ = srcFile.Close() }()

	// Make remote directories
	dstDir := path.Dir(dst)
	err = n.mkDir(dstDir)
	if err != nil {
		logCh <- logger.LogRecord{
			Level:       logrus.ErrorLevel,
			StorageName: n.name,
			JobName:     jobName,
			Message:     fmt.Sprintf("Unable to create remote directory '%s': '%s'", dstDir, err),
		}
		return err
	}

	destination, err := n.target.OpenFile(dst, 0666)
	if err != nil {
		logCh <- logger.Log(jobName, n.name).Errorf("Unable to create destination file '%s': '%s'", dstDir, err)
		return err
	}
	defer func() { _ = destination.Close() }()

	_, err = io.Copy(destination, srcFile)
	if err != nil {
		logCh <- logger.Log(jobName, n.name).Errorf("Unable to make copy '%s': '%s'", dstDir, err)
		return err
	}
	logCh <- logger.Log(jobName, n.name).Infof("Successfully copied temp backup to %s", dst)

	return nil
}

func (n *NFS) DeleteOldBackups(logCh chan logger.LogRecord, ofsPart string, job interfaces.Job, full bool) error {

	if job.GetType() == misc.IncBackupType {
		return n.deleteIncBackup(logCh, job.GetName(), ofsPart, full)
	} else {
		return n.deleteDescBackup(logCh, job.GetName(), ofsPart, job.IsBackupSafety())
	}
}

func (n *NFS) deleteDescBackup(logCh chan logger.LogRecord, jobName, ofsPart string, safety bool) error {
	var errs *multierror.Error
	curDate := time.Now().Round(24 * time.Hour)

	for _, period := range []string{"daily", "weekly", "monthly"} {
		var retentionDate time.Time
		retentionCount := 0

		switch period {
		case "daily":
			retentionCount = n.Retention.Days
			retentionDate = curDate.AddDate(0, 0, -n.Retention.Days)
		case "weekly":
			if misc.GetDateTimeNow("dow") != misc.WeeklyBackupDay {
				continue
			}
			retentionCount = n.Retention.Weeks
			retentionDate = curDate.AddDate(0, 0, -n.Retention.Weeks*7)
		case "monthly":
			if misc.GetDateTimeNow("dom") != misc.MonthlyBackupDay {
				continue
			}
			retentionCount = n.Retention.Months
			retentionDate = curDate.AddDate(0, -n.Retention.Months, 0)
		}

		bakDir := path.Join(n.backupPath, ofsPart, period)
		files, err := n.target.ReadDirPlus(bakDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			logCh <- logger.Log(jobName, n.name).Errorf("Failed to read files in remote directory '%s' with next error: %s", bakDir, err)
			return err
		}

		if n.Retention.UseCount {
			sort.Slice(files, func(i, j int) bool {
				return files[i].ModTime().Before(files[j].ModTime())
			})

			if !safety {
				retentionCount--
			}
			if retentionCount <= len(files) {
				files = files[:len(files)-retentionCount]
			} else {
				files = files[:0]
			}
		} else {
			i := 0
			for _, file := range files {
				if file.ModTime().Before(retentionDate) {
					files[i] = file
					i++
				}
			}
			files = files[:i]
		}

		for _, file := range files {
			if file.Name() == ".." || file.Name() == "." {
				continue
			}

			err = n.target.Remove(path.Join(bakDir, file.Name()))
			if err != nil {
				logCh <- logger.Log(jobName, n.name).Errorf("Failed to delete file '%s' in remote directory '%s' with next error: %s",
					file.Name(), bakDir, err)
				errs = multierror.Append(errs, err)
			} else {
				logCh <- logger.Log(jobName, n.name).Infof("Deleted old backup file '%s' in remote directory '%s'", file.Name(), bakDir)
			}
		}
	}

	return errs
}

func (n *NFS) deleteIncBackup(logCh chan logger.LogRecord, jobName, ofsPart string, full bool) error {
	var errs *multierror.Error

	if full {
		backupDir := path.Join(n.backupPath, ofsPart)

		err := n.target.RemoveAll(backupDir)
		if err != nil {
			logCh <- logger.Log(jobName, n.name).Errorf("Failed to delete '%s' with next error: %s", backupDir, err)
			errs = multierror.Append(errs, err)
		}
	} else {
		intMoy, _ := strconv.Atoi(misc.GetDateTimeNow("moy"))
		lastMonth := intMoy - n.Months

		var year string
		if lastMonth > 0 {
			year = misc.GetDateTimeNow("year")
		} else {
			year = misc.GetDateTimeNow("previous_year")
			lastMonth += 12
		}

		backupDir := path.Join(n.backupPath, ofsPart, year)

		dirs, err := n.target.ReadDirPlus(backupDir)
		if err != nil {
			logCh <- logger.Log(jobName, n.name).Errorf("Failed to get access to directory '%s' with next error: %v", backupDir, err)
			return err
		}

		for _, dir := range dirs {
			dirName := dir.Name()
			rx := regexp.MustCompile(`month_\d\d`)
			if rx.MatchString(dirName) {
				dirParts := strings.Split(dirName, "_")
				dirMonth, _ := strconv.Atoi(dirParts[1])
				if dirMonth < lastMonth {
					if err = n.target.RemoveAll(path.Join(backupDir, dirName)); err != nil {
						logCh <- logger.Log(jobName, n.name).Errorf("Failed to delete '%s' in dir '%s' with next error: %s",
							dir.Name(), backupDir, err)
						errs = multierror.Append(errs, err)
					} else {
						logCh <- logger.Log(jobName, n.name).Infof("Deleted old backup '%s' in directory '%s'", dir.Name(), backupDir)
					}
				}
			}
		}
	}

	return errs.ErrorOrNil()
}

func (n *NFS) mkDir(dstPath string) error {

	dstPath = path.Clean(dstPath)
	if dstPath == "." || dstPath == "/" {
		return nil
	}
	fi, err := n.getInfo(dstPath)
	if err == nil {
		if fi.IsDir() {
			return nil
		}
		return fmt.Errorf("%s is a file not a directory", dstPath)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("mkdir %q failed: %w", dstPath, err)
	}

	dir := path.Dir(dstPath)
	err = n.mkDir(dir)
	if err != nil {
		return err
	}
	_, err = n.target.Mkdir(dstPath, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func (n *NFS) getInfo(dstPath string) (os.FileInfo, error) {

	dir := path.Dir(dstPath)
	base := path.Base(dstPath)

	files, err := n.target.ReadDirPlus(dir)
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

func (n *NFS) GetFileReader(ofsPath string) (io.Reader, error) {

	file, err := n.target.Open(path.Join(n.backupPath, ofsPath))
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var buf []byte
	buf, err = io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf), err
}

func (n *NFS) Close() error {
	return n.target.Close()
}

func (n *NFS) Clone() interfaces.Storage {
	cl := *n
	return &cl
}

func (n *NFS) GetName() string {
	return n.name
}
