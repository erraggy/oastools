package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// equalParameterMap tests
// =============================================================================

func TestEqualParameterMap(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]*Parameter
		b    map[string]*Parameter
		want bool
	}{
		// Nil and empty handling
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "both empty",
			a:    map[string]*Parameter{},
			b:    map[string]*Parameter{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    map[string]*Parameter{},
			want: true,
		},
		{
			name: "empty vs nil",
			a:    map[string]*Parameter{},
			b:    nil,
			want: true,
		},
		// Same entries
		{
			name: "same single entry",
			a: map[string]*Parameter{
				"userId": {Name: "userId", In: "path", Required: true},
			},
			b: map[string]*Parameter{
				"userId": {Name: "userId", In: "path", Required: true},
			},
			want: true,
		},
		{
			name: "same multiple entries",
			a: map[string]*Parameter{
				"userId": {Name: "userId", In: "path", Required: true},
				"limit":  {Name: "limit", In: "query"},
			},
			b: map[string]*Parameter{
				"userId": {Name: "userId", In: "path", Required: true},
				"limit":  {Name: "limit", In: "query"},
			},
			want: true,
		},
		// Different entries
		{
			name: "different values same key",
			a: map[string]*Parameter{
				"userId": {Name: "userId", In: "path"},
			},
			b: map[string]*Parameter{
				"userId": {Name: "userId", In: "query"},
			},
			want: false,
		},
		{
			name: "different keys",
			a: map[string]*Parameter{
				"userId": {Name: "userId", In: "path"},
			},
			b: map[string]*Parameter{
				"accountId": {Name: "userId", In: "path"},
			},
			want: false,
		},
		{
			name: "a has extra key",
			a: map[string]*Parameter{
				"userId": {Name: "userId", In: "path"},
				"limit":  {Name: "limit", In: "query"},
			},
			b: map[string]*Parameter{
				"userId": {Name: "userId", In: "path"},
			},
			want: false,
		},
		{
			name: "b has extra key",
			a: map[string]*Parameter{
				"userId": {Name: "userId", In: "path"},
			},
			b: map[string]*Parameter{
				"userId": {Name: "userId", In: "path"},
				"limit":  {Name: "limit", In: "query"},
			},
			want: false,
		},
		// nil pointer in map
		{
			name: "same key with nil values",
			a: map[string]*Parameter{
				"userId": nil,
			},
			b: map[string]*Parameter{
				"userId": nil,
			},
			want: true,
		},
		{
			name: "nil vs non-nil value",
			a: map[string]*Parameter{
				"userId": nil,
			},
			b: map[string]*Parameter{
				"userId": {Name: "userId"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalParameterMap(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalRequestBodyMap tests
// =============================================================================

func TestEqualRequestBodyMap(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]*RequestBody
		b    map[string]*RequestBody
		want bool
	}{
		// Nil and empty handling
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "both empty",
			a:    map[string]*RequestBody{},
			b:    map[string]*RequestBody{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    map[string]*RequestBody{},
			want: true,
		},
		// Same entries
		{
			name: "same single entry",
			a: map[string]*RequestBody{
				"UserInput": {Description: "User data", Required: true},
			},
			b: map[string]*RequestBody{
				"UserInput": {Description: "User data", Required: true},
			},
			want: true,
		},
		{
			name: "same multiple entries",
			a: map[string]*RequestBody{
				"UserInput":    {Description: "User data", Required: true},
				"AccountInput": {Description: "Account data", Required: false},
			},
			b: map[string]*RequestBody{
				"UserInput":    {Description: "User data", Required: true},
				"AccountInput": {Description: "Account data", Required: false},
			},
			want: true,
		},
		// Different entries
		{
			name: "different values same key",
			a: map[string]*RequestBody{
				"UserInput": {Description: "User data", Required: true},
			},
			b: map[string]*RequestBody{
				"UserInput": {Description: "User data", Required: false},
			},
			want: false,
		},
		{
			name: "different keys",
			a: map[string]*RequestBody{
				"UserInput": {Description: "User data"},
			},
			b: map[string]*RequestBody{
				"AccountInput": {Description: "User data"},
			},
			want: false,
		},
		{
			name: "a has extra key",
			a: map[string]*RequestBody{
				"UserInput":    {Description: "User data"},
				"AccountInput": {Description: "Account data"},
			},
			b: map[string]*RequestBody{
				"UserInput": {Description: "User data"},
			},
			want: false,
		},
		{
			name: "b has extra key",
			a: map[string]*RequestBody{
				"UserInput": {Description: "User data"},
			},
			b: map[string]*RequestBody{
				"UserInput":    {Description: "User data"},
				"AccountInput": {Description: "Account data"},
			},
			want: false,
		},
		// RequestBody with Content map
		{
			name: "same key with Content",
			a: map[string]*RequestBody{
				"UserInput": {
					Description: "User data",
					Content: map[string]*MediaType{
						"application/json": {Schema: &Schema{Type: "object"}},
					},
				},
			},
			b: map[string]*RequestBody{
				"UserInput": {
					Description: "User data",
					Content: map[string]*MediaType{
						"application/json": {Schema: &Schema{Type: "object"}},
					},
				},
			},
			want: true,
		},
		{
			name: "same key with different Content",
			a: map[string]*RequestBody{
				"UserInput": {
					Description: "User data",
					Content: map[string]*MediaType{
						"application/json": {Schema: &Schema{Type: "object"}},
					},
				},
			},
			b: map[string]*RequestBody{
				"UserInput": {
					Description: "User data",
					Content: map[string]*MediaType{
						"application/xml": {Schema: &Schema{Type: "object"}},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalRequestBodyMap(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
