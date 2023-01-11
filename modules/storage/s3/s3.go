package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"nxs-backup/interfaces"
	"nxs-backup/misc"
	"nxs-backup/modules/logger"
	. "nxs-backup/modules/storage"
)

type s3 struct {
	client     *minio.Client
	bucketName string
	backupPath string
	name       string
	Retention
}

type Params struct {
	BucketName  string
	AccessKeyID string
	SecretKey   string
	Endpoint    string
	Region      string
	Secure      bool
}

func Init(name string, params Params) (*s3, error) {

	s3Client, err := minio.New(params.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(params.AccessKeyID, params.SecretKey, ""),
		Secure: params.Secure,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to init '%s' S3 storage. Error: %v ", name, err)
	}

	return &s3{
		name:       name,
		client:     s3Client,
		bucketName: params.BucketName,
	}, nil
}

func (s *s3) IsLocal() int { return 0 }

func (s *s3) SetBackupPath(path string) {
	s.backupPath = path
}

func (s *s3) SetRetention(r Retention) {
	s.Retention = r
}

func (s *s3) DeliveryBackup(logCh chan logger.LogRecord, jobName, tmpBackupFile, ofs, bakType string) error {
	var bakRemPaths, mtdRemPaths []string

	if bakType == misc.IncBackupType {
		bakRemPaths, mtdRemPaths = GetIncBackupDstList(tmpBackupFile, ofs, s.backupPath)
	} else {
		bakRemPaths = GetDescBackupDstList(tmpBackupFile, ofs, s.backupPath, s.Retention)
	}

	if len(mtdRemPaths) > 0 {
		mtdSrc, err := os.Open(tmpBackupFile + ".inc")
		if err != nil {
			return err
		}
		defer func() { _ = mtdSrc.Close() }()

		mtdSrcStat, err := mtdSrc.Stat()
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

	source, err := os.Open(tmpBackupFile)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	sourceStat, err := source.Stat()
	if err != nil {
		return err
	}

	for _, bucketPath := range bakRemPaths {
		_, err = s.client.PutObject(context.Background(), s.bucketName, bucketPath, source, sourceStat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
		if err != nil {
			return err
		}
		logCh <- logger.Log(jobName, s.name).Infof("Successfully uploaded object '%s' in bucket %s", bucketPath, s.bucketName)
	}

	return nil
}

func (s *s3) DeleteOldBackups(logCh chan logger.LogRecord, ofsPartsList []string, jobName, bakType string, full bool) error {

	var errs *multierror.Error

	objCh := make(chan minio.ObjectInfo)
	curDate := time.Now()

	// Send object that are needed to be removed to objCh
	go func() {
		defer close(objCh)
		for _, ofs := range ofsPartsList {
			backupDir := path.Join(s.backupPath, ofs)
			basePath := strings.TrimPrefix(backupDir, "/")

			for object := range s.client.ListObjects(context.Background(), s.bucketName, minio.ListObjectsOptions{Recursive: true, Prefix: basePath}) {
				if object.Err != nil {
					logCh <- logger.Log(jobName, s.name).Errorf("Failed get objects: '%s'", object.Err)
					errs = multierror.Append(errs, object.Err)
				}

				if bakType == misc.IncBackupType {
					if full {
						objCh <- object
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
								objCh <- object
							}
						}
					}
				} else {
					fileDate := object.LastModified
					var retentionDate time.Time

					if strings.Contains(object.Key, "daily") {
						retentionDate = fileDate.AddDate(0, 0, s.Retention.Days)
					}
					if strings.Contains(object.Key, "weekly") {
						retentionDate = fileDate.AddDate(0, 0, s.Retention.Weeks*7)
					}
					if strings.Contains(object.Key, "monthly") {
						retentionDate = fileDate.AddDate(0, s.Retention.Months, 0)
					}
					retentionDate = retentionDate.Truncate(24 * time.Hour)
					if curDate.After(retentionDate) {
						objCh <- object
					}
				}
			}
		}
	}()

	for rErr := range s.client.RemoveObjects(context.Background(), s.bucketName, objCh, minio.RemoveObjectsOptions{GovernanceBypass: true}) {
		logCh <- logger.Log(jobName, s.name).Errorf("Error detected during object deletion: '%s'", rErr)
		errs = multierror.Append(errs, rErr.Err)
	}

	return errs.ErrorOrNil()
}

func (s *s3) GetFileReader(ofsPath string) (io.Reader, error) {
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

func (s *s3) Close() error {
	return nil
}

func (s *s3) Clone() interfaces.Storage {
	cl := *s
	return &cl
}

func (s *s3) GetName() string {
	return s.name
}
