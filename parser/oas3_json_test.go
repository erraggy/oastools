package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAS3DocumentMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		doc      *OAS3Document
		validate func(*testing.T, string)
	}{
		{
			name: "minimal document without extra",
			doc: &OAS3Document{
				OpenAPI: "3.0.3",
				Info: &Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Paths: map[string]*PathItem{},
			},
			validate: func(t *testing.T, result string) {
				assert.NotContains(t, result, "extra")
				assert.Contains(t, result, `"openapi":"3.0.3"`)
				assert.Contains(t, result, `"info"`)
			},
		},
		{
			name: "document with all optional fields",
			doc: &OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &Info{Title: "Test", Version: "1.0"},
				Servers: []*Server{
					{URL: "https://api.example.com"},
					{URL: "https://staging.example.com"},
				},
				Paths: map[string]*PathItem{
					"/users": {
						Get: &Operation{Summary: "List users"},
					},
				},
				Webhooks: map[string]*PathItem{
					"newUser": {
						Post: &Operation{Summary: "New user webhook"},
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
				ExternalDocs: &ExternalDocs{
					Description: "More info",
					URL:         "https://example.com/docs",
				},
				JSONSchemaDialect: "https://json-schema.org/draft/2020-12/schema",
			},
			validate: func(t *testing.T, result string) {
				expectedFields := []string{
					`"servers"`,
					`"paths"`,
					`"webhooks"`,
					`"components"`,
					`"security"`,
					`"tags"`,
					`"externalDocs"`,
					`"jsonSchemaDialect"`,
				}
				for _, field := range expectedFields {
					assert.Contains(t, result, field)
				}
			},
		},
		{
			name: "document with extra fields",
			doc: &OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &Info{Title: "Test", Version: "1.0"},
				Paths:   map[string]*PathItem{},
				Extra: map[string]any{
					"x-custom":    "value",
					"x-api-id":    "12345",
					"x-extension": map[string]any{"nested": true},
				},
			},
			validate: func(t *testing.T, result string) {
				expectedExtensions := []string{
					`"x-custom":"value"`,
					`"x-api-id":"12345"`,
					`"x-extension"`,
				}
				for _, ext := range expectedExtensions {
					assert.Contains(t, result, ext)
				}
			},
		},
		{
			name: "document with empty optional fields",
			doc: &OAS3Document{
				OpenAPI:           "3.0.3",
				Info:              &Info{Title: "Test", Version: "1.0"},
				Servers:           []*Server{},
				Paths:             map[string]*PathItem{},
				Webhooks:          map[string]*PathItem{},
				Security:          []SecurityRequirement{},
				Tags:              []*Tag{},
				JSONSchemaDialect: "",
			},
			validate: func(t *testing.T, result string) {
				// Empty slices/maps/strings should be omitted
				omittedFields := []string{"servers", "webhooks", "security", "tags", "jsonSchemaDialect"}
				for _, field := range omittedFields {
					fieldPattern := `"` + field + `":`
					assert.NotContains(t, result, fieldPattern)
				}
			},
		},
		{
			name: "OAS 3.1 specific fields",
			doc: &OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &Info{Title: "Test", Version: "1.0"},
				Paths:   map[string]*PathItem{},
				Webhooks: map[string]*PathItem{
					"userCreated": {
						Post: &Operation{Summary: "User created"},
					},
				},
				JSONSchemaDialect: "https://json-schema.org/draft/2020-12/schema",
			},
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, `"webhooks"`)
				assert.Contains(t, result, `"jsonSchemaDialect"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.doc)
			assert.NoError(t, err)
			if err != nil {
				return
			}

			if tt.validate != nil {
				tt.validate(t, string(got))
			}

			// Verify it's valid JSON
			var check map[string]any
			assert.NoError(t, json.Unmarshal(got, &check), "Marshaled JSON is not valid")
		})
	}
}

func TestOAS3DocumentUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantErr   bool
		validate  func(*testing.T, *OAS3Document)
	}{
		{
			name: "minimal document",
			jsonInput: `{
				"openapi": "3.0.3",
				"info": {"title": "Test", "version": "1.0"},
				"paths": {}
			}`,
			validate: func(t *testing.T, doc *OAS3Document) {
				assert.Equal(t, "3.0.3", doc.OpenAPI)
				require.NotNil(t, doc.Info)
				assert.Equal(t, "Test", doc.Info.Title)
				assert.Empty(t, doc.Extra)
			},
		},
		{
			name: "document with specification extensions",
			jsonInput: `{
				"openapi": "3.0.3",
				"info": {"title": "Test", "version": "1.0"},
				"paths": {},
				"x-custom": "value",
				"x-api-id": "12345",
				"x-nested": {"field": "value"}
			}`,
			validate: func(t *testing.T, doc *OAS3Document) {
				require.NotNil(t, doc.Extra)
				require.Len(t, doc.Extra, 3)
				assert.Equal(t, "value", doc.Extra["x-custom"])
			},
		},
		{
			name: "document with all standard fields",
			jsonInput: `{
				"openapi": "3.0.3",
				"info": {"title": "Test", "version": "1.0"},
				"servers": [{"url": "https://api.example.com"}],
				"paths": {"/users": {"get": {"summary": "List users"}}},
				"components": {"schemas": {"User": {"type": "object"}}},
				"security": [{"api_key": []}],
				"tags": [{"name": "users"}],
				"externalDocs": {"url": "https://example.com"}
			}`,
			validate: func(t *testing.T, doc *OAS3Document) {
				assert.Len(t, doc.Servers, 1)
				assert.Len(t, doc.Paths, 1)
				require.NotNil(t, doc.Components)
				assert.Len(t, doc.Components.Schemas, 1)
				assert.Len(t, doc.Security, 1)
				assert.Len(t, doc.Tags, 1)
				assert.NotNil(t, doc.ExternalDocs)
			},
		},
		{
			name: "OAS 3.1 specific fields",
			jsonInput: `{
				"openapi": "3.1.0",
				"info": {"title": "Test", "version": "1.0"},
				"paths": {},
				"webhooks": {"newUser": {"post": {"summary": "New user"}}},
				"jsonSchemaDialect": "https://json-schema.org/draft/2020-12/schema"
			}`,
			validate: func(t *testing.T, doc *OAS3Document) {
				assert.Equal(t, "3.1.0", doc.OpenAPI)
				assert.Len(t, doc.Webhooks, 1)
				assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", doc.JSONSchemaDialect)
			},
		},
		{
			name: "document with non-extension unknown fields",
			jsonInput: `{
				"openapi": "3.0.3",
				"info": {"title": "Test", "version": "1.0"},
				"paths": {},
				"unknownField": "should be ignored"
			}`,
			validate: func(t *testing.T, doc *OAS3Document) {
				assert.Empty(t, doc.Extra)
			},
		},
		{
			name:      "invalid JSON",
			jsonInput: `{invalid json}`,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc OAS3Document
			err := json.Unmarshal([]byte(tt.jsonInput), &doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("OAS3Document.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil {
				tt.validate(t, &doc)
			}
		})
	}
}

func TestOAS3DocumentMarshalUnmarshalRoundtrip(t *testing.T) {
	original := &OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &Info{Title: "Test API", Version: "1.0.0", Description: "Test"},
		Servers: []*Server{
			{URL: "https://api.example.com", Description: "Production"},
			{URL: "https://staging.example.com", Description: "Staging"},
		},
		Paths: map[string]*PathItem{
			"/users": {
				Get: &Operation{
					Summary:     "List users",
					OperationID: "listUsers",
				},
			},
		},
		Components: &Components{
			Schemas: map[string]*Schema{
				"User": {
					Type: "object",
					Properties: map[string]*Schema{
						"id":   {Type: "integer"},
						"name": {Type: "string"},
					},
					Required: []string{"id", "name"},
				},
			},
		},
		Security: []SecurityRequirement{
			{"api_key": []string{}},
		},
		Tags: []*Tag{
			{Name: "users", Description: "User operations"},
		},
		ExternalDocs: &ExternalDocs{
			Description: "More info",
			URL:         "https://example.com/docs",
		},
		Extra: map[string]any{
			"x-custom-field": "custom-value",
			"x-api-id":       "api-123",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back
	var restored OAS3Document
	require.NoError(t, json.Unmarshal(data, &restored))

	// Verify key fields
	assert.Equal(t, original.OpenAPI, restored.OpenAPI)
	assert.Equal(t, len(original.Servers), len(restored.Servers))
	assert.Equal(t, len(original.Paths), len(restored.Paths))
	assert.Equal(t, len(original.Extra), len(restored.Extra))

	// Verify extensions
	for k, v := range original.Extra {
		assert.Equal(t, v, restored.Extra[k], "Extension %s mismatch", k)
	}
}

func TestComponentsMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		comp     *Components
		validate func(*testing.T, string)
	}{
		{
			name: "components without extra",
			comp: &Components{
				Schemas: map[string]*Schema{
					"User": {Type: "object"},
				},
				Responses: map[string]*Response{
					"NotFound": {Description: "Not found"},
				},
			},
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, `"schemas"`)
				assert.Contains(t, result, `"responses"`)
			},
		},
		{
			name: "components with extra",
			comp: &Components{
				Schemas: map[string]*Schema{
					"User": {Type: "object"},
				},
				Extra: map[string]any{
					"x-custom": "value",
				},
			},
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, `"x-custom":"value"`)
			},
		},
		{
			name: "empty components",
			comp: &Components{},
			validate: func(t *testing.T, result string) {
				// Should produce valid empty object
				assert.Equal(t, "{}", result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.comp)
			assert.NoError(t, err)
			if err != nil {
				return
			}

			if tt.validate != nil {
				tt.validate(t, string(got))
			}
		})
	}
}

func TestComponentsUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantErr   bool
		validate  func(*testing.T, *Components)
	}{
		{
			name: "components with schemas",
			jsonInput: `{
				"schemas": {"User": {"type": "object"}},
				"responses": {"NotFound": {"description": "Not found"}}
			}`,
			validate: func(t *testing.T, comp *Components) {
				assert.Len(t, comp.Schemas, 1)
				assert.Len(t, comp.Responses, 1)
			},
		},
		{
			name: "components with extensions",
			jsonInput: `{
				"schemas": {"User": {"type": "object"}},
				"x-custom": "value"
			}`,
			validate: func(t *testing.T, comp *Components) {
				require.Len(t, comp.Extra, 1)
				assert.Equal(t, "value", comp.Extra["x-custom"])
			},
		},
		{
			name:      "invalid JSON",
			jsonInput: `{invalid}`,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var comp Components
			err := json.Unmarshal([]byte(tt.jsonInput), &comp)
			if (err != nil) != tt.wantErr {
				t.Errorf("Components.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil {
				tt.validate(t, &comp)
			}
		})
	}
}
