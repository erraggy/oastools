package httpvalidator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewPathMatcher Tests
// =============================================================================

func TestNewPathMatcher(t *testing.T) {
	t.Run("creates matcher for simple path", func(t *testing.T) {
		pm, err := NewPathMatcher("/pets")
		require.NoError(t, err)
		assert.Equal(t, "/pets", pm.Template())
		assert.Empty(t, pm.ParamNames())
	})

	t.Run("creates matcher for path with single parameter", func(t *testing.T) {
		pm, err := NewPathMatcher("/pets/{petId}")
		require.NoError(t, err)
		assert.Equal(t, "/pets/{petId}", pm.Template())
		assert.Equal(t, []string{"petId"}, pm.ParamNames())
	})

	t.Run("creates matcher for path with multiple parameters", func(t *testing.T) {
		pm, err := NewPathMatcher("/users/{userId}/posts/{postId}")
		require.NoError(t, err)
		assert.Equal(t, []string{"userId", "postId"}, pm.ParamNames())
	})

	t.Run("errors on empty template", func(t *testing.T) {
		_, err := NewPathMatcher("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("errors on unclosed brace", func(t *testing.T) {
		_, err := NewPathMatcher("/pets/{petId")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unclosed")
	})

	t.Run("errors on empty parameter name", func(t *testing.T) {
		_, err := NewPathMatcher("/pets/{}")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty path parameter")
	})

	t.Run("errors on duplicate parameter names", func(t *testing.T) {
		_, err := NewPathMatcher("/users/{id}/posts/{id}")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("escapes regex special characters", func(t *testing.T) {
		// Test that special regex chars in paths are properly escaped
		pm, err := NewPathMatcher("/api.v1/users")
		require.NoError(t, err)

		matched, _ := pm.Match("/api.v1/users")
		assert.True(t, matched)

		// The dot should be escaped, not match any char
		matched, _ = pm.Match("/apiXv1/users")
		assert.False(t, matched)
	})

	t.Run("handles path with special chars", func(t *testing.T) {
		pm, err := NewPathMatcher("/api+beta/items")
		require.NoError(t, err)

		matched, _ := pm.Match("/api+beta/items")
		assert.True(t, matched)
	})
}

// =============================================================================
// PathMatcher.Match Tests
// =============================================================================

func TestPathMatcher_Match(t *testing.T) {
	t.Run("matches exact path", func(t *testing.T) {
		pm, _ := NewPathMatcher("/pets")
		matched, params := pm.Match("/pets")

		assert.True(t, matched)
		assert.Empty(t, params)
	})

	t.Run("extracts single parameter", func(t *testing.T) {
		pm, _ := NewPathMatcher("/pets/{petId}")
		matched, params := pm.Match("/pets/123")

		assert.True(t, matched)
		assert.Equal(t, "123", params["petId"])
	})

	t.Run("extracts multiple parameters", func(t *testing.T) {
		pm, _ := NewPathMatcher("/users/{userId}/posts/{postId}")
		matched, params := pm.Match("/users/42/posts/99")

		assert.True(t, matched)
		assert.Equal(t, "42", params["userId"])
		assert.Equal(t, "99", params["postId"])
	})

	t.Run("does not match different path", func(t *testing.T) {
		pm, _ := NewPathMatcher("/pets")
		matched, _ := pm.Match("/users")

		assert.False(t, matched)
	})

	t.Run("does not match path with extra segments", func(t *testing.T) {
		pm, _ := NewPathMatcher("/pets")
		matched, _ := pm.Match("/pets/123")

		assert.False(t, matched)
	})

	t.Run("does not match shorter path", func(t *testing.T) {
		pm, _ := NewPathMatcher("/pets/{petId}")
		matched, _ := pm.Match("/pets")

		assert.False(t, matched)
	})

	t.Run("parameters cannot contain slashes", func(t *testing.T) {
		pm, _ := NewPathMatcher("/files/{path}")
		matched, _ := pm.Match("/files/dir/file.txt")

		assert.False(t, matched, "parameter should not match slashes")
	})

	t.Run("handles URL-encoded values", func(t *testing.T) {
		pm, _ := NewPathMatcher("/search/{query}")
		matched, params := pm.Match("/search/hello%20world")

		assert.True(t, matched)
		assert.Equal(t, "hello%20world", params["query"])
	})
}

// =============================================================================
// PathMatcher Getters Tests
// =============================================================================

func TestPathMatcher_Template(t *testing.T) {
	pm, _ := NewPathMatcher("/users/{id}/profile")
	assert.Equal(t, "/users/{id}/profile", pm.Template())
}

func TestPathMatcher_ParamNames(t *testing.T) {
	pm, _ := NewPathMatcher("/users/{userId}/posts/{postId}/comments/{commentId}")
	assert.Equal(t, []string{"userId", "postId", "commentId"}, pm.ParamNames())
}

// =============================================================================
// PathMatcherSet Tests
// =============================================================================

func TestNewPathMatcherSet(t *testing.T) {
	t.Run("creates set from templates", func(t *testing.T) {
		pms, err := NewPathMatcherSet([]string{"/pets", "/pets/{petId}"})
		require.NoError(t, err)

		templates := pms.Templates()
		assert.Len(t, templates, 2)
		assert.Contains(t, templates, "/pets")
		assert.Contains(t, templates, "/pets/{petId}")
	})

	t.Run("errors on invalid template", func(t *testing.T) {
		_, err := NewPathMatcherSet([]string{"/pets", "/pets/{unclosed"})
		assert.Error(t, err)
	})

	t.Run("handles empty set", func(t *testing.T) {
		pms, err := NewPathMatcherSet([]string{})
		require.NoError(t, err)
		assert.Empty(t, pms.Templates())
	})
}

func TestPathMatcherSet_Match(t *testing.T) {
	t.Run("matches exact path before parameterized", func(t *testing.T) {
		pms, _ := NewPathMatcherSet([]string{"/pets/{petId}", "/pets"})

		template, params, found := pms.Match("/pets")
		assert.True(t, found)
		assert.Equal(t, "/pets", template)
		assert.Empty(t, params)
	})

	t.Run("extracts parameters from parameterized path", func(t *testing.T) {
		pms, _ := NewPathMatcherSet([]string{"/pets", "/pets/{petId}"})

		template, params, found := pms.Match("/pets/123")
		assert.True(t, found)
		assert.Equal(t, "/pets/{petId}", template)
		assert.Equal(t, "123", params["petId"])
	})

	t.Run("returns not found for unknown path", func(t *testing.T) {
		pms, _ := NewPathMatcherSet([]string{"/pets"})

		template, params, found := pms.Match("/unknown")
		assert.False(t, found)
		assert.Empty(t, template)
		assert.Nil(t, params)
	})

	t.Run("prefers more specific paths", func(t *testing.T) {
		pms, _ := NewPathMatcherSet([]string{
			"/users/{id}",
			"/users/{id}/profile",
		})

		template, _, found := pms.Match("/users/123/profile")
		assert.True(t, found)
		assert.Equal(t, "/users/{id}/profile", template)
	})

	t.Run("longer exact matches win over shorter", func(t *testing.T) {
		pms, _ := NewPathMatcherSet([]string{
			"/api",
			"/api/v1",
			"/api/v1/users",
		})

		template, _, found := pms.Match("/api/v1/users")
		assert.True(t, found)
		assert.Equal(t, "/api/v1/users", template)
	})
}

func TestPathMatcherSet_Templates(t *testing.T) {
	templates := []string{"/a", "/b", "/c/{id}"}
	pms, _ := NewPathMatcherSet(templates)

	result := pms.Templates()
	assert.Len(t, result, 3)
	// All templates should be present, order may differ due to sorting
	for _, tmpl := range templates {
		assert.Contains(t, result, tmpl)
	}
}

// =============================================================================
// Specificity Tests
// =============================================================================

func TestPathMatcherSet_Specificity(t *testing.T) {
	t.Run("prefers exact over parameterized", func(t *testing.T) {
		pms, _ := NewPathMatcherSet([]string{
			"/{a}/{b}",
			"/users/{id}",
			"/users/admin",
		})

		template, _, found := pms.Match("/users/admin")
		assert.True(t, found)
		assert.Equal(t, "/users/admin", template, "exact path should match over parameterized")
	})

	t.Run("prefers less parameters", func(t *testing.T) {
		pms, _ := NewPathMatcherSet([]string{
			"/{a}/{b}/{c}",
			"/users/{id}/{action}",
			"/users/me/profile",
		})

		template, _, found := pms.Match("/users/me/profile")
		assert.True(t, found)
		assert.Equal(t, "/users/me/profile", template)
	})

	t.Run("alphabetical tiebreaker", func(t *testing.T) {
		pms, _ := NewPathMatcherSet([]string{
			"/pets/{a}",
			"/pets/{b}",
		})

		template, _, found := pms.Match("/pets/123")
		assert.True(t, found)
		// Should match one consistently (alphabetically first)
		assert.Equal(t, "/pets/{a}", template)
	})
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestPathMatcher_EdgeCases(t *testing.T) {
	t.Run("root path", func(t *testing.T) {
		pm, err := NewPathMatcher("/")
		require.NoError(t, err)

		matched, _ := pm.Match("/")
		assert.True(t, matched)

		matched, _ = pm.Match("/anything")
		assert.False(t, matched)
	})

	t.Run("trailing slash matters", func(t *testing.T) {
		pm, err := NewPathMatcher("/pets/")
		require.NoError(t, err)

		matched, _ := pm.Match("/pets/")
		assert.True(t, matched)

		matched, _ = pm.Match("/pets")
		assert.False(t, matched)
	})

	t.Run("parameter at start", func(t *testing.T) {
		pm, err := NewPathMatcher("/{version}/api")
		require.NoError(t, err)

		matched, params := pm.Match("/v1/api")
		assert.True(t, matched)
		assert.Equal(t, "v1", params["version"])
	})

	t.Run("parameter at end", func(t *testing.T) {
		pm, err := NewPathMatcher("/files/{filename}")
		require.NoError(t, err)

		matched, params := pm.Match("/files/document.pdf")
		assert.True(t, matched)
		assert.Equal(t, "document.pdf", params["filename"])
	})

	t.Run("consecutive parameters", func(t *testing.T) {
		// This is unusual but valid OAS
		pm, err := NewPathMatcher("/a/{x}/{y}")
		require.NoError(t, err)

		matched, params := pm.Match("/a/1/2")
		assert.True(t, matched)
		assert.Equal(t, "1", params["x"])
		assert.Equal(t, "2", params["y"])
	})
}
