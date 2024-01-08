package workflow

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var ErrNotFound = errors.New("could not find workflow.yaml")

const (
	WorkflowVersion = "1.0.0"
)

type Workflow struct {
	Version string            `yaml:"workflowVersion"`
	Sources map[string]Source `yaml:"sources"`
	Targets map[string]Target `yaml:"targets"`
}

func Load(dir string) (*Workflow, string, error) {
	data, path, err := findWorkflowFile(dir)
	if err != nil {
		return nil, "", err
	}

	var workflow Workflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal workflow.yaml: %w", err)
	}

	return &workflow, path, nil
}

func (w Workflow) Validate(supportLangs []string) error {
	if w.Version != WorkflowVersion {
		return fmt.Errorf("unsupported workflow version: %s", w.Version)
	}

	if len(w.Targets) == 0 {
		return fmt.Errorf("no targets found")
	}

	for targetID, target := range w.Targets {
		if err := target.Validate(supportLangs, w.Sources); err != nil {
			return fmt.Errorf("failed to validate target %s: %w", targetID, err)
		}
	}

	for sourceID, source := range w.Sources {
		if err := source.Validate(); err != nil {
			return fmt.Errorf("failed to validate source %s: %w", sourceID, err)
		}
	}

	return nil
}

func findWorkflowFile(dir string) ([]byte, string, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	path := filepath.Join(absPath, ".speakeasy", "workflow.yaml")

	for {
		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// Get the parent directory of the .speakeasy dir
				currentDir := filepath.Dir(filepath.Dir(path))

				// Check for the root of the filesystem or path
				// ie `.` for `./something`
				// or `/` for `/some/absolute/path` in linux
				// or `:\\` for `C:\\` in windows
				if currentDir == "." || currentDir == "/" || currentDir[1:] == ":\\" {
					return nil, "", ErrNotFound
				}

				// Get the parent directory of the current dir and append ".speakeasy" as we only check in side the .speakeasy dir
				path = filepath.Join(filepath.Dir(currentDir), ".speakeasy", "workflow.yaml")
				continue
			}

			return nil, "", fmt.Errorf("could not read workflow.yaml: %w", err)
		}

		return data, path, nil
	}
}

func validateSecret(secret string) error {
	if !strings.HasPrefix(secret, "$") {
		return fmt.Errorf("secret must be a environment variable reference (ie $MY_SECRET)")
	}

	return nil
}
