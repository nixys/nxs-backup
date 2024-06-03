package metrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Exporter struct {
	metrics map[string]*prometheus.Desc
	ctx     context.Context
	log     *logrus.Logger
}

type InitSettings struct {
	Log *logrus.Logger
}

func Init(s InitSettings) *Exporter {

	metrics := map[string]*prometheus.Desc{
		"size": prometheus.NewDesc(
			prometheus.BuildFQName("backup", "file", "size"),
			"Backup file size",
			[]string{"project", "server", "job_name", "job_type", "source", "target"}, nil,
		),
		"backup_ok": prometheus.NewDesc(
			prometheus.BuildFQName("backup", "collection", "success"),
			"Backup finished successfully",
			[]string{"project", "server", "job_name", "job_type", "source", "target"}, nil,
		),
		"backup_time": prometheus.NewDesc(
			prometheus.BuildFQName("backup", "collection", "time"),
			"Backup collection time",
			[]string{"project", "server", "job_name", "job_type", "source", "target"}, nil,
		),
		"delivery_ok": prometheus.NewDesc(
			prometheus.BuildFQName("backup", "delivery", "success"),
			"Backup delivery finished successfully",
			[]string{"project", "server", "job_name", "job_type", "source", "target"}, nil,
		),
		"delivery_time": prometheus.NewDesc(
			prometheus.BuildFQName("backup", "delivery", "time"),
			"Backup delivering time",
			[]string{"project", "server", "job_name", "job_type", "source", "target"}, nil,
		),
		"update": prometheus.NewDesc(
			prometheus.BuildFQName("", "update", "available"),
			"A new version of nxs-backup is available",
			[]string{"project", "server"}, nil,
		),
	}

	return &Exporter{
		metrics: metrics,
		log:     s.Log,
	}
}

func (e *Exporter) ContextSet(c context.Context) {
	e.ctx = c
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.metrics {
		ch <- m
	}
}

// Collect function, called on by Prometheus Client library
// This function is called when a scrape is performed on the /metrics page
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {

	data, err := ReadFile()
	if err != nil {
		e.log.Warnf("Failed to read metric file: %v", err)
		return
	}

	for _, j := range data.Job {
		for _, t := range j.TargetMetrics {
			for k, v := range t.Values {
				d, err := prometheus.NewConstMetric(
					e.metrics[k],
					prometheus.GaugeValue,
					v,
					data.Project,
					data.Server,
					j.JobName,
					j.JobType,
					t.Source,
					t.Target,
				)
				if err != nil {
					e.log.Warnf("Failed to export prometheus metric: %v", err)
					continue
				}
				ch <- d
			}
		}

	}

	d, err := prometheus.NewConstMetric(
		e.metrics["update"],
		prometheus.GaugeValue,
		data.NewVersionAvailable,
		data.Project,
		data.Server,
	)
	if err != nil {
		e.log.Warnf("Failed to export prometheus metric: %v", err)
		return
	}
	ch <- d
}
