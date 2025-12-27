package builder

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/erraggy/oastools/parser"
)

func TestNewTestRequest(t *testing.T) {
	t.Parallel()

	req := NewTestRequest(http.MethodGet, "/test")

	if req.method != http.MethodGet {
		t.Errorf("Expected method GET, got %s", req.method)
	}
	if req.path != "/test" {
		t.Errorf("Expected path /test, got %s", req.path)
	}
	if req.headers == nil {
		t.Error("headers should be initialized")
	}
	if req.query == nil {
		t.Error("query should be initialized")
	}
}

func TestTestRequest_Header(t *testing.T) {
	t.Parallel()

	req := NewTestRequest(http.MethodGet, "/test").
		Header("Authorization", "Bearer token").
		Header("X-Custom", "value")

	if req.headers.Get("Authorization") != "Bearer token" {
		t.Error("Authorization header not set correctly")
	}
	if req.headers.Get("X-Custom") != "value" {
		t.Error("X-Custom header not set correctly")
	}
}

func TestTestRequest_Query(t *testing.T) {
	t.Parallel()

	req := NewTestRequest(http.MethodGet, "/test").
		Query("page", "1").
		Query("limit", "10")

	if req.query.Get("page") != "1" {
		t.Error("page query param not set correctly")
	}
	if req.query.Get("limit") != "10" {
		t.Error("limit query param not set correctly")
	}
}

func TestTestRequest_JSONBody(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}

	req := NewTestRequest(http.MethodPost, "/test").
		JSONBody(payload{Name: "test"})

	if req.headers.Get("Content-Type") != "application/json" {
		t.Error("Content-Type header not set for JSON body")
	}
	if req.body == nil {
		t.Error("body should be set")
	}
}

func TestTestRequest_Build(t *testing.T) {
	t.Parallel()

	req := NewTestRequest(http.MethodGet, "/test").
		Query("key", "value").
		Header("X-Test", "header").
		Build()

	if req.Method != http.MethodGet {
		t.Errorf("Expected method GET, got %s", req.Method)
	}
	if req.URL.Path != "/test" {
		t.Errorf("Expected path /test, got %s", req.URL.Path)
	}
	if req.URL.Query().Get("key") != "value" {
		t.Error("Query parameter not included in URL")
	}
	if req.Header.Get("X-Test") != "header" {
		t.Error("Header not included in request")
	}
}

func TestTestRequest_Execute(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	rec := NewTestRequest(http.MethodGet, "/test").Execute(handler)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != `{"status":"ok"}` {
		t.Errorf("Unexpected body: %s", rec.Body.String())
	}
}

func TestStubHandler(t *testing.T) {
	t.Parallel()

	response := JSON(http.StatusOK, map[string]string{"message": "hello"})
	handler := StubHandler(response)

	ctx := context.Background()
	req := &Request{}
	resp := handler(ctx, req)

	if resp.StatusCode() != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode())
	}
}

func TestStubHandlerFunc(t *testing.T) {
	t.Parallel()

	var receivedPath string
	handler := StubHandlerFunc(func(req *Request) Response {
		receivedPath = req.MatchedPath
		return JSON(http.StatusOK, nil)
	})

	ctx := context.Background()
	req := &Request{MatchedPath: "/test/path"}
	resp := handler(ctx, req)

	if resp.StatusCode() != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode())
	}
	if receivedPath != "/test/path" {
		t.Errorf("Expected matched path /test/path, got %s", receivedPath)
	}
}

func TestErrorStubHandler(t *testing.T) {
	t.Parallel()

	handler := ErrorStubHandler(http.StatusNotFound, "not found")

	ctx := context.Background()
	req := &Request{}
	resp := handler(ctx, req)

	if resp.StatusCode() != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode())
	}

	body := resp.Body().(map[string]any)
	if body["error"] != "not found" {
		t.Errorf("Expected error 'not found', got %v", body["error"])
	}
}

func TestServerTest(t *testing.T) {
	t.Parallel()

	type Pet struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/pets",
		WithOperationID("listPets"),
		WithResponse(http.StatusOK, []Pet{}),
	)

	srv.AddOperation(http.MethodPost, "/pets",
		WithOperationID("createPet"),
		WithRequestBody("application/json", Pet{}),
		WithResponse(http.StatusCreated, Pet{}),
	)

	srv.AddOperation(http.MethodPut, "/pets/{id}",
		WithOperationID("updatePet"),
		WithPathParam("id", int64(0)),
		WithRequestBody("application/json", Pet{}),
		WithResponse(http.StatusOK, Pet{}),
	)

	srv.AddOperation(http.MethodDelete, "/pets/{id}",
		WithOperationID("deletePet"),
		WithPathParam("id", int64(0)),
		WithResponse(http.StatusNoContent, nil),
	)

	pets := []Pet{{ID: 1, Name: "Fluffy"}}

	srv.Handle(http.MethodGet, "/pets", func(_ context.Context, _ *Request) Response {
		return JSON(http.StatusOK, pets)
	})

	srv.Handle(http.MethodPost, "/pets", func(_ context.Context, req *Request) Response {
		return JSON(http.StatusCreated, req.Body)
	})

	srv.Handle(http.MethodPut, "/pets/{id}", func(_ context.Context, req *Request) Response {
		return JSON(http.StatusOK, req.Body)
	})

	srv.Handle(http.MethodDelete, "/pets/{id}", func(_ context.Context, _ *Request) Response {
		return NoContent()
	})

	result := srv.MustBuildServer()
	test := NewServerTest(result)

	t.Run("GetJSON", func(t *testing.T) {
		var gotPets []Pet
		rec, err := test.GetJSON("/pets", &gotPets)
		if err != nil {
			t.Fatalf("GetJSON failed: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
		if len(gotPets) != 1 {
			t.Errorf("Expected 1 pet, got %d", len(gotPets))
		}
	})

	t.Run("PostJSON", func(t *testing.T) {
		newPet := Pet{ID: 2, Name: "Spot"}
		var created map[string]any
		rec, err := test.PostJSON("/pets", newPet, &created)
		if err != nil {
			t.Fatalf("PostJSON failed: %v", err)
		}
		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", rec.Code)
		}
	})

	t.Run("PutJSON", func(t *testing.T) {
		updatedPet := Pet{ID: 1, Name: "Fluffy Updated"}
		var updated map[string]any
		rec, err := test.PutJSON("/pets/1", updatedPet, &updated)
		if err != nil {
			t.Fatalf("PutJSON failed: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		rec := test.Delete("/pets/1")
		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", rec.Code)
		}
	})

	t.Run("Request", func(t *testing.T) {
		req := test.Request(http.MethodGet, "/pets")
		if req.method != http.MethodGet {
			t.Error("Request method not set correctly")
		}
	})

	t.Run("Execute", func(t *testing.T) {
		req := test.Request(http.MethodGet, "/pets")
		rec := test.Execute(req)
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})
}

func TestServerTest_GetJSON_NonSuccessStatus(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodGet, "/error",
		WithOperationID("getError"),
		WithResponse(http.StatusInternalServerError, map[string]string{}),
	)

	srv.Handle(http.MethodGet, "/error", func(_ context.Context, _ *Request) Response {
		return Error(http.StatusInternalServerError, "something went wrong")
	})

	result := srv.MustBuildServer()
	test := NewServerTest(result)

	var target map[string]any
	rec, err := test.GetJSON("/error", &target)

	if err != nil {
		t.Fatalf("GetJSON returned error: %v", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}

	// Target should not be populated on error status
	if len(target) > 0 {
		t.Error("Target should not be populated on error status")
	}
}

func TestServerTest_PostJSON_DecodeError(t *testing.T) {
	t.Parallel()

	srv := NewServerBuilder(parser.OASVersion320, WithoutValidation()).
		SetTitle("Test API").
		SetVersion("1.0.0")

	srv.AddOperation(http.MethodPost, "/text",
		WithOperationID("postText"),
		WithResponse(http.StatusOK, nil),
	)

	srv.Handle(http.MethodPost, "/text", func(_ context.Context, _ *Request) Response {
		return NewResponse(http.StatusOK).Text("plain text response")
	})

	result := srv.MustBuildServer()
	test := NewServerTest(result)

	var target map[string]any
	_, err := test.PostJSON("/text", map[string]string{}, &target)

	if err == nil {
		t.Error("Expected decode error for non-JSON response")
	}
}

func TestTestRequest_MultipleQueryValues(t *testing.T) {
	t.Parallel()

	req := NewTestRequest(http.MethodGet, "/test").
		Query("tag", "a").
		Query("tag", "b").
		Build()

	tags := req.URL.Query()["tag"]
	if len(tags) != 2 {
		t.Errorf("Expected 2 tag values, got %d", len(tags))
	}
}

func TestTestRequest_Body(t *testing.T) {
	t.Parallel()

	body := []byte("raw body content")
	req := NewTestRequest(http.MethodPost, "/test").
		Body("text/plain", http.NoBody)

	req.body = nil // Reset for test

	type customBody struct {
		Data string `json:"data"`
	}

	req = NewTestRequest(http.MethodPost, "/test").
		JSONBody(customBody{Data: "test"})

	httpReq := req.Build()
	if httpReq.Header.Get("Content-Type") != "application/json" {
		t.Error("Content-Type not set correctly")
	}

	var decoded customBody
	if err := json.NewDecoder(httpReq.Body).Decode(&decoded); err != nil {
		t.Fatalf("Failed to decode body: %v", err)
	}
	if decoded.Data != "test" {
		t.Errorf("Expected data 'test', got '%s'", decoded.Data)
	}
	_ = body // Prevent unused variable warning
}
