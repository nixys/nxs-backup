package psql

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/jmoiron/sqlx"

	"github.com/nixys/nxs-backup/ds/psql_connect"
	"github.com/nixys/nxs-backup/interfaces"
	"github.com/nixys/nxs-backup/misc"
	"github.com/nixys/nxs-backup/modules/backend/exec_cmd"
	"github.com/nixys/nxs-backup/modules/backend/targz"
	"github.com/nixys/nxs-backup/modules/logger"
	"github.com/nixys/nxs-backup/modules/metrics"
)

type job struct {
	name             string
	tmpDir           string
	needToMakeBackup bool
	safetyBackup     bool
	deferredCopying  bool
	diskRateLimit    int64
	storages         interfaces.Storages
	targets          map[string]target
	dumpedObjects    map[string]interfaces.DumpObject
	appMetrics       *metrics.Data
}

type target struct {
	connUrl      *url.URL
	dbName       string
	ignoreTables []string
	extraKeys    []string
	gzip         bool
}

type JobParams struct {
	Name             string
	TmpDir           string
	NeedToMakeBackup bool
	SafetyBackup     bool
	DeferredCopying  bool
	DiskRateLimit    int64
	Storages         interfaces.Storages
	Sources          []SourceParams
	Metrics          *metrics.Data
}

type SourceParams struct {
	Name          string
	ConnectParams psql_connect.Params
	TargetDBs     []string
	Excludes      []string
	ExtraKeys     []string
	Gzip          bool
	IsSlave       bool
}

func Init(jp JobParams) (interfaces.Job, error) {

	// check if mysqldump available
	_, err := exec_cmd.Exec("pg_dump", "--version")
	if err != nil {
		return nil, fmt.Errorf("Job `%s` init failed. Can't to check `pg_dump` version. Please install `pg_dump`. Error: %s ", jp.Name, err)
	}

	j := job{
		name:             jp.Name,
		tmpDir:           jp.TmpDir,
		needToMakeBackup: jp.NeedToMakeBackup,
		safetyBackup:     jp.SafetyBackup,
		deferredCopying:  jp.DeferredCopying,
		diskRateLimit:    jp.DiskRateLimit,
		storages:         jp.Storages,
		targets:          make(map[string]target),
		dumpedObjects:    make(map[string]interfaces.DumpObject),
		appMetrics: jp.Metrics.RegisterJob(
			metrics.JobData{
				JobName:       jp.Name,
				JobType:       misc.Postgresql,
				TargetMetrics: make(map[string]metrics.TargetData),
			},
		),
	}

	for _, src := range jp.Sources {

		for _, key := range src.ExtraKeys {
			if matched, _ := regexp.MatchString(`(-f|--file)`, key); matched {
				return nil, fmt.Errorf("Job `%s` init failed. Forbidden usage \"--file|-f\" parameter as extra_keys for `postgresql` jobs type ", jp.Name)
			}
		}

		// fetch databases list to make backup
		var databases []string
		var connUrl *url.URL
		var dbConn *sqlx.DB
		udb := strings.Split(src.ConnectParams.User, "@")

		if misc.Contains(src.TargetDBs, "all") {
			cp := src.ConnectParams
			if len(udb) > 1 {
				cp.Database = udb[1]
				cp.User = udb[0]
			}
			if err = func() error {
				dbConn, err = psql_connect.GetConnect(psql_connect.GetConnUrl(cp))
				if err != nil {
					return fmt.Errorf("Job `%s` init failed. User: `%s`, db: `%s`, PSQL connect error: %s ", jp.Name, cp.User, cp.Database, err)
				}
				if err = dbConn.Ping(); err != nil {
					return fmt.Errorf("Job `%s` init failed. PSQL ping check error: %s ", jp.Name, err)
				}
				defer func() { _ = dbConn.Close() }()
				if err = dbConn.Select(&databases, "SELECT datname FROM pg_database WHERE datistemplate = false;"); err != nil {
					return fmt.Errorf("Job `%s` init failed. Unable to list databases. Error: %s ", jp.Name, err)
				}
				return nil
			}(); err != nil {
				return nil, err
			}
		} else {
			databases = src.TargetDBs
		}

		for _, db := range databases {
			if misc.Contains(src.Excludes, db) {
				continue
			}

			cp := src.ConnectParams
			cp.Database = db
			if len(udb) > 1 {
				cp.User = udb[0]
			}
			connUrl = psql_connect.GetConnUrl(cp)

			var ignoreTables []string
			compRegEx := regexp.MustCompile(`^(?P<db>` + db + `)\.(?P<table>.*$)`)
			for _, excl := range src.Excludes {
				if match := compRegEx.FindStringSubmatch(excl); len(match) > 0 {
					ignoreTables = append(ignoreTables, match[2])
				}
			}

			ofs := src.Name + "/" + db
			j.targets[ofs] = target{
				connUrl:      connUrl,
				dbName:       db,
				ignoreTables: ignoreTables,
				extraKeys:    src.ExtraKeys,
				gzip:         src.Gzip,
			}
			j.appMetrics.Job[j.name].TargetMetrics[ofs] = metrics.TargetData{
				Source: src.Name,
				Target: db,
				Values: make(map[string]float64),
			}
		}
	}

	return &j, nil
}

func (j *job) SetOfsMetrics(ofs string, metricsMap map[string]float64) {
	for m, v := range metricsMap {
		j.appMetrics.Job[j.name].TargetMetrics[ofs].Values[m] = v
	}
}

func (j *job) GetName() string {
	return j.name
}

func (j *job) GetTempDir() string {
	return j.tmpDir
}

func (j *job) GetType() misc.BackupType {
	return misc.Postgresql
}

func (j *job) GetTargetOfsList() (ofsList []string) {
	for ofs := range j.targets {
		ofsList = append(ofsList, ofs)
	}
	return
}

func (j *job) GetStoragesCount() int {
	return len(j.storages)
}

func (j *job) GetDumpObjects() map[string]interfaces.DumpObject {
	return j.dumpedObjects
}

func (j *job) SetDumpObjectDelivered(ofs string) {
	dumpObj := j.dumpedObjects[ofs]
	dumpObj.Delivered = true
	j.dumpedObjects[ofs] = dumpObj
}

func (j *job) IsBackupSafety() bool {
	return j.safetyBackup
}

func (j *job) NeedToMakeBackup() bool {
	return j.needToMakeBackup
}

func (j *job) NeedToUpdateIncMeta() bool {
	return false
}

func (j *job) DeleteOldBackups(logCh chan logger.LogRecord, ofsPath string) error {
	logCh <- logger.Log(j.name, "").Debugf("Starting rotate outdated backups.")
	return j.storages.DeleteOldBackups(logCh, j, ofsPath)
}

func (j *job) CleanupTmpData() error {
	return j.storages.CleanupTmpData(j)
}

func (j *job) DoBackup(logCh chan logger.LogRecord, tmpDir string) error {
	var errs *multierror.Error

	for ofsPart, tgt := range j.targets {
		startTime := time.Now()

		j.SetOfsMetrics(ofsPart, map[string]float64{
			metrics.BackupOk:        float64(0),
			metrics.BackupTime:      float64(0),
			metrics.DeliveryOk:      float64(0),
			metrics.DeliveryTime:    float64(0),
			metrics.BackupSize:      float64(0),
			metrics.BackupTimestamp: float64(startTime.Unix()),
		})

		tmpBackupFile := misc.GetFileFullPath(tmpDir, ofsPart, "sql", "", tgt.gzip)
		err := os.MkdirAll(path.Dir(tmpBackupFile), os.ModePerm)
		if err != nil {
			logCh <- logger.Log(j.name, "").Errorf("Unable to create tmp dir with next error: %s", err)
			errs = multierror.Append(errs, err)
			continue
		}

		if err = j.createTmpBackup(logCh, tmpBackupFile, tgt); err != nil {
			j.SetOfsMetrics(ofsPart, map[string]float64{
				metrics.BackupTime: float64(time.Since(startTime).Nanoseconds() / 1e6),
			})
			logCh <- logger.Log(j.name, "").Errorf("Failed to create temp backups %s", tmpBackupFile)
			errs = multierror.Append(errs, err)
			continue
		}
		fileInfo, _ := os.Stat(tmpBackupFile)
		j.SetOfsMetrics(ofsPart, map[string]float64{
			metrics.BackupOk:   float64(1),
			metrics.BackupTime: float64(time.Since(startTime).Nanoseconds() / 1e6),
			metrics.BackupSize: float64(fileInfo.Size()),
		})

		logCh <- logger.Log(j.name, "").Debugf("Created temp backups %s", tmpBackupFile)

		j.dumpedObjects[ofsPart] = interfaces.DumpObject{TmpFile: tmpBackupFile}

		if !j.deferredCopying {
			if err = j.storages.Delivery(logCh, j); err != nil {
				logCh <- logger.Log(j.name, "").Errorf("Failed to delivery backup. Errors: %v", err)
				errs = multierror.Append(errs, err)
			}
		}
	}

	if err := j.storages.Delivery(logCh, j); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Failed to delivery backup. Errors: %v", err)
		errs = multierror.Append(errs, err)
	}

	return errs.ErrorOrNil()
}

func (j *job) createTmpBackup(logCh chan logger.LogRecord, tmpBackupPath string, target target) error {
	var stderr bytes.Buffer

	backupWriter, err := targz.GetGZipFileWriter(tmpBackupPath, target.gzip, j.diskRateLimit)
	if err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to create tmp file. Error: %s", err)
		return err
	}
	defer func() { _ = backupWriter.Close() }()

	var args []string
	// define command args
	// add tables exclude
	for _, ex := range target.ignoreTables {
		args = append(args, "--exclude-table="+ex)
	}
	// add extra dump cmd options
	if len(target.extraKeys) > 0 {
		args = append(args, target.extraKeys...)
	}
	args = append(args, "--dbname="+target.connUrl.String())

	cmd := exec.Command("pg_dump", args...)
	cmd.Stdout = backupWriter
	cmd.Stderr = &stderr

	logCh <- logger.Log(j.name, "").Debugf("Dump cmd: %s", cmd.String())

	if err = cmd.Start(); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to start pd_dump. Error: %s", err)
		return err
	}
	logCh <- logger.Log(j.name, "").Infof("Starting a `%s` dump", target.dbName)

	if err = cmd.Wait(); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to dump `%s`. Error: %s", target.dbName, stderr.String())
		return err
	}

	logCh <- logger.Log(j.name, "").Infof("Dump of `%s` completed", target.dbName)

	return nil
}

func (j *job) Close() error {
	for _, st := range j.storages {
		_ = st.Close()
	}
	return nil
}
