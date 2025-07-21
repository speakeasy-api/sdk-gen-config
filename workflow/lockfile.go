package workflow

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/sdk-gen-config/workspace"
	"gopkg.in/yaml.v3"
)

const workflowLockfile = "workflow.lock"

type LockFile struct {
	SpeakeasyVersion string                `yaml:"speakeasyVersion"`
	Sources          map[string]SourceLock `yaml:"sources"`
	Targets          map[string]TargetLock `yaml:"targets"`

	Workflow Workflow `yaml:"workflow"`
}

type SourceLock struct {
	SourceNamespace      string   `yaml:"sourceNamespace,omitempty"`
	SourceRevisionDigest string   `yaml:"sourceRevisionDigest,omitempty"`
	SourceBlobDigest     string   `yaml:"sourceBlobDigest,omitempty"`
	Tags                 []string `yaml:"tags,omitempty"`
}

type TargetLock struct {
	Source                    string `yaml:"source"`
	SourceNamespace           string `yaml:"sourceNamespace,omitempty"`
	SourceRevisionDigest      string `yaml:"sourceRevisionDigest,omitempty"`
	SourceBlobDigest          string `yaml:"sourceBlobDigest,omitempty"`
	CodeSamplesNamespace      string `yaml:"codeSamplesNamespace,omitempty"`
	CodeSamplesRevisionDigest string `yaml:"codeSamplesRevisionDigest,omitempty"`
	CodeSamplesBlobDigest     string `yaml:"codeSamplesBlobDigest,omitempty"`
	ReleaseNotes              string `yaml:"releaseNotes,omitempty"`
}

func LoadLockfile(dir string) (*LockFile, error) {
	res, err := workspace.FindWorkspace(dir, workspace.FindWorkspaceOptions{
		FindFile:  workflowLockfile,
		Recursive: true,
	})
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		return nil, fmt.Errorf("%w in %s", err, filepath.Join(dir, workspace.SpeakeasyFolder, workflowLockfile))
	}

	var lockfile LockFile
	if err := yaml.Unmarshal(res.Data, &lockfile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow.lock: %w", err)
	}

	return &lockfile, nil
}

// Save the workflow lockfile to the given directory, dir should generally be the root of the project,
// and the lockfile will be saved to ${projectRoot}/.speakeasy/workflow.lock
func SaveLockfile(dir string, lockfile *LockFile) error {
	data, err := yaml.Marshal(lockfile)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow lockfile: %w", err)
	}

	res, err := workspace.FindWorkspace(dir, workspace.FindWorkspaceOptions{
		FindFile:  workflowLockfile,
		Recursive: true,
	})
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		res = &workspace.FindWorkspaceResult{
			Path: filepath.Join(dir, workspace.SpeakeasyFolder, workflowLockfile),
		}
	}

	if err := os.WriteFile(res.Path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write workflow.lock: %w", err)
	}

	return nil
}
