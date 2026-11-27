//go:build integration

package harness

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

// AssertOASVersion asserts the OAS version enum of a parsed document.
func AssertOASVersion(t *testing.T, result *parser.ParseResult, expected parser.OASVersion) {
	t.Helper()
	if result.OASVersion != expected {
		t.Errorf("expected OAS version %v, got %v", expected, result.OASVersion)
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
