package mysql_xtrabackup

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"os"
	"os/exec"
	"path"
	"strings"

	"nxs-backup/interfaces"
	"nxs-backup/misc"
	"nxs-backup/modules/backend/exec_cmd"
	"nxs-backup/modules/backend/targz"
	"nxs-backup/modules/connectors/mysql_connect"
	"nxs-backup/modules/logger"
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
}

type target struct {
	extraKeys       []string
	authFile        string
	ignoreDatabases string
	gzip            bool
	isSlave         bool
	prepare         bool
}

type JobParams struct {
	Name             string
	TmpDir           string
	NeedToMakeBackup bool
	SafetyBackup     bool
	DeferredCopying  bool
	Storages         interfaces.Storages
	Sources          []SourceParams
}

type SourceParams struct {
	Name          string
	ConnectParams mysql_connect.Params
	TargetDBs     []string
	Excludes      []string
	ExtraKeys     []string
	Gzip          bool
	IsSlave       bool
	Prepare       bool
}

func Init(jp JobParams) (interfaces.Job, error) {

	// check if xtrabackup available
	if _, err := exec_cmd.Exec("xtrabackup", "--version"); err != nil {
		return nil, fmt.Errorf("Job `%s` init failed. Can't to check `xtrabackup` version. Please install `xtrabackup`. Error: %s ", jp.Name, err)
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
	}

	for _, src := range jp.Sources {

		_, authFile, err := mysql_connect.GetConnectAndCnfFile(src.ConnectParams, "xtrabackup")
		if err != nil {
			return nil, err
		}

		var ignoreDBs string
		if len(src.Excludes) > 0 {
			ignoreDBs = "--databases-exclude="
			for _, excl := range src.Excludes {
				ignoreDBs += excl + " "
			}
		}
		ignoreDBs = strings.TrimSuffix(ignoreDBs, " ")

		j.targets[src.Name] = target{
			authFile:        authFile,
			ignoreDatabases: ignoreDBs,
			extraKeys:       src.ExtraKeys,
			gzip:            src.Gzip,
			isSlave:         src.IsSlave,
			prepare:         src.Prepare,
		}
	}

	return j, nil
}

func (j *job) GetName() string {
	return j.name
}

func (j *job) GetTempDir() string {
	return j.tmpDir
}

func (j *job) GetType() string {
	return "mysql_xtrabackup"
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

		if err = j.createTmpBackup(logCh, tmpBackupFile, ofsPart, tgt); err != nil {
			logCh <- logger.Log(j.name, "").Errorf("Failed to create temp backups %s", tmpBackupFile)
			errs = multierror.Append(errs, err)
			continue
		} else {
			logCh <- logger.Log(j.name, "").Debugf("Created temp backups %s", tmpBackupFile)
		}

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

func (j *job) createTmpBackup(logCh chan logger.LogRecord, tmpBackupFile, tgtName string, target target) error {

	var (
		stderr, stdout          bytes.Buffer
		backupArgs, prepareArgs []string
	)

	tmpXtrabackupPath := path.Join(path.Dir(tmpBackupFile), "xtrabackup_"+tgtName+"_"+misc.GetDateTimeNow(""))

	// define commands args with auth options
	backupArgs = append(backupArgs, "--defaults-extra-file="+target.authFile)
	prepareArgs = backupArgs
	// add backup options
	backupArgs = append(backupArgs, "--backup", "--target-dir="+tmpXtrabackupPath)
	if target.ignoreDatabases != "" {
		backupArgs = append(backupArgs, target.ignoreDatabases)
	}
	if target.isSlave {
		backupArgs = append(backupArgs, "--safe-slave-backup")
	}
	// add extra backup options
	if len(target.extraKeys) > 0 {
		backupArgs = append(backupArgs, target.extraKeys...)
	}

	cmd := exec.Command("xtrabackup", backupArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logCh <- logger.Log(j.name, "").Debugf("Dump cmd: %s", cmd.String())

	if err := cmd.Start(); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to start xtrabackup. Error: %s", err)
		return err
	}
	logCh <- logger.Log(j.name, "").Infof("Starting `%s` dump", tgtName)

	if err := cmd.Wait(); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to dump `%s`. Error: %s", tgtName, err)
		logCh <- logger.Log(j.name, "").Error(stderr.String())
		return err
	}

	logCh <- logger.Log(j.name, "").Debugf("Exit code: %d", cmd.ProcessState.ExitCode())
	logCh <- logger.Log(j.name, "").Debugf("STDERR:\n%s", stderr.String())

	if cmd.ProcessState.ExitCode() != 0 {
		err := xtrabackupStatusErr(stderr.String())
		logCh <- logger.Log(j.name, "").Error(err)
		return err
	}

	stdout.Reset()
	stderr.Reset()

	if target.prepare {
		// add prepare options
		prepareArgs = append(prepareArgs, "--prepare", "--target-dir="+tmpXtrabackupPath)
		cmd = exec.Command("xtrabackup", prepareArgs...)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			logCh <- logger.Log(j.name, "").Errorf("Unable to prepare xtrabackup. Error: %s", err)
			logCh <- logger.Log(j.name, "").Error(stderr.String())
			return err
		}

		logCh <- logger.Log(j.name, "").Debugf("Exit code: %d", cmd.ProcessState.ExitCode())
		logCh <- logger.Log(j.name, "").Debugf("STDERR:\n%s", stderr.String())

		if cmd.ProcessState.ExitCode() != 0 {
			err := xtrabackupStatusErr(stderr.String())
			logCh <- logger.Log(j.name, "").Error(err)
			return err
		}
	}

	if err := targz.Tar(tmpXtrabackupPath, tmpBackupFile, false, target.gzip, false, nil); err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Unable to make tar: %s", err)
		if serr, ok := err.(targz.Error); ok {
			logCh <- logger.Log(j.name, "").Debugf("STDERR: %s", serr.Stderr)
		}
		return err
	}
	_ = os.RemoveAll(tmpXtrabackupPath)

	logCh <- logger.Log(j.name, "").Infof("Dump of `%s` completed", tgtName)

	return nil
}

func xtrabackupStatusErr(out string) error {
	return fmt.Errorf("xtrabackup finished not success. Please check result:\n%s", out)
}

func (j *job) Close() error {
	for _, st := range j.storages {
		_ = st.Close()
	}
	return nil
}
