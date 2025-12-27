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

func TestDispatcher_RouteToCorrectHandler(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/pets",
		WithOperationID("listPets"),
		WithResponse(http.StatusOK, []string{}),
	)

	srv.AddOperation(http.MethodPost, "/pets",
		WithOperationID("createPet"),
		WithRequestBody("application/json", struct{}{}),
		WithResponse(http.StatusCreated, struct{}{}),
	)

	var listCalled, createCalled bool

	srv.Handle(http.MethodGet, "/pets", func(_ context.Context, _ *Request) Response {
		listCalled = true
		return JSON(http.StatusOK, []string{"pet1"})
	})

	srv.Handle(http.MethodPost, "/pets", func(_ context.Context, _ *Request) Response {
		createCalled = true
		return JSON(http.StatusCreated, map[string]string{"id": "1"})
	})

	result := srv.MustBuildServer()

	// Test GET
	listCalled = false
	createCalled = false
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pets", nil)
	result.Handler.ServeHTTP(rec, req)

	if !listCalled {
		t.Error("listPets handler was not called for GET")
	}
	if createCalled {
		t.Error("createPet handler was incorrectly called for GET")
	}

	// Test POST
	listCalled = false
	createCalled = false
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/pets", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	result.Handler.ServeHTTP(rec, req)

	if listCalled {
		t.Error("listPets handler was incorrectly called for POST")
	}
	if !createCalled {
		t.Error("createPet handler was not called for POST")
	}
}

func TestDispatcher_PathParamsExtracted(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/users/{userId}/pets/{petId}",
		WithOperationID("getUserPet"),
		WithPathParam("userId", ""),
		WithPathParam("petId", ""),
		WithResponse(http.StatusOK, struct{}{}),
	)

	var capturedUserId, capturedPetId string

	srv.Handle(http.MethodGet, "/users/{userId}/pets/{petId}", func(_ context.Context, req *Request) Response {
		capturedUserId, _ = req.PathParams["userId"].(string)
		capturedPetId, _ = req.PathParams["petId"].(string)
		return JSON(http.StatusOK, nil)
	})

	result := srv.MustBuildServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/42/pets/99", nil)
	result.Handler.ServeHTTP(rec, req)

	if capturedUserId != "42" {
		t.Errorf("Expected userId '42', got '%s'", capturedUserId)
	}
	if capturedPetId != "99" {
		t.Errorf("Expected petId '99', got '%s'", capturedPetId)
	}
}

func TestDispatcher_QueryParamsExtracted(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/search",
		WithOperationID("search"),
		WithQueryParam("q", ""),
		WithQueryParam("page", int(0)),
		WithResponse(http.StatusOK, struct{}{}),
	)

	var capturedQ string
	var capturedPage any

	srv.Handle(http.MethodGet, "/search", func(_ context.Context, req *Request) Response {
		capturedQ, _ = req.QueryParams["q"].(string)
		capturedPage = req.QueryParams["page"]
		return JSON(http.StatusOK, nil)
	})

	result := srv.MustBuildServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/search?q=test&page=2", nil)
	result.Handler.ServeHTTP(rec, req)

	if capturedQ != "test" {
		t.Errorf("Expected q 'test', got '%s'", capturedQ)
	}
	if capturedPage != "2" {
		t.Errorf("Expected page '2', got '%v'", capturedPage)
	}
}

func TestDispatcher_RequestBodyParsed(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodPost, "/data",
		WithOperationID("postData"),
		WithRequestBody("application/json", struct{}{}),
		WithResponse(http.StatusOK, struct{}{}),
	)

	var capturedBody any
	var capturedRawBody []byte

	srv.Handle(http.MethodPost, "/data", func(_ context.Context, req *Request) Response {
		capturedBody = req.Body
		capturedRawBody = req.RawBody
		return JSON(http.StatusOK, nil)
	})

	result := srv.MustBuildServer()

	body := `{"name": "test", "value": 42}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/data", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	result.Handler.ServeHTTP(rec, req)

	if string(capturedRawBody) != body {
		t.Errorf("Expected raw body '%s', got '%s'", body, string(capturedRawBody))
	}

	bodyMap, ok := capturedBody.(map[string]any)
	if !ok {
		t.Fatalf("Expected body to be map[string]any, got %T", capturedBody)
	}
	if bodyMap["name"] != "test" {
		t.Errorf("Expected name 'test', got '%v'", bodyMap["name"])
	}
	if bodyMap["value"] != float64(42) {
		t.Errorf("Expected value 42, got '%v'", bodyMap["value"])
	}
}

func TestDispatcher_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/pets",
		WithOperationID("listPets"),
		WithResponse(http.StatusOK, []string{}),
	)

	srv.Handle(http.MethodGet, "/pets", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, nil)
	})

	result := srv.MustBuildServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/pets", nil)
	result.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rec.Code)
	}

	allow := rec.Header().Get("Allow")
	if allow == "" {
		t.Error("Allow header should be set")
	}
}

func TestDispatcher_NotImplemented(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/unhandled",
		WithOperationID("unhandled"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	// Note: No handler registered

	result := srv.MustBuildServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unhandled", nil)
	result.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("Expected status 501, got %d", rec.Code)
	}
}

func TestDispatcher_OperationIDInRequest(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/test",
		WithOperationID("myOperation"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	var capturedOperationID string

	srv.Handle(http.MethodGet, "/test", func(_ context.Context, req *Request) Response {
		capturedOperationID = req.OperationID
		return JSON(http.StatusOK, nil)
	})

	result := srv.MustBuildServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	result.Handler.ServeHTTP(rec, req)

	if capturedOperationID != "myOperation" {
		t.Errorf("Expected operationID 'myOperation', got '%s'", capturedOperationID)
	}
}

func TestDispatcher_MatchedPathInRequest(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/pets/{petId}",
		WithOperationID("getPet"),
		WithPathParam("petId", ""),
		WithResponse(http.StatusOK, struct{}{}),
	)

	var capturedMatchedPath string

	srv.Handle(http.MethodGet, "/pets/{petId}", func(_ context.Context, req *Request) Response {
		capturedMatchedPath = req.MatchedPath
		return JSON(http.StatusOK, nil)
	})

	result := srv.MustBuildServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pets/123", nil)
	result.Handler.ServeHTTP(rec, req)

	if capturedMatchedPath != "/pets/{petId}" {
		t.Errorf("Expected matched path '/pets/{petId}', got '%s'", capturedMatchedPath)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	t.Parallel()

	var loggedMethod, loggedPath string
	var loggedStatus int
	var loggedDuration time.Duration

	logger := func(method, path string, status int, duration time.Duration) {
		loggedMethod = method
		loggedPath = path
		loggedStatus = status
		loggedDuration = duration
	}

	mw := loggingMiddleware(logger)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(rec, req)

	if loggedMethod != http.MethodGet {
		t.Errorf("Expected method GET, got %s", loggedMethod)
	}
	if loggedPath != "/test" {
		t.Errorf("Expected path /test, got %s", loggedPath)
	}
	if loggedStatus != http.StatusOK {
		t.Errorf("Expected status 200, got %d", loggedStatus)
	}
	if loggedDuration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", loggedDuration)
	}
}

func TestStatusRecorder(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: rec, status: http.StatusOK}

	sr.WriteHeader(http.StatusCreated)

	if sr.status != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", sr.status)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("Expected underlying recorder status 201, got %d", rec.Code)
	}
}

func TestDispatcher_ErrorHandler(t *testing.T) {
	t.Parallel()

	var errorHandlerCalled bool

	srv := NewServerBuilder(parser.OASVersion320,
		WithoutValidation(),
		WithErrorHandler(func(_ http.ResponseWriter, _ *http.Request, _ error) {
			errorHandlerCalled = true
		}),
	).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/test",
		WithOperationID("test"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	srv.Handle(http.MethodGet, "/test", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, nil)
	})

	result := srv.MustBuildServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	result.Handler.ServeHTTP(rec, req)

	// Error handler should not be called for normal requests
	if errorHandlerCalled {
		t.Error("Error handler should not be called for successful requests")
	}
}

func TestDispatcher_HTTPRequestAccessible(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/test",
		WithOperationID("test"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	var capturedHTTPRequest *http.Request

	srv.Handle(http.MethodGet, "/test", func(_ context.Context, req *Request) Response {
		capturedHTTPRequest = req.HTTPRequest
		return JSON(http.StatusOK, nil)
	})

	result := srv.MustBuildServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?foo=bar", nil)
	req.Header.Set("X-Custom", "value")
	result.Handler.ServeHTTP(rec, req)

	if capturedHTTPRequest == nil {
		t.Fatal("HTTPRequest should not be nil")
	}
	if capturedHTTPRequest.URL.Path != "/test" {
		t.Errorf("Expected path /test, got %s", capturedHTTPRequest.URL.Path)
	}
	if capturedHTTPRequest.Header.Get("X-Custom") != "value" {
		t.Error("Custom header not accessible")
	}
}

func TestDispatcher_AllowedMethodsForMultipleMethods(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/resource",
		WithOperationID("getResource"),
		WithResponse(http.StatusOK, struct{}{}),
	)
	srv.AddOperation(http.MethodPost, "/resource",
		WithOperationID("createResource"),
		WithResponse(http.StatusCreated, struct{}{}),
	)
	srv.AddOperation(http.MethodPut, "/resource",
		WithOperationID("updateResource"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	srv.Handle(http.MethodGet, "/resource", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, nil)
	})
	srv.Handle(http.MethodPost, "/resource", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusCreated, nil)
	})
	srv.Handle(http.MethodPut, "/resource", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, nil)
	})

	result := srv.MustBuildServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/resource", nil)
	result.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rec.Code)
	}

	allow := rec.Header().Get("Allow")
	// Allow header should contain GET, POST, PUT (sorted)
	if !strings.Contains(allow, "GET") || !strings.Contains(allow, "POST") || !strings.Contains(allow, "PUT") {
		t.Errorf("Allow header should contain GET, POST, PUT, got '%s'", allow)
	}
}

func TestDispatcher_EmptyBody(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodPost, "/empty",
		WithOperationID("postEmpty"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	var capturedBody any
	var capturedRawBody []byte

	srv.Handle(http.MethodPost, "/empty", func(_ context.Context, req *Request) Response {
		capturedBody = req.Body
		capturedRawBody = req.RawBody
		return JSON(http.StatusOK, nil)
	})

	result := srv.MustBuildServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/empty", nil)
	result.Handler.ServeHTTP(rec, req)

	if capturedBody != nil {
		t.Error("Body should be nil for empty request")
	}
	if len(capturedRawBody) != 0 {
		t.Error("RawBody should be empty for empty request")
	}
}

func TestDispatcher_NonJSONBody(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodPost, "/text",
		WithOperationID("postText"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	var capturedBody any
	var capturedRawBody []byte

	srv.Handle(http.MethodPost, "/text", func(_ context.Context, req *Request) Response {
		capturedBody = req.Body
		capturedRawBody = req.RawBody
		return JSON(http.StatusOK, nil)
	})

	result := srv.MustBuildServer()

	body := "plain text body"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/text", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	result.Handler.ServeHTTP(rec, req)

	if string(capturedRawBody) != body {
		t.Errorf("Expected raw body '%s', got '%s'", body, string(capturedRawBody))
	}

	// Body should be nil since it's not valid JSON
	if capturedBody != nil {
		t.Errorf("Body should be nil for non-JSON content, got %v", capturedBody)
	}
}

func TestDispatcher_ContextPropagation(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/ctx",
		WithOperationID("testCtx"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	var capturedCtx context.Context

	srv.Handle(http.MethodGet, "/ctx", func(ctx context.Context, _ *Request) Response {
		capturedCtx = ctx
		return JSON(http.StatusOK, nil)
	})

	result := srv.MustBuildServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ctx", nil)
	result.Handler.ServeHTTP(rec, req)

	if capturedCtx == nil {
		t.Error("Context should not be nil")
	}
}

func TestDispatcher_ResponseJSON(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/json",
		WithOperationID("getJSON"),
		WithResponse(http.StatusOK, struct{}{}),
	)

	srv.Handle(http.MethodGet, "/json", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, map[string]string{"message": "hello"})
	})

	result := srv.MustBuildServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	result.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if body["message"] != "hello" {
		t.Errorf("Expected message 'hello', got '%s'", body["message"])
	}
}
