// This file implements schema conversion between OAS 2.0 and OAS 3.x formats.

package converter

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// convertOAS2SchemaToOAS3 converts an OAS 2.0 schema to OAS 3.x format
func (c *Converter) convertOAS2SchemaToOAS3(schema *parser.Schema) *parser.Schema {
	if schema == nil {
		return nil
	}

	// Deep copy to avoid mutations
	converted := schema.DeepCopy()

	// Rewrite all $ref paths from OAS 2.0 to OAS 3.x format
	rewriteSchemaRefsOAS2ToOAS3(converted)

	return converted
}

// convertOAS3SchemaToOAS2 converts an OAS 3.x schema to OAS 2.0 format
func (c *Converter) convertOAS3SchemaToOAS2(schema *parser.Schema, result *ConversionResult, path string) *parser.Schema {
	if schema == nil {
		return nil
	}

	// Deep copy to avoid mutations on the returned schema
	converted := schema.DeepCopy()

	// Recursively detect OAS 3.x features in the original schema (read-only traversal)
	walkSchemaFeatures(c, schema, result, path, make(map[*parser.Schema]bool))

	// Rewrite all $ref paths from OAS 3.x to OAS 2.0 format on the deep copy
	rewriteSchemaRefsOAS3ToOAS2(converted)

	return converted
}

// detectOAS3SchemaFeatures checks a single schema for OAS 3.x-only features
// that are incompatible with OAS 2.0 and records issues in the conversion result.
func detectOAS3SchemaFeatures(c *Converter, schema *parser.Schema, result *ConversionResult, path string) {
	// Check for nullable (OAS 3.0+)
	if schema.Nullable {
		c.addIssueWithContext(result, path, "Schema uses 'nullable' which is OAS 3.0+",
			"Consider using 'x-nullable' extension for OAS 2.0 compatibility")
	}

	// Check for writeOnly (OAS 3.0+)
	if schema.WriteOnly {
		c.addIssueWithContext(result, path, "Schema uses 'writeOnly' which is OAS 3.0+",
			"Consider using 'x-writeOnly' extension for OAS 2.0 compatibility")
	}

	// Check for deprecated on schemas (OAS 3.0+)
	if schema.Deprecated {
		c.addIssueWithContext(result, path, "Schema uses 'deprecated' which is OAS 3.0+",
			"Consider using 'x-deprecated' extension for OAS 2.0 compatibility")
	}

	// Check for if/then/else (JSON Schema 2020-12, OAS 3.1+)
	if schema.If != nil {
		c.addIssueWithContext(result, path, "Schema uses 'if' which is OAS 3.1+ (JSON Schema 2020-12)",
			"Conditional schema composition has no OAS 2.0 equivalent")
	}
	if schema.Then != nil {
		c.addIssueWithContext(result, path, "Schema uses 'then' which is OAS 3.1+ (JSON Schema 2020-12)",
			"Conditional schema composition has no OAS 2.0 equivalent")
	}
	if schema.Else != nil {
		c.addIssueWithContext(result, path, "Schema uses 'else' which is OAS 3.1+ (JSON Schema 2020-12)",
			"Conditional schema composition has no OAS 2.0 equivalent")
	}

	// Check for prefixItems (JSON Schema 2020-12, OAS 3.1+)
	if len(schema.PrefixItems) > 0 {
		c.addIssueWithContext(result, path, "Schema uses 'prefixItems' which is OAS 3.1+ (JSON Schema 2020-12)",
			"Tuple validation via 'prefixItems' has no OAS 2.0 equivalent")
	}

	// Check for contains (JSON Schema 2020-12, OAS 3.1+)
	if schema.Contains != nil {
		c.addIssueWithContext(result, path, "Schema uses 'contains' which is OAS 3.1+ (JSON Schema 2020-12)",
			"Array containment validation has no OAS 2.0 equivalent")
	}

	// Check for propertyNames (JSON Schema 2020-12, OAS 3.1+)
	if schema.PropertyNames != nil {
		c.addIssueWithContext(result, path, "Schema uses 'propertyNames' which is OAS 3.1+ (JSON Schema 2020-12)",
			"Property name validation has no OAS 2.0 equivalent")
	}
}

// walkSchemaFeatures recursively walks a schema and all nested schemas to detect
// OAS 3.x-only features that are incompatible with OAS 2.0. The visited map provides
// identity-based cycle detection using schema pointer identity. Schemas with a $ref
// set are skipped since the referenced definition will be checked separately at the
// top level.
func walkSchemaFeatures(c *Converter, schema *parser.Schema, result *ConversionResult, path string, visited map[*parser.Schema]bool) {
	if schema == nil || visited[schema] {
		return
	}
	visited[schema] = true

	// Skip schemas that have a $ref set — these point to definitions that will
	// be checked at the top level, so detecting features here would produce
	// duplicate warnings.
	if schema.Ref != "" {
		return
	}

	// Detect OAS 3.x features on the current schema
	detectOAS3SchemaFeatures(c, schema, result, path)

	// Recursively walk nested schemas in properties
	for name, propSchema := range schema.Properties {
		walkSchemaFeatures(c, propSchema, result, fmt.Sprintf("%s.properties.%s", path, name), visited)
	}

	for pattern, propSchema := range schema.PatternProperties {
		walkSchemaFeatures(c, propSchema, result, fmt.Sprintf("%s.patternProperties.%s", path, pattern), visited)
	}

	// Handle polymorphic fields with type assertion.
	// These can be bool (OAS 3.1+) or *Schema — only *Schema needs traversal.
	if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
		walkSchemaFeatures(c, addProps, result, fmt.Sprintf("%s.additionalProperties", path), visited)
	}

	if items, ok := schema.Items.(*parser.Schema); ok {
		walkSchemaFeatures(c, items, result, fmt.Sprintf("%s.items", path), visited)
	}

	// Composition keywords
	for i, subSchema := range schema.AllOf {
		walkSchemaFeatures(c, subSchema, result, fmt.Sprintf("%s.allOf[%d]", path, i), visited)
	}

	for i, subSchema := range schema.AnyOf {
		walkSchemaFeatures(c, subSchema, result, fmt.Sprintf("%s.anyOf[%d]", path, i), visited)
	}

	for i, subSchema := range schema.OneOf {
		walkSchemaFeatures(c, subSchema, result, fmt.Sprintf("%s.oneOf[%d]", path, i), visited)
	}

	walkSchemaFeatures(c, schema.Not, result, fmt.Sprintf("%s.not", path), visited)

	// Array-related keywords
	if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
		walkSchemaFeatures(c, addItems, result, fmt.Sprintf("%s.additionalItems", path), visited)
	}

	for i, prefixItem := range schema.PrefixItems {
		walkSchemaFeatures(c, prefixItem, result, fmt.Sprintf("%s.prefixItems[%d]", path, i), visited)
	}

	walkSchemaFeatures(c, schema.Contains, result, fmt.Sprintf("%s.contains", path), visited)

	// Object validation keywords
	walkSchemaFeatures(c, schema.PropertyNames, result, fmt.Sprintf("%s.propertyNames", path), visited)

	for name, depSchema := range schema.DependentSchemas {
		walkSchemaFeatures(c, depSchema, result, fmt.Sprintf("%s.dependentSchemas.%s", path, name), visited)
	}

	// JSON Schema 2020-12 unevaluated keywords (can be bool or *Schema)
	if unevalProps, ok := schema.UnevaluatedProperties.(*parser.Schema); ok {
		walkSchemaFeatures(c, unevalProps, result, fmt.Sprintf("%s.unevaluatedProperties", path), visited)
	}

	if unevalItems, ok := schema.UnevaluatedItems.(*parser.Schema); ok {
		walkSchemaFeatures(c, unevalItems, result, fmt.Sprintf("%s.unevaluatedItems", path), visited)
	}

	// JSON Schema 2020-12 content keywords
	walkSchemaFeatures(c, schema.ContentSchema, result, fmt.Sprintf("%s.contentSchema", path), visited)

	// Conditional keywords
	walkSchemaFeatures(c, schema.If, result, fmt.Sprintf("%s.if", path), visited)
	walkSchemaFeatures(c, schema.Then, result, fmt.Sprintf("%s.then", path), visited)
	walkSchemaFeatures(c, schema.Else, result, fmt.Sprintf("%s.else", path), visited)

	// Schema definitions
	for name, defSchema := range schema.Defs {
		walkSchemaFeatures(c, defSchema, result, fmt.Sprintf("%s.$defs.%s", path, name), visited)
	}
}
