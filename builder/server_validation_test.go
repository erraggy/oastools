package builder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erraggy/oastools/httpvalidator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextWithValidationResult(t *testing.T) {
	t.Parallel()

	result := &httpvalidator.RequestValidationResult{
		PathParams: map[string]any{"petId": int64(123)},
	}

	ctx := contextWithValidationResult(context.Background(), result)
	retrieved := validationResultFromContext(ctx)

	require.NotNil(t, retrieved)
	assert.Equal(t, int64(123), retrieved.PathParams["petId"])
}

func TestValidationResultFromContext_NoResult(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	result := validationResultFromContext(ctx)

	assert.Nil(t, result)
}

func TestValidationResultFromContext_WrongType(t *testing.T) {
	t.Parallel()

	// Store something other than *httpvalidator.RequestValidationResult
	ctx := context.WithValue(context.Background(), validationResultKey{}, "wrong type")
	result := validationResultFromContext(ctx)

	assert.Nil(t, result)
}

func TestWriteValidationError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	writeValidationError(rec, http.StatusBadRequest, "invalid parameter")

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "invalid parameter", body["error"])
}

func TestWriteValidationResult(t *testing.T) {
	t.Parallel()

	result := &httpvalidator.RequestValidationResult{
		Errors: []httpvalidator.ValidationError{
			{Path: "/petId", Message: "must be an integer"},
			{Path: "/query.limit", Message: "must be positive"},
		},
	}

	rec := httptest.NewRecorder()
	writeValidationResult(rec, result)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "validation failed", body["error"])

	errors, ok := body["errors"].([]any)
	require.True(t, ok, "Expected errors to be an array")

	assert.Len(t, errors, 2)
}

func TestWriteValidationResult_WithWarnings(t *testing.T) {
	t.Parallel()

	result := &httpvalidator.RequestValidationResult{
		Errors: []httpvalidator.ValidationError{
			{Path: "/petId", Message: "must be an integer"},
		},
		Warnings: []httpvalidator.ValidationError{
			{Path: "/query.extra", Message: "unknown parameter"},
		},
	}

	rec := httptest.NewRecorder()
	writeValidationResult(rec, result)

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)

	warnings, ok := body["warnings"].([]any)
	require.True(t, ok, "Expected warnings to be an array")

	assert.Len(t, warnings, 1)
}

func TestWriteValidationResult_NoWarnings(t *testing.T) {
	t.Parallel()

	result := &httpvalidator.RequestValidationResult{
		Errors: []httpvalidator.ValidationError{
			{Path: "/petId", Message: "must be an integer"},
		},
		Warnings: []httpvalidator.ValidationError{}, // Empty, should not appear in response
	}

	rec := httptest.NewRecorder()
	writeValidationResult(rec, result)

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)

	_, hasWarnings := body["warnings"]
	assert.False(t, hasWarnings, "Expected no warnings field when warnings slice is empty")
}

func TestValidationMiddleware_Disabled(t *testing.T) {
	t.Parallel()

	// Test with validation disabled - should pass through without validation
	cfg := ValidationConfig{
		IncludeRequestValidation: false,
	}

	var nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create a minimal validator (won't be used since validation is disabled)
	handler := validationMiddleware(nil, cfg)(next)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler.ServeHTTP(rec, req)

	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}
