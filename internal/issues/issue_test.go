package issues

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/internal/severity"
	"github.com/stretchr/testify/assert"
)

func TestIssueString(t *testing.T) {
	tests := []struct {
		name        string
		issue       Issue
		contains    []string // Strings that must be present in output
		notContains []string // Strings that must NOT be present in output
	}{
		{
			name: "error severity with basic fields",
			issue: Issue{
				Path:     "paths./pets.get",
				Message:  "Missing required field",
				Severity: severity.SeverityError,
			},
			contains: []string{
				"✗",
				"paths./pets.get",
				"Missing required field",
			},
			notContains: []string{"Spec:", "Context:"},
		},
		{
			name: "critical severity with basic fields",
			issue: Issue{
				Path:     "components.schemas.Pet",
				Message:  "Cannot convert type",
				Severity: severity.SeverityCritical,
			},
			contains: []string{
				"✗",
				"components.schemas.Pet",
				"Cannot convert type",
			},
			notContains: []string{"Spec:", "Context:"},
		},
		{
			name: "warning severity with basic fields",
			issue: Issue{
				Path:     "info.version",
				Message:  "Version should follow semver",
				Severity: severity.SeverityWarning,
			},
			contains: []string{
				"⚠",
				"info.version",
				"Version should follow semver",
			},
			notContains: []string{"Spec:", "Context:"},
		},
		{
			name: "info severity with basic fields",
			issue: Issue{
				Path:     "servers[0]",
				Message:  "Using first server for conversion",
				Severity: severity.SeverityInfo,
			},
			contains: []string{
				"ℹ",
				"servers[0]",
				"Using first server for conversion",
			},
			notContains: []string{"Spec:", "Context:"},
		},
		{
			name: "error with SpecRef (validation use case)",
			issue: Issue{
				Path:     "paths./users.post.requestBody",
				Message:  "RequestBody is required",
				Severity: severity.SeverityError,
				SpecRef:  "https://spec.openapis.org/oas/v3.1.0#request-body-object",
			},
			contains: []string{
				"✗",
				"paths./users.post.requestBody",
				"RequestBody is required",
				"Spec: https://spec.openapis.org/oas/v3.1.0#request-body-object",
			},
			notContains: []string{"Context:"},
		},
		{
			name: "warning with Context (conversion use case)",
			issue: Issue{
				Path:     "components.securitySchemes.oauth2",
				Message:  "OAuth2 flows restructured",
				Severity: severity.SeverityWarning,
				Context:  "OAS 2.0 uses flow field; OAS 3.x uses flows object",
			},
			contains: []string{
				"⚠",
				"components.securitySchemes.oauth2",
				"OAuth2 flows restructured",
				"Context: OAS 2.0 uses flow field; OAS 3.x uses flows object",
			},
			notContains: []string{"Spec:"},
		},
		{
			name: "critical with both SpecRef and Context",
			issue: Issue{
				Path:     "paths./api.get.parameters[0]",
				Message:  "Unsupported parameter type",
				Severity: severity.SeverityCritical,
				SpecRef:  "https://spec.openapis.org/oas/v3.0.0#parameter-object",
				Context:  "Parameter type 'file' is not supported in OAS 3.0",
			},
			contains: []string{
				"✗",
				"paths./api.get.parameters[0]",
				"Unsupported parameter type",
				"Spec: https://spec.openapis.org/oas/v3.0.0#parameter-object",
				"Context: Parameter type 'file' is not supported in OAS 3.0",
			},
		},
		{
			name: "unknown severity (edge case)",
			issue: Issue{
				Path:     "test.path",
				Message:  "Test message",
				Severity: severity.Severity(999), // Invalid severity
			},
			contains: []string{
				"?",
				"test.path",
				"Test message",
			},
			notContains: []string{"Spec:", "Context:"},
		},
		{
			name: "empty path",
			issue: Issue{
				Path:     "",
				Message:  "Root level issue",
				Severity: severity.SeverityError,
			},
			contains: []string{
				"✗",
				"Root level issue",
			},
		},
		{
			name: "complex path with arrays and nested fields",
			issue: Issue{
				Path:     "paths./users/{id}.get.responses.200.content.application/json.schema.properties.address.properties.zip",
				Message:  "Invalid zip code format",
				Severity: severity.SeverityError,
			},
			contains: []string{
				"✗",
				"paths./users/{id}.get.responses.200.content.application/json.schema.properties.address.properties.zip",
				"Invalid zip code format",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.issue.String()

			// Check that all required strings are present
			for _, substr := range tt.contains {
				assert.Contains(t, result, substr, "String() output should contain %q", substr)
			}

			// Check that forbidden strings are absent
			for _, substr := range tt.notContains {
				assert.NotContains(t, result, substr, "String() output should not contain %q", substr)
			}

			// Verify format structure: symbol, path, message on first line
			lines := strings.Split(result, "\n")
			assert.NotEmpty(t, lines[0], "First line should not be empty")
		})
	}
}

// TestIssueSeveritySymbols verifies that each severity level produces the correct symbol.
func TestIssueSeveritySymbols(t *testing.T) {
	tests := []struct {
		severity       severity.Severity
		expectedSymbol string
	}{
		{severity.SeverityError, "✗"},
		{severity.SeverityCritical, "✗"},
		{severity.SeverityWarning, "⚠"},
		{severity.SeverityInfo, "ℹ"},
		{severity.Severity(-1), "?"},  // Unknown severity
		{severity.Severity(999), "?"}, // Unknown severity
	}

	for _, tt := range tests {
		t.Run(tt.severity.String(), func(t *testing.T) {
			issue := Issue{
				Path:     "test.path",
				Message:  "Test message",
				Severity: tt.severity,
			}
			result := issue.String()
			assert.True(t, strings.HasPrefix(result, tt.expectedSymbol),
				"Issue with severity %s should start with symbol %q, got: %s",
				tt.severity.String(), tt.expectedSymbol, result)
		})
	}
}

// TestIssueStructFields verifies that all Issue struct fields can be set and used.
func TestIssueStructFields(t *testing.T) {
	issue := Issue{
		Path:     "paths./test",
		Message:  "Test message",
		Severity: severity.SeverityWarning,
		Field:    "operationId",
		Value:    "duplicateId",
		SpecRef:  "https://spec.openapis.org/oas/v3.1.0#operation-object",
		Context:  "Duplicate operation IDs found",
	}

	// Verify fields are accessible
	assert.Equal(t, "paths./test", issue.Path)
	assert.Equal(t, "Test message", issue.Message)
	assert.Equal(t, severity.SeverityWarning, issue.Severity)
	assert.Equal(t, "operationId", issue.Field)
	assert.Equal(t, "duplicateId", issue.Value)
	assert.Equal(t, "https://spec.openapis.org/oas/v3.1.0#operation-object", issue.SpecRef)
	assert.Equal(t, "Duplicate operation IDs found", issue.Context)

	// Verify String() incorporates relevant fields
	result := issue.String()
	assert.Contains(t, result, "paths./test")
	assert.Contains(t, result, "Test message")
	assert.Contains(t, result, "Spec: https://spec.openapis.org/oas/v3.1.0#operation-object")
	assert.Contains(t, result, "Context: Duplicate operation IDs found")
}

// TestIssueMultilineFormatting verifies that issues with SpecRef and Context
// produce properly formatted multiline output.
func TestIssueMultilineFormatting(t *testing.T) {
	issue := Issue{
		Path:     "components.schemas.User",
		Message:  "Schema conversion issue",
		Severity: severity.SeverityCritical,
		SpecRef:  "https://spec.openapis.org/oas/v3.0.0#schema-object",
		Context:  "Type array [string, null] not supported in OAS 2.0",
	}

	result := issue.String()
	lines := strings.Split(result, "\n")

	// Should have 3 lines: main message, SpecRef indent, Context indent
	assert.Len(t, lines, 3, "Issue with SpecRef and Context should have 3 lines")

	// First line: symbol, path, message
	assert.Contains(t, lines[0], "✗")
	assert.Contains(t, lines[0], "components.schemas.User")
	assert.Contains(t, lines[0], "Schema conversion issue")

	// Second line: indented SpecRef
	assert.True(t, strings.HasPrefix(lines[1], "    "), "SpecRef line should be indented with 4 spaces")
	assert.Contains(t, lines[1], "Spec:")

	// Third line: indented Context
	assert.True(t, strings.HasPrefix(lines[2], "    "), "Context line should be indented with 4 spaces")
	assert.Contains(t, lines[2], "Context:")
}
