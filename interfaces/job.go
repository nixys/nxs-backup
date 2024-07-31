package interfaces

import (
	"github.com/nixys/nxs-backup/misc"
	"github.com/nixys/nxs-backup/modules/logger"
)

type JobTargets map[string]TargetsOnStorages

type Job interface {
	SetOfsMetrics(ofs string, metrics map[string]float64)
	GetName() string
	GetTempDir() string
	GetType() misc.BackupType
	GetTargetOfsList() []string
	GetStoragesCount() int
	GetDumpObjects() map[string]DumpObject
	SetDumpObjectDelivered(ofs string)
	IsBackupSafety() bool
	ListBackups() JobTargets
	NeedToMakeBackup() bool
	NeedToUpdateIncMeta() bool
	DoBackup(logCh chan logger.LogRecord, tmpDir string) error
	DeleteOldBackups(logCh chan logger.LogRecord, ofsPath string) error
	CleanupTmpData() error
	Close() error
}

type Jobs []Job

func (j Jobs) Close() error {
	for _, job := range j {
		_ = job.Close()
	}
	return nil
}

type DumpObject struct {
	TmpFile   string
	Delivered bool
}
