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

	// Rewrite all $ref paths from OAS 3.x to OAS 2.0 format
	rewriteSchemaRefsOAS3ToOAS2(converted)

	return converted
}
