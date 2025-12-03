package config_test

//go:generate sh -c "cd tools/schema-gen && go run . -type config -out ../../schemas/gen.config.schema.json"

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConfigSchemaInSync verifies that gen.config.schema.json is in sync with
// what the schema generator produces from the Configuration struct.
func TestConfigSchemaInSync(t *testing.T) {
	// Generate schema from current Go structs
	cmd := exec.Command("go", "run", ".", "-type", "config", "-out", "-")
	cmd.Dir = filepath.Join("tools", "schema-gen")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	require.NoError(t, err, "schema generator failed: %s", stderr.String())

	// Read the committed schema
	committedPath := filepath.Join("schemas", "gen.config.schema.json")
	committedBytes, err := os.ReadFile(committedPath)
	require.NoError(t, err, "Failed to read committed schema")

	// Compare byte-for-byte
	generated := stdout.Bytes()
	require.Equal(t, string(committedBytes), string(generated),
		"Generated config schema does not match committed schemas/gen.config.schema.json.\n"+
			"Run: cd tools/schema-gen && go run . -type config -out ../../schemas/gen.config.schema.json\n"+
			"Then commit the updated file.")
}
