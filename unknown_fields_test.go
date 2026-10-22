package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreserveUnknownFieldsRoundTrip(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "enabled"}[enabled], func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, ".speakeasy"), 0o755))
			cfg := &Configuration{
				ConfigVersion: Version,
				Languages: map[string]LanguageConfig{
					"python": {Version: "0.0.1", Cfg: map[string]any{"preserveUnknownFields": enabled}},
				},
			}
			require.NoError(t, SaveConfig(dir, cfg))
			loaded, err := Load(dir, WithLanguages("python"), WithDontWrite())
			require.NoError(t, err)
			require.Equal(t, enabled, loaded.Config.Languages["python"].Cfg["preserveUnknownFields"])
		})
	}
}
