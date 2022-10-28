package targz

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/klauspost/pgzip"
)

func GetFileWriter(filePath string, gZip bool) (io.WriteCloser, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	var writer io.WriteCloser
	if gZip {
		writer = pgzip.NewWriter(file)
	} else {
		writer = file
	}

	return writer, nil
}

func GZip(src, dst string) error {
	fileWriter, err := GetFileWriter(dst, true)
	if err != nil {
		return err
	}
	defer func() { _ = fileWriter.Close() }()

	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(fileWriter, file)
	return err
}

func Tar(src, dst string, gz, saveAbsPath bool, excludes []*regexp.Regexp) error {

	fileWriter, err := GetFileWriter(dst, gz)
	if err != nil {
		return err
	}
	defer func() { _ = fileWriter.Close() }()

	tarWriter := tar.NewWriter(fileWriter)
	defer func() { _ = tarWriter.Close() }()

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(src)
	}

	return filepath.Walk(src,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			for _, excl := range excludes {
				if excl.MatchString(path) {
					return nil
				}
			}

			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			if saveAbsPath {
				header.Name = path
			} else if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, src))
			}

			header.Format = tar.FormatPAX
			header.PAXRecords = map[string]string{
				"mtime": fmt.Sprintf("%f", float64(header.ModTime.UnixNano())/float64(time.Second)),
				"atime": fmt.Sprintf("%f", float64(header.AccessTime.UnixNano())/float64(time.Second)),
				"ctime": fmt.Sprintf("%f", float64(header.ChangeTime.UnixNano())/float64(time.Second)),
			}

			if err = tarWriter.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() { _ = file.Close() }()

			_, err = io.Copy(tarWriter, file)
			return err
		})
}
