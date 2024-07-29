package tests

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/speakeasy-api/sdk-gen-config/workspace"
	"gopkg.in/yaml.v3"
)

const (
	TestsVersion = "0.0.1"
	testsFile    = "tests.yaml"
)

type Tests struct {
	Version string            `yaml:"testsVersion"`
	Tests   map[string][]Test `yaml:"tests"`
}

type Test struct {
	Name        string                          `yaml:"name"`
	Description string                          `yaml:"description,omitempty"`
	Targets     []string                        `yaml:"targets,omitempty"`
	Server      string                          `yaml:"server,omitempty"`
	Security    yaml.Node                       `yaml:"security,omitempty"`
	Parameters  *Parameters                     `yaml:"parameters,omitempty"`
	RequestBody map[string]yaml.Node            `yaml:"requestBody,omitempty"`
	Responses   map[string]map[string]yaml.Node `yaml:"responses,omitempty"`

	// Internal use only
	InternalID string   `yaml:"internalId,omitempty"`
	TestGroups []string `yaml:"testGroups,omitempty"`
}

type Parameters struct {
	Path   map[string]yaml.Node `yaml:"path,omitempty"`
	Query  map[string]yaml.Node `yaml:"query,omitempty"`
	Header map[string]yaml.Node `yaml:"header,omitempty"`
}

func Load(dir string) (*Tests, string, error) {
	res, err := workspace.FindWorkspace(dir, workspace.FindWorkspaceOptions{
		FindFile:  testsFile,
		Recursive: true,
	})
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, "", err
		}
		return nil, "", fmt.Errorf("%w in %s", err, filepath.Join(dir, workspace.SpeakeasyFolder, testsFile))
	}

	var tests Tests
	if err := yaml.Unmarshal(res.Data, &tests); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal %s: %w", testsFile, err)
	}

	if err := tests.Validate(); err != nil {
		return nil, "", err
	}

	return &tests, res.Path, nil
}

func (t Tests) Validate() error {
	if t.Version != TestsVersion {
		return fmt.Errorf("unsupported tests version: %s", t.Version)
	}

	for operationID, tests := range t.Tests {
		if operationID == "" {
			return fmt.Errorf("empty operationId found")
		}

		for _, test := range tests {
			name := fmt.Sprintf("%s[%s]", operationID, test.Name)

			if test.Name == "" {
				return fmt.Errorf("test %s has no name", operationID)
			}

			if len(test.RequestBody) > 1 {
				return fmt.Errorf("test %s has more than one request body", name)
			}

			if len(test.Responses) > 1 {
				return fmt.Errorf("test %s has more than one response code", name)
			}

			if test.Responses != nil {
				for _, responseBody := range test.Responses {
					if len(responseBody) > 1 {
						return fmt.Errorf("test %s has more than one response body", name)
					}
				}
			}
		}
	}

	return nil
}
