package metrics

import (
	"errors"
	"io"
	"os"
	"time"

	"github.com/vmihailenco/msgpack"
)

const (
	AccessRetry = 3
	MetricsFile = "/tmp/nxs-backup.metrics"
)

type Data struct {
	Project             string
	Server              string
	NewVersionAvailable float64

	Job map[string]JobData
}

type JobData struct {
	JobName       string
	JobType       string
	TargetMetrics map[string]TargetData
}

type TargetData struct {
	Source string
	Target string
	Values map[string]float64
}

type Opts struct {
	Project             string
	Server              string
	NewVersionAvailable float64
}

func InitMetrics(opts Opts) (data, oldMetrics *Data, err error) {
	oldMetrics, err = ReadFile()
	if err != nil {
		return
	}

	data = &Data{
		Project:             opts.Project,
		Server:              opts.Server,
		NewVersionAvailable: opts.NewVersionAvailable,
		Job:                 make(map[string]JobData),
	}

	return
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
	f, err := os.OpenFile(MetricsFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	enc := msgpack.NewEncoder(f)
	return enc.Encode(md)
}

func ReadFile() (*Data, error) {
	var (
		f   *os.File
		d   Data
		err error
	)

	retry := AccessRetry
	for retry > 0 {
		f, err = os.OpenFile(MetricsFile, os.O_RDONLY|os.O_CREATE, 0600)
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
