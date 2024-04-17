package cmd_handler

import (
	appctx "github.com/nixys/nxs-go-appctx/v3"
	"github.com/sirupsen/logrus"

	"github.com/nixys/nxs-backup/ctx"
)

func Runtime(app appctx.App) error {
	var err error

	cc := app.ValueGet().(*ctx.Ctx)
	cc.Log.Trace("cmd routine: start")

	go cc.Cmd.Run()

	for {
		select {
		case <-app.SelfCtxDone():
			cc.Log.Trace("cmd routine: shutdown")
			cc.EventsWG.Wait()
			return nil
		case err = <-cc.Done:
			if err != nil {
				cc.Log.WithFields(logrus.Fields{"details": err}).Errorf("cmd routine fail:")
			}
			cc.Log.Trace("cmd routine: done")
			app.RoutineShutdown("notification")
			cc.EventsWG.Wait()
			return err
		}
	}
}
