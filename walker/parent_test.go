package walker

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

func TestParentTracking_Disabled(t *testing.T) {
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

	var parentWasNil bool
	err := Walk(result,
		// Note: WithParentTracking() is NOT used
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if wc.Parent == nil {
				parentWasNil = true
			}
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if !parentWasNil {
		t.Error("Expected Parent to be nil when parent tracking is disabled")
	}
}

func TestParentTracking_SchemaInOperation(t *testing.T) {
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
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var foundParentOp bool
	var parentOpID string
	err := Walk(result,
		WithParentTracking(),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if op, ok := wc.ParentOperation(); ok {
				foundParentOp = true
				parentOpID = op.OperationID
			}
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if !foundParentOp {
		t.Error("Expected to find parent operation for schema")
	}
	if parentOpID != "getPets" {
		t.Errorf("Expected parent operation ID 'getPets', got '%s'", parentOpID)
	}
}

func TestParentTracking_NestedSchemas(t *testing.T) {
	outerSchema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"inner": {Type: "string"},
		},
	}

	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Outer": outerSchema,
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var foundParentSchema bool
	err := Walk(result,
		WithParentTracking(),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if schemaType, ok := schema.Type.(string); ok && schemaType == "string" {
				// This is the inner schema
				if parent, ok := wc.ParentSchema(); ok {
					if parentType, ok := parent.Type.(string); ok && parentType == "object" {
						foundParentSchema = true
					}
				}
			}
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if !foundParentSchema {
		t.Error("Expected inner schema to find outer schema as parent")
	}
}

func TestParentTracking_Ancestors(t *testing.T) {
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
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var ancestorCount int
	err := Walk(result,
		WithParentTracking(),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			ancestors := wc.Ancestors()
			ancestorCount = len(ancestors)
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	// Ancestors should include: MediaType, RequestBody, Operation, PathItem
	if ancestorCount < 4 {
		t.Errorf("Expected at least 4 ancestors, got %d", ancestorCount)
	}
}

func TestParentTracking_ParentSchema(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"User": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"address": {
							Type: "object",
							Properties: map[string]*parser.Schema{
								"street": {Type: "string"},
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

	schemaParents := make(map[string]string)
	err := Walk(result,
		WithParentTracking(),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if schemaType, ok := schema.Type.(string); ok {
				if parent, ok := wc.ParentSchema(); ok {
					if parentType, ok := parent.Type.(string); ok {
						schemaParents[schemaType+"@"+wc.JSONPath] = parentType
					}
				}
			}
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// The string schema's parent should be the address object schema
	var foundStringWithObjectParent bool
	for key, parentType := range schemaParents {
		if parentType == "object" && key[:6] == "string" {
			foundStringWithObjectParent = true
			break
		}
	}
	if !foundStringWithObjectParent {
		t.Error("Expected string schema to have object parent schema")
	}
}

func TestParentTracking_ParentOperation(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPets",
					Parameters: []*parser.Parameter{
						{Name: "limit", In: "query"},
					},
				},
				Post: &parser.Operation{
					OperationID: "createPet",
					Parameters: []*parser.Parameter{
						{Name: "body", In: "body"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	paramOperations := make(map[string]string)
	err := Walk(result,
		WithParentTracking(),
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			if op, ok := wc.ParentOperation(); ok {
				paramOperations[param.Name] = op.OperationID
			}
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if paramOperations["limit"] != "getPets" {
		t.Errorf("Expected 'limit' param to be in 'getPets', got '%s'", paramOperations["limit"])
	}
	if paramOperations["body"] != "createPet" {
		t.Errorf("Expected 'body' param to be in 'createPet', got '%s'", paramOperations["body"])
	}
}

func TestParentTracking_ParentPathItem(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Summary: "Pet operations",
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

	var foundPathItem bool
	var pathItemSummary string
	err := Walk(result,
		WithParentTracking(),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			if pi, ok := wc.ParentPathItem(); ok {
				foundPathItem = true
				pathItemSummary = pi.Summary
			}
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if !foundPathItem {
		t.Error("Expected to find parent path item for operation")
	}
	if pathItemSummary != "Pet operations" {
		t.Errorf("Expected path item summary 'Pet operations', got '%s'", pathItemSummary)
	}
}

func TestParentTracking_ParentResponse(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "Success response",
								Headers: map[string]*parser.Header{
									"X-Rate-Limit": {Description: "Rate limit"},
								},
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

	var foundResponse bool
	var responseDesc string
	err := Walk(result,
		WithParentTracking(),
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			if resp, ok := wc.ParentResponse(); ok {
				foundResponse = true
				responseDesc = resp.Description
			}
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if !foundResponse {
		t.Error("Expected to find parent response for header")
	}
	if responseDesc != "Success response" {
		t.Errorf("Expected response description 'Success response', got '%s'", responseDesc)
	}
}

func TestParentTracking_ParentRequestBody(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Post: &parser.Operation{
					RequestBody: &parser.RequestBody{
						Description: "Pet to create",
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{Type: "object"},
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

	var foundRequestBody bool
	var reqBodyDesc string
	err := Walk(result,
		WithParentTracking(),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if rb, ok := wc.ParentRequestBody(); ok {
				foundRequestBody = true
				reqBodyDesc = rb.Description
			}
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if !foundRequestBody {
		t.Error("Expected to find parent request body for schema")
	}
	if reqBodyDesc != "Pet to create" {
		t.Errorf("Expected request body description 'Pet to create', got '%s'", reqBodyDesc)
	}
}

func TestParentTracking_Depth(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Level0": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"level1": {
							Type: "object",
							Properties: map[string]*parser.Schema{
								"level2": {Type: "string"},
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

	depths := make(map[string]int)
	err := Walk(result,
		WithParentTracking(),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if schemaType, ok := schema.Type.(string); ok {
				depths[schemaType+"@"+wc.JSONPath] = wc.Depth()
			}
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// Check that depth increases as we go deeper
	var minDepth, maxDepth int
	first := true
	for _, d := range depths {
		if first {
			minDepth = d
			maxDepth = d
			first = false
		} else {
			if d < minDepth {
				minDepth = d
			}
			if d > maxDepth {
				maxDepth = d
			}
		}
	}

	if maxDepth <= minDepth {
		t.Error("Expected depth to increase for nested schemas")
	}
}

func TestParentTracking_OAS2(t *testing.T) {
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
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
	}

	var foundParentOp bool
	err := Walk(result,
		WithParentTracking(),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if _, ok := wc.ParentOperation(); ok {
				foundParentOp = true
			}
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if !foundParentOp {
		t.Error("Expected OAS 2.0 schema to find parent operation")
	}
}

func TestParentTracking_ParentInfoChain(t *testing.T) {
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
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var chainValid bool
	err := Walk(result,
		WithParentTracking(),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			// Verify we can walk up the chain via Parent.Parent
			if wc.Parent != nil {
				current := wc.Parent
				depth := 1
				for current.Parent != nil {
					depth++
					current = current.Parent
				}
				// Should be able to trace back multiple levels
				chainValid = depth >= 3
			}
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if !chainValid {
		t.Error("Expected to be able to traverse parent chain")
	}
}

func TestParentTracking_NoParentAtRoot(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var parentWasNil bool
	err := Walk(result,
		WithParentTracking(),
		WithDocumentHandler(func(wc *WalkContext, doc any) Action {
			parentWasNil = wc.Parent == nil
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if !parentWasNil {
		t.Error("Expected Parent to be nil at document root")
	}
}

func TestParentTracking_WalkWithOptions(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/test": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "test",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Type: "string"},
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
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var foundParent bool
	err := WalkWithOptions(
		WithParsed(result),
		WithParentTracking(),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if wc.Parent != nil {
				foundParent = true
			}
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("WalkWithOptions failed: %v", err)
	}
	if !foundParent {
		t.Error("Expected to find parent when using WalkWithOptions with parent tracking")
	}
}

func BenchmarkWalk_WithParentTracking(b *testing.B) {
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
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{
											Type: "object",
											Properties: map[string]*parser.Schema{
												"id":   {Type: "integer"},
												"name": {Type: "string"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"id":   {Type: "integer"},
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

	b.Run("without_parent_tracking", func(b *testing.B) {
		for b.Loop() {
			_ = Walk(result,
				WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
					return Continue
				}),
			)
		}
	})

	b.Run("with_parent_tracking", func(b *testing.B) {
		for b.Loop() {
			_ = Walk(result,
				WithParentTracking(),
				WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
					_, _ = wc.ParentOperation()
					return Continue
				}),
			)
		}
	})
}
