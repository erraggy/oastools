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
