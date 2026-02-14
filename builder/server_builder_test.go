package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.NotNil(t, srv)
			assert.NotNil(t, srv.Builder)
			assert.NotNil(t, srv.handlers)
			assert.NotNil(t, srv.middleware)
		})
	}
}

func TestFromBuilder(t *testing.T) {
	t.Parallel()

	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv := FromBuilder(b)

	require.NotNil(t, srv)
	assert.Equal(t, b, srv.Builder)
}

func TestServerBuilder_Handle(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320)

	handler := func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, nil)
	}

	result := srv.Handle(http.MethodGet, "/test", handler)

	assert.Equal(t, srv, result)

	srv.mu.RLock()
	methodHandlers, pathOk := srv.handlers["/test"]
	assert.True(t, pathOk, "Handler was not registered for path")
	_, methodOk := methodHandlers[http.MethodGet]
	assert.True(t, methodOk, "Handler was not registered for method")
	srv.mu.RUnlock()
}

func TestServerBuilder_HandleFunc(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320)

	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}

	result := srv.HandleFunc(http.MethodGet, "/test", handler)

	assert.Equal(t, srv, result)

	srv.mu.RLock()
	methodHandlers, pathOk := srv.handlers["/test"]
	assert.True(t, pathOk, "Handler was not registered for path")
	_, methodOk := methodHandlers[http.MethodGet]
	assert.True(t, methodOk, "Handler was not registered for method")
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

	assert.Len(t, srv.middleware, 2)
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

	srv.Handle(http.MethodGet, "/pets", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, []string{"dog", "cat"})
	})

	result, err := srv.BuildServer()
	require.NoError(t, err)

	require.NotNil(t, result)
	assert.NotNil(t, result.Handler)
	assert.NotNil(t, result.Spec)
	assert.NotNil(t, result.ParseResult)
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

	srv.Handle(http.MethodGet, "/pets", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, []string{"dog", "cat"})
	})

	result, err := srv.BuildServer()
	require.NoError(t, err)

	assert.NotNil(t, result.Validator)
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

	srv.Handle(http.MethodGet, "/health", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	result := srv.MustBuildServer()

	require.NotNil(t, result)
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
	srv.configError = http.ErrAbortHandler // Force an error

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

	srv.Handle(http.MethodGet, "/pets", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, pets)
	})

	srv.Handle(http.MethodGet, "/pets/{petId}", func(_ context.Context, req *Request) Response {
		petID, ok := req.PathParams["petId"].(string)
		if !ok || petID != "1" {
			return Error(http.StatusNotFound, "pet not found")
		}
		return JSON(http.StatusOK, pets[0])
	})

	srv.Handle(http.MethodPost, "/pets", func(_ context.Context, req *Request) Response {
		// Body is already parsed as map
		return JSON(http.StatusCreated, req.Body)
	})

	result, err := srv.BuildServer()
	require.NoError(t, err)

	// Test GET /pets
	t.Run("GET /pets", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/pets", nil)
		result.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var gotPets []Pet
		err := json.NewDecoder(rec.Body).Decode(&gotPets)
		require.NoError(t, err)

		assert.Len(t, gotPets, 2)
	})

	// Test GET /pets/1
	t.Run("GET /pets/1", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/pets/1", nil)
		result.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	// Test GET /pets/999 (not found)
	t.Run("GET /pets/999", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/pets/999", nil)
		result.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	// Test POST /pets
	t.Run("POST /pets", func(t *testing.T) {
		body := `{"id": 3, "name": "Buddy"}`
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/pets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		result.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	// Test 404 for unknown path
	t.Run("Unknown path", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
		result.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	// Test 405 for unsupported method
	t.Run("Method not allowed", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/pets", nil)
		result.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
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
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unhandled", nil)
	result.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotImplemented, rec.Code)
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

	srv.Handle(http.MethodGet, "/test", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, nil)
	})

	srv.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	})

	result, err := srv.BuildServer()
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	result.Handler.ServeHTTP(rec, req)

	assert.True(t, middlewareCalled)
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

	srv.Handle(http.MethodGet, "/panic", func(_ context.Context, _ *Request) Response {
		panic("test panic")
	})

	result, err := srv.BuildServer()
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	// Should not panic
	result.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
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

	srv.Handle(http.MethodGet, "/logged", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, nil)
	})

	result, err := srv.BuildServer()
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/logged", nil)
	result.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.MethodGet, loggedMethod)
	assert.Equal(t, "/logged", loggedPath)
	assert.Equal(t, http.StatusOK, loggedStatus)
}

func TestServerBuilderOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithRouter", func(t *testing.T) {
		t.Parallel()
		router := &stdlibRouter{}
		srv := NewServerBuilder(parser.OASVersion320, WithRouter(router))
		assert.Equal(t, router, srv.router)
	})

	t.Run("WithStdlibRouter", func(t *testing.T) {
		t.Parallel()
		srv := NewServerBuilder(parser.OASVersion320, WithStdlibRouter())
		assert.NotNil(t, srv.router)
	})

	t.Run("WithoutValidation", func(t *testing.T) {
		t.Parallel()
		srv := NewServerBuilder(parser.OASVersion320, WithoutValidation())
		assert.False(t, srv.config.enableValidation)
	})

	t.Run("WithValidationConfig", func(t *testing.T) {
		t.Parallel()
		cfg := ValidationConfig{StrictMode: true}
		srv := NewServerBuilder(parser.OASVersion320, WithValidationConfig(cfg))
		assert.True(t, srv.config.validationConfig.StrictMode)
	})

	t.Run("WithErrorHandler", func(t *testing.T) {
		t.Parallel()
		handler := func(_ http.ResponseWriter, _ *http.Request, _ error) {}
		srv := NewServerBuilder(parser.OASVersion320, WithErrorHandler(handler))
		assert.NotNil(t, srv.errorHandler)
	})

	t.Run("WithNotFoundHandler", func(t *testing.T) {
		t.Parallel()
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		srv := NewServerBuilder(parser.OASVersion320, WithNotFoundHandler(handler))
		assert.NotNil(t, srv.config.notFoundHandler)
	})

	t.Run("WithMethodNotAllowedHandler", func(t *testing.T) {
		t.Parallel()
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusMethodNotAllowed)
		})
		srv := NewServerBuilder(parser.OASVersion320, WithMethodNotAllowedHandler(handler))
		assert.NotNil(t, srv.config.methodNotAllowed)
	})

	t.Run("WithRecovery", func(t *testing.T) {
		t.Parallel()
		srv := NewServerBuilder(parser.OASVersion320, WithRecovery())
		assert.True(t, srv.config.enableRecovery)
	})
}

func TestDefaultValidationConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultValidationConfig()

	assert.True(t, cfg.IncludeRequestValidation)
	assert.False(t, cfg.IncludeResponseValidation)
	assert.False(t, cfg.StrictMode)
	assert.Nil(t, cfg.OnValidationError)
}

func TestResponseCapture(t *testing.T) {
	t.Parallel()

	rec := &responseCapture{header: make(http.Header)}

	// Test Header
	rec.Header().Set("X-Test", "value")
	assert.Equal(t, "value", rec.header.Get("X-Test"))

	// Test Write
	n, err := rec.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(rec.body))

	// Test WriteHeader
	rec.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, rec.status)
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

		assert.Equal(t, http.StatusCreated, resp.StatusCode())
		assert.Equal(t, "value", resp.Headers().Get("X-Test"))
		assert.Equal(t, "test", string(resp.Body().([]byte)))
	})

	t.Run("default status", func(t *testing.T) {
		t.Parallel()
		resp := &capturedResponse{}
		assert.Equal(t, http.StatusOK, resp.StatusCode())
	})

	t.Run("WriteTo", func(t *testing.T) {
		t.Parallel()
		resp := &capturedResponse{
			status:  http.StatusOK,
			headers: http.Header{"X-Custom": {"test"}},
			body:    []byte("response body"),
		}

		rec := httptest.NewRecorder()
		err := resp.WriteTo(rec)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "test", rec.Header().Get("X-Custom"))
		assert.Equal(t, "response body", rec.Body.String())
	})
}

func TestRecoveryMiddleware_ErrorPanic(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation(), WithRecovery()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/panic-error",
		WithOperationID("panicError"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	// Test panic with error type
	srv.Handle(http.MethodGet, "/panic-error", func(_ context.Context, _ *Request) Response {
		panic(fmt.Errorf("test error panic"))
	})

	result, err := srv.BuildServer()
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic-error", nil)

	// Should not panic - recovery middleware should catch it
	result.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRecoveryMiddleware_OtherTypePanic(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation(), WithRecovery()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/panic-other",
		WithOperationID("panicOther"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	// Test panic with non-string, non-error type (int)
	srv.Handle(http.MethodGet, "/panic-other", func(_ context.Context, _ *Request) Response {
		panic(42)
	})

	result, err := srv.BuildServer()
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic-other", nil)

	// Should not panic - recovery middleware should catch it
	result.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRecoveryMiddleware_WithCustomErrorHandler(t *testing.T) {
	t.Parallel()

	var capturedErr error
	customErrorHandler := func(w http.ResponseWriter, _ *http.Request, err error) {
		capturedErr = err
		w.WriteHeader(http.StatusServiceUnavailable) // Custom status
	}

	srv := NewServerBuilder(parser.OASVersion320,
		WithoutValidation(),
		WithRecovery(),
		WithErrorHandler(customErrorHandler),
	).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/panic-custom",
		WithOperationID("panicCustom"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	srv.Handle(http.MethodGet, "/panic-custom", func(_ context.Context, _ *Request) Response {
		panic("custom error test")
	})

	result, err := srv.BuildServer()
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic-custom", nil)

	result.Handler.ServeHTTP(rec, req)

	// Should use custom error handler's status code
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	// Verify error was captured
	require.NotNil(t, capturedErr)
	assert.Contains(t, capturedErr.Error(), "builder: panic recovered")
}
