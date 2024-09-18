package workflow

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
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

func (o *Overlay) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Overlay is flat, so we need to unmarshal it into a map to determine if it's a document or fallbackCodeSamples
	var overlayMap map[string]interface{}
	if err := unmarshal(&overlayMap); err != nil {
		return err
	}

	if _, ok := overlayMap["fallbackCodeSamplesLanguage"]; ok {
		var fallbackCodeSamples FallbackCodeSamples
		if err := unmarshal(&fallbackCodeSamples); err != nil {
			return err
		}

		o.FallbackCodeSamples = &fallbackCodeSamples
		return nil
	}

	if _, ok := overlayMap["location"]; ok {
		var document Document
		if err := unmarshal(&document); err != nil {
			return err
		}

		o.Document = &document
		return nil
	}

	return fmt.Errorf("failed to unmarshal Overlay")
}

func (o Overlay) MarshalYAML() (interface{}, error) {
	if o.Document != nil {
		return o.Document, nil
	}

	if o.FallbackCodeSamples != nil {
		return o.FallbackCodeSamples, nil
	}

	return nil, fmt.Errorf("failed to marshal Overlay")
}

type FallbackCodeSamples struct {
	FallbackCodeSamplesLanguage string `yaml:"fallbackCodeSamplesLanguage,omitempty"`
}

type LocationString string

func (l LocationString) Resolve() string {
	if strings.HasPrefix(string(l), "$") {
		return os.ExpandEnv(string(l))
	}

	return string(l)
}

func (l LocationString) Reference() string {
	return string(l)
}

type Document struct {
	Location LocationString `yaml:"location"`
	Auth     *Auth          `yaml:",inline"`
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

type (
	SourceRegistryLocation string
	SourceRegistry         struct {
		Location SourceRegistryLocation `yaml:"location"`
		Tags     []string               `yaml:"tags,omitempty"`
	}
)

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
	if s.Output != nil {
		if len(s.Inputs) > 1 && !isYAMLFile(*s.Output) {
			return "", fmt.Errorf("when merging multiple inputs, output must be a yaml file")
		}
		return *s.Output, nil
	}

	if len(s.Inputs) == 1 && len(s.Overlays) == 0 {
		return s.handleSingleInput()
	}

	return s.generateOutputPath()
}

func (s Source) handleSingleInput() (string, error) {
	input := s.Inputs[0].Location.Resolve()
	switch getFileStatus(input) {
	case fileStatusLocal:
		return input, nil
	case fileStatusNotExists:
		return "", fmt.Errorf("input file %s does not exist", input)
	case fileStatusRemote, fileStatusRegistry:
		return s.generateRegistryPath(input)
	default:
		return "", fmt.Errorf("unknown file status for %s", input)
	}
}

func (s Source) generateRegistryPath(input string) (string, error) {
	ext := filepath.Ext(input)
	if ext == "" {
		ext = ".yaml"
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(input)))
	return filepath.Join(GetTempDir(), fmt.Sprintf("registry_%s%s", hash[:6], ext)), nil
}

func (s Source) generateOutputPath() (string, error) {
	hashInputs := func() string {
		var combined string
		for _, input := range s.Inputs {
			combined += input.Location.Resolve()
		}
		hash := sha256.Sum256([]byte(combined))
		return fmt.Sprintf("%x", hash)[:6]
	}

	// If there's only one input, we can output to the same file type as that input even if we're applying overlays
	ext := ".yaml"
	if len(s.Inputs) == 1 {
		ext = getExt(s.Inputs[0].Location.Resolve())
	}

	return filepath.Join(GetTempDir(), fmt.Sprintf("output_%s%s", hashInputs(), ext)), nil
}

func getExt(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		ext = ".yaml"
	}
	return ext
}

func isYAMLFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".yaml" || ext == ".yml"
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
		if getFileStatus(d.Location.Resolve()) != fileStatusRemote {
			return fmt.Errorf("auth is only supported for remote documents")
		}

		if err := validateSecret(d.Auth.Secret); err != nil {
			return fmt.Errorf("failed to validate authSecret: %w", err)
		}
	}

	return nil
}

func (d Document) IsRemote() bool {
	return getFileStatus(d.Location.Resolve()) == fileStatusRemote
}

func (d Document) IsSpeakeasyRegistry() bool {
	return strings.Contains(d.Location.Resolve(), "registry.speakeasyapi.dev")
}

func (f FallbackCodeSamples) Validate() error {
	if f.FallbackCodeSamplesLanguage == "" {
		return fmt.Errorf("fallbackCodeSamplesLanguage is required")
	}

	return nil
}

func (o Overlay) Validate() error {
	if o.Document != nil {
		if err := o.Document.Validate(); err != nil {
			return fmt.Errorf("failed to validate overlay document: %w", err)
		}
	}

	if o.FallbackCodeSamples != nil {
		if err := o.FallbackCodeSamples.Validate(); err != nil {
			return fmt.Errorf("failed to validate overlay fallbackCodeSamples: %w", err)
		}
	}

	if o.Document == nil && o.FallbackCodeSamples == nil {
		return fmt.Errorf("overlay must have either a document or fallbackCodeSamples")
	}

	return nil
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
	return filepath.Join(tempDir, fmt.Sprintf("downloaded_%s%s", randStringBytes(10), filepath.Ext(d.Location.Resolve())))
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
