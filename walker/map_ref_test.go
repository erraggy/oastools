package walker

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapRefTracking_Disabled(t *testing.T) {
	// When map ref tracking is disabled, refs in map[string]any should NOT be visited
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "array",
					Items: map[string]any{
						"$ref": "#/components/schemas/Animal",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Empty(t, refs, "Map-stored refs should not be visited when map ref tracking is disabled")
}

func TestMapRefTracking_Items(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "array",
					Items: map[string]any{
						"$ref": "#/components/schemas/Animal",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "#/components/schemas/Animal", refs[0].Ref)
	assert.Equal(t, "$.components.schemas['Pet'].items", refs[0].SourcePath)
	assert.Equal(t, RefNodeSchema, refs[0].NodeType)
}

func TestMapRefTracking_AdditionalItems(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "array",
					AdditionalItems: map[string]any{
						"$ref": "#/components/schemas/ExtraItem",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "#/components/schemas/ExtraItem", refs[0].Ref)
	assert.Equal(t, "$.components.schemas['Pet'].additionalItems", refs[0].SourcePath)
	assert.Equal(t, RefNodeSchema, refs[0].NodeType)
}

func TestMapRefTracking_UnevaluatedItems(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "array",
					UnevaluatedItems: map[string]any{
						"$ref": "#/components/schemas/Unevaluated",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "#/components/schemas/Unevaluated", refs[0].Ref)
	assert.Equal(t, "$.components.schemas['Pet'].unevaluatedItems", refs[0].SourcePath)
	assert.Equal(t, RefNodeSchema, refs[0].NodeType)
}

func TestMapRefTracking_AdditionalProperties(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					AdditionalProperties: map[string]any{
						"$ref": "#/components/schemas/ExtraProperty",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "#/components/schemas/ExtraProperty", refs[0].Ref)
	assert.Equal(t, "$.components.schemas['Pet'].additionalProperties", refs[0].SourcePath)
	assert.Equal(t, RefNodeSchema, refs[0].NodeType)
}

func TestMapRefTracking_UnevaluatedProperties(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					UnevaluatedProperties: map[string]any{
						"$ref": "#/components/schemas/UnevaluatedProp",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "#/components/schemas/UnevaluatedProp", refs[0].Ref)
	assert.Equal(t, "$.components.schemas['Pet'].unevaluatedProperties", refs[0].SourcePath)
	assert.Equal(t, RefNodeSchema, refs[0].NodeType)
}

func TestMapRefTracking_AllFields(t *testing.T) {
	// Test that all polymorphic fields are tracked in a single document
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"ArraySchema": {
					Type: "array",
					Items: map[string]any{
						"$ref": "#/components/schemas/ItemRef",
					},
					AdditionalItems: map[string]any{
						"$ref": "#/components/schemas/AdditionalItemRef",
					},
					UnevaluatedItems: map[string]any{
						"$ref": "#/components/schemas/UnevaluatedItemRef",
					},
				},
				"ObjectSchema": {
					Type: "object",
					AdditionalProperties: map[string]any{
						"$ref": "#/components/schemas/AdditionalPropRef",
					},
					UnevaluatedProperties: map[string]any{
						"$ref": "#/components/schemas/UnevaluatedPropRef",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	require.Len(t, refs, 5)

	// Collect all refs by their path suffix for easier verification
	refsByPath := make(map[string]string)
	for _, r := range refs {
		refsByPath[r.SourcePath] = r.Ref
	}

	// Check array-related refs
	assert.Equal(t, "#/components/schemas/ItemRef", refsByPath["$.components.schemas['ArraySchema'].items"])
	assert.Equal(t, "#/components/schemas/AdditionalItemRef", refsByPath["$.components.schemas['ArraySchema'].additionalItems"])
	assert.Equal(t, "#/components/schemas/UnevaluatedItemRef", refsByPath["$.components.schemas['ArraySchema'].unevaluatedItems"])

	// Check object-related refs
	assert.Equal(t, "#/components/schemas/AdditionalPropRef", refsByPath["$.components.schemas['ObjectSchema'].additionalProperties"])
	assert.Equal(t, "#/components/schemas/UnevaluatedPropRef", refsByPath["$.components.schemas['ObjectSchema'].unevaluatedProperties"])
}

func TestMapRefTracking_MixedSchemaAndMap(t *testing.T) {
	// Test that *Schema refs are still visited alongside map refs
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"MixedSchema": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"regular": {Ref: "#/components/schemas/RegularRef"},
					},
					AdditionalProperties: map[string]any{
						"$ref": "#/components/schemas/MapRef",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	require.Len(t, refs, 2)

	// Collect refs by their target
	refTargets := make(map[string]bool)
	for _, r := range refs {
		refTargets[r.Ref] = true
	}

	assert.True(t, refTargets["#/components/schemas/RegularRef"], "Regular *Schema ref should be visited")
	assert.True(t, refTargets["#/components/schemas/MapRef"], "Map-stored ref should be visited")
}

func TestMapRefTracking_NestedSchema(t *testing.T) {
	// Test that map refs in nested schemas are also tracked
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Parent": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"child": {
							Type: "array",
							Items: map[string]any{
								"$ref": "#/components/schemas/NestedRef",
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "#/components/schemas/NestedRef", refs[0].Ref)
	assert.Equal(t, "$.components.schemas['Parent'].properties['child'].items", refs[0].SourcePath)
}

func TestMapRefTracking_EmptyRef(t *testing.T) {
	// Test that empty $ref values in maps are ignored
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "array",
					Items: map[string]any{
						"$ref": "", // Empty ref
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Empty(t, refs, "Empty refs should not be tracked")
}

func TestMapRefTracking_NoRefKey(t *testing.T) {
	// Test that maps without $ref are ignored
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "array",
					Items: map[string]any{
						"type": "string", // No $ref key
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Empty(t, refs, "Maps without $ref should not trigger ref handler")
}

func TestMapRefTracking_RefNotString(t *testing.T) {
	// Test that $ref values that aren't strings are ignored
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "array",
					Items: map[string]any{
						"$ref": 123, // Non-string ref
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Empty(t, refs, "Non-string refs should not be tracked")
}

func TestMapRefTracking_Stop(t *testing.T) {
	// Test that Stop action works correctly with map refs
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Schema1": {
					Type: "array",
					Items: map[string]any{
						"$ref": "#/components/schemas/First",
					},
				},
				"Schema2": {
					Type: "array",
					Items: map[string]any{
						"$ref": "#/components/schemas/Second",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Stop // Stop after first ref
		}),
	)

	require.NoError(t, err)
	assert.Len(t, refs, 1, "Should stop after first map ref")
}

func TestMapRefTracking_ImplicitlyEnablesRefTracking(t *testing.T) {
	// Test that WithMapRefTracking implicitly enables ref tracking
	w := New()
	WithMapRefTracking()(w)

	assert.True(t, w.trackRefs, "trackRefs should be enabled")
	assert.True(t, w.trackMapRefs, "trackMapRefs should be enabled")
}

func TestMapRefTracking_OAS2(t *testing.T) {
	// Test map ref tracking with OAS 2.0 documents
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Definitions: map[string]*parser.Schema{
			"Pet": {
				Type: "array",
				Items: map[string]any{
					"$ref": "#/definitions/Animal",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "#/definitions/Animal", refs[0].Ref)
	assert.Equal(t, "$.definitions['Pet'].items", refs[0].SourcePath)
}

func TestMapRefTracking_BoolValue(t *testing.T) {
	// Test that bool values (like additionalProperties: false) don't cause issues
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type:                 "object",
					AdditionalProperties: false, // bool value
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Empty(t, refs, "Bool values should not trigger ref handler")
}

func TestMapRefTracking_SkipChildren(t *testing.T) {
	// Test that SkipChildren allows the walk to continue processing other schemas
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"FirstSchema": {
					Type: "array",
					Items: map[string]any{
						"$ref": "#/components/schemas/FirstRef",
					},
				},
				"SecondSchema": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
					},
				},
				"ThirdSchema": {
					Type: "array",
					Items: map[string]any{
						"$ref": "#/components/schemas/ThirdRef",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	var schemaCount int
	err := Walk(result,
		WithMapRefTracking(),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaCount++
			return Continue
		}),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return SkipChildren // Skip children but continue walking
		}),
	)

	require.NoError(t, err)

	// SkipChildren should not prevent other schemas from being visited
	// All three top-level schemas plus the nested property schema should be visited
	assert.GreaterOrEqual(t, schemaCount, 3, "Multiple schemas should be visited despite SkipChildren")

	// Both map refs should be found (SkipChildren doesn't stop sibling processing)
	assert.Len(t, refs, 2, "Both map refs should be visited")
}

func TestMapRefTracking_NilPolymorphicFields(t *testing.T) {
	// Test that explicit nil values in polymorphic fields don't cause panics
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"NilFieldsSchema": {
					Type:                  "object",
					Items:                 nil, // Explicit nil
					AdditionalItems:       nil, // Explicit nil
					UnevaluatedItems:      nil, // Explicit nil
					AdditionalProperties:  nil, // Explicit nil
					UnevaluatedProperties: nil, // Explicit nil
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Empty(t, refs, "Nil polymorphic fields should not produce refs")
}

func TestMapRefTracking_OAS303(t *testing.T) {
	// Test map ref tracking with OAS 3.0.3 documents
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "array",
					Items: map[string]any{
						"$ref": "#/components/schemas/Animal",
					},
				},
				"Container": {
					Type: "object",
					AdditionalProperties: map[string]any{
						"$ref": "#/components/schemas/Value",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	require.Len(t, refs, 2)

	// Collect refs by their target
	refTargets := make(map[string]bool)
	for _, r := range refs {
		refTargets[r.Ref] = true
	}

	assert.True(t, refTargets["#/components/schemas/Animal"], "Items ref should be tracked")
	assert.True(t, refTargets["#/components/schemas/Value"], "AdditionalProperties ref should be tracked")
}

func BenchmarkWalk_WithMapRefTracking(b *testing.B) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
					},
					AdditionalProperties: map[string]any{
						"$ref": "#/components/schemas/Extra",
					},
				},
				"Extra": {
					Type: "string",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	for b.Loop() {
		_ = Walk(result,
			WithMapRefTracking(),
			WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
				return Continue
			}),
		)
	}
}
