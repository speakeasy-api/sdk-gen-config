package lockfile

import (
	"fmt"
	"io/fs"

	"github.com/speakeasy-api/openapi/sequencedmap"
	"gopkg.in/yaml.v3"
)

type LoadOption func(*loadOptions)

type loadOptions struct {
	fileSystem fs.FS
}

func WithFileSystem(fileSystem fs.FS) LoadOption {
	return func(o *loadOptions) {
		o.fileSystem = fileSystem
	}
}

func Load(data []byte, opts ...LoadOption) (*LockFile, error) {
	o := &loadOptions{}
	for _, opt := range opts {
		opt(o)
	}

	var lf LockFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("could not unmarshal lockfile: %w", err)
	}

	if lf.TrackedFiles == nil {
		lf.TrackedFiles = sequencedmap.New[string, TrackedFile]()
	}

	return &lf, nil
}
