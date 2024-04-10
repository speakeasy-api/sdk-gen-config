package workflow_test

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/speakeasy-api/sdk-gen-config/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSource_Validate(t *testing.T) {
	type args struct {
		source workflow.Source
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "simple source successfully validates",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "simple source of reference path successfully validates",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "~/openapi.yaml",
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "simple source of absolute path",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "/openapi.yaml",
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "simple source successfully validates if output matches input",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
					Output: pointer.ToString("openapi.yaml"),
				},
			},
			wantErr: nil,
		},
		{
			name: "simple source fails if local input does not exist",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
				},
			},
			wantErr: fmt.Errorf("failed to get output location: input file openapi.yaml does not exist"),
		},
		{
			name: "source with multiple documents successfully validates",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
						{
							Location: "openapi1.yaml",
						},
						{
							Location: "http://example.com/openapi.yaml",
						},
						{
							Location: "http://example.com/openapi1.yaml",
							Auth: &workflow.Auth{
								Header: "Authorization",
								Secret: "$AUTH_TOKEN",
							},
						},
					},
					Overlays: []workflow.Document{
						{
							Location: "overlay.yaml",
						},
						{
							Location: "http://example.com/overlay.yaml",
						},
					},
					Output: pointer.ToString("openapi.yaml"),
				},
			},
			wantErr: nil,
		},
		{
			name: "source with multiple overlays succeeds if output is not yaml",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
					Overlays: []workflow.Document{
						{
							Location: "overlay.yaml",
						},
						{
							Location: "overlay.yaml",
						},
					},
					Output: pointer.ToString("openapi.json"),
				},
			},
			wantErr: nil,
		},
		{
			name: "source with multiple merged documents fails if output is not yaml",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
						{
							Location: "openapi.yaml",
						},
					},
					Overlays: []workflow.Document{
						{
							Location: "overlay.yaml",
						},
					},
					Output: pointer.ToString("openapi.json"),
				},
			},
			wantErr: fmt.Errorf("failed to get output location: when merging multiple inputs, output must be a yaml file"),
		},
		{
			name: "fails with no inputs",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{},
					Overlays: []workflow.Document{
						{
							Location: "overlay.yaml",
						},
					},
				},
			},
			wantErr: fmt.Errorf("no inputs found"),
		},
		{
			name: "input fails with no location",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{},
					},
				},
			},
			wantErr: fmt.Errorf("failed to validate input 0: location is required"),
		},
		{
			name: "local input fails with auth details",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
							Auth: &workflow.Auth{
								Header: "Authorization",
								Secret: "$AUTH_TOKEN",
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf("failed to validate input 0: auth is only supported for remote documents"),
		},
		{
			name: "remote input fails when secret is not an env var ref",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "http://example.com/openapi.yaml",
							Auth: &workflow.Auth{
								Header: "Authorization",
								Secret: "some-secret",
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf("failed to validate input 0: failed to validate authSecret: secret must be a environment variable reference (ie $MY_SECRET)"),
		},
		{
			name: "overlay fails with no location",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
					Overlays: []workflow.Document{
						{},
					},
				},
			},
			wantErr: fmt.Errorf("failed to validate overlay 0: location is required"),
		},
		{
			name: "publish success",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
					Publish: &workflow.SourcePublishing{
						Location: "@org/workspace/image",
					},
				},
			},
		},
		{
			name: "publish fails with invalid location",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
					Publish: &workflow.SourcePublishing{
						Location: "@not-enough-parts",
					},
				},
			},
			wantErr: fmt.Errorf("failed to validate publish: publish location should look like @<org>/<workspace>/<image>"),
		},
		{
			name: "publish fails with no location",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
					Publish: &workflow.SourcePublishing{},
				},
			},
			wantErr: fmt.Errorf("failed to validate publish: location is required"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr == nil {
				workDir, err := createLocalFiles(tt.args.source)
				require.NoError(t, err)
				defer os.RemoveAll(workDir)

				err = os.Chdir(workDir)
				require.NoError(t, err)
			}

			err := tt.args.source.Validate()
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSource_GetOutputLocation(t *testing.T) {
	workflow.SetRandStringBytesFunc(func(n int) string {
		return "testrandomstring"
	})

	type args struct {
		source workflow.Source
	}
	tests := []struct {
		name               string
		args               args
		wantOutputLocation string
	}{
		{
			name: "simple source returns input location as output location",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
				},
			},
			wantOutputLocation: "openapi.yaml",
		},
		{
			name: "simple remote source returns auto-generated output location",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "http://example.com/openapi.json",
						},
					},
				},
			},
			wantOutputLocation: ".speakeasy/temp/output_testrandomstring.json",
		},
		{
			name: "simple remote source without extension returns auto-generated output location assumed to be yaml",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "http://example.com/openapi",
						},
					},
				},
			},
			wantOutputLocation: ".speakeasy/temp/output_testrandomstring.yaml",
		},
		{
			name: "source with multiple inputs returns specified output location",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
						{
							Location: "openapi1.yaml",
						},
					},
					Output: pointer.ToString("merged.yaml"),
				},
			},
			wantOutputLocation: "merged.yaml",
		},
		{
			name: "source with multiple inputs returns auto-generated output location",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
						{
							Location: "openapi1.yaml",
						},
					},
				},
			},
			wantOutputLocation: ".speakeasy/temp/output_testrandomstring.yaml",
		},
		{
			name: "source with overlays returns specified output location",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
					Overlays: []workflow.Document{
						{
							Location: "overlay.yaml",
						},
					},
					Output: pointer.ToString("processed.yaml"),
				},
			},
			wantOutputLocation: "processed.yaml",
		},
		{
			name: "source with overlays returns auto-generated output location",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
					Overlays: []workflow.Document{
						{
							Location: "overlay.yaml",
						},
					},
				},
			},
			wantOutputLocation: ".speakeasy/temp/output_testrandomstring.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir, err := createLocalFiles(tt.args.source)
			require.NoError(t, err)
			defer os.RemoveAll(workDir)

			err = os.Chdir(workDir)
			require.NoError(t, err)

			err = tt.args.source.Validate()
			require.NoError(t, err)

			outputLocation, err := tt.args.source.GetOutputLocation()
			require.NoError(t, err)

			assert.Contains(t, outputLocation, tt.wantOutputLocation)
		})
	}
}

func createLocalFiles(s workflow.Source) (string, error) {
	tmpDir, err := os.MkdirTemp("", "workflow*")
	if err != nil {
		return "", err
	}

	for _, input := range s.Inputs {
		var filePath string
		if strings.HasPrefix(input.Location, "~/") {
			filePath = workflow.SanitizeFilePath(input.Location)
		} else {
			filePath = filepath.Join(tmpDir, input.Location)
		}
		_, err := url.ParseRequestURI(input.Location)
		if err != nil {
			if err := createEmptyFile(filePath); err != nil {
				return "", err
			}
		}
	}

	for _, overlay := range s.Overlays {
		_, err := url.ParseRequestURI(overlay.Location)
		if err != nil {
			if err := createEmptyFile(filepath.Join(tmpDir, overlay.Location)); err != nil {
				return "", err
			}
		}
	}

	return tmpDir, nil
}

func createEmptyFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	return f.Close()
}
