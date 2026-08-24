package fixer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/internal/maputil"
	"github.com/erraggy/oastools/internal/schemautil"
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
	for _, name := range maputil.SortedKeys(doc.Definitions) {
		f.fixSchemaCSVEnums(doc.Definitions[name], "definitions."+name, result)
	}

	// Fix parameters
	for _, name := range maputil.SortedKeys(doc.Parameters) {
		f.fixParameterCSVEnumsOAS2(doc.Parameters[name], "parameters."+name, result)
	}

	// Fix responses
	for _, name := range maputil.SortedKeys(doc.Responses) {
		f.fixResponseCSVEnumsOAS2(doc.Responses[name], "responses."+name, result)
	}

	// Fix path operations
	for _, path := range maputil.SortedKeys(doc.Paths) {
		f.fixPathItemCSVEnumsOAS2(doc.Paths[path], "paths."+path, result)
	}
}

// fixPathItemCSVEnumsOAS2 fixes CSV enums in a path item.
func (f *Fixer) fixPathItemCSVEnumsOAS2(pathItem *parser.PathItem, path string, result *FixResult) {
	if pathItem == nil {
		return
	}

	// A path item's parameters apply to every operation it holds, so they are
	// visited once here rather than inside the operation loop below.
	f.fixParametersCSVEnumsOAS2(pathItem.Parameters, path, result)

	operations := parser.GetOperations(pathItem, parser.OASVersion20)
	for _, method := range maputil.SortedKeys(operations) {
		op := operations[method]
		if op == nil {
			continue
		}
		basePath := path + "." + method

		f.fixParametersCSVEnumsOAS2(op.Parameters, basePath, result)

		if op.Responses != nil {
			f.fixResponseCSVEnumsOAS2(op.Responses.Default, basePath+".responses.default", result)
			for _, code := range maputil.SortedKeys(op.Responses.Codes) {
				f.fixResponseCSVEnumsOAS2(op.Responses.Codes[code], basePath+".responses."+code, result)
			}
		}
	}
}

// fixParametersCSVEnumsOAS2 fixes CSV enums in one OAS 2.0 parameter list,
// reporting each fix by the parameter's position.
func (f *Fixer) fixParametersCSVEnumsOAS2(params []*parser.Parameter, path string, result *FixResult) {
	for i, param := range params {
		f.fixParameterCSVEnumsOAS2(param, fmt.Sprintf("%s.parameters[%d]", path, i), result)
	}
}

// fixParameterCSVEnumsOAS2 fixes CSV enums in one OAS 2.0 parameter.
//
// Only a body parameter describes its values through a schema. Every other `in`
// carries type and enum on the parameter object itself, and OAS 2.0 offers it
// nowhere else to put them, so reaching the enum through Schema alone leaves
// those parameters unexamined (#513).
func (f *Fixer) fixParameterCSVEnumsOAS2(param *parser.Parameter, path string, result *FixResult) {
	if param == nil {
		return
	}
	f.fixSchemaCSVEnums(param.Schema, path+".schema", result)
	f.fixTypedCSVEnum(param.Type, &param.Enum, path, result)
	f.fixItemsCSVEnumsOAS2(param.Items, path+".items", result)
}

// fixResponseCSVEnumsOAS2 fixes CSV enums in one OAS 2.0 response: its schema
// and each of its headers.
func (f *Fixer) fixResponseCSVEnumsOAS2(resp *parser.Response, path string, result *FixResult) {
	if resp == nil {
		return
	}
	f.fixSchemaCSVEnums(resp.Schema, path+".schema", result)
	for _, name := range maputil.SortedKeys(resp.Headers) {
		f.fixHeaderCSVEnumsOAS2(resp.Headers[name], path+".headers."+name, result)
	}
}

// fixHeaderCSVEnumsOAS2 fixes CSV enums in one OAS 2.0 response header, which
// declares type and enum directly just as a non-body parameter does.
func (f *Fixer) fixHeaderCSVEnumsOAS2(header *parser.Header, path string, result *FixResult) {
	if header == nil {
		return
	}
	f.fixTypedCSVEnum(header.Type, &header.Enum, path, result)
	f.fixItemsCSVEnumsOAS2(header.Items, path+".items", result)
}

// fixItemsCSVEnumsOAS2 fixes CSV enums down an OAS 2.0 items chain. An array
// parameter constrains its elements here, so this is where such a parameter's
// enum lives.
func (f *Fixer) fixItemsCSVEnumsOAS2(items *parser.Items, path string, result *FixResult) {
	if items == nil {
		return
	}
	f.fixTypedCSVEnum(items.Type, &items.Enum, path, result)
	f.fixItemsCSVEnumsOAS2(items.Items, path+".items", result)
}

// fixCSVEnumsOAS3 expands CSV enum values in OAS 3.x documents.
func (f *Fixer) fixCSVEnumsOAS3(doc *parser.OAS3Document, result *FixResult) {
	if doc == nil {
		return
	}

	if comp := doc.Components; comp != nil {
		for _, name := range maputil.SortedKeys(comp.Schemas) {
			f.fixSchemaCSVEnums(comp.Schemas[name], "components.schemas."+name, result)
		}

		for _, name := range maputil.SortedKeys(comp.Parameters) {
			f.fixParameterCSVEnumsOAS3(comp.Parameters[name], "components.parameters."+name, result)
		}

		for _, name := range maputil.SortedKeys(comp.Headers) {
			f.fixHeaderCSVEnumsOAS3(comp.Headers[name], "components.headers."+name, result)
		}

		for _, name := range maputil.SortedKeys(comp.RequestBodies) {
			if reqBody := comp.RequestBodies[name]; reqBody != nil {
				f.fixContentCSVEnums(reqBody.Content, "components.requestBodies."+name+".content", result)
			}
		}

		for _, name := range maputil.SortedKeys(comp.Responses) {
			f.fixResponseCSVEnumsOAS3(comp.Responses[name], "components.responses."+name, result)
		}

		for _, name := range maputil.SortedKeys(comp.MediaTypes) {
			f.fixMediaTypeCSVEnums(comp.MediaTypes[name], "components.mediaTypes."+name, result)
		}

		for _, name := range maputil.SortedKeys(comp.PathItems) {
			f.fixPathItemCSVEnumsOAS3(comp.PathItems[name], "components.pathItems."+name, doc.OASVersion, result)
		}
	}

	// Fix path operations
	for _, path := range maputil.SortedKeys(doc.Paths) {
		f.fixPathItemCSVEnumsOAS3(doc.Paths[path], "paths."+path, doc.OASVersion, result)
	}
}

// fixPathItemCSVEnumsOAS3 fixes CSV enums in an OAS 3.x path item. The version
// decides which operations the path item may hold: TRACE arrives in 3.0, QUERY
// and additionalOperations in 3.2.
func (f *Fixer) fixPathItemCSVEnumsOAS3(pathItem *parser.PathItem, path string, version parser.OASVersion, result *FixResult) {
	if pathItem == nil {
		return
	}

	// A path item's parameters apply to every operation it holds, so they are
	// visited once here rather than inside the operation loop below.
	f.fixParametersCSVEnumsOAS3(pathItem.Parameters, path, result)

	operations := parser.GetOperations(pathItem, version)
	for _, method := range maputil.SortedKeys(operations) {
		op := operations[method]
		if op == nil {
			continue
		}
		basePath := path + "." + method

		f.fixParametersCSVEnumsOAS3(op.Parameters, basePath, result)

		if op.RequestBody != nil {
			f.fixContentCSVEnums(op.RequestBody.Content, basePath+".requestBody.content", result)
		}

		if op.Responses != nil {
			f.fixResponseCSVEnumsOAS3(op.Responses.Default, basePath+".responses.default", result)
			for _, code := range maputil.SortedKeys(op.Responses.Codes) {
				f.fixResponseCSVEnumsOAS3(op.Responses.Codes[code], basePath+".responses."+code, result)
			}
		}
	}
}

// fixParametersCSVEnumsOAS3 fixes CSV enums in one OAS 3.x parameter list,
// reporting each fix by the parameter's position.
func (f *Fixer) fixParametersCSVEnumsOAS3(params []*parser.Parameter, path string, result *FixResult) {
	for i, param := range params {
		f.fixParameterCSVEnumsOAS3(param, fmt.Sprintf("%s.parameters[%d]", path, i), result)
	}
}

// fixParameterCSVEnumsOAS3 fixes CSV enums in one OAS 3.x parameter. A
// parameter describes its values through either schema or content, never both.
func (f *Fixer) fixParameterCSVEnumsOAS3(param *parser.Parameter, path string, result *FixResult) {
	if param == nil {
		return
	}
	f.fixSchemaCSVEnums(param.Schema, path+".schema", result)
	f.fixContentCSVEnums(param.Content, path+".content", result)
}

// fixHeaderCSVEnumsOAS3 fixes CSV enums in one OAS 3.x header, which takes the
// same schema-or-content form as a parameter.
func (f *Fixer) fixHeaderCSVEnumsOAS3(header *parser.Header, path string, result *FixResult) {
	if header == nil {
		return
	}
	f.fixSchemaCSVEnums(header.Schema, path+".schema", result)
	f.fixContentCSVEnums(header.Content, path+".content", result)
}

// fixResponseCSVEnumsOAS3 fixes CSV enums in one OAS 3.x response: each media
// type it offers and each of its headers.
func (f *Fixer) fixResponseCSVEnumsOAS3(resp *parser.Response, path string, result *FixResult) {
	if resp == nil {
		return
	}
	f.fixContentCSVEnums(resp.Content, path+".content", result)
	for _, name := range maputil.SortedKeys(resp.Headers) {
		f.fixHeaderCSVEnumsOAS3(resp.Headers[name], path+".headers."+name, result)
	}
}

// fixContentCSVEnums fixes CSV enums in every media type of a content map.
func (f *Fixer) fixContentCSVEnums(content map[string]*parser.MediaType, path string, result *FixResult) {
	for _, mediaType := range maputil.SortedKeys(content) {
		f.fixMediaTypeCSVEnums(content[mediaType], path+"."+mediaType, result)
	}
}

// fixMediaTypeCSVEnums fixes CSV enums in one media type: the schema of the
// payload, and the schema of each item when the payload is sequential.
func (f *Fixer) fixMediaTypeCSVEnums(mediaType *parser.MediaType, path string, result *FixResult) {
	if mediaType == nil {
		return
	}
	f.fixSchemaCSVEnums(mediaType.Schema, path+".schema", result)
	f.fixSchemaCSVEnums(mediaType.ItemSchema, path+".itemSchema", result)
}

// fixSchemaCSVEnums recursively fixes CSV enum values in a schema.
func (f *Fixer) fixSchemaCSVEnums(schema *parser.Schema, path string, result *FixResult) {
	if schema == nil {
		return
	}

	f.fixTypedCSVEnum(getSchemaType(schema), &schema.Enum, path, result)

	// Recurse into nested schemas
	for propName, propSchema := range schema.Properties {
		f.fixSchemaCSVEnums(propSchema, fmt.Sprintf("%s.properties.%s", path, propName), result)
	}

	for i, itemsSchema := range schemautil.SchemaOrBoolSchemas(schema.Items) {
		f.fixSchemaCSVEnums(itemsSchema, path+".items"+schemautil.IndexSuffix(i), result)
	}

	for i, addPropsSchema := range schemautil.SchemaOrBoolSchemas(schema.AdditionalProperties) {
		f.fixSchemaCSVEnums(addPropsSchema, path+".additionalProperties"+schemautil.IndexSuffix(i), result)
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

// fixTypedCSVEnum expands one enum declaration in place and records the fix.
// The enum is addressed through a pointer because the declaration may be a
// schema's or an OAS 2.0 parameter's, header's, or items' own, and those share
// no type.
func (f *Fixer) fixTypedCSVEnum(schemaType string, enum *[]any, path string, result *FixResult) {
	expanded, skippedParts, hadExpansion := expandCSVEnumValues(schemaType, *enum)
	if !hadExpansion || len(expanded) == 0 {
		return
	}

	before := *enum
	*enum = expanded

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

// isCSVEnumCandidate returns true if the enum declared under schemaType looks
// like it contains CSV values that should be expanded.
func isCSVEnumCandidate(schemaType string, enum []any) bool {
	if len(enum) == 0 {
		return false
	}

	// Only apply to integer or number types
	if schemaType != "integer" && schemaType != "number" {
		return false
	}

	// Check if any enum value is a string containing a comma
	for _, v := range enum {
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
func expandCSVEnumValues(schemaType string, enum []any) (expanded []any, skippedParts []string, hadExpansion bool) {
	if !isCSVEnumCandidate(schemaType, enum) {
		return enum, nil, false
	}

	for _, v := range enum {
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
