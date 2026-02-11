package differ

import (
	"testing"

	"github.com/erraggy/oastools/internal/testutil"
	"github.com/erraggy/oastools/parser"
)

// TestDiffSchemaNumericConstraintsUnified tests numeric constraint comparison
func TestDiffSchemaNumericConstraintsUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.Schema
		target        *parser.Schema
		mode          DiffMode
		expectedCount int
		checkPath     string
		checkSeverity Severity
	}{
		{
			name: "multipleOf changed",
			source: &parser.Schema{
				MultipleOf: testutil.Ptr(5.0),
			},
			target: &parser.Schema{
				MultipleOf: testutil.Ptr(10.0),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.multipleOf",
			checkSeverity: SeverityWarning,
		},
		{
			name: "maximum tightened (lowered) - error",
			source: &parser.Schema{
				Maximum: testutil.Ptr(100.0),
			},
			target: &parser.Schema{
				Maximum: testutil.Ptr(50.0),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.maximum",
			checkSeverity: SeverityError,
		},
		{
			name: "maximum relaxed (raised) - warning",
			source: &parser.Schema{
				Maximum: testutil.Ptr(50.0),
			},
			target: &parser.Schema{
				Maximum: testutil.Ptr(100.0),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.maximum",
			checkSeverity: SeverityWarning,
		},
		{
			name: "maximum added - error",
			source: &parser.Schema{
				Maximum: nil,
			},
			target: &parser.Schema{
				Maximum: testutil.Ptr(100.0),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.maximum",
			checkSeverity: SeverityError,
		},
		{
			name: "minimum tightened (raised) - error",
			source: &parser.Schema{
				Minimum: testutil.Ptr(10.0),
			},
			target: &parser.Schema{
				Minimum: testutil.Ptr(20.0),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.minimum",
			checkSeverity: SeverityError,
		},
		{
			name: "minimum relaxed (lowered) - warning",
			source: &parser.Schema{
				Minimum: testutil.Ptr(20.0),
			},
			target: &parser.Schema{
				Minimum: testutil.Ptr(10.0),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.minimum",
			checkSeverity: SeverityWarning,
		},
		{
			name: "minimum added - error",
			source: &parser.Schema{
				Minimum: nil,
			},
			target: &parser.Schema{
				Minimum: testutil.Ptr(10.0),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.minimum",
			checkSeverity: SeverityError,
		},
		{
			name: "no changes",
			source: &parser.Schema{
				Minimum: testutil.Ptr(10.0),
				Maximum: testutil.Ptr(100.0),
			},
			target: &parser.Schema{
				Minimum: testutil.Ptr(10.0),
				Maximum: testutil.Ptr(100.0),
			},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffSchemaNumericConstraintsUnified(tt.source, tt.target, "test", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s (severity: %v)", c.Path, c.Message, c.Severity)
				}
				return
			}

			if tt.checkPath != "" && tt.expectedCount > 0 {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkPath {
						found = true
						if tt.checkSeverity != 0 && c.Severity != tt.checkSeverity {
							t.Errorf("Expected severity %v, got %v for path %s", tt.checkSeverity, c.Severity, c.Path)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected change at path %s not found", tt.checkPath)
				}
			}
		})
	}
}

// TestDiffSchemaStringConstraintsUnified tests string constraint comparison
func TestDiffSchemaStringConstraintsUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.Schema
		target        *parser.Schema
		mode          DiffMode
		expectedCount int
		checkPath     string
		checkSeverity Severity
	}{
		{
			name: "maxLength tightened (lowered) - error",
			source: &parser.Schema{
				MaxLength: testutil.Ptr(100),
			},
			target: &parser.Schema{
				MaxLength: testutil.Ptr(50),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.maxLength",
			checkSeverity: SeverityError,
		},
		{
			name: "maxLength relaxed (raised) - warning",
			source: &parser.Schema{
				MaxLength: testutil.Ptr(50),
			},
			target: &parser.Schema{
				MaxLength: testutil.Ptr(100),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.maxLength",
			checkSeverity: SeverityWarning,
		},
		{
			name: "maxLength added - error",
			source: &parser.Schema{
				MaxLength: nil,
			},
			target: &parser.Schema{
				MaxLength: testutil.Ptr(100),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.maxLength",
			checkSeverity: SeverityError,
		},
		{
			name: "minLength tightened (raised) - error",
			source: &parser.Schema{
				MinLength: testutil.Ptr(5),
			},
			target: &parser.Schema{
				MinLength: testutil.Ptr(10),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.minLength",
			checkSeverity: SeverityError,
		},
		{
			name: "minLength relaxed (lowered) - warning",
			source: &parser.Schema{
				MinLength: testutil.Ptr(10),
			},
			target: &parser.Schema{
				MinLength: testutil.Ptr(5),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.minLength",
			checkSeverity: SeverityWarning,
		},
		{
			name: "minLength added - error",
			source: &parser.Schema{
				MinLength: nil,
			},
			target: &parser.Schema{
				MinLength: testutil.Ptr(5),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.minLength",
			checkSeverity: SeverityError,
		},
		{
			name: "pattern added - error",
			source: &parser.Schema{
				Pattern: "",
			},
			target: &parser.Schema{
				Pattern: "^[a-z]+$",
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.pattern",
			checkSeverity: SeverityError,
		},
		{
			name: "pattern changed - warning",
			source: &parser.Schema{
				Pattern: "^[a-z]+$",
			},
			target: &parser.Schema{
				Pattern: "^[a-zA-Z]+$",
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.pattern",
			checkSeverity: SeverityWarning,
		},
		{
			name: "pattern removed",
			source: &parser.Schema{
				Pattern: "^[a-z]+$",
			},
			target: &parser.Schema{
				Pattern: "",
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.pattern",
		},
		{
			name: "no changes",
			source: &parser.Schema{
				MinLength: testutil.Ptr(5),
				MaxLength: testutil.Ptr(100),
				Pattern:   "^[a-z]+$",
			},
			target: &parser.Schema{
				MinLength: testutil.Ptr(5),
				MaxLength: testutil.Ptr(100),
				Pattern:   "^[a-z]+$",
			},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffSchemaStringConstraintsUnified(tt.source, tt.target, "test", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s (severity: %v)", c.Path, c.Message, c.Severity)
				}
				return
			}

			if tt.checkPath != "" && tt.expectedCount > 0 {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkPath {
						found = true
						if tt.checkSeverity != 0 && c.Severity != tt.checkSeverity {
							t.Errorf("Expected severity %v, got %v for path %s", tt.checkSeverity, c.Severity, c.Path)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected change at path %s not found", tt.checkPath)
				}
			}
		})
	}
}

// TestDiffSchemaArrayConstraintsUnified tests array constraint comparison
func TestDiffSchemaArrayConstraintsUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.Schema
		target        *parser.Schema
		mode          DiffMode
		expectedCount int
		checkPath     string
		checkSeverity Severity
	}{
		{
			name: "maxItems tightened (lowered) - error",
			source: &parser.Schema{
				MaxItems: testutil.Ptr(100),
			},
			target: &parser.Schema{
				MaxItems: testutil.Ptr(50),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.maxItems",
			checkSeverity: SeverityError,
		},
		{
			name: "maxItems relaxed (raised) - warning",
			source: &parser.Schema{
				MaxItems: testutil.Ptr(50),
			},
			target: &parser.Schema{
				MaxItems: testutil.Ptr(100),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.maxItems",
			checkSeverity: SeverityWarning,
		},
		{
			name: "maxItems added - error",
			source: &parser.Schema{
				MaxItems: nil,
			},
			target: &parser.Schema{
				MaxItems: testutil.Ptr(100),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.maxItems",
			checkSeverity: SeverityError,
		},
		{
			name: "minItems tightened (raised) - error",
			source: &parser.Schema{
				MinItems: testutil.Ptr(5),
			},
			target: &parser.Schema{
				MinItems: testutil.Ptr(10),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.minItems",
			checkSeverity: SeverityError,
		},
		{
			name: "minItems relaxed (lowered) - warning",
			source: &parser.Schema{
				MinItems: testutil.Ptr(10),
			},
			target: &parser.Schema{
				MinItems: testutil.Ptr(5),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.minItems",
			checkSeverity: SeverityWarning,
		},
		{
			name: "minItems added - error",
			source: &parser.Schema{
				MinItems: nil,
			},
			target: &parser.Schema{
				MinItems: testutil.Ptr(5),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.minItems",
			checkSeverity: SeverityError,
		},
		{
			name: "uniqueItems enabled - error",
			source: &parser.Schema{
				UniqueItems: false,
			},
			target: &parser.Schema{
				UniqueItems: true,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.uniqueItems",
			checkSeverity: SeverityError,
		},
		{
			name: "uniqueItems disabled - warning",
			source: &parser.Schema{
				UniqueItems: true,
			},
			target: &parser.Schema{
				UniqueItems: false,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.uniqueItems",
			checkSeverity: SeverityWarning,
		},
		{
			name: "no changes",
			source: &parser.Schema{
				MinItems:    testutil.Ptr(5),
				MaxItems:    testutil.Ptr(100),
				UniqueItems: true,
			},
			target: &parser.Schema{
				MinItems:    testutil.Ptr(5),
				MaxItems:    testutil.Ptr(100),
				UniqueItems: true,
			},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffSchemaArrayConstraintsUnified(tt.source, tt.target, "test", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s (severity: %v)", c.Path, c.Message, c.Severity)
				}
				return
			}

			if tt.checkPath != "" && tt.expectedCount > 0 {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkPath {
						found = true
						if tt.checkSeverity != 0 && c.Severity != tt.checkSeverity {
							t.Errorf("Expected severity %v, got %v for path %s", tt.checkSeverity, c.Severity, c.Path)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected change at path %s not found", tt.checkPath)
				}
			}
		})
	}
}

// TestDiffSchemaObjectConstraintsUnified tests object constraint comparison
func TestDiffSchemaObjectConstraintsUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.Schema
		target        *parser.Schema
		mode          DiffMode
		expectedCount int
		checkPath     string
		checkSeverity Severity
	}{
		{
			name: "maxProperties tightened (lowered) - error",
			source: &parser.Schema{
				MaxProperties: testutil.Ptr(100),
			},
			target: &parser.Schema{
				MaxProperties: testutil.Ptr(50),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.maxProperties",
			checkSeverity: SeverityError,
		},
		{
			name: "maxProperties relaxed (raised) - warning",
			source: &parser.Schema{
				MaxProperties: testutil.Ptr(50),
			},
			target: &parser.Schema{
				MaxProperties: testutil.Ptr(100),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.maxProperties",
			checkSeverity: SeverityWarning,
		},
		{
			name: "maxProperties added - error",
			source: &parser.Schema{
				MaxProperties: nil,
			},
			target: &parser.Schema{
				MaxProperties: testutil.Ptr(100),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.maxProperties",
			checkSeverity: SeverityError,
		},
		{
			name: "minProperties tightened (raised) - error",
			source: &parser.Schema{
				MinProperties: testutil.Ptr(5),
			},
			target: &parser.Schema{
				MinProperties: testutil.Ptr(10),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.minProperties",
			checkSeverity: SeverityError,
		},
		{
			name: "minProperties relaxed (lowered) - warning",
			source: &parser.Schema{
				MinProperties: testutil.Ptr(10),
			},
			target: &parser.Schema{
				MinProperties: testutil.Ptr(5),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.minProperties",
			checkSeverity: SeverityWarning,
		},
		{
			name: "minProperties added - error",
			source: &parser.Schema{
				MinProperties: nil,
			},
			target: &parser.Schema{
				MinProperties: testutil.Ptr(5),
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.minProperties",
			checkSeverity: SeverityError,
		},
		{
			name: "no changes",
			source: &parser.Schema{
				MinProperties: testutil.Ptr(5),
				MaxProperties: testutil.Ptr(100),
			},
			target: &parser.Schema{
				MinProperties: testutil.Ptr(5),
				MaxProperties: testutil.Ptr(100),
			},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffSchemaObjectConstraintsUnified(tt.source, tt.target, "test", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s (severity: %v)", c.Path, c.Message, c.Severity)
				}
				return
			}

			if tt.checkPath != "" && tt.expectedCount > 0 {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkPath {
						found = true
						if tt.checkSeverity != 0 && c.Severity != tt.checkSeverity {
							t.Errorf("Expected severity %v, got %v for path %s", tt.checkSeverity, c.Severity, c.Path)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected change at path %s not found", tt.checkPath)
				}
			}
		})
	}
}

// TestDiffSchemaRequiredFieldsUnified tests required fields comparison
func TestDiffSchemaRequiredFieldsUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.Schema
		target        *parser.Schema
		mode          DiffMode
		expectedCount int
	}{
		{
			name: "required field added - error",
			source: &parser.Schema{
				Required: []string{},
			},
			target: &parser.Schema{
				Required: []string{"name"},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "required field removed - info",
			source: &parser.Schema{
				Required: []string{"name"},
			},
			target: &parser.Schema{
				Required: []string{},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "multiple required changes",
			source: &parser.Schema{
				Required: []string{"name", "age"},
			},
			target: &parser.Schema{
				Required: []string{"name", "email"},
			},
			mode:          ModeBreaking,
			expectedCount: 2, // age removed, email added
		},
		{
			name: "no changes",
			source: &parser.Schema{
				Required: []string{"name", "email"},
			},
			target: &parser.Schema{
				Required: []string{"name", "email"},
			},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffSchemaRequiredFieldsUnified(tt.source, tt.target, "test", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s (severity: %v)", c.Path, c.Message, c.Severity)
				}
			}
		})
	}
}

// TestDiffSchemaOASFieldsUnified tests OAS-specific field comparison
func TestDiffSchemaOASFieldsUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.Schema
		target        *parser.Schema
		mode          DiffMode
		expectedCount int
		checkPath     string
		checkSeverity Severity
	}{
		{
			name: "nullable removed - error",
			source: &parser.Schema{
				Nullable: true,
			},
			target: &parser.Schema{
				Nullable: false,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.nullable",
			checkSeverity: SeverityError,
		},
		{
			name: "nullable added - warning",
			source: &parser.Schema{
				Nullable: false,
			},
			target: &parser.Schema{
				Nullable: true,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.nullable",
			checkSeverity: SeverityWarning,
		},
		{
			name: "readOnly changed",
			source: &parser.Schema{
				ReadOnly: false,
			},
			target: &parser.Schema{
				ReadOnly: true,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.readOnly",
			checkSeverity: SeverityWarning,
		},
		{
			name: "writeOnly changed",
			source: &parser.Schema{
				WriteOnly: false,
			},
			target: &parser.Schema{
				WriteOnly: true,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.writeOnly",
			checkSeverity: SeverityWarning,
		},
		{
			name: "deprecated enabled - warning",
			source: &parser.Schema{
				Deprecated: false,
			},
			target: &parser.Schema{
				Deprecated: true,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.deprecated",
			checkSeverity: SeverityWarning,
		},
		{
			name: "deprecated disabled - info",
			source: &parser.Schema{
				Deprecated: true,
			},
			target: &parser.Schema{
				Deprecated: false,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.deprecated",
			checkSeverity: SeverityInfo,
		},
		{
			name: "no changes",
			source: &parser.Schema{
				Nullable:   true,
				ReadOnly:   false,
				WriteOnly:  false,
				Deprecated: false,
			},
			target: &parser.Schema{
				Nullable:   true,
				ReadOnly:   false,
				WriteOnly:  false,
				Deprecated: false,
			},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffSchemaOASFieldsUnified(tt.source, tt.target, "test", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s (severity: %v)", c.Path, c.Message, c.Severity)
				}
				return
			}

			if tt.checkPath != "" && tt.expectedCount > 0 {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkPath {
						found = true
						if tt.checkSeverity != 0 && c.Severity != tt.checkSeverity {
							t.Errorf("Expected severity %v, got %v for path %s", tt.checkSeverity, c.Severity, c.Path)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected change at path %s not found", tt.checkPath)
				}
			}
		})
	}
}
