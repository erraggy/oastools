package converter

import (
	"strings"
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

// TestConvertOAS3SchemaToOAS2_AdditionalFeatures tests detection of OAS 3.x schema features
// that have no OAS 2.0 equivalent during downgrade conversion.
func TestConvertOAS3SchemaToOAS2_AdditionalFeatures(t *testing.T) {
	t.Run("writeOnly detected", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type:      "string",
			WriteOnly: true,
		}

		c.convertOAS3SchemaToOAS2(schema, result, "test.writeOnly")

		require.NotEmpty(t, result.Issues, "Expected issue for writeOnly")
		assertHasIssueContaining(t, result.Issues, "writeOnly")
	})

	t.Run("deprecated on schema detected", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type:       "object",
			Deprecated: true,
		}

		c.convertOAS3SchemaToOAS2(schema, result, "test.deprecated")

		require.NotEmpty(t, result.Issues, "Expected issue for deprecated")
		assertHasIssueContaining(t, result.Issues, "deprecated")
	})

	t.Run("if/then/else detected", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type: "object",
			If:   &parser.Schema{Properties: map[string]*parser.Schema{"country": {Const: "US"}}},
			Then: &parser.Schema{Properties: map[string]*parser.Schema{"state": {Type: "string"}}},
			Else: &parser.Schema{Properties: map[string]*parser.Schema{"province": {Type: "string"}}},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "test.conditional")

		// Should have 3 issues: one each for if, then, else
		ifCount := countIssuesContaining(result.Issues, "if")
		thenCount := countIssuesContaining(result.Issues, "then")
		elseCount := countIssuesContaining(result.Issues, "else")
		assert.Equal(t, 1, ifCount, "Expected exactly 1 issue for 'if'")
		assert.Equal(t, 1, thenCount, "Expected exactly 1 issue for 'then'")
		assert.Equal(t, 1, elseCount, "Expected exactly 1 issue for 'else'")
	})

	t.Run("prefixItems detected", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type: "array",
			PrefixItems: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "test.prefixItems")

		require.NotEmpty(t, result.Issues, "Expected issue for prefixItems")
		assertHasIssueContaining(t, result.Issues, "prefixItems")
	})

	t.Run("contains detected", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type:     "array",
			Contains: &parser.Schema{Type: "string"},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "test.contains")

		require.NotEmpty(t, result.Issues, "Expected issue for contains")
		assertHasIssueContaining(t, result.Issues, "contains")
	})

	t.Run("propertyNames detected", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type:          "object",
			PropertyNames: &parser.Schema{Pattern: "^[a-z]+$"},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "test.propertyNames")

		require.NotEmpty(t, result.Issues, "Expected issue for propertyNames")
		assertHasIssueContaining(t, result.Issues, "propertyNames")
	})

	t.Run("no issues for plain schema", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"name": {Type: "string"},
				"age":  {Type: "integer"},
			},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "test.plain")

		assert.Empty(t, result.Issues, "Expected no issues for plain OAS 2.0-compatible schema")
	})

	t.Run("multiple features detected simultaneously", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type:          "object",
			WriteOnly:     true,
			Deprecated:    true,
			PropertyNames: &parser.Schema{Pattern: "^[a-z]"},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "test.multi")

		assert.GreaterOrEqual(t, len(result.Issues), 3,
			"Expected at least 3 issues for writeOnly + deprecated + propertyNames")
		assertHasIssueContaining(t, result.Issues, "writeOnly")
		assertHasIssueContaining(t, result.Issues, "deprecated")
		assertHasIssueContaining(t, result.Issues, "propertyNames")
	})

	t.Run("if without then/else", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type: "object",
			If:   &parser.Schema{Properties: map[string]*parser.Schema{"x": {Type: "string"}}},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "test.ifOnly")

		assert.Equal(t, 1, countIssuesContaining(result.Issues, "if"),
			"Expected exactly 1 issue for 'if' alone")
		assert.Equal(t, 0, countIssuesContaining(result.Issues, "then"),
			"Expected no issue for 'then' when absent")
		assert.Equal(t, 0, countIssuesContaining(result.Issues, "else"),
			"Expected no issue for 'else' when absent")
	})

	t.Run("empty prefixItems no issue", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type:        "array",
			PrefixItems: []*parser.Schema{},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "test.emptyPrefix")

		assert.Empty(t, result.Issues, "Expected no issues for empty prefixItems slice")
	})
}

// TestConvertOAS3SchemaToOAS2_NestedFeatures tests that OAS 3.x feature detection
// works recursively through nested schemas (Properties, Items, AllOf, AdditionalProperties, etc.).
func TestConvertOAS3SchemaToOAS2_NestedFeatures(t *testing.T) {
	t.Run("writeOnly in nested property", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"password": {Type: "string", WriteOnly: true},
			},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "components.schemas.User")

		require.NotEmpty(t, result.Issues, "Expected issue for writeOnly in nested property")
		assertHasIssueContaining(t, result.Issues, "writeOnly")
		// Verify the issue path references the nested location
		found := false
		for _, issue := range result.Issues {
			if strings.Contains(issue.Message, "writeOnly") &&
				strings.Contains(issue.Path, "properties.password") {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected issue path to contain 'properties.password'")
	})

	t.Run("deprecated in array items", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type:  "array",
			Items: &parser.Schema{Type: "object", Deprecated: true},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "components.schemas.List")

		require.NotEmpty(t, result.Issues, "Expected issue for deprecated in items")
		assertHasIssueContaining(t, result.Issues, "deprecated")
		found := false
		for _, issue := range result.Issues {
			if strings.Contains(issue.Message, "deprecated") &&
				strings.Contains(issue.Path, ".items") {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected issue path to contain '.items'")
	})

	t.Run("nested feature in allOf member", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type: "object",
			AllOf: []*parser.Schema{
				{Type: "object", WriteOnly: true},
			},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "components.schemas.Combined")

		require.NotEmpty(t, result.Issues, "Expected issue for writeOnly in allOf member")
		assertHasIssueContaining(t, result.Issues, "writeOnly")
		found := false
		for _, issue := range result.Issues {
			if strings.Contains(issue.Message, "writeOnly") &&
				strings.Contains(issue.Path, "allOf[0]") {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected issue path to contain 'allOf[0]'")
	})

	t.Run("deeply nested feature", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"address": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"geo": {Type: "object", WriteOnly: true},
					},
				},
			},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "components.schemas.User")

		require.NotEmpty(t, result.Issues, "Expected issue for deeply nested writeOnly")
		assertHasIssueContaining(t, result.Issues, "writeOnly")
		found := false
		for _, issue := range result.Issues {
			if strings.Contains(issue.Message, "writeOnly") &&
				strings.Contains(issue.Path, "properties.address") &&
				strings.Contains(issue.Path, "properties.geo") {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected issue path to contain both 'properties.address' and 'properties.geo'")
	})

	t.Run("multiple nested features across locations", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"secret": {Type: "string", WriteOnly: true},
			},
			Items: &parser.Schema{Type: "object", Deprecated: true},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "components.schemas.Mixed")

		writeOnlyCount := countIssuesContaining(result.Issues, "writeOnly")
		deprecatedCount := countIssuesContaining(result.Issues, "deprecated")
		assert.Equal(t, 1, writeOnlyCount, "Expected exactly 1 writeOnly issue")
		assert.Equal(t, 1, deprecatedCount, "Expected exactly 1 deprecated issue")
	})

	t.Run("plain nested schema no issues", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"name": {Type: "string"},
				"tags": {
					Type:  "array",
					Items: &parser.Schema{Type: "string"},
				},
			},
			AllOf: []*parser.Schema{
				{Type: "object", Properties: map[string]*parser.Schema{
					"id": {Type: "integer"},
				}},
			},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "components.schemas.Plain")

		assert.Empty(t, result.Issues, "Expected no issues for plain nested schemas without OAS 3.x features")
	})

	t.Run("feature in additionalProperties", func(t *testing.T) {
		c := New()
		result := &ConversionResult{Issues: []ConversionIssue{}}
		schema := &parser.Schema{
			Type:                 "object",
			AdditionalProperties: &parser.Schema{Type: "string", Nullable: true},
		}

		c.convertOAS3SchemaToOAS2(schema, result, "components.schemas.Map")

		require.NotEmpty(t, result.Issues, "Expected issue for nullable in additionalProperties")
		assertHasIssueContaining(t, result.Issues, "nullable")
		found := false
		for _, issue := range result.Issues {
			if strings.Contains(issue.Message, "nullable") &&
				strings.Contains(issue.Path, "additionalProperties") {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected issue path to contain 'additionalProperties'")
	})
}

// assertHasIssueContaining asserts that at least one issue contains the given substring.
func assertHasIssueContaining(t *testing.T, issues []ConversionIssue, substring string) {
	t.Helper()
	for _, issue := range issues {
		if strings.Contains(issue.Message, substring) {
			return
		}
	}
	t.Errorf("Expected at least one issue containing %q, but none found in %d issues", substring, len(issues))
}

// TestConvertOAS2ParameterToOAS3_ArrayItems verifies that Items and validation
// keywords on OAS 2.0 array parameters are transferred to the OAS 3.0 schema
// (issue #357).
func TestConvertOAS2ParameterToOAS3_ArrayItems(t *testing.T) {
	c := newConverter()
	result := &ConversionResult{}

	strType := "string"
	param := &parser.Parameter{
		Name:  "chain_id",
		In:    "query",
		Type:  "array",
		Items: &parser.Items{Type: strType},
	}

	converted := c.convertOAS2ParameterToOAS3(param, result, "test")
	require.NotNil(t, converted)
	require.NotNil(t, converted.Schema)
	assert.Equal(t, "array", converted.Schema.Type)
	require.NotNil(t, converted.Schema.Items, "Items should be transferred from OAS 2.0 parameter")

	itemsSchema, ok := converted.Schema.Items.(*parser.Schema)
	require.True(t, ok, "Items should be *Schema")
	assert.Equal(t, strType, itemsSchema.Type)
}

func TestConvertOAS2ParameterToOAS3_ValidationKeywords(t *testing.T) {
	c := newConverter()
	result := &ConversionResult{}

	min := float64(1)
	max := float64(100)
	minLen := 2
	maxLen := 50
	param := &parser.Parameter{
		Name:      "q",
		In:        "query",
		Type:      "string",
		Minimum:   &min,
		Maximum:   &max,
		MinLength: &minLen,
		MaxLength: &maxLen,
		Pattern:   "^[a-z]+$",
		Enum:      []any{"a", "b"},
	}

	converted := c.convertOAS2ParameterToOAS3(param, result, "test")
	require.NotNil(t, converted.Schema)
	assert.Equal(t, &min, converted.Schema.Minimum)
	assert.Equal(t, &max, converted.Schema.Maximum)
	assert.Equal(t, &minLen, converted.Schema.MinLength)
	assert.Equal(t, &maxLen, converted.Schema.MaxLength)
	assert.Equal(t, "^[a-z]+$", converted.Schema.Pattern)
	assert.Equal(t, []any{"a", "b"}, converted.Schema.Enum)
}

// TestConvertOAS2ParameterToOAS3_ExclusiveKeywords verifies that ExclusiveMaximum
// and ExclusiveMinimum bool flags are transferred to the OAS 3.x schema.
func TestConvertOAS2ParameterToOAS3_ExclusiveKeywords(t *testing.T) {
	c := newConverter()
	result := &ConversionResult{}

	min := float64(0)
	max := float64(10)
	param := &parser.Parameter{
		Name:             "count",
		In:               "query",
		Type:             "integer",
		Minimum:          &min,
		Maximum:          &max,
		ExclusiveMinimum: true,
		ExclusiveMaximum: true,
	}

	converted := c.convertOAS2ParameterToOAS3(param, result, "test")
	require.NotNil(t, converted.Schema)
	assert.Equal(t, true, converted.Schema.ExclusiveMinimum, "ExclusiveMinimum should be transferred")
	assert.Equal(t, true, converted.Schema.ExclusiveMaximum, "ExclusiveMaximum should be transferred")
}

// TestConvertOAS2ParameterToOAS3_ItemsCollectionFormat verifies that a
// non-csv collectionFormat on items generates a conversion warning.
func TestConvertOAS2ParameterToOAS3_ItemsCollectionFormat(t *testing.T) {
	c := newConverter()
	result := &ConversionResult{}

	param := &parser.Parameter{
		Name: "tags",
		In:   "query",
		Type: "array",
		Items: &parser.Items{
			Type:             "string",
			CollectionFormat: "pipes",
		},
	}

	c.convertOAS2ParameterToOAS3(param, result, "test")
	assert.Greater(t, countIssuesContaining(result.Issues, "collectionFormat"), 0,
		"should warn about non-csv collectionFormat on items")
}

// TestConvertOAS2ParameterToOAS3_ArrayWithoutItems verifies that an array
// parameter with nil Items is handled gracefully.
func TestConvertOAS2ParameterToOAS3_ArrayWithoutItems(t *testing.T) {
	c := newConverter()
	result := &ConversionResult{}

	param := &parser.Parameter{
		Name: "tags",
		In:   "query",
		Type: "array",
		// Items intentionally nil
	}

	converted := c.convertOAS2ParameterToOAS3(param, result, "test")
	require.NotNil(t, converted)
	require.NotNil(t, converted.Schema)
	assert.Equal(t, "array", converted.Schema.Type, "Schema.Type should be array")
	assert.Nil(t, converted.Schema.Items, "Schema.Items should be nil when source has no Items")
}

// TestConvertOAS2ParameterToOAS3_ExclusiveKeywordsOAS31 verifies that for OAS 3.1+
// targets, boolean ExclusiveMaximum/ExclusiveMinimum are converted to numeric form.
func TestConvertOAS2ParameterToOAS3_ExclusiveKeywordsOAS31(t *testing.T) {
	t.Run("OAS 3.1 target converts boolean to numeric", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion310}

		min := float64(0)
		max := float64(100)
		param := &parser.Parameter{
			Name:             "count",
			In:               "query",
			Type:             "integer",
			Minimum:          &min,
			Maximum:          &max,
			ExclusiveMinimum: true,
			ExclusiveMaximum: true,
		}

		converted := c.convertOAS2ParameterToOAS3(param, result, "test")
		require.NotNil(t, converted.Schema)
		assert.Equal(t, float64(100), converted.Schema.ExclusiveMaximum, "ExclusiveMaximum should be numeric 100")
		assert.Nil(t, converted.Schema.Maximum, "Maximum should be nil after conversion")
		assert.Equal(t, float64(0), converted.Schema.ExclusiveMinimum, "ExclusiveMinimum should be numeric 0")
		assert.Nil(t, converted.Schema.Minimum, "Minimum should be nil after conversion")
	})

	t.Run("OAS 3.0 target preserves boolean form", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion303}

		min := float64(0)
		max := float64(100)
		param := &parser.Parameter{
			Name:             "count",
			In:               "query",
			Type:             "integer",
			Minimum:          &min,
			Maximum:          &max,
			ExclusiveMinimum: true,
			ExclusiveMaximum: true,
		}

		converted := c.convertOAS2ParameterToOAS3(param, result, "test")
		require.NotNil(t, converted.Schema)
		assert.Equal(t, true, converted.Schema.ExclusiveMaximum, "ExclusiveMaximum should be boolean for OAS 3.0")
		assert.Equal(t, &max, converted.Schema.Maximum, "Maximum should be preserved for OAS 3.0")
		assert.Equal(t, true, converted.Schema.ExclusiveMinimum, "ExclusiveMinimum should be boolean for OAS 3.0")
		assert.Equal(t, &min, converted.Schema.Minimum, "Minimum should be preserved for OAS 3.0")
	})

	t.Run("OAS 3.1 target with no maximum emits warning", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion310}

		param := &parser.Parameter{
			Name:             "count",
			In:               "query",
			Type:             "integer",
			ExclusiveMaximum: true,
			// Maximum intentionally nil — malformed OAS 2.0
		}

		converted := c.convertOAS2ParameterToOAS3(param, result, "test")
		require.NotNil(t, converted.Schema)
		assert.Nil(t, converted.Schema.ExclusiveMaximum, "ExclusiveMaximum should be nil when no Maximum present")
		assert.Nil(t, converted.Schema.Maximum, "Maximum should remain nil")
		assert.Greater(t, countIssuesContaining(result.Issues, "exclusiveMaximum: true"), 0,
			"should emit warning about dropped exclusiveMaximum constraint")
	})
}

// TestConvertOAS2ItemsToSchema_ExclusiveOAS31 verifies that Items conversion
// handles exclusiveMaximum/exclusiveMinimum correctly for different target versions.
func TestConvertOAS2ItemsToSchema_ExclusiveOAS31(t *testing.T) {
	t.Run("OAS 3.1 target converts boolean to numeric in items", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion310}
		max := float64(50)
		min := float64(5)
		items := &parser.Items{
			Type:             "integer",
			Maximum:          &max,
			Minimum:          &min,
			ExclusiveMaximum: true,
			ExclusiveMinimum: true,
		}

		s := convertOAS2ItemsToSchema(c, items, result, "test.items")
		require.NotNil(t, s)
		assert.Equal(t, float64(50), s.ExclusiveMaximum, "ExclusiveMaximum should be numeric 50")
		assert.Nil(t, s.Maximum, "Maximum should be nil after conversion")
		assert.Equal(t, float64(5), s.ExclusiveMinimum, "ExclusiveMinimum should be numeric 5")
		assert.Nil(t, s.Minimum, "Minimum should be nil after conversion")
	})

	t.Run("OAS 3.0 target preserves boolean in items", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion303}
		max := float64(50)
		min := float64(5)
		items := &parser.Items{
			Type:             "integer",
			Maximum:          &max,
			Minimum:          &min,
			ExclusiveMaximum: true,
			ExclusiveMinimum: true,
		}

		s := convertOAS2ItemsToSchema(c, items, result, "test.items")
		require.NotNil(t, s)
		assert.Equal(t, true, s.ExclusiveMaximum, "ExclusiveMaximum should be boolean for OAS 3.0")
		assert.Equal(t, &max, s.Maximum, "Maximum should be preserved for OAS 3.0")
		assert.Equal(t, true, s.ExclusiveMinimum, "ExclusiveMinimum should be boolean for OAS 3.0")
		assert.Equal(t, &min, s.Minimum, "Minimum should be preserved for OAS 3.0")
	})

	t.Run("nested items inherit target version", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion310}
		innerMax := float64(25)
		items := &parser.Items{
			Type: "array",
			Items: &parser.Items{
				Type:             "integer",
				Maximum:          &innerMax,
				ExclusiveMaximum: true,
			},
		}

		s := convertOAS2ItemsToSchema(c, items, result, "test.items")
		require.NotNil(t, s)
		inner, ok := s.Items.(*parser.Schema)
		require.True(t, ok, "nested Items should be *parser.Schema")
		assert.Equal(t, float64(25), inner.ExclusiveMaximum, "nested ExclusiveMaximum should be numeric")
		assert.Nil(t, inner.Maximum, "nested Maximum should be nil after conversion")
	})

	t.Run("OAS 3.1 items with no maximum emits warning", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion310}
		items := &parser.Items{
			Type:             "integer",
			ExclusiveMaximum: true,
			// Maximum intentionally nil
		}

		s := convertOAS2ItemsToSchema(c, items, result, "test.items")
		require.NotNil(t, s)
		assert.Nil(t, s.ExclusiveMaximum, "ExclusiveMaximum should remain nil")
		assert.Greater(t, countIssuesContaining(result.Issues, "exclusiveMaximum: true"), 0,
			"should emit warning about dropped exclusiveMaximum constraint")
	})
}

// TestConvertOAS2SchemaToOAS3_ExclusiveOAS31 verifies that convertOAS2SchemaToOAS3
// fixes boolean exclusive min/max in schemas when targeting OAS 3.1+.
func TestConvertOAS2SchemaToOAS3_ExclusiveOAS31(t *testing.T) {
	t.Run("OAS 3.1 target converts boolean exclusive in schema", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion310}
		max := float64(200)
		min := float64(10)
		schema := &parser.Schema{
			Type:             "integer",
			Maximum:          &max,
			Minimum:          &min,
			ExclusiveMaximum: true,
			ExclusiveMinimum: true,
		}

		converted := c.convertOAS2SchemaToOAS3(schema, parser.OASVersion310, result, "test")
		require.NotNil(t, converted)
		assert.Equal(t, float64(200), converted.ExclusiveMaximum, "ExclusiveMaximum should be numeric 200")
		assert.Nil(t, converted.Maximum, "Maximum should be nil after conversion")
		assert.Equal(t, float64(10), converted.ExclusiveMinimum, "ExclusiveMinimum should be numeric 10")
		assert.Nil(t, converted.Minimum, "Minimum should be nil after conversion")
	})

	t.Run("OAS 3.0 target preserves boolean in schema", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion303}
		max := float64(200)
		min := float64(10)
		schema := &parser.Schema{
			Type:             "integer",
			Maximum:          &max,
			Minimum:          &min,
			ExclusiveMaximum: true,
			ExclusiveMinimum: true,
		}

		converted := c.convertOAS2SchemaToOAS3(schema, parser.OASVersion303, result, "test")
		require.NotNil(t, converted)
		assert.Equal(t, true, converted.ExclusiveMaximum, "ExclusiveMaximum should be boolean for OAS 3.0")
		assert.NotNil(t, converted.Maximum, "Maximum should be preserved for OAS 3.0")
		assert.Equal(t, true, converted.ExclusiveMinimum, "ExclusiveMinimum should be boolean for OAS 3.0")
		assert.NotNil(t, converted.Minimum, "Minimum should be preserved for OAS 3.0")
	})

	t.Run("OAS 3.1 converts nested property exclusive", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion311}
		propMax := float64(99)
		schema := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"score": {
					Type:             "number",
					Maximum:          &propMax,
					ExclusiveMaximum: true,
				},
			},
		}

		converted := c.convertOAS2SchemaToOAS3(schema, parser.OASVersion311, result, "test")
		require.NotNil(t, converted)
		scoreProp, ok := converted.Properties["score"]
		require.True(t, ok, "score property should exist")
		assert.Equal(t, float64(99), scoreProp.ExclusiveMaximum, "nested ExclusiveMaximum should be numeric")
		assert.Nil(t, scoreProp.Maximum, "nested Maximum should be nil after conversion")
	})

	t.Run("OAS 3.1 boolean exclusive with no maximum emits warning", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion310}
		schema := &parser.Schema{
			Type:             "integer",
			ExclusiveMaximum: true,
			// Maximum intentionally nil
		}

		converted := c.convertOAS2SchemaToOAS3(schema, parser.OASVersion310, result, "test")
		require.NotNil(t, converted)
		assert.Nil(t, converted.ExclusiveMaximum, "ExclusiveMaximum should be nil when no Maximum present")
		assert.Greater(t, countIssuesContaining(result.Issues, "exclusiveMaximum: true"), 0,
			"should emit warning about dropped exclusiveMaximum constraint")
	})

	t.Run("nil schema returns nil", func(t *testing.T) {
		c := newConverter()
		assert.Nil(t, c.convertOAS2SchemaToOAS3(nil, parser.OASVersion310, nil, ""))
	})

	t.Run("does not mutate original schema", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion310}
		max := float64(100)
		original := &parser.Schema{
			Type:             "integer",
			Maximum:          &max,
			ExclusiveMaximum: true,
		}

		_ = c.convertOAS2SchemaToOAS3(original, parser.OASVersion310, result, "test")
		assert.Equal(t, true, original.ExclusiveMaximum, "original should not be mutated")
		assert.NotNil(t, original.Maximum, "original Maximum should not be mutated")
	})

	t.Run("false boolean exclusiveMaximum becomes nil in OAS 3.1", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion310}
		max := float64(100)
		schema := &parser.Schema{
			Type:             "integer",
			Maximum:          &max,
			ExclusiveMaximum: false, // boolean false
		}

		converted := c.convertOAS2SchemaToOAS3(schema, parser.OASVersion310, result, "test")
		require.NotNil(t, converted)
		assert.Nil(t, converted.ExclusiveMaximum, "false ExclusiveMaximum should become nil in OAS 3.1")
		assert.NotNil(t, converted.Maximum, "Maximum should be preserved when ExclusiveMaximum was false")
	})

	t.Run("allOf traversal converts exclusive in OAS 3.1", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion310}
		val := float64(42)
		schema := &parser.Schema{
			AllOf: []*parser.Schema{
				{
					Type:             "integer",
					Maximum:          &val,
					ExclusiveMaximum: true,
				},
			},
		}

		converted := c.convertOAS2SchemaToOAS3(schema, parser.OASVersion310, result, "test")
		require.NotNil(t, converted)
		require.Len(t, converted.AllOf, 1)
		assert.Equal(t, float64(42), converted.AllOf[0].ExclusiveMaximum, "allOf member ExclusiveMaximum should be numeric")
		assert.Nil(t, converted.AllOf[0].Maximum, "allOf member Maximum should be nil after conversion")
	})

	t.Run("nil result is safe for schema conversion", func(t *testing.T) {
		c := newConverter()
		schema := &parser.Schema{
			Type:             "integer",
			ExclusiveMaximum: true,
			// Maximum intentionally nil -- would emit warning but result is nil
		}

		converted := c.convertOAS2SchemaToOAS3(schema, parser.OASVersion310, nil, "")
		require.NotNil(t, converted)
		assert.Nil(t, converted.ExclusiveMaximum, "ExclusiveMaximum should be nil even with nil result")
	})
}

// TestConvertOAS2FormDataToRequestBody_ExclusiveOAS31 verifies that formData parameter
// conversion handles exclusiveMaximum/exclusiveMinimum correctly for different target versions.
func TestConvertOAS2FormDataToRequestBody_ExclusiveOAS31(t *testing.T) {
	t.Run("OAS 3.1 target converts boolean to numeric in formData", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion310}
		max := float64(100)
		min := float64(1)
		src := &parser.Operation{
			Parameters: []*parser.Parameter{
				{
					Name:             "score",
					In:               "formData",
					Type:             "integer",
					Maximum:          &max,
					Minimum:          &min,
					ExclusiveMaximum: true,
					ExclusiveMinimum: true,
				},
			},
		}

		rb := c.convertOAS2FormDataToRequestBody(src, &parser.OAS2Document{}, result)
		require.NotNil(t, rb)
		require.NotEmpty(t, rb.Content, "Content should have at least one media type")
		// Find the schema in the content
		for _, mt := range rb.Content {
			require.NotNil(t, mt.Schema)
			scoreProp, ok := mt.Schema.Properties["score"]
			require.True(t, ok, "score property should exist")
			assert.Equal(t, float64(100), scoreProp.ExclusiveMaximum, "ExclusiveMaximum should be numeric 100")
			assert.Nil(t, scoreProp.Maximum, "Maximum should be nil after conversion")
			assert.Equal(t, float64(1), scoreProp.ExclusiveMinimum, "ExclusiveMinimum should be numeric 1")
			assert.Nil(t, scoreProp.Minimum, "Minimum should be nil after conversion")
			break
		}
	})

	t.Run("OAS 3.0 target preserves boolean in formData", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion303}
		max := float64(100)
		min := float64(1)
		src := &parser.Operation{
			Parameters: []*parser.Parameter{
				{
					Name:             "score",
					In:               "formData",
					Type:             "integer",
					Maximum:          &max,
					Minimum:          &min,
					ExclusiveMaximum: true,
					ExclusiveMinimum: true,
				},
			},
		}

		rb := c.convertOAS2FormDataToRequestBody(src, &parser.OAS2Document{}, result)
		require.NotNil(t, rb)
		require.NotEmpty(t, rb.Content, "Content should have at least one media type")
		for _, mt := range rb.Content {
			require.NotNil(t, mt.Schema)
			scoreProp, ok := mt.Schema.Properties["score"]
			require.True(t, ok, "score property should exist")
			assert.Equal(t, true, scoreProp.ExclusiveMaximum, "ExclusiveMaximum should be boolean for OAS 3.0")
			assert.Equal(t, &max, scoreProp.Maximum, "Maximum should be preserved for OAS 3.0")
			assert.Equal(t, true, scoreProp.ExclusiveMinimum, "ExclusiveMinimum should be boolean for OAS 3.0")
			assert.Equal(t, &min, scoreProp.Minimum, "Minimum should be preserved for OAS 3.0")
			break
		}
	})

	t.Run("OAS 3.1 target with no maximum emits warning in formData", func(t *testing.T) {
		c := newConverter()
		result := &ConversionResult{TargetOASVersion: parser.OASVersion310}
		src := &parser.Operation{
			Parameters: []*parser.Parameter{
				{
					Name:             "score",
					In:               "formData",
					Type:             "integer",
					ExclusiveMaximum: true,
					// Maximum intentionally nil
				},
			},
		}

		rb := c.convertOAS2FormDataToRequestBody(src, &parser.OAS2Document{}, result)
		require.NotNil(t, rb)
		assert.Greater(t, countIssuesContaining(result.Issues, "exclusiveMaximum: true"), 0,
			"should emit warning about dropped exclusiveMaximum constraint in formData")
	})
}

// newConverter creates a Converter for unit testing helpers using the same
// initialization path as production code.
func newConverter() *Converter {
	return New()
}

// countIssuesContaining counts issues whose message contains the given substring.
func countIssuesContaining(issues []ConversionIssue, substring string) int {
	count := 0
	for _, issue := range issues {
		if strings.Contains(issue.Message, substring) {
			count++
		}
	}
	return count
}
