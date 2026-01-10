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

// =============================================================================
// equalOperation tests
// =============================================================================

func TestEqualOperation(t *testing.T) {
	tests := []struct {
		name string
		a    *Operation
		b    *Operation
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
			b:    &Operation{OperationID: "getUsers"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Operation{OperationID: "getUsers"},
			b:    nil,
			want: false,
		},
		// Empty operations
		{
			name: "both empty",
			a:    &Operation{},
			b:    &Operation{},
			want: true,
		},
		// Boolean fields
		{
			name: "same Deprecated true",
			a:    &Operation{OperationID: "old", Deprecated: true},
			b:    &Operation{OperationID: "old", Deprecated: true},
			want: true,
		},
		{
			name: "different Deprecated",
			a:    &Operation{OperationID: "getUsers", Deprecated: true},
			b:    &Operation{OperationID: "getUsers", Deprecated: false},
			want: false,
		},
		// String fields
		{
			name: "different Summary",
			a:    &Operation{Summary: "Get users"},
			b:    &Operation{Summary: "List users"},
			want: false,
		},
		{
			name: "different Description",
			a:    &Operation{Description: "Returns all users"},
			b:    &Operation{Description: "Fetches user list"},
			want: false,
		},
		{
			name: "different OperationID",
			a:    &Operation{OperationID: "getUsers"},
			b:    &Operation{OperationID: "listUsers"},
			want: false,
		},
		// Tags slice
		{
			name: "same Tags",
			a:    &Operation{Tags: []string{"users", "admin"}},
			b:    &Operation{Tags: []string{"users", "admin"}},
			want: true,
		},
		{
			name: "different Tags",
			a:    &Operation{Tags: []string{"users"}},
			b:    &Operation{Tags: []string{"users", "admin"}},
			want: false,
		},
		{
			name: "Tags different order",
			a:    &Operation{Tags: []string{"admin", "users"}},
			b:    &Operation{Tags: []string{"users", "admin"}},
			want: false,
		},
		{
			name: "Tags nil vs empty",
			a:    &Operation{Tags: nil},
			b:    &Operation{Tags: []string{}},
			want: true,
		},
		// OAS 2.0 specific fields
		{
			name: "same Consumes",
			a:    &Operation{Consumes: []string{"application/json"}},
			b:    &Operation{Consumes: []string{"application/json"}},
			want: true,
		},
		{
			name: "different Consumes",
			a:    &Operation{Consumes: []string{"application/json"}},
			b:    &Operation{Consumes: []string{"application/xml"}},
			want: false,
		},
		{
			name: "same Produces",
			a:    &Operation{Produces: []string{"application/json"}},
			b:    &Operation{Produces: []string{"application/json"}},
			want: true,
		},
		{
			name: "different Produces",
			a:    &Operation{Produces: []string{"application/json"}},
			b:    &Operation{Produces: []string{"application/xml"}},
			want: false,
		},
		{
			name: "same Schemes",
			a:    &Operation{Schemes: []string{"https"}},
			b:    &Operation{Schemes: []string{"https"}},
			want: true,
		},
		{
			name: "different Schemes",
			a:    &Operation{Schemes: []string{"https"}},
			b:    &Operation{Schemes: []string{"http", "https"}},
			want: false,
		},
		// ExternalDocs
		{
			name: "same ExternalDocs",
			a:    &Operation{ExternalDocs: &ExternalDocs{URL: "https://example.com/docs"}},
			b:    &Operation{ExternalDocs: &ExternalDocs{URL: "https://example.com/docs"}},
			want: true,
		},
		{
			name: "different ExternalDocs",
			a:    &Operation{ExternalDocs: &ExternalDocs{URL: "https://example.com/docs"}},
			b:    &Operation{ExternalDocs: &ExternalDocs{URL: "https://other.com/docs"}},
			want: false,
		},
		{
			name: "ExternalDocs nil vs non-nil",
			a:    &Operation{ExternalDocs: nil},
			b:    &Operation{ExternalDocs: &ExternalDocs{URL: "https://example.com"}},
			want: false,
		},
		// RequestBody (OAS 3.0+)
		{
			name: "same RequestBody",
			a: &Operation{
				RequestBody: &RequestBody{
					Description: "User data",
					Required:    true,
				},
			},
			b: &Operation{
				RequestBody: &RequestBody{
					Description: "User data",
					Required:    true,
				},
			},
			want: true,
		},
		{
			name: "different RequestBody",
			a: &Operation{
				RequestBody: &RequestBody{
					Description: "User data",
					Required:    true,
				},
			},
			b: &Operation{
				RequestBody: &RequestBody{
					Description: "Different data",
					Required:    false,
				},
			},
			want: false,
		},
		{
			name: "RequestBody nil vs non-nil",
			a:    &Operation{RequestBody: nil},
			b:    &Operation{RequestBody: &RequestBody{Required: true}},
			want: false,
		},
		// Responses
		{
			name: "same Responses",
			a: &Operation{
				Responses: &Responses{
					Default: &Response{Description: "Success"},
				},
			},
			b: &Operation{
				Responses: &Responses{
					Default: &Response{Description: "Success"},
				},
			},
			want: true,
		},
		{
			name: "different Responses",
			a: &Operation{
				Responses: &Responses{
					Default: &Response{Description: "Success"},
				},
			},
			b: &Operation{
				Responses: &Responses{
					Default: &Response{Description: "Error"},
				},
			},
			want: false,
		},
		{
			name: "Responses with status codes",
			a: &Operation{
				Responses: &Responses{
					Codes: map[string]*Response{
						"200": {Description: "OK"},
						"404": {Description: "Not found"},
					},
				},
			},
			b: &Operation{
				Responses: &Responses{
					Codes: map[string]*Response{
						"200": {Description: "OK"},
						"404": {Description: "Not found"},
					},
				},
			},
			want: true,
		},
		{
			name: "Responses nil vs non-nil",
			a:    &Operation{Responses: nil},
			b:    &Operation{Responses: &Responses{Default: &Response{}}},
			want: false,
		},
		// Parameters
		{
			name: "same Parameters",
			a: &Operation{
				Parameters: []*Parameter{
					{Name: "id", In: "path", Required: true},
				},
			},
			b: &Operation{
				Parameters: []*Parameter{
					{Name: "id", In: "path", Required: true},
				},
			},
			want: true,
		},
		{
			name: "different Parameters",
			a: &Operation{
				Parameters: []*Parameter{
					{Name: "id", In: "path"},
				},
			},
			b: &Operation{
				Parameters: []*Parameter{
					{Name: "name", In: "query"},
				},
			},
			want: false,
		},
		{
			name: "Parameters different length",
			a: &Operation{
				Parameters: []*Parameter{
					{Name: "id", In: "path"},
				},
			},
			b: &Operation{
				Parameters: []*Parameter{
					{Name: "id", In: "path"},
					{Name: "name", In: "query"},
				},
			},
			want: false,
		},
		// Security
		{
			name: "same Security",
			a: &Operation{
				Security: []SecurityRequirement{
					{"api_key": []string{}},
				},
			},
			b: &Operation{
				Security: []SecurityRequirement{
					{"api_key": []string{}},
				},
			},
			want: true,
		},
		{
			name: "different Security",
			a: &Operation{
				Security: []SecurityRequirement{
					{"api_key": []string{}},
				},
			},
			b: &Operation{
				Security: []SecurityRequirement{
					{"oauth2": []string{"read:users"}},
				},
			},
			want: false,
		},
		{
			name: "Security nil vs empty",
			a:    &Operation{Security: nil},
			b:    &Operation{Security: []SecurityRequirement{}},
			want: true,
		},
		// Servers (OAS 3.0+)
		{
			name: "same Servers",
			a: &Operation{
				Servers: []*Server{
					{URL: "https://api.example.com"},
				},
			},
			b: &Operation{
				Servers: []*Server{
					{URL: "https://api.example.com"},
				},
			},
			want: true,
		},
		{
			name: "different Servers",
			a: &Operation{
				Servers: []*Server{
					{URL: "https://api.example.com"},
				},
			},
			b: &Operation{
				Servers: []*Server{
					{URL: "https://api.other.com"},
				},
			},
			want: false,
		},
		// Callbacks (OAS 3.0+)
		{
			name: "same Callbacks",
			a: &Operation{
				Callbacks: map[string]*Callback{
					"onEvent": {
						"{$request.body#/callbackUrl}": &PathItem{
							Post: &Operation{OperationID: "eventCallback"},
						},
					},
				},
			},
			b: &Operation{
				Callbacks: map[string]*Callback{
					"onEvent": {
						"{$request.body#/callbackUrl}": &PathItem{
							Post: &Operation{OperationID: "eventCallback"},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "different Callbacks",
			a: &Operation{
				Callbacks: map[string]*Callback{
					"onEvent": {},
				},
			},
			b: &Operation{
				Callbacks: map[string]*Callback{
					"onOther": {},
				},
			},
			want: false,
		},
		{
			name: "Callbacks nil vs empty",
			a:    &Operation{Callbacks: nil},
			b:    &Operation{Callbacks: map[string]*Callback{}},
			want: true,
		},
		// Extra (extensions)
		{
			name: "same Extra",
			a:    &Operation{Extra: map[string]any{"x-custom": "value"}},
			b:    &Operation{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &Operation{Extra: map[string]any{"x-custom": "value1"}},
			b:    &Operation{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		// Complete operation comparison
		{
			name: "complete OAS 3.0 operation",
			a: &Operation{
				Tags:        []string{"users"},
				Summary:     "Get user by ID",
				Description: "Returns a single user",
				OperationID: "getUserById",
				Parameters: []*Parameter{
					{Name: "id", In: "path", Required: true, Schema: &Schema{Type: "string"}},
				},
				Responses: &Responses{
					Codes: map[string]*Response{
						"200": {Description: "Successful response"},
						"404": {Description: "User not found"},
					},
				},
				Deprecated: false,
				Security: []SecurityRequirement{
					{"api_key": []string{}},
				},
			},
			b: &Operation{
				Tags:        []string{"users"},
				Summary:     "Get user by ID",
				Description: "Returns a single user",
				OperationID: "getUserById",
				Parameters: []*Parameter{
					{Name: "id", In: "path", Required: true, Schema: &Schema{Type: "string"}},
				},
				Responses: &Responses{
					Codes: map[string]*Response{
						"200": {Description: "Successful response"},
						"404": {Description: "User not found"},
					},
				},
				Deprecated: false,
				Security: []SecurityRequirement{
					{"api_key": []string{}},
				},
			},
			want: true,
		},
		{
			name: "complete OAS 2.0 operation",
			a: &Operation{
				Tags:        []string{"users"},
				Summary:     "Get user by ID",
				OperationID: "getUserById",
				Consumes:    []string{"application/json"},
				Produces:    []string{"application/json"},
				Schemes:     []string{"https"},
				Parameters: []*Parameter{
					{Name: "id", In: "path", Required: true, Type: "string"},
				},
				Responses: &Responses{
					Codes: map[string]*Response{
						"200": {Description: "Success"},
					},
				},
			},
			b: &Operation{
				Tags:        []string{"users"},
				Summary:     "Get user by ID",
				OperationID: "getUserById",
				Consumes:    []string{"application/json"},
				Produces:    []string{"application/json"},
				Schemes:     []string{"https"},
				Parameters: []*Parameter{
					{Name: "id", In: "path", Required: true, Type: "string"},
				},
				Responses: &Responses{
					Codes: map[string]*Response{
						"200": {Description: "Success"},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalOperation(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

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
			b:    &Header{Explode: boolPtr(true)},
			want: false,
		},
		{
			name: "same Explode true",
			a:    &Header{Explode: boolPtr(true)},
			b:    &Header{Explode: boolPtr(true)},
			want: true,
		},
		{
			name: "different Explode values",
			a:    &Header{Explode: boolPtr(true)},
			b:    &Header{Explode: boolPtr(false)},
			want: false,
		},
		// Pointer fields - numeric
		{
			name: "different Maximum",
			a:    &Header{Maximum: ptr(100.0)},
			b:    &Header{Maximum: ptr(200.0)},
			want: false,
		},
		{
			name: "Maximum nil vs non-nil",
			a:    &Header{Maximum: nil},
			b:    &Header{Maximum: ptr(100.0)},
			want: false,
		},
		{
			name: "different Minimum",
			a:    &Header{Minimum: ptr(0.0)},
			b:    &Header{Minimum: ptr(1.0)},
			want: false,
		},
		{
			name: "different MultipleOf",
			a:    &Header{MultipleOf: ptr(5.0)},
			b:    &Header{MultipleOf: ptr(10.0)},
			want: false,
		},
		{
			name: "different MaxLength",
			a:    &Header{MaxLength: intPtr(100)},
			b:    &Header{MaxLength: intPtr(200)},
			want: false,
		},
		{
			name: "different MinLength",
			a:    &Header{MinLength: intPtr(1)},
			b:    &Header{MinLength: intPtr(5)},
			want: false,
		},
		{
			name: "different MaxItems",
			a:    &Header{MaxItems: intPtr(10)},
			b:    &Header{MaxItems: intPtr(20)},
			want: false,
		},
		{
			name: "different MinItems",
			a:    &Header{MinItems: intPtr(1)},
			b:    &Header{MinItems: intPtr(2)},
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
				Minimum:     ptr(0.0),
				Maximum:     ptr(1000.0),
			},
			b: &Header{
				Description: "Rate limit remaining",
				Type:        "integer",
				Format:      "int32",
				Minimum:     ptr(0.0),
				Maximum:     ptr(1000.0),
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
