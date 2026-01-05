package walker

import (
	"testing"

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

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// Expected order: pre Pet, pre name, post name, post Pet
	expected := []string{
		"pre:$.components.schemas['Pet']",
		"pre:$.components.schemas['Pet'].properties['name']",
		"post:$.components.schemas['Pet'].properties['name']",
		"post:$.components.schemas['Pet']",
	}

	if len(events) != len(expected) {
		t.Fatalf("Expected %d events, got %d: %v", len(expected), len(events), events)
	}

	for i, exp := range expected {
		if events[i] != exp {
			t.Errorf("Event %d: expected %q, got %q", i, exp, events[i])
		}
	}
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

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if postCalled {
		t.Error("Post handler should not be called when SkipChildren is returned")
	}
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

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if postCalled {
		t.Error("Post handler should not be called when Stop is returned")
	}
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

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// Post-visit order should be inner to outer: c (string), b (object), A (object)
	expected := []string{"string", "object", "object"}
	if len(postOrder) != len(expected) {
		t.Fatalf("Expected %d post calls, got %d: %v", len(expected), len(postOrder), postOrder)
	}

	for i, exp := range expected {
		if postOrder[i] != exp {
			t.Errorf("Post order %d: expected %q, got %q", i, exp, postOrder[i])
		}
	}
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

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if !called.schema {
		t.Error("SchemaPostHandler was not called")
	}
	if !called.operation {
		t.Error("OperationPostHandler was not called")
	}
	if !called.pathItem {
		t.Error("PathItemPostHandler was not called")
	}
	if !called.response {
		t.Error("ResponsePostHandler was not called")
	}
	if !called.requestBody {
		t.Error("RequestBodyPostHandler was not called")
	}
	if !called.callback {
		t.Error("CallbackPostHandler was not called")
	}
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

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if propertyCounts["Pet"] != 3 {
		t.Errorf("Pet should have 3 properties, got %d", propertyCounts["Pet"])
	}
	if propertyCounts["Empty"] != 0 {
		t.Errorf("Empty should have 0 properties, got %d", propertyCounts["Empty"])
	}
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

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if !called.schema {
		t.Error("SchemaPostHandler was not called for OAS 2.0")
	}
	if !called.operation {
		t.Error("OperationPostHandler was not called for OAS 2.0")
	}
	if !called.pathItem {
		t.Error("PathItemPostHandler was not called for OAS 2.0")
	}
	if !called.response {
		t.Error("ResponsePostHandler was not called for OAS 2.0")
	}
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

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if parentPostCalled {
		t.Error("Parent post handler should not be called when child returns Stop")
	}
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

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if preContext == nil || postContext == nil {
		t.Fatal("Both pre and post handlers should be called")
	}

	// Context should be the same in both
	if preContext.JSONPath != postContext.JSONPath {
		t.Errorf("JSONPath mismatch: pre=%q, post=%q", preContext.JSONPath, postContext.JSONPath)
	}
	if preContext.PathTemplate != postContext.PathTemplate {
		t.Errorf("PathTemplate mismatch: pre=%q, post=%q", preContext.PathTemplate, postContext.PathTemplate)
	}
	if preContext.Method != postContext.Method {
		t.Errorf("Method mismatch: pre=%q, post=%q", preContext.Method, postContext.Method)
	}
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

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if !postCalled {
		t.Error("Post handler should be called even without pre handler")
	}
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
