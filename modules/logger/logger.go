package logger

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type LogRecord struct {
	Level       logrus.Level
	JobName     string
	StorageName string
	Message     string
}

func (r LogRecord) Debugf(format string, args ...interface{}) LogRecord {
	r.Level = logrus.DebugLevel
	r.Message = fmt.Sprintf(format, args...)
	return r
}

func (r LogRecord) Debug(args ...interface{}) LogRecord {
	r.Level = logrus.DebugLevel
	r.Message = fmt.Sprint(args...)
	return r
}

func (r LogRecord) Infof(format string, args ...interface{}) LogRecord {
	r.Level = logrus.InfoLevel
	r.Message = fmt.Sprintf(format, args...)
	return r
}

func (r LogRecord) Info(args ...interface{}) LogRecord {
	r.Level = logrus.InfoLevel
	r.Message = fmt.Sprint(args...)
	return r
}

func (r LogRecord) Warnf(format string, args ...interface{}) LogRecord {
	r.Level = logrus.WarnLevel
	r.Message = fmt.Sprintf(format, args...)
	return r
}

func (r LogRecord) Warn(args ...interface{}) LogRecord {
	r.Level = logrus.WarnLevel
	r.Message = fmt.Sprint(args...)
	return r
}

func (r LogRecord) Errorf(format string, args ...interface{}) LogRecord {
	r.Level = logrus.ErrorLevel
	r.Message = fmt.Sprintf(format, args...)
	return r
}

func (r LogRecord) Error(args ...interface{}) LogRecord {
	r.Level = logrus.ErrorLevel
	r.Message = fmt.Sprint(args...)
	return r
}

func Log(jobName, storageName string) LogRecord {
	return LogRecord{
		JobName:     jobName,
		StorageName: storageName,
	}
}

func WriteLog(logger *logrus.Logger, log LogRecord) {
	logger.WithFields(logrus.Fields{"storage": log.StorageName, "job": log.JobName}).Log(log.Level, log.Message)
}
