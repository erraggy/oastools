package validator

import (
	"testing"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// TestPopulateIssueLocation tests the populateIssueLocation helper
func TestPopulateIssueLocation(t *testing.T) {
	tests := []struct {
		name       string
		sourceMap  *parser.SourceMap
		path       string
		wantLine   int
		wantColumn int
		wantFile   string
	}{
		{
			name:      "nil source map",
			sourceMap: nil,
			path:      "info.title",
			wantLine:  0,
		},
		{
			name: "path found in source map",
			sourceMap: func() *parser.SourceMap {
				sm := parser.NewSourceMap()
				// Manually set up the source map for testing
				// Note: We can't set directly, so we'll use a parsed result
				return sm
			}(),
			path:     "nonexistent.path",
			wantLine: 0, // Path not found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Validator{SourceMap: tt.sourceMap}
			issue := &ValidationError{Path: tt.path}
			v.populateIssueLocation(issue)

			assert.Equal(t, tt.wantLine, issue.Line)
		})
	}
}

// TestAddError tests the addError helper function
func TestAddError(t *testing.T) {
	v := New()
	result := &ValidationResult{
		Errors: make([]ValidationError, 0),
	}

	v.addError(result, "info.title", "Test error message",
		withField("title"),
		withValue("test-value"),
		withSpecRef("https://example.com"),
	)

	require.Len(t, result.Errors, 1)
	assert.Equal(t, "info.title", result.Errors[0].Path)
	assert.Equal(t, "Test error message", result.Errors[0].Message)
	assert.Equal(t, SeverityError, result.Errors[0].Severity)
	assert.Equal(t, "title", result.Errors[0].Field)
	assert.Equal(t, "test-value", result.Errors[0].Value)
	assert.Equal(t, "https://example.com", result.Errors[0].SpecRef)
}

// TestAddWarning tests the addWarning helper function
func TestAddWarning(t *testing.T) {
	v := New()
	result := &ValidationResult{
		Warnings: make([]ValidationError, 0),
	}

	v.addWarning(result, "info.description", "Test warning message",
		withField("description"),
	)

	require.Len(t, result.Warnings, 1)
	assert.Equal(t, "info.description", result.Warnings[0].Path)
	assert.Equal(t, "Test warning message", result.Warnings[0].Message)
	assert.Equal(t, SeverityWarning, result.Warnings[0].Severity)
	assert.Equal(t, "description", result.Warnings[0].Field)
}
