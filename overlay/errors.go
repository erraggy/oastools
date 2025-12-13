package overlay

import (
	"fmt"
)

// ValidationError represents an error in the overlay document structure.
type ValidationError struct {
	// Field is the name of the field with the error.
	Field string

	// Path is the location in the overlay document (e.g., "actions[0].target").
	Path string

	// Message describes the validation error.
	Message string
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("overlay: validation error at %s: %s", e.Path, e.Message)
	}
	if e.Field != "" {
		return fmt.Sprintf("overlay: validation error in field %q: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("overlay: validation error: %s", e.Message)
}

// ApplyError represents an error during overlay application.
type ApplyError struct {
	// ActionIndex is the zero-based index of the action that failed.
	ActionIndex int

	// Target is the JSONPath expression that was being evaluated.
	Target string

	// Cause is the underlying error.
	Cause error
}

// Error implements the error interface.
func (e *ApplyError) Error() string {
	return fmt.Sprintf("overlay: action[%d] target=%q: %v", e.ActionIndex, e.Target, e.Cause)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *ApplyError) Unwrap() error {
	return e.Cause
}

// ParseError represents an error during overlay document parsing.
type ParseError struct {
	// Path is the file path or source identifier.
	Path string

	// Cause is the underlying error.
	Cause error
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("overlay: failed to parse %s: %v", e.Path, e.Cause)
	}
	return fmt.Sprintf("overlay: failed to parse: %v", e.Cause)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *ParseError) Unwrap() error {
	return e.Cause
}
