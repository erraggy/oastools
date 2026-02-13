package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAS2DocumentMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		doc      *OAS2Document
		want     string
		wantErr  bool
		validate func(*testing.T, string)
	}{
		{
			name: "minimal document without extra",
			doc: &OAS2Document{
				Swagger: "2.0",
				Info: &Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Paths: map[string]*PathItem{},
			},
			validate: func(t *testing.T, result string) {
				// Should not contain "extra" field
				assert.NotContains(t, result, "extra")
				// Should contain required fields
				assert.Contains(t, result, `"swagger":"2.0"`)
				assert.Contains(t, result, `"info"`)
				assert.Contains(t, result, `"paths"`)
			},
		},
		{
			name: "document with all optional fields",
			doc: &OAS2Document{
				Swagger:  "2.0",
				Info:     &Info{Title: "Test", Version: "1.0"},
				Host:     "api.example.com",
				BasePath: "/v1",
				Schemes:  []string{"https", "http"},
				Consumes: []string{"application/json"},
				Produces: []string{"application/json"},
				Paths:    map[string]*PathItem{},
				Definitions: map[string]*Schema{
					"User": {Type: "object"},
				},
				Parameters: map[string]*Parameter{
					"limit": {Name: "limit", In: "query", Type: "integer"},
				},
				Responses: map[string]*Response{
					"404": {Description: "Not found"},
				},
				SecurityDefinitions: map[string]*SecurityScheme{
					"api_key": {Type: "apiKey", Name: "api_key", In: "header"},
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
			},
			validate: func(t *testing.T, result string) {
				expectedFields := []string{
					`"host":"api.example.com"`,
					`"basePath":"/v1"`,
					`"schemes":["https","http"]`,
					`"consumes":["application/json"]`,
					`"produces":["application/json"]`,
					`"definitions"`,
					`"parameters"`,
					`"responses"`,
					`"securityDefinitions"`,
					`"security"`,
					`"tags"`,
					`"externalDocs"`,
				}
				for _, field := range expectedFields {
					assert.Contains(t, result, field)
				}
			},
		},
		{
			name: "document with extra fields",
			doc: &OAS2Document{
				Swagger: "2.0",
				Info:    &Info{Title: "Test", Version: "1.0"},
				Paths:   map[string]*PathItem{},
				Extra: map[string]any{
					"x-custom":    "value",
					"x-api-id":    "12345",
					"x-extension": map[string]any{"nested": true},
				},
			},
			validate: func(t *testing.T, result string) {
				// Extra fields should be flattened to root level
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
			name: "document with empty optional slices and maps",
			doc: &OAS2Document{
				Swagger:             "2.0",
				Info:                &Info{Title: "Test", Version: "1.0"},
				Schemes:             []string{},
				Consumes:            []string{},
				Produces:            []string{},
				Paths:               map[string]*PathItem{},
				Definitions:         map[string]*Schema{},
				Parameters:          map[string]*Parameter{},
				Responses:           map[string]*Response{},
				SecurityDefinitions: map[string]*SecurityScheme{},
				Security:            []SecurityRequirement{},
				Tags:                []*Tag{},
			},
			validate: func(t *testing.T, result string) {
				// Empty slices and maps should be omitted (json:",omitempty" behavior)
				omittedFields := []string{"schemes", "consumes", "produces", "definitions", "parameters", "responses", "securityDefinitions", "security", "tags"}
				for _, field := range omittedFields {
					fieldPattern := `"` + field + `":`
					assert.NotContains(t, result, fieldPattern)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("OAS2Document.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil {
				tt.validate(t, string(got))
			}

			// Verify it's valid JSON by unmarshaling
			var check map[string]any
			assert.NoError(t, json.Unmarshal(got, &check), "Marshaled JSON is not valid")
		})
	}
}

func TestOAS2DocumentUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantErr   bool
		validate  func(*testing.T, *OAS2Document)
	}{
		{
			name: "minimal document",
			jsonInput: `{
				"swagger": "2.0",
				"info": {"title": "Test", "version": "1.0"},
				"paths": {}
			}`,
			validate: func(t *testing.T, doc *OAS2Document) {
				assert.Equal(t, "2.0", doc.Swagger)
				require.NotNil(t, doc.Info)
				assert.Equal(t, "Test", doc.Info.Title)
				assert.Empty(t, doc.Extra)
			},
		},
		{
			name: "document with specification extensions",
			jsonInput: `{
				"swagger": "2.0",
				"info": {"title": "Test", "version": "1.0"},
				"paths": {},
				"x-custom": "value",
				"x-api-id": "12345",
				"x-nested": {"field": "value"}
			}`,
			validate: func(t *testing.T, doc *OAS2Document) {
				require.NotNil(t, doc.Extra)
				assert.Len(t, doc.Extra, 3)
				assert.Equal(t, "value", doc.Extra["x-custom"])
				assert.Equal(t, "12345", doc.Extra["x-api-id"])
				assert.NotNil(t, doc.Extra["x-nested"])
			},
		},
		{
			name: "document with all standard fields",
			jsonInput: `{
				"swagger": "2.0",
				"info": {"title": "Test", "version": "1.0"},
				"host": "api.example.com",
				"basePath": "/v1",
				"schemes": ["https"],
				"consumes": ["application/json"],
				"produces": ["application/json"],
				"paths": {},
				"definitions": {"User": {"type": "object"}},
				"parameters": {"limit": {"name": "limit", "in": "query", "type": "integer"}},
				"responses": {"404": {"description": "Not found"}},
				"securityDefinitions": {"api_key": {"type": "apiKey", "name": "api_key", "in": "header"}},
				"security": [{"api_key": []}],
				"tags": [{"name": "users"}],
				"externalDocs": {"url": "https://example.com"}
			}`,
			validate: func(t *testing.T, doc *OAS2Document) {
				assert.Equal(t, "api.example.com", doc.Host)
				assert.Equal(t, "/v1", doc.BasePath)
				assert.Len(t, doc.Schemes, 1)
				assert.Equal(t, "https", doc.Schemes[0])
				assert.Len(t, doc.Definitions, 1)
				assert.Len(t, doc.Parameters, 1)
				assert.Len(t, doc.Responses, 1)
				assert.Len(t, doc.SecurityDefinitions, 1)
				assert.Len(t, doc.Security, 1)
				assert.Len(t, doc.Tags, 1)
				assert.NotNil(t, doc.ExternalDocs)
			},
		},
		{
			name: "document with non-extension unknown fields",
			jsonInput: `{
				"swagger": "2.0",
				"info": {"title": "Test", "version": "1.0"},
				"paths": {},
				"unknownField": "should be ignored",
				"anotherUnknown": 123
			}`,
			validate: func(t *testing.T, doc *OAS2Document) {
				// Non x- fields should be ignored, not captured in Extra
				assert.Empty(t, doc.Extra)
			},
		},
		{
			name:      "invalid JSON",
			jsonInput: `{invalid json}`,
			wantErr:   true,
		},
		{
			name: "x- field at start of name",
			jsonInput: `{
				"swagger": "2.0",
				"info": {"title": "Test", "version": "1.0"},
				"paths": {},
				"x-custom": "should be captured",
				"xCustom": "should not be captured"
			}`,
			validate: func(t *testing.T, doc *OAS2Document) {
				require.Len(t, doc.Extra, 1)
				assert.Contains(t, doc.Extra, "x-custom")
				assert.NotContains(t, doc.Extra, "xCustom")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc OAS2Document
			err := json.Unmarshal([]byte(tt.jsonInput), &doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("OAS2Document.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil {
				tt.validate(t, &doc)
			}
		})
	}
}

func TestOAS2DocumentMarshalUnmarshalRoundtrip(t *testing.T) {
	original := &OAS2Document{
		Swagger:  "2.0",
		Info:     &Info{Title: "Test API", Version: "1.0.0", Description: "Test"},
		Host:     "api.example.com",
		BasePath: "/v1",
		Schemes:  []string{"https", "http"},
		Consumes: []string{"application/json"},
		Produces: []string{"application/json", "application/xml"},
		Paths: map[string]*PathItem{
			"/users": {
				Get: &Operation{
					Summary:     "List users",
					OperationID: "listUsers",
				},
			},
		},
		Definitions: map[string]*Schema{
			"User": {
				Type: "object",
				Properties: map[string]*Schema{
					"id":   {Type: "integer"},
					"name": {Type: "string"},
				},
				Required: []string{"id", "name"},
			},
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
	var restored OAS2Document
	require.NoError(t, json.Unmarshal(data, &restored))

	// Verify key fields
	assert.Equal(t, original.Swagger, restored.Swagger)
	assert.Equal(t, original.Host, restored.Host)
	assert.Equal(t, original.BasePath, restored.BasePath)
	assert.Equal(t, len(original.Schemes), len(restored.Schemes))
	assert.Equal(t, len(original.Paths), len(restored.Paths))
	assert.Equal(t, len(original.Definitions), len(restored.Definitions))
	assert.Equal(t, len(original.Extra), len(restored.Extra))

	// Verify extensions
	for k, v := range original.Extra {
		assert.Equal(t, v, restored.Extra[k], "Extension %s mismatch", k)
	}
}
