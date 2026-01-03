package walker

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalk_NilInput(t *testing.T) {
	err := Walk(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil ParseResult")
}

func TestWalk_NilDocument(t *testing.T) {
	result := &parser.ParseResult{}
	err := Walk(result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil Document")
}

func TestWalk_UnsupportedDocumentType(t *testing.T) {
	result := &parser.ParseResult{
		Document: "not a document",
	}
	err := Walk(result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported document type")
}

func TestWalk_OAS3Document(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
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
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var visitedOps []string
	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOps = append(visitedOps, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"listPets"}, visitedOps)
}

func TestWalk_Stop(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/a": &parser.PathItem{Get: &parser.Operation{OperationID: "opA"}},
			"/b": &parser.PathItem{Get: &parser.Operation{OperationID: "opB"}},
			"/c": &parser.PathItem{Get: &parser.Operation{OperationID: "opC"}},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var visitedOps []string
	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOps = append(visitedOps, op.OperationID)
			return Stop // Stop after first
		}),
	)

	require.NoError(t, err)
	assert.Len(t, visitedOps, 1) // Only one visited
}

func TestWalk_SkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/public":   &parser.PathItem{Get: &parser.Operation{OperationID: "publicOp"}},
			"/internal": &parser.PathItem{Get: &parser.Operation{OperationID: "internalOp"}},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var visitedOps []string
	err := Walk(result,
		WithPathHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			if wc.PathTemplate == "/internal" {
				return SkipChildren
			}
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOps = append(visitedOps, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"publicOp"}, visitedOps)
}

func TestWalk_AllHandlers(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Servers: []*parser.Server{{URL: "https://api.example.com"}},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Parameters: []*parser.Parameter{{Name: "pathParam", In: "query"}},
				Get: &parser.Operation{
					OperationID: "listPets",
					Parameters:  []*parser.Parameter{{Name: "limit", In: "query"}},
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {Schema: &parser.Schema{Type: "object"}},
						},
					},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Headers: map[string]*parser.Header{
									"X-Rate-Limit": {},
								},
								Content: map[string]*parser.MediaType{
									"application/json": {},
								},
								Links: map[string]*parser.Link{
									"next": {},
								},
							},
						},
					},
					Callbacks: map[string]*parser.Callback{
						"onEvent": {
							"{$request.body#/callbackUrl}": &parser.PathItem{},
						},
					},
				},
			},
		},
		Components: &parser.Components{
			Schemas:         map[string]*parser.Schema{"Pet": {Type: "object"}},
			SecuritySchemes: map[string]*parser.SecurityScheme{"api_key": {Type: "apiKey"}},
			Examples:        map[string]*parser.Example{"pet": {}},
		},
		Tags:         []*parser.Tag{{Name: "pets"}},
		ExternalDocs: &parser.ExternalDocs{URL: "https://docs.example.com"},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	visited := make(map[string]bool)

	err := Walk(result,
		WithDocumentHandler(func(wc *WalkContext, doc any) Action {
			visited["document"] = true
			return Continue
		}),
		WithInfoHandler(func(wc *WalkContext, info *parser.Info) Action {
			visited["info"] = true
			return Continue
		}),
		WithServerHandler(func(wc *WalkContext, server *parser.Server) Action {
			visited["server"] = true
			return Continue
		}),
		WithTagHandler(func(wc *WalkContext, tag *parser.Tag) Action {
			visited["tag"] = true
			return Continue
		}),
		WithPathHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			visited["path"] = true
			return Continue
		}),
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			visited["pathItem"] = true
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visited["operation"] = true
			return Continue
		}),
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visited["parameter"] = true
			return Continue
		}),
		WithRequestBodyHandler(func(wc *WalkContext, reqBody *parser.RequestBody) Action {
			visited["requestBody"] = true
			return Continue
		}),
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			visited["response"] = true
			return Continue
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visited["schema"] = true
			return Continue
		}),
		WithSecuritySchemeHandler(func(wc *WalkContext, scheme *parser.SecurityScheme) Action {
			visited["securityScheme"] = true
			return Continue
		}),
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			visited["header"] = true
			return Continue
		}),
		WithMediaTypeHandler(func(wc *WalkContext, mt *parser.MediaType) Action {
			visited["mediaType"] = true
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
		WithExternalDocsHandler(func(wc *WalkContext, extDocs *parser.ExternalDocs) Action {
			visited["externalDocs"] = true
			return Continue
		}),
	)

	require.NoError(t, err)

	expected := []string{
		"document", "info", "server", "tag", "path", "pathItem",
		"operation", "parameter", "requestBody", "response", "schema",
		"securityScheme", "header", "mediaType", "link", "callback",
		"example", "externalDocs",
	}

	for _, name := range expected {
		assert.True(t, visited[name], "expected %s handler to be called", name)
	}
}

func TestWalk_Mutation(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {Type: "object", Description: "Original"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schema.Description = "Modified"
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, "Modified", doc.Components.Schemas["Pet"].Description)
}

func TestWalk_JSONPaths(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets/{petId}": &parser.PathItem{
				Get: &parser.Operation{OperationID: "getPet"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var paths []string
	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			paths = append(paths, wc.JSONPath)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Contains(t, paths[0], "$.paths['/pets/{petId}'].get")
}

func TestAction_IsValid(t *testing.T) {
	tests := []struct {
		action   Action
		expected bool
	}{
		{Continue, true},
		{SkipChildren, true},
		{Stop, true},
		{Action(-1), false},
		{Action(100), false},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expected, tc.action.IsValid(), "Action(%d).IsValid()", tc.action)
	}
}

func TestAction_String(t *testing.T) {
	tests := []struct {
		action   Action
		expected string
	}{
		{Continue, "Continue"},
		{SkipChildren, "SkipChildren"},
		{Stop, "Stop"},
		{Action(99), "Action(99)"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expected, tc.action.String())
	}
}

func TestWalk_StopAtDocumentLevel(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{OperationID: "listPets"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	infoCalled := false
	operationCalled := false
	err := Walk(result,
		WithDocumentHandler(func(wc *WalkContext, doc any) Action {
			return Stop // Stop immediately at document level
		}),
		WithInfoHandler(func(wc *WalkContext, info *parser.Info) Action {
			infoCalled = true
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			operationCalled = true
			return Continue
		}),
	)
	require.NoError(t, err)

	// Nothing else should be visited after Stop at document level
	assert.False(t, infoCalled, "info should not be called after Stop at document")
	assert.False(t, operationCalled, "operation should not be called after Stop at document")
}

func TestWalk_OAS3DocumentHandler(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "OAS3 API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{OperationID: "listPets"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var oas3Called bool
	var capturedOpenAPI string
	err := Walk(result,
		WithOAS3DocumentHandler(func(wc *WalkContext, doc *parser.OAS3Document) Action {
			oas3Called = true
			capturedOpenAPI = doc.OpenAPI
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.True(t, oas3Called, "OAS3DocumentHandler should be called for OAS 3.x document")
	assert.Equal(t, "3.1.0", capturedOpenAPI)
}

func TestWalk_TypedAndGenericDocumentHandlers(t *testing.T) {
	t.Run("OAS2 with both handlers", func(t *testing.T) {
		doc := &parser.OAS2Document{
			Swagger: "2.0",
			Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		}

		result := &parser.ParseResult{
			Version:    "2.0",
			OASVersion: parser.OASVersion20,
			Document:   doc,
		}

		var callOrder []string
		err := Walk(result,
			WithOAS2DocumentHandler(func(wc *WalkContext, doc *parser.OAS2Document) Action {
				callOrder = append(callOrder, "typed-oas2")
				return Continue
			}),
			WithDocumentHandler(func(wc *WalkContext, doc any) Action {
				callOrder = append(callOrder, "generic")
				return Continue
			}),
		)

		require.NoError(t, err)
		assert.Equal(t, []string{"typed-oas2", "generic"}, callOrder,
			"typed handler should be called before generic handler")
	})

	t.Run("OAS3 with both handlers", func(t *testing.T) {
		doc := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		}

		result := &parser.ParseResult{
			Version:    "3.0.3",
			OASVersion: parser.OASVersion303,
			Document:   doc,
		}

		var callOrder []string
		err := Walk(result,
			WithOAS3DocumentHandler(func(wc *WalkContext, doc *parser.OAS3Document) Action {
				callOrder = append(callOrder, "typed-oas3")
				return Continue
			}),
			WithDocumentHandler(func(wc *WalkContext, doc any) Action {
				callOrder = append(callOrder, "generic")
				return Continue
			}),
		)

		require.NoError(t, err)
		assert.Equal(t, []string{"typed-oas3", "generic"}, callOrder,
			"typed handler should be called before generic handler")
	})

	t.Run("typed handler returns Stop skips generic", func(t *testing.T) {
		doc := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		}

		result := &parser.ParseResult{
			Version:    "3.0.3",
			OASVersion: parser.OASVersion303,
			Document:   doc,
		}

		var genericCalled bool
		err := Walk(result,
			WithOAS3DocumentHandler(func(wc *WalkContext, doc *parser.OAS3Document) Action {
				return Stop
			}),
			WithDocumentHandler(func(wc *WalkContext, doc any) Action {
				genericCalled = true
				return Continue
			}),
		)

		require.NoError(t, err)
		assert.False(t, genericCalled,
			"generic handler should not be called when typed handler returns Stop")
	})

	t.Run("typed handler returns SkipChildren still calls generic", func(t *testing.T) {
		doc := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		}

		result := &parser.ParseResult{
			Version:    "3.0.3",
			OASVersion: parser.OASVersion303,
			Document:   doc,
		}

		var callOrder []string
		var infoCalled bool
		err := Walk(result,
			WithOAS3DocumentHandler(func(wc *WalkContext, doc *parser.OAS3Document) Action {
				callOrder = append(callOrder, "typed-oas3")
				return SkipChildren
			}),
			WithDocumentHandler(func(wc *WalkContext, doc any) Action {
				callOrder = append(callOrder, "generic")
				return Continue
			}),
			WithInfoHandler(func(wc *WalkContext, info *parser.Info) Action {
				infoCalled = true
				return Continue
			}),
		)

		require.NoError(t, err)
		// Both document handlers called, but Info should be skipped due to SkipChildren
		assert.Equal(t, []string{"typed-oas3", "generic"}, callOrder)
		assert.False(t, infoCalled, "info should not be called when typed handler returns SkipChildren")
	})

	t.Run("OAS2 document does not call OAS3 handler", func(t *testing.T) {
		doc := &parser.OAS2Document{
			Swagger: "2.0",
			Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		}

		result := &parser.ParseResult{
			Version:    "2.0",
			OASVersion: parser.OASVersion20,
			Document:   doc,
		}

		var oas3Called bool
		err := Walk(result,
			WithOAS3DocumentHandler(func(wc *WalkContext, doc *parser.OAS3Document) Action {
				oas3Called = true
				return Continue
			}),
		)

		require.NoError(t, err)
		assert.False(t, oas3Called,
			"OAS3DocumentHandler should not be called for OAS 2.0 document")
	})

	t.Run("OAS3 document does not call OAS2 handler", func(t *testing.T) {
		doc := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		}

		result := &parser.ParseResult{
			Version:    "3.0.3",
			OASVersion: parser.OASVersion303,
			Document:   doc,
		}

		var oas2Called bool
		err := Walk(result,
			WithOAS2DocumentHandler(func(wc *WalkContext, doc *parser.OAS2Document) Action {
				oas2Called = true
				return Continue
			}),
		)

		require.NoError(t, err)
		assert.False(t, oas2Called,
			"OAS2DocumentHandler should not be called for OAS 3.x document")
	})
}
