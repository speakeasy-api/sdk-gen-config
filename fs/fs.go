package fs

import (
	originalFs "io/fs"
	"os"
)

type FS interface {
	originalFs.ReadFileFS
	originalFs.StatFS
	WriteFile(name string, data []byte, perm os.FileMode) error
	Abs(path string) (string, error)
}

type FileInfo = originalFs.FileInfo

var (
	ErrNotExist = originalFs.ErrNotExist
)
