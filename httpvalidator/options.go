package httpvalidator

import (
	"fmt"
	"net/http"

	"github.com/erraggy/oastools/parser"
)

// Option is a functional option for configuring validation.
type Option func(*config) error

// config holds the configuration for validation operations.
type config struct {
	// Spec source (one of these must be set)
	filePath string
	parsed   *parser.ParseResult

	// Validation behavior
	includeWarnings bool
	strictMode      bool

	// Skip options
	skipBodyValidation   bool
	skipQueryValidation  bool
	skipHeaderValidation bool
	skipCookieValidation bool

	// Resource limits
	maxBodySize int64 // max request/response body size (0 = default 10 MiB)
}

// defaultConfig returns the default configuration.
func defaultConfig() *config {
	return &config{
		includeWarnings: true,
		strictMode:      false,
	}
}

// WithFilePath sets the path to the OpenAPI specification file.
// The file will be parsed automatically.
func WithFilePath(path string) Option {
	return func(c *config) error {
		c.filePath = path
		return nil
	}
}

// WithParsed uses a pre-parsed OpenAPI specification.
// This is more efficient when validating multiple requests.
func WithParsed(result *parser.ParseResult) Option {
	return func(c *config) error {
		if result == nil {
			return fmt.Errorf("httpvalidator: parsed result cannot be nil")
		}
		c.parsed = result
		return nil
	}
}

// WithIncludeWarnings sets whether to include best-practice warnings.
// Default is true.
func WithIncludeWarnings(include bool) Option {
	return func(c *config) error {
		c.includeWarnings = include
		return nil
	}
}

// WithStrictMode enables stricter validation:
//   - Rejects requests with unknown query parameters
//   - Rejects requests with unknown headers (except standard HTTP headers)
//   - Rejects requests with unknown cookies
//   - Rejects responses with undocumented status codes
//
// Default is false.
func WithStrictMode(strict bool) Option {
	return func(c *config) error {
		c.strictMode = strict
		return nil
	}
}

// WithSkipBodyValidation skips request/response body validation.
// Useful when body validation is too expensive or handled elsewhere.
func WithSkipBodyValidation(skip bool) Option {
	return func(c *config) error {
		c.skipBodyValidation = skip
		return nil
	}
}

// WithSkipQueryValidation skips query parameter validation.
func WithSkipQueryValidation(skip bool) Option {
	return func(c *config) error {
		c.skipQueryValidation = skip
		return nil
	}
}

// WithSkipHeaderValidation skips header parameter validation.
func WithSkipHeaderValidation(skip bool) Option {
	return func(c *config) error {
		c.skipHeaderValidation = skip
		return nil
	}
}

// WithSkipCookieValidation skips cookie parameter validation.
func WithSkipCookieValidation(skip bool) Option {
	return func(c *config) error {
		c.skipCookieValidation = skip
		return nil
	}
}

// WithMaxBodySize sets the maximum request/response body size in bytes.
// Bodies exceeding this limit will produce a validation error.
// Default: 10 MiB.
func WithMaxBodySize(n int64) Option {
	return func(c *config) error {
		if n < 0 {
			return fmt.Errorf("httpvalidator: maxBodySize cannot be negative")
		}
		c.maxBodySize = n
		return nil
	}
}

// ValidateRequestWithOptions validates an HTTP request against an OpenAPI specification
// using functional options.
//
// This is a convenience function for one-off validations. For validating multiple
// requests, use New() to create a reusable Validator instance.
//
// Example:
//
//	result, err := httpvalidator.ValidateRequestWithOptions(
//	    req,
//	    httpvalidator.WithFilePath("openapi.yaml"),
//	    httpvalidator.WithStrictMode(true),
//	)
func ValidateRequestWithOptions(req *http.Request, opts ...Option) (*RequestValidationResult, error) {
	cfg := defaultConfig()

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Get parsed spec
	parsed, err := getParsedSpec(cfg)
	if err != nil {
		return nil, err
	}

	// Create validator
	v, err := New(parsed)
	if err != nil {
		return nil, err
	}

	// Apply config
	v.IncludeWarnings = cfg.includeWarnings
	v.StrictMode = cfg.strictMode
	v.maxBodySize = cfg.maxBodySize

	// For skip options, we need to create a wrapper that respects them
	if cfg.skipBodyValidation || cfg.skipQueryValidation || cfg.skipHeaderValidation || cfg.skipCookieValidation {
		return v.validateRequestWithSkips(req, cfg)
	}

	return v.ValidateRequest(req)
}

// ValidateResponseWithOptions validates an HTTP response against an OpenAPI specification
// using functional options.
//
// This is a convenience function for one-off validations. For validating multiple
// responses, use New() to create a reusable Validator instance.
//
// Example:
//
//	result, err := httpvalidator.ValidateResponseWithOptions(
//	    req, resp,
//	    httpvalidator.WithFilePath("openapi.yaml"),
//	    httpvalidator.WithIncludeWarnings(false),
//	)
func ValidateResponseWithOptions(req *http.Request, resp *http.Response, opts ...Option) (*ResponseValidationResult, error) {
	cfg := defaultConfig()

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Get parsed spec
	parsed, err := getParsedSpec(cfg)
	if err != nil {
		return nil, err
	}

	// Create validator
	v, err := New(parsed)
	if err != nil {
		return nil, err
	}

	// Apply config
	v.IncludeWarnings = cfg.includeWarnings
	v.StrictMode = cfg.strictMode
	v.maxBodySize = cfg.maxBodySize

	return v.ValidateResponse(req, resp)
}

// ValidateResponseDataWithOptions validates response data (for middleware use)
// using functional options.
//
// This is useful in middleware where you've captured response parts in a
// ResponseRecorder but don't have an *http.Response.
//
// Example:
//
//	result, err := httpvalidator.ValidateResponseDataWithOptions(
//	    req, recorder.Code, recorder.Header(), recorder.Body.Bytes(),
//	    httpvalidator.WithFilePath("openapi.yaml"),
//	)
func ValidateResponseDataWithOptions(req *http.Request, statusCode int, headers http.Header, body []byte, opts ...Option) (*ResponseValidationResult, error) {
	cfg := defaultConfig()

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Get parsed spec
	parsed, err := getParsedSpec(cfg)
	if err != nil {
		return nil, err
	}

	// Create validator
	v, err := New(parsed)
	if err != nil {
		return nil, err
	}

	// Apply config
	v.IncludeWarnings = cfg.includeWarnings
	v.StrictMode = cfg.strictMode
	v.maxBodySize = cfg.maxBodySize

	return v.ValidateResponseData(req, statusCode, headers, body)
}

// getParsedSpec returns the parsed specification from config.
func getParsedSpec(cfg *config) (*parser.ParseResult, error) {
	if cfg.parsed != nil {
		return cfg.parsed, nil
	}

	if cfg.filePath != "" {
		return parser.ParseWithOptions(parser.WithFilePath(cfg.filePath))
	}

	return nil, fmt.Errorf("httpvalidator: no specification provided (use WithFilePath or WithParsed)")
}

// validateRequestWithSkips validates a request but skips certain validations.
func (v *Validator) validateRequestWithSkips(req *http.Request, cfg *config) (*RequestValidationResult, error) {
	result := newRequestResult()

	// 1. Find matching path
	matchedPath, pathParams, found := v.matchPath(req.URL.Path)
	if !found {
		sanitizedPath := truncateForError(req.URL.Path, maxErrorValueLen)
		result.addError("request.path", fmt.Sprintf("no matching path found for %q", sanitizedPath), SeverityError)
		return result, nil
	}
	result.MatchedPath = matchedPath
	result.MatchedMethod = req.Method

	// 2. Get operation for method
	operation := v.getOperation(matchedPath, req.Method)
	if operation == nil {
		result.addError(
			fmt.Sprintf("%s.%s", matchedPath, req.Method),
			fmt.Sprintf("method %s not allowed for path %s", req.Method, matchedPath),
			SeverityError,
		)
		return result, nil
	}

	// Snapshot mutable fields for consistent behavior within this call.
	flags := validationFlags{
		strictMode:      v.StrictMode,
		includeWarnings: v.IncludeWarnings,
	}

	// 3. Validate path parameters (always)
	v.validatePathParams(pathParams, matchedPath, operation, result)

	// 4. Validate query parameters (if not skipped)
	if !cfg.skipQueryValidation {
		v.validateQueryParams(req, matchedPath, operation, result, flags)
	}

	// 5. Validate header parameters (if not skipped)
	if !cfg.skipHeaderValidation {
		v.validateHeaderParams(req, matchedPath, operation, result, flags)
	}

	// 6. Validate cookie parameters (if not skipped)
	if !cfg.skipCookieValidation {
		v.validateCookieParams(req, matchedPath, operation, result, flags)
	}

	// 7. Validate request body (if not skipped)
	if !cfg.skipBodyValidation {
		v.validateRequestBody(req, matchedPath, operation, result, flags)
	}

	return result, nil
}
