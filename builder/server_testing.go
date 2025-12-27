package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
)

// TestRequest builds requests for testing.
type TestRequest struct {
	method  string
	path    string
	headers http.Header
	body    io.Reader
	query   url.Values
}

// NewTestRequest creates a new test request builder.
//
// Example:
//
//	req := builder.NewTestRequest(http.MethodGet, "/pets").
//		Query("limit", "10").
//		Header("Authorization", "Bearer token")
func NewTestRequest(method, path string) *TestRequest {
	return &TestRequest{
		method:  method,
		path:    path,
		headers: make(http.Header),
		query:   make(url.Values),
	}
}

// Header adds a header.
func (r *TestRequest) Header(key, value string) *TestRequest {
	r.headers.Add(key, value)
	return r
}

// Query adds a query parameter.
func (r *TestRequest) Query(key, value string) *TestRequest {
	r.query.Add(key, value)
	return r
}

// JSONBody sets a JSON request body.
// Panics if the body cannot be marshaled to JSON, indicating a test setup error.
func (r *TestRequest) JSONBody(body any) *TestRequest {
	data, err := json.Marshal(body)
	if err != nil {
		panic(fmt.Sprintf("builder: JSONBody failed to marshal body: %v", err))
	}
	r.body = bytes.NewReader(data)
	r.headers.Set("Content-Type", "application/json")
	return r
}

// Body sets a raw request body.
func (r *TestRequest) Body(contentType string, body io.Reader) *TestRequest {
	r.body = body
	r.headers.Set("Content-Type", contentType)
	return r
}

// Build creates the http.Request.
func (r *TestRequest) Build() *http.Request {
	path := r.path
	if len(r.query) > 0 {
		path += "?" + r.query.Encode()
	}

	req := httptest.NewRequest(r.method, path, r.body)
	for k, v := range r.headers {
		req.Header[k] = v
	}
	return req
}

// Execute runs the request against a handler and returns the response.
func (r *TestRequest) Execute(handler http.Handler) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, r.Build())
	return rec
}

// StubHandler creates a handler that returns a fixed response.
//
// Example:
//
//	srv.Handle("listPets", builder.StubHandler(builder.JSON(http.StatusOK, pets)))
func StubHandler(response Response) HandlerFunc {
	return func(_ context.Context, _ *Request) Response {
		return response
	}
}

// StubHandlerFunc creates a handler that calls a function.
// Useful for asserting request contents in tests.
//
// Example:
//
//	srv.Handle("getPet", builder.StubHandlerFunc(func(req *builder.Request) builder.Response {
//		petID := req.PathParams["petId"]
//		// assertions on petID
//		return builder.JSON(http.StatusOK, pet)
//	}))
func StubHandlerFunc(fn func(req *Request) Response) HandlerFunc {
	return func(_ context.Context, req *Request) Response {
		return fn(req)
	}
}

// ErrorStubHandler creates a handler that returns an error response.
//
// Example:
//
//	srv.Handle("deletePet", builder.ErrorStubHandler(http.StatusNotFound, "pet not found"))
func ErrorStubHandler(status int, message string) HandlerFunc {
	return StubHandler(Error(status, message))
}

// ServerTest provides testing utilities for a built server.
type ServerTest struct {
	Result *ServerResult
}

// NewServerTest creates a ServerTest from a ServerResult.
// Panics if result is nil or result.Handler is nil, indicating a test setup error.
//
// Example:
//
//	result := srv.MustBuildServer()
//	test := builder.NewServerTest(result)
//	rec := test.Execute(builder.NewTestRequest(http.MethodGet, "/pets"))
func NewServerTest(result *ServerResult) *ServerTest {
	if result == nil {
		panic("builder: NewServerTest called with nil result")
	}
	if result.Handler == nil {
		panic("builder: NewServerTest called with nil Handler in result")
	}
	return &ServerTest{Result: result}
}

// Request creates a test request builder.
func (t *ServerTest) Request(method, path string) *TestRequest {
	return NewTestRequest(method, path)
}

// Execute runs a request and returns the recorder.
func (t *ServerTest) Execute(req *TestRequest) *httptest.ResponseRecorder {
	return req.Execute(t.Result.Handler)
}

// GetJSON performs a GET and unmarshals the JSON response.
//
// Example:
//
//	var pets []Pet
//	rec, err := test.GetJSON("/pets", &pets)
func (t *ServerTest) GetJSON(path string, target any) (*httptest.ResponseRecorder, error) {
	rec := t.Execute(NewTestRequest(http.MethodGet, path))
	if target != nil && rec.Code >= 200 && rec.Code < 300 {
		if err := json.NewDecoder(rec.Body).Decode(target); err != nil {
			return rec, err
		}
	}
	return rec, nil
}

// PostJSON performs a POST with a JSON body and unmarshals the response.
//
// Example:
//
//	var created Pet
//	rec, err := test.PostJSON("/pets", newPet, &created)
func (t *ServerTest) PostJSON(path string, body any, target any) (*httptest.ResponseRecorder, error) {
	rec := t.Execute(NewTestRequest(http.MethodPost, path).JSONBody(body))
	if target != nil && rec.Code >= 200 && rec.Code < 300 {
		if err := json.NewDecoder(rec.Body).Decode(target); err != nil {
			return rec, err
		}
	}
	return rec, nil
}

// PutJSON performs a PUT with a JSON body and unmarshals the response.
//
// Example:
//
//	var updated Pet
//	rec, err := test.PutJSON("/pets/123", updatedPet, &updated)
func (t *ServerTest) PutJSON(path string, body any, target any) (*httptest.ResponseRecorder, error) {
	rec := t.Execute(NewTestRequest(http.MethodPut, path).JSONBody(body))
	if target != nil && rec.Code >= 200 && rec.Code < 300 {
		if err := json.NewDecoder(rec.Body).Decode(target); err != nil {
			return rec, err
		}
	}
	return rec, nil
}

// Delete performs a DELETE request.
//
// Example:
//
//	rec := test.Delete("/pets/123")
func (t *ServerTest) Delete(path string) *httptest.ResponseRecorder {
	return t.Execute(NewTestRequest(http.MethodDelete, path))
}
