package tests

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/speakeasy-api/sdk-gen-config/workspace"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"gopkg.in/yaml.v3"
)

const (
	TestsVersion = "0.0.1"
	testsFile    = "tests.yaml"
)

type Tests struct {
	Version string                                 `yaml:"testsVersion"`
	Tests   *orderedmap.OrderedMap[string, []Test] `yaml:"tests"`
}

type Test struct {
	Name        string                                    `yaml:"name"`
	Description string                                    `yaml:"description,omitempty"`
	Targets     []string                                  `yaml:"targets,omitempty"`
	Server      string                                    `yaml:"server,omitempty"`
	Security    yaml.Node                                 `yaml:"security,omitempty"`
	Parameters  *Parameters                               `yaml:"parameters,omitempty"`
	RequestBody *orderedmap.OrderedMap[string, yaml.Node] `yaml:"requestBody,omitempty"`
	Responses   *orderedmap.OrderedMap[string, yaml.Node] `yaml:"responses,omitempty"`

	// Internal use only
	InternalID      string                                 `yaml:"internalId,omitempty"`
	TestGroups      []string                               `yaml:"testGroups,omitempty"`
	InternalEnvVars *orderedmap.OrderedMap[string, string] `yaml:"internalEnvVars,omitempty"`
}

type Parameters struct {
	Path   *orderedmap.OrderedMap[string, yaml.Node] `yaml:"path,omitempty"`
	Query  *orderedmap.OrderedMap[string, yaml.Node] `yaml:"query,omitempty"`
	Header *orderedmap.OrderedMap[string, yaml.Node] `yaml:"header,omitempty"`
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

	for operationID, tests := range t.Tests.FromOldest() {
		if operationID == "" {
			return fmt.Errorf("empty operationId found")
		}

		for _, test := range tests {
			name := fmt.Sprintf("%s[%s]", operationID, test.Name)

			if test.Name == "" {
				return fmt.Errorf("test %s has no name", operationID)
			}

			if test.RequestBody.Len() > 1 {
				return fmt.Errorf("test %s has more than one request body", name)
			}

			if test.Responses.Len() == 0 {
				return fmt.Errorf("test %s should defined at least one response", name)
			}

			if test.Responses.Len() > 1 {
				return fmt.Errorf("test %s has more than one response code", name)
			}

			if test.Responses != nil {
				for _, responseBody := range test.Responses.FromOldest() {
					switch responseBody.Kind {
					case yaml.ScalarNode:
						if responseBody.Tag != "!!bool" {
							return fmt.Errorf("test %s has invalid response body", name)
						}
					case yaml.MappingNode:
						var contentTypes *orderedmap.OrderedMap[string, yaml.Node]
						if err := responseBody.Decode(&contentTypes); err != nil {
							return fmt.Errorf("failed to decode response body: %w", err)
						}

						if contentTypes.Len() > 1 {
							return fmt.Errorf("test %s has more than one response body", name)
						}
					default:
						return fmt.Errorf("test %s has invalid response body", name)
					}
				}
			}
		}
	}

	return nil
}

func (t Test) GetResponse(statusCode string) (*orderedmap.OrderedMap[string, yaml.Node], bool, error) {
	if t.Responses == nil {
		return nil, false, nil
	}
	resNode, ok := t.Responses.Get(statusCode)
	if !ok {
		return nil, false, nil
	}

	var res *orderedmap.OrderedMap[string, yaml.Node]
	if err := resNode.Decode(&res); err == nil {
		return res, false, nil
	}

	var assertStatusCode bool
	if err := resNode.Decode(&assertStatusCode); err != nil {
		return nil, false, fmt.Errorf("failed to decode response body: %w", err)
	}

	return nil, assertStatusCode, nil
}
