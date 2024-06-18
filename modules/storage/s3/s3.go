package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/nixys/nxs-backup/interfaces"
	"github.com/nixys/nxs-backup/misc"
	"github.com/nixys/nxs-backup/modules/backend/files"
	"github.com/nixys/nxs-backup/modules/logger"
	. "github.com/nixys/nxs-backup/modules/storage"
)

type S3 struct {
	client        *minio.Client
	bucketName    string
	backupPath    string
	name          string
	batchDeletion bool
	rateLimit     int64
	Retention
}

type Opts struct {
	BucketName    string
	AccessKeyID   string
	SecretKey     string
	Endpoint      string
	Region        string
	BatchDeletion bool
	Secure        bool
}

func Init(name string, opts Opts, rl int64) (*S3, error) {

	s3Client, err := minio.New(opts.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(opts.AccessKeyID, opts.SecretKey, ""),
		Secure: opts.Secure,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to init '%s' S3 storage. Error: %v ", name, err)
	}

	exist, err := s3Client.BucketExists(context.Background(), opts.BucketName)
	if err != nil {
		return nil, fmt.Errorf("Failed to check bucket exist in S3 storage '%s'. Error: %v ", name, err)
	}
	if !exist {
		return nil, fmt.Errorf("Bucket '%s' doesn't exist. ", opts.BucketName)
	}

	return &S3{
		name:          name,
		client:        s3Client,
		bucketName:    opts.BucketName,
		batchDeletion: opts.BatchDeletion,
		rateLimit:     rl,
	}, nil
}

func (s *S3) IsLocal() int { return 0 }

func (s *S3) SetBackupPath(path string) {
	s.backupPath = strings.TrimPrefix(path, "/")
}

func (s *S3) SetRateLimit(rl int64) {
	s.rateLimit = rl
}

func (s *S3) SetRetention(r Retention) {
	s.Retention = r
}

func (s *S3) DeliveryBackup(logCh chan logger.LogRecord, jobName, tmpBackupFile, ofs, bakType string) error {
	var bakRemPaths, mtdRemPaths []string

	if bakType == string(misc.IncFiles) {
		bakRemPaths, mtdRemPaths = GetIncBackupDstList(tmpBackupFile, ofs, s.backupPath)
	} else {
		bakRemPaths = GetDescBackupDstList(tmpBackupFile, ofs, s.backupPath, s.Retention)
	}

	if len(mtdRemPaths) > 0 {
		mtdSrc, err := files.GetLimitedFileReader(tmpBackupFile+".inc", s.rateLimit)
		if err != nil {
			return err
		}
		defer func() { _ = mtdSrc.Close() }()

		mtdSrcStat, err := os.Stat(tmpBackupFile + ".inc")
		if err != nil {
			return err
		}

		for _, bucketPath := range mtdRemPaths {
			_, err = s.client.PutObject(context.Background(), s.bucketName, bucketPath, mtdSrc, mtdSrcStat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
			if err != nil {
				return err
			}
			logCh <- logger.Log(jobName, s.name).Infof("Successfully uploaded object '%s' in bucket %s", bucketPath, s.bucketName)
		}
	}

	source, err := files.GetLimitedFileReader(tmpBackupFile, s.rateLimit)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	sourceStat, err := os.Stat(tmpBackupFile)
	if err != nil {
		return err
	}

	for _, bucketPath := range bakRemPaths {
		res, err := s.client.PutObject(context.Background(), s.bucketName, bucketPath, source, sourceStat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
		if err != nil {
			logCh <- logger.Log(jobName, s.name).Errorf("Failed to uploaded object '%s' to bucket %s", bucketPath, s.bucketName)
			logCh <- logger.Log(jobName, s.name).Errorf("Error: %s", err)
			logCh <- logger.Log(jobName, s.name).Debugf("Response: %+v\n", res)
			return err
		}
		logCh <- logger.Log(jobName, s.name).Infof("Successfully uploaded object '%s' to bucket %s", bucketPath, s.bucketName)
	}

	return nil
}

func (s *S3) DeleteOldBackups(logCh chan logger.LogRecord, ofs string, job interfaces.Job, full bool) error {

	curDate := time.Now().Round(24 * time.Hour)

	objCh := make(chan minio.ObjectInfo)

	// Send object that are needed to be removed to objCh
	filesList := make(map[string][]minio.ObjectInfo)

	backupDir := path.Join(s.backupPath, ofs)

	for object := range s.client.ListObjects(context.Background(), s.bucketName, minio.ListObjectsOptions{Recursive: true, Prefix: backupDir}) {
		if object.Err != nil {
			logCh <- logger.Log(job.GetName(), s.name).Errorf("Failed get objects: '%s'", object.Err)
			return object.Err
		}

		if job.GetType() == misc.IncFiles {
			if full {
				filesList["inc"] = append(filesList["inc"], object)
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
				rx := regexp.MustCompile(year + "/month_\\d\\d")
				if rx.MatchString(object.Key) {
					dirParts := strings.Split(path.Base(object.Key), "_")
					dirMonth, _ := strconv.Atoi(dirParts[1])
					if dirMonth < lastMonth {
						filesList["inc"] = append(filesList["inc"], object)
					}
				}
			}
		} else {
			if strings.Contains(object.Key, Daily.String()) && s.Retention.Days > 0 {
				if s.Retention.UseCount || object.LastModified.Before(curDate.AddDate(0, 0, -s.Retention.Days)) {
					filesList["daily"] = append(filesList["daily"], object)
				}
			} else if strings.Contains(object.Key, Weekly.String()) && s.Retention.Weeks > 0 && misc.GetDateTimeNow("dow") == misc.WeeklyBackupDay {
				if s.Retention.UseCount || object.LastModified.Before(curDate.AddDate(0, 0, -s.Retention.Weeks*7)) {
					filesList["weekly"] = append(filesList["weekly"], object)
				}
			} else if strings.Contains(object.Key, Monthly.String()) && s.Retention.Weeks > 0 && misc.GetDateTimeNow("dom") == misc.MonthlyBackupDay {
				if s.Retention.UseCount || object.LastModified.Before(curDate.AddDate(0, -s.Retention.Months, 0)) {
					filesList["monthly"] = append(filesList["monthly"], object)
				}
			}
		}
	}

	go func() {
		defer close(objCh)
		for period, files := range filesList {
			needSort := true
			retentionCount := 0
			switch period {
			case "inc":
				needSort = false
			case "daily":
				retentionCount = s.Retention.Days
			case "weekly":
				retentionCount = s.Retention.Weeks
			case "monthly":
				retentionCount = s.Retention.Months
			}

			if needSort && s.Retention.UseCount {
				sort.Slice(files, func(i, j int) bool {
					return files[i].LastModified.Before(files[j].LastModified)
				})

				if !job.IsBackupSafety() {
					retentionCount--
				}
				if retentionCount <= len(files) {
					files = files[:len(files)-retentionCount]
				} else {
					files = files[:0]
				}
			}

			for _, file := range files {
				logCh <- logger.Log(job.GetName(), s.name).Infof("File '%s' going to be deleted", file.Key)
				objCh <- file
			}
		}
	}()

	if s.batchDeletion {
		for err := range s.client.RemoveObjects(context.Background(), s.bucketName, objCh, minio.RemoveObjectsOptions{GovernanceBypass: true}) {
			logCh <- logger.Log(job.GetName(), s.name).Errorf("Error detected during multiple objects deletion: '%s'", err)
			return err.Err
		}
	} else {
		for object := range objCh {
			if err := s.client.RemoveObject(context.Background(), s.bucketName, object.Key, minio.RemoveObjectOptions{GovernanceBypass: true}); err != nil {
				logCh <- logger.Log(job.GetName(), s.name).Errorf("Error detected during single object deletion: '%s'", err)
				return err
			}
		}
	}
	return nil
}

func (s *S3) GetFileReader(ofsPath string) (io.Reader, error) {
	o, err := s.client.GetObject(context.Background(), s.bucketName, path.Join(s.backupPath, ofsPath), minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = o.Close() }()

	var buf []byte
	buf, err = io.ReadAll(o)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf), err
}

func (s *S3) Close() error {
	return nil
}

func (s *S3) Clone() interfaces.Storage {
	cl := *s
	return &cl
}

func (s *S3) GetName() string {
	return s.name
}
