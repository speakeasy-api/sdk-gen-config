package config

import (
	"github.com/speakeasy-api/sdk-gen-config/lockfile"
)

type (
	LockFile          = lockfile.LockFile
	Management        = lockfile.Management
	Examples          = lockfile.Examples
	GeneratedTests    = lockfile.GeneratedTests
	TrackedFiles      = lockfile.TrackedFiles
	TrackedFile       = lockfile.TrackedFile
	OperationExamples = lockfile.OperationExamples
	ParameterExamples = lockfile.ParameterExamples
	LockfileOption    = lockfile.LoadOption
)

var getUUID = lockfile.GetUUID

func NewLockFile() *LockFile {
	return lockfile.New()
}

func WithLockfileFileSystem(fs FS) LockfileOption {
	return lockfile.WithFileSystem(fs)
}

func LoadLockfile(data []byte, fileSystem FS) (*LockFile, error) {
	if fileSystem != nil {
		return lockfile.Load(data, lockfile.WithFileSystem(fileSystem))
	}
	return lockfile.Load(data)
}
