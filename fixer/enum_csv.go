package fixer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// fixCSVEnumsOAS2 expands CSV enum values in OAS 2.0 documents.
// This handles a common pattern where enum values for integer/number types
// are mistakenly stored as comma-separated strings (e.g., "1,2,3" instead of [1, 2, 3]).
func (f *Fixer) fixCSVEnumsOAS2(doc *parser.OAS2Document, result *FixResult) {
	if doc == nil {
		return
	}

	// Fix definitions
	for name, schema := range doc.Definitions {
		f.fixSchemaCSVEnums(schema, fmt.Sprintf("definitions.%s", name), result)
	}

	// Fix parameters
	for name, param := range doc.Parameters {
		if param != nil && param.Schema != nil {
			f.fixSchemaCSVEnums(param.Schema, fmt.Sprintf("parameters.%s.schema", name), result)
		}
	}

	// Fix path operations
	for path, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}
		f.fixPathItemCSVEnumsOAS2(pathItem, path, result)
	}
}

// fixPathItemCSVEnumsOAS2 fixes CSV enums in a path item.
func (f *Fixer) fixPathItemCSVEnumsOAS2(pathItem *parser.PathItem, path string, result *FixResult) {
	operations := []struct {
		method string
		op     *parser.Operation
	}{
		{"get", pathItem.Get},
		{"put", pathItem.Put},
		{"post", pathItem.Post},
		{"delete", pathItem.Delete},
		{"options", pathItem.Options},
		{"head", pathItem.Head},
		{"patch", pathItem.Patch},
	}

	for _, entry := range operations {
		if entry.op == nil {
			continue
		}
		basePath := fmt.Sprintf("paths.%s.%s", path, entry.method)

		// Fix parameters
		for i, param := range entry.op.Parameters {
			if param != nil && param.Schema != nil {
				f.fixSchemaCSVEnums(param.Schema, fmt.Sprintf("%s.parameters[%d].schema", basePath, i), result)
			}
		}

		// Fix responses
		if entry.op.Responses != nil {
			if entry.op.Responses.Default != nil && entry.op.Responses.Default.Schema != nil {
				f.fixSchemaCSVEnums(entry.op.Responses.Default.Schema, fmt.Sprintf("%s.responses.default.schema", basePath), result)
			}
			for code, resp := range entry.op.Responses.Codes {
				if resp != nil && resp.Schema != nil {
					f.fixSchemaCSVEnums(resp.Schema, fmt.Sprintf("%s.responses.%s.schema", basePath, code), result)
				}
			}
		}
	}
}

// fixCSVEnumsOAS3 expands CSV enum values in OAS 3.x documents.
func (f *Fixer) fixCSVEnumsOAS3(doc *parser.OAS3Document, result *FixResult) {
	if doc == nil {
		return
	}

	// Fix component schemas
	if doc.Components != nil {
		for name, schema := range doc.Components.Schemas {
			f.fixSchemaCSVEnums(schema, fmt.Sprintf("components.schemas.%s", name), result)
		}

		// Fix parameters
		for name, param := range doc.Components.Parameters {
			if param != nil && param.Schema != nil {
				f.fixSchemaCSVEnums(param.Schema, fmt.Sprintf("components.parameters.%s.schema", name), result)
			}
		}

		// Fix request bodies
		for name, reqBody := range doc.Components.RequestBodies {
			if reqBody != nil && reqBody.Content != nil {
				for mediaType, content := range reqBody.Content {
					if content != nil && content.Schema != nil {
						f.fixSchemaCSVEnums(content.Schema, fmt.Sprintf("components.requestBodies.%s.content.%s.schema", name, mediaType), result)
					}
				}
			}
		}

		// Fix responses
		for name, resp := range doc.Components.Responses {
			if resp != nil && resp.Content != nil {
				for mediaType, content := range resp.Content {
					if content != nil && content.Schema != nil {
						f.fixSchemaCSVEnums(content.Schema, fmt.Sprintf("components.responses.%s.content.%s.schema", name, mediaType), result)
					}
				}
			}
		}
	}

	// Fix path operations
	for path, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}
		f.fixPathItemCSVEnumsOAS3(pathItem, path, result)
	}
}

// fixPathItemCSVEnumsOAS3 fixes CSV enums in an OAS 3.x path item.
func (f *Fixer) fixPathItemCSVEnumsOAS3(pathItem *parser.PathItem, path string, result *FixResult) {
	operations := []struct {
		method string
		op     *parser.Operation
	}{
		{"get", pathItem.Get},
		{"put", pathItem.Put},
		{"post", pathItem.Post},
		{"delete", pathItem.Delete},
		{"options", pathItem.Options},
		{"head", pathItem.Head},
		{"patch", pathItem.Patch},
		{"trace", pathItem.Trace},
	}

	for _, entry := range operations {
		if entry.op == nil {
			continue
		}
		basePath := fmt.Sprintf("paths.%s.%s", path, entry.method)

		// Fix parameters
		for i, param := range entry.op.Parameters {
			if param != nil && param.Schema != nil {
				f.fixSchemaCSVEnums(param.Schema, fmt.Sprintf("%s.parameters[%d].schema", basePath, i), result)
			}
		}

		// Fix request body
		if entry.op.RequestBody != nil && entry.op.RequestBody.Content != nil {
			for mediaType, content := range entry.op.RequestBody.Content {
				if content != nil && content.Schema != nil {
					f.fixSchemaCSVEnums(content.Schema, fmt.Sprintf("%s.requestBody.content.%s.schema", basePath, mediaType), result)
				}
			}
		}

		// Fix responses
		if entry.op.Responses != nil {
			if entry.op.Responses.Default != nil && entry.op.Responses.Default.Content != nil {
				for mediaType, content := range entry.op.Responses.Default.Content {
					if content != nil && content.Schema != nil {
						f.fixSchemaCSVEnums(content.Schema, fmt.Sprintf("%s.responses.default.content.%s.schema", basePath, mediaType), result)
					}
				}
			}
			for code, resp := range entry.op.Responses.Codes {
				if resp != nil && resp.Content != nil {
					for mediaType, content := range resp.Content {
						if content != nil && content.Schema != nil {
							f.fixSchemaCSVEnums(content.Schema, fmt.Sprintf("%s.responses.%s.content.%s.schema", basePath, code, mediaType), result)
						}
					}
				}
			}
		}
	}
}

// fixSchemaCSVEnums recursively fixes CSV enum values in a schema.
func (f *Fixer) fixSchemaCSVEnums(schema *parser.Schema, path string, result *FixResult) {
	if schema == nil {
		return
	}

	// Check if this schema has CSV enums
	if isCSVEnumCandidate(schema) {
		expanded, skippedParts, hadExpansion := expandCSVEnumValues(schema)
		if hadExpansion && len(expanded) > 0 {
			before := schema.Enum
			schema.Enum = expanded

			description := fmt.Sprintf("expanded CSV enum string to %d individual values", len(expanded))
			if len(skippedParts) > 0 {
				description = fmt.Sprintf("expanded CSV enum string to %d values (skipped %d invalid: %s)",
					len(expanded), len(skippedParts), strings.Join(skippedParts, ", "))
			}

			fix := Fix{
				Type:        FixTypeEnumCSVExpanded,
				Path:        path,
				Description: description,
				Before:      before,
				After:       expanded,
			}
			f.populateFixLocation(&fix)
			result.Fixes = append(result.Fixes, fix)
		}
	}

	// Recurse into nested schemas
	for propName, propSchema := range schema.Properties {
		f.fixSchemaCSVEnums(propSchema, fmt.Sprintf("%s.properties.%s", path, propName), result)
	}

	if itemsSchema, ok := schema.Items.(*parser.Schema); ok && itemsSchema != nil {
		f.fixSchemaCSVEnums(itemsSchema, path+".items", result)
	}

	if addPropsSchema, ok := schema.AdditionalProperties.(*parser.Schema); ok && addPropsSchema != nil {
		f.fixSchemaCSVEnums(addPropsSchema, path+".additionalProperties", result)
	}

	for i, allOf := range schema.AllOf {
		f.fixSchemaCSVEnums(allOf, fmt.Sprintf("%s.allOf[%d]", path, i), result)
	}

	for i, anyOf := range schema.AnyOf {
		f.fixSchemaCSVEnums(anyOf, fmt.Sprintf("%s.anyOf[%d]", path, i), result)
	}

	for i, oneOf := range schema.OneOf {
		f.fixSchemaCSVEnums(oneOf, fmt.Sprintf("%s.oneOf[%d]", path, i), result)
	}

	if schema.Not != nil {
		f.fixSchemaCSVEnums(schema.Not, path+".not", result)
	}
}

// isCSVEnumCandidate returns true if the schema has an enum that looks like
// it contains CSV values that should be expanded.
func isCSVEnumCandidate(schema *parser.Schema) bool {
	if schema == nil || len(schema.Enum) == 0 {
		return false
	}

	// Only apply to integer or number types
	schemaType := getSchemaType(schema)
	if schemaType != "integer" && schemaType != "number" {
		return false
	}

	// Check if any enum value is a string containing a comma
	for _, v := range schema.Enum {
		if s, ok := v.(string); ok && strings.Contains(s, ",") {
			return true
		}
	}

	return false
}

// getSchemaType extracts the type from a schema, handling OAS 3.1+ type arrays.
func getSchemaType(schema *parser.Schema) string {
	if schema.Type == nil {
		return ""
	}
	switch t := schema.Type.(type) {
	case string:
		return t
	case []any:
		// For type arrays, look for non-null type
		for _, v := range t {
			if s, ok := v.(string); ok && s != "null" {
				return s
			}
		}
	}
	return ""
}

// expandCSVEnumValues expands CSV strings in enum values to individual values.
// Returns the expanded enum, any parts that were skipped due to parse errors,
// and whether any expansion occurred. Invalid values within a CSV string
// (e.g., non-numeric strings for integer type) are tracked in skippedParts.
func expandCSVEnumValues(schema *parser.Schema) (expanded []any, skippedParts []string, hadExpansion bool) {
	if schema == nil {
		return nil, nil, false
	}
	if len(schema.Enum) == 0 {
		return schema.Enum, nil, false
	}

	schemaType := getSchemaType(schema)
	if schemaType != "integer" && schemaType != "number" {
		return schema.Enum, nil, false
	}

	for _, v := range schema.Enum {
		switch val := v.(type) {
		case string:
			if strings.Contains(val, ",") {
				// This is a CSV string - expand it
				hadExpansion = true
				for part := range strings.SplitSeq(val, ",") {
					part = strings.TrimSpace(part)
					if part == "" {
						continue
					}
					parsed, err := parseNumericValue(part, schemaType)
					if err == nil {
						expanded = append(expanded, parsed)
					} else {
						// Track skipped parts for reporting
						skippedParts = append(skippedParts, part)
					}
				}
			} else {
				// Single value string - keep as-is
				expanded = append(expanded, val)
			}
		default:
			// Keep non-string values (already proper numeric types)
			expanded = append(expanded, v)
		}
	}

	return expanded, skippedParts, hadExpansion
}

// parseNumericValue parses a string into the appropriate numeric type.
func parseNumericValue(s string, schemaType string) (any, error) {
	switch schemaType {
	case "integer":
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("fixer: invalid integer value: %s", s)
		}
		return v, nil
	case "number":
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("fixer: invalid number value: %s", s)
		}
		return v, nil
	default:
		return nil, fmt.Errorf("fixer: unsupported type: %s", schemaType)
	}
}
