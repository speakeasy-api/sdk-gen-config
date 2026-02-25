package workflow_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/sdk-gen-config/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflow_Load_Success(t *testing.T) {
	type args struct {
		workflowLocation string
		workflowContents string
		workingDir       string
	}
	tests := []struct {
		name string
		args args
		want *workflow.Workflow
	}{
		{
			name: "loads simple workflow file",
			args: args{
				workflowLocation: "test/.speakeasy",
				workflowContents: `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: "./openapi.yaml"
targets:
  typescript:
    target: typescript
    source: testSource
`,
				workingDir: "test",
			},
			want: &workflow.Workflow{
				Version: "1.0.0",
				Sources: map[string]workflow.Source{
					"testSource": {
						Inputs: []workflow.Document{
							{
								Location: "./openapi.yaml",
							},
						},
					},
				},
				Targets: map[string]workflow.Target{
					"typescript": {
						Target: "typescript",
						Source: "testSource",
					},
				},
			},
		},
		{
			name: "loads workflow file with target testing",
			args: args{
				workflowLocation: "test/.speakeasy",
				workflowContents: `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: "./openapi.yaml"
targets:
  typescript:
    target: typescript
    source: testSource
    testing:
      enabled: true
`,
				workingDir: "test",
			},
			want: &workflow.Workflow{
				Version: "1.0.0",
				Sources: map[string]workflow.Source{
					"testSource": {
						Inputs: []workflow.Document{
							{
								Location: "./openapi.yaml",
							},
						},
					},
				},
				Targets: map[string]workflow.Target{
					"typescript": {
						Target: "typescript",
						Source: "testSource",
						Testing: &workflow.Testing{
							Enabled: pointer.From(true),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			basePath, err := os.MkdirTemp("", "workflow*")
			require.NoError(t, err)
			defer os.RemoveAll(basePath)

			err = createTempFile(filepath.Join(basePath, tt.args.workflowLocation), "workflow.yaml", tt.args.workflowContents)
			require.NoError(t, err)

			workflowFile, workflowPath, err := workflow.Load(filepath.Join(basePath, tt.args.workingDir))
			require.NoError(t, err)

			assert.Equal(t, tt.want, workflowFile)
			assert.Contains(t, workflowPath, filepath.Join(tt.args.workflowLocation, "workflow.yaml"))
		})
	}
}

func TestWorkflow_Validate(t *testing.T) {
	type args struct {
		supportedLangs []string
		workflow       *workflow.Workflow
		createSource   bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "simple workflow file with target successfully validates",
			args: args{
				supportedLangs: []string{"typescript"},
				workflow: &workflow.Workflow{
					Version: workflow.WorkflowVersion,
					Targets: map[string]workflow.Target{
						"typescript": {
							Target: "typescript",
							Source: "./openapi.yaml",
						},
					},
				},
				createSource: true,
			},
			wantErr: nil,
		},
		{
			name: "simple workflow file with source successfully validates",
			args: args{
				workflow: &workflow.Workflow{
					Version: workflow.WorkflowVersion,
					Sources: map[string]workflow.Source{
						"testSource": {
							Inputs: []workflow.Document{
								{
									Location: "./openapi.yaml",
								},
							},
						},
					},
				},
				createSource: true,
			},
			wantErr: nil,
		},
		{
			name: "workflow successfully validates",
			args: args{
				supportedLangs: []string{"typescript"},
				workflow: &workflow.Workflow{
					Version: workflow.WorkflowVersion,
					Sources: map[string]workflow.Source{
						"testSource": {
							Inputs: []workflow.Document{
								{
									Location: "./openapi.yaml",
								},
							},
						},
					},
					Targets: map[string]workflow.Target{
						"typescript": {
							Target: "typescript",
							Source: "testSource",
						},
					},
				},
				createSource: true,
			},
			wantErr: nil,
		},
		{
			name: "workflow version is not supported",
			args: args{
				supportedLangs: []string{"typescript"},
				workflow: &workflow.Workflow{
					Version: "0.0.0",
				},
			},
			wantErr: fmt.Errorf("unsupported workflow version: 0.0.0"),
		},
		{
			name: "workflow fails to validate with no targets or sources",
			args: args{
				supportedLangs: []string{"typescript"},
				workflow: &workflow.Workflow{
					Version: workflow.WorkflowVersion,
				},
			},
			wantErr: fmt.Errorf("no sources or targets found"),
		},
		{
			name: "workflow fails if target is invalid",
			args: args{
				supportedLangs: []string{"typescript"},
				workflow: &workflow.Workflow{
					Version: workflow.WorkflowVersion,
					Targets: map[string]workflow.Target{
						"typescript": {},
					},
				},
			},
			wantErr: fmt.Errorf("failed to validate target typescript: target is required"),
		},
		{
			name: "workflow fails to validate if source is invalid",
			args: args{
				supportedLangs: []string{"typescript"},
				workflow: &workflow.Workflow{
					Version: workflow.WorkflowVersion,
					Sources: map[string]workflow.Source{
						"testSource": {
							Inputs: []workflow.Document{
								{
									Location: "http://example.com/openapi.yaml",
								},
							},
						},
						"testSource2": {
							Inputs: []workflow.Document{
								{
									Location: "./openapi1.yaml",
									Auth: &workflow.Auth{
										Header: "Authorization",
										Secret: "$AUTH_TOKEN",
									},
								},
							},
						},
					},
					Targets: map[string]workflow.Target{
						"typescript": {
							Target: "typescript",
							Source: "testSource",
						},
					},
				},
				createSource: true,
			},
			wantErr: fmt.Errorf("failed to validate source testSource2: failed to validate input 0: auth is only supported for remote documents"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.createSource {
				if len(tt.args.workflow.Sources) > 0 {
					for _, source := range tt.args.workflow.Sources {
						workDir, err := createLocalFiles(source)
						require.NoError(t, err)
						defer os.RemoveAll(workDir)

						err = os.Chdir(workDir)
						require.NoError(t, err)
					}
				} else {
					workDir, err := os.MkdirTemp("", "workflow*")
					require.NoError(t, err)

					for _, target := range tt.args.workflow.Targets {
						err = createEmptyFile(filepath.Join(workDir, target.Source))
						require.NoError(t, err)
					}

					err = os.Chdir(workDir)
					require.NoError(t, err)
				}
			}

			err := tt.args.workflow.Validate(tt.args.supportedLangs)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestWorkflow_ValidateSourceDependencies(t *testing.T) {
	tests := []struct {
		name    string
		sources map[string]workflow.Source
		wantErr string
	}{
		{
			name: "no source refs - passes",
			sources: map[string]workflow.Source{
				"a": {Inputs: []workflow.Document{{Location: "http://example.com/a.yaml"}}},
				"b": {Inputs: []workflow.Document{{Location: "http://example.com/b.yaml"}}},
			},
		},
		{
			name: "valid source ref - passes",
			sources: map[string]workflow.Source{
				"base": {
					Inputs: []workflow.Document{{Location: "http://example.com/base.yaml"}},
					Output: pointer.From("base.yaml"),
				},
				"combined": {
					Inputs: []workflow.Document{
						{Location: "source:base"},
						{Location: "http://example.com/other.yaml"},
					},
					Output: pointer.From("combined.yaml"),
				},
			},
		},
		{
			name: "diamond dependency - passes",
			sources: map[string]workflow.Source{
				"a": {Inputs: []workflow.Document{{Location: "http://example.com/a.yaml"}}, Output: pointer.From("a.yaml")},
				"b": {Inputs: []workflow.Document{{Location: "http://example.com/b.yaml"}}, Output: pointer.From("b.yaml")},
				"c": {
					Inputs: []workflow.Document{{Location: "source:a"}, {Location: "source:b"}},
					Output: pointer.From("c.yaml"),
				},
			},
		},
		{
			name: "chain A -> B -> C - passes",
			sources: map[string]workflow.Source{
				"a": {Inputs: []workflow.Document{{Location: "http://example.com/a.yaml"}}, Output: pointer.From("a.yaml")},
				"b": {Inputs: []workflow.Document{{Location: "source:a"}}, Output: pointer.From("b.yaml")},
				"c": {Inputs: []workflow.Document{{Location: "source:b"}}, Output: pointer.From("c.yaml")},
			},
		},
		{
			name: "self-reference - fails",
			sources: map[string]workflow.Source{
				"a": {
					Inputs: []workflow.Document{{Location: "source:a"}},
					Output: pointer.From("a.yaml"),
				},
			},
			wantErr: "circular source dependency detected: a -> a",
		},
		{
			name: "simple cycle A -> B -> A - fails",
			sources: map[string]workflow.Source{
				"a": {Inputs: []workflow.Document{{Location: "source:b"}}, Output: pointer.From("a.yaml")},
				"b": {Inputs: []workflow.Document{{Location: "source:a"}}, Output: pointer.From("b.yaml")},
			},
			wantErr: "circular source dependency detected:",
		},
		{
			name: "missing source ref - fails",
			sources: map[string]workflow.Source{
				"a": {
					Inputs: []workflow.Document{{Location: "source:nonexistent"}},
					Output: pointer.From("a.yaml"),
				},
			},
			wantErr: `source "a" references unknown source "nonexistent"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := workflow.Workflow{
				Version: workflow.WorkflowVersion,
				Sources: tt.sources,
			}
			err := wf.ValidateSourceDependencies()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestMigrate_Success(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		expected string
	}{{
		name: "migrates a simple workflow",
		in: `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: ./openapi.yaml
    registry:
      location: registry.speakeasyapi.dev/org/workspace/testSource
targets:
  typescript:
    target: typescript
    source: testSource
`,
		expected: `workflowVersion: 1.0.0
speakeasyVersion: latest
sources:
    testSource:
        inputs:
            - location: ./openapi.yaml
        registry:
            location: registry.speakeasyapi.dev/org/workspace/testSource
targets:
    typescript:
        target: typescript
        source: testSource
        codeSamples:
            registry:
                location: registry.speakeasyapi.dev/org/workspace/testSource-typescript-code-samples
            labelOverride:
                fixedValue: Typescript (SDK)
            blocking: false
`,
	}, {
		name: "doesn't migrate a blocking workflow with a registry location",
		in: `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: ./openapi.yaml
    registry:
      location: registry.speakeasyapi.dev/org/workspace/testSource
targets:
  typescript:
    target: typescript
    source: testSource
    codeSamples:
      registry:
        location: registry.speakeasyapi.dev/org/workspace/testSource-custom-code-samples
`,
		expected: `workflowVersion: 1.0.0
speakeasyVersion: latest
sources:
    testSource:
        inputs:
            - location: ./openapi.yaml
        registry:
            location: registry.speakeasyapi.dev/org/workspace/testSource
targets:
    typescript:
        target: typescript
        source: testSource
        codeSamples:
            registry:
                location: registry.speakeasyapi.dev/org/workspace/testSource-custom-code-samples
`,
	}, {
		name: "migrates a workflow with a tagged registry location",
		in: `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: ./openapi.yaml
    registry:
      location: registry.speakeasyapi.dev/org/workspace/testSource:main
targets:
  typescript:
    target: typescript
    source: testSource
`,
		expected: `workflowVersion: 1.0.0
speakeasyVersion: latest
sources:
    testSource:
        inputs:
            - location: ./openapi.yaml
        registry:
            location: registry.speakeasyapi.dev/org/workspace/testSource:main
targets:
    typescript:
        target: typescript
        source: testSource
        codeSamples:
            registry:
                location: registry.speakeasyapi.dev/org/workspace/testSource-typescript-code-samples
            labelOverride:
                fixedValue: Typescript (SDK)
            blocking: false
`,
	}, {
		name: "migrates a workflow with a duplicate registry location",
		in: `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: ./openapi.yaml
    registry:
      location: registry.speakeasyapi.dev/org/workspace/testSource
targets:
  typescript:
    target: typescript
    source: testSource
    codeSamples:
      registry:
        location: registry.speakeasyapi.dev/org/workspace/testSource
      blocking: false
`,
		expected: `workflowVersion: 1.0.0
speakeasyVersion: latest
sources:
    testSource:
        inputs:
            - location: ./openapi.yaml
        registry:
            location: registry.speakeasyapi.dev/org/workspace/testSource
targets:
    typescript:
        target: typescript
        source: testSource
        codeSamples:
            registry:
                location: registry.speakeasyapi.dev/org/workspace/testSource-typescript-code-samples
            blocking: false
`,
	}, {
		name: "migrates a workflow with a code samples output and a source with a name that contains the target",
		in: `workflowVersion: 1.0.0
sources:
  testSource-typescript:
    inputs:
      - location: ./openapi.yaml
    registry:
      location: registry.speakeasyapi.dev/org/workspace/testSource
targets:
  typescript:
    target: typescript
    source: testSource-typescript
    codeSamples:
      output: output.yaml
`,
		expected: `workflowVersion: 1.0.0
speakeasyVersion: latest
sources:
    testSource-typescript:
        inputs:
            - location: ./openapi.yaml
        registry:
            location: registry.speakeasyapi.dev/org/workspace/testSource
targets:
    typescript:
        target: typescript
        source: testSource-typescript
        codeSamples:
            output: output.yaml
            registry:
                location: registry.speakeasyapi.dev/org/workspace/testSource-typescript-code-samples
`,
	}, {
		name: "migrates a workflow with multiple targets, some with multiple -code-samples suffixes",
		in: `workflowVersion: 1.0.0
speakeasyVersion: latest
sources:
    Acuvity-OAS:
        inputs:
            - location: ./apex-openapi.yaml
        registry:
            location: registry.speakeasyapi.dev/acuvity-9dx/acuvity/acuvity-oas
targets:
    golang:
        target: go
        source: Acuvity-OAS
        output: acuvity-go
        codeSamples:
            registry:
                location: registry.speakeasyapi.dev/acuvity-9dx/acuvity/acuvity-oas-code-samples
            blocking: false
    python:
        target: python
        source: Acuvity-OAS
        output: acuvity-python
        codeSamples:
            registry:
                location: registry.speakeasyapi.dev/acuvity-9dx/acuvity/acuvity-oas-code-samples-code-samples
            blocking: false
    typescript:
        target: typescript
        source: Acuvity-OAS
        output: acuvity-ts
        codeSamples:
            registry:
                location: registry.speakeasyapi.dev/acuvity-9dx/acuvity/acuvity-oas-code-samples-code-samples
            blocking: false`,
		expected: `workflowVersion: 1.0.0
speakeasyVersion: latest
sources:
    Acuvity-OAS:
        inputs:
            - location: ./apex-openapi.yaml
        registry:
            location: registry.speakeasyapi.dev/acuvity-9dx/acuvity/acuvity-oas
targets:
    golang:
        target: go
        source: Acuvity-OAS
        output: acuvity-go
        codeSamples:
            registry:
                location: registry.speakeasyapi.dev/acuvity-9dx/acuvity/acuvity-oas-go-code-samples
            blocking: false
    python:
        target: python
        source: Acuvity-OAS
        output: acuvity-python
        codeSamples:
            registry:
                location: registry.speakeasyapi.dev/acuvity-9dx/acuvity/acuvity-oas-python-code-samples
            blocking: false
    typescript:
        target: typescript
        source: Acuvity-OAS
        output: acuvity-ts
        codeSamples:
            registry:
                location: registry.speakeasyapi.dev/acuvity-9dx/acuvity/acuvity-oas-typescript-code-samples
            blocking: false
`,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var workflow workflow.Workflow
			require.NoError(t, yaml.Unmarshal([]byte(tt.in), &workflow))

			workflow = workflow.Migrate()

			actual, err := yaml.Marshal(workflow)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, string(actual))
		})
	}
}

func createTempFile(dir string, fileName, contents string) error {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	if contents != "" {
		tmpFile := filepath.Join(dir, fileName)
		if err := os.WriteFile(tmpFile, []byte(contents), os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

func TestWorkflow_LoadWithLocal_Success(t *testing.T) {
	type args struct {
		workflowLocation string
		workflowContents string
		localContents    string
		workingDir       string
	}
	tests := []struct {
		name string
		args args
		want *workflow.Workflow
	}{
		{
			name: "loads workflow with local override for sources",
			args: args{
				workflowLocation: "test/.speakeasy",
				workflowContents: `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: "./openapi.yaml"
targets:
  typescript:
    target: typescript
    source: testSource
`,
				localContents: `sources:
  testSource:
    inputs:
      - location: "./local-openapi.yaml"
`,
				workingDir: "test",
			},
			want: &workflow.Workflow{
				Version: "1.0.0",
				Sources: map[string]workflow.Source{
					"testSource": {
						Inputs: []workflow.Document{
							{
								Location: "./local-openapi.yaml",
							},
						},
					},
				},
				Targets: map[string]workflow.Target{
					"typescript": {
						Target: "typescript",
						Source: "testSource",
					},
				},
			},
		},
		{
			name: "loads workflow with local override for targets",
			args: args{
				workflowLocation: "test/.speakeasy",
				workflowContents: `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: "./openapi.yaml"
targets:
  typescript:
    target: typescript
    source: testSource
    output: ./ts-sdk
`,
				localContents: `targets:
  typescript:
    output: ./local-ts-sdk
    testing:
      enabled: true
`,
				workingDir: "test",
			},
			want: &workflow.Workflow{
				Version: "1.0.0",
				Sources: map[string]workflow.Source{
					"testSource": {
						Inputs: []workflow.Document{
							{
								Location: "./openapi.yaml",
							},
						},
					},
				},
				Targets: map[string]workflow.Target{
					"typescript": {
						Target: "typescript",
						Source: "testSource",
						Output: pointer.From("./local-ts-sdk"),
						Testing: &workflow.Testing{
							Enabled: pointer.From(true),
						},
					},
				},
			},
		},
		{
			name: "loads workflow with local override adding new targets",
			args: args{
				workflowLocation: "test/.speakeasy",
				workflowContents: `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: "./openapi.yaml"
targets:
  typescript:
    target: typescript
    source: testSource
`,
				localContents: `targets:
  python:
    target: python
    source: testSource
    output: ./py-sdk
`,
				workingDir: "test",
			},
			want: &workflow.Workflow{
				Version: "1.0.0",
				Sources: map[string]workflow.Source{
					"testSource": {
						Inputs: []workflow.Document{
							{
								Location: "./openapi.yaml",
							},
						},
					},
				},
				Targets: map[string]workflow.Target{
					"typescript": {
						Target: "typescript",
						Source: "testSource",
					},
					"python": {
						Target: "python",
						Source: "testSource",
						Output: pointer.From("./py-sdk"),
					},
				},
			},
		},
		{
			name: "loads workflow with local override for workflow version and speakeasy version",
			args: args{
				workflowLocation: "test/.speakeasy",
				workflowContents: `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: "./openapi.yaml"
targets:
  typescript:
    target: typescript
    source: testSource
`,
				localContents: `workflowVersion: 1.0.0
speakeasyVersion: v1.2.3
`,
				workingDir: "test",
			},
			want: &workflow.Workflow{
				Version:          "1.0.0",
				SpeakeasyVersion: "v1.2.3",
				Sources: map[string]workflow.Source{
					"testSource": {
						Inputs: []workflow.Document{
							{
								Location: "./openapi.yaml",
							},
						},
					},
				},
				Targets: map[string]workflow.Target{
					"typescript": {
						Target: "typescript",
						Source: "testSource",
					},
				},
			},
		},
		{
			name: "loads workflow with complex local override merging nested structures",
			args: args{
				workflowLocation: "test/.speakeasy",
				workflowContents: `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: "./openapi.yaml"
    registry:
      location: "registry.example.com/org/workspace/api"
      tags: 
        - abc
targets:
  typescript:
    target: typescript
    source: testSource
    output: ./ts-sdk
    codeSamples:
      output: ./code-samples.yaml
      blocking: false
`,
				localContents: `sources:
  testSource:
    registry:
      location: OVERRIDE
targets:
  typescript:
    codeSamples:
      blocking: true
      registry:
        location: "registry.local.dev/org/workspace/samples"
`,
				workingDir: "test",
			},
			want: &workflow.Workflow{
				Version: "1.0.0",
				Sources: map[string]workflow.Source{
					"testSource": {
						Inputs: []workflow.Document{
							{
								Location: "./openapi.yaml",
							},
						},
						Registry: &workflow.SourceRegistry{
							Location: "OVERRIDE",
							Tags:     []string{"abc"},
						},
					},
				},
				Targets: map[string]workflow.Target{
					"typescript": {
						Target: "typescript",
						Source: "testSource",
						Output: pointer.From("./ts-sdk"),
						CodeSamples: &workflow.CodeSamples{
							Output:   "./code-samples.yaml",
							Blocking: pointer.From(true),
							Registry: &workflow.SourceRegistry{
								Location: "registry.local.dev/org/workspace/samples",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			basePath, err := os.MkdirTemp("", "workflow*")
			require.NoError(t, err)
			defer os.RemoveAll(basePath)

			err = createTempFile(filepath.Join(basePath, tt.args.workflowLocation), "workflow.yaml", tt.args.workflowContents)
			require.NoError(t, err)

			err = createTempFile(filepath.Join(basePath, tt.args.workflowLocation), "workflow.local.yaml", tt.args.localContents)
			require.NoError(t, err)

			workflowFile, workflowPath, err := workflow.Load(filepath.Join(basePath, tt.args.workingDir))
			require.NoError(t, err)

			assert.Equal(t, tt.want, workflowFile)
			assert.Contains(t, workflowPath, filepath.Join(tt.args.workflowLocation, "workflow.yaml"))
		})
	}
}

func TestWorkflow_LoadWithLocal_NoLocalFile(t *testing.T) {
	basePath, err := os.MkdirTemp("", "workflow*")
	require.NoError(t, err)
	defer os.RemoveAll(basePath)

	workflowContents := `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: "./openapi.yaml"
targets:
  typescript:
    target: typescript
    source: testSource
`

	err = createTempFile(filepath.Join(basePath, "test/.speakeasy"), "workflow.yaml", workflowContents)
	require.NoError(t, err)

	workflowFile, workflowPath, err := workflow.Load(filepath.Join(basePath, "test"))
	require.NoError(t, err)

	expected := &workflow.Workflow{
		Version: "1.0.0",
		Sources: map[string]workflow.Source{
			"testSource": {
				Inputs: []workflow.Document{
					{
						Location: "./openapi.yaml",
					},
				},
			},
		},
		Targets: map[string]workflow.Target{
			"typescript": {
				Target: "typescript",
				Source: "testSource",
			},
		},
	}

	assert.Equal(t, expected, workflowFile)
	assert.Contains(t, workflowPath, filepath.Join("test/.speakeasy", "workflow.yaml"))
}

func TestWorkflow_LoadWithLocal_InvalidLocalFile(t *testing.T) {
	basePath, err := os.MkdirTemp("", "workflow*")
	require.NoError(t, err)
	defer os.RemoveAll(basePath)

	workflowContents := `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: "./openapi.yaml"
`

	invalidLocalContents := `invalid yaml content: [
`

	err = createTempFile(filepath.Join(basePath, "test/.speakeasy"), "workflow.yaml", workflowContents)
	require.NoError(t, err)

	err = createTempFile(filepath.Join(basePath, "test/.speakeasy"), "workflow.local.yaml", invalidLocalContents)
	require.NoError(t, err)

	_, _, err = workflow.Load(filepath.Join(basePath, "test"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal workflow.local.yaml")
}

func TestWorkflow_Merge_Method(t *testing.T) {
	baseWorkflow := &workflow.Workflow{
		Version:          "1.0.0",
		SpeakeasyVersion: "latest",
		Sources: map[string]workflow.Source{
			"source1": {
				Inputs: []workflow.Document{
					{Location: "./api.yaml"},
				},
			},
		},
		Targets: map[string]workflow.Target{
			"typescript": {
				Target: "typescript",
				Source: "source1",
				Output: pointer.From("./ts-sdk"),
			},
		},
		Dependents: map[string]workflow.Dependent{
			"dep1": {
				Location: "source1",
			},
		},
	}

	localWorkflow := &workflow.Workflow{
		Version:          "1.0.0",
		SpeakeasyVersion: "v1.2.3",
		Sources: map[string]workflow.Source{
			"source1": {
				Registry: &workflow.SourceRegistry{
					Location: "registry.local.dev/org/workspace/api",
				},
			},
			"source2": {
				Inputs: []workflow.Document{
					{Location: "./local-api.yaml"},
				},
			},
		},
		Targets: map[string]workflow.Target{
			"typescript": {
				Testing: &workflow.Testing{
					Enabled: pointer.From(true),
				},
			},
			"python": {
				Target: "python",
				Source: "source2",
				Output: pointer.From("./py-sdk"),
			},
		},
		Dependents: map[string]workflow.Dependent{
			"dep2": {
				Location: "source2",
			},
		},
	}

	baseWorkflow.Merge(localWorkflow)

	assert.Equal(t, "v1.2.3", string(baseWorkflow.SpeakeasyVersion))

	assert.Len(t, baseWorkflow.Sources, 2)
	assert.Equal(t, "./api.yaml", string(baseWorkflow.Sources["source1"].Inputs[0].Location))
	assert.Equal(t, "registry.local.dev/org/workspace/api", string(baseWorkflow.Sources["source1"].Registry.Location))
	assert.Equal(t, "./local-api.yaml", string(baseWorkflow.Sources["source2"].Inputs[0].Location))

	assert.Len(t, baseWorkflow.Targets, 2)
	assert.Equal(t, "typescript", baseWorkflow.Targets["typescript"].Target)
	assert.Equal(t, "./ts-sdk", *baseWorkflow.Targets["typescript"].Output)
	assert.True(t, *baseWorkflow.Targets["typescript"].Testing.Enabled)
	assert.Equal(t, "python", baseWorkflow.Targets["python"].Target)
	assert.Equal(t, "./py-sdk", *baseWorkflow.Targets["python"].Output)

	assert.Len(t, baseWorkflow.Dependents, 2)
	assert.Equal(t, "source1", baseWorkflow.Dependents["dep1"].Location)
	assert.Equal(t, "source2", baseWorkflow.Dependents["dep2"].Location)
}

func TestWorkflow_LoadWithRemote_WithAuth(t *testing.T) {
	basePath, err := os.MkdirTemp("", "workflow*")
	require.NoError(t, err)
	defer os.RemoveAll(basePath)

	workflowContents := `workflowVersion: 1.0.0
sources:
  testSource:
    inputs:
      - location: "http://example.com/openapi.yaml"
        authHeader: Authorization
        authSecret: $AUTH_TOKEN
`

	err = createTempFile(filepath.Join(basePath, "test/.speakeasy"), "workflow.yaml", workflowContents)
	require.NoError(t, err)

	workflowFile, _, err := workflow.Load(filepath.Join(basePath, "test"))
	require.NoError(t, err)

	assert.Equal(t, workflowFile.Sources["testSource"].Inputs[0].Auth, &workflow.Auth{
		Header: "Authorization",
		Secret: "$AUTH_TOKEN",
	})
}
