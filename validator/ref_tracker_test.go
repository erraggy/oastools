package validator

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefTrackerOAS3(t *testing.T) {
	// Create a simple OAS3 document with refs
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
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
				Post: &parser.Operation{
					OperationID: "createUser",
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{
									Ref: "#/components/schemas/User",
								},
							},
						},
					},
				},
			},
			"/users/{id}": &parser.PathItem{
				Parameters: []*parser.Parameter{
					{Name: "id", In: "path", Required: true},
				},
				Get: &parser.Operation{
					OperationID: "getUser",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{
											Ref: "#/components/schemas/User",
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
				"User": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"id":   {Type: "integer"},
						"name": {Type: "string"},
					},
				},
				"UserList": {
					Type: "array",
					Items: &parser.Schema{
						Ref: "#/components/schemas/User",
					},
				},
				"OrphanedSchema": {
					Type: "object",
				},
			},
		},
	}

	tracker := buildRefTrackerOAS3(doc)

	t.Run("User schema referenced by multiple operations", func(t *testing.T) {
		ops := tracker.getOperationsForComponent("components.schemas.User")
		require.NotEmpty(t, ops)
		// Should include createUser, getUser, and listUsers (via UserList)
		assert.GreaterOrEqual(t, len(ops), 2)

		// Check one of the operations has correct data
		var foundGetUser bool
		for _, op := range ops {
			if op.OperationID == "getUser" {
				foundGetUser = true
				assert.Equal(t, "GET", op.Method)
				assert.Equal(t, "/users/{id}", op.Path)
			}
		}
		assert.True(t, foundGetUser, "should find getUser operation")
	})

	t.Run("UserList schema referenced by listUsers", func(t *testing.T) {
		ops := tracker.getOperationsForComponent("components.schemas.UserList")
		require.Len(t, ops, 1)
		assert.Equal(t, "listUsers", ops[0].OperationID)
		assert.Equal(t, "GET", ops[0].Method)
		assert.Equal(t, "/users", ops[0].Path)
	})

	t.Run("OrphanedSchema has no references", func(t *testing.T) {
		ops := tracker.getOperationsForComponent("components.schemas.OrphanedSchema")
		assert.Empty(t, ops)
	})

	t.Run("non-existent component returns empty", func(t *testing.T) {
		ops := tracker.getOperationsForComponent("components.schemas.DoesNotExist")
		assert.Empty(t, ops)
	})
}

func TestRefTrackerOAS2(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Schema: &parser.Schema{
									Type: "array",
									Items: &parser.Schema{
										Ref: "#/definitions/Pet",
									},
								},
							},
						},
					},
				},
			},
		},
		Definitions: map[string]*parser.Schema{
			"Pet": {
				Type: "object",
			},
		},
	}

	tracker := buildRefTrackerOAS2(doc)

	t.Run("Pet definition referenced by listPets", func(t *testing.T) {
		ops := tracker.getOperationsForComponent("definitions.Pet")
		require.Len(t, ops, 1)
		assert.Equal(t, "listPets", ops[0].OperationID)
	})
}

func TestRefTrackerTransitiveRefs(t *testing.T) {
	// A -> B -> C: operation references A, which refs B, which refs C
	// All three should be tracked as used by the operation
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Paths: parser.Paths{
			"/test": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "testOp",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{
											Ref: "#/components/schemas/A",
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
				"A": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"b": {Ref: "#/components/schemas/B"},
					},
				},
				"B": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"c": {Ref: "#/components/schemas/C"},
					},
				},
				"C": {
					Type: "object",
				},
			},
		},
	}

	tracker := buildRefTrackerOAS3(doc)

	// All three schemas should be tracked
	for _, schema := range []string{"A", "B", "C"} {
		t.Run("schema "+schema+" is tracked", func(t *testing.T) {
			ops := tracker.getOperationsForComponent("components.schemas." + schema)
			require.Len(t, ops, 1, "schema %s should have 1 operation", schema)
			assert.Equal(t, "testOp", ops[0].OperationID)
		})
	}
}

func TestRefTrackerCircularRefs(t *testing.T) {
	// A -> B -> A (circular)
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Paths: parser.Paths{
			"/test": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "testOp",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{
											Ref: "#/components/schemas/A",
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
				"A": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"b": {Ref: "#/components/schemas/B"},
					},
				},
				"B": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"a": {Ref: "#/components/schemas/A"}, // circular
					},
				},
			},
		},
	}

	// Should not hang or panic
	tracker := buildRefTrackerOAS3(doc)

	// Both schemas should be tracked
	opsA := tracker.getOperationsForComponent("components.schemas.A")
	opsB := tracker.getOperationsForComponent("components.schemas.B")
	assert.Len(t, opsA, 1)
	assert.Len(t, opsB, 1)
}

func TestRefTrackerWebhooks(t *testing.T) {
	// Create an OAS 3.1 document with webhooks
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Webhooks: parser.Paths{
			"orderCreated": &parser.PathItem{
				Post: &parser.Operation{
					OperationID: "handleOrderCreated",
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{
									Ref: "#/components/schemas/Order",
								},
							},
						},
					},
				},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Order": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"id": {Type: "string"},
					},
				},
			},
		},
	}

	tracker := buildRefTrackerOAS3(doc)

	t.Run("webhook schema is tracked", func(t *testing.T) {
		ops := tracker.getOperationsForComponent("components.schemas.Order")
		require.Len(t, ops, 1)
		assert.Equal(t, "POST", ops[0].Method)
		assert.Equal(t, "orderCreated", ops[0].Path)
		assert.Equal(t, "handleOrderCreated", ops[0].OperationID)
		assert.True(t, ops[0].IsWebhook)
	})

	t.Run("webhook operation context from issue path", func(t *testing.T) {
		ctx := tracker.getOperationContext("webhooks.orderCreated.post.requestBody", doc)
		require.NotNil(t, ctx)
		assert.Equal(t, "POST", ctx.Method)
		assert.Equal(t, "orderCreated", ctx.Path)
		assert.Equal(t, "handleOrderCreated", ctx.OperationID)
		assert.True(t, ctx.IsWebhook)
	})

	t.Run("webhook path-level context", func(t *testing.T) {
		ctx := tracker.getOperationContext("webhooks.orderCreated.parameters", doc)
		require.NotNil(t, ctx)
		assert.Equal(t, "orderCreated", ctx.Path)
		assert.Empty(t, ctx.Method)
		assert.True(t, ctx.IsWebhook)
	})

	t.Run("component context shows webhook reference", func(t *testing.T) {
		ctx := tracker.getOperationContext("components.schemas.Order.properties.id", doc)
		require.NotNil(t, ctx)
		assert.True(t, ctx.IsReusableComponent)
		assert.True(t, ctx.IsWebhook)
		assert.Equal(t, "handleOrderCreated", ctx.OperationID)
	})
}

func TestNormalizeRef(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"local schema ref", "#/components/schemas/User", "components.schemas.User"},
		{"local definition ref", "#/definitions/Pet", "definitions.Pet"},
		{"local parameter ref", "#/components/parameters/userId", "components.parameters.userId"},
		{"external file ref", "./external.yaml#/components/schemas/X", ""},
		{"external URL ref", "https://example.com/api.yaml#/components/schemas/Y", ""},
		{"empty string", "", ""},
		{"just hash", "#", ""},
		{"hash slash only", "#/", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeRef(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetComponentRoot(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// OAS3 components (3 parts before properties)
		{"schema with properties", "components.schemas.User.properties.id", "components.schemas.User"},
		{"schema direct", "components.schemas.User", "components.schemas.User"},
		{"parameter", "components.parameters.userId", "components.parameters.userId"},
		{"response with content", "components.responses.NotFound.content", "components.responses.NotFound"},

		// OAS2 definitions (2 parts before properties)
		{"definition with properties", "definitions.Pet.properties.name", "definitions.Pet"},
		{"definition direct", "definitions.Pet", "definitions.Pet"},
		{"parameter OAS2", "parameters.userId", "parameters.userId"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getComponentRoot(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsReusableComponentPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// OAS3 component paths
		{"schemas", "components.schemas.User", true},
		{"schemas nested", "components.schemas.User.properties.id", true},
		{"parameters", "components.parameters.userId", true},
		{"responses", "components.responses.NotFound", true},
		{"requestBodies", "components.requestBodies.UserInput", true},
		{"headers", "components.headers.X-Rate-Limit", true},
		{"securitySchemes", "components.securitySchemes.bearerAuth", true},
		{"links", "components.links.GetUserById", true},
		{"callbacks", "components.callbacks.onEvent", true},
		{"pathItems", "components.pathItems.UserPath", true},

		// OAS2 component paths
		{"definitions", "definitions.Pet", true},
		{"parameters OAS2", "parameters.userId", true},

		// Non-component paths
		{"paths", "paths./users.get", false},
		{"webhooks", "webhooks.orderCreated.post", false},
		{"info", "info.title", false},
		{"empty", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isReusableComponentPath(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseMethod(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"get lowercase", "get", "GET"},
		{"post lowercase", "post", "POST"},
		{"put lowercase", "put", "PUT"},
		{"delete lowercase", "delete", "DELETE"},
		{"patch lowercase", "patch", "PATCH"},
		{"options lowercase", "options", "OPTIONS"},
		{"head lowercase", "head", "HEAD"},
		{"trace lowercase", "trace", "TRACE"},
		{"query lowercase", "query", "QUERY"},
		{"unknown method", "connect", ""},
		{"uppercase (not matched)", "GET", ""},
		{"empty string", "", ""},
		{"parameters (not a method)", "parameters", ""},
		{"requestBody (not a method)", "requestBody", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseMethod(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
