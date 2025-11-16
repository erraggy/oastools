// Package httputil provides HTTP-related validation utilities and constants.
package httputil

import (
	"mime"
	"strconv"
	"strings"
)

// HTTP Status Code Constants
const (
	StatusCodeLength     = 3   // Standard length of HTTP status codes (e.g., "200", "404")
	MinStatusCode        = 100 // Minimum valid HTTP status code
	MaxStatusCode        = 599 // Maximum valid HTTP status code
	WildcardChar         = 'X' // Wildcard character used in status code patterns (e.g., "2XX")
	MinWildcardFirstChar = '1' // Minimum first digit for wildcard patterns
	MaxWildcardFirstChar = '5' // Maximum first digit for wildcard patterns
)

// HTTP Method Constants
const (
	MethodGet     = "get"
	MethodPut     = "put"
	MethodPost    = "post"
	MethodDelete  = "delete"
	MethodOptions = "options"
	MethodHead    = "head"
	MethodPatch   = "patch"
	MethodTrace   = "trace" // OAS 3.0+ only
)

// Wildcard boundary characters for validation
const (
	minWildcardBoundary = '1'
	maxWildcardBoundary = '5'
)

// StandardHTTPStatusCodes contains RFC 9110 officially defined HTTP status codes.
// These are used in strict mode validation to warn about non-standard codes.
var StandardHTTPStatusCodes = map[string]bool{
	// 1xx Informational
	"100": true, "101": true, "102": true, "103": true,
	// 2xx Success
	"200": true, "201": true, "202": true, "203": true, "204": true, "205": true,
	"206": true, "207": true, "208": true, "226": true,
	// 3xx Redirection
	"300": true, "301": true, "302": true, "303": true, "304": true, "305": true,
	"307": true, "308": true,
	// 4xx Client Error
	"400": true, "401": true, "402": true, "403": true, "404": true, "405": true,
	"406": true, "407": true, "408": true, "409": true, "410": true, "411": true,
	"412": true, "413": true, "414": true, "415": true, "416": true, "417": true,
	"418": true, "421": true, "422": true, "423": true, "424": true, "425": true,
	"426": true, "428": true, "429": true, "431": true, "451": true,
	// 5xx Server Error
	"500": true, "501": true, "502": true, "503": true, "504": true, "505": true,
	"506": true, "507": true, "508": true, "510": true, "511": true,
}

// ValidateStatusCode checks if a status code string is valid according to OpenAPI spec.
// Valid values are:
//   - "default" for default response
//   - Extension fields starting with "x-"
//   - Wildcard patterns: 1XX, 2XX, 3XX, 4XX, 5XX
//   - Numeric codes: 100-599
func ValidateStatusCode(code string) bool {
	if code == "default" {
		return true
	}

	if strings.HasPrefix(code, "x-") {
		return true
	}

	if len(code) == StatusCodeLength {
		// Check for wildcard patterns (e.g., "2XX", "4XX")
		if code[1] == WildcardChar && code[2] == WildcardChar {
			firstChar := code[0]
			if firstChar >= minWildcardBoundary && firstChar <= maxWildcardBoundary {
				return true
			}
		}

		// Check for numeric codes
		if code[0] >= '0' && code[0] <= '9' &&
			code[1] >= '0' && code[1] <= '9' &&
			code[2] >= '0' && code[2] <= '9' {
			statusCode, err := strconv.Atoi(code)
			if err == nil && statusCode >= MinStatusCode && statusCode <= MaxStatusCode {
				return true
			}
		}
	}

	return false
}

// IsStandardStatusCode checks if a status code is a well-defined standard HTTP code.
// Returns true only for codes in StandardHTTPStatusCodes map.
func IsStandardStatusCode(code string) bool {
	return StandardHTTPStatusCodes[code]
}

// IsValidMediaType validates a media type string according to RFC 2045/2046.
// Handles wildcards (*/* and type/*) and prevents invalid combinations (*/subtype).
func IsValidMediaType(mediaType string) bool {
	if mediaType == "*/*" {
		return true
	}

	if strings.HasSuffix(mediaType, "/*") {
		// Check format: type/* (e.g., application/*)
		parts := strings.Split(mediaType, "/")
		if len(parts) == 2 && parts[0] != "" && parts[0] != "*" {
			return true
		}
		return false
	}

	// Use standard MIME type parser for regular types
	_, _, err := mime.ParseMediaType(mediaType)
	return err == nil
}
