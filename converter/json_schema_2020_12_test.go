// json_schema_2020_12_test.go tests JSON Schema Draft 2020-12 field handling in converter.
package converter

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWalkSchemaRefs_UnevaluatedProperties tests that refs in unevaluatedProperties are traversed
func TestWalkSchemaRefs_UnevaluatedProperties(t *testing.T) {
	t.Run("unevaluatedProperties as *Schema with ref", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "object",
			UnevaluatedProperties: &parser.Schema{
				Ref: "#/definitions/AdditionalData",
			},
		}

		rewriteSchemaRefsOAS2ToOAS3(schema)

		unevalProps := schema.UnevaluatedProperties.(*parser.Schema)
		assert.Equal(t, "#/components/schemas/AdditionalData", unevalProps.Ref,
			"UnevaluatedProperties ref should be rewritten")
	})

	t.Run("unevaluatedProperties as bool", func(t *testing.T) {
		schema := &parser.Schema{
			Type:                  "object",
			UnevaluatedProperties: false, // bool, not *Schema
		}

		// Should not panic
		rewriteSchemaRefsOAS2ToOAS3(schema)
		assert.Equal(t, false, schema.UnevaluatedProperties)
	})

	t.Run("unevaluatedProperties as nested *Schema with ref", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "object",
			UnevaluatedProperties: &parser.Schema{
				Properties: map[string]*parser.Schema{
					"nested": {Ref: "#/definitions/NestedType"},
				},
			},
		}

		rewriteSchemaRefsOAS2ToOAS3(schema)

		unevalProps := schema.UnevaluatedProperties.(*parser.Schema)
		assert.Equal(t, "#/components/schemas/NestedType", unevalProps.Properties["nested"].Ref,
			"Nested ref in UnevaluatedProperties should be rewritten")
	})
}

// TestWalkSchemaRefs_UnevaluatedItems tests that refs in unevaluatedItems are traversed
func TestWalkSchemaRefs_UnevaluatedItems(t *testing.T) {
	t.Run("unevaluatedItems as *Schema with ref", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "array",
			UnevaluatedItems: &parser.Schema{
				Ref: "#/definitions/ArrayItem",
			},
		}

		rewriteSchemaRefsOAS2ToOAS3(schema)

		unevalItems := schema.UnevaluatedItems.(*parser.Schema)
		assert.Equal(t, "#/components/schemas/ArrayItem", unevalItems.Ref,
			"UnevaluatedItems ref should be rewritten")
	})

	t.Run("unevaluatedItems as bool", func(t *testing.T) {
		schema := &parser.Schema{
			Type:             "array",
			UnevaluatedItems: true, // bool, not *Schema
		}

		// Should not panic
		rewriteSchemaRefsOAS2ToOAS3(schema)
		assert.Equal(t, true, schema.UnevaluatedItems)
	})

	t.Run("unevaluatedItems as nested *Schema with ref", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "array",
			UnevaluatedItems: &parser.Schema{
				AllOf: []*parser.Schema{
					{Ref: "#/definitions/ItemType"},
				},
			},
		}

		rewriteSchemaRefsOAS2ToOAS3(schema)

		unevalItems := schema.UnevaluatedItems.(*parser.Schema)
		assert.Equal(t, "#/components/schemas/ItemType", unevalItems.AllOf[0].Ref,
			"Nested ref in UnevaluatedItems should be rewritten")
	})
}

// TestWalkSchemaRefs_ContentSchema tests that refs in contentSchema are traversed
func TestWalkSchemaRefs_ContentSchema(t *testing.T) {
	t.Run("contentSchema with ref", func(t *testing.T) {
		schema := &parser.Schema{
			Type:             "string",
			ContentEncoding:  "base64",
			ContentMediaType: "application/json",
			ContentSchema: &parser.Schema{
				Ref: "#/definitions/EncodedPayload",
			},
		}

		rewriteSchemaRefsOAS2ToOAS3(schema)

		assert.Equal(t, "#/components/schemas/EncodedPayload", schema.ContentSchema.Ref,
			"ContentSchema ref should be rewritten")
	})

	t.Run("contentSchema with nested refs", func(t *testing.T) {
		schema := &parser.Schema{
			Type:             "string",
			ContentMediaType: "application/json",
			ContentSchema: &parser.Schema{
				Properties: map[string]*parser.Schema{
					"data": {Ref: "#/definitions/DataPayload"},
				},
			},
		}

		rewriteSchemaRefsOAS2ToOAS3(schema)

		assert.Equal(t, "#/components/schemas/DataPayload", schema.ContentSchema.Properties["data"].Ref,
			"Nested ref in ContentSchema should be rewritten")
	})

	t.Run("nil contentSchema", func(t *testing.T) {
		schema := &parser.Schema{
			Type:            "string",
			ContentEncoding: "base64",
			ContentSchema:   nil,
		}

		// Should not panic
		rewriteSchemaRefsOAS2ToOAS3(schema)
	})
}

// TestConvertOAS3ToOAS2_AdditionalOperations tests that additional operations (custom HTTP methods)
// in OAS 3.2+ are properly reported as unsupported when converting to OAS 2.0
func TestConvertOAS3ToOAS2_AdditionalOperations(t *testing.T) {
	t.Run("path with additionalOperations generates critical issue", func(t *testing.T) {
		doc := &parser.OAS3Document{
			OpenAPI: "3.2.0",
			Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
			Paths: map[string]*parser.PathItem{
				"/custom": {
					AdditionalOperations: map[string]*parser.Operation{
						"CUSTOM": {
							OperationID: "customOperation",
							Summary:     "Custom HTTP method",
						},
					},
				},
			},
		}

		parseResult := parser.ParseResult{
			Document:   doc,
			Version:    "3.2.0",
			OASVersion: parser.OASVersion320,
			Data:       make(map[string]any),
			SourcePath: "test.yaml",
		}

		c := New()
		result, err := c.ConvertParsed(parseResult, "2.0")

		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have an issue about additionalOperations
		var found bool
		for _, issue := range result.Issues {
			if issue.Severity == SeverityCritical &&
				issue.Path == "paths./custom.additionalOperations.CUSTOM" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should have critical issue for additionalOperations")
	})

	t.Run("multiple custom methods all reported", func(t *testing.T) {
		doc := &parser.OAS3Document{
			OpenAPI: "3.2.0",
			Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
			Paths: map[string]*parser.PathItem{
				"/resource": {
					AdditionalOperations: map[string]*parser.Operation{
						"LINK":   {OperationID: "linkResource"},
						"UNLINK": {OperationID: "unlinkResource"},
					},
				},
			},
		}

		parseResult := parser.ParseResult{
			Document:   doc,
			Version:    "3.2.0",
			OASVersion: parser.OASVersion320,
			Data:       make(map[string]any),
			SourcePath: "test.yaml",
		}

		c := New()
		result, err := c.ConvertParsed(parseResult, "2.0")

		require.NoError(t, err)
		require.NotNil(t, result)

		// Count additionalOperations issues
		var count int
		for _, issue := range result.Issues {
			if issue.Severity == SeverityCritical {
				count++
			}
		}
		assert.GreaterOrEqual(t, count, 2, "Should have issues for each custom method")
	})
}

// TestWalkSchemaRefs_OAS3ToOAS2_JSONSchema2020_12 tests OAS 3.x to OAS 2.0 conversion
// handles JSON Schema 2020-12 fields correctly
func TestWalkSchemaRefs_OAS3ToOAS2_JSONSchema2020_12(t *testing.T) {
	t.Run("all JSON Schema 2020-12 refs rewritten", func(t *testing.T) {
		schema := &parser.Schema{
			Type:                  "object",
			UnevaluatedProperties: &parser.Schema{Ref: "#/components/schemas/UnevalProps"},
			UnevaluatedItems:      &parser.Schema{Ref: "#/components/schemas/UnevalItems"},
			ContentSchema:         &parser.Schema{Ref: "#/components/schemas/Content"},
			PrefixItems:           []*parser.Schema{{Ref: "#/components/schemas/Prefix"}},
			Contains:              &parser.Schema{Ref: "#/components/schemas/Contains"},
			PropertyNames:         &parser.Schema{Ref: "#/components/schemas/PropNames"},
			DependentSchemas: map[string]*parser.Schema{
				"foo": {Ref: "#/components/schemas/Dependent"},
			},
			If:   &parser.Schema{Ref: "#/components/schemas/If"},
			Then: &parser.Schema{Ref: "#/components/schemas/Then"},
			Else: &parser.Schema{Ref: "#/components/schemas/Else"},
			Defs: map[string]*parser.Schema{
				"local": {Ref: "#/components/schemas/Local"},
			},
		}

		rewriteSchemaRefsOAS3ToOAS2(schema)

		// Verify all refs were rewritten from OAS 3.x to OAS 2.0 format
		assert.Equal(t, "#/definitions/UnevalProps", schema.UnevaluatedProperties.(*parser.Schema).Ref)
		assert.Equal(t, "#/definitions/UnevalItems", schema.UnevaluatedItems.(*parser.Schema).Ref)
		assert.Equal(t, "#/definitions/Content", schema.ContentSchema.Ref)
		assert.Equal(t, "#/definitions/Prefix", schema.PrefixItems[0].Ref)
		assert.Equal(t, "#/definitions/Contains", schema.Contains.Ref)
		assert.Equal(t, "#/definitions/PropNames", schema.PropertyNames.Ref)
		assert.Equal(t, "#/definitions/Dependent", schema.DependentSchemas["foo"].Ref)
		assert.Equal(t, "#/definitions/If", schema.If.Ref)
		assert.Equal(t, "#/definitions/Then", schema.Then.Ref)
		assert.Equal(t, "#/definitions/Else", schema.Else.Ref)
		assert.Equal(t, "#/definitions/Local", schema.Defs["local"].Ref)
	})
}

// TestWalkSchemaRefs_AllInterfaceTypedFields tests that all interface{}-typed fields
// are properly handled with type assertions
func TestWalkSchemaRefs_AllInterfaceTypedFields(t *testing.T) {
	t.Run("map[string]any type for AdditionalProperties", func(t *testing.T) {
		// In some cases, interface{} fields might be map[string]any instead of *Schema
		schema := &parser.Schema{
			Type: "object",
			AdditionalProperties: map[string]any{
				"$ref": "#/definitions/Unknown",
			},
		}

		// Should not panic - map[string]any is not *Schema, so it won't be traversed
		rewriteSchemaRefsOAS2ToOAS3(schema)

		// The map value won't be rewritten since it's not a *Schema
		addProps, ok := schema.AdditionalProperties.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "#/definitions/Unknown", addProps["$ref"], "map[string]any refs are not rewritten")
	})

	t.Run("map[string]any type for Items", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "array",
			Items: map[string]any{
				"$ref": "#/definitions/ItemType",
			},
		}

		// Should not panic
		rewriteSchemaRefsOAS2ToOAS3(schema)

		items, ok := schema.Items.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "#/definitions/ItemType", items["$ref"], "map[string]any refs are not rewritten")
	})

	t.Run("map[string]any type for UnevaluatedProperties", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "object",
			UnevaluatedProperties: map[string]any{
				"$ref": "#/definitions/Unevaluated",
			},
		}

		// Should not panic
		rewriteSchemaRefsOAS2ToOAS3(schema)

		unevalProps, ok := schema.UnevaluatedProperties.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "#/definitions/Unevaluated", unevalProps["$ref"], "map[string]any refs are not rewritten")
	})

	t.Run("map[string]any type for UnevaluatedItems", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "array",
			UnevaluatedItems: map[string]any{
				"$ref": "#/definitions/UnevalItem",
			},
		}

		// Should not panic
		rewriteSchemaRefsOAS2ToOAS3(schema)

		unevalItems, ok := schema.UnevaluatedItems.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "#/definitions/UnevalItem", unevalItems["$ref"], "map[string]any refs are not rewritten")
	})

	t.Run("map[string]any type for AdditionalItems", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "array",
			AdditionalItems: map[string]any{
				"$ref": "#/definitions/AddItem",
			},
		}

		// Should not panic
		rewriteSchemaRefsOAS2ToOAS3(schema)

		addItems, ok := schema.AdditionalItems.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "#/definitions/AddItem", addItems["$ref"], "map[string]any refs are not rewritten")
	})
}
