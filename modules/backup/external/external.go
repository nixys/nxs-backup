package external

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/nixys/nxs-backup/interfaces"
	"github.com/nixys/nxs-backup/modules/logger"
	"github.com/nixys/nxs-backup/modules/metrics"
	"os"
	"os/exec"
	"time"
)

type job struct {
	name             string
	dumpCmd          string
	args             []string
	envs             map[string]string
	needToMakeBackup bool
	safetyBackup     bool
	skipBackupRotate bool
	storages         interfaces.Storages
	dumpedObjects    map[string]interfaces.DumpObject
	appMetrics       *metrics.Data
	jobMetrics       metrics.JobData
}

type JobParams struct {
	Name             string
	DumpCmd          string
	Args             []string
	Envs             map[string]string
	NeedToMakeBackup bool
	SafetyBackup     bool
	SkipBackupRotate bool
	Storages         interfaces.Storages
	Metrics          *metrics.Data
	OldMetrics       *metrics.Data
}

func Init(jp JobParams) (interfaces.Job, error) {

	j := job{
		name:             jp.Name,
		dumpCmd:          jp.DumpCmd,
		args:             jp.Args,
		envs:             jp.Envs,
		needToMakeBackup: jp.NeedToMakeBackup,
		safetyBackup:     jp.SafetyBackup,
		skipBackupRotate: jp.SkipBackupRotate,
		storages:         jp.Storages,
		dumpedObjects:    make(map[string]interfaces.DumpObject),
		appMetrics:       jp.Metrics,
	}
	j.jobMetrics = metrics.JobData{
		JobName:       jp.Name,
		JobType:       j.GetType(),
		TargetMetrics: make(map[string]metrics.TargetData),
	}

	ojm := jp.OldMetrics.GetMetrics(jp.Name)
	if otm, ok := ojm.TargetMetrics[jp.Name]; ok {
		j.jobMetrics.TargetMetrics[jp.Name] = otm
	} else {
		j.jobMetrics.TargetMetrics[jp.Name] = metrics.TargetData{
			Source: "",
			Target: "",
			Values: make(map[string]float64),
		}
	}

	j.ExportMetrics()
	return &j, nil
}

func (j *job) SetOfsMetrics(_ string, metrics map[string]float64) {
	for m, v := range metrics {
		j.jobMetrics.TargetMetrics[j.name].Values[m] = v
	}
}

func (j *job) ExportMetrics() {
	j.appMetrics.JobMetricsSet(j.jobMetrics)
}

func (j *job) GetName() string {
	return j.name
}

func (j *job) GetTempDir() string {
	return ""
}

func (j *job) GetType() string {
	return "external"
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

	j.SetOfsMetrics("", map[string]float64{
		"backup_ok":     float64(0),
		"backup_time":   float64(0),
		"delivery_ok":   float64(0),
		"delivery_time": float64(0),
		"size":          float64(0),
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
	startTime := time.Now()
	if err = cmd.Run(); err != nil {
		j.SetOfsMetrics("", map[string]float64{
			"backup_time": float64(time.Since(startTime).Nanoseconds() / 1e6),
		})
		logCh <- logger.Log(j.name, "").Errorf("Unable to finish `%s`. Error: %s", j.dumpCmd, err)
		logCh <- logger.Log(j.name, "").Debugf("STDOUT: %s", stdout.String())
		logCh <- logger.Log(j.name, "").Debugf("STDERR: %s", stderr.String())
		return err
	}
	j.SetOfsMetrics("", map[string]float64{
		"backup_ok":   float64(1),
		"backup_time": float64(time.Since(startTime).Nanoseconds() / 1e6),
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

	logCh <- logger.Log(j.name, "").Debugf("Created temp backup %s.", out.FullPath)

	j.dumpedObjects[j.name] = interfaces.DumpObject{TmpFile: out.FullPath}
	fileInfo, _ := os.Stat(out.FullPath)
	j.SetOfsMetrics("", map[string]float64{
		"size": float64(fileInfo.Size()),
	})

	return j.storages.Delivery(logCh, j)
}

func (j *job) Close() error {
	for _, st := range j.storages {
		_ = st.Close()
	}
	return nil
}
