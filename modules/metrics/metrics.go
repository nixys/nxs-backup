package metrics

import (
	"errors"
	"io"
	"os"
	"time"

	"github.com/vmihailenco/msgpack"

	"github.com/nixys/nxs-backup/misc"
)

const (
	AccessRetry = 3

	BackupOk        = "backup_ok"
	BackupTime      = "backup_time"
	BackupSize      = "size"
	DeliveryOk      = "delivery_ok"
	DeliveryTime    = "delivery_time"
	UpdateAvailable = "update_available"
)

type Data struct {
	Project             string
	Server              string
	NewVersionAvailable float64

	Job map[string]JobData

	metricsFile string
}

type JobData struct {
	JobName       string
	JobType       misc.BackupType
	TargetMetrics map[string]TargetData
}

type TargetData struct {
	Source string
	Target string
	Values map[string]float64
}

type DataOpts struct {
	Project             string
	Server              string
	MetricsFile         string
	NewVersionAvailable float64
}

// InitData initializes the metrics data by reading from a file and
// creating a new Data instance with the provided options.
// It returns the old metrics data, the new initialized data, and any error encountered.
//
// TODO Remove the reading of old metrics for the save step.
func InitData(opts DataOpts) (data, oldMetrics *Data, err error) {
	oldMetrics, err = readFile(opts.MetricsFile)
	if err != nil {
		return
	}

	data = &Data{
		Project:             opts.Project,
		Server:              opts.Server,
		NewVersionAvailable: opts.NewVersionAvailable,
		Job:                 make(map[string]JobData),
		metricsFile:         opts.MetricsFile,
	}

	return
}

func (md *Data) MetricFilePath() string {
	return md.metricsFile
}

func (md *Data) GetMetrics(jobName string) JobData {
	if job, ok := md.Job[jobName]; ok {
		return job
	}
	return JobData{
		TargetMetrics: make(map[string]TargetData),
	}
}

func (md *Data) JobMetricsSet(jd JobData) {
	md.Job[jd.JobName] = jd
}

func (md *Data) SaveFile() error {
	f, err := os.OpenFile(md.metricsFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	enc := msgpack.NewEncoder(f)
	return enc.Encode(md)
}

func readFile(path string) (*Data, error) {
	var (
		f   *os.File
		d   Data
		err error
	)

	retry := AccessRetry
	for retry > 0 {
		f, err = os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0600)
		if err != nil {
			time.Sleep(1 * time.Second)
			retry--
		} else {
			break
		}
	}
	if err != nil {
		return &d, err
	}
	defer func() { _ = f.Close() }()

	dec := msgpack.NewDecoder(f)
	err = dec.Decode(&d)
	if errors.Is(err, io.EOF) {
		err = nil
		d.Job = make(map[string]JobData)
	}
	return &d, err
}
