package converter

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertOAS2RequestBody tests the convertOAS2RequestBody method.
func TestConvertOAS2RequestBody(t *testing.T) {
	tests := []struct {
		name      string
		operation *parser.Operation
		document  *parser.OAS2Document
		hasBody   bool
	}{
		{
			name: "operation with body parameter",
			operation: &parser.Operation{
				Parameters: []*parser.Parameter{
					{
						Name:        "body",
						In:          "body",
						Description: "User object",
						Required:    true,
						Schema:      &parser.Schema{Type: "object"},
					},
				},
			},
			document: &parser.OAS2Document{
				Consumes: []string{"application/json"},
			},
			hasBody: true,
		},
		{
			name: "operation without body parameter",
			operation: &parser.Operation{
				Parameters: []*parser.Parameter{
					{
						Name: "id",
						In:   "path",
					},
				},
			},
			document: &parser.OAS2Document{},
			hasBody:  false,
		},
		{
			name: "operation with body and operation-level consumes",
			operation: &parser.Operation{
				Consumes: []string{"application/xml"},
				Parameters: []*parser.Parameter{
					{
						Name:   "body",
						In:     "body",
						Schema: &parser.Schema{Type: "object"},
					},
				},
			},
			document: &parser.OAS2Document{
				Consumes: []string{"application/json"},
			},
			hasBody: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := c.convertOAS2RequestBody(tt.operation, tt.document)

			if tt.hasBody {
				require.NotNil(t, result, "convertOAS2RequestBody should return non-nil for operation with body parameter")
				assert.NotNil(t, result.Content, "RequestBody should have content")
				assert.NotEmpty(t, result.Content, "RequestBody content should not be empty")
			} else {
				assert.Nil(t, result, "convertOAS2RequestBody should return nil for operation without body parameter")
			}
		})
	}
}

// TestGetConsumes tests the getConsumes method.
func TestGetConsumes(t *testing.T) {
	tests := []struct {
		name      string
		operation *parser.Operation
		document  *parser.OAS2Document
		expected  []string
	}{
		{
			name: "operation-level consumes",
			operation: &parser.Operation{
				Consumes: []string{"application/xml", "application/json"},
			},
			document: &parser.OAS2Document{
				Consumes: []string{"text/plain"},
			},
			expected: []string{"application/xml", "application/json"},
		},
		{
			name:      "document-level consumes",
			operation: &parser.Operation{},
			document: &parser.OAS2Document{
				Consumes: []string{"application/json"},
			},
			expected: []string{"application/json"},
		},
		{
			name:      "no consumes defined",
			operation: &parser.Operation{},
			document:  &parser.OAS2Document{},
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := c.getConsumes(tt.operation, tt.document)
			assert.Equal(t, tt.expected, result, "getConsumes should return expected consumes array")
		})
	}
}

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

// TestConvertOAS3RequestBodyToOAS2 tests the convertOAS3RequestBodyToOAS2 method.
func TestConvertOAS3RequestBodyToOAS2(t *testing.T) {
	tests := []struct {
		name        string
		requestBody *parser.RequestBody
		expectParam bool
	}{
		{
			name: "request body with JSON content",
			requestBody: &parser.RequestBody{
				Description: "User object",
				Required:    true,
				Content: map[string]*parser.MediaType{
					"application/json": {
						Schema: &parser.Schema{Type: "object"},
					},
				},
			},
			expectParam: true,
		},
		{
			name: "request body with multiple media types",
			requestBody: &parser.RequestBody{
				Description: "Pet object",
				Content: map[string]*parser.MediaType{
					"application/json": {
						Schema: &parser.Schema{Type: "object"},
					},
					"application/xml": {
						Schema: &parser.Schema{Type: "object"},
					},
				},
			},
			expectParam: true,
		},
		{
			name:        "nil request body",
			requestBody: nil,
			expectParam: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := &ConversionResult{Issues: []ConversionIssue{}}
			converted, mediaTypes := c.convertOAS3RequestBodyToOAS2(tt.requestBody, result, "test.path")

			if tt.expectParam {
				require.NotNil(t, converted, "convertOAS3RequestBodyToOAS2 should return non-nil parameter")
				assert.Equal(t, "body", converted.In, "Converted parameter should be in body")
				assert.NotNil(t, mediaTypes, "Media types should be returned")
			} else {
				assert.Nil(t, converted, "convertOAS3RequestBodyToOAS2 should return nil for nil request body")
			}
		})
	}
}

// TestRewriteParameterRefsOAS2ToOAS3 tests the rewriteParameterRefsOAS2ToOAS3 function.
func TestRewriteParameterRefsOAS2ToOAS3(t *testing.T) {
	tests := []struct {
		name     string
		param    *parser.Parameter
		hasRef   bool
		expected string
	}{
		{
			name: "parameter with OAS 2.0 reference",
			param: &parser.Parameter{
				Ref: "#/parameters/UserId",
			},
			hasRef:   true,
			expected: "#/components/parameters/UserId",
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
			rewriteParameterRefsOAS2ToOAS3(tt.param)

			if tt.hasRef {
				assert.Equal(t, tt.expected, tt.param.Ref, "Reference should be rewritten to OAS 3.x format")
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
