package lint

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"slices"
	"time"

	"github.com/speakeasy-api/sdk-gen-config/workspace"
	"gopkg.in/yaml.v3"
)

const (
	LintVersion2 = "2.0.0"
	LintVersion1 = "1.0.0"
)

const (
	lintFile = "lint.yaml"
)

// CustomRulesConfig configures custom rule loading.
type CustomRulesConfig struct {
	// Paths are glob patterns for rule files (e.g., "./rules/*.ts")
	Paths []string `json:"paths,omitempty" yaml:"paths,omitempty"`

	// Timeout is the maximum execution time per rule (default: 30s)
	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// Rule is a structure that represents a rule as part of a ruleset.
type Rule struct {
	ID       string         `json:"id,omitempty" yaml:"id,omitempty"`             // The unique identifier for the rule
	Severity string         `json:"severity,omitempty" yaml:"severity,omitempty"` // An overload for severity
	Disabled bool           `json:"disabled,omitempty" yaml:"disabled,omitempty"` // Whether the rule is disabled
	Match    *regexp.Regexp `json:"match,omitempty" yaml:"match,omitempty"`       // A regex pattern to match against
}

type Ruleset struct {
	Rulesets []string `yaml:"rulesets,omitempty"` // Rulesets to extend
	Rules    []Rule   `yaml:"rules"`              // List of rules in this ruleset to mutate or add to rules from the extended rulesets
}

func (r *Ruleset) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("unexpected kind %v for Ruleset", value.Kind)
	}

	// Handle each of the fields of Ruleset individually
	var rulesetsNode *yaml.Node
	var rulesNode *yaml.Node

	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valNode := value.Content[i+1]

		switch keyNode.Value {
		case "rulesets":
			rulesetsNode = valNode
		case "rules":
			rulesNode = valNode
		}
	}

	// Unmarshal Rulesets
	if rulesetsNode != nil {
		var rulesets []string
		if err := rulesetsNode.Decode(&rulesets); err != nil {
			return err
		}
		r.Rulesets = rulesets
	}

	// Unmarshal Rules
	if rulesNode == nil {
		return nil
	}

	// For Rules if the node is a map we need to convert it to a slice otherwise handle it as a slice
	switch rulesNode.Kind {
	case yaml.MappingNode:
		var rulesMap map[string]Rule
		if err := rulesNode.Decode(&rulesMap); err != nil {
			return err
		}
		for name, rule := range rulesMap {
			if rule.ID == "" {
				rule.ID = name
			}
			r.Rules = append(r.Rules, rule)
		}
	case yaml.SequenceNode:
		var rulesSlice []Rule
		if err := rulesNode.Decode(&rulesSlice); err != nil {
			return err
		}
		r.Rules = rulesSlice
	default:
		return fmt.Errorf("unexpected kind %v for Ruleset", rulesNode.Kind)
	}
	return nil
}

type Lint struct {
	Version        string             `yaml:"lintVersion"`
	DefaultRuleset string             `yaml:"defaultRuleset"`
	Rulesets       map[string]Ruleset `yaml:"rulesets"`
	CustomRules    *CustomRulesConfig `yaml:"customRules,omitempty"`
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

	if !slices.Contains([]string{LintVersion1, LintVersion2}, header.Version) {
		return nil, "", fmt.Errorf("unsupported lint version: %s", header.Version)
	}

	var lint Lint
	if err := yaml.Unmarshal(res.Data, &lint); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal lint.yaml: %w", err)
	}

	return &lint, res.Path, nil
}
