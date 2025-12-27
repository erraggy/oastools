package builder

import (
	"context"
	"net/http"
	"time"

	"github.com/erraggy/oastools/httpvalidator"
	"github.com/erraggy/oastools/parser"
)

// HandlerFunc is the signature for operation handlers.
// The Request contains validated parameters and the raw http.Request.
// The Response interface allows type-safe response construction.
type HandlerFunc func(ctx context.Context, req *Request) Response

// Request contains validated request data passed to operation handlers.
type Request struct {
	// HTTPRequest is the original HTTP request.
	HTTPRequest *http.Request

	// PathParams contains the extracted and validated path parameters.
	// Keys are parameter names, values are deserialized values.
	PathParams map[string]any

	// QueryParams contains the extracted and validated query parameters.
	QueryParams map[string]any

	// HeaderParams contains the extracted and validated header parameters.
	HeaderParams map[string]any

	// CookieParams contains the extracted and validated cookie parameters.
	CookieParams map[string]any

	// Body is the unmarshaled request body (typically a map or struct).
	Body any

	// RawBody is the raw request body bytes.
	RawBody []byte

	// OperationID is the operation ID for this request.
	OperationID string

	// MatchedPath is the OpenAPI path template that matched (e.g., "/pets/{petId}").
	MatchedPath string
}

// Response is implemented by response types.
// All response helpers return types that implement this interface.
type Response interface {
	// StatusCode returns the HTTP status code.
	StatusCode() int

	// Headers returns the HTTP headers to include in the response.
	Headers() http.Header

	// Body returns the response body (may be nil for no-content responses).
	Body() any

	// WriteTo writes the response to the ResponseWriter.
	WriteTo(w http.ResponseWriter) error
}

// ServerResult contains the built server and related artifacts.
type ServerResult struct {
	// Handler is the HTTP handler ready to serve requests.
	Handler http.Handler

	// Spec is the built OAS document (*parser.OAS3Document or *parser.OAS2Document).
	Spec any

	// ParseResult is the parse result for compatibility with other packages.
	ParseResult *parser.ParseResult

	// Validator is the httpvalidator instance (nil if validation is disabled).
	Validator *httpvalidator.Validator
}

// Middleware wraps handlers with additional behavior.
type Middleware func(http.Handler) http.Handler

// RouterStrategy defines how paths are matched to handlers.
// The stdlib router and chi router both implement this interface.
type RouterStrategy interface {
	// Build creates an http.Handler that routes requests to the dispatcher.
	Build(routes []operationRoute, dispatcher http.Handler) http.Handler

	// PathParam extracts a path parameter from the request context.
	PathParam(r *http.Request, name string) string
}

// ErrorHandler handles errors during request processing.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// operationRoute represents a registered route.
type operationRoute struct {
	Method      string
	Path        string
	OperationID string
	Handler     HandlerFunc
}

// ValidationConfig configures request/response validation.
type ValidationConfig struct {
	// IncludeRequestValidation enables request validation (default: true).
	IncludeRequestValidation bool

	// IncludeResponseValidation enables response validation (default: false).
	IncludeResponseValidation bool

	// StrictMode treats warnings as errors (default: false).
	StrictMode bool

	// OnValidationError is called when validation fails.
	// If nil, a default JSON error response is returned.
	OnValidationError ValidationErrorHandler
}

// ValidationErrorHandler handles validation failures.
type ValidationErrorHandler func(w http.ResponseWriter, r *http.Request, result *httpvalidator.RequestValidationResult)

// DefaultValidationConfig returns sensible defaults for validation.
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		IncludeRequestValidation:  true,
		IncludeResponseValidation: false,
		StrictMode:                false,
		OnValidationError:         nil,
	}
}

// serverBuilderConfig holds configuration for the ServerBuilder.
type serverBuilderConfig struct {
	router               RouterStrategy
	errorHandler         ErrorHandler
	enableValidation     bool
	validationConfig     ValidationConfig
	notFoundHandler      http.Handler
	methodNotAllowed     http.Handler
	enableRecovery       bool
	requestLogger        func(method, path string, status int, duration time.Duration)
}

// defaultServerBuilderConfig returns default server builder configuration.
func defaultServerBuilderConfig() serverBuilderConfig {
	return serverBuilderConfig{
		router:           &stdlibRouter{},
		errorHandler:     defaultErrorHandler,
		enableValidation: true,
		validationConfig: DefaultValidationConfig(),
	}
}

// defaultErrorHandler is the default error handler for the server.
func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
