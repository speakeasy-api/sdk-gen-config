package workflow

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/speakeasy-api/sdk-gen-config/workspace"
)

// Ensure you update schema/workflow.schema.json on changes
type Source struct {
	Inputs          []Document       `yaml:"inputs"`
	Overlays        []Overlay        `yaml:"overlays,omitempty"`
	Transformations []Transformation `yaml:"transformations,omitempty"`
	Output          *string          `yaml:"output,omitempty"`
	Ruleset         *string          `yaml:"ruleset,omitempty"`
	Registry        *SourceRegistry  `yaml:"registry,omitempty"`
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
		s := string(l)

		// Check for union syntax on env vars: ${var1} || ${var2} || fallback
		if strings.Contains(s, " || ") {
			parts := strings.Split(s, " || ")

			for _, part := range parts {
				part = strings.TrimSpace(part)

				// If it starts with $, try to expand it
				if strings.HasPrefix(part, "$") {
					expanded := os.ExpandEnv(part)
					if expanded != "" {
						return expanded
					}
				} else {
					// Not a variable, use as literal fallback
					return part
				}
			}
		}

		return os.ExpandEnv(s)
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

	for i, transformation := range s.Transformations {
		if err := transformation.Validate(); err != nil {
			return fmt.Errorf("failed to validate transformation %d: %w", i, err)
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
		return *s.Output, nil
	}

	if s.IsSingleInput() {
		return s.handleSingleInput()
	}

	return s.generateOutputPath()
}

func (s Source) IsSingleInput() bool {
	return len(s.Inputs) == 1 && len(s.Overlays) == 0 && len(s.Transformations) == 0
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

	return filepath.Join(GetTempDir(), fmt.Sprintf("output_%s%s", hashInputs(), s.outputExt())), nil
}

func (s Source) outputExt() string {
	if s.Output != nil {
		return getExt(*s.Output)
	}

	return getExt(s.Inputs[0].Location.Resolve())
}

func getExt(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		ext = ".yaml"
	}
	return ext
}

func GetTempDir() string {
	wd, _ := os.Getwd()

	return workspace.FindWorkspaceTempDir(wd, workspace.FindWorkspaceOptions{})
}

func (s Source) GetTempMergeLocation() string {
	return filepath.Join(GetTempDir(), fmt.Sprintf("merge_%s%s", randStringBytes(10), s.outputExt()))
}

func (s Source) GetTempOverlayLocation() string {
	return filepath.Join(GetTempDir(), fmt.Sprintf("overlay_%s%s", randStringBytes(10), s.outputExt()))
}

func (s Source) GetTempTransformLocation() string {
	return filepath.Join(GetTempDir(), fmt.Sprintf("transform_%s%s", randStringBytes(10), s.outputExt()))
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

type Transformation struct {
	RemoveUnused     *bool                    `yaml:"removeUnused,omitempty"`
	FilterOperations *FilterOperationsOptions `yaml:"filterOperations,omitempty"`
	Cleanup          *bool                    `yaml:"cleanup,omitempty"`
	Format           *bool                    `yaml:"format,omitempty"`
	Normalize        *NormalizeOptions        `yaml:"normalize,omitempty"`
}

type NormalizeOptions struct {
	PrefixItems *bool `yaml:"prefixItems,omitempty"`
}

type FilterOperationsOptions struct {
	Operations string `yaml:"operations"` // Comma-separated list of operations to filter
	Include    *bool  `yaml:"include,omitempty"`
	Exclude    *bool  `yaml:"exclude,omitempty"`
}

var transformList = []string{"removeUnused", "filterOperations", "cleanup", "format", "normalize"}

func (t Transformation) Validate() error {
	numNil := 0
	if t.RemoveUnused != nil {
		numNil++
	}
	if t.FilterOperations != nil {
		numNil++
	}
	if t.Cleanup != nil {
		numNil++
	}
	if t.Format != nil {
		numNil++
	}
	if t.Normalize != nil {
		numNil++
	}
	if numNil != 1 {
		return fmt.Errorf("transformation must have exactly one of %s", strings.Join(transformList, ", "))
	}

	if t.FilterOperations != nil {
		if len(t.FilterOperations.ParseOperations()) == 0 {
			return fmt.Errorf("filterOperations.operations must not be empty")
		}

		if t.FilterOperations.Include != nil && t.FilterOperations.Exclude != nil {
			return fmt.Errorf("filterOperations.include and filterOperations.exclude cannot both be set")
		}
	}

	return nil
}

func (f FilterOperationsOptions) ParseOperations() []string {
	var operations []string

	// If it's a faux-array, like:
	// filterOperations:
	//   operations: >
	//     - getPets
	//     - createPet
	if strings.Contains(f.Operations, "\n") {
		secondLineAndBeyond := strings.SplitN(f.Operations, "\n", 2)[1]
		pattern := regexp.MustCompile(`-\s*(\S+)`)
		matches := pattern.FindAllStringSubmatch(secondLineAndBeyond, -1)
		for _, match := range matches {
			if len(match) > 1 && match[1] != "" {
				operations = append(operations, strings.TrimSpace(match[1]))
			}
		}
	} else {
		// If it's a normal CSV
		for _, op := range strings.Split(f.Operations, ",") {
			op = strings.TrimSpace(op)
			if op != "" {
				operations = append(operations, op)
			}
		}
	}

	return operations
}
