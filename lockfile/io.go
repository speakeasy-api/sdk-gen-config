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

	if lf.AdditionalProperties != nil {
		delete(lf.AdditionalProperties, "generatedFileHashes")
	}

	if lf.TrackedFiles == nil {
		lf.TrackedFiles = sequencedmap.New[string, TrackedFile]()
	}

	// Migrate old fields to new structure
	for path := range lf.TrackedFiles.Keys() {
		tf, ok := lf.TrackedFiles.Get(path)
		if !ok {
			continue
		}

		modified := false

		// Check if integrity exists in AdditionalProperties
		if tf.AdditionalProperties != nil {
			if integrity, ok := tf.AdditionalProperties["integrity"].(string); ok {
				// Migrate to LastWriteChecksum if not already set
				if tf.LastWriteChecksum == "" {
					tf.LastWriteChecksum = integrity
				}
				// Remove from AdditionalProperties
				delete(tf.AdditionalProperties, "integrity")
				modified = true
			}

			// Migrate old "pristine_blob_hash" to "pristine_git_object"
			if pristineBlobHash, ok := tf.AdditionalProperties["pristine_blob_hash"].(string); ok {
				// Migrate to PristineGitObject if not already set
				if tf.PristineGitObject == "" {
					tf.PristineGitObject = pristineBlobHash
				}
				// Remove from AdditionalProperties
				delete(tf.AdditionalProperties, "pristine_blob_hash")
				modified = true
			}
		}

		if modified {
			lf.TrackedFiles.Set(path, tf)
		}
	}

	return &lf, nil
}
