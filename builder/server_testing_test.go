package builder

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestRequest(t *testing.T) {
	t.Parallel()

	req := NewTestRequest(http.MethodGet, "/test")

	assert.Equal(t, http.MethodGet, req.method)
	assert.Equal(t, "/test", req.path)
	assert.NotNil(t, req.headers)
	assert.NotNil(t, req.query)
}

func TestTestRequest_Header(t *testing.T) {
	t.Parallel()

	req := NewTestRequest(http.MethodGet, "/test").
		Header("Authorization", "Bearer token").
		Header("X-Custom", "value")

	assert.Equal(t, "Bearer token", req.headers.Get("Authorization"))
	assert.Equal(t, "value", req.headers.Get("X-Custom"))
}

func TestTestRequest_Query(t *testing.T) {
	t.Parallel()

	req := NewTestRequest(http.MethodGet, "/test").
		Query("page", "1").
		Query("limit", "10")

	assert.Equal(t, "1", req.query.Get("page"))
	assert.Equal(t, "10", req.query.Get("limit"))
}

func TestTestRequest_JSONBody(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}

	req := NewTestRequest(http.MethodPost, "/test").
		JSONBody(payload{Name: "test"})

	assert.Equal(t, "application/json", req.headers.Get("Content-Type"))
	assert.NotNil(t, req.body)
}

func TestTestRequest_Build(t *testing.T) {
	t.Parallel()

	req := NewTestRequest(http.MethodGet, "/test").
		Query("key", "value").
		Header("X-Test", "header").
		Build()

	assert.Equal(t, http.MethodGet, req.Method)
	assert.Equal(t, "/test", req.URL.Path)
	assert.Equal(t, "value", req.URL.Query().Get("key"))
	assert.Equal(t, "header", req.Header.Get("X-Test"))
}

func TestTestRequest_Execute(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	rec := NewTestRequest(http.MethodGet, "/test").Execute(handler)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, `{"status":"ok"}`, rec.Body.String())
}

func TestStubHandler(t *testing.T) {
	t.Parallel()

	response := JSON(http.StatusOK, map[string]string{"message": "hello"})
	handler := StubHandler(response)

	ctx := context.Background()
	req := &Request{}
	resp := handler(ctx, req)

	assert.Equal(t, http.StatusOK, resp.StatusCode())
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

	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, "/test/path", receivedPath)
}

func TestErrorStubHandler(t *testing.T) {
	t.Parallel()

	handler := ErrorStubHandler(http.StatusNotFound, "not found")

	ctx := context.Background()
	req := &Request{}
	resp := handler(ctx, req)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode())

	body := resp.Body().(map[string]any)
	assert.Equal(t, "not found", body["error"])
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
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Len(t, gotPets, 1)
	})

	t.Run("PostJSON", func(t *testing.T) {
		newPet := Pet{ID: 2, Name: "Spot"}
		var created map[string]any
		rec, err := test.PostJSON("/pets", newPet, &created)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("PutJSON", func(t *testing.T) {
		updatedPet := Pet{ID: 1, Name: "Fluffy Updated"}
		var updated map[string]any
		rec, err := test.PutJSON("/pets/1", updatedPet, &updated)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Delete", func(t *testing.T) {
		rec := test.Delete("/pets/1")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("Request", func(t *testing.T) {
		req := test.Request(http.MethodGet, "/pets")
		assert.Equal(t, http.MethodGet, req.method)
	})

	t.Run("Execute", func(t *testing.T) {
		req := test.Request(http.MethodGet, "/pets")
		rec := test.Execute(req)
		assert.Equal(t, http.StatusOK, rec.Code)
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

	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// Target should not be populated on error status
	assert.Empty(t, target)
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

	assert.Error(t, err)
}

func TestTestRequest_MultipleQueryValues(t *testing.T) {
	t.Parallel()

	req := NewTestRequest(http.MethodGet, "/test").
		Query("tag", "a").
		Query("tag", "b").
		Build()

	tags := req.URL.Query()["tag"]
	assert.Len(t, tags, 2)
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
	assert.Equal(t, "application/json", httpReq.Header.Get("Content-Type"))

	var decoded customBody
	err := json.NewDecoder(httpReq.Body).Decode(&decoded)
	require.NoError(t, err)
	assert.Equal(t, "test", decoded.Data)
	_ = body // Prevent unused variable warning
}
