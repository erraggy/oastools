package validator

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========================================
// Enum Validation Edge Case Tests
// ========================================

// TestValidateEnumEdgeCases tests enum validation for various edge cases
// including type mismatches, null enums (OAS 3.1+), and array-type schemas.
func TestValidateEnumEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		schemaType    string
		enumValues    []any
		expectError   bool
		errorContains string
	}{
		{
			name:        "string enum - valid",
			schemaType:  "string",
			enumValues:  []any{"red", "green", "blue"},
			expectError: false,
		},
		{
			name:          "string enum - integer value",
			schemaType:    "string",
			enumValues:    []any{"red", 42, "blue"},
			expectError:   true,
			errorContains: "must be a string",
		},
		{
			name:        "integer enum - valid",
			schemaType:  "integer",
			enumValues:  []any{1, 2, 3},
			expectError: false,
		},
		{
			name:        "integer enum - float64 whole number (valid from JSON parsing)",
			schemaType:  "integer",
			enumValues:  []any{float64(1), float64(2), float64(3)},
			expectError: false,
		},
		{
			name:          "integer enum - float with decimal",
			schemaType:    "integer",
			enumValues:    []any{1, 2.5, 3},
			expectError:   true,
			errorContains: "must be an integer",
		},
		{
			name:          "integer enum - string value",
			schemaType:    "integer",
			enumValues:    []any{1, "two", 3},
			expectError:   true,
			errorContains: "must be an integer",
		},
		{
			name:        "number enum - valid integers",
			schemaType:  "number",
			enumValues:  []any{1, 2, 3},
			expectError: false,
		},
		{
			name:        "number enum - valid floats",
			schemaType:  "number",
			enumValues:  []any{1.5, 2.5, 3.5},
			expectError: false,
		},
		{
			name:        "number enum - mixed int and float",
			schemaType:  "number",
			enumValues:  []any{1, 2.5, 3},
			expectError: false,
		},
		{
			name:          "number enum - string value",
			schemaType:    "number",
			enumValues:    []any{1.5, "not a number", 3.5},
			expectError:   true,
			errorContains: "must be a number",
		},
		{
			name:        "boolean enum - valid",
			schemaType:  "boolean",
			enumValues:  []any{true, false},
			expectError: false,
		},
		{
			name:          "boolean enum - string value",
			schemaType:    "boolean",
			enumValues:    []any{true, "false"},
			expectError:   true,
			errorContains: "must be a boolean",
		},
		{
			name:        "null enum - valid nil (OAS 3.1+)",
			schemaType:  "null",
			enumValues:  []any{nil},
			expectError: false,
		},
		{
			name:          "null enum - non-nil value",
			schemaType:    "null",
			enumValues:    []any{"not null"},
			expectError:   true,
			errorContains: "must be null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info: &parser.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Paths: make(map[string]*parser.PathItem),
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestEnum": {
							Type: tt.schemaType,
							Enum: tt.enumValues,
						},
					},
				},
			}

			parseResult := &parser.ParseResult{
				Version:    "3.1.0",
				OASVersion: parser.OASVersion310,
				Document:   doc,
			}

			v := New()
			result, err := v.ValidateParsed(*parseResult)
			require.NoError(t, err)

			if tt.expectError {
				// Find the enum-related error
				foundEnumError := false
				for _, e := range result.Errors {
					if strings.Contains(e.Path, "enum") && strings.Contains(e.Message, tt.errorContains) {
						foundEnumError = true
						break
					}
				}
				assert.True(t, foundEnumError, "Expected enum validation error containing %q, got errors: %v", tt.errorContains, result.Errors)
			} else {
				// Should not have enum-related errors
				for _, e := range result.Errors {
					if strings.Contains(e.Path, "enum") {
						t.Errorf("Unexpected enum error: %s", e.String())
					}
				}
			}
		})
	}
}

// TestValidateEnumWithArrayType tests that array-type enums work correctly.
// In array schemas, the enum applies to individual array items.
func TestValidateEnumWithArrayType(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]*parser.PathItem),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"ColorArray": {
					Type: "array",
					Items: &parser.Schema{
						Type: "string",
						Enum: []any{"red", "green", "blue"},
					},
				},
			},
		},
	}

	parseResult := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	// Array with enum items should not produce errors
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "ColorArray") {
			t.Errorf("Unexpected error for ColorArray: %s", e.String())
		}
	}
}

// TestValidateEnumEmptyArray tests that empty enum arrays are handled.
func TestValidateEnumEmptyArray(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]*parser.PathItem),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"EmptyEnum": {
					Type: "string",
					Enum: []any{}, // Empty enum array
				},
			},
		},
	}

	parseResult := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	// Empty enum should not cause validation errors in the enum validator itself
	// (it may cause other warnings about being pointless, but not type errors)
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "EmptyEnum") && strings.Contains(e.Message, "must be") {
			t.Errorf("Unexpected type error for empty enum: %s", e.String())
		}
	}
}

// ========================================
// Schema Name Validation Tests
// ========================================

// TestValidate_EmptySchemaName_OAS2 tests that empty schema names in definitions report an error
func TestValidate_EmptySchemaName_OAS2(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]*parser.PathItem{},
		Definitions: map[string]*parser.Schema{
			"": { // Empty schema name
				Type: "object",
			},
		},
	}

	parseResult := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	assert.False(t, result.Valid, "Document should be invalid with empty schema name")
	assert.NotEmpty(t, result.Errors, "Should have validation errors")

	// Check for empty schema name error
	foundError := false
	for _, e := range result.Errors {
		if e.Path == "definitions" && strings.Contains(e.Message, "schema name cannot be empty") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about empty schema name")
}

// TestValidate_WhitespaceSchemaName_OAS2 tests that whitespace-only schema names report an error
func TestValidate_WhitespaceSchemaName_OAS2(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]*parser.PathItem{},
		Definitions: map[string]*parser.Schema{
			"   ": { // Whitespace-only schema name
				Type: "object",
			},
		},
	}

	parseResult := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	assert.False(t, result.Valid, "Document should be invalid with whitespace-only schema name")
	assert.NotEmpty(t, result.Errors, "Should have validation errors")

	// Check for whitespace schema name error
	foundError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "definitions") && strings.Contains(e.Message, "whitespace-only") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about whitespace-only schema name")
}

// TestValidate_EmptySchemaName_OAS3 tests that empty schema names in components.schemas report an error
func TestValidate_EmptySchemaName_OAS3(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]*parser.PathItem{},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"": { // Empty schema name
					Type: "object",
				},
			},
		},
	}

	parseResult := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	assert.False(t, result.Valid, "Document should be invalid with empty schema name")
	assert.NotEmpty(t, result.Errors, "Should have validation errors")

	// Check for empty schema name error
	foundError := false
	for _, e := range result.Errors {
		if e.Path == "components.schemas" && strings.Contains(e.Message, "schema name cannot be empty") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about empty schema name")
}

// TestValidate_WhitespaceSchemaName_OAS3 tests that whitespace-only schema names report an error
func TestValidate_WhitespaceSchemaName_OAS3(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]*parser.PathItem{},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"\t": { // Whitespace-only schema name (tab)
					Type: "object",
				},
			},
		},
	}

	parseResult := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	assert.False(t, result.Valid, "Document should be invalid with whitespace-only schema name")
	assert.NotEmpty(t, result.Errors, "Should have validation errors")

	// Check for whitespace schema name error
	foundError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "components.schemas") && strings.Contains(e.Message, "whitespace-only") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about whitespace-only schema name")
}

// TestValidate_ValidSchemaNames tests that valid schema names pass validation
func TestValidate_ValidSchemaNames(t *testing.T) {
	tests := []struct {
		name       string
		schemaName string
	}{
		{"simple", "Pet"},
		{"with underscore", "Pet_Model"},
		{"with numbers", "Pet123"},
		{"camelCase", "petModel"},
		{"snake_case", "pet_model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info: &parser.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Paths: map[string]*parser.PathItem{},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						tt.schemaName: {
							Type: "object",
						},
					},
				},
			}

			parseResult := &parser.ParseResult{
				Version:    "3.0.3",
				OASVersion: parser.OASVersion303,
				Document:   doc,
			}

			v := New()
			result, err := v.ValidateParsed(*parseResult)
			require.NoError(t, err)

			// Should not have schema name errors
			for _, e := range result.Errors {
				if strings.Contains(e.Message, "schema name") {
					t.Errorf("Unexpected schema name error for %q: %s", tt.schemaName, e.Message)
				}
			}
		})
	}
}
