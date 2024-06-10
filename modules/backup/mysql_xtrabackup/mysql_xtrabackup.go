package mysql_xtrabackup

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/hashicorp/go-multierror"
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
	appMetrics       *metrics.Data
	jobMetrics       metrics.JobData
}

type target struct {
	extraKeys       []string
	authFile        *ini.File
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
			JobType:       misc.MysqlXtrabackup,
			TargetMetrics: make(map[string]metrics.TargetData),
		},
	}

	ojm := jp.OldMetrics.GetMetrics(jp.Name)

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
	return misc.MysqlXtrabackup
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

		tmpBackupFile := misc.GetFileFullPath(tmpDir, ofsPart, "tar", "", tgt.gzip)
		err := os.MkdirAll(path.Dir(tmpBackupFile), os.ModePerm)
		if err != nil {
			logCh <- logger.Log(j.name, "").Errorf("Unable to create tmp dir with next error: %s", err)
			errs = multierror.Append(errs, err)
			continue
		}

		startTime := time.Now()
		if err = j.createTmpBackup(logCh, tmpBackupFile, ofsPart, tgt); err != nil {
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

func (j *job) createTmpBackup(logCh chan logger.LogRecord, tmpBackupFile, tgtName string, target target) error {

	var (
		stderr, stdout          bytes.Buffer
		backupArgs, prepareArgs []string
	)

	tmpXtrabackupPath := path.Join(path.Dir(tmpBackupFile), "xtrabackup_"+tgtName+"_"+misc.GetDateTimeNow(""))

	authFile, err := files.CreateTmpMysqlAuthFile(target.authFile)
	if err != nil {
		logCh <- logger.Log(j.name, "").Errorf("Failed to create tmp auth file. Error: %s", err)
		return err
	}
	defer func() {
		if err = files.DeleteTmpMysqlAuthFile(authFile); err != nil {
			logCh <- logger.Log(j.name, "").Errorf("Failed to delete tmp auth file. Error: %s", err)
		}
	}()

	// define commands args with auth options
	backupArgs = append(backupArgs, "--defaults-extra-file="+authFile)
	prepareArgs = backupArgs
	// add backup options
	backupArgs = append(backupArgs, "--backup", "--target-dir="+tmpXtrabackupPath)
	if target.ignoreDatabases != "" {
		backupArgs = append(backupArgs, target.ignoreDatabases)
	}
	if target.isSlave {
		backupArgs = append(backupArgs, "--safe-slave-backup")
	}
	if j.diskRateLimit != 0 {
		rateLim := j.diskRateLimit / units.MB
		if rateLim < 1 {
			rateLim = 1
		}
		// This option limits the number of chunks copied per second. The chunk size is 10 MB.
		backupArgs = append(backupArgs, "--throttle="+strconv.FormatInt(rateLim, 10))
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

	if err := targz.Tar(tmpXtrabackupPath, tmpBackupFile, false, target.gzip, false, j.diskRateLimit, nil); err != nil {
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
