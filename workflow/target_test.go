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

func TestTarget_Validate(t *testing.T) {
	type args struct {
		dontCreateSource bool
		supportedLangs   []string
		target           workflow.Target
		sources          map[string]workflow.Source
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "simple target successfully validates",
			args: args{
				supportedLangs: []string{"go"},
				target: workflow.Target{
					Target: "go",
					Source: "openapi.yaml",
				},
			},
			wantErr: nil,
		},
		{
			name: "target that references a simple source successfully validates",
			args: args{
				supportedLangs: []string{"go"},
				target: workflow.Target{
					Target: "go",
					Source: "testSource",
				},
				sources: map[string]workflow.Source{
					"testSource": {
						Inputs: []workflow.Document{
							{
								Location: "openapi.yaml",
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "target with publishing successfully validates",
			args: args{
				supportedLangs: []string{"typescript"},
				target: workflow.Target{
					Target: "typescript",
					Source: "openapi.yaml",
					Publishing: &workflow.Publishing{
						NPM: &workflow.NPM{
							Token: "$TEST_TOKEN",
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "missing target fails",
			args: args{
				target: workflow.Target{},
			},
			wantErr: fmt.Errorf("target is required"),
		},
		{
			name: "target with unsupported language fails",
			args: args{
				supportedLangs: []string{"go", "typescript"},
				target: workflow.Target{
					Target: "python",
				},
			},
			wantErr: fmt.Errorf("target python is not supported"),
		},
		{
			name: "missing source fails",
			args: args{
				supportedLangs: []string{"go"},
				target: workflow.Target{
					Target: "go",
				},
			},
			wantErr: fmt.Errorf("source is required"),
		},
		{
			name: "source doesn't validate",
			args: args{
				supportedLangs: []string{"go"},
				target: workflow.Target{
					Target: "go",
					Source: "testSource",
				},
				sources: map[string]workflow.Source{
					"testSource": {
						Inputs: []workflow.Document{},
					},
				},
			},
			wantErr: fmt.Errorf("failed to validate source testSource: no inputs found"),
		},
		{
			name: "target with missing local source fails",
			args: args{
				dontCreateSource: true,
				supportedLangs:   []string{"go"},
				target: workflow.Target{
					Target: "go",
					Source: "openapi.yaml",
				},
			},
			wantErr: fmt.Errorf("source openapi.yaml does not exist"),
		},
		{
			name: "target with invalid simple publishing token fails",
			args: args{
				supportedLangs: []string{"typescript"},
				target: workflow.Target{
					Target: "typescript",
					Source: "openapi.yaml",
					Publishing: &workflow.Publishing{
						NPM: &workflow.NPM{
							Token: "some-token",
						},
					},
				},
			},
			wantErr: fmt.Errorf("failed to validate publish: failed to validate npm token: secret must be a environment variable reference (ie $MY_SECRET)"),
		},
		{
			name: "target with complex publishing config fails when non-secret is missing",
			args: args{
				supportedLangs: []string{"php"},
				target: workflow.Target{
					Target: "php",
					Source: "openapi.yaml",
					Publishing: &workflow.Publishing{
						Packagist: &workflow.Packagist{
							Username: "",
							Token:    "$TEST_TOKEN",
						},
					},
				},
			},
			wantErr: fmt.Errorf("failed to validate publish: packagist username and token must be provided"),
		},
		{
			name: "target with complex publishing config fails when token is invalid",
			args: args{
				supportedLangs: []string{"php"},
				target: workflow.Target{
					Target: "php",
					Source: "openapi.yaml",
					Publishing: &workflow.Publishing{
						Packagist: &workflow.Packagist{
							Username: "some-username",
							Token:    "some-token",
						},
					},
				},
			},
			wantErr: fmt.Errorf("failed to validate publish: failed to validate packagist token: secret must be a environment variable reference (ie $MY_SECRET)"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var workDir string

			if !tt.args.dontCreateSource {
				if filepath.Ext(tt.args.target.Source) != "" {
					var err error
					workDir, err = os.MkdirTemp("", "workflow*")
					require.NoError(t, err)

					err = createEmptyFile(filepath.Join(workDir, tt.args.target.Source))
					require.NoError(t, err)
				} else {
					for _, source := range tt.args.sources {
						var err error
						workDir, err = createLocalFiles(source)
						require.NoError(t, err)
					}
				}
			}
			if workDir != "" {
				defer os.RemoveAll(workDir)

				err := os.Chdir(workDir)
				require.NoError(t, err)
			}

			err := tt.args.target.Validate(tt.args.supportedLangs, tt.args.sources)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
