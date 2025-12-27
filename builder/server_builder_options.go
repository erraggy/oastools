package builder

import (
	"net/http"
	"time"
)

// ServerBuilderOption configures a ServerBuilder.
type ServerBuilderOption func(*serverBuilderConfig)

// WithRouter sets the routing strategy.
// Default: StdlibRouter (uses net/http with PathMatcherSet).
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

// WithNotFoundHandler sets the handler for unmatched paths.
func WithNotFoundHandler(handler http.Handler) ServerBuilderOption {
	return func(cfg *serverBuilderConfig) {
		cfg.notFoundHandler = handler
	}
}

// WithMethodNotAllowedHandler sets the handler for unmatched methods.
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
