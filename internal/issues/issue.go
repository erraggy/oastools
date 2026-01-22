// Package issues provides a unified issue type for validation and conversion problems.
package issues

import (
	"fmt"

	"github.com/erraggy/oastools/internal/severity"
)

// Issue represents a single problem found during validation or conversion.
type Issue struct {
	// Path is the JSON path to the problematic field (e.g., "paths./pets.get.responses")
	Path string
	// Message is a human-readable description of the issue
	Message string
	// Severity indicates the severity level of the issue
	Severity severity.Severity
	// Field is the specific field name that has the issue
	Field string
	// Value is the problematic value (optional)
	Value any
	// SpecRef is the URL to the relevant section of the OAS specification (optional, validation use)
	SpecRef string
	// Context provides additional information about the issue (optional, conversion use)
	Context string
	// Line is the 1-based line number in the source file (0 if unknown)
	Line int
	// Column is the 1-based column number in the source file (0 if unknown)
	Column int
	// File is the source file path (empty for main document)
	File string
	// OperationContext provides API operation context when the issue relates to
	// an operation or a component referenced by operations. Nil when not applicable.
	OperationContext *OperationContext
}

// String returns a formatted string representation of the issue.
// Uses different symbols based on severity level:
// - "✗" for Error or Critical severity
// - "⚠" for Warning severity
// - "ℹ" for Info severity
func (i Issue) String() string {
	var symbol string
	switch i.Severity {
	case severity.SeverityError, severity.SeverityCritical:
		symbol = "✗"
	case severity.SeverityWarning:
		symbol = "⚠"
	case severity.SeverityInfo:
		symbol = "ℹ"
	default:
		symbol = "?"
	}

	var result string
	pathWithContext := i.Path
	if i.OperationContext != nil && !i.OperationContext.IsEmpty() {
		pathWithContext = fmt.Sprintf("%s %s", i.Path, i.OperationContext.String())
	}

	if i.Line > 0 {
		result = fmt.Sprintf("%s %s (line %d, col %d): %s", symbol, pathWithContext, i.Line, i.Column, i.Message)
	} else {
		result = fmt.Sprintf("%s %s: %s", symbol, pathWithContext, i.Message)
	}

	// Add SpecRef if present (validation use case)
	if i.SpecRef != "" {
		result += fmt.Sprintf("\n    Spec: %s", i.SpecRef)
	}

	// Add Context if present (conversion use case)
	if i.Context != "" {
		result += fmt.Sprintf("\n    Context: %s", i.Context)
	}

	return result
}

// Location returns the source location in IDE-friendly format.
// Returns "file:line:column" if file is set, "line:column" if only line is set,
// or the JSON path if location is unknown.
func (i Issue) Location() string {
	if i.Line == 0 {
		return i.Path
	}
	if i.File != "" {
		return fmt.Sprintf("%s:%d:%d", i.File, i.Line, i.Column)
	}
	return fmt.Sprintf("%d:%d", i.Line, i.Column)
}

// HasLocation returns true if this issue has source location information.
func (i Issue) HasLocation() bool {
	return i.Line > 0
}
