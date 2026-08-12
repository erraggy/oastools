// This file implements schema conversion between OAS 2.0 and OAS 3.x formats.

package converter

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// convertOAS2SchemaToOAS3 converts an OAS 2.0 schema to OAS 3.x format
func (c *Converter) convertOAS2SchemaToOAS3(schema *parser.Schema, targetVersion parser.OASVersion, result *ConversionResult, path string) *parser.Schema {
	if schema == nil {
		return nil
	}

	// Deep copy to avoid mutations
	converted := schema.DeepCopy()

	// Rewrite all $ref paths from OAS 2.0 to OAS 3.x format
	rewriteSchemaRefsOAS2ToOAS3(converted)

	// Promote the OAS 2.0 bare-string discriminator to the OAS 3.x object form
	discriminatorToObjectForm(converted)

	// For OAS 3.1+, convert boolean exclusiveMaximum/exclusiveMinimum to numeric form
	if c.isOAS31OrLater(targetVersion) {
		fixSchemaExclusiveMinMaxForOAS31(c, converted, result, path, make(map[*parser.Schema]bool))
	}

	return converted
}

// discriminatorToObjectForm clears StringForm on every discriminator in the
// schema tree so they serialize as OAS 3.x objects. The property name carries
// over unchanged; only the spelling differs between dialects.
func discriminatorToObjectForm(schema *parser.Schema) {
	walkSchemas(schema, func(s *parser.Schema) {
		if s.Discriminator != nil {
			s.Discriminator.StringForm = false
		}
	})
}

// discriminatorToStringForm sets StringForm on every discriminator in the
// schema tree so they serialize as OAS 2.0 bare strings. OAS 2.0 spells the
// discriminator as a bare string with no object to hang anything off, so both
// the 3.x mapping and any specification extensions are dropped and reported.
func discriminatorToStringForm(c *Converter, schema *parser.Schema, result *ConversionResult, path string) {
	walkSchemas(schema, func(s *parser.Schema) {
		d := s.Discriminator
		if d == nil {
			return
		}
		if len(d.Mapping) > 0 {
			c.addIssueWithContext(result, path,
				"Schema discriminator uses 'mapping' which has no OAS 2.0 equivalent; mapping dropped",
				"OAS 2.0 resolves the discriminator by schema name only; rename the target definitions to match the discriminator values")
			d.Mapping = nil
		}
		// defaultMapping (OAS 3.2+) has no OAS 2.0 equivalent either, and unlike
		// mapping it names a single fallback schema, so losing it changes which
		// schema validates a payload with no discriminating value. Reported and
		// cleared here rather than left set: the string form drops it on
		// serialization regardless, and dropping it silently is the defect issue
		// #397 is about.
		if d.DefaultMapping != "" {
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Schema discriminator uses 'defaultMapping' (%s) which has no OAS 2.0 equivalent; defaultMapping dropped", d.DefaultMapping),
				"OAS 2.0 has no fallback for a missing or unmapped discriminator value; describe the fallback schema explicitly in the enclosing definition")
			d.DefaultMapping = ""
		}
		if len(d.Extra) > 0 {
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Schema discriminator carries extensions (%s) which have no OAS 2.0 equivalent; extensions dropped",
					strings.Join(slices.Sorted(maps.Keys(d.Extra)), ", ")),
				"OAS 2.0 spells the discriminator as a bare string, so it has no object to hold extensions; move them onto the enclosing Schema Object, which does accept them")
			d.Extra = nil
		}
		d.StringForm = true
	})
}

// fixSchemaExclusiveMinMaxForOAS31 recursively converts boolean exclusiveMaximum/exclusiveMinimum
// to OAS 3.1+ numeric semantics (number replaces boolean+maximum pair).
// Schemas with a $ref are skipped -- they are resolved separately.
// When result is non-nil, warnings are emitted for malformed constraints (true with no bound).
func fixSchemaExclusiveMinMaxForOAS31(c *Converter, schema *parser.Schema, result *ConversionResult, path string, visited map[*parser.Schema]bool) {
	if schema == nil || visited[schema] || schema.Ref != "" {
		return
	}
	visited[schema] = true

	if v, ok := schema.ExclusiveMaximum.(bool); ok {
		if v && schema.Maximum != nil {
			schema.ExclusiveMaximum = *schema.Maximum
			schema.Maximum = nil
		} else if v {
			if result != nil {
				c.addIssueWithContext(result, path,
					"Schema has 'exclusiveMaximum: true' but no 'maximum' value; constraint dropped in OAS 3.1 conversion",
					"Add a 'maximum' value to preserve this exclusive boundary in OAS 3.1")
			}
			schema.ExclusiveMaximum = nil
		} else {
			// false -- remove the no-op keyword
			schema.ExclusiveMaximum = nil
		}
	}
	if v, ok := schema.ExclusiveMinimum.(bool); ok {
		if v && schema.Minimum != nil {
			schema.ExclusiveMinimum = *schema.Minimum
			schema.Minimum = nil
		} else if v {
			if result != nil {
				c.addIssueWithContext(result, path,
					"Schema has 'exclusiveMinimum: true' but no 'minimum' value; constraint dropped in OAS 3.1 conversion",
					"Add a 'minimum' value to preserve this exclusive boundary in OAS 3.1")
			}
			schema.ExclusiveMinimum = nil
		} else {
			schema.ExclusiveMinimum = nil
		}
	}

	for name, s := range schema.Properties {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.properties.%s", path, name), visited)
	}
	for pattern, s := range schema.PatternProperties {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.patternProperties.%s", path, pattern), visited)
	}
	for i, s := range schemautil.SchemaOrBoolSchemas(schema.AdditionalProperties) {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, path+".additionalProperties"+schemautil.IndexSuffix(i), visited)
	}
	for i, s := range schemautil.SchemaOrBoolSchemas(schema.Items) {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, path+".items"+schemautil.IndexSuffix(i), visited)
	}
	for i, s := range schema.AllOf {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.allOf[%d]", path, i), visited)
	}
	for i, s := range schema.AnyOf {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.anyOf[%d]", path, i), visited)
	}
	for i, s := range schema.OneOf {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.oneOf[%d]", path, i), visited)
	}
	fixSchemaExclusiveMinMaxForOAS31(c, schema.Not, result, path+".not", visited)
	for i, s := range schemautil.SchemaOrBoolSchemas(schema.AdditionalItems) {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, path+".additionalItems"+schemautil.IndexSuffix(i), visited)
	}
	for i, s := range schema.PrefixItems {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.prefixItems[%d]", path, i), visited)
	}
	fixSchemaExclusiveMinMaxForOAS31(c, schema.Contains, result, path+".contains", visited)
	fixSchemaExclusiveMinMaxForOAS31(c, schema.PropertyNames, result, path+".propertyNames", visited)
	for name, s := range schema.DependentSchemas {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.dependentSchemas.%s", path, name), visited)
	}
	for i, s := range schemautil.SchemaOrBoolSchemas(schema.UnevaluatedProperties) {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, path+".unevaluatedProperties"+schemautil.IndexSuffix(i), visited)
	}
	for i, s := range schemautil.SchemaOrBoolSchemas(schema.UnevaluatedItems) {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, path+".unevaluatedItems"+schemautil.IndexSuffix(i), visited)
	}
	fixSchemaExclusiveMinMaxForOAS31(c, schema.ContentSchema, result, path+".contentSchema", visited)
	fixSchemaExclusiveMinMaxForOAS31(c, schema.If, result, path+".if", visited)
	fixSchemaExclusiveMinMaxForOAS31(c, schema.Then, result, path+".then", visited)
	fixSchemaExclusiveMinMaxForOAS31(c, schema.Else, result, path+".else", visited)
	for name, s := range schema.Defs {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.$defs.%s", path, name), visited)
	}
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

	// Demote the OAS 3.x discriminator object to the OAS 2.0 bare-string form
	discriminatorToStringForm(c, converted, result, path)

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

	// Schema-or-bool fields
	for i, addProps := range schemautil.SchemaOrBoolSchemas(schema.AdditionalProperties) {
		walkSchemaFeatures(c, addProps, result, fmt.Sprintf("%s.additionalProperties%s", path, schemautil.IndexSuffix(i)), visited)
	}

	for i, items := range schemautil.SchemaOrBoolSchemas(schema.Items) {
		walkSchemaFeatures(c, items, result, fmt.Sprintf("%s.items%s", path, schemautil.IndexSuffix(i)), visited)
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
	for i, addItems := range schemautil.SchemaOrBoolSchemas(schema.AdditionalItems) {
		walkSchemaFeatures(c, addItems, result, fmt.Sprintf("%s.additionalItems%s", path, schemautil.IndexSuffix(i)), visited)
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

	// JSON Schema 2020-12 unevaluated keywords
	for i, unevalProps := range schemautil.SchemaOrBoolSchemas(schema.UnevaluatedProperties) {
		walkSchemaFeatures(c, unevalProps, result, fmt.Sprintf("%s.unevaluatedProperties%s", path, schemautil.IndexSuffix(i)), visited)
	}

	for i, unevalItems := range schemautil.SchemaOrBoolSchemas(schema.UnevaluatedItems) {
		walkSchemaFeatures(c, unevalItems, result, fmt.Sprintf("%s.unevaluatedItems%s", path, schemautil.IndexSuffix(i)), visited)
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
