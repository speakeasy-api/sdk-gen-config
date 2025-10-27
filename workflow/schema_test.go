package workflow_test

//go:generate sh -c "cd ../tools/schema-gen && go run . -out ../../schemas/workflow.schema.generated.json"

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSchemaInSync verifies that workflow.schema.generated.json is in sync with
// what the schema generator produces. This ensures the committed schema matches
// the current Go struct definitions.
func TestSchemaInSync(t *testing.T) {
	// Generate schema from current Go structs
	cmd := exec.Command("go", "run", ".", "-out", "-")
	cmd.Dir = filepath.Join("..", "tools", "schema-gen")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	require.NoError(t, err, "schema generator failed: %s", stderr.String())

	// Read the committed schema
	committedPath := filepath.Join("..", "schemas", "workflow.schema.generated.json")
	committedBytes, err := os.ReadFile(committedPath)
	require.NoError(t, err, "Failed to read committed schema")

	// Compare byte-for-byte
	generated := stdout.Bytes()
	require.Equal(t, string(committedBytes), string(generated),
		"Generated schema does not match committed workflow.schema.generated.json.\n"+
			"Run: cd tools/schema-gen && go run . -out workflow.schema.generated.json\n"+
			"Then commit the updated file.")
}
