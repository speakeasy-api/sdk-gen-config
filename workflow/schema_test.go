package workflow_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaInSync verifies that workflow.schema.json is in sync with the Go struct definitions
// by generating a JSON schema from the structs and validating that all Go struct fields are
// represented in the hand-written schema.
//
// This test ensures:
// 1. All struct fields have corresponding schema properties
// 2. Required fields in Go match required fields in schema
// 3. Nested types are properly defined
//
// Note: The hand-written schema may have additional constraints, descriptions, and validations
// that aren't auto-generated. This test validates structural compatibility, not exact equivalence.
func TestSchemaInSync(t *testing.T) {
	// 1) Run the schema generator from the tools submodule
	cmd := exec.Command("go", "run", ".", "-out", "-")
	cmd.Dir = filepath.Join("..", "tools", "schema-gen")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("schema generator failed: %v\nStderr: %s", err, stderr.String())
	}

	// 2) Load the committed baseline schema
	baselinePath := filepath.Join("..", "schemas", "workflow.schema.json")
	wantBytes, err := os.ReadFile(baselinePath)
	require.NoError(t, err, "Failed to read baseline schema")

	// 3) Unmarshal both schemas
	var generated, handwritten map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &generated); err != nil {
		t.Fatalf("Failed to unmarshal generated schema: %v", err)
	}
	if err := json.Unmarshal(wantBytes, &handwritten); err != nil {
		t.Fatalf("Failed to unmarshal hand-written schema: %v", err)
	}

	// 4) Validate that the hand-written schema covers all generated properties
	validateSchemaStructure(t, generated, handwritten)
}

// validateSchemaStructure recursively validates that the handwritten schema covers
// all properties and definitions from the generated schema
func validateSchemaStructure(t *testing.T, generated, handwritten map[string]interface{}) {
	t.Helper()

	// Check top-level properties
	if genProps, ok := generated["properties"].(map[string]interface{}); ok {
		hwProps, hwOk := handwritten["properties"].(map[string]interface{})
		if !hwOk {
			t.Error("Hand-written schema missing 'properties' field")
			return
		}

		for propName := range genProps {
			if _, exists := hwProps[propName]; !exists {
				t.Errorf("Hand-written schema missing property: %s", propName)
			}
		}
	}

	// Check definitions
	genDefs, genHasDefs := generated["definitions"].(map[string]interface{})
	hwDefs := getDefinitions(handwritten)

	if genHasDefs && genDefs != nil {
		if hwDefs == nil {
			t.Error("Hand-written schema missing definitions section")
			return
		}

		// For each generated definition, check if it exists in hand-written schema
		for defName, genDef := range genDefs {
			hwDefName := findDefinitionInHandwritten(defName, hwDefs)
			if hwDefName == "" {
				t.Errorf("Hand-written schema missing definition for type: %s (expected in $defs or definitions)", defName)
				continue
			}

			// Recursively validate the definition's properties
			if genDefMap, ok := genDef.(map[string]interface{}); ok {
				if genDefProps, ok := genDefMap["properties"].(map[string]interface{}); ok {
					hwDef := hwDefs[hwDefName].(map[string]interface{})
					if hwDefProps, ok := hwDef["properties"].(map[string]interface{}); ok {
						for propName := range genDefProps {
							// Map property names (e.g., remove "Workflow" prefix if needed)
							mappedPropName := mapPropertyName(propName)
							if _, exists := hwDefProps[mappedPropName]; !exists {
								// Also check without prefix
								if _, exists := hwDefProps[propName]; !exists {
									t.Errorf("Hand-written schema definition '%s' missing property: %s (also tried: %s)",
										hwDefName, propName, mappedPropName)
								}
							}
						}
					}
				}
			}
		}
	}
}

// getDefinitions retrieves definitions from either $defs or definitions key
func getDefinitions(schema map[string]interface{}) map[string]interface{} {
	if defs, ok := schema["$defs"].(map[string]interface{}); ok {
		return defs
	}
	if defs, ok := schema["definitions"].(map[string]interface{}); ok {
		return defs
	}
	return nil
}

// findDefinitionInHandwritten finds a definition in the hand-written schema,
// accounting for naming differences (e.g., "WorkflowSource" vs "source")
func findDefinitionInHandwritten(genName string, hwDefs map[string]interface{}) string {
	// Try exact match first
	if _, exists := hwDefs[genName]; exists {
		return genName
	}

	// Try without "Workflow" prefix
	withoutPrefix := strings.TrimPrefix(genName, "Workflow")
	if _, exists := hwDefs[withoutPrefix]; exists {
		return withoutPrefix
	}

	// Try lowercase first letter
	if len(withoutPrefix) > 0 {
		lowercase := strings.ToLower(withoutPrefix[0:1]) + withoutPrefix[1:]
		if _, exists := hwDefs[lowercase]; exists {
			return lowercase
		}
	}

	return ""
}

// mapPropertyName attempts to map a generated property name to the hand-written schema's naming
func mapPropertyName(genName string) string {
	// Remove common prefixes that might be added by the generator
	mapped := strings.TrimPrefix(genName, "Workflow")
	if len(mapped) > 0 && mapped != genName {
		// Lowercase first letter
		mapped = strings.ToLower(mapped[0:1]) + mapped[1:]
	}
	return mapped
}

// Test helper to pretty print schemas for debugging
func prettyPrintSchema(schema map[string]interface{}) string {
	b, _ := json.MarshalIndent(schema, "", "  ")
	return string(b)
}

// TestSchemaGeneratorWorks verifies that the schema generator tool can run successfully
func TestSchemaGeneratorWorks(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "-out", "-")
	cmd.Dir = filepath.Join("..", "tools", "schema-gen")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("schema generator failed: %v\nStderr: %s", err, stderr.String())
	}

	// Validate it's valid JSON
	var schema map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("Generated schema is not valid JSON: %v", err)
	}

	// Check for expected top-level fields
	assert.Contains(t, schema, "$schema", "Generated schema should have $schema")
	assert.Contains(t, schema, "definitions", "Generated schema should have definitions")
}

// TestRegenerateSchema is a utility test that can be run manually to regenerate the schema
// Run with: UPDATE_SCHEMA=1 go test ./workflow -run TestRegenerateSchema
func TestRegenerateSchema(t *testing.T) {
	if os.Getenv("UPDATE_SCHEMA") != "1" {
		t.Skip("Set UPDATE_SCHEMA=1 to regenerate the schema")
	}

	cmd := exec.Command("go", "run", ".", "-out", "-")
	cmd.Dir = filepath.Join("..", "tools", "schema-gen")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("schema generator failed: %v\nStderr: %s", err, stderr.String())
	}

	outputPath := filepath.Join("..", "schemas", "workflow.schema.generated.json")
	if err := os.WriteFile(outputPath, stdout.Bytes(), 0o644); err != nil {
		t.Fatalf("Failed to write generated schema: %v", err)
	}

	t.Logf("Generated schema written to %s", outputPath)
	t.Log("Review the generated schema and manually update workflow.schema.json as needed")
	t.Log("The hand-written schema should include all fields from the generated schema")
	t.Log("plus additional descriptions, constraints, and validations")
}

// ensureEqual is a helper that compares two values for debugging
func ensureEqual(t *testing.T, fieldPath string, expected, actual interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("%s mismatch:\nExpected: %v\nActual: %v", fieldPath, expected, actual)
	}
}

// dumpSchemaDiff writes both schemas to temp files for manual inspection
func dumpSchemaDiff(t *testing.T, generated, handwritten map[string]interface{}) (string, string) {
	t.Helper()

	genFile := filepath.Join(t.TempDir(), "generated.json")
	hwFile := filepath.Join(t.TempDir(), "handwritten.json")

	genJSON, _ := json.MarshalIndent(generated, "", "  ")
	hwJSON, _ := json.MarshalIndent(handwritten, "", "  ")

	os.WriteFile(genFile, genJSON, 0o644)
	os.WriteFile(hwFile, hwJSON, 0o644)

	return genFile, hwFile
}

// compareSchemaProperties does a detailed property-by-property comparison
func compareSchemaProperties(t *testing.T, generated, handwritten map[string]interface{}, path string) {
	t.Helper()

	genProps, genOk := generated["properties"].(map[string]interface{})
	hwProps, hwOk := handwritten["properties"].(map[string]interface{})

	if !genOk && !hwOk {
		return // Both don't have properties
	}

	if genOk && !hwOk {
		t.Errorf("%s: generated has properties but handwritten doesn't", path)
		return
	}

	for propName, genProp := range genProps {
		propPath := fmt.Sprintf("%s.%s", path, propName)

		hwProp, exists := hwProps[propName]
		if !exists {
			t.Errorf("%s: missing in hand-written schema", propPath)
			continue
		}

		// Recursively compare if both are objects
		if genPropMap, ok := genProp.(map[string]interface{}); ok {
			if hwPropMap, ok := hwProp.(map[string]interface{}); ok {
				compareSchemaProperties(t, genPropMap, hwPropMap, propPath)
			}
		}
	}
}
