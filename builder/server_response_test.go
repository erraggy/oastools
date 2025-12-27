package builder

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	t.Parallel()

	type testData struct {
		Name string `json:"name"`
	}

	data := testData{Name: "test"}
	resp := JSON(http.StatusOK, data)

	if resp.StatusCode() != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode())
	}

	if resp.Headers() == nil {
		t.Error("Headers should not be nil")
	}

	body := resp.Body()
	if body == nil {
		t.Error("Body should not be nil")
	}

	rec := httptest.NewRecorder()
	if err := resp.WriteTo(rec); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}

	var got testData
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if got.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", got.Name)
	}
}

func TestJSON_NilBody(t *testing.T) {
	t.Parallel()

	resp := JSON(http.StatusNoContent, nil)

	rec := httptest.NewRecorder()
	if err := resp.WriteTo(rec); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rec.Code)
	}

	if rec.Body.Len() > 0 {
		t.Errorf("Expected empty body, got %s", rec.Body.String())
	}
}

func TestNoContent(t *testing.T) {
	t.Parallel()

	resp := NoContent()

	if resp.StatusCode() != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode())
	}

	if resp.Headers() != nil {
		t.Error("Headers should be nil for NoContent")
	}

	if resp.Body() != nil {
		t.Error("Body should be nil for NoContent")
	}

	rec := httptest.NewRecorder()
	if err := resp.WriteTo(rec); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rec.Code)
	}
}

func TestError(t *testing.T) {
	t.Parallel()

	resp := Error(http.StatusNotFound, "resource not found")

	if resp.StatusCode() != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode())
	}

	body := resp.Body().(map[string]any)
	if body["error"] != "resource not found" {
		t.Errorf("Expected error message 'resource not found', got %v", body["error"])
	}

	rec := httptest.NewRecorder()
	if err := resp.WriteTo(rec); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}

	var got map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if got["error"] != "resource not found" {
		t.Errorf("Expected error 'resource not found', got %v", got["error"])
	}
}

func TestErrorWithDetails(t *testing.T) {
	t.Parallel()

	details := map[string]string{
		"field": "name",
		"issue": "required",
	}

	resp := ErrorWithDetails(http.StatusBadRequest, "validation failed", details)

	if resp.StatusCode() != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode())
	}

	body := resp.Body().(map[string]any)
	if body["error"] != "validation failed" {
		t.Errorf("Expected error message 'validation failed', got %v", body["error"])
	}
	if body["details"] == nil {
		t.Error("Expected details to be present")
	}

	rec := httptest.NewRecorder()
	if err := resp.WriteTo(rec); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	var got map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	gotDetails := got["details"].(map[string]any)
	if gotDetails["field"] != "name" {
		t.Errorf("Expected details.field 'name', got %v", gotDetails["field"])
	}
}

func TestRedirect(t *testing.T) {
	t.Parallel()

	resp := Redirect(http.StatusMovedPermanently, "/new-location")

	if resp.StatusCode() != http.StatusMovedPermanently {
		t.Errorf("Expected status 301, got %d", resp.StatusCode())
	}

	headers := resp.Headers()
	if headers.Get("Location") != "/new-location" {
		t.Errorf("Expected Location header '/new-location', got '%s'", headers.Get("Location"))
	}

	if resp.Body() != nil {
		t.Error("Body should be nil for Redirect")
	}

	rec := httptest.NewRecorder()
	if err := resp.WriteTo(rec); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("Expected status 301, got %d", rec.Code)
	}

	if rec.Header().Get("Location") != "/new-location" {
		t.Errorf("Expected Location header '/new-location', got '%s'", rec.Header().Get("Location"))
	}
}

func TestStream(t *testing.T) {
	t.Parallel()

	data := []byte("streaming data content")
	reader := bytes.NewReader(data)

	resp := Stream(http.StatusOK, "application/octet-stream", reader)

	if resp.StatusCode() != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode())
	}

	body := resp.Body()
	if _, ok := body.(io.Reader); !ok {
		t.Error("Body should be an io.Reader")
	}

	rec := httptest.NewRecorder()
	if err := resp.WriteTo(rec); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if rec.Header().Get("Content-Type") != "application/octet-stream" {
		t.Errorf("Expected Content-Type application/octet-stream, got %s", rec.Header().Get("Content-Type"))
	}

	if rec.Body.String() != string(data) {
		t.Errorf("Expected body '%s', got '%s'", string(data), rec.Body.String())
	}
}

func TestResponseBuilder_JSON(t *testing.T) {
	t.Parallel()

	type testData struct {
		Value int `json:"value"`
	}

	resp := NewResponse(http.StatusOK).
		Header("X-Custom", "test").
		JSON(testData{Value: 42})

	if resp.StatusCode() != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode())
	}

	if resp.Headers().Get("X-Custom") != "test" {
		t.Errorf("Expected X-Custom header 'test', got '%s'", resp.Headers().Get("X-Custom"))
	}

	rec := httptest.NewRecorder()
	if err := resp.WriteTo(rec); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}

	var got testData
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if got.Value != 42 {
		t.Errorf("Expected value 42, got %d", got.Value)
	}
}

func TestResponseBuilder_XML(t *testing.T) {
	t.Parallel()

	type testData struct {
		XMLName xml.Name `xml:"root"`
		Value   string   `xml:"value"`
	}

	resp := NewResponse(http.StatusOK).XML(testData{Value: "test"})

	rec := httptest.NewRecorder()
	if err := resp.WriteTo(rec); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if rec.Header().Get("Content-Type") != "application/xml" {
		t.Errorf("Expected Content-Type application/xml, got %s", rec.Header().Get("Content-Type"))
	}

	var got testData
	if err := xml.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if got.Value != "test" {
		t.Errorf("Expected value 'test', got '%s'", got.Value)
	}
}

func TestResponseBuilder_Text(t *testing.T) {
	t.Parallel()

	resp := NewResponse(http.StatusOK).Text("plain text response")

	rec := httptest.NewRecorder()
	if err := resp.WriteTo(rec); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if rec.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", rec.Header().Get("Content-Type"))
	}

	if rec.Body.String() != "plain text response" {
		t.Errorf("Expected body 'plain text response', got '%s'", rec.Body.String())
	}
}

func TestResponseBuilder_Binary(t *testing.T) {
	t.Parallel()

	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header bytes
	resp := NewResponse(http.StatusOK).Binary("image/png", data)

	rec := httptest.NewRecorder()
	if err := resp.WriteTo(rec); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if rec.Header().Get("Content-Type") != "image/png" {
		t.Errorf("Expected Content-Type image/png, got %s", rec.Header().Get("Content-Type"))
	}

	if !bytes.Equal(rec.Body.Bytes(), data) {
		t.Errorf("Expected body %v, got %v", data, rec.Body.Bytes())
	}
}

func TestResponseBuilder_NilBody(t *testing.T) {
	t.Parallel()

	builder := NewResponse(http.StatusOK)
	// Don't set a body or encoder

	rec := httptest.NewRecorder()
	if err := builder.WriteTo(rec); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestResponseBuilder_MultipleHeaders(t *testing.T) {
	t.Parallel()

	resp := NewResponse(http.StatusOK).
		Header("X-First", "one").
		Header("X-Second", "two").
		Header("X-First", "another"). // Add another value to same header
		JSON(nil)

	if len(resp.Headers()["X-First"]) != 2 {
		t.Errorf("Expected 2 values for X-First header, got %d", len(resp.Headers()["X-First"]))
	}

	if resp.Headers().Get("X-Second") != "two" {
		t.Errorf("Expected X-Second 'two', got '%s'", resp.Headers().Get("X-Second"))
	}
}
