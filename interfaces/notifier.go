package interfaces

import (
	"sync"

	appctx "github.com/nixys/nxs-go-appctx/v2"

	"nxs-backup/modules/logger"
)

type Notifier interface {
	Send(ctx *appctx.AppContext, log logger.LogRecord, wg *sync.WaitGroup)
}
