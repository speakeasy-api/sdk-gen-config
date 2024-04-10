package workflow

import (
	"fmt"
	"github.com/speakeasy-api/sdk-gen-config/workspace"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Ensure your update schema/workflow.schema.json on changes
type Source struct {
	Inputs   []Document        `yaml:"inputs"`
	Overlays []Document        `yaml:"overlays,omitempty"`
	Output   *string           `yaml:"output,omitempty"`
	Ruleset  *string           `yaml:"ruleset,omitempty"`
	Publish  *SourcePublishing `yaml:"publish,omitempty"`
}

type Document struct {
	Location string `yaml:"location"`
	Auth     *Auth  `yaml:",inline"`
}

type Auth struct {
	Header string `yaml:"authHeader,omitempty"`
	Secret string `yaml:"authSecret,omitempty"`
}

type Location string
type SourcePublishing struct {
	Location Location `yaml:"location"`
	Tags     []string `yaml:"tags,omitempty"`
}

func (s Source) Validate() error {
	if len(s.Inputs) == 0 {
		return fmt.Errorf("no inputs found")
	}

	for i, input := range s.Inputs {
		if err := input.Validate(); err != nil {
			return fmt.Errorf("failed to validate input %d: %w", i, err)
		}
	}

	for i, overlay := range s.Overlays {
		if err := overlay.Validate(); err != nil {
			return fmt.Errorf("failed to validate overlay %d: %w", i, err)
		}
	}

	if s.Publish != nil {
		if err := s.Publish.Validate(); err != nil {
			return fmt.Errorf("failed to validate publish: %w", err)
		}
	}

	_, err := s.GetOutputLocation()
	if err != nil {
		return fmt.Errorf("failed to get output location: %w", err)
	}

	return nil
}

func (s Source) GetOutputLocation() (string, error) {
	// If we have an output location, we can just return that
	if s.Output != nil {
		output := *s.Output

		ext := filepath.Ext(output)
		if len(s.Inputs) > 1 && !slices.Contains([]string{".yaml", ".yml"}, ext) {
			return "", fmt.Errorf("when merging multiple inputs, output must be a yaml file")
		}

		return output, nil
	}

	ext := ".yaml"

	// If we only have a single input, no overlays and its a local path, we can just use that
	if len(s.Inputs) == 1 && len(s.Overlays) == 0 {
		inputFile := s.Inputs[0].Location

		switch getFileStatus(inputFile) {
		case fileStatusLocal:
			return inputFile, nil
		case fileStatusNotExists:
			return "", fmt.Errorf("input file %s does not exist", inputFile)
		case fileStatusRemote:
			ext = filepath.Ext(inputFile)
			if ext == "" {
				ext = ".yaml"
			}
		}
	}

	// Otherwise output will go to a temp file
	return filepath.Join(GetTempDir(), fmt.Sprintf("output_%s%s", randStringBytes(10), ext)), nil
}

func GetTempDir() string {
	wd, _ := os.Getwd()

	return workspace.FindWorkspaceTempDir(wd, workspace.FindWorkspaceOptions{})
}

func (s Source) GetTempMergeLocation() string {
	return filepath.Join(GetTempDir(), fmt.Sprintf("merge_%s.yaml", randStringBytes(10)))
}

func (s Source) GetTempOverlayLocation() string {
	return filepath.Join(GetTempDir(), fmt.Sprintf("overlay_%s.yaml", randStringBytes(10)))
}

func (d Document) Validate() error {
	if d.Location == "" {
		return fmt.Errorf("location is required")
	}

	if d.Auth != nil {
		if getFileStatus(d.Location) != fileStatusRemote {
			return fmt.Errorf("auth is only supported for remote documents")
		}

		if err := validateSecret(d.Auth.Secret); err != nil {
			return fmt.Errorf("failed to validate authSecret: %w", err)
		}
	}

	return nil
}

func (d Document) IsRemote() bool {
	return getFileStatus(d.Location) == fileStatusRemote
}

func (d Document) GetTempDownloadPath(tempDir string) string {
	return filepath.Join(tempDir, fmt.Sprintf("downloaded_%s%s", randStringBytes(10), filepath.Ext(d.Location)))
}

const namespacePrefix = "@"

func (p SourcePublishing) Validate() error {
	if p.Location == "" {
		return fmt.Errorf("location is required")
	}

	if !strings.HasPrefix(p.Location.String(), namespacePrefix) {
		return fmt.Errorf("publish location must begin with %s", namespacePrefix)
	}

	if strings.Count(p.Location.Namespace(), "/") != 2 {
		return fmt.Errorf("publish location should look like %s<org>/<workspace>/<image>", namespacePrefix)
	}

	return nil
}

func (p SourcePublishing) SetNamespace(namespace string) error {
	p.Location = Location(namespacePrefix + namespace)
	return p.Validate()
}

// @<org>/<workspace>/<namespace_name> => <org>/<workspace>/<namespace_name>
func (n Location) Namespace() string {
	return strings.TrimPrefix(n.String(), namespacePrefix)
}

// @<org>/<workspace>/<image> => <namespace_name>
func (n Location) NamespaceName() string {
	return n.Namespace()[strings.LastIndex(n.Namespace(), "/")+1:]
}

func (n Location) String() string {
	return string(n)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var randStringBytes = func(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
