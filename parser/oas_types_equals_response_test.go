package parser

import (
	"testing"

	"github.com/erraggy/oastools/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// equalHeader tests
// =============================================================================

func TestEqualHeader(t *testing.T) {
	tests := []struct {
		name string
		a    *Header
		b    *Header
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
			b:    &Header{Description: "test"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Header{Description: "test"},
			b:    nil,
			want: false,
		},
		// Empty headers
		{
			name: "both empty",
			a:    &Header{},
			b:    &Header{},
			want: true,
		},
		// Boolean fields
		{
			name: "different Required",
			a:    &Header{Required: true},
			b:    &Header{Required: false},
			want: false,
		},
		{
			name: "different Deprecated",
			a:    &Header{Deprecated: true},
			b:    &Header{Deprecated: false},
			want: false,
		},
		{
			name: "different ExclusiveMaximum",
			a:    &Header{ExclusiveMaximum: true},
			b:    &Header{ExclusiveMaximum: false},
			want: false,
		},
		{
			name: "different ExclusiveMinimum",
			a:    &Header{ExclusiveMinimum: true},
			b:    &Header{ExclusiveMinimum: false},
			want: false,
		},
		{
			name: "different UniqueItems",
			a:    &Header{UniqueItems: true},
			b:    &Header{UniqueItems: false},
			want: false,
		},
		// String fields
		{
			name: "different Ref",
			a:    &Header{Ref: "#/components/headers/X-Rate-Limit"},
			b:    &Header{Ref: "#/components/headers/X-Request-ID"},
			want: false,
		},
		{
			name: "different Description",
			a:    &Header{Description: "Rate limit header"},
			b:    &Header{Description: "Request ID header"},
			want: false,
		},
		{
			name: "different Style",
			a:    &Header{Style: "simple"},
			b:    &Header{Style: "form"},
			want: false,
		},
		{
			name: "different Type (OAS 2.0)",
			a:    &Header{Type: "string"},
			b:    &Header{Type: "integer"},
			want: false,
		},
		{
			name: "different Format",
			a:    &Header{Format: "int32"},
			b:    &Header{Format: "int64"},
			want: false,
		},
		{
			name: "different CollectionFormat",
			a:    &Header{CollectionFormat: "csv"},
			b:    &Header{CollectionFormat: "ssv"},
			want: false,
		},
		{
			name: "different Pattern",
			a:    &Header{Pattern: "^[a-z]+$"},
			b:    &Header{Pattern: "^[0-9]+$"},
			want: false,
		},
		// Pointer fields - Explode
		{
			name: "both Explode nil",
			a:    &Header{Explode: nil},
			b:    &Header{Explode: nil},
			want: true,
		},
		{
			name: "Explode nil vs non-nil",
			a:    &Header{Explode: nil},
			b:    &Header{Explode: testutil.Ptr(true)},
			want: false,
		},
		{
			name: "same Explode true",
			a:    &Header{Explode: testutil.Ptr(true)},
			b:    &Header{Explode: testutil.Ptr(true)},
			want: true,
		},
		{
			name: "different Explode values",
			a:    &Header{Explode: testutil.Ptr(true)},
			b:    &Header{Explode: testutil.Ptr(false)},
			want: false,
		},
		// Pointer fields - numeric
		{
			name: "different Maximum",
			a:    &Header{Maximum: testutil.Ptr(100.0)},
			b:    &Header{Maximum: testutil.Ptr(200.0)},
			want: false,
		},
		{
			name: "Maximum nil vs non-nil",
			a:    &Header{Maximum: nil},
			b:    &Header{Maximum: testutil.Ptr(100.0)},
			want: false,
		},
		{
			name: "different Minimum",
			a:    &Header{Minimum: testutil.Ptr(0.0)},
			b:    &Header{Minimum: testutil.Ptr(1.0)},
			want: false,
		},
		{
			name: "different MultipleOf",
			a:    &Header{MultipleOf: testutil.Ptr(5.0)},
			b:    &Header{MultipleOf: testutil.Ptr(10.0)},
			want: false,
		},
		{
			name: "different MaxLength",
			a:    &Header{MaxLength: testutil.Ptr(100)},
			b:    &Header{MaxLength: testutil.Ptr(200)},
			want: false,
		},
		{
			name: "different MinLength",
			a:    &Header{MinLength: testutil.Ptr(1)},
			b:    &Header{MinLength: testutil.Ptr(5)},
			want: false,
		},
		{
			name: "different MaxItems",
			a:    &Header{MaxItems: testutil.Ptr(10)},
			b:    &Header{MaxItems: testutil.Ptr(20)},
			want: false,
		},
		{
			name: "different MinItems",
			a:    &Header{MinItems: testutil.Ptr(1)},
			b:    &Header{MinItems: testutil.Ptr(2)},
			want: false,
		},
		// Any fields
		{
			name: "same Example",
			a:    &Header{Example: "abc123"},
			b:    &Header{Example: "abc123"},
			want: true,
		},
		{
			name: "different Example",
			a:    &Header{Example: "abc123"},
			b:    &Header{Example: "xyz789"},
			want: false,
		},
		{
			name: "same Default",
			a:    &Header{Default: 10},
			b:    &Header{Default: 10},
			want: true,
		},
		{
			name: "different Default",
			a:    &Header{Default: 10},
			b:    &Header{Default: 20},
			want: false,
		},
		{
			name: "same Enum",
			a:    &Header{Enum: []any{"active", "inactive"}},
			b:    &Header{Enum: []any{"active", "inactive"}},
			want: true,
		},
		{
			name: "different Enum",
			a:    &Header{Enum: []any{"active", "inactive"}},
			b:    &Header{Enum: []any{"active", "deleted"}},
			want: false,
		},
		// Schema (OAS 3.0+)
		{
			name: "same Schema",
			a:    &Header{Schema: &Schema{Type: "string"}},
			b:    &Header{Schema: &Schema{Type: "string"}},
			want: true,
		},
		{
			name: "different Schema",
			a:    &Header{Schema: &Schema{Type: "string"}},
			b:    &Header{Schema: &Schema{Type: "integer"}},
			want: false,
		},
		{
			name: "Schema nil vs non-nil",
			a:    &Header{Schema: nil},
			b:    &Header{Schema: &Schema{Type: "string"}},
			want: false,
		},
		// Items (OAS 2.0)
		{
			name: "same Items",
			a:    &Header{Items: &Items{Type: "string"}},
			b:    &Header{Items: &Items{Type: "string"}},
			want: true,
		},
		{
			name: "different Items",
			a:    &Header{Items: &Items{Type: "string"}},
			b:    &Header{Items: &Items{Type: "integer"}},
			want: false,
		},
		{
			name: "Items nil vs non-nil",
			a:    &Header{Items: nil},
			b:    &Header{Items: &Items{Type: "string"}},
			want: false,
		},
		// Examples map (OAS 3.0+)
		{
			name: "same Examples",
			a: &Header{
				Examples: map[string]*Example{
					"default": {Summary: "Default example", Value: "abc123"},
				},
			},
			b: &Header{
				Examples: map[string]*Example{
					"default": {Summary: "Default example", Value: "abc123"},
				},
			},
			want: true,
		},
		{
			name: "different Examples",
			a: &Header{
				Examples: map[string]*Example{
					"default": {Summary: "Default example", Value: "abc123"},
				},
			},
			b: &Header{
				Examples: map[string]*Example{
					"default": {Summary: "Different example", Value: "xyz789"},
				},
			},
			want: false,
		},
		{
			name: "Examples nil vs empty",
			a:    &Header{Examples: nil},
			b:    &Header{Examples: map[string]*Example{}},
			want: true,
		},
		// Content map (OAS 3.0+)
		{
			name: "same Content",
			a: &Header{
				Content: map[string]*MediaType{
					"application/json": {Schema: &Schema{Type: "object"}},
				},
			},
			b: &Header{
				Content: map[string]*MediaType{
					"application/json": {Schema: &Schema{Type: "object"}},
				},
			},
			want: true,
		},
		{
			name: "different Content",
			a: &Header{
				Content: map[string]*MediaType{
					"application/json": {Schema: &Schema{Type: "object"}},
				},
			},
			b: &Header{
				Content: map[string]*MediaType{
					"application/xml": {Schema: &Schema{Type: "object"}},
				},
			},
			want: false,
		},
		// Extra (extensions)
		{
			name: "same Extra",
			a:    &Header{Extra: map[string]any{"x-custom": "value"}},
			b:    &Header{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &Header{Extra: map[string]any{"x-custom": "value1"}},
			b:    &Header{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		// Complete header comparison
		{
			name: "complete OAS 3.0 header",
			a: &Header{
				Description: "Rate limit remaining",
				Required:    true,
				Deprecated:  false,
				Style:       "simple",
				Schema:      &Schema{Type: "integer"},
			},
			b: &Header{
				Description: "Rate limit remaining",
				Required:    true,
				Deprecated:  false,
				Style:       "simple",
				Schema:      &Schema{Type: "integer"},
			},
			want: true,
		},
		{
			name: "complete OAS 2.0 header",
			a: &Header{
				Description: "Rate limit remaining",
				Type:        "integer",
				Format:      "int32",
				Minimum:     testutil.Ptr(0.0),
				Maximum:     testutil.Ptr(1000.0),
			},
			b: &Header{
				Description: "Rate limit remaining",
				Type:        "integer",
				Format:      "int32",
				Minimum:     testutil.Ptr(0.0),
				Maximum:     testutil.Ptr(1000.0),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalHeader(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalResponse tests
// =============================================================================

func TestEqualResponse(t *testing.T) {
	tests := []struct {
		name string
		a    *Response
		b    *Response
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
			b:    &Response{Description: "Success"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Response{Description: "Success"},
			b:    nil,
			want: false,
		},
		// Empty responses
		{
			name: "both empty",
			a:    &Response{},
			b:    &Response{},
			want: true,
		},
		// String fields
		{
			name: "different Ref",
			a:    &Response{Ref: "#/components/responses/NotFound"},
			b:    &Response{Ref: "#/components/responses/BadRequest"},
			want: false,
		},
		{
			name: "different Description",
			a:    &Response{Description: "Success"},
			b:    &Response{Description: "OK"},
			want: false,
		},
		{
			name: "same Description",
			a:    &Response{Description: "Success"},
			b:    &Response{Description: "Success"},
			want: true,
		},
		// Schema (OAS 2.0)
		{
			name: "same Schema",
			a:    &Response{Description: "OK", Schema: &Schema{Type: "object"}},
			b:    &Response{Description: "OK", Schema: &Schema{Type: "object"}},
			want: true,
		},
		{
			name: "different Schema",
			a:    &Response{Description: "OK", Schema: &Schema{Type: "object"}},
			b:    &Response{Description: "OK", Schema: &Schema{Type: "array"}},
			want: false,
		},
		{
			name: "Schema nil vs non-nil",
			a:    &Response{Description: "OK", Schema: nil},
			b:    &Response{Description: "OK", Schema: &Schema{Type: "object"}},
			want: false,
		},
		// Headers map
		{
			name: "same Headers",
			a: &Response{
				Description: "OK",
				Headers: map[string]*Header{
					"X-Rate-Limit": {Description: "Rate limit"},
				},
			},
			b: &Response{
				Description: "OK",
				Headers: map[string]*Header{
					"X-Rate-Limit": {Description: "Rate limit"},
				},
			},
			want: true,
		},
		{
			name: "different Headers",
			a: &Response{
				Description: "OK",
				Headers: map[string]*Header{
					"X-Rate-Limit": {Description: "Rate limit"},
				},
			},
			b: &Response{
				Description: "OK",
				Headers: map[string]*Header{
					"X-Request-ID": {Description: "Request ID"},
				},
			},
			want: false,
		},
		{
			name: "Headers nil vs empty",
			a:    &Response{Description: "OK", Headers: nil},
			b:    &Response{Description: "OK", Headers: map[string]*Header{}},
			want: true,
		},
		{
			name: "Headers different values same key",
			a: &Response{
				Description: "OK",
				Headers: map[string]*Header{
					"X-Rate-Limit": {Description: "Rate limit", Type: "integer"},
				},
			},
			b: &Response{
				Description: "OK",
				Headers: map[string]*Header{
					"X-Rate-Limit": {Description: "Rate limit", Type: "string"},
				},
			},
			want: false,
		},
		// Content map (OAS 3.0+)
		{
			name: "same Content",
			a: &Response{
				Description: "OK",
				Content: map[string]*MediaType{
					"application/json": {Schema: &Schema{Type: "object"}},
				},
			},
			b: &Response{
				Description: "OK",
				Content: map[string]*MediaType{
					"application/json": {Schema: &Schema{Type: "object"}},
				},
			},
			want: true,
		},
		{
			name: "different Content media types",
			a: &Response{
				Description: "OK",
				Content: map[string]*MediaType{
					"application/json": {Schema: &Schema{Type: "object"}},
				},
			},
			b: &Response{
				Description: "OK",
				Content: map[string]*MediaType{
					"application/xml": {Schema: &Schema{Type: "object"}},
				},
			},
			want: false,
		},
		{
			name: "different Content schemas",
			a: &Response{
				Description: "OK",
				Content: map[string]*MediaType{
					"application/json": {Schema: &Schema{Type: "object"}},
				},
			},
			b: &Response{
				Description: "OK",
				Content: map[string]*MediaType{
					"application/json": {Schema: &Schema{Type: "array"}},
				},
			},
			want: false,
		},
		{
			name: "Content nil vs empty",
			a:    &Response{Description: "OK", Content: nil},
			b:    &Response{Description: "OK", Content: map[string]*MediaType{}},
			want: true,
		},
		// Links map (OAS 3.0+)
		{
			name: "same Links",
			a: &Response{
				Description: "OK",
				Links: map[string]*Link{
					"GetUserById": {OperationID: "getUserById"},
				},
			},
			b: &Response{
				Description: "OK",
				Links: map[string]*Link{
					"GetUserById": {OperationID: "getUserById"},
				},
			},
			want: true,
		},
		{
			name: "different Links",
			a: &Response{
				Description: "OK",
				Links: map[string]*Link{
					"GetUserById": {OperationID: "getUserById"},
				},
			},
			b: &Response{
				Description: "OK",
				Links: map[string]*Link{
					"GetUserByName": {OperationID: "getUserByName"},
				},
			},
			want: false,
		},
		{
			name: "Links nil vs empty",
			a:    &Response{Description: "OK", Links: nil},
			b:    &Response{Description: "OK", Links: map[string]*Link{}},
			want: true,
		},
		// Examples map (OAS 2.0)
		{
			name: "same Examples",
			a: &Response{
				Description: "OK",
				Examples: map[string]any{
					"application/json": map[string]any{"id": 1},
				},
			},
			b: &Response{
				Description: "OK",
				Examples: map[string]any{
					"application/json": map[string]any{"id": 1},
				},
			},
			want: true,
		},
		{
			name: "different Examples",
			a: &Response{
				Description: "OK",
				Examples: map[string]any{
					"application/json": map[string]any{"id": 1},
				},
			},
			b: &Response{
				Description: "OK",
				Examples: map[string]any{
					"application/json": map[string]any{"id": 2},
				},
			},
			want: false,
		},
		{
			name: "Examples nil vs empty",
			a:    &Response{Description: "OK", Examples: nil},
			b:    &Response{Description: "OK", Examples: map[string]any{}},
			want: true,
		},
		// Extra (extensions)
		{
			name: "same Extra",
			a:    &Response{Description: "OK", Extra: map[string]any{"x-custom": "value"}},
			b:    &Response{Description: "OK", Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &Response{Description: "OK", Extra: map[string]any{"x-custom": "value1"}},
			b:    &Response{Description: "OK", Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		// Complete response comparison
		{
			name: "complete OAS 3.0 response",
			a: &Response{
				Description: "Successful response",
				Headers: map[string]*Header{
					"X-Rate-Limit-Remaining": {Schema: &Schema{Type: "integer"}},
				},
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"id":   {Type: "string"},
								"name": {Type: "string"},
							},
						},
					},
				},
				Links: map[string]*Link{
					"GetUserById": {OperationID: "getUserById"},
				},
			},
			b: &Response{
				Description: "Successful response",
				Headers: map[string]*Header{
					"X-Rate-Limit-Remaining": {Schema: &Schema{Type: "integer"}},
				},
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"id":   {Type: "string"},
								"name": {Type: "string"},
							},
						},
					},
				},
				Links: map[string]*Link{
					"GetUserById": {OperationID: "getUserById"},
				},
			},
			want: true,
		},
		{
			name: "complete OAS 2.0 response",
			a: &Response{
				Description: "Successful response",
				Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"id":   {Type: "string"},
						"name": {Type: "string"},
					},
				},
				Headers: map[string]*Header{
					"X-Rate-Limit": {Type: "integer"},
				},
				Examples: map[string]any{
					"application/json": map[string]any{
						"id":   "123",
						"name": "John Doe",
					},
				},
			},
			b: &Response{
				Description: "Successful response",
				Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"id":   {Type: "string"},
						"name": {Type: "string"},
					},
				},
				Headers: map[string]*Header{
					"X-Rate-Limit": {Type: "integer"},
				},
				Examples: map[string]any{
					"application/json": map[string]any{
						"id":   "123",
						"name": "John Doe",
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalResponse(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
