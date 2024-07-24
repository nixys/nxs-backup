package external

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/nixys/nxs-backup/modules/backend/targz"
	"os"
	"os/exec"
	"time"

	"github.com/nixys/nxs-backup/interfaces"
	"github.com/nixys/nxs-backup/misc"
	"github.com/nixys/nxs-backup/modules/logger"
	"github.com/nixys/nxs-backup/modules/metrics"
)

type job struct {
	needToMakeBackup bool
	gzip             bool
	safetyBackup     bool
	skipBackupRotate bool // deprecated
	diskRateLimit    int64
	name             string
	appMetrics       *metrics.Data
	dumpCmd          string
	args             []string
	envs             map[string]string
	storages         interfaces.Storages
	dumpedObjects    map[string]interfaces.DumpObject
}

type JobParams struct {
	NeedToMakeBackup bool
	Gzip             bool
	SafetyBackup     bool
	SkipBackupRotate bool // deprecated
	DiskRateLimit    int64
	Name             string
	Metrics          *metrics.Data
	DumpCmd          string
	Args             []string
	Envs             map[string]string
	Storages         interfaces.Storages
}

func Init(jp JobParams) (interfaces.Job, error) {

	j := job{
		name:             jp.Name,
		dumpCmd:          jp.DumpCmd,
		args:             jp.Args,
		envs:             jp.Envs,
		needToMakeBackup: jp.NeedToMakeBackup,
		gzip:             jp.Gzip,
		safetyBackup:     jp.SafetyBackup,
		skipBackupRotate: jp.SkipBackupRotate,
		diskRateLimit:    jp.DiskRateLimit,
		storages:         jp.Storages,
		dumpedObjects:    make(map[string]interfaces.DumpObject),
		appMetrics: jp.Metrics.RegisterJob(
			metrics.JobData{
				JobName:       jp.Name,
				JobType:       misc.External,
				TargetMetrics: make(map[string]metrics.TargetData),
			},
		),
	}

	j.appMetrics.Job[j.name].TargetMetrics[jp.Name] = metrics.TargetData{
		Source: "",
		Target: "",
		Values: make(map[string]float64),
	}

	return &j, nil
}

func (j *job) SetOfsMetrics(_ string, metrics map[string]float64) {
	for m, v := range metrics {
		j.appMetrics.Job[j.name].TargetMetrics[j.name].Values[m] = v
	}
}

func (j *job) GetName() string {
	return j.name
}

func (j *job) GetTempDir() string {
	return ""
}

func (j *job) GetType() misc.BackupType {
	return misc.External
}

func (j *job) GetTargetOfsList() []string {
	return []string{j.name}
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
	if j.skipBackupRotate {
		logCh <- logger.Log(j.name, "").Debugf("Backup rotate skipped by config.")
		return nil
	}
	return j.storages.DeleteOldBackups(logCh, j, ofsPath)
}

func (j *job) CleanupTmpData() error {
	return j.storages.CleanupTmpData(j)
}

func (j *job) DoBackup(logCh chan logger.LogRecord, _ string) (err error) {

	var stderr, stdout bytes.Buffer

	startTime := time.Now()

	j.SetOfsMetrics("", map[string]float64{
		metrics.BackupOk:        float64(0),
		metrics.BackupTime:      float64(0),
		metrics.DeliveryOk:      float64(0),
		metrics.DeliveryTime:    float64(0),
		metrics.BackupSize:      float64(0),
		metrics.BackupTimestamp: float64(startTime.Unix()),
	})

	defer func() {
		if err != nil {
			logCh <- logger.Log(j.name, "").Error("Failed to create temp backup.")
		}
	}()

	cmd := exec.Command(j.dumpCmd, j.args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if len(j.envs) > 0 {
		var envs []string
		for k, v := range j.envs {
			envs = append(envs, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = envs
	}

	logCh <- logger.Log(j.name, "").Debugf("Dump cmd: %s", cmd.String())

	logCh <- logger.Log(j.name, "").Infof("Starting of `%s`", j.dumpCmd)
	if err = cmd.Run(); err != nil {
		j.SetOfsMetrics("", map[string]float64{
			metrics.BackupTime: float64(time.Since(startTime).Nanoseconds() / 1e6),
		})
		logCh <- logger.Log(j.name, "").Errorf("Unable to finish `%s`. Error: %s", j.dumpCmd, err)
		logCh <- logger.Log(j.name, "").Debugf("STDOUT: %s", stdout.String())
		logCh <- logger.Log(j.name, "").Debugf("STDERR: %s", stderr.String())
		return err
	}
	j.SetOfsMetrics("", map[string]float64{
		metrics.BackupOk:   float64(1),
		metrics.BackupTime: float64(time.Since(startTime).Nanoseconds() / 1e6),
	})

	logCh <- logger.Log(j.name, "").Infof("Dumping completed")
	logCh <- logger.Log(j.name, "").Debugf("STDOUT: %s", stdout.String())

	if j.skipBackupRotate {
		return
	}

	var out struct {
		FullPath string `json:"full_path"`
	}
	err = json.Unmarshal(stdout.Bytes(), &out)
	if err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to parse execution result. Error: %s", err)
		return err
	}
	tmpBackupPath := out.FullPath
	if j.gzip {
		newTmpBackup := tmpBackupPath + ".gz"
		if err = targz.GZip(tmpBackupPath, newTmpBackup, j.diskRateLimit); err != nil {
			logCh <- logger.Log(j.name, "").Errorf("Unable to gzip tmp backup: %s", err)
			return err
		}
		_ = os.RemoveAll(tmpBackupPath)
		tmpBackupPath = newTmpBackup
	}

	logCh <- logger.Log(j.name, "").Debugf("Created temp backup %s.", tmpBackupPath)

	j.dumpedObjects[j.name] = interfaces.DumpObject{TmpFile: tmpBackupPath}
	fileInfo, _ := os.Stat(tmpBackupPath)
	j.SetOfsMetrics("", map[string]float64{
		metrics.BackupSize: float64(fileInfo.Size()),
	})

	return j.storages.Delivery(logCh, j)
}

func (j *job) Close() error {
	for _, st := range j.storages {
		_ = st.Close()
	}
	return nil
}
