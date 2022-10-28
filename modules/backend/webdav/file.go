package webdav

import (
	"os"
	"time"
)

type webDavFile struct {
	name  string
	size  int64
	mode  os.FileMode
	mtime time.Time
	raw   string
}

func (f *webDavFile) Name() string {
	return f.name
}

func (f *webDavFile) Size() int64 {
	return f.size
}

func (f *webDavFile) Mode() os.FileMode {
	return f.mode
}

func (f *webDavFile) ModTime() time.Time {
	return f.mtime
}

func (f *webDavFile) IsDir() bool {
	return f.mode.IsDir()
}

func (f *webDavFile) Sys() interface{} {
	return f.raw
}
