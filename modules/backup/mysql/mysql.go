package mysql

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/jmoiron/sqlx"
	"gopkg.in/ini.v1"

	"github.com/nixys/nxs-backup/ds/mysql_connect"
	"github.com/nixys/nxs-backup/interfaces"
	"github.com/nixys/nxs-backup/misc"
	"github.com/nixys/nxs-backup/modules/backend/exec_cmd"
	"github.com/nixys/nxs-backup/modules/backend/files"
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
	authFilesKeys    map[string][]byte
	appMetrics       *metrics.Data
	jobMetrics       metrics.JobData
}

type target struct {
	connect      *sqlx.DB
	authFile     *ini.File
	dbName       string
	ignoreTables []string
	extraKeys    []string
	isSlave      bool
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
	OldMetrics       *metrics.Data
}

type SourceParams struct {
	Name          string
	ConnectParams mysql_connect.Params
	TargetDBs     []string
	Excludes      []string
	ExtraKeys     []string
	Gzip          bool
	IsSlave       bool
}

func Init(jp JobParams) (interfaces.Job, error) {

	// check if mysqldump available
	if _, err := exec_cmd.Exec("mysqldump", "--version"); err != nil {
		return nil, fmt.Errorf("Job `%s` init failed. Can't to check `mysqldump` version. Please install `mysqldump`. Error: %s ", jp.Name, err)
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
		appMetrics:       jp.Metrics,
		jobMetrics: metrics.JobData{
			JobName:       jp.Name,
			JobType:       misc.Mysql,
			TargetMetrics: make(map[string]metrics.TargetData),
		},
	}

	ojm := jp.OldMetrics.GetMetrics(jp.Name)

	for _, src := range jp.Sources {

		dbConn, authFile, err := mysql_connect.GetConnectAndCnfFile(src.ConnectParams, "mysqldump")
		if err != nil {
			return nil, fmt.Errorf("Job `%s` init failed. MySQL connect error: %s ", jp.Name, err)
		}

		// fetch all databases
		var databases []string
		if misc.Contains(src.TargetDBs, "all") {
			err = dbConn.Select(&databases, "show databases")
			if err != nil {
				return nil, fmt.Errorf("Job `%s` init failed. Unable to list databases. Error: %s ", jp.Name, err)
			}
		} else {
			databases = src.TargetDBs
		}

		for _, db := range databases {
			if misc.Contains(src.Excludes, db) {
				continue
			}

			var ignoreTables []string
			for _, excl := range src.Excludes {
				if matched, _ := regexp.MatchString(`^`+db+`\..*$`, excl); matched {
					ignoreTables = append(ignoreTables, "--ignore-table="+excl)
				}
			}

			ofs := src.Name + "/" + db
			j.targets[ofs] = target{
				connect:      dbConn,
				authFile:     authFile,
				dbName:       db,
				ignoreTables: ignoreTables,
				extraKeys:    src.ExtraKeys,
				gzip:         src.Gzip,
				isSlave:      src.IsSlave,
			}
			if otm, ok := ojm.TargetMetrics[ofs]; ok {
				j.jobMetrics.TargetMetrics[ofs] = otm
			} else {
				j.jobMetrics.TargetMetrics[ofs] = metrics.TargetData{
					Source: src.Name,
					Target: db,
					Values: make(map[string]float64),
				}
			}
		}
	}

	j.ExportMetrics()
	return &j, nil
}

func (j *job) SetOfsMetrics(ofs string, metricsMap map[string]float64) {
	for m, v := range metricsMap {
		j.jobMetrics.TargetMetrics[ofs].Values[m] = v
	}
}

func (j *job) ExportMetrics() {
	j.appMetrics.JobMetricsSet(j.jobMetrics)
}

func (j *job) GetName() string {
	return j.name
}

func (j *job) GetTempDir() string {
	return j.tmpDir
}

func (j *job) GetType() misc.BackupType {
	return misc.Mysql
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
		j.SetOfsMetrics(ofsPart, map[string]float64{
			metrics.BackupOk:     float64(0),
			metrics.BackupTime:   float64(0),
			metrics.DeliveryOk:   float64(0),
			metrics.DeliveryTime: float64(0),
			metrics.BackupSize:   float64(0),
		})

		tmpBackupFile := misc.GetFileFullPath(tmpDir, ofsPart, "sql", "", tgt.gzip)
		err := os.MkdirAll(path.Dir(tmpBackupFile), os.ModePerm)
		if err != nil {
			logCh <- logger.Log(j.name, "").Errorf("Unable to create tmp dir with next error: %s", err)
			errs = multierror.Append(errs, err)
			continue
		}

		startTime := time.Now()
		if err = j.createTmpBackup(logCh, tmpBackupFile, tgt); err != nil {
			j.SetOfsMetrics(ofsPart, map[string]float64{
				metrics.BackupTime: float64(time.Since(startTime).Nanoseconds() / 1e6),
			})
			logCh <- logger.Log(j.name, "").Errorf("Unable to create temp backups %s", tmpBackupFile)
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

func (j *job) createTmpBackup(logCh chan logger.LogRecord, tmpBackupFile string, target target) error {
	var errs *multierror.Error

	backupWriter, err := targz.GetGZipFileWriter(tmpBackupFile, target.gzip, j.diskRateLimit)
	if err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to create tmp file. Error: %s", err)
		errs = multierror.Append(errs, err)
		return errs
	}
	defer func() { _ = backupWriter.Close() }()

	if target.isSlave {
		_, err = target.connect.Exec("STOP SLAVE")
		if err != nil {
			logCh <- logger.Log(j.name, "").Errorf("Unable to stop slave. Error: %s", err)
			errs = multierror.Append(errs, err)
			return errs
		}
		logCh <- logger.Log(j.name, "").Infof("Slave stopped")
		defer func() {
			_, err = target.connect.Exec("START SLAVE")
			if err != nil {
				logCh <- logger.Log(j.name, "").Errorf("Unable to start slave. Error: %s", err)
				errs = multierror.Append(errs, err)
			} else {
				logCh <- logger.Log(j.name, "").Infof("Slave started")
			}
		}()
	}

	authFile, err := files.CreateTmpMysqlAuthFile(target.authFile)
	if err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Failed to create tmp auth file. Error: %s", err)
		errs = multierror.Append(errs, err)
		return errs.ErrorOrNil()
	}
	defer func() {
		if err = files.DeleteTmpMysqlAuthFile(authFile); err != nil {
			logCh <- logger.Log(j.name, "").Errorf("Failed to delete tmp auth file. Error: %s", err)
		}
	}()

	var args []string
	// define command args with auth options
	args = append(args, "--defaults-file="+authFile)
	// add tables exclude
	if len(target.ignoreTables) > 0 {
		args = append(args, target.ignoreTables...)
	}
	// add extra dump cmd options
	if len(target.extraKeys) > 0 {
		args = append(args, target.extraKeys...)
	}
	// add db name
	args = append(args, target.dbName)

	var stderr bytes.Buffer
	cmd := exec.Command("mysqldump", args...)
	cmd.Stdout = backupWriter
	cmd.Stderr = &stderr

	logCh <- logger.Log(j.name, "").Debugf("Dump cmd: %s", cmd.String())

	if err = cmd.Start(); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to start mysqldump. Error: %s", err)
		errs = multierror.Append(errs, err)
		return errs
	}
	logCh <- logger.Log(j.name, "").Infof("Starting a `%s` dump", target.dbName)

	if err = cmd.Wait(); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to dump `%s`. Error: %s", target.dbName, stderr.String())
		errs = multierror.Append(errs, err)
		return errs
	}

	logCh <- logger.Log(j.name, "").Infof("Dump of `%s` completed", target.dbName)

	return errs.ErrorOrNil()
}

func (j *job) Close() error {
	for _, tgt := range j.targets {
		_ = tgt.connect.Close()
	}
	for _, st := range j.storages {
		_ = st.Close()
	}
	return nil
}
