package builder

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
)

// Content type constants.
const (
	contentTypeJSON = "application/json"
	contentTypeXML  = "application/xml"
	contentTypeText = "text/plain"
)

// jsonResponse implements Response for JSON bodies.
type jsonResponse struct {
	status  int
	headers http.Header
	body    any
}

// JSON creates a JSON response with the given status and body.
//
// Example:
//
//	return builder.JSON(http.StatusOK, pets)
func JSON(status int, body any) Response {
	return &jsonResponse{
		status:  status,
		headers: make(http.Header),
		body:    body,
	}
}

func (r *jsonResponse) StatusCode() int      { return r.status }
func (r *jsonResponse) Headers() http.Header { return r.headers }
func (r *jsonResponse) Body() any            { return r.body }

func (r *jsonResponse) WriteTo(w http.ResponseWriter) error {
	for k, v := range r.headers {
		w.Header()[k] = v
	}
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(r.status)

	if r.body == nil {
		return nil
	}
	return json.NewEncoder(w).Encode(r.body)
}

// noContentResponse implements Response for 204 No Content.
type noContentResponse struct{}

// NoContent creates a 204 No Content response.
//
// Example:
//
//	return builder.NoContent()
func NoContent() Response {
	return &noContentResponse{}
}

func (r *noContentResponse) StatusCode() int      { return http.StatusNoContent }
func (r *noContentResponse) Headers() http.Header { return nil }
func (r *noContentResponse) Body() any            { return nil }

func (r *noContentResponse) WriteTo(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// errorResponse implements Response for error messages.
type errorResponse struct {
	status  int
	message string
	details any
}

// Error creates an error response with status and message.
//
// Example:
//
//	return builder.Error(http.StatusNotFound, "pet not found")
func Error(status int, message string) Response {
	return &errorResponse{status: status, message: message}
}

// ErrorWithDetails creates an error response with additional details.
//
// Example:
//
//	return builder.ErrorWithDetails(http.StatusBadRequest, "validation failed", errors)
func ErrorWithDetails(status int, message string, details any) Response {
	return &errorResponse{status: status, message: message, details: details}
}

func (r *errorResponse) StatusCode() int      { return r.status }
func (r *errorResponse) Headers() http.Header { return nil }
func (r *errorResponse) Body() any {
	body := map[string]any{
		"error": r.message,
	}
	if r.details != nil {
		body["details"] = r.details
	}
	return body
}

func (r *errorResponse) WriteTo(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(r.status)
	return json.NewEncoder(w).Encode(r.Body())
}

// redirectResponse implements Response for redirects.
type redirectResponse struct {
	status   int
	location string
}

// Redirect creates a redirect response.
//
// Example:
//
//	return builder.Redirect(http.StatusMovedPermanently, "/new-location")
func Redirect(status int, location string) Response {
	return &redirectResponse{status: status, location: location}
}

func (r *redirectResponse) StatusCode() int { return r.status }
func (r *redirectResponse) Headers() http.Header {
	return http.Header{"Location": {r.location}}
}
func (r *redirectResponse) Body() any { return nil }

func (r *redirectResponse) WriteTo(w http.ResponseWriter) error {
	w.Header().Set("Location", r.location)
	w.WriteHeader(r.status)
	return nil
}

// streamResponse implements Response for streaming bodies.
type streamResponse struct {
	status      int
	contentType string
	reader      io.Reader
}

// Stream creates a streaming response.
//
// Example:
//
//	return builder.Stream(http.StatusOK, "application/octet-stream", file)
func Stream(status int, contentType string, reader io.Reader) Response {
	return &streamResponse{status: status, contentType: contentType, reader: reader}
}

func (r *streamResponse) StatusCode() int      { return r.status }
func (r *streamResponse) Headers() http.Header { return nil }
func (r *streamResponse) Body() any            { return r.reader }

func (r *streamResponse) WriteTo(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", r.contentType)
	w.WriteHeader(r.status)
	_, err := io.Copy(w, r.reader)
	return err
}

// ResponseBuilder provides fluent response construction.
type ResponseBuilder struct {
	status      int
	headers     http.Header
	body        any
	contentType string
	encoder     func(w io.Writer, v any) error
}

// NewResponse creates a new ResponseBuilder.
//
// Example:
//
//	return builder.NewResponse(http.StatusOK).
//		Header("X-Request-Id", requestID).
//		JSON(pets)
func NewResponse(status int) *ResponseBuilder {
	return &ResponseBuilder{
		status:  status,
		headers: make(http.Header),
	}
}

// Header adds a header to the response.
func (b *ResponseBuilder) Header(key, value string) *ResponseBuilder {
	b.headers.Add(key, value)
	return b
}

// JSON sets a JSON body.
func (b *ResponseBuilder) JSON(body any) Response {
	b.body = body
	b.contentType = contentTypeJSON
	b.encoder = func(w io.Writer, v any) error {
		return json.NewEncoder(w).Encode(v)
	}
	return b
}

// XML sets an XML body.
func (b *ResponseBuilder) XML(body any) Response {
	b.body = body
	b.contentType = contentTypeXML
	b.encoder = func(w io.Writer, v any) error {
		return xml.NewEncoder(w).Encode(v)
	}
	return b
}

// Text sets a plain text body.
func (b *ResponseBuilder) Text(body string) Response {
	b.body = body
	b.contentType = contentTypeText
	b.encoder = func(w io.Writer, v any) error {
		_, err := w.Write([]byte(v.(string)))
		return err
	}
	return b
}

// Binary sets a binary body.
func (b *ResponseBuilder) Binary(contentType string, data []byte) Response {
	b.body = data
	b.contentType = contentType
	b.encoder = func(w io.Writer, v any) error {
		_, err := w.Write(v.([]byte))
		return err
	}
	return b
}

func (b *ResponseBuilder) StatusCode() int      { return b.status }
func (b *ResponseBuilder) Headers() http.Header { return b.headers }
func (b *ResponseBuilder) Body() any            { return b.body }

func (b *ResponseBuilder) WriteTo(w http.ResponseWriter) error {
	for k, v := range b.headers {
		w.Header()[k] = v
	}
	if b.contentType != "" {
		w.Header().Set("Content-Type", b.contentType)
	}
	w.WriteHeader(b.status)
	if b.body == nil || b.encoder == nil {
		return nil
	}
	return b.encoder(w, b.body)
}
