package validator

import (
	"fmt"
	"mime"
	"net/url"
	"regexp"
	"strings"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/parser"
)

// Compile regex once at package level for performance
var (
	pathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)
	emailRegex     = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// validateInfoObject validates the info object fields shared between OAS 2.0 and 3.x.
// Set validateSPDX to true for OAS 3.1+ to validate the SPDX license identifier.
func (v *Validator) validateInfoObject(info *parser.Info, result *ValidationResult, baseURL string, validateSPDX bool) {
	if info.Title == "" {
		v.addError(result, "info.title", "Info object must have a title",
			withSpecRef(fmt.Sprintf("%s#info-object", baseURL)),
			withField("title"),
		)
	}

	if info.Version == "" {
		v.addError(result, "info.version", "Info object must have a version",
			withSpecRef(fmt.Sprintf("%s#info-object", baseURL)),
			withField("version"),
		)
	}

	// Validate contact information if present
	if info.Contact != nil {
		if info.Contact.URL != "" && !isValidURL(info.Contact.URL) {
			v.addError(result, "info.contact.url", fmt.Sprintf("Invalid URL format: %s", info.Contact.URL),
				withSpecRef(fmt.Sprintf("%s#contact-object", baseURL)),
				withField("url"),
				withValue(info.Contact.URL),
			)
		}
		if info.Contact.Email != "" && !isValidEmail(info.Contact.Email) {
			v.addError(result, "info.contact.email", fmt.Sprintf("Invalid email format: %s", info.Contact.Email),
				withSpecRef(fmt.Sprintf("%s#contact-object", baseURL)),
				withField("email"),
				withValue(info.Contact.Email),
			)
		}
	}

	// Validate license information if present
	if info.License != nil {
		if info.License.URL != "" && !isValidURL(info.License.URL) {
			v.addError(result, "info.license.url", fmt.Sprintf("Invalid URL format: %s", info.License.URL),
				withSpecRef(fmt.Sprintf("%s#license-object", baseURL)),
				withField("url"),
				withValue(info.License.URL),
			)
		}
		// SPDX license identifier validation (OAS 3.1+)
		if validateSPDX && info.License.Identifier != "" && !validateSPDXLicense(info.License.Identifier) {
			v.addError(result, "info.license.identifier", fmt.Sprintf("Invalid SPDX license identifier format: %s", info.License.Identifier),
				withSpecRef(fmt.Sprintf("%s#license-object", baseURL)),
				withField("identifier"),
				withValue(info.License.Identifier),
			)
		}
	}
}

// validateResponseStatusCodes validates HTTP status codes in an operation's responses.
// This helper is shared by both OAS 2.0 and OAS 3.x operation validators.
func (v *Validator) validateResponseStatusCodes(responses *parser.Responses, path string, result *ValidationResult, baseURL string) {
	if responses == nil || responses.Codes == nil {
		return
	}

	hasSuccess := false
	for code := range responses.Codes {
		// Validate HTTP status code format
		if !httputil.ValidateStatusCode(code) {
			result.Errors = append(result.Errors, ValidationError{
				Path:     fmt.Sprintf("%s.responses.%s", path, code),
				Message:  fmt.Sprintf("Invalid HTTP status code: %s", code),
				SpecRef:  fmt.Sprintf("%s#responses-object", baseURL),
				Severity: SeverityError,
				Value:    code,
			})
		} else if v.StrictMode && !httputil.IsStandardStatusCode(code) {
			// In strict mode, warn about non-standard status codes
			result.Warnings = append(result.Warnings, ValidationError{
				Path:     fmt.Sprintf("%s.responses.%s", path, code),
				Message:  fmt.Sprintf("Non-standard HTTP status code: %s (not defined in HTTP RFCs)", code),
				SpecRef:  fmt.Sprintf("%s#responses-object", baseURL),
				Severity: SeverityWarning,
				Value:    code,
			})
		}

		if strings.HasPrefix(code, "2") || code == "default" {
			hasSuccess = true
		}
	}
	if !hasSuccess && v.StrictMode {
		result.Warnings = append(result.Warnings, ValidationError{
			Path:     fmt.Sprintf("%s.responses", path),
			Message:  "Operation should define at least one successful response (2XX or default)",
			SpecRef:  fmt.Sprintf("%s#responses-object", baseURL),
			Severity: SeverityWarning,
		})
	}
}

// checkDuplicateOperationIds checks for duplicate operationIds in a set of operations
// and reports errors when found. Updates the operationIds map as it processes operations.
func (v *Validator) checkDuplicateOperationIds(
	operations map[string]*parser.Operation,
	pathType string,
	pathPattern string,
	operationIds map[string]string,
	result *ValidationResult,
	baseURL string,
) {
	for method, op := range operations {
		if op == nil || op.OperationID == "" {
			continue
		}

		opPath := fmt.Sprintf("%s.%s.%s", pathType, pathPattern, method)

		if firstSeenAt, exists := operationIds[op.OperationID]; exists {
			// Determine the correct spec reference based on path type
			specRef := fmt.Sprintf("%s#operation-object", baseURL)
			if pathType == "webhooks" || strings.Contains(baseURL, "v3") {
				specRef = fmt.Sprintf("%s#operation-object", baseURL)
			}

			result.Errors = append(result.Errors, ValidationError{
				Path:     opPath,
				Message:  fmt.Sprintf("Duplicate operationId '%s' (first seen at %s)", op.OperationID, firstSeenAt),
				SpecRef:  specRef,
				Severity: SeverityError,
				Field:    "operationId",
				Value:    op.OperationID,
			})
		} else {
			operationIds[op.OperationID] = opPath
		}
	}
}

// validatePathTemplate validates that a path template is well-formed
// Returns an error if the template is malformed (unclosed braces, empty parameters, etc.)
func validatePathTemplate(pathPattern string) error {
	// Check for empty braces explicitly (regex won't catch {})
	if strings.Contains(pathPattern, "{}") {
		return fmt.Errorf("empty parameter name in path template")
	}

	// Check for consecutive slashes
	if strings.Contains(pathPattern, "//") {
		return fmt.Errorf("path contains consecutive slashes")
	}

	// Check for reserved characters (fragment identifier and query string)
	if strings.Contains(pathPattern, "#") {
		return fmt.Errorf("path contains reserved character '#'")
	}
	if strings.Contains(pathPattern, "?") {
		return fmt.Errorf("path contains reserved character '?'")
	}

	// Note: Trailing slashes are handled separately as warnings, not errors
	// Empty segments in the middle are caught by the consecutive slash check above

	// Check for unclosed or unopened braces
	openCount := 0
	for i, ch := range pathPattern {
		switch ch {
		case '{':
			openCount++
			if openCount > 1 {
				return fmt.Errorf("nested braces are not allowed at position %d", i)
			}
		case '}':
			openCount--
			if openCount < 0 {
				return fmt.Errorf("unexpected closing brace at position %d", i)
			}
		}
	}
	if openCount != 0 {
		return fmt.Errorf("unclosed brace in path template")
	}

	// Check for empty or invalid parameters, and track duplicates
	paramNames := make(map[string]bool)
	matches := pathParamRegex.FindAllStringSubmatch(pathPattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			paramName := match[1]
			if strings.TrimSpace(paramName) == "" {
				return fmt.Errorf("empty parameter name in path template")
			}
			// Check for invalid characters in parameter name
			if strings.Contains(paramName, "{") || strings.Contains(paramName, "}") {
				return fmt.Errorf("invalid parameter name '%s' contains braces", paramName)
			}
			// Check for duplicate parameter names
			if paramNames[paramName] {
				return fmt.Errorf("duplicate parameter name '%s' in path template", paramName)
			}
			paramNames[paramName] = true
		}
	}

	return nil
}

// checkTrailingSlash adds a warning if the path has a trailing slash
// Trailing slashes are discouraged by REST best practices but not forbidden by OAS spec
func checkTrailingSlash(v *Validator, pathPattern string, result *ValidationResult, baseURL string) {
	if v.IncludeWarnings && len(pathPattern) > 1 && strings.HasSuffix(pathPattern, "/") {
		result.Warnings = append(result.Warnings, ValidationError{
			Path:     fmt.Sprintf("paths.%s", pathPattern),
			Message:  "Path has trailing slash, which is discouraged by REST best practices",
			SpecRef:  fmt.Sprintf("%s#paths-object", baseURL),
			Severity: SeverityWarning,
			Value:    pathPattern,
		})
	}
}

// extractPathParameters extracts parameter names from a path template
// e.g., "/pets/{petId}/owners/{ownerId}" -> {"petId": true, "ownerId": true}
func extractPathParameters(pathPattern string) map[string]bool {
	params := make(map[string]bool)
	matches := pathParamRegex.FindAllStringSubmatch(pathPattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params[match[1]] = true
		}
	}
	return params
}

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

// isValidURL performs URL validation using standard library's url.Parse
// Validates contact.url, externalDocs.url, license.url, and OAuth URLs
func isValidURL(s string) bool {
	if s == "" {
		return false
	}

	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	// Accept http/https schemes, or relative URLs starting with /
	// Reject bare strings without proper URL structure
	if u.Scheme == "http" || u.Scheme == "https" {
		return true
	}
	if u.Scheme == "" && strings.HasPrefix(s, "/") {
		return true
	}
	return false
}

// isValidEmail performs email validation using regex
// Validates contact.email in the info object
func isValidEmail(s string) bool {
	if s == "" {
		return true // Empty is valid (optional field)
	}
	return emailRegex.MatchString(s)
}

// validateSPDXLicense validates SPDX license identifier (basic validation)
// Used to validate license.identifier in the info object (OAS 3.1+)
func validateSPDXLicense(identifier string) bool {
	if identifier == "" {
		return true
	}
	// Basic validation - should not contain spaces and follow SPDX format
	// For a complete implementation, you'd need the full SPDX license list
	return !strings.Contains(identifier, " ")
}
