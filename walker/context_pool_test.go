package walker

import (
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
