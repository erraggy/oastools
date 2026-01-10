package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSchemaEquals_Properties tests the Properties map comparison.
func TestSchemaEquals_Properties(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "both nil Properties",
			a:    &Schema{Properties: nil},
			b:    &Schema{Properties: nil},
			want: true,
		},
		{
			name: "both empty Properties",
			a:    &Schema{Properties: map[string]*Schema{}},
			b:    &Schema{Properties: map[string]*Schema{}},
			want: true,
		},
		{
			name: "nil vs empty Properties",
			a:    &Schema{Properties: nil},
			b:    &Schema{Properties: map[string]*Schema{}},
			want: true,
		},
		{
			name: "same Properties",
			a: &Schema{Properties: map[string]*Schema{
				"id":   {Type: "integer"},
				"name": {Type: "string"},
			}},
			b: &Schema{Properties: map[string]*Schema{
				"id":   {Type: "integer"},
				"name": {Type: "string"},
			}},
			want: true,
		},
		{
			name: "different property names",
			a: &Schema{Properties: map[string]*Schema{
				"id": {Type: "integer"},
			}},
			b: &Schema{Properties: map[string]*Schema{
				"userId": {Type: "integer"},
			}},
			want: false,
		},
		{
			name: "different property schemas",
			a: &Schema{Properties: map[string]*Schema{
				"id": {Type: "integer"},
			}},
			b: &Schema{Properties: map[string]*Schema{
				"id": {Type: "string"},
			}},
			want: false,
		},
		{
			name: "different number of properties",
			a: &Schema{Properties: map[string]*Schema{
				"id":   {Type: "integer"},
				"name": {Type: "string"},
			}},
			b: &Schema{Properties: map[string]*Schema{
				"id": {Type: "integer"},
			}},
			want: false,
		},
		{
			name: "nested properties same",
			a: &Schema{Properties: map[string]*Schema{
				"user": {
					Type: "object",
					Properties: map[string]*Schema{
						"email": {Type: "string", Format: "email"},
					},
				},
			}},
			b: &Schema{Properties: map[string]*Schema{
				"user": {
					Type: "object",
					Properties: map[string]*Schema{
						"email": {Type: "string", Format: "email"},
					},
				},
			}},
			want: true,
		},
		{
			name: "nested properties different",
			a: &Schema{Properties: map[string]*Schema{
				"user": {
					Type: "object",
					Properties: map[string]*Schema{
						"email": {Type: "string", Format: "email"},
					},
				},
			}},
			b: &Schema{Properties: map[string]*Schema{
				"user": {
					Type: "object",
					Properties: map[string]*Schema{
						"email": {Type: "string", Format: "uri"},
					},
				},
			}},
			want: false,
		},
		{
			name: "deeply nested properties (3 levels)",
			a: &Schema{Properties: map[string]*Schema{
				"level1": {
					Type: "object",
					Properties: map[string]*Schema{
						"level2": {
							Type: "object",
							Properties: map[string]*Schema{
								"level3": {Type: "string"},
							},
						},
					},
				},
			}},
			b: &Schema{Properties: map[string]*Schema{
				"level1": {
					Type: "object",
					Properties: map[string]*Schema{
						"level2": {
							Type: "object",
							Properties: map[string]*Schema{
								"level3": {Type: "string"},
							},
						},
					},
				},
			}},
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

// TestSchemaEquals_Composition tests AllOf, OneOf, AnyOf comparisons.
func TestSchemaEquals_Composition(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		// AllOf
		{
			name: "AllOf both nil",
			a:    &Schema{AllOf: nil},
			b:    &Schema{AllOf: nil},
			want: true,
		},
		{
			name: "AllOf empty vs nil",
			a:    &Schema{AllOf: []*Schema{}},
			b:    &Schema{AllOf: nil},
			want: true,
		},
		{
			name: "AllOf same composition",
			a: &Schema{AllOf: []*Schema{
				{Type: "object"},
				{Properties: map[string]*Schema{"id": {Type: "integer"}}},
			}},
			b: &Schema{AllOf: []*Schema{
				{Type: "object"},
				{Properties: map[string]*Schema{"id": {Type: "integer"}}},
			}},
			want: true,
		},
		{
			name: "AllOf different length",
			a: &Schema{AllOf: []*Schema{
				{Type: "object"},
				{Properties: map[string]*Schema{"id": {Type: "integer"}}},
			}},
			b: &Schema{AllOf: []*Schema{
				{Type: "object"},
			}},
			want: false,
		},
		{
			name: "AllOf different content",
			a: &Schema{AllOf: []*Schema{
				{Type: "object"},
			}},
			b: &Schema{AllOf: []*Schema{
				{Type: "string"},
			}},
			want: false,
		},
		// OneOf
		{
			name: "OneOf same composition",
			a: &Schema{OneOf: []*Schema{
				{Type: "string"},
				{Type: "integer"},
			}},
			b: &Schema{OneOf: []*Schema{
				{Type: "string"},
				{Type: "integer"},
			}},
			want: true,
		},
		{
			name: "OneOf different composition",
			a: &Schema{OneOf: []*Schema{
				{Type: "string"},
			}},
			b: &Schema{OneOf: []*Schema{
				{Type: "integer"},
			}},
			want: false,
		},
		// AnyOf
		{
			name: "AnyOf same composition",
			a: &Schema{AnyOf: []*Schema{
				{Type: "string"},
				{Type: "null"},
			}},
			b: &Schema{AnyOf: []*Schema{
				{Type: "string"},
				{Type: "null"},
			}},
			want: true,
		},
		{
			name: "AnyOf different order",
			a: &Schema{AnyOf: []*Schema{
				{Type: "string"},
				{Type: "null"},
			}},
			b: &Schema{AnyOf: []*Schema{
				{Type: "null"},
				{Type: "string"},
			}},
			want: false,
		},
		// PrefixItems
		{
			name: "PrefixItems same composition",
			a: &Schema{PrefixItems: []*Schema{
				{Type: "string"},
				{Type: "integer"},
			}},
			b: &Schema{PrefixItems: []*Schema{
				{Type: "string"},
				{Type: "integer"},
			}},
			want: true,
		},
		{
			name: "PrefixItems different",
			a: &Schema{PrefixItems: []*Schema{
				{Type: "string"},
			}},
			b: &Schema{PrefixItems: []*Schema{
				{Type: "integer"},
			}},
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

// TestSchemaEquals_AdditionalProperties tests the polymorphic AdditionalProperties field.
func TestSchemaEquals_AdditionalProperties(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "both nil",
			a:    &Schema{AdditionalProperties: nil},
			b:    &Schema{AdditionalProperties: nil},
			want: true,
		},
		{
			name: "both bool true",
			a:    &Schema{AdditionalProperties: true},
			b:    &Schema{AdditionalProperties: true},
			want: true,
		},
		{
			name: "both bool false",
			a:    &Schema{AdditionalProperties: false},
			b:    &Schema{AdditionalProperties: false},
			want: true,
		},
		{
			name: "bool true vs false",
			a:    &Schema{AdditionalProperties: true},
			b:    &Schema{AdditionalProperties: false},
			want: false,
		},
		{
			name: "both same *Schema",
			a:    &Schema{AdditionalProperties: &Schema{Type: "string"}},
			b:    &Schema{AdditionalProperties: &Schema{Type: "string"}},
			want: true,
		},
		{
			name: "both different *Schema",
			a:    &Schema{AdditionalProperties: &Schema{Type: "string"}},
			b:    &Schema{AdditionalProperties: &Schema{Type: "integer"}},
			want: false,
		},
		{
			name: "bool vs *Schema - type mismatch",
			a:    &Schema{AdditionalProperties: true},
			b:    &Schema{AdditionalProperties: &Schema{Type: "string"}},
			want: false,
		},
		{
			name: "*Schema vs bool - type mismatch",
			a:    &Schema{AdditionalProperties: &Schema{Type: "string"}},
			b:    &Schema{AdditionalProperties: false},
			want: false,
		},
		{
			name: "nil vs bool",
			a:    &Schema{AdditionalProperties: nil},
			b:    &Schema{AdditionalProperties: true},
			want: false,
		},
		{
			name: "nil vs *Schema",
			a:    &Schema{AdditionalProperties: nil},
			b:    &Schema{AdditionalProperties: &Schema{Type: "string"}},
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

// TestSchemaEquals_Required tests the Required string slice field.
func TestSchemaEquals_Required(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "Required both nil",
			a:    &Schema{Required: nil},
			b:    &Schema{Required: nil},
			want: true,
		},
		{
			name: "Required nil vs empty",
			a:    &Schema{Required: nil},
			b:    &Schema{Required: []string{}},
			want: true,
		},
		{
			name: "Required same",
			a:    &Schema{Required: []string{"id", "name"}},
			b:    &Schema{Required: []string{"id", "name"}},
			want: true,
		},
		{
			name: "Required different order",
			a:    &Schema{Required: []string{"id", "name"}},
			b:    &Schema{Required: []string{"name", "id"}},
			want: false,
		},
		{
			name: "Required different values",
			a:    &Schema{Required: []string{"id", "name"}},
			b:    &Schema{Required: []string{"id", "email"}},
			want: false,
		},
		{
			name: "Required different length",
			a:    &Schema{Required: []string{"id", "name"}},
			b:    &Schema{Required: []string{"id"}},
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

// TestSchemaEquals_Enum tests the Enum any slice field.
func TestSchemaEquals_Enum(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "Enum both nil",
			a:    &Schema{Enum: nil},
			b:    &Schema{Enum: nil},
			want: true,
		},
		{
			name: "Enum nil vs empty",
			a:    &Schema{Enum: nil},
			b:    &Schema{Enum: []any{}},
			want: true,
		},
		{
			name: "Enum same strings",
			a:    &Schema{Enum: []any{"pending", "active", "inactive"}},
			b:    &Schema{Enum: []any{"pending", "active", "inactive"}},
			want: true,
		},
		{
			name: "Enum different order",
			a:    &Schema{Enum: []any{"pending", "active"}},
			b:    &Schema{Enum: []any{"active", "pending"}},
			want: false,
		},
		{
			name: "Enum mixed types",
			a:    &Schema{Enum: []any{"red", 1, true, nil}},
			b:    &Schema{Enum: []any{"red", 1, true, nil}},
			want: true,
		},
		{
			name: "Enum different values",
			a:    &Schema{Enum: []any{"a", "b"}},
			b:    &Schema{Enum: []any{"x", "y"}},
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

// TestSchemaEquals_Default tests the Default any field.
func TestSchemaEquals_Default(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "Default both nil",
			a:    &Schema{Default: nil},
			b:    &Schema{Default: nil},
			want: true,
		},
		{
			name: "Default same string",
			a:    &Schema{Default: "default value"},
			b:    &Schema{Default: "default value"},
			want: true,
		},
		{
			name: "Default different string",
			a:    &Schema{Default: "value1"},
			b:    &Schema{Default: "value2"},
			want: false,
		},
		{
			name: "Default same number",
			a:    &Schema{Default: float64(42)},
			b:    &Schema{Default: float64(42)},
			want: true,
		},
		{
			name: "Default same map",
			a:    &Schema{Default: map[string]any{"key": "value"}},
			b:    &Schema{Default: map[string]any{"key": "value"}},
			want: true,
		},
		{
			name: "Default nil vs value",
			a:    &Schema{Default: nil},
			b:    &Schema{Default: "value"},
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

// TestSchemaEquals_PatternProperties tests PatternProperties map comparison.
func TestSchemaEquals_PatternProperties(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "PatternProperties both nil",
			a:    &Schema{PatternProperties: nil},
			b:    &Schema{PatternProperties: nil},
			want: true,
		},
		{
			name: "PatternProperties nil vs empty",
			a:    &Schema{PatternProperties: nil},
			b:    &Schema{PatternProperties: map[string]*Schema{}},
			want: true,
		},
		{
			name: "PatternProperties same",
			a: &Schema{PatternProperties: map[string]*Schema{
				"^x-": {Type: "string"},
			}},
			b: &Schema{PatternProperties: map[string]*Schema{
				"^x-": {Type: "string"},
			}},
			want: true,
		},
		{
			name: "PatternProperties different pattern",
			a: &Schema{PatternProperties: map[string]*Schema{
				"^x-": {Type: "string"},
			}},
			b: &Schema{PatternProperties: map[string]*Schema{
				"^y-": {Type: "string"},
			}},
			want: false,
		},
		{
			name: "PatternProperties different schema",
			a: &Schema{PatternProperties: map[string]*Schema{
				"^x-": {Type: "string"},
			}},
			b: &Schema{PatternProperties: map[string]*Schema{
				"^x-": {Type: "integer"},
			}},
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

// TestSchemaEquals_Defs tests $defs map comparison.
func TestSchemaEquals_Defs(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "Defs both nil",
			a:    &Schema{Defs: nil},
			b:    &Schema{Defs: nil},
			want: true,
		},
		{
			name: "Defs nil vs empty",
			a:    &Schema{Defs: nil},
			b:    &Schema{Defs: map[string]*Schema{}},
			want: true,
		},
		{
			name: "Defs same",
			a: &Schema{Defs: map[string]*Schema{
				"Pet": {Type: "object"},
			}},
			b: &Schema{Defs: map[string]*Schema{
				"Pet": {Type: "object"},
			}},
			want: true,
		},
		{
			name: "Defs different definition name",
			a: &Schema{Defs: map[string]*Schema{
				"Pet": {Type: "object"},
			}},
			b: &Schema{Defs: map[string]*Schema{
				"User": {Type: "object"},
			}},
			want: false,
		},
		{
			name: "Defs different schema",
			a: &Schema{Defs: map[string]*Schema{
				"Pet": {Type: "object"},
			}},
			b: &Schema{Defs: map[string]*Schema{
				"Pet": {Type: "array"},
			}},
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

// TestSchemaEquals_Vocabulary tests $vocabulary map comparison.
func TestSchemaEquals_Vocabulary(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "Vocabulary both nil",
			a:    &Schema{Vocabulary: nil},
			b:    &Schema{Vocabulary: nil},
			want: true,
		},
		{
			name: "Vocabulary nil vs empty",
			a:    &Schema{Vocabulary: nil},
			b:    &Schema{Vocabulary: map[string]bool{}},
			want: true,
		},
		{
			name: "Vocabulary same",
			a: &Schema{Vocabulary: map[string]bool{
				"https://json-schema.org/draft/2020-12/vocab/core":       true,
				"https://json-schema.org/draft/2020-12/vocab/validation": true,
			}},
			b: &Schema{Vocabulary: map[string]bool{
				"https://json-schema.org/draft/2020-12/vocab/core":       true,
				"https://json-schema.org/draft/2020-12/vocab/validation": true,
			}},
			want: true,
		},
		{
			name: "Vocabulary different values",
			a: &Schema{Vocabulary: map[string]bool{
				"https://json-schema.org/draft/2020-12/vocab/core": true,
			}},
			b: &Schema{Vocabulary: map[string]bool{
				"https://json-schema.org/draft/2020-12/vocab/core": false,
			}},
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

// TestSchemaEquals_DependentRequired tests DependentRequired map comparison.
func TestSchemaEquals_DependentRequired(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "DependentRequired both nil",
			a:    &Schema{DependentRequired: nil},
			b:    &Schema{DependentRequired: nil},
			want: true,
		},
		{
			name: "DependentRequired same",
			a: &Schema{DependentRequired: map[string][]string{
				"credit_card": {"billing_address"},
			}},
			b: &Schema{DependentRequired: map[string][]string{
				"credit_card": {"billing_address"},
			}},
			want: true,
		},
		{
			name: "DependentRequired different",
			a: &Schema{DependentRequired: map[string][]string{
				"credit_card": {"billing_address"},
			}},
			b: &Schema{DependentRequired: map[string][]string{
				"credit_card": {"shipping_address"},
			}},
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

// TestSchemaEquals_DependentSchemas tests DependentSchemas map comparison.
func TestSchemaEquals_DependentSchemas(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "DependentSchemas both nil",
			a:    &Schema{DependentSchemas: nil},
			b:    &Schema{DependentSchemas: nil},
			want: true,
		},
		{
			name: "DependentSchemas same",
			a: &Schema{DependentSchemas: map[string]*Schema{
				"credit_card": {Required: []string{"billing_address"}},
			}},
			b: &Schema{DependentSchemas: map[string]*Schema{
				"credit_card": {Required: []string{"billing_address"}},
			}},
			want: true,
		},
		{
			name: "DependentSchemas different",
			a: &Schema{DependentSchemas: map[string]*Schema{
				"credit_card": {Required: []string{"billing_address"}},
			}},
			b: &Schema{DependentSchemas: map[string]*Schema{
				"credit_card": {Required: []string{"shipping_address"}},
			}},
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
