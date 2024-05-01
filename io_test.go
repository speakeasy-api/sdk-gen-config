package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/speakeasy-api/sdk-gen-config/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDir = "gen/test"

func TestLoad_Success(t *testing.T) {
	getUUID = func() string {
		return "123"
	}

	type args struct {
		langs        []string
		configDir    string
		targetDir    string
		genYaml      string
		lockFile     string
		configSubDir string
	}
	tests := []struct {
		name string
		args args
		want *Config
	}{
		{
			name: "creates config file and lock file if it doesn't exist in the .speakeasy dir",
			args: args{
				langs:        []string{"go"},
				configDir:    testDir,
				targetDir:    testDir,
				configSubDir: ".speakeasy",
			},
			want: &Config{
				Config: &Configuration{
					ConfigVersion: Version,
					Languages: map[string]LanguageConfig{
						"go": {
							Version: "0.0.1",
						},
					},
					Generation: Generation{
						SDKClassName:         "SDK",
						MaintainOpenAPIOrder: true,
						UsageSnippets: &UsageSnippets{
							OptionalPropertyRendering: "withExample",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                true,
							ParameterOrderingFeb2024:             true,
							RequestResponseComponentNamesFeb2024: true,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: true,
						},
						UseClassNamesForArrayFields: true,
					},
					New: map[string]bool{
						"go": true,
					},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, ".speakeasy/gen.yaml"),
				LockFile: &LockFile{
					LockVersion: Version,
					ID:          "123",
					Management:  Management{},
					Features:    make(map[string]map[string]string),
				},
			},
		},
		{
			name: "loads and upgrades pre v1.0.0 config file",
			args: args{
				langs:     []string{"go"},
				configDir: testDir,
				targetDir: testDir,
				genYaml:   readTestFile(t, "pre-v100-gen.yaml"),
			},
			want: &Config{
				Config: &Configuration{
					ConfigVersion: Version,
					Languages: map[string]LanguageConfig{
						"go": {
							Version: "1.3.0",
							Cfg: map[string]any{
								"packageName": "github.com/speakeasy-api/speakeasy-client-sdk-go",
							},
						},
					},
					Generation: Generation{
						BaseServerURL: "https://api.prod.speakeasyapi.dev",
						SDKClassName:  "speakeasy",
						UsageSnippets: &UsageSnippets{
							OptionalPropertyRendering: "withExample",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                false,
							ParameterOrderingFeb2024:             false,
							RequestResponseComponentNamesFeb2024: false,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: false,
						},
					},
					New: map[string]bool{},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, "gen.yaml"),
				LockFile: &LockFile{
					LockVersion: Version,
					ID:          "123",
					Management: Management{
						DocChecksum:      "2bba3b8f9d211b02569b3f9aff0d34b4",
						DocVersion:       "0.3.0",
						SpeakeasyVersion: "1.3.1",
						ReleaseVersion:   "1.3.0",
					},
					Features: make(map[string]map[string]string),
				},
			},
		},
		{
			name: "loads v1.0.0 config file",
			args: args{
				langs:        []string{"go"},
				configDir:    testDir,
				targetDir:    testDir,
				genYaml:      readTestFile(t, "v100-gen.yaml"),
				configSubDir: ".speakeasy",
			},
			want: &Config{
				Config: &Configuration{
					ConfigVersion: Version,
					Languages: map[string]LanguageConfig{
						"go": {
							Version: "1.3.0",
							Cfg: map[string]any{
								"packageName": "github.com/speakeasy-api/speakeasy-client-sdk-go",
							},
						},
					},
					Generation: Generation{
						BaseServerURL: "https://api.prod.speakeasyapi.dev",
						SDKClassName:  "speakeasy",
						UsageSnippets: &UsageSnippets{
							OptionalPropertyRendering: "withExample",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                false,
							ParameterOrderingFeb2024:             false,
							RequestResponseComponentNamesFeb2024: false,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: false,
						},
					},
					New: map[string]bool{},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, ".speakeasy/gen.yaml"),
				LockFile: &LockFile{
					LockVersion: Version,
					ID:          "123",
					Management: Management{
						DocChecksum:      "2bba3b8f9d211b02569b3f9aff0d34b4",
						DocVersion:       "0.3.0",
						SpeakeasyVersion: "1.3.1",
						ReleaseVersion:   "1.3.0",
					},
					Features: map[string]map[string]string{
						"go": {
							"core": "2.90.0",
						},
					},
				},
			},
		},
		{
			name: "loads v2.0.0 config file",
			args: args{
				langs:        []string{"go"},
				configDir:    testDir,
				targetDir:    testDir,
				genYaml:      readTestFile(t, "v200-gen.yaml"),
				lockFile:     readTestFile(t, "v200-gen.lock"),
				configSubDir: ".speakeasy",
			},
			want: &Config{
				Config: &Configuration{
					ConfigVersion: Version,
					Languages: map[string]LanguageConfig{
						"go": {
							Version: "1.3.0",
							Cfg: map[string]any{
								"packageName": "github.com/speakeasy-api/speakeasy-client-sdk-go",
							},
						},
					},
					Generation: Generation{
						BaseServerURL: "https://api.prod.speakeasyapi.dev",
						SDKClassName:  "speakeasy",
						UsageSnippets: &UsageSnippets{
							OptionalPropertyRendering: "withExample",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                false,
							ParameterOrderingFeb2024:             false,
							RequestResponseComponentNamesFeb2024: false,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: false,
						},
					},
					New: map[string]bool{},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, ".speakeasy/gen.yaml"),
				LockFile: &LockFile{
					LockVersion: Version,
					ID:          "0f8fad5b-d9cb-469f-a165-70867728950e",
					Management: Management{
						DocChecksum:      "2bba3b8f9d211b02569b3f9aff0d34b4",
						DocVersion:       "0.3.0",
						SpeakeasyVersion: "1.3.1",
						ReleaseVersion:   "1.3.0",
					},
					Features: map[string]map[string]string{
						"go": {
							"core": "2.90.0",
						},
					},
				},
			},
		},
		{
			name: "loads v2.0.0 config file without existing lock file as a new sdk",
			args: args{
				langs:        []string{"go"},
				configDir:    testDir,
				targetDir:    testDir,
				genYaml:      readTestFile(t, "v200-gen.yaml"),
				configSubDir: ".speakeasy",
			},
			want: &Config{
				Config: &Configuration{
					ConfigVersion: Version,
					Languages: map[string]LanguageConfig{
						"go": {
							Version: "1.3.0",
							Cfg: map[string]any{
								"packageName": "github.com/speakeasy-api/speakeasy-client-sdk-go",
							},
						},
					},
					Generation: Generation{
						BaseServerURL: "https://api.prod.speakeasyapi.dev",
						SDKClassName:  "speakeasy",
						UsageSnippets: &UsageSnippets{
							OptionalPropertyRendering: "withExample",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                true,
							ParameterOrderingFeb2024:             true,
							RequestResponseComponentNamesFeb2024: true,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: true,
						},
						MaintainOpenAPIOrder:        true,
						UseClassNamesForArrayFields: true,
					},
					New: map[string]bool{
						"go": true,
					},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, ".speakeasy/gen.yaml"),
				LockFile: &LockFile{
					LockVersion: Version,
					ID:          "123",
					Management:  Management{},
					Features:    make(map[string]map[string]string),
				},
			},
		},
		{
			name: "loads v2.0.0 config file without existing lock file as a new sdk from .gen folder",
			args: args{
				langs:        []string{"go"},
				configDir:    testDir,
				targetDir:    testDir,
				genYaml:      readTestFile(t, "v200-gen.yaml"),
				configSubDir: ".gen",
			},
			want: &Config{
				Config: &Configuration{
					ConfigVersion: Version,
					Languages: map[string]LanguageConfig{
						"go": {
							Version: "1.3.0",
							Cfg: map[string]any{
								"packageName": "github.com/speakeasy-api/speakeasy-client-sdk-go",
							},
						},
					},
					Generation: Generation{
						BaseServerURL: "https://api.prod.speakeasyapi.dev",
						SDKClassName:  "speakeasy",
						UsageSnippets: &UsageSnippets{
							OptionalPropertyRendering: "withExample",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                true,
							ParameterOrderingFeb2024:             true,
							RequestResponseComponentNamesFeb2024: true,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: true,
						},
						MaintainOpenAPIOrder:        true,
						UseClassNamesForArrayFields: true,
					},
					New: map[string]bool{
						"go": true,
					},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, ".gen/gen.yaml"),
				LockFile: &LockFile{
					LockVersion: Version,
					ID:          "123",
					Management:  Management{},
					Features:    make(map[string]map[string]string),
				},
			},
		},
		{
			name: "loads v2.0.0 config file from higher level directory",
			args: args{
				langs:     []string{"go"},
				configDir: filepath.Dir(testDir),
				targetDir: testDir,
				genYaml:   readTestFile(t, "v200-gen.yaml"),
				lockFile:  readTestFile(t, "v200-gen.lock"),
			},
			want: &Config{
				Config: &Configuration{
					ConfigVersion: Version,
					Languages: map[string]LanguageConfig{
						"go": {
							Version: "1.3.0",
							Cfg: map[string]any{
								"packageName": "github.com/speakeasy-api/speakeasy-client-sdk-go",
							},
						},
					},
					Generation: Generation{
						BaseServerURL: "https://api.prod.speakeasyapi.dev",
						SDKClassName:  "speakeasy",
						UsageSnippets: &UsageSnippets{
							OptionalPropertyRendering: "withExample",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                false,
							ParameterOrderingFeb2024:             false,
							RequestResponseComponentNamesFeb2024: false,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: false,
						},
					},
					New: map[string]bool{},
				},
				ConfigPath: filepath.Join(os.TempDir(), filepath.Dir(testDir), "gen.yaml"),
				LockFile: &LockFile{
					LockVersion: Version,
					ID:          "0f8fad5b-d9cb-469f-a165-70867728950e",
					Management: Management{
						DocChecksum:      "2bba3b8f9d211b02569b3f9aff0d34b4",
						DocVersion:       "0.3.0",
						SpeakeasyVersion: "1.3.1",
						ReleaseVersion:   "1.3.0",
					},
					Features: map[string]map[string]string{
						"go": {
							"core": "2.90.0",
						},
					},
				},
			},
		},
		{
			name: "loads v100 config file and detects new config for language",
			args: args{
				langs:     []string{"go", "typescript"},
				configDir: testDir,
				targetDir: testDir,
				genYaml:   readTestFile(t, "v100-gen.yaml"),
			},
			want: &Config{
				Config: &Configuration{
					ConfigVersion: Version,
					Languages: map[string]LanguageConfig{
						"go": {
							Version: "1.3.0",
							Cfg: map[string]any{
								"packageName": "github.com/speakeasy-api/speakeasy-client-sdk-go",
							},
						},
						"typescript": {
							Version: "0.0.1",
						},
					},
					Generation: Generation{
						BaseServerURL: "https://api.prod.speakeasyapi.dev",
						SDKClassName:  "speakeasy",
						UsageSnippets: &UsageSnippets{
							OptionalPropertyRendering: "withExample",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                false,
							ParameterOrderingFeb2024:             false,
							RequestResponseComponentNamesFeb2024: false,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: false,
						},
					},
					New: map[string]bool{
						"typescript": true,
					},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, "gen.yaml"),
				LockFile: &LockFile{
					LockVersion: Version,
					ID:          "123",
					Management: Management{
						DocChecksum:      "2bba3b8f9d211b02569b3f9aff0d34b4",
						DocVersion:       "0.3.0",
						SpeakeasyVersion: "1.3.1",
						ReleaseVersion:   "1.3.0",
					},
					Features: map[string]map[string]string{
						"go": {
							"core": "2.90.0",
						},
					},
				},
			},
		},
		{
			name: "loads v2.0.0 config file and detects new config for language",
			args: args{
				langs:        []string{"go", "typescript"},
				configDir:    testDir,
				targetDir:    testDir,
				genYaml:      readTestFile(t, "v200-gen.yaml"),
				lockFile:     readTestFile(t, "v200-gen.lock"),
				configSubDir: ".speakeasy",
			},
			want: &Config{
				Config: &Configuration{
					ConfigVersion: Version,
					Languages: map[string]LanguageConfig{
						"go": {
							Version: "1.3.0",
							Cfg: map[string]any{
								"packageName": "github.com/speakeasy-api/speakeasy-client-sdk-go",
							},
						},
						"typescript": {
							Version: "0.0.1",
						},
					},
					Generation: Generation{
						BaseServerURL: "https://api.prod.speakeasyapi.dev",
						SDKClassName:  "speakeasy",
						UsageSnippets: &UsageSnippets{
							OptionalPropertyRendering: "withExample",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                false,
							ParameterOrderingFeb2024:             false,
							RequestResponseComponentNamesFeb2024: false,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: false,
						},
					},
					New: map[string]bool{
						"typescript": true,
					},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, ".speakeasy/gen.yaml"),
				LockFile: &LockFile{
					LockVersion: Version,
					ID:          "0f8fad5b-d9cb-469f-a165-70867728950e",
					Management: Management{
						DocChecksum:      "2bba3b8f9d211b02569b3f9aff0d34b4",
						DocVersion:       "0.3.0",
						SpeakeasyVersion: "1.3.1",
						ReleaseVersion:   "1.3.0",
					},
					Features: map[string]map[string]string{
						"go": {
							"core": "2.90.0",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configDir := filepath.Join(os.TempDir(), tt.args.configDir)
			if tt.args.configSubDir != "" {
				configDir = filepath.Join(configDir, tt.args.configSubDir)
			}
			targetDir := filepath.Join(os.TempDir(), tt.args.targetDir)

			lockFileDir := filepath.Join(targetDir, tt.args.configSubDir)
			if tt.args.configSubDir == "" {
				lockFileDir = filepath.Join(targetDir, ".speakeasy")
			}

			err := createTempFile(configDir, "gen.yaml", tt.args.genYaml)
			require.NoError(t, err)

			err = createTempFile(lockFileDir, "gen.lock", tt.args.lockFile)
			require.NoError(t, err)

			defer os.RemoveAll(configDir)
			defer os.RemoveAll(targetDir)

			opts := []Option{
				WithUpgradeFunc(testUpdateLang),
			}

			for _, lang := range tt.args.langs {
				opts = append(opts, WithLanguages(lang))
			}

			cfg, err := Load(targetDir, opts...)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, cfg)
			_, err = os.Stat(filepath.Join(configDir, "gen.yaml"))
			assert.NoError(t, err)
			_, err = os.Stat(filepath.Join(lockFileDir, "gen.lock"))
			assert.NoError(t, err)
		})
	}
}

func TestLoad_BackwardsCompatibility_Success(t *testing.T) {
	getUUID = func() string {
		return "123"
	}

	// Create new config file in .speakeasy dir
	speakeasyDir := filepath.Join(os.TempDir(), testDir, workspace.SpeakeasyFolder)
	err := createTempFile(speakeasyDir, "gen.yaml", readTestFile(t, "v200-gen.yaml"))
	require.NoError(t, err)

	// Create old config file in root dir
	rootDir := filepath.Join(os.TempDir(), testDir)
	err = createTempFile(rootDir, "gen.yaml", readTestFile(t, "v100-gen.yaml"))
	require.NoError(t, err)

	defer os.RemoveAll(speakeasyDir)
	defer os.RemoveAll(rootDir)

	opts := []Option{
		WithUpgradeFunc(testUpdateLang),
	}

	opts = append(opts, WithLanguages("go"))

	cfg, err := Load(rootDir, opts...)
	assert.NoError(t, err)
	assert.Equal(t, &Config{
		Config: &Configuration{
			ConfigVersion: Version,
			Languages: map[string]LanguageConfig{
				"go": {
					Version: "1.3.0",
					Cfg: map[string]any{
						"packageName": "github.com/speakeasy-api/speakeasy-client-sdk-go",
					},
				},
			},
			Generation: Generation{
				BaseServerURL: "https://api.prod.speakeasyapi.dev",
				SDKClassName:  "speakeasy",
				UsageSnippets: &UsageSnippets{
					OptionalPropertyRendering: "withExample",
				},
				Fixes: &Fixes{
					NameResolutionDec2023:                false,
					ParameterOrderingFeb2024:             false,
					RequestResponseComponentNamesFeb2024: false,
				},
				Auth: &Auth{
					OAuth2ClientCredentialsEnabled: false,
				},
			},
			New: map[string]bool{},
		},
		ConfigPath: filepath.Join(os.TempDir(), testDir, "gen.yaml"),
		LockFile: &LockFile{
			LockVersion: Version,
			ID:          "123",
			Management: Management{
				DocChecksum:      "2bba3b8f9d211b02569b3f9aff0d34b4",
				DocVersion:       "0.3.0",
				SpeakeasyVersion: "1.3.1",
				ReleaseVersion:   "1.3.0",
			},
			Features: map[string]map[string]string{
				"go": {
					"core": "2.90.0",
				},
			},
		},
	}, cfg)
	_, err = os.Stat(filepath.Join(rootDir, "gen.yaml"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(speakeasyDir, "gen.lock"))
	assert.NoError(t, err)
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

func readTestFile(t *testing.T, file string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", file))
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}
