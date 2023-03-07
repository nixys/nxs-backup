package targz

import (
	"bytes"
	"github.com/klauspost/pgzip"
	"io"
	"os"
	"os/exec"
	"path"
)

type Error struct {
	Err    error
	Stderr string
}

func (e Error) Error() string {
	return e.Err.Error()
}

func GetFileWriter(filePath string, gZip bool) (io.WriteCloser, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	var writer io.WriteCloser
	if gZip {
		writer, err = pgzip.NewWriterLevel(file, pgzip.BestCompression)
	} else {
		writer = file
	}

	return writer, err
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

func Tar(src, dst string, incremental, gzip, saveAbsPath bool, excludes []string) error {

	tarWriter, err := GetFileWriter(dst, gzip)
	if err != nil {
		return err
	}
	defer func() { _ = tarWriter.Close() }()

	var stderr bytes.Buffer
	var args []string

	args = append(args, "--format=pax")

	if incremental {
		args = append(args, "--listed-incremental="+dst+".inc")
	}
	for _, ex := range excludes {
		args = append(args, "--exclude="+ex)

	}
	args = append(args, "--create")
	args = append(args, "--file=-")
	if saveAbsPath {
		args = append(args, src)
	} else {
		args = append(args, "--directory="+path.Dir(src))
		args = append(args, path.Base(src))
	}

	cmd := exec.Command("tar", args...)
	cmd.Stdout = tarWriter
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		return Error{
			Err:    err,
			Stderr: stderr.String(),
		}
	}

	return nil
}
