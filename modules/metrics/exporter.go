package metrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Exporter struct {
	metrics        map[string]*prometheus.Desc
	ctx            context.Context
	log            *logrus.Logger
	metricFilePath string
}

type ExporterOpts struct {
	Log            *logrus.Logger
	MetricFilePath string
}

func InitExporter(s ExporterOpts) *Exporter {

	metrics := map[string]*prometheus.Desc{
		BackupSize: prometheus.NewDesc(
			prometheus.BuildFQName("nxs_backup", "file", "size"),
			"Backup file size",
			[]string{"project", "server", "job_name", "job_type", "source", "target"}, nil,
		),
		BackupOk: prometheus.NewDesc(
			prometheus.BuildFQName("nxs_backup", "collection", "success"),
			"Backup finished successfully",
			[]string{"project", "server", "job_name", "job_type", "source", "target"}, nil,
		),
		BackupTime: prometheus.NewDesc(
			prometheus.BuildFQName("nxs_backup", "collection", "time"),
			"Backup collection time",
			[]string{"project", "server", "job_name", "job_type", "source", "target"}, nil,
		),
		BackupTimestamp: prometheus.NewDesc(
			prometheus.BuildFQName("nxs_backup", "creation", "ts"),
			"Backup creation timestamp",
			[]string{"project", "server", "job_name", "job_type", "source", "target"}, nil,
		),
		DeliveryOk: prometheus.NewDesc(
			prometheus.BuildFQName("nxs_backup", "delivery", "success"),
			"Backup delivery finished successfully",
			[]string{"project", "server", "job_name", "job_type", "source", "target"}, nil,
		),
		DeliveryTime: prometheus.NewDesc(
			prometheus.BuildFQName("nxs_backup", "delivery", "time"),
			"Backup delivering time",
			[]string{"project", "server", "job_name", "job_type", "source", "target"}, nil,
		),
		UpdateAvailable: prometheus.NewDesc(
			prometheus.BuildFQName("nxs_backup", "update", "available"),
			"A new version of nxs-backup is available",
			[]string{"project", "server"}, nil,
		),
	}

	return &Exporter{
		metrics:        metrics,
		log:            s.Log,
		metricFilePath: s.MetricFilePath,
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

	data, err := readFile(e.metricFilePath)
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
					string(j.JobType),
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
		e.metrics[UpdateAvailable],
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
