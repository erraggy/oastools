// Package oaserrors provides structured error types for the oastools library.
//
// Import path: github.com/erraggy/oastools/oaserrors
//
// This package enables programmatic error handling via [errors.Is] and [errors.As],
// allowing callers to distinguish between different categories of errors and implement
// appropriate recovery strategies.
//
// # Error Types
//
// The package provides six core error types:
//
//   - [ParseError]: YAML/JSON parsing failures and structural issues
//   - [ReferenceError]: $ref resolution failures, circular references, path traversal
//   - [ValidationError]: OpenAPI specification violations
//   - [ResourceLimitError]: Resource exhaustion (depth, size, count limits)
//   - [ConversionError]: Version conversion failures between OAS versions
//   - [ConfigError]: Invalid configuration or input options
//
// # Sentinel Errors
//
// Each error type has a corresponding sentinel error for use with errors.Is():
//
//   - [ErrParse]: Matches any [ParseError]
//   - [ErrReference]: Matches any [ReferenceError]
//   - [ErrCircularReference]: Matches [ReferenceError] with IsCircular=true
//   - [ErrPathTraversal]: Matches [ReferenceError] with IsPathTraversal=true
//   - [ErrValidation]: Matches any [ValidationError]
//   - [ErrResourceLimit]: Matches any [ResourceLimitError]
//   - [ErrConversion]: Matches any [ConversionError]
//   - [ErrConfig]: Matches any [ConfigError]
//
// # Usage Examples
//
// Check error category with errors.Is():
//
//	result, err := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
//	if errors.Is(err, oaserrors.ErrParse) {
//	    // Handle parse error
//	}
//
// Extract error details with errors.As():
//
//	var refErr *oaserrors.ReferenceError
//	if errors.As(err, &refErr) {
//	    fmt.Printf("Failed to resolve ref: %s\n", refErr.Ref)
//	    if refErr.IsCircular {
//	        // Handle circular reference specifically
//	    }
//	}
//
// Check for specific conditions:
//
//	if errors.Is(err, oaserrors.ErrCircularReference) {
//	    // Circular reference detected - may be recoverable
//	}
//	if errors.Is(err, oaserrors.ErrPathTraversal) {
//	    // Security issue - log and reject
//	}
//
// # Error Chaining
//
// All error types support error chaining via the Cause field and Unwrap() method.
// This allows finding root causes through the standard error chain:
//
//	var refErr *oaserrors.ReferenceError
//	if errors.As(err, &refErr) {
//	    if errors.Is(refErr.Cause, os.ErrNotExist) {
//	        // The reference file doesn't exist
//	    }
//	}
package oaserrors
