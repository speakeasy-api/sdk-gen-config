package config

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/sdk-gen-config/lockfile"
	"github.com/speakeasy-api/sdk-gen-config/testutils"
	"github.com/speakeasy-api/sdk-gen-config/workspace"
	"github.com/stretchr/testify/assert"
)

const testDir = "gen/test"

func TestLoad_Success(t *testing.T) {
	getUUID = func() string {
		return "123"
	}
	lockfile.GetUUID = getUUID

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
							SDKInitStyle:              "constructor",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                true,
							NameResolutionFeb2025:                true,
							ParameterOrderingFeb2024:             true,
							RequestResponseComponentNamesFeb2024: true,
							SecurityFeb2025:                      true,
							SharedErrorComponentsApr2025:         true,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: true,
							OAuth2PasswordEnabled:          true,
							HoistGlobalSecurity:            true,
						},
						Tests: Tests{
							GenerateTests:    false,
							GenerateNewTests: true,
						},
						UseClassNamesForArrayFields: true,
						InferSSEOverload:            true,
						SDKHooksConfigAccess:        true,
						Schemas: Schemas{
							AllOfMergeStrategy: AllOfMergeStrategyShallowMerge,
						},
						RequestBodyFieldName: "body",
					},
					New: map[string]bool{
						"go": true,
					},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, ".speakeasy/gen.yaml"),
				LockFile: &LockFile{
					LockVersion:    lockfile.LockV2,
					ID:             "123",
					Management:     Management{},
					Features:       make(map[string]map[string]string),
					TrackedFiles: sequencedmap.New[string, TrackedFile](),
				},
			},
		},
		{
			name: "loads and upgrades pre v1.0.0 config file",
			args: args{
				langs:     []string{"go"},
				configDir: testDir,
				targetDir: testDir,
				genYaml:   testutils.ReadTestFile(t, "pre-v100-gen.yaml"),
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
							SDKInitStyle:              "constructor",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                false,
							ParameterOrderingFeb2024:             false,
							RequestResponseComponentNamesFeb2024: false,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: false,
							OAuth2PasswordEnabled:          false,
							HoistGlobalSecurity:            true,
						},
						Tests: Tests{
							GenerateTests:    true,
							GenerateNewTests: false,
						},
						Schemas: Schemas{
							AllOfMergeStrategy: AllOfMergeStrategyShallowMerge,
						},
					},
					New: map[string]bool{},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, "gen.yaml"),
				LockFile: &LockFile{
					LockVersion: lockfile.LockV2,
					ID:          "123",
					Management: Management{
						DocChecksum:      "2bba3b8f9d211b02569b3f9aff0d34b4",
						DocVersion:       "0.3.0",
						SpeakeasyVersion: "1.3.1",
						ReleaseVersion:   "1.3.0",
					},
					Features:       make(map[string]map[string]string),
					TrackedFiles: sequencedmap.New[string, TrackedFile](),
				},
			},
		},
		{
			name: "loads v1.0.0 config file",
			args: args{
				langs:        []string{"go"},
				configDir:    testDir,
				targetDir:    testDir,
				genYaml:      testutils.ReadTestFile(t, "v100-gen.yaml"),
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
							SDKInitStyle:              "constructor",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                false,
							ParameterOrderingFeb2024:             false,
							RequestResponseComponentNamesFeb2024: false,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: false,
							OAuth2PasswordEnabled:          false,
							HoistGlobalSecurity:            true,
						},
						Tests: Tests{
							GenerateTests:    true,
							GenerateNewTests: false,
						},
						Schemas: Schemas{
							AllOfMergeStrategy: AllOfMergeStrategyShallowMerge,
						},
					},
					New: map[string]bool{},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, ".speakeasy/gen.yaml"),
				LockFile: &LockFile{
					LockVersion: lockfile.LockV2,
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
					TrackedFiles: sequencedmap.New[string, TrackedFile](),
				},
			},
		},
		{
			name: "loads v2.0.0 config file",
			args: args{
				langs:        []string{"go"},
				configDir:    testDir,
				targetDir:    testDir,
				genYaml:      testutils.ReadTestFile(t, "v200-gen.yaml"),
				lockFile:     testutils.ReadTestFile(t, "v200-gen.lock"),
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
							SDKInitStyle:              "constructor",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                false,
							ParameterOrderingFeb2024:             false,
							RequestResponseComponentNamesFeb2024: false,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: false,
							OAuth2PasswordEnabled:          false,
							HoistGlobalSecurity:            true,
						},
						Tests: Tests{
							GenerateTests:    true,
							GenerateNewTests: false,
						},
						Schemas: Schemas{
							AllOfMergeStrategy: AllOfMergeStrategyShallowMerge,
						},
					},
					New: map[string]bool{},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, ".speakeasy/gen.yaml"),
				LockFile: &LockFile{
					LockVersion: lockfile.LockV2,
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
					TrackedFiles: sequencedmap.New[string, TrackedFile](),
				},
			},
		},
		{
			name: "loads v2.0.0 config file without existing lock file as a new sdk",
			args: args{
				langs:        []string{"go"},
				configDir:    testDir,
				targetDir:    testDir,
				genYaml:      testutils.ReadTestFile(t, "v200-gen.yaml"),
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
							SDKInitStyle:              "constructor",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                true,
							ParameterOrderingFeb2024:             true,
							RequestResponseComponentNamesFeb2024: true,
							NameResolutionFeb2025:                true,
							SecurityFeb2025:                      true,
							SharedErrorComponentsApr2025:         true,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: true,
							OAuth2PasswordEnabled:          true,
							HoistGlobalSecurity:            true,
						},
						Tests: Tests{
							GenerateTests:    false,
							GenerateNewTests: true,
						},
						MaintainOpenAPIOrder:        true,
						UseClassNamesForArrayFields: true,
						InferSSEOverload:            true,
						SDKHooksConfigAccess:        true,
						Schemas: Schemas{
							AllOfMergeStrategy: AllOfMergeStrategyShallowMerge,
						},
						RequestBodyFieldName: "body",
					},
					New: map[string]bool{
						"go": true,
					},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, ".speakeasy/gen.yaml"),
				LockFile: &LockFile{
					LockVersion:    lockfile.LockV2,
					ID:             "123",
					Management:     Management{},
					Features:       make(map[string]map[string]string),
					TrackedFiles: sequencedmap.New[string, TrackedFile](),
				},
			},
		},
		{
			name: "loads v2.0.0 config file without existing lock file as a new sdk from .gen folder",
			args: args{
				langs:        []string{"go"},
				configDir:    testDir,
				targetDir:    testDir,
				genYaml:      testutils.ReadTestFile(t, "v200-gen.yaml"),
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
							SDKInitStyle:              "constructor",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                true,
							ParameterOrderingFeb2024:             true,
							RequestResponseComponentNamesFeb2024: true,
							SecurityFeb2025:                      true,
							SharedErrorComponentsApr2025:         true,
							NameResolutionFeb2025:                true,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: true,
							OAuth2PasswordEnabled:          true,
							HoistGlobalSecurity:            true,
						},
						Tests: Tests{
							GenerateTests:    false,
							GenerateNewTests: true,
						},
						MaintainOpenAPIOrder:        true,
						UseClassNamesForArrayFields: true,
						InferSSEOverload:            true,
						SDKHooksConfigAccess:        true,
						Schemas: Schemas{
							AllOfMergeStrategy: AllOfMergeStrategyShallowMerge,
						},
						RequestBodyFieldName: "body",
					},
					New: map[string]bool{
						"go": true,
					},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, ".gen/gen.yaml"),
				LockFile: &LockFile{
					LockVersion:    lockfile.LockV2,
					ID:             "123",
					Management:     Management{},
					Features:       make(map[string]map[string]string),
					TrackedFiles: sequencedmap.New[string, TrackedFile](),
				},
			},
		},
		{
			name: "loads v2.0.0 config file from higher level directory",
			args: args{
				langs:     []string{"go"},
				configDir: filepath.Dir(testDir),
				targetDir: testDir,
				genYaml:   testutils.ReadTestFile(t, "v200-gen.yaml"),
				lockFile:  testutils.ReadTestFile(t, "v200-gen.lock"),
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
							SDKInitStyle:              "constructor",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                false,
							ParameterOrderingFeb2024:             false,
							RequestResponseComponentNamesFeb2024: false,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: false,
							OAuth2PasswordEnabled:          false,
							HoistGlobalSecurity:            true,
						},
						Tests: Tests{
							GenerateTests:    true,
							GenerateNewTests: false,
						},
						Schemas: Schemas{
							AllOfMergeStrategy: AllOfMergeStrategyShallowMerge,
						},
					},
					New: map[string]bool{},
				},
				ConfigPath: filepath.Join(os.TempDir(), filepath.Dir(testDir), "gen.yaml"),
				LockFile: &LockFile{
					LockVersion: lockfile.LockV2,
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
					TrackedFiles: sequencedmap.New[string, TrackedFile](),
				},
			},
		},
		{
			name: "loads v100 config file and detects new config for language",
			args: args{
				langs:     []string{"go", "typescript"},
				configDir: testDir,
				targetDir: testDir,
				genYaml:   testutils.ReadTestFile(t, "v100-gen.yaml"),
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
							SDKInitStyle:              "constructor",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                false,
							ParameterOrderingFeb2024:             false,
							RequestResponseComponentNamesFeb2024: false,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: false,
							OAuth2PasswordEnabled:          false,
							HoistGlobalSecurity:            true,
						},
						Tests: Tests{
							GenerateTests:    true,
							GenerateNewTests: false,
						},
						Schemas: Schemas{
							AllOfMergeStrategy: AllOfMergeStrategyShallowMerge,
						},
					},
					New: map[string]bool{
						"typescript": true,
					},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, "gen.yaml"),
				LockFile: &LockFile{
					LockVersion: lockfile.LockV2,
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
					TrackedFiles: sequencedmap.New[string, TrackedFile](),
				},
			},
		},
		{
			name: "loads v2.0.0 config file and detects new config for language",
			args: args{
				langs:        []string{"go", "typescript"},
				configDir:    testDir,
				targetDir:    testDir,
				genYaml:      testutils.ReadTestFile(t, "v200-gen.yaml"),
				lockFile:     testutils.ReadTestFile(t, "v200-gen.lock"),
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
							SDKInitStyle:              "constructor",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                false,
							ParameterOrderingFeb2024:             false,
							RequestResponseComponentNamesFeb2024: false,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: false,
							OAuth2PasswordEnabled:          false,
							HoistGlobalSecurity:            true,
						},
						Tests: Tests{
							GenerateTests:    true,
							GenerateNewTests: false,
						},
						Schemas: Schemas{
							AllOfMergeStrategy: AllOfMergeStrategyShallowMerge,
						},
					},
					New: map[string]bool{
						"typescript": true,
					},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, ".speakeasy/gen.yaml"),
				LockFile: &LockFile{
					LockVersion: lockfile.LockV2,
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
					TrackedFiles: sequencedmap.New[string, TrackedFile](),
				},
			},
		},
		{
			name: "loads v2.0.0 config file with mockServer.disabled",
			args: args{
				langs:        []string{"go"},
				configDir:    testDir,
				targetDir:    testDir,
				genYaml:      testutils.ReadTestFile(t, "v200-generation-mockserver-disabled.yaml"),
				lockFile:     testutils.ReadTestFile(t, "v200-gen.lock"),
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
							SDKInitStyle:              "constructor",
						},
						Fixes: &Fixes{
							NameResolutionDec2023:                false,
							ParameterOrderingFeb2024:             false,
							RequestResponseComponentNamesFeb2024: false,
						},
						Auth: &Auth{
							OAuth2ClientCredentialsEnabled: false,
							OAuth2PasswordEnabled:          false,
							HoistGlobalSecurity:            true,
						},
						Tests: Tests{
							GenerateTests:    true,
							GenerateNewTests: false,
						},
						MockServer: &MockServer{
							Disabled: true,
						},
						Schemas: Schemas{
							AllOfMergeStrategy: AllOfMergeStrategyShallowMerge,
						},
					},
					New: map[string]bool{},
				},
				ConfigPath: filepath.Join(os.TempDir(), testDir, ".speakeasy/gen.yaml"),
				LockFile: &LockFile{
					LockVersion: lockfile.LockV2,
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
					TrackedFiles: sequencedmap.New[string, TrackedFile](),
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

			testutils.CreateTempFile(t, configDir, "gen.yaml", tt.args.genYaml)
			testutils.CreateTempFile(t, lockFileDir, "gen.lock", tt.args.lockFile)

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
	lockfile.GetUUID = getUUID

	// Create new config file in .speakeasy dir
	speakeasyDir := filepath.Join(os.TempDir(), testDir, workspace.SpeakeasyFolder)
	testutils.CreateTempFile(t, speakeasyDir, "gen.yaml", testutils.ReadTestFile(t, "v200-gen.yaml"))

	// Create old config file in root dir
	rootDir := filepath.Join(os.TempDir(), testDir)
	testutils.CreateTempFile(t, rootDir, "gen.yaml", testutils.ReadTestFile(t, "v100-gen.yaml"))

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
					SDKInitStyle:              "constructor",
				},
				Fixes: &Fixes{
					NameResolutionDec2023:                false,
					ParameterOrderingFeb2024:             false,
					RequestResponseComponentNamesFeb2024: false,
				},
				Auth: &Auth{
					OAuth2ClientCredentialsEnabled: false,
					OAuth2PasswordEnabled:          false,
					HoistGlobalSecurity:            true,
				},
				Tests: Tests{
					GenerateTests:    true,
					GenerateNewTests: false,
				},
				Schemas: Schemas{
					AllOfMergeStrategy: AllOfMergeStrategyShallowMerge,
				},
			},
			New: map[string]bool{},
		},
		ConfigPath: filepath.Join(os.TempDir(), testDir, "gen.yaml"),
		LockFile: &LockFile{
			LockVersion: lockfile.LockV2,
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
			TrackedFiles: sequencedmap.New[string, TrackedFile](),
		},
	}, cfg)
	_, err = os.Stat(filepath.Join(rootDir, "gen.yaml"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(speakeasyDir, "gen.lock"))
	assert.NoError(t, err)
}

func TestSaveConfig(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		cfg      *Configuration
		opts     []Option
		expected string
	}{
		"no-options": {
			cfg: &Configuration{
				ConfigVersion: "0.0.0",
				Generation: Generation{
					Schemas: Schemas{
						AllOfMergeStrategy: AllOfMergeStrategyShallowMerge,
					},
				},
			},
			expected: `configVersion: 0.0.0
generation:
  schemas:
    allOfMergeStrategy: shallowMerge
  requestBodyFieldName: ""
  persistentEdits: {}
`,
		},
		"option-dontwrite": {
			cfg: &Configuration{
				ConfigVersion: Version,
			},
			opts: []Option{WithDontWrite()},
		},
	}

	for name, testCase := range testCases {
		name, testCase := name, testCase

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			speakeasyPath := filepath.Join(tempDir, ".speakeasy")
			configPath := filepath.Join(speakeasyPath, "gen.yaml")

			err := os.Mkdir(speakeasyPath, 0o755)
			assert.NoError(t, err)

			err = SaveConfig(tempDir, testCase.cfg, testCase.opts...)
			assert.NoError(t, err)

			fileInfo, err := os.Stat(configPath)

			if len(testCase.expected) == 0 {
				assert.ErrorIs(t, err, fs.ErrNotExist)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, false, fileInfo.IsDir())
			assert.Equal(t, fs.FileMode(0o644), fileInfo.Mode())

			contents, err := os.ReadFile(configPath)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, string(contents))
		})
	}
}

func TestSaveLockFile(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		lf       *LockFile
		opts     []Option
		expected []byte
	}{
		"no-options": {
			lf: &LockFile{
				LockVersion: "0.0.0",
			},
			expected: []byte(`lockVersion: 0.0.0
id: ""
management: {}
`),
		},
		"option-dontwrite": {
			lf: &LockFile{
				LockVersion: v2,
			},
			opts: []Option{WithDontWrite()},
		},
	}

	for name, testCase := range testCases {
		name, testCase := name, testCase

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			speakeasyPath := filepath.Join(tempDir, ".speakeasy")
			configPath := filepath.Join(speakeasyPath, "gen.lock")

			err := os.Mkdir(speakeasyPath, 0o755)
			assert.NoError(t, err)

			err = SaveLockFile(tempDir, testCase.lf, testCase.opts...)
			assert.NoError(t, err)

			fileInfo, err := os.Stat(configPath)

			if len(testCase.expected) == 0 {
				assert.ErrorIs(t, err, fs.ErrNotExist)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, false, fileInfo.IsDir())
			assert.Equal(t, fs.FileMode(0o644), fileInfo.Mode())

			contents, err := os.ReadFile(configPath)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, contents)
		})
	}
}
