// walker_context_test.go - Tests for WalkContext features
// Tests IsComponent, scope methods (InPathsScope, InOperationScope, InResponseScope),
// context propagation, and field population.

package walker

import (
	"context"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkContext_IsComponent(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Type: "array"},
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
				"Pet": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	componentSchemas := 0
	inlineSchemas := 0
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if wc.IsComponent {
				componentSchemas++
			} else {
				inlineSchemas++
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Equal(t, 1, componentSchemas, "expected 1 component schema (Pet)")
	assert.Equal(t, 1, inlineSchemas, "expected 1 inline schema (response schema)")
}

func TestWalkContext_ScopeMethods(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Type: "array"},
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
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var inPathsScope, inOperationScope, inResponseScope bool
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			inPathsScope = wc.InPathsScope()
			inOperationScope = wc.InOperationScope()
			inResponseScope = wc.InResponseScope()
			return Continue
		}),
	)
	require.NoError(t, err)

	// Schema in response content should be in paths, operation, and response scope
	assert.True(t, inPathsScope, "schema should be in paths scope")
	assert.True(t, inOperationScope, "schema should be in operation scope")
	assert.True(t, inResponseScope, "schema should be in response scope")
}

func TestWalk_WithContext(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{OperationID: "listPets"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	type ctxKey string
	var capturedValue string
	ctx := context.WithValue(context.Background(), ctxKey("test"), "test-value")

	err := WalkWithOptions(
		WithParsed(result),
		WithContext(ctx),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			if wc.Context() != nil {
				if v, ok := wc.Context().Value(ctxKey("test")).(string); ok {
					capturedValue = v
				}
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Equal(t, "test-value", capturedValue, "context should be propagated to handlers")
}

func TestWalkContext_AllFields(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets/{petId}": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPet",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
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
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var capturedPathTemplate, capturedMethod, capturedStatusCode, capturedHeaderName string
	err := Walk(result,
		WithPathHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			capturedPathTemplate = wc.PathTemplate
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			capturedMethod = wc.Method
			return Continue
		}),
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			capturedStatusCode = wc.StatusCode
			return Continue
		}),
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			capturedHeaderName = wc.Name
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Equal(t, "/pets/{petId}", capturedPathTemplate, "PathTemplate should be set")
	assert.Equal(t, "get", capturedMethod, "Method should be set")
	assert.Equal(t, "200", capturedStatusCode, "StatusCode should be set")
	assert.Equal(t, "X-Rate-Limit", capturedHeaderName, "Name should be set for header")
}
