package workflow

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/speakeasy-api/sdk-gen-config/workspace"
)

// Ensure your update schema/workflow.schema.json on changes
type Source struct {
	Inputs   []Document      `yaml:"inputs"`
	Overlays []Overlay       `yaml:"overlays,omitempty"`
	Output   *string         `yaml:"output,omitempty"`
	Ruleset  *string         `yaml:"ruleset,omitempty"`
	Registry *SourceRegistry `yaml:"registry,omitempty"`
}

// Either FallBackCodeSamples or Document
type Overlay struct {
	FallbackCodeSamples *FallbackCodeSamples `yaml:"fallbackCodeSamples,omitempty"`
	Document            *Document            `yaml:"document,omitempty"`
}

type FallbackCodeSamples struct {
	FallbackCodeSamplesLanguage string `yaml:"fallbackCodeSamplesLanguage,omitempty"`
}

type Document struct {
	Location string `yaml:"location"`
	Auth     *Auth  `yaml:",inline"`
}

type SpeakeasyRegistryDocument struct {
	OrganizationSlug string
	WorkspaceSlug    string
	NamespaceID      string
	NamespaceName    string
	// Reference could be tag or revision hash sha256:...
	Reference string
}

type Auth struct {
	Header string `yaml:"authHeader,omitempty"`
	Secret string `yaml:"authSecret,omitempty"`
}

type SourceRegistryLocation string
type SourceRegistry struct {
	Location SourceRegistryLocation `yaml:"location"`
	Tags     []string               `yaml:"tags,omitempty"`
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

	if s.Registry != nil {
		if err := s.Registry.Validate(); err != nil {
			return fmt.Errorf("failed to validate registry: %w", err)
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
		case fileStatusRegistry:
			return filepath.Join(GetTempDir(), fmt.Sprintf("registry_%s", randStringBytes(10))), nil
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

func (d Document) IsSpeakeasyRegistry() bool {
	return strings.Contains(d.Location, "registry.speakeasyapi.dev")
}

// Parse the location to extract the namespace ID, namespace name, and reference
// The location should be in the format registry.speakeasyapi.dev/org/workspace/name[:tag|@sha256:digest]
func ParseSpeakeasyRegistryReference(location string) *SpeakeasyRegistryDocument {
	// Parse the location to extract the organization, workspace, namespace, and reference
	// Examples:
	// registry.speakeasyapi.dev/org/workspace/name (default reference: latest)
	// registry.speakeasyapi.dev/org/workspace/name@sha256:1234567890abcdef
	// registry.speakeasyapi.dev/org/workspace/name:tag

	// Assert it starts with the registry prefix
	if !strings.HasPrefix(location, "registry.speakeasyapi.dev/") {
		return nil
	}

	// Extract the organization, workspace, and namespace
	parts := strings.Split(strings.TrimPrefix(location, "registry.speakeasyapi.dev/"), "/")
	if len(parts) != 3 {
		return nil
	}

	organizationSlug := parts[0]
	workspaceSlug := parts[1]
	suffix := parts[2]

	reference := "latest"
	namespaceName := suffix

	// Check if the suffix contains a reference
	if strings.Contains(suffix, "@") {
		// Reference is a digest
		reference = suffix[strings.Index(suffix, "@")+1:]
		namespaceName = suffix[:strings.Index(suffix, "@")]
	} else if strings.Contains(suffix, ":") {
		// Reference is a tag
		reference = suffix[strings.Index(suffix, ":")+1:]
		namespaceName = suffix[:strings.Index(suffix, ":")]
	}

	return &SpeakeasyRegistryDocument{
		OrganizationSlug: organizationSlug,
		WorkspaceSlug:    workspaceSlug,
		NamespaceID:      organizationSlug + "/" + workspaceSlug + "/" + namespaceName,
		NamespaceName:    namespaceName,
		Reference:        reference,
	}
}

func (d Document) GetTempDownloadPath(tempDir string) string {
	return filepath.Join(tempDir, fmt.Sprintf("downloaded_%s%s", randStringBytes(10), filepath.Ext(d.Location)))
}

func (d Document) GetTempRegistryDir(tempDir string) string {
	return filepath.Join(tempDir, fmt.Sprintf("registry_%s", randStringBytes(10)))
}

const namespacePrefix = "registry.speakeasyapi.dev/"

func (p SourceRegistry) Validate() error {
	if p.Location == "" {
		return fmt.Errorf("location is required")
	}

	location := p.Location.String()
	// perfectly valid for someone to add http prefixes
	location = strings.TrimPrefix(location, "https://")
	location = strings.TrimPrefix(location, "http://")

	if !strings.HasPrefix(location, namespacePrefix) {
		return fmt.Errorf("registry location must begin with %s", namespacePrefix)
	}

	if strings.Count(p.Location.Namespace(), "/") != 2 {
		return fmt.Errorf("registry location should look like %s<org>/<workspace>/<image>", namespacePrefix)
	}

	return nil
}

func (p *SourceRegistry) SetNamespace(namespace string) error {
	p.Location = SourceRegistryLocation(namespacePrefix + namespace)
	return p.Validate()
}

func (p *SourceRegistry) ParseRegistryLocation() (string, string, string, error) {
	if err := p.Validate(); err != nil {
		return "", "", "", err
	}

	location := p.Location.String()
	// perfectly valid for someone to add http prefixes
	location = strings.TrimPrefix(location, "https://")
	location = strings.TrimPrefix(location, "http://")

	subParts := strings.Split(location, namespacePrefix)
	components := strings.Split(strings.TrimSuffix(subParts[1], "/"), "/")
	return components[0], components[1], components[2], nil

}

// @<org>/<workspace>/<namespace_name> => <org>/<workspace>/<namespace_name>
func (n SourceRegistryLocation) Namespace() string {
	location := string(n)
	// perfectly valid for someone to add http prefixes
	location = strings.TrimPrefix(location, "https://")
	location = strings.TrimPrefix(location, "http://")
	return strings.TrimPrefix(location, namespacePrefix)
}

// @<org>/<workspace>/<namespace_name> => <namespace_name>
func (n SourceRegistryLocation) NamespaceName() string {
	return n.Namespace()[strings.LastIndex(n.Namespace(), "/")+1:]
}

func (n SourceRegistryLocation) String() string {
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
