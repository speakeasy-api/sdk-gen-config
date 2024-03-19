package lint_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/speakeasy-api/sdk-gen-config/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLint_Load_Success(t *testing.T) {
	type args struct {
		lintLocation string
		lintContents string
		workingDir   string
	}
	tests := []struct {
		name string
		args args
		want *lint.Lint
	}{
		{
			name: "loads simple lint file",
			args: args{
				lintLocation: "test/.speakeasy",
				lintContents: `lintVersion: 1.0.0
defaultRuleset: default
rulesets:
  default:
    rulesets:
      - test
    rules:
      test: {}`,
				workingDir: "test",
			},
			want: &lint.Lint{
				Version:        "1.0.0",
				DefaultRuleset: "default",
				Rulesets: map[string]lint.Ruleset{
					"default": {
						Rulesets: []string{"test"},
						Rules: map[string]*model.Rule{
							"test": {},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			basePath, err := os.MkdirTemp("", "lint*")
			require.NoError(t, err)
			defer os.RemoveAll(basePath)

			err = createTempFile(filepath.Join(basePath, tt.args.lintLocation), "lint.yaml", tt.args.lintContents)
			require.NoError(t, err)

			workflowFile, workflowPath, err := lint.Load(filepath.Join(basePath, tt.args.workingDir))
			require.NoError(t, err)

			assert.Equal(t, tt.want, workflowFile)
			assert.Contains(t, workflowPath, filepath.Join(tt.args.lintLocation, "lint.yaml"))
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
