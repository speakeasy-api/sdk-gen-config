package workflow

import (
	"errors"
	"fmt"
	"github.com/AlekSi/pointer"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/speakeasy-api/sdk-gen-config/workspace"
	"gopkg.in/yaml.v3"
)

const (
	WorkflowVersion = "1.0.0"
	workflowFile    = "workflow.yaml"
)

// Ensure your update schema/workflow.schema.json on changes
type Workflow struct {
	Version          string            `yaml:"workflowVersion"`
	SpeakeasyVersion Version           `yaml:"speakeasyVersion,omitempty"`
	Sources          map[string]Source `yaml:"sources"`
	Targets          map[string]Target `yaml:"targets"`
}

type Version string

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

			if target.CodeSamples == nil {
				target.CodeSamples = &CodeSamples{
					Registry: &SourceRegistry{
						Location: codeSamplesRegistryLocation(source.Registry.Location),
					},
					Blocking: pointer.ToBool(false),
				}
			} else {
				// Fix the registry location if it needs fixing
				target.CodeSamples.Registry.Location = codeSamplesRegistryUpdatedLocation(w, target.CodeSamples)
			}

			w.Targets[targetID] = target
		}
	}

	return w
}

// For a brief time we were not properly adding -code-samples to the namespace and so we created some duplicated registry locations
// This function checks if the registry location is a duplicate and if so, updates it to include -code-samples
func codeSamplesRegistryUpdatedLocation(wf Workflow, codeSamples *CodeSamples) SourceRegistryLocation {
	if codeSamples.Registry == nil {
		return ""
	}

	namespace := getNamespace(codeSamples.Registry.Location)
	if namespace == "" {
		return ""
	}

	// Registry namespaces should be unique
	var namespaces []string
	for _, source := range wf.Sources {
		if source.Registry != nil {
			namespaces = append(namespaces, getNamespace(source.Registry.Location))
		}
	}

	for _, target := range wf.Targets {
		if target.CodeSamples != nil && target.CodeSamples != codeSamples && target.CodeSamples.Registry != nil {
			namespaces = append(namespaces, getNamespace(target.CodeSamples.Registry.Location))
		}
	}

	// For a brief time we were not properly adding -code-samples to the namespace and so we created some duplicated registry locations
	if slices.Contains(namespaces, namespace) {
		namespace += "-code-samples"
	}

	// Even if the namespace was already unique, we still want to return this because it also trims any tags/revisions,
	// which should not be present in an output registry location
	return makeRegistryLocation(namespace)
}

func getNamespace(location SourceRegistryLocation) string {
	if parsed := ParseSpeakeasyRegistryReference(string(location)); parsed != nil {
		return parsed.NamespaceID
	}
	return ""
}

func codeSamplesRegistryLocation(sourceRegistryURL SourceRegistryLocation) SourceRegistryLocation {
	registryDocument := ParseSpeakeasyRegistryReference(string(sourceRegistryURL))
	newNamespace := registryDocument.NamespaceID + "-code-samples"
	return makeRegistryLocation(newNamespace)
}

func makeRegistryLocation(namespace string) SourceRegistryLocation {
	return SourceRegistryLocation(path.Join(baseRegistryURL, namespace))
}
