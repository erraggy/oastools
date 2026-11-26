// This file implements format validation helpers for media types, URLs, emails,
// and SPDX license identifiers used during OAS document validation.

package validator

import (
	"mime"
	"strings"

	"github.com/erraggy/oastools/internal/stringutil"
)

// isValidMediaType checks if a media type string is valid using RFC-compliant parsing
// This uses the standard library's mime.ParseMediaType which validates according to RFC 2045 and RFC 2046.
// This allows custom and vendor-specific media types (e.g., application/vnd.custom+json).
func isValidMediaType(mediaType string) bool {
	if mediaType == "" {
		return false
	}

	// Check for wildcard patterns first (mime.ParseMediaType doesn't handle these)
	// Valid: */* (both wildcards) or type/* (subtype wildcard)
	// Invalid: */subtype (type wildcard with specific subtype)
	if strings.Contains(mediaType, "*") {
		parts := strings.Split(strings.Split(mediaType, ";")[0], "/") // Remove parameters before checking
		if len(parts) != 2 {
			return false
		}
		if parts[0] == "*" {
			return parts[1] == "*" // */subtype is invalid
		}
		if parts[1] == "*" {
			return parts[0] != "" // type/* is valid if type is not empty
		}
	}

	// Use standard library for RFC-compliant validation
	_, _, err := mime.ParseMediaType(mediaType)
	return err == nil
}

// getJSONSchemaRef returns the JSON Schema specification reference URL
func getJSONSchemaRef() string {
	return "https://www.ietf.org/archive/id/draft-bhutton-json-schema-01.html"
}

// isValidURL validates externalDocs.url and OAuth URLs. The Info, Contact and
// License Object URLs are checked by internal/inforules.
func isValidURL(s string) bool {
	return stringutil.IsValidURL(s)
}
