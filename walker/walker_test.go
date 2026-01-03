package walker

import (
	"strings"
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
		WithMaxDepth(3),
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

func TestWalk_OAS2Parameters(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Parameters: map[string]*parser.Parameter{
			"limitParam": {
				Name: "limit",
				In:   "query",
				Type: "integer",
			},
			"offsetParam": {
				Name: "offset",
				In:   "query",
				Type: "integer",
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var visitedParams []string
	err := Walk(result,
		WithParameterHandler(func(param *parser.Parameter, path string) Action {
			visitedParams = append(visitedParams, param.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedParams, 2)
	assert.Contains(t, visitedParams, "limit")
	assert.Contains(t, visitedParams, "offset")
}

func TestWalk_OAS2Responses(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Responses: map[string]*parser.Response{
			"NotFound": {
				Description: "Resource not found",
			},
			"ServerError": {
				Description: "Internal server error",
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var visitedResponses []string
	err := Walk(result,
		WithResponseHandler(func(statusCode string, resp *parser.Response, path string) Action {
			visitedResponses = append(visitedResponses, statusCode)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedResponses, 2)
	assert.Contains(t, visitedResponses, "NotFound")
	assert.Contains(t, visitedResponses, "ServerError")
}

func TestWalk_OAS2SecurityDefinitions(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"api_key": {
				Type: "apiKey",
				Name: "X-API-Key",
				In:   "header",
			},
			"oauth2": {
				Type: "oauth2",
				Flow: "accessCode",
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var visitedSchemes []string
	err := Walk(result,
		WithSecuritySchemeHandler(func(name string, scheme *parser.SecurityScheme, path string) Action {
			visitedSchemes = append(visitedSchemes, name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedSchemes, 2)
	assert.Contains(t, visitedSchemes, "api_key")
	assert.Contains(t, visitedSchemes, "oauth2")
}

func TestWalk_OAS2Tags(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Tags: []*parser.Tag{
			{Name: "users", Description: "User operations"},
			{Name: "orders", Description: "Order operations"},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var visitedTags []string
	err := Walk(result,
		WithTagHandler(func(tag *parser.Tag, path string) Action {
			visitedTags = append(visitedTags, tag.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedTags, 2)
	assert.Contains(t, visitedTags, "users")
	assert.Contains(t, visitedTags, "orders")
}

func TestWalk_OAS2ExternalDocs(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		ExternalDocs: &parser.ExternalDocs{
			Description: "Find more info here",
			URL:         "https://example.com/docs",
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	externalDocsCalled := false
	var externalDocsURL string
	err := Walk(result,
		WithExternalDocsHandler(func(docs *parser.ExternalDocs, path string) Action {
			externalDocsCalled = true
			externalDocsURL = docs.URL
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.True(t, externalDocsCalled)
	assert.Equal(t, "https://example.com/docs", externalDocsURL)
}

func TestWalk_OAS2DefinitionsWithProperties(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Definitions: map[string]*parser.Schema{
			"User": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"id":   {Type: "integer"},
					"name": {Type: "string"},
				},
			},
			"Error": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"message": {Type: "string"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var visitedSchemas []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			// Only count top-level definitions, not nested properties
			if strings.HasPrefix(path, "$.definitions['") && strings.Count(path, ".") == 1 {
				visitedSchemas = append(visitedSchemas, path)
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	// Should visit User, Error, and their property schemas
	assert.GreaterOrEqual(t, len(visitedSchemas), 2)
}

// Schema Keywords Tests - Work Package 5d

func TestWalk_SchemaPatternProperties(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"DynamicObject": {
					Type: "object",
					PatternProperties: map[string]*parser.Schema{
						"^x-": {Type: "string"},
						"^y-": {Type: "integer"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	schemaCount := 0
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaCount++
			return Continue
		}),
	)
	require.NoError(t, err)

	// DynamicObject + 2 pattern property schemas
	assert.GreaterOrEqual(t, schemaCount, 3)
}

func TestWalk_SchemaAdditionalProperties(t *testing.T) {
	additionalSchema := &parser.Schema{Type: "string"}
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"StringMap": {
					Type:                 "object",
					AdditionalProperties: additionalSchema,
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			visitedPaths = append(visitedPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	// Should visit the additionalProperties schema
	found := false
	for _, p := range visitedPaths {
		if strings.Contains(p, "additionalProperties") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit additionalProperties schema")
}

func TestWalk_SchemaUnevaluatedProperties(t *testing.T) {
	unevalSchema := &parser.Schema{Type: "string"}
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"StrictObject": {
					Type:                  "object",
					UnevaluatedProperties: unevalSchema,
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			visitedPaths = append(visitedPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range visitedPaths {
		if strings.Contains(p, "unevaluatedProperties") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit unevaluatedProperties schema")
}

func TestWalk_SchemaPrefixItems(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"TupleType": {
					Type: "array",
					PrefixItems: []*parser.Schema{
						{Type: "string"},
						{Type: "integer"},
						{Type: "boolean"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	schemaCount := 0
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaCount++
			return Continue
		}),
	)
	require.NoError(t, err)

	// TupleType + 3 prefixItems schemas
	assert.GreaterOrEqual(t, schemaCount, 4)
}

func TestWalk_SchemaConditionals(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"ConditionalSchema": {
					Type: "object",
					If:   &parser.Schema{Properties: map[string]*parser.Schema{"type": {Const: "premium"}}},
					Then: &parser.Schema{Required: []string{"premiumFeatures"}},
					Else: &parser.Schema{Required: []string{"basicFeatures"}},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			visitedPaths = append(visitedPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	hasIf, hasThen, hasElse := false, false, false
	for _, p := range visitedPaths {
		if strings.Contains(p, ".if") {
			hasIf = true
		}
		if strings.Contains(p, ".then") {
			hasThen = true
		}
		if strings.Contains(p, ".else") {
			hasElse = true
		}
	}
	assert.True(t, hasIf, "should visit if schema")
	assert.True(t, hasThen, "should visit then schema")
	assert.True(t, hasElse, "should visit else schema")
}

func TestWalk_SchemaDefs(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"ParentSchema": {
					Type: "object",
					Defs: map[string]*parser.Schema{
						"NestedDef":  {Type: "string"},
						"AnotherDef": {Type: "integer"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			visitedPaths = append(visitedPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	defsCount := 0
	for _, p := range visitedPaths {
		if strings.Contains(p, "$defs") {
			defsCount++
		}
	}
	assert.GreaterOrEqual(t, defsCount, 2, "should visit $defs schemas")
}

func TestWalk_SchemaContains(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"ArrayWithContains": {
					Type:     "array",
					Contains: &parser.Schema{Type: "integer", Minimum: floatPtr(0)},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			visitedPaths = append(visitedPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range visitedPaths {
		if strings.Contains(p, ".contains") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit contains schema")
}

// floatPtr is a helper function for creating float64 pointers
func floatPtr(f float64) *float64 {
	return &f
}

func TestWalk_SchemaPropertyNames(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"RestrictedKeys": {
					Type:          "object",
					PropertyNames: &parser.Schema{Pattern: "^[a-z]+$"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			visitedPaths = append(visitedPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range visitedPaths {
		if strings.Contains(p, ".propertyNames") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit propertyNames schema")
}

// OAS 3.x Component Tests - Work Package 5c

func TestWalk_OAS3ComponentResponses(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Responses: map[string]*parser.Response{
				"NotFound": {
					Description: "Resource not found",
					Content: map[string]*parser.MediaType{
						"application/json": {
							Schema: &parser.Schema{Type: "object"},
						},
					},
				},
				"ServerError": {
					Description: "Internal server error",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedResponses []string
	err := Walk(result,
		WithResponseHandler(func(statusCode string, resp *parser.Response, path string) Action {
			visitedResponses = append(visitedResponses, statusCode)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedResponses, 2)
	assert.Contains(t, visitedResponses, "NotFound")
	assert.Contains(t, visitedResponses, "ServerError")
}

func TestWalk_OAS3ComponentParameters(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Parameters: map[string]*parser.Parameter{
				"pageParam": {
					Name:   "page",
					In:     "query",
					Schema: &parser.Schema{Type: "integer"},
				},
				"limitParam": {
					Name:   "limit",
					In:     "query",
					Schema: &parser.Schema{Type: "integer"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedParams []string
	err := Walk(result,
		WithParameterHandler(func(param *parser.Parameter, path string) Action {
			visitedParams = append(visitedParams, param.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedParams, 2)
	assert.Contains(t, visitedParams, "page")
	assert.Contains(t, visitedParams, "limit")
}

func TestWalk_OAS3ComponentRequestBodies(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			RequestBodies: map[string]*parser.RequestBody{
				"UserInput": {
					Description: "User input data",
					Content: map[string]*parser.MediaType{
						"application/json": {
							Schema: &parser.Schema{Type: "object"},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithRequestBodyHandler(func(body *parser.RequestBody, path string) Action {
			visitedPaths = append(visitedPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedPaths, 1)
	assert.Contains(t, visitedPaths[0], "requestBodies['UserInput']")
}

func TestWalk_OAS3ComponentCallbacks(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Callbacks: map[string]*parser.Callback{
				"onPayment": {
					"{$request.body#/callbackUrl}": &parser.PathItem{
						Post: &parser.Operation{
							Summary: "Payment callback",
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedCallbacks []string
	err := Walk(result,
		WithCallbackHandler(func(name string, callback parser.Callback, path string) Action {
			visitedCallbacks = append(visitedCallbacks, name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedCallbacks, 1)
	assert.Contains(t, visitedCallbacks, "onPayment")
}

func TestWalk_OAS3ComponentLinks(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Links: map[string]*parser.Link{
				"GetUserById": {
					OperationID: "getUser",
					Description: "Get user by ID link",
				},
				"GetOrderById": {
					OperationID: "getOrder",
					Description: "Get order by ID link",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedLinks []string
	err := Walk(result,
		WithLinkHandler(func(name string, link *parser.Link, path string) Action {
			visitedLinks = append(visitedLinks, name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedLinks, 2)
	assert.Contains(t, visitedLinks, "GetUserById")
	assert.Contains(t, visitedLinks, "GetOrderById")
}

func TestWalk_OAS3ComponentPathItems(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			PathItems: map[string]*parser.PathItem{
				"SharedEndpoint": {
					Get: &parser.Operation{
						Summary: "Shared GET operation",
					},
					Post: &parser.Operation{
						Summary: "Shared POST operation",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedPathItems []string
	err := Walk(result,
		WithPathItemHandler(func(pathItem *parser.PathItem, path string) Action {
			visitedPathItems = append(visitedPathItems, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	// Should visit the component path item
	found := false
	for _, p := range visitedPathItems {
		if strings.Contains(p, "components.pathItems") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit components.pathItems")
}

func TestWalk_OAS3ComponentExamples(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Examples: map[string]*parser.Example{
				"UserExample": {
					Summary: "Example user",
					Value:   map[string]any{"id": 1, "name": "John"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedExamples []string
	err := Walk(result,
		WithExampleHandler(func(name string, example *parser.Example, path string) Action {
			visitedExamples = append(visitedExamples, name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedExamples, 1)
	assert.Contains(t, visitedExamples, "UserExample")
}

func TestWalk_OAS3ComponentHeaders(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"X-Rate-Limit": {
					Description: "Rate limit header",
					Schema:      &parser.Schema{Type: "integer"},
				},
				"X-Request-ID": {
					Description: "Request ID header",
					Schema:      &parser.Schema{Type: "string"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedHeaders []string
	err := Walk(result,
		WithHeaderHandler(func(name string, header *parser.Header, path string) Action {
			visitedHeaders = append(visitedHeaders, name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedHeaders, 2)
	assert.Contains(t, visitedHeaders, "X-Rate-Limit")
	assert.Contains(t, visitedHeaders, "X-Request-ID")
}

func TestWalk_OAS3ComponentSecuritySchemes(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"bearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				},
				"apiKeyAuth": {
					Type: "apiKey",
					Name: "X-API-Key",
					In:   "header",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedSchemes []string
	err := Walk(result,
		WithSecuritySchemeHandler(func(name string, scheme *parser.SecurityScheme, path string) Action {
			visitedSchemes = append(visitedSchemes, name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedSchemes, 2)
	assert.Contains(t, visitedSchemes, "bearerAuth")
	assert.Contains(t, visitedSchemes, "apiKeyAuth")
}

func TestWalk_OAS3AllComponents(t *testing.T) {
	// Test walking a document with all component types
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"User": {Type: "object"},
			},
			Responses: map[string]*parser.Response{
				"NotFound": {Description: "Not found"},
			},
			Parameters: map[string]*parser.Parameter{
				"pageParam": {Name: "page", In: "query"},
			},
			RequestBodies: map[string]*parser.RequestBody{
				"UserInput": {Description: "User input"},
			},
			Headers: map[string]*parser.Header{
				"X-Rate-Limit": {Description: "Rate limit"},
			},
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"bearerAuth": {Type: "http"},
			},
			Links: map[string]*parser.Link{
				"GetUserById": {OperationID: "getUser"},
			},
			Callbacks: map[string]*parser.Callback{
				"onEvent": {
					"{$url}": &parser.PathItem{},
				},
			},
			Examples: map[string]*parser.Example{
				"UserExample": {Summary: "User example"},
			},
			PathItems: map[string]*parser.PathItem{
				"SharedPath": {Get: &parser.Operation{Summary: "Shared"}},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	visited := make(map[string]bool)

	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			visited["schema"] = true
			return Continue
		}),
		WithResponseHandler(func(statusCode string, resp *parser.Response, path string) Action {
			visited["response"] = true
			return Continue
		}),
		WithParameterHandler(func(param *parser.Parameter, path string) Action {
			visited["parameter"] = true
			return Continue
		}),
		WithRequestBodyHandler(func(body *parser.RequestBody, path string) Action {
			visited["requestBody"] = true
			return Continue
		}),
		WithHeaderHandler(func(name string, header *parser.Header, path string) Action {
			visited["header"] = true
			return Continue
		}),
		WithSecuritySchemeHandler(func(name string, scheme *parser.SecurityScheme, path string) Action {
			visited["securityScheme"] = true
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
		WithPathItemHandler(func(pathItem *parser.PathItem, path string) Action {
			if strings.Contains(path, "components.pathItems") {
				visited["pathItem"] = true
			}
			return Continue
		}),
	)

	require.NoError(t, err)

	expected := []string{
		"schema", "response", "parameter", "requestBody",
		"header", "securityScheme", "link", "callback",
		"example", "pathItem",
	}

	for _, name := range expected {
		assert.True(t, visited[name], "expected %s component to be visited", name)
	}
}

// Action Tests - Coverage for Action type methods

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

// WalkWithOptions Error Tests

func TestWalkWithOptions_InvalidMaxSchemaDepth(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
		},
	}

	err := WalkWithOptions(
		WithParsed(result),
		WithMaxSchemaDepth(0),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maxDepth must be positive")
}

func TestWalkWithOptions_NegativeMaxSchemaDepth(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
		},
	}

	err := WalkWithOptions(
		WithParsed(result),
		WithMaxSchemaDepth(-5),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maxDepth must be positive")
}

func TestWalkWithOptions_ValidMaxSchemaDepth(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
		},
	}

	err := WalkWithOptions(
		WithParsed(result),
		WithMaxSchemaDepth(50),
	)
	require.NoError(t, err)
}

// WalkWithOptions Handler Tests - Testing On* handler options

func TestWalkWithOptions_OnInfo(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test API", Version: "1.0"},
		},
	}

	var infoTitle string
	err := WalkWithOptions(
		WithParsed(result),
		OnInfo(func(info *parser.Info, path string) Action {
			infoTitle = info.Title
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, "Test API", infoTitle)
}

func TestWalkWithOptions_OnServer(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Servers: []*parser.Server{
				{URL: "https://api.example.com"},
				{URL: "https://staging.example.com"},
			},
		},
	}

	var serverURLs []string
	err := WalkWithOptions(
		WithParsed(result),
		OnServer(func(server *parser.Server, path string) Action {
			serverURLs = append(serverURLs, server.URL)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Len(t, serverURLs, 2)
}

func TestWalkWithOptions_OnTag(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Tags: []*parser.Tag{
				{Name: "pets"},
				{Name: "users"},
			},
		},
	}

	var tagNames []string
	err := WalkWithOptions(
		WithParsed(result),
		OnTag(func(tag *parser.Tag, path string) Action {
			tagNames = append(tagNames, tag.Name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Len(t, tagNames, 2)
}

func TestWalkWithOptions_OnPath(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Paths: parser.Paths{
				"/pets":  &parser.PathItem{},
				"/users": &parser.PathItem{},
			},
		},
	}

	var paths []string
	err := WalkWithOptions(
		WithParsed(result),
		OnPath(func(pathTemplate string, pathItem *parser.PathItem, path string) Action {
			paths = append(paths, pathTemplate)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Len(t, paths, 2)
}

func TestWalkWithOptions_OnPathItem(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Paths: parser.Paths{
				"/pets": &parser.PathItem{},
			},
		},
	}

	pathItemCount := 0
	err := WalkWithOptions(
		WithParsed(result),
		OnPathItem(func(pathItem *parser.PathItem, path string) Action {
			pathItemCount++
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, 1, pathItemCount)
}

func TestWalkWithOptions_OnOperation(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Paths: parser.Paths{
				"/pets": &parser.PathItem{
					Get:  &parser.Operation{OperationID: "listPets"},
					Post: &parser.Operation{OperationID: "createPet"},
				},
			},
		},
	}

	var methods []string
	err := WalkWithOptions(
		WithParsed(result),
		OnOperation(func(method string, op *parser.Operation, path string) Action {
			methods = append(methods, method)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Len(t, methods, 2)
}

func TestWalkWithOptions_OnParameter(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Paths: parser.Paths{
				"/pets/{id}": &parser.PathItem{
					Get: &parser.Operation{
						OperationID: "getPet",
						Parameters: []*parser.Parameter{
							{Name: "id", In: "path"},
						},
					},
				},
			},
		},
	}

	var paramNames []string
	err := WalkWithOptions(
		WithParsed(result),
		OnParameter(func(param *parser.Parameter, path string) Action {
			paramNames = append(paramNames, param.Name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, paramNames, "id")
}

func TestWalkWithOptions_OnRequestBody(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Paths: parser.Paths{
				"/pets": &parser.PathItem{
					Post: &parser.Operation{
						OperationID: "createPet",
						RequestBody: &parser.RequestBody{
							Description: "Pet to add",
						},
					},
				},
			},
		},
	}

	requestBodyCount := 0
	err := WalkWithOptions(
		WithParsed(result),
		OnRequestBody(func(reqBody *parser.RequestBody, path string) Action {
			requestBodyCount++
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, 1, requestBodyCount)
}

func TestWalkWithOptions_OnResponse(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Paths: parser.Paths{
				"/pets": &parser.PathItem{
					Get: &parser.Operation{
						Responses: &parser.Responses{
							Codes: map[string]*parser.Response{
								"200": {Description: "OK"},
							},
						},
					},
				},
			},
		},
	}

	var statusCodes []string
	err := WalkWithOptions(
		WithParsed(result),
		OnResponse(func(statusCode string, resp *parser.Response, path string) Action {
			statusCodes = append(statusCodes, statusCode)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, statusCodes, "200")
}

func TestWalkWithOptions_OnSecurityScheme(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Components: &parser.Components{
				SecuritySchemes: map[string]*parser.SecurityScheme{
					"bearerAuth": {Type: "http", Scheme: "bearer"},
				},
			},
		},
	}

	var schemeNames []string
	err := WalkWithOptions(
		WithParsed(result),
		OnSecurityScheme(func(name string, scheme *parser.SecurityScheme, path string) Action {
			schemeNames = append(schemeNames, name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, schemeNames, "bearerAuth")
}

func TestWalkWithOptions_OnHeader(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Components: &parser.Components{
				Headers: map[string]*parser.Header{
					"X-Rate-Limit": {Description: "Rate limit"},
				},
			},
		},
	}

	var headerNames []string
	err := WalkWithOptions(
		WithParsed(result),
		OnHeader(func(name string, header *parser.Header, path string) Action {
			headerNames = append(headerNames, name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, headerNames, "X-Rate-Limit")
}

func TestWalkWithOptions_OnMediaType(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Paths: parser.Paths{
				"/pets": &parser.PathItem{
					Get: &parser.Operation{
						Responses: &parser.Responses{
							Codes: map[string]*parser.Response{
								"200": {
									Description: "OK",
									Content: map[string]*parser.MediaType{
										"application/json": {},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	var mediaTypes []string
	err := WalkWithOptions(
		WithParsed(result),
		OnMediaType(func(mediaTypeName string, mt *parser.MediaType, path string) Action {
			mediaTypes = append(mediaTypes, mediaTypeName)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, mediaTypes, "application/json")
}

func TestWalkWithOptions_OnLink(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Components: &parser.Components{
				Links: map[string]*parser.Link{
					"GetUserById": {OperationID: "getUser"},
				},
			},
		},
	}

	var linkNames []string
	err := WalkWithOptions(
		WithParsed(result),
		OnLink(func(name string, link *parser.Link, path string) Action {
			linkNames = append(linkNames, name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, linkNames, "GetUserById")
}

func TestWalkWithOptions_OnCallback(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Components: &parser.Components{
				Callbacks: map[string]*parser.Callback{
					"onEvent": {
						"{$request.body#/callbackUrl}": &parser.PathItem{},
					},
				},
			},
		},
	}

	var callbackNames []string
	err := WalkWithOptions(
		WithParsed(result),
		OnCallback(func(name string, callback parser.Callback, path string) Action {
			callbackNames = append(callbackNames, name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, callbackNames, "onEvent")
}

func TestWalkWithOptions_OnExample(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
			Components: &parser.Components{
				Examples: map[string]*parser.Example{
					"petExample": {Summary: "A pet example"},
				},
			},
		},
	}

	var exampleNames []string
	err := WalkWithOptions(
		WithParsed(result),
		OnExample(func(name string, example *parser.Example, path string) Action {
			exampleNames = append(exampleNames, name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, exampleNames, "petExample")
}

func TestWalkWithOptions_OnExternalDocs(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI:      "3.0.0",
			Info:         &parser.Info{Title: "Test", Version: "1.0"},
			ExternalDocs: &parser.ExternalDocs{URL: "https://docs.example.com"},
		},
	}

	var docsURL string
	err := WalkWithOptions(
		WithParsed(result),
		OnExternalDocs(func(extDocs *parser.ExternalDocs, path string) Action {
			docsURL = extDocs.URL
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, "https://docs.example.com", docsURL)
}

// walkExamples Coverage Test

func TestWalk_MediaTypeExamples(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Examples: map[string]*parser.Example{
											"cat":  {Summary: "A cat"},
											"dog":  {Summary: "A dog"},
											"bird": {Summary: "A bird"},
										},
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
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var exampleNames []string
	err := Walk(result,
		WithExampleHandler(func(name string, example *parser.Example, path string) Action {
			exampleNames = append(exampleNames, name)
			return Continue
		}),
	)
	require.NoError(t, err)

	// Should visit all 3 examples in the media type
	assert.Len(t, exampleNames, 3)
	assert.Contains(t, exampleNames, "cat")
	assert.Contains(t, exampleNames, "dog")
	assert.Contains(t, exampleNames, "bird")
}

// Stop at Document Level Test

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
		WithDocumentHandler(func(doc any, path string) Action {
			return Stop // Stop immediately at document level
		}),
		WithInfoHandler(func(info *parser.Info, path string) Action {
			infoCalled = true
			return Continue
		}),
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			operationCalled = true
			return Continue
		}),
	)
	require.NoError(t, err)

	// Nothing else should be visited after Stop at document level
	assert.False(t, infoCalled, "info should not be called after Stop at document")
	assert.False(t, operationCalled, "operation should not be called after Stop at document")
}

// SchemaSkippedHandler Tests

func TestWalk_SchemaSkippedDepthLimit(t *testing.T) {
	// Create deeply nested schema that will exceed depth limit
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

	var skippedReasons []string
	var skippedPaths []string
	err := Walk(result,
		WithMaxDepth(3),
		WithSchemaSkippedHandler(func(reason string, schema *parser.Schema, path string) {
			skippedReasons = append(skippedReasons, reason)
			skippedPaths = append(skippedPaths, path)
		}),
	)

	require.NoError(t, err)
	// Should have skipped schemas due to depth limit
	assert.NotEmpty(t, skippedReasons, "expected schemas to be skipped due to depth")
	for _, reason := range skippedReasons {
		assert.Equal(t, "depth", reason, "expected skip reason to be 'depth'")
	}
	// The paths should show the nested structure
	for _, path := range skippedPaths {
		assert.Contains(t, path, "Deep", "path should reference the Deep schema")
	}
}

func TestWalk_SchemaSkippedCycle(t *testing.T) {
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

	var skippedReasons []string
	var skippedSchemas []*parser.Schema
	var skippedPaths []string
	err := Walk(result,
		WithSchemaSkippedHandler(func(reason string, schema *parser.Schema, path string) {
			skippedReasons = append(skippedReasons, reason)
			skippedSchemas = append(skippedSchemas, schema)
			skippedPaths = append(skippedPaths, path)
		}),
	)

	require.NoError(t, err)
	// Should have skipped the circular reference
	assert.Len(t, skippedReasons, 1, "expected one schema to be skipped due to cycle")
	assert.Equal(t, "cycle", skippedReasons[0], "expected skip reason to be 'cycle'")
	assert.Equal(t, petSchema, skippedSchemas[0], "expected skipped schema to be the pet schema")
	assert.Contains(t, skippedPaths[0], "parent", "expected path to include 'parent' property")
}

func TestWalkWithOptions_OnSchemaSkipped(t *testing.T) {
	// Create a circular schema reference
	nodeSchema := &parser.Schema{Type: "object"}
	nodeSchema.Properties = map[string]*parser.Schema{
		"child": nodeSchema, // Circular reference
	}

	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Node": nodeSchema,
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var skippedCount int
	var lastReason string
	err := WalkWithOptions(
		WithParsed(result),
		OnSchemaSkipped(func(reason string, schema *parser.Schema, path string) {
			skippedCount++
			lastReason = reason
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, 1, skippedCount, "expected one schema to be skipped")
	assert.Equal(t, "cycle", lastReason, "expected skip reason to be 'cycle'")
}

func TestWalk_SchemaSkippedHandlerNotCalledForNil(t *testing.T) {
	// Test that the handler is NOT called for nil schemas
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Empty": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	handlerCalled := false
	err := Walk(result,
		WithSchemaSkippedHandler(func(reason string, schema *parser.Schema, path string) {
			handlerCalled = true
		}),
	)

	require.NoError(t, err)
	assert.False(t, handlerCalled, "handler should not be called when no schemas are skipped")
}

// Typed Document Handler Tests

func TestWalk_OAS2DocumentHandler(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "OAS2 API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{OperationID: "listPets"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var oas2Called bool
	var capturedSwagger string
	err := Walk(result,
		WithOAS2DocumentHandler(func(doc *parser.OAS2Document, path string) Action {
			oas2Called = true
			capturedSwagger = doc.Swagger
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.True(t, oas2Called, "OAS2DocumentHandler should be called for OAS 2.0 document")
	assert.Equal(t, "2.0", capturedSwagger)
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
		WithOAS3DocumentHandler(func(doc *parser.OAS3Document, path string) Action {
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
			WithOAS2DocumentHandler(func(doc *parser.OAS2Document, path string) Action {
				callOrder = append(callOrder, "typed-oas2")
				return Continue
			}),
			WithDocumentHandler(func(doc any, path string) Action {
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
			WithOAS3DocumentHandler(func(doc *parser.OAS3Document, path string) Action {
				callOrder = append(callOrder, "typed-oas3")
				return Continue
			}),
			WithDocumentHandler(func(doc any, path string) Action {
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
			WithOAS3DocumentHandler(func(doc *parser.OAS3Document, path string) Action {
				return Stop
			}),
			WithDocumentHandler(func(doc any, path string) Action {
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
			WithOAS3DocumentHandler(func(doc *parser.OAS3Document, path string) Action {
				callOrder = append(callOrder, "typed-oas3")
				return SkipChildren
			}),
			WithDocumentHandler(func(doc any, path string) Action {
				callOrder = append(callOrder, "generic")
				return Continue
			}),
			WithInfoHandler(func(info *parser.Info, path string) Action {
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
			WithOAS3DocumentHandler(func(doc *parser.OAS3Document, path string) Action {
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
			WithOAS2DocumentHandler(func(doc *parser.OAS2Document, path string) Action {
				oas2Called = true
				return Continue
			}),
		)

		require.NoError(t, err)
		assert.False(t, oas2Called,
			"OAS2DocumentHandler should not be called for OAS 3.x document")
	})
}

// WalkWithOptions Typed Document Handler Tests

func TestWalkWithOptions_OnOAS2Document(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var oas2Called bool
	err := WalkWithOptions(
		WithParsed(result),
		OnOAS2Document(func(doc *parser.OAS2Document, path string) Action {
			oas2Called = true
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.True(t, oas2Called, "OnOAS2Document should be called for OAS 2.0 document")
}

func TestWalkWithOptions_OnOAS3Document(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var oas3Called bool
	err := WalkWithOptions(
		WithParsed(result),
		OnOAS3Document(func(doc *parser.OAS3Document, path string) Action {
			oas3Called = true
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.True(t, oas3Called, "OnOAS3Document should be called for OAS 3.x document")
}
