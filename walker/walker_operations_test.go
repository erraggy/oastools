// walker_operations_test.go - Tests for operation and webhook traversal
// Tests HTTP methods, operation flow control, webhooks, and OAS 3.2+ features
// like Query method and AdditionalOperations.

package walker

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOps = append(visitedOps, op.OperationID)
			return Continue
		}),
	)

	require.NoError(t, err)
	assert.Contains(t, visitedOps, "newPetWebhook")
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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedMethods[wc.Method] = true
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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			if wc.Method == "get" {
				return SkipChildren // Skip GET operation's children
			}
			return Continue
		}),
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visitedParams = append(visitedParams, param.Name)
			return Continue
		}),
		WithRequestBodyHandler(func(wc *WalkContext, reqBody *parser.RequestBody) Action {
			visitedRequestBodies++
			return Continue
		}),
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			visitedResponses = append(visitedResponses, wc.StatusCode)
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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			if wc.Method == "trace" {
				traceVisited = true
				tracePath = wc.JSONPath
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
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visitedParams = append(visitedParams, param.Name)
			return Stop // Stop after first parameter
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
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
		WithRequestBodyHandler(func(wc *WalkContext, reqBody *parser.RequestBody) Action {
			visitedRequestBodies = append(visitedRequestBodies, reqBody.Description)
			return Continue
		}),
		WithMediaTypeHandler(func(wc *WalkContext, mt *parser.MediaType) Action {
			visitedMediaTypes = append(visitedMediaTypes, wc.Name)
			return Continue
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
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
		WithLinkHandler(func(wc *WalkContext, link *parser.Link) Action {
			visitedLinks = append(visitedLinks, wc.Name)
			return Stop // Stop after first link
		}),
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
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
		WithCallbackHandler(func(wc *WalkContext, callback parser.Callback) Action {
			visitedCallbacks = append(visitedCallbacks, wc.Name)
			if wc.Name == "onPaymentComplete" {
				return SkipChildren // Skip children of first callback
			}
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
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
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			visitedPathItems = append(visitedPathItems, wc.JSONPath)
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOperations = append(visitedOperations, op.OperationID)
			return Continue
		}),
		WithRequestBodyHandler(func(wc *WalkContext, reqBody *parser.RequestBody) Action {
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
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			if strings.Contains(wc.JSONPath, "webhooks") {
				visitedWebhooks = append(visitedWebhooks, wc.JSONPath)
			}
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedOperations = append(visitedOperations, op.OperationID)
			return Continue
		}),
		WithRequestBodyHandler(func(wc *WalkContext, reqBody *parser.RequestBody) Action {
			requestBodyCount++
			return Continue
		}),
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
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
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			if strings.Contains(wc.JSONPath, "webhooks") {
				visitedWebhooks = append(visitedWebhooks, wc.JSONPath)
				if strings.Contains(wc.JSONPath, "aWebhook") {
					return SkipChildren // Skip first webhook's operations
				}
			}
			return Continue
		}),
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
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
		WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action {
			if strings.Contains(wc.JSONPath, "webhooks") {
				visitedWebhooks = append(visitedWebhooks, wc.JSONPath)
				return Stop // Stop after first webhook
			}
			return Continue
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaVisited = true
			return Continue
		}),
		WithTagHandler(func(wc *WalkContext, tag *parser.Tag) Action {
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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedMethods = append(visitedMethods, wc.Method)
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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedMethods = append(visitedMethods, wc.Method)
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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
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
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			visitedMethods = append(visitedMethods, wc.Method)
			// Stop after the first operation to exercise the w.stopped check in the loop
			return Stop
		}),
	)

	require.NoError(t, err)
	// Only one method should be visited due to Stop
	assert.Len(t, visitedMethods, 1)
}
