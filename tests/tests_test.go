package tests_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/speakeasy-api/sdk-gen-config/tests"
	"github.com/speakeasy-api/sdk-gen-config/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"gopkg.in/yaml.v3"
)

func TestLoad_Success(t *testing.T) {
	type args struct {
		contents string
	}
	testss := []struct {
		name string
		args args
		want *tests.Tests
	}{
		{
			name: "loads tests file",
			args: args{
				contents: `testsVersion: 0.0.1
tests:
  test:
    - name: test
      description: test
      targets:
        - typescript
      server: http://localhost:8080
      security:
        - api_key: []
      parameters:
        path:
          id: test
        query:
          limit: 100
      requestBody:
          application/json: {"id": "test", "name": "test"}
      responses:
        "200":
          application/json:
            id: test
            name: test
`,
			},
			want: &tests.Tests{
				Version: "0.0.1",
				Tests: orderedmap.New[string, []tests.Test](orderedmap.WithInitialData(
					orderedmap.Pair[string, []tests.Test]{
						Key: "test",
						Value: []tests.Test{
							{
								Name:        "test",
								Description: "test",
								Targets:     []string{"typescript"},
								Server:      "http://localhost:8080",
								Security: yaml.Node{
									Kind:   yaml.SequenceNode,
									Tag:    "!!seq",
									Line:   10,
									Column: 9,
									Content: []*yaml.Node{
										{
											Kind:   yaml.MappingNode,
											Tag:    "!!map",
											Line:   10,
											Column: 11,
											Content: []*yaml.Node{
												{
													Kind:   yaml.ScalarNode,
													Value:  "api_key",
													Tag:    "!!str",
													Line:   10,
													Column: 11,
												},
												{
													Kind:   yaml.SequenceNode,
													Tag:    "!!seq",
													Style:  yaml.FlowStyle,
													Line:   10,
													Column: 20,
												},
											},
										},
									},
								},
								Parameters: &tests.Parameters{
									Path: orderedmap.New[string, yaml.Node](orderedmap.WithInitialData(
										orderedmap.Pair[string, yaml.Node]{
											Key: "id",
											Value: yaml.Node{
												Kind:   yaml.ScalarNode,
												Value:  "test",
												Tag:    "!!str",
												Line:   13,
												Column: 15,
											},
										},
									)),
									Query: orderedmap.New[string, yaml.Node](orderedmap.WithInitialData(
										orderedmap.Pair[string, yaml.Node]{
											Key: "limit",
											Value: yaml.Node{
												Kind:   yaml.ScalarNode,
												Value:  "100",
												Tag:    "!!int",
												Line:   15,
												Column: 18,
											},
										},
									)),
								},
								RequestBody: orderedmap.New[string, yaml.Node](orderedmap.WithInitialData(
									orderedmap.Pair[string, yaml.Node]{
										Key: "application/json",
										Value: yaml.Node{
											Kind:   yaml.MappingNode,
											Tag:    "!!map",
											Line:   17,
											Column: 29,
											Style:  yaml.FlowStyle,
											Content: []*yaml.Node{
												{
													Kind:   yaml.ScalarNode,
													Value:  "id",
													Tag:    "!!str",
													Style:  yaml.DoubleQuotedStyle,
													Line:   17,
													Column: 30,
												},
												{
													Kind:   yaml.ScalarNode,
													Value:  "test",
													Tag:    "!!str",
													Style:  yaml.DoubleQuotedStyle,
													Line:   17,
													Column: 36,
												},
												{
													Kind:   yaml.ScalarNode,
													Value:  "name",
													Tag:    "!!str",
													Style:  yaml.DoubleQuotedStyle,
													Line:   17,
													Column: 44,
												},
												{
													Kind:   yaml.ScalarNode,
													Value:  "test",
													Tag:    "!!str",
													Style:  yaml.DoubleQuotedStyle,
													Line:   17,
													Column: 52,
												},
											},
										},
									},
								)),
								Responses: orderedmap.New[string, yaml.Node](orderedmap.WithInitialData(
									orderedmap.Pair[string, yaml.Node]{
										Key: "200",
										Value: yaml.Node{
											Kind:   yaml.MappingNode,
											Tag:    "!!map",
											Line:   20,
											Column: 11,
											Content: []*yaml.Node{
												{
													Kind:   yaml.ScalarNode,
													Value:  "application/json",
													Tag:    "!!str",
													Line:   20,
													Column: 11,
												},
												{
													Kind:   yaml.MappingNode,
													Tag:    "!!map",
													Line:   21,
													Column: 13,
													Content: []*yaml.Node{
														{
															Kind:   yaml.ScalarNode,
															Value:  "id",
															Tag:    "!!str",
															Line:   21,
															Column: 13,
														},
														{
															Kind:   yaml.ScalarNode,
															Value:  "test",
															Tag:    "!!str",
															Line:   21,
															Column: 17,
														},
														{
															Kind:   yaml.ScalarNode,
															Value:  "name",
															Tag:    "!!str",
															Line:   22,
															Column: 13,
														},
														{
															Kind:   yaml.ScalarNode,
															Value:  "test",
															Tag:    "!!str",
															Line:   22,
															Column: 19,
														},
													},
												},
											},
										},
									},
								)),
							},
						},
					},
				)),
			},
		},
		{
			name: "loads test file with simple test",
			args: args{
				contents: `testsVersion: 0.0.1
tests:
  test:
    - name: test
      responses:
        "200": true
`,
			},
			want: &tests.Tests{
				Version: "0.0.1",
				Tests: orderedmap.New[string, []tests.Test](orderedmap.WithInitialData(
					orderedmap.Pair[string, []tests.Test]{
						Key: "test",
						Value: []tests.Test{
							{
								Name: "test",
								Responses: orderedmap.New[string, yaml.Node](orderedmap.WithInitialData(
									orderedmap.Pair[string, yaml.Node]{
										Key: "200",
										Value: yaml.Node{
											Kind:   yaml.ScalarNode,
											Value:  "true",
											Tag:    "!!bool",
											Line:   6,
											Column: 16,
										},
									},
								)),
							},
						},
					},
				)),
			},
		},
	}
	for _, tt := range testss {
		t.Run(tt.name, func(t *testing.T) {
			testsDir := filepath.Join(os.TempDir(), "tests/.speakeasy")
			testutils.CreateTempFile(t, testsDir, "tests.yaml", tt.args.contents)
			defer os.RemoveAll(testsDir)

			loadedTests, _, err := tests.Load(testsDir)
			require.NoError(t, err)

			assert.Equal(t, tt.want, loadedTests)
		})
	}
}

func TestTest_GetResponse_Success(t *testing.T) {
	type args struct {
		contents string
	}
	testss := []struct {
		name                 string
		args                 args
		wantResBody          *orderedmap.OrderedMap[string, yaml.Node]
		wantAssertStatusCode bool
	}{
		{
			name: "get response body",
			args: args{
				contents: `testsVersion: 0.0.1
tests:
  test:
    - name: test
      responses:
        "200":
          application/json:
            id: test
            name: test
`,
			},
			wantResBody: orderedmap.New[string, yaml.Node](orderedmap.WithInitialData(
				orderedmap.Pair[string, yaml.Node]{
					Key: "application/json",
					Value: yaml.Node{
						Kind:   yaml.MappingNode,
						Tag:    "!!map",
						Line:   8,
						Column: 13,
						Content: []*yaml.Node{
							{
								Kind:   yaml.ScalarNode,
								Value:  "id",
								Tag:    "!!str",
								Line:   8,
								Column: 13,
							},
							{
								Kind:   yaml.ScalarNode,
								Value:  "test",
								Tag:    "!!str",
								Line:   8,
								Column: 17,
							},
							{
								Kind:   yaml.ScalarNode,
								Value:  "name",
								Tag:    "!!str",
								Line:   9,
								Column: 13,
							},
							{
								Kind:   yaml.ScalarNode,
								Value:  "test",
								Tag:    "!!str",
								Line:   9,
								Column: 19,
							},
						},
					},
				},
			)),
		},
		{
			name: "get assert status code",
			args: args{
				contents: `testsVersion: 0.0.1
tests:
  test:
    - name: test
      responses:
        "200": true
`,
			},
			wantAssertStatusCode: true,
		},
	}
	for _, tt := range testss {
		t.Run(tt.name, func(t *testing.T) {
			testsDir := filepath.Join(os.TempDir(), "tests/.speakeasy")
			testutils.CreateTempFile(t, testsDir, "tests.yaml", tt.args.contents)
			defer os.RemoveAll(testsDir)

			loadedTests, _, err := tests.Load(testsDir)
			require.NoError(t, err)

			test, ok := loadedTests.Tests.Get("test")
			require.True(t, ok)

			resBody, assertStatusCode, err := test[0].GetResponse("200")
			require.NoError(t, err)

			assert.Equal(t, tt.wantResBody, resBody)
			assert.Equal(t, tt.wantAssertStatusCode, assertStatusCode)
		})
	}
}
