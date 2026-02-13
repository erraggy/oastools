package joiner

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Graph Construction Tests
// =============================================================================

func TestBuildRefGraphOAS3_Empty(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Empty API", Version: "1.0.0"},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	require.NotNil(t, graph)
	assert.Empty(t, graph.schemaRefs)
	assert.Empty(t, graph.operationRefs)
	assert.Empty(t, graph.resolved)
}

func TestBuildRefGraphOAS3_NilDocument(t *testing.T) {
	graph := buildRefGraphOAS3(nil, parser.OASVersion300)

	require.NotNil(t, graph)
	assert.Empty(t, graph.schemaRefs)
	assert.Empty(t, graph.operationRefs)
}

func TestBuildRefGraphOAS3_SingleOperation(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getUsers",
					Tags:        []string{"Users"},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{
											Ref: "#/components/schemas/UserList",
										},
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
				"UserList": {
					Type: "array",
				},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	require.NotNil(t, graph)

	// Verify operationRefs contains UserList schema
	refs, ok := graph.operationRefs["UserList"]
	require.True(t, ok, "Expected operationRefs to contain UserList")
	require.Len(t, refs, 1)

	// Verify the OperationRef details
	opRef := refs[0]
	assert.Equal(t, "/users", opRef.Path)
	assert.Equal(t, "get", opRef.Method)
	assert.Equal(t, "getUsers", opRef.OperationID)
	assert.Equal(t, UsageTypeResponse, opRef.UsageType)
	assert.Equal(t, "200", opRef.StatusCode)
	assert.Equal(t, "application/json", opRef.MediaType)
}

func TestBuildRefGraphOAS3_MultipleUsageTypes(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users/{id}": &parser.PathItem{
				Put: &parser.Operation{
					OperationID: "updateUser",
					Tags:        []string{"Users"},
					Parameters: []*parser.Parameter{
						{
							Name:   "filter",
							In:     "query",
							Schema: &parser.Schema{Ref: "#/components/schemas/FilterParams"},
						},
					},
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{Ref: "#/components/schemas/UserUpdate"},
							},
						},
					},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Ref: "#/components/schemas/User"},
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
				"FilterParams": {Type: "object"},
				"UserUpdate":   {Type: "object"},
				"User":         {Type: "object"},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	// Verify request body schema (UsageRequest)
	requestRefs, ok := graph.operationRefs["UserUpdate"]
	require.True(t, ok, "Expected operationRefs to contain UserUpdate")
	require.Len(t, requestRefs, 1)
	assert.Equal(t, UsageTypeRequest, requestRefs[0].UsageType)

	// Verify response schema (UsageResponse)
	responseRefs, ok := graph.operationRefs["User"]
	require.True(t, ok, "Expected operationRefs to contain User")
	require.Len(t, responseRefs, 1)
	assert.Equal(t, UsageTypeResponse, responseRefs[0].UsageType)

	// Verify parameter schema (UsageParameter)
	paramRefs, ok := graph.operationRefs["FilterParams"]
	require.True(t, ok, "Expected operationRefs to contain FilterParams")
	require.Len(t, paramRefs, 1)
	assert.Equal(t, UsageTypeParameter, paramRefs[0].UsageType)
}

func TestBuildRefGraphOAS3_NestedSchemas(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"SchemaA": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"b": {Ref: "#/components/schemas/SchemaB"},
					},
				},
				"SchemaB": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"c": {Ref: "#/components/schemas/SchemaC"},
					},
				},
				"SchemaC": {
					Type: "string",
				},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	// Verify SchemaB is referenced by SchemaA
	schemaBRefs, ok := graph.schemaRefs["SchemaB"]
	require.True(t, ok, "Expected schemaRefs to contain SchemaB")
	require.Len(t, schemaBRefs, 1)
	assert.Equal(t, "SchemaA", schemaBRefs[0].FromSchema)

	// Verify SchemaC is referenced by SchemaB
	schemaCRefs, ok := graph.schemaRefs["SchemaC"]
	require.True(t, ok, "Expected schemaRefs to contain SchemaC")
	require.Len(t, schemaCRefs, 1)
	assert.Equal(t, "SchemaB", schemaCRefs[0].FromSchema)
}

func TestBuildRefGraphOAS3_CircularRefs(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"SchemaA": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"b": {Ref: "#/components/schemas/SchemaB"},
					},
				},
				"SchemaB": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"a": {Ref: "#/components/schemas/SchemaA"},
					},
				},
			},
		},
	}

	// This should not hang or panic
	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	require.NotNil(t, graph)

	// Verify both refs are recorded
	assert.Contains(t, graph.schemaRefs, "SchemaA")
	assert.Contains(t, graph.schemaRefs, "SchemaB")
}

func TestBuildRefGraphOAS3_MultiOperationRef(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Ref: "#/components/schemas/User"},
									},
								},
							},
						},
					},
				},
				Post: &parser.Operation{
					OperationID: "createUser",
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{Ref: "#/components/schemas/User"},
							},
						},
					},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"201": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Ref: "#/components/schemas/User"},
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
				"User": {Type: "object"},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	refs, ok := graph.operationRefs["User"]
	require.True(t, ok, "Expected operationRefs to contain User")

	// Should have 3 refs: GET 200 response, POST request, POST 201 response
	assert.Len(t, refs, 3)

	// Verify we have different operation IDs
	opIDs := make(map[string]bool)
	for _, ref := range refs {
		opIDs[ref.OperationID] = true
	}
	assert.True(t, opIDs["listUsers"], "Expected listUsers in operations")
	assert.True(t, opIDs["createUser"], "Expected createUser in operations")
}

func TestBuildRefGraphOAS2_Basic(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Schema: &parser.Schema{Ref: "#/definitions/PetList"},
							},
						},
					},
				},
			},
		},
		Definitions: map[string]*parser.Schema{
			"PetList": {
				Type: "array",
				Items: &parser.Schema{
					Ref: "#/definitions/Pet",
				},
			},
			"Pet": {
				Type: "object",
			},
		},
	}

	graph := buildRefGraphOAS2(doc)

	require.NotNil(t, graph)

	// Verify operationRefs contains PetList
	refs, ok := graph.operationRefs["PetList"]
	require.True(t, ok, "Expected operationRefs to contain PetList")
	require.Len(t, refs, 1)
	assert.Equal(t, UsageTypeResponse, refs[0].UsageType)

	// Verify schemaRefs contains Pet (referenced by PetList.items)
	schemaRefs, ok := graph.schemaRefs["Pet"]
	require.True(t, ok, "Expected schemaRefs to contain Pet")
	require.Len(t, schemaRefs, 1)
	assert.Equal(t, "PetList", schemaRefs[0].FromSchema)
}

func TestBuildRefGraphOAS2_NilDocument(t *testing.T) {
	graph := buildRefGraphOAS2(nil)

	require.NotNil(t, graph)
	assert.Empty(t, graph.schemaRefs)
}

func TestBuildRefGraphOAS3_Webhooks(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Webhooks: map[string]*parser.PathItem{
			"newPet": {
				Post: &parser.Operation{
					OperationID: "newPetWebhook",
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{Ref: "#/components/schemas/Pet"},
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

	graph := buildRefGraphOAS3(doc, parser.OASVersion310)

	refs, ok := graph.operationRefs["Pet"]
	require.True(t, ok, "Expected operationRefs to contain Pet from webhook")
	require.Len(t, refs, 1)
	assert.Equal(t, "webhook:newPet", refs[0].Path)
}

func TestBuildRefGraphOAS3_Callbacks(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Orders API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/orders": &parser.PathItem{
				Post: &parser.Operation{
					OperationID: "createOrder",
					Tags:        []string{"Orders"},
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{Ref: "#/components/schemas/OrderRequest"},
							},
						},
					},
					Callbacks: map[string]*parser.Callback{
						"onOrderComplete": {
							"{$request.body#/callbackUrl}": &parser.PathItem{
								Post: &parser.Operation{
									OperationID: "orderCallback",
									RequestBody: &parser.RequestBody{
										Content: map[string]*parser.MediaType{
											"application/json": {
												Schema: &parser.Schema{Ref: "#/components/schemas/OrderCallback"},
											},
										},
									},
									Responses: &parser.Responses{
										Codes: map[string]*parser.Response{
											"200": {
												Content: map[string]*parser.MediaType{
													"application/json": {
														Schema: &parser.Schema{Ref: "#/components/schemas/CallbackResponse"},
													},
												},
											},
										},
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
				"OrderRequest":     {Type: "object"},
				"OrderCallback":    {Type: "object"},
				"CallbackResponse": {Type: "object"},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	// Verify OrderRequest is recorded as a regular request (not callback)
	orderReqRefs, ok := graph.operationRefs["OrderRequest"]
	require.True(t, ok, "Expected operationRefs to contain OrderRequest")
	require.Len(t, orderReqRefs, 1)
	assert.Equal(t, UsageTypeRequest, orderReqRefs[0].UsageType)

	// Verify OrderCallback schema is recorded with callback usage type
	callbackRefs, ok := graph.operationRefs["OrderCallback"]
	require.True(t, ok, "Expected operationRefs to contain OrderCallback from callback request body")
	require.Len(t, callbackRefs, 1)
	assert.Equal(t, UsageTypeCallback, callbackRefs[0].UsageType)
	// Verify callback path format: path->callbackName:callbackPath
	expectedCallbackPath := "/orders->onOrderComplete:{$request.body#/callbackUrl}"
	assert.Equal(t, expectedCallbackPath, callbackRefs[0].Path)
	assert.Equal(t, "orderCallback", callbackRefs[0].OperationID)
	assert.Equal(t, "application/json", callbackRefs[0].MediaType)

	// Verify CallbackResponse schema is recorded from callback response
	respRefs, ok := graph.operationRefs["CallbackResponse"]
	require.True(t, ok, "Expected operationRefs to contain CallbackResponse from callback response")
	require.Len(t, respRefs, 1)
	// Note: callback responses use UsageTypeResponse, not UsageTypeCallback
	assert.Equal(t, UsageTypeResponse, respRefs[0].UsageType)
	assert.Equal(t, "200", respRefs[0].StatusCode)
}

func TestBuildRefGraphOAS3_PathLevelParameters(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users/{id}": &parser.PathItem{
				Parameters: []*parser.Parameter{
					{
						Name:   "id",
						In:     "path",
						Schema: &parser.Schema{Ref: "#/components/schemas/UserId"},
					},
				},
				Get: &parser.Operation{
					OperationID: "getUser",
				},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"UserId": {Type: "string"},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	refs, ok := graph.operationRefs["UserId"]
	require.True(t, ok, "Expected operationRefs to contain UserId from path-level parameter")
	require.Len(t, refs, 1)
	assert.Equal(t, UsageTypeParameter, refs[0].UsageType)
	assert.Equal(t, "id", refs[0].ParamName)
}

func TestBuildRefGraphOAS3_ResponseHeaders(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Ref: "#/components/schemas/UserList"},
									},
								},
								Headers: map[string]*parser.Header{
									"X-Total-Count": {
										Schema: &parser.Schema{Ref: "#/components/schemas/Count"},
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
				"UserList": {Type: "array"},
				"Count":    {Type: "integer"},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	// Verify header schema is recorded
	refs, ok := graph.operationRefs["Count"]
	require.True(t, ok, "Expected operationRefs to contain Count from response header")
	require.Len(t, refs, 1)
	assert.Equal(t, UsageTypeHeader, refs[0].UsageType)
}

func TestBuildRefGraphOAS3_DefaultResponse(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
					Responses: &parser.Responses{
						Default: &parser.Response{
							Content: map[string]*parser.MediaType{
								"application/json": {
									Schema: &parser.Schema{Ref: "#/components/schemas/Error"},
								},
							},
						},
					},
				},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Error": {Type: "object"},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	refs, ok := graph.operationRefs["Error"]
	require.True(t, ok, "Expected operationRefs to contain Error from default response")
	require.Len(t, refs, 1)
	assert.Equal(t, "default", refs[0].StatusCode)
}

// =============================================================================
// Lineage Resolution Tests
// =============================================================================

func TestResolveLineage_Direct(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Ref: "#/components/schemas/UserList"},
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
				"UserList": {Type: "array"},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)
	lineage := graph.ResolveLineage("UserList")

	require.Len(t, lineage, 1)
	assert.Equal(t, "listUsers", lineage[0].OperationID)
}

func TestResolveLineage_Indirect(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Ref: "#/components/schemas/SchemaA"},
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
				"SchemaA": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"b": {Ref: "#/components/schemas/SchemaB"},
					},
				},
				"SchemaB": {
					Type: "string",
				},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	// SchemaB is indirectly referenced through SchemaA
	lineage := graph.ResolveLineage("SchemaB")

	require.Len(t, lineage, 1)
	assert.Equal(t, "listUsers", lineage[0].OperationID)
}

func TestResolveLineage_MultiHop(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Ref: "#/components/schemas/SchemaA"},
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
				"SchemaA": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"b": {Ref: "#/components/schemas/SchemaB"},
					},
				},
				"SchemaB": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"c": {Ref: "#/components/schemas/SchemaC"},
					},
				},
				"SchemaC": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"d": {Ref: "#/components/schemas/SchemaD"},
					},
				},
				"SchemaD": {
					Type: "string",
				},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	// SchemaD is 3 hops away from the operation
	lineage := graph.ResolveLineage("SchemaD")

	require.Len(t, lineage, 1)
	assert.Equal(t, "listUsers", lineage[0].OperationID)
}

func TestResolveLineage_Cycles(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Ref: "#/components/schemas/SchemaA"},
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
				"SchemaA": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"b": {Ref: "#/components/schemas/SchemaB"},
					},
				},
				"SchemaB": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"a": {Ref: "#/components/schemas/SchemaA"}, // Circular back to A
					},
				},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	// This should not hang - cycle detection should kick in
	lineageA := graph.ResolveLineage("SchemaA")
	lineageB := graph.ResolveLineage("SchemaB")

	// Both should resolve to the same operation
	require.Len(t, lineageA, 1)
	require.Len(t, lineageB, 1)
	assert.Equal(t, "listUsers", lineageA[0].OperationID)
	assert.Equal(t, "listUsers", lineageB[0].OperationID)
}

func TestResolveLineage_MultipleOperations(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Ref: "#/components/schemas/User"},
									},
								},
							},
						},
					},
				},
			},
			"/orders": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listOrders",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Ref: "#/components/schemas/OrderList"},
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
				"User": {Type: "object"},
				"OrderList": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"user": {Ref: "#/components/schemas/User"},
					},
				},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	// User is referenced by both operations (directly and indirectly)
	lineage := graph.ResolveLineage("User")

	require.Len(t, lineage, 2)

	opIDs := make(map[string]bool)
	for _, ref := range lineage {
		opIDs[ref.OperationID] = true
	}
	assert.True(t, opIDs["listUsers"], "Expected listUsers in lineage")
	assert.True(t, opIDs["listOrders"], "Expected listOrders in lineage")
}

func TestResolveLineage_Caching(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Ref: "#/components/schemas/User"},
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
				"User": {Type: "object"},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	// First call
	lineage1 := graph.ResolveLineage("User")

	// Verify it's cached
	assert.Contains(t, graph.resolved, "User")

	// Second call should return cached result
	lineage2 := graph.ResolveLineage("User")

	// Should be the same slice (cached)
	assert.Equal(t, lineage1, lineage2)
}

func TestResolveLineage_NilGraph(t *testing.T) {
	var graph *RefGraph

	// Should not panic
	lineage := graph.ResolveLineage("AnySchema")

	assert.Nil(t, lineage)
}

func TestResolveLineage_UnknownSchema(t *testing.T) {
	graph := newRefGraph()

	lineage := graph.ResolveLineage("NonExistentSchema")

	assert.Empty(t, lineage)
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestExtractSchemaNameFromRef(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "OAS3 schema ref",
			ref:      "#/components/schemas/Pet",
			expected: "Pet",
		},
		{
			name:     "OAS2 definition ref",
			ref:      "#/definitions/Pet",
			expected: "Pet",
		},
		{
			name:     "response ref (not a schema)",
			ref:      "#/components/responses/Error",
			expected: "",
		},
		{
			name:     "parameter ref (not a schema)",
			ref:      "#/components/parameters/PageSize",
			expected: "",
		},
		{
			name:     "empty string",
			ref:      "",
			expected: "",
		},
		{
			name:     "external ref",
			ref:      "./common.yaml#/components/schemas/Error",
			expected: "",
		},
		{
			name:     "schema name with path separators",
			ref:      "#/components/schemas/User.Profile",
			expected: "User.Profile",
		},
		{
			name:     "OAS3 nested schema name",
			ref:      "#/components/schemas/Order_Item",
			expected: "Order_Item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSchemaNameFromRef(tt.ref)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDeduplicateOperationRefs(t *testing.T) {
	tests := []struct {
		name        string
		refs        []OperationRef
		expectedLen int
	}{
		{
			name:        "empty slice",
			refs:        []OperationRef{},
			expectedLen: 0,
		},
		{
			name:        "nil slice",
			refs:        nil,
			expectedLen: 0,
		},
		{
			name: "no duplicates",
			refs: []OperationRef{
				{Path: "/users", Method: "get", UsageType: UsageTypeResponse, StatusCode: "200"},
				{Path: "/users", Method: "post", UsageType: UsageTypeRequest, StatusCode: ""},
				{Path: "/orders", Method: "get", UsageType: UsageTypeResponse, StatusCode: "200"},
			},
			expectedLen: 3,
		},
		{
			name: "with duplicates",
			refs: []OperationRef{
				{Path: "/users", Method: "get", UsageType: UsageTypeResponse, StatusCode: "200"},
				{Path: "/users", Method: "get", UsageType: UsageTypeResponse, StatusCode: "200"}, // duplicate
				{Path: "/users", Method: "post", UsageType: UsageTypeRequest, StatusCode: ""},
			},
			expectedLen: 2,
		},
		{
			name: "same path and method but different usage type",
			refs: []OperationRef{
				{Path: "/users", Method: "post", UsageType: UsageTypeRequest, StatusCode: ""},
				{Path: "/users", Method: "post", UsageType: UsageTypeResponse, StatusCode: "201"},
			},
			expectedLen: 2,
		},
		{
			name: "same path and method but different status code",
			refs: []OperationRef{
				{Path: "/users", Method: "get", UsageType: UsageTypeResponse, StatusCode: "200"},
				{Path: "/users", Method: "get", UsageType: UsageTypeResponse, StatusCode: "404"},
			},
			expectedLen: 2,
		},
		{
			name: "all duplicates",
			refs: []OperationRef{
				{Path: "/users", Method: "get", UsageType: UsageTypeResponse, StatusCode: "200"},
				{Path: "/users", Method: "get", UsageType: UsageTypeResponse, StatusCode: "200"},
				{Path: "/users", Method: "get", UsageType: UsageTypeResponse, StatusCode: "200"},
			},
			expectedLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateOperationRefs(tt.refs)
			assert.Len(t, result, tt.expectedLen)
		})
	}
}

func TestDeduplicateOperationRefs_PreservesOrder(t *testing.T) {
	refs := []OperationRef{
		{Path: "/a", Method: "get", UsageType: UsageTypeResponse, StatusCode: "200"},
		{Path: "/b", Method: "post", UsageType: UsageTypeRequest, StatusCode: ""},
		{Path: "/a", Method: "get", UsageType: UsageTypeResponse, StatusCode: "200"}, // duplicate of first
		{Path: "/c", Method: "delete", UsageType: UsageTypeResponse, StatusCode: "204"},
	}

	result := deduplicateOperationRefs(refs)

	require.Len(t, result, 3)

	// Verify order is preserved
	expectedPaths := []string{"/a", "/b", "/c"}
	for i, expected := range expectedPaths {
		assert.Equal(t, expected, result[i].Path, "path at index %d", i)
	}
}

func TestDeduplicateOperationRefs_PreservesAllFields(t *testing.T) {
	refs := []OperationRef{
		{
			Path:        "/users",
			Method:      "get",
			OperationID: "listUsers",
			Tags:        []string{"Users"},
			UsageType:   UsageTypeResponse,
			StatusCode:  "200",
			ParamName:   "",
			MediaType:   "application/json",
		},
	}

	result := deduplicateOperationRefs(refs)

	require.Len(t, result, 1)

	// Verify all fields are preserved
	assert.Equal(t, "listUsers", result[0].OperationID)
	assert.Equal(t, []string{"Users"}, result[0].Tags)
	assert.Equal(t, "application/json", result[0].MediaType)
}

// =============================================================================
// Schema Reference Location Tests
// =============================================================================

func TestBuildRefGraphOAS3_SchemaRefLocations(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Container": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"item": {Ref: "#/components/schemas/Item"},
					},
					AllOf: []*parser.Schema{
						{Ref: "#/components/schemas/Base"},
					},
				},
				"List": {
					Type: "array",
					Items: &parser.Schema{
						Ref: "#/components/schemas/ListItem",
					},
				},
				"Item":     {Type: "object"},
				"Base":     {Type: "object"},
				"ListItem": {Type: "object"},
			},
		},
	}

	graph := buildRefGraphOAS3(doc, parser.OASVersion300)

	// Check properties ref location
	itemRefs := graph.schemaRefs["Item"]
	require.Len(t, itemRefs, 1)
	assert.Equal(t, "properties.item", itemRefs[0].RefLocation)

	// Check allOf ref location
	baseRefs := graph.schemaRefs["Base"]
	require.Len(t, baseRefs, 1)
	assert.Equal(t, "allOf[0]", baseRefs[0].RefLocation)

	// Check items ref location
	listItemRefs := graph.schemaRefs["ListItem"]
	require.Len(t, listItemRefs, 1)
	assert.Equal(t, "items", listItemRefs[0].RefLocation)
}

func TestJoinLocation(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		segment  string
		expected string
	}{
		{
			name:     "empty base",
			base:     "",
			segment:  "properties.name",
			expected: "properties.name",
		},
		{
			name:     "non-empty base",
			base:     "properties.address",
			segment:  "properties.street",
			expected: "properties.address.properties.street",
		},
		{
			name:     "both non-empty",
			base:     "allOf[0]",
			segment:  "properties.id",
			expected: "allOf[0].properties.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinLocation(tt.base, tt.segment)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// OAS 2.0 Specific Tests
// =============================================================================

func TestBuildRefGraphOAS2_BodyParameter(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Post: &parser.Operation{
					OperationID: "createUser",
					Parameters: []*parser.Parameter{
						{
							Name:   "body",
							In:     "body",
							Schema: &parser.Schema{Ref: "#/definitions/UserCreate"},
						},
					},
				},
			},
		},
		Definitions: map[string]*parser.Schema{
			"UserCreate": {Type: "object"},
		},
	}

	graph := buildRefGraphOAS2(doc)

	refs, ok := graph.operationRefs["UserCreate"]
	require.True(t, ok, "Expected operationRefs to contain UserCreate")
	require.Len(t, refs, 1)
	// OAS 2.0 body parameters should be marked as request usage
	assert.Equal(t, UsageTypeRequest, refs[0].UsageType)
}

func TestBuildRefGraphOAS2_PathLevelParameter(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users/{id}": &parser.PathItem{
				Parameters: []*parser.Parameter{
					{
						Name:   "body",
						In:     "body",
						Schema: &parser.Schema{Ref: "#/definitions/SharedBody"},
					},
				},
				Get: &parser.Operation{
					OperationID: "getUser",
				},
			},
		},
		Definitions: map[string]*parser.Schema{
			"SharedBody": {Type: "object"},
		},
	}

	graph := buildRefGraphOAS2(doc)

	refs, ok := graph.operationRefs["SharedBody"]
	require.True(t, ok, "Expected operationRefs to contain SharedBody from path-level parameter")
	require.Len(t, refs, 1)
}

func TestBuildRefGraphOAS2_DefaultResponse(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
					Responses: &parser.Responses{
						Default: &parser.Response{
							Schema: &parser.Schema{Ref: "#/definitions/Error"},
						},
					},
				},
			},
		},
		Definitions: map[string]*parser.Schema{
			"Error": {Type: "object"},
		},
	}

	graph := buildRefGraphOAS2(doc)

	refs, ok := graph.operationRefs["Error"]
	require.True(t, ok, "Expected operationRefs to contain Error from default response")
	require.Len(t, refs, 1)
	assert.Equal(t, "default", refs[0].StatusCode)
}

// =============================================================================
// New RefGraph Helper Tests
// =============================================================================

func TestNewRefGraph(t *testing.T) {
	graph := newRefGraph()

	require.NotNil(t, graph)
	assert.NotNil(t, graph.schemaRefs)
	assert.NotNil(t, graph.operationRefs)
	assert.NotNil(t, graph.resolved)
}
