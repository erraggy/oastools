package walker

import (
	"context"
	"testing"

	"github.com/erraggy/oastools/parser"
)

// TestContextPool_FieldsCleared verifies that WalkContext fields are properly
// cleared when returned to the pool, preventing data leakage between walks.
func TestContextPool_FieldsCleared(t *testing.T) {
	// Create a document that will populate all context fields
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/test": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "testOp",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "OK"},
						},
					},
				},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"TestSchema": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	// First walk to populate contexts
	var firstWalkContexts []WalkContext
	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, _ *parser.Operation) Action {
			// Copy fields, not the pointer (which will be reused)
			firstWalkContexts = append(firstWalkContexts, WalkContext{
				JSONPath:     wc.JSONPath,
				PathTemplate: wc.PathTemplate,
				Method:       wc.Method,
				StatusCode:   wc.StatusCode,
				Name:         wc.Name,
				IsComponent:  wc.IsComponent,
			})
			return Continue
		}),
		WithResponseHandler(func(wc *WalkContext, _ *parser.Response) Action {
			firstWalkContexts = append(firstWalkContexts, WalkContext{
				JSONPath:     wc.JSONPath,
				PathTemplate: wc.PathTemplate,
				Method:       wc.Method,
				StatusCode:   wc.StatusCode,
				Name:         wc.Name,
				IsComponent:  wc.IsComponent,
			})
			return Continue
		}),
		WithSchemaHandler(func(wc *WalkContext, _ *parser.Schema) Action {
			firstWalkContexts = append(firstWalkContexts, WalkContext{
				JSONPath:     wc.JSONPath,
				PathTemplate: wc.PathTemplate,
				Method:       wc.Method,
				StatusCode:   wc.StatusCode,
				Name:         wc.Name,
				IsComponent:  wc.IsComponent,
			})
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("first walk failed: %v", err)
	}

	// Verify we captured some contexts
	if len(firstWalkContexts) == 0 {
		t.Fatal("no contexts captured from first walk")
	}

	// Second walk - verify the contexts have correct values (not leaked from first walk)
	// Use a simple document that should have different context values
	simpleDoc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Simple", Version: "2.0.0"},
		Paths: parser.Paths{
			"/other": &parser.PathItem{
				Post: &parser.Operation{
					OperationID: "otherOp",
				},
			},
		},
	}

	simpleResult := &parser.ParseResult{
		Document:   simpleDoc,
		OASVersion: parser.OASVersion303,
	}

	var secondWalkContexts []WalkContext
	err = Walk(simpleResult,
		WithOperationHandler(func(wc *WalkContext, _ *parser.Operation) Action {
			secondWalkContexts = append(secondWalkContexts, WalkContext{
				JSONPath:     wc.JSONPath,
				PathTemplate: wc.PathTemplate,
				Method:       wc.Method,
				StatusCode:   wc.StatusCode,
				Name:         wc.Name,
				IsComponent:  wc.IsComponent,
			})
			return Continue
		}),
	)
	if err != nil {
		t.Fatalf("second walk failed: %v", err)
	}

	// Verify second walk has correct values
	if len(secondWalkContexts) == 0 {
		t.Fatal("no contexts captured from second walk")
	}

	for _, ctx := range secondWalkContexts {
		// Should have /other path, not /test from first walk
		if ctx.PathTemplate == "/test" {
			t.Error("context leaked /test PathTemplate from first walk")
		}
		// Should have post method, not get from first walk
		if ctx.Method == "get" {
			t.Error("context leaked 'get' Method from first walk")
		}
		// Verify expected values
		if ctx.PathTemplate != "/other" {
			t.Errorf("expected PathTemplate /other, got %s", ctx.PathTemplate)
		}
		if ctx.Method != "post" {
			t.Errorf("expected Method post, got %s", ctx.Method)
		}
	}
}

// TestContextPool_NoDataLeakageBetweenWalks performs multiple walks to verify
// that pooled contexts don't leak data between independent walks.
func TestContextPool_NoDataLeakageBetweenWalks(t *testing.T) {
	// Run many iterations to increase chances of reusing pooled contexts
	for i := range 100 {
		doc := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
			Components: &parser.Components{
				Schemas: map[string]*parser.Schema{
					"Schema": {
						Type: "object",
						Properties: map[string]*parser.Schema{
							"prop": {Type: "string"},
						},
					},
				},
			},
		}

		result := &parser.ParseResult{
			Document:   doc,
			OASVersion: parser.OASVersion303,
		}

		var capturedNames []string
		err := Walk(result,
			WithSchemaHandler(func(wc *WalkContext, _ *parser.Schema) Action {
				capturedNames = append(capturedNames, wc.Name)
				return Continue
			}),
		)
		if err != nil {
			t.Fatalf("iteration %d: walk failed: %v", i, err)
		}

		// Verify the captured names are correct for this walk
		// Root schema should have name "Schema", nested prop should have name "prop"
		expectedNames := []string{"Schema", "prop"}
		if len(capturedNames) != len(expectedNames) {
			t.Fatalf("iteration %d: expected %d names, got %d: %v",
				i, len(expectedNames), len(capturedNames), capturedNames)
		}
		for j, name := range capturedNames {
			if name != expectedNames[j] {
				t.Errorf("iteration %d: name[%d] = %q, want %q",
					i, j, name, expectedNames[j])
			}
		}
	}
}

// TestContextPool_ConcurrentWalks verifies that pooling is safe for concurrent use.
func TestContextPool_ConcurrentWalks(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/test": &parser.PathItem{
				Get: &parser.Operation{OperationID: "testOp"},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Test": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	// Run concurrent walks
	done := make(chan bool, 10)
	for range 10 {
		go func() {
			for range 100 {
				err := Walk(result,
					WithSchemaHandler(func(wc *WalkContext, _ *parser.Schema) Action {
						// Access all fields to catch any data races
						_ = wc.JSONPath
						_ = wc.PathTemplate
						_ = wc.Method
						_ = wc.StatusCode
						_ = wc.Name
						_ = wc.IsComponent
						return Continue
					}),
					WithOperationHandler(func(wc *WalkContext, _ *parser.Operation) Action {
						_ = wc.JSONPath
						_ = wc.PathTemplate
						_ = wc.Method
						return Continue
					}),
				)
				if err != nil {
					t.Errorf("concurrent walk failed: %v", err)
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}
}

// BenchmarkWalk_WithPooling measures allocations with context pooling.
// Compare with the baseline benchmarks to see allocation reduction.
func BenchmarkWalk_WithPooling(b *testing.B) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "OK"},
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

	for b.Loop() {
		_ = Walk(result,
			WithOperationHandler(func(wc *WalkContext, _ *parser.Operation) Action {
				return Continue
			}),
			WithSchemaHandler(func(wc *WalkContext, _ *parser.Schema) Action {
				return Continue
			}),
			WithResponseHandler(func(wc *WalkContext, _ *parser.Response) Action {
				return Continue
			}),
		)
	}
}

// TestWalkContext_WithContext verifies that WalkContext.WithContext creates a
// copy with the new context while preserving all other fields.
func TestWalkContext_WithContext(t *testing.T) {
	wc := &WalkContext{
		JSONPath:     "$.test",
		PathTemplate: "/pets",
		Method:       "get",
		StatusCode:   "200",
		Name:         "TestSchema",
		IsComponent:  true,
	}

	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("testKey"), "testValue")
	wc2 := wc.WithContext(ctx)

	// Should be a different instance
	if wc == wc2 {
		t.Error("WithContext should return a new instance")
	}

	// Should copy all fields
	if wc.JSONPath != wc2.JSONPath {
		t.Errorf("JSONPath mismatch: got %s, want %s", wc2.JSONPath, wc.JSONPath)
	}
	if wc.PathTemplate != wc2.PathTemplate {
		t.Errorf("PathTemplate mismatch: got %s, want %s", wc2.PathTemplate, wc.PathTemplate)
	}

	// Should have new context
	if wc2.Context() != ctx {
		t.Error("new WalkContext should have the provided context")
	}
	if wc2.Context().Value(ctxKey("testKey")) != "testValue" {
		t.Error("context value not preserved")
	}
}

// TestWithContext_Propagation verifies that WithContext option propagates
// the context to handlers via WalkContext.Context().
func TestWithContext_Propagation(t *testing.T) {
	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("testKey"), "testValue")

	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Test": {Type: "string"},
			},
		},
	}
	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var receivedCtx context.Context
	err := Walk(result,
		WithContext(ctx),
		WithSchemaHandler(func(wc *WalkContext, _ *parser.Schema) Action {
			receivedCtx = wc.Context()
			return Continue
		}),
	)

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if receivedCtx != ctx {
		t.Error("handler did not receive the provided context")
	}
}

// TestWithContext_Cancellation verifies cancelled context is accessible in handlers.
func TestWithContext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Test": {Type: "string"},
			},
		},
	}
	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var ctxErr error
	err := Walk(result,
		WithContext(ctx),
		WithSchemaHandler(func(wc *WalkContext, _ *parser.Schema) Action {
			ctxErr = wc.Context().Err()
			return Continue
		}),
	)

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if ctxErr != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", ctxErr)
	}
}

// TestWalkContext_Context_NilReturnsBackground verifies Context() returns
// context.Background() when no context is set.
func TestWalkContext_Context_NilReturnsBackground(t *testing.T) {
	wc := &WalkContext{JSONPath: "$.test"}

	ctx := wc.Context()
	if ctx == nil {
		t.Error("Context() should not return nil")
	}
	if ctx != context.Background() {
		t.Error("Context() should return context.Background() when no context is set")
	}
}
