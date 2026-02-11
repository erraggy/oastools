package parser

import (
	"encoding/json"
	"strings"
	"testing"
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
				if strings.Contains(result, "extra") {
					t.Error("Should not include extra field when empty")
				}
				if !strings.Contains(result, `"openapi":"3.0.3"`) {
					t.Error("Should include openapi field")
				}
				if !strings.Contains(result, `"info"`) {
					t.Error("Should include info field")
				}
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
					if !strings.Contains(result, field) {
						t.Errorf("Expected field missing: %s", field)
					}
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
					if !strings.Contains(result, ext) {
						t.Errorf("Expected extension missing: %s", ext)
					}
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
					if strings.Contains(result, fieldPattern) {
						t.Errorf("Empty field should be omitted: %s", field)
					}
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
				if !strings.Contains(result, `"webhooks"`) {
					t.Error("OAS 3.1 webhooks field missing")
				}
				if !strings.Contains(result, `"jsonSchemaDialect"`) {
					t.Error("OAS 3.1 jsonSchemaDialect field missing")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.doc)
			if err != nil {
				t.Errorf("OAS3Document.MarshalJSON() error = %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, string(got))
			}

			// Verify it's valid JSON
			var check map[string]any
			if err := json.Unmarshal(got, &check); err != nil {
				t.Errorf("Marshaled JSON is not valid: %v", err)
			}
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
				if doc.OpenAPI != "3.0.3" {
					t.Errorf("Expected openapi 3.0.3, got %s", doc.OpenAPI)
				}
				if doc.Info == nil || doc.Info.Title != "Test" {
					t.Error("Info not unmarshaled correctly")
				}
				if len(doc.Extra) > 0 {
					t.Error("Should not have Extra fields")
				}
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
				if doc.Extra == nil {
					t.Fatal("Extra should not be nil")
				}
				if len(doc.Extra) != 3 {
					t.Errorf("Expected 3 extra fields, got %d", len(doc.Extra))
				}
				if doc.Extra["x-custom"] != "value" {
					t.Error("x-custom not captured correctly")
				}
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
				if len(doc.Servers) != 1 {
					t.Error("Servers not unmarshaled correctly")
				}
				if len(doc.Paths) != 1 {
					t.Error("Paths not unmarshaled correctly")
				}
				if doc.Components == nil || len(doc.Components.Schemas) != 1 {
					t.Error("Components not unmarshaled correctly")
				}
				if len(doc.Security) != 1 {
					t.Error("Security not unmarshaled correctly")
				}
				if len(doc.Tags) != 1 {
					t.Error("Tags not unmarshaled correctly")
				}
				if doc.ExternalDocs == nil {
					t.Error("ExternalDocs not unmarshaled correctly")
				}
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
				if doc.OpenAPI != "3.1.0" {
					t.Errorf("Expected openapi 3.1.0, got %s", doc.OpenAPI)
				}
				if len(doc.Webhooks) != 1 {
					t.Error("Webhooks not unmarshaled correctly")
				}
				if doc.JSONSchemaDialect != "https://json-schema.org/draft/2020-12/schema" {
					t.Error("JSONSchemaDialect not unmarshaled correctly")
				}
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
				if len(doc.Extra) > 0 {
					t.Errorf("Non x- fields should not be in Extra, got: %v", doc.Extra)
				}
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
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var restored OAS3Document
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify key fields
	if restored.OpenAPI != original.OpenAPI {
		t.Errorf("OpenAPI mismatch: got %s, want %s", restored.OpenAPI, original.OpenAPI)
	}
	if len(restored.Servers) != len(original.Servers) {
		t.Errorf("Servers length mismatch: got %d, want %d", len(restored.Servers), len(original.Servers))
	}
	if len(restored.Paths) != len(original.Paths) {
		t.Errorf("Paths length mismatch: got %d, want %d", len(restored.Paths), len(original.Paths))
	}
	if len(restored.Extra) != len(original.Extra) {
		t.Errorf("Extra length mismatch: got %d, want %d", len(restored.Extra), len(original.Extra))
	}

	// Verify extensions
	for k, v := range original.Extra {
		if restored.Extra[k] != v {
			t.Errorf("Extension %s mismatch: got %v, want %v", k, restored.Extra[k], v)
		}
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
				if !strings.Contains(result, `"schemas"`) {
					t.Error("Should include schemas field")
				}
				if !strings.Contains(result, `"responses"`) {
					t.Error("Should include responses field")
				}
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
				if !strings.Contains(result, `"x-custom":"value"`) {
					t.Error("Extra field should be included")
				}
			},
		},
		{
			name: "empty components",
			comp: &Components{},
			validate: func(t *testing.T, result string) {
				// Should produce valid empty object
				if result != "{}" {
					t.Errorf("Empty components should marshal to {}, got %s", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.comp)
			if err != nil {
				t.Errorf("Components.MarshalJSON() error = %v", err)
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
				if len(comp.Schemas) != 1 {
					t.Error("Schemas not unmarshaled correctly")
				}
				if len(comp.Responses) != 1 {
					t.Error("Responses not unmarshaled correctly")
				}
			},
		},
		{
			name: "components with extensions",
			jsonInput: `{
				"schemas": {"User": {"type": "object"}},
				"x-custom": "value"
			}`,
			validate: func(t *testing.T, comp *Components) {
				if comp.Extra == nil || len(comp.Extra) != 1 {
					t.Error("Extensions not captured correctly")
				}
				if comp.Extra["x-custom"] != "value" {
					t.Error("x-custom not captured correctly")
				}
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
