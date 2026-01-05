package walker

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectSchemas_Components(t *testing.T) {
	t.Run("OAS3 component schemas", func(t *testing.T) {
		doc := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
			Components: &parser.Components{
				Schemas: map[string]*parser.Schema{
					"Pet": {
						Type:        "object",
						Description: "A pet",
					},
					"Error": {
						Type:        "object",
						Description: "An error",
					},
				},
			},
		}

		result := &parser.ParseResult{
			Document:   doc,
			OASVersion: parser.OASVersion303,
		}

		collector, err := CollectSchemas(result)
		require.NoError(t, err)

		assert.Len(t, collector.Components, 2)
		assert.Len(t, collector.Inline, 0)

		// Verify all components are marked as such
		for _, info := range collector.Components {
			assert.True(t, info.IsComponent)
			assert.NotEmpty(t, info.Name)
		}
	})

	t.Run("OAS2 definitions", func(t *testing.T) {
		doc := &parser.OAS2Document{
			Swagger: "2.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
			Definitions: map[string]*parser.Schema{
				"User": {
					Type:        "object",
					Description: "A user",
				},
			},
		}

		result := &parser.ParseResult{
			Document:   doc,
			OASVersion: parser.OASVersion20,
		}

		collector, err := CollectSchemas(result)
		require.NoError(t, err)

		assert.Len(t, collector.Components, 1)
		assert.Equal(t, "User", collector.Components[0].Name)
		assert.True(t, collector.Components[0].IsComponent)
	})
}

func TestCollectSchemas_Inline(t *testing.T) {
	t.Run("OAS3 inline schemas in operations", func(t *testing.T) {
		doc := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
			Paths: parser.Paths{
				"/pets": &parser.PathItem{
					Get: &parser.Operation{
						OperationID: "listPets",
						Responses: &parser.Responses{
							Codes: map[string]*parser.Response{
								"200": {
									Description: "OK",
									Content: map[string]*parser.MediaType{
										"application/json": {
											Schema: &parser.Schema{
												Type: "array",
												Items: &parser.Schema{
													Type: "object",
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
		}

		result := &parser.ParseResult{
			Document:   doc,
			OASVersion: parser.OASVersion303,
		}

		collector, err := CollectSchemas(result)
		require.NoError(t, err)

		// Should have inline schemas (response schema and items)
		assert.NotEmpty(t, collector.Inline)
		assert.Empty(t, collector.Components)

		// All inline schemas should not be components
		for _, info := range collector.Inline {
			assert.False(t, info.IsComponent)
		}
	})

	t.Run("OAS2 inline schemas in operations", func(t *testing.T) {
		doc := &parser.OAS2Document{
			Swagger: "2.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
			Paths: parser.Paths{
				"/users": &parser.PathItem{
					Post: &parser.Operation{
						OperationID: "createUser",
						Parameters: []*parser.Parameter{
							{
								Name:   "body",
								In:     "body",
								Schema: &parser.Schema{Type: "object"},
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

		collector, err := CollectSchemas(result)
		require.NoError(t, err)

		assert.NotEmpty(t, collector.Inline)
	})
}

func TestCollectSchemas_ByPath(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	collector, err := CollectSchemas(result)
	require.NoError(t, err)

	// Lookup by path should work
	petSchema, ok := collector.ByPath["$.components.schemas['Pet']"]
	require.True(t, ok, "should find Pet schema by path")
	assert.Equal(t, "Pet", petSchema.Name)
	assert.Equal(t, "object", petSchema.Schema.Type)
}

func TestCollectSchemas_ByName(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet":   {Type: "object", Description: "A pet"},
				"Error": {Type: "object", Description: "An error"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	collector, err := CollectSchemas(result)
	require.NoError(t, err)

	// Lookup by name should work for component schemas
	petSchema, ok := collector.ByName["Pet"]
	require.True(t, ok, "should find Pet schema by name")
	assert.Equal(t, "A pet", petSchema.Schema.Description)

	errorSchema, ok := collector.ByName["Error"]
	require.True(t, ok, "should find Error schema by name")
	assert.Equal(t, "An error", errorSchema.Schema.Description)
}

func TestCollectSchemas_Empty(t *testing.T) {
	t.Run("OAS3 empty document", func(t *testing.T) {
		doc := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		}

		result := &parser.ParseResult{
			Document:   doc,
			OASVersion: parser.OASVersion303,
		}

		collector, err := CollectSchemas(result)
		require.NoError(t, err)

		assert.Empty(t, collector.All)
		assert.Empty(t, collector.Components)
		assert.Empty(t, collector.Inline)
		assert.Empty(t, collector.ByPath)
		assert.Empty(t, collector.ByName)
	})

	t.Run("OAS2 empty document", func(t *testing.T) {
		doc := &parser.OAS2Document{
			Swagger: "2.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		}

		result := &parser.ParseResult{
			Document:   doc,
			OASVersion: parser.OASVersion20,
		}

		collector, err := CollectSchemas(result)
		require.NoError(t, err)

		assert.Empty(t, collector.All)
	})
}

func TestCollectSchemas_NestedSchemas(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
						"age":  {Type: "integer"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	collector, err := CollectSchemas(result)
	require.NoError(t, err)

	// Should collect all schemas: Pet + 2 properties
	assert.Len(t, collector.All, 3)

	// All schemas within components have names (Pet, name, age)
	// The top-level schema has the component name, nested schemas have property names
	assert.Len(t, collector.ByName, 3)
	_, ok := collector.ByName["Pet"]
	assert.True(t, ok, "Pet schema should be in ByName")

	// Property schemas are also named within components context
	_, ok = collector.ByName["name"]
	assert.True(t, ok, "name property schema should be in ByName")
	_, ok = collector.ByName["age"]
	assert.True(t, ok, "age property schema should be in ByName")
}

func TestCollectSchemas_ErrorHandling(t *testing.T) {
	// Nil result should return error
	collector, err := CollectSchemas(nil)
	require.Error(t, err)
	assert.Nil(t, collector)
}

func TestCollectOperations_All(t *testing.T) {
	t.Run("OAS3 operations", func(t *testing.T) {
		doc := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
			Paths: parser.Paths{
				"/pets": &parser.PathItem{
					Get:  &parser.Operation{OperationID: "listPets"},
					Post: &parser.Operation{OperationID: "createPet"},
				},
				"/pets/{petId}": &parser.PathItem{
					Get:    &parser.Operation{OperationID: "getPet"},
					Delete: &parser.Operation{OperationID: "deletePet"},
				},
			},
		}

		result := &parser.ParseResult{
			Document:   doc,
			OASVersion: parser.OASVersion303,
		}

		collector, err := CollectOperations(result)
		require.NoError(t, err)

		assert.Len(t, collector.All, 4)
	})

	t.Run("OAS2 operations", func(t *testing.T) {
		doc := &parser.OAS2Document{
			Swagger: "2.0",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
			Paths: parser.Paths{
				"/users": &parser.PathItem{
					Get:  &parser.Operation{OperationID: "listUsers"},
					Post: &parser.Operation{OperationID: "createUser"},
				},
			},
		}

		result := &parser.ParseResult{
			Document:   doc,
			OASVersion: parser.OASVersion20,
		}

		collector, err := CollectOperations(result)
		require.NoError(t, err)

		assert.Len(t, collector.All, 2)
	})
}

func TestCollectOperations_ByPath(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get:  &parser.Operation{OperationID: "listPets"},
				Post: &parser.Operation{OperationID: "createPet"},
			},
			"/pets/{petId}": &parser.PathItem{
				Get: &parser.Operation{OperationID: "getPet"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	collector, err := CollectOperations(result)
	require.NoError(t, err)

	// Check /pets path has 2 operations
	petsOps, ok := collector.ByPath["/pets"]
	require.True(t, ok)
	assert.Len(t, petsOps, 2)

	// Check /pets/{petId} has 1 operation
	petIdOps, ok := collector.ByPath["/pets/{petId}"]
	require.True(t, ok)
	assert.Len(t, petIdOps, 1)
	assert.Equal(t, "getPet", petIdOps[0].Operation.OperationID)
}

func TestCollectOperations_ByMethod(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get:  &parser.Operation{OperationID: "listPets"},
				Post: &parser.Operation{OperationID: "createPet"},
			},
			"/users": &parser.PathItem{
				Get:  &parser.Operation{OperationID: "listUsers"},
				Post: &parser.Operation{OperationID: "createUser"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	collector, err := CollectOperations(result)
	require.NoError(t, err)

	// Should have 2 GET operations
	getOps, ok := collector.ByMethod["get"]
	require.True(t, ok)
	assert.Len(t, getOps, 2)

	// Should have 2 POST operations
	postOps, ok := collector.ByMethod["post"]
	require.True(t, ok)
	assert.Len(t, postOps, 2)
}

func TestCollectOperations_ByTag(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Tags:        []string{"pets"},
				},
				Post: &parser.Operation{
					OperationID: "createPet",
					Tags:        []string{"pets", "admin"},
				},
			},
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
					Tags:        []string{"users"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	collector, err := CollectOperations(result)
	require.NoError(t, err)

	// pets tag should have 2 operations
	petsOps, ok := collector.ByTag["pets"]
	require.True(t, ok)
	assert.Len(t, petsOps, 2)

	// admin tag should have 1 operation
	adminOps, ok := collector.ByTag["admin"]
	require.True(t, ok)
	assert.Len(t, adminOps, 1)
	assert.Equal(t, "createPet", adminOps[0].Operation.OperationID)

	// users tag should have 1 operation
	usersOps, ok := collector.ByTag["users"]
	require.True(t, ok)
	assert.Len(t, usersOps, 1)
}

func TestCollectOperations_NoTags(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					// No tags
				},
			},
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
					Tags:        []string{"users"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	collector, err := CollectOperations(result)
	require.NoError(t, err)

	// All operations collected
	assert.Len(t, collector.All, 2)

	// Only one tag group (users)
	assert.Len(t, collector.ByTag, 1)
	_, ok := collector.ByTag["users"]
	assert.True(t, ok)

	// Operations without tags don't appear in ByTag
	for tag, ops := range collector.ByTag {
		for _, op := range ops {
			assert.NotEmpty(t, op.Operation.Tags, "operations in ByTag[%s] should have tags", tag)
		}
	}
}

func TestCollectOperations_Empty(t *testing.T) {
	t.Run("OAS3 no paths", func(t *testing.T) {
		doc := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		}

		result := &parser.ParseResult{
			Document:   doc,
			OASVersion: parser.OASVersion303,
		}

		collector, err := CollectOperations(result)
		require.NoError(t, err)

		assert.Empty(t, collector.All)
		assert.Empty(t, collector.ByPath)
		assert.Empty(t, collector.ByMethod)
		assert.Empty(t, collector.ByTag)
	})

	t.Run("OAS3 paths with no operations", func(t *testing.T) {
		doc := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
			Paths: parser.Paths{
				"/empty": &parser.PathItem{},
			},
		}

		result := &parser.ParseResult{
			Document:   doc,
			OASVersion: parser.OASVersion303,
		}

		collector, err := CollectOperations(result)
		require.NoError(t, err)

		assert.Empty(t, collector.All)
	})
}

func TestCollectOperations_OperationInfo(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets/{petId}": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPet",
					Summary:     "Get a pet by ID",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	collector, err := CollectOperations(result)
	require.NoError(t, err)

	require.Len(t, collector.All, 1)
	info := collector.All[0]

	assert.Equal(t, "/pets/{petId}", info.PathTemplate)
	assert.Equal(t, "get", info.Method)
	assert.Equal(t, "$.paths['/pets/{petId}'].get", info.JSONPath)
	assert.Equal(t, "getPet", info.Operation.OperationID)
	assert.Equal(t, "Get a pet by ID", info.Operation.Summary)
}

func TestCollectOperations_ErrorHandling(t *testing.T) {
	// Nil result should return error
	collector, err := CollectOperations(nil)
	require.Error(t, err)
	assert.Nil(t, collector)
}

func TestCollectOperations_AllMethods(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/resource": &parser.PathItem{
				Get:     &parser.Operation{OperationID: "get"},
				Post:    &parser.Operation{OperationID: "post"},
				Put:     &parser.Operation{OperationID: "put"},
				Delete:  &parser.Operation{OperationID: "delete"},
				Patch:   &parser.Operation{OperationID: "patch"},
				Options: &parser.Operation{OperationID: "options"},
				Head:    &parser.Operation{OperationID: "head"},
				Trace:   &parser.Operation{OperationID: "trace"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	collector, err := CollectOperations(result)
	require.NoError(t, err)

	assert.Len(t, collector.All, 8)
	assert.Len(t, collector.ByMethod, 8)

	// Verify all methods are present
	expectedMethods := []string{"get", "post", "put", "delete", "patch", "options", "head", "trace"}
	for _, method := range expectedMethods {
		ops, ok := collector.ByMethod[method]
		assert.True(t, ok, "should have method %s", method)
		assert.Len(t, ops, 1)
	}
}

func TestCollectSchemas_MixedComponentsAndInline(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{
											Type: "array",
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
				"Pet": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	collector, err := CollectSchemas(result)
	require.NoError(t, err)

	// Total = 1 component + 1 inline
	assert.Len(t, collector.All, 2)
	assert.Len(t, collector.Components, 1)
	assert.Len(t, collector.Inline, 1)

	// Component should be Pet
	assert.Equal(t, "Pet", collector.Components[0].Name)
	assert.True(t, collector.Components[0].IsComponent)

	// Inline should not have a name and not be a component
	assert.Empty(t, collector.Inline[0].Name)
	assert.False(t, collector.Inline[0].IsComponent)
}
