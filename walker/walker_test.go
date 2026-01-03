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

// Tests for walkOAS3PathItemOperations - Coverage for all HTTP methods

func TestWalk_OAS3AllHTTPMethods(t *testing.T) {
	// Test that all HTTP methods including TRACE are visited
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/resource": &parser.PathItem{
				Get:     &parser.Operation{OperationID: "getResource"},
				Put:     &parser.Operation{OperationID: "putResource"},
				Post:    &parser.Operation{OperationID: "postResource"},
				Delete:  &parser.Operation{OperationID: "deleteResource"},
				Options: &parser.Operation{OperationID: "optionsResource"},
				Head:    &parser.Operation{OperationID: "headResource"},
				Patch:   &parser.Operation{OperationID: "patchResource"},
				Trace:   &parser.Operation{OperationID: "traceResource"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	visitedMethods := make(map[string]bool)
	visitedOpIDs := make(map[string]bool)
	err := Walk(result,
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedMethods[method] = true
			visitedOpIDs[op.OperationID] = true
			return Continue
		}),
	)

	require.NoError(t, err)

	// All 8 HTTP methods should be visited
	expectedMethods := []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}
	for _, method := range expectedMethods {
		assert.True(t, visitedMethods[method], "expected %s method to be visited", method)
	}

	// All operation IDs should be visited
	expectedOpIDs := []string{
		"getResource", "putResource", "postResource", "deleteResource",
		"optionsResource", "headResource", "patchResource", "traceResource",
	}
	for _, opID := range expectedOpIDs {
		assert.True(t, visitedOpIDs[opID], "expected operation %s to be visited", opID)
	}
}

func TestWalk_OAS3OperationSkipChildren(t *testing.T) {
	// Test that SkipChildren from operation handler skips operation's children
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/resource": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getResource",
					Parameters: []*parser.Parameter{
						{Name: "id", In: "query"},
					},
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {Schema: &parser.Schema{Type: "object"}},
						},
					},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "OK"},
						},
					},
				},
				Post: &parser.Operation{
					OperationID: "postResource",
					Parameters: []*parser.Parameter{
						{Name: "body", In: "body"},
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

	var visitedParams []string
	var visitedResponses []string
	var visitedRequestBodies int
	err := Walk(result,
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			if method == "get" {
				return SkipChildren // Skip GET operation's children
			}
			return Continue
		}),
		WithParameterHandler(func(param *parser.Parameter, path string) Action {
			visitedParams = append(visitedParams, param.Name)
			return Continue
		}),
		WithRequestBodyHandler(func(reqBody *parser.RequestBody, path string) Action {
			visitedRequestBodies++
			return Continue
		}),
		WithResponseHandler(func(statusCode string, resp *parser.Response, path string) Action {
			visitedResponses = append(visitedResponses, statusCode)
			return Continue
		}),
	)

	require.NoError(t, err)

	// Only POST's parameters should be visited (GET's were skipped)
	assert.Equal(t, []string{"body"}, visitedParams)

	// GET's responses and request body should be skipped
	assert.Empty(t, visitedResponses)
	assert.Equal(t, 0, visitedRequestBodies)
}

func TestWalk_OAS3OperationStop(t *testing.T) {
	// Test that Stop from operation handler stops the entire walk
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/a": &parser.PathItem{
				Get:  &parser.Operation{OperationID: "getA"},
				Post: &parser.Operation{OperationID: "postA"},
			},
			"/b": &parser.PathItem{
				Get: &parser.Operation{OperationID: "getB"},
			},
			"/c": &parser.PathItem{
				Get: &parser.Operation{OperationID: "getC"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var visitedOps []string
	err := Walk(result,
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedOps = append(visitedOps, op.OperationID)
			// Stop after visiting first operation
			return Stop
		}),
	)

	require.NoError(t, err)
	// Only one operation should be visited due to Stop
	assert.Len(t, visitedOps, 1)
}

func TestWalk_OAS3TRACEMethod(t *testing.T) {
	// Specifically test TRACE method handling
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/debug": &parser.PathItem{
				Trace: &parser.Operation{
					OperationID: "traceDebug",
					Summary:     "Debug trace endpoint",
					Parameters: []*parser.Parameter{
						{Name: "X-Trace-ID", In: "header"},
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

	var traceVisited bool
	var tracePath string
	err := Walk(result,
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			if method == "trace" {
				traceVisited = true
				tracePath = path
			}
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.True(t, traceVisited, "TRACE operation should be visited")
	assert.Contains(t, tracePath, ".trace", "path should contain .trace")
}

// Tests for walkOAS3Components - Coverage for component types

func TestWalk_OAS3ComponentParametersWithStop(t *testing.T) {
	// Test Stop action from component parameters
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Parameters: map[string]*parser.Parameter{
				"aParam": {Name: "a", In: "query"},
				"bParam": {Name: "b", In: "query"},
				"cParam": {Name: "c", In: "query"},
			},
			Schemas: map[string]*parser.Schema{
				"ShouldNotVisit": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var visitedParams []string
	var schemaVisited bool
	err := Walk(result,
		WithParameterHandler(func(param *parser.Parameter, path string) Action {
			visitedParams = append(visitedParams, param.Name)
			return Stop // Stop after first parameter
		}),
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaVisited = true
			return Continue
		}),
	)

	require.NoError(t, err)
	// Only one parameter should be visited due to Stop
	assert.Len(t, visitedParams, 1)
	// Schemas should still be visited since they come before parameters
	// (component order is schemas, responses, parameters, ...)
	assert.True(t, schemaVisited)
}

func TestWalk_OAS3ComponentRequestBodiesWithContent(t *testing.T) {
	// Test request bodies in components with nested content
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			RequestBodies: map[string]*parser.RequestBody{
				"CreateUser": {
					Description: "Create user request",
					Required:    true,
					Content: map[string]*parser.MediaType{
						"application/json": {
							Schema: &parser.Schema{
								Type: "object",
								Properties: map[string]*parser.Schema{
									"name":  {Type: "string"},
									"email": {Type: "string"},
								},
							},
						},
						"application/xml": {
							Schema: &parser.Schema{Type: "object"},
						},
					},
				},
				"UpdateUser": {
					Description: "Update user request",
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
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var visitedRequestBodies []string
	var visitedMediaTypes []string
	var schemaCount int
	err := Walk(result,
		WithRequestBodyHandler(func(reqBody *parser.RequestBody, path string) Action {
			visitedRequestBodies = append(visitedRequestBodies, reqBody.Description)
			return Continue
		}),
		WithMediaTypeHandler(func(mtName string, mt *parser.MediaType, path string) Action {
			visitedMediaTypes = append(visitedMediaTypes, mtName)
			return Continue
		}),
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaCount++
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Len(t, visitedRequestBodies, 2)
	assert.GreaterOrEqual(t, len(visitedMediaTypes), 3) // At least 3 media types
	assert.GreaterOrEqual(t, schemaCount, 4)            // Multiple schemas
}

func TestWalk_OAS3ComponentLinksWithStop(t *testing.T) {
	// Test Stop action from component links
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Links: map[string]*parser.Link{
				"GetUserByID": {
					OperationID: "getUser",
					Description: "Link to get user by ID",
				},
				"GetOrderByID": {
					OperationID: "getOrder",
					Description: "Link to get order by ID",
				},
				"GetProductByID": {
					OperationID: "getProduct",
					Description: "Link to get product by ID",
				},
			},
			// These should not be visited after Stop
			Examples: map[string]*parser.Example{
				"UserExample": {Summary: "User example"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var visitedLinks []string
	var examplesVisited bool
	err := Walk(result,
		WithLinkHandler(func(name string, link *parser.Link, path string) Action {
			visitedLinks = append(visitedLinks, name)
			return Stop // Stop after first link
		}),
		WithExampleHandler(func(name string, example *parser.Example, path string) Action {
			examplesVisited = true
			return Continue
		}),
	)

	require.NoError(t, err)
	// Only one link should be visited due to Stop
	assert.Len(t, visitedLinks, 1)
	// Examples should NOT be visited since they come after links and Stop was called
	assert.False(t, examplesVisited)
}

func TestWalk_OAS3ComponentCallbacksWithSkipChildren(t *testing.T) {
	// Test SkipChildren action from component callbacks
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Callbacks: map[string]*parser.Callback{
				"onPaymentComplete": {
					"{$request.body#/callbackUrl}": &parser.PathItem{
						Post: &parser.Operation{
							OperationID: "paymentCallback",
							Summary:     "Payment callback endpoint",
							Parameters: []*parser.Parameter{
								{Name: "X-Signature", In: "header"},
							},
						},
					},
				},
				"onShipmentUpdate": {
					"{$request.body#/shipmentCallback}": &parser.PathItem{
						Post: &parser.Operation{
							OperationID: "shipmentCallback",
							Summary:     "Shipment callback endpoint",
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

	var visitedCallbacks []string
	var visitedOperations []string
	err := Walk(result,
		WithCallbackHandler(func(name string, callback parser.Callback, path string) Action {
			visitedCallbacks = append(visitedCallbacks, name)
			if name == "onPaymentComplete" {
				return SkipChildren // Skip children of first callback
			}
			return Continue
		}),
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedOperations = append(visitedOperations, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	// Both callbacks should be visited
	assert.Len(t, visitedCallbacks, 2)
	assert.Contains(t, visitedCallbacks, "onPaymentComplete")
	assert.Contains(t, visitedCallbacks, "onShipmentUpdate")
	// Only the second callback's operations should be visited
	assert.Len(t, visitedOperations, 1)
	assert.Contains(t, visitedOperations, "shipmentCallback")
}

func TestWalk_OAS3ComponentPathItemsWithOperations(t *testing.T) {
	// Test component path items (OAS 3.1+) with full operation traversal
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			PathItems: map[string]*parser.PathItem{
				"SharedHealthCheck": {
					Get: &parser.Operation{
						OperationID: "healthCheck",
						Summary:     "Health check endpoint",
						Responses: &parser.Responses{
							Codes: map[string]*parser.Response{
								"200": {Description: "OK"},
							},
						},
					},
				},
				"SharedCRUD": {
					Get: &parser.Operation{
						OperationID: "listItems",
						Summary:     "List items",
					},
					Post: &parser.Operation{
						OperationID: "createItem",
						Summary:     "Create item",
						RequestBody: &parser.RequestBody{
							Content: map[string]*parser.MediaType{
								"application/json": {
									Schema: &parser.Schema{Type: "object"},
								},
							},
						},
					},
					Put: &parser.Operation{
						OperationID: "updateItem",
						Summary:     "Update item",
					},
					Delete: &parser.Operation{
						OperationID: "deleteItem",
						Summary:     "Delete item",
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
	var visitedOperations []string
	var requestBodyCount int
	err := Walk(result,
		WithPathItemHandler(func(pathItem *parser.PathItem, path string) Action {
			visitedPathItems = append(visitedPathItems, path)
			return Continue
		}),
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedOperations = append(visitedOperations, op.OperationID)
			return Continue
		}),
		WithRequestBodyHandler(func(reqBody *parser.RequestBody, path string) Action {
			requestBodyCount++
			return Continue
		}),
	)

	require.NoError(t, err)
	// Both component path items should be visited
	assert.Len(t, visitedPathItems, 2)
	// All 5 operations should be visited
	assert.Len(t, visitedOperations, 5)
	assert.Contains(t, visitedOperations, "healthCheck")
	assert.Contains(t, visitedOperations, "listItems")
	assert.Contains(t, visitedOperations, "createItem")
	assert.Contains(t, visitedOperations, "updateItem")
	assert.Contains(t, visitedOperations, "deleteItem")
	// One request body should be visited
	assert.Equal(t, 1, requestBodyCount)
}

// Tests for walkOAS3Webhooks - Coverage improvements

func TestWalk_OAS3WebhooksWithOperations(t *testing.T) {
	// Test webhooks with full operation traversal
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Webhooks: map[string]*parser.PathItem{
			"newOrder": {
				Post: &parser.Operation{
					OperationID: "newOrderWebhook",
					Summary:     "New order webhook",
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{
									Type: "object",
									Properties: map[string]*parser.Schema{
										"orderId": {Type: "string"},
										"amount":  {Type: "number"},
									},
								},
							},
						},
					},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "Webhook received"},
						},
					},
				},
			},
			"orderCancelled": {
				Post: &parser.Operation{
					OperationID: "orderCancelledWebhook",
					Summary:     "Order cancelled webhook",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedWebhooks []string
	var visitedOperations []string
	var requestBodyCount int
	var responseCount int
	err := Walk(result,
		WithPathItemHandler(func(pathItem *parser.PathItem, path string) Action {
			if strings.Contains(path, "webhooks") {
				visitedWebhooks = append(visitedWebhooks, path)
			}
			return Continue
		}),
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedOperations = append(visitedOperations, op.OperationID)
			return Continue
		}),
		WithRequestBodyHandler(func(reqBody *parser.RequestBody, path string) Action {
			requestBodyCount++
			return Continue
		}),
		WithResponseHandler(func(statusCode string, resp *parser.Response, path string) Action {
			responseCount++
			return Continue
		}),
	)

	require.NoError(t, err)
	// Both webhooks should be visited
	assert.Len(t, visitedWebhooks, 2)
	// Both operations should be visited
	assert.Len(t, visitedOperations, 2)
	assert.Contains(t, visitedOperations, "newOrderWebhook")
	assert.Contains(t, visitedOperations, "orderCancelledWebhook")
	// One request body from newOrder webhook
	assert.Equal(t, 1, requestBodyCount)
	// One response from newOrder webhook
	assert.Equal(t, 1, responseCount)
}

func TestWalk_OAS3WebhooksSkipChildren(t *testing.T) {
	// Test SkipChildren from webhook's PathItem handler
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Webhooks: map[string]*parser.PathItem{
			"aWebhook": {
				Post: &parser.Operation{
					OperationID: "aWebhookOp",
					Parameters: []*parser.Parameter{
						{Name: "X-Signature", In: "header"},
					},
				},
			},
			"bWebhook": {
				Post: &parser.Operation{
					OperationID: "bWebhookOp",
					Parameters: []*parser.Parameter{
						{Name: "X-Timestamp", In: "header"},
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

	var visitedWebhooks []string
	var visitedOperations []string
	err := Walk(result,
		WithPathItemHandler(func(pathItem *parser.PathItem, path string) Action {
			if strings.Contains(path, "webhooks") {
				visitedWebhooks = append(visitedWebhooks, path)
				if strings.Contains(path, "aWebhook") {
					return SkipChildren // Skip first webhook's operations
				}
			}
			return Continue
		}),
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedOperations = append(visitedOperations, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	// Both webhooks should be visited
	assert.Len(t, visitedWebhooks, 2)
	// Only second webhook's operation should be visited
	assert.Len(t, visitedOperations, 1)
	assert.Contains(t, visitedOperations, "bWebhookOp")
}

func TestWalk_OAS3WebhooksStop(t *testing.T) {
	// Test Stop from webhook handler
	// Order in walkOAS3: Paths -> Webhooks -> Components -> Tags
	// So stopping at webhooks should prevent components from being visited
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Webhooks: map[string]*parser.PathItem{
			"aWebhook": {
				Post: &parser.Operation{OperationID: "aOp"},
			},
			"bWebhook": {
				Post: &parser.Operation{OperationID: "bOp"},
			},
			"cWebhook": {
				Post: &parser.Operation{OperationID: "cOp"},
			},
		},
		// Components should NOT be visited after Stop in webhooks
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"ShouldNotVisit": {Type: "object"},
			},
		},
		// Tags should NOT be visited either
		Tags: []*parser.Tag{
			{Name: "shouldNotVisit"},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var visitedWebhooks []string
	var schemaVisited bool
	var tagVisited bool
	err := Walk(result,
		WithPathItemHandler(func(pathItem *parser.PathItem, path string) Action {
			if strings.Contains(path, "webhooks") {
				visitedWebhooks = append(visitedWebhooks, path)
				return Stop // Stop after first webhook
			}
			return Continue
		}),
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaVisited = true
			return Continue
		}),
		WithTagHandler(func(tag *parser.Tag, path string) Action {
			tagVisited = true
			return Continue
		}),
	)

	require.NoError(t, err)
	// Only one webhook should be visited due to Stop
	assert.Len(t, visitedWebhooks, 1)
	// Components and Tags come after webhooks in the traversal order, so they should NOT be visited
	assert.False(t, schemaVisited, "schemas should not be visited after Stop in webhooks")
	assert.False(t, tagVisited, "tags should not be visited after Stop in webhooks")
}

// Additional tests for walkOAS3PathItemOperations edge cases

func TestWalk_OAS3QueryMethod(t *testing.T) {
	// Test OAS 3.2+ Query method
	doc := &parser.OAS3Document{
		OpenAPI: "3.2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/search": &parser.PathItem{
				Query: &parser.Operation{
					OperationID: "searchQuery",
					Summary:     "Search using QUERY method",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.2.0",
		OASVersion: parser.OASVersion320,
		Document:   doc,
	}

	var visitedMethods []string
	err := Walk(result,
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedMethods = append(visitedMethods, method)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Contains(t, visitedMethods, "query")
}

func TestWalk_OAS3AdditionalOperations(t *testing.T) {
	// Test OAS 3.2+ AdditionalOperations
	doc := &parser.OAS3Document{
		OpenAPI: "3.2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/custom": &parser.PathItem{
				Get: &parser.Operation{OperationID: "getCustom"},
				AdditionalOperations: map[string]*parser.Operation{
					"customMethod1": {OperationID: "customOp1"},
					"customMethod2": {OperationID: "customOp2"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.2.0",
		OASVersion: parser.OASVersion320,
		Document:   doc,
	}

	var visitedMethods []string
	var visitedOpIDs []string
	err := Walk(result,
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedMethods = append(visitedMethods, method)
			visitedOpIDs = append(visitedOpIDs, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	// Standard GET plus 2 additional operations
	assert.Len(t, visitedMethods, 3)
	assert.Contains(t, visitedMethods, "get")
	assert.Contains(t, visitedMethods, "customMethod1")
	assert.Contains(t, visitedMethods, "customMethod2")
	assert.Contains(t, visitedOpIDs, "getCustom")
	assert.Contains(t, visitedOpIDs, "customOp1")
	assert.Contains(t, visitedOpIDs, "customOp2")
}

func TestWalk_OAS3AdditionalOperationsStop(t *testing.T) {
	// Test Stop during AdditionalOperations traversal
	doc := &parser.OAS3Document{
		OpenAPI: "3.2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/custom": &parser.PathItem{
				AdditionalOperations: map[string]*parser.Operation{
					"aMethod": {OperationID: "aOp"},
					"bMethod": {OperationID: "bOp"},
					"cMethod": {OperationID: "cOp"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.2.0",
		OASVersion: parser.OASVersion320,
		Document:   doc,
	}

	var visitedOpIDs []string
	err := Walk(result,
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedOpIDs = append(visitedOpIDs, op.OperationID)
			return Stop // Stop after first additional operation
		}),
	)

	require.NoError(t, err)
	// Only one operation should be visited due to Stop
	assert.Len(t, visitedOpIDs, 1)
}

func TestWalk_OAS3StopDuringOperationLoop(t *testing.T) {
	// Test that Stop during the operation loop (w.stopped check) prevents visiting remaining methods
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/resource": &parser.PathItem{
				Get:    &parser.Operation{OperationID: "getOp"},
				Put:    &parser.Operation{OperationID: "putOp"},
				Post:   &parser.Operation{OperationID: "postOp"},
				Delete: &parser.Operation{OperationID: "deleteOp"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	var visitedMethods []string
	err := Walk(result,
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedMethods = append(visitedMethods, method)
			// Stop after the first operation to exercise the w.stopped check in the loop
			return Stop
		}),
	)

	require.NoError(t, err)
	// Only one method should be visited due to Stop
	assert.Len(t, visitedMethods, 1)
}

// OAS 2.0 PathItem Walking Tests

func TestWalk_OAS2PathItemAllMethods(t *testing.T) {
	// Test walking a PathItem with all HTTP methods defined
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/resource": &parser.PathItem{
				Get:     &parser.Operation{OperationID: "getResource"},
				Put:     &parser.Operation{OperationID: "putResource"},
				Post:    &parser.Operation{OperationID: "postResource"},
				Delete:  &parser.Operation{OperationID: "deleteResource"},
				Options: &parser.Operation{OperationID: "optionsResource"},
				Head:    &parser.Operation{OperationID: "headResource"},
				Patch:   &parser.Operation{OperationID: "patchResource"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var visitedMethods []string
	err := Walk(result,
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedMethods = append(visitedMethods, method)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Len(t, visitedMethods, 7, "expected 7 HTTP methods")
	assert.Contains(t, visitedMethods, "get")
	assert.Contains(t, visitedMethods, "put")
	assert.Contains(t, visitedMethods, "post")
	assert.Contains(t, visitedMethods, "delete")
	assert.Contains(t, visitedMethods, "options")
	assert.Contains(t, visitedMethods, "head")
	assert.Contains(t, visitedMethods, "patch")
}

func TestWalk_OAS2PathItemWithPathLevelParameters(t *testing.T) {
	// Test walking a PathItem with path-level parameters
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/users/{userId}/posts/{postId}": &parser.PathItem{
				Parameters: []*parser.Parameter{
					{Name: "userId", In: "path", Type: "string", Required: true},
					{Name: "postId", In: "path", Type: "integer", Required: true},
				},
				Get: &parser.Operation{
					OperationID: "getPost",
					Parameters: []*parser.Parameter{
						{Name: "includeComments", In: "query", Type: "boolean"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var visitedParams []string
	var paramPaths []string
	err := Walk(result,
		WithParameterHandler(func(param *parser.Parameter, path string) Action {
			visitedParams = append(visitedParams, param.Name)
			paramPaths = append(paramPaths, path)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Len(t, visitedParams, 3, "expected 3 parameters (2 path-level + 1 operation-level)")
	assert.Contains(t, visitedParams, "userId")
	assert.Contains(t, visitedParams, "postId")
	assert.Contains(t, visitedParams, "includeComments")

	// Verify path-level parameters have correct path format
	pathLevelCount := 0
	for _, p := range paramPaths {
		if strings.Contains(p, "parameters[0]") || strings.Contains(p, "parameters[1]") {
			if strings.Contains(p, ".get.") {
				continue
			}
			pathLevelCount++
		}
	}
	assert.GreaterOrEqual(t, pathLevelCount, 2, "expected at least 2 path-level parameters")
}

func TestWalk_OAS2PathItemSkipChildren(t *testing.T) {
	// Test that SkipChildren from PathItem handler skips operations
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/public": &parser.PathItem{
				Get: &parser.Operation{OperationID: "publicGet"},
			},
			"/internal": &parser.PathItem{
				Get:  &parser.Operation{OperationID: "internalGet"},
				Post: &parser.Operation{OperationID: "internalPost"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var visitedOps []string
	err := Walk(result,
		WithPathItemHandler(func(pathItem *parser.PathItem, path string) Action {
			if strings.Contains(path, "/internal") {
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
	assert.Len(t, visitedOps, 1, "expected only 1 operation (internal ones should be skipped)")
	assert.Equal(t, "publicGet", visitedOps[0])
}

func TestWalk_OAS2PathItemStop(t *testing.T) {
	// Test that Stop from PathItem handler stops the entire walk
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/a": &parser.PathItem{Get: &parser.Operation{OperationID: "opA"}},
			"/b": &parser.PathItem{Get: &parser.Operation{OperationID: "opB"}},
			"/c": &parser.PathItem{Get: &parser.Operation{OperationID: "opC"}},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var visitedPathItems int
	err := Walk(result,
		WithPathItemHandler(func(pathItem *parser.PathItem, path string) Action {
			visitedPathItems++
			return Stop // Stop immediately on first PathItem
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, 1, visitedPathItems, "expected only 1 PathItem to be visited before Stop")
}

// OAS 2.0 Operation Walking Tests

func TestWalk_OAS2OperationWithExternalDocs(t *testing.T) {
	// Test walking an operation with ExternalDocs
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					ExternalDocs: &parser.ExternalDocs{
						Description: "More information about listing pets",
						URL:         "https://example.com/docs/list-pets",
					},
				},
				Post: &parser.Operation{
					OperationID: "createPet",
					ExternalDocs: &parser.ExternalDocs{
						URL: "https://example.com/docs/create-pet",
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var externalDocsURLs []string
	var externalDocsPaths []string
	err := Walk(result,
		WithExternalDocsHandler(func(extDocs *parser.ExternalDocs, path string) Action {
			externalDocsURLs = append(externalDocsURLs, extDocs.URL)
			externalDocsPaths = append(externalDocsPaths, path)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Len(t, externalDocsURLs, 2, "expected 2 ExternalDocs")
	assert.Contains(t, externalDocsURLs, "https://example.com/docs/list-pets")
	assert.Contains(t, externalDocsURLs, "https://example.com/docs/create-pet")

	// Verify paths are correct for operation-level external docs
	for _, p := range externalDocsPaths {
		assert.Contains(t, p, ".externalDocs", "path should include .externalDocs")
	}
}

func TestWalk_OAS2OperationWithParameters(t *testing.T) {
	// Test walking an operation with multiple parameters
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/search": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "search",
					Parameters: []*parser.Parameter{
						{Name: "q", In: "query", Type: "string", Required: true},
						{Name: "page", In: "query", Type: "integer"},
						{Name: "limit", In: "query", Type: "integer"},
						{Name: "sort", In: "query", Type: "string"},
					},
				},
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
	assert.Len(t, visitedParams, 4)
	assert.Contains(t, visitedParams, "q")
	assert.Contains(t, visitedParams, "page")
	assert.Contains(t, visitedParams, "limit")
	assert.Contains(t, visitedParams, "sort")
}

func TestWalk_OAS2OperationWithResponses(t *testing.T) {
	// Test walking an operation with multiple responses
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Default: &parser.Response{
							Description: "Unexpected error",
						},
						Codes: map[string]*parser.Response{
							"200": {Description: "OK"},
							"400": {Description: "Bad Request"},
							"404": {Description: "Not Found"},
							"500": {Description: "Internal Server Error"},
						},
					},
				},
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
	assert.Len(t, visitedResponses, 5, "expected 5 responses (1 default + 4 status codes)")
	assert.Contains(t, visitedResponses, "default")
	assert.Contains(t, visitedResponses, "200")
	assert.Contains(t, visitedResponses, "400")
	assert.Contains(t, visitedResponses, "404")
	assert.Contains(t, visitedResponses, "500")
}

func TestWalk_OAS2OperationSkipChildren(t *testing.T) {
	// Test that SkipChildren from Operation handler skips parameters/responses
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Parameters: []*parser.Parameter{
						{Name: "limit", In: "query", Type: "integer"},
					},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "OK"},
						},
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
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var visitedParams []string
	var visitedResponses []string
	err := Walk(result,
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			if op.OperationID == "listPets" {
				return SkipChildren // Skip children for listPets
			}
			return Continue
		}),
		WithParameterHandler(func(param *parser.Parameter, path string) Action {
			visitedParams = append(visitedParams, param.Name)
			return Continue
		}),
		WithResponseHandler(func(statusCode string, resp *parser.Response, path string) Action {
			visitedResponses = append(visitedResponses, statusCode)
			return Continue
		}),
	)

	require.NoError(t, err)
	// Only createPet's parameter should be visited
	assert.Equal(t, []string{"body"}, visitedParams, "only createPet parameter should be visited")
	// listPets responses should be skipped
	assert.Empty(t, visitedResponses, "no responses should be visited due to SkipChildren")
}

func TestWalk_OAS2OperationStop(t *testing.T) {
	// Test that Stop from Operation handler stops the entire walk
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/a": &parser.PathItem{
				Get:  &parser.Operation{OperationID: "getA"},
				Post: &parser.Operation{OperationID: "postA"},
			},
			"/b": &parser.PathItem{
				Get: &parser.Operation{OperationID: "getB"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var visitedOps []string
	err := Walk(result,
		WithOperationHandler(func(method string, op *parser.Operation, path string) Action {
			visitedOps = append(visitedOps, op.OperationID)
			if len(visitedOps) >= 2 {
				return Stop // Stop after visiting 2 operations
			}
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Len(t, visitedOps, 2, "expected walk to stop after 2 operations")
}

// OAS 2.0 Response Walking Tests

func TestWalk_OAS2ResponseWithHeaders(t *testing.T) {
	// Test walking a response with headers
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Headers: map[string]*parser.Header{
									"X-Rate-Limit":     {Type: "integer", Description: "Rate limit"},
									"X-Rate-Remaining": {Type: "integer", Description: "Remaining requests"},
									"X-Request-ID":     {Type: "string", Description: "Request ID"},
								},
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
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
	assert.Len(t, visitedHeaders, 3, "expected 3 headers")
	assert.Contains(t, visitedHeaders, "X-Rate-Limit")
	assert.Contains(t, visitedHeaders, "X-Rate-Remaining")
	assert.Contains(t, visitedHeaders, "X-Request-ID")
}

func TestWalk_OAS2ResponseWithSchema(t *testing.T) {
	// Test walking a response with a schema
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Schema: &parser.Schema{
									Type: "array",
									Items: &parser.Schema{
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
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaPaths = append(schemaPaths, path)
			return Continue
		}),
	)

	require.NoError(t, err)
	// Should visit: response schema, items schema, id property, name property
	assert.GreaterOrEqual(t, len(schemaPaths), 4, "expected at least 4 schemas")

	// Verify response schema path
	foundResponseSchema := false
	for _, p := range schemaPaths {
		if strings.Contains(p, "responses['200'].schema") {
			foundResponseSchema = true
			break
		}
	}
	assert.True(t, foundResponseSchema, "should visit response schema")
}

func TestWalk_OAS2ResponseSkipChildren(t *testing.T) {
	// Test that SkipChildren from Response handler skips headers and schema
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK - skip this one",
								Headers: map[string]*parser.Header{
									"X-Skip": {Type: "string"},
								},
								Schema: &parser.Schema{Type: "object"},
							},
							"400": {
								Description: "Bad Request - visit this one",
								Headers: map[string]*parser.Header{
									"X-Error": {Type: "string"},
								},
								Schema: &parser.Schema{Type: "object"},
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var visitedHeaders []string
	var visitedSchemas []string
	err := Walk(result,
		WithResponseHandler(func(statusCode string, resp *parser.Response, path string) Action {
			if statusCode == "200" {
				return SkipChildren
			}
			return Continue
		}),
		WithHeaderHandler(func(name string, header *parser.Header, path string) Action {
			visitedHeaders = append(visitedHeaders, name)
			return Continue
		}),
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			visitedSchemas = append(visitedSchemas, path)
			return Continue
		}),
	)

	require.NoError(t, err)
	// Only 400 response's children should be visited
	assert.Equal(t, []string{"X-Error"}, visitedHeaders, "only X-Error header should be visited")
	assert.Len(t, visitedSchemas, 1, "only 400 response schema should be visited")
	for _, p := range visitedSchemas {
		assert.Contains(t, p, "400", "visited schema should be from 400 response")
	}
}

// walkParameter Coverage Tests

func TestWalk_ParameterWithSchema(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets/{id}": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPet",
					Parameters: []*parser.Parameter{
						{
							Name:   "id",
							In:     "path",
							Schema: &parser.Schema{Type: "integer"},
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

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaPaths = append(schemaPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range schemaPaths {
		if strings.Contains(p, "parameters[0].schema") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit parameter schema")
}

func TestWalk_ParameterWithContent(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Parameters: []*parser.Parameter{
						{
							Name: "filter",
							In:   "query",
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
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var mediaTypePaths []string
	err := Walk(result,
		WithMediaTypeHandler(func(name string, mt *parser.MediaType, path string) Action {
			mediaTypePaths = append(mediaTypePaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range mediaTypePaths {
		if strings.Contains(p, "parameters[0].content") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit parameter content media type")
}

func TestWalk_ParameterWithExamples(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets/{id}": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPet",
					Parameters: []*parser.Parameter{
						{
							Name: "id",
							In:   "path",
							Examples: map[string]*parser.Example{
								"petId1": {Summary: "First pet", Value: 1},
								"petId2": {Summary: "Second pet", Value: 2},
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

	assert.Len(t, exampleNames, 2)
	assert.Contains(t, exampleNames, "petId1")
	assert.Contains(t, exampleNames, "petId2")
}

func TestWalk_ParameterSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets/{id}": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPet",
					Parameters: []*parser.Parameter{
						{
							Name:   "id",
							In:     "path",
							Schema: &parser.Schema{Type: "integer"},
							Examples: map[string]*parser.Example{
								"example1": {Summary: "Example"},
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

	schemaVisited := false
	exampleVisited := false
	err := Walk(result,
		WithParameterHandler(func(param *parser.Parameter, path string) Action {
			return SkipChildren
		}),
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			if strings.Contains(path, "parameters") {
				schemaVisited = true
			}
			return Continue
		}),
		WithExampleHandler(func(name string, example *parser.Example, path string) Action {
			if strings.Contains(path, "parameters") {
				exampleVisited = true
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.False(t, schemaVisited, "schema should not be visited when parameter handler returns SkipChildren")
	assert.False(t, exampleVisited, "example should not be visited when parameter handler returns SkipChildren")
}

func TestWalk_ParameterStop(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Parameters: []*parser.Parameter{
						{Name: "limit", In: "query"},
						{Name: "offset", In: "query"},
						{Name: "filter", In: "query"},
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

	var visitedParams []string
	err := Walk(result,
		WithParameterHandler(func(param *parser.Parameter, path string) Action {
			visitedParams = append(visitedParams, param.Name)
			if param.Name == "limit" {
				return Stop
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedParams, 1, "should stop after first parameter")
	assert.Equal(t, "limit", visitedParams[0])
}

// walkHeader Coverage Tests

func TestWalk_HeaderWithContent(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"X-Custom-Header": {
					Description: "Custom header with content",
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

	var mediaTypePaths []string
	err := Walk(result,
		WithMediaTypeHandler(func(name string, mt *parser.MediaType, path string) Action {
			mediaTypePaths = append(mediaTypePaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range mediaTypePaths {
		if strings.Contains(p, "headers") && strings.Contains(p, "content") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit header content media type")
}

func TestWalk_HeaderWithExamples(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"X-Request-ID": {
					Description: "Request ID header",
					Schema:      &parser.Schema{Type: "string"},
					Examples: map[string]*parser.Example{
						"uuid1": {Summary: "UUID example", Value: "123e4567-e89b-12d3-a456-426614174000"},
						"uuid2": {Summary: "Another UUID", Value: "987fcdeb-51a2-43e6-b7c8-123456789abc"},
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

	assert.Len(t, exampleNames, 2)
	assert.Contains(t, exampleNames, "uuid1")
	assert.Contains(t, exampleNames, "uuid2")
}

func TestWalk_HeaderSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"X-Rate-Limit": {
					Description: "Rate limit header",
					Schema:      &parser.Schema{Type: "integer"},
					Examples: map[string]*parser.Example{
						"example1": {Summary: "Example"},
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

	schemaVisited := false
	exampleVisited := false
	err := Walk(result,
		WithHeaderHandler(func(name string, header *parser.Header, path string) Action {
			return SkipChildren
		}),
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			if strings.Contains(path, "headers") {
				schemaVisited = true
			}
			return Continue
		}),
		WithExampleHandler(func(name string, example *parser.Example, path string) Action {
			if strings.Contains(path, "headers") {
				exampleVisited = true
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.False(t, schemaVisited, "schema should not be visited when header handler returns SkipChildren")
	assert.False(t, exampleVisited, "example should not be visited when header handler returns SkipChildren")
}

// walkSchemaProperties Coverage Tests

func TestWalk_SchemaDependentSchemas(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"ConditionalObject": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name":    {Type: "string"},
						"address": {Type: "string"},
					},
					DependentSchemas: map[string]*parser.Schema{
						"name": {
							Properties: map[string]*parser.Schema{
								"firstName": {Type: "string"},
								"lastName":  {Type: "string"},
							},
						},
						"address": {
							Properties: map[string]*parser.Schema{
								"street": {Type: "string"},
								"city":   {Type: "string"},
							},
						},
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

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaPaths = append(schemaPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	// Count dependentSchemas entries
	dependentSchemaCount := 0
	for _, p := range schemaPaths {
		if strings.Contains(p, "dependentSchemas") {
			dependentSchemaCount++
		}
	}
	// Should have 2 dependentSchemas (name and address) plus their nested properties
	assert.GreaterOrEqual(t, dependentSchemaCount, 2, "should visit dependentSchemas")
}

func TestWalk_SchemaPropertiesSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Object": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"nested": {
							Type: "object",
							Properties: map[string]*parser.Schema{
								"deep": {Type: "string"},
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

	var visitedPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			visitedPaths = append(visitedPaths, path)
			// Skip children of the nested schema
			if strings.Contains(path, "nested") && !strings.Contains(path, "deep") {
				return SkipChildren
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	// Should visit Object and nested, but not deep
	deepVisited := false
	for _, p := range visitedPaths {
		if strings.Contains(p, "deep") {
			deepVisited = true
		}
	}
	assert.False(t, deepVisited, "deep schema should not be visited when parent returns SkipChildren")
}

func TestWalk_SchemaPropertiesStop(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Object": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"a": {Type: "string"},
						"b": {Type: "string"},
						"c": {Type: "string"},
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

	visitCount := 0
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			visitCount++
			// Stop after visiting 2 schemas (Object + first property)
			if visitCount >= 2 {
				return Stop
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Equal(t, 2, visitCount, "should stop after 2 schemas")
}

// walkSchemaArrayKeywords Coverage Tests

func TestWalk_SchemaItemsAsSchema(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"StringArray": {
					Type:  "array",
					Items: &parser.Schema{Type: "string"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaPaths = append(schemaPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range schemaPaths {
		if strings.Contains(p, ".items") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit items schema")
}

func TestWalk_SchemaAdditionalItems(t *testing.T) {
	// AdditionalItems is a JSON Schema draft-07 keyword still sometimes used
	additionalItemsSchema := &parser.Schema{Type: "number"}
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"TupleWithExtra": {
					Type: "array",
					PrefixItems: []*parser.Schema{
						{Type: "string"},
						{Type: "integer"},
					},
					AdditionalItems: additionalItemsSchema,
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaPaths = append(schemaPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range schemaPaths {
		if strings.Contains(p, "additionalItems") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit additionalItems schema")
}

func TestWalk_SchemaUnevaluatedItems(t *testing.T) {
	unevaluatedItemsSchema := &parser.Schema{Type: "boolean"}
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"StrictArray": {
					Type: "array",
					PrefixItems: []*parser.Schema{
						{Type: "string"},
					},
					UnevaluatedItems: unevaluatedItemsSchema,
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaPaths = append(schemaPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range schemaPaths {
		if strings.Contains(p, "unevaluatedItems") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit unevaluatedItems schema")
}

func TestWalk_SchemaArrayKeywordsSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Array": {
					Type: "array",
					Items: &parser.Schema{
						Type: "object",
						Properties: map[string]*parser.Schema{
							"nested": {Type: "string"},
						},
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
			// Skip children of items schema
			if strings.Contains(path, ".items") {
				return SkipChildren
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	nestedVisited := false
	for _, p := range visitedPaths {
		if strings.Contains(p, "nested") {
			nestedVisited = true
		}
	}
	assert.False(t, nestedVisited, "nested property should not be visited when items returns SkipChildren")
}

// walkSchemaMisc Coverage Tests

func TestWalk_SchemaContentSchema(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"EncodedContent": {
					Type:             "string",
					ContentEncoding:  "base64",
					ContentMediaType: "application/json",
					ContentSchema: &parser.Schema{
						Type: "object",
						Properties: map[string]*parser.Schema{
							"data": {Type: "string"},
						},
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

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaPaths = append(schemaPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range schemaPaths {
		if strings.Contains(p, "contentSchema") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit contentSchema")
}

func TestWalk_SchemaMiscSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"WithContentSchema": {
					Type:            "string",
					ContentEncoding: "base64",
					ContentSchema: &parser.Schema{
						Type: "object",
						Properties: map[string]*parser.Schema{
							"nested": {Type: "string"},
						},
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
			// Skip children of the root schema (which has contentSchema)
			if strings.HasSuffix(path, "['WithContentSchema']") {
				return SkipChildren
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	contentSchemaVisited := false
	for _, p := range visitedPaths {
		if strings.Contains(p, "contentSchema") {
			contentSchemaVisited = true
		}
	}
	assert.False(t, contentSchemaVisited, "contentSchema should not be visited when parent returns SkipChildren")
}

func TestWalk_SchemaNotKeyword(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"NotNull": {
					Not: &parser.Schema{Type: "null"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			schemaPaths = append(schemaPaths, path)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range schemaPaths {
		if strings.Contains(p, ".not") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit not schema")
}

// Additional edge case tests for better coverage

func TestWalk_HeaderStop(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"A-Header": {Description: "First header"},
				"B-Header": {Description: "Second header"},
				"C-Header": {Description: "Third header"},
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
			// Stop after first header
			return Stop
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedHeaders, 1, "should stop after first header")
}

func TestWalk_NilParameterInSlice(t *testing.T) {
	// Test that nil parameters in a slice are handled gracefully
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Parameters: []*parser.Parameter{
						{Name: "valid", In: "query"},
						nil, // nil parameter
						{Name: "another", In: "query"},
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

	var visitedParams []string
	err := Walk(result,
		WithParameterHandler(func(param *parser.Parameter, path string) Action {
			visitedParams = append(visitedParams, param.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	// Should visit only non-nil parameters
	assert.Len(t, visitedParams, 2)
	assert.Contains(t, visitedParams, "valid")
	assert.Contains(t, visitedParams, "another")
}

func TestWalk_MediaTypeSkipChildren(t *testing.T) {
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
										Schema: &parser.Schema{Type: "object"},
										Examples: map[string]*parser.Example{
											"example1": {Summary: "Test"},
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

	schemaVisited := false
	exampleVisited := false
	err := Walk(result,
		WithMediaTypeHandler(func(name string, mt *parser.MediaType, path string) Action {
			return SkipChildren
		}),
		WithSchemaHandler(func(schema *parser.Schema, path string) Action {
			if strings.Contains(path, "content") {
				schemaVisited = true
			}
			return Continue
		}),
		WithExampleHandler(func(name string, example *parser.Example, path string) Action {
			if strings.Contains(path, "content") {
				exampleVisited = true
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.False(t, schemaVisited, "schema should not be visited when mediaType returns SkipChildren")
	assert.False(t, exampleVisited, "example should not be visited when mediaType returns SkipChildren")
}
