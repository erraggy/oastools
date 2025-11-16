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
	Value interface{}
	// SpecRef is the URL to the relevant section of the OAS specification (optional, validation use)
	SpecRef string
	// Context provides additional information about the issue (optional, conversion use)
	Context string
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

	result := fmt.Sprintf("%s %s: %s", symbol, i.Path, i.Message)

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
