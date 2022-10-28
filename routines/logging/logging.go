package logging

import (
	"context"

	appctx "github.com/nixys/nxs-go-appctx/v2"

	"nxs-backup/ctx"
	"nxs-backup/modules/logger"
)

// Runtime executes the routine
func Runtime(c context.Context, appCtx *appctx.AppContext, crc chan interface{}) {

	cc := appCtx.CustomCtx().(*ctx.Ctx)

	for {
		select {
		case log := <-cc.LogCh:
			logger.WriteLog(appCtx.Log(), log)
			go cc.Mailer.Send(appCtx, log, cc.WG)
			go cc.Alerter.Send(appCtx, log, cc.WG)
		case <-c.Done():
			// Program termination.
			return
		case <-crc:
			// Updated context application data.
			// Set the new one in current goroutine.
		}
	}
}
