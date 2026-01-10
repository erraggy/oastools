package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// equalServerVariable tests
// =============================================================================

func TestEqualServerVariable(t *testing.T) {
	tests := []struct {
		name string
		a    ServerVariable
		b    ServerVariable
		want bool
	}{
		// Zero values
		{
			name: "both zero values",
			a:    ServerVariable{},
			b:    ServerVariable{},
			want: true,
		},
		// Default field
		{
			name: "same Default",
			a:    ServerVariable{Default: "production"},
			b:    ServerVariable{Default: "production"},
			want: true,
		},
		{
			name: "different Default",
			a:    ServerVariable{Default: "production"},
			b:    ServerVariable{Default: "staging"},
			want: false,
		},
		// Description field
		{
			name: "same Description",
			a:    ServerVariable{Description: "Server environment"},
			b:    ServerVariable{Description: "Server environment"},
			want: true,
		},
		{
			name: "different Description",
			a:    ServerVariable{Description: "Server environment"},
			b:    ServerVariable{Description: "Deployment stage"},
			want: false,
		},
		// Enum field
		{
			name: "same Enum",
			a:    ServerVariable{Enum: []string{"production", "staging", "development"}},
			b:    ServerVariable{Enum: []string{"production", "staging", "development"}},
			want: true,
		},
		{
			name: "different Enum values",
			a:    ServerVariable{Enum: []string{"production", "staging"}},
			b:    ServerVariable{Enum: []string{"production", "development"}},
			want: false,
		},
		{
			name: "different Enum lengths",
			a:    ServerVariable{Enum: []string{"production", "staging"}},
			b:    ServerVariable{Enum: []string{"production"}},
			want: false,
		},
		{
			name: "Enum nil vs empty",
			a:    ServerVariable{Enum: nil},
			b:    ServerVariable{Enum: []string{}},
			want: true,
		},
		{
			name: "Enum different order",
			a:    ServerVariable{Enum: []string{"a", "b", "c"}},
			b:    ServerVariable{Enum: []string{"c", "b", "a"}},
			want: false,
		},
		// Extra field (extensions)
		{
			name: "same Extra",
			a:    ServerVariable{Extra: map[string]any{"x-custom": "value"}},
			b:    ServerVariable{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    ServerVariable{Extra: map[string]any{"x-custom": "value1"}},
			b:    ServerVariable{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		{
			name: "Extra nil vs empty",
			a:    ServerVariable{Extra: nil},
			b:    ServerVariable{Extra: map[string]any{}},
			want: true,
		},
		// Complete server variable
		{
			name: "complete server variable equal",
			a: ServerVariable{
				Default:     "production",
				Description: "Server environment",
				Enum:        []string{"production", "staging", "development"},
				Extra:       map[string]any{"x-priority": 1},
			},
			b: ServerVariable{
				Default:     "production",
				Description: "Server environment",
				Enum:        []string{"production", "staging", "development"},
				Extra:       map[string]any{"x-priority": 1},
			},
			want: true,
		},
		{
			name: "complete server variable different",
			a: ServerVariable{
				Default:     "production",
				Description: "Server environment",
				Enum:        []string{"production", "staging"},
			},
			b: ServerVariable{
				Default:     "staging",
				Description: "Server environment",
				Enum:        []string{"production", "staging"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalServerVariable(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalServerVariableMap tests
// =============================================================================

func TestEqualServerVariableMap(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]ServerVariable
		b    map[string]ServerVariable
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
			a:    map[string]ServerVariable{},
			b:    map[string]ServerVariable{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    map[string]ServerVariable{},
			want: true,
		},
		{
			name: "empty vs nil",
			a:    map[string]ServerVariable{},
			b:    nil,
			want: true,
		},
		// Same entries
		{
			name: "same single entry",
			a: map[string]ServerVariable{
				"environment": {Default: "production", Enum: []string{"production", "staging"}},
			},
			b: map[string]ServerVariable{
				"environment": {Default: "production", Enum: []string{"production", "staging"}},
			},
			want: true,
		},
		{
			name: "same multiple entries",
			a: map[string]ServerVariable{
				"environment": {Default: "production"},
				"version":     {Default: "v1"},
			},
			b: map[string]ServerVariable{
				"environment": {Default: "production"},
				"version":     {Default: "v1"},
			},
			want: true,
		},
		// Different entries
		{
			name: "different values same key",
			a: map[string]ServerVariable{
				"environment": {Default: "production"},
			},
			b: map[string]ServerVariable{
				"environment": {Default: "staging"},
			},
			want: false,
		},
		{
			name: "different keys",
			a: map[string]ServerVariable{
				"environment": {Default: "production"},
			},
			b: map[string]ServerVariable{
				"env": {Default: "production"},
			},
			want: false,
		},
		{
			name: "a has extra key",
			a: map[string]ServerVariable{
				"environment": {Default: "production"},
				"version":     {Default: "v1"},
			},
			b: map[string]ServerVariable{
				"environment": {Default: "production"},
			},
			want: false,
		},
		{
			name: "b has extra key",
			a: map[string]ServerVariable{
				"environment": {Default: "production"},
			},
			b: map[string]ServerVariable{
				"environment": {Default: "production"},
				"version":     {Default: "v1"},
			},
			want: false,
		},
		// Key exists but variable differs in nested fields
		{
			name: "same key different enum",
			a: map[string]ServerVariable{
				"environment": {Default: "production", Enum: []string{"production"}},
			},
			b: map[string]ServerVariable{
				"environment": {Default: "production", Enum: []string{"production", "staging"}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalServerVariableMap(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalTag tests
// =============================================================================

func TestEqualTag(t *testing.T) {
	tests := []struct {
		name string
		a    *Tag
		b    *Tag
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
			b:    &Tag{Name: "users"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Tag{Name: "users"},
			b:    nil,
			want: false,
		},
		// Empty tags
		{
			name: "both empty",
			a:    &Tag{},
			b:    &Tag{},
			want: true,
		},
		// Name field
		{
			name: "same Name",
			a:    &Tag{Name: "users"},
			b:    &Tag{Name: "users"},
			want: true,
		},
		{
			name: "different Name",
			a:    &Tag{Name: "users"},
			b:    &Tag{Name: "accounts"},
			want: false,
		},
		// Description field
		{
			name: "same Description",
			a:    &Tag{Name: "users", Description: "User management"},
			b:    &Tag{Name: "users", Description: "User management"},
			want: true,
		},
		{
			name: "different Description",
			a:    &Tag{Name: "users", Description: "User management"},
			b:    &Tag{Name: "users", Description: "User operations"},
			want: false,
		},
		// ExternalDocs field
		{
			name: "same ExternalDocs",
			a:    &Tag{Name: "users", ExternalDocs: &ExternalDocs{URL: "https://example.com/docs"}},
			b:    &Tag{Name: "users", ExternalDocs: &ExternalDocs{URL: "https://example.com/docs"}},
			want: true,
		},
		{
			name: "different ExternalDocs",
			a:    &Tag{Name: "users", ExternalDocs: &ExternalDocs{URL: "https://example.com/docs"}},
			b:    &Tag{Name: "users", ExternalDocs: &ExternalDocs{URL: "https://other.com/docs"}},
			want: false,
		},
		{
			name: "ExternalDocs nil vs non-nil",
			a:    &Tag{Name: "users", ExternalDocs: nil},
			b:    &Tag{Name: "users", ExternalDocs: &ExternalDocs{URL: "https://example.com"}},
			want: false,
		},
		{
			name: "ExternalDocs non-nil vs nil",
			a:    &Tag{Name: "users", ExternalDocs: &ExternalDocs{URL: "https://example.com"}},
			b:    &Tag{Name: "users", ExternalDocs: nil},
			want: false,
		},
		// Extra field
		{
			name: "same Extra",
			a:    &Tag{Name: "users", Extra: map[string]any{"x-order": 1}},
			b:    &Tag{Name: "users", Extra: map[string]any{"x-order": 1}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &Tag{Name: "users", Extra: map[string]any{"x-order": 1}},
			b:    &Tag{Name: "users", Extra: map[string]any{"x-order": 2}},
			want: false,
		},
		{
			name: "Extra nil vs empty",
			a:    &Tag{Name: "users", Extra: nil},
			b:    &Tag{Name: "users", Extra: map[string]any{}},
			want: true,
		},
		// Complete tag comparison
		{
			name: "complete tag equal",
			a: &Tag{
				Name:         "users",
				Description:  "User management operations",
				ExternalDocs: &ExternalDocs{URL: "https://docs.example.com/users", Description: "User docs"},
				Extra:        map[string]any{"x-display-name": "Users"},
			},
			b: &Tag{
				Name:         "users",
				Description:  "User management operations",
				ExternalDocs: &ExternalDocs{URL: "https://docs.example.com/users", Description: "User docs"},
				Extra:        map[string]any{"x-display-name": "Users"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalTag(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalTagSlice tests
// =============================================================================

func TestEqualTagSlice(t *testing.T) {
	tests := []struct {
		name string
		a    []*Tag
		b    []*Tag
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
			a:    []*Tag{},
			b:    []*Tag{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    []*Tag{},
			want: true,
		},
		// Same elements
		{
			name: "same single element",
			a:    []*Tag{{Name: "users"}},
			b:    []*Tag{{Name: "users"}},
			want: true,
		},
		{
			name: "same multiple elements",
			a:    []*Tag{{Name: "users"}, {Name: "accounts"}},
			b:    []*Tag{{Name: "users"}, {Name: "accounts"}},
			want: true,
		},
		// Different elements
		{
			name: "different elements",
			a:    []*Tag{{Name: "users"}},
			b:    []*Tag{{Name: "accounts"}},
			want: false,
		},
		{
			name: "different lengths",
			a:    []*Tag{{Name: "users"}, {Name: "accounts"}},
			b:    []*Tag{{Name: "users"}},
			want: false,
		},
		{
			name: "different order",
			a:    []*Tag{{Name: "users"}, {Name: "accounts"}},
			b:    []*Tag{{Name: "accounts"}, {Name: "users"}},
			want: false,
		},
		// Nil elements in slice
		{
			name: "both have nil element at same position",
			a:    []*Tag{nil, {Name: "users"}},
			b:    []*Tag{nil, {Name: "users"}},
			want: true,
		},
		{
			name: "nil vs non-nil element",
			a:    []*Tag{nil},
			b:    []*Tag{{Name: "users"}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalTagSlice(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalServer tests
// =============================================================================

func TestEqualServer(t *testing.T) {
	tests := []struct {
		name string
		a    *Server
		b    *Server
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
			b:    &Server{URL: "https://api.example.com"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Server{URL: "https://api.example.com"},
			b:    nil,
			want: false,
		},
		// Empty servers
		{
			name: "both empty",
			a:    &Server{},
			b:    &Server{},
			want: true,
		},
		// URL field
		{
			name: "same URL",
			a:    &Server{URL: "https://api.example.com"},
			b:    &Server{URL: "https://api.example.com"},
			want: true,
		},
		{
			name: "different URL",
			a:    &Server{URL: "https://api.example.com"},
			b:    &Server{URL: "https://api.other.com"},
			want: false,
		},
		// Description field
		{
			name: "same Description",
			a:    &Server{URL: "https://api.example.com", Description: "Production server"},
			b:    &Server{URL: "https://api.example.com", Description: "Production server"},
			want: true,
		},
		{
			name: "different Description",
			a:    &Server{URL: "https://api.example.com", Description: "Production server"},
			b:    &Server{URL: "https://api.example.com", Description: "Staging server"},
			want: false,
		},
		// Variables field
		{
			name: "same Variables",
			a: &Server{
				URL: "https://{environment}.example.com",
				Variables: map[string]ServerVariable{
					"environment": {Default: "production"},
				},
			},
			b: &Server{
				URL: "https://{environment}.example.com",
				Variables: map[string]ServerVariable{
					"environment": {Default: "production"},
				},
			},
			want: true,
		},
		{
			name: "different Variables",
			a: &Server{
				URL: "https://{environment}.example.com",
				Variables: map[string]ServerVariable{
					"environment": {Default: "production"},
				},
			},
			b: &Server{
				URL: "https://{environment}.example.com",
				Variables: map[string]ServerVariable{
					"environment": {Default: "staging"},
				},
			},
			want: false,
		},
		{
			name: "Variables nil vs empty",
			a:    &Server{URL: "https://api.example.com", Variables: nil},
			b:    &Server{URL: "https://api.example.com", Variables: map[string]ServerVariable{}},
			want: true,
		},
		// Extra field
		{
			name: "same Extra",
			a:    &Server{URL: "https://api.example.com", Extra: map[string]any{"x-internal": true}},
			b:    &Server{URL: "https://api.example.com", Extra: map[string]any{"x-internal": true}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &Server{URL: "https://api.example.com", Extra: map[string]any{"x-internal": true}},
			b:    &Server{URL: "https://api.example.com", Extra: map[string]any{"x-internal": false}},
			want: false,
		},
		// Complete server comparison
		{
			name: "complete server equal",
			a: &Server{
				URL:         "https://{environment}.example.com/api/{version}",
				Description: "Multi-environment API server",
				Variables: map[string]ServerVariable{
					"environment": {Default: "production", Enum: []string{"production", "staging"}},
					"version":     {Default: "v1", Enum: []string{"v1", "v2"}},
				},
				Extra: map[string]any{"x-tier": "premium"},
			},
			b: &Server{
				URL:         "https://{environment}.example.com/api/{version}",
				Description: "Multi-environment API server",
				Variables: map[string]ServerVariable{
					"environment": {Default: "production", Enum: []string{"production", "staging"}},
					"version":     {Default: "v1", Enum: []string{"v1", "v2"}},
				},
				Extra: map[string]any{"x-tier": "premium"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalServer(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalServerSlice tests
// =============================================================================

func TestEqualServerSlice(t *testing.T) {
	tests := []struct {
		name string
		a    []*Server
		b    []*Server
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
			a:    []*Server{},
			b:    []*Server{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    []*Server{},
			want: true,
		},
		// Same elements
		{
			name: "same single element",
			a:    []*Server{{URL: "https://api.example.com"}},
			b:    []*Server{{URL: "https://api.example.com"}},
			want: true,
		},
		{
			name: "same multiple elements",
			a: []*Server{
				{URL: "https://api.example.com"},
				{URL: "https://staging.example.com"},
			},
			b: []*Server{
				{URL: "https://api.example.com"},
				{URL: "https://staging.example.com"},
			},
			want: true,
		},
		// Different elements
		{
			name: "different elements",
			a:    []*Server{{URL: "https://api.example.com"}},
			b:    []*Server{{URL: "https://api.other.com"}},
			want: false,
		},
		{
			name: "different lengths",
			a: []*Server{
				{URL: "https://api.example.com"},
				{URL: "https://staging.example.com"},
			},
			b: []*Server{
				{URL: "https://api.example.com"},
			},
			want: false,
		},
		{
			name: "different order",
			a: []*Server{
				{URL: "https://api.example.com"},
				{URL: "https://staging.example.com"},
			},
			b: []*Server{
				{URL: "https://staging.example.com"},
				{URL: "https://api.example.com"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalServerSlice(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalInfo tests
// =============================================================================

func TestEqualInfo(t *testing.T) {
	tests := []struct {
		name string
		a    *Info
		b    *Info
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
			b:    &Info{Title: "API"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Info{Title: "API"},
			b:    nil,
			want: false,
		},
		// Title field
		{
			name: "different Title",
			a:    &Info{Title: "API v1"},
			b:    &Info{Title: "API v2"},
			want: false,
		},
		// Description field
		{
			name: "different Description",
			a:    &Info{Title: "API", Description: "Description 1"},
			b:    &Info{Title: "API", Description: "Description 2"},
			want: false,
		},
		// TermsOfService field
		{
			name: "different TermsOfService",
			a:    &Info{Title: "API", TermsOfService: "https://example.com/tos1"},
			b:    &Info{Title: "API", TermsOfService: "https://example.com/tos2"},
			want: false,
		},
		// Version field
		{
			name: "different Version",
			a:    &Info{Title: "API", Version: "1.0.0"},
			b:    &Info{Title: "API", Version: "2.0.0"},
			want: false,
		},
		// Summary field (OAS 3.1+)
		{
			name: "different Summary",
			a:    &Info{Title: "API", Summary: "Summary 1"},
			b:    &Info{Title: "API", Summary: "Summary 2"},
			want: false,
		},
		// Contact field
		{
			name: "different Contact",
			a:    &Info{Title: "API", Contact: &Contact{Name: "John"}},
			b:    &Info{Title: "API", Contact: &Contact{Name: "Jane"}},
			want: false,
		},
		// License field
		{
			name: "different License",
			a:    &Info{Title: "API", License: &License{Name: "MIT"}},
			b:    &Info{Title: "API", License: &License{Name: "Apache-2.0"}},
			want: false,
		},
		// Complete info equal
		{
			name: "complete info equal",
			a: &Info{
				Title:          "My API",
				Description:    "API Description",
				TermsOfService: "https://example.com/tos",
				Version:        "1.0.0",
				Summary:        "API Summary",
				Contact:        &Contact{Name: "Support", Email: "support@example.com"},
				License:        &License{Name: "MIT", URL: "https://opensource.org/licenses/MIT"},
			},
			b: &Info{
				Title:          "My API",
				Description:    "API Description",
				TermsOfService: "https://example.com/tos",
				Version:        "1.0.0",
				Summary:        "API Summary",
				Contact:        &Contact{Name: "Support", Email: "support@example.com"},
				License:        &License{Name: "MIT", URL: "https://opensource.org/licenses/MIT"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalInfo(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalContact tests
// =============================================================================

func TestEqualContact(t *testing.T) {
	tests := []struct {
		name string
		a    *Contact
		b    *Contact
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b non-nil",
			a:    nil,
			b:    &Contact{Name: "John"},
			want: false,
		},
		{
			name: "different Name",
			a:    &Contact{Name: "John"},
			b:    &Contact{Name: "Jane"},
			want: false,
		},
		{
			name: "different URL",
			a:    &Contact{URL: "https://example.com"},
			b:    &Contact{URL: "https://other.com"},
			want: false,
		},
		{
			name: "different Email",
			a:    &Contact{Email: "john@example.com"},
			b:    &Contact{Email: "jane@example.com"},
			want: false,
		},
		{
			name: "complete contact equal",
			a:    &Contact{Name: "Support", URL: "https://example.com", Email: "support@example.com"},
			b:    &Contact{Name: "Support", URL: "https://example.com", Email: "support@example.com"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalContact(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalLicense tests
// =============================================================================

func TestEqualLicense(t *testing.T) {
	tests := []struct {
		name string
		a    *License
		b    *License
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b non-nil",
			a:    nil,
			b:    &License{Name: "MIT"},
			want: false,
		},
		{
			name: "different Name",
			a:    &License{Name: "MIT"},
			b:    &License{Name: "Apache-2.0"},
			want: false,
		},
		{
			name: "different URL",
			a:    &License{Name: "MIT", URL: "https://opensource.org/licenses/MIT"},
			b:    &License{Name: "MIT", URL: "https://mit-license.org"},
			want: false,
		},
		{
			name: "different Identifier (OAS 3.1+)",
			a:    &License{Name: "MIT", Identifier: "MIT"},
			b:    &License{Name: "MIT", Identifier: "Apache-2.0"},
			want: false,
		},
		{
			name: "complete license equal",
			a:    &License{Name: "MIT", URL: "https://opensource.org/licenses/MIT", Identifier: "MIT"},
			b:    &License{Name: "MIT", URL: "https://opensource.org/licenses/MIT", Identifier: "MIT"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalLicense(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
