package builder

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/erraggy/oastools/httpvalidator"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// routingTestServer builds a server with GET /pets (required integer query
// parameter "limit") and GET /pets/{petId}. The handler for GET /pets records
// the Request it receives.
func routingTestServer(t *testing.T, validation bool, opts ...ServerBuilderOption) (*ServerResult, *[]*Request) {
	t.Helper()

	if !validation {
		opts = append(opts, WithoutValidation())
	}
	srv := NewServerBuilder(parser.OASVersion320, opts...).
		SetTitle("Routing API").
		SetVersion("1.0.0")

	var seen []*Request
	srv.AddOperation(http.MethodGet, "/pets",
		WithOperationID("listPets"),
		WithQueryParam("limit", int32(0), WithParamRequired(true)),
		WithResponse(http.StatusOK, []string{}),
		WithHandler(func(_ context.Context, req *Request) Response {
			seen = append(seen, req)
			return JSON(http.StatusOK, []string{})
		}),
	)
	srv.AddOperation(http.MethodGet, "/pets/{petId}",
		WithOperationID("getPet"),
		WithPathParam("petId", ""),
		WithResponse(http.StatusOK, struct{}{}),
		WithHandler(func(_ context.Context, _ *Request) Response {
			return JSON(http.StatusOK, struct{}{})
		}),
	)

	result, err := srv.BuildServer()
	require.NoError(t, err)
	return result, &seen
}

func serve(h http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestServer_RoutingRunsBeforeValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		method    string
		target    string
		wantCode  int
		wantAllow string
	}{
		{name: "unknown path", method: http.MethodGet, target: "/owners", wantCode: http.StatusNotFound},
		{name: "wrong method", method: http.MethodDelete, target: "/pets?limit=1", wantCode: http.StatusMethodNotAllowed, wantAllow: http.MethodGet},
		{name: "valid request", method: http.MethodGet, target: "/pets?limit=1", wantCode: http.StatusOK},
	}

	for _, validation := range []bool{true, false} {
		for _, tc := range cases {
			name := tc.name + " validation off"
			if validation {
				name = tc.name + " validation on"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				result, _ := routingTestServer(t, validation)

				rec := serve(result.Handler, tc.method, tc.target)

				assert.Equal(t, tc.wantCode, rec.Code)
				assert.Equal(t, tc.wantAllow, rec.Header().Get("Allow"))
			})
		}
	}
}

func TestServer_NotFoundAndMethodNotAllowedDefaultsMatchAcrossValidationModes(t *testing.T) {
	t.Parallel()

	on, _ := routingTestServer(t, true)
	off, _ := routingTestServer(t, false)

	for _, tc := range []struct{ method, target string }{
		{http.MethodGet, "/owners"},
		{http.MethodDelete, "/pets"},
		{http.MethodPost, "/pets/1"},
	} {
		recOn := serve(on.Handler, tc.method, tc.target)
		recOff := serve(off.Handler, tc.method, tc.target)

		assert.Equal(t, recOff.Code, recOn.Code, "%s %s", tc.method, tc.target)
		assert.Equal(t, recOff.Header(), recOn.Header(), "%s %s", tc.method, tc.target)
		assert.Equal(t, recOff.Body.String(), recOn.Body.String(), "%s %s", tc.method, tc.target)
	}
}

func TestServer_CustomNotFoundAndMethodNotAllowedHandlers(t *testing.T) {
	t.Parallel()

	for _, validation := range []bool{true, false} {
		name := "validation off"
		if validation {
			name = "validation on"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var notFoundCalls, methodNotAllowedCalls int
			var allowSeen string
			notFound := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				notFoundCalls++
				w.WriteHeader(http.StatusTeapot)
			})
			methodNotAllowed := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				methodNotAllowedCalls++
				allowSeen = w.Header().Get("Allow")
				w.WriteHeader(http.StatusPaymentRequired)
			})

			result, _ := routingTestServer(t, validation,
				WithNotFoundHandler(notFound),
				WithMethodNotAllowedHandler(methodNotAllowed),
			)

			rec := serve(result.Handler, http.MethodGet, "/owners")
			assert.Equal(t, http.StatusTeapot, rec.Code)
			assert.Equal(t, 1, notFoundCalls)

			rec = serve(result.Handler, http.MethodDelete, "/pets")
			assert.Equal(t, http.StatusPaymentRequired, rec.Code)
			assert.Equal(t, 1, methodNotAllowedCalls)
			assert.Equal(t, http.MethodGet, allowSeen)
			assert.Equal(t, http.MethodGet, rec.Header().Get("Allow"))

			rec = serve(result.Handler, http.MethodGet, "/pets?limit=1")
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, 1, notFoundCalls)
			assert.Equal(t, 1, methodNotAllowedCalls)
		})
	}
}

func TestServer_CustomNotFoundDoesNotMutateConfiguredRouter(t *testing.T) {
	t.Parallel()

	router := &stdlibRouter{}
	notFound := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	result, _ := routingTestServer(t, true, WithRouter(router), WithNotFoundHandler(notFound))

	assert.Equal(t, http.StatusTeapot, serve(result.Handler, http.MethodGet, "/owners").Code)
	assert.Nil(t, router.notFound)
}

func TestDispatcher_NotFoundUsesCustomHandler(t *testing.T) {
	t.Parallel()

	notFound := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	srv := NewServerBuilder(parser.OASVersion320, WithNotFoundHandler(notFound))
	d := srv.buildDispatcher([]operationRoute{{Method: http.MethodGet, Path: "/pets"}}, nil)

	// No matched path in the context.
	rec := serve(d, http.MethodGet, "/pets")
	assert.Equal(t, http.StatusTeapot, rec.Code)

	// A matched path the dispatcher has no route for.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/owners", nil)
	req = req.WithContext(context.WithValue(req.Context(), matchedPathKey{}, "/owners"))
	d.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTeapot, rec.Code)
}

func TestServer_ValidationStillAppliesToMatchedRoutes(t *testing.T) {
	t.Parallel()

	t.Run("default error response", func(t *testing.T) {
		t.Parallel()
		result, seen := routingTestServer(t, true)

		rec := serve(result.Handler, http.MethodGet, "/pets")

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "validation failed")
		assert.Empty(t, *seen)
	})

	t.Run("OnValidationError", func(t *testing.T) {
		t.Parallel()
		var got *httpvalidator.RequestValidationResult
		cfg := DefaultValidationConfig()
		cfg.OnValidationError = func(w http.ResponseWriter, _ *http.Request, result *httpvalidator.RequestValidationResult) {
			got = result
			w.WriteHeader(http.StatusUnprocessableEntity)
		}
		result, seen := routingTestServer(t, true, WithValidationConfig(cfg))

		rec := serve(result.Handler, http.MethodGet, "/pets")

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		require.NotNil(t, got)
		assert.NotEmpty(t, got.Errors)
		assert.Empty(t, *seen)
	})

	t.Run("validated params reach the handler", func(t *testing.T) {
		t.Parallel()
		result, seen := routingTestServer(t, true)

		rec := serve(result.Handler, http.MethodGet, "/pets?limit=5")

		require.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, *seen, 1)
		limit := (*seen)[0].QueryParams["limit"]
		assert.NotNil(t, limit)
		assert.IsNotType(t, "", limit, "validation deserializes the query parameter")
	})
}

func TestServer_OuterMiddlewareSeesRoutingResponses(t *testing.T) {
	t.Parallel()

	var statuses []int
	result, _ := routingTestServer(t, true,
		WithRequestLogging(func(_, _ string, status int, _ time.Duration) {
			statuses = append(statuses, status)
		}),
	)

	serve(result.Handler, http.MethodGet, "/owners")
	serve(result.Handler, http.MethodDelete, "/pets")
	serve(result.Handler, http.MethodGet, "/pets")

	assert.Equal(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusBadRequest}, statuses)
}
