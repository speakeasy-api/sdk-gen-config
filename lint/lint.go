package lint

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/daveshanley/vacuum/model"
	"github.com/speakeasy-api/sdk-gen-config/workspace"
	"gopkg.in/yaml.v3"
)

const (
	LintVersion = "1.0.0"
)

const (
	lintFile = "lint.yaml"
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
	var res *workspace.FindWorkspaceResult

	dirsToSearch := map[string]bool{}

	for _, dir := range searchDirs {
		dirsToSearch[dir] = true
	}

	// Allow searching in the user's home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		dirsToSearch[homeDir] = false
	}

	for dir, allowRecursive := range dirsToSearch {
		var err error

		res, err = workspace.FindWorkspace(dir, workspace.FindWorkspaceOptions{
			FindFile:  lintFile,
			Recursive: allowRecursive,
		})
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, "", err
			}
			continue
		}

		break
	}
	if res == nil || res.Data == nil {
		return nil, "", fs.ErrNotExist
	}

	type lintHeader struct {
		Version string `yaml:"lintVersion"`
	}

	var header lintHeader
	if err := yaml.Unmarshal(res.Data, &header); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal lint.yaml: %w", err)
	}

	if header.Version != LintVersion {
		return nil, "", fmt.Errorf("unsupported lint version: %s", header.Version)
	}

	var lint Lint
	if err := yaml.Unmarshal(res.Data, &lint); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal lint.yaml: %w", err)
	}

	return &lint, res.Path, nil
}
