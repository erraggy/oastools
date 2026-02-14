package builder

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON(t *testing.T) {
	t.Parallel()

	type testData struct {
		Name string `json:"name"`
	}

	data := testData{Name: "test"}
	resp := JSON(http.StatusOK, data)

	assert.Equal(t, http.StatusOK, resp.StatusCode())

	assert.NotNil(t, resp.Headers())

	body := resp.Body()
	assert.NotNil(t, body)

	rec := httptest.NewRecorder()
	err := resp.WriteTo(rec)
	require.NoError(t, err)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var got testData
	err = json.NewDecoder(rec.Body).Decode(&got)
	require.NoError(t, err)

	assert.Equal(t, "test", got.Name)
}

func TestJSON_NilBody(t *testing.T) {
	t.Parallel()

	resp := JSON(http.StatusNoContent, nil)

	rec := httptest.NewRecorder()
	err := resp.WriteTo(rec)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNoContent, rec.Code)

	assert.Equal(t, 0, rec.Body.Len())
}

func TestNoContent(t *testing.T) {
	t.Parallel()

	resp := NoContent()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode())

	assert.Nil(t, resp.Headers())

	assert.Nil(t, resp.Body())

	rec := httptest.NewRecorder()
	err := resp.WriteTo(rec)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestError(t *testing.T) {
	t.Parallel()

	resp := Error(http.StatusNotFound, "resource not found")

	assert.Equal(t, http.StatusNotFound, resp.StatusCode())

	body := resp.Body().(map[string]any)
	assert.Equal(t, "resource not found", body["error"])

	rec := httptest.NewRecorder()
	err := resp.WriteTo(rec)
	require.NoError(t, err)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var got map[string]any
	err = json.NewDecoder(rec.Body).Decode(&got)
	require.NoError(t, err)

	assert.Equal(t, "resource not found", got["error"])
}

func TestErrorWithDetails(t *testing.T) {
	t.Parallel()

	details := map[string]string{
		"field": "name",
		"issue": "required",
	}

	resp := ErrorWithDetails(http.StatusBadRequest, "validation failed", details)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode())

	body := resp.Body().(map[string]any)
	assert.Equal(t, "validation failed", body["error"])
	assert.NotNil(t, body["details"])

	rec := httptest.NewRecorder()
	err := resp.WriteTo(rec)
	require.NoError(t, err)

	var got map[string]any
	err = json.NewDecoder(rec.Body).Decode(&got)
	require.NoError(t, err)

	gotDetails := got["details"].(map[string]any)
	assert.Equal(t, "name", gotDetails["field"])
}

func TestRedirect(t *testing.T) {
	t.Parallel()

	resp := Redirect(http.StatusMovedPermanently, "/new-location")

	assert.Equal(t, http.StatusMovedPermanently, resp.StatusCode())

	headers := resp.Headers()
	assert.Equal(t, "/new-location", headers.Get("Location"))

	assert.Nil(t, resp.Body())

	rec := httptest.NewRecorder()
	err := resp.WriteTo(rec)
	require.NoError(t, err)

	assert.Equal(t, http.StatusMovedPermanently, rec.Code)

	assert.Equal(t, "/new-location", rec.Header().Get("Location"))
}

func TestStream(t *testing.T) {
	t.Parallel()

	data := []byte("streaming data content")
	reader := bytes.NewReader(data)

	resp := Stream(http.StatusOK, "application/octet-stream", reader)

	assert.Equal(t, http.StatusOK, resp.StatusCode())

	body := resp.Body()
	_, ok := body.(io.Reader)
	assert.True(t, ok, "Body should be an io.Reader")

	rec := httptest.NewRecorder()
	err := resp.WriteTo(rec)
	require.NoError(t, err)

	assert.Equal(t, "application/octet-stream", rec.Header().Get("Content-Type"))

	assert.Equal(t, string(data), rec.Body.String())
}

func TestResponseBuilder_JSON(t *testing.T) {
	t.Parallel()

	type testData struct {
		Value int `json:"value"`
	}

	resp := NewResponse(http.StatusOK).
		Header("X-Custom", "test").
		JSON(testData{Value: 42})

	assert.Equal(t, http.StatusOK, resp.StatusCode())

	assert.Equal(t, "test", resp.Headers().Get("X-Custom"))

	rec := httptest.NewRecorder()
	err := resp.WriteTo(rec)
	require.NoError(t, err)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var got testData
	err = json.NewDecoder(rec.Body).Decode(&got)
	require.NoError(t, err)

	assert.Equal(t, 42, got.Value)
}

func TestResponseBuilder_XML(t *testing.T) {
	t.Parallel()

	type testData struct {
		XMLName xml.Name `xml:"root"`
		Value   string   `xml:"value"`
	}

	resp := NewResponse(http.StatusOK).XML(testData{Value: "test"})

	rec := httptest.NewRecorder()
	err := resp.WriteTo(rec)
	require.NoError(t, err)

	assert.Equal(t, "application/xml", rec.Header().Get("Content-Type"))

	var got testData
	err = xml.NewDecoder(rec.Body).Decode(&got)
	require.NoError(t, err)

	assert.Equal(t, "test", got.Value)
}

func TestResponseBuilder_Text(t *testing.T) {
	t.Parallel()

	resp := NewResponse(http.StatusOK).Text("plain text response")

	rec := httptest.NewRecorder()
	err := resp.WriteTo(rec)
	require.NoError(t, err)

	assert.Equal(t, "text/plain", rec.Header().Get("Content-Type"))

	assert.Equal(t, "plain text response", rec.Body.String())
}

func TestResponseBuilder_Binary(t *testing.T) {
	t.Parallel()

	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header bytes
	resp := NewResponse(http.StatusOK).Binary("image/png", data)

	rec := httptest.NewRecorder()
	err := resp.WriteTo(rec)
	require.NoError(t, err)

	assert.Equal(t, "image/png", rec.Header().Get("Content-Type"))

	assert.True(t, bytes.Equal(rec.Body.Bytes(), data))
}

func TestResponseBuilder_NilBody(t *testing.T) {
	t.Parallel()

	builder := NewResponse(http.StatusOK)
	// Don't set a body or encoder

	rec := httptest.NewRecorder()
	err := builder.WriteTo(rec)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestResponseBuilder_MultipleHeaders(t *testing.T) {
	t.Parallel()

	resp := NewResponse(http.StatusOK).
		Header("X-First", "one").
		Header("X-Second", "two").
		Header("X-First", "another"). // Add another value to same header
		JSON(nil)

	assert.Len(t, resp.Headers()["X-First"], 2)

	assert.Equal(t, "two", resp.Headers().Get("X-Second"))
}
