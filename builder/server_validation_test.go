package builder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erraggy/oastools/httpvalidator"
)

func TestContextWithValidationResult(t *testing.T) {
	t.Parallel()

	result := &httpvalidator.RequestValidationResult{
		PathParams: map[string]any{"petId": int64(123)},
	}

	ctx := contextWithValidationResult(context.Background(), result)
	retrieved := validationResultFromContext(ctx)

	if retrieved == nil {
		t.Fatal("Expected to retrieve validation result from context")
	}
	if retrieved.PathParams["petId"] != int64(123) {
		t.Errorf("Expected petId=123, got %v", retrieved.PathParams["petId"])
	}
}

func TestValidationResultFromContext_NoResult(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	result := validationResultFromContext(ctx)

	if result != nil {
		t.Error("Expected nil when no validation result in context")
	}
}

func TestValidationResultFromContext_WrongType(t *testing.T) {
	t.Parallel()

	// Store something other than *httpvalidator.RequestValidationResult
	ctx := context.WithValue(context.Background(), validationResultKey{}, "wrong type")
	result := validationResultFromContext(ctx)

	if result != nil {
		t.Error("Expected nil when wrong type in context")
	}
}

func TestWriteValidationError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	writeValidationError(rec, http.StatusBadRequest, "invalid parameter")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if body["error"] != "invalid parameter" {
		t.Errorf("Expected error message 'invalid parameter', got '%s'", body["error"])
	}
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

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if body["error"] != "validation failed" {
		t.Errorf("Expected error message 'validation failed', got '%s'", body["error"])
	}

	errors, ok := body["errors"].([]any)
	if !ok {
		t.Fatal("Expected errors to be an array")
	}

	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}
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
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	warnings, ok := body["warnings"].([]any)
	if !ok {
		t.Fatal("Expected warnings to be an array")
	}

	if len(warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(warnings))
	}
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
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if _, hasWarnings := body["warnings"]; hasWarnings {
		t.Error("Expected no warnings field when warnings slice is empty")
	}
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

	if !nextCalled {
		t.Error("Expected next handler to be called when validation is disabled")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}
