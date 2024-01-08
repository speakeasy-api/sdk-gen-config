package workflow_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

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
			name: "simple workflow file successfully validates",
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
			name: "workflow fails to validate with no targets",
			args: args{
				supportedLangs: []string{"typescript"},
				workflow: &workflow.Workflow{
					Version: workflow.WorkflowVersion,
				},
			},
			wantErr: fmt.Errorf("no targets found"),
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
