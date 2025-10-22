package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/speakeasy-api/sdk-gen-config/workflow"
	jsg "github.com/swaggest/jsonschema-go"
)

func main() {
	var (
		out string
	)
	flag.StringVar(&out, "out", "-", "output file path or - for stdout")
	flag.Parse()

	r := jsg.Reflector{}

	schema, err := r.Reflect(workflow.Workflow{}, func(rc *jsg.ReflectContext) {
		// Use yaml tags for property names to match the Go structs
		rc.PropertyNameTag = "yaml"
		rc.InlineRefs = false // Use $ref for reusable definitions
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "reflect: %v\n", err)
		os.Exit(1)
	}

	// Set the $schema field to match the existing schema
	schemaMap := schema.ToSchemaOrBool().TypeObject
	if schemaMap == nil {
		schemaMap = &jsg.Schema{}
	}
	schemaMap.WithTitle("Speakeasy Workflow Schema")
	schemaMap.WithAdditionalProperties(jsg.SchemaOrBool{TypeBoolean: boolPtr(false)})

	b, err := json.MarshalIndent(schemaMap, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal schema: %v\n", err)
		os.Exit(1)
	}

	// Prepend $schema field and apply customizations
	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal for $schema addition: %v\n", err)
		os.Exit(1)
	}
	result["$schema"] = "https://json-schema.org/draft/2020-12/schema"

	// Convert "definitions" to "$defs" for draft 2020-12
	if defs, ok := result["definitions"].(map[string]interface{}); ok {
		result["$defs"] = defs
		delete(result, "definitions")

		// Update all $ref pointers from #/definitions/ to #/$defs/
		updateRefs(result)
	}

	b, err = json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "final marshal: %v\n", err)
		os.Exit(1)
	}

	if out == "-" {
		os.Stdout.Write(b)
		os.Stdout.Write([]byte("\n"))
		return
	}
	if err := os.WriteFile(out, append(b, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", out, err)
		os.Exit(1)
	}
}

func boolPtr(b bool) *bool {
	return &b
}

// updateRefs recursively updates all $ref from #/definitions/ to #/$defs/
func updateRefs(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		for k, v := range val {
			if k == "$ref" {
				if ref, ok := v.(string); ok {
					val[k] = strings.Replace(ref, "#/definitions/", "#/$defs/", 1)
				}
			} else {
				updateRefs(v)
			}
		}
	case []interface{}:
		for _, item := range val {
			updateRefs(item)
		}
	}
}
