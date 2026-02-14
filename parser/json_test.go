package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	jsonStr := string(data)

	// Check that the field is "openapi" not "OpenAPI"
	assert.Contains(t, jsonStr, `"openapi"`)
	assert.NotContains(t, jsonStr, `"OpenAPI"`)

	// Check other important fields
	assert.Contains(t, jsonStr, `"info"`)
	assert.Contains(t, jsonStr, `"paths"`)
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
	require.NoError(t, err)

	jsonStr := string(data)

	// Check that the field is "swagger" not "Swagger"
	assert.Contains(t, jsonStr, `"swagger"`)
	assert.NotContains(t, jsonStr, `"Swagger"`)
}

func TestExtraFieldsJSONInline(t *testing.T) {
	// Test that Extra fields are inlined in JSON output
	doc := &OAS3Document{
		OpenAPI: "3.2.0",
		Info: &Info{
			Title:   "Test API",
			Version: "1.0.0",
			Extra: map[string]any{
				"x-custom-field": "custom-value",
			},
		},
		Paths: Paths{},
		Extra: map[string]any{
			"x-root-extension": "root-value",
		},
	}

	data, err := json.Marshal(doc)
	require.NoError(t, err)

	jsonStr := string(data)

	// Check that extra fields are inlined (not under "Extra" or "extra" key)
	assert.False(t, strings.Contains(jsonStr, `"Extra"`) || strings.Contains(jsonStr, `"extra"`), "Extra field should not appear as a key in JSON output: %s", jsonStr)

	// Check that the custom fields are present at the root level
	assert.Contains(t, jsonStr, `"x-root-extension"`)
	assert.Contains(t, jsonStr, `"root-value"`)
}

func TestInfoExtraFieldsJSONInline(t *testing.T) {
	info := &Info{
		Title:   "Test API",
		Version: "1.0.0",
		Extra: map[string]any{
			"x-custom":  "value",
			"x-another": 123,
		},
	}

	data, err := json.Marshal(info)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))

	// Check that custom fields are at the root level
	assert.Contains(t, result, "x-custom")
	assert.Contains(t, result, "x-another")

	// Check that there's no "Extra" or "extra" field
	assert.NotContains(t, result, "Extra")
	assert.NotContains(t, result, "extra")
}

func TestJSONRoundTrip(t *testing.T) {
	// Test that we can marshal and unmarshal without losing data
	original := &OAS3Document{
		OpenAPI: "3.2.0",
		Info: &Info{
			Title:   "Test API",
			Version: "1.0.0",
			Extra: map[string]any{
				"x-info-custom": "info-value",
			},
		},
		Servers: []*Server{
			{
				URL:         "https://api.example.com",
				Description: "Production server",
				Extra: map[string]any{
					"x-server-custom": "server-value",
				},
			},
		},
		Paths: Paths{},
		Extra: map[string]any{
			"x-doc-custom": "doc-value",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back
	var restored OAS3Document
	require.NoError(t, json.Unmarshal(data, &restored))

	// Verify basic fields
	assert.Equal(t, original.OpenAPI, restored.OpenAPI)
	assert.Equal(t, original.Info.Title, restored.Info.Title)

	// Verify Extra fields were preserved
	assert.Equal(t, "doc-value", restored.Extra["x-doc-custom"])
	assert.Equal(t, "info-value", restored.Info.Extra["x-info-custom"])
	if len(restored.Servers) > 0 {
		assert.Equal(t, "server-value", restored.Servers[0].Extra["x-server-custom"])
	}
}

func TestSchemaJSONFieldCasing(t *testing.T) {
	schema := &Schema{
		Type:        "string",
		Description: "A test schema",
		Example:     "test",
		Extra: map[string]any{
			"x-custom-schema": "schema-value",
		},
	}

	data, err := json.Marshal(schema)
	require.NoError(t, err)

	jsonStr := string(data)

	// Check field casing
	assert.Contains(t, jsonStr, `"type"`)
	assert.Contains(t, jsonStr, `"description"`)

	// Check Extra field inlining
	assert.Contains(t, jsonStr, `"x-custom-schema"`)
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
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))

	// Check that status codes are at root level
	assert.Contains(t, result, "200")
	assert.Contains(t, result, "404")
	assert.Contains(t, result, "default")

	// Check that there's no "Codes" field
	assert.NotContains(t, result, "Codes")
	assert.NotContains(t, result, "codes")
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
				Extra: map[string]any{
					"x-contact-custom": "contact-value",
				},
			},
			Extra: map[string]any{
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
								Extra: map[string]any{
									"x-response-custom": "response-value",
								},
							},
						},
					},
					Extra: map[string]any{
						"x-operation-custom": "operation-value",
					},
				},
				Extra: map[string]any{
					"x-path-custom": "path-value",
				},
			},
		},
		Extra: map[string]any{
			"x-doc-custom": "doc-value",
		},
	}

	// Marshal and unmarshal
	data, err := json.Marshal(doc)
	require.NoError(t, err)

	var restored OAS3Document
	require.NoError(t, json.Unmarshal(data, &restored))

	// Verify all Extra fields at different levels
	tests := []struct {
		name  string
		got   any
		want  any
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
			assert.Equal(t, tt.want, tt.got, tt.field)
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
		Extra: map[string]any{}, // Explicitly empty
	}

	data, err := json.Marshal(doc)
	require.NoError(t, err)

	jsonStr := string(data)

	// Should not contain "Extra" or "extra" key
	assert.False(t, strings.Contains(jsonStr, `"Extra"`) || strings.Contains(jsonStr, `"extra"`), "Empty Extra field should not appear in JSON output: %s", jsonStr)

	// Verify round-trip
	var restored OAS3Document
	require.NoError(t, json.Unmarshal(data, &restored))

	assert.Equal(t, doc.OpenAPI, restored.OpenAPI)
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
	require.NoError(t, err)

	// Verify fields are omitted
	jsonStr := string(data)
	assert.NotContains(t, jsonStr, `"contact"`)
	assert.NotContains(t, jsonStr, `"license"`)

	// Verify round-trip
	var restored OAS3Document
	require.NoError(t, json.Unmarshal(data, &restored))
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
		Extra: map[string]any{
			"openapi": "2.0",     // Conflicts with real field
			"info":    "invalid", // Conflicts with real field
			"x-safe":  "value",   // Safe extension
		},
	}

	data, err := json.Marshal(doc)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))

	// The Extra fields should overwrite the real fields in the JSON output
	// (This is the current behavior - Extra is merged last)
	assert.Equal(t, "2.0", result["openapi"])

	// Verify x-safe extension is present
	assert.Equal(t, "value", result["x-safe"])
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status code")
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
			assert.Error(t, err)
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
	require.NoError(t, json.Unmarshal([]byte(validJSON), &responses))

	// Verify standard codes are present
	assert.NotNil(t, responses.Codes["200"])

	// Verify extension fields are captured
	assert.NotNil(t, responses.Codes["x-custom"])
	assert.NotNil(t, responses.Codes["x-rate-limit"])

	// Verify default is present
	assert.NotNil(t, responses.Default)
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
			assert.Error(t, err)
		})
	}
}

func TestLargeExtraMap(t *testing.T) {
	// Test performance with many extension fields
	largeExtra := make(map[string]any)
	for i := range 100 {
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
	require.NoError(t, err)

	// Should unmarshal without error
	var restored OAS3Document
	require.NoError(t, json.Unmarshal(data, &restored))

	// Verify all fields were preserved
	require.Len(t, restored.Extra, 100)

	// Spot check a few fields
	assert.Equal(t, "value-0", restored.Extra["x-field-0"])
	assert.Equal(t, "value-99", restored.Extra["x-field-99"])
}
