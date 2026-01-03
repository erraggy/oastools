// walker_handlers_test.go - Tests for individual handler types
// Tests schema traversal, parameters, headers, responses, media types,
// and OAS3 components.

package walker

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Schema Tests
// =============================================================================

func TestWalk_NestedSchemas(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
						"tag":  {Type: "string"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaPaths = append(schemaPaths, wc.JSONPath)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Len(t, schemaPaths, 3) // Pet + name + tag
	assert.Contains(t, schemaPaths, "$.components.schemas['Pet']")
	assert.Contains(t, schemaPaths, "$.components.schemas['Pet'].properties['name']")
	assert.Contains(t, schemaPaths, "$.components.schemas['Pet'].properties['tag']")
}

func TestWalk_CircularSchemas(t *testing.T) {
	// Create a circular schema reference
	petSchema := &parser.Schema{Type: "object"}
	petSchema.Properties = map[string]*parser.Schema{
		"parent": petSchema, // Circular reference
	}

	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": petSchema,
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	visitCount := 0
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitCount++
			return Continue
		}),
	)

	require.NoError(t, err)
	// Should only visit once due to cycle detection
	assert.Equal(t, 1, visitCount)
}

func TestWalk_MaxSchemaDepth(t *testing.T) {
	// Create deeply nested schema
	deepSchema := &parser.Schema{Type: "object"}
	current := deepSchema
	for i := 0; i < 10; i++ {
		nested := &parser.Schema{Type: "object"}
		current.Properties = map[string]*parser.Schema{"nested": nested}
		current = nested
	}

	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Deep": deepSchema,
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	visitCount := 0
	err := Walk(result,
		WithMaxDepth(3),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitCount++
			return Continue
		}),
	)

	require.NoError(t, err)
	// Should stop at depth 3
	assert.LessOrEqual(t, visitCount, 4)
}

func TestWalk_SchemaComposition(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Combined": {
					AllOf: []*parser.Schema{
						{Type: "object"},
						{Type: "object"},
					},
					OneOf: []*parser.Schema{
						{Type: "string"},
					},
					AnyOf: []*parser.Schema{
						{Type: "integer"},
					},
					Not: &parser.Schema{Type: "null"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	visitCount := 0
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitCount++
			return Continue
		}),
	)

	require.NoError(t, err)
	// Combined + 2 allOf + 1 oneOf + 1 anyOf + 1 not = 6
	assert.Equal(t, 6, visitCount)
}

func TestWalk_Webhooks(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Webhooks: map[string]*parser.PathItem{
			"newPet": {
				Post: &parser.Operation{OperationID: "newPetWebhook"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var visitedOps []string
	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOps = append(visitedOps, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Contains(t, visitedOps, "newPetWebhook")
}

// Schema Keywords Tests - Work Package 5d

func TestWalk_SchemaPatternProperties(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"DynamicObject": {
					Type: "object",
					PatternProperties: map[string]*parser.Schema{
						"^x-": {Type: "string"},
						"^y-": {Type: "integer"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	schemaCount := 0
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaCount++
			return Continue
		}),
	)
	require.NoError(t, err)

	// DynamicObject + 2 pattern property schemas
	assert.GreaterOrEqual(t, schemaCount, 3)
}

func TestWalk_SchemaAdditionalProperties(t *testing.T) {
	additionalSchema := &parser.Schema{Type: "string"}
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"StringMap": {
					Type:                 "object",
					AdditionalProperties: additionalSchema,
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitedPaths = append(visitedPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	// Should visit the additionalProperties schema
	found := false
	for _, p := range visitedPaths {
		if strings.Contains(p, "additionalProperties") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit additionalProperties schema")
}

func TestWalk_SchemaUnevaluatedProperties(t *testing.T) {
	unevalSchema := &parser.Schema{Type: "string"}
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"StrictObject": {
					Type:                  "object",
					UnevaluatedProperties: unevalSchema,
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitedPaths = append(visitedPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range visitedPaths {
		if strings.Contains(p, "unevaluatedProperties") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit unevaluatedProperties schema")
}

func TestWalk_SchemaPrefixItems(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"TupleType": {
					Type: "array",
					PrefixItems: []*parser.Schema{
						{Type: "string"},
						{Type: "integer"},
						{Type: "boolean"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	schemaCount := 0
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaCount++
			return Continue
		}),
	)
	require.NoError(t, err)

	// TupleType + 3 prefixItems schemas
	assert.GreaterOrEqual(t, schemaCount, 4)
}

func TestWalk_SchemaConditionals(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"ConditionalSchema": {
					Type: "object",
					If:   &parser.Schema{Properties: map[string]*parser.Schema{"type": {Const: "premium"}}},
					Then: &parser.Schema{Required: []string{"premiumFeatures"}},
					Else: &parser.Schema{Required: []string{"basicFeatures"}},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitedPaths = append(visitedPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	hasIf, hasThen, hasElse := false, false, false
	for _, p := range visitedPaths {
		if strings.Contains(p, ".if") {
			hasIf = true
		}
		if strings.Contains(p, ".then") {
			hasThen = true
		}
		if strings.Contains(p, ".else") {
			hasElse = true
		}
	}
	assert.True(t, hasIf, "should visit if schema")
	assert.True(t, hasThen, "should visit then schema")
	assert.True(t, hasElse, "should visit else schema")
}

func TestWalk_SchemaDefs(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"ParentSchema": {
					Type: "object",
					Defs: map[string]*parser.Schema{
						"NestedDef":  {Type: "string"},
						"AnotherDef": {Type: "integer"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitedPaths = append(visitedPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	defsCount := 0
	for _, p := range visitedPaths {
		if strings.Contains(p, "$defs") {
			defsCount++
		}
	}
	assert.GreaterOrEqual(t, defsCount, 2, "should visit $defs schemas")
}

// floatPtr is a helper function for creating float64 pointers
func floatPtr(f float64) *float64 {
	return &f
}

func TestWalk_SchemaContains(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"ArrayWithContains": {
					Type:     "array",
					Contains: &parser.Schema{Type: "integer", Minimum: floatPtr(0)},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitedPaths = append(visitedPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range visitedPaths {
		if strings.Contains(p, ".contains") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit contains schema")
}

func TestWalk_SchemaPropertyNames(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"RestrictedKeys": {
					Type:          "object",
					PropertyNames: &parser.Schema{Pattern: "^[a-z]+$"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitedPaths = append(visitedPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range visitedPaths {
		if strings.Contains(p, ".propertyNames") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit propertyNames schema")
}

// SchemaSkippedHandler Tests

func TestWalk_SchemaSkippedDepthLimit(t *testing.T) {
	// Create deeply nested schema that will exceed depth limit
	deepSchema := &parser.Schema{Type: "object"}
	current := deepSchema
	for i := 0; i < 10; i++ {
		nested := &parser.Schema{Type: "object"}
		current.Properties = map[string]*parser.Schema{"nested": nested}
		current = nested
	}

	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Deep": deepSchema,
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var skippedReasons []string
	var skippedPaths []string
	err := Walk(result,
		WithMaxDepth(3),
		WithSchemaSkippedHandler(func(wc *WalkContext, reason string, schema *parser.Schema) {
			skippedReasons = append(skippedReasons, reason)
			skippedPaths = append(skippedPaths, wc.JSONPath)
		}),
	)

	require.NoError(t, err)
	// Should have skipped schemas due to depth limit
	assert.NotEmpty(t, skippedReasons, "expected schemas to be skipped due to depth")
	for _, reason := range skippedReasons {
		assert.Equal(t, "depth", reason, "expected skip reason to be 'depth'")
	}
	// The paths should show the nested structure
	for _, p := range skippedPaths {
		assert.Contains(t, p, "Deep", "path should reference the Deep schema")
	}
}

func TestWalk_SchemaSkippedCycle(t *testing.T) {
	// Create a circular schema reference
	petSchema := &parser.Schema{Type: "object"}
	petSchema.Properties = map[string]*parser.Schema{
		"parent": petSchema, // Circular reference
	}

	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": petSchema,
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var skippedReasons []string
	var skippedSchemas []*parser.Schema
	var skippedPaths []string
	err := Walk(result,
		WithSchemaSkippedHandler(func(wc *WalkContext, reason string, schema *parser.Schema) {
			skippedReasons = append(skippedReasons, reason)
			skippedSchemas = append(skippedSchemas, schema)
			skippedPaths = append(skippedPaths, wc.JSONPath)
		}),
	)

	require.NoError(t, err)
	// Should have skipped the circular reference
	assert.Len(t, skippedReasons, 1, "expected one schema to be skipped due to cycle")
	assert.Equal(t, "cycle", skippedReasons[0], "expected skip reason to be 'cycle'")
	assert.Equal(t, petSchema, skippedSchemas[0], "expected skipped schema to be the pet schema")
	assert.Contains(t, skippedPaths[0], "parent", "expected path to include 'parent' property")
}

func TestWalk_SchemaSkippedHandlerNotCalledForNil(t *testing.T) {
	// Test that the handler is NOT called for nil schemas
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Empty": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	handlerCalled := false
	err := Walk(result,
		WithSchemaSkippedHandler(func(wc *WalkContext, reason string, schema *parser.Schema) {
			handlerCalled = true
		}),
	)

	require.NoError(t, err)
	assert.False(t, handlerCalled, "handler should not be called when no schemas are skipped")
}

// walkSchemaProperties Coverage Tests

func TestWalk_SchemaDependentSchemas(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"ConditionalObject": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name":    {Type: "string"},
						"address": {Type: "string"},
					},
					DependentSchemas: map[string]*parser.Schema{
						"name": {
							Properties: map[string]*parser.Schema{
								"firstName": {Type: "string"},
								"lastName":  {Type: "string"},
							},
						},
						"address": {
							Properties: map[string]*parser.Schema{
								"street": {Type: "string"},
								"city":   {Type: "string"},
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaPaths = append(schemaPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	// Count dependentSchemas entries
	dependentSchemaCount := 0
	for _, p := range schemaPaths {
		if strings.Contains(p, "dependentSchemas") {
			dependentSchemaCount++
		}
	}
	// Should have 2 dependentSchemas (name and address) plus their nested properties
	assert.GreaterOrEqual(t, dependentSchemaCount, 2, "should visit dependentSchemas")
}

func TestWalk_SchemaPropertiesSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Object": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"nested": {
							Type: "object",
							Properties: map[string]*parser.Schema{
								"deep": {Type: "string"},
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitedPaths = append(visitedPaths, wc.JSONPath)
			// Skip children of the nested schema
			if strings.Contains(wc.JSONPath, "nested") && !strings.Contains(wc.JSONPath, "deep") {
				return SkipChildren
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	// Should visit Object and nested, but not deep
	deepVisited := false
	for _, p := range visitedPaths {
		if strings.Contains(p, "deep") {
			deepVisited = true
		}
	}
	assert.False(t, deepVisited, "deep schema should not be visited when parent returns SkipChildren")
}

func TestWalk_SchemaPropertiesStop(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Object": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"a": {Type: "string"},
						"b": {Type: "string"},
						"c": {Type: "string"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	visitCount := 0
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitCount++
			// Stop after visiting 2 schemas (Object + first property)
			if visitCount >= 2 {
				return Stop
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Equal(t, 2, visitCount, "should stop after 2 schemas")
}

// walkSchemaArrayKeywords Coverage Tests

func TestWalk_SchemaItemsAsSchema(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"StringArray": {
					Type:  "array",
					Items: &parser.Schema{Type: "string"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaPaths = append(schemaPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range schemaPaths {
		if strings.Contains(p, ".items") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit items schema")
}

func TestWalk_SchemaAdditionalItems(t *testing.T) {
	// AdditionalItems is a JSON Schema draft-07 keyword still sometimes used
	additionalItemsSchema := &parser.Schema{Type: "number"}
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"TupleWithExtra": {
					Type: "array",
					PrefixItems: []*parser.Schema{
						{Type: "string"},
						{Type: "integer"},
					},
					AdditionalItems: additionalItemsSchema,
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaPaths = append(schemaPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range schemaPaths {
		if strings.Contains(p, "additionalItems") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit additionalItems schema")
}

func TestWalk_SchemaUnevaluatedItems(t *testing.T) {
	unevaluatedItemsSchema := &parser.Schema{Type: "boolean"}
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"StrictArray": {
					Type: "array",
					PrefixItems: []*parser.Schema{
						{Type: "string"},
					},
					UnevaluatedItems: unevaluatedItemsSchema,
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaPaths = append(schemaPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range schemaPaths {
		if strings.Contains(p, "unevaluatedItems") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit unevaluatedItems schema")
}

func TestWalk_SchemaArrayKeywordsSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Array": {
					Type: "array",
					Items: &parser.Schema{
						Type: "object",
						Properties: map[string]*parser.Schema{
							"nested": {Type: "string"},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitedPaths = append(visitedPaths, wc.JSONPath)
			// Skip children of items schema
			if strings.Contains(wc.JSONPath, ".items") {
				return SkipChildren
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	nestedVisited := false
	for _, p := range visitedPaths {
		if strings.Contains(p, "nested") {
			nestedVisited = true
		}
	}
	assert.False(t, nestedVisited, "nested property should not be visited when items returns SkipChildren")
}

// walkSchemaMisc Coverage Tests

func TestWalk_SchemaContentSchema(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"EncodedContent": {
					Type:             "string",
					ContentEncoding:  "base64",
					ContentMediaType: "application/json",
					ContentSchema: &parser.Schema{
						Type: "object",
						Properties: map[string]*parser.Schema{
							"data": {Type: "string"},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaPaths = append(schemaPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range schemaPaths {
		if strings.Contains(p, "contentSchema") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit contentSchema")
}

func TestWalk_SchemaMiscSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"WithContentSchema": {
					Type:            "string",
					ContentEncoding: "base64",
					ContentSchema: &parser.Schema{
						Type: "object",
						Properties: map[string]*parser.Schema{
							"nested": {Type: "string"},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitedPaths = append(visitedPaths, wc.JSONPath)
			// Skip children of the root schema (which has contentSchema)
			if strings.HasSuffix(wc.JSONPath, "['WithContentSchema']") {
				return SkipChildren
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	contentSchemaVisited := false
	for _, p := range visitedPaths {
		if strings.Contains(p, "contentSchema") {
			contentSchemaVisited = true
		}
	}
	assert.False(t, contentSchemaVisited, "contentSchema should not be visited when parent returns SkipChildren")
}

func TestWalk_SchemaNotKeyword(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"NotNull": {
					Not: &parser.Schema{Type: "null"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaPaths = append(schemaPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range schemaPaths {
		if strings.Contains(p, ".not") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit not schema")
}

// =============================================================================
// Parameter Tests
// =============================================================================

func TestWalk_ParameterWithSchema(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets/{id}": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPet",
					Parameters: []*parser.Parameter{
						{
							Name:   "id",
							In:     "path",
							Schema: &parser.Schema{Type: "integer"},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaPaths = append(schemaPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range schemaPaths {
		if strings.Contains(p, "parameters[0].schema") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit parameter schema")
}

func TestWalk_ParameterWithContent(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Parameters: []*parser.Parameter{
						{
							Name: "filter",
							In:   "query",
							Content: map[string]*parser.MediaType{
								"application/json": {
									Schema: &parser.Schema{Type: "object"},
								},
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var mediaTypePaths []string
	err := Walk(result,
		WithMediaTypeHandler(func(wc *WalkContext, mt *parser.MediaType) Action {
			mediaTypePaths = append(mediaTypePaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range mediaTypePaths {
		if strings.Contains(p, "parameters[0].content") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit parameter content media type")
}

func TestWalk_ParameterWithExamples(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets/{id}": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPet",
					Parameters: []*parser.Parameter{
						{
							Name: "id",
							In:   "path",
							Examples: map[string]*parser.Example{
								"petId1": {Summary: "First pet", Value: 1},
								"petId2": {Summary: "Second pet", Value: 2},
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var exampleNames []string
	err := Walk(result,
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			exampleNames = append(exampleNames, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, exampleNames, 2)
	assert.Contains(t, exampleNames, "petId1")
	assert.Contains(t, exampleNames, "petId2")
}

func TestWalk_ParameterSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets/{id}": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPet",
					Parameters: []*parser.Parameter{
						{
							Name:   "id",
							In:     "path",
							Schema: &parser.Schema{Type: "integer"},
							Examples: map[string]*parser.Example{
								"example1": {Summary: "Example"},
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	schemaVisited := false
	exampleVisited := false
	err := Walk(result,
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			return SkipChildren
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if strings.Contains(wc.JSONPath, "parameters") {
				schemaVisited = true
			}
			return Continue
		}),
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			if strings.Contains(wc.JSONPath, "parameters") {
				exampleVisited = true
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.False(t, schemaVisited, "schema should not be visited when parameter handler returns SkipChildren")
	assert.False(t, exampleVisited, "example should not be visited when parameter handler returns SkipChildren")
}

func TestWalk_ParameterStop(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Parameters: []*parser.Parameter{
						{Name: "limit", In: "query"},
						{Name: "offset", In: "query"},
						{Name: "filter", In: "query"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedParams []string
	err := Walk(result,
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visitedParams = append(visitedParams, param.Name)
			if param.Name == "limit" {
				return Stop
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedParams, 1, "should stop after first parameter")
	assert.Equal(t, "limit", visitedParams[0])
}

func TestWalk_NilParameterInSlice(t *testing.T) {
	// Test that nil parameters in a slice are handled gracefully
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Parameters: []*parser.Parameter{
						{Name: "valid", In: "query"},
						nil, // nil parameter
						{Name: "another", In: "query"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedParams []string
	err := Walk(result,
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visitedParams = append(visitedParams, param.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	// Should visit only non-nil parameters
	assert.Len(t, visitedParams, 2)
	assert.Contains(t, visitedParams, "valid")
	assert.Contains(t, visitedParams, "another")
}

// =============================================================================
// Header Tests
// =============================================================================

func TestWalk_HeaderWithContent(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"X-Custom-Header": {
					Description: "Custom header with content",
					Content: map[string]*parser.MediaType{
						"application/json": {
							Schema: &parser.Schema{Type: "object"},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var mediaTypePaths []string
	err := Walk(result,
		WithMediaTypeHandler(func(wc *WalkContext, mt *parser.MediaType) Action {
			mediaTypePaths = append(mediaTypePaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range mediaTypePaths {
		if strings.Contains(p, "headers") && strings.Contains(p, "content") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit header content media type")
}

func TestWalk_HeaderWithExamples(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"X-Request-ID": {
					Description: "Request ID header",
					Schema:      &parser.Schema{Type: "string"},
					Examples: map[string]*parser.Example{
						"uuid1": {Summary: "UUID example", Value: "123e4567-e89b-12d3-a456-426614174000"},
						"uuid2": {Summary: "Another UUID", Value: "987fcdeb-51a2-43e6-b7c8-123456789abc"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var exampleNames []string
	err := Walk(result,
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			exampleNames = append(exampleNames, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, exampleNames, 2)
	assert.Contains(t, exampleNames, "uuid1")
	assert.Contains(t, exampleNames, "uuid2")
}

func TestWalk_HeaderSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"X-Rate-Limit": {
					Description: "Rate limit header",
					Schema:      &parser.Schema{Type: "integer"},
					Examples: map[string]*parser.Example{
						"example1": {Summary: "Example"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	schemaVisited := false
	exampleVisited := false
	err := Walk(result,
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			return SkipChildren
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if strings.Contains(wc.JSONPath, "headers") {
				schemaVisited = true
			}
			return Continue
		}),
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			if strings.Contains(wc.JSONPath, "headers") {
				exampleVisited = true
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.False(t, schemaVisited, "schema should not be visited when header handler returns SkipChildren")
	assert.False(t, exampleVisited, "example should not be visited when header handler returns SkipChildren")
}

func TestWalk_HeaderStop(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"A-Header": {Description: "First header"},
				"B-Header": {Description: "Second header"},
				"C-Header": {Description: "Third header"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedHeaders []string
	err := Walk(result,
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			visitedHeaders = append(visitedHeaders, wc.Name)
			// Stop after first header
			return Stop
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedHeaders, 1, "should stop after first header")
}

// =============================================================================
// MediaType Tests
// =============================================================================

func TestWalk_MediaTypeExamples(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Examples: map[string]*parser.Example{
											"cat":  {Summary: "A cat"},
											"dog":  {Summary: "A dog"},
											"bird": {Summary: "A bird"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var exampleNames []string
	err := Walk(result,
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			exampleNames = append(exampleNames, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	// Should visit all 3 examples in the media type
	assert.Len(t, exampleNames, 3)
	assert.Contains(t, exampleNames, "cat")
	assert.Contains(t, exampleNames, "dog")
	assert.Contains(t, exampleNames, "bird")
}

func TestWalk_MediaTypeSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Type: "object"},
										Examples: map[string]*parser.Example{
											"example1": {Summary: "Test"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	schemaVisited := false
	exampleVisited := false
	err := Walk(result,
		WithMediaTypeHandler(func(wc *WalkContext, mt *parser.MediaType) Action {
			return SkipChildren
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if strings.Contains(wc.JSONPath, "content") {
				schemaVisited = true
			}
			return Continue
		}),
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			if strings.Contains(wc.JSONPath, "content") {
				exampleVisited = true
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.False(t, schemaVisited, "schema should not be visited when mediaType returns SkipChildren")
	assert.False(t, exampleVisited, "example should not be visited when mediaType returns SkipChildren")
}

// =============================================================================
// OAS 3.x Component Tests - Work Package 5c
// =============================================================================

func TestWalk_OAS3ComponentResponses(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Responses: map[string]*parser.Response{
				"NotFound": {
					Description: "Resource not found",
					Content: map[string]*parser.MediaType{
						"application/json": {
							Schema: &parser.Schema{Type: "object"},
						},
					},
				},
				"ServerError": {
					Description: "Internal server error",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedResponses []string
	err := Walk(result,
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			// Component responses use wc.Name (the response key), not wc.StatusCode
			visitedResponses = append(visitedResponses, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedResponses, 2)
	assert.Contains(t, visitedResponses, "NotFound")
	assert.Contains(t, visitedResponses, "ServerError")
}

func TestWalk_OAS3ComponentParameters(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Parameters: map[string]*parser.Parameter{
				"pageParam": {
					Name:   "page",
					In:     "query",
					Schema: &parser.Schema{Type: "integer"},
				},
				"limitParam": {
					Name:   "limit",
					In:     "query",
					Schema: &parser.Schema{Type: "integer"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedParams []string
	err := Walk(result,
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visitedParams = append(visitedParams, param.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedParams, 2)
	assert.Contains(t, visitedParams, "page")
	assert.Contains(t, visitedParams, "limit")
}

func TestWalk_OAS3ComponentRequestBodies(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			RequestBodies: map[string]*parser.RequestBody{
				"UserInput": {
					Description: "User input data",
					Content: map[string]*parser.MediaType{
						"application/json": {
							Schema: &parser.Schema{Type: "object"},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithRequestBodyHandler(func(wc *WalkContext, body *parser.RequestBody) Action {
			visitedPaths = append(visitedPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedPaths, 1)
	assert.Contains(t, visitedPaths[0], "requestBodies['UserInput']")
}

func TestWalk_OAS3ComponentCallbacks(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Callbacks: map[string]*parser.Callback{
				"onPayment": {
					"{$request.body#/callbackUrl}": &parser.PathItem{
						Post: &parser.Operation{
							Summary: "Payment callback",
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedCallbacks []string
	err := Walk(result,
		WithCallbackHandler(func(wc *WalkContext, callback parser.Callback) Action {
			visitedCallbacks = append(visitedCallbacks, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedCallbacks, 1)
	assert.Contains(t, visitedCallbacks, "onPayment")
}

func TestWalk_OAS3ComponentLinks(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Links: map[string]*parser.Link{
				"GetUserById": {
					OperationID: "getUser",
					Description: "Get user by ID link",
				},
				"GetOrderById": {
					OperationID: "getOrder",
					Description: "Get order by ID link",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedLinks []string
	err := Walk(result,
		WithLinkHandler(func(wc *WalkContext, link *parser.Link) Action {
			visitedLinks = append(visitedLinks, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedLinks, 2)
	assert.Contains(t, visitedLinks, "GetUserById")
	assert.Contains(t, visitedLinks, "GetOrderById")
}

func TestWalk_OAS3ComponentPathItems(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			PathItems: map[string]*parser.PathItem{
				"SharedEndpoint": {
					Get: &parser.Operation{
						Summary: "Shared GET operation",
					},
					Post: &parser.Operation{
						Summary: "Shared POST operation",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPathItems []string
	err := Walk(result,
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			visitedPathItems = append(visitedPathItems, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	// Should visit the component path item
	found := false
	for _, p := range visitedPathItems {
		if strings.Contains(p, "components.pathItems") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit components.pathItems")
}

func TestWalk_OAS3ComponentExamples(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Examples: map[string]*parser.Example{
				"UserExample": {
					Summary: "Example user",
					Value:   map[string]any{"id": 1, "name": "John"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedExamples []string
	err := Walk(result,
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			visitedExamples = append(visitedExamples, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedExamples, 1)
	assert.Contains(t, visitedExamples, "UserExample")
}

func TestWalk_OAS3ComponentHeaders(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"X-Rate-Limit": {
					Description: "Rate limit header",
					Schema:      &parser.Schema{Type: "integer"},
				},
				"X-Request-ID": {
					Description: "Request ID header",
					Schema:      &parser.Schema{Type: "string"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedHeaders []string
	err := Walk(result,
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			visitedHeaders = append(visitedHeaders, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedHeaders, 2)
	assert.Contains(t, visitedHeaders, "X-Rate-Limit")
	assert.Contains(t, visitedHeaders, "X-Request-ID")
}

func TestWalk_OAS3ComponentSecuritySchemes(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"bearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				},
				"apiKeyAuth": {
					Type: "apiKey",
					Name: "X-API-Key",
					In:   "header",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedSchemes []string
	err := Walk(result,
		WithSecuritySchemeHandler(func(wc *WalkContext, scheme *parser.SecurityScheme) Action {
			visitedSchemes = append(visitedSchemes, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedSchemes, 2)
	assert.Contains(t, visitedSchemes, "bearerAuth")
	assert.Contains(t, visitedSchemes, "apiKeyAuth")
}

func TestWalk_OAS3AllComponents(t *testing.T) {
	// Test walking a document with all component types
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"User": {Type: "object"},
			},
			Responses: map[string]*parser.Response{
				"NotFound": {Description: "Not found"},
			},
			Parameters: map[string]*parser.Parameter{
				"pageParam": {Name: "page", In: "query"},
			},
			RequestBodies: map[string]*parser.RequestBody{
				"UserInput": {Description: "User input"},
			},
			Headers: map[string]*parser.Header{
				"X-Rate-Limit": {Description: "Rate limit"},
			},
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"bearerAuth": {Type: "http"},
			},
			Links: map[string]*parser.Link{
				"GetUserById": {OperationID: "getUser"},
			},
			Callbacks: map[string]*parser.Callback{
				"onEvent": {
					"{$url}": &parser.PathItem{},
				},
			},
			Examples: map[string]*parser.Example{
				"UserExample": {Summary: "User example"},
			},
			PathItems: map[string]*parser.PathItem{
				"SharedPath": {Get: &parser.Operation{Summary: "Shared"}},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	visited := make(map[string]bool)

	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visited["schema"] = true
			return Continue
		}),
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			visited["response"] = true
			return Continue
		}),
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visited["parameter"] = true
			return Continue
		}),
		WithRequestBodyHandler(func(wc *WalkContext, body *parser.RequestBody) Action {
			visited["requestBody"] = true
			return Continue
		}),
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			visited["header"] = true
			return Continue
		}),
		WithSecuritySchemeHandler(func(wc *WalkContext, scheme *parser.SecurityScheme) Action {
			visited["securityScheme"] = true
			return Continue
		}),
		WithLinkHandler(func(wc *WalkContext, link *parser.Link) Action {
			visited["link"] = true
			return Continue
		}),
		WithCallbackHandler(func(wc *WalkContext, callback parser.Callback) Action {
			visited["callback"] = true
			return Continue
		}),
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			visited["example"] = true
			return Continue
		}),
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			if strings.Contains(wc.JSONPath, "components.pathItems") {
				visited["pathItem"] = true
			}
			return Continue
		}),
	)

	require.NoError(t, err)

	expected := []string{
		"schema", "response", "parameter", "requestBody",
		"header", "securityScheme", "link", "callback",
		"example", "pathItem",
	}

	for _, name := range expected {
		assert.True(t, visited[name], "expected %s component to be visited", name)
	}
}

// Tests for walkOAS3PathItemOperations - Coverage for all HTTP methods

func TestWalk_OAS3AllHTTPMethods(t *testing.T) {
	// Test that all HTTP methods including TRACE are visited
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/resource": &parser.PathItem{
				Get:     &parser.Operation{OperationID: "getResource"},
				Put:     &parser.Operation{OperationID: "putResource"},
				Post:    &parser.Operation{OperationID: "postResource"},
				Delete:  &parser.Operation{OperationID: "deleteResource"},
				Options: &parser.Operation{OperationID: "optionsResource"},
				Head:    &parser.Operation{OperationID: "headResource"},
				Patch:   &parser.Operation{OperationID: "patchResource"},
				Trace:   &parser.Operation{OperationID: "traceResource"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	visitedMethods := make(map[string]bool)
	visitedOpIDs := make(map[string]bool)
	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedMethods[wc.Method] = true
			visitedOpIDs[op.OperationID] = true
			return Continue
		}),
	)

	require.NoError(t, err)

	// All 8 HTTP methods should be visited
	expectedMethods := []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}
	for _, method := range expectedMethods {
		assert.True(t, visitedMethods[method], "expected %s method to be visited", method)
	}

	// All operation IDs should be visited
	expectedOpIDs := []string{
		"getResource", "putResource", "postResource", "deleteResource",
		"optionsResource", "headResource", "patchResource", "traceResource",
	}
	for _, opID := range expectedOpIDs {
		assert.True(t, visitedOpIDs[opID], "expected operation %s to be visited", opID)
	}
}

func TestWalk_OAS3OperationSkipChildren(t *testing.T) {
	// Test that SkipChildren from operation handler skips operation's children
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/resource": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getResource",
					Parameters: []*parser.Parameter{
						{Name: "id", In: "query"},
					},
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {Schema: &parser.Schema{Type: "object"}},
						},
					},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "OK"},
						},
					},
				},
				Post: &parser.Operation{
					OperationID: "postResource",
					Parameters: []*parser.Parameter{
						{Name: "body", In: "body"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var visitedParams []string
	var visitedResponses []string
	var visitedRequestBodies int
	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			if wc.Method == "get" {
				return SkipChildren // Skip GET operation's children
			}
			return Continue
		}),
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visitedParams = append(visitedParams, param.Name)
			return Continue
		}),
		WithRequestBodyHandler(func(wc *WalkContext, reqBody *parser.RequestBody) Action {
			visitedRequestBodies++
			return Continue
		}),
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			visitedResponses = append(visitedResponses, wc.StatusCode)
			return Continue
		}),
	)

	require.NoError(t, err)

	// Only POST's parameters should be visited (GET's were skipped)
	assert.Equal(t, []string{"body"}, visitedParams)

	// GET's responses and request body should be skipped
	assert.Empty(t, visitedResponses)
	assert.Equal(t, 0, visitedRequestBodies)
}

func TestWalk_OAS3OperationStop(t *testing.T) {
	// Test that Stop from operation handler stops the entire walk
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/a": &parser.PathItem{
				Get:  &parser.Operation{OperationID: "getA"},
				Post: &parser.Operation{OperationID: "postA"},
			},
			"/b": &parser.PathItem{
				Get: &parser.Operation{OperationID: "getB"},
			},
			"/c": &parser.PathItem{
				Get: &parser.Operation{OperationID: "getC"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var visitedOps []string
	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOps = append(visitedOps, op.OperationID)
			// Stop after visiting first operation
			return Stop
		}),
	)

	require.NoError(t, err)
	// Only one operation should be visited due to Stop
	assert.Len(t, visitedOps, 1)
}

func TestWalk_OAS3TRACEMethod(t *testing.T) {
	// Specifically test TRACE method handling
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/debug": &parser.PathItem{
				Trace: &parser.Operation{
					OperationID: "traceDebug",
					Summary:     "Debug trace endpoint",
					Parameters: []*parser.Parameter{
						{Name: "X-Trace-ID", In: "header"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var traceVisited bool
	var tracePath string
	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			if wc.Method == "trace" {
				traceVisited = true
				tracePath = wc.JSONPath
			}
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.True(t, traceVisited, "TRACE operation should be visited")
	assert.Contains(t, tracePath, ".trace", "path should contain .trace")
}

// Tests for walkOAS3Components - Coverage for component types

func TestWalk_OAS3ComponentParametersWithStop(t *testing.T) {
	// Test Stop action from component parameters
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Parameters: map[string]*parser.Parameter{
				"aParam": {Name: "a", In: "query"},
				"bParam": {Name: "b", In: "query"},
				"cParam": {Name: "c", In: "query"},
			},
			Schemas: map[string]*parser.Schema{
				"ShouldNotVisit": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var visitedParams []string
	var schemaVisited bool
	err := Walk(result,
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visitedParams = append(visitedParams, param.Name)
			return Stop // Stop after first parameter
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaVisited = true
			return Continue
		}),
	)

	require.NoError(t, err)
	// Only one parameter should be visited due to Stop
	assert.Len(t, visitedParams, 1)
	// Schemas should still be visited since they come before parameters
	// (component order is schemas, responses, parameters, ...)
	assert.True(t, schemaVisited)
}

func TestWalk_OAS3ComponentRequestBodiesWithContent(t *testing.T) {
	// Test request bodies in components with nested content
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			RequestBodies: map[string]*parser.RequestBody{
				"CreateUser": {
					Description: "Create user request",
					Required:    true,
					Content: map[string]*parser.MediaType{
						"application/json": {
							Schema: &parser.Schema{
								Type: "object",
								Properties: map[string]*parser.Schema{
									"name":  {Type: "string"},
									"email": {Type: "string"},
								},
							},
						},
						"application/xml": {
							Schema: &parser.Schema{Type: "object"},
						},
					},
				},
				"UpdateUser": {
					Description: "Update user request",
					Content: map[string]*parser.MediaType{
						"application/json": {
							Schema: &parser.Schema{Type: "object"},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var visitedRequestBodies []string
	var visitedMediaTypes []string
	var schemaCount int
	err := Walk(result,
		WithRequestBodyHandler(func(wc *WalkContext, reqBody *parser.RequestBody) Action {
			visitedRequestBodies = append(visitedRequestBodies, reqBody.Description)
			return Continue
		}),
		WithMediaTypeHandler(func(wc *WalkContext, mt *parser.MediaType) Action {
			visitedMediaTypes = append(visitedMediaTypes, wc.Name)
			return Continue
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaCount++
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Len(t, visitedRequestBodies, 2)
	assert.GreaterOrEqual(t, len(visitedMediaTypes), 3) // At least 3 media types
	assert.GreaterOrEqual(t, schemaCount, 4)            // Multiple schemas
}

func TestWalk_OAS3ComponentLinksWithStop(t *testing.T) {
	// Test Stop action from component links
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Links: map[string]*parser.Link{
				"GetUserByID": {
					OperationID: "getUser",
					Description: "Link to get user by ID",
				},
				"GetOrderByID": {
					OperationID: "getOrder",
					Description: "Link to get order by ID",
				},
				"GetProductByID": {
					OperationID: "getProduct",
					Description: "Link to get product by ID",
				},
			},
			// These should not be visited after Stop
			Examples: map[string]*parser.Example{
				"UserExample": {Summary: "User example"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var visitedLinks []string
	var examplesVisited bool
	err := Walk(result,
		WithLinkHandler(func(wc *WalkContext, link *parser.Link) Action {
			visitedLinks = append(visitedLinks, wc.Name)
			return Stop // Stop after first link
		}),
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			examplesVisited = true
			return Continue
		}),
	)

	require.NoError(t, err)
	// Only one link should be visited due to Stop
	assert.Len(t, visitedLinks, 1)
	// Examples should NOT be visited since they come after links and Stop was called
	assert.False(t, examplesVisited)
}

func TestWalk_OAS3ComponentCallbacksWithSkipChildren(t *testing.T) {
	// Test SkipChildren action from component callbacks
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Callbacks: map[string]*parser.Callback{
				"onPaymentComplete": {
					"{$request.body#/callbackUrl}": &parser.PathItem{
						Post: &parser.Operation{
							OperationID: "paymentCallback",
							Summary:     "Payment callback endpoint",
							Parameters: []*parser.Parameter{
								{Name: "X-Signature", In: "header"},
							},
						},
					},
				},
				"onShipmentUpdate": {
					"{$request.body#/shipmentCallback}": &parser.PathItem{
						Post: &parser.Operation{
							OperationID: "shipmentCallback",
							Summary:     "Shipment callback endpoint",
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var visitedCallbacks []string
	var visitedOperations []string
	err := Walk(result,
		WithCallbackHandler(func(wc *WalkContext, callback parser.Callback) Action {
			visitedCallbacks = append(visitedCallbacks, wc.Name)
			if wc.Name == "onPaymentComplete" {
				return SkipChildren // Skip children of first callback
			}
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOperations = append(visitedOperations, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	// Both callbacks should be visited
	assert.Len(t, visitedCallbacks, 2)
	assert.Contains(t, visitedCallbacks, "onPaymentComplete")
	assert.Contains(t, visitedCallbacks, "onShipmentUpdate")
	// Only the second callback's operations should be visited
	assert.Len(t, visitedOperations, 1)
	assert.Contains(t, visitedOperations, "shipmentCallback")
}

func TestWalk_OAS3ComponentPathItemsWithOperations(t *testing.T) {
	// Test component path items (OAS 3.1+) with full operation traversal
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			PathItems: map[string]*parser.PathItem{
				"SharedHealthCheck": {
					Get: &parser.Operation{
						OperationID: "healthCheck",
						Summary:     "Health check endpoint",
						Responses: &parser.Responses{
							Codes: map[string]*parser.Response{
								"200": {Description: "OK"},
							},
						},
					},
				},
				"SharedCRUD": {
					Get: &parser.Operation{
						OperationID: "listItems",
						Summary:     "List items",
					},
					Post: &parser.Operation{
						OperationID: "createItem",
						Summary:     "Create item",
						RequestBody: &parser.RequestBody{
							Content: map[string]*parser.MediaType{
								"application/json": {
									Schema: &parser.Schema{Type: "object"},
								},
							},
						},
					},
					Put: &parser.Operation{
						OperationID: "updateItem",
						Summary:     "Update item",
					},
					Delete: &parser.Operation{
						OperationID: "deleteItem",
						Summary:     "Delete item",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPathItems []string
	var visitedOperations []string
	var requestBodyCount int
	err := Walk(result,
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			visitedPathItems = append(visitedPathItems, wc.JSONPath)
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOperations = append(visitedOperations, op.OperationID)
			return Continue
		}),
		WithRequestBodyHandler(func(wc *WalkContext, reqBody *parser.RequestBody) Action {
			requestBodyCount++
			return Continue
		}),
	)

	require.NoError(t, err)
	// Both component path items should be visited
	assert.Len(t, visitedPathItems, 2)
	// All 5 operations should be visited
	assert.Len(t, visitedOperations, 5)
	assert.Contains(t, visitedOperations, "healthCheck")
	assert.Contains(t, visitedOperations, "listItems")
	assert.Contains(t, visitedOperations, "createItem")
	assert.Contains(t, visitedOperations, "updateItem")
	assert.Contains(t, visitedOperations, "deleteItem")
	// One request body should be visited
	assert.Equal(t, 1, requestBodyCount)
}

// Tests for walkOAS3Webhooks - Coverage improvements

func TestWalk_OAS3WebhooksWithOperations(t *testing.T) {
	// Test webhooks with full operation traversal
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Webhooks: map[string]*parser.PathItem{
			"newOrder": {
				Post: &parser.Operation{
					OperationID: "newOrderWebhook",
					Summary:     "New order webhook",
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{
									Type: "object",
									Properties: map[string]*parser.Schema{
										"orderId": {Type: "string"},
										"amount":  {Type: "number"},
									},
								},
							},
						},
					},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "Webhook received"},
						},
					},
				},
			},
			"orderCancelled": {
				Post: &parser.Operation{
					OperationID: "orderCancelledWebhook",
					Summary:     "Order cancelled webhook",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedWebhooks []string
	var visitedOperations []string
	var requestBodyCount int
	var responseCount int
	err := Walk(result,
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			if strings.Contains(wc.JSONPath, "webhooks") {
				visitedWebhooks = append(visitedWebhooks, wc.JSONPath)
			}
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOperations = append(visitedOperations, op.OperationID)
			return Continue
		}),
		WithRequestBodyHandler(func(wc *WalkContext, reqBody *parser.RequestBody) Action {
			requestBodyCount++
			return Continue
		}),
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			responseCount++
			return Continue
		}),
	)

	require.NoError(t, err)
	// Both webhooks should be visited
	assert.Len(t, visitedWebhooks, 2)
	// Both operations should be visited
	assert.Len(t, visitedOperations, 2)
	assert.Contains(t, visitedOperations, "newOrderWebhook")
	assert.Contains(t, visitedOperations, "orderCancelledWebhook")
	// One request body from newOrder webhook
	assert.Equal(t, 1, requestBodyCount)
	// One response from newOrder webhook
	assert.Equal(t, 1, responseCount)
}

func TestWalk_OAS3WebhooksSkipChildren(t *testing.T) {
	// Test SkipChildren from webhook's PathItem handler
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Webhooks: map[string]*parser.PathItem{
			"aWebhook": {
				Post: &parser.Operation{
					OperationID: "aWebhookOp",
					Parameters: []*parser.Parameter{
						{Name: "X-Signature", In: "header"},
					},
				},
			},
			"bWebhook": {
				Post: &parser.Operation{
					OperationID: "bWebhookOp",
					Parameters: []*parser.Parameter{
						{Name: "X-Timestamp", In: "header"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedWebhooks []string
	var visitedOperations []string
	err := Walk(result,
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			if strings.Contains(wc.JSONPath, "webhooks") {
				visitedWebhooks = append(visitedWebhooks, wc.JSONPath)
				if strings.Contains(wc.JSONPath, "aWebhook") {
					return SkipChildren // Skip first webhook's operations
				}
			}
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOperations = append(visitedOperations, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	// Both webhooks should be visited
	assert.Len(t, visitedWebhooks, 2)
	// Only second webhook's operation should be visited
	assert.Len(t, visitedOperations, 1)
	assert.Contains(t, visitedOperations, "bWebhookOp")
}

func TestWalk_OAS3WebhooksStop(t *testing.T) {
	// Test Stop from webhook handler
	// Order in walkOAS3: Paths -> Webhooks -> Components -> Tags
	// So stopping at webhooks should prevent components from being visited
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Webhooks: map[string]*parser.PathItem{
			"aWebhook": {
				Post: &parser.Operation{OperationID: "aOp"},
			},
			"bWebhook": {
				Post: &parser.Operation{OperationID: "bOp"},
			},
			"cWebhook": {
				Post: &parser.Operation{OperationID: "cOp"},
			},
		},
		// Components should NOT be visited after Stop in webhooks
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"ShouldNotVisit": {Type: "object"},
			},
		},
		// Tags should NOT be visited either
		Tags: []*parser.Tag{
			{Name: "shouldNotVisit"},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedWebhooks []string
	var schemaVisited bool
	var tagVisited bool
	err := Walk(result,
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			if strings.Contains(wc.JSONPath, "webhooks") {
				visitedWebhooks = append(visitedWebhooks, wc.JSONPath)
				return Stop // Stop after first webhook
			}
			return Continue
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaVisited = true
			return Continue
		}),
		WithTagHandler(func(wc *WalkContext, tag *parser.Tag) Action {
			tagVisited = true
			return Continue
		}),
	)

	require.NoError(t, err)
	// Only one webhook should be visited due to Stop
	assert.Len(t, visitedWebhooks, 1)
	// Components and Tags come after webhooks in the traversal order, so they should NOT be visited
	assert.False(t, schemaVisited, "schemas should not be visited after Stop in webhooks")
	assert.False(t, tagVisited, "tags should not be visited after Stop in webhooks")
}

// Additional tests for walkOAS3PathItemOperations edge cases

func TestWalk_OAS3QueryMethod(t *testing.T) {
	// Test OAS 3.2+ Query method
	doc := &parser.OAS3Document{
		OpenAPI: "3.2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/search": &parser.PathItem{
				Query: &parser.Operation{
					OperationID: "searchQuery",
					Summary:     "Search using QUERY method",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.2.0",
		OASVersion: parser.OASVersion320,
		Document:   doc,
	}

	var visitedMethods []string
	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedMethods = append(visitedMethods, wc.Method)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Contains(t, visitedMethods, "query")
}

func TestWalk_OAS3AdditionalOperations(t *testing.T) {
	// Test OAS 3.2+ AdditionalOperations
	doc := &parser.OAS3Document{
		OpenAPI: "3.2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/custom": &parser.PathItem{
				Get: &parser.Operation{OperationID: "getCustom"},
				AdditionalOperations: map[string]*parser.Operation{
					"customMethod1": {OperationID: "customOp1"},
					"customMethod2": {OperationID: "customOp2"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.2.0",
		OASVersion: parser.OASVersion320,
		Document:   doc,
	}

	var visitedMethods []string
	var visitedOpIDs []string
	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedMethods = append(visitedMethods, wc.Method)
			visitedOpIDs = append(visitedOpIDs, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	// Standard GET plus 2 additional operations
	assert.Len(t, visitedMethods, 3)
	assert.Contains(t, visitedMethods, "get")
	assert.Contains(t, visitedMethods, "customMethod1")
	assert.Contains(t, visitedMethods, "customMethod2")
	assert.Contains(t, visitedOpIDs, "getCustom")
	assert.Contains(t, visitedOpIDs, "customOp1")
	assert.Contains(t, visitedOpIDs, "customOp2")
}

func TestWalk_OAS3AdditionalOperationsStop(t *testing.T) {
	// Test Stop during AdditionalOperations traversal
	doc := &parser.OAS3Document{
		OpenAPI: "3.2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/custom": &parser.PathItem{
				AdditionalOperations: map[string]*parser.Operation{
					"aMethod": {OperationID: "aOp"},
					"bMethod": {OperationID: "bOp"},
					"cMethod": {OperationID: "cOp"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.2.0",
		OASVersion: parser.OASVersion320,
		Document:   doc,
	}

	var visitedOpIDs []string
	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOpIDs = append(visitedOpIDs, op.OperationID)
			return Stop // Stop after first additional operation
		}),
	)

	require.NoError(t, err)
	// Only one operation should be visited due to Stop
	assert.Len(t, visitedOpIDs, 1)
}

func TestWalk_OAS3StopDuringOperationLoop(t *testing.T) {
	// Test that Stop during the operation loop (w.stopped check) prevents visiting remaining methods
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/resource": &parser.PathItem{
				Get:    &parser.Operation{OperationID: "getOp"},
				Put:    &parser.Operation{OperationID: "putOp"},
				Post:   &parser.Operation{OperationID: "postOp"},
				Delete: &parser.Operation{OperationID: "deleteOp"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var visitedMethods []string
	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedMethods = append(visitedMethods, wc.Method)
			// Stop after the first operation to exercise the w.stopped check in the loop
			return Stop
		}),
	)

	require.NoError(t, err)
	// Only one method should be visited due to Stop
	assert.Len(t, visitedMethods, 1)
}
