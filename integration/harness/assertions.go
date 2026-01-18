//go:build integration

package harness

import (
	"fmt"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
)

// AssertValid asserts that a validation result indicates a valid document.
func AssertValid(t *testing.T, result *validator.ValidationResult) {
	t.Helper()
	if !result.Valid {
		t.Errorf("expected valid document, got %d errors:", result.ErrorCount)
		for _, e := range result.Errors {
			t.Errorf("  - %s", e.String())
		}
	}
}

// AssertInvalid asserts that a validation result indicates an invalid document.
func AssertInvalid(t *testing.T, result *validator.ValidationResult) {
	t.Helper()
	if result.Valid {
		t.Error("expected invalid document, but validation passed")
	}
}

// AssertErrorCount asserts the exact number of validation errors.
func AssertErrorCount(t *testing.T, result *validator.ValidationResult, expected int) {
	t.Helper()
	if result.ErrorCount != expected {
		t.Errorf("expected %d errors, got %d", expected, result.ErrorCount)
		for _, e := range result.Errors {
			t.Logf("  - %s", e.String())
		}
	}
}

// AssertWarningCount asserts the exact number of validation warnings.
func AssertWarningCount(t *testing.T, result *validator.ValidationResult, expected int) {
	t.Helper()
	if result.WarningCount != expected {
		t.Errorf("expected %d warnings, got %d", expected, result.WarningCount)
		for _, w := range result.Warnings {
			t.Logf("  - %s", w.String())
		}
	}
}

// AssertSchemaCount asserts the number of schemas in a parsed document.
func AssertSchemaCount(t *testing.T, result *parser.ParseResult, expected int) {
	t.Helper()
	actual := result.Stats.SchemaCount
	if actual != expected {
		t.Errorf("expected %d schemas, got %d", expected, actual)
	}
}

// AssertPathCount asserts the number of paths in a parsed document.
func AssertPathCount(t *testing.T, result *parser.ParseResult, expected int) {
	t.Helper()
	actual := result.Stats.PathCount
	if actual != expected {
		t.Errorf("expected %d paths, got %d", expected, actual)
	}
}

// AssertOperationCount asserts the number of operations in a parsed document.
func AssertOperationCount(t *testing.T, result *parser.ParseResult, expected int) {
	t.Helper()
	actual := result.Stats.OperationCount
	if actual != expected {
		t.Errorf("expected %d operations, got %d", expected, actual)
	}
}

// AssertVersion asserts the OAS version of a parsed document.
func AssertVersion(t *testing.T, result *parser.ParseResult, expected string) {
	t.Helper()
	if result.Version != expected {
		t.Errorf("expected version %q, got %q", expected, result.Version)
	}
}

// AssertOASVersion asserts the OAS version enum of a parsed document.
func AssertOASVersion(t *testing.T, result *parser.ParseResult, expected parser.OASVersion) {
	t.Helper()
	if result.OASVersion != expected {
		t.Errorf("expected OAS version %v, got %v", expected, result.OASVersion)
	}
}

// AssertSchemaExists asserts that a schema with the given name exists.
func AssertSchemaExists(t *testing.T, result *parser.ParseResult, schemaName string) {
	t.Helper()
	names := getSchemaNames(result)
	if !containsString(names, schemaName) {
		t.Errorf("expected schema %q to exist, but it was not found", schemaName)
		t.Logf("  available schemas: %v", names)
	}
}

// AssertSchemaNotExists asserts that a schema with the given name does not exist.
func AssertSchemaNotExists(t *testing.T, result *parser.ParseResult, schemaName string) {
	t.Helper()
	names := getSchemaNames(result)
	if containsString(names, schemaName) {
		t.Errorf("expected schema %q to not exist, but it was found", schemaName)
	}
}

// AssertNoParseErrors asserts that parsing produced no errors.
func AssertNoParseErrors(t *testing.T, result *parser.ParseResult) {
	t.Helper()
	if len(result.Errors) > 0 {
		t.Errorf("expected no parse errors, got %d:", len(result.Errors))
		for _, e := range result.Errors {
			t.Errorf("  - %v", e)
		}
	}
}

// AssertParseErrorContains asserts that at least one parse error contains the given substring.
func AssertParseErrorContains(t *testing.T, result *parser.ParseResult, substr string) {
	t.Helper()
	for _, e := range result.Errors {
		if containsSubstring(e.Error(), substr) {
			return
		}
	}
	t.Errorf("expected a parse error containing %q, but none found", substr)
	for _, e := range result.Errors {
		t.Logf("  - %v", e)
	}
}

// AssertValidationErrorContains asserts that at least one validation error contains the given substring.
func AssertValidationErrorContains(t *testing.T, result *validator.ValidationResult, substr string) {
	t.Helper()
	for _, e := range result.Errors {
		if containsSubstring(e.String(), substr) {
			return
		}
	}
	t.Errorf("expected a validation error containing %q, but none found", substr)
	for _, e := range result.Errors {
		t.Logf("  - %s", e.String())
	}
}

// ExpectResult encapsulates common assertions based on the expect field.
func ExpectResult(t *testing.T, expect string, validationResult *validator.ValidationResult, stepErr error) error {
	t.Helper()

	switch expect {
	case "valid":
		if stepErr != nil {
			return fmt.Errorf("expected valid but step failed: %w", stepErr)
		}
		if validationResult != nil && !validationResult.Valid {
			return fmt.Errorf("expected valid but got %d errors", validationResult.ErrorCount)
		}
	case "invalid":
		if stepErr != nil {
			return fmt.Errorf("expected invalid but step failed: %w", stepErr)
		}
		if validationResult != nil && validationResult.Valid {
			return fmt.Errorf("expected invalid but document is valid")
		}
	case "error":
		if stepErr == nil {
			return fmt.Errorf("expected error but step succeeded")
		}
		// Error was expected, this is success
		return nil
	case "success", "":
		if stepErr != nil {
			return fmt.Errorf("expected success but step failed: %w", stepErr)
		}
	default:
		return fmt.Errorf("unknown expect value: %s", expect)
	}

	return nil
}
