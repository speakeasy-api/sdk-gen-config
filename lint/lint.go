package lint

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"

	"github.com/speakeasy-api/sdk-gen-config/workspace"
	"gopkg.in/yaml.v3"
)

const (
	LintVersion = "1.0.0"
)

const (
	lintFile = "lint.yaml"
)

// RuleCategory is a structure that represents a category of rules.
type RuleCategory struct {
	Id          string `json:"id" yaml:"id"`                   // The category ID
	Name        string `json:"name" yaml:"name"`               // The name of the category
	Description string `json:"description" yaml:"description"` // What is the category all about?
}

// Rule is a structure that represents a rule as part of a ruleset.
type Rule struct {
	Id                 string         `json:"id,omitempty" yaml:"id,omitempty"`
	Description        string         `json:"description,omitempty" yaml:"description,omitempty"`
	Message            string         `json:"message,omitempty" yaml:"message,omitempty"`
	Given              interface{}    `json:"given,omitempty" yaml:"given,omitempty"`
	Formats            []string       `json:"formats,omitempty" yaml:"formats,omitempty"`
	Resolved           bool           `json:"resolved,omitempty" yaml:"resolved,omitempty"`
	Recommended        bool           `json:"recommended,omitempty" yaml:"recommended,omitempty"`
	Type               string         `json:"type,omitempty" yaml:"type,omitempty"`
	Severity           string         `json:"severity,omitempty" yaml:"severity,omitempty"`
	Then               interface{}    `json:"then,omitempty" yaml:"then,omitempty"`
	PrecompiledPattern *regexp.Regexp `json:"-" yaml:"-"` // regex is slow.
	RuleCategory       *RuleCategory  `json:"category,omitempty" yaml:"category,omitempty"`
	Name               string         `json:"-" yaml:"-"`
	HowToFix           string         `json:"howToFix,omitempty" yaml:"howToFix,omitempty"`
}

type Ruleset struct {
	Rulesets []string         `yaml:"rulesets"`
	Rules    map[string]*Rule `yaml:"rules"`
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
