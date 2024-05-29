package psql_basebackup

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/nixys/nxs-backup/modules/metrics"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"time"

	"github.com/nixys/nxs-backup/ds/psql_connect"
	"github.com/nixys/nxs-backup/interfaces"
	"github.com/nixys/nxs-backup/misc"
	"github.com/nixys/nxs-backup/modules/backend/exec_cmd"
	"github.com/nixys/nxs-backup/modules/backend/targz"
	"github.com/nixys/nxs-backup/modules/logger"
)

type job struct {
	name             string
	tmpDir           string
	needToMakeBackup bool
	safetyBackup     bool
	deferredCopying  bool
	storages         interfaces.Storages
	targets          map[string]target
	dumpedObjects    map[string]interfaces.DumpObject
	metrics          *metrics.Data
}

type target struct {
	connUrl   *url.URL
	extraKeys []string
	gzip      bool
	metrics   map[string]float64
}

type JobParams struct {
	Name             string
	TmpDir           string
	NeedToMakeBackup bool
	SafetyBackup     bool
	DeferredCopying  bool
	Storages         interfaces.Storages
	Sources          []SourceParams
	Metrics          *metrics.Data
}

type SourceParams struct {
	Name          string
	ConnectParams psql_connect.Params
	ExtraKeys     []string
	Gzip          bool
	IsSlave       bool
}

func Init(jp JobParams) (interfaces.Job, error) {

	// check if mysqldump available
	if _, err := exec_cmd.Exec("pg_basebackup", "--version"); err != nil {
		return nil, fmt.Errorf("Job `%s` init failed. Can't check `pg_basebackup` version. Please cinstall `pg_basebackup`. Error: %s ", jp.Name, err)
	}
	// check if tar and gzip available
	if _, err := exec_cmd.Exec("tar", "--version"); err != nil {
		return nil, fmt.Errorf("Job `%s` init failed. Can't check `tar` version. Please install `tar`. Error: %s ", jp.Name, err)
	}

	j := &job{
		name:             jp.Name,
		tmpDir:           jp.TmpDir,
		needToMakeBackup: jp.NeedToMakeBackup,
		safetyBackup:     jp.SafetyBackup,
		deferredCopying:  jp.DeferredCopying,
		storages:         jp.Storages,
		targets:          make(map[string]target),
		dumpedObjects:    make(map[string]interfaces.DumpObject),
		metrics:          jp.Metrics,
	}

	for _, src := range jp.Sources {

		for _, key := range src.ExtraKeys {
			if matched, _ := regexp.MatchString(`(-D|--pgdata=)`, key); matched {
				return nil, fmt.Errorf("Job `%s` init failed. Forbidden usage \"--pgdata|-D\" parameter as extra_keys for `postgresql_basebackup` jobs type ", jp.Name)
			}
		}

		connUrl := psql_connect.GetConnUrl(src.ConnectParams)
		conn, err := psql_connect.GetConnect(connUrl)
		if err != nil {
			return nil, fmt.Errorf("Job `%s` init failed. PSQL connect error: %s ", jp.Name, err)
		}
		if err = conn.Ping(); err != nil {
			return nil, fmt.Errorf("Job `%s` init failed. PSQL ping check error: %s ", jp.Name, err)
		}
		_ = conn.Close()

		j.targets[src.Name] = target{
			extraKeys: src.ExtraKeys,
			gzip:      src.Gzip,
			connUrl:   connUrl,
			metrics:   make(map[string]float64),
		}
	}

	return j, nil
}

func (j *job) FillMetrics(ofs string, metrics map[string]float64) {
	for m, v := range metrics {
		j.targets[ofs].metrics[m] = v
	}
}

func (j *job) ExportMetrics() {
	for ofs, t := range j.targets {
		j.metrics.AddTargetMetric(metrics.TargetData{
			JobName: j.name,
			JobType: j.GetType(),
			Source:  ofs,
			Target:  "",
			Values:  t.metrics,
		})
	}
}

func (j *job) GetName() string {
	return j.name
}

func (j *job) GetTempDir() string {
	return j.tmpDir
}

func (j *job) GetType() string {
	return "postgresql_basebackup"
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

		tmpBackupFile := misc.GetFileFullPath(tmpDir, ofsPart, "tar", "", tgt.gzip)
		err := os.MkdirAll(path.Dir(tmpBackupFile), os.ModePerm)
		if err != nil {
			logCh <- logger.Log(j.name, "").Errorf("Unable to create tmp dir with next error: %s", err)
			errs = multierror.Append(errs, err)
			continue
		}

		startTime := time.Now()
		if err = j.createTmpBackup(logCh, tmpBackupFile, ofsPart, tgt); err != nil {
			j.FillMetrics(ofsPart, map[string]float64{
				"backup_ok":   float64(0),
				"backup_time": float64(time.Since(startTime).Nanoseconds() / 1e6),
			})
			logCh <- logger.Log(j.name, "").Errorf("Failed to create temp backups %s", tmpBackupFile)
			errs = multierror.Append(errs, err)
			continue
		}
		fileInfo, _ := os.Stat(tmpBackupFile)
		j.FillMetrics(ofsPart, map[string]float64{
			"backup_ok":   float64(1),
			"backup_time": float64(time.Since(startTime).Nanoseconds() / 1e6),
			"size":        float64(fileInfo.Size()),
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

func (j *job) createTmpBackup(logCh chan logger.LogRecord, tmpBackupFile, tgtName string, tgt target) error {

	var stderr, stdout bytes.Buffer

	tmpBasebackupPath := path.Join(path.Dir(tmpBackupFile), "pg_basebackup_"+tgtName+"_"+misc.GetDateTimeNow(""))

	var args []string
	// define command args
	// add extra dump cmd options
	if len(tgt.extraKeys) > 0 {
		args = append(args, tgt.extraKeys...)
	}
	// add db connect
	args = append(args, "--dbname="+tgt.connUrl.String())
	// add data catalog path
	args = append(args, "--pgdata="+tmpBasebackupPath)

	cmd := exec.Command("pg_basebackup", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logCh <- logger.Log(j.name, "").Debugf("Dump cmd: %s", cmd.String())

	if err := cmd.Start(); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to start pg_basebackup. Error: %s", err)
		return err
	}
	logCh <- logger.Log(j.name, "").Infof("Starting to dump `%s` source", tgtName)

	if err := cmd.Wait(); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to make dump `%s`. Error: %s", tgtName, stderr.String())
		return err
	}

	if err := targz.Tar(tmpBasebackupPath, tmpBackupFile, false, tgt.gzip, false, nil); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to make tar: %s", err)
		if serr, ok := err.(targz.Error); ok {
			logCh <- logger.Log(j.name, "").Debugf("STDERR: %s", serr.Stderr)
		}
		return err
	}
	_ = os.RemoveAll(tmpBasebackupPath)

	logCh <- logger.Log(j.name, "").Infof("Dumping of source `%s` completed", tgtName)

	return nil
}

func (j *job) Close() error {
	for _, st := range j.storages {
		_ = st.Close()
	}
	return nil
}
