package mcpserver

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecInput_ResolveFile(t *testing.T) {
	specCache.reset()
	// Use an existing testdata file from the repo
	input := specInput{File: "../../testdata/petstore-3.0.yaml"}
	result, err := input.resolve()
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Version)
}

func TestSpecInput_ResolveContent(t *testing.T) {
	specCache.reset()
	content := `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`
	input := specInput{Content: content}
	result, err := input.resolve()
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "3.0.0", result.Version)
}

func TestSpecInput_ResolveNoneProvided(t *testing.T) {
	input := specInput{}
	_, err := input.resolve()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one of file, url, or content must be provided")
}

func TestSpecInput_ResolveMultipleProvided(t *testing.T) {
	input := specInput{File: "foo.yaml", Content: "bar"}
	_, err := input.resolve()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one of file, url, or content must be provided")
}

func TestSpecInput_ResolveFileNotFound(t *testing.T) {
	specCache.reset()
	input := specInput{File: "/nonexistent/path.yaml"}
	_, err := input.resolve()
	assert.Error(t, err)
}

func TestSpecCache_HitOnSameFile(t *testing.T) {
	specCache.reset()
	input := specInput{File: "../../testdata/petstore-3.0.yaml"}

	// First call populates cache.
	result1, err := input.resolve()
	require.NoError(t, err)
	assert.Equal(t, 1, specCache.size())

	// Second call should return the same pointer (cache hit).
	result2, err := input.resolve()
	require.NoError(t, err)
	assert.Same(t, result1, result2, "expected same pointer from cache hit")
}

func TestSpecCache_MissOnModifiedFile(t *testing.T) {
	specCache.reset()

	// Create a temp file.
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.yaml")
	content1 := []byte(`openapi: "3.0.0"
info:
  title: Test V1
  version: "1.0"
paths: {}
`)
	require.NoError(t, os.WriteFile(path, content1, 0644))

	input := specInput{File: path}
	result1, err := input.resolve()
	require.NoError(t, err)
	accessor1 := result1.AsAccessor()
	require.NotNil(t, accessor1)
	assert.Equal(t, "Test V1", accessor1.GetInfo().Title)

	// Modify the file (change mtime).
	content2 := []byte(`openapi: "3.0.0"
info:
  title: Test V2
  version: "2.0"
paths: {}
`)
	require.NoError(t, os.WriteFile(path, content2, 0644))

	// Ensure mtime differs from the first write on coarse-grained filesystems.
	future := time.Now().Add(2 * time.Second)
	require.NoError(t, os.Chtimes(path, future, future))

	result2, err := input.resolve()
	require.NoError(t, err)
	// Should be a different result since mtime changed.
	assert.NotSame(t, result1, result2)
	accessor2 := result2.AsAccessor()
	require.NotNil(t, accessor2)
	assert.Equal(t, "Test V2", accessor2.GetInfo().Title)
}

func TestSpecCache_ContentHash(t *testing.T) {
	specCache.reset()
	content := `openapi: "3.0.0"
info:
  title: Hash Test
  version: "1.0"
paths: {}
`
	input := specInput{Content: content}

	result1, err := input.resolve()
	require.NoError(t, err)

	// Same content should hit cache.
	result2, err := input.resolve()
	require.NoError(t, err)
	assert.Same(t, result1, result2)
}

func TestSpecCache_LRUEviction(t *testing.T) {
	specCache.reset()

	// Insert 11 specs into a cache of size 10.
	// Track the first content's cache key to verify it is evicted.
	var firstKey string
	for i := range 11 {
		content := `openapi: "3.0.0"
info:
  title: "Spec ` + string(rune('A'+i)) + `"
  version: "1.0"
paths: {}
`
		if i == 0 {
			firstKey = makeCacheKey(specInput{Content: content}, nil)
		}
		input := specInput{Content: content}
		_, err := input.resolve()
		require.NoError(t, err)
	}

	// Cache should not exceed max size.
	assert.Equal(t, 10, specCache.size())

	// The first entry (oldest) should have been evicted.
	assert.Nil(t, specCache.get(firstKey), "expected oldest entry to be evicted")
}
