package generator

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// maxDescriptionLength is the maximum length for descriptions in Go comments
// before truncation. This keeps generated code readable and prevents excessively
// long comment lines.
const maxDescriptionLength = 200

// httpResponseType is the Go type for HTTP responses, used in zero value checks.
const httpResponseType = "*http.Response"

// goReservedWords contains Go reserved keywords that cannot be used as identifiers.
// Note: We only include actual keywords, not predeclared identifiers like "error",
// because those can be shadowed and are commonly used as type names (e.g., "Error").
var goReservedWords = map[string]bool{
	// Keywords (these are truly reserved and cannot be used)
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// escapeReservedWord checks if a name is a Go reserved keyword and escapes it
// by appending an underscore if necessary. Note: This only checks exact matches
// since Go keywords are case-sensitive.
func escapeReservedWord(name string) string {
	// Check the lowercase version for keywords (they're all lowercase)
	if goReservedWords[strings.ToLower(name)] {
		return name + "_"
	}
	return name
}

// toTypeName converts an OpenAPI name to a valid Go type name (PascalCase).
// It handles special characters, ensures the name starts with a letter,
// and escapes Go reserved words.
func toTypeName(s string) string {
	if s == "" {
		return "Type"
	}

	// Split on non-alphanumeric and capitalize each part
	var result strings.Builder
	capitalizeNext := true

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if capitalizeNext {
				result.WriteRune(unicode.ToUpper(r))
				capitalizeNext = false
			} else {
				result.WriteRune(r)
			}
		} else {
			capitalizeNext = true
		}
	}

	name := result.String()

	// Ensure starts with a letter
	if len(name) > 0 && !unicode.IsLetter(rune(name[0])) {
		name = "T" + name
	}

	return escapeReservedWord(name)
}

// toFieldName converts an OpenAPI property name to a valid Go field name (PascalCase).
// It handles special characters, ensures the name starts with a letter,
// and escapes Go reserved words.
func toFieldName(s string) string {
	return toTypeName(s)
}

// toParamName converts an OpenAPI parameter name to a valid Go parameter name (camelCase).
// It handles special characters and escapes Go reserved words.
func toParamName(s string) string {
	name := toTypeName(s)
	if len(name) > 0 {
		// Convert first character to lowercase for camelCase
		name = strings.ToLower(name[:1]) + name[1:]
	} else {
		name = "param"
	}
	return escapeReservedWord(name)
}

// operationToMethodName generates a Go method name from an operation.
// It uses the operationId if available, otherwise generates from path and method.
func operationToMethodName(op *parser.Operation, path, method string) string {
	if op.OperationID != "" {
		return toTypeName(op.OperationID)
	}
	// Generate from path and method
	pathPart := path
	pathPart = strings.ReplaceAll(pathPart, "/", " ")
	pathPart = strings.ReplaceAll(pathPart, "{", "By ")
	pathPart = strings.ReplaceAll(pathPart, "}", "")
	return toTypeName(method + " " + pathPart)
}

// cleanDescription prepares an OpenAPI description for use in Go comments.
// It removes newlines, trims whitespace, and truncates long descriptions.
func cleanDescription(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) > maxDescriptionLength {
		s = s[:maxDescriptionLength-3] + "..."
	}
	return s
}

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

// isRequired checks if a property name is in the required list.
func isRequired(required []string, name string) bool {
	for _, r := range required {
		if r == name {
			return true
		}
	}
	return false
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
	return t + "{}"
}

// schemaFromMap attempts to convert a map[string]interface{} to schema properties.
// This handles cases where the parser returns additionalProperties as a map
// rather than a *parser.Schema.
func schemaTypeFromMap(m map[string]interface{}) string {
	if typeVal, ok := m["type"]; ok {
		if typeStr, ok := typeVal.(string); ok {
			switch typeStr {
			case "string":
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

// buildDefaultUserAgent generates the default User-Agent string for generated clients.
// Format: oastools/{version}/generated/{title}
// If title is empty, it uses "API Client" as a fallback.
func buildDefaultUserAgent(info *parser.Info) string {
	version := oastools.Version()
	title := "API Client"
	if info != nil && info.Title != "" {
		title = info.Title
	}
	return fmt.Sprintf("oastools/%s/generated/%s", version, title)
}
