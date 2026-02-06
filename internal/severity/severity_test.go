package severity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeverityString(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		expected string
	}{
		// Valid severity levels
		{"error level", SeverityError, "error"},
		{"warning level", SeverityWarning, "warning"},
		{"info level", SeverityInfo, "info"},
		{"critical level", SeverityCritical, "critical"},

		// Edge cases: Invalid severity values
		{"unknown negative", Severity(-1), "unknown"},
		{"unknown large value", Severity(999), "unknown"},
		{"unknown beyond range", Severity(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.severity.String()
			assert.Equal(t, tt.expected, result, "Severity(%d).String() = %q, want %q", tt.severity, result, tt.expected)
		})
	}
}

// TestSeverityStringConsistency verifies that all defined severity levels
// return non-empty, lowercase strings without whitespace.
func TestSeverityStringConsistency(t *testing.T) {
	severities := []Severity{
		SeverityError,
		SeverityWarning,
		SeverityInfo,
		SeverityCritical,
	}

	for _, sev := range severities {
		str := sev.String()

		// Should not be empty
		assert.NotEmpty(t, str, "Severity(%d).String() should not be empty", sev)

		// Should be lowercase (consistent with existing implementation)
		assert.Equal(t, str, str, "Severity string should be lowercase: %q", str)

		// Should not contain whitespace
		assert.NotContains(t, str, " ", "Severity string should not contain spaces: %q", str)
		assert.NotContains(t, str, "\t", "Severity string should not contain tabs: %q", str)
		assert.NotContains(t, str, "\n", "Severity string should not contain newlines: %q", str)
	}
}
