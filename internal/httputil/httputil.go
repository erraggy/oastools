// Package httputil provides HTTP-related validation utilities and constants.
package httputil

import (
	"mime"
	"strings"
)

// HTTP Status Code Constants
const (
	StatusCodeLength = 3   // Standard length of HTTP status codes (e.g., "200", "404")
	MinStatusCode    = 100 // Minimum valid HTTP status code
	MaxStatusCode    = 599 // Maximum valid HTTP status code
	WildcardChar     = 'X' // Wildcard character used in status code patterns (e.g., "2XX")
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
	MethodTrace   = "trace"   // OAS 3.0+ only
	MethodConnect = "connect" // Standard HTTP method, rarely used in APIs
	MethodQuery   = "query"   // OAS 3.2+ only
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

// ResponsesKeyDefault is the Responses Object key holding the default response.
const ResponsesKeyDefault = "default"

// ExtensionPrefix marks a specification extension field.
const ExtensionPrefix = "x-"

// IsExtensionKey reports whether key names a specification extension.
func IsExtensionKey(key string) bool {
	return strings.HasPrefix(key, ExtensionPrefix)
}

// IsValidResponsesKey reports whether key may appear in a Responses Object.
// That is a broader question than [IsStatusCode]: the Responses Object also
// admits "default" and specification extensions.
//
// Use this to accept or reject a key while decoding. Use [IsStatusCode] to ask
// whether the key denotes a status code, which is what a caller ranging
// Responses.Codes means.
func IsValidResponsesKey(key string) bool {
	return key == ResponsesKeyDefault || IsExtensionKey(key) || IsStatusCode(key)
}

// IsStatusCode reports whether code is an HTTP status code or a wildcard range:
//   - Wildcard patterns: 1XX, 2XX, 3XX, 4XX, 5XX
//   - Numeric codes: 100-599
//
// It is deliberately narrow. "default" and specification extensions are legal
// Responses Object keys but are not status codes, so both return false here;
// [IsValidResponsesKey] is the predicate that accepts them.
//
// It is also version-blind, and the two forms it accepts are not defined by the
// same OAS versions. A caller that knows the document version wants
// [IsNumericStatusCode] and [IsWildcardStatusCode] instead.
func IsStatusCode(code string) bool {
	return IsNumericStatusCode(code) || IsWildcardStatusCode(code)
}

// IsWildcardStatusCode reports whether code is a wildcard response range: 1XX,
// 2XX, 3XX, 4XX or 5XX.
//
// OAS 3.0 introduced these. The OAS 2.0 Responses Object states only that "any
// HTTP status code can be used as the property name (one property per HTTP
// status code)", so a 2.0 document may not use one.
//
// https://spec.openapis.org/oas/v2.0.html#responses-object
func IsWildcardStatusCode(code string) bool {
	if len(code) != StatusCodeLength {
		return false
	}
	if code[1] != WildcardChar || code[2] != WildcardChar {
		return false
	}
	return code[0] >= minWildcardBoundary && code[0] <= maxWildcardBoundary
}

// IsNumericStatusCode reports whether code is a numeric HTTP status code from
// [MinStatusCode] to [MaxStatusCode]. Every OAS version defines these, unlike
// the wildcard ranges [IsWildcardStatusCode] recognizes.
func IsNumericStatusCode(code string) bool {
	if len(code) != StatusCodeLength {
		return false
	}
	if code[0] < '0' || code[0] > '9' ||
		code[1] < '0' || code[1] > '9' ||
		code[2] < '0' || code[2] > '9' {
		return false
	}
	// Three ASCII digits, so the value is computed rather than parsed: strconv
	// could not fail here, and there would be no answer to give if it did.
	value := int(code[0]-'0')*100 + int(code[1]-'0')*10 + int(code[2]-'0')
	return value >= MinStatusCode && value <= MaxStatusCode
}

// IsSuccessStatusCode reports whether code denotes a successful response: a
// numeric 2xx code, or the 2XX wildcard range.
//
// Only a status code can be a successful one, so anything [IsStatusCode]
// rejects is rejected here too. That matters for a caller ranging
// Responses.Codes, which may hold a key no decoder validated.
func IsSuccessStatusCode(code string) bool {
	return IsStatusCode(code) && code[0] == '2'
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
