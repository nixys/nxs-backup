package targz

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/klauspost/pgzip"
	"github.com/mb0/glob"

	"nxs-backup/misc"
)

const delimiterSymbol = "\u0000" // for incremental tars headers

type Metadata map[string]float64

type dumpDirs map[string]string

type hardLinks map[uint64]string

type Error struct {
	Err    error
	File   string
	Header *tar.Header
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

func Tar(src, dst string, gz, saveAbsPath bool, excludes []string) error {

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
		func(fPath string, info os.FileInfo, err error) error {
			if err != nil {
				// skip removed file
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}

			for _, pattern := range excludes {
				if match, _ := glob.Match(pattern, fPath); match {
					return nil
				}
			}

			// skipping sockets
			if info.Mode()&fs.ModeSocket != 0 {
				return nil
			}

			link, _ := os.Readlink(fPath)
			header, err := tar.FileInfoHeader(info, link)
			if err != nil {
				return err
			}

			stat := info.Sys().(*syscall.Stat_t)
			if stat.Nlink > 1 {
				l, ok := hLinks[stat.Ino]
				if ok {
					link, err = filepath.Rel(filepath.Dir(fPath), l)
					if err != nil {
						return err
					}
					header.Linkname = link
					header.Typeflag = tar.TypeLink
					header.Size = 0
				} else {
					hLinks[stat.Ino] = fPath
				}
			}

			if saveAbsPath {
				header.Name = fPath
			} else if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(fPath, src))
			}

			header.Format = tar.FormatPAX
			header.PAXRecords = map[string]string{
				"mtime": fmt.Sprintf("%f", float64(header.ModTime.UnixNano())/float64(time.Second)),
				"atime": fmt.Sprintf("%f", float64(header.AccessTime.UnixNano())/float64(time.Second)),
				"ctime": fmt.Sprintf("%f", float64(header.ChangeTime.UnixNano())/float64(time.Second)),
			}

			if info.IsDir() {
				header.Name = misc.DirNormalize(header.Name)
			}

			if err = tarWriter.WriteHeader(header); err != nil {
				return err
			}
			// skip not regular files
			if header.Typeflag != tar.TypeReg {
				return nil
			}

			file, err := os.Open(fPath)
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

func TarIncremental(src, dst string, gz, saveAbsPath, inc bool, excludes []string, prevMtd Metadata) error {
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

	// create new metadata index and dumpdirs
	mtd := make(Metadata)
	dd := make(dumpDirs)

	// index files and prepare "GNU.dumpdir" PAX header
	if err = filepath.WalkDir(src,
		func(fPath string, dir fs.DirEntry, err error) error {
			if err != nil {
				// skip removed file
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}

			for _, pattern := range excludes {
				if match, _ := glob.Match(pattern, fPath); match {
					return nil
				}
			}

			info, err = dir.Info()
			if err != nil {
				return err
			}
			// skipping sockets
			if info.Mode()&fs.ModeSocket != 0 {
				return nil
			}

			mTime := float64(info.ModTime().UnixNano()) / float64(time.Second)
			mtd[fPath] = mTime

			if _, ok := dd[path.Dir(fPath)]; ok {
				flag := ""
				if info.IsDir() {
					flag += "D"
				} else if prevMtd[fPath] == mTime {
					flag += "N"
				} else {
					flag += "Y"
				}
				dd[path.Dir(fPath)] += flag + info.Name() + delimiterSymbol
			}

			return nil
		}); err != nil {
		return err
	}

	hLinks := make(hardLinks)
	for fPath := range mtd {
		var headerName string
		if saveAbsPath {
			headerName = fPath
		} else if baseDir != "" {
			headerName = path.Join(baseDir, strings.TrimPrefix(fPath, src))
		}
		err = writeToTar(tarWriter, fPath, headerName, hLinks, dd, prevMtd, inc)
		if err != nil {
			return err
		}
	}

	file, err := json.Marshal(mtd)
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join(path.Dir(dst), path.Base(dst)+".inc"), file, 0644)
	if err != nil {
		return err
	}

	return nil
}

func writeToTar(tw *tar.Writer, fPath, headerName string, hLinks hardLinks, dd dumpDirs, prevMtd Metadata, inc bool) error {
	info, err := os.Lstat(fPath)
	if err != nil {
		return err
	}

	link, _ := os.Readlink(fPath)
	header, err := tar.FileInfoHeader(info, link)
	if err != nil {
		return Error{
			Err:    err,
			File:   fPath,
			Header: header,
		}
	}

	stat := info.Sys().(*syscall.Stat_t)
	if stat.Nlink > 1 {
		if l, ok := hLinks[stat.Ino]; ok {
			link, err = filepath.Rel(path.Dir(fPath), l)
			if err != nil {
				return Error{
					Err:    err,
					File:   fPath,
					Header: header,
				}
			}
			header.Linkname = link
			header.Typeflag = tar.TypeLink
			header.Size = 0
		} else {
			hLinks[stat.Ino] = fPath
		}
	}

	if info.IsDir() {
		headerName = misc.DirNormalize(headerName)
	}
	header.Name = headerName
	header.Format = tar.FormatPAX

	mTime := float64(header.ModTime.UnixNano()) / float64(time.Second)
	aTime := float64(header.AccessTime.UnixNano()) / float64(time.Second)
	cTime := float64(header.ChangeTime.UnixNano()) / float64(time.Second)

	paxRecs := map[string]string{
		"mtime": fmt.Sprintf("%f", mTime),
		"atime": fmt.Sprintf("%f", aTime),
		"ctime": fmt.Sprintf("%f", cTime),
	}

	if dumpDir, ok := dd[fPath]; ok {
		paxRecs["GNU.dumpdir"] = dumpDir + delimiterSymbol
	}
	header.PAXRecords = paxRecs

	// skip unchanged regular files for incremental tar archives
	if inc && header.Typeflag == tar.TypeReg && prevMtd[fPath] == mTime {
		return nil
	}

	if err = tw.WriteHeader(header); err != nil {
		return Error{
			Err:    err,
			File:   fPath,
			Header: header,
		}
	}
	// skip non regular files
	if header.Typeflag != tar.TypeReg {
		return nil
	}

	file, err := os.Open(fPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	_, err = io.CopyN(tw, file, header.Size)
	if err != nil {
		return fmt.Errorf("failed to archivate file %s: %v", file.Name(), err)
	}
	return nil
}
