package parser

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOAS3DocumentJSONFieldCasing(t *testing.T) {
	doc := &OAS3Document{
		OpenAPI: "3.2.0",
		Info: &Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: Paths{
			"/test": &PathItem{
				Get: &Operation{
					Summary: "Test operation",
					Responses: &Responses{
						Codes: map[string]*Response{
							"200": {Description: "OK"},
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal OAS3Document: %v", err)
	}

	jsonStr := string(data)

	// Check that the field is "openapi" not "OpenAPI"
	if !strings.Contains(jsonStr, `"openapi"`) {
		t.Errorf("Expected 'openapi' field in JSON output, got: %s", jsonStr)
	}
	if strings.Contains(jsonStr, `"OpenAPI"`) {
		t.Errorf("Found incorrect 'OpenAPI' field (should be 'openapi') in JSON output: %s", jsonStr)
	}

	// Check other important fields
	if !strings.Contains(jsonStr, `"info"`) {
		t.Errorf("Expected 'info' field in JSON output, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"paths"`) {
		t.Errorf("Expected 'paths' field in JSON output, got: %s", jsonStr)
	}
}

func TestOAS2DocumentJSONFieldCasing(t *testing.T) {
	doc := &OAS2Document{
		Swagger: "2.0",
		Info: &Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: Paths{},
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal OAS2Document: %v", err)
	}

	jsonStr := string(data)

	// Check that the field is "swagger" not "Swagger"
	if !strings.Contains(jsonStr, `"swagger"`) {
		t.Errorf("Expected 'swagger' field in JSON output, got: %s", jsonStr)
	}
	if strings.Contains(jsonStr, `"Swagger"`) {
		t.Errorf("Found incorrect 'Swagger' field (should be 'swagger') in JSON output: %s", jsonStr)
	}
}

func TestExtraFieldsJSONInline(t *testing.T) {
	// Test that Extra fields are inlined in JSON output
	doc := &OAS3Document{
		OpenAPI: "3.2.0",
		Info: &Info{
			Title:   "Test API",
			Version: "1.0.0",
			Extra: map[string]interface{}{
				"x-custom-field": "custom-value",
			},
		},
		Paths: Paths{},
		Extra: map[string]interface{}{
			"x-root-extension": "root-value",
		},
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal OAS3Document with Extra: %v", err)
	}

	jsonStr := string(data)

	// Check that extra fields are inlined (not under "Extra" or "extra" key)
	if strings.Contains(jsonStr, `"Extra"`) || strings.Contains(jsonStr, `"extra"`) {
		t.Errorf("Extra field should not appear as a key in JSON output: %s", jsonStr)
	}

	// Check that the custom fields are present at the root level
	if !strings.Contains(jsonStr, `"x-root-extension"`) {
		t.Errorf("Expected 'x-root-extension' to be inlined in JSON output, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"root-value"`) {
		t.Errorf("Expected 'root-value' in JSON output, got: %s", jsonStr)
	}
}

func TestInfoExtraFieldsJSONInline(t *testing.T) {
	info := &Info{
		Title:   "Test API",
		Version: "1.0.0",
		Extra: map[string]interface{}{
			"x-custom":  "value",
			"x-another": 123,
		},
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal Info with Extra: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that custom fields are at the root level
	if _, ok := result["x-custom"]; !ok {
		t.Errorf("Expected 'x-custom' field at root level in JSON")
	}
	if _, ok := result["x-another"]; !ok {
		t.Errorf("Expected 'x-another' field at root level in JSON")
	}

	// Check that there's no "Extra" or "extra" field
	if _, ok := result["Extra"]; ok {
		t.Errorf("'Extra' field should not be present in JSON output")
	}
	if _, ok := result["extra"]; ok {
		t.Errorf("'extra' field should not be present in JSON output")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	// Test that we can marshal and unmarshal without losing data
	original := &OAS3Document{
		OpenAPI: "3.2.0",
		Info: &Info{
			Title:   "Test API",
			Version: "1.0.0",
			Extra: map[string]interface{}{
				"x-info-custom": "info-value",
			},
		},
		Servers: []*Server{
			{
				URL:         "https://api.example.com",
				Description: "Production server",
				Extra: map[string]interface{}{
					"x-server-custom": "server-value",
				},
			},
		},
		Paths: Paths{},
		Extra: map[string]interface{}{
			"x-doc-custom": "doc-value",
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

	// Verify basic fields
	if restored.OpenAPI != original.OpenAPI {
		t.Errorf("OpenAPI field mismatch: got %s, want %s", restored.OpenAPI, original.OpenAPI)
	}
	if restored.Info.Title != original.Info.Title {
		t.Errorf("Title mismatch: got %s, want %s", restored.Info.Title, original.Info.Title)
	}

	// Verify Extra fields were preserved
	if restored.Extra["x-doc-custom"] != "doc-value" {
		t.Errorf("Document Extra field not preserved: got %v", restored.Extra)
	}
	if restored.Info.Extra["x-info-custom"] != "info-value" {
		t.Errorf("Info Extra field not preserved: got %v", restored.Info.Extra)
	}
	if len(restored.Servers) > 0 && restored.Servers[0].Extra["x-server-custom"] != "server-value" {
		t.Errorf("Server Extra field not preserved: got %v", restored.Servers[0].Extra)
	}
}

func TestSchemaJSONFieldCasing(t *testing.T) {
	schema := &Schema{
		Type:        "string",
		Description: "A test schema",
		Example:     "test",
		Extra: map[string]interface{}{
			"x-custom-schema": "schema-value",
		},
	}

	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("Failed to marshal Schema: %v", err)
	}

	jsonStr := string(data)

	// Check field casing
	if !strings.Contains(jsonStr, `"type"`) {
		t.Errorf("Expected 'type' field in JSON output, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"description"`) {
		t.Errorf("Expected 'description' field in JSON output, got: %s", jsonStr)
	}

	// Check Extra field inlining
	if !strings.Contains(jsonStr, `"x-custom-schema"`) {
		t.Errorf("Expected 'x-custom-schema' to be inlined in JSON output, got: %s", jsonStr)
	}
}

func TestResponsesJSONMarshaling(t *testing.T) {
	// Test that Responses correctly marshals Codes inline
	responses := &Responses{
		Default: &Response{
			Description: "Default response",
		},
		Codes: map[string]*Response{
			"200": {
				Description: "Success",
			},
			"404": {
				Description: "Not found",
			},
		},
	}

	data, err := json.Marshal(responses)
	if err != nil {
		t.Fatalf("Failed to marshal Responses: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that status codes are at root level
	if _, ok := result["200"]; !ok {
		t.Errorf("Expected '200' field at root level in JSON")
	}
	if _, ok := result["404"]; !ok {
		t.Errorf("Expected '404' field at root level in JSON")
	}
	if _, ok := result["default"]; !ok {
		t.Errorf("Expected 'default' field at root level in JSON")
	}

	// Check that there's no "Codes" field
	if _, ok := result["Codes"]; ok {
		t.Errorf("'Codes' field should not be present in JSON output")
	}
	if _, ok := result["codes"]; ok {
		t.Errorf("'codes' field should not be present in JSON output")
	}
}

func TestComplexNestedExtraFields(t *testing.T) {
	// Test a complex document with Extra fields at multiple levels
	doc := &OAS3Document{
		OpenAPI: "3.2.0",
		Info: &Info{
			Title:   "Complex API",
			Version: "1.0.0",
			Contact: &Contact{
				Name:  "API Team",
				Email: "api@example.com",
				Extra: map[string]interface{}{
					"x-contact-custom": "contact-value",
				},
			},
			Extra: map[string]interface{}{
				"x-info-custom": "info-value",
			},
		},
		Paths: Paths{
			"/test": &PathItem{
				Get: &Operation{
					Summary: "Test operation",
					Responses: &Responses{
						Codes: map[string]*Response{
							"200": {
								Description: "Success",
								Extra: map[string]interface{}{
									"x-response-custom": "response-value",
								},
							},
						},
					},
					Extra: map[string]interface{}{
						"x-operation-custom": "operation-value",
					},
				},
				Extra: map[string]interface{}{
					"x-path-custom": "path-value",
				},
			},
		},
		Extra: map[string]interface{}{
			"x-doc-custom": "doc-value",
		},
	}

	// Marshal and unmarshal
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal complex document: %v", err)
	}

	var restored OAS3Document
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal complex document: %v", err)
	}

	// Verify all Extra fields at different levels
	tests := []struct {
		name  string
		got   interface{}
		want  interface{}
		field string
	}{
		{"Document Extra", restored.Extra["x-doc-custom"], "doc-value", "x-doc-custom"},
		{"Info Extra", restored.Info.Extra["x-info-custom"], "info-value", "x-info-custom"},
		{"Contact Extra", restored.Info.Contact.Extra["x-contact-custom"], "contact-value", "x-contact-custom"},
		{"PathItem Extra", restored.Paths["/test"].Extra["x-path-custom"], "path-value", "x-path-custom"},
		{"Operation Extra", restored.Paths["/test"].Get.Extra["x-operation-custom"], "operation-value", "x-operation-custom"},
		{"Response Extra", restored.Paths["/test"].Get.Responses.Codes["200"].Extra["x-response-custom"], "response-value", "x-response-custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s: got %v, want %v", tt.field, tt.got, tt.want)
			}
		})
	}
}
