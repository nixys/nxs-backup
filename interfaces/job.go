package interfaces

import (
	"nxs-backup/modules/logger"
)

type Job interface {
	GetName() string
	GetTempDir() string
	GetType() string
	GetTargetOfsList() []string
	GetStoragesCount() int
	GetDumpObjects() map[string]DumpObject
	SetDumpObjectDelivered(ofs string)
	IsBackupSafety() bool
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
