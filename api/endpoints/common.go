package endpoints

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func Logger(log *logrus.Logger) gin.HandlerFunc {
	return func(gc *gin.Context) {
		gc.Next()
		log.WithFields(logrus.Fields{
			"type":      "accesslog",
			"remote":    gc.RemoteIP(),
			"method":    gc.Request.Method,
			"url":       gc.Request.RequestURI,
			"code":      gc.Writer.Status(),
			"userAgent": gc.Request.UserAgent(),
		}).Debug("request processed")
	}
}
