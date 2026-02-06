package httputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		// Valid: "default" keyword
		{"default keyword", "default", true},

		// Valid: Extension fields (x-)
		{"extension x-custom", "x-custom", true},
		{"extension x-200", "x-200", true},
		{"extension x-", "x-", true},

		// Valid: Wildcard patterns (1XX-5XX)
		{"wildcard 1XX", "1XX", true},
		{"wildcard 2XX", "2XX", true},
		{"wildcard 3XX", "3XX", true},
		{"wildcard 4XX", "4XX", true},
		{"wildcard 5XX", "5XX", true},

		// Invalid: Wildcards outside 1-5 range
		{"invalid wildcard 0XX", "0XX", false},
		{"invalid wildcard 6XX", "6XX", false},
		{"invalid wildcard 7XX", "7XX", false},
		{"invalid wildcard 9XX", "9XX", false},

		// Invalid: Partial wildcards
		{"partial wildcard 2X", "2X", false},
		{"partial wildcard 20X", "20X", false},
		{"partial wildcard X2X", "X2X", false},
		{"partial wildcard XX2", "XX2", false},

		// Valid: Numeric codes in valid range (100-599)
		{"valid 100", "100", true},
		{"valid 200", "200", true},
		{"valid 201", "201", true},
		{"valid 204", "204", true},
		{"valid 301", "301", true},
		{"valid 400", "400", true},
		{"valid 404", "404", true},
		{"valid 418", "418", true}, // I'm a teapot
		{"valid 500", "500", true},
		{"valid 503", "503", true},
		{"valid 599", "599", true},

		// Invalid: Numeric codes outside valid range
		{"invalid 099", "099", false}, // Below MinStatusCode
		{"invalid 600", "600", false}, // Above MaxStatusCode
		{"invalid 999", "999", false},
		{"invalid 000", "000", false},

		// Invalid: Too short or too long
		{"too short 99", "99", false},
		{"too short 1", "1", false},
		{"too long 1000", "1000", false},
		{"too long 20000", "20000", false},

		// Invalid: Empty and whitespace
		{"empty string", "", false},
		{"whitespace", "   ", false},
		{"space in code", "2 00", false},

		// Invalid: Non-numeric characters
		{"alphabetic abc", "abc", false},
		{"alphanumeric 2a0", "2a0", false},
		{"alphanumeric a00", "a00", false},
		{"alphanumeric 00a", "00a", false},

		// Invalid: Special characters
		{"special char @00", "@00", false},
		{"special char 2-0", "2-0", false},
		{"special char 20!", "20!", false},

		// Edge cases: Boundary values
		{"boundary 100", "100", true},  // MinStatusCode
		{"boundary 599", "599", true},  // MaxStatusCode
		{"boundary 99", "99", false},   // Just below min
		{"boundary 600", "600", false}, // Just above max

		// Edge cases: Extensions that might look like codes
		{"not extension x", "x", false},       // Too short
		{"not extension x200", "x200", false}, // Wrong format (4 chars but not wildcard)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateStatusCode(tt.code)
			assert.Equal(t, tt.expected, result, "ValidateStatusCode(%q) = %v, want %v", tt.code, result, tt.expected)
		})
	}
}

func TestIsStandardStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		// 1xx Informational responses
		{"standard 100", "100", true},
		{"standard 101", "101", true},
		{"standard 102", "102", true},
		{"standard 103", "103", true},
		{"non-standard 104", "104", false},
		{"non-standard 199", "199", false},

		// 2xx Success
		{"standard 200", "200", true},
		{"standard 201", "201", true},
		{"standard 204", "204", true},
		{"standard 206", "206", true},
		{"non-standard 299", "299", false},

		// 3xx Redirection
		{"standard 300", "300", true},
		{"standard 301", "301", true},
		{"standard 302", "302", true},
		{"standard 304", "304", true},
		{"standard 308", "308", true},
		{"non-standard 306", "306", false}, // Unused code
		{"non-standard 399", "399", false},

		// 4xx Client errors
		{"standard 400", "400", true},
		{"standard 401", "401", true},
		{"standard 403", "403", true},
		{"standard 404", "404", true},
		{"standard 418", "418", true}, // I'm a teapot
		{"standard 429", "429", true},
		{"standard 451", "451", true},
		{"non-standard 499", "499", false},

		// 5xx Server errors
		{"standard 500", "500", true},
		{"standard 501", "501", true},
		{"standard 502", "502", true},
		{"standard 503", "503", true},
		{"standard 504", "504", true},
		{"non-standard 509", "509", false}, // Not in RFC 9110
		{"non-standard 599", "599", false},

		// Special values (should not be in standard codes)
		{"not standard default", "default", false},
		{"not standard 1XX", "1XX", false},
		{"not standard 2XX", "2XX", false},
		{"not standard x-200", "x-200", false},

		// Invalid codes
		{"invalid empty", "", false},
		{"invalid 999", "999", false},
		{"invalid abc", "abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStandardStatusCode(tt.code)
			assert.Equal(t, tt.expected, result, "IsStandardStatusCode(%q) = %v, want %v", tt.code, result, tt.expected)
		})
	}
}

func TestIsValidMediaType(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		expected  bool
	}{
		// Valid: Universal wildcard
		{"universal wildcard", "*/*", true},

		// Valid: Type wildcards
		{"type wildcard application", "application/*", true},
		{"type wildcard text", "text/*", true},
		{"type wildcard image", "image/*", true},
		{"type wildcard audio", "audio/*", true},
		{"type wildcard video", "video/*", true},
		{"type wildcard multipart", "multipart/*", true},

		// Note: mime.ParseMediaType actually accepts */subtype (though uncommon)
		// The Go MIME parser is permissive here
		{"subtype wildcard json", "*/json", true},
		{"subtype wildcard xml", "*/xml", true},
		{"subtype wildcard html", "*/html", true},

		// Valid: Standard media types
		{"standard application/json", "application/json", true},
		{"standard text/html", "text/html", true},
		{"standard text/plain", "text/plain", true},
		{"standard application/xml", "application/xml", true},
		{"standard image/png", "image/png", true},
		{"standard image/jpeg", "image/jpeg", true},
		{"standard audio/mpeg", "audio/mpeg", true},
		{"standard video/mp4", "video/mp4", true},
		{"standard multipart/form-data", "multipart/form-data", true},

		// Valid: Media types with parameters
		{"with charset", "text/html; charset=utf-8", true},
		{"with boundary", "multipart/form-data; boundary=----WebKitFormBoundary", true},
		{"with multiple params", "text/html; charset=utf-8; version=1.0", true},

		// Valid: Vendor-specific types
		{"vendor json api", "application/vnd.api+json", true},
		{"vendor hal", "application/hal+json", true},
		{"vendor custom", "application/vnd.mycompany.myapp-v1+json", true},

		// Invalid: Malformed media types
		{"missing subtype", "application/", false},
		{"missing type", "/json", false},
		// Note: mime.ParseMediaType accepts single tokens as media types
		{"no slash", "applicationjson", true},
		{"multiple slashes", "application/json/extra", false},
		{"empty", "", false},
		{"whitespace only", "   ", false},

		// Invalid: Wildcard only on left
		{"type wildcard only", "application/", false},
		{"empty type wildcard", "/", false},

		// Edge cases: Special characters
		{"with plus", "application/json+ld", true},
		{"with dash", "application/atom+xml", true},
		{"with dot", "application/vnd.ms-excel", true},

		// Edge cases: Case sensitivity (MIME types are case-insensitive)
		{"uppercase", "APPLICATION/JSON", true},
		{"mixed case", "Application/Json", true},
		{"lowercase", "application/json", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidMediaType(tt.mediaType)
			assert.Equal(t, tt.expected, result, "IsValidMediaType(%q) = %v, want %v", tt.mediaType, result, tt.expected)
		})
	}
}

// TestHTTPMethodConstants verifies that method constants have expected lowercase values.
// This ensures consistency with OpenAPI specification requirements.
func TestHTTPMethodConstants(t *testing.T) {
	assert.Equal(t, "get", MethodGet, "MethodGet should be lowercase")
	assert.Equal(t, "put", MethodPut, "MethodPut should be lowercase")
	assert.Equal(t, "post", MethodPost, "MethodPost should be lowercase")
	assert.Equal(t, "delete", MethodDelete, "MethodDelete should be lowercase")
	assert.Equal(t, "options", MethodOptions, "MethodOptions should be lowercase")
	assert.Equal(t, "head", MethodHead, "MethodHead should be lowercase")
	assert.Equal(t, "patch", MethodPatch, "MethodPatch should be lowercase")
	assert.Equal(t, "trace", MethodTrace, "MethodTrace should be lowercase")
}

// TestStandardHTTPStatusCodesCompleteness verifies that StandardHTTPStatusCodes
// contains expected RFC 9110 codes and doesn't include unexpected ones.
func TestStandardHTTPStatusCodesCompleteness(t *testing.T) {
	// Sample of codes that MUST be present
	requiredCodes := []string{
		"200", "201", "204", // 2xx
		"301", "302", "304", // 3xx
		"400", "401", "403", "404", // 4xx
		"500", "502", "503", // 5xx
	}

	for _, code := range requiredCodes {
		assert.True(t, StandardHTTPStatusCodes[code], "Standard code %s should be in map", code)
	}

	// Codes that should NOT be present (non-standard or out of range)
	excludedCodes := []string{
		"099", "600", "999", // Out of range
		"306",                   // Unused/reserved
		"default", "1XX", "2XX", // Special values
	}

	for _, code := range excludedCodes {
		assert.False(t, StandardHTTPStatusCodes[code], "Non-standard code %s should not be in map", code)
	}

	// Verify map has reasonable size (RFC 9110 defines ~60 codes)
	assert.Greater(t, len(StandardHTTPStatusCodes), 40, "Should have at least 40 standard codes")
	assert.Less(t, len(StandardHTTPStatusCodes), 100, "Should have fewer than 100 codes")
}
