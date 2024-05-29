package api_server

import (
	"github.com/nixys/nxs-backup/modules/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/nixys/nxs-backup/api"
)

type Opts struct {
	Bind string
	Log  *logrus.Logger
	Done chan error
}

type httpServer struct {
	http.Server
	log      *logrus.Logger
	exporter *metrics.Exporter
	registry *prometheus.Registry
	done     chan error
}

func Init(o Opts) (*httpServer, error) {

	exporter := metrics.Init(
		metrics.InitSettings{
			Log: o.Log,
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
