package builder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/erraggy/oastools/parser"
)

func TestNewServerBuilder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version parser.OASVersion
	}{
		{"OAS 3.2.0", parser.OASVersion320},
		{"OAS 3.1.0", parser.OASVersion310},
		{"OAS 3.0.0", parser.OASVersion300},
		{"OAS 2.0", parser.OASVersion20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := NewServerBuilder(tt.version)
			if srv == nil {
				t.Fatal("NewServerBuilder returned nil")
			}
			if srv.Builder == nil {
				t.Error("ServerBuilder.Builder is nil")
			}
			if srv.handlers == nil {
				t.Error("ServerBuilder.handlers is nil")
			}
			if srv.middleware == nil {
				t.Error("ServerBuilder.middleware is nil")
			}
		})
	}
}

func TestFromBuilder(t *testing.T) {
	t.Parallel()

	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv := FromBuilder(b)

	if srv == nil {
		t.Fatal("FromBuilder returned nil")
	}
	if srv.Builder != b {
		t.Error("FromBuilder did not preserve the original builder")
	}
}

func TestServerBuilder_Handle(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320)

	handler := func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, nil)
	}

	result := srv.Handle("testOp", handler)

	if result != srv {
		t.Error("Handle did not return the same ServerBuilder for chaining")
	}

	srv.mu.RLock()
	if _, ok := srv.handlers["testOp"]; !ok {
		t.Error("Handler was not registered")
	}
	srv.mu.RUnlock()
}

func TestServerBuilder_HandleFunc(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320)

	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}

	result := srv.HandleFunc("testOp", handler)

	if result != srv {
		t.Error("HandleFunc did not return the same ServerBuilder for chaining")
	}

	srv.mu.RLock()
	if _, ok := srv.handlers["testOp"]; !ok {
		t.Error("Handler was not registered")
	}
	srv.mu.RUnlock()
}

func TestServerBuilder_Use(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320)

	mw1 := func(next http.Handler) http.Handler {
		return next
	}
	mw2 := func(next http.Handler) http.Handler {
		return next
	}

	srv.Use(mw1, mw2)

	if len(srv.middleware) != 2 {
		t.Errorf("Expected 2 middleware, got %d", len(srv.middleware))
	}
}

func TestServerBuilder_BuildServer(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/pets",
		WithOperationID("listPets"),
		WithResponse(http.StatusOK, []string{}),
	)

	srv.Handle("listPets", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, []string{"dog", "cat"})
	})

	result, err := srv.BuildServer()
	if err != nil {
		t.Fatalf("BuildServer failed: %v", err)
	}

	if result == nil {
		t.Fatal("BuildServer returned nil result")
	}
	if result.Handler == nil {
		t.Error("ServerResult.Handler is nil")
	}
	if result.Spec == nil {
		t.Error("ServerResult.Spec is nil")
	}
	if result.ParseResult == nil {
		t.Error("ServerResult.ParseResult is nil")
	}
}

func TestServerBuilder_BuildServer_WithValidation(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/pets",
		WithOperationID("listPets"),
		WithResponse(http.StatusOK, []string{}),
	)

	srv.Handle("listPets", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, []string{"dog", "cat"})
	})

	result, err := srv.BuildServer()
	if err != nil {
		t.Fatalf("BuildServer failed: %v", err)
	}

	if result.Validator == nil {
		t.Error("ServerResult.Validator is nil when validation is enabled")
	}
}

func TestServerBuilder_MustBuildServer(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/health",
		WithOperationID("healthCheck"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	srv.Handle("healthCheck", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	result := srv.MustBuildServer()

	if result == nil {
		t.Fatal("MustBuildServer returned nil")
	}
}

func TestServerBuilder_MustBuildServer_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuildServer did not panic on error")
		}
	}()

	// Create a builder with a config error
	srv := NewServerBuilder(parser.OASVersion320)
	srv.Builder.configError = http.ErrAbortHandler // Force an error

	_ = srv.MustBuildServer()
}

func TestServerBuilder_EndToEnd(t *testing.T) {
	t.Parallel()

	type Pet struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Pet Store").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/pets",
		WithOperationID("listPets"),
		WithResponse(http.StatusOK, []Pet{}),
	)

	srv.AddOperation(http.MethodGet, "/pets/{petId}",
		WithOperationID("getPet"),
		WithPathParam("petId", int64(0)),
		WithResponse(http.StatusOK, Pet{}),
		WithResponse(http.StatusNotFound, map[string]string{}),
	)

	srv.AddOperation(http.MethodPost, "/pets",
		WithOperationID("createPet"),
		WithRequestBody("application/json", Pet{}, WithRequired(true)),
		WithResponse(http.StatusCreated, Pet{}),
	)

	pets := []Pet{{ID: 1, Name: "Fluffy"}, {ID: 2, Name: "Spot"}}

	srv.Handle("listPets", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, pets)
	})

	srv.Handle("getPet", func(_ context.Context, req *Request) Response {
		petID, ok := req.PathParams["petId"].(string)
		if !ok || petID != "1" {
			return Error(http.StatusNotFound, "pet not found")
		}
		return JSON(http.StatusOK, pets[0])
	})

	srv.Handle("createPet", func(_ context.Context, req *Request) Response {
		// Body is already parsed as map
		return JSON(http.StatusCreated, req.Body)
	})

	result, err := srv.BuildServer()
	if err != nil {
		t.Fatalf("BuildServer failed: %v", err)
	}

	// Test GET /pets
	t.Run("GET /pets", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/pets", nil)
		result.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var gotPets []Pet
		if err := json.NewDecoder(rec.Body).Decode(&gotPets); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(gotPets) != 2 {
			t.Errorf("Expected 2 pets, got %d", len(gotPets))
		}
	})

	// Test GET /pets/1
	t.Run("GET /pets/1", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/pets/1", nil)
		result.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	// Test GET /pets/999 (not found)
	t.Run("GET /pets/999", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/pets/999", nil)
		result.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	// Test POST /pets
	t.Run("POST /pets", func(t *testing.T) {
		body := `{"id": 3, "name": "Buddy"}`
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/pets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		result.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", rec.Code)
		}
	})

	// Test 404 for unknown path
	t.Run("Unknown path", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
		result.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	// Test 405 for unsupported method
	t.Run("Method not allowed", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/pets", nil)
		result.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", rec.Code)
		}
	})
}

func TestServerBuilder_UnhandledOperation(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/unhandled",
		WithOperationID("unhandledOp"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	// Note: We don't register a handler for "unhandledOp"

	result, err := srv.BuildServer()
	if err != nil {
		t.Fatalf("BuildServer failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unhandled", nil)
	result.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("Expected status 501, got %d", rec.Code)
	}
}

func TestServerBuilder_WithMiddleware(t *testing.T) {
	t.Parallel()

	var middlewareCalled bool

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/test",
		WithOperationID("test"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	srv.Handle("test", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, nil)
	})

	srv.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	})

	result, err := srv.BuildServer()
	if err != nil {
		t.Fatalf("BuildServer failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	result.Handler.ServeHTTP(rec, req)

	if !middlewareCalled {
		t.Error("Middleware was not called")
	}
}

func TestServerBuilder_WithRecovery(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation(), WithRecovery()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/panic",
		WithOperationID("panic"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	srv.Handle("panic", func(_ context.Context, _ *Request) Response {
		panic("test panic")
	})

	result, err := srv.BuildServer()
	if err != nil {
		t.Fatalf("BuildServer failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	// Should not panic
	result.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 after panic, got %d", rec.Code)
	}
}

func TestServerBuilder_WithLogging(t *testing.T) {
	t.Parallel()

	var loggedMethod, loggedPath string
	var loggedStatus int

	srv := NewServerBuilder(parser.OASVersion320,
		WithoutValidation(),
		WithRequestLogging(func(method, path string, status int, _ time.Duration) {
			loggedMethod = method
			loggedPath = path
			loggedStatus = status
		}),
	).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/logged",
		WithOperationID("logged"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	srv.Handle("logged", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, nil)
	})

	result, err := srv.BuildServer()
	if err != nil {
		t.Fatalf("BuildServer failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/logged", nil)
	result.Handler.ServeHTTP(rec, req)

	if loggedMethod != http.MethodGet {
		t.Errorf("Expected logged method GET, got %s", loggedMethod)
	}
	if loggedPath != "/logged" {
		t.Errorf("Expected logged path /logged, got %s", loggedPath)
	}
	if loggedStatus != http.StatusOK {
		t.Errorf("Expected logged status 200, got %d", loggedStatus)
	}
}

func TestServerBuilderOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithRouter", func(t *testing.T) {
		t.Parallel()
		router := &stdlibRouter{}
		srv := NewServerBuilder(parser.OASVersion320, WithRouter(router))
		if srv.router != router {
			t.Error("WithRouter did not set the router")
		}
	})

	t.Run("WithStdlibRouter", func(t *testing.T) {
		t.Parallel()
		srv := NewServerBuilder(parser.OASVersion320, WithStdlibRouter())
		if srv.router == nil {
			t.Error("WithStdlibRouter did not set the router")
		}
	})

	t.Run("WithoutValidation", func(t *testing.T) {
		t.Parallel()
		srv := NewServerBuilder(parser.OASVersion320, WithoutValidation())
		if srv.config.enableValidation {
			t.Error("WithoutValidation did not disable validation")
		}
	})

	t.Run("WithValidationConfig", func(t *testing.T) {
		t.Parallel()
		cfg := ValidationConfig{StrictMode: true}
		srv := NewServerBuilder(parser.OASVersion320, WithValidationConfig(cfg))
		if !srv.config.validationConfig.StrictMode {
			t.Error("WithValidationConfig did not set strict mode")
		}
	})

	t.Run("WithErrorHandler", func(t *testing.T) {
		t.Parallel()
		handler := func(_ http.ResponseWriter, _ *http.Request, _ error) {}
		srv := NewServerBuilder(parser.OASVersion320, WithErrorHandler(handler))
		if srv.errorHandler == nil {
			t.Error("WithErrorHandler did not set error handler")
		}
	})

	t.Run("WithNotFoundHandler", func(t *testing.T) {
		t.Parallel()
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		srv := NewServerBuilder(parser.OASVersion320, WithNotFoundHandler(handler))
		if srv.config.notFoundHandler == nil {
			t.Error("WithNotFoundHandler did not set handler")
		}
	})

	t.Run("WithMethodNotAllowedHandler", func(t *testing.T) {
		t.Parallel()
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusMethodNotAllowed)
		})
		srv := NewServerBuilder(parser.OASVersion320, WithMethodNotAllowedHandler(handler))
		if srv.config.methodNotAllowed == nil {
			t.Error("WithMethodNotAllowedHandler did not set handler")
		}
	})

	t.Run("WithRecovery", func(t *testing.T) {
		t.Parallel()
		srv := NewServerBuilder(parser.OASVersion320, WithRecovery())
		if !srv.config.enableRecovery {
			t.Error("WithRecovery did not enable recovery")
		}
	})
}

func TestDefaultValidationConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultValidationConfig()

	if !cfg.IncludeRequestValidation {
		t.Error("Default should include request validation")
	}
	if cfg.IncludeResponseValidation {
		t.Error("Default should not include response validation")
	}
	if cfg.StrictMode {
		t.Error("Default should not use strict mode")
	}
	if cfg.OnValidationError != nil {
		t.Error("Default should have nil error handler")
	}
}

func TestResponseCapture(t *testing.T) {
	t.Parallel()

	rec := &responseCapture{header: make(http.Header)}

	// Test Header
	rec.Header().Set("X-Test", "value")
	if rec.header.Get("X-Test") != "value" {
		t.Error("Header() did not work correctly")
	}

	// Test Write
	n, err := rec.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected 5 bytes written, got %d", n)
	}
	if string(rec.body) != "hello" {
		t.Errorf("Expected body 'hello', got '%s'", string(rec.body))
	}

	// Test WriteHeader
	rec.WriteHeader(http.StatusCreated)
	if rec.status != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.status)
	}
}

func TestCapturedResponse(t *testing.T) {
	t.Parallel()

	t.Run("with status", func(t *testing.T) {
		t.Parallel()
		resp := &capturedResponse{
			status:  http.StatusCreated,
			headers: http.Header{"X-Test": {"value"}},
			body:    []byte("test"),
		}

		if resp.StatusCode() != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode())
		}
		if resp.Headers().Get("X-Test") != "value" {
			t.Error("Headers not returned correctly")
		}
		if string(resp.Body().([]byte)) != "test" {
			t.Error("Body not returned correctly")
		}
	})

	t.Run("default status", func(t *testing.T) {
		t.Parallel()
		resp := &capturedResponse{}
		if resp.StatusCode() != http.StatusOK {
			t.Errorf("Expected default status 200, got %d", resp.StatusCode())
		}
	})

	t.Run("WriteTo", func(t *testing.T) {
		t.Parallel()
		resp := &capturedResponse{
			status:  http.StatusOK,
			headers: http.Header{"X-Custom": {"test"}},
			body:    []byte("response body"),
		}

		rec := httptest.NewRecorder()
		if err := resp.WriteTo(rec); err != nil {
			t.Fatalf("WriteTo failed: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
		if rec.Header().Get("X-Custom") != "test" {
			t.Error("Custom header not set")
		}
		if rec.Body.String() != "response body" {
			t.Errorf("Expected body 'response body', got '%s'", rec.Body.String())
		}
	})
}
