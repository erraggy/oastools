package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSchemaEquals_NestedSchemas tests nested *Schema pointer fields.
func TestSchemaEquals_NestedSchemas(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		// Contains
		{
			name: "Contains both nil",
			a:    &Schema{Contains: nil},
			b:    &Schema{Contains: nil},
			want: true,
		},
		{
			name: "Contains same",
			a:    &Schema{Contains: &Schema{Type: "string"}},
			b:    &Schema{Contains: &Schema{Type: "string"}},
			want: true,
		},
		{
			name: "Contains different",
			a:    &Schema{Contains: &Schema{Type: "string"}},
			b:    &Schema{Contains: &Schema{Type: "integer"}},
			want: false,
		},
		{
			name: "Contains nil vs set",
			a:    &Schema{Contains: nil},
			b:    &Schema{Contains: &Schema{Type: "string"}},
			want: false,
		},
		// Not
		{
			name: "Not both nil",
			a:    &Schema{Not: nil},
			b:    &Schema{Not: nil},
			want: true,
		},
		{
			name: "Not same",
			a:    &Schema{Not: &Schema{Type: "null"}},
			b:    &Schema{Not: &Schema{Type: "null"}},
			want: true,
		},
		{
			name: "Not different",
			a:    &Schema{Not: &Schema{Type: "null"}},
			b:    &Schema{Not: &Schema{Type: "string"}},
			want: false,
		},
		// If/Then/Else
		{
			name: "If same",
			a:    &Schema{If: &Schema{Properties: map[string]*Schema{"type": {Const: "A"}}}},
			b:    &Schema{If: &Schema{Properties: map[string]*Schema{"type": {Const: "A"}}}},
			want: true,
		},
		{
			name: "If different",
			a:    &Schema{If: &Schema{Properties: map[string]*Schema{"type": {Const: "A"}}}},
			b:    &Schema{If: &Schema{Properties: map[string]*Schema{"type": {Const: "B"}}}},
			want: false,
		},
		{
			name: "Then same",
			a:    &Schema{Then: &Schema{Required: []string{"a"}}},
			b:    &Schema{Then: &Schema{Required: []string{"a"}}},
			want: true,
		},
		{
			name: "Then different",
			a:    &Schema{Then: &Schema{Required: []string{"a"}}},
			b:    &Schema{Then: &Schema{Required: []string{"b"}}},
			want: false,
		},
		{
			name: "Else same",
			a:    &Schema{Else: &Schema{Required: []string{"x"}}},
			b:    &Schema{Else: &Schema{Required: []string{"x"}}},
			want: true,
		},
		{
			name: "Else different",
			a:    &Schema{Else: &Schema{Required: []string{"x"}}},
			b:    &Schema{Else: &Schema{Required: []string{"y"}}},
			want: false,
		},
		// PropertyNames
		{
			name: "PropertyNames same",
			a:    &Schema{PropertyNames: &Schema{Pattern: "^[a-z]+$"}},
			b:    &Schema{PropertyNames: &Schema{Pattern: "^[a-z]+$"}},
			want: true,
		},
		{
			name: "PropertyNames different",
			a:    &Schema{PropertyNames: &Schema{Pattern: "^[a-z]+$"}},
			b:    &Schema{PropertyNames: &Schema{Pattern: "^[A-Z]+$"}},
			want: false,
		},
		// ContentSchema
		{
			name: "ContentSchema same",
			a:    &Schema{ContentSchema: &Schema{Type: "object"}},
			b:    &Schema{ContentSchema: &Schema{Type: "object"}},
			want: true,
		},
		{
			name: "ContentSchema different",
			a:    &Schema{ContentSchema: &Schema{Type: "object"}},
			b:    &Schema{ContentSchema: &Schema{Type: "array"}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Equals(tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSchemaEquals_Items tests the polymorphic Items field.
func TestSchemaEquals_Items(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "Items both nil",
			a:    &Schema{Items: nil},
			b:    &Schema{Items: nil},
			want: true,
		},
		{
			name: "Items same *Schema",
			a:    &Schema{Items: &Schema{Type: "string"}},
			b:    &Schema{Items: &Schema{Type: "string"}},
			want: true,
		},
		{
			name: "Items different *Schema",
			a:    &Schema{Items: &Schema{Type: "string"}},
			b:    &Schema{Items: &Schema{Type: "integer"}},
			want: false,
		},
		{
			name: "Items both bool true",
			a:    &Schema{Items: true},
			b:    &Schema{Items: true},
			want: true,
		},
		{
			name: "Items both bool false",
			a:    &Schema{Items: false},
			b:    &Schema{Items: false},
			want: true,
		},
		{
			name: "Items bool vs *Schema",
			a:    &Schema{Items: true},
			b:    &Schema{Items: &Schema{Type: "string"}},
			want: false,
		},
		{
			name: "Items nil vs *Schema",
			a:    &Schema{Items: nil},
			b:    &Schema{Items: &Schema{Type: "string"}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Equals(tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSchemaEquals_UnevaluatedFields tests UnevaluatedItems and UnevaluatedProperties.
func TestSchemaEquals_UnevaluatedFields(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		// UnevaluatedItems
		{
			name: "UnevaluatedItems both nil",
			a:    &Schema{UnevaluatedItems: nil},
			b:    &Schema{UnevaluatedItems: nil},
			want: true,
		},
		{
			name: "UnevaluatedItems same bool",
			a:    &Schema{UnevaluatedItems: false},
			b:    &Schema{UnevaluatedItems: false},
			want: true,
		},
		{
			name: "UnevaluatedItems different bool",
			a:    &Schema{UnevaluatedItems: true},
			b:    &Schema{UnevaluatedItems: false},
			want: false,
		},
		{
			name: "UnevaluatedItems same *Schema",
			a:    &Schema{UnevaluatedItems: &Schema{Type: "string"}},
			b:    &Schema{UnevaluatedItems: &Schema{Type: "string"}},
			want: true,
		},
		{
			name: "UnevaluatedItems different *Schema",
			a:    &Schema{UnevaluatedItems: &Schema{Type: "string"}},
			b:    &Schema{UnevaluatedItems: &Schema{Type: "integer"}},
			want: false,
		},
		{
			name: "UnevaluatedItems bool vs *Schema",
			a:    &Schema{UnevaluatedItems: false},
			b:    &Schema{UnevaluatedItems: &Schema{Type: "string"}},
			want: false,
		},
		// UnevaluatedProperties
		{
			name: "UnevaluatedProperties both nil",
			a:    &Schema{UnevaluatedProperties: nil},
			b:    &Schema{UnevaluatedProperties: nil},
			want: true,
		},
		{
			name: "UnevaluatedProperties same bool",
			a:    &Schema{UnevaluatedProperties: false},
			b:    &Schema{UnevaluatedProperties: false},
			want: true,
		},
		{
			name: "UnevaluatedProperties same *Schema",
			a:    &Schema{UnevaluatedProperties: &Schema{Type: "string"}},
			b:    &Schema{UnevaluatedProperties: &Schema{Type: "string"}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Equals(tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSchemaEquals_ComplexSchemas tests real-world schema patterns.
func TestSchemaEquals_ComplexSchemas(t *testing.T) {
	// Typical Pet schema
	petSchema := func() *Schema {
		return &Schema{
			Type:     "object",
			Required: []string{"id", "name"},
			Properties: map[string]*Schema{
				"id":   {Type: "integer", Format: "int64"},
				"name": {Type: "string"},
				"tag":  {Type: "string"},
			},
		}
	}

	// Schema with allOf composition
	composedSchema := func() *Schema {
		return &Schema{
			AllOf: []*Schema{
				{Ref: "#/components/schemas/Pet"},
				{
					Type: "object",
					Properties: map[string]*Schema{
						"packSize": {
							Type:    "integer",
							Minimum: ptr(0.0),
						},
					},
					Required: []string{"packSize"},
				},
			},
		}
	}

	// Schema with discriminator
	discriminatedSchema := func() *Schema {
		return &Schema{
			OneOf: []*Schema{
				{Ref: "#/components/schemas/Cat"},
				{Ref: "#/components/schemas/Dog"},
			},
			Discriminator: &Discriminator{
				PropertyName: "petType",
				Mapping: map[string]string{
					"cat": "#/components/schemas/Cat",
					"dog": "#/components/schemas/Dog",
				},
			},
		}
	}

	// Deeply nested schema (4 levels)
	deeplyNestedSchema := func() *Schema {
		return &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"level1": {
					Type: "object",
					Properties: map[string]*Schema{
						"level2": {
							Type: "object",
							Properties: map[string]*Schema{
								"level3": {
									Type: "object",
									Properties: map[string]*Schema{
										"level4": {Type: "string"},
									},
								},
							},
						},
					},
				},
			},
		}
	}

	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "identical Pet schemas",
			a:    petSchema(),
			b:    petSchema(),
			want: true,
		},
		{
			name: "Pet schema vs modified Pet schema",
			a:    petSchema(),
			b: func() *Schema {
				s := petSchema()
				s.Properties["status"] = &Schema{Type: "string"}
				return s
			}(),
			want: false,
		},
		{
			name: "identical composed schemas",
			a:    composedSchema(),
			b:    composedSchema(),
			want: true,
		},
		{
			name: "identical discriminated schemas",
			a:    discriminatedSchema(),
			b:    discriminatedSchema(),
			want: true,
		},
		{
			name: "discriminated schema with different mapping",
			a:    discriminatedSchema(),
			b: func() *Schema {
				s := discriminatedSchema()
				s.Discriminator.Mapping["bird"] = "#/components/schemas/Bird"
				return s
			}(),
			want: false,
		},
		{
			name: "identical deeply nested schemas",
			a:    deeplyNestedSchema(),
			b:    deeplyNestedSchema(),
			want: true,
		},
		{
			name: "deeply nested with difference at level 4",
			a:    deeplyNestedSchema(),
			b: func() *Schema {
				s := deeplyNestedSchema()
				s.Properties["level1"].Properties["level2"].Properties["level3"].Properties["level4"].Type = "integer"
				return s
			}(),
			want: false,
		},
		{
			name: "schema with all composition types",
			a: &Schema{
				AllOf: []*Schema{{Type: "object"}},
				OneOf: []*Schema{{Type: "string"}, {Type: "integer"}},
				AnyOf: []*Schema{{Minimum: ptr(0.0)}},
				Not:   &Schema{Type: "null"},
			},
			b: &Schema{
				AllOf: []*Schema{{Type: "object"}},
				OneOf: []*Schema{{Type: "string"}, {Type: "integer"}},
				AnyOf: []*Schema{{Minimum: ptr(0.0)}},
				Not:   &Schema{Type: "null"},
			},
			want: true,
		},
		{
			name: "schema with If/Then/Else",
			a: &Schema{
				If: &Schema{
					Properties: map[string]*Schema{
						"type": {Const: "premium"},
					},
				},
				Then: &Schema{
					Required: []string{"premiumFeature"},
				},
				Else: &Schema{
					Required: []string{"basicFeature"},
				},
			},
			b: &Schema{
				If: &Schema{
					Properties: map[string]*Schema{
						"type": {Const: "premium"},
					},
				},
				Then: &Schema{
					Required: []string{"premiumFeature"},
				},
				Else: &Schema{
					Required: []string{"basicFeature"},
				},
			},
			want: true,
		},
		{
			name: "OAS 3.1 nullable type array",
			a: &Schema{
				Type: []string{"string", "null"},
			},
			b: &Schema{
				Type: []string{"string", "null"},
			},
			want: true,
		},
		{
			name: "OAS 3.0 nullable flag vs OAS 3.1 type array",
			a: &Schema{
				Type:     "string",
				Nullable: true,
			},
			b: &Schema{
				Type: []string{"string", "null"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Equals(tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSchemaEquals_CyclicReferences tests that cyclic schema references don't cause infinite loops.
func TestSchemaEquals_CyclicReferences(t *testing.T) {
	t.Run("direct self-reference via Properties", func(t *testing.T) {
		// Create a schema that references itself
		schemaA := &Schema{Type: "object"}
		schemaA.Properties = map[string]*Schema{
			"self": schemaA,
		}

		schemaB := &Schema{Type: "object"}
		schemaB.Properties = map[string]*Schema{
			"self": schemaB,
		}

		// Should complete without infinite loop and return true (same structure)
		assert.True(t, schemaA.Equals(schemaB))
	})

	t.Run("direct self-reference via Items", func(t *testing.T) {
		schemaA := &Schema{Type: "array"}
		schemaA.Items = schemaA

		schemaB := &Schema{Type: "array"}
		schemaB.Items = schemaB

		assert.True(t, schemaA.Equals(schemaB))
	})

	t.Run("mutual reference between two schemas", func(t *testing.T) {
		schemaA1 := &Schema{Type: "object"}
		schemaA2 := &Schema{Type: "string"}
		schemaA1.Properties = map[string]*Schema{"child": schemaA2}
		schemaA2.Properties = map[string]*Schema{"parent": schemaA1}

		schemaB1 := &Schema{Type: "object"}
		schemaB2 := &Schema{Type: "string"}
		schemaB1.Properties = map[string]*Schema{"child": schemaB2}
		schemaB2.Properties = map[string]*Schema{"parent": schemaB1}

		assert.True(t, schemaA1.Equals(schemaB1))
	})

	t.Run("cyclic but different structures return false", func(t *testing.T) {
		schemaA := &Schema{Type: "object"}
		schemaA.Properties = map[string]*Schema{
			"self": schemaA,
		}

		schemaB := &Schema{Type: "array"} // Different type
		schemaB.Items = schemaB

		assert.False(t, schemaA.Equals(schemaB))
	})

	t.Run("deep cycle via allOf", func(t *testing.T) {
		schemaA := &Schema{Type: "object"}
		schemaA.AllOf = []*Schema{schemaA}

		schemaB := &Schema{Type: "object"}
		schemaB.AllOf = []*Schema{schemaB}

		assert.True(t, schemaA.Equals(schemaB))
	})
}
