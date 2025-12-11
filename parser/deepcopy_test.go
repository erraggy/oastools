package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaDeepCopy_NilVsEmpty verifies that nil and empty values are preserved correctly.
func TestSchemaDeepCopy_NilVsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		schema *Schema
		check  func(t *testing.T, cp *Schema)
	}{
		{
			name:   "nil Properties",
			schema: &Schema{Properties: nil},
			check: func(t *testing.T, cp *Schema) {
				assert.Nil(t, cp.Properties)
			},
		},
		{
			name:   "empty Properties",
			schema: &Schema{Properties: map[string]*Schema{}},
			check: func(t *testing.T, cp *Schema) {
				require.NotNil(t, cp.Properties)
				assert.Empty(t, cp.Properties)
			},
		},
		{
			name:   "nil Required",
			schema: &Schema{Required: nil},
			check: func(t *testing.T, cp *Schema) {
				assert.Nil(t, cp.Required)
			},
		},
		{
			name:   "empty Required",
			schema: &Schema{Required: []string{}},
			check: func(t *testing.T, cp *Schema) {
				require.NotNil(t, cp.Required)
				assert.Empty(t, cp.Required)
			},
		},
		{
			name:   "nil Extra",
			schema: &Schema{Extra: nil},
			check: func(t *testing.T, cp *Schema) {
				assert.Nil(t, cp.Extra)
			},
		},
		{
			name:   "empty Extra",
			schema: &Schema{Extra: map[string]any{}},
			check: func(t *testing.T, cp *Schema) {
				require.NotNil(t, cp.Extra)
				assert.Empty(t, cp.Extra)
			},
		},
		{
			name:   "nil Enum",
			schema: &Schema{Enum: nil},
			check: func(t *testing.T, cp *Schema) {
				assert.Nil(t, cp.Enum)
			},
		},
		{
			name:   "empty Enum",
			schema: &Schema{Enum: []any{}},
			check: func(t *testing.T, cp *Schema) {
				require.NotNil(t, cp.Enum)
				assert.Empty(t, cp.Enum)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := tt.schema.DeepCopy()
			tt.check(t, cp)
		})
	}
}

// TestSchemaDeepCopy_AnyFields tests deep copying of OAS-typed polymorphic fields.
func TestSchemaDeepCopy_AnyFields(t *testing.T) {
	t.Run("Type as string", func(t *testing.T) {
		s := &Schema{Type: "string"}
		cp := s.DeepCopy()
		assert.Equal(t, "string", cp.Type)
	})

	t.Run("Type as []string (OAS 3.1)", func(t *testing.T) {
		s := &Schema{Type: []string{"string", "null"}}
		cp := s.DeepCopy()

		typeArr, ok := cp.Type.([]string)
		require.True(t, ok, "Type should be []string")
		assert.Equal(t, []string{"string", "null"}, typeArr)

		// Verify independence - modifying original doesn't affect copy
		s.Type.([]string)[0] = "modified"
		assert.Equal(t, "string", typeArr[0])
	})

	t.Run("Items as *Schema", func(t *testing.T) {
		itemSchema := &Schema{Type: "string", Format: "email"}
		s := &Schema{Type: "array", Items: itemSchema}
		cp := s.DeepCopy()

		cpItems, ok := cp.Items.(*Schema)
		require.True(t, ok, "Items should be *Schema")
		assert.NotSame(t, itemSchema, cpItems, "Should be different pointers")
		assert.Equal(t, "email", cpItems.Format)

		// Verify independence
		itemSchema.Format = "uri"
		assert.Equal(t, "email", cpItems.Format)
	})

	t.Run("Items as bool (OAS 3.1)", func(t *testing.T) {
		s := &Schema{Type: "array", Items: false}
		cp := s.DeepCopy()
		assert.Equal(t, false, cp.Items)
	})

	t.Run("AdditionalProperties as *Schema", func(t *testing.T) {
		addProps := &Schema{Type: "integer"}
		s := &Schema{Type: "object", AdditionalProperties: addProps}
		cp := s.DeepCopy()

		cpAddProps, ok := cp.AdditionalProperties.(*Schema)
		require.True(t, ok, "AdditionalProperties should be *Schema")
		assert.NotSame(t, addProps, cpAddProps)
		assert.Equal(t, "integer", cpAddProps.Type)
	})

	t.Run("AdditionalProperties as bool", func(t *testing.T) {
		s := &Schema{Type: "object", AdditionalProperties: false}
		cp := s.DeepCopy()
		assert.Equal(t, false, cp.AdditionalProperties)
	})

	t.Run("ExclusiveMinimum as bool (OAS 3.0)", func(t *testing.T) {
		minVal := 0.0
		s := &Schema{Minimum: &minVal, ExclusiveMinimum: true}
		cp := s.DeepCopy()
		assert.Equal(t, true, cp.ExclusiveMinimum)
	})

	t.Run("ExclusiveMinimum as number (OAS 3.1)", func(t *testing.T) {
		s := &Schema{ExclusiveMinimum: 5.0}
		cp := s.DeepCopy()
		assert.Equal(t, 5.0, cp.ExclusiveMinimum)
	})

	t.Run("Default with nested map", func(t *testing.T) {
		s := &Schema{
			Default: map[string]any{
				"nested": map[string]any{
					"value": "original",
				},
			},
		}
		cp := s.DeepCopy()

		// Modify original
		s.Default.(map[string]any)["nested"].(map[string]any)["value"] = "modified"

		// Verify copy is independent
		cpDefault := cp.Default.(map[string]any)
		cpNested := cpDefault["nested"].(map[string]any)
		assert.Equal(t, "original", cpNested["value"])
	})

	t.Run("Enum with mixed types", func(t *testing.T) {
		s := &Schema{
			Enum: []any{"red", "green", "blue", 1, 2, 3, true, nil},
		}
		cp := s.DeepCopy()

		assert.Equal(t, s.Enum, cp.Enum)

		// Verify independence
		s.Enum[0] = "modified"
		assert.Equal(t, "red", cp.Enum[0])
	})
}

// TestSchemaDeepCopy_Extensions tests deep copying of x-* extension fields.
func TestSchemaDeepCopy_Extensions(t *testing.T) {
	t.Run("Simple extensions", func(t *testing.T) {
		s := &Schema{
			Type: "object",
			Extra: map[string]any{
				"x-deprecated-since": "v2.0",
				"x-custom-flag":      true,
			},
		}
		cp := s.DeepCopy()

		assert.Equal(t, "v2.0", cp.Extra["x-deprecated-since"])
		assert.Equal(t, true, cp.Extra["x-custom-flag"])

		// Verify independence
		s.Extra["x-deprecated-since"] = "modified"
		assert.Equal(t, "v2.0", cp.Extra["x-deprecated-since"])
	})

	t.Run("Nested extensions", func(t *testing.T) {
		s := &Schema{
			Type: "object",
			Extra: map[string]any{
				"x-custom": map[string]any{
					"nested": "value",
					"deep": map[string]any{
						"level": 3,
					},
				},
			},
		}
		cp := s.DeepCopy()

		// Modify original
		s.Extra["x-custom"].(map[string]any)["nested"] = "modified"
		s.Extra["x-custom"].(map[string]any)["deep"].(map[string]any)["level"] = 999

		// Verify copy is independent
		assert.Equal(t, "value", cp.Extra["x-custom"].(map[string]any)["nested"])
		assert.Equal(t, 3, cp.Extra["x-custom"].(map[string]any)["deep"].(map[string]any)["level"])
	})
}

// TestSchemaDeepCopy_PointerFields tests deep copying of pointer fields.
func TestSchemaDeepCopy_PointerFields(t *testing.T) {
	t.Run("Primitive pointers", func(t *testing.T) {
		multipleOf := 2.0
		maximum := 100.0
		maxLength := 50
		s := &Schema{
			MultipleOf: &multipleOf,
			Maximum:    &maximum,
			MaxLength:  &maxLength,
		}
		cp := s.DeepCopy()

		// Verify values are copied
		require.NotNil(t, cp.MultipleOf)
		require.NotNil(t, cp.Maximum)
		require.NotNil(t, cp.MaxLength)
		assert.Equal(t, 2.0, *cp.MultipleOf)
		assert.Equal(t, 100.0, *cp.Maximum)
		assert.Equal(t, 50, *cp.MaxLength)

		// Verify pointers are different
		assert.NotSame(t, s.MultipleOf, cp.MultipleOf)
		assert.NotSame(t, s.Maximum, cp.Maximum)
		assert.NotSame(t, s.MaxLength, cp.MaxLength)

		// Verify independence
		*s.MultipleOf = 999.0
		assert.Equal(t, 2.0, *cp.MultipleOf)
	})

	t.Run("Struct pointers", func(t *testing.T) {
		s := &Schema{
			Discriminator: &Discriminator{
				PropertyName: "type",
				Mapping: map[string]string{
					"dog": "#/components/schemas/Dog",
				},
			},
			XML: &XML{
				Name:      "user",
				Namespace: "http://example.com",
			},
		}
		cp := s.DeepCopy()

		// Verify values are copied
		require.NotNil(t, cp.Discriminator)
		require.NotNil(t, cp.XML)
		assert.Equal(t, "type", cp.Discriminator.PropertyName)
		assert.Equal(t, "user", cp.XML.Name)

		// Verify pointers are different
		assert.NotSame(t, s.Discriminator, cp.Discriminator)
		assert.NotSame(t, s.XML, cp.XML)

		// Verify independence
		s.Discriminator.PropertyName = "modified"
		assert.Equal(t, "type", cp.Discriminator.PropertyName)
	})
}

// TestSchemaDeepCopy_NestedSchemas tests deep copying of nested schema structures.
func TestSchemaDeepCopy_NestedSchemas(t *testing.T) {
	t.Run("AllOf composition", func(t *testing.T) {
		s := &Schema{
			AllOf: []*Schema{
				{Type: "object", Properties: map[string]*Schema{
					"id": {Type: "integer"},
				}},
				{Type: "object", Properties: map[string]*Schema{
					"name": {Type: "string"},
				}},
			},
		}
		cp := s.DeepCopy()

		require.Len(t, cp.AllOf, 2)
		assert.NotSame(t, s.AllOf[0], cp.AllOf[0])
		assert.NotSame(t, s.AllOf[1], cp.AllOf[1])

		// Verify nested properties are independent
		s.AllOf[0].Properties["id"].Type = "modified"
		assert.Equal(t, "integer", cp.AllOf[0].Properties["id"].Type)
	})

	t.Run("Properties map", func(t *testing.T) {
		s := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"user": {
					Type: "object",
					Properties: map[string]*Schema{
						"email": {Type: "string", Format: "email"},
					},
				},
			},
		}
		cp := s.DeepCopy()

		// Verify nested structure
		require.NotNil(t, cp.Properties["user"])
		require.NotNil(t, cp.Properties["user"].Properties["email"])
		assert.Equal(t, "email", cp.Properties["user"].Properties["email"].Format)

		// Verify independence
		s.Properties["user"].Properties["email"].Format = "modified"
		assert.Equal(t, "email", cp.Properties["user"].Properties["email"].Format)
	})
}

// TestOAS3DocumentDeepCopy tests deep copying of OAS3Document.
func TestOAS3DocumentDeepCopy(t *testing.T) {
	t.Run("Complete document", func(t *testing.T) {
		doc := &OAS3Document{
			OpenAPI: "3.0.3",
			Info: &Info{
				Title:   "Test API",
				Version: "1.0.0",
			},
			Servers: []*Server{
				{URL: "https://api.example.com"},
			},
			Paths: Paths{
				"/users": &PathItem{
					Get: &Operation{
						Summary: "List users",
						Responses: &Responses{
							Codes: map[string]*Response{
								"200": {Description: "Success"},
							},
						},
					},
				},
			},
			Components: &Components{
				Schemas: map[string]*Schema{
					"User": {Type: "object"},
				},
			},
		}
		cp := doc.DeepCopy()

		// Verify structure
		require.NotNil(t, cp.Info)
		require.Len(t, cp.Servers, 1)
		require.NotNil(t, cp.Paths["/users"])
		require.NotNil(t, cp.Components)

		// Verify independence
		doc.Info.Title = "Modified"
		doc.Servers[0].URL = "https://modified.com"
		doc.Paths["/users"].Get.Summary = "Modified"

		assert.Equal(t, "Test API", cp.Info.Title)
		assert.Equal(t, "https://api.example.com", cp.Servers[0].URL)
		assert.Equal(t, "List users", cp.Paths["/users"].Get.Summary)
	})
}

// TestOAS2DocumentDeepCopy tests deep copying of OAS2Document.
func TestOAS2DocumentDeepCopy(t *testing.T) {
	t.Run("Complete document", func(t *testing.T) {
		doc := &OAS2Document{
			Swagger: "2.0",
			Info: &Info{
				Title:   "Test API",
				Version: "1.0.0",
			},
			Host:     "api.example.com",
			BasePath: "/v1",
			Schemes:  []string{"https"},
			Paths: Paths{
				"/users": &PathItem{
					Get: &Operation{
						Summary: "List users",
					},
				},
			},
			Definitions: map[string]*Schema{
				"User": {Type: "object"},
			},
		}
		cp := doc.DeepCopy()

		// Verify structure
		require.NotNil(t, cp.Info)
		require.Len(t, cp.Schemes, 1)
		require.NotNil(t, cp.Paths["/users"])
		require.NotNil(t, cp.Definitions["User"])

		// Verify independence
		doc.Info.Title = "Modified"
		doc.Schemes[0] = "http"
		doc.Paths["/users"].Get.Summary = "Modified"

		assert.Equal(t, "Test API", cp.Info.Title)
		assert.Equal(t, "https", cp.Schemes[0])
		assert.Equal(t, "List users", cp.Paths["/users"].Get.Summary)
	})
}

// TestParameterDeepCopy tests deep copying of Parameter with $ref.
func TestParameterDeepCopy(t *testing.T) {
	t.Run("Parameter with $ref only", func(t *testing.T) {
		// This is the key test for Issue #103 - parameters with $ref
		// should not have empty name/in added during copy
		param := &Parameter{
			Ref: "#/components/parameters/LimitParam",
		}
		cp := param.DeepCopy()

		assert.Equal(t, "#/components/parameters/LimitParam", cp.Ref)
		assert.Empty(t, cp.Name, "Name should remain empty for $ref parameter")
		assert.Empty(t, cp.In, "In should remain empty for $ref parameter")
	})

	t.Run("Parameter with all fields", func(t *testing.T) {
		explode := true
		param := &Parameter{
			Name:        "limit",
			In:          "query",
			Description: "Number of items",
			Required:    true,
			Explode:     &explode,
			Schema: &Schema{
				Type: "integer",
			},
			Example: 10,
			Extra: map[string]any{
				"x-custom": "value",
			},
		}
		cp := param.DeepCopy()

		// Verify all fields copied
		assert.Equal(t, "limit", cp.Name)
		assert.Equal(t, "query", cp.In)
		assert.Equal(t, "Number of items", cp.Description)
		assert.True(t, cp.Required)
		require.NotNil(t, cp.Explode)
		assert.True(t, *cp.Explode)
		require.NotNil(t, cp.Schema)
		assert.Equal(t, "integer", cp.Schema.Type)
		assert.Equal(t, 10, cp.Example)

		// Verify independence
		*param.Explode = false
		param.Schema.Type = "modified"
		assert.True(t, *cp.Explode)
		assert.Equal(t, "integer", cp.Schema.Type)
	})
}

// TestResponseDeepCopy tests deep copying of Response.
func TestResponseDeepCopy(t *testing.T) {
	t.Run("Response with $ref only", func(t *testing.T) {
		// Issue #103 related - responses with $ref should preserve empty description
		resp := &Response{
			Ref: "#/components/responses/NotFound",
		}
		cp := resp.DeepCopy()

		assert.Equal(t, "#/components/responses/NotFound", cp.Ref)
		assert.Empty(t, cp.Description, "Description should remain empty for $ref response")
	})
}

// TestSecurityRequirementDeepCopy tests deep copying of security requirements.
func TestSecurityRequirementDeepCopy(t *testing.T) {
	t.Run("Security requirements", func(t *testing.T) {
		reqs := []SecurityRequirement{
			{
				"oauth2": {"read:users", "write:users"},
			},
			{
				"apiKey": {},
			},
		}
		cp := deepCopySecurityRequirements(reqs)

		require.Len(t, cp, 2)
		assert.Equal(t, []string{"read:users", "write:users"}, cp[0]["oauth2"])
		assert.Empty(t, cp[1]["apiKey"])

		// Verify independence
		reqs[0]["oauth2"][0] = "modified"
		assert.Equal(t, "read:users", cp[0]["oauth2"][0])
	})
}

// TestNilReceiver tests that nil receivers are handled correctly.
func TestNilReceiver(t *testing.T) {
	t.Run("nil Schema", func(t *testing.T) {
		var s *Schema
		cp := s.DeepCopy()
		assert.Nil(t, cp)
	})

	t.Run("nil OAS3Document", func(t *testing.T) {
		var doc *OAS3Document
		cp := doc.DeepCopy()
		assert.Nil(t, cp)
	})

	t.Run("nil Parameter", func(t *testing.T) {
		var p *Parameter
		cp := p.DeepCopy()
		assert.Nil(t, cp)
	})
}

// TestCallbackDeepCopy tests deep copying of Callback type alias.
func TestCallbackDeepCopy(t *testing.T) {
	callback := Callback{
		"{$request.body#/callbackUrl}": &PathItem{
			Post: &Operation{
				Summary: "Callback notification",
			},
		},
	}
	cp := deepCopyCallback(callback)

	require.NotNil(t, cp["{$request.body#/callbackUrl}"])
	assert.Equal(t, "Callback notification", cp["{$request.body#/callbackUrl}"].Post.Summary)

	// Verify independence
	callback["{$request.body#/callbackUrl}"].Post.Summary = "Modified"
	assert.Equal(t, "Callback notification", cp["{$request.body#/callbackUrl}"].Post.Summary)
}
