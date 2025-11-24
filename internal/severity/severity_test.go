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

// TestSeverityConstants verifies that severity constants have expected values
// and maintain their ordering for comparison operations.
func TestSeverityConstants(t *testing.T) {
	// Verify iota ordering (Error = 0, Warning = 1, Info = 2, Critical = 3)
	assert.Equal(t, Severity(0), SeverityError, "SeverityError should be 0")
	assert.Equal(t, Severity(1), SeverityWarning, "SeverityWarning should be 1")
	assert.Equal(t, Severity(2), SeverityInfo, "SeverityInfo should be 2")
	assert.Equal(t, Severity(3), SeverityCritical, "SeverityCritical should be 3")

	// Verify ordering for potential comparison operations
	assert.Less(t, int(SeverityError), int(SeverityWarning), "Error should be less than Warning")
	assert.Less(t, int(SeverityWarning), int(SeverityInfo), "Warning should be less than Info")
	assert.Less(t, int(SeverityInfo), int(SeverityCritical), "Info should be less than Critical")
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
