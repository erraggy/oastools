package builder

import (
	"context"
	"fmt"
	"net/http"

	"github.com/erraggy/oastools/httpvalidator"
)

// stdlibRouter implements RouterStrategy using net/http and PathMatcherSet.
// This is the default router that adds no dependencies.
type stdlibRouter struct {
	notFound http.Handler
}

// Build creates an http.Handler that routes requests using PathMatcherSet.
// Returns an error if the path patterns cannot be compiled (e.g., invalid path syntax).
func (r *stdlibRouter) Build(routes []operationRoute, dispatcher http.Handler) (http.Handler, error) {
	// Build PathMatcherSet from routes
	patterns := make([]string, 0, len(routes))
	seen := make(map[string]bool)
	for _, route := range routes {
		if !seen[route.Path] {
			patterns = append(patterns, route.Path)
			seen[route.Path] = true
		}
	}

	matcher, err := httpvalidator.NewPathMatcherSet(patterns)
	if err != nil {
		return nil, fmt.Errorf("builder: failed to create path matcher: %w", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Match path
		matched, params, found := matcher.Match(req.URL.Path)
		if !found {
			if r.notFound != nil {
				r.notFound.ServeHTTP(w, req)
			} else {
				http.NotFound(w, req)
			}
			return
		}

		// Store matched path and params in context
		ctx := req.Context()
		ctx = context.WithValue(ctx, matchedPathKey{}, matched)
		ctx = context.WithValue(ctx, pathParamsKey{}, params)

		dispatcher.ServeHTTP(w, req.WithContext(ctx))
	})

	return handler, nil
}

// PathParam extracts a path parameter from the request context.
func (r *stdlibRouter) PathParam(req *http.Request, name string) string {
	if params, ok := req.Context().Value(pathParamsKey{}).(map[string]string); ok {
		return params[name]
	}
	return ""
}

// matchedPathKey is the context key for the matched path template.
type matchedPathKey struct{}

// pathParamsKey is the context key for path parameters.
type pathParamsKey struct{}

// PathParam extracts a path parameter from the request context.
// This is a package-level function for convenience.
//
// Example:
//
//	petID := builder.PathParam(r, "petId")
func PathParam(r *http.Request, name string) string {
	if params, ok := r.Context().Value(pathParamsKey{}).(map[string]string); ok {
		return params[name]
	}
	return ""
}

// MatchedPath returns the matched path template from the request context.
//
// Example:
//
//	template := builder.MatchedPath(r) // e.g., "/pets/{petId}"
func MatchedPath(r *http.Request) string {
	if matched, ok := r.Context().Value(matchedPathKey{}).(string); ok {
		return matched
	}
	return ""
}
