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

const (
	speakeasyFolder = ".speakeasy"
	genFolder       = ".gen"
)

type Workflow struct {
	Version string            `yaml:"workflowVersion"`
	Sources map[string]Source `yaml:"sources"`
	Targets map[string]Target `yaml:"targets"`
}

func Load(dir string) (*Workflow, string, error) {
	data, path, err := findWorkflowFile(dir, "")
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, "", err
		}
		return nil, "", fmt.Errorf("%w in %s", err, filepath.Join(dir, speakeasyFolder, "workflow.yaml"))
	}

	var workflow Workflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal workflow.yaml: %w", err)
	}

	return &workflow, path, nil
}

// Save the workflow to the given directory, dir should generally be the root of the project, and the workflow will be saved to ${projectRoot}/.speakeasy/workflow.yaml
func Save(dir string, workflow *Workflow) error {
	data, err := yaml.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	_, workflowFilePath, err := findWorkflowFile(dir, "")
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return err
		}
		workflowFilePath = filepath.Join(dir, speakeasyFolder, "workflow.yaml")
	}

	if err := os.WriteFile(workflowFilePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write workflow.yaml: %w", err)
	}

	return nil
}

func (w Workflow) Validate(supportLangs []string) error {
	if w.Version != WorkflowVersion {
		return fmt.Errorf("unsupported workflow version: %s", w.Version)
	}

	if len(w.Sources) == 0 && len(w.Targets) == 0 {
		return fmt.Errorf("no sources or targets found")
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

func (w Workflow) GetTargetSource(target string) (*Source, string, error) {
	t, ok := w.Targets[target]
	if !ok {
		return nil, "", fmt.Errorf("target %s not found", target)
	}

	s, ok := w.Sources[t.Source]
	if ok {
		return &s, "", nil
	} else {
		return nil, t.Source, nil
	}
}

func findWorkflowFile(dir, configDir string) ([]byte, string, error) {
	if configDir == "" {
		configDir = speakeasyFolder
	}

	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	path := filepath.Join(absPath, configDir, "workflow.yaml")

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
					if configDir == speakeasyFolder {
						return findWorkflowFile(dir, genFolder)
					}

					return nil, "", ErrNotFound
				}

				// Get the parent directory of the current dir and append ".speakeasy" as we only check in side the .speakeasy dir
				path = filepath.Join(filepath.Dir(currentDir), configDir, "workflow.yaml")
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
