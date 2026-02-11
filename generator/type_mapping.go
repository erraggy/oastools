// This file implements OpenAPI type/format to Go type mapping for code generation.

package generator

import (
	"strings"

	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// httpResponseType is the Go type for HTTP responses, used in zero value checks.
const httpResponseType = "*http.Response"

// getSchemaType extracts the type from a schema, handling both OAS 2.0/3.0
// string types and OAS 3.1+ type arrays.
func getSchemaType(schema *parser.Schema) string {
	if schema == nil {
		return ""
	}

	// Use schemautil for type extraction
	if primaryType := schemautil.GetPrimaryType(schema); primaryType != "" {
		return primaryType
	}

	// Infer type from other fields when no explicit type is set
	if schema.Properties != nil {
		return "object"
	}
	if schema.Items != nil {
		return "array"
	}
	if len(schema.Enum) > 0 {
		return "string"
	}

	return ""
}

// stringFormatToGoType maps OpenAPI string formats to Go types.
func stringFormatToGoType(format string) string {
	switch format {
	case "date-time":
		return "time.Time"
	case "date":
		return "string" // Could use time.Time with custom parsing
	case "time":
		return "string"
	case "byte":
		return "[]byte"
	case "binary":
		return "[]byte"
	default:
		return "string"
	}
}

// integerFormatToGoType maps OpenAPI integer formats to Go types.
func integerFormatToGoType(format string) string {
	switch format {
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	default:
		return "int64"
	}
}

// numberFormatToGoType maps OpenAPI number formats to Go types.
func numberFormatToGoType(format string) string {
	switch format {
	case "float":
		return "float32"
	case "double":
		return "float64"
	default:
		return "float64"
	}
}

// paramTypeToGoType converts an OAS parameter type/format to a Go type.
// This is shared between OAS 2.0 and OAS 3.x for parameters without schemas.
func paramTypeToGoType(paramType, format string) string {
	switch paramType {
	case "string":
		return stringFormatToGoType(format)
	case "integer":
		return integerFormatToGoType(format)
	case "number":
		return numberFormatToGoType(format)
	case "boolean":
		return "bool"
	case "array":
		return "[]string"
	default:
		return "string"
	}
}

// needsTimeImport recursively checks if a schema requires the "time" package.
func needsTimeImport(schema *parser.Schema) bool {
	if schema == nil {
		return false
	}

	schemaType := getSchemaType(schema)
	if schemaType == "string" && schema.Format == "date-time" {
		return true
	}

	// Check properties
	for _, prop := range schema.Properties {
		if needsTimeImport(prop) {
			return true
		}
	}

	// Check items
	if items, ok := schema.Items.(*parser.Schema); ok {
		if needsTimeImport(items) {
			return true
		}
	}

	return false
}

// zeroValue returns the Go zero value expression for a type.
func zeroValue(t string) string {
	if t == "" || t == httpResponseType {
		return "nil"
	}
	if strings.HasPrefix(t, "*") || strings.HasPrefix(t, "[]") || strings.HasPrefix(t, "map") {
		return "nil"
	}
	switch t {
	case "string":
		return `""`
	case "bool":
		return "false"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return "0"
	case "any":
		return "nil"
	default:
		return t + "{}"
	}
}

// schemaTypeFromMap returns the Go type string for a schema represented as a raw map.
// This handles cases where the parser returns additionalProperties or items as a
// map[string]any rather than a *parser.Schema.
func schemaTypeFromMap(m map[string]any) string {
	if typeVal, ok := m["type"]; ok {
		if typeStr, ok := typeVal.(string); ok {
			switch typeStr {
			case "string":
				if format, ok := m["format"].(string); ok {
					return stringFormatToGoType(format)
				}
				return "string"
			case "integer":
				if format, ok := m["format"].(string); ok {
					return integerFormatToGoType(format)
				}
				return "int64"
			case "number":
				if format, ok := m["format"].(string); ok {
					return numberFormatToGoType(format)
				}
				return "float64"
			case "boolean":
				return "bool"
			case "array":
				return "[]any"
			case "object":
				return "map[string]any"
			}
		}
	}
	return "any"
}
