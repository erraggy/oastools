// base_code_generator.go contains shared code generation logic for OAS 2.0 and 3.x

package generator

import (
	"strings"

	"github.com/erraggy/oastools/parser"
)

// baseCodeGenerator contains shared fields and methods for both OAS versions
type baseCodeGenerator struct {
	g              *Generator
	result         *GenerateResult
	schemaNames    map[string]string // maps schema references to generated type names
	generatedTypes map[string]bool   // tracks which type names have been generated
	splitPlan      *SplitPlan        // file splitting plan for large APIs
}

// initBase initializes the base code generator fields
func (b *baseCodeGenerator) initBase(g *Generator, result *GenerateResult) {
	b.g = g
	b.result = result
	b.schemaNames = make(map[string]string)
	b.generatedTypes = make(map[string]bool)
}

// resolveRef resolves a $ref to a Go type name
func (b *baseCodeGenerator) resolveRef(ref string) string {
	if typeName, ok := b.schemaNames[ref]; ok {
		return typeName
	}
	// Extract name from ref path
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return toTypeName(parts[len(parts)-1])
	}
	return "any"
}

// addIssue adds a generation issue
//
//nolint:unparam // severity parameter kept for API consistency and future extensibility
func (b *baseCodeGenerator) addIssue(path, message string, severity Severity) {
	issue := GenerateIssue{
		Path:     path,
		Message:  message,
		Severity: severity,
	}
	b.populateIssueLocation(&issue, path)
	b.result.Issues = append(b.result.Issues, issue)
}

// populateIssueLocation fills in Line/Column/File from the SourceMap if available.
func (b *baseCodeGenerator) populateIssueLocation(issue *GenerateIssue, path string) {
	if b.g.SourceMap == nil {
		return
	}

	// Convert path format if needed (generator uses dotted paths like "definitions.Pet",
	// while SourceMap uses JSON path notation like "$.definitions.Pet")
	jsonPath := path
	if len(jsonPath) == 0 || jsonPath[0] != '$' {
		jsonPath = "$." + path
	}

	loc := b.g.SourceMap.Get(jsonPath)
	if loc.IsKnown() {
		issue.Line = loc.Line
		issue.Column = loc.Column
		issue.File = loc.File
	}
}

// getAdditionalPropertiesType extracts the Go type for additionalProperties
func (b *baseCodeGenerator) getAdditionalPropertiesType(schema *parser.Schema, schemaToGoType func(*parser.Schema, bool) string) string {
	if schema.AdditionalProperties == nil {
		return "any"
	}

	switch addProps := schema.AdditionalProperties.(type) {
	case *parser.Schema:
		return schemaToGoType(addProps, true)
	case map[string]any:
		return schemaTypeFromMap(addProps)
	case bool:
		if addProps {
			return "any"
		}
	}
	return "any"
}

// getArrayItemType extracts the Go type for array items, handling $ref properly
func (b *baseCodeGenerator) getArrayItemType(schema *parser.Schema, schemaToGoType func(*parser.Schema, bool) string) string {
	if schema.Items == nil {
		return "any"
	}

	switch items := schema.Items.(type) {
	case *parser.Schema:
		if items.Ref != "" {
			return b.resolveRef(items.Ref)
		}
		return schemaToGoType(items, true)
	case map[string]any:
		if ref, ok := items["$ref"].(string); ok {
			return b.resolveRef(ref)
		}
		return schemaTypeFromMap(items)
	}
	return "any"
}

// schemaToGoTypeBase is the shared logic for converting a schema to a Go type.
// isNullable is provided by the caller since OAS3 has additional nullable checks.
func (b *baseCodeGenerator) schemaToGoTypeBase(schema *parser.Schema, required bool, isNullable bool, schemaToGoType func(*parser.Schema, bool) string) string {
	if schema == nil {
		return "any"
	}

	// Handle $ref
	if schema.Ref != "" {
		refType := b.resolveRef(schema.Ref)
		if !required && b.g.UsePointers {
			return "*" + refType
		}
		return refType
	}

	schemaType := getSchemaType(schema)
	var goType string

	switch schemaType {
	case "string":
		goType = stringFormatToGoType(schema.Format)
	case "integer":
		goType = integerFormatToGoType(schema.Format)
	case "number":
		goType = numberFormatToGoType(schema.Format)
	case "boolean":
		goType = "bool"
	case "array":
		goType = "[]" + b.getArrayItemType(schema, schemaToGoType)
	case "object":
		if schema.Properties == nil && schema.AdditionalProperties != nil {
			// Map type
			goType = "map[string]" + b.getAdditionalPropertiesType(schema, schemaToGoType)
		} else {
			goType = "map[string]any"
		}
	default:
		goType = "any"
	}

	// Handle optional fields with pointers
	if !required && b.g.UsePointers && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map") {
		return "*" + goType
	}

	// Handle nullable with pointers (OAS 3.x)
	if isNullable && b.g.UsePointers && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map") && !strings.HasPrefix(goType, "*") {
		return "*" + goType
	}

	return goType
}
