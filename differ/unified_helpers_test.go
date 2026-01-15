package differ

import (
	"fmt"
	"testing"
)

// TestIsErrorCode_Extended tests additional error code detection scenarios
func TestIsErrorCode_Extended(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		// 4xx codes
		{"400", true},
		{"401", true},
		{"403", true},
		{"404", true},
		{"405", true},
		{"429", true},
		{"499", true},

		// 5xx codes
		{"500", true},
		{"501", true},
		{"502", true},
		{"503", true},
		{"504", true},
		{"599", true},

		// 2xx codes are not error codes
		{"200", false},
		{"201", false},
		{"204", false},
		{"299", false},

		// 3xx codes are not error codes
		{"300", false},
		{"301", false},
		{"302", false},
		{"304", false},
		{"399", false},

		// 1xx codes are not error codes
		{"100", false},
		{"101", false},
		{"199", false},

		// String-based prefix matching (e.g., "4xx", "5XX")
		{"4xx", true},
		{"4XX", true},
		{"5xx", true},
		{"5XX", true},

		// Non-numeric codes that don't start with 4 or 5
		{"default", false},
		{"xxx", false},
		{"2XX", false},
		{"3xx", false},
		{"1xx", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := isErrorCode(tt.code)
			if got != tt.expected {
				t.Errorf("isErrorCode(%q) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}

// TestIsSuccessCode_Extended tests additional success code detection scenarios
func TestIsSuccessCode_Extended(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		// 2xx codes are success codes
		{"200", true},
		{"201", true},
		{"202", true},
		{"204", true},
		{"206", true},
		{"299", true},

		// String-based prefix matching
		{"2xx", true},
		{"2XX", true},

		// Non-2xx codes are not success codes
		{"100", false},
		{"199", false},
		{"300", false},
		{"301", false},
		{"400", false},
		{"404", false},
		{"500", false},
		{"default", false},
		{"xxx", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := isSuccessCode(tt.code)
			if got != tt.expected {
				t.Errorf("isSuccessCode(%q) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}

// TestAnyToString_Extended tests additional type conversion scenarios
func TestAnyToString_Extended(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		// Int64 type (not covered in breaking_test.go)
		{"int64 positive", int64(9999999999), "9999999999"},
		{"int64 negative", int64(-9999999999), "-9999999999"},
		{"int64 zero", int64(0), "0"},

		// Float64 type (not covered in breaking_test.go)
		{"float64 positive", 3.14, "3.14"},
		{"float64 negative", -2.5, "-2.5"},
		{"float64 integer-like", 10.0, "10"},
		{"float64 zero", 0.0, "0"},

		// Stringer interface (not covered in breaking_test.go)
		{"stringer", stringerTypeForHelper("custom"), "custom"},

		// Other types (uses fmt.Sprint)
		{"map", map[string]int{"a": 1}, "map[a:1]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := anyToString(tt.input)
			if got != tt.expected {
				t.Errorf("anyToString(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// stringerTypeForHelper implements fmt.Stringer for testing
type stringerTypeForHelper string

func (s stringerTypeForHelper) String() string {
	return string(s)
}

// TestSeverityWithRule_SimpleMode tests that simple mode returns zero severity
func TestSeverityWithRule_SimpleMode(t *testing.T) {
	d := New()
	d.Mode = ModeSimple

	sev, ignore := d.severityWithRule(SeverityError, RuleKey{Category: CategorySchema, ChangeType: ChangeTypeModified})

	if sev != 0 {
		t.Errorf("Expected severity 0 in simple mode, got %v", sev)
	}
	if ignore {
		t.Error("Expected ignore=false in simple mode")
	}
}

// TestSeverityWithRule_BreakingMode tests that breaking mode returns appropriate severity
func TestSeverityWithRule_BreakingMode(t *testing.T) {
	d := New()
	d.Mode = ModeBreaking

	sev, ignore := d.severityWithRule(SeverityError, RuleKey{Category: CategorySchema, ChangeType: ChangeTypeModified})

	if sev != SeverityError {
		t.Errorf("Expected severity Error in breaking mode, got %v", sev)
	}
	if ignore {
		t.Error("Expected ignore=false without ignore rule")
	}
}

// TestSeverityConditionalWithRule_SimpleMode tests conditional severity in simple mode
func TestSeverityConditionalWithRule_SimpleMode(t *testing.T) {
	d := New()
	d.Mode = ModeSimple

	sev, ignore := d.severityConditionalWithRule(true, SeverityError, SeverityWarning, RuleKey{})

	if sev != 0 {
		t.Errorf("Expected severity 0 in simple mode, got %v", sev)
	}
	if ignore {
		t.Error("Expected ignore=false in simple mode")
	}
}

// TestSeverityConditionalWithRule_BreakingMode tests conditional severity in breaking mode
func TestSeverityConditionalWithRule_BreakingMode(t *testing.T) {
	d := New()
	d.Mode = ModeBreaking

	// Test condition=true
	sev, ignore := d.severityConditionalWithRule(true, SeverityError, SeverityWarning, RuleKey{})
	if sev != SeverityError {
		t.Errorf("Expected severity Error when condition=true, got %v", sev)
	}
	if ignore {
		t.Error("Expected ignore=false")
	}

	// Test condition=false
	sev, ignore = d.severityConditionalWithRule(false, SeverityError, SeverityWarning, RuleKey{})
	if sev != SeverityWarning {
		t.Errorf("Expected severity Warning when condition=false, got %v", sev)
	}
	if ignore {
		t.Error("Expected ignore=false")
	}
}

// TestAddChange tests the addChange helper method
func TestAddChange(t *testing.T) {
	t.Run("simple mode does not add severity", func(t *testing.T) {
		d := New()
		d.Mode = ModeSimple
		result := &DiffResult{}

		d.addChange(result, "test.path", ChangeTypeModified, CategorySchema, SeverityError, "old", "new", "test message")

		if len(result.Changes) != 1 {
			t.Fatalf("Expected 1 change, got %d", len(result.Changes))
		}
		if result.Changes[0].Severity != 0 {
			t.Errorf("Expected severity 0 in simple mode, got %v", result.Changes[0].Severity)
		}
	})

	t.Run("breaking mode adds severity", func(t *testing.T) {
		d := New()
		d.Mode = ModeBreaking
		result := &DiffResult{}

		d.addChange(result, "test.path", ChangeTypeModified, CategorySchema, SeverityError, "old", "new", "test message")

		if len(result.Changes) != 1 {
			t.Fatalf("Expected 1 change, got %d", len(result.Changes))
		}
		if result.Changes[0].Severity != SeverityError {
			t.Errorf("Expected severity Error, got %v", result.Changes[0].Severity)
		}
	})
}

// TestAddChangeConditional tests the addChangeConditional helper method
func TestAddChangeConditional(t *testing.T) {
	tests := []struct {
		name             string
		mode             DiffMode
		condition        bool
		severityIfTrue   Severity
		severityIfFalse  Severity
		expectedSeverity Severity
	}{
		{
			name:             "breaking mode condition true",
			mode:             ModeBreaking,
			condition:        true,
			severityIfTrue:   SeverityError,
			severityIfFalse:  SeverityWarning,
			expectedSeverity: SeverityError,
		},
		{
			name:             "breaking mode condition false",
			mode:             ModeBreaking,
			condition:        false,
			severityIfTrue:   SeverityError,
			severityIfFalse:  SeverityWarning,
			expectedSeverity: SeverityWarning,
		},
		{
			name:             "simple mode",
			mode:             ModeSimple,
			condition:        true,
			severityIfTrue:   SeverityError,
			severityIfFalse:  SeverityWarning,
			expectedSeverity: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.addChangeConditional(result, "test.path", ChangeTypeModified, CategorySchema,
				tt.condition, tt.severityIfTrue, tt.severityIfFalse, "old", "new", "test message")

			if len(result.Changes) != 1 {
				t.Fatalf("Expected 1 change, got %d", len(result.Changes))
			}
			if result.Changes[0].Severity != tt.expectedSeverity {
				t.Errorf("Expected severity %v, got %v", tt.expectedSeverity, result.Changes[0].Severity)
			}
		})
	}
}

// TestIsCompatibleTypeChange_Extended tests additional type compatibility cases
func TestIsCompatibleTypeChange_Extended(t *testing.T) {
	tests := []struct {
		oldType  string
		newType  string
		expected bool
	}{
		// integer to number is compatible (widening)
		{"integer", "number", true},

		// All other combinations are not compatible
		{"number", "integer", false},
		{"string", "integer", false},
		{"string", "number", false},
		{"boolean", "string", false},
		{"array", "object", false},
		{"integer", "string", false},
		{"number", "string", false},
		{"", "string", false},
		{"string", "", false},
		{"object", "array", false},
		{"null", "string", false},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s_to_%s", tt.oldType, tt.newType)
		if tt.oldType == "" {
			name = fmt.Sprintf("empty_to_%s", tt.newType)
		}
		if tt.newType == "" {
			name = fmt.Sprintf("%s_to_empty", tt.oldType)
		}
		t.Run(name, func(t *testing.T) {
			got := isCompatibleTypeChange(tt.oldType, tt.newType)
			if got != tt.expected {
				t.Errorf("isCompatibleTypeChange(%q, %q) = %v, want %v", tt.oldType, tt.newType, got, tt.expected)
			}
		})
	}
}
