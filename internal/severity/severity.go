// Package severity provides severity level constants and utilities
// for issues reported by validator, converter, differ, and generator packages.
//
// All four severity levels are exported by each public package that uses them:
//   - SeverityInfo: Informational messages about choices made
//   - SeverityWarning: Lossy conversions, best-practice violations, or recommendations
//   - SeverityError: Spec violations that make documents invalid
//   - SeverityCritical: Features that cannot be processed (data loss)
//
// The severity levels are ordered from least to most severe:
// Info < Warning < Error < Critical
package severity

// Severity indicates the severity level of an issue during validation, conversion,
// diff analysis, or code generation.
type Severity int

const (
	// SeverityError indicates a spec violation that makes the document invalid.
	// Used primarily by the validator package for structural/semantic errors.
	SeverityError Severity = iota

	// SeverityWarning indicates lossy conversions, best-practice violations,
	// or recommendations that don't prevent processing but should be addressed.
	SeverityWarning

	// SeverityInfo indicates informational messages about processing choices.
	// These are non-actionable notices that may be useful for debugging.
	SeverityInfo

	// SeverityCritical indicates features that cannot be processed without data loss.
	// Used when conversion or generation must skip or alter functionality.
	SeverityCritical
)

// String returns the string representation of the severity level.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}
