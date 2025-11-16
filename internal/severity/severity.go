// Package severity provides severity level constants and utilities
// for issues reported by validator and converter packages.
package severity

// Severity indicates the severity level of an issue during validation or conversion.
type Severity int

const (
	// SeverityError indicates a spec violation that makes the document invalid (validation only)
	SeverityError Severity = iota
	// SeverityWarning indicates lossy conversions, best-practice violations, or recommendations
	SeverityWarning
	// SeverityInfo indicates informational messages about conversion choices
	SeverityInfo
	// SeverityCritical indicates features that cannot be converted (data loss)
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
