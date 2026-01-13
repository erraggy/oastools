package httpvalidator

import (
	"github.com/erraggy/oastools/internal/issues"
	"github.com/erraggy/oastools/internal/severity"
)

// ValidationError represents a single HTTP validation issue.
// This is an alias to issues.Issue for consistency with other oastools packages.
type ValidationError = issues.Issue

// Severity levels for validation errors.
type Severity = severity.Severity

// Severity constants re-exported for convenience.
const (
	SeverityError    = severity.SeverityError
	SeverityWarning  = severity.SeverityWarning
	SeverityInfo     = severity.SeverityInfo
	SeverityCritical = severity.SeverityCritical
)

// ValidationLocation indicates where in the HTTP message the error occurred.
type ValidationLocation string

// Validation location constants.
const (
	LocationPath        ValidationLocation = "path"
	LocationQuery       ValidationLocation = "query"
	LocationHeader      ValidationLocation = "header"
	LocationCookie      ValidationLocation = "cookie"
	LocationRequestBody ValidationLocation = "requestBody"
	LocationResponse    ValidationLocation = "response"
)

// RequestValidationResult contains the results of validating an HTTP request
// against an OpenAPI specification.
type RequestValidationResult struct {
	// Valid is true if the request passes all validation checks.
	Valid bool

	// Errors contains all validation errors found.
	Errors []ValidationError

	// Warnings contains best-practice warnings (if IncludeWarnings is enabled).
	Warnings []ValidationError

	// MatchedPath is the OpenAPI path template that matched the request
	// (e.g., "/pets/{petId}"). Empty if no path matched.
	MatchedPath string

	// MatchedMethod is the HTTP method of the request (e.g., "GET", "POST").
	MatchedMethod string

	// PathParams contains the extracted and validated path parameters.
	// Keys are parameter names, values are the deserialized values.
	PathParams map[string]any

	// QueryParams contains the extracted and validated query parameters.
	QueryParams map[string]any

	// HeaderParams contains the extracted and validated header parameters.
	HeaderParams map[string]any

	// CookieParams contains the extracted and validated cookie parameters.
	CookieParams map[string]any
}

// ResponseValidationResult contains the results of validating an HTTP response
// against an OpenAPI specification.
type ResponseValidationResult struct {
	// Valid is true if the response passes all validation checks.
	Valid bool

	// Errors contains all validation errors found.
	Errors []ValidationError

	// Warnings contains best-practice warnings (if IncludeWarnings is enabled).
	Warnings []ValidationError

	// StatusCode is the HTTP status code of the response.
	StatusCode int

	// ContentType is the Content-Type of the response.
	ContentType string

	// MatchedPath is the OpenAPI path template that matched the original request.
	MatchedPath string

	// MatchedMethod is the HTTP method of the original request.
	MatchedMethod string
}

// newRequestResult creates a new RequestValidationResult with initialized maps.
func newRequestResult() *RequestValidationResult {
	return &RequestValidationResult{
		Valid:        true,
		PathParams:   make(map[string]any),
		QueryParams:  make(map[string]any),
		HeaderParams: make(map[string]any),
		CookieParams: make(map[string]any),
	}
}

// newResponseResult creates a new ResponseValidationResult.
func newResponseResult() *ResponseValidationResult {
	return &ResponseValidationResult{
		Valid: true,
	}
}

// addError adds an error to the request result and marks it as invalid.
func (r *RequestValidationResult) addError(path, message string, sev Severity) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{
		Path:     path,
		Message:  message,
		Severity: sev,
	})
}

// addWarning adds a warning to the request result.
func (r *RequestValidationResult) addWarning(path, message string) {
	r.Warnings = append(r.Warnings, ValidationError{
		Path:     path,
		Message:  message,
		Severity: SeverityWarning,
	})
}

// addError adds an error to the response result and marks it as invalid.
func (r *ResponseValidationResult) addError(path, message string, sev Severity) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{
		Path:     path,
		Message:  message,
		Severity: sev,
	})
}

// addWarning adds a warning to the response result.
func (r *ResponseValidationResult) addWarning(path, message string) {
	r.Warnings = append(r.Warnings, ValidationError{
		Path:     path,
		Message:  message,
		Severity: SeverityWarning,
	})
}

// reset clears the result for reuse from pool.
func (r *RequestValidationResult) reset() {
	r.Valid = true
	r.MatchedPath = ""
	r.MatchedMethod = ""
	r.Errors = r.Errors[:0]
	r.Warnings = r.Warnings[:0]
	clear(r.PathParams)
	clear(r.QueryParams)
	clear(r.HeaderParams)
	clear(r.CookieParams)
}

// reset clears the result for reuse from pool.
func (r *ResponseValidationResult) reset() {
	r.Valid = true
	r.MatchedPath = ""
	r.MatchedMethod = ""
	r.StatusCode = 0
	r.ContentType = ""
	r.Errors = r.Errors[:0]
	r.Warnings = r.Warnings[:0]
}
