package targz

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"

	"github.com/klauspost/pgzip"

	"github.com/nixys/nxs-backup/misc"
	"github.com/nixys/nxs-backup/modules/backend/files"
)

const (
	defaultBlockSize = 1 << 20
	regexToIgnoreErr = "^tar:.*(Removing leading|socket ignored|file changed as we read it|Удаляется начальный|сокет проигнорирован|файл изменился во время чтения)"
)

type Error struct {
	Err    error
	Stderr string
}

type TarOpts struct {
	Src         string
	Dst         string
	Incremental bool
	Gzip        bool
	SaveAbsPath bool
	RateLim     int64
	Excludes    []string
}

func (e Error) Error() string {
	return e.Err.Error()
}

func GetGZipFileWriter(filePath string, gZip bool, rateLim int64) (io.WriteCloser, error) {
	var gzw *pgzip.Writer

	lwc, err := files.GetLimitedFileWriter(filePath, rateLim)
	if err != nil {
		return nil, err
	}

	if gZip {
		if gzw, err = pgzip.NewWriterLevel(lwc, pgzip.BestCompression); err != nil {
			return nil, err
		}
		err = gzw.SetConcurrency(defaultBlockSize, runtime.GOMAXPROCS(misc.CPULimit))
		lwc = gzw
	}

	return lwc, err
}

func GZip(src, dst string, rateLim int64) error {
	fileWriter, err := GetGZipFileWriter(dst, true, rateLim)
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

func Tar(o TarOpts) error {
	tarWriter, err := GetGZipFileWriter(o.Dst, o.Gzip, o.RateLim)
	if err != nil {
		return err
	}
	defer func() { _ = tarWriter.Close() }()

	var stderr bytes.Buffer
	var args []string

	args = append(args, "--format=pax")

	if o.Incremental {
		args = append(args, "--listed-incremental="+o.Dst+".inc")
	}
	for _, ex := range o.Excludes {
		args = append(args, "--exclude="+ex)

	}
	args = append(args, "--ignore-failed-read")
	args = append(args, "--create")
	args = append(args, "--file=-")
	if o.SaveAbsPath {
		args = append(args, o.Src)
	} else {
		args = append(args, "--directory="+path.Dir(o.Src))
		args = append(args, path.Base(o.Src))
	}

	cmd := exec.Command("tar", args...)
	cmd.Stdout = tarWriter
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		if cmd.ProcessState.ExitCode() == 2 || checkIsRealError(stderr.String()) {
			return Error{
				Err:    err,
				Stderr: stderr.String(),
			}
		}
	}

	return nil
}

func checkIsRealError(stderr string) bool {
	realErr := false
	reTar := regexp.MustCompile("^tar:.*\n")
	reErr := regexp.MustCompile(regexToIgnoreErr)
	strTupl := reTar.FindAllString(stderr, -1)
	for _, s := range strTupl {
		if match := reErr.MatchString(s); !match {
			realErr = true
			break
		}
	}

	return realErr
}
