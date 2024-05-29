package desc_files

import (
	"fmt"
	"github.com/nixys/nxs-backup/modules/metrics"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/mb0/glob"

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
	path        string
	gzip        bool
	saveAbsPath bool
	excludes    []string
	metrics     map[string]float64
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
	Name        string
	Targets     []string
	Excludes    []string
	Gzip        bool
	SaveAbsPath bool
}

func Init(jp JobParams) (interfaces.Job, error) {

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

		for _, targetPattern := range src.Targets {

			for strings.HasSuffix(targetPattern, "/") {
				targetPattern = strings.TrimSuffix(targetPattern, "/")
			}

			targetOfsList, err := filepath.Glob(targetPattern)
			if err != nil {
				return nil, fmt.Errorf("Job `%s` init failed. Unable to process pattern: %s. Error: %s. ", jp.Name, targetPattern, err)
			}

			for _, ofs := range targetOfsList {
				var excludes []string

				skipOfs := false
				for _, pattern := range src.Excludes {
					match, err := glob.Match(pattern, ofs)
					if err != nil {
						return nil, fmt.Errorf("Job `%s` init failed. Unable to process pattern: %s. Error: %s. ", jp.Name, pattern, err)
					}
					if match {
						skipOfs = true
					}

					excludes = append(excludes, pattern)
				}

				if !skipOfs {
					ofsPart := src.Name + "/" + misc.GetOfsPart(targetPattern, ofs)

					j.targets[ofsPart] = target{
						path:        ofs,
						gzip:        src.Gzip,
						saveAbsPath: src.SaveAbsPath,
						excludes:    excludes,
						metrics:     make(map[string]float64),
					}
				}
			}
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
	for ofsPart, t := range j.targets {
		ofs := strings.Split(ofsPart, "/")
		j.metrics.AddTargetMetric(metrics.TargetData{
			JobName: j.name,
			JobType: j.GetType(),
			Source:  ofs[0],
			Target:  ofs[1],
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
	return "desc_files"
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

func (j *job) DeleteOldBackups(logCh chan logger.LogRecord, ofsPath string) error {
	logCh <- logger.Log(j.name, "").Debugf("Starting rotate outdated backups.")
	return j.storages.DeleteOldBackups(logCh, j, ofsPath)
}

func (j *job) CleanupTmpData() error {
	return j.storages.CleanupTmpData(j)
}

func (j *job) NeedToMakeBackup() bool {
	return j.needToMakeBackup
}

func (j *job) NeedToUpdateIncMeta() bool {
	return false
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
		if err = targz.Tar(tgt.path, tmpBackupFile, false, tgt.gzip, tgt.saveAbsPath, tgt.excludes); err != nil {
			j.FillMetrics(ofsPart, map[string]float64{
				"backup_ok":   float64(0),
				"backup_time": float64(time.Since(startTime).Nanoseconds() / 1e6),
			})
			logCh <- logger.Log(j.name, "").Errorf("Failed to create temp backup %s", tmpBackupFile)
			logCh <- logger.Log(j.name, "").Error(err)
			if serr, ok := err.(targz.Error); ok {
				logCh <- logger.Log(j.name, "").Debugf("STDERR: %s", serr.Stderr)
			}
			errs = multierror.Append(errs, err)
			continue
		}
		fileInfo, _ := os.Stat(tmpBackupFile)
		j.FillMetrics(ofsPart, map[string]float64{
			"backup_ok":   float64(1),
			"backup_time": float64(time.Since(startTime).Nanoseconds() / 1e6),
			"size":        float64(fileInfo.Size()),
		})

		logCh <- logger.Log(j.name, "").Debugf("Created temp backup %s", tmpBackupFile)

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

func (j *job) Close() error {
	for _, st := range j.storages {
		_ = st.Close()
	}
	return nil
}
