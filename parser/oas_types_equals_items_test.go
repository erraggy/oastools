package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// equalItems tests (OAS 2.0)
// =============================================================================

func TestEqualItems(t *testing.T) {
	tests := []struct {
		name string
		a    *Items
		b    *Items
		want bool
	}{
		// Nil handling
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b non-nil",
			a:    nil,
			b:    &Items{Type: "string"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Items{Type: "string"},
			b:    nil,
			want: false,
		},
		// Empty items
		{
			name: "both empty",
			a:    &Items{},
			b:    &Items{},
			want: true,
		},
		// Boolean fields
		{
			name: "same ExclusiveMaximum true",
			a:    &Items{ExclusiveMaximum: true},
			b:    &Items{ExclusiveMaximum: true},
			want: true,
		},
		{
			name: "different ExclusiveMaximum",
			a:    &Items{ExclusiveMaximum: true},
			b:    &Items{ExclusiveMaximum: false},
			want: false,
		},
		{
			name: "same ExclusiveMinimum true",
			a:    &Items{ExclusiveMinimum: true},
			b:    &Items{ExclusiveMinimum: true},
			want: true,
		},
		{
			name: "different ExclusiveMinimum",
			a:    &Items{ExclusiveMinimum: true},
			b:    &Items{ExclusiveMinimum: false},
			want: false,
		},
		{
			name: "same UniqueItems true",
			a:    &Items{UniqueItems: true},
			b:    &Items{UniqueItems: true},
			want: true,
		},
		{
			name: "different UniqueItems",
			a:    &Items{UniqueItems: true},
			b:    &Items{UniqueItems: false},
			want: false,
		},
		// String fields
		{
			name: "same Type",
			a:    &Items{Type: "string"},
			b:    &Items{Type: "string"},
			want: true,
		},
		{
			name: "different Type",
			a:    &Items{Type: "string"},
			b:    &Items{Type: "integer"},
			want: false,
		},
		{
			name: "same Format",
			a:    &Items{Format: "int32"},
			b:    &Items{Format: "int32"},
			want: true,
		},
		{
			name: "different Format",
			a:    &Items{Format: "int32"},
			b:    &Items{Format: "int64"},
			want: false,
		},
		{
			name: "same CollectionFormat",
			a:    &Items{CollectionFormat: "csv"},
			b:    &Items{CollectionFormat: "csv"},
			want: true,
		},
		{
			name: "different CollectionFormat",
			a:    &Items{CollectionFormat: "csv"},
			b:    &Items{CollectionFormat: "ssv"},
			want: false,
		},
		{
			name: "same Pattern",
			a:    &Items{Pattern: "^[a-z]+$"},
			b:    &Items{Pattern: "^[a-z]+$"},
			want: true,
		},
		{
			name: "different Pattern",
			a:    &Items{Pattern: "^[a-z]+$"},
			b:    &Items{Pattern: "^[0-9]+$"},
			want: false,
		},
		// Pointer fields - float64
		{
			name: "same Maximum",
			a:    &Items{Maximum: ptr(100.0)},
			b:    &Items{Maximum: ptr(100.0)},
			want: true,
		},
		{
			name: "different Maximum",
			a:    &Items{Maximum: ptr(100.0)},
			b:    &Items{Maximum: ptr(200.0)},
			want: false,
		},
		{
			name: "Maximum nil vs non-nil",
			a:    &Items{Maximum: nil},
			b:    &Items{Maximum: ptr(100.0)},
			want: false,
		},
		{
			name: "same Minimum",
			a:    &Items{Minimum: ptr(0.0)},
			b:    &Items{Minimum: ptr(0.0)},
			want: true,
		},
		{
			name: "different Minimum",
			a:    &Items{Minimum: ptr(0.0)},
			b:    &Items{Minimum: ptr(1.0)},
			want: false,
		},
		{
			name: "Minimum nil vs non-nil",
			a:    &Items{Minimum: nil},
			b:    &Items{Minimum: ptr(0.0)},
			want: false,
		},
		{
			name: "same MultipleOf",
			a:    &Items{MultipleOf: ptr(5.0)},
			b:    &Items{MultipleOf: ptr(5.0)},
			want: true,
		},
		{
			name: "different MultipleOf",
			a:    &Items{MultipleOf: ptr(5.0)},
			b:    &Items{MultipleOf: ptr(10.0)},
			want: false,
		},
		{
			name: "MultipleOf nil vs non-nil",
			a:    &Items{MultipleOf: nil},
			b:    &Items{MultipleOf: ptr(5.0)},
			want: false,
		},
		// Pointer fields - int
		{
			name: "same MaxLength",
			a:    &Items{MaxLength: intPtr(100)},
			b:    &Items{MaxLength: intPtr(100)},
			want: true,
		},
		{
			name: "different MaxLength",
			a:    &Items{MaxLength: intPtr(100)},
			b:    &Items{MaxLength: intPtr(200)},
			want: false,
		},
		{
			name: "MaxLength nil vs non-nil",
			a:    &Items{MaxLength: nil},
			b:    &Items{MaxLength: intPtr(100)},
			want: false,
		},
		{
			name: "same MinLength",
			a:    &Items{MinLength: intPtr(1)},
			b:    &Items{MinLength: intPtr(1)},
			want: true,
		},
		{
			name: "different MinLength",
			a:    &Items{MinLength: intPtr(1)},
			b:    &Items{MinLength: intPtr(5)},
			want: false,
		},
		{
			name: "MinLength nil vs non-nil",
			a:    &Items{MinLength: nil},
			b:    &Items{MinLength: intPtr(1)},
			want: false,
		},
		{
			name: "same MaxItems",
			a:    &Items{MaxItems: intPtr(10)},
			b:    &Items{MaxItems: intPtr(10)},
			want: true,
		},
		{
			name: "different MaxItems",
			a:    &Items{MaxItems: intPtr(10)},
			b:    &Items{MaxItems: intPtr(20)},
			want: false,
		},
		{
			name: "MaxItems nil vs non-nil",
			a:    &Items{MaxItems: nil},
			b:    &Items{MaxItems: intPtr(10)},
			want: false,
		},
		{
			name: "same MinItems",
			a:    &Items{MinItems: intPtr(1)},
			b:    &Items{MinItems: intPtr(1)},
			want: true,
		},
		{
			name: "different MinItems",
			a:    &Items{MinItems: intPtr(1)},
			b:    &Items{MinItems: intPtr(2)},
			want: false,
		},
		{
			name: "MinItems nil vs non-nil",
			a:    &Items{MinItems: nil},
			b:    &Items{MinItems: intPtr(1)},
			want: false,
		},
		// Any fields
		{
			name: "same Default",
			a:    &Items{Default: "default value"},
			b:    &Items{Default: "default value"},
			want: true,
		},
		{
			name: "different Default",
			a:    &Items{Default: "default1"},
			b:    &Items{Default: "default2"},
			want: false,
		},
		{
			name: "Default nil vs non-nil",
			a:    &Items{Default: nil},
			b:    &Items{Default: "default"},
			want: false,
		},
		{
			name: "same Enum",
			a:    &Items{Enum: []any{"active", "inactive"}},
			b:    &Items{Enum: []any{"active", "inactive"}},
			want: true,
		},
		{
			name: "different Enum",
			a:    &Items{Enum: []any{"active", "inactive"}},
			b:    &Items{Enum: []any{"active", "deleted"}},
			want: false,
		},
		{
			name: "Enum nil vs empty",
			a:    &Items{Enum: nil},
			b:    &Items{Enum: []any{}},
			want: true,
		},
		// Recursive Items field
		{
			name: "same nested Items",
			a:    &Items{Type: "array", Items: &Items{Type: "string"}},
			b:    &Items{Type: "array", Items: &Items{Type: "string"}},
			want: true,
		},
		{
			name: "different nested Items",
			a:    &Items{Type: "array", Items: &Items{Type: "string"}},
			b:    &Items{Type: "array", Items: &Items{Type: "integer"}},
			want: false,
		},
		{
			name: "nested Items nil vs non-nil",
			a:    &Items{Type: "array", Items: nil},
			b:    &Items{Type: "array", Items: &Items{Type: "string"}},
			want: false,
		},
		{
			name: "deeply nested Items equal",
			a: &Items{
				Type: "array",
				Items: &Items{
					Type: "array",
					Items: &Items{
						Type: "string",
					},
				},
			},
			b: &Items{
				Type: "array",
				Items: &Items{
					Type: "array",
					Items: &Items{
						Type: "string",
					},
				},
			},
			want: true,
		},
		{
			name: "deeply nested Items different",
			a: &Items{
				Type: "array",
				Items: &Items{
					Type: "array",
					Items: &Items{
						Type: "string",
					},
				},
			},
			b: &Items{
				Type: "array",
				Items: &Items{
					Type: "array",
					Items: &Items{
						Type: "integer",
					},
				},
			},
			want: false,
		},
		// Extra field
		{
			name: "same Extra",
			a:    &Items{Extra: map[string]any{"x-custom": "value"}},
			b:    &Items{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &Items{Extra: map[string]any{"x-custom": "value1"}},
			b:    &Items{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		{
			name: "Extra nil vs empty",
			a:    &Items{Extra: nil},
			b:    &Items{Extra: map[string]any{}},
			want: true,
		},
		// Complete Items (OAS 2.0 array item schema)
		{
			name: "complete string Items equal",
			a: &Items{
				Type:      "string",
				Format:    "email",
				Pattern:   "^[a-z]+@example.com$",
				MaxLength: intPtr(100),
				MinLength: intPtr(5),
				Enum:      []any{"user@example.com", "admin@example.com"},
				Default:   "user@example.com",
			},
			b: &Items{
				Type:      "string",
				Format:    "email",
				Pattern:   "^[a-z]+@example.com$",
				MaxLength: intPtr(100),
				MinLength: intPtr(5),
				Enum:      []any{"user@example.com", "admin@example.com"},
				Default:   "user@example.com",
			},
			want: true,
		},
		{
			name: "complete integer Items equal",
			a: &Items{
				Type:             "integer",
				Format:           "int32",
				Minimum:          ptr(0.0),
				Maximum:          ptr(100.0),
				ExclusiveMinimum: false,
				ExclusiveMaximum: true,
				MultipleOf:       ptr(5.0),
			},
			b: &Items{
				Type:             "integer",
				Format:           "int32",
				Minimum:          ptr(0.0),
				Maximum:          ptr(100.0),
				ExclusiveMinimum: false,
				ExclusiveMaximum: true,
				MultipleOf:       ptr(5.0),
			},
			want: true,
		},
		{
			name: "complete array Items with nested Items equal",
			a: &Items{
				Type:        "array",
				MinItems:    intPtr(1),
				MaxItems:    intPtr(100),
				UniqueItems: true,
				Items: &Items{
					Type:   "string",
					Format: "uuid",
				},
				CollectionFormat: "csv",
			},
			b: &Items{
				Type:        "array",
				MinItems:    intPtr(1),
				MaxItems:    intPtr(100),
				UniqueItems: true,
				Items: &Items{
					Type:   "string",
					Format: "uuid",
				},
				CollectionFormat: "csv",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalItems(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
