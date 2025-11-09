package validator

import (
	"path/filepath"
	"testing"
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
	if v.parser == nil {
		t.Error("Expected parser to be initialized")
	}
}

// TestValidateOAS2Valid tests validation of a valid OAS 2.0 document
func TestValidateOAS2Valid(t *testing.T) {
	v := New()
	testFile := filepath.Join("..", "..", "testdata", "petstore-2.0.yaml")

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
	testFile := filepath.Join("..", "..", "testdata", "invalid-oas2.yaml")

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
			testFile := filepath.Join("..", "..", "testdata", tc.file)

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
	testFile := filepath.Join("..", "..", "testdata", "invalid-oas3.yaml")

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
	testFile := filepath.Join("..", "..", "testdata", "with-external-refs.yaml")

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

	testFile := filepath.Join("..", "..", "testdata", "petstore-3.0.yaml")

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

	testFile := filepath.Join("..", "..", "testdata", "petstore-3.0.yaml")

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
				if !contains(result, substr) {
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
		{"invalid", false},
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

// TestValidateHTTPStatusCode tests the validateHTTPStatusCode helper
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
			result := validateHTTPStatusCode(tc.code)
			if result != tc.valid {
				t.Errorf("validateHTTPStatusCode(%q) = %v, expected %v", tc.code, result, tc.valid)
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

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
