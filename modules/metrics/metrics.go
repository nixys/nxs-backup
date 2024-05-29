package metrics

import (
	"os"
	"time"

	"github.com/vmihailenco/msgpack"
)

type Data struct {
	Project string
	Server  string

	NewVersionAvailable float64

	TargetMetrics []TargetData
}

type TargetData struct {
	JobName string
	JobType string
	Source  string
	Target  string
	Values  map[string]float64
}

func InitData(project, server string) *Data {
	return &Data{
		Project: project,
		Server:  server,
	}
}

func (md *Data) AddTargetMetric(td TargetData) {
	md.TargetMetrics = append(md.TargetMetrics, td)
}

func (md *Data) SaveFile() error {
	f, err := os.OpenFile("/tmp/nxs-backup.metrics", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
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

	retry := 3

	for retry > 0 {
		f, err = os.Open("/tmp/nxs-backup.metrics")
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
	return &d, err
}
