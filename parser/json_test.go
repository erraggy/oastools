package parser

import (
	"encoding/json"
	"fmt"
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

func TestEmptyExtraFields(t *testing.T) {
	// Test that structs with nil/empty Extra fields marshal correctly
	doc := &OAS3Document{
		OpenAPI: "3.2.0",
		Info: &Info{
			Title:   "Test API",
			Version: "1.0.0",
			Extra:   nil, // Explicitly nil
		},
		Paths: Paths{},
		Extra: map[string]interface{}{}, // Explicitly empty
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal document with empty Extra: %v", err)
	}

	jsonStr := string(data)

	// Should not contain "Extra" or "extra" key
	if strings.Contains(jsonStr, `"Extra"`) || strings.Contains(jsonStr, `"extra"`) {
		t.Errorf("Empty Extra field should not appear in JSON output: %s", jsonStr)
	}

	// Verify round-trip
	var restored OAS3Document
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if restored.OpenAPI != doc.OpenAPI {
		t.Errorf("OpenAPI field mismatch after round-trip")
	}
}

func TestNilPointerHandling(t *testing.T) {
	// Test marshaling structs with nil pointers
	doc := &OAS3Document{
		OpenAPI: "3.2.0",
		Info: &Info{
			Title:   "Test API",
			Version: "1.0.0",
			Contact: nil, // Nil pointer
			License: nil, // Nil pointer
		},
		Servers:      nil,
		Paths:        Paths{},
		Components:   nil,
		ExternalDocs: nil,
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal document with nil pointers: %v", err)
	}

	// Verify fields are omitted
	jsonStr := string(data)
	if strings.Contains(jsonStr, `"contact"`) {
		t.Errorf("Nil contact should be omitted from JSON")
	}
	if strings.Contains(jsonStr, `"license"`) {
		t.Errorf("Nil license should be omitted from JSON")
	}

	// Verify round-trip
	var restored OAS3Document
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
}

func TestExtraFieldConflicts(t *testing.T) {
	// Test what happens when Extra contains keys that conflict with real fields
	doc := &OAS3Document{
		OpenAPI: "3.2.0",
		Info: &Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: Paths{},
		Extra: map[string]interface{}{
			"openapi": "2.0",     // Conflicts with real field
			"info":    "invalid", // Conflicts with real field
			"x-safe":  "value",   // Safe extension
		},
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal document with conflicting Extra: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// The Extra fields should overwrite the real fields in the JSON output
	// (This is the current behavior - Extra is merged last)
	if result["openapi"] != "2.0" {
		t.Errorf("Expected Extra to overwrite openapi field, got %v", result["openapi"])
	}

	// Verify x-safe extension is present
	if result["x-safe"] != "value" {
		t.Errorf("Expected x-safe extension to be present")
	}
}

func TestInvalidStatusCodeInJSON(t *testing.T) {
	// Test that invalid status codes in JSON cause unmarshal errors
	invalidJSON := `{
		"200": {"description": "OK"},
		"999": {"description": "Invalid"},
		"default": {"description": "Default"}
	}`

	var responses Responses
	err := json.Unmarshal([]byte(invalidJSON), &responses)
	if err == nil {
		t.Errorf("Expected error for invalid status code '999', but got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid status code") {
		t.Errorf("Expected error about invalid status code, got: %v", err)
	}
}

func TestInvalidStatusCodePatternInJSON(t *testing.T) {
	// Test various invalid status code patterns
	testCases := []struct {
		name string
		json string
	}{
		{
			name: "Too low status code",
			json: `{"99": {"description": "Too low"}}`,
		},
		{
			name: "Too high status code",
			json: `{"600": {"description": "Too high"}}`,
		},
		{
			name: "Invalid wildcard",
			json: `{"6XX": {"description": "Invalid wildcard"}}`,
		},
		{
			name: "All wildcards",
			json: `{"XXX": {"description": "All wildcards"}}`,
		},
		{
			name: "Non-numeric",
			json: `{"abc": {"description": "Non-numeric"}}`,
		},
		{
			name: "Extension without x- prefix",
			json: `{"custom": {"description": "Invalid extension"}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var responses Responses
			err := json.Unmarshal([]byte(tc.json), &responses)
			if err == nil {
				t.Errorf("Expected error for invalid status code pattern, but got nil")
			}
		})
	}
}

func TestValidExtensionFieldsInResponses(t *testing.T) {
	// Test that valid extension fields in Responses work correctly
	validJSON := `{
		"200": {"description": "OK"},
		"x-custom": {"description": "Custom extension"},
		"x-rate-limit": {"description": "Rate limit info"},
		"default": {"description": "Default"}
	}`

	var responses Responses
	err := json.Unmarshal([]byte(validJSON), &responses)
	if err != nil {
		t.Fatalf("Unexpected error unmarshaling valid extensions: %v", err)
	}

	// Verify standard codes are present
	if responses.Codes["200"] == nil {
		t.Errorf("Expected 200 response to be present")
	}

	// Verify extension fields are captured
	if responses.Codes["x-custom"] == nil {
		t.Errorf("Expected x-custom extension to be captured")
	}
	if responses.Codes["x-rate-limit"] == nil {
		t.Errorf("Expected x-rate-limit extension to be captured")
	}

	// Verify default is present
	if responses.Default == nil {
		t.Errorf("Expected default response to be present")
	}
}

func TestMarshalUnmarshalErrors(t *testing.T) {
	// Test that malformed JSON causes unmarshal errors
	testCases := []struct {
		name    string
		jsonStr string
	}{
		{
			name:    "Invalid JSON syntax",
			jsonStr: `{"openapi": "3.2.0"`, // Missing closing brace
		},
		{
			name:    "Wrong type for field",
			jsonStr: `{"openapi": 123}`, // openapi should be string
		},
		{
			name:    "Invalid nested object",
			jsonStr: `{"openapi": "3.2.0", "info": "not an object"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var doc OAS3Document
			err := json.Unmarshal([]byte(tc.jsonStr), &doc)
			if err == nil {
				t.Errorf("Expected unmarshal error for malformed JSON, but got nil")
			}
		})
	}
}

func TestLargeExtraMap(t *testing.T) {
	// Test performance with many extension fields
	largeExtra := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeExtra[fmt.Sprintf("x-field-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	doc := &OAS3Document{
		OpenAPI: "3.2.0",
		Info: &Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: Paths{},
		Extra: largeExtra,
	}

	// Should marshal without error
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal document with large Extra: %v", err)
	}

	// Should unmarshal without error
	var restored OAS3Document
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal document with large Extra: %v", err)
	}

	// Verify all fields were preserved
	if len(restored.Extra) != 100 {
		t.Errorf("Expected 100 Extra fields, got %d", len(restored.Extra))
	}

	// Spot check a few fields
	if restored.Extra["x-field-0"] != "value-0" {
		t.Errorf("Extra field not preserved correctly")
	}
	if restored.Extra["x-field-99"] != "value-99" {
		t.Errorf("Extra field not preserved correctly")
	}
}
