package sftp

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
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"nxs-backup/interfaces"
	"nxs-backup/misc"
	"nxs-backup/modules/logger"
	. "nxs-backup/modules/storage"
)

type SFTP struct {
	client     *sftp.Client
	backupPath string
	name       string
	Retention
}

type Params struct {
	User           string
	Host           string
	Port           int
	Password       string
	KeyFile        string
	ConnectTimeout time.Duration
}

func Init(name string, params Params) (*SFTP, error) {

	sshConfig := &ssh.ClientConfig{
		User:            params.User,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         params.ConnectTimeout * time.Second,
		ClientVersion:   "SSH-2.0-" + "nxs-backup/" + misc.VERSION,
	}

	if params.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(params.Password))
	}

	// Load key file if specified
	if params.KeyFile != "" {
		key, err := os.ReadFile(params.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to init '%s' SFTP storage. Error: %v ", name, fmt.Errorf("failed to read private key file: %w", err))
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("Failed to init '%s' SFTP storage. Error: %v ", name, fmt.Errorf("failed to parse private key file: %w", err))
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	}

	sshConn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", params.Host, params.Port), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to init '%s' SFTP storage. Error: %v ", name, fmt.Errorf("couldn't connect SSH: %w", err))
	}

	sftpClient, err := sftp.NewClient(sshConn)
	if err != nil {
		_ = sshConn.Close()
		return nil, fmt.Errorf("Failed to init '%s' SFTP storage. Error: %v ", name, fmt.Errorf("couldn't initialise SFTP: %w", err))
	}

	return &SFTP{
		name:   name,
		client: sftpClient,
	}, nil

}

func (s *SFTP) IsLocal() int { return 0 }

func (s *SFTP) SetBackupPath(path string) {
	s.backupPath = path
}

func (s *SFTP) SetRetention(r Retention) {
	s.Retention = r
}

func (s *SFTP) DeliveryBackup(logCh chan logger.LogRecord, jobName, tmpBackupFile, ofs, bakType string) (err error) {
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
		if err = s.deliveryBackupMetadata(logCh, jobName, tmpBackupFile, mtdDstPath); err != nil {
			return
		}
	}

	// Make remote directories
	rmDir := path.Dir(bakDstPath)
	if err = s.client.MkdirAll(rmDir); err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to create remote directory '%s': '%s'", rmDir, err)
		return err
	}

	dstFile, err := s.client.Create(bakDstPath)
	if err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to create remote file: %s", err)
		return err
	}
	defer func() { _ = dstFile.Close() }()

	srcFile, err := os.Open(tmpBackupFile)
	if err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to open tmp backup: '%s'", err)
		return err
	}
	defer func() { _ = srcFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to upload file: %s", err)
		return err
	}
	logCh <- logger.Log(jobName, s.name).Infof("file %s uploaded", dstFile.Name())

	for dst, src := range links {
		rmDir = path.Dir(dst)
		err = s.client.MkdirAll(rmDir)
		if err != nil {
			logCh <- logger.Log(jobName, s.name).Errorf("Unable to create remote directory '%s': '%s'", rmDir, err)
			return
		}
		err = s.client.Symlink(src, dst)
		if err != nil {
			logCh <- logger.Log(jobName, s.name).Errorf("Unable to create symlink: %s", err)
			return
		}
	}

	return
}

func (s *SFTP) deliveryBackupMetadata(logCh chan logger.LogRecord, jobName, tmpBackupFile, mtdDstPath string) error {
	mtdSrcPath := tmpBackupFile + ".inc"

	// Make remote directories
	rmDir := path.Dir(mtdDstPath)
	if err := s.client.MkdirAll(rmDir); err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to create remote directory '%s': '%s'", rmDir, err)
		return err
	}

	_ = s.client.Remove(mtdDstPath)
	mtdDst, err := s.client.Create(mtdDstPath)
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
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to make copy: %s", err)
		return err
	}
	logCh <- logger.Log(jobName, s.name).Infof("Successfully copied metadata to %s", mtdDstPath)

	return nil
}

func (s *SFTP) DeleteOldBackups(logCh chan logger.LogRecord, ofsPart string, job interfaces.Job, full bool) error {

	if job.GetType() == misc.IncBackupType {
		return s.deleteIncBackup(logCh, job.GetName(), ofsPart, full)
	} else {
		return s.deleteDescBackup(logCh, job.GetName(), ofsPart, job.IsBackupSafety())
	}
}

func (s *SFTP) deleteDescBackup(logCh chan logger.LogRecord, jobName, ofsPart string, safety bool) error {
	var errs *multierror.Error
	curDate := time.Now().Round(24 * time.Hour)

	for _, period := range []string{"daily", "weekly", "monthly"} {
		var retentionDate time.Time
		retentionCount := 0

		switch period {
		case "daily":
			retentionCount = s.Retention.Days
			retentionDate = curDate.AddDate(0, 0, -s.Retention.Days)
		case "weekly":
			if misc.GetDateTimeNow("dow") != misc.WeeklyBackupDay {
				continue
			}
			retentionCount = s.Retention.Weeks
			retentionDate = curDate.AddDate(0, 0, -s.Retention.Weeks*7)
		case "monthly":
			if misc.GetDateTimeNow("dom") != misc.MonthlyBackupDay {
				continue
			}
			retentionCount = s.Retention.Months
			retentionDate = curDate.AddDate(0, -s.Retention.Months, 0)
		}

		bakDir := path.Join(s.backupPath, ofsPart, period)
		files, err := s.client.ReadDir(bakDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			logCh <- logger.Log(jobName, s.name).Errorf("Failed to read files in remote directory '%s' with next error: %s", bakDir, err)
			return err
		}

		if s.Retention.UseCount {
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

			err = s.client.Remove(path.Join(bakDir, file.Name()))
			if err != nil {
				logCh <- logger.Log(jobName, s.name).Errorf("Failed to delete file '%s' in remote directory '%s' with next error: %s",
					file.Name(), bakDir, err)
				errs = multierror.Append(errs, err)
			} else {
				logCh <- logger.Log(jobName, s.name).Infof("Deleted old backup file '%s' in remote directory '%s'", file.Name(), bakDir)
			}
		}
	}

	return errs.ErrorOrNil()
}

func (s *SFTP) deleteIncBackup(logCh chan logger.LogRecord, jobName, ofsPart string, full bool) error {
	var errs *multierror.Error

	if full {
		backupDir := path.Join(s.backupPath, ofsPart)
		if err := s.client.Remove(backupDir); err != nil {
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

		dirs, err := s.client.ReadDir(backupDir)
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
					if err = s.client.Remove(path.Join(backupDir, dirName)); err != nil {
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

func (s *SFTP) GetFileReader(ofsPath string) (io.Reader, error) {
	f, err := s.client.Open(ofsPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = s.Close() }()

	var buf []byte
	buf, err = io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf), err
}

func (s *SFTP) Close() error {
	return s.client.Close()
}

func (s *SFTP) Clone() interfaces.Storage {
	cl := *s
	return &cl
}

func (s *SFTP) GetName() string {
	return s.name
}
