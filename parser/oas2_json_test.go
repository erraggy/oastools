package parser

import (
	"encoding/json"
	"strings"
	"testing"
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
				if strings.Contains(result, "extra") {
					t.Error("Should not include extra field when empty")
				}
				// Should contain required fields
				if !strings.Contains(result, `"swagger":"2.0"`) {
					t.Error("Should include swagger field")
				}
				if !strings.Contains(result, `"info"`) {
					t.Error("Should include info field")
				}
				if !strings.Contains(result, `"paths"`) {
					t.Error("Should include paths field")
				}
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
					if !strings.Contains(result, field) {
						t.Errorf("Expected field missing: %s", field)
					}
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
					if !strings.Contains(result, ext) {
						t.Errorf("Expected extension missing: %s", ext)
					}
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
					if strings.Contains(result, fieldPattern) {
						t.Errorf("Empty field should be omitted: %s", field)
					}
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
			if err := json.Unmarshal(got, &check); err != nil {
				t.Errorf("Marshaled JSON is not valid: %v", err)
			}
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
				if doc.Swagger != "2.0" {
					t.Errorf("Expected swagger 2.0, got %s", doc.Swagger)
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
				"swagger": "2.0",
				"info": {"title": "Test", "version": "1.0"},
				"paths": {},
				"x-custom": "value",
				"x-api-id": "12345",
				"x-nested": {"field": "value"}
			}`,
			validate: func(t *testing.T, doc *OAS2Document) {
				if doc.Extra == nil {
					t.Fatal("Extra should not be nil")
				}
				if len(doc.Extra) != 3 {
					t.Errorf("Expected 3 extra fields, got %d", len(doc.Extra))
				}
				if doc.Extra["x-custom"] != "value" {
					t.Error("x-custom not captured correctly")
				}
				if doc.Extra["x-api-id"] != "12345" {
					t.Error("x-api-id not captured correctly")
				}
				if doc.Extra["x-nested"] == nil {
					t.Error("x-nested not captured correctly")
				}
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
				if doc.Host != "api.example.com" {
					t.Errorf("Host not unmarshaled correctly: %s", doc.Host)
				}
				if doc.BasePath != "/v1" {
					t.Errorf("BasePath not unmarshaled correctly: %s", doc.BasePath)
				}
				if len(doc.Schemes) != 1 || doc.Schemes[0] != "https" {
					t.Error("Schemes not unmarshaled correctly")
				}
				if len(doc.Definitions) != 1 {
					t.Error("Definitions not unmarshaled correctly")
				}
				if len(doc.Parameters) != 1 {
					t.Error("Parameters not unmarshaled correctly")
				}
				if len(doc.Responses) != 1 {
					t.Error("Responses not unmarshaled correctly")
				}
				if len(doc.SecurityDefinitions) != 1 {
					t.Error("SecurityDefinitions not unmarshaled correctly")
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
				if doc.Extra == nil || len(doc.Extra) != 1 {
					t.Fatal("Should have exactly one extension field")
				}
				if _, ok := doc.Extra["x-custom"]; !ok {
					t.Error("x-custom should be captured")
				}
				if _, ok := doc.Extra["xCustom"]; ok {
					t.Error("xCustom (without dash) should not be captured")
				}
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
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var restored OAS2Document
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify key fields
	if restored.Swagger != original.Swagger {
		t.Errorf("Swagger mismatch: got %s, want %s", restored.Swagger, original.Swagger)
	}
	if restored.Host != original.Host {
		t.Errorf("Host mismatch: got %s, want %s", restored.Host, original.Host)
	}
	if restored.BasePath != original.BasePath {
		t.Errorf("BasePath mismatch: got %s, want %s", restored.BasePath, original.BasePath)
	}
	if len(restored.Schemes) != len(original.Schemes) {
		t.Errorf("Schemes length mismatch: got %d, want %d", len(restored.Schemes), len(original.Schemes))
	}
	if len(restored.Paths) != len(original.Paths) {
		t.Errorf("Paths length mismatch: got %d, want %d", len(restored.Paths), len(original.Paths))
	}
	if len(restored.Definitions) != len(original.Definitions) {
		t.Errorf("Definitions length mismatch: got %d, want %d", len(restored.Definitions), len(original.Definitions))
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
