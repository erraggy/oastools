package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
