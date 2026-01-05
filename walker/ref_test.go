package walker

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefTracking_Disabled(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {Ref: "#/components/schemas/Animal"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	// When ref tracking is disabled, CurrentRef should be nil
	var currentRefSeen *RefInfo
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			currentRefSeen = wc.CurrentRef
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Nil(t, currentRefSeen, "CurrentRef should be nil when ref tracking is disabled")
}

func TestRefTracking_SchemaRef(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {Ref: "#/components/schemas/Animal"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "#/components/schemas/Animal", refs[0].Ref)
	assert.Equal(t, "$.components.schemas['Pet']", refs[0].SourcePath)
	assert.Equal(t, "schema", refs[0].NodeType)
}

func TestRefTracking_ParameterRef(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPets",
					Parameters: []*parser.Parameter{
						{Ref: "#/components/parameters/LimitParam"},
					},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "OK"},
						},
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
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "#/components/parameters/LimitParam", refs[0].Ref)
	assert.Equal(t, "parameter", refs[0].NodeType)
}

func TestRefTracking_ResponseRef(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPets",
					Responses: &parser.Responses{
						Default: &parser.Response{Ref: "#/components/responses/Error"},
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
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "#/components/responses/Error", refs[0].Ref)
	assert.Equal(t, "response", refs[0].NodeType)
}

func TestRefTracking_AllRefs(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Ref: "#/components/pathItems/PetsPath",
				Get: &parser.Operation{
					OperationID: "getPets",
					Parameters: []*parser.Parameter{
						{Ref: "#/components/parameters/LimitParam"},
					},
					RequestBody: &parser.RequestBody{
						Ref: "#/components/requestBodies/PetBody",
					},
					Responses: &parser.Responses{
						Default: &parser.Response{Ref: "#/components/responses/Error"},
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Headers: map[string]*parser.Header{
									"X-Rate-Limit": {Ref: "#/components/headers/RateLimit"},
								},
								Links: map[string]*parser.Link{
									"next": {Ref: "#/components/links/NextPage"},
								},
							},
						},
					},
				},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet":    {Ref: "#/components/schemas/Animal"},
				"Animal": {Type: "object"},
			},
			Examples: map[string]*parser.Example{
				"PetExample": {Ref: "#/components/examples/AnimalExample"},
			},
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"api_key": {Ref: "#/components/securitySchemes/oauth"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)

	// Collect ref values and node types
	refValues := make(map[string]string)
	for _, r := range refs {
		refValues[r.Ref] = r.NodeType
	}

	// Verify all expected refs were found
	assert.Equal(t, "pathItem", refValues["#/components/pathItems/PetsPath"])
	assert.Equal(t, "parameter", refValues["#/components/parameters/LimitParam"])
	assert.Equal(t, "requestBody", refValues["#/components/requestBodies/PetBody"])
	assert.Equal(t, "response", refValues["#/components/responses/Error"])
	assert.Equal(t, "header", refValues["#/components/headers/RateLimit"])
	assert.Equal(t, "link", refValues["#/components/links/NextPage"])
	assert.Equal(t, "schema", refValues["#/components/schemas/Animal"])
	assert.Equal(t, "example", refValues["#/components/examples/AnimalExample"])
	assert.Equal(t, "securityScheme", refValues["#/components/securitySchemes/oauth"])
}

func TestRefTracking_RefHandler(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"owner": {Ref: "#/components/schemas/User"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var handlerCalled bool
	var receivedContext *WalkContext
	var receivedRef *RefInfo

	err := Walk(result,
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			handlerCalled = true
			receivedContext = wc
			receivedRef = ref
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, "#/components/schemas/User", receivedRef.Ref)
	assert.Equal(t, "$.components.schemas['Pet'].properties['owner']", receivedContext.JSONPath)
	assert.NotNil(t, receivedContext.CurrentRef)
	assert.Equal(t, receivedRef, receivedContext.CurrentRef)
}

func TestRefTracking_StopOnRef(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"A": {Ref: "#/components/schemas/X"},
				"B": {Ref: "#/components/schemas/Y"},
				"C": {Ref: "#/components/schemas/Z"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Stop // Stop after first ref
		}),
	)

	require.NoError(t, err)
	assert.Len(t, refs, 1, "Should only process one ref before stopping")
}

func TestRefTracking_OAS2(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Ref: "#/pathItems/PetsPath",
				Get: &parser.Operation{
					OperationID: "getPets",
					Parameters: []*parser.Parameter{
						{Ref: "#/parameters/LimitParam"},
					},
					Responses: &parser.Responses{
						Default: &parser.Response{Ref: "#/responses/Error"},
					},
				},
			},
		},
		Definitions: map[string]*parser.Schema{
			"Pet": {Ref: "#/definitions/Animal"},
		},
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"api_key": {Ref: "#/securityDefinitions/oauth"},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)

	// Collect ref values
	refValues := make(map[string]string)
	for _, r := range refs {
		refValues[r.Ref] = r.NodeType
	}

	assert.Equal(t, "pathItem", refValues["#/pathItems/PetsPath"])
	assert.Equal(t, "parameter", refValues["#/parameters/LimitParam"])
	assert.Equal(t, "response", refValues["#/responses/Error"])
	assert.Equal(t, "schema", refValues["#/definitions/Animal"])
	assert.Equal(t, "securityScheme", refValues["#/securityDefinitions/oauth"])
}

func TestRefTracking_OAS3(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {Ref: "#/components/schemas/Animal"},
			},
			RequestBodies: map[string]*parser.RequestBody{
				"PetBody": {Ref: "#/components/requestBodies/AnimalBody"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var refs []*RefInfo
	err := Walk(result,
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refs = append(refs, ref)
			return Continue
		}),
	)

	require.NoError(t, err)

	// Collect ref values
	refValues := make(map[string]string)
	for _, r := range refs {
		refValues[r.Ref] = r.NodeType
	}

	assert.Equal(t, "schema", refValues["#/components/schemas/Animal"])
	assert.Equal(t, "requestBody", refValues["#/components/requestBodies/AnimalBody"])
}

func TestRefTracking_WithRefTrackingOnly(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {Ref: "#/components/schemas/Animal"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	// WithRefTracking enables tracking but doesn't set a handler
	// CurrentRef should still be nil in schema handler because
	// the ref is processed before the schema handler runs
	err := Walk(result,
		WithRefTracking(),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			// CurrentRef is not set in schema handler - it's set in ref handler
			return Continue
		}),
	)

	require.NoError(t, err)
}

func TestRefTracking_EmptyRef(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {Type: "object"}, // No ref
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var refHandlerCalled bool
	err := Walk(result,
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refHandlerCalled = true
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.False(t, refHandlerCalled, "RefHandler should not be called for empty refs")
}

func BenchmarkWalk_WithRefTracking(b *testing.B) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name":  {Type: "string"},
						"owner": {Ref: "#/components/schemas/User"},
					},
				},
				"User": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name":  {Type: "string"},
						"email": {Type: "string"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	for b.Loop() {
		_ = Walk(result,
			WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
				return Continue
			}),
		)
	}
}

func BenchmarkWalk_WithoutRefTracking(b *testing.B) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name":  {Type: "string"},
						"owner": {Ref: "#/components/schemas/User"},
					},
				},
				"User": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name":  {Type: "string"},
						"email": {Type: "string"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	for b.Loop() {
		_ = Walk(result,
			WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
				return Continue
			}),
		)
	}
}
