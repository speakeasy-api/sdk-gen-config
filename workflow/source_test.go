package workflow_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/speakeasy-api/sdk-gen-config/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
					Overlays: []workflow.Overlay{
						{Document: &workflow.Document{Location: "overlay.yaml"}},
						{Document: &workflow.Document{Location: "http://example.com/overlay.yaml"}},
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
					Overlays: []workflow.Overlay{
						{Document: &workflow.Document{Location: "overlay.yaml"}},
						{Document: &workflow.Document{Location: "overlay.yaml"}},
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
					Overlays: []workflow.Overlay{
						{Document: &workflow.Document{Location: "overlay.yaml"}},
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
					Overlays: []workflow.Overlay{
						{Document: &workflow.Document{Location: "overlay.yaml"}},
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
					Overlays: []workflow.Overlay{{
						Document: &workflow.Document{},
					}},
				},
			},
			wantErr: fmt.Errorf("failed to validate overlay 0: failed to validate overlay document: location is required"),
		},
		{
			name: "overlay fails with no fallbackCodeSamplesLanguage",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
					Overlays: []workflow.Overlay{{
						FallbackCodeSamples: &workflow.FallbackCodeSamples{},
					}},
				},
			},
			wantErr: fmt.Errorf("failed to validate overlay 0: failed to validate overlay fallbackCodeSamples: fallbackCodeSamplesLanguage is required"),
		},
		{
			name: "overlay with fallbackCodeSamplesLanguage",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
					Overlays: []workflow.Overlay{{
						FallbackCodeSamples: &workflow.FallbackCodeSamples{
							FallbackCodeSamplesLanguage: "python",
						},
					}},
				},
			},
		},
		{
			name: "registry success",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
					Registry: &workflow.SourceRegistry{
						Location: "registry.speakeasyapi.dev/org/workspace/image",
					},
				},
			},
		},
		{
			name: "registry fails with invalid location",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
					Registry: &workflow.SourceRegistry{
						Location: "registry.speakeasyapi.dev/not-enough-parts",
					},
				},
			},
			wantErr: fmt.Errorf("failed to validate registry: registry location should look like registry.speakeasyapi.dev/<org>/<workspace>/<image>"),
		},
		{
			name: "registry fails with no location",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.yaml",
						},
					},
					Registry: &workflow.SourceRegistry{},
				},
			},
			wantErr: fmt.Errorf("failed to validate registry: location is required"),
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

			if tt.wantErr == nil {
				// Marshal to yaml
				w := workflow.Workflow{
					Version:          workflow.WorkflowVersion,
					SpeakeasyVersion: "latest",
					Sources: map[string]workflow.Source{
						"source": tt.args.source,
					},
				}
				data, err := yaml.Marshal(w)
				require.NoError(t, err)

				// Unmarshal yaml
				var w2 workflow.Workflow
				err = yaml.Unmarshal(data, &w2)
				require.NoError(t, err)

				// Validate
				err = w2.Validate([]string{})

				assert.NoError(t, err)
			}
		})
	}
}

func TestSource_GetOutputLocation(t *testing.T) {
	type args struct {
		source workflow.Source
	}

	// The URL needs to be deterministic because the hash is based on the URL + path
	testServer, err := newTestServerWithURL("127.0.0.1:1234", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		// Determine the file extension from the URL path
		fileExt := filepath.Ext(req.URL.Path)

		// Default to JSON or YAML if no file extension is present
		if fileExt == "" {
			switch {
			case strings.Contains(req.URL.Path, "json"):
				fileExt = ".json"
			case strings.Contains(req.URL.Path, "yaml"):
				fileExt = ".yaml"
			}
		}

		// Determine the content type and response body based on the file extension
		var (
			contentType   string
			response      interface{}
			err           error
			responseBytes []byte
		)
		response = map[string]interface{}{"openapi": "3.0.0"}

		switch fileExt {
		case ".json":
			contentType = "application/json"
		case ".yaml":
			contentType = "application/yaml"
		default:
			http.Error(res, "Unsupported file format", http.StatusBadRequest)
			return
		}

		// Set the content type header
		res.Header().Set("Content-Type", contentType)

		// Marshal and write the response based on content type
		if contentType == "application/json" {
			responseBytes, err = json.Marshal(response)
		} else {
			responseBytes, err = yaml.Marshal(response)
		}
		assert.NoError(t, err)
		res.Write(responseBytes)
	}))

	require.NoError(t, err)
	defer func() { testServer.Close() }()

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
							Location: fmt.Sprintf("%s/openapi.json", testServer.URL),
						},
					},
				},
			},
			wantOutputLocation: ".speakeasy/temp/registry_4b5145.json",
		},
		{
			name: "simple remote source without extension returns auto-generated output location assumed to be yaml",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: fmt.Sprintf("%s/openapi", testServer.URL),
						},
					},
				},
			},
			wantOutputLocation: ".speakeasy/temp/registry_61ea27.yaml",
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
			wantOutputLocation: ".speakeasy/temp/output_6a0196.yaml",
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
					Overlays: []workflow.Overlay{
						{Document: &workflow.Document{Location: "overlay.yaml"}},
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
					Overlays: []workflow.Overlay{
						{Document: &workflow.Document{Location: "overlay.yaml"}},
					},
				},
			},
			wantOutputLocation: ".speakeasy/temp/output_d910ba.yaml",
		},
		{
			name: "single local source uses same extension as source",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.json",
						},
					},
				},
			},
			wantOutputLocation: "openapi.json",
		},
		{
			name: "single local source with overlays uses same extension as source",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: "openapi.json",
						},
					},
					Overlays: []workflow.Overlay{
						{Document: &workflow.Document{Location: "overlay.yaml"}},
					},
				},
			},
			wantOutputLocation: ".speakeasy/temp/output_a98653.json",
		},
		{
			name: "single remote source with unknown format uses resolved extension",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: fmt.Sprintf("%s/thepathincludesjson", testServer.URL),
						},
					},
				},
			},
			wantOutputLocation: ".speakeasy/temp/registry_411616.json",
		},
		{
			name: "single remote source with unsupported file extension returns auto-generated output location",
			args: args{
				source: workflow.Source{
					Inputs: []workflow.Document{
						{
							Location: fmt.Sprintf("%s/foo.txt", testServer.URL),
						},
					},
				},
			},
			wantOutputLocation: ".speakeasy/temp/registry_69a6f2.yaml",
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

func TestSource_ParseSpeakeasyRegistryReference(t *testing.T) {
	// Examples:
	// registry.speakeasyapi.dev/org/workspace/name
	// registry.speakeasyapi.dev/org/workspace/name@sha256:1234567890abcdef
	// registry.speakeasyapi.dev/org/workspace/name:tag
	// Expected output:
	// NamespaceID: org/workspace/name, Reference: latest, NamespaceName: name

	type args struct {
		location string
	}
	tests := []struct {
		name string
		args args
		want *workflow.SpeakeasyRegistryDocument
	}{
		{
			name: "simple reference",
			args: args{
				location: "registry.speakeasyapi.dev/org/workspace/name",
			},
			want: &workflow.SpeakeasyRegistryDocument{
				NamespaceID:      "org/workspace/name",
				OrganizationSlug: "org",
				WorkspaceSlug:    "workspace",
				NamespaceName:    "name",
				Reference:        "latest",
			},
		},
		{
			name: "reference with sha256",
			args: args{
				location: "registry.speakeasyapi.dev/org/workspace/name@sha256:1234567890abcdef",
			},
			want: &workflow.SpeakeasyRegistryDocument{
				OrganizationSlug: "org",
				WorkspaceSlug:    "workspace",
				NamespaceID:      "org/workspace/name",
				NamespaceName:    "name",
				Reference:        "sha256:1234567890abcdef",
			},
		},
		{
			name: "reference with tag",
			args: args{
				location: "registry.speakeasyapi.dev/org/workspace/name:tag",
			},
			want: &workflow.SpeakeasyRegistryDocument{
				OrganizationSlug: "org",
				WorkspaceSlug:    "workspace",
				NamespaceID:      "org/workspace/name",
				NamespaceName:    "name",
				Reference:        "tag",
			},
		},
		{
			name: "reference with invalid format",
			args: args{
				location: "registry.speakeasyapi.dev/org/workspace",
			},
			want: nil,
		},
		{
			name: "reference with invalid format",
			args: args{
				location: "reg.speakeasyapi.dev/org/workspace/name",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registryDocument := workflow.ParseSpeakeasyRegistryReference(tt.args.location)
			assert.Equal(t, tt.want, registryDocument)
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
		if overlay.Document != nil {
			_, err := url.ParseRequestURI(overlay.Document.Location)
			if err != nil {
				if err := createEmptyFile(filepath.Join(tmpDir, overlay.Document.Location)); err != nil {
					return "", err
				}
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

func newTestServerWithURL(URL string, handler http.Handler) (*httptest.Server, error) {
	ts := httptest.NewUnstartedServer(handler)
	if URL != "" {
		l, err := net.Listen("tcp", URL)
		if err != nil {
			return nil, err
		}
		ts.Listener.Close()
		ts.Listener = l
	}
	ts.Start()
	return ts, nil
}
