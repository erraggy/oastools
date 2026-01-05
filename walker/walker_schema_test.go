// walker_schema_test.go - Tests for schema traversal
// Tests nested schemas, circular references, depth limits, schema composition,
// and various JSON Schema keywords.

package walker

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
