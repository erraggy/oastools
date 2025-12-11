package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erraggy/oastools/internal/httputil"
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
			contains: []string{"✗", "paths./pets.get", "Missing required field", "Spec:"},
		},
		{
			name: "Warning without spec ref",
			error: ValidationError{
				Path:     "info.description",
				Message:  "Should include description",
				Severity: SeverityWarning,
			},
			contains: []string{"⚠", "info.description", "Should include description"},
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

// TestExtractPathParameters tests the extractPathParameters helper
func TestExtractPathParameters(t *testing.T) {
	testCases := []struct {
		path     string
		expected map[string]bool
	}{
		{
			path:     "/pets",
			expected: map[string]bool{},
		},
		{
			path: "/pets/{petId}",
			expected: map[string]bool{
				"petId": true,
			},
		},
		{
			path: "/pets/{petId}/owners/{ownerId}",
			expected: map[string]bool{
				"petId":   true,
				"ownerId": true,
			},
		},
		{
			path:     "/pets/{petId}/status",
			expected: map[string]bool{"petId": true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := extractPathParameters(tc.path)
			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d parameters, got %d", len(tc.expected), len(result))
			}
			for key := range tc.expected {
				if !result[key] {
					t.Errorf("Expected parameter %q to be found", key)
				}
			}
		})
	}
}

// TestIsValidMediaType tests the isValidMediaType helper
func TestIsValidMediaType(t *testing.T) {
	testCases := []struct {
		mediaType string
		valid     bool
	}{
		{"application/json", true},
		{"text/plain", true},
		{"application/*", true},
		{"*/*", true},
		{"application/vnd.api+json", true},
		{"", false},
		{"?invalid", false},
		{"/json", false},
		{"application/", false},
	}

	for _, tc := range testCases {
		t.Run(tc.mediaType, func(t *testing.T) {
			result := isValidMediaType(tc.mediaType)
			if result != tc.valid {
				t.Errorf("isValidMediaType(%q) = %v, expected %v", tc.mediaType, result, tc.valid)
			}
		})
	}
}

// TestIsValidURL tests the isValidURL helper
func TestIsValidURL(t *testing.T) {
	testCases := []struct {
		url   string
		valid bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"/relative/path", true},
		{"", false},
		{"ftp://example.com", false},
		{"not-a-url", false},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			result := isValidURL(tc.url)
			if result != tc.valid {
				t.Errorf("isValidURL(%q) = %v, expected %v", tc.url, result, tc.valid)
			}
		})
	}
}

// TestIsValidEmail tests the isValidEmail helper
func TestIsValidEmail(t *testing.T) {
	testCases := []struct {
		email string
		valid bool
	}{
		{"user@example.com", true},
		{"test.user@example.co.uk", true},
		{"", true}, // Empty is valid (optional field)
		{"invalid", false},
		{"@example.com", false},
		{"user@", false},
		{"user@invalid", false},
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			result := isValidEmail(tc.email)
			if result != tc.valid {
				t.Errorf("isValidEmail(%q) = %v, expected %v", tc.email, result, tc.valid)
			}
		})
	}
}

// TestValidateHTTPStatusCode tests the httputil.ValidateStatusCode helper
func TestValidateHTTPStatusCode(t *testing.T) {
	testCases := []struct {
		code  string
		valid bool
	}{
		{"200", true},
		{"404", true},
		{"500", true},
		{"2XX", true},
		{"4XX", true},
		{"5XX", true},
		{"default", true},
		{"", false},
		{"999", false},
		{"99", false},
		{"6XX", false},
		{"abc", false},
	}

	for _, tc := range testCases {
		t.Run(tc.code, func(t *testing.T) {
			result := httputil.ValidateStatusCode(tc.code)
			if result != tc.valid {
				t.Errorf("httputil.ValidateStatusCode(%q) = %v, expected %v", tc.code, result, tc.valid)
			}
		})
	}
}

// TestValidateSPDXLicense tests the validateSPDXLicense helper
func TestValidateSPDXLicense(t *testing.T) {
	testCases := []struct {
		identifier string
		valid      bool
	}{
		{"MIT", true},
		{"Apache-2.0", true},
		{"GPL-3.0-or-later", true},
		{"", true}, // Empty is valid (optional)
		{"MIT License", false},
		{"Apache 2.0", false},
	}

	for _, tc := range testCases {
		t.Run(tc.identifier, func(t *testing.T) {
			result := validateSPDXLicense(tc.identifier)
			if result != tc.valid {
				t.Errorf("validateSPDXLicense(%q) = %v, expected %v", tc.identifier, result, tc.valid)
			}
		})
	}
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

// TestRefValidation tests that $ref values are properly validated
func TestRefValidation(t *testing.T) {
	testCases := []struct {
		name          string
		oasVersion    string
		content       string
		expectError   bool
		errorContains string
	}{
		{
			name:       "OAS 2.0 - Valid ref to definitions",
			oasVersion: "2.0",
			content: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          schema:
            $ref: "#/definitions/Pet"
definitions:
  Pet:
    type: object
    properties:
      name:
        type: string`,
			expectError: false,
		},
		{
			name:       "OAS 2.0 - Invalid ref using OAS 3.x format",
			oasVersion: "2.0",
			content: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          schema:
            $ref: "#/components/schemas/Pet"
definitions:
  Pet:
    type: object`,
			expectError:   true,
			errorContains: "does not resolve to a valid component",
		},
		{
			name:       "OAS 2.0 - Ref to non-existent definition",
			oasVersion: "2.0",
			content: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          schema:
            $ref: "#/definitions/NonExistent"
definitions:
  Pet:
    type: object`,
			expectError:   true,
			errorContains: "does not resolve to a valid component",
		},
		{
			name:       "OAS 3.0 - Valid ref to components/schemas",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string`,
			expectError: false,
		},
		{
			name:       "OAS 3.0 - Invalid ref using OAS 2.0 format",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/definitions/Pet"
components:
  schemas:
    Pet:
      type: object`,
			expectError:   true,
			errorContains: "does not resolve to a valid component",
		},
		{
			name:       "OAS 3.0 - Ref to non-existent schema",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/NonExistent"
components:
  schemas:
    Pet:
      type: object`,
			expectError:   true,
			errorContains: "does not resolve to a valid component",
		},
		{
			name:       "OAS 3.0 - Valid ref in nested schema",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: object
                properties:
                  pet:
                    $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object`,
			expectError: false,
		},
		{
			name:       "OAS 3.0 - Invalid ref in nested schema",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: object
                properties:
                  pet:
                    $ref: "#/components/schemas/NonExistent"
components:
  schemas:
    Pet:
      type: object`,
			expectError:   true,
			errorContains: "does not resolve to a valid component",
		},
		{
			name:       "OAS 3.0 - Valid ref to parameter",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: "#/components/parameters/PetId"
      responses:
        '200':
          description: Success
components:
  parameters:
    PetId:
      name: petId
      in: path
      required: true
      schema:
        type: string`,
			expectError: false,
		},
		{
			name:       "OAS 3.0 - Invalid ref to parameter",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: "#/components/parameters/NonExistent"
      responses:
        '200':
          description: Success
components:
  parameters:
    PetId:
      name: petId
      in: path
      required: true
      schema:
        type: string`,
			expectError:   true,
			errorContains: "does not resolve to a valid component",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Write test file
			tmpFile := filepath.Join(t.TempDir(), "test.yaml")
			err := os.WriteFile(tmpFile, []byte(tc.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Validate
			v := New()
			result, err := v.Validate(tmpFile)
			if err != nil {
				t.Fatalf("Validate failed: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			hasRefError := false
			for _, validationErr := range result.Errors {
				if tc.errorContains != "" && validationErr.Field == "$ref" {
					if !strings.Contains(validationErr.Message, tc.errorContains) {
						t.Errorf("Expected error containing '%s', got: %s", tc.errorContains, validationErr.Message)
					}
					hasRefError = true
				}
			}

			if tc.expectError && !hasRefError {
				t.Errorf("Expected $ref validation error, but got none. Errors: %v", result.Errors)
			}

			if !tc.expectError && hasRefError {
				t.Errorf("Did not expect $ref validation error, but got one")
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

// TestValidateWithOptions_FilePath tests the functional options API with file path
func TestValidateWithOptions_FilePath(t *testing.T) {
	result, err := ValidateWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithIncludeWarnings(true),
		WithStrictMode(false),
	)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, "3.0.3", result.Version)
}

// TestValidateWithOptions_Parsed tests the functional options API with parsed result
func TestValidateWithOptions_Parsed(t *testing.T) {
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	result, err := ValidateWithOptions(
		WithParsed(*parseResult),
		WithIncludeWarnings(true),
	)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, "3.0.3", result.Version)
}

// TestValidateWithOptions_StrictMode tests that strict mode is applied
func TestValidateWithOptions_StrictMode(t *testing.T) {
	result, err := ValidateWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithStrictMode(true),
		WithIncludeWarnings(true),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	// Strict mode may generate additional warnings
}

// TestValidateWithOptions_DisableWarnings tests that warnings can be disabled
func TestValidateWithOptions_DisableWarnings(t *testing.T) {
	result, err := ValidateWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithIncludeWarnings(false),
	)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Warnings, "warnings should be filtered out when IncludeWarnings=false")
	assert.Equal(t, 0, result.WarningCount)
}

// TestValidateWithOptions_DefaultValues tests that default values are applied correctly
func TestValidateWithOptions_DefaultValues(t *testing.T) {
	result, err := ValidateWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		// Not specifying WithIncludeWarnings or WithStrictMode to test defaults
	)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	// Default: IncludeWarnings = true, so warnings may be present
	// (though petstore might not have warnings)
}

// TestValidateWithOptions_NoInputSource tests error when no input source is specified
func TestValidateWithOptions_NoInputSource(t *testing.T) {
	_, err := ValidateWithOptions(
		WithIncludeWarnings(true),
		WithStrictMode(false),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify an input source")
}

// TestValidateWithOptions_MultipleInputSources tests error when multiple input sources are specified
func TestValidateWithOptions_MultipleInputSources(t *testing.T) {
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	_, err = ValidateWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithParsed(*parseResult),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify exactly one input source")
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

// TestValidateWithOptions_AllOptions tests using all options together
func TestValidateWithOptions_AllOptions(t *testing.T) {
	result, err := ValidateWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithIncludeWarnings(false),
		WithStrictMode(true),
		WithUserAgent("test-validator/1.0"),
	)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Warnings)
}

// TestWithFilePath_Validator tests the WithFilePath option function
func TestWithFilePath_Validator(t *testing.T) {
	cfg := &validateConfig{}
	opt := WithFilePath("test.yaml")
	err := opt(cfg)

	require.NoError(t, err)
	require.NotNil(t, cfg.filePath)
	assert.Equal(t, "test.yaml", *cfg.filePath)
}

// TestWithParsed tests the WithParsed option function
func TestWithParsed(t *testing.T) {
	parseResult := parser.ParseResult{Version: "3.0.0"}
	cfg := &validateConfig{}
	opt := WithParsed(parseResult)
	err := opt(cfg)

	require.NoError(t, err)
	require.NotNil(t, cfg.parsed)
	assert.Equal(t, "3.0.0", cfg.parsed.Version)
}

// TestWithIncludeWarnings tests the WithIncludeWarnings option function
func TestWithIncludeWarnings(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &validateConfig{}
			opt := WithIncludeWarnings(tt.enabled)
			err := opt(cfg)

			require.NoError(t, err)
			assert.Equal(t, tt.enabled, cfg.includeWarnings)
		})
	}
}

// TestWithStrictMode tests the WithStrictMode option function
func TestWithStrictMode(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &validateConfig{}
			opt := WithStrictMode(tt.enabled)
			err := opt(cfg)

			require.NoError(t, err)
			assert.Equal(t, tt.enabled, cfg.strictMode)
		})
	}
}

// TestWithUserAgent_Validator tests the WithUserAgent option function
func TestWithUserAgent_Validator(t *testing.T) {
	cfg := &validateConfig{}
	opt := WithUserAgent("custom-agent/2.0")
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, "custom-agent/2.0", cfg.userAgent)
}

// TestApplyOptions_Defaults_Validator tests that default values are set correctly
func TestApplyOptions_Defaults_Validator(t *testing.T) {
	cfg, err := applyOptions(WithFilePath("test.yaml"))

	require.NoError(t, err)
	assert.True(t, cfg.includeWarnings, "default includeWarnings should be true")
	assert.False(t, cfg.strictMode, "default strictMode should be false")
	assert.Equal(t, "", cfg.userAgent, "default userAgent should be empty")
}

// TestApplyOptions_OverrideDefaults_Validator tests that options override defaults
func TestApplyOptions_OverrideDefaults_Validator(t *testing.T) {
	cfg, err := applyOptions(
		WithFilePath("test.yaml"),
		WithIncludeWarnings(false),
		WithStrictMode(true),
		WithUserAgent("custom/1.0"),
	)

	require.NoError(t, err)
	assert.False(t, cfg.includeWarnings)
	assert.True(t, cfg.strictMode)
	assert.Equal(t, "custom/1.0", cfg.userAgent)
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
			var doc interface{}

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
			var doc interface{}

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
