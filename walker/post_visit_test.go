package walker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

func TestPostVisit_SchemaOrder(t *testing.T) {
	// Create a schema with nested properties
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var events []string

	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			events = append(events, "pre:"+wc.JSONPath)
			return Continue
		}),
		WithSchemaPostHandler(func(wc *WalkContext, schema *parser.Schema) {
			events = append(events, "post:"+wc.JSONPath)
		}),
	)

	require.NoError(t, err, "Walk failed")

	// Expected order: pre Pet, pre name, post name, post Pet
	expected := []string{
		"pre:$.components.schemas['Pet']",
		"pre:$.components.schemas['Pet'].properties['name']",
		"post:$.components.schemas['Pet'].properties['name']",
		"post:$.components.schemas['Pet']",
	}

	assert.Equal(t, expected, events)
}

func TestPostVisit_SkipChildrenSkipsPost(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var postCalled bool

	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			// Skip children for Pet schema
			if wc.Name == "Pet" {
				return SkipChildren
			}
			return Continue
		}),
		WithSchemaPostHandler(func(wc *WalkContext, schema *parser.Schema) {
			postCalled = true
		}),
	)

	require.NoError(t, err, "Walk failed")
	assert.False(t, postCalled, "Post handler should not be called when SkipChildren is returned")
}

func TestPostVisit_StopPreventsPost(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var postCalled bool

	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			// Stop at Pet schema
			if wc.Name == "Pet" {
				return Stop
			}
			return Continue
		}),
		WithSchemaPostHandler(func(wc *WalkContext, schema *parser.Schema) {
			postCalled = true
		}),
	)

	require.NoError(t, err, "Walk failed")
	assert.False(t, postCalled, "Post handler should not be called when Stop is returned")
}

func TestPostVisit_NestedSchemas(t *testing.T) {
	// Create a deeply nested schema structure
	// A -> B -> C (A contains B which contains C)
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"A": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"b": {
							Type: "object",
							Properties: map[string]*parser.Schema{
								"c": {Type: "string"},
							},
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

	var postOrder []string

	err := Walk(result,
		WithSchemaPostHandler(func(wc *WalkContext, schema *parser.Schema) {
			// Record the type of each schema to track order
			if t, ok := schema.Type.(string); ok {
				postOrder = append(postOrder, t)
			}
		}),
	)

	require.NoError(t, err, "Walk failed")

	// Post-visit order should be inner to outer: c (string), b (object), A (object)
	expected := []string{"string", "object", "object"}
	assert.Equal(t, expected, postOrder)
}

func TestPostVisit_AllTypes(t *testing.T) {
	// Create a document with all types that have post handlers
	callback := parser.Callback{
		"{$request.body#/callbackUrl}": &parser.PathItem{
			Post: &parser.Operation{
				OperationID: "callbackOp",
			},
		},
	}

	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPets",
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{Type: "object"},
							},
						},
					},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "Success",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Type: "array"},
									},
								},
							},
						},
					},
					Callbacks: map[string]*parser.Callback{
						"onEvent": &callback,
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var called struct {
		schema      bool
		operation   bool
		pathItem    bool
		response    bool
		requestBody bool
		callback    bool
	}

	err := Walk(result,
		WithSchemaPostHandler(func(wc *WalkContext, schema *parser.Schema) {
			called.schema = true
		}),
		WithOperationPostHandler(func(wc *WalkContext, op *parser.Operation) {
			called.operation = true
		}),
		WithPathItemPostHandler(func(wc *WalkContext, pathItem *parser.PathItem) {
			called.pathItem = true
		}),
		WithResponsePostHandler(func(wc *WalkContext, resp *parser.Response) {
			called.response = true
		}),
		WithRequestBodyPostHandler(func(wc *WalkContext, reqBody *parser.RequestBody) {
			called.requestBody = true
		}),
		WithCallbackPostHandler(func(wc *WalkContext, cb parser.Callback) {
			called.callback = true
		}),
	)

	require.NoError(t, err, "Walk failed")

	assert.True(t, called.schema, "SchemaPostHandler was not called")
	assert.True(t, called.operation, "OperationPostHandler was not called")
	assert.True(t, called.pathItem, "PathItemPostHandler was not called")
	assert.True(t, called.response, "ResponsePostHandler was not called")
	assert.True(t, called.requestBody, "RequestBodyPostHandler was not called")
	assert.True(t, called.callback, "CallbackPostHandler was not called")
}

func TestPostVisit_Aggregation(t *testing.T) {
	// Use case: count children (property count) after visiting
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name":   {Type: "string"},
						"age":    {Type: "integer"},
						"status": {Type: "string"},
					},
				},
				"Empty": {
					Type: "object",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	propertyCounts := make(map[string]int)

	err := Walk(result,
		WithSchemaPostHandler(func(wc *WalkContext, schema *parser.Schema) {
			// Only track component schemas by name
			if wc.IsComponent && wc.Name != "" {
				propertyCounts[wc.Name] = len(schema.Properties)
			}
		}),
	)

	require.NoError(t, err, "Walk failed")

	assert.Equal(t, 3, propertyCounts["Pet"], "Pet should have 3 properties")
	assert.Equal(t, 0, propertyCounts["Empty"], "Empty should have 0 properties")
}

func TestPostVisit_OAS2(t *testing.T) {
	// Test post handlers work with OAS 2.0 documents
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "Success",
								Schema:      &parser.Schema{Type: "array"},
							},
						},
					},
				},
			},
		},
		Definitions: map[string]*parser.Schema{
			"Pet": {Type: "object"},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
	}

	var called struct {
		schema    bool
		operation bool
		pathItem  bool
		response  bool
	}

	err := Walk(result,
		WithSchemaPostHandler(func(wc *WalkContext, schema *parser.Schema) {
			called.schema = true
		}),
		WithOperationPostHandler(func(wc *WalkContext, op *parser.Operation) {
			called.operation = true
		}),
		WithPathItemPostHandler(func(wc *WalkContext, pathItem *parser.PathItem) {
			called.pathItem = true
		}),
		WithResponsePostHandler(func(wc *WalkContext, resp *parser.Response) {
			called.response = true
		}),
	)

	require.NoError(t, err, "Walk failed")

	assert.True(t, called.schema, "SchemaPostHandler was not called for OAS 2.0")
	assert.True(t, called.operation, "OperationPostHandler was not called for OAS 2.0")
	assert.True(t, called.pathItem, "PathItemPostHandler was not called for OAS 2.0")
	assert.True(t, called.response, "ResponsePostHandler was not called for OAS 2.0")
}

func TestPostVisit_StopDuringChildren(t *testing.T) {
	// If Stop is returned while walking children, post handler should not be called
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var parentPostCalled bool

	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			// Stop when we reach the nested "name" property
			if wc.JSONPath == "$.components.schemas['Pet'].properties['name']" {
				return Stop
			}
			return Continue
		}),
		WithSchemaPostHandler(func(wc *WalkContext, schema *parser.Schema) {
			// This should only be called for schemas whose children completed
			if wc.JSONPath == "$.components.schemas['Pet']" {
				parentPostCalled = true
			}
		}),
	)

	require.NoError(t, err, "Walk failed")
	assert.False(t, parentPostCalled, "Parent post handler should not be called when child returns Stop")
}

func TestPostVisit_PreAndPostWithContext(t *testing.T) {
	// Verify WalkContext fields are correct in both pre and post handlers
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "Success",
							},
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

	var preContext, postContext *WalkContext

	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			preContext = &WalkContext{
				JSONPath:     wc.JSONPath,
				PathTemplate: wc.PathTemplate,
				Method:       wc.Method,
			}
			return Continue
		}),
		WithOperationPostHandler(func(wc *WalkContext, op *parser.Operation) {
			postContext = &WalkContext{
				JSONPath:     wc.JSONPath,
				PathTemplate: wc.PathTemplate,
				Method:       wc.Method,
			}
		}),
	)

	require.NoError(t, err, "Walk failed")

	require.NotNil(t, preContext, "Pre handler should be called")
	require.NotNil(t, postContext, "Post handler should be called")

	// Context should be the same in both
	assert.Equal(t, preContext.JSONPath, postContext.JSONPath, "JSONPath mismatch")
	assert.Equal(t, preContext.PathTemplate, postContext.PathTemplate, "PathTemplate mismatch")
	assert.Equal(t, preContext.Method, postContext.Method, "Method mismatch")
}

func TestPostVisit_OnlyPostHandler(t *testing.T) {
	// Post handler can be registered without pre handler
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var postCalled bool

	err := Walk(result,
		// No pre-handler, only post handler
		WithSchemaPostHandler(func(wc *WalkContext, schema *parser.Schema) {
			postCalled = true
		}),
	)

	require.NoError(t, err, "Walk failed")
	assert.True(t, postCalled, "Post handler should be called even without pre handler")
}

func BenchmarkWalk_WithPostHandler(b *testing.B) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Benchmark API", Version: "1.0.0"},
		Paths:   make(parser.Paths),
	}

	// Create 50 paths with 3 operations each
	for i := range 50 {
		path := &parser.PathItem{
			Get: &parser.Operation{
				OperationID: "op" + string(rune(i)),
				Responses: &parser.Responses{
					Codes: map[string]*parser.Response{
						"200": {
							Description: "Success",
							Content: map[string]*parser.MediaType{
								"application/json": {
									Schema: &parser.Schema{Type: "object"},
								},
							},
						},
					},
				},
			},
			Post: &parser.Operation{
				OperationID: "create" + string(rune(i)),
				RequestBody: &parser.RequestBody{
					Content: map[string]*parser.MediaType{
						"application/json": {
							Schema: &parser.Schema{Type: "object"},
						},
					},
				},
			},
		}
		doc.Paths["/path"+string(rune(i))] = path
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var count int

	for b.Loop() {
		count = 0
		_ = Walk(result,
			WithSchemaPostHandler(func(wc *WalkContext, schema *parser.Schema) {
				count++
			}),
			WithOperationPostHandler(func(wc *WalkContext, op *parser.Operation) {
				count++
			}),
		)
	}
}

// TestPostVisit_OAS3DocumentPostHandler tests that the OAS3 document post handler
// is called after all children have been processed.
func TestPostVisit_OAS3DocumentPostHandler(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPets",
				},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var events []string

	err := Walk(result,
		WithOAS3DocumentHandler(func(wc *WalkContext, doc *parser.OAS3Document) Action {
			events = append(events, "pre:document")
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			events = append(events, "visit:operation:"+op.OperationID)
			return Continue
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			events = append(events, "visit:schema:"+wc.Name)
			return Continue
		}),
		WithOAS3DocumentPostHandler(func(wc *WalkContext, doc *parser.OAS3Document) {
			events = append(events, "post:document")
		}),
	)

	require.NoError(t, err, "Walk failed")

	// Document post should be called after all other visits
	require.GreaterOrEqual(t, len(events), 2, "Expected at least 2 events")

	// Pre-document should be first
	assert.Equal(t, "pre:document", events[0], "First event should be pre:document")

	// Post-document should be last
	assert.Equal(t, "post:document", events[len(events)-1], "Last event should be post:document")
}

// TestPostVisit_OAS2DocumentPostHandler tests that the OAS2 document post handler
// is called after all children have been processed.
func TestPostVisit_OAS2DocumentPostHandler(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPets",
				},
			},
		},
		Definitions: map[string]*parser.Schema{
			"Pet": {Type: "object"},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
	}

	var events []string

	err := Walk(result,
		WithOAS2DocumentHandler(func(wc *WalkContext, doc *parser.OAS2Document) Action {
			events = append(events, "pre:document")
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			events = append(events, "visit:operation:"+op.OperationID)
			return Continue
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			events = append(events, "visit:schema:"+wc.Name)
			return Continue
		}),
		WithOAS2DocumentPostHandler(func(wc *WalkContext, doc *parser.OAS2Document) {
			events = append(events, "post:document")
		}),
	)

	require.NoError(t, err, "Walk failed")

	// Document post should be called after all other visits
	require.GreaterOrEqual(t, len(events), 2, "Expected at least 2 events")

	// Pre-document should be first
	assert.Equal(t, "pre:document", events[0], "First event should be pre:document")

	// Post-document should be last
	assert.Equal(t, "post:document", events[len(events)-1], "Last event should be post:document")
}

// TestPostVisit_DocumentPostSkipChildren tests that document post handlers are NOT called
// when the document pre-handler returns SkipChildren.
func TestPostVisit_DocumentPostSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPets",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var postCalled bool

	err := Walk(result,
		WithOAS3DocumentHandler(func(wc *WalkContext, doc *parser.OAS3Document) Action {
			return SkipChildren
		}),
		WithOAS3DocumentPostHandler(func(wc *WalkContext, doc *parser.OAS3Document) {
			postCalled = true
		}),
	)

	require.NoError(t, err, "Walk failed")
	assert.False(t, postCalled, "Document post handler should NOT be called when SkipChildren is returned")
}

// TestPostVisit_DocumentPostStop tests that document post handlers are NOT called
// when Stop is returned during child traversal.
func TestPostVisit_DocumentPostStop(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPets",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var postCalled bool

	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			return Stop // Stop during child traversal
		}),
		WithOAS3DocumentPostHandler(func(wc *WalkContext, doc *parser.OAS3Document) {
			postCalled = true
		}),
	)

	require.NoError(t, err, "Walk failed")
	assert.False(t, postCalled, "Document post handler should NOT be called when Stop is returned during children")
}

// TestPostVisit_DocumentPostAggregation demonstrates the primary use case:
// collecting information from children and using it in the post handler.
func TestPostVisit_DocumentPostAggregation(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPets",
					Tags:        []string{"pets"},
				},
				Post: &parser.Operation{
					OperationID: "createPet",
					Tags:        []string{"pets"},
				},
			},
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getUsers",
					Tags:        []string{"users"},
				},
			},
		},
		Components: &parser.Components{
			SecuritySchemes: make(map[string]*parser.SecurityScheme),
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	// Simulate the use case from the issue: collect info from operations,
	// then modify document in post handler
	var operationCount int
	usedTags := make(map[string]bool)

	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			operationCount++
			for _, tag := range op.Tags {
				usedTags[tag] = true
			}
			return Continue
		}),
		WithOAS3DocumentPostHandler(func(wc *WalkContext, doc *parser.OAS3Document) {
			// In post handler, we have complete information from children
			// For this test, we'll add security scheme based on operation count
			if operationCount > 2 {
				doc.Components.SecuritySchemes["api_key"] = &parser.SecurityScheme{
					Type: "apiKey",
					In:   "header",
					Name: "X-API-Key",
				}
			}
		}),
	)

	require.NoError(t, err, "Walk failed")

	// Verify operation count is correct
	assert.Equal(t, 3, operationCount, "Expected 3 operations")

	// Verify tags were collected
	assert.True(t, usedTags["pets"], "Expected pets tag")
	assert.True(t, usedTags["users"], "Expected users tag")

	// Verify document was modified in post handler
	assert.NotNil(t, doc.Components.SecuritySchemes["api_key"], "Expected api_key security scheme to be added in post handler")
}

// TestPostVisit_DocumentPostHandlerOnlyPost tests that document post handler
// can be registered without a pre-handler.
func TestPostVisit_DocumentPostHandlerOnlyPost(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var postCalled bool

	err := Walk(result,
		// No pre-handler, only post handler
		WithOAS3DocumentPostHandler(func(wc *WalkContext, doc *parser.OAS3Document) {
			postCalled = true
			// Verify WalkContext is properly set
			assert.Equal(t, "$", wc.JSONPath)
		}),
	)

	require.NoError(t, err, "Walk failed")
	assert.True(t, postCalled, "Document post handler should be called even without pre handler")
}

// TestPostVisit_DocumentPostWithGenericHandler tests interaction between
// typed document handlers and generic document handlers.
func TestPostVisit_DocumentPostWithGenericHandler(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var events []string

	err := Walk(result,
		WithDocumentHandler(func(wc *WalkContext, doc any) Action {
			events = append(events, "generic:pre")
			return Continue
		}),
		WithOAS3DocumentHandler(func(wc *WalkContext, doc *parser.OAS3Document) Action {
			events = append(events, "typed:pre")
			return Continue
		}),
		WithOAS3DocumentPostHandler(func(wc *WalkContext, doc *parser.OAS3Document) {
			events = append(events, "typed:post")
		}),
	)

	require.NoError(t, err, "Walk failed")

	// Order: typed pre, generic pre, typed post
	expected := []string{"typed:pre", "generic:pre", "typed:post"}
	assert.Equal(t, expected, events)
}

// TestPostVisit_OAS2DocumentPostSkipChildren tests that OAS2 document post handlers
// are NOT called when the document pre-handler returns SkipChildren.
func TestPostVisit_OAS2DocumentPostSkipChildren(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPets",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
	}

	var postCalled bool

	err := Walk(result,
		WithOAS2DocumentHandler(func(wc *WalkContext, doc *parser.OAS2Document) Action {
			return SkipChildren
		}),
		WithOAS2DocumentPostHandler(func(wc *WalkContext, doc *parser.OAS2Document) {
			postCalled = true
		}),
	)

	require.NoError(t, err, "Walk failed")
	assert.False(t, postCalled, "OAS2 document post handler should NOT be called when SkipChildren is returned")
}

// TestPostVisit_OAS2DocumentPostStop tests that OAS2 document post handlers are NOT called
// when Stop is returned during child traversal.
func TestPostVisit_OAS2DocumentPostStop(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPets",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
	}

	var postCalled bool

	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			return Stop // Stop during child traversal
		}),
		WithOAS2DocumentPostHandler(func(wc *WalkContext, doc *parser.OAS2Document) {
			postCalled = true
		}),
	)

	require.NoError(t, err, "Walk failed")
	assert.False(t, postCalled, "OAS2 document post handler should NOT be called when Stop is returned during children")
}
