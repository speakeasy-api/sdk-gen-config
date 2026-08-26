package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/speakeasy-api/sdk-gen-config/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGetNameResolution(t *testing.T) {
	tests := []struct {
		name     string
		gen      Generation
		expected NameResolutionMode
	}{
		{
			name:     "explicit mode wins over stale booleans",
			gen:      Generation{NameResolution: NameResolutionOrdered, Fixes: &Fixes{NameResolutionFeb2025: true}},
			expected: NameResolutionOrdered,
		},
		{
			name:     "derived from dec2023 boolean",
			gen:      Generation{Fixes: &Fixes{NameResolutionDec2023: true}},
			expected: NameResolutionOrdered,
		},
		{
			name:     "derived from feb2025 boolean",
			gen:      Generation{Fixes: &Fixes{NameResolutionFeb2025: true}},
			expected: NameResolutionShortest,
		},
		{
			name:     "false booleans derive legacy",
			gen:      Generation{Fixes: &Fixes{}},
			expected: NameResolutionLegacy,
		},
		{
			name:     "zero config defaults to the current default",
			gen:      Generation{},
			expected: NameResolutionShortest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.gen.GetNameResolution())
		})
	}
}

func TestNameResolutionMode_IsValid(t *testing.T) {
	assert.True(t, NameResolutionLegacy.IsValid())
	assert.True(t, NameResolutionOrdered.IsValid())
	assert.True(t, NameResolutionShortest.IsValid())
	assert.False(t, NameResolutionMode("").IsValid())
	assert.False(t, NameResolutionMode("newest").IsValid())
}

func TestSyncNameResolution_MaterializesDerivedMode(t *testing.T) {
	g := Generation{Fixes: &Fixes{NameResolutionFeb2025: true}}
	g.SyncNameResolution()

	assert.Equal(t, NameResolutionShortest, g.NameResolution)
	assert.True(t, g.Fixes.NameResolutionDec2023)
	assert.True(t, g.Fixes.NameResolutionFeb2025)
}

func TestSyncNameResolution_ExplicitModeWinsAndBackfills(t *testing.T) {
	g := Generation{NameResolution: NameResolutionOrdered, Fixes: &Fixes{NameResolutionDec2023: true, NameResolutionFeb2025: true}}
	g.SyncNameResolution()

	assert.Equal(t, NameResolutionOrdered, g.NameResolution)
	assert.True(t, g.Fixes.NameResolutionDec2023)
	assert.False(t, g.Fixes.NameResolutionFeb2025)
}

func TestSyncNameResolution_ModeBackfillsAbsentFixes(t *testing.T) {
	g := Generation{NameResolution: NameResolutionShortest}
	g.SyncNameResolution()

	require.NotNil(t, g.Fixes)
	assert.True(t, g.Fixes.NameResolutionDec2023)
	assert.True(t, g.Fixes.NameResolutionFeb2025)
}

func TestSyncNameResolution_LegacyStaysUntouched(t *testing.T) {
	g := Generation{SDKClassName: "SDK"}
	g.SyncNameResolution()
	assert.Equal(t, NameResolutionMode(""), g.NameResolution)
	assert.Nil(t, g.Fixes)

	g = Generation{Fixes: &Fixes{}}
	g.SyncNameResolution()
	assert.Equal(t, NameResolutionMode(""), g.NameResolution)
	assert.False(t, g.Fixes.NameResolutionDec2023)

	out, err := yaml.Marshal(g)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "nameResolution:")
}

func TestSyncNameResolution_ExplicitLegacyKeepsModeWithoutFixesBlock(t *testing.T) {
	g := Generation{NameResolution: NameResolutionLegacy}
	g.SyncNameResolution()

	assert.Equal(t, NameResolutionLegacy, g.NameResolution)
	assert.Nil(t, g.Fixes)
}

func TestNameResolution_PlainYAMLRoundTrip(t *testing.T) {
	var g Generation
	require.NoError(t, yaml.Unmarshal([]byte("nameResolution: ordered\n"), &g))
	assert.Equal(t, NameResolutionOrdered, g.NameResolution)
	assert.Nil(t, g.Fixes)

	g.SyncNameResolution()
	out, err := yaml.Marshal(g)
	require.NoError(t, err)
	assert.Contains(t, string(out), "nameResolution: ordered")
	assert.Contains(t, string(out), "nameResolutionDec2023: true")
	assert.Contains(t, string(out), "nameResolutionFeb2025: false")
}

// File-level translation: gen.yaml -> Load -> effective mode -> SyncNameResolution -> SaveConfig -> gen.yaml.
func TestNameResolution_LoadSaveRoundTrip(t *testing.T) {
	tests := []struct {
		name           string
		genYaml        string
		expectedMode   NameResolutionMode
		expectContains []string
		expectAbsent   []string
	}{
		{
			name: "booleans-only file gains explicit mode",
			genYaml: `configVersion: 2.0.0
generation:
  sdkClassName: test
  fixes:
    nameResolutionDec2023: true
    nameResolutionFeb2025: true
go:
  version: 1.0.0
`,
			expectedMode:   NameResolutionShortest,
			expectContains: []string{"nameResolution: shortest", "nameResolutionDec2023: true", "nameResolutionFeb2025: true"},
		},
		{
			name: "dec2023-only file gains ordered mode",
			genYaml: `configVersion: 2.0.0
generation:
  sdkClassName: test
  fixes:
    nameResolutionDec2023: true
    nameResolutionFeb2025: false
go:
  version: 1.0.0
`,
			expectedMode:   NameResolutionOrdered,
			expectContains: []string{"nameResolution: ordered", "nameResolutionDec2023: true", "nameResolutionFeb2025: false"},
		},
		{
			name: "legacy file gains no mode",
			genYaml: `configVersion: 2.0.0
generation:
  sdkClassName: test
go:
  version: 1.0.0
`,
			expectedMode:   NameResolutionLegacy,
			expectContains: []string{"nameResolutionFeb2025: false"},
			expectAbsent:   []string{"nameResolution:"},
		},
		{
			name: "mode-only file back-fills booleans",
			genYaml: `configVersion: 2.0.0
generation:
  sdkClassName: test
  nameResolution: ordered
go:
  version: 1.0.0
`,
			expectedMode:   NameResolutionOrdered,
			expectContains: []string{"nameResolution: ordered", "nameResolutionDec2023: true", "nameResolutionFeb2025: false"},
		},
		{
			name: "mode wins over stale booleans",
			genYaml: `configVersion: 2.0.0
generation:
  sdkClassName: test
  nameResolution: legacy
  fixes:
    nameResolutionDec2023: true
    nameResolutionFeb2025: true
go:
  version: 1.0.0
`,
			expectedMode:   NameResolutionLegacy,
			expectContains: []string{"nameResolution: legacy", "nameResolutionFeb2025: false"},
			expectAbsent:   []string{"nameResolutionDec2023: true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			speakeasyDir := filepath.Join(dir, ".speakeasy")
			testutils.CreateTempFile(t, speakeasyDir, "gen.yaml", tt.genYaml)
			testutils.CreateTempFile(t, speakeasyDir, "gen.lock", testutils.ReadTestFile(t, "v200-gen.lock"))

			cfg, err := Load(dir, WithLanguages("go"))
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMode, cfg.Config.Generation.GetNameResolution())

			cfg.Config.Generation.SyncNameResolution()
			require.NoError(t, SaveConfig(dir, cfg.Config))

			out, err := os.ReadFile(filepath.Join(speakeasyDir, "gen.yaml"))
			require.NoError(t, err)
			for _, s := range tt.expectContains {
				assert.Contains(t, string(out), s)
			}
			for _, s := range tt.expectAbsent {
				assert.NotContains(t, string(out), s)
			}
		})
	}
}

func TestNameResolutionMode_AtLeast(t *testing.T) {
	assert.True(t, NameResolutionShortest.AtLeast(NameResolutionOrdered))
	assert.True(t, NameResolutionOrdered.AtLeast(NameResolutionOrdered))
	assert.False(t, NameResolutionLegacy.AtLeast(NameResolutionOrdered))
	assert.False(t, NameResolutionOrdered.AtLeast(NameResolutionShortest))
}
