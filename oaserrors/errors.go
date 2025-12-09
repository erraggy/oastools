// Package oaserrors provides structured error types for oastools.
//
// These error types enable programmatic error handling via errors.Is() and
// errors.As(), allowing callers to distinguish between different categories
// of errors and implement appropriate recovery strategies.
//
// # Error Categories
//
//   - ParseError: YAML/JSON parsing failures and structural issues
//   - ReferenceError: $ref resolution failures, circular references, path traversal
//   - ValidationError: OpenAPI specification violations
//   - ResourceLimitError: Resource exhaustion (depth, size, count limits)
//   - ConversionError: Version conversion failures between OAS versions
//   - ConfigError: Invalid configuration or input options
//
// # Usage with errors.Is
//
//	result, err := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
//	if err != nil {
//	    var refErr *oaserrors.ReferenceError
//	    if errors.As(err, &refErr) {
//	        if refErr.IsCircular {
//	            // Handle circular reference specifically
//	        }
//	    }
//	}
package oaserrors

import (
	"errors"
	"fmt"
)

// Sentinel errors for use with errors.Is().
// These allow quick checks without type assertions.
var (
	// ErrParse indicates a parsing failure occurred.
	ErrParse = errors.New("parse error")

	// ErrReference indicates a reference resolution failure.
	ErrReference = errors.New("reference error")

	// ErrCircularReference indicates a circular $ref was detected.
	ErrCircularReference = errors.New("circular reference")

	// ErrPathTraversal indicates a path traversal attempt was blocked.
	ErrPathTraversal = errors.New("path traversal detected")

	// ErrValidation indicates a specification validation failure.
	ErrValidation = errors.New("validation error")

	// ErrResourceLimit indicates a resource limit was exceeded.
	ErrResourceLimit = errors.New("resource limit exceeded")

	// ErrConversion indicates a version conversion failure.
	ErrConversion = errors.New("conversion error")

	// ErrConfig indicates an invalid configuration.
	ErrConfig = errors.New("configuration error")
)

// ParseError represents a failure to parse an OpenAPI document.
// This includes YAML/JSON deserialization errors and structural issues.
type ParseError struct {
	// Path is the file path or source identifier
	Path string
	// Line is the line number where the error occurred (0 if unknown)
	Line int
	// Column is the column number where the error occurred (0 if unknown)
	Column int
	// Message describes the parsing failure
	Message string
	// Cause is the underlying error, if any
	Cause error
}

// Error returns a human-readable error message.
func (e *ParseError) Error() string {
	msg := "parse error"
	if e.Path != "" {
		msg += " in " + e.Path
	}
	if e.Line > 0 {
		msg += fmt.Sprintf(" at line %d", e.Line)
		if e.Column > 0 {
			msg += fmt.Sprintf(", column %d", e.Column)
		}
	}
	if e.Message != "" {
		msg += ": " + e.Message
	}
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

// Unwrap returns the underlying cause for error chaining.
func (e *ParseError) Unwrap() error {
	return e.Cause
}

// Is reports whether target matches this error type.
func (e *ParseError) Is(target error) bool {
	return target == ErrParse
}

// ReferenceError represents a failure to resolve a $ref.
// This includes missing references, circular references, and path traversal attempts.
type ReferenceError struct {
	// Ref is the reference string that failed to resolve
	Ref string
	// RefType indicates the reference type: "local", "file", or "http"
	RefType string
	// IsCircular is true if this error is due to a circular reference
	IsCircular bool
	// IsPathTraversal is true if this error is due to a path traversal attempt
	IsPathTraversal bool
	// Message provides additional context about the failure
	Message string
	// Cause is the underlying error, if any
	Cause error
}

// Error returns a human-readable error message.
func (e *ReferenceError) Error() string {
	msg := "reference error"
	if e.IsCircular {
		msg = "circular reference"
	} else if e.IsPathTraversal {
		msg = "path traversal detected"
	}
	if e.Ref != "" {
		msg += ": " + e.Ref
	}
	if e.Message != "" {
		msg += ": " + e.Message
	}
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

// Unwrap returns the underlying cause for error chaining.
func (e *ReferenceError) Unwrap() error {
	return e.Cause
}

// Is reports whether target matches this error type.
// Matches ErrReference, and also ErrCircularReference or ErrPathTraversal
// when appropriate flags are set.
func (e *ReferenceError) Is(target error) bool {
	if target == ErrReference {
		return true
	}
	if target == ErrCircularReference && e.IsCircular {
		return true
	}
	if target == ErrPathTraversal && e.IsPathTraversal {
		return true
	}
	return false
}

// ValidationError represents an OpenAPI specification violation.
type ValidationError struct {
	// Path is the JSON path to the problematic field (e.g., "paths./pets.get.responses")
	Path string
	// Field is the specific field name with the issue
	Field string
	// Value is the problematic value (may be nil)
	Value any
	// Message describes the validation failure
	Message string
	// SpecRef is a URL to the relevant OAS specification section
	SpecRef string
	// Cause is the underlying error, if any
	Cause error
}

// Error returns a human-readable error message.
func (e *ValidationError) Error() string {
	msg := "validation error"
	if e.Path != "" {
		msg += " at " + e.Path
	}
	if e.Field != "" {
		msg += "." + e.Field
	}
	if e.Message != "" {
		msg += ": " + e.Message
	}
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

// Unwrap returns the underlying cause for error chaining.
func (e *ValidationError) Unwrap() error {
	return e.Cause
}

// Is reports whether target matches this error type.
func (e *ValidationError) Is(target error) bool {
	return target == ErrValidation
}

// ResourceLimitError represents a resource exhaustion condition.
// This occurs when parsing or validation exceeds configured limits.
type ResourceLimitError struct {
	// ResourceType identifies what limit was exceeded
	// Common values: "ref_depth", "cached_documents", "file_size", "nesting_depth"
	ResourceType string
	// Limit is the configured maximum value
	Limit int64
	// Actual is the value that exceeded the limit (may be 0 if unknown)
	Actual int64
	// Message provides additional context
	Message string
}

// Error returns a human-readable error message.
func (e *ResourceLimitError) Error() string {
	msg := "resource limit exceeded"
	if e.ResourceType != "" {
		msg += ": " + e.ResourceType
	}
	if e.Limit > 0 {
		msg += fmt.Sprintf(" (limit: %d", e.Limit)
		if e.Actual > 0 {
			msg += fmt.Sprintf(", actual: %d", e.Actual)
		}
		msg += ")"
	}
	if e.Message != "" {
		msg += ": " + e.Message
	}
	return msg
}

// Unwrap returns nil as ResourceLimitError has no underlying cause.
func (e *ResourceLimitError) Unwrap() error {
	return nil
}

// Is reports whether target matches this error type.
func (e *ResourceLimitError) Is(target error) bool {
	return target == ErrResourceLimit
}

// ConversionError represents a failure during OAS version conversion.
type ConversionError struct {
	// SourceVersion is the source OAS version (e.g., "2.0", "3.0.3")
	SourceVersion string
	// TargetVersion is the target OAS version
	TargetVersion string
	// Path is the JSON path where conversion failed
	Path string
	// Message describes the conversion failure
	Message string
	// Cause is the underlying error, if any
	Cause error
}

// Error returns a human-readable error message.
func (e *ConversionError) Error() string {
	msg := "conversion error"
	if e.SourceVersion != "" && e.TargetVersion != "" {
		msg += fmt.Sprintf(" (%s -> %s)", e.SourceVersion, e.TargetVersion)
	}
	if e.Path != "" {
		msg += " at " + e.Path
	}
	if e.Message != "" {
		msg += ": " + e.Message
	}
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

// Unwrap returns the underlying cause for error chaining.
func (e *ConversionError) Unwrap() error {
	return e.Cause
}

// Is reports whether target matches this error type.
func (e *ConversionError) Is(target error) bool {
	return target == ErrConversion
}

// ConfigError represents an invalid configuration or input.
// This includes invalid options, missing required inputs, and conflicting settings.
type ConfigError struct {
	// Option is the name of the problematic configuration option
	Option string
	// Value is the invalid value that was provided (may be nil)
	Value any
	// Message describes the configuration error
	Message string
	// Cause is the underlying error, if any
	Cause error
}

// Error returns a human-readable error message.
func (e *ConfigError) Error() string {
	msg := "configuration error"
	if e.Option != "" {
		msg += " for " + e.Option
	}
	if e.Value != nil {
		msg += fmt.Sprintf(" (value: %v)", e.Value)
	}
	if e.Message != "" {
		msg += ": " + e.Message
	}
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

// Unwrap returns the underlying cause for error chaining.
func (e *ConfigError) Unwrap() error {
	return e.Cause
}

// Is reports whether target matches this error type.
func (e *ConfigError) Is(target error) bool {
	return target == ErrConfig
}
