package httputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidResponsesKey(t *testing.T) {
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
			result := IsValidResponsesKey(tt.code)
			assert.Equal(t, tt.expected, result, "IsValidResponsesKey(%q) = %v, want %v", tt.code, result, tt.expected)
		})
	}
}

// TestIsStatusCode pins the narrow predicate against the broad one. The two
// disagree on exactly the keys a Responses Object admits without their being
// status codes, and a caller ranging Responses.Codes needs the narrow answer.
func TestIsStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		// The keys that separate this predicate from IsValidResponsesKey.
		{"default is not a status code", "default", false},
		{"extension x-custom is not a status code", "x-custom", false},
		{"extension x-200 is not a status code", "x-200", false},
		{"extension x- is not a status code", "x-", false},

		// Wildcard ranges.
		{"wildcard 1XX", "1XX", true},
		{"wildcard 5XX", "5XX", true},
		{"wildcard 0XX", "0XX", false},
		{"wildcard 6XX", "6XX", false},

		// Numeric codes and their boundaries.
		{"numeric 100", "100", true},
		{"numeric 200", "200", true},
		{"numeric 599", "599", true},
		{"numeric 099", "099", false},
		{"numeric 600", "600", false},
		{"numeric 999", "999", false},

		// Shape.
		{"empty string", "", false},
		{"too short", "99", false},
		{"too long", "1000", false},
		{"alphabetic", "abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsStatusCode(tt.code)
			assert.Equal(t, tt.expected, got,
				"IsStatusCode(%q) = %v, want %v", tt.code, got, tt.expected)
		})
	}
}

// TestIsWildcardStatusCode pins the form OAS 3.0 introduced and OAS 2.0 does not
// define. A caller that knows the document version uses it to reject "2XX" in a
// 2.0 document, so the lowercase and partial cases matter: the specification
// says the wildcard is the uppercase X.
func TestIsWildcardStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{"wildcard 1XX", "1XX", true},
		{"wildcard 2XX", "2XX", true},
		{"wildcard 5XX", "5XX", true},

		// Boundaries of the leading digit.
		{"wildcard 0XX below range", "0XX", false},
		{"wildcard 6XX above range", "6XX", false},

		// The wildcard is uppercase, per the specification.
		{"lowercase 2xx", "2xx", false},
		{"mixed 2Xx", "2Xx", false},

		// Shape.
		{"numeric 200 is not a range", "200", false},
		{"partial 2X", "2X", false},
		{"partial 20X", "20X", false},
		{"reversed XX2", "XX2", false},
		{"all wildcards XXX", "XXX", false},
		{"default", "default", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsWildcardStatusCode(tt.code)
			assert.Equal(t, tt.expected, got,
				"IsWildcardStatusCode(%q) = %v, want %v", tt.code, got, tt.expected)
		})
	}
}

// TestIsNumericStatusCode pins the form every OAS version defines, which is what
// a caller falls back to when the version does not admit a wildcard range.
func TestIsNumericStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{"numeric 100 lower boundary", "100", true},
		{"numeric 200", "200", true},
		{"numeric 599 upper boundary", "599", true},
		{"numeric 099 below range", "099", false},
		{"numeric 600 above range", "600", false},
		{"numeric 999", "999", false},

		{"wildcard 2XX is not numeric", "2XX", false},
		{"empty string", "", false},
		{"too short", "99", false},
		{"too long", "1000", false},
		{"alphabetic", "abc", false},
		{"mixed 2a0", "2a0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNumericStatusCode(tt.code)
			assert.Equal(t, tt.expected, got,
				"IsNumericStatusCode(%q) = %v, want %v", tt.code, got, tt.expected)
		})
	}
}

// TestStatusCodeFormsPartitionIsStatusCode holds the two narrow predicates to
// the property their callers depend on: every key [IsStatusCode] accepts is
// either a numeric code or a wildcard range and never both, and neither narrow
// predicate accepts a key [IsStatusCode] rejects.
//
// A version-scoped caller rejects wildcard ranges and keeps numeric codes. An
// overlap would therefore reject a numeric code that OAS 2.0 defines, and a gap
// would let a key through with no diagnostic at all. Exhaustive over every
// three-byte string, which is the only length either predicate can accept.
func TestStatusCodeFormsPartitionIsStatusCode(t *testing.T) {
	buf := make([]byte, StatusCodeLength)
	for a := range 256 {
		for b := range 256 {
			for c := range 256 {
				buf[0], buf[1], buf[2] = byte(a), byte(b), byte(c)
				code := string(buf)
				numeric := IsNumericStatusCode(code)
				wildcard := IsWildcardStatusCode(code)
				if numeric && wildcard {
					t.Fatalf("%q is both a numeric code and a wildcard range", code)
				}
				if got := IsStatusCode(code); got != (numeric || wildcard) {
					t.Fatalf("IsStatusCode(%q) = %v, but numeric = %v and wildcard = %v",
						code, got, numeric, wildcard)
				}
			}
		}
	}

	// No other length can be accepted, so the union holds there trivially.
	for _, code := range []string{"", "2", "20", "2000", "2XXX", "default", "x-note"} {
		if IsNumericStatusCode(code) || IsWildcardStatusCode(code) || IsStatusCode(code) {
			t.Fatalf("%q was accepted by a status code predicate", code)
		}
	}
}

func TestIsExtensionKey(t *testing.T) {
	assert.True(t, IsExtensionKey("x-custom"))
	assert.True(t, IsExtensionKey("x-"))
	assert.False(t, IsExtensionKey("default"))
	assert.False(t, IsExtensionKey("200"))
	assert.False(t, IsExtensionKey("x"))
	assert.False(t, IsExtensionKey(""))
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

// TestIsSuccessStatusCode covers the narrower of the three questions: which
// keys denote a successful response. A key that is not a status code cannot,
// which matters because Responses.Codes may hold one.
func TestIsSuccessStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{"wildcard 2XX", "2XX", true},
		{"numeric 200", "200", true},
		{"numeric 204", "204", true},
		{"numeric 299", "299", true},

		{"numeric 199", "199", false},
		{"numeric 300", "300", false},
		{"wildcard 3XX", "3XX", false},
		{"wildcard 5XX", "5XX", false},

		// Not status codes at all, so not successful ones either. The leading
		// 2 is what a bare prefix check would accept.
		{"malformed 2foo", "2foo", false},
		{"malformed 2X", "2X", false},
		{"extension x-2xx", "x-2xx", false},
		{"default", "default", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSuccessStatusCode(tt.code)
			assert.Equal(t, tt.expected, got,
				"IsSuccessStatusCode(%q) = %v, want %v", tt.code, got, tt.expected)
		})
	}
}
