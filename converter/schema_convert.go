// This file implements schema conversion between OAS 2.0 and OAS 3.x formats.

package converter

import (
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

	// Check for OAS 3.1+ features that may not be compatible with OAS 2.0
	converted := schema.DeepCopy()

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

	// Rewrite all $ref paths from OAS 3.x to OAS 2.0 format
	rewriteSchemaRefsOAS3ToOAS2(converted)

	return converted
}
