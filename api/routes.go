package api

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/nixys/nxs-backup/api/endpoints"
)

func RoutesSet(log *logrus.Logger, reg *prometheus.Registry) *gin.Engine {

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(endpoints.Logger(log))

	router.GET("/metrics", gin.WrapH(
		promhttp.HandlerFor(
			reg,
			promhttp.HandlerOpts{
				EnableOpenMetrics: false,
			},
		),
	))

	return router
}
