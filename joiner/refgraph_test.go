package joiner

import (
	"reflect"
	"testing"

	"github.com/erraggy/oastools/parser"
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

	if graph == nil {
		t.Fatal("Expected non-nil graph")
	}
	if len(graph.schemaRefs) != 0 {
		t.Errorf("Expected empty schemaRefs, got %d entries", len(graph.schemaRefs))
	}
	if len(graph.operationRefs) != 0 {
		t.Errorf("Expected empty operationRefs, got %d entries", len(graph.operationRefs))
	}
	if len(graph.resolved) != 0 {
		t.Errorf("Expected empty resolved, got %d entries", len(graph.resolved))
	}
}

func TestBuildRefGraphOAS3_NilDocument(t *testing.T) {
	graph := buildRefGraphOAS3(nil, parser.OASVersion300)

	if graph == nil {
		t.Fatal("Expected non-nil graph even for nil document")
	}
	if len(graph.schemaRefs) != 0 {
		t.Errorf("Expected empty schemaRefs, got %d entries", len(graph.schemaRefs))
	}
	if len(graph.operationRefs) != 0 {
		t.Errorf("Expected empty operationRefs, got %d entries", len(graph.operationRefs))
	}
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

	if graph == nil {
		t.Fatal("Expected non-nil graph")
	}

	// Verify operationRefs contains UserList schema
	refs, ok := graph.operationRefs["UserList"]
	if !ok {
		t.Fatal("Expected operationRefs to contain UserList")
	}
	if len(refs) != 1 {
		t.Fatalf("Expected 1 operation ref, got %d", len(refs))
	}

	// Verify the OperationRef details
	opRef := refs[0]
	if opRef.Path != "/users" {
		t.Errorf("Expected path /users, got %s", opRef.Path)
	}
	if opRef.Method != "get" {
		t.Errorf("Expected method get, got %s", opRef.Method)
	}
	if opRef.OperationID != "getUsers" {
		t.Errorf("Expected operationID getUsers, got %s", opRef.OperationID)
	}
	if opRef.UsageType != UsageTypeResponse {
		t.Errorf("Expected usageType response, got %s", opRef.UsageType)
	}
	if opRef.StatusCode != "200" {
		t.Errorf("Expected statusCode 200, got %s", opRef.StatusCode)
	}
	if opRef.MediaType != "application/json" {
		t.Errorf("Expected mediaType application/json, got %s", opRef.MediaType)
	}
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
	if !ok {
		t.Error("Expected operationRefs to contain UserUpdate")
	} else if len(requestRefs) != 1 || requestRefs[0].UsageType != UsageTypeRequest {
		t.Errorf("Expected UserUpdate to have UsageTypeRequest, got %v", requestRefs)
	}

	// Verify response schema (UsageResponse)
	responseRefs, ok := graph.operationRefs["User"]
	if !ok {
		t.Error("Expected operationRefs to contain User")
	} else if len(responseRefs) != 1 || responseRefs[0].UsageType != UsageTypeResponse {
		t.Errorf("Expected User to have UsageTypeResponse, got %v", responseRefs)
	}

	// Verify parameter schema (UsageParameter)
	paramRefs, ok := graph.operationRefs["FilterParams"]
	if !ok {
		t.Error("Expected operationRefs to contain FilterParams")
	} else if len(paramRefs) != 1 || paramRefs[0].UsageType != UsageTypeParameter {
		t.Errorf("Expected FilterParams to have UsageTypeParameter, got %v", paramRefs)
	}
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
	if !ok {
		t.Fatal("Expected schemaRefs to contain SchemaB")
	}
	if len(schemaBRefs) != 1 {
		t.Fatalf("Expected 1 reference to SchemaB, got %d", len(schemaBRefs))
	}
	if schemaBRefs[0].FromSchema != "SchemaA" {
		t.Errorf("Expected SchemaB to be referenced from SchemaA, got %s", schemaBRefs[0].FromSchema)
	}

	// Verify SchemaC is referenced by SchemaB
	schemaCRefs, ok := graph.schemaRefs["SchemaC"]
	if !ok {
		t.Fatal("Expected schemaRefs to contain SchemaC")
	}
	if len(schemaCRefs) != 1 {
		t.Fatalf("Expected 1 reference to SchemaC, got %d", len(schemaCRefs))
	}
	if schemaCRefs[0].FromSchema != "SchemaB" {
		t.Errorf("Expected SchemaC to be referenced from SchemaB, got %s", schemaCRefs[0].FromSchema)
	}
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

	if graph == nil {
		t.Fatal("Expected non-nil graph")
	}

	// Verify both refs are recorded
	if _, ok := graph.schemaRefs["SchemaA"]; !ok {
		t.Error("Expected schemaRefs to contain SchemaA")
	}
	if _, ok := graph.schemaRefs["SchemaB"]; !ok {
		t.Error("Expected schemaRefs to contain SchemaB")
	}
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
	if !ok {
		t.Fatal("Expected operationRefs to contain User")
	}

	// Should have 3 refs: GET 200 response, POST request, POST 201 response
	if len(refs) != 3 {
		t.Errorf("Expected 3 operation refs for User, got %d", len(refs))
	}

	// Verify we have different operation IDs
	opIDs := make(map[string]bool)
	for _, ref := range refs {
		opIDs[ref.OperationID] = true
	}
	if !opIDs["listUsers"] || !opIDs["createUser"] {
		t.Errorf("Expected both listUsers and createUser operations, got %v", opIDs)
	}
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

	if graph == nil {
		t.Fatal("Expected non-nil graph")
	}

	// Verify operationRefs contains PetList
	refs, ok := graph.operationRefs["PetList"]
	if !ok {
		t.Fatal("Expected operationRefs to contain PetList")
	}
	if len(refs) != 1 {
		t.Fatalf("Expected 1 operation ref, got %d", len(refs))
	}
	if refs[0].UsageType != UsageTypeResponse {
		t.Errorf("Expected UsageTypeResponse, got %s", refs[0].UsageType)
	}

	// Verify schemaRefs contains Pet (referenced by PetList.items)
	schemaRefs, ok := graph.schemaRefs["Pet"]
	if !ok {
		t.Fatal("Expected schemaRefs to contain Pet")
	}
	if len(schemaRefs) != 1 {
		t.Fatalf("Expected 1 schema ref, got %d", len(schemaRefs))
	}
	if schemaRefs[0].FromSchema != "PetList" {
		t.Errorf("Expected Pet to be referenced from PetList, got %s", schemaRefs[0].FromSchema)
	}
}

func TestBuildRefGraphOAS2_NilDocument(t *testing.T) {
	graph := buildRefGraphOAS2(nil)

	if graph == nil {
		t.Fatal("Expected non-nil graph even for nil document")
	}
	if len(graph.schemaRefs) != 0 {
		t.Errorf("Expected empty schemaRefs, got %d entries", len(graph.schemaRefs))
	}
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
	if !ok {
		t.Fatal("Expected operationRefs to contain Pet from webhook")
	}
	if len(refs) != 1 {
		t.Fatalf("Expected 1 operation ref, got %d", len(refs))
	}
	if refs[0].Path != "webhook:newPet" {
		t.Errorf("Expected path webhook:newPet, got %s", refs[0].Path)
	}
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
	if !ok {
		t.Fatal("Expected operationRefs to contain OrderRequest")
	}
	if len(orderReqRefs) != 1 {
		t.Fatalf("Expected 1 operation ref for OrderRequest, got %d", len(orderReqRefs))
	}
	if orderReqRefs[0].UsageType != UsageTypeRequest {
		t.Errorf("Expected UsageTypeRequest for OrderRequest, got %s", orderReqRefs[0].UsageType)
	}

	// Verify OrderCallback schema is recorded with callback usage type
	callbackRefs, ok := graph.operationRefs["OrderCallback"]
	if !ok {
		t.Fatal("Expected operationRefs to contain OrderCallback from callback request body")
	}
	if len(callbackRefs) != 1 {
		t.Fatalf("Expected 1 operation ref for OrderCallback, got %d", len(callbackRefs))
	}
	if callbackRefs[0].UsageType != UsageTypeCallback {
		t.Errorf("Expected UsageTypeCallback for OrderCallback, got %s", callbackRefs[0].UsageType)
	}
	// Verify callback path format: path->callbackName:callbackPath
	expectedCallbackPath := "/orders->onOrderComplete:{$request.body#/callbackUrl}"
	if callbackRefs[0].Path != expectedCallbackPath {
		t.Errorf("Expected callback path %q, got %q", expectedCallbackPath, callbackRefs[0].Path)
	}
	if callbackRefs[0].OperationID != "orderCallback" {
		t.Errorf("Expected operationID 'orderCallback', got %s", callbackRefs[0].OperationID)
	}
	if callbackRefs[0].MediaType != "application/json" {
		t.Errorf("Expected mediaType 'application/json', got %s", callbackRefs[0].MediaType)
	}

	// Verify CallbackResponse schema is recorded from callback response
	respRefs, ok := graph.operationRefs["CallbackResponse"]
	if !ok {
		t.Fatal("Expected operationRefs to contain CallbackResponse from callback response")
	}
	if len(respRefs) != 1 {
		t.Fatalf("Expected 1 operation ref for CallbackResponse, got %d", len(respRefs))
	}
	// Note: callback responses use UsageTypeResponse, not UsageTypeCallback
	if respRefs[0].UsageType != UsageTypeResponse {
		t.Errorf("Expected UsageTypeResponse for CallbackResponse, got %s", respRefs[0].UsageType)
	}
	if respRefs[0].StatusCode != "200" {
		t.Errorf("Expected statusCode '200', got %s", respRefs[0].StatusCode)
	}
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
	if !ok {
		t.Fatal("Expected operationRefs to contain UserId from path-level parameter")
	}
	if len(refs) != 1 {
		t.Fatalf("Expected 1 operation ref, got %d", len(refs))
	}
	if refs[0].UsageType != UsageTypeParameter {
		t.Errorf("Expected UsageTypeParameter, got %s", refs[0].UsageType)
	}
	if refs[0].ParamName != "id" {
		t.Errorf("Expected param name 'id', got %s", refs[0].ParamName)
	}
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
	if !ok {
		t.Fatal("Expected operationRefs to contain Count from response header")
	}
	if len(refs) != 1 {
		t.Fatalf("Expected 1 operation ref, got %d", len(refs))
	}
	if refs[0].UsageType != UsageTypeHeader {
		t.Errorf("Expected UsageTypeHeader, got %s", refs[0].UsageType)
	}
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
	if !ok {
		t.Fatal("Expected operationRefs to contain Error from default response")
	}
	if len(refs) != 1 {
		t.Fatalf("Expected 1 operation ref, got %d", len(refs))
	}
	if refs[0].StatusCode != "default" {
		t.Errorf("Expected statusCode 'default', got %s", refs[0].StatusCode)
	}
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

	if len(lineage) != 1 {
		t.Fatalf("Expected 1 operation in lineage, got %d", len(lineage))
	}
	if lineage[0].OperationID != "listUsers" {
		t.Errorf("Expected operationID listUsers, got %s", lineage[0].OperationID)
	}
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

	if len(lineage) != 1 {
		t.Fatalf("Expected 1 operation in lineage for SchemaB, got %d", len(lineage))
	}
	if lineage[0].OperationID != "listUsers" {
		t.Errorf("Expected operationID listUsers, got %s", lineage[0].OperationID)
	}
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

	if len(lineage) != 1 {
		t.Fatalf("Expected 1 operation in lineage for SchemaD, got %d", len(lineage))
	}
	if lineage[0].OperationID != "listUsers" {
		t.Errorf("Expected operationID listUsers, got %s", lineage[0].OperationID)
	}
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
	if len(lineageA) != 1 {
		t.Fatalf("Expected 1 operation in lineage for SchemaA, got %d", len(lineageA))
	}
	if len(lineageB) != 1 {
		t.Fatalf("Expected 1 operation in lineage for SchemaB, got %d", len(lineageB))
	}
	if lineageA[0].OperationID != "listUsers" {
		t.Errorf("Expected operationID listUsers for SchemaA, got %s", lineageA[0].OperationID)
	}
	if lineageB[0].OperationID != "listUsers" {
		t.Errorf("Expected operationID listUsers for SchemaB, got %s", lineageB[0].OperationID)
	}
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

	if len(lineage) != 2 {
		t.Fatalf("Expected 2 operations in lineage for User, got %d", len(lineage))
	}

	opIDs := make(map[string]bool)
	for _, ref := range lineage {
		opIDs[ref.OperationID] = true
	}
	if !opIDs["listUsers"] {
		t.Error("Expected listUsers in lineage")
	}
	if !opIDs["listOrders"] {
		t.Error("Expected listOrders in lineage")
	}
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
	if _, ok := graph.resolved["User"]; !ok {
		t.Error("Expected User to be cached in resolved map")
	}

	// Second call should return cached result
	lineage2 := graph.ResolveLineage("User")

	// Should be the same slice (cached)
	if !reflect.DeepEqual(lineage1, lineage2) {
		t.Error("Expected cached result to match first result")
	}
}

func TestResolveLineage_NilGraph(t *testing.T) {
	var graph *RefGraph

	// Should not panic
	lineage := graph.ResolveLineage("AnySchema")

	if lineage != nil {
		t.Errorf("Expected nil lineage for nil graph, got %v", lineage)
	}
}

func TestResolveLineage_UnknownSchema(t *testing.T) {
	graph := newRefGraph()

	lineage := graph.ResolveLineage("NonExistentSchema")

	if len(lineage) != 0 {
		t.Errorf("Expected empty lineage for unknown schema, got %d entries", len(lineage))
	}
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
			if got != tt.expected {
				t.Errorf("extractSchemaNameFromRef(%q) = %q, want %q", tt.ref, got, tt.expected)
			}
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
			if len(result) != tt.expectedLen {
				t.Errorf("deduplicateOperationRefs() returned %d refs, want %d", len(result), tt.expectedLen)
			}
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

	if len(result) != 3 {
		t.Fatalf("Expected 3 refs, got %d", len(result))
	}

	// Verify order is preserved
	expectedPaths := []string{"/a", "/b", "/c"}
	for i, expected := range expectedPaths {
		if result[i].Path != expected {
			t.Errorf("Expected path %s at index %d, got %s", expected, i, result[i].Path)
		}
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

	if len(result) != 1 {
		t.Fatalf("Expected 1 ref, got %d", len(result))
	}

	// Verify all fields are preserved
	if result[0].OperationID != "listUsers" {
		t.Errorf("OperationID not preserved: got %s", result[0].OperationID)
	}
	if len(result[0].Tags) != 1 || result[0].Tags[0] != "Users" {
		t.Errorf("Tags not preserved: got %v", result[0].Tags)
	}
	if result[0].MediaType != "application/json" {
		t.Errorf("MediaType not preserved: got %s", result[0].MediaType)
	}
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
	if len(itemRefs) != 1 {
		t.Fatalf("Expected 1 ref to Item, got %d", len(itemRefs))
	}
	if itemRefs[0].RefLocation != "properties.item" {
		t.Errorf("Expected RefLocation 'properties.item', got '%s'", itemRefs[0].RefLocation)
	}

	// Check allOf ref location
	baseRefs := graph.schemaRefs["Base"]
	if len(baseRefs) != 1 {
		t.Fatalf("Expected 1 ref to Base, got %d", len(baseRefs))
	}
	if baseRefs[0].RefLocation != "allOf[0]" {
		t.Errorf("Expected RefLocation 'allOf[0]', got '%s'", baseRefs[0].RefLocation)
	}

	// Check items ref location
	listItemRefs := graph.schemaRefs["ListItem"]
	if len(listItemRefs) != 1 {
		t.Fatalf("Expected 1 ref to ListItem, got %d", len(listItemRefs))
	}
	if listItemRefs[0].RefLocation != "items" {
		t.Errorf("Expected RefLocation 'items', got '%s'", listItemRefs[0].RefLocation)
	}
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
			if result != tt.expected {
				t.Errorf("joinLocation(%q, %q) = %q, want %q", tt.base, tt.segment, result, tt.expected)
			}
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
	if !ok {
		t.Fatal("Expected operationRefs to contain UserCreate")
	}
	if len(refs) != 1 {
		t.Fatalf("Expected 1 operation ref, got %d", len(refs))
	}
	// OAS 2.0 body parameters should be marked as request usage
	if refs[0].UsageType != UsageTypeRequest {
		t.Errorf("Expected UsageTypeRequest for body parameter, got %s", refs[0].UsageType)
	}
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
	if !ok {
		t.Fatal("Expected operationRefs to contain SharedBody from path-level parameter")
	}
	if len(refs) != 1 {
		t.Fatalf("Expected 1 operation ref, got %d", len(refs))
	}
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
	if !ok {
		t.Fatal("Expected operationRefs to contain Error from default response")
	}
	if len(refs) != 1 {
		t.Fatalf("Expected 1 operation ref, got %d", len(refs))
	}
	if refs[0].StatusCode != "default" {
		t.Errorf("Expected statusCode 'default', got %s", refs[0].StatusCode)
	}
}

// =============================================================================
// New RefGraph Helper Tests
// =============================================================================

func TestNewRefGraph(t *testing.T) {
	graph := newRefGraph()

	if graph == nil {
		t.Fatal("Expected non-nil graph")
	}
	if graph.schemaRefs == nil {
		t.Error("Expected non-nil schemaRefs map")
	}
	if graph.operationRefs == nil {
		t.Error("Expected non-nil operationRefs map")
	}
	if graph.resolved == nil {
		t.Error("Expected non-nil resolved map")
	}
}
