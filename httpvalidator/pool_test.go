package httpvalidator

import (
	"testing"
)

func TestRequestResultPool(t *testing.T) {
	t.Run("get returns valid result", func(t *testing.T) {
		r := getRequestResult()
		if r == nil {
			t.Fatal("getRequestResult() returned nil")
		}
		if !r.Valid {
			t.Error("expected Valid to be true")
		}
		if r.MatchedPath != "" {
			t.Error("expected MatchedPath to be empty")
		}
		if r.MatchedMethod != "" {
			t.Error("expected MatchedMethod to be empty")
		}
		if len(r.Errors) != 0 {
			t.Errorf("expected Errors to be empty, got %d", len(r.Errors))
		}
		if len(r.Warnings) != 0 {
			t.Errorf("expected Warnings to be empty, got %d", len(r.Warnings))
		}
		if r.PathParams == nil {
			t.Error("expected PathParams to be initialized")
		}
		if r.QueryParams == nil {
			t.Error("expected QueryParams to be initialized")
		}
		if r.HeaderParams == nil {
			t.Error("expected HeaderParams to be initialized")
		}
		if r.CookieParams == nil {
			t.Error("expected CookieParams to be initialized")
		}
		putRequestResult(r)
	})

	t.Run("reset clears all fields", func(t *testing.T) {
		r := getRequestResult()

		// Populate with data
		r.Valid = false
		r.MatchedPath = "/pets/{petId}"
		r.MatchedMethod = "GET"
		r.addError("test.path", "test error", SeverityError)
		r.addWarning("test.path", "test warning")
		r.PathParams["petId"] = "123"
		r.QueryParams["limit"] = 10
		r.HeaderParams["X-Custom"] = "value"
		r.CookieParams["session"] = "abc"

		// Reset and verify
		r.reset()

		if !r.Valid {
			t.Error("expected Valid to be true after reset")
		}
		if r.MatchedPath != "" {
			t.Error("expected MatchedPath to be empty after reset")
		}
		if r.MatchedMethod != "" {
			t.Error("expected MatchedMethod to be empty after reset")
		}
		if len(r.Errors) != 0 {
			t.Errorf("expected Errors to be empty after reset, got %d", len(r.Errors))
		}
		if len(r.Warnings) != 0 {
			t.Errorf("expected Warnings to be empty after reset, got %d", len(r.Warnings))
		}
		if len(r.PathParams) != 0 {
			t.Errorf("expected PathParams to be empty after reset, got %d", len(r.PathParams))
		}
		if len(r.QueryParams) != 0 {
			t.Errorf("expected QueryParams to be empty after reset, got %d", len(r.QueryParams))
		}
		if len(r.HeaderParams) != 0 {
			t.Errorf("expected HeaderParams to be empty after reset, got %d", len(r.HeaderParams))
		}
		if len(r.CookieParams) != 0 {
			t.Errorf("expected CookieParams to be empty after reset, got %d", len(r.CookieParams))
		}

		putRequestResult(r)
	})

	t.Run("put nil is safe", func(t *testing.T) {
		// Should not panic
		putRequestResult(nil)
	})

	t.Run("reuse preserves capacity", func(t *testing.T) {
		r := getRequestResult()

		// Add errors to use slice
		for range 5 {
			r.addError("path", "error", SeverityError)
		}

		putRequestResult(r)

		// Get again and verify minimum capacity is preserved
		r2 := getRequestResult()
		if cap(r2.Errors) < requestResultErrorsCap {
			t.Errorf("expected Errors capacity >= %d, got %d", requestResultErrorsCap, cap(r2.Errors))
		}
		putRequestResult(r2)
	})
}

func TestResponseResultPool(t *testing.T) {
	t.Run("get returns valid result", func(t *testing.T) {
		r := getResponseResult()
		if r == nil {
			t.Fatal("getResponseResult() returned nil")
		}
		if !r.Valid {
			t.Error("expected Valid to be true")
		}
		if r.MatchedPath != "" {
			t.Error("expected MatchedPath to be empty")
		}
		if r.MatchedMethod != "" {
			t.Error("expected MatchedMethod to be empty")
		}
		if r.StatusCode != 0 {
			t.Errorf("expected StatusCode to be 0, got %d", r.StatusCode)
		}
		if r.ContentType != "" {
			t.Error("expected ContentType to be empty")
		}
		if len(r.Errors) != 0 {
			t.Errorf("expected Errors to be empty, got %d", len(r.Errors))
		}
		if len(r.Warnings) != 0 {
			t.Errorf("expected Warnings to be empty, got %d", len(r.Warnings))
		}
		putResponseResult(r)
	})

	t.Run("reset clears all fields", func(t *testing.T) {
		r := getResponseResult()

		// Populate with data
		r.Valid = false
		r.MatchedPath = "/pets/{petId}"
		r.MatchedMethod = "GET"
		r.StatusCode = 200
		r.ContentType = "application/json"
		r.addError("test.path", "test error", SeverityError)
		r.addWarning("test.path", "test warning")

		// Reset and verify
		r.reset()

		if !r.Valid {
			t.Error("expected Valid to be true after reset")
		}
		if r.MatchedPath != "" {
			t.Error("expected MatchedPath to be empty after reset")
		}
		if r.MatchedMethod != "" {
			t.Error("expected MatchedMethod to be empty after reset")
		}
		if r.StatusCode != 0 {
			t.Errorf("expected StatusCode to be 0 after reset, got %d", r.StatusCode)
		}
		if r.ContentType != "" {
			t.Error("expected ContentType to be empty after reset")
		}
		if len(r.Errors) != 0 {
			t.Errorf("expected Errors to be empty after reset, got %d", len(r.Errors))
		}
		if len(r.Warnings) != 0 {
			t.Errorf("expected Warnings to be empty after reset, got %d", len(r.Warnings))
		}

		putResponseResult(r)
	})

	t.Run("put nil is safe", func(t *testing.T) {
		// Should not panic
		putResponseResult(nil)
	})
}

func BenchmarkRequestResultPool(b *testing.B) {
	b.Run("pooled", func(b *testing.B) {
		for b.Loop() {
			r := getRequestResult()
			r.MatchedPath = "/pets/{petId}"
			r.MatchedMethod = "GET"
			r.addError("path", "test error", SeverityError)
			r.PathParams["petId"] = "123"
			putRequestResult(r)
		}
	})

	b.Run("non-pooled", func(b *testing.B) {
		for b.Loop() {
			r := newRequestResult()
			r.MatchedPath = "/pets/{petId}"
			r.MatchedMethod = "GET"
			r.addError("path", "test error", SeverityError)
			r.PathParams["petId"] = "123"
			_ = r
		}
	})
}

func BenchmarkResponseResultPool(b *testing.B) {
	b.Run("pooled", func(b *testing.B) {
		for b.Loop() {
			r := getResponseResult()
			r.MatchedPath = "/pets/{petId}"
			r.MatchedMethod = "GET"
			r.StatusCode = 200
			r.ContentType = "application/json"
			r.addError("path", "test error", SeverityError)
			putResponseResult(r)
		}
	})

	b.Run("non-pooled", func(b *testing.B) {
		for b.Loop() {
			r := newResponseResult()
			r.MatchedPath = "/pets/{petId}"
			r.MatchedMethod = "GET"
			r.StatusCode = 200
			r.ContentType = "application/json"
			r.addError("path", "test error", SeverityError)
			_ = r
		}
	})
}

func BenchmarkRequestResultPoolParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r := getRequestResult()
			r.MatchedPath = "/pets/{petId}"
			r.MatchedMethod = "GET"
			r.addError("path", "test error", SeverityError)
			r.PathParams["petId"] = "123"
			putRequestResult(r)
		}
	})
}

func BenchmarkResponseResultPoolParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r := getResponseResult()
			r.MatchedPath = "/pets/{petId}"
			r.MatchedMethod = "GET"
			r.StatusCode = 200
			r.ContentType = "application/json"
			r.addError("path", "test error", SeverityError)
			putResponseResult(r)
		}
	})
}
