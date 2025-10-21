package workflow

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"dario.cat/mergo"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/sdk-gen-config/workspace"
	jsg "github.com/swaggest/jsonschema-go"
	"gopkg.in/yaml.v3"
)

const (
	WorkflowVersion = "1.0.0"
	workflowFile    = "workflow.yaml"
)

// Ensure you update schema/workflow.schema.json on changes
type Workflow struct {
	_                struct{}             `title:"Speakeasy Workflow Schema" additionalProperties:"false"`
	Version          string               `yaml:"workflowVersion" description:"The version of the workflow schema" const:"1.0.0" required:"true"`
	SpeakeasyVersion Version              `yaml:"speakeasyVersion,omitempty" description:"The version of the speakeasy CLI"`
	Sources          map[string]Source    `yaml:"sources" description:"A map of source names to source configurations, where the output is an OpenAPI document"`
	Targets          map[string]Target    `yaml:"targets" description:"A map of target names to target configurations, where the output is a speakeasy generation target"`
	Dependents       map[string]Dependent `yaml:"dependents,omitempty" description:"A map of dependent names to dependent configurations, used to define external repositories and their locations"` // Currently only used by one customer. Used to make rebuilding dependent SDKs that live elsewhere easier.
}

type Version string

func (Version) PrepareJSONSchema(schema *jsg.Schema) error {
	latest := "latest"
	latestDesc := "The latest version of the speakeasy CLI"
	pattern := `^\d+\.\d+\.\d+$`
	semverDesc := "A semver version of the speakeasy CLI, corresponding to https://github.com/speakeasy-api/speakeasy/releases"
	stringType := jsg.String.Type()

	schema.WithOneOf(
		jsg.SchemaOrBool{
			TypeObject: (&jsg.Schema{}).
				WithConst(latest).
				WithDescription(latestDesc).
				ToSchemaOrBool().TypeObject,
		},
		jsg.SchemaOrBool{
			TypeObject: (&jsg.Schema{}).
				WithType(stringType).
				WithPattern(pattern).
				WithDescription(semverDesc).
				ToSchemaOrBool().TypeObject,
		},
	)
	schema.Type = nil
	return nil
}

func Load(dir string) (*Workflow, string, error) {
	res, err := workspace.FindWorkspace(dir, workspace.FindWorkspaceOptions{
		FindFile:  workflowFile,
		Recursive: true,
	})
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, "", err
		}
		return nil, "", fmt.Errorf("%w in %s", err, filepath.Join(dir, workspace.SpeakeasyFolder, workflowFile))
	}

	var workflow Workflow
	if err := yaml.Unmarshal(res.Data, &workflow); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal %s: %w", workflowFile, err)
	}

	// Look for a workflow.local.yaml file and merge any set values into the workflow
	localPath := strings.Replace(res.Path, "workflow.yaml", "workflow.local.yaml", 1)
	localData, err := os.ReadFile(localPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, "", fmt.Errorf("failed to read workflow.local.yaml: %w", err)
		}
	} else {
		var localWorkflow Workflow
		if err := yaml.Unmarshal(localData, &localWorkflow); err != nil {
			return nil, "", fmt.Errorf("failed to unmarshal workflow.local.yaml: %w", err)
		}
		workflow.Merge(&localWorkflow)
	}

	return &workflow, res.Path, nil
}

// Save the workflow to the given directory, dir should generally be the root of the project, and the workflow will be saved to ${projectRoot}/.speakeasy/workflow.yaml
func Save(dir string, workflow *Workflow) error {
	data, err := yaml.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	res, err := workspace.FindWorkspace(dir, workspace.FindWorkspaceOptions{
		FindFile:  workflowFile,
		Recursive: true,
	})
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		res = &workspace.FindWorkspaceResult{
			Path: filepath.Join(dir, workspace.SpeakeasyFolder, "workflow.yaml"),
		}
	}

	if err := os.WriteFile(res.Path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write workflow.yaml: %w", err)
	}

	return nil
}

func (w Workflow) Validate(supportLangs []string) error {
	if w.Version != WorkflowVersion {
		return fmt.Errorf("unsupported workflow version: %s", w.Version)
	}

	if len(w.Sources) == 0 && len(w.Targets) == 0 {
		return fmt.Errorf("no sources or targets found")
	}

	for targetID, target := range w.Targets {
		if err := target.Validate(supportLangs, w.Sources); err != nil {
			return fmt.Errorf("failed to validate target %s: %w", targetID, err)
		}
	}

	for sourceID, source := range w.Sources {
		if err := source.Validate(); err != nil {
			return fmt.Errorf("failed to validate source %s: %w", sourceID, err)
		}
	}

	for dependentID, dependent := range w.Dependents {
		if err := dependent.Validate(); err != nil {
			return fmt.Errorf("failed to validate dependent %s: %w", dependentID, err)
		}
	}

	return nil
}

func (w Workflow) GetTargetSource(target string) (*Source, string, error) {
	t, ok := w.Targets[target]
	if !ok {
		return nil, "", fmt.Errorf("target %s not found", target)
	}

	s, ok := w.Sources[t.Source]
	if ok {
		return &s, "", nil
	} else {
		return nil, t.Source, nil
	}
}

func validateSecret(secret string) error {
	if !strings.HasPrefix(secret, "$") {
		return fmt.Errorf("secret must be a environment variable reference (ie $MY_SECRET)")
	}

	return nil
}

// Merge merges another workflow into this workflow
func (w *Workflow) Merge(other *Workflow) {
	// Merge maps using generic helper
	mergeMap(w.Sources, other.Sources)
	mergeMap(w.Targets, other.Targets)
	mergeMap(w.Dependents, other.Dependents)

	// Merge other top-level fields
	if other.Version != "" {
		w.Version = other.Version
	}
	if other.SpeakeasyVersion != "" {
		w.SpeakeasyVersion = other.SpeakeasyVersion
	}
}

// mergeMap merges two maps by using mergo on matching keys
func mergeMap[T any](base map[string]T, other map[string]T) {
	for key, otherValue := range other {
		if baseValue, exists := base[key]; exists {
			mergo.Merge(&baseValue, &otherValue, mergo.WithOverride)
			base[key] = baseValue
		} else {
			base[key] = otherValue
		}
	}
}

func (v Version) String() string {
	if v == "" || v == "latest" {
		return "latest"
	}

	if !strings.HasPrefix(string(v), "v") {
		return "v" + string(v)
	}

	return string(v)
}

func (w Workflow) Migrate() Workflow {
	return w.migrate(false)
}

func (w Workflow) MigrateNoTelemetry() Workflow {
	return w.migrate(true)
}

func (w Workflow) migrate(telemetryDisabled bool) Workflow {
	// Backfill speakeasyVersion
	if w.SpeakeasyVersion == "" {
		// This is the pinned version from the GitHub action. If it's set, backfill using it.
		if ghPinned := os.Getenv("PINNED_VERSION"); ghPinned != "" {
			w.SpeakeasyVersion = Version(ghPinned)
		} else {
			w.SpeakeasyVersion = "latest"
		}
	}

	// Add codeSamples by default, unless telemetry is disabled
	if !telemetryDisabled {
		for targetID, target := range w.Targets {
			if !slices.Contains(SupportedLanguagesUsageSnippets, target.Target) {
				continue
			}

			// Only add code samples if there's a registry source. This is mostly because we need to know an org and workspace slug
			// in order to construct the new registry location for the code samples.
			source, ok := w.Sources[target.Source]
			if !ok || source.Registry == nil {
				continue
			}

			codeSamplesRegistry := &SourceRegistry{
				Location: codeSamplesRegistryLocation(target.Target, source.Registry.Location),
			}

			if target.CodeSamples == nil {
				// Capitalize the first letter of language
				sdkLabel := fmt.Sprintf("%s (SDK)", strings.ToUpper(string(target.Target[0]))+target.Target[1:])
				target.CodeSamples = &CodeSamples{
					Registry: codeSamplesRegistry,
					Blocking: pointer.From(false),
					LabelOverride: &CodeSamplesLabelOverride{
						FixedValue: &sdkLabel,
					},
				}
			} else if !target.CodeSamples.Enabled() {
				// do nothing
			} else if target.CodeSamples.Registry == nil {
				target.CodeSamples.Registry = codeSamplesRegistry
			} else if target.CodeSamples.Blocking != nil && !*target.CodeSamples.Blocking {
				// Fix the registry location if it needs fixing.
				// We only do this when the target is not blocking because that means we likely set it up automatically.
				// We don't want to migrate a registry location that the user set up manually.
				target.CodeSamples.Registry.Location = codeSamplesRegistryUpdatedLocation(target)
			}

			w.Targets[targetID] = target
		}
	}

	return w
}

// This function cleans up any issues with the codeSamples registry location.
// It trims tags/revisions, removes duplicate "-code-samples" suffixes, and adds the target type to the namespace if it's missing
func codeSamplesRegistryUpdatedLocation(target Target) SourceRegistryLocation {
	codeSamples := target.CodeSamples

	if codeSamples.Registry == nil {
		return ""
	}

	namespace := getNamespace(codeSamples.Registry.Location)
	if namespace == "" {
		return ""
	}

	namespace = codeSamplesNamespace(namespace, target.Target)

	// Trims any tags/revisions, which should not be present in an output registry location
	return makeRegistryLocation(namespace)
}

func getNamespace(location SourceRegistryLocation) string {
	if parsed := ParseSpeakeasyRegistryReference(string(location)); parsed != nil {
		return parsed.NamespaceID
	}
	return ""
}

func codeSamplesRegistryLocation(target string, sourceRegistryURL SourceRegistryLocation) SourceRegistryLocation {
	registryDocument := ParseSpeakeasyRegistryReference(string(sourceRegistryURL))
	newNamespace := codeSamplesNamespace(registryDocument.NamespaceID, target)
	return makeRegistryLocation(newNamespace)
}

func codeSamplesNamespace(namespace, target string) string {
	// There was a bug where we were adding the -code-samples suffix repeatedly to the namespace if there was more than one target
	for strings.HasSuffix(namespace, "-code-samples") {
		namespace = strings.TrimSuffix(namespace, "-code-samples")
	}

	// Some people have their sources named things like "java-OAS"
	if strings.Contains(namespace, target) {
		return fmt.Sprintf("%s-code-samples", namespace)
	} else {
		return fmt.Sprintf("%s-%s-code-samples", namespace, target)
	}
}

func makeRegistryLocation(namespace string) SourceRegistryLocation {
	return SourceRegistryLocation(path.Join(baseRegistryURL, namespace))
}
