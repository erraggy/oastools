package builder

import (
	"net/http"
	"time"
)

// ServerBuilderOption configures a ServerBuilder.
type ServerBuilderOption func(*serverBuilderConfig)

// WithRouter sets the routing strategy.
// Default: the stdlib router (net/http with PathMatcherSet), which WithStdlibRouter selects.
func WithRouter(strategy RouterStrategy) ServerBuilderOption {
	return func(cfg *serverBuilderConfig) {
		cfg.router = strategy
	}
}

// WithStdlibRouter uses net/http with PathMatcherSet for routing.
// This is the default and adds no dependencies.
func WithStdlibRouter() ServerBuilderOption {
	return func(cfg *serverBuilderConfig) {
		cfg.router = &stdlibRouter{}
	}
}

// WithoutValidation disables automatic request validation.
// Use when validation is handled elsewhere or for maximum performance.
func WithoutValidation() ServerBuilderOption {
	return func(cfg *serverBuilderConfig) {
		cfg.enableValidation = false
	}
}

// WithValidationConfig sets validation middleware configuration.
func WithValidationConfig(validationCfg ValidationConfig) ServerBuilderOption {
	return func(cfg *serverBuilderConfig) {
		cfg.validationConfig = validationCfg
	}
}

// WithErrorHandler sets the error handler for handler panics and errors.
func WithErrorHandler(handler ErrorHandler) ServerBuilderOption {
	return func(cfg *serverBuilderConfig) {
		cfg.errorHandler = handler
	}
}

// WithNotFoundHandler sets the handler that serves every 404 the server
// produces, which is a request whose path matches no operation. It is called
// whether or not request validation is enabled, since routing runs first.
// Default: http.NotFound.
func WithNotFoundHandler(handler http.Handler) ServerBuilderOption {
	return func(cfg *serverBuilderConfig) {
		cfg.notFoundHandler = handler
	}
}

// WithMethodNotAllowedHandler sets the handler that serves the 405 for a
// request whose path matches an operation but whose method does not. The
// Allow header listing the path's methods is already set when the handler is
// called. It is called whether or not request validation is enabled, since
// routing runs first. Default: a plain-text "Method Not Allowed" response.
func WithMethodNotAllowedHandler(handler http.Handler) ServerBuilderOption {
	return func(cfg *serverBuilderConfig) {
		cfg.methodNotAllowed = handler
	}
}

// WithRecovery enables panic recovery middleware.
// Recovered panics are passed to the error handler.
func WithRecovery() ServerBuilderOption {
	return func(cfg *serverBuilderConfig) {
		cfg.enableRecovery = true
	}
}

// WithRequestLogging enables request logging middleware.
func WithRequestLogging(logger func(method, path string, status int, duration time.Duration)) ServerBuilderOption {
	return func(cfg *serverBuilderConfig) {
		cfg.requestLogger = logger
	}
}
