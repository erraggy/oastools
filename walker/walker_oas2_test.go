// walker_oas2_test.go - Tests for OAS 2.0 (Swagger) specific walker behavior
// Tests definitions, parameters, responses, security definitions, tags, and operations.

package walker

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOps = append(visitedOps, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"listPets"}, visitedOps)
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
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaPaths = append(schemaPaths, wc.JSONPath)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Len(t, schemaPaths, 2)
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
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
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
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			// Component responses use wc.Name (the response key), not wc.StatusCode
			visitedResponses = append(visitedResponses, wc.Name)
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
		WithSecuritySchemeHandler(func(wc *WalkContext, scheme *parser.SecurityScheme) Action {
			visitedSchemes = append(visitedSchemes, wc.Name)
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
		WithTagHandler(func(wc *WalkContext, tag *parser.Tag) Action {
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
		WithExternalDocsHandler(func(wc *WalkContext, docs *parser.ExternalDocs) Action {
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
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			// Only count top-level definitions, not nested properties
			if strings.HasPrefix(wc.JSONPath, "$.definitions['") && strings.Count(wc.JSONPath, ".") == 1 {
				visitedSchemas = append(visitedSchemas, wc.JSONPath)
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	// Should visit User, Error, and their property schemas
	assert.GreaterOrEqual(t, len(visitedSchemas), 2)
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
		WithOAS2DocumentHandler(func(wc *WalkContext, doc *parser.OAS2Document) Action {
			oas2Called = true
			capturedSwagger = doc.Swagger
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.True(t, oas2Called, "OAS2DocumentHandler should be called for OAS 2.0 document")
	assert.Equal(t, "2.0", capturedSwagger)
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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedMethods = append(visitedMethods, wc.Method)
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
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visitedParams = append(visitedParams, param.Name)
			paramPaths = append(paramPaths, wc.JSONPath)
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
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			if strings.Contains(wc.JSONPath, "/internal") {
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
	require.Len(t, visitedOps, 1, "expected only 1 operation (internal ones should be skipped)")
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
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
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
		WithExternalDocsHandler(func(wc *WalkContext, extDocs *parser.ExternalDocs) Action {
			externalDocsURLs = append(externalDocsURLs, extDocs.URL)
			externalDocsPaths = append(externalDocsPaths, wc.JSONPath)
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
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
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
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			visitedResponses = append(visitedResponses, wc.StatusCode)
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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			if op.OperationID == "listPets" {
				return SkipChildren // Skip children for listPets
			}
			return Continue
		}),
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visitedParams = append(visitedParams, param.Name)
			return Continue
		}),
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			visitedResponses = append(visitedResponses, wc.StatusCode)
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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
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
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			visitedHeaders = append(visitedHeaders, wc.Name)
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
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaPaths = append(schemaPaths, wc.JSONPath)
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
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			if wc.StatusCode == "200" {
				return SkipChildren
			}
			return Continue
		}),
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			visitedHeaders = append(visitedHeaders, wc.Name)
			return Continue
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visitedSchemas = append(visitedSchemas, wc.JSONPath)
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

// OAS 2.0 Paths Edge Case Tests

func TestWalk_OAS2EmptyPaths(t *testing.T) {
	// Test walking a document with empty paths map
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths:   parser.Paths{}, // Empty paths
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var pathHandlerCalled bool
	var operationHandlerCalled bool
	err := Walk(result,
		WithPathHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			pathHandlerCalled = true
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			operationHandlerCalled = true
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.False(t, pathHandlerCalled, "path handler should not be called for empty paths")
	assert.False(t, operationHandlerCalled, "operation handler should not be called for empty paths")
}

func TestWalk_OAS2NilPaths(t *testing.T) {
	// Test walking a document with nil paths
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths:   nil,
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var pathHandlerCalled bool
	err := Walk(result,
		WithPathHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			pathHandlerCalled = true
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.False(t, pathHandlerCalled, "path handler should not be called for nil paths")
}

func TestWalk_OAS2StopDuringPathsIteration(t *testing.T) {
	// Test that Stop during paths iteration halts the walk correctly
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/a": &parser.PathItem{Get: &parser.Operation{OperationID: "getA"}},
			"/b": &parser.PathItem{Get: &parser.Operation{OperationID: "getB"}},
			"/c": &parser.PathItem{Get: &parser.Operation{OperationID: "getC"}},
			"/d": &parser.PathItem{Get: &parser.Operation{OperationID: "getD"}},
			"/e": &parser.PathItem{Get: &parser.Operation{OperationID: "getE"}},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var visitedPaths []string
	err := Walk(result,
		WithPathHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			visitedPaths = append(visitedPaths, wc.PathTemplate)
			if len(visitedPaths) >= 2 {
				return Stop
			}
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Len(t, visitedPaths, 2, "expected exactly 2 paths visited before Stop")
}

func TestWalk_OAS2PathItemWithRef(t *testing.T) {
	// Test walking a PathItem that has a $ref
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Ref: "#/x-path-items/PetsPath",
				Get: &parser.Operation{OperationID: "listPets"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	var refCalled bool
	var capturedRef string
	var capturedNodeType RefNodeType
	err := Walk(result,
		WithRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			refCalled = true
			capturedRef = ref.Ref
			capturedNodeType = ref.NodeType
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.True(t, refCalled, "ref handler should be called for PathItem with $ref")
	assert.Equal(t, "#/x-path-items/PetsPath", capturedRef)
	assert.Equal(t, RefNodePathItem, capturedNodeType)
}

func TestWalk_OAS2PathItemWithRefStopsWalk(t *testing.T) {
	// Test that returning Stop from ref handler stops the walk
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/a": &parser.PathItem{
				Ref: "#/x-path-items/PathA",
				Get: &parser.Operation{OperationID: "getA"},
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

	var visitedPaths int
	var visitedOps []string
	err := Walk(result,
		WithRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			return Stop // Stop on first ref
		}),
		WithPathHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			visitedPaths++
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOps = append(visitedOps, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	// The ref on /a should stop the walk before PathItem handler is called
	assert.LessOrEqual(t, visitedPaths, 1, "walk should stop early due to ref handler returning Stop")
}
