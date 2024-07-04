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
	BackupTimestamp = "backup_timestamp"
	BackupSize      = "size"
	DeliveryOk      = "delivery_ok"
	DeliveryTime    = "delivery_time"
	UpdateAvailable = "update_available"
)

type Data struct {
	Project             string
	Server              string
	NewVersionAvailable float64
	Enabled             bool

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
	Enabled             bool
}

// InitData initializes the metrics data by
// creating a new Data instance with the provided options.
func InitData(opts DataOpts) *Data {
	return &Data{
		Project:             opts.Project,
		Server:              opts.Server,
		NewVersionAvailable: opts.NewVersionAvailable,
		Enabled:             opts.Enabled,
		Job:                 make(map[string]JobData),
		metricsFile:         opts.MetricsFile,
	}
}

func (md *Data) MetricFilePath() string {
	return md.metricsFile
}

func (md *Data) RegisterJob(jd JobData) *Data {
	md.Job[jd.JobName] = jd
	return md
}

func (md *Data) SaveFile() error {
	//skip if metrics disabled
	if !md.Enabled {
		return nil
	}

	od, err := readFile(md.metricsFile)
	if err != nil {
		return err
	}

	// reuse old metrics for not run jobs
	for jobName, job := range od.Job {
		if _, ok := md.Job[jobName]; !ok {
			md.Job[jobName] = job
		}
	}

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
