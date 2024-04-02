package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	appctx "github.com/nixys/nxs-go-appctx/v2"

	"nxs-backup/ctx"
	"nxs-backup/modules/arg_cmd"
	"nxs-backup/modules/logger"
	"nxs-backup/routines/logging"
)

func main() {

	subCmds := ctx.SubCmds{
		"start":    arg_cmd.Start,
		"testCfg":  arg_cmd.TestConfig,
		"generate": arg_cmd.GenerateConfig,
		"update":   arg_cmd.SelfUpdate,
	}

	// Read command line arguments
	a := ctx.ReadArgs(subCmds)

	// Init appctx
	appCtx, err := appctx.ContextInit(appctx.Settings{
		CustomContext:    &ctx.Ctx{},
		Args:             &a,
		CfgPath:          a.ConfigPath,
		TermSignals:      []os.Signal{syscall.SIGTERM, syscall.SIGINT},
		ReloadSignals:    []os.Signal{syscall.SIGHUP},
		LogrotateSignals: []os.Signal{syscall.SIGUSR1},
		LogFormatter:     &logger.LogFormatter{},
	})
	if err != nil {
		fmt.Printf("Failed to init nxs-backup with next error:\n%s", err)
		os.Exit(1)
	}

	// Create logging and notification routine
	appCtx.RoutineCreate(context.Background(), logging.Runtime)

	cc := appCtx.CustomCtx().(*ctx.Ctx)

	// exec command
	err = a.CmdHandler(appCtx)
	// wait for logging and notification tasks complete
	time.Sleep(time.Millisecond)
	cc.WG.Wait()
	if err != nil {
		fmt.Println("exec error: ", err)
		os.Exit(1)
	}

	os.Exit(0)
}
