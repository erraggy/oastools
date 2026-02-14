package builder

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdlibRouter_Build(t *testing.T) {
	t.Parallel()

	router := &stdlibRouter{}

	routes := []operationRoute{
		{Method: http.MethodGet, Path: "/pets", OperationID: "listPets"},
		{Method: http.MethodGet, Path: "/pets/{petId}", OperationID: "getPet"},
		{Method: http.MethodPost, Path: "/pets", OperationID: "createPet"},
	}

	dispatcher := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		matched := MatchedPath(r)
		w.Header().Set("X-Matched-Path", matched)
		w.WriteHeader(http.StatusOK)
	})

	handler, err := router.Build(routes, dispatcher)
	require.NoError(t, err)

	tests := []struct {
		name         string
		path         string
		expectedPath string
	}{
		{"exact match", "/pets", "/pets"},
		{"parameterized match", "/pets/123", "/pets/{petId}"},
		{"parameterized with string", "/pets/fluffy", "/pets/{petId}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedPath, rec.Header().Get("X-Matched-Path"))
		})
	}
}

func TestStdlibRouter_NotFound(t *testing.T) {
	t.Parallel()

	router := &stdlibRouter{}

	routes := []operationRoute{
		{Method: http.MethodGet, Path: "/pets", OperationID: "listPets"},
	}

	dispatcher := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler, err := router.Build(routes, dispatcher)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestStdlibRouter_CustomNotFoundHandler(t *testing.T) {
	t.Parallel()

	customNotFound := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Custom", "true")
		w.WriteHeader(http.StatusNotFound)
	})

	router := &stdlibRouter{notFound: customNotFound}

	routes := []operationRoute{
		{Method: http.MethodGet, Path: "/pets", OperationID: "listPets"},
	}

	dispatcher := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler, err := router.Build(routes, dispatcher)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "true", rec.Header().Get("X-Custom"))
}

func TestStdlibRouter_PathParam(t *testing.T) {
	t.Parallel()

	router := &stdlibRouter{}

	routes := []operationRoute{
		{Method: http.MethodGet, Path: "/pets/{petId}", OperationID: "getPet"},
	}

	var capturedParam string
	dispatcher := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedParam = router.PathParam(r, "petId")
		w.WriteHeader(http.StatusOK)
	})

	handler, err := router.Build(routes, dispatcher)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pets/123", nil)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "123", capturedParam)
}

func TestStdlibRouter_MultiplePathParams(t *testing.T) {
	t.Parallel()

	router := &stdlibRouter{}

	routes := []operationRoute{
		{Method: http.MethodGet, Path: "/users/{userId}/pets/{petId}", OperationID: "getUserPet"},
	}

	var userId, petId string
	dispatcher := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userId = router.PathParam(r, "userId")
		petId = router.PathParam(r, "petId")
		w.WriteHeader(http.StatusOK)
	})

	handler, err := router.Build(routes, dispatcher)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/42/pets/99", nil)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "42", userId)
	assert.Equal(t, "99", petId)
}

func TestStdlibRouter_PathParamNotFound(t *testing.T) {
	t.Parallel()

	router := &stdlibRouter{}

	routes := []operationRoute{
		{Method: http.MethodGet, Path: "/pets/{petId}", OperationID: "getPet"},
	}

	var capturedParam string
	dispatcher := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedParam = router.PathParam(r, "nonexistent")
		w.WriteHeader(http.StatusOK)
	})

	handler, err := router.Build(routes, dispatcher)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pets/123", nil)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "", capturedParam)
}

func TestPathParam(t *testing.T) {
	t.Parallel()

	router := &stdlibRouter{}

	routes := []operationRoute{
		{Method: http.MethodGet, Path: "/pets/{petId}", OperationID: "getPet"},
	}

	var capturedParam string
	dispatcher := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedParam = PathParam(r, "petId")
		w.WriteHeader(http.StatusOK)
	})

	handler, err := router.Build(routes, dispatcher)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pets/abc", nil)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "abc", capturedParam)
}

func TestMatchedPath(t *testing.T) {
	t.Parallel()

	router := &stdlibRouter{}

	routes := []operationRoute{
		{Method: http.MethodGet, Path: "/pets/{petId}", OperationID: "getPet"},
	}

	var matchedPath string
	dispatcher := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		matchedPath = MatchedPath(r)
		w.WriteHeader(http.StatusOK)
	})

	handler, err := router.Build(routes, dispatcher)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pets/123", nil)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "/pets/{petId}", matchedPath)
}

func TestPathParam_NoContext(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	result := PathParam(req, "id")

	assert.Equal(t, "", result)
}

func TestMatchedPath_NoContext(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	result := MatchedPath(req)

	assert.Equal(t, "", result)
}

func TestStdlibRouter_DuplicatePaths(t *testing.T) {
	t.Parallel()

	router := &stdlibRouter{}

	// Same path, different methods
	routes := []operationRoute{
		{Method: http.MethodGet, Path: "/pets", OperationID: "listPets"},
		{Method: http.MethodPost, Path: "/pets", OperationID: "createPet"},
		{Method: http.MethodGet, Path: "/pets/{petId}", OperationID: "getPet"},
		{Method: http.MethodPut, Path: "/pets/{petId}", OperationID: "updatePet"},
	}

	dispatcher := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Matched-Path", MatchedPath(r))
		w.WriteHeader(http.StatusOK)
	})

	handler, err := router.Build(routes, dispatcher)
	require.NoError(t, err)

	// Test that path matching still works
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pets", nil)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "/pets", rec.Header().Get("X-Matched-Path"))
}

func TestStdlibRouter_EmptyRoutes(t *testing.T) {
	t.Parallel()

	router := &stdlibRouter{}

	routes := []operationRoute{}

	dispatcher := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler, err := router.Build(routes, dispatcher)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	handler.ServeHTTP(rec, req)

	// Should return 404 for any path when there are no routes
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
