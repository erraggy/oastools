// walker_components_test.go - Tests for OAS3 component section traversal
// Tests component responses, parameters, request bodies, callbacks, links,
// path items, examples, headers, and security schemes.

package walker

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
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
		WithRequestBodyHandler(func(wc *WalkContext, body *parser.RequestBody) Action {
			visitedPaths = append(visitedPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	require.Len(t, visitedPaths, 1)
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
		WithCallbackHandler(func(wc *WalkContext, callback parser.Callback) Action {
			visitedCallbacks = append(visitedCallbacks, wc.Name)
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
		WithLinkHandler(func(wc *WalkContext, link *parser.Link) Action {
			visitedLinks = append(visitedLinks, wc.Name)
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
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			visitedPathItems = append(visitedPathItems, wc.JSONPath)
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
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			visitedExamples = append(visitedExamples, wc.Name)
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
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			visitedHeaders = append(visitedHeaders, wc.Name)
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
		WithSecuritySchemeHandler(func(wc *WalkContext, scheme *parser.SecurityScheme) Action {
			visitedSchemes = append(visitedSchemes, wc.Name)
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
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			visited["schema"] = true
			return Continue
		}),
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			visited["response"] = true
			return Continue
		}),
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visited["parameter"] = true
			return Continue
		}),
		WithRequestBodyHandler(func(wc *WalkContext, body *parser.RequestBody) Action {
			visited["requestBody"] = true
			return Continue
		}),
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			visited["header"] = true
			return Continue
		}),
		WithSecuritySchemeHandler(func(wc *WalkContext, scheme *parser.SecurityScheme) Action {
			visited["securityScheme"] = true
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
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			if strings.Contains(wc.JSONPath, "components.pathItems") {
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
