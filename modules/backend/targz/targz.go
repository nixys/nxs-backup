package targz

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/klauspost/pgzip"
)

type Metadata map[string]float64

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

	hLinks := make(map[uint64]string)

	return filepath.Walk(src,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// skip removed file
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}

			for _, excl := range excludes {
				if excl.MatchString(path) {
					return nil
				}
			}

			// skipping sockets
			if info.Mode()&fs.ModeSocket != 0 {
				return nil
			}

			link, _ := os.Readlink(path)
			header, err := tar.FileInfoHeader(info, link)
			if err != nil {
				return err
			}

			stat := info.Sys().(*syscall.Stat_t)
			if stat.Nlink > 1 {
				l, ok := hLinks[stat.Ino]
				if ok {
					link, err = filepath.Rel(filepath.Dir(path), l)
					if err != nil {
						return err
					}
					header.Linkname = link
					header.Typeflag = tar.TypeLink
					header.Size = 0
				} else {
					hLinks[stat.Ino] = path
				}
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

			if info.IsDir() {
				header.Name += "/"
			}

			if err = tarWriter.WriteHeader(header); err != nil {
				return err
			}
			// skip not regular files
			if header.Typeflag != tar.TypeReg {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() { _ = file.Close() }()

			_, err = io.CopyN(tarWriter, file, header.Size)
			if err != nil {
				return fmt.Errorf("failed to archivate file %s: %v", file.Name(), err)
			}
			return nil
		})
}

func TarIncremental(src, dst string, gz, saveAbsPath, initMtd bool, excludes []*regexp.Regexp, prevMtd Metadata) error {
	// create new index
	mtd := make(Metadata)

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
		baseDir = path.Base(src)
	}

	hLinks := make(map[uint64]string)

	err = filepath.Walk(src,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			for _, excl := range excludes {
				if excl.MatchString(path) {
					return nil
				}
			}

			// skipping sockets
			if info.Mode()&fs.ModeSocket != 0 {
				return nil
			}

			link, _ := os.Readlink(path)
			header, err := tar.FileInfoHeader(info, link)
			if err != nil {
				return err
			}

			stat := info.Sys().(*syscall.Stat_t)
			if stat.Nlink > 1 {
				l, ok := hLinks[stat.Ino]
				if ok {
					link, err = filepath.Rel(filepath.Dir(path), l)
					if err != nil {
						return err
					}
					header.Linkname = link
					header.Typeflag = tar.TypeLink
					header.Size = 0
				} else {
					hLinks[stat.Ino] = path
				}
			}

			if saveAbsPath {
				header.Name = path
			} else if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, src))
			}
			header.Format = tar.FormatPAX

			mTime := float64(header.ModTime.UnixNano()) / float64(time.Second)
			aTime := float64(header.AccessTime.UnixNano()) / float64(time.Second)
			cTime := float64(header.ChangeTime.UnixNano()) / float64(time.Second)

			paxRecs := map[string]string{
				"mtime": fmt.Sprintf("%f", mTime),
				"atime": fmt.Sprintf("%f", aTime),
				"ctime": fmt.Sprintf("%f", cTime),
			}

			if info.IsDir() {
				var (
					files   []fs.FileInfo
					dumpDir string
				)
				delimiterSymbol := "\u0000"
				header.Name += "/"

				files, err = ioutil.ReadDir(path)
				if err != nil {
					return err
				}
				for _, fi := range files {
					excluded := false
					for _, excl := range excludes {
						if excl.MatchString(filepath.Join(path, fi.Name())) {
							excluded = true
							break
						}
					}
					if excluded {
						continue
					}

					if fi.IsDir() {
						dumpDir += "D"
					} else if prevMtd[path] == mtd[path] {
						dumpDir += "N"
					} else {
						dumpDir += "Y"
					}
					dumpDir += fi.Name() + delimiterSymbol
				}
				paxRecs["GNU.dumpdir"] = dumpDir + delimiterSymbol
			} else {
				mtd[path] = mTime
			}
			header.PAXRecords = paxRecs

			// skip unchanged regular files
			if header.Typeflag == tar.TypeReg && prevMtd[path] == mtd[path] {
				return nil
			}
			if err = tarWriter.WriteHeader(header); err != nil {
				return err
			}
			// skip not regular files
			if header.Typeflag != tar.TypeReg {
				return nil
			}
			return func() error {
				var file fs.File
				file, err = os.Open(path)
				defer func() { _ = file.Close() }()
				if err != nil {
					return err
				}
				_, err = io.CopyN(tarWriter, file, header.Size)
				if err != nil {
					return err
				}
				return nil
			}()
		})
	if err != nil {
		return err
	}

	file, err := json.Marshal(mtd)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(path.Dir(dst), path.Base(dst)+".inc"), file, 0644)
	if err != nil {
		return err
	}

	if initMtd {
		_, err = os.Create(path.Join(path.Dir(dst), path.Base(dst)+".init"))
	}
	return err
}
