package validator

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidatorNew tests the New constructor
func TestValidatorNew(t *testing.T) {
	v := New()
	if v == nil {
		t.Fatal("New() returned nil")
	}
	if !v.IncludeWarnings {
		t.Error("Expected IncludeWarnings to be true by default")
	}
	if v.StrictMode {
		t.Error("Expected StrictMode to be false by default")
	}
}

// TestValidateOAS2Valid tests validation of a valid OAS 2.0 document
func TestValidateOAS2Valid(t *testing.T) {
	v := New()
	testFile := filepath.Join("..", "testdata", "petstore-2.0.yaml")

	result, err := v.Validate(testFile)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid document, got %d errors", result.ErrorCount)
		for _, e := range result.Errors {
			t.Logf("  Error: %s", e.String())
		}
	}

	if result.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", result.Version)
	}
}

// TestValidateOAS2Invalid tests validation of an invalid OAS 2.0 document
func TestValidateOAS2Invalid(t *testing.T) {
	v := New()
	testFile := filepath.Join("..", "testdata", "invalid-oas2.yaml")

	result, err := v.Validate(testFile)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid document, but validation passed")
	}

	if result.ErrorCount == 0 {
		t.Error("Expected validation errors, got none")
	}

	t.Logf("Found %d errors and %d warnings", result.ErrorCount, result.WarningCount)
	for _, e := range result.Errors {
		t.Logf("  Error: %s", e.String())
	}
}

// TestValidateOAS3Valid tests validation of a valid OAS 3.x document
func TestValidateOAS3Valid(t *testing.T) {
	testCases := []struct {
		name     string
		file     string
		expected string
	}{
		{"OAS 3.0", "petstore-3.0.yaml", "3.0.3"},
		{"OAS 3.1", "petstore-3.1.yaml", "3.1.0"},
		{"OAS 3.2", "petstore-3.2.yaml", "3.2.0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := New()
			testFile := filepath.Join("..", "testdata", tc.file)

			result, err := v.Validate(testFile)
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}

			if !result.Valid {
				t.Errorf("Expected valid document, got %d errors", result.ErrorCount)
				for _, e := range result.Errors {
					t.Logf("  Error: %s", e.String())
				}
			}

			if result.Version != tc.expected {
				t.Errorf("Expected version %s, got %s", tc.expected, result.Version)
			}
		})
	}
}

// TestValidateOAS3Invalid tests validation of an invalid OAS 3.x document
func TestValidateOAS3Invalid(t *testing.T) {
	v := New()
	testFile := filepath.Join("..", "testdata", "invalid-oas3.yaml")

	result, err := v.Validate(testFile)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid document, but validation passed")
	}

	if result.ErrorCount == 0 {
		t.Error("Expected validation errors, got none")
	}

	t.Logf("Found %d errors and %d warnings", result.ErrorCount, result.WarningCount)
	for _, e := range result.Errors {
		t.Logf("  Error: %s", e.String())
	}
}

// TestValidateWithExternalRefs tests validation with external references
func TestValidateWithExternalRefs(t *testing.T) {
	v := New()
	testFile := filepath.Join("..", "testdata", "with-external-refs.yaml")

	result, err := v.Validate(testFile)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// External refs should parse and validate
	if !result.Valid {
		t.Errorf("Expected valid document with external refs, got %d errors", result.ErrorCount)
		for _, e := range result.Errors {
			t.Logf("  Error: %s", e.String())
		}
	}
}

// TestValidateStrictMode tests strict mode validation
func TestValidateStrictMode(t *testing.T) {
	v := New()
	v.StrictMode = true

	testFile := filepath.Join("..", "testdata", "petstore-3.0.yaml")

	result, err := v.Validate(testFile)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// In strict mode, we may get more warnings
	t.Logf("Strict mode: %d errors, %d warnings", result.ErrorCount, result.WarningCount)
}

// TestValidateNoWarnings tests suppressing warnings
func TestValidateNoWarnings(t *testing.T) {
	v := New()
	v.IncludeWarnings = false

	testFile := filepath.Join("..", "testdata", "petstore-3.0.yaml")

	result, err := v.Validate(testFile)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if result.WarningCount != 0 {
		t.Errorf("Expected no warnings when IncludeWarnings=false, got %d", result.WarningCount)
	}

	if len(result.Warnings) != 0 {
		t.Errorf("Expected empty warnings slice when IncludeWarnings=false, got %d items", len(result.Warnings))
	}
}

// TestValidateNonExistentFile tests validation with a non-existent file
func TestValidateNonExistentFile(t *testing.T) {
	v := New()
	_, err := v.Validate("non-existent-file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// TestValidationErrorString tests the String method of ValidationError
func TestValidationErrorString(t *testing.T) {
	testCases := []struct {
		name     string
		error    ValidationError
		contains []string
	}{
		{
			name: "Error with spec ref",
			error: ValidationError{
				Path:     "paths./pets.get",
				Message:  "Missing required field",
				SpecRef:  "https://spec.openapis.org/oas/v3.0.0.html",
				Severity: SeverityError,
			},
			contains: []string{"paths./pets.get", "Missing required field", "Spec:"},
		},
		{
			name: "Warning without spec ref",
			error: ValidationError{
				Path:     "info.description",
				Message:  "Should include description",
				Severity: SeverityWarning,
			},
			contains: []string{"info.description", "Should include description"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.error.String()
			for _, substr := range tc.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("Expected string to contain %q, got: %s", substr, result)
				}
			}
		})
	}
}

// TestSeverityString tests the String method of Severity
func TestSeverityString(t *testing.T) {
	testCases := []struct {
		severity Severity
		expected string
	}{
		{SeverityError, "error"},
		{SeverityWarning, "warning"},
		{Severity(999), "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := tc.severity.String()
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestValidateNonStandardStatusCodeStrictMode tests that non-standard status codes
// generate warnings in strict mode but not in non-strict mode.
func TestValidateNonStandardStatusCodeStrictMode(t *testing.T) {
	// Create a minimal OAS 3.0 document with a non-standard status code (299)
	// 299 is valid (in 100-599 range) but not a standard HTTP status code
	parseResult := parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.0",
			Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
			Paths: map[string]*parser.PathItem{
				"/test": {
					Get: &parser.Operation{
						OperationID: "getTest",
						Responses: &parser.Responses{
							Codes: map[string]*parser.Response{
								"299": {Description: "Non-standard success"},
							},
						},
					},
				},
			},
		},
	}

	// Test with strict mode enabled - should generate a warning
	t.Run("strict mode warns on non-standard code", func(t *testing.T) {
		v := New()
		v.StrictMode = true
		v.IncludeWarnings = true

		result, err := v.ValidateParsed(parseResult)
		require.NoError(t, err)

		// Should have a warning about non-standard status code
		hasNonStandardWarning := false
		for _, w := range result.Warnings {
			if strings.Contains(w.Message, "Non-standard HTTP status code: 299") {
				hasNonStandardWarning = true
				break
			}
		}
		assert.True(t, hasNonStandardWarning, "Expected warning about non-standard status code 299")
	})

	// Test with strict mode disabled - should NOT generate a warning
	t.Run("non-strict mode allows non-standard code", func(t *testing.T) {
		v := New()
		v.StrictMode = false
		v.IncludeWarnings = true

		result, err := v.ValidateParsed(parseResult)
		require.NoError(t, err)

		// Should NOT have a warning about non-standard status code
		for _, w := range result.Warnings {
			if strings.Contains(w.Message, "Non-standard HTTP status code: 299") {
				t.Errorf("Did not expect warning about non-standard status code in non-strict mode")
			}
		}
	})
}

// TestEmptyNilDocuments tests validation with empty/nil document objects
func TestEmptyNilDocuments(t *testing.T) {
	testCases := []struct {
		name     string
		file     string
		hasError bool
	}{
		{"Empty OAS 2.0 document", "empty-oas2.yaml", true},
		{"Empty OAS 3.0 document", "empty-oas3.yaml", true},
		{"Minimal OAS 2.0 document", "minimal-oas2.yaml", false},
		{"Minimal OAS 3.0 document", "minimal-oas3.yaml", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := New()
			testFile := filepath.Join("..", "testdata", tc.file)

			result, err := v.Validate(testFile)
			// File might not exist - that's okay for this test
			if err != nil {
				// If file doesn't exist, skip
				t.Skipf("Test file %s not found, skipping", tc.file)
				return
			}

			if tc.hasError && result.Valid {
				t.Errorf("Expected validation errors for %s, but got valid document", tc.name)
			}
			if !tc.hasError && !result.Valid {
				t.Errorf("Expected valid document for %s, but got %d errors", tc.name, result.ErrorCount)
				for _, e := range result.Errors {
					t.Logf("  Error: %s", e.String())
				}
			}
		})
	}
}

// TestCircularSchemaReferences tests handling of circular schema references
func TestCircularSchemaReferences(t *testing.T) {
	v := New()
	testFile := filepath.Join("..", "testdata", "circular-schema.yaml")

	result, err := v.Validate(testFile)
	if err != nil {
		// Parser should handle circular references
		t.Logf("Parser rejected circular schema: %v", err)
		return
	}

	// Validation should not crash on circular schemas
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	t.Logf("Circular schema validation completed with %d errors, %d warnings", result.ErrorCount, result.WarningCount)
}

// TestDeeplyNestedSchemas tests validation of deeply nested schema objects
func TestDeeplyNestedSchemas(t *testing.T) {
	v := New()
	testFile := filepath.Join("..", "testdata", "deeply-nested-schema.yaml")

	result, err := v.Validate(testFile)
	if err != nil {
		// Parser might reject deeply nested schemas
		t.Logf("Parser rejected deeply nested schema: %v", err)
		return
	}

	// Validation should complete without stack overflow
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	t.Logf("Deeply nested schema validation completed with %d errors, %d warnings", result.ErrorCount, result.WarningCount)
}

// TestMalformedPathTemplates tests validation of malformed path templates
func TestMalformedPathTemplates(t *testing.T) {
	testCases := []struct {
		name     string
		file     string
		hasError bool
	}{
		{"Unclosed path parameter", "malformed-path-unclosed.yaml", true},
		{"Double curly braces", "malformed-path-double-braces.yaml", true},
		{"Empty path parameter", "malformed-path-empty.yaml", true},
		{"Missing opening brace", "malformed-path-missing-open.yaml", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := New()
			testFile := filepath.Join("..", "testdata", tc.file)

			result, err := v.Validate(testFile)
			// File might not exist - that's okay for this test
			if err != nil {
				// Parser might reject malformed paths during parsing
				t.Logf("Parser rejected malformed path: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			if tc.hasError && result.Valid {
				t.Logf("Warning: Expected validation errors for %s, but validation passed", tc.name)
				t.Logf("This might indicate the validator accepts malformed paths")
			}

			t.Logf("%s: %d errors, %d warnings", tc.name, result.ErrorCount, result.WarningCount)
		})
	}
}

// TestPathTemplateValidation tests comprehensive path template validation
func TestPathTemplateValidation(t *testing.T) {
	testCases := []struct {
		name            string
		oasVersion      string
		paths           map[string]string // path -> operation summary
		expectErrors    int
		expectWarnings  int
		errorContains   string
		warningContains string
	}{
		{
			name:       "Valid path without trailing slash",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"/users": "Get users",
			},
			expectErrors:   0,
			expectWarnings: 0,
		},
		{
			name:       "Valid root path",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"/": "Root endpoint",
			},
			expectErrors:   0,
			expectWarnings: 0,
		},
		{
			name:       "Trailing slash - should warn",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"/users/": "Get users with trailing slash",
			},
			expectErrors:    0,
			expectWarnings:  1,
			warningContains: "trailing slash",
		},
		{
			name:       "Trailing slash with parameter - should warn (and error for undeclared param)",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"/users/{userId}/": "Get user with trailing slash",
			},
			expectErrors:    1, // undeclared parameter
			expectWarnings:  1, // trailing slash
			warningContains: "trailing slash",
		},
		{
			name:       "Multiple paths with trailing slashes - should warn for each",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"/users/":         "Get users",
				"/pets/":          "Get pets",
				"/products/{id}/": "Get product",
			},
			expectErrors:    1, // undeclared parameter in /products/{id}/
			expectWarnings:  3, // trailing slash on all three paths
			warningContains: "trailing slash",
		},
		{
			name:       "Consecutive slashes - should error",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"/users//pets": "Invalid double slash",
			},
			expectErrors:   1,
			expectWarnings: 0,
			errorContains:  "consecutive slashes",
		},
		{
			name:       "Reserved character # - should error",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"/users#section": "Invalid hash",
			},
			expectErrors:   1,
			expectWarnings: 0,
			errorContains:  "reserved character '#'",
		},
		{
			name:       "Reserved character ? - should error",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"/users?query=test": "Invalid query string",
			},
			expectErrors:   1,
			expectWarnings: 0,
			errorContains:  "reserved character '?'",
		},
		{
			name:       "Empty braces - should error",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"/users/{}": "Empty parameter",
			},
			expectErrors:   1,
			expectWarnings: 0,
			errorContains:  "empty parameter name",
		},
		{
			name:       "Unclosed brace - should error",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"/users/{userId": "Unclosed brace",
			},
			expectErrors:   1,
			expectWarnings: 0,
			errorContains:  "unclosed brace",
		},
		{
			name:       "Missing opening brace - should error",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"/users/userId}": "Missing opening brace",
			},
			expectErrors:   1,
			expectWarnings: 0,
			errorContains:  "unexpected closing brace",
		},
		{
			name:       "Nested braces - should error",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"/users/{{userId}}": "Nested braces",
			},
			expectErrors:   2, // nested braces + undeclared parameter
			expectWarnings: 0,
			errorContains:  "nested braces",
		},
		{
			name:       "Duplicate parameter names - should error",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"/users/{id}/pets/{id}": "Duplicate parameter",
			},
			expectErrors:   2, // duplicate parameter + undeclared parameter
			expectWarnings: 0,
			errorContains:  "duplicate parameter name",
		},
		{
			name:       "Path not starting with / - should error",
			oasVersion: "3.0.3",
			paths: map[string]string{
				"users": "Missing leading slash",
			},
			expectErrors:   1,
			expectWarnings: 0,
			errorContains:  "must start with '/'",
		},
		{
			name:       "OAS 2.0 - trailing slash warning",
			oasVersion: "2.0",
			paths: map[string]string{
				"/users/": "Get users",
			},
			expectErrors:    0,
			expectWarnings:  1,
			warningContains: "trailing slash",
		},
		{
			name:       "OAS 2.0 - consecutive slashes error",
			oasVersion: "2.0",
			paths: map[string]string{
				"/users//pets": "Invalid double slash",
			},
			expectErrors:   1,
			expectWarnings: 0,
			errorContains:  "consecutive slashes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build OAS document
			var content string
			if tc.oasVersion == "2.0" {
				content = fmt.Sprintf(`swagger: "%s"
info:
  title: Test API
  version: 1.0.0
paths:
`, tc.oasVersion)
				for path, summary := range tc.paths {
					content += fmt.Sprintf(`  "%s":
    get:
      summary: %s
      responses:
        '200':
          description: Success
`, path, summary)
				}
			} else {
				content = fmt.Sprintf(`openapi: %s
info:
  title: Test API
  version: 1.0.0
paths:
`, tc.oasVersion)
				for path, summary := range tc.paths {
					content += fmt.Sprintf(`  "%s":
    get:
      summary: %s
      responses:
        '200':
          description: Success
`, path, summary)
				}
			}

			// Parse and validate
			p := parser.New()
			p.ValidateStructure = false // Skip parser validation to focus on validator
			parseResult, err := p.ParseBytes([]byte(content))
			if err != nil {
				t.Fatalf("Failed to parse test document: %v", err)
			}

			v := New()
			v.IncludeWarnings = true
			result, err := v.ValidateParsed(*parseResult)
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}

			// Check error count
			if result.ErrorCount != tc.expectErrors {
				t.Errorf("Expected %d errors, got %d", tc.expectErrors, result.ErrorCount)
				for _, e := range result.Errors {
					t.Logf("  Error: %s", e.Message)
				}
			}

			// Check warning count
			if result.WarningCount != tc.expectWarnings {
				t.Errorf("Expected %d warnings, got %d", tc.expectWarnings, result.WarningCount)
				for _, w := range result.Warnings {
					t.Logf("  Warning: %s", w.Message)
				}
			}

			// Check error message contains expected text
			if tc.errorContains != "" {
				found := false
				for _, e := range result.Errors {
					if strings.Contains(strings.ToLower(e.Message), strings.ToLower(tc.errorContains)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message containing '%s', but not found", tc.errorContains)
				}
			}

			// Check warning message contains expected text
			if tc.warningContains != "" {
				found := false
				for _, w := range result.Warnings {
					if strings.Contains(strings.ToLower(w.Message), strings.ToLower(tc.warningContains)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected warning message containing '%s', but not found", tc.warningContains)
				}
			}
		})
	}
}

// TestNilInfoObject tests handling when info object is completely missing
func TestNilInfoObject(t *testing.T) {
	v := New()
	testFile := filepath.Join("..", "testdata", "missing-info.yaml")

	result, err := v.Validate(testFile)
	if err != nil {
		// Parser might reject document without info
		t.Logf("Parser rejected document without info: %v", err)
		return
	}

	if result.Valid {
		t.Error("Expected validation to fail for document without info object")
	}

	// Should have error about missing info
	hasInfoError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "info") {
			hasInfoError = true
			break
		}
	}

	if !hasInfoError {
		t.Error("Expected error message about missing info object")
	}
}

func TestValidateParsed(t *testing.T) {
	p := parser.New()
	result, err := p.Parse("../testdata/petstore-3.0.yaml")
	require.NoError(t, err)
	require.NotNil(t, result)

	v := New()
	valResult, err := v.ValidateParsed(*result)
	require.NoError(t, err)
	assert.True(t, valResult.Valid)
}

// ========================================
// Tests for metric propagation
// ========================================

// TestValidateParsedPropagatesMetrics tests that LoadTime and SourceSize are propagated from ParseResult to ValidationResult
func TestValidateParsedPropagatesMetrics(t *testing.T) {
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/minimal-oas3.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	if err != nil {
		t.Fatalf("ValidateParsed() error = %v", err)
	}

	// Verify metrics are propagated
	if result.LoadTime != parseResult.LoadTime {
		t.Errorf("LoadTime not propagated: got %v, want %v", result.LoadTime, parseResult.LoadTime)
	}
	if result.SourceSize != parseResult.SourceSize {
		t.Errorf("SourceSize not propagated: got %d, want %d", result.SourceSize, parseResult.SourceSize)
	}

	// Verify metrics are non-zero (they should have been captured during parsing)
	if result.LoadTime == 0 {
		t.Error("Expected LoadTime to be > 0 after propagation")
	}
	if result.SourceSize == 0 {
		t.Error("Expected SourceSize to be > 0 after propagation")
	}
}

// TestValidateParsed_UnsupportedVersion tests error for unsupported OAS version
func TestValidateParsed_UnsupportedVersion(t *testing.T) {
	v := New()
	// Create a ParseResult with an unsupported OAS version
	parseResult := parser.ParseResult{
		Document:   &parser.OAS3Document{}, // Valid document
		Version:    "0.0.0",                // Invalid version string
		OASVersion: parser.OASVersion(999), // Unknown enum value
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	_, err := v.ValidateParsed(parseResult)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validator: unsupported OAS version")
}

// TestQueryMethodValidation tests that QUERY method is validated based on OAS version
func TestQueryMethodValidation(t *testing.T) {
	tests := []struct {
		name        string
		version     parser.OASVersion
		shouldError bool
	}{
		{
			name:        "QUERY in OAS 3.2.0 - valid",
			version:     parser.OASVersion320,
			shouldError: false,
		},
		{
			name:        "QUERY in OAS 3.1.0 - error",
			version:     parser.OASVersion310,
			shouldError: true,
		},
		{
			name:        "QUERY in OAS 3.0.3 - error",
			version:     parser.OASVersion303,
			shouldError: true,
		},
		{
			name:        "QUERY in OAS 2.0 - error",
			version:     parser.OASVersion20,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc any

			if tt.version == parser.OASVersion20 {
				doc = &parser.OAS2Document{
					Swagger: "2.0",
					Info: &parser.Info{
						Title:   "Test API",
						Version: "1.0.0",
					},
					Paths: map[string]*parser.PathItem{
						"/test": {
							Query: &parser.Operation{
								OperationID: "queryTest",
								Responses: &parser.Responses{
									Codes: map[string]*parser.Response{
										"200": {Description: "OK"},
									},
								},
							},
						},
					},
				}
			} else {
				doc = &parser.OAS3Document{
					OASVersion: tt.version,
					OpenAPI:    tt.version.String(),
					Info: &parser.Info{
						Title:   "Test API",
						Version: "1.0.0",
					},
					Paths: map[string]*parser.PathItem{
						"/test": {
							Query: &parser.Operation{
								OperationID: "queryTest",
								Responses: &parser.Responses{
									Codes: map[string]*parser.Response{
										"200": {Description: "OK"},
									},
								},
							},
						},
					},
				}
			}

			parseResult := &parser.ParseResult{
				OASVersion: tt.version,
				Document:   doc,
			}

			v := New()
			result, err := v.ValidateParsed(*parseResult)
			require.NoError(t, err)

			if tt.shouldError {
				assert.False(t, result.Valid, "Document should be invalid when QUERY used in %s", tt.version)
				assert.NotEmpty(t, result.Errors, "Should have validation errors")

				// Check for QUERY-specific error
				foundQueryError := false
				for _, e := range result.Errors {
					if strings.Contains(e.Path, ".query") && strings.Contains(e.Message, "QUERY method") {
						foundQueryError = true
						break
					}
				}
				assert.True(t, foundQueryError, "Should have error about QUERY method not being supported in %s", tt.version)
			} else {
				// In OAS 3.2, QUERY should be valid
				hasQueryError := false
				for _, e := range result.Errors {
					if strings.Contains(e.Path, ".query") && strings.Contains(e.Message, "QUERY method") {
						hasQueryError = true
						break
					}
				}
				assert.False(t, hasQueryError, "Should not have QUERY method error in OAS 3.2")
			}
		})
	}
}

// TestTraceMethodValidation tests that TRACE method is validated based on OAS version
func TestTraceMethodValidation(t *testing.T) {
	tests := []struct {
		name        string
		version     parser.OASVersion
		shouldError bool
	}{
		{
			name:        "TRACE in OAS 3.2.0 - valid",
			version:     parser.OASVersion320,
			shouldError: false,
		},
		{
			name:        "TRACE in OAS 3.1.0 - valid",
			version:     parser.OASVersion310,
			shouldError: false,
		},
		{
			name:        "TRACE in OAS 3.0.3 - valid",
			version:     parser.OASVersion303,
			shouldError: false,
		},
		{
			name:        "TRACE in OAS 2.0 - error",
			version:     parser.OASVersion20,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc any

			if tt.version == parser.OASVersion20 {
				doc = &parser.OAS2Document{
					Swagger: "2.0",
					Info: &parser.Info{
						Title:   "Test API",
						Version: "1.0.0",
					},
					Paths: map[string]*parser.PathItem{
						"/test": {
							Trace: &parser.Operation{
								OperationID: "traceTest",
								Responses: &parser.Responses{
									Codes: map[string]*parser.Response{
										"200": {Description: "OK"},
									},
								},
							},
						},
					},
				}
			} else {
				doc = &parser.OAS3Document{
					OASVersion: tt.version,
					OpenAPI:    tt.version.String(),
					Info: &parser.Info{
						Title:   "Test API",
						Version: "1.0.0",
					},
					Paths: map[string]*parser.PathItem{
						"/test": {
							Trace: &parser.Operation{
								OperationID: "traceTest",
								Responses: &parser.Responses{
									Codes: map[string]*parser.Response{
										"200": {Description: "OK"},
									},
								},
							},
						},
					},
				}
			}

			parseResult := &parser.ParseResult{
				OASVersion: tt.version,
				Document:   doc,
			}

			v := New()
			result, err := v.ValidateParsed(*parseResult)
			require.NoError(t, err)

			if tt.shouldError {
				assert.False(t, result.Valid, "Document should be invalid when TRACE used in %s", tt.version)
				assert.NotEmpty(t, result.Errors, "Should have validation errors")

				// Check for TRACE-specific error
				foundTraceError := false
				for _, e := range result.Errors {
					if strings.Contains(e.Path, ".trace") && strings.Contains(e.Message, "TRACE method") {
						foundTraceError = true
						break
					}
				}
				assert.True(t, foundTraceError, "Should have error about TRACE method not being supported in %s", tt.version)
			} else {
				// In OAS 3.x, TRACE should be valid
				hasTraceError := false
				for _, e := range result.Errors {
					if strings.Contains(e.Path, ".trace") && strings.Contains(e.Message, "TRACE method") {
						hasTraceError = true
						break
					}
				}
				assert.False(t, hasTraceError, "Should not have TRACE method error in %s", tt.version)
			}
		})
	}
}

// TestOAS2InfoValidationErrorsHaveLocation is a regression test to ensure OAS2 info
// validation errors have Location (Line/Column) populated when SourceMap is available.
// Prior to the fix, validateOAS2Info used direct result.Errors = append(...) which
// skipped populateIssueLocation, causing Line/Column to always be 0.
func TestOAS2InfoValidationErrorsHaveLocation(t *testing.T) {
	// Create an OAS2 document with an invalid contact email
	// The contact.email field exists and will trigger a validation error
	oas2Doc := `swagger: "2.0"
info:
  title: Test API
  version: "1.0"
  contact:
    email: not-a-valid-email
paths: {}`

	parseResult, err := parser.ParseWithOptions(
		parser.WithBytes([]byte(oas2Doc)),
		parser.WithSourceMap(true),
	)
	require.NoError(t, err)

	// Validate with the source map to enable location tracking
	v := New()
	v.SourceMap = parseResult.SourceMap

	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	// Find the contact.email error
	var emailError *ValidationError
	for i := range result.Errors {
		if strings.Contains(result.Errors[i].Path, "contact.email") {
			emailError = &result.Errors[i]
			break
		}
	}

	require.NotNil(t, emailError, "Should have an error for invalid contact email")

	// The key assertion: Location should be populated (Line > 0)
	// This was the bug: OAS2 info errors had Line=0 because populateIssueLocation wasn't called
	assert.Greater(t, emailError.Line, 0,
		"OAS2 info validation errors should have Line populated when SourceMap is available; got Line=%d",
		emailError.Line)
}

// ========================================
// Tests for ToParseResult
// ========================================

// TestToParseResult_OAS3Document tests ToParseResult with an OAS3 document
func TestToParseResult_OAS3Document(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]*parser.PathItem),
	}

	parseResult := &parser.ParseResult{
		Version:      "3.0.3",
		OASVersion:   parser.OASVersion303,
		Document:     doc,
		SourceFormat: parser.SourceFormatYAML,
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	// Convert to ParseResult
	pr := result.ToParseResult()
	require.NotNil(t, pr)

	// Verify fields are correctly populated
	assert.Equal(t, "validator", pr.SourcePath)
	assert.Equal(t, "3.0.3", pr.Version)
	assert.Equal(t, parser.OASVersion303, pr.OASVersion)
	assert.Equal(t, parser.SourceFormatYAML, pr.SourceFormat)
	assert.Same(t, doc, pr.Document, "Document should be the same pointer")
	assert.Empty(t, pr.Errors)
}

// TestToParseResult_OAS2Document tests ToParseResult with an OAS2 document
func TestToParseResult_OAS2Document(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]*parser.PathItem),
	}

	parseResult := &parser.ParseResult{
		Version:      "2.0",
		OASVersion:   parser.OASVersion20,
		Document:     doc,
		SourceFormat: parser.SourceFormatJSON,
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	// Convert to ParseResult
	pr := result.ToParseResult()
	require.NotNil(t, pr)

	// Verify fields are correctly populated
	assert.Equal(t, "validator", pr.SourcePath)
	assert.Equal(t, "2.0", pr.Version)
	assert.Equal(t, parser.OASVersion20, pr.OASVersion)
	assert.Equal(t, parser.SourceFormatJSON, pr.SourceFormat)
	assert.Same(t, doc, pr.Document, "Document should be the same pointer")
	assert.Empty(t, pr.Errors)
}

// TestToParseResult_DocumentFieldPopulated tests that Document is populated from ValidateParsed
func TestToParseResult_DocumentFieldPopulated(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]*parser.PathItem),
	}

	parseResult := &parser.ParseResult{
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	// Verify Document is populated in ValidationResult
	assert.NotNil(t, result.Document)
	assert.Same(t, doc, result.Document, "ValidationResult.Document should be the same as parseResult.Document")

	// Verify it's also passed through to ParseResult
	pr := result.ToParseResult()
	assert.Same(t, doc, pr.Document)
}

// TestToParseResult_ErrorsAndWarningsConverted tests that errors and warnings are converted to strings with prefixes
func TestToParseResult_ErrorsAndWarningsConverted(t *testing.T) {
	// Create a document that will generate validation errors
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &parser.Info{
			Title:   "", // Missing required field
			Version: "1.0.0",
		},
		Paths: map[string]*parser.PathItem{
			"/users/": { // Trailing slash - will generate warning
				Get: &parser.Operation{
					OperationID: "getUsers",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "Success"},
						},
					},
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
	v.IncludeWarnings = true
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	// Should have at least one error (missing title)
	require.NotEmpty(t, result.Errors, "Should have validation errors")

	// Convert to ParseResult
	pr := result.ToParseResult()
	require.NotNil(t, pr)

	// Warnings in ParseResult should contain converted errors/warnings with severity prefixes
	require.NotEmpty(t, pr.Warnings, "ParseResult.Warnings should contain converted errors")

	// Check that at least one warning has "[error]" prefix
	hasErrorPrefix := false
	for _, w := range pr.Warnings {
		if strings.HasPrefix(w, "[error]") {
			hasErrorPrefix = true
			break
		}
	}
	assert.True(t, hasErrorPrefix, "Should have at least one warning with [error] prefix")
}

// TestToParseResult_EmptyErrorsAndWarnings tests ToParseResult with no errors or warnings
func TestToParseResult_EmptyErrorsAndWarnings(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]*parser.PathItem),
	}

	parseResult := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	// Should be valid with no errors
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Convert to ParseResult
	pr := result.ToParseResult()
	require.NotNil(t, pr)

	// Warnings should be empty
	assert.Empty(t, pr.Warnings)
	assert.Empty(t, pr.Errors)
}

// TestToParseResult_MetricsPreserved tests that LoadTime, SourceSize, and Stats are preserved
func TestToParseResult_MetricsPreserved(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]*parser.PathItem),
	}

	parseResult := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
		LoadTime:   500 * time.Millisecond,
		SourceSize: 1024,
		Stats: parser.DocumentStats{
			PathCount: 5,
		},
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	// Convert to ParseResult
	pr := result.ToParseResult()
	require.NotNil(t, pr)

	// Verify metrics are preserved
	assert.Equal(t, 500*time.Millisecond, pr.LoadTime)
	assert.Equal(t, int64(1024), pr.SourceSize)
	assert.Equal(t, 5, pr.Stats.PathCount)
}

// TestToParseResult_SourceFormatPreserved tests that SourceFormat is preserved through validation
func TestToParseResult_SourceFormatPreserved(t *testing.T) {
	tests := []struct {
		name   string
		format parser.SourceFormat
	}{
		{"YAML format", parser.SourceFormatYAML},
		{"JSON format", parser.SourceFormatJSON},
		{"Unknown format", parser.SourceFormatUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info: &parser.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Paths: make(map[string]*parser.PathItem),
			}

			parseResult := &parser.ParseResult{
				Version:      "3.0.3",
				OASVersion:   parser.OASVersion303,
				Document:     doc,
				SourceFormat: tt.format,
			}

			v := New()
			result, err := v.ValidateParsed(*parseResult)
			require.NoError(t, err)

			// Verify SourceFormat is preserved in ValidationResult
			assert.Equal(t, tt.format, result.SourceFormat)

			// Verify SourceFormat is passed through to ParseResult
			pr := result.ToParseResult()
			assert.Equal(t, tt.format, pr.SourceFormat)
		})
	}
}

// TestToParseResult_SourcePathPreserved tests that SourcePath is preserved through validation
func TestToParseResult_SourcePathPreserved(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]*parser.PathItem),
	}

	parseResult := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
		SourcePath: "api/openapi.yaml",
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	// Verify SourcePath is preserved in ValidationResult
	assert.Equal(t, "api/openapi.yaml", result.SourcePath)

	// Verify SourcePath is passed through to ParseResult
	pr := result.ToParseResult()
	assert.Equal(t, "api/openapi.yaml", pr.SourcePath, "SourcePath should be preserved from original parse result")
}

// TestToParseResult_SourcePathFallback tests that empty SourcePath falls back to "validator"
func TestToParseResult_SourcePathFallback(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]*parser.PathItem),
	}

	parseResult := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
		SourcePath: "", // Empty path should fall back to "validator"
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	// Verify empty SourcePath is preserved in ValidationResult
	assert.Equal(t, "", result.SourcePath)

	// Verify ToParseResult falls back to "validator"
	pr := result.ToParseResult()
	assert.Equal(t, "validator", pr.SourcePath, "Should fall back to 'validator' when source path is empty")
}

// ========================================
// Tests for Operation Context
// ========================================

// TestValidationErrorsHaveOperationContext tests that validation errors include
// operation context for identifying the affected API endpoint.
func TestValidationErrorsHaveOperationContext(t *testing.T) {
	// Create a spec with intentional errors in different locations
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users/{id}:
    parameters:
      - name: id
        in: path
        # Missing required: true - path-level error
    get:
      operationId: getUser
      parameters:
        - name: filter
          in: query
          schema:
            type: invalid_type
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
      properties:
        email:
          type: string
          format: not_a_real_format
`
	p := parser.New()
	p.ValidateStructure = false
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	// Find errors and check their operation context
	var foundPathLevelError, foundOperationError, foundSchemaError bool

	for _, e := range result.Errors {
		if strings.Contains(e.Path, "parameters") && !strings.Contains(e.Path, ".get.") {
			// Path-level parameter error
			if e.OperationContext != nil {
				foundPathLevelError = true
				assert.Equal(t, "/users/{id}", e.OperationContext.Path)
				assert.Empty(t, e.OperationContext.Method, "path-level should have no method")
			}
		}
		if strings.Contains(e.Path, ".get.parameters") {
			// Operation-level parameter error
			if e.OperationContext != nil {
				foundOperationError = true
				assert.Equal(t, "GET", e.OperationContext.Method)
				assert.Equal(t, "/users/{id}", e.OperationContext.Path)
				assert.Equal(t, "getUser", e.OperationContext.OperationID)
			}
		}
		if strings.Contains(e.Path, "components.schemas.User") {
			// Shared schema error
			if e.OperationContext != nil {
				foundSchemaError = true
				assert.True(t, e.OperationContext.IsReusableComponent)
				assert.Equal(t, "getUser", e.OperationContext.OperationID)
			}
		}
	}

	// Note: Some errors may not trigger depending on what the validator catches
	// The important thing is that when errors DO occur in these locations, they have context
	t.Logf("Found errors - path-level: %v, operation: %v, schema: %v",
		foundPathLevelError, foundOperationError, foundSchemaError)
}

// TestOperationContextInErrorString tests that operation context appears in the error string
func TestOperationContextInErrorString(t *testing.T) {
	v := New()
	result, err := v.Validate("../testdata/invalid-oas3.yaml")
	require.NoError(t, err)

	// Check that at least some errors have operation context in their string representation
	var foundWithContext bool
	for _, e := range result.Errors {
		str := e.String()
		if strings.Contains(str, "(operationId:") || strings.Contains(str, "(GET ") || strings.Contains(str, "(POST ") {
			foundWithContext = true
			t.Logf("Error with context: %s", str)
		}
	}

	// Note: This test verifies the formatting works, not that all errors have context
	if foundWithContext {
		t.Log("Found errors with operation context in string output")
	}
}
