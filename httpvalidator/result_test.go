package httpvalidator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Severity Constants Tests
// =============================================================================

func TestSeverityConstants(t *testing.T) {
	// Verify severity constants are properly defined
	// These are re-exports from severity package
	// Severity is an int type, so we just verify the constants exist and are distinct

	// Verify they are distinct from each other
	assert.NotEqual(t, SeverityError, SeverityWarning, "error and warning should be different")
	assert.NotEqual(t, SeverityError, SeverityInfo, "error and info should be different")
	assert.NotEqual(t, SeverityError, SeverityCritical, "error and critical should be different")
	assert.NotEqual(t, SeverityWarning, SeverityInfo, "warning and info should be different")
	assert.NotEqual(t, SeverityWarning, SeverityCritical, "warning and critical should be different")
	assert.NotEqual(t, SeverityInfo, SeverityCritical, "info and critical should be different")

	// Verify Severity type alias works
	sev := SeverityError
	assert.Equal(t, SeverityError, sev)
}

// =============================================================================
// ValidationLocation Constants Tests
// =============================================================================

func TestValidationLocationConstants(t *testing.T) {
	assert.Equal(t, ValidationLocation("path"), LocationPath)
	assert.Equal(t, ValidationLocation("query"), LocationQuery)
	assert.Equal(t, ValidationLocation("header"), LocationHeader)
	assert.Equal(t, ValidationLocation("cookie"), LocationCookie)
	assert.Equal(t, ValidationLocation("requestBody"), LocationRequestBody)
	assert.Equal(t, ValidationLocation("response"), LocationResponse)
}

// =============================================================================
// newRequestResult Tests
// =============================================================================

func TestNewRequestResult(t *testing.T) {
	result := newRequestResult()

	assert.True(t, result.Valid, "new request result should be valid by default")
	assert.Empty(t, result.Errors, "new request result should have no errors")
	assert.Empty(t, result.Warnings, "new request result should have no warnings")
	assert.Empty(t, result.MatchedPath, "matched path should be empty")
	assert.Empty(t, result.MatchedMethod, "matched method should be empty")
	assert.NotNil(t, result.PathParams, "path params map should be initialized")
	assert.NotNil(t, result.QueryParams, "query params map should be initialized")
	assert.NotNil(t, result.HeaderParams, "header params map should be initialized")
	assert.NotNil(t, result.CookieParams, "cookie params map should be initialized")
}

// =============================================================================
// newResponseResult Tests
// =============================================================================

func TestNewResponseResult(t *testing.T) {
	result := newResponseResult()

	assert.True(t, result.Valid, "new response result should be valid by default")
	assert.Empty(t, result.Errors, "new response result should have no errors")
	assert.Empty(t, result.Warnings, "new response result should have no warnings")
	assert.Equal(t, 0, result.StatusCode, "status code should be zero")
	assert.Empty(t, result.ContentType, "content type should be empty")
	assert.Empty(t, result.MatchedPath, "matched path should be empty")
	assert.Empty(t, result.MatchedMethod, "matched method should be empty")
}

// =============================================================================
// RequestValidationResult.addError Tests
// =============================================================================

func TestRequestValidationResult_AddError(t *testing.T) {
	t.Run("adds error and marks result invalid", func(t *testing.T) {
		result := newRequestResult()
		result.addError("path.userId", "user ID must be an integer", SeverityError)

		assert.False(t, result.Valid, "result should be invalid after adding error")
		assert.Len(t, result.Errors, 1, "should have one error")
		assert.Equal(t, "path.userId", result.Errors[0].Path)
		assert.Equal(t, "user ID must be an integer", result.Errors[0].Message)
		assert.Equal(t, SeverityError, result.Errors[0].Severity)
	})

	t.Run("adds multiple errors", func(t *testing.T) {
		result := newRequestResult()
		result.addError("path.userId", "error 1", SeverityError)
		result.addError("query.limit", "error 2", SeverityWarning)
		result.addError("header.X-API-Key", "error 3", SeverityCritical)

		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 3)
	})

	t.Run("preserves different severity levels", func(t *testing.T) {
		result := newRequestResult()
		result.addError("path", "critical error", SeverityCritical)
		result.addError("query", "info", SeverityInfo)

		assert.Equal(t, SeverityCritical, result.Errors[0].Severity)
		assert.Equal(t, SeverityInfo, result.Errors[1].Severity)
	})
}

// =============================================================================
// RequestValidationResult.addWarning Tests
// =============================================================================

func TestRequestValidationResult_AddWarning(t *testing.T) {
	t.Run("adds warning without marking invalid", func(t *testing.T) {
		result := newRequestResult()
		result.addWarning("requestBody", "optional field missing")

		assert.True(t, result.Valid, "warnings should not mark result invalid")
		assert.Len(t, result.Warnings, 1)
		assert.Equal(t, "requestBody", result.Warnings[0].Path)
		assert.Equal(t, "optional field missing", result.Warnings[0].Message)
		assert.Equal(t, SeverityWarning, result.Warnings[0].Severity)
	})

	t.Run("adds multiple warnings", func(t *testing.T) {
		result := newRequestResult()
		result.addWarning("path1", "warning 1")
		result.addWarning("path2", "warning 2")
		result.addWarning("path3", "warning 3")

		assert.True(t, result.Valid)
		assert.Len(t, result.Warnings, 3)
	})
}

// =============================================================================
// ResponseValidationResult.addError Tests
// =============================================================================

func TestResponseValidationResult_AddError(t *testing.T) {
	t.Run("adds error and marks result invalid", func(t *testing.T) {
		result := newResponseResult()
		result.addError("response.body", "invalid JSON", SeverityError)

		assert.False(t, result.Valid, "result should be invalid after adding error")
		assert.Len(t, result.Errors, 1)
		assert.Equal(t, "response.body", result.Errors[0].Path)
		assert.Equal(t, "invalid JSON", result.Errors[0].Message)
		assert.Equal(t, SeverityError, result.Errors[0].Severity)
	})

	t.Run("adds multiple errors", func(t *testing.T) {
		result := newResponseResult()
		result.addError("response.body", "error 1", SeverityError)
		result.addError("response.header.X-Custom", "error 2", SeverityError)

		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 2)
	})
}

// =============================================================================
// ResponseValidationResult.addWarning Tests
// =============================================================================

func TestResponseValidationResult_AddWarning(t *testing.T) {
	t.Run("adds warning without marking invalid", func(t *testing.T) {
		result := newResponseResult()
		result.addWarning("response.body", "schema not defined for content type")

		assert.True(t, result.Valid, "warnings should not mark result invalid")
		assert.Len(t, result.Warnings, 1)
		assert.Equal(t, "response.body", result.Warnings[0].Path)
		assert.Equal(t, "schema not defined for content type", result.Warnings[0].Message)
		assert.Equal(t, SeverityWarning, result.Warnings[0].Severity)
	})

	t.Run("adds multiple warnings", func(t *testing.T) {
		result := newResponseResult()
		result.addWarning("path1", "warning 1")
		result.addWarning("path2", "warning 2")

		assert.True(t, result.Valid)
		assert.Len(t, result.Warnings, 2)
	})
}

// =============================================================================
// Combined Error and Warning Tests
// =============================================================================

func TestRequestResult_ErrorsAndWarnings(t *testing.T) {
	result := newRequestResult()

	// Add warnings first
	result.addWarning("w1", "warning one")
	result.addWarning("w2", "warning two")

	assert.True(t, result.Valid, "should still be valid with only warnings")
	assert.Len(t, result.Warnings, 2)
	assert.Len(t, result.Errors, 0)

	// Add an error
	result.addError("e1", "error one", SeverityError)

	assert.False(t, result.Valid, "should be invalid after adding error")
	assert.Len(t, result.Warnings, 2, "warnings should be preserved")
	assert.Len(t, result.Errors, 1)
}

func TestResponseResult_ErrorsAndWarnings(t *testing.T) {
	result := newResponseResult()

	// Add warnings first
	result.addWarning("w1", "warning one")

	assert.True(t, result.Valid)
	assert.Len(t, result.Warnings, 1)
	assert.Len(t, result.Errors, 0)

	// Add an error
	result.addError("e1", "error one", SeverityCritical)

	assert.False(t, result.Valid)
	assert.Len(t, result.Warnings, 1, "warnings should be preserved")
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, SeverityCritical, result.Errors[0].Severity)
}

// =============================================================================
// Request Result Param Maps Tests
// =============================================================================

func TestRequestResult_ParamMaps(t *testing.T) {
	result := newRequestResult()

	// Verify we can add to the param maps
	result.PathParams["userId"] = 123
	result.QueryParams["limit"] = 10
	result.HeaderParams["X-API-Key"] = "secret"
	result.CookieParams["session"] = "abc123"

	assert.Equal(t, 123, result.PathParams["userId"])
	assert.Equal(t, 10, result.QueryParams["limit"])
	assert.Equal(t, "secret", result.HeaderParams["X-API-Key"])
	assert.Equal(t, "abc123", result.CookieParams["session"])
}

// =============================================================================
// Response Result Fields Tests
// =============================================================================

func TestResponseResult_Fields(t *testing.T) {
	result := newResponseResult()

	// Set fields
	result.StatusCode = 200
	result.ContentType = "application/json"
	result.MatchedPath = "/users/{id}"
	result.MatchedMethod = "GET"

	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, "application/json", result.ContentType)
	assert.Equal(t, "/users/{id}", result.MatchedPath)
	assert.Equal(t, "GET", result.MatchedMethod)
}

// =============================================================================
// ValidationError Type Tests
// =============================================================================

func TestValidationError(t *testing.T) {
	// ValidationError is an alias to issues.Issue
	// Test that it works as expected
	err := ValidationError{
		Path:     "test.path",
		Message:  "test message",
		Severity: SeverityError,
	}

	assert.Equal(t, "test.path", err.Path)
	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, SeverityError, err.Severity)
}
