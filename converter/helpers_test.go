package converter

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertOAS3ParameterToOAS2 tests the convertOAS3ParameterToOAS2 method.
func TestConvertOAS3ParameterToOAS2(t *testing.T) {
	tests := []struct {
		name      string
		param     *parser.Parameter
		expectNil bool
	}{
		{
			name: "simple query parameter",
			param: &parser.Parameter{
				Name:   "limit",
				In:     "query",
				Schema: &parser.Schema{Type: "integer"},
			},
			expectNil: false,
		},
		{
			name: "path parameter",
			param: &parser.Parameter{
				Name:     "id",
				In:       "path",
				Required: true,
				Schema:   &parser.Schema{Type: "string"},
			},
			expectNil: false,
		},
		{
			name: "header parameter",
			param: &parser.Parameter{
				Name:   "X-API-Key",
				In:     "header",
				Schema: &parser.Schema{Type: "string"},
			},
			expectNil: false,
		},
		{
			name: "cookie parameter (OAS 3.x only)",
			param: &parser.Parameter{
				Name:   "session",
				In:     "cookie",
				Schema: &parser.Schema{Type: "string"},
			},
			expectNil: true, // Should be nil - not supported in OAS 2.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := &ConversionResult{Issues: []ConversionIssue{}}
			converted := c.convertOAS3ParameterToOAS2(tt.param, result, "test.path")

			if tt.expectNil {
				assert.Nil(t, converted, "convertOAS3ParameterToOAS2 should return nil")
			} else {
				require.NotNil(t, converted, "convertOAS3ParameterToOAS2 should return non-nil")
				assert.Equal(t, tt.param.Name, converted.Name, "Name should be preserved")
			}
		})
	}
}

// TestRewriteParameterRefsOAS3ToOAS2 tests the rewriteParameterRefsOAS3ToOAS2 function.
func TestRewriteParameterRefsOAS3ToOAS2(t *testing.T) {
	tests := []struct {
		name     string
		param    *parser.Parameter
		hasRef   bool
		expected string
	}{
		{
			name: "parameter with OAS 3.x reference",
			param: &parser.Parameter{
				Ref: "#/components/parameters/UserId",
			},
			hasRef:   true,
			expected: "#/parameters/UserId",
		},
		{
			name: "parameter without reference",
			param: &parser.Parameter{
				Name: "limit",
				In:   "query",
			},
			hasRef: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewriteParameterRefsOAS3ToOAS2(tt.param)

			if tt.hasRef {
				assert.Equal(t, tt.expected, tt.param.Ref, "Reference should be rewritten to OAS 2.0 format")
			}
		})
	}
}

// TestRewriteRequestBodyRefsOAS2ToOAS3 tests the rewriteRequestBodyRefsOAS2ToOAS3 function.
func TestRewriteRequestBodyRefsOAS2ToOAS3(t *testing.T) {
	tests := []struct {
		name        string
		requestBody *parser.RequestBody
		hasRefs     bool
		expected    string
	}{
		{
			name: "request body with schema reference",
			requestBody: &parser.RequestBody{
				Content: map[string]*parser.MediaType{
					"application/json": {
						Schema: &parser.Schema{
							Ref: "#/definitions/User",
						},
					},
				},
			},
			hasRefs:  true,
			expected: "#/components/schemas/User",
		},
		{
			name: "request body without reference",
			requestBody: &parser.RequestBody{
				Content: map[string]*parser.MediaType{
					"application/json": {
						Schema: &parser.Schema{
							Type: "object",
						},
					},
				},
			},
			hasRefs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewriteRequestBodyRefsOAS2ToOAS3(tt.requestBody)

			if tt.hasRefs {
				for _, mediaType := range tt.requestBody.Content {
					if mediaType.Schema != nil && mediaType.Schema.Ref != "" {
						assert.Equal(t, tt.expected, mediaType.Schema.Ref, "Reference should be rewritten to OAS 3.x format")
					}
				}
			}
		})
	}
}

// TestWalkSchemaRefs tests the walkSchemaRefs function to ensure all schema locations are traversed.
func TestWalkSchemaRefs(t *testing.T) {
	t.Run("nil schema", func(t *testing.T) {
		// Should not panic
		rewriteSchemaRefsOAS2ToOAS3(nil)
	})

	t.Run("empty ref not rewritten", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		rewriteSchemaRefsOAS2ToOAS3(schema)
		assert.Equal(t, "", schema.Ref, "Empty ref should remain empty")
	})

	t.Run("all schema locations with refs", func(t *testing.T) {
		// Build a schema with refs in all possible locations
		schema := &parser.Schema{
			Ref: "#/definitions/Root",
			Properties: map[string]*parser.Schema{
				"prop": {Ref: "#/definitions/Prop"},
			},
			PatternProperties: map[string]*parser.Schema{
				"^x-": {Ref: "#/definitions/Pattern"},
			},
			AdditionalProperties: &parser.Schema{Ref: "#/definitions/AddProps"},
			Items:                &parser.Schema{Ref: "#/definitions/Items"},
			AllOf:                []*parser.Schema{{Ref: "#/definitions/AllOf"}},
			AnyOf:                []*parser.Schema{{Ref: "#/definitions/AnyOf"}},
			OneOf:                []*parser.Schema{{Ref: "#/definitions/OneOf"}},
			Not:                  &parser.Schema{Ref: "#/definitions/Not"},
			AdditionalItems:      &parser.Schema{Ref: "#/definitions/AddItems"},
			PrefixItems:          []*parser.Schema{{Ref: "#/definitions/Prefix"}},
			Contains:             &parser.Schema{Ref: "#/definitions/Contains"},
			PropertyNames:        &parser.Schema{Ref: "#/definitions/PropNames"},
			DependentSchemas: map[string]*parser.Schema{
				"dep": {Ref: "#/definitions/Dep"},
			},
			If:   &parser.Schema{Ref: "#/definitions/If"},
			Then: &parser.Schema{Ref: "#/definitions/Then"},
			Else: &parser.Schema{Ref: "#/definitions/Else"},
			Defs: map[string]*parser.Schema{
				"LocalDef": {Ref: "#/definitions/LocalDef"},
			},
		}

		// Rewrite OAS 2.0 refs to OAS 3.x format
		rewriteSchemaRefsOAS2ToOAS3(schema)

		// Verify all refs were rewritten from #/definitions/ to #/components/schemas/
		assert.Equal(t, "#/components/schemas/Root", schema.Ref, "Root ref")
		assert.Equal(t, "#/components/schemas/Prop", schema.Properties["prop"].Ref, "Properties ref")
		assert.Equal(t, "#/components/schemas/Pattern", schema.PatternProperties["^x-"].Ref, "PatternProperties ref")
		assert.Equal(t, "#/components/schemas/AddProps", schema.AdditionalProperties.(*parser.Schema).Ref, "AdditionalProperties ref")
		assert.Equal(t, "#/components/schemas/Items", schema.Items.(*parser.Schema).Ref, "Items ref")
		assert.Equal(t, "#/components/schemas/AllOf", schema.AllOf[0].Ref, "AllOf ref")
		assert.Equal(t, "#/components/schemas/AnyOf", schema.AnyOf[0].Ref, "AnyOf ref")
		assert.Equal(t, "#/components/schemas/OneOf", schema.OneOf[0].Ref, "OneOf ref")
		assert.Equal(t, "#/components/schemas/Not", schema.Not.Ref, "Not ref")
		assert.Equal(t, "#/components/schemas/AddItems", schema.AdditionalItems.(*parser.Schema).Ref, "AdditionalItems ref")
		assert.Equal(t, "#/components/schemas/Prefix", schema.PrefixItems[0].Ref, "PrefixItems ref")
		assert.Equal(t, "#/components/schemas/Contains", schema.Contains.Ref, "Contains ref")
		assert.Equal(t, "#/components/schemas/PropNames", schema.PropertyNames.Ref, "PropertyNames ref")
		assert.Equal(t, "#/components/schemas/Dep", schema.DependentSchemas["dep"].Ref, "DependentSchemas ref")
		assert.Equal(t, "#/components/schemas/If", schema.If.Ref, "If ref")
		assert.Equal(t, "#/components/schemas/Then", schema.Then.Ref, "Then ref")
		assert.Equal(t, "#/components/schemas/Else", schema.Else.Ref, "Else ref")
		assert.Equal(t, "#/components/schemas/LocalDef", schema.Defs["LocalDef"].Ref, "Defs ref")
	})

	t.Run("OAS 3.x to OAS 2.0 rewrite", func(t *testing.T) {
		schema := &parser.Schema{
			Ref: "#/components/schemas/User",
			Properties: map[string]*parser.Schema{
				"address": {Ref: "#/components/schemas/Address"},
			},
		}

		rewriteSchemaRefsOAS3ToOAS2(schema)

		assert.Equal(t, "#/definitions/User", schema.Ref)
		assert.Equal(t, "#/definitions/Address", schema.Properties["address"].Ref)
	})

	t.Run("bool-typed AdditionalProperties skipped", func(t *testing.T) {
		// In OAS 3.1+, AdditionalProperties can be a bool
		schema := &parser.Schema{
			Ref:                  "#/definitions/Test",
			AdditionalProperties: true, // bool, not *Schema
		}

		// Should not panic and should still rewrite the root ref
		rewriteSchemaRefsOAS2ToOAS3(schema)
		assert.Equal(t, "#/components/schemas/Test", schema.Ref)
	})

	t.Run("bool-typed Items skipped", func(t *testing.T) {
		// Items can also be bool in some contexts
		schema := &parser.Schema{
			Ref:   "#/definitions/Test",
			Items: false, // bool, not *Schema
		}

		// Should not panic and should still rewrite the root ref
		rewriteSchemaRefsOAS2ToOAS3(schema)
		assert.Equal(t, "#/components/schemas/Test", schema.Ref)
	})

	t.Run("deeply nested refs", func(t *testing.T) {
		schema := &parser.Schema{
			Properties: map[string]*parser.Schema{
				"level1": {
					Properties: map[string]*parser.Schema{
						"level2": {
							AllOf: []*parser.Schema{
								{Ref: "#/definitions/DeepRef"},
							},
						},
					},
				},
			},
		}

		rewriteSchemaRefsOAS2ToOAS3(schema)

		assert.Equal(t, "#/components/schemas/DeepRef",
			schema.Properties["level1"].Properties["level2"].AllOf[0].Ref,
			"Deeply nested ref should be rewritten")
	})
}
