package arg_cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	appctx "github.com/nixys/nxs-go-appctx/v2"

	"nxs-backup/ctx"
	"nxs-backup/misc"
)

func SelfUpdate(appCtx *appctx.AppContext) error {
	var tmpBinFile *os.File

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
	tarPath := exePath + ".tgz"
	newExePath := exePath + "-new"

	tarFile, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tarFile.Name()) }()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	_, err = io.Copy(tarFile, resp.Body)
	if err != nil {
		return err
	}
	defer func() { _ = tarFile.Close() }()

	_, err = tarFile.Seek(0, 0)
	if err != nil {
		return err
	}

	gr, err := gzip.NewReader(tarFile)
	if err != nil {
		return err
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)

	tmpBinFile, err = os.OpenFile(newExePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmpBinFile.Name()) }()

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Name == "./nxs-backup" {
			if _, err := io.Copy(tmpBinFile, tr); err != nil {
				return err
			}
			break
		}
	}

	err = tmpBinFile.Close()
	if err != nil {
		return err
	}

	err = os.Rename(tmpBinFile.Name(), exePath)
	if err != nil {
		return err
	}

	fmt.Println("Update completed.")
	return nil
}
