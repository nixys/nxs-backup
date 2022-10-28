package external

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"nxs-backup/interfaces"
	"nxs-backup/modules/logger"
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
}

func Init(jp JobParams) (interfaces.Job, error) {

	return &job{
		name:             jp.Name,
		dumpCmd:          jp.DumpCmd,
		args:             jp.Args,
		envs:             jp.Envs,
		needToMakeBackup: jp.NeedToMakeBackup,
		safetyBackup:     jp.SafetyBackup,
		skipBackupRotate: jp.SkipBackupRotate,
		storages:         jp.Storages,
		dumpedObjects:    make(map[string]interfaces.DumpObject),
	}, nil
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
	return []string{""}
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
	if j.skipBackupRotate {
		return nil
	}
	return j.storages.DeleteOldBackups(logCh, j, ofsPath)
}

func (j *job) CleanupTmpData() error {
	return j.storages.CleanupTmpData(j)
}

func (j *job) DoBackup(logCh chan logger.LogRecord, _ string) (err error) {

	var stderr, stdout bytes.Buffer

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

	if err = cmd.Start(); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to start %s. Error: %s", j.dumpCmd, err)
		return err
	}
	logCh <- logger.Log(j.name, "").Infof("Starting of `%s`", j.dumpCmd)

	if err = cmd.Wait(); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to finish `%s`. Error: %s", j.dumpCmd, stderr.String())
		return err
	}

	logCh <- logger.Log(j.name, "").Infof("Dumping completed")

	if j.skipBackupRotate {
		return
	}

	var out struct {
		FullPath string `json:"full_path"`
	}
	err = json.Unmarshal(stdout.Bytes(), &out)
	if err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to parse execution result. Error: %s", stderr.String())
		return err
	}

	logCh <- logger.Log(j.name, "").Debugf("Created temp backup %s.", out.FullPath)

	j.dumpedObjects[j.name] = interfaces.DumpObject{TmpFile: out.FullPath}

	return j.storages.Delivery(logCh, j)
}

func (j *job) Close() error {
	for _, st := range j.storages {
		_ = st.Close()
	}
	return nil
}
