package lint

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/daveshanley/vacuum/model"
	"gopkg.in/yaml.v3"
)

var ErrNotFound = errors.New("could not find lint.yaml")

const (
	LintVersion = "1.0.0"
)

const (
	speakeasyFolder = ".speakeasy"
	genFolder       = ".gen"
)

type Ruleset struct {
	Rulesets []string               `yaml:"rulesets"`
	Rules    map[string]*model.Rule `yaml:"rules"`
}

type Lint struct {
	Version        string             `yaml:"lintVersion"`
	DefaultRuleset string             `yaml:"defaultRuleset"`
	Rulesets       map[string]Ruleset `yaml:"rulesets"`
}

func Load(searchDirs []string) (*Lint, string, error) {
	var data []byte
	var path string
	for _, dir := range searchDirs {
		var err error
		data, path, err = findLintFile(dir, "")
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				return nil, "", err
			}
			continue
		}
		break
	}
	if data == nil {
		return nil, "", ErrNotFound
	}

	type lintHeader struct {
		Version string `yaml:"lintVersion"`
	}

	var header lintHeader
	if err := yaml.Unmarshal(data, &header); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal lint.yaml: %w", err)
	}

	if header.Version != LintVersion {
		return nil, "", fmt.Errorf("unsupported lint version: %s", header.Version)
	}

	var lint Lint
	if err := yaml.Unmarshal(data, &lint); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal lint.yaml: %w", err)
	}

	return &lint, path, nil
}

func findLintFile(dir, configDir string) ([]byte, string, error) {
	if configDir == "" {
		configDir = speakeasyFolder
	}

	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	path := filepath.Join(absPath, configDir, "lint.yaml")

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
						return findLintFile(dir, genFolder)
					}

					return nil, "", ErrNotFound
				}

				// Get the parent directory of the current dir and append ".speakeasy" as we only check in side the .speakeasy dir
				path = filepath.Join(filepath.Dir(currentDir), configDir, "lint.yaml")
				continue
			}

			return nil, "", fmt.Errorf("could not read lint.yaml: %w", err)
		}

		return data, path, nil
	}
}
