package parser

import (
	"testing"

	"github.com/erraggy/oastools/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// equalEncoding tests
// =============================================================================

func TestEqualEncoding(t *testing.T) {
	tests := []struct {
		name string
		a    *Encoding
		b    *Encoding
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
			b:    &Encoding{ContentType: "application/json"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Encoding{ContentType: "application/json"},
			b:    nil,
			want: false,
		},
		// Empty encodings
		{
			name: "both empty",
			a:    &Encoding{},
			b:    &Encoding{},
			want: true,
		},
		// AllowReserved field (boolean, checked first as cheapest)
		{
			name: "same AllowReserved true",
			a:    &Encoding{AllowReserved: true},
			b:    &Encoding{AllowReserved: true},
			want: true,
		},
		{
			name: "different AllowReserved",
			a:    &Encoding{AllowReserved: true},
			b:    &Encoding{AllowReserved: false},
			want: false,
		},
		// ContentType field
		{
			name: "same ContentType",
			a:    &Encoding{ContentType: "application/json"},
			b:    &Encoding{ContentType: "application/json"},
			want: true,
		},
		{
			name: "different ContentType",
			a:    &Encoding{ContentType: "application/json"},
			b:    &Encoding{ContentType: "application/xml"},
			want: false,
		},
		// Style field
		{
			name: "same Style",
			a:    &Encoding{Style: "form"},
			b:    &Encoding{Style: "form"},
			want: true,
		},
		{
			name: "different Style",
			a:    &Encoding{Style: "form"},
			b:    &Encoding{Style: "spaceDelimited"},
			want: false,
		},
		// Explode field (pointer)
		{
			name: "both Explode nil",
			a:    &Encoding{Explode: nil},
			b:    &Encoding{Explode: nil},
			want: true,
		},
		{
			name: "Explode nil vs non-nil",
			a:    &Encoding{Explode: nil},
			b:    &Encoding{Explode: testutil.Ptr(true)},
			want: false,
		},
		{
			name: "same Explode true",
			a:    &Encoding{Explode: testutil.Ptr(true)},
			b:    &Encoding{Explode: testutil.Ptr(true)},
			want: true,
		},
		{
			name: "same Explode false",
			a:    &Encoding{Explode: testutil.Ptr(false)},
			b:    &Encoding{Explode: testutil.Ptr(false)},
			want: true,
		},
		{
			name: "different Explode values",
			a:    &Encoding{Explode: testutil.Ptr(true)},
			b:    &Encoding{Explode: testutil.Ptr(false)},
			want: false,
		},
		// Headers field
		{
			name: "same Headers",
			a: &Encoding{
				Headers: map[string]*Header{
					"X-Rate-Limit": {Description: "Rate limit"},
				},
			},
			b: &Encoding{
				Headers: map[string]*Header{
					"X-Rate-Limit": {Description: "Rate limit"},
				},
			},
			want: true,
		},
		{
			name: "different Headers",
			a: &Encoding{
				Headers: map[string]*Header{
					"X-Rate-Limit": {Description: "Rate limit"},
				},
			},
			b: &Encoding{
				Headers: map[string]*Header{
					"X-Request-ID": {Description: "Request ID"},
				},
			},
			want: false,
		},
		{
			name: "Headers nil vs empty",
			a:    &Encoding{Headers: nil},
			b:    &Encoding{Headers: map[string]*Header{}},
			want: true,
		},
		{
			name: "Headers same key different value",
			a: &Encoding{
				Headers: map[string]*Header{
					"X-Custom": {Description: "Custom header 1"},
				},
			},
			b: &Encoding{
				Headers: map[string]*Header{
					"X-Custom": {Description: "Custom header 2"},
				},
			},
			want: false,
		},
		// Extra field (extensions)
		{
			name: "same Extra",
			a:    &Encoding{Extra: map[string]any{"x-custom": "value"}},
			b:    &Encoding{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &Encoding{Extra: map[string]any{"x-custom": "value1"}},
			b:    &Encoding{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		{
			name: "Extra nil vs empty",
			a:    &Encoding{Extra: nil},
			b:    &Encoding{Extra: map[string]any{}},
			want: true,
		},
		// Complete encoding - form style
		{
			name: "complete form encoding equal",
			a: &Encoding{
				ContentType:   "application/x-www-form-urlencoded",
				Style:         "form",
				Explode:       testutil.Ptr(true),
				AllowReserved: false,
				Headers: map[string]*Header{
					"X-Custom-Header": {Description: "Custom header", Schema: &Schema{Type: "string"}},
				},
				Extra: map[string]any{"x-encoding-version": 1},
			},
			b: &Encoding{
				ContentType:   "application/x-www-form-urlencoded",
				Style:         "form",
				Explode:       testutil.Ptr(true),
				AllowReserved: false,
				Headers: map[string]*Header{
					"X-Custom-Header": {Description: "Custom header", Schema: &Schema{Type: "string"}},
				},
				Extra: map[string]any{"x-encoding-version": 1},
			},
			want: true,
		},
		// Complete encoding - multipart
		{
			name: "complete multipart encoding equal",
			a: &Encoding{
				ContentType: "image/png",
				Headers: map[string]*Header{
					"Content-Disposition": {Schema: &Schema{Type: "string"}},
				},
			},
			b: &Encoding{
				ContentType: "image/png",
				Headers: map[string]*Header{
					"Content-Disposition": {Schema: &Schema{Type: "string"}},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalEncoding(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalEncodingMap tests
// =============================================================================

func TestEqualEncodingMap(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]*Encoding
		b    map[string]*Encoding
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
			a:    map[string]*Encoding{},
			b:    map[string]*Encoding{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    map[string]*Encoding{},
			want: true,
		},
		// Same entries
		{
			name: "same single entry",
			a: map[string]*Encoding{
				"file": {ContentType: "application/octet-stream"},
			},
			b: map[string]*Encoding{
				"file": {ContentType: "application/octet-stream"},
			},
			want: true,
		},
		{
			name: "same multiple entries",
			a: map[string]*Encoding{
				"file":     {ContentType: "application/octet-stream"},
				"metadata": {ContentType: "application/json"},
			},
			b: map[string]*Encoding{
				"file":     {ContentType: "application/octet-stream"},
				"metadata": {ContentType: "application/json"},
			},
			want: true,
		},
		// Different entries
		{
			name: "different values same key",
			a: map[string]*Encoding{
				"file": {ContentType: "application/octet-stream"},
			},
			b: map[string]*Encoding{
				"file": {ContentType: "image/png"},
			},
			want: false,
		},
		{
			name: "different keys",
			a: map[string]*Encoding{
				"file": {ContentType: "application/octet-stream"},
			},
			b: map[string]*Encoding{
				"document": {ContentType: "application/octet-stream"},
			},
			want: false,
		},
		{
			name: "a has extra key",
			a: map[string]*Encoding{
				"file":     {ContentType: "application/octet-stream"},
				"metadata": {ContentType: "application/json"},
			},
			b: map[string]*Encoding{
				"file": {ContentType: "application/octet-stream"},
			},
			want: false,
		},
		{
			name: "b has extra key",
			a: map[string]*Encoding{
				"file": {ContentType: "application/octet-stream"},
			},
			b: map[string]*Encoding{
				"file":     {ContentType: "application/octet-stream"},
				"metadata": {ContentType: "application/json"},
			},
			want: false,
		},
		// Encoding with nested differences
		{
			name: "same key different encoding style",
			a: map[string]*Encoding{
				"ids": {Style: "form", Explode: testutil.Ptr(true)},
			},
			b: map[string]*Encoding{
				"ids": {Style: "spaceDelimited", Explode: testutil.Ptr(false)},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalEncodingMap(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalOperationMap tests
// =============================================================================

func TestEqualOperationMap(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]*Operation
		b    map[string]*Operation
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
			a:    map[string]*Operation{},
			b:    map[string]*Operation{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    map[string]*Operation{},
			want: true,
		},
		// Same entries
		{
			name: "same single entry",
			a: map[string]*Operation{
				"CUSTOM": {OperationID: "customOperation", Summary: "Custom operation"},
			},
			b: map[string]*Operation{
				"CUSTOM": {OperationID: "customOperation", Summary: "Custom operation"},
			},
			want: true,
		},
		{
			name: "same multiple entries",
			a: map[string]*Operation{
				"CUSTOM":  {OperationID: "customOperation"},
				"SPECIAL": {OperationID: "specialOperation"},
			},
			b: map[string]*Operation{
				"CUSTOM":  {OperationID: "customOperation"},
				"SPECIAL": {OperationID: "specialOperation"},
			},
			want: true,
		},
		// Different entries
		{
			name: "different values same key",
			a: map[string]*Operation{
				"CUSTOM": {OperationID: "customOperation1"},
			},
			b: map[string]*Operation{
				"CUSTOM": {OperationID: "customOperation2"},
			},
			want: false,
		},
		{
			name: "different keys",
			a: map[string]*Operation{
				"CUSTOM": {OperationID: "customOperation"},
			},
			b: map[string]*Operation{
				"SPECIAL": {OperationID: "customOperation"},
			},
			want: false,
		},
		{
			name: "a has extra key",
			a: map[string]*Operation{
				"CUSTOM":  {OperationID: "customOperation"},
				"SPECIAL": {OperationID: "specialOperation"},
			},
			b: map[string]*Operation{
				"CUSTOM": {OperationID: "customOperation"},
			},
			want: false,
		},
		{
			name: "b has extra key",
			a: map[string]*Operation{
				"CUSTOM": {OperationID: "customOperation"},
			},
			b: map[string]*Operation{
				"CUSTOM":  {OperationID: "customOperation"},
				"SPECIAL": {OperationID: "specialOperation"},
			},
			want: false,
		},
		// Operation with complex nested structures
		{
			name: "same key different operation responses",
			a: map[string]*Operation{
				"CUSTOM": {
					OperationID: "customOperation",
					Responses: &Responses{
						Default: &Response{Description: "Success"},
					},
				},
			},
			b: map[string]*Operation{
				"CUSTOM": {
					OperationID: "customOperation",
					Responses: &Responses{
						Default: &Response{Description: "Error"},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalOperationMap(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalPathItem tests
// =============================================================================

func TestEqualPathItem(t *testing.T) {
	tests := []struct {
		name string
		a    *PathItem
		b    *PathItem
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
			b:    &PathItem{},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &PathItem{},
			b:    nil,
			want: false,
		},
		// Empty path items
		{
			name: "both empty",
			a:    &PathItem{},
			b:    &PathItem{},
			want: true,
		},
		// Ref field
		{
			name: "same Ref",
			a:    &PathItem{Ref: "#/components/pathItems/users"},
			b:    &PathItem{Ref: "#/components/pathItems/users"},
			want: true,
		},
		{
			name: "different Ref",
			a:    &PathItem{Ref: "#/components/pathItems/users"},
			b:    &PathItem{Ref: "#/components/pathItems/accounts"},
			want: false,
		},
		// Summary field
		{
			name: "different Summary",
			a:    &PathItem{Summary: "User operations"},
			b:    &PathItem{Summary: "Account operations"},
			want: false,
		},
		// Description field
		{
			name: "different Description",
			a:    &PathItem{Description: "Operations for users"},
			b:    &PathItem{Description: "Operations for accounts"},
			want: false,
		},
		// Operations - Get
		{
			name: "same Get operation",
			a:    &PathItem{Get: &Operation{OperationID: "getUsers"}},
			b:    &PathItem{Get: &Operation{OperationID: "getUsers"}},
			want: true,
		},
		{
			name: "different Get operation",
			a:    &PathItem{Get: &Operation{OperationID: "getUsers"}},
			b:    &PathItem{Get: &Operation{OperationID: "listUsers"}},
			want: false,
		},
		{
			name: "Get nil vs non-nil",
			a:    &PathItem{Get: nil},
			b:    &PathItem{Get: &Operation{OperationID: "getUsers"}},
			want: false,
		},
		// Operations - Put
		{
			name: "different Put operation",
			a:    &PathItem{Put: &Operation{OperationID: "updateUser"}},
			b:    &PathItem{Put: &Operation{OperationID: "replaceUser"}},
			want: false,
		},
		// Operations - Post
		{
			name: "different Post operation",
			a:    &PathItem{Post: &Operation{OperationID: "createUser"}},
			b:    &PathItem{Post: &Operation{OperationID: "addUser"}},
			want: false,
		},
		// Operations - Delete
		{
			name: "different Delete operation",
			a:    &PathItem{Delete: &Operation{OperationID: "deleteUser"}},
			b:    &PathItem{Delete: &Operation{OperationID: "removeUser"}},
			want: false,
		},
		// Operations - Options
		{
			name: "different Options operation",
			a:    &PathItem{Options: &Operation{OperationID: "optionsUser"}},
			b:    &PathItem{Options: &Operation{OperationID: "userOptions"}},
			want: false,
		},
		// Operations - Head
		{
			name: "different Head operation",
			a:    &PathItem{Head: &Operation{OperationID: "headUser"}},
			b:    &PathItem{Head: &Operation{OperationID: "userHead"}},
			want: false,
		},
		// Operations - Patch
		{
			name: "different Patch operation",
			a:    &PathItem{Patch: &Operation{OperationID: "patchUser"}},
			b:    &PathItem{Patch: &Operation{OperationID: "modifyUser"}},
			want: false,
		},
		// Operations - Trace (OAS 3.0+)
		{
			name: "different Trace operation",
			a:    &PathItem{Trace: &Operation{OperationID: "traceUser"}},
			b:    &PathItem{Trace: &Operation{OperationID: "userTrace"}},
			want: false,
		},
		// Operations - Query (OAS 3.2+)
		{
			name: "different Query operation",
			a:    &PathItem{Query: &Operation{OperationID: "queryUser"}},
			b:    &PathItem{Query: &Operation{OperationID: "searchUser"}},
			want: false,
		},
		// Servers field
		{
			name: "same Servers",
			a:    &PathItem{Servers: []*Server{{URL: "https://api.example.com"}}},
			b:    &PathItem{Servers: []*Server{{URL: "https://api.example.com"}}},
			want: true,
		},
		{
			name: "different Servers",
			a:    &PathItem{Servers: []*Server{{URL: "https://api.example.com"}}},
			b:    &PathItem{Servers: []*Server{{URL: "https://api.other.com"}}},
			want: false,
		},
		// Parameters field
		{
			name: "same Parameters",
			a:    &PathItem{Parameters: []*Parameter{{Name: "id", In: "path"}}},
			b:    &PathItem{Parameters: []*Parameter{{Name: "id", In: "path"}}},
			want: true,
		},
		{
			name: "different Parameters",
			a:    &PathItem{Parameters: []*Parameter{{Name: "id", In: "path"}}},
			b:    &PathItem{Parameters: []*Parameter{{Name: "userId", In: "path"}}},
			want: false,
		},
		// AdditionalOperations field (OAS 3.2+)
		{
			name: "same AdditionalOperations",
			a: &PathItem{
				AdditionalOperations: map[string]*Operation{
					"CUSTOM": {OperationID: "customOp"},
				},
			},
			b: &PathItem{
				AdditionalOperations: map[string]*Operation{
					"CUSTOM": {OperationID: "customOp"},
				},
			},
			want: true,
		},
		{
			name: "different AdditionalOperations",
			a: &PathItem{
				AdditionalOperations: map[string]*Operation{
					"CUSTOM": {OperationID: "customOp1"},
				},
			},
			b: &PathItem{
				AdditionalOperations: map[string]*Operation{
					"CUSTOM": {OperationID: "customOp2"},
				},
			},
			want: false,
		},
		{
			name: "AdditionalOperations nil vs empty",
			a:    &PathItem{AdditionalOperations: nil},
			b:    &PathItem{AdditionalOperations: map[string]*Operation{}},
			want: true,
		},
		// Extra field
		{
			name: "same Extra",
			a:    &PathItem{Extra: map[string]any{"x-custom": "value"}},
			b:    &PathItem{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &PathItem{Extra: map[string]any{"x-custom": "value1"}},
			b:    &PathItem{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		// Complete path item
		{
			name: "complete path item equal",
			a: &PathItem{
				Summary:     "User endpoint",
				Description: "Operations for user management",
				Get:         &Operation{OperationID: "getUser", Summary: "Get user"},
				Put:         &Operation{OperationID: "updateUser", Summary: "Update user"},
				Delete:      &Operation{OperationID: "deleteUser", Summary: "Delete user"},
				Parameters:  []*Parameter{{Name: "id", In: "path", Required: true}},
				Servers:     []*Server{{URL: "https://api.example.com"}},
			},
			b: &PathItem{
				Summary:     "User endpoint",
				Description: "Operations for user management",
				Get:         &Operation{OperationID: "getUser", Summary: "Get user"},
				Put:         &Operation{OperationID: "updateUser", Summary: "Update user"},
				Delete:      &Operation{OperationID: "deleteUser", Summary: "Delete user"},
				Parameters:  []*Parameter{{Name: "id", In: "path", Required: true}},
				Servers:     []*Server{{URL: "https://api.example.com"}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalPathItem(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalLink tests
// =============================================================================

func TestEqualLink(t *testing.T) {
	tests := []struct {
		name string
		a    *Link
		b    *Link
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
			b:    &Link{OperationID: "getUser"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Link{OperationID: "getUser"},
			b:    nil,
			want: false,
		},
		// Empty links
		{
			name: "both empty",
			a:    &Link{},
			b:    &Link{},
			want: true,
		},
		// Ref field
		{
			name: "same Ref",
			a:    &Link{Ref: "#/components/links/GetUser"},
			b:    &Link{Ref: "#/components/links/GetUser"},
			want: true,
		},
		{
			name: "different Ref",
			a:    &Link{Ref: "#/components/links/GetUser"},
			b:    &Link{Ref: "#/components/links/GetAccount"},
			want: false,
		},
		// OperationRef field
		{
			name: "same OperationRef",
			a:    &Link{OperationRef: "#/paths/~1users~1{id}/get"},
			b:    &Link{OperationRef: "#/paths/~1users~1{id}/get"},
			want: true,
		},
		{
			name: "different OperationRef",
			a:    &Link{OperationRef: "#/paths/~1users~1{id}/get"},
			b:    &Link{OperationRef: "#/paths/~1users~1{id}/put"},
			want: false,
		},
		// OperationID field
		{
			name: "same OperationID",
			a:    &Link{OperationID: "getUserById"},
			b:    &Link{OperationID: "getUserById"},
			want: true,
		},
		{
			name: "different OperationID",
			a:    &Link{OperationID: "getUserById"},
			b:    &Link{OperationID: "getAccountById"},
			want: false,
		},
		// Description field
		{
			name: "different Description",
			a:    &Link{OperationID: "getUser", Description: "Get user by ID"},
			b:    &Link{OperationID: "getUser", Description: "Retrieve user"},
			want: false,
		},
		// Parameters field
		{
			name: "same Parameters",
			a: &Link{
				OperationID: "getUser",
				Parameters:  map[string]any{"userId": "$response.body#/id"},
			},
			b: &Link{
				OperationID: "getUser",
				Parameters:  map[string]any{"userId": "$response.body#/id"},
			},
			want: true,
		},
		{
			name: "different Parameters",
			a: &Link{
				OperationID: "getUser",
				Parameters:  map[string]any{"userId": "$response.body#/id"},
			},
			b: &Link{
				OperationID: "getUser",
				Parameters:  map[string]any{"userId": "$response.body#/userId"},
			},
			want: false,
		},
		{
			name: "Parameters nil vs empty",
			a:    &Link{OperationID: "getUser", Parameters: nil},
			b:    &Link{OperationID: "getUser", Parameters: map[string]any{}},
			want: true,
		},
		// RequestBody field (any type)
		{
			name: "same RequestBody string",
			a:    &Link{OperationID: "createUser", RequestBody: "$response.body"},
			b:    &Link{OperationID: "createUser", RequestBody: "$response.body"},
			want: true,
		},
		{
			name: "different RequestBody",
			a:    &Link{OperationID: "createUser", RequestBody: "$response.body"},
			b:    &Link{OperationID: "createUser", RequestBody: "$request.body"},
			want: false,
		},
		{
			name: "RequestBody nil vs non-nil",
			a:    &Link{OperationID: "createUser", RequestBody: nil},
			b:    &Link{OperationID: "createUser", RequestBody: "$response.body"},
			want: false,
		},
		{
			name: "same RequestBody map",
			a: &Link{
				OperationID: "createUser",
				RequestBody: map[string]any{"name": "$response.body#/name"},
			},
			b: &Link{
				OperationID: "createUser",
				RequestBody: map[string]any{"name": "$response.body#/name"},
			},
			want: true,
		},
		// Server field
		{
			name: "same Server",
			a: &Link{
				OperationID: "getUser",
				Server:      &Server{URL: "https://api.example.com"},
			},
			b: &Link{
				OperationID: "getUser",
				Server:      &Server{URL: "https://api.example.com"},
			},
			want: true,
		},
		{
			name: "different Server",
			a: &Link{
				OperationID: "getUser",
				Server:      &Server{URL: "https://api.example.com"},
			},
			b: &Link{
				OperationID: "getUser",
				Server:      &Server{URL: "https://api.other.com"},
			},
			want: false,
		},
		{
			name: "Server nil vs non-nil",
			a:    &Link{OperationID: "getUser", Server: nil},
			b:    &Link{OperationID: "getUser", Server: &Server{URL: "https://api.example.com"}},
			want: false,
		},
		// Extra field
		{
			name: "same Extra",
			a:    &Link{OperationID: "getUser", Extra: map[string]any{"x-custom": "value"}},
			b:    &Link{OperationID: "getUser", Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &Link{OperationID: "getUser", Extra: map[string]any{"x-custom": "value1"}},
			b:    &Link{OperationID: "getUser", Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		// Complete link
		{
			name: "complete link equal",
			a: &Link{
				OperationID: "getUserById",
				Description: "Get user by the ID returned in the response",
				Parameters: map[string]any{
					"userId": "$response.body#/id",
				},
				Server: &Server{URL: "https://api.example.com"},
				Extra:  map[string]any{"x-timeout": 30},
			},
			b: &Link{
				OperationID: "getUserById",
				Description: "Get user by the ID returned in the response",
				Parameters: map[string]any{
					"userId": "$response.body#/id",
				},
				Server: &Server{URL: "https://api.example.com"},
				Extra:  map[string]any{"x-timeout": 30},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalLink(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalLinkMap tests
// =============================================================================

func TestEqualLinkMap(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]*Link
		b    map[string]*Link
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
			a:    map[string]*Link{},
			b:    map[string]*Link{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    map[string]*Link{},
			want: true,
		},
		// Same entries
		{
			name: "same single entry",
			a: map[string]*Link{
				"GetUserById": {OperationID: "getUserById"},
			},
			b: map[string]*Link{
				"GetUserById": {OperationID: "getUserById"},
			},
			want: true,
		},
		// Different entries
		{
			name: "different values same key",
			a: map[string]*Link{
				"GetUser": {OperationID: "getUserById"},
			},
			b: map[string]*Link{
				"GetUser": {OperationID: "getUserByName"},
			},
			want: false,
		},
		{
			name: "different keys",
			a: map[string]*Link{
				"GetUserById": {OperationID: "getUserById"},
			},
			b: map[string]*Link{
				"GetUserByName": {OperationID: "getUserById"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalLinkMap(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalPathItemMap tests
// =============================================================================

func TestEqualPathItemMap(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]*PathItem
		b    map[string]*PathItem
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
			a:    map[string]*PathItem{},
			b:    map[string]*PathItem{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    map[string]*PathItem{},
			want: true,
		},
		{
			name: "empty vs nil",
			a:    map[string]*PathItem{},
			b:    nil,
			want: true,
		},
		// Same entries
		{
			name: "same single entry",
			a: map[string]*PathItem{
				"newUser": {Post: &Operation{OperationID: "newUserWebhook"}},
			},
			b: map[string]*PathItem{
				"newUser": {Post: &Operation{OperationID: "newUserWebhook"}},
			},
			want: true,
		},
		{
			name: "same multiple entries",
			a: map[string]*PathItem{
				"newUser":  {Post: &Operation{OperationID: "newUserWebhook"}},
				"newOrder": {Post: &Operation{OperationID: "newOrderWebhook"}},
			},
			b: map[string]*PathItem{
				"newUser":  {Post: &Operation{OperationID: "newUserWebhook"}},
				"newOrder": {Post: &Operation{OperationID: "newOrderWebhook"}},
			},
			want: true,
		},
		// Different entries
		{
			name: "different values same key",
			a: map[string]*PathItem{
				"newUser": {Post: &Operation{OperationID: "webhook1"}},
			},
			b: map[string]*PathItem{
				"newUser": {Post: &Operation{OperationID: "webhook2"}},
			},
			want: false,
		},
		{
			name: "different keys same value",
			a: map[string]*PathItem{
				"newUser": {Post: &Operation{OperationID: "webhook"}},
			},
			b: map[string]*PathItem{
				"newOrder": {Post: &Operation{OperationID: "webhook"}},
			},
			want: false,
		},
		{
			name: "a has extra key",
			a: map[string]*PathItem{
				"newUser":  {Post: &Operation{OperationID: "webhook1"}},
				"newOrder": {Post: &Operation{OperationID: "webhook2"}},
			},
			b: map[string]*PathItem{
				"newUser": {Post: &Operation{OperationID: "webhook1"}},
			},
			want: false,
		},
		{
			name: "b has extra key",
			a: map[string]*PathItem{
				"newUser": {Post: &Operation{OperationID: "webhook1"}},
			},
			b: map[string]*PathItem{
				"newUser":  {Post: &Operation{OperationID: "webhook1"}},
				"newOrder": {Post: &Operation{OperationID: "webhook2"}},
			},
			want: false,
		},
		// Key exists but PathItem not found
		{
			name: "key not found in b",
			a: map[string]*PathItem{
				"webhook1": {Get: &Operation{OperationID: "get1"}},
			},
			b: map[string]*PathItem{
				"webhook2": {Get: &Operation{OperationID: "get1"}},
			},
			want: false,
		},
		// Nil PathItem values
		{
			name: "both have nil PathItem values",
			a: map[string]*PathItem{
				"webhook": nil,
			},
			b: map[string]*PathItem{
				"webhook": nil,
			},
			want: true,
		},
		{
			name: "nil vs non-nil PathItem value",
			a: map[string]*PathItem{
				"webhook": nil,
			},
			b: map[string]*PathItem{
				"webhook": {Get: &Operation{OperationID: "get"}},
			},
			want: false,
		},
		// Complete webhooks-like scenario
		{
			name: "webhooks scenario equal",
			a: map[string]*PathItem{
				"newUserWebhook": {
					Summary:     "New user notification",
					Description: "Webhook triggered when a new user is created",
					Post: &Operation{
						OperationID: "handleNewUser",
						RequestBody: &RequestBody{
							Description: "User data",
							Required:    true,
						},
					},
				},
			},
			b: map[string]*PathItem{
				"newUserWebhook": {
					Summary:     "New user notification",
					Description: "Webhook triggered when a new user is created",
					Post: &Operation{
						OperationID: "handleNewUser",
						RequestBody: &RequestBody{
							Description: "User data",
							Required:    true,
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalPathItemMap(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalResponseMap tests
// =============================================================================

func TestEqualResponseMap(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]*Response
		b    map[string]*Response
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
			a:    map[string]*Response{},
			b:    map[string]*Response{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    map[string]*Response{},
			want: true,
		},
		{
			name: "empty vs nil",
			a:    map[string]*Response{},
			b:    nil,
			want: true,
		},
		// Same entries
		{
			name: "same single entry",
			a: map[string]*Response{
				"200": {Description: "Success"},
			},
			b: map[string]*Response{
				"200": {Description: "Success"},
			},
			want: true,
		},
		{
			name: "same multiple entries",
			a: map[string]*Response{
				"200": {Description: "Success"},
				"404": {Description: "Not found"},
				"500": {Description: "Internal error"},
			},
			b: map[string]*Response{
				"200": {Description: "Success"},
				"404": {Description: "Not found"},
				"500": {Description: "Internal error"},
			},
			want: true,
		},
		// Different entries
		{
			name: "different values same key",
			a: map[string]*Response{
				"200": {Description: "Success"},
			},
			b: map[string]*Response{
				"200": {Description: "OK"},
			},
			want: false,
		},
		{
			name: "different keys",
			a: map[string]*Response{
				"200": {Description: "Success"},
			},
			b: map[string]*Response{
				"201": {Description: "Success"},
			},
			want: false,
		},
		{
			name: "a has extra key",
			a: map[string]*Response{
				"200": {Description: "Success"},
				"404": {Description: "Not found"},
			},
			b: map[string]*Response{
				"200": {Description: "Success"},
			},
			want: false,
		},
		{
			name: "b has extra key",
			a: map[string]*Response{
				"200": {Description: "Success"},
			},
			b: map[string]*Response{
				"200": {Description: "Success"},
				"404": {Description: "Not found"},
			},
			want: false,
		},
		// Key not found scenario
		{
			name: "key not found in b",
			a: map[string]*Response{
				"200": {Description: "Success"},
			},
			b: map[string]*Response{
				"201": {Description: "Created"},
			},
			want: false,
		},
		// Nil Response values
		{
			name: "both have nil Response values",
			a: map[string]*Response{
				"200": nil,
			},
			b: map[string]*Response{
				"200": nil,
			},
			want: true,
		},
		{
			name: "nil vs non-nil Response value",
			a: map[string]*Response{
				"200": nil,
			},
			b: map[string]*Response{
				"200": {Description: "Success"},
			},
			want: false,
		},
		// Response with nested content
		{
			name: "same Response with Content",
			a: map[string]*Response{
				"200": {
					Description: "Success",
					Content: map[string]*MediaType{
						"application/json": {Schema: &Schema{Type: "object"}},
					},
				},
			},
			b: map[string]*Response{
				"200": {
					Description: "Success",
					Content: map[string]*MediaType{
						"application/json": {Schema: &Schema{Type: "object"}},
					},
				},
			},
			want: true,
		},
		{
			name: "different Response Content",
			a: map[string]*Response{
				"200": {
					Description: "Success",
					Content: map[string]*MediaType{
						"application/json": {Schema: &Schema{Type: "object"}},
					},
				},
			},
			b: map[string]*Response{
				"200": {
					Description: "Success",
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
			got := equalResponseMap(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalCallback tests
// =============================================================================

func TestEqualCallback(t *testing.T) {
	tests := []struct {
		name string
		a    *Callback
		b    *Callback
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
			b:    &Callback{"{$request.body#/callbackUrl}": &PathItem{}},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Callback{"{$request.body#/callbackUrl}": &PathItem{}},
			b:    nil,
			want: false,
		},
		// Empty callbacks
		{
			name: "both empty",
			a:    &Callback{},
			b:    &Callback{},
			want: true,
		},
		// Same entries
		{
			name: "same single entry",
			a: &Callback{
				"{$request.body#/callbackUrl}": &PathItem{
					Post: &Operation{OperationID: "callback"},
				},
			},
			b: &Callback{
				"{$request.body#/callbackUrl}": &PathItem{
					Post: &Operation{OperationID: "callback"},
				},
			},
			want: true,
		},
		{
			name: "same multiple entries",
			a: &Callback{
				"{$request.body#/callbackUrl}": &PathItem{Post: &Operation{OperationID: "callback1"}},
				"{$request.body#/fallbackUrl}": &PathItem{Post: &Operation{OperationID: "callback2"}},
			},
			b: &Callback{
				"{$request.body#/callbackUrl}": &PathItem{Post: &Operation{OperationID: "callback1"}},
				"{$request.body#/fallbackUrl}": &PathItem{Post: &Operation{OperationID: "callback2"}},
			},
			want: true,
		},
		// Different entries
		{
			name: "different values same key",
			a: &Callback{
				"{$request.body#/callbackUrl}": &PathItem{
					Post: &Operation{OperationID: "callback1"},
				},
			},
			b: &Callback{
				"{$request.body#/callbackUrl}": &PathItem{
					Post: &Operation{OperationID: "callback2"},
				},
			},
			want: false,
		},
		{
			name: "different keys",
			a: &Callback{
				"{$request.body#/callbackUrl}": &PathItem{},
			},
			b: &Callback{
				"{$request.body#/otherUrl}": &PathItem{},
			},
			want: false,
		},
		{
			name: "a has extra entry",
			a: &Callback{
				"{$request.body#/callbackUrl}": &PathItem{},
				"{$request.body#/fallbackUrl}": &PathItem{},
			},
			b: &Callback{
				"{$request.body#/callbackUrl}": &PathItem{},
			},
			want: false,
		},
		{
			name: "b has extra entry",
			a: &Callback{
				"{$request.body#/callbackUrl}": &PathItem{},
			},
			b: &Callback{
				"{$request.body#/callbackUrl}": &PathItem{},
				"{$request.body#/fallbackUrl}": &PathItem{},
			},
			want: false,
		},
		// Real-world callback scenario
		{
			name: "subscription callback equal",
			a: &Callback{
				"{$request.body#/callbackUrl}?event={$request.body#/eventType}": &PathItem{
					Post: &Operation{
						OperationID: "subscriptionCallback",
						RequestBody: &RequestBody{
							Description: "Event payload",
							Required:    true,
							Content: map[string]*MediaType{
								"application/json": {Schema: &Schema{Type: "object"}},
							},
						},
						Responses: &Responses{
							Codes: map[string]*Response{
								"200": {Description: "Callback processed"},
							},
						},
					},
				},
			},
			b: &Callback{
				"{$request.body#/callbackUrl}?event={$request.body#/eventType}": &PathItem{
					Post: &Operation{
						OperationID: "subscriptionCallback",
						RequestBody: &RequestBody{
							Description: "Event payload",
							Required:    true,
							Content: map[string]*MediaType{
								"application/json": {Schema: &Schema{Type: "object"}},
							},
						},
						Responses: &Responses{
							Codes: map[string]*Response{
								"200": {Description: "Callback processed"},
							},
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalCallback(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalMediaType tests
// =============================================================================

func TestEqualMediaType(t *testing.T) {
	tests := []struct {
		name string
		a    *MediaType
		b    *MediaType
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
			b:    &MediaType{Schema: &Schema{Type: "object"}},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &MediaType{Schema: &Schema{Type: "object"}},
			b:    nil,
			want: false,
		},
		// Empty media types
		{
			name: "both empty",
			a:    &MediaType{},
			b:    &MediaType{},
			want: true,
		},
		// Schema field
		{
			name: "same Schema",
			a:    &MediaType{Schema: &Schema{Type: "object"}},
			b:    &MediaType{Schema: &Schema{Type: "object"}},
			want: true,
		},
		{
			name: "different Schema",
			a:    &MediaType{Schema: &Schema{Type: "object"}},
			b:    &MediaType{Schema: &Schema{Type: "string"}},
			want: false,
		},
		{
			name: "Schema nil vs non-nil",
			a:    &MediaType{Schema: nil},
			b:    &MediaType{Schema: &Schema{Type: "object"}},
			want: false,
		},
		{
			name: "Schema non-nil vs nil",
			a:    &MediaType{Schema: &Schema{Type: "object"}},
			b:    &MediaType{Schema: nil},
			want: false,
		},
		// Example field
		{
			name: "same Example string",
			a:    &MediaType{Example: "example value"},
			b:    &MediaType{Example: "example value"},
			want: true,
		},
		{
			name: "different Example",
			a:    &MediaType{Example: "value1"},
			b:    &MediaType{Example: "value2"},
			want: false,
		},
		{
			name: "same Example map",
			a:    &MediaType{Example: map[string]any{"id": 1, "name": "test"}},
			b:    &MediaType{Example: map[string]any{"id": 1, "name": "test"}},
			want: true,
		},
		{
			name: "Example nil vs non-nil",
			a:    &MediaType{Example: nil},
			b:    &MediaType{Example: "value"},
			want: false,
		},
		// Examples field
		{
			name: "same Examples",
			a: &MediaType{Examples: map[string]*Example{
				"default": {Summary: "Default example", Value: "example"},
			}},
			b: &MediaType{Examples: map[string]*Example{
				"default": {Summary: "Default example", Value: "example"},
			}},
			want: true,
		},
		{
			name: "different Examples",
			a: &MediaType{Examples: map[string]*Example{
				"example1": {Summary: "Example 1"},
			}},
			b: &MediaType{Examples: map[string]*Example{
				"example2": {Summary: "Example 2"},
			}},
			want: false,
		},
		{
			name: "Examples nil vs empty",
			a:    &MediaType{Examples: nil},
			b:    &MediaType{Examples: map[string]*Example{}},
			want: true,
		},
		// Encoding field
		{
			name: "same Encoding",
			a: &MediaType{Encoding: map[string]*Encoding{
				"file": {ContentType: "application/octet-stream"},
			}},
			b: &MediaType{Encoding: map[string]*Encoding{
				"file": {ContentType: "application/octet-stream"},
			}},
			want: true,
		},
		{
			name: "different Encoding",
			a: &MediaType{Encoding: map[string]*Encoding{
				"file": {ContentType: "application/octet-stream"},
			}},
			b: &MediaType{Encoding: map[string]*Encoding{
				"file": {ContentType: "image/png"},
			}},
			want: false,
		},
		{
			name: "Encoding nil vs empty",
			a:    &MediaType{Encoding: nil},
			b:    &MediaType{Encoding: map[string]*Encoding{}},
			want: true,
		},
		// Extra field
		{
			name: "same Extra",
			a:    &MediaType{Extra: map[string]any{"x-custom": "value"}},
			b:    &MediaType{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &MediaType{Extra: map[string]any{"x-custom": "value1"}},
			b:    &MediaType{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		{
			name: "Extra nil vs empty",
			a:    &MediaType{Extra: nil},
			b:    &MediaType{Extra: map[string]any{}},
			want: true,
		},
		// Complete MediaType
		{
			name: "complete MediaType equal",
			a: &MediaType{
				Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"id":   {Type: "integer"},
						"name": {Type: "string"},
					},
				},
				Example: map[string]any{"id": 1, "name": "John"},
				Examples: map[string]*Example{
					"default": {Summary: "Default example", Value: map[string]any{"id": 1}},
				},
				Encoding: map[string]*Encoding{
					"avatar": {ContentType: "image/png"},
				},
				Extra: map[string]any{"x-version": 1},
			},
			b: &MediaType{
				Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"id":   {Type: "integer"},
						"name": {Type: "string"},
					},
				},
				Example: map[string]any{"id": 1, "name": "John"},
				Examples: map[string]*Example{
					"default": {Summary: "Default example", Value: map[string]any{"id": 1}},
				},
				Encoding: map[string]*Encoding{
					"avatar": {ContentType: "image/png"},
				},
				Extra: map[string]any{"x-version": 1},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalMediaType(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalExample tests
// =============================================================================

func TestEqualExample(t *testing.T) {
	tests := []struct {
		name string
		a    *Example
		b    *Example
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
			b:    &Example{Summary: "Example"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Example{Summary: "Example"},
			b:    nil,
			want: false,
		},
		// Empty examples
		{
			name: "both empty",
			a:    &Example{},
			b:    &Example{},
			want: true,
		},
		// Ref field
		{
			name: "same Ref",
			a:    &Example{Ref: "#/components/examples/UserExample"},
			b:    &Example{Ref: "#/components/examples/UserExample"},
			want: true,
		},
		{
			name: "different Ref",
			a:    &Example{Ref: "#/components/examples/UserExample"},
			b:    &Example{Ref: "#/components/examples/AccountExample"},
			want: false,
		},
		// Summary field
		{
			name: "same Summary",
			a:    &Example{Summary: "Example summary"},
			b:    &Example{Summary: "Example summary"},
			want: true,
		},
		{
			name: "different Summary",
			a:    &Example{Summary: "Summary 1"},
			b:    &Example{Summary: "Summary 2"},
			want: false,
		},
		// Description field
		{
			name: "same Description",
			a:    &Example{Description: "Example description"},
			b:    &Example{Description: "Example description"},
			want: true,
		},
		{
			name: "different Description",
			a:    &Example{Description: "Description 1"},
			b:    &Example{Description: "Description 2"},
			want: false,
		},
		// ExternalValue field
		{
			name: "same ExternalValue",
			a:    &Example{ExternalValue: "https://example.com/examples/user.json"},
			b:    &Example{ExternalValue: "https://example.com/examples/user.json"},
			want: true,
		},
		{
			name: "different ExternalValue",
			a:    &Example{ExternalValue: "https://example.com/examples/user.json"},
			b:    &Example{ExternalValue: "https://example.com/examples/account.json"},
			want: false,
		},
		// Value field
		{
			name: "same Value string",
			a:    &Example{Value: "example string"},
			b:    &Example{Value: "example string"},
			want: true,
		},
		{
			name: "different Value string",
			a:    &Example{Value: "value1"},
			b:    &Example{Value: "value2"},
			want: false,
		},
		{
			name: "same Value map",
			a:    &Example{Value: map[string]any{"id": 1, "name": "John"}},
			b:    &Example{Value: map[string]any{"id": 1, "name": "John"}},
			want: true,
		},
		{
			name: "different Value map",
			a:    &Example{Value: map[string]any{"id": 1}},
			b:    &Example{Value: map[string]any{"id": 2}},
			want: false,
		},
		{
			name: "Value nil vs non-nil",
			a:    &Example{Value: nil},
			b:    &Example{Value: "value"},
			want: false,
		},
		{
			name: "same Value array",
			a:    &Example{Value: []any{1, 2, 3}},
			b:    &Example{Value: []any{1, 2, 3}},
			want: true,
		},
		// Extra field
		{
			name: "same Extra",
			a:    &Example{Extra: map[string]any{"x-custom": "value"}},
			b:    &Example{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &Example{Extra: map[string]any{"x-custom": "value1"}},
			b:    &Example{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		{
			name: "Extra nil vs empty",
			a:    &Example{Extra: nil},
			b:    &Example{Extra: map[string]any{}},
			want: true,
		},
		// Complete Example
		{
			name: "complete Example equal",
			a: &Example{
				Summary:     "User example",
				Description: "An example of a user object",
				Value: map[string]any{
					"id":    1,
					"name":  "John Doe",
					"email": "john@example.com",
				},
				Extra: map[string]any{"x-version": 1},
			},
			b: &Example{
				Summary:     "User example",
				Description: "An example of a user object",
				Value: map[string]any{
					"id":    1,
					"name":  "John Doe",
					"email": "john@example.com",
				},
				Extra: map[string]any{"x-version": 1},
			},
			want: true,
		},
		// Example with ExternalValue
		{
			name: "external value example equal",
			a: &Example{
				Summary:       "External user example",
				ExternalValue: "https://example.com/examples/user.json",
			},
			b: &Example{
				Summary:       "External user example",
				ExternalValue: "https://example.com/examples/user.json",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalExample(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
