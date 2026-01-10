package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// OAS3Document.Equals tests
// =============================================================================

func TestOAS3Document_Equals(t *testing.T) {
	tests := []struct {
		name string
		a    *OAS3Document
		b    *OAS3Document
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
			b:    &OAS3Document{OpenAPI: "3.0.0"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &OAS3Document{OpenAPI: "3.0.0"},
			b:    nil,
			want: false,
		},
		// Empty documents
		{
			name: "both empty",
			a:    &OAS3Document{},
			b:    &OAS3Document{},
			want: true,
		},
		// OASVersion field (enum - cheapest comparison)
		{
			name: "same OASVersion",
			a:    &OAS3Document{OASVersion: OASVersion310},
			b:    &OAS3Document{OASVersion: OASVersion310},
			want: true,
		},
		{
			name: "different OASVersion",
			a:    &OAS3Document{OASVersion: OASVersion300},
			b:    &OAS3Document{OASVersion: OASVersion310},
			want: false,
		},
		// OpenAPI field
		{
			name: "same OpenAPI",
			a:    &OAS3Document{OpenAPI: "3.0.0"},
			b:    &OAS3Document{OpenAPI: "3.0.0"},
			want: true,
		},
		{
			name: "different OpenAPI",
			a:    &OAS3Document{OpenAPI: "3.0.0"},
			b:    &OAS3Document{OpenAPI: "3.1.0"},
			want: false,
		},
		// JSONSchemaDialect field (OAS 3.1+)
		{
			name: "same JSONSchemaDialect",
			a:    &OAS3Document{JSONSchemaDialect: "https://json-schema.org/draft/2020-12/schema"},
			b:    &OAS3Document{JSONSchemaDialect: "https://json-schema.org/draft/2020-12/schema"},
			want: true,
		},
		{
			name: "different JSONSchemaDialect",
			a:    &OAS3Document{JSONSchemaDialect: "https://json-schema.org/draft/2020-12/schema"},
			b:    &OAS3Document{JSONSchemaDialect: "https://json-schema.org/draft/2019-09/schema"},
			want: false,
		},
		// Self field (OAS 3.2+)
		{
			name: "same Self",
			a:    &OAS3Document{Self: "https://example.com/openapi.yaml"},
			b:    &OAS3Document{Self: "https://example.com/openapi.yaml"},
			want: true,
		},
		{
			name: "different Self",
			a:    &OAS3Document{Self: "https://example.com/openapi.yaml"},
			b:    &OAS3Document{Self: "https://other.com/openapi.yaml"},
			want: false,
		},
		// Info field
		{
			name: "same Info",
			a:    &OAS3Document{Info: &Info{Title: "My API", Version: "1.0.0"}},
			b:    &OAS3Document{Info: &Info{Title: "My API", Version: "1.0.0"}},
			want: true,
		},
		{
			name: "different Info",
			a:    &OAS3Document{Info: &Info{Title: "My API", Version: "1.0.0"}},
			b:    &OAS3Document{Info: &Info{Title: "Other API", Version: "1.0.0"}},
			want: false,
		},
		{
			name: "Info nil vs non-nil",
			a:    &OAS3Document{Info: nil},
			b:    &OAS3Document{Info: &Info{Title: "My API"}},
			want: false,
		},
		// ExternalDocs field
		{
			name: "same ExternalDocs",
			a:    &OAS3Document{ExternalDocs: &ExternalDocs{URL: "https://docs.example.com"}},
			b:    &OAS3Document{ExternalDocs: &ExternalDocs{URL: "https://docs.example.com"}},
			want: true,
		},
		{
			name: "different ExternalDocs",
			a:    &OAS3Document{ExternalDocs: &ExternalDocs{URL: "https://docs.example.com"}},
			b:    &OAS3Document{ExternalDocs: &ExternalDocs{URL: "https://other.example.com"}},
			want: false,
		},
		{
			name: "ExternalDocs nil vs non-nil",
			a:    &OAS3Document{ExternalDocs: nil},
			b:    &OAS3Document{ExternalDocs: &ExternalDocs{URL: "https://docs.example.com"}},
			want: false,
		},
		// Components field
		{
			name: "same Components",
			a: &OAS3Document{
				Components: &Components{
					Schemas: map[string]*Schema{
						"User": {Type: "object"},
					},
				},
			},
			b: &OAS3Document{
				Components: &Components{
					Schemas: map[string]*Schema{
						"User": {Type: "object"},
					},
				},
			},
			want: true,
		},
		{
			name: "different Components",
			a: &OAS3Document{
				Components: &Components{
					Schemas: map[string]*Schema{
						"User": {Type: "object"},
					},
				},
			},
			b: &OAS3Document{
				Components: &Components{
					Schemas: map[string]*Schema{
						"Account": {Type: "object"},
					},
				},
			},
			want: false,
		},
		{
			name: "Components nil vs non-nil",
			a:    &OAS3Document{Components: nil},
			b:    &OAS3Document{Components: &Components{}},
			want: false,
		},
		// Servers field
		{
			name: "same Servers",
			a:    &OAS3Document{Servers: []*Server{{URL: "https://api.example.com"}}},
			b:    &OAS3Document{Servers: []*Server{{URL: "https://api.example.com"}}},
			want: true,
		},
		{
			name: "different Servers",
			a:    &OAS3Document{Servers: []*Server{{URL: "https://api.example.com"}}},
			b:    &OAS3Document{Servers: []*Server{{URL: "https://api.other.com"}}},
			want: false,
		},
		{
			name: "Servers nil vs empty",
			a:    &OAS3Document{Servers: nil},
			b:    &OAS3Document{Servers: []*Server{}},
			want: true,
		},
		{
			name: "Servers different length",
			a: &OAS3Document{Servers: []*Server{
				{URL: "https://api.example.com"},
				{URL: "https://staging.example.com"},
			}},
			b:    &OAS3Document{Servers: []*Server{{URL: "https://api.example.com"}}},
			want: false,
		},
		// Security field
		{
			name: "same Security",
			a: &OAS3Document{Security: []SecurityRequirement{
				{"api_key": []string{}},
			}},
			b: &OAS3Document{Security: []SecurityRequirement{
				{"api_key": []string{}},
			}},
			want: true,
		},
		{
			name: "different Security",
			a: &OAS3Document{Security: []SecurityRequirement{
				{"api_key": []string{}},
			}},
			b: &OAS3Document{Security: []SecurityRequirement{
				{"oauth2": []string{"read:users"}},
			}},
			want: false,
		},
		{
			name: "Security nil vs empty",
			a:    &OAS3Document{Security: nil},
			b:    &OAS3Document{Security: []SecurityRequirement{}},
			want: true,
		},
		// Tags field
		{
			name: "same Tags",
			a:    &OAS3Document{Tags: []*Tag{{Name: "users"}}},
			b:    &OAS3Document{Tags: []*Tag{{Name: "users"}}},
			want: true,
		},
		{
			name: "different Tags",
			a:    &OAS3Document{Tags: []*Tag{{Name: "users"}}},
			b:    &OAS3Document{Tags: []*Tag{{Name: "accounts"}}},
			want: false,
		},
		{
			name: "Tags nil vs empty",
			a:    &OAS3Document{Tags: nil},
			b:    &OAS3Document{Tags: []*Tag{}},
			want: true,
		},
		// Paths field
		{
			name: "same Paths",
			a: &OAS3Document{Paths: Paths{
				"/users": &PathItem{Get: &Operation{OperationID: "getUsers"}},
			}},
			b: &OAS3Document{Paths: Paths{
				"/users": &PathItem{Get: &Operation{OperationID: "getUsers"}},
			}},
			want: true,
		},
		{
			name: "different Paths",
			a: &OAS3Document{Paths: Paths{
				"/users": &PathItem{Get: &Operation{OperationID: "getUsers"}},
			}},
			b: &OAS3Document{Paths: Paths{
				"/accounts": &PathItem{Get: &Operation{OperationID: "getAccounts"}},
			}},
			want: false,
		},
		{
			name: "Paths nil vs empty",
			a:    &OAS3Document{Paths: nil},
			b:    &OAS3Document{Paths: Paths{}},
			want: true,
		},
		// Webhooks field (OAS 3.1+)
		{
			name: "same Webhooks",
			a: &OAS3Document{Webhooks: map[string]*PathItem{
				"newUser": {Post: &Operation{OperationID: "newUserWebhook"}},
			}},
			b: &OAS3Document{Webhooks: map[string]*PathItem{
				"newUser": {Post: &Operation{OperationID: "newUserWebhook"}},
			}},
			want: true,
		},
		{
			name: "different Webhooks",
			a: &OAS3Document{Webhooks: map[string]*PathItem{
				"newUser": {Post: &Operation{OperationID: "newUserWebhook"}},
			}},
			b: &OAS3Document{Webhooks: map[string]*PathItem{
				"newOrder": {Post: &Operation{OperationID: "newOrderWebhook"}},
			}},
			want: false,
		},
		{
			name: "Webhooks nil vs empty",
			a:    &OAS3Document{Webhooks: nil},
			b:    &OAS3Document{Webhooks: map[string]*PathItem{}},
			want: true,
		},
		// Extra field
		{
			name: "same Extra",
			a:    &OAS3Document{Extra: map[string]any{"x-custom": "value"}},
			b:    &OAS3Document{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &OAS3Document{Extra: map[string]any{"x-custom": "value1"}},
			b:    &OAS3Document{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		{
			name: "Extra nil vs empty",
			a:    &OAS3Document{Extra: nil},
			b:    &OAS3Document{Extra: map[string]any{}},
			want: true,
		},
		// Complete OAS 3.0 document
		{
			name: "complete OAS 3.0 document equal",
			a: &OAS3Document{
				OpenAPI:    "3.0.3",
				OASVersion: OASVersion303,
				Info:       &Info{Title: "My API", Version: "1.0.0"},
				Servers: []*Server{
					{URL: "https://api.example.com", Description: "Production"},
				},
				Paths: Paths{
					"/users": &PathItem{
						Get: &Operation{
							OperationID: "getUsers",
							Summary:     "Get all users",
						},
					},
				},
				Components: &Components{
					Schemas: map[string]*Schema{
						"User": {Type: "object"},
					},
				},
				Security: []SecurityRequirement{
					{"api_key": []string{}},
				},
				Tags: []*Tag{
					{Name: "users", Description: "User operations"},
				},
				ExternalDocs: &ExternalDocs{URL: "https://docs.example.com"},
			},
			b: &OAS3Document{
				OpenAPI:    "3.0.3",
				OASVersion: OASVersion303,
				Info:       &Info{Title: "My API", Version: "1.0.0"},
				Servers: []*Server{
					{URL: "https://api.example.com", Description: "Production"},
				},
				Paths: Paths{
					"/users": &PathItem{
						Get: &Operation{
							OperationID: "getUsers",
							Summary:     "Get all users",
						},
					},
				},
				Components: &Components{
					Schemas: map[string]*Schema{
						"User": {Type: "object"},
					},
				},
				Security: []SecurityRequirement{
					{"api_key": []string{}},
				},
				Tags: []*Tag{
					{Name: "users", Description: "User operations"},
				},
				ExternalDocs: &ExternalDocs{URL: "https://docs.example.com"},
			},
			want: true,
		},
		// Complete OAS 3.1 document with new fields
		{
			name: "complete OAS 3.1 document equal",
			a: &OAS3Document{
				OpenAPI:           "3.1.0",
				OASVersion:        OASVersion310,
				Info:              &Info{Title: "My API", Version: "1.0.0", Summary: "API Summary"},
				JSONSchemaDialect: "https://json-schema.org/draft/2020-12/schema",
				Webhooks: map[string]*PathItem{
					"newUser": {Post: &Operation{OperationID: "newUserWebhook"}},
				},
				Components: &Components{
					PathItems: map[string]*PathItem{
						"SharedPath": {Get: &Operation{OperationID: "sharedGet"}},
					},
				},
			},
			b: &OAS3Document{
				OpenAPI:           "3.1.0",
				OASVersion:        OASVersion310,
				Info:              &Info{Title: "My API", Version: "1.0.0", Summary: "API Summary"},
				JSONSchemaDialect: "https://json-schema.org/draft/2020-12/schema",
				Webhooks: map[string]*PathItem{
					"newUser": {Post: &Operation{OperationID: "newUserWebhook"}},
				},
				Components: &Components{
					PathItems: map[string]*PathItem{
						"SharedPath": {Get: &Operation{OperationID: "sharedGet"}},
					},
				},
			},
			want: true,
		},
		// Complete OAS 3.2 document with Self field
		{
			name: "complete OAS 3.2 document equal",
			a: &OAS3Document{
				OpenAPI:    "3.2.0",
				OASVersion: OASVersion320,
				Self:       "https://example.com/openapi.yaml",
				Info:       &Info{Title: "My API", Version: "1.0.0"},
				Components: &Components{
					MediaTypes: map[string]*MediaType{
						"application/json": {Schema: &Schema{Type: "object"}},
					},
				},
			},
			b: &OAS3Document{
				OpenAPI:    "3.2.0",
				OASVersion: OASVersion320,
				Self:       "https://example.com/openapi.yaml",
				Info:       &Info{Title: "My API", Version: "1.0.0"},
				Components: &Components{
					MediaTypes: map[string]*MediaType{
						"application/json": {Schema: &Schema{Type: "object"}},
					},
				},
			},
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

// =============================================================================
// OAS2Document.Equals tests
// =============================================================================

func TestOAS2Document_Equals(t *testing.T) {
	tests := []struct {
		name string
		a    *OAS2Document
		b    *OAS2Document
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
			b:    &OAS2Document{Swagger: "2.0"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &OAS2Document{Swagger: "2.0"},
			b:    nil,
			want: false,
		},
		// Empty documents
		{
			name: "both empty",
			a:    &OAS2Document{},
			b:    &OAS2Document{},
			want: true,
		},
		// OASVersion field (enum - cheapest comparison)
		{
			name: "same OASVersion",
			a:    &OAS2Document{OASVersion: OASVersion20},
			b:    &OAS2Document{OASVersion: OASVersion20},
			want: true,
		},
		{
			name: "different OASVersion",
			a:    &OAS2Document{OASVersion: OASVersion20},
			b:    &OAS2Document{OASVersion: Unknown},
			want: false,
		},
		// Swagger field
		{
			name: "same Swagger",
			a:    &OAS2Document{Swagger: "2.0"},
			b:    &OAS2Document{Swagger: "2.0"},
			want: true,
		},
		{
			name: "different Swagger",
			a:    &OAS2Document{Swagger: "2.0"},
			b:    &OAS2Document{Swagger: "2.1"},
			want: false,
		},
		// Host field
		{
			name: "same Host",
			a:    &OAS2Document{Host: "api.example.com"},
			b:    &OAS2Document{Host: "api.example.com"},
			want: true,
		},
		{
			name: "different Host",
			a:    &OAS2Document{Host: "api.example.com"},
			b:    &OAS2Document{Host: "api.other.com"},
			want: false,
		},
		// BasePath field
		{
			name: "same BasePath",
			a:    &OAS2Document{BasePath: "/v1"},
			b:    &OAS2Document{BasePath: "/v1"},
			want: true,
		},
		{
			name: "different BasePath",
			a:    &OAS2Document{BasePath: "/v1"},
			b:    &OAS2Document{BasePath: "/v2"},
			want: false,
		},
		// Info field
		{
			name: "same Info",
			a:    &OAS2Document{Info: &Info{Title: "My API", Version: "1.0.0"}},
			b:    &OAS2Document{Info: &Info{Title: "My API", Version: "1.0.0"}},
			want: true,
		},
		{
			name: "different Info",
			a:    &OAS2Document{Info: &Info{Title: "My API", Version: "1.0.0"}},
			b:    &OAS2Document{Info: &Info{Title: "Other API", Version: "1.0.0"}},
			want: false,
		},
		{
			name: "Info nil vs non-nil",
			a:    &OAS2Document{Info: nil},
			b:    &OAS2Document{Info: &Info{Title: "My API"}},
			want: false,
		},
		// ExternalDocs field
		{
			name: "same ExternalDocs",
			a:    &OAS2Document{ExternalDocs: &ExternalDocs{URL: "https://docs.example.com"}},
			b:    &OAS2Document{ExternalDocs: &ExternalDocs{URL: "https://docs.example.com"}},
			want: true,
		},
		{
			name: "different ExternalDocs",
			a:    &OAS2Document{ExternalDocs: &ExternalDocs{URL: "https://docs.example.com"}},
			b:    &OAS2Document{ExternalDocs: &ExternalDocs{URL: "https://other.example.com"}},
			want: false,
		},
		{
			name: "ExternalDocs nil vs non-nil",
			a:    &OAS2Document{ExternalDocs: nil},
			b:    &OAS2Document{ExternalDocs: &ExternalDocs{URL: "https://docs.example.com"}},
			want: false,
		},
		// Schemes field
		{
			name: "same Schemes",
			a:    &OAS2Document{Schemes: []string{"https", "http"}},
			b:    &OAS2Document{Schemes: []string{"https", "http"}},
			want: true,
		},
		{
			name: "different Schemes",
			a:    &OAS2Document{Schemes: []string{"https"}},
			b:    &OAS2Document{Schemes: []string{"http"}},
			want: false,
		},
		{
			name: "Schemes nil vs empty",
			a:    &OAS2Document{Schemes: nil},
			b:    &OAS2Document{Schemes: []string{}},
			want: true,
		},
		{
			name: "Schemes different order",
			a:    &OAS2Document{Schemes: []string{"https", "http"}},
			b:    &OAS2Document{Schemes: []string{"http", "https"}},
			want: false,
		},
		// Consumes field
		{
			name: "same Consumes",
			a:    &OAS2Document{Consumes: []string{"application/json"}},
			b:    &OAS2Document{Consumes: []string{"application/json"}},
			want: true,
		},
		{
			name: "different Consumes",
			a:    &OAS2Document{Consumes: []string{"application/json"}},
			b:    &OAS2Document{Consumes: []string{"application/xml"}},
			want: false,
		},
		{
			name: "Consumes nil vs empty",
			a:    &OAS2Document{Consumes: nil},
			b:    &OAS2Document{Consumes: []string{}},
			want: true,
		},
		// Produces field
		{
			name: "same Produces",
			a:    &OAS2Document{Produces: []string{"application/json"}},
			b:    &OAS2Document{Produces: []string{"application/json"}},
			want: true,
		},
		{
			name: "different Produces",
			a:    &OAS2Document{Produces: []string{"application/json"}},
			b:    &OAS2Document{Produces: []string{"application/xml"}},
			want: false,
		},
		{
			name: "Produces nil vs empty",
			a:    &OAS2Document{Produces: nil},
			b:    &OAS2Document{Produces: []string{}},
			want: true,
		},
		// Security field
		{
			name: "same Security",
			a: &OAS2Document{Security: []SecurityRequirement{
				{"api_key": []string{}},
			}},
			b: &OAS2Document{Security: []SecurityRequirement{
				{"api_key": []string{}},
			}},
			want: true,
		},
		{
			name: "different Security",
			a: &OAS2Document{Security: []SecurityRequirement{
				{"api_key": []string{}},
			}},
			b: &OAS2Document{Security: []SecurityRequirement{
				{"oauth2": []string{"read:users"}},
			}},
			want: false,
		},
		{
			name: "Security nil vs empty",
			a:    &OAS2Document{Security: nil},
			b:    &OAS2Document{Security: []SecurityRequirement{}},
			want: true,
		},
		// Tags field
		{
			name: "same Tags",
			a:    &OAS2Document{Tags: []*Tag{{Name: "users"}}},
			b:    &OAS2Document{Tags: []*Tag{{Name: "users"}}},
			want: true,
		},
		{
			name: "different Tags",
			a:    &OAS2Document{Tags: []*Tag{{Name: "users"}}},
			b:    &OAS2Document{Tags: []*Tag{{Name: "accounts"}}},
			want: false,
		},
		{
			name: "Tags nil vs empty",
			a:    &OAS2Document{Tags: nil},
			b:    &OAS2Document{Tags: []*Tag{}},
			want: true,
		},
		// Paths field
		{
			name: "same Paths",
			a: &OAS2Document{Paths: Paths{
				"/users": &PathItem{Get: &Operation{OperationID: "getUsers"}},
			}},
			b: &OAS2Document{Paths: Paths{
				"/users": &PathItem{Get: &Operation{OperationID: "getUsers"}},
			}},
			want: true,
		},
		{
			name: "different Paths",
			a: &OAS2Document{Paths: Paths{
				"/users": &PathItem{Get: &Operation{OperationID: "getUsers"}},
			}},
			b: &OAS2Document{Paths: Paths{
				"/accounts": &PathItem{Get: &Operation{OperationID: "getAccounts"}},
			}},
			want: false,
		},
		{
			name: "Paths nil vs empty",
			a:    &OAS2Document{Paths: nil},
			b:    &OAS2Document{Paths: Paths{}},
			want: true,
		},
		// Definitions field
		{
			name: "same Definitions",
			a: &OAS2Document{Definitions: map[string]*Schema{
				"User": {Type: "object"},
			}},
			b: &OAS2Document{Definitions: map[string]*Schema{
				"User": {Type: "object"},
			}},
			want: true,
		},
		{
			name: "different Definitions",
			a: &OAS2Document{Definitions: map[string]*Schema{
				"User": {Type: "object"},
			}},
			b: &OAS2Document{Definitions: map[string]*Schema{
				"Account": {Type: "object"},
			}},
			want: false,
		},
		{
			name: "Definitions nil vs empty",
			a:    &OAS2Document{Definitions: nil},
			b:    &OAS2Document{Definitions: map[string]*Schema{}},
			want: true,
		},
		// Parameters field
		{
			name: "same Parameters",
			a: &OAS2Document{Parameters: map[string]*Parameter{
				"userId": {Name: "userId", In: "path", Required: true},
			}},
			b: &OAS2Document{Parameters: map[string]*Parameter{
				"userId": {Name: "userId", In: "path", Required: true},
			}},
			want: true,
		},
		{
			name: "different Parameters",
			a: &OAS2Document{Parameters: map[string]*Parameter{
				"userId": {Name: "userId", In: "path"},
			}},
			b: &OAS2Document{Parameters: map[string]*Parameter{
				"accountId": {Name: "accountId", In: "path"},
			}},
			want: false,
		},
		{
			name: "Parameters nil vs empty",
			a:    &OAS2Document{Parameters: nil},
			b:    &OAS2Document{Parameters: map[string]*Parameter{}},
			want: true,
		},
		// Responses field
		{
			name: "same Responses",
			a: &OAS2Document{Responses: map[string]*Response{
				"NotFound": {Description: "Not found"},
			}},
			b: &OAS2Document{Responses: map[string]*Response{
				"NotFound": {Description: "Not found"},
			}},
			want: true,
		},
		{
			name: "different Responses",
			a: &OAS2Document{Responses: map[string]*Response{
				"NotFound": {Description: "Not found"},
			}},
			b: &OAS2Document{Responses: map[string]*Response{
				"BadRequest": {Description: "Bad request"},
			}},
			want: false,
		},
		{
			name: "Responses nil vs empty",
			a:    &OAS2Document{Responses: nil},
			b:    &OAS2Document{Responses: map[string]*Response{}},
			want: true,
		},
		// SecurityDefinitions field
		{
			name: "same SecurityDefinitions",
			a: &OAS2Document{SecurityDefinitions: map[string]*SecurityScheme{
				"api_key": {Type: "apiKey", Name: "X-API-Key", In: "header"},
			}},
			b: &OAS2Document{SecurityDefinitions: map[string]*SecurityScheme{
				"api_key": {Type: "apiKey", Name: "X-API-Key", In: "header"},
			}},
			want: true,
		},
		{
			name: "different SecurityDefinitions",
			a: &OAS2Document{SecurityDefinitions: map[string]*SecurityScheme{
				"api_key": {Type: "apiKey", Name: "X-API-Key", In: "header"},
			}},
			b: &OAS2Document{SecurityDefinitions: map[string]*SecurityScheme{
				"basic": {Type: "basic"},
			}},
			want: false,
		},
		{
			name: "SecurityDefinitions nil vs empty",
			a:    &OAS2Document{SecurityDefinitions: nil},
			b:    &OAS2Document{SecurityDefinitions: map[string]*SecurityScheme{}},
			want: true,
		},
		// Extra field
		{
			name: "same Extra",
			a:    &OAS2Document{Extra: map[string]any{"x-custom": "value"}},
			b:    &OAS2Document{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &OAS2Document{Extra: map[string]any{"x-custom": "value1"}},
			b:    &OAS2Document{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		{
			name: "Extra nil vs empty",
			a:    &OAS2Document{Extra: nil},
			b:    &OAS2Document{Extra: map[string]any{}},
			want: true,
		},
		// Complete OAS 2.0 document
		{
			name: "complete OAS 2.0 document equal",
			a: &OAS2Document{
				Swagger:    "2.0",
				OASVersion: OASVersion20,
				Info:       &Info{Title: "My API", Version: "1.0.0"},
				Host:       "api.example.com",
				BasePath:   "/v1",
				Schemes:    []string{"https"},
				Consumes:   []string{"application/json"},
				Produces:   []string{"application/json"},
				Paths: Paths{
					"/users": &PathItem{
						Get: &Operation{
							OperationID: "getUsers",
							Summary:     "Get all users",
						},
					},
				},
				Definitions: map[string]*Schema{
					"User": {Type: "object"},
				},
				Parameters: map[string]*Parameter{
					"userId": {Name: "userId", In: "path", Required: true},
				},
				Responses: map[string]*Response{
					"NotFound": {Description: "Not found"},
				},
				SecurityDefinitions: map[string]*SecurityScheme{
					"api_key": {Type: "apiKey", Name: "X-API-Key", In: "header"},
				},
				Security: []SecurityRequirement{
					{"api_key": []string{}},
				},
				Tags: []*Tag{
					{Name: "users", Description: "User operations"},
				},
				ExternalDocs: &ExternalDocs{URL: "https://docs.example.com"},
			},
			b: &OAS2Document{
				Swagger:    "2.0",
				OASVersion: OASVersion20,
				Info:       &Info{Title: "My API", Version: "1.0.0"},
				Host:       "api.example.com",
				BasePath:   "/v1",
				Schemes:    []string{"https"},
				Consumes:   []string{"application/json"},
				Produces:   []string{"application/json"},
				Paths: Paths{
					"/users": &PathItem{
						Get: &Operation{
							OperationID: "getUsers",
							Summary:     "Get all users",
						},
					},
				},
				Definitions: map[string]*Schema{
					"User": {Type: "object"},
				},
				Parameters: map[string]*Parameter{
					"userId": {Name: "userId", In: "path", Required: true},
				},
				Responses: map[string]*Response{
					"NotFound": {Description: "Not found"},
				},
				SecurityDefinitions: map[string]*SecurityScheme{
					"api_key": {Type: "apiKey", Name: "X-API-Key", In: "header"},
				},
				Security: []SecurityRequirement{
					{"api_key": []string{}},
				},
				Tags: []*Tag{
					{Name: "users", Description: "User operations"},
				},
				ExternalDocs: &ExternalDocs{URL: "https://docs.example.com"},
			},
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

// =============================================================================
// equalComponents tests
// =============================================================================

func TestEqualComponents(t *testing.T) {
	tests := []struct {
		name string
		a    *Components
		b    *Components
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
			b:    &Components{},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Components{},
			b:    nil,
			want: false,
		},
		// Empty components
		{
			name: "both empty",
			a:    &Components{},
			b:    &Components{},
			want: true,
		},
		// Schemas field
		{
			name: "same Schemas",
			a: &Components{Schemas: map[string]*Schema{
				"User": {Type: "object"},
			}},
			b: &Components{Schemas: map[string]*Schema{
				"User": {Type: "object"},
			}},
			want: true,
		},
		{
			name: "different Schemas",
			a: &Components{Schemas: map[string]*Schema{
				"User": {Type: "object"},
			}},
			b: &Components{Schemas: map[string]*Schema{
				"Account": {Type: "object"},
			}},
			want: false,
		},
		{
			name: "Schemas nil vs empty",
			a:    &Components{Schemas: nil},
			b:    &Components{Schemas: map[string]*Schema{}},
			want: true,
		},
		{
			name: "Schemas same key different value",
			a: &Components{Schemas: map[string]*Schema{
				"User": {Type: "object"},
			}},
			b: &Components{Schemas: map[string]*Schema{
				"User": {Type: "string"},
			}},
			want: false,
		},
		// Responses field
		{
			name: "same Responses",
			a: &Components{Responses: map[string]*Response{
				"NotFound": {Description: "Not found"},
			}},
			b: &Components{Responses: map[string]*Response{
				"NotFound": {Description: "Not found"},
			}},
			want: true,
		},
		{
			name: "different Responses",
			a: &Components{Responses: map[string]*Response{
				"NotFound": {Description: "Not found"},
			}},
			b: &Components{Responses: map[string]*Response{
				"BadRequest": {Description: "Bad request"},
			}},
			want: false,
		},
		{
			name: "Responses nil vs empty",
			a:    &Components{Responses: nil},
			b:    &Components{Responses: map[string]*Response{}},
			want: true,
		},
		// Parameters field
		{
			name: "same Parameters",
			a: &Components{Parameters: map[string]*Parameter{
				"userId": {Name: "userId", In: "path"},
			}},
			b: &Components{Parameters: map[string]*Parameter{
				"userId": {Name: "userId", In: "path"},
			}},
			want: true,
		},
		{
			name: "different Parameters",
			a: &Components{Parameters: map[string]*Parameter{
				"userId": {Name: "userId", In: "path"},
			}},
			b: &Components{Parameters: map[string]*Parameter{
				"accountId": {Name: "accountId", In: "path"},
			}},
			want: false,
		},
		{
			name: "Parameters nil vs empty",
			a:    &Components{Parameters: nil},
			b:    &Components{Parameters: map[string]*Parameter{}},
			want: true,
		},
		// Examples field
		{
			name: "same Examples",
			a: &Components{Examples: map[string]*Example{
				"UserExample": {Summary: "Example user", Value: map[string]any{"id": 1}},
			}},
			b: &Components{Examples: map[string]*Example{
				"UserExample": {Summary: "Example user", Value: map[string]any{"id": 1}},
			}},
			want: true,
		},
		{
			name: "different Examples",
			a: &Components{Examples: map[string]*Example{
				"UserExample": {Summary: "Example user"},
			}},
			b: &Components{Examples: map[string]*Example{
				"AccountExample": {Summary: "Example account"},
			}},
			want: false,
		},
		{
			name: "Examples nil vs empty",
			a:    &Components{Examples: nil},
			b:    &Components{Examples: map[string]*Example{}},
			want: true,
		},
		// RequestBodies field
		{
			name: "same RequestBodies",
			a: &Components{RequestBodies: map[string]*RequestBody{
				"UserInput": {Description: "User data", Required: true},
			}},
			b: &Components{RequestBodies: map[string]*RequestBody{
				"UserInput": {Description: "User data", Required: true},
			}},
			want: true,
		},
		{
			name: "different RequestBodies",
			a: &Components{RequestBodies: map[string]*RequestBody{
				"UserInput": {Description: "User data"},
			}},
			b: &Components{RequestBodies: map[string]*RequestBody{
				"AccountInput": {Description: "Account data"},
			}},
			want: false,
		},
		{
			name: "RequestBodies nil vs empty",
			a:    &Components{RequestBodies: nil},
			b:    &Components{RequestBodies: map[string]*RequestBody{}},
			want: true,
		},
		// Headers field
		{
			name: "same Headers",
			a: &Components{Headers: map[string]*Header{
				"X-Rate-Limit": {Description: "Rate limit"},
			}},
			b: &Components{Headers: map[string]*Header{
				"X-Rate-Limit": {Description: "Rate limit"},
			}},
			want: true,
		},
		{
			name: "different Headers",
			a: &Components{Headers: map[string]*Header{
				"X-Rate-Limit": {Description: "Rate limit"},
			}},
			b: &Components{Headers: map[string]*Header{
				"X-Request-ID": {Description: "Request ID"},
			}},
			want: false,
		},
		{
			name: "Headers nil vs empty",
			a:    &Components{Headers: nil},
			b:    &Components{Headers: map[string]*Header{}},
			want: true,
		},
		// SecuritySchemes field
		{
			name: "same SecuritySchemes",
			a: &Components{SecuritySchemes: map[string]*SecurityScheme{
				"api_key": {Type: "apiKey", Name: "X-API-Key", In: "header"},
			}},
			b: &Components{SecuritySchemes: map[string]*SecurityScheme{
				"api_key": {Type: "apiKey", Name: "X-API-Key", In: "header"},
			}},
			want: true,
		},
		{
			name: "different SecuritySchemes",
			a: &Components{SecuritySchemes: map[string]*SecurityScheme{
				"api_key": {Type: "apiKey"},
			}},
			b: &Components{SecuritySchemes: map[string]*SecurityScheme{
				"oauth2": {Type: "oauth2"},
			}},
			want: false,
		},
		{
			name: "SecuritySchemes nil vs empty",
			a:    &Components{SecuritySchemes: nil},
			b:    &Components{SecuritySchemes: map[string]*SecurityScheme{}},
			want: true,
		},
		// Links field
		{
			name: "same Links",
			a: &Components{Links: map[string]*Link{
				"GetUserById": {OperationID: "getUserById"},
			}},
			b: &Components{Links: map[string]*Link{
				"GetUserById": {OperationID: "getUserById"},
			}},
			want: true,
		},
		{
			name: "different Links",
			a: &Components{Links: map[string]*Link{
				"GetUserById": {OperationID: "getUserById"},
			}},
			b: &Components{Links: map[string]*Link{
				"GetAccountById": {OperationID: "getAccountById"},
			}},
			want: false,
		},
		{
			name: "Links nil vs empty",
			a:    &Components{Links: nil},
			b:    &Components{Links: map[string]*Link{}},
			want: true,
		},
		// Callbacks field
		{
			name: "same Callbacks",
			a: &Components{Callbacks: map[string]*Callback{
				"onEvent": {
					"{$request.body#/callbackUrl}": &PathItem{
						Post: &Operation{OperationID: "eventCallback"},
					},
				},
			}},
			b: &Components{Callbacks: map[string]*Callback{
				"onEvent": {
					"{$request.body#/callbackUrl}": &PathItem{
						Post: &Operation{OperationID: "eventCallback"},
					},
				},
			}},
			want: true,
		},
		{
			name: "different Callbacks",
			a: &Components{Callbacks: map[string]*Callback{
				"onEvent": {},
			}},
			b: &Components{Callbacks: map[string]*Callback{
				"onOther": {},
			}},
			want: false,
		},
		{
			name: "Callbacks nil vs empty",
			a:    &Components{Callbacks: nil},
			b:    &Components{Callbacks: map[string]*Callback{}},
			want: true,
		},
		// PathItems field (OAS 3.1+)
		{
			name: "same PathItems",
			a: &Components{PathItems: map[string]*PathItem{
				"SharedPath": {Get: &Operation{OperationID: "sharedGet"}},
			}},
			b: &Components{PathItems: map[string]*PathItem{
				"SharedPath": {Get: &Operation{OperationID: "sharedGet"}},
			}},
			want: true,
		},
		{
			name: "different PathItems",
			a: &Components{PathItems: map[string]*PathItem{
				"SharedPath": {Get: &Operation{OperationID: "sharedGet"}},
			}},
			b: &Components{PathItems: map[string]*PathItem{
				"OtherPath": {Get: &Operation{OperationID: "otherGet"}},
			}},
			want: false,
		},
		{
			name: "PathItems nil vs empty",
			a:    &Components{PathItems: nil},
			b:    &Components{PathItems: map[string]*PathItem{}},
			want: true,
		},
		// MediaTypes field (OAS 3.2+)
		{
			name: "same MediaTypes",
			a: &Components{MediaTypes: map[string]*MediaType{
				"application/json": {Schema: &Schema{Type: "object"}},
			}},
			b: &Components{MediaTypes: map[string]*MediaType{
				"application/json": {Schema: &Schema{Type: "object"}},
			}},
			want: true,
		},
		{
			name: "different MediaTypes",
			a: &Components{MediaTypes: map[string]*MediaType{
				"application/json": {Schema: &Schema{Type: "object"}},
			}},
			b: &Components{MediaTypes: map[string]*MediaType{
				"application/xml": {Schema: &Schema{Type: "object"}},
			}},
			want: false,
		},
		{
			name: "MediaTypes nil vs empty",
			a:    &Components{MediaTypes: nil},
			b:    &Components{MediaTypes: map[string]*MediaType{}},
			want: true,
		},
		// Extra field
		{
			name: "same Extra",
			a:    &Components{Extra: map[string]any{"x-custom": "value"}},
			b:    &Components{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &Components{Extra: map[string]any{"x-custom": "value1"}},
			b:    &Components{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		{
			name: "Extra nil vs empty",
			a:    &Components{Extra: nil},
			b:    &Components{Extra: map[string]any{}},
			want: true,
		},
		// Complete Components
		{
			name: "complete Components equal",
			a: &Components{
				Schemas: map[string]*Schema{
					"User": {Type: "object"},
				},
				Responses: map[string]*Response{
					"NotFound": {Description: "Not found"},
				},
				Parameters: map[string]*Parameter{
					"userId": {Name: "userId", In: "path"},
				},
				Examples: map[string]*Example{
					"UserExample": {Summary: "Example user"},
				},
				RequestBodies: map[string]*RequestBody{
					"UserInput": {Description: "User data"},
				},
				Headers: map[string]*Header{
					"X-Rate-Limit": {Description: "Rate limit"},
				},
				SecuritySchemes: map[string]*SecurityScheme{
					"api_key": {Type: "apiKey"},
				},
				Links: map[string]*Link{
					"GetUserById": {OperationID: "getUserById"},
				},
				Callbacks: map[string]*Callback{
					"onEvent": {},
				},
				PathItems: map[string]*PathItem{
					"SharedPath": {},
				},
				MediaTypes: map[string]*MediaType{
					"application/json": {},
				},
				Extra: map[string]any{"x-version": 1},
			},
			b: &Components{
				Schemas: map[string]*Schema{
					"User": {Type: "object"},
				},
				Responses: map[string]*Response{
					"NotFound": {Description: "Not found"},
				},
				Parameters: map[string]*Parameter{
					"userId": {Name: "userId", In: "path"},
				},
				Examples: map[string]*Example{
					"UserExample": {Summary: "Example user"},
				},
				RequestBodies: map[string]*RequestBody{
					"UserInput": {Description: "User data"},
				},
				Headers: map[string]*Header{
					"X-Rate-Limit": {Description: "Rate limit"},
				},
				SecuritySchemes: map[string]*SecurityScheme{
					"api_key": {Type: "apiKey"},
				},
				Links: map[string]*Link{
					"GetUserById": {OperationID: "getUserById"},
				},
				Callbacks: map[string]*Callback{
					"onEvent": {},
				},
				PathItems: map[string]*PathItem{
					"SharedPath": {},
				},
				MediaTypes: map[string]*MediaType{
					"application/json": {},
				},
				Extra: map[string]any{"x-version": 1},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalComponents(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
