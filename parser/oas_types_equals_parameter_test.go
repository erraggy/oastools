package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// equalParameter tests
// =============================================================================

func TestEqualParameter(t *testing.T) {
	tests := []struct {
		name string
		a    *Parameter
		b    *Parameter
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
			b:    &Parameter{Name: "test"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Parameter{Name: "test"},
			b:    nil,
			want: false,
		},
		// Empty parameters
		{
			name: "both empty",
			a:    &Parameter{},
			b:    &Parameter{},
			want: true,
		},
		// Different In values
		{
			name: "same query parameter",
			a:    &Parameter{Name: "id", In: "query"},
			b:    &Parameter{Name: "id", In: "query"},
			want: true,
		},
		{
			name: "same path parameter",
			a:    &Parameter{Name: "id", In: "path", Required: true},
			b:    &Parameter{Name: "id", In: "path", Required: true},
			want: true,
		},
		{
			name: "same header parameter",
			a:    &Parameter{Name: "X-Request-ID", In: "header"},
			b:    &Parameter{Name: "X-Request-ID", In: "header"},
			want: true,
		},
		{
			name: "same cookie parameter",
			a:    &Parameter{Name: "session", In: "cookie"},
			b:    &Parameter{Name: "session", In: "cookie"},
			want: true,
		},
		{
			name: "different In values",
			a:    &Parameter{Name: "id", In: "query"},
			b:    &Parameter{Name: "id", In: "path"},
			want: false,
		},
		// Boolean fields
		{
			name: "different Required",
			a:    &Parameter{Name: "id", Required: true},
			b:    &Parameter{Name: "id", Required: false},
			want: false,
		},
		{
			name: "different Deprecated",
			a:    &Parameter{Name: "id", Deprecated: true},
			b:    &Parameter{Name: "id", Deprecated: false},
			want: false,
		},
		{
			name: "different AllowReserved",
			a:    &Parameter{Name: "id", AllowReserved: true},
			b:    &Parameter{Name: "id", AllowReserved: false},
			want: false,
		},
		{
			name: "different AllowEmptyValue",
			a:    &Parameter{Name: "id", AllowEmptyValue: true},
			b:    &Parameter{Name: "id", AllowEmptyValue: false},
			want: false,
		},
		{
			name: "different ExclusiveMaximum",
			a:    &Parameter{Name: "id", ExclusiveMaximum: true},
			b:    &Parameter{Name: "id", ExclusiveMaximum: false},
			want: false,
		},
		{
			name: "different ExclusiveMinimum",
			a:    &Parameter{Name: "id", ExclusiveMinimum: true},
			b:    &Parameter{Name: "id", ExclusiveMinimum: false},
			want: false,
		},
		{
			name: "different UniqueItems",
			a:    &Parameter{Name: "id", UniqueItems: true},
			b:    &Parameter{Name: "id", UniqueItems: false},
			want: false,
		},
		// String fields
		{
			name: "different Ref",
			a:    &Parameter{Ref: "#/components/parameters/Id"},
			b:    &Parameter{Ref: "#/components/parameters/Name"},
			want: false,
		},
		{
			name: "different Name",
			a:    &Parameter{Name: "id"},
			b:    &Parameter{Name: "name"},
			want: false,
		},
		{
			name: "different Description",
			a:    &Parameter{Name: "id", Description: "The ID"},
			b:    &Parameter{Name: "id", Description: "An identifier"},
			want: false,
		},
		{
			name: "different Style",
			a:    &Parameter{Name: "id", Style: "form"},
			b:    &Parameter{Name: "id", Style: "simple"},
			want: false,
		},
		{
			name: "different Type (OAS 2.0)",
			a:    &Parameter{Name: "id", Type: "string"},
			b:    &Parameter{Name: "id", Type: "integer"},
			want: false,
		},
		{
			name: "different Format",
			a:    &Parameter{Name: "id", Format: "int32"},
			b:    &Parameter{Name: "id", Format: "int64"},
			want: false,
		},
		{
			name: "different CollectionFormat",
			a:    &Parameter{Name: "ids", CollectionFormat: "csv"},
			b:    &Parameter{Name: "ids", CollectionFormat: "ssv"},
			want: false,
		},
		{
			name: "different Pattern",
			a:    &Parameter{Name: "id", Pattern: "^[a-z]+$"},
			b:    &Parameter{Name: "id", Pattern: "^[0-9]+$"},
			want: false,
		},
		// Pointer fields - Explode
		{
			name: "both Explode nil",
			a:    &Parameter{Name: "id", Explode: nil},
			b:    &Parameter{Name: "id", Explode: nil},
			want: true,
		},
		{
			name: "Explode nil vs non-nil",
			a:    &Parameter{Name: "id", Explode: nil},
			b:    &Parameter{Name: "id", Explode: boolPtr(true)},
			want: false,
		},
		{
			name: "same Explode true",
			a:    &Parameter{Name: "id", Explode: boolPtr(true)},
			b:    &Parameter{Name: "id", Explode: boolPtr(true)},
			want: true,
		},
		{
			name: "different Explode values",
			a:    &Parameter{Name: "id", Explode: boolPtr(true)},
			b:    &Parameter{Name: "id", Explode: boolPtr(false)},
			want: false,
		},
		// Pointer fields - numeric
		{
			name: "different Maximum",
			a:    &Parameter{Name: "id", Maximum: ptr(100.0)},
			b:    &Parameter{Name: "id", Maximum: ptr(200.0)},
			want: false,
		},
		{
			name: "Maximum nil vs non-nil",
			a:    &Parameter{Name: "id", Maximum: nil},
			b:    &Parameter{Name: "id", Maximum: ptr(100.0)},
			want: false,
		},
		{
			name: "different Minimum",
			a:    &Parameter{Name: "id", Minimum: ptr(0.0)},
			b:    &Parameter{Name: "id", Minimum: ptr(1.0)},
			want: false,
		},
		{
			name: "different MultipleOf",
			a:    &Parameter{Name: "id", MultipleOf: ptr(5.0)},
			b:    &Parameter{Name: "id", MultipleOf: ptr(10.0)},
			want: false,
		},
		{
			name: "different MaxLength",
			a:    &Parameter{Name: "id", MaxLength: intPtr(100)},
			b:    &Parameter{Name: "id", MaxLength: intPtr(200)},
			want: false,
		},
		{
			name: "different MinLength",
			a:    &Parameter{Name: "id", MinLength: intPtr(1)},
			b:    &Parameter{Name: "id", MinLength: intPtr(5)},
			want: false,
		},
		{
			name: "different MaxItems",
			a:    &Parameter{Name: "ids", MaxItems: intPtr(10)},
			b:    &Parameter{Name: "ids", MaxItems: intPtr(20)},
			want: false,
		},
		{
			name: "different MinItems",
			a:    &Parameter{Name: "ids", MinItems: intPtr(1)},
			b:    &Parameter{Name: "ids", MinItems: intPtr(2)},
			want: false,
		},
		// Any fields
		{
			name: "same Example",
			a:    &Parameter{Name: "id", Example: "abc123"},
			b:    &Parameter{Name: "id", Example: "abc123"},
			want: true,
		},
		{
			name: "different Example",
			a:    &Parameter{Name: "id", Example: "abc123"},
			b:    &Parameter{Name: "id", Example: "xyz789"},
			want: false,
		},
		{
			name: "same Default",
			a:    &Parameter{Name: "id", Default: 10},
			b:    &Parameter{Name: "id", Default: 10},
			want: true,
		},
		{
			name: "different Default",
			a:    &Parameter{Name: "id", Default: 10},
			b:    &Parameter{Name: "id", Default: 20},
			want: false,
		},
		{
			name: "same Enum",
			a:    &Parameter{Name: "status", Enum: []any{"active", "inactive"}},
			b:    &Parameter{Name: "status", Enum: []any{"active", "inactive"}},
			want: true,
		},
		{
			name: "different Enum",
			a:    &Parameter{Name: "status", Enum: []any{"active", "inactive"}},
			b:    &Parameter{Name: "status", Enum: []any{"active", "deleted"}},
			want: false,
		},
		// Schema
		{
			name: "same Schema",
			a:    &Parameter{Name: "id", Schema: &Schema{Type: "string"}},
			b:    &Parameter{Name: "id", Schema: &Schema{Type: "string"}},
			want: true,
		},
		{
			name: "different Schema",
			a:    &Parameter{Name: "id", Schema: &Schema{Type: "string"}},
			b:    &Parameter{Name: "id", Schema: &Schema{Type: "integer"}},
			want: false,
		},
		{
			name: "Schema nil vs non-nil",
			a:    &Parameter{Name: "id", Schema: nil},
			b:    &Parameter{Name: "id", Schema: &Schema{Type: "string"}},
			want: false,
		},
		// Items (OAS 2.0)
		{
			name: "same Items",
			a:    &Parameter{Name: "ids", Items: &Items{Type: "string"}},
			b:    &Parameter{Name: "ids", Items: &Items{Type: "string"}},
			want: true,
		},
		{
			name: "different Items",
			a:    &Parameter{Name: "ids", Items: &Items{Type: "string"}},
			b:    &Parameter{Name: "ids", Items: &Items{Type: "integer"}},
			want: false,
		},
		// Examples map
		{
			name: "same Examples",
			a: &Parameter{
				Name: "id",
				Examples: map[string]*Example{
					"default": {Summary: "Default example", Value: "abc123"},
				},
			},
			b: &Parameter{
				Name: "id",
				Examples: map[string]*Example{
					"default": {Summary: "Default example", Value: "abc123"},
				},
			},
			want: true,
		},
		{
			name: "different Examples",
			a: &Parameter{
				Name: "id",
				Examples: map[string]*Example{
					"default": {Summary: "Default example", Value: "abc123"},
				},
			},
			b: &Parameter{
				Name: "id",
				Examples: map[string]*Example{
					"default": {Summary: "Different example", Value: "xyz789"},
				},
			},
			want: false,
		},
		{
			name: "Examples nil vs empty",
			a:    &Parameter{Name: "id", Examples: nil},
			b:    &Parameter{Name: "id", Examples: map[string]*Example{}},
			want: true,
		},
		// Content map
		{
			name: "same Content",
			a: &Parameter{
				Name: "body",
				Content: map[string]*MediaType{
					"application/json": {Schema: &Schema{Type: "object"}},
				},
			},
			b: &Parameter{
				Name: "body",
				Content: map[string]*MediaType{
					"application/json": {Schema: &Schema{Type: "object"}},
				},
			},
			want: true,
		},
		{
			name: "different Content",
			a: &Parameter{
				Name: "body",
				Content: map[string]*MediaType{
					"application/json": {Schema: &Schema{Type: "object"}},
				},
			},
			b: &Parameter{
				Name: "body",
				Content: map[string]*MediaType{
					"application/xml": {Schema: &Schema{Type: "object"}},
				},
			},
			want: false,
		},
		// Extra (extensions)
		{
			name: "same Extra",
			a:    &Parameter{Name: "id", Extra: map[string]any{"x-custom": "value"}},
			b:    &Parameter{Name: "id", Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &Parameter{Name: "id", Extra: map[string]any{"x-custom": "value1"}},
			b:    &Parameter{Name: "id", Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		// Full parameter comparison
		{
			name: "complete OAS 3.0 query parameter",
			a: &Parameter{
				Name:        "filter",
				In:          "query",
				Description: "Filter results",
				Required:    false,
				Deprecated:  false,
				Style:       "form",
				Explode:     boolPtr(true),
				Schema:      &Schema{Type: "string"},
			},
			b: &Parameter{
				Name:        "filter",
				In:          "query",
				Description: "Filter results",
				Required:    false,
				Deprecated:  false,
				Style:       "form",
				Explode:     boolPtr(true),
				Schema:      &Schema{Type: "string"},
			},
			want: true,
		},
		{
			name: "complete OAS 2.0 parameter",
			a: &Parameter{
				Name:        "limit",
				In:          "query",
				Description: "Maximum number of results",
				Required:    false,
				Type:        "integer",
				Format:      "int32",
				Minimum:     ptr(1.0),
				Maximum:     ptr(100.0),
				Default:     10,
			},
			b: &Parameter{
				Name:        "limit",
				In:          "query",
				Description: "Maximum number of results",
				Required:    false,
				Type:        "integer",
				Format:      "int32",
				Minimum:     ptr(1.0),
				Maximum:     ptr(100.0),
				Default:     10,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalParameter(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
