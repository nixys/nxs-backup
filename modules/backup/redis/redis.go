package redis

import (
	"bytes"
	"fmt"
	"github.com/nixys/nxs-backup/modules/metrics"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/nixys/nxs-backup/ds/redis_connect"
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
	appMetrics       *metrics.Data
	jobMetrics       metrics.JobData
}

type target struct {
	dsn  string
	gzip bool
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
	OldMetrics       *metrics.Data
}

type SourceParams struct {
	Name          string
	ConnectParams redis_connect.Params
	Gzip          bool
}

func Init(jp JobParams) (interfaces.Job, error) {

	// check if redis-cli available
	_, err := exec_cmd.Exec("redis-cli", "--version")
	if err != nil {
		return nil, fmt.Errorf("Job `%s` init failed. Failed to check redis-cli version. Please check that `redis-cli` installed. Error: %s ", jp.Name, err)
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
		appMetrics:       jp.Metrics,
	}

	j.jobMetrics = metrics.JobData{
		JobName:       jp.Name,
		JobType:       j.GetType(),
		TargetMetrics: make(map[string]metrics.TargetData),
	}
	ojm := jp.OldMetrics.GetMetrics(jp.Name)

	for _, src := range jp.Sources {

		conn, dsn, err := redis_connect.GetConnectAndDSN(src.ConnectParams)
		if err != nil {
			return nil, fmt.Errorf("Job `%s` init failed. Redis connect error: %s ", jp.Name, err)
		}
		_ = conn.Close()

		j.targets[src.Name] = target{
			gzip: src.Gzip,
			dsn:  dsn,
		}
		if otm, ok := ojm.TargetMetrics[src.Name]; ok {
			j.jobMetrics.TargetMetrics[src.Name] = otm
		} else {
			j.jobMetrics.TargetMetrics[src.Name] = metrics.TargetData{
				Source: src.Name,
				Target: "",
				Values: make(map[string]float64),
			}
		}
	}

	j.ExportMetrics()
	return j, nil
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

func (j *job) GetType() string {
	return "redis"
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
			"backup_ok":     float64(0),
			"backup_time":   float64(0),
			"delivery_ok":   float64(0),
			"delivery_time": float64(0),
			"size":          float64(0),
		})

		tmpBackupFile := misc.GetFileFullPath(tmpDir, ofsPart, "rdb", "", tgt.gzip)
		err := os.MkdirAll(path.Dir(tmpBackupFile), os.ModePerm)
		if err != nil {
			logCh <- logger.Log(j.name, "").Errorf("Unable to create tmp dir with next error: %s", err)
			errs = multierror.Append(errs, err)
			continue
		}

		startTime := time.Now()
		if err = j.createTmpBackup(logCh, tmpBackupFile, ofsPart, tgt); err != nil {
			j.SetOfsMetrics(ofsPart, map[string]float64{
				"backup_time": float64(time.Since(startTime).Nanoseconds() / 1e6),
			})
			logCh <- logger.Log(j.name, "").Error("Failed to create temp backup.")
			errs = multierror.Append(errs, err)
			continue
		}
		fileInfo, _ := os.Stat(tmpBackupFile)
		j.SetOfsMetrics(ofsPart, map[string]float64{
			"backup_ok":   float64(1),
			"backup_time": float64(time.Since(startTime).Nanoseconds() / 1e6),
			"size":        float64(fileInfo.Size()),
		})
		logCh <- logger.Log(j.name, "").Debugf("Created temp backup %s", tmpBackupFile)

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

	tmpBackupRdb := strings.TrimSuffix(tmpBackupFile, ".gz")

	var args []string
	// define command args
	// add db connect
	args = append(args, "-u", tgt.dsn)
	// add data catalog path
	args = append(args, "--rdb", tmpBackupRdb)

	cmd := exec.Command("redis-cli", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logCh <- logger.Log(j.name, "").Debugf("Dump cmd: %s", cmd.String())

	if err := cmd.Start(); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to start redis-cli. Error: %s", err)
		return err
	}
	logCh <- logger.Log(j.name, "").Infof("Starting to dump `%s` source", tgtName)

	if err := cmd.Wait(); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to make dump `%s`. Error: %s", tgtName, stderr.String())
		return err
	}

	if tgt.gzip {
		if err := targz.GZip(tmpBackupRdb, tmpBackupFile); err != nil {
			logCh <- logger.Log(j.name, "").Errorf("Unable to archivate tmp backup: %s", err)
			return err
		}
		_ = os.RemoveAll(tmpBackupRdb)
	}

	logCh <- logger.Log(j.name, "").Infof("Dumping of source `%s` completed", tgtName)
	logCh <- logger.Log(j.name, "").Debugf("Created temp backup %s", tmpBackupFile)

	j.dumpedObjects[tgtName] = interfaces.DumpObject{TmpFile: tmpBackupFile}

	return nil
}

func (j *job) Close() error {
	for _, st := range j.storages {
		_ = st.Close()
	}
	return nil
}
