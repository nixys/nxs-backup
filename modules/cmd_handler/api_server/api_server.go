package api_server

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/nixys/nxs-backup/api"
	"github.com/nixys/nxs-backup/modules/metrics"
)

type Opts struct {
	Bind           string
	MetricFilePath string
	Log            *logrus.Logger
	Done           chan error
}

type httpServer struct {
	http.Server
	log      *logrus.Logger
	exporter *metrics.Exporter
	registry *prometheus.Registry
	done     chan error
}

func Init(o Opts) (*httpServer, error) {

	exporter := metrics.InitExporter(
		metrics.ExporterOpts{
			Log:            o.Log,
			MetricFilePath: o.MetricFilePath,
		},
	)

	registry := prometheus.NewRegistry()
	if err := registry.Register(exporter); err != nil {
		o.Log.Errorf("ctx init: %s", err.Error())
		return nil, err
	}

	return &httpServer{
		Server: http.Server{
			Addr:         o.Bind,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			Handler:      api.RoutesSet(o.Log, registry),
		},
		exporter: exporter,
		registry: registry,
		log:      o.Log,
		done:     o.Done,
	}, nil
}

func (s *httpServer) Run() {

	s.log.Trace("api: starting")
	err := s.ListenAndServe()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"details": err,
		}).Debugf("api: server fail")
	}
	s.done <- err
}
