package builder

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"mime"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/erraggy/oastools/httpvalidator"
)

// dispatcher handles routing validated requests to handlers.
type dispatcher struct {
	routes       map[string]map[string]operationRoute // path -> method -> route
	errorHandler ErrorHandler
}

// buildDispatcher creates the dispatcher that routes requests to handlers.
// The validator parameter is reserved for future response validation integration.
func (s *ServerBuilder) buildDispatcher(routes []operationRoute, _ *httpvalidator.Validator) http.Handler {
	d := &dispatcher{
		routes:       make(map[string]map[string]operationRoute),
		errorHandler: s.errorHandler,
	}

	// Index routes by path and method
	for _, route := range routes {
		if d.routes[route.Path] == nil {
			d.routes[route.Path] = make(map[string]operationRoute)
		}
		d.routes[route.Path][route.Method] = route
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get matched path from context
		matchedPath, _ := r.Context().Value(matchedPathKey{}).(string)
		if matchedPath == "" {
			http.NotFound(w, r)
			return
		}

		// Find route
		methods, ok := d.routes[matchedPath]
		if !ok {
			http.NotFound(w, r)
			return
		}

		route, ok := methods[r.Method]
		if !ok {
			w.Header().Set("Allow", strings.Join(d.allowedMethods(methods), ", "))
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if handler is registered
		if route.Handler == nil {
			operationInfo := route.Path
			if route.OperationID != "" {
				operationInfo = fmt.Sprintf("%s (%s)", route.Path, route.OperationID)
			}
			http.Error(w, fmt.Sprintf("Operation %q not implemented", operationInfo), http.StatusNotImplemented)
			return
		}

		// Build Request struct
		req := d.buildRequest(r, route)

		// Call handler
		ctx := r.Context()
		resp := route.Handler(ctx, req)

		// Write response
		if err := resp.WriteTo(w); err != nil {
			if d.errorHandler != nil {
				d.errorHandler(w, r, fmt.Errorf("builder: failed to write response: %w", err))
			}
			// When error handler is nil or doesn't write a response,
			// there's nothing more we can do - the response may be partially written
		}
	})
}

// buildRequest creates a Request from an http.Request and route.
func (d *dispatcher) buildRequest(r *http.Request, route operationRoute) *Request {
	req := &Request{
		HTTPRequest:  r,
		OperationID:  route.OperationID,
		MatchedPath:  route.Path,
		PathParams:   make(map[string]any),
		QueryParams:  make(map[string]any),
		HeaderParams: make(map[string]any),
		CookieParams: make(map[string]any),
	}

	// Get validation result from context (if validation is enabled)
	if result := validationResultFromContext(r.Context()); result != nil {
		// Use validated/deserialized params from httpvalidator
		maps.Copy(req.PathParams, result.PathParams)
		maps.Copy(req.QueryParams, result.QueryParams)
		maps.Copy(req.HeaderParams, result.HeaderParams)
		maps.Copy(req.CookieParams, result.CookieParams)
	} else {
		// Fallback: extract raw path params from context
		if params, ok := r.Context().Value(pathParamsKey{}).(map[string]string); ok {
			for k, v := range params {
				req.PathParams[k] = v
			}
		}

		// Extract raw query params
		for k, v := range r.URL.Query() {
			if len(v) == 1 {
				req.QueryParams[k] = v[0]
			} else {
				req.QueryParams[k] = v
			}
		}
	}

	// Read and unmarshal body if present
	// Note: We check for ContentLength != 0 to handle chunked encoding (ContentLength == -1)
	// Skip reading for multipart requests to allow handlers to use FormFile()
	// Use mime.ParseMediaType for case-insensitive handling per RFC 1521
	contentType := r.Header.Get("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(contentType)
	isMultipart := strings.HasPrefix(mediaType, "multipart/")
	if r.Body != nil && r.ContentLength != 0 && !isMultipart {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			// Store the error for handlers that want to check it
			// The handler can inspect RawBody == nil to detect read failure
			req.RawBody = nil
			req.Body = nil
		} else {
			req.RawBody = body
			// Attempt JSON unmarshal if content type suggests JSON
			if len(body) > 0 && (mediaType == "" || strings.Contains(mediaType, "json")) {
				var parsed any
				if json.Unmarshal(body, &parsed) == nil {
					req.Body = parsed
				}
				// If unmarshal fails, Body remains nil but RawBody has the bytes
			}
		}
	}

	return req
}

// allowedMethods returns a sorted list of allowed methods for a path.
func (d *dispatcher) allowedMethods(methods map[string]operationRoute) []string {
	result := make([]string, 0, len(methods))
	for method := range methods {
		result = append(result, method)
	}
	slices.Sort(result)
	return result
}

// loggingMiddleware creates a middleware that logs requests.
func loggingMiddleware(logger func(method, path string, status int, duration time.Duration)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(wrapped, r)
			logger(r.Method, r.URL.Path, wrapped.status, time.Since(start))
		})
	}
}

// statusRecorder wraps a ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
