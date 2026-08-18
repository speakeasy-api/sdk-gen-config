package workflow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/speakeasy-api/sdk-gen-config/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestFormatOptions_YAMLRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		expected     workflow.FormatStyle
		expectedYAML string
	}{
		{
			name:         "legacy true",
			input:        "format: true\n",
			expected:     workflow.FormatStyleReadable,
			expectedYAML: "format: true\n",
		},
		{
			name:         "legacy false",
			input:        "format: false\n",
			expected:     workflow.FormatStyleReadable,
			expectedYAML: "format: false\n",
		},
		{
			name:         "legacy YAML 1.1 true",
			input:        "format: yes\n",
			expected:     workflow.FormatStyleReadable,
			expectedYAML: "format: true\n",
		},
		{
			name:         "legacy YAML 1.1 false",
			input:        "format: off\n",
			expected:     workflow.FormatStyleReadable,
			expectedYAML: "format: false\n",
		},
		{
			name:         "readable object",
			input:        "format:\n  style: readable\n",
			expected:     workflow.FormatStyleReadable,
			expectedYAML: "format:\n    style: readable\n",
		},
		{
			name:         "sorted object",
			input:        "format:\n  style: sorted\n",
			expected:     workflow.FormatStyleSorted,
			expectedYAML: "format:\n    style: sorted\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var transformation workflow.Transformation
			err := yaml.Unmarshal([]byte(tt.input), &transformation)
			require.NoError(t, err, "format options should unmarshal")
			require.NotNil(t, transformation.Format, "format options should be present")
			assert.Equal(t, tt.expected, transformation.Format.GetStyle(), "style should match")
			require.NoError(t, transformation.Validate(), "transformation should validate")

			output, err := yaml.Marshal(transformation)
			require.NoError(t, err, "format options should marshal")
			assert.Equal(t, tt.expectedYAML, string(output), "YAML representation should be preserved")
		})
	}
}

func TestFormatOptions_ValidateError(t *testing.T) {
	t.Parallel()

	transformation := workflow.Transformation{
		Format: &workflow.FormatOptions{Style: "unsupported"},
	}

	err := transformation.Validate()
	require.EqualError(t, err, "format.style must be one of readable, sorted", "unsupported style should fail")
}

func TestFormatOptions_InvalidYAMLReturnsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "scalar style",
			input:    "format: sorted\n",
			expected: "format must be a boolean or an options object",
		},
		{
			name:     "quoted boolean",
			input:    "format: \"true\"\n",
			expected: "format must be a boolean or an options object",
		},
		{
			name:     "missing style",
			input:    "format: {}\n",
			expected: "format.style is required",
		},
		{
			name:     "unknown field",
			input:    "format:\n  style: sorted\n  extra: true\n",
			expected: `format contains unsupported field "extra"`,
		},
		{
			name:     "null style",
			input:    "format:\n  style: null\n",
			expected: "format.style is required",
		},
		{
			name:     "empty style",
			input:    "format:\n  style: \"\"\n",
			expected: "format.style is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var transformation workflow.Transformation
			err := yaml.Unmarshal([]byte(tt.input), &transformation)
			require.EqualError(t, err, tt.expected, "invalid format options should fail")
		})
	}
}

func TestFormatOptions_NullYAMLRemainsUnset(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"format: null\n", "format:\n"} {
		var transformation workflow.Transformation
		require.NoError(t, yaml.Unmarshal([]byte(input), &transformation))
		assert.Nil(t, transformation.Format)
		require.EqualError(t, transformation.Validate(), "transformation must have exactly one of removeUnused, filterOperations, cleanup, format, jqSymbolicExecution, normalize")
	}
}

func TestFormatOptions_ProgrammaticDefaultMarshalsReadable(t *testing.T) {
	t.Parallel()

	output, err := yaml.Marshal(workflow.Transformation{Format: &workflow.FormatOptions{}})
	require.NoError(t, err)
	assert.Equal(t, "format:\n    style: readable\n", string(output))
}

func TestFormatOptions_SourceValidation(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	jsonInput := filepath.Join(tempDir, "input.json")
	yamlInput := filepath.Join(tempDir, "input.yaml")
	require.NoError(t, os.WriteFile(jsonInput, []byte("{}\n"), 0o600))
	require.NoError(t, os.WriteFile(yamlInput, []byte("openapi: 3.0.0\n"), 0o600))

	jsonOutput := filepath.Join(tempDir, "output.json")
	yamlOutput := filepath.Join(tempDir, "output.yaml")
	uppercaseJSONOutput := filepath.Join(tempDir, "output.JSON")
	sorted := func() workflow.Transformation {
		return workflow.Transformation{
			Format: &workflow.FormatOptions{Style: workflow.FormatStyleSorted},
		}
	}

	tests := []struct {
		name            string
		input           string
		output          *string
		transformations []workflow.Transformation
		expectedError   string
	}{
		{
			name:            "JSON output",
			input:           yamlInput,
			output:          &jsonOutput,
			transformations: []workflow.Transformation{sorted()},
		},
		{
			name:            "implicit JSON output",
			input:           jsonInput,
			transformations: []workflow.Transformation{sorted()},
		},
		{
			name:            "uppercase JSON output",
			input:           yamlInput,
			output:          &uppercaseJSONOutput,
			transformations: []workflow.Transformation{sorted()},
		},
		{
			name:            "YAML output",
			input:           jsonInput,
			output:          &yamlOutput,
			transformations: []workflow.Transformation{sorted()},
			expectedError:   "failed to validate transformation 0: format.style sorted requires a JSON output (set output to a *.json path)",
		},
		{
			name:            "implicit YAML output",
			input:           yamlInput,
			transformations: []workflow.Transformation{sorted()},
			expectedError:   "failed to validate transformation 0: format.style sorted requires a JSON output (set output to a *.json path)",
		},
		{
			name:   "final transformation",
			input:  jsonInput,
			output: &jsonOutput,
			transformations: []workflow.Transformation{
				{Cleanup: boolPointer(true)},
				sorted(),
			},
		},
		{
			name:   "not final transformation",
			input:  jsonInput,
			output: &jsonOutput,
			transformations: []workflow.Transformation{
				sorted(),
				{Cleanup: boolPointer(true)},
			},
			expectedError: "failed to validate transformation 0: format.style sorted must be the final transformation",
		},
		{
			name:            "multiple sorted transformations",
			input:           jsonInput,
			output:          &jsonOutput,
			transformations: []workflow.Transformation{sorted(), sorted()},
			expectedError:   "failed to validate transformation 0: format.style sorted must be the final transformation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			source := workflow.Source{
				Inputs:          []workflow.Document{{Location: workflow.LocationString(tt.input)}},
				Transformations: tt.transformations,
				Output:          tt.output,
			}

			err := source.Validate()
			if tt.expectedError != "" {
				require.EqualError(t, err, tt.expectedError)
				return
			}
			require.NoError(t, err)
		})
	}
}

func boolPointer(value bool) *bool {
	return &value
}
