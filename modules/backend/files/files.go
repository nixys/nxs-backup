package files

import (
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"

	"nxs-backup/misc"
)

func CreateTmpMysqlAuthFile(af *ini.File) (authFile string, err error) {
	authFile = filepath.Join("/tmp", misc.RandString(20))
	file, err := os.OpenFile(authFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0400)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	if _, err = af.WriteTo(file); err != nil {
		return
	}
	return
}

func DeleteTmpMysqlAuthFile(path string) error {
	return os.RemoveAll(path)
}
