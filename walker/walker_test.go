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
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedOps = append(visitedOps, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"listPets"}, visitedOps)
}

func TestWalk_OAS2Document(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
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
		OASVersion: parser.OASVersion20,
	}

	var visitedOps []string
	err := Walk(result,
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
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
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
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
		WithPathHandler(func(pathTemplate string, pathItem *parser.PathItem, path string) Action {
			if pathTemplate == "/internal" {
				return SkipChildren
			}
			return Continue
		}),
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedOps = append(visitedOps, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"publicOp"}, visitedOps)
}

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
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaPaths = append(schemaPaths, path)
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
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
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
		WithMaxSchemaDepth(3),
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			visitCount++
			return Continue
		}),
	)

	require.NoError(t, err)
	// Should stop at depth 3
	assert.LessOrEqual(t, visitCount, 4)
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
		WithDocumentHandler(func(doc any, path string) Action {
			visited["document"] = true
			return Continue
		}),
		WithInfoHandler(func(info *parser.Info, path string) Action {
			visited["info"] = true
			return Continue
		}),
		WithServerHandler(func(server *parser.Server, path string) Action {
			visited["server"] = true
			return Continue
		}),
		WithTagHandler(func(tag *parser.Tag, path string) Action {
			visited["tag"] = true
			return Continue
		}),
		WithPathHandler(func(pathTemplate string, pathItem *parser.PathItem, path string) Action {
			visited["path"] = true
			return Continue
		}),
		WithPathItemHandler(func(pathItem *parser.PathItem, path string) Action {
			visited["pathItem"] = true
			return Continue
		}),
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visited["operation"] = true
			return Continue
		}),
		WithParameterHandler(func(param *parser.Parameter, path string) Action {
			visited["parameter"] = true
			return Continue
		}),
		WithRequestBodyHandler(func(reqBody *parser.RequestBody, path string) Action {
			visited["requestBody"] = true
			return Continue
		}),
		WithResponseHandler(func(statusCode string, resp *parser.Response, path string) Action {
			visited["response"] = true
			return Continue
		}),
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			visited["schema"] = true
			return Continue
		}),
		WithSecuritySchemeHandler(func(name string, scheme *parser.SecurityScheme, path string) Action {
			visited["securityScheme"] = true
			return Continue
		}),
		WithHeaderHandler(func(name string, header *parser.Header, path string) Action {
			visited["header"] = true
			return Continue
		}),
		WithMediaTypeHandler(func(mediaTypeName string, mt *parser.MediaType, path string) Action {
			visited["mediaType"] = true
			return Continue
		}),
		WithLinkHandler(func(name string, link *parser.Link, path string) Action {
			visited["link"] = true
			return Continue
		}),
		WithCallbackHandler(func(name string, callback parser.Callback, path string) Action {
			visited["callback"] = true
			return Continue
		}),
		WithExampleHandler(func(name string, example *parser.Example, path string) Action {
			visited["example"] = true
			return Continue
		}),
		WithExternalDocsHandler(func(extDocs *parser.ExternalDocs, path string) Action {
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
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
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
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			paths = append(paths, path)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Contains(t, paths[0], "$.paths['/pets/{petId}'].get")
}

func TestWalkWithOptions_NoInput(t *testing.T) {
	err := WalkWithOptions()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no input source specified")
}

func TestWalkWithOptions_MultipleInputs(t *testing.T) {
	result := &parser.ParseResult{
		Document:   &parser.OAS3Document{OpenAPI: "3.0.3", Info: &parser.Info{Title: "Test", Version: "1.0.0"}},
		OASVersion: parser.OASVersion303,
	}

	err := WalkWithOptions(
		WithFilePath("test.yaml"),
		WithParsed(result),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple input sources")
}

func TestWalkWithOptions_WithParsed(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var called bool
	err := WalkWithOptions(
		WithParsed(result),
		OnDocument(func(doc any, path string) Action {
			called = true
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.True(t, called)
}

func TestWalk_OAS2Definitions(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Definitions: map[string]*parser.Schema{
			"Pet":   {Type: "object"},
			"Error": {Type: "object"},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaPaths = append(schemaPaths, path)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Len(t, schemaPaths, 2)
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
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			visitCount++
			return Continue
		}),
	)

	require.NoError(t, err)
	// Combined + 2 allOf + 1 oneOf + 1 anyOf + 1 not = 6
	assert.Equal(t, 6, visitCount)
}

func TestWalk_Webhooks(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Webhooks: map[string]*parser.PathItem{
			"newPet": {
				Post: &parser.Operation{OperationID: "newPetWebhook"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var visitedOps []string
	err := Walk(result,
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedOps = append(visitedOps, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Contains(t, visitedOps, "newPetWebhook")
}
