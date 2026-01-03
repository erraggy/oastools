// walker_options_test.go - Tests for WalkWithOptions
// Tests the functional options API for the walker.

package walker

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Basic WalkWithOptions Tests

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
		WithDocumentHandler(func(wc *WalkContext, doc any) Action {
			called = true
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.True(t, called)
}

// MaxSchemaDepth Tests

func TestWalkWithOptions_InvalidMaxSchemaDepth(t *testing.T) {
	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0"},
		},
	}

	// WithMaxSchemaDepth silently ignores invalid values (uses default of 100)
	err := WalkWithOptions(
		WithParsed(result),
		WithMaxSchemaDepth(0),
	)
	require.NoError(t, err, "invalid maxDepth should be silently ignored")
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

	// WithMaxSchemaDepth silently ignores invalid values (uses default of 100)
	err := WalkWithOptions(
		WithParsed(result),
		WithMaxSchemaDepth(-5),
	)
	require.NoError(t, err, "negative maxDepth should be silently ignored")
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

// WalkWithOptions Handler Tests - Testing WrapOption with With*Handler options

func TestWalkWithOptions_WithInfoHandler(t *testing.T) {
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
		WithInfoHandler(func(wc *WalkContext, info *parser.Info) Action {
			infoTitle = info.Title
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, "Test API", infoTitle)
}

func TestWalkWithOptions_WithServerHandler(t *testing.T) {
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
		WithServerHandler(func(wc *WalkContext, server *parser.Server) Action {
			serverURLs = append(serverURLs, server.URL)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Len(t, serverURLs, 2)
}

func TestWalkWithOptions_WithTagHandler(t *testing.T) {
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
		WithTagHandler(func(wc *WalkContext, tag *parser.Tag) Action {
			tagNames = append(tagNames, tag.Name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Len(t, tagNames, 2)
}

func TestWalkWithOptions_WithPathHandler(t *testing.T) {
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
		WithPathHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			paths = append(paths, wc.PathTemplate)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Len(t, paths, 2)
}

func TestWalkWithOptions_WithPathItemHandler(t *testing.T) {
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
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			pathItemCount++
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, 1, pathItemCount)
}

func TestWalkWithOptions_WithOperationHandler(t *testing.T) {
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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			methods = append(methods, wc.Method)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Len(t, methods, 2)
}

func TestWalkWithOptions_WithParameterHandler(t *testing.T) {
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
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			paramNames = append(paramNames, param.Name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, paramNames, "id")
}

func TestWalkWithOptions_WithRequestBodyHandler(t *testing.T) {
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
		WithRequestBodyHandler(func(wc *WalkContext, reqBody *parser.RequestBody) Action {
			requestBodyCount++
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, 1, requestBodyCount)
}

func TestWalkWithOptions_WithResponseHandler(t *testing.T) {
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
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			statusCodes = append(statusCodes, wc.StatusCode)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, statusCodes, "200")
}

func TestWalkWithOptions_WithSecuritySchemeHandler(t *testing.T) {
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
		WithSecuritySchemeHandler(func(wc *WalkContext, scheme *parser.SecurityScheme) Action {
			schemeNames = append(schemeNames, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, schemeNames, "bearerAuth")
}

func TestWalkWithOptions_WithHeaderHandler(t *testing.T) {
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
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			headerNames = append(headerNames, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, headerNames, "X-Rate-Limit")
}

func TestWalkWithOptions_WithMediaTypeHandler(t *testing.T) {
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
		WithMediaTypeHandler(func(wc *WalkContext, mt *parser.MediaType) Action {
			mediaTypes = append(mediaTypes, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, mediaTypes, "application/json")
}

func TestWalkWithOptions_WithLinkHandler(t *testing.T) {
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
		WithLinkHandler(func(wc *WalkContext, link *parser.Link) Action {
			linkNames = append(linkNames, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, linkNames, "GetUserById")
}

func TestWalkWithOptions_WithCallbackHandler(t *testing.T) {
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
		WithCallbackHandler(func(wc *WalkContext, callback parser.Callback) Action {
			callbackNames = append(callbackNames, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, callbackNames, "onEvent")
}

func TestWalkWithOptions_WithExampleHandler(t *testing.T) {
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
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			exampleNames = append(exampleNames, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Contains(t, exampleNames, "petExample")
}

func TestWalkWithOptions_WithExternalDocsHandler(t *testing.T) {
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
		WithExternalDocsHandler(func(wc *WalkContext, extDocs *parser.ExternalDocs) Action {
			docsURL = extDocs.URL
			return Continue
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, "https://docs.example.com", docsURL)
}

func TestWalkWithOptions_WithSchemaSkippedHandler(t *testing.T) {
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
		WithSchemaSkippedHandler(func(wc *WalkContext, reason string, schema *parser.Schema) {
			skippedCount++
			lastReason = reason
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, 1, skippedCount, "expected one schema to be skipped")
	assert.Equal(t, "cycle", lastReason, "expected skip reason to be 'cycle'")
}

// WalkWithOptions Typed Document Handler Tests

func TestWalkWithOptions_WithOAS2DocumentHandler(t *testing.T) {
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
		WithOAS2DocumentHandler(func(wc *WalkContext, doc *parser.OAS2Document) Action {
			oas2Called = true
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.True(t, oas2Called, "WithOAS2DocumentHandler should be called for OAS 2.0 document")
}

func TestWalkWithOptions_WithOAS3DocumentHandler(t *testing.T) {
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
		WithOAS3DocumentHandler(func(wc *WalkContext, doc *parser.OAS3Document) Action {
			oas3Called = true
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.True(t, oas3Called, "WithOAS3DocumentHandler should be called for OAS 3.x document")
}
