package arg_cmd

import (
	"fmt"
	appctx "github.com/nixys/nxs-go-appctx/v2"
	"io"
	"log"
	"net/http"
	"nxs-backup/ctx"
	"os"

	"nxs-backup/misc"
)

func SelfUpdate(appCtx *appctx.AppContext) error {

	cc := appCtx.CustomCtx().(*ctx.Ctx)

	ver := cc.CmdParams.(*ctx.UpdateCmd).Version

	newVer, url, err := misc.CheckNewVersionAvailable(ver)
	if err != nil {
		return err
	}

	if newVer == "" {
		fmt.Println("No new versions.")
		return nil
	}
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get current executable: %v", err)
	}
	newExePath := exePath + "-new"

	tmpFile, err := os.Create(newExePath)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return err
	}

	err = tmpFile.Close()
	if err != nil {
		return err
	}

	err = os.Rename(tmpFile.Name(), exePath)
	if err != nil {
		return err
	}

	fmt.Println("Update completed.")
	return nil
}
