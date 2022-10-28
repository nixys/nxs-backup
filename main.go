package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/nightlyone/lockfile"
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

	cc := appCtx.CustomCtx().(*ctx.Ctx)

	// Crate lockfile
	lock, _ := lockfile.New(path.Join(os.TempDir(), "nxs-backup.lck"))
	if cc.Cfg.WaitingTimeout != 0 {
		now := time.Now()
		waitTill := now.Add(time.Minute * cc.Cfg.WaitingTimeout)
		for waitTill.After(time.Now()) {
			if err = lock.TryLock(); err != nil {
				time.Sleep(time.Second * 5)
			} else {
				break
			}
		}
	} else {
		err = lock.TryLock()
	}
	if err != nil {
		fmt.Printf("Another nxs-backup already running")
		os.Exit(1)
	}
	defer func() { _ = lock.Unlock() }()

	// Create logging and notification routine
	appCtx.RoutineCreate(context.Background(), logging.Runtime)

	// exec command
	err = a.CmdHandler(appCtx)
	// wait for logging and notification tasks complete
	cc.WG.Wait()
	if err != nil {
		fmt.Println("exec error: ", err)
		os.Exit(1)
	}

	os.Exit(0)
}
