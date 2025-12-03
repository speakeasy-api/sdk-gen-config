package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/speakeasy-api/sdk-gen-config"
	"github.com/speakeasy-api/sdk-gen-config/workflow"
	jsg "github.com/swaggest/jsonschema-go"
)

func main() {
	var (
		out        string
		schemaType string
	)
	flag.StringVar(&out, "out", "-", "output file path or - for stdout")
	flag.StringVar(&schemaType, "type", "workflow", "schema type to generate: workflow or config")
	flag.Parse()

	r := jsg.Reflector{}

	var (
		schema jsg.Schema
		err    error
	)

	// Customize reflection based on type
	switch schemaType {
	case "workflow":
		schema, err = r.Reflect(workflow.Workflow{}, func(rc *jsg.ReflectContext) {
			rc.PropertyNameTag = "yaml"
			rc.InlineRefs = false
		})
	case "config":
		schema, err = r.Reflect(config.Configuration{}, func(rc *jsg.ReflectContext) {
			rc.PropertyNameTag = "yaml"
			rc.InlineRefs = false
		})
	default:
		fmt.Fprintf(os.Stderr, "unknown schema type: %s\n", schemaType)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "reflect: %v\n", err)
		os.Exit(1)
	}

	// Set the $schema field to match the existing schema
	schemaMap := schema.ToSchemaOrBool().TypeObject
	if schemaMap == nil {
		schemaMap = &jsg.Schema{}
	}

	// If the root schema is a reference (common with InlineRefs=false),
	// unwrap it to make the properties top-level
	if schemaMap.Ref != nil {
		refName := strings.TrimPrefix(*schemaMap.Ref, "#/definitions/")
		if def, ok := schemaMap.Definitions[refName]; ok {
			defObj := def.TypeObject
			if defObj != nil {
				// Copy essential fields from definition to root
				schemaMap.Properties = defObj.Properties
				schemaMap.Required = defObj.Required
				schemaMap.AdditionalProperties = defObj.AdditionalProperties
				schemaMap.Description = defObj.Description
				schemaMap.Title = defObj.Title
				schemaMap.Type = defObj.Type

				// Remove the reference
				schemaMap.Ref = nil

				// Optionally remove the definition itself since it's now at root
				delete(schemaMap.Definitions, refName)
			}
		}
	}

	if schemaType == "workflow" {
		if schemaMap.Title == nil {
			schemaMap.WithTitle("Speakeasy Workflow Schema")
		}
		if schemaMap.AdditionalProperties == nil {
			schemaMap.WithAdditionalProperties(jsg.SchemaOrBool{TypeBoolean: boolPtr(false)})
		}
	} else if schemaType == "config" {
		if schemaMap.Title == nil {
			schemaMap.WithTitle("Gen YAML Configuration Schema")
		}
		if schemaMap.AdditionalProperties == nil {
			schemaMap.WithAdditionalProperties(jsg.SchemaOrBool{TypeBoolean: boolPtr(false)})
		}
	}

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

	// Config-specific post-processing
	if schemaType == "config" {
		patchConfigSchema(result)
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

// patchConfigSchema injects the specific language references and cleans up AdditionalProperties fields.
func patchConfigSchema(root map[string]interface{}) {
	// Helper function to process any schema object
	processSchema := func(schema map[string]interface{}) {
		props, ok := schema["properties"].(map[string]interface{})
		if !ok {
			return
		}

		// Remove AdditionalProperties field (from yaml:",inline" map fields)
		// and set additionalProperties: true to allow unknown properties
		if _, exists := props["AdditionalProperties"]; exists {
			delete(props, "AdditionalProperties")
			schema["additionalProperties"] = true
		}
	}

	// Apply language patches to ROOT properties only
	rootProps, ok := root["properties"].(map[string]interface{})
	if ok {
		// Remove the synthetic Languages property (comes from inline map in Go struct)
		delete(rootProps, "Languages")

		languages := map[string]string{
			"go":         "Go",
			"typescript": "TypeScript",
			"python":     "Python",
			"java":       "Java",
			"csharp":     "C#",
			"unity":      "Unity",
			"php":        "PHP",
			"ruby":       "Ruby",
			"postman":    "Postman Collections",
			"terraform":  "Terraform Providers",
		}
		for lang := range languages {
			rootProps[lang] = map[string]interface{}{
				"$ref": fmt.Sprintf("./languages/%s.schema.json", lang),
			}
		}
	}

	// Apply AdditionalProperties cleanup to root and all $defs
	processSchema(root)

	if defs, ok := root["$defs"].(map[string]interface{}); ok {
		for _, d := range defs {
			if schema, ok := d.(map[string]interface{}); ok {
				processSchema(schema)
			}
		}
	}
}
