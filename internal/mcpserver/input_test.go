package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/erraggy/oastools/parser"
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

func TestSpecInput_ResolveCacheDisabled(t *testing.T) {
	specCache.reset()
	origCfg := cfg
	cfg = &serverConfig{
		CacheEnabled:       false,
		CacheMaxSize:       10,
		CacheFileTTL:       15 * time.Minute,
		CacheURLTTL:        5 * time.Minute,
		CacheContentTTL:    15 * time.Minute,
		CacheSweepInterval: 60 * time.Second,
		WalkLimit:          100,
		WalkDetailLimit:    25,
	}
	t.Cleanup(func() { cfg = origCfg })

	input := specInput{File: "../../testdata/petstore-3.0.yaml"}
	result1, err := input.resolve()
	require.NoError(t, err)
	assert.Equal(t, 0, specCache.size(), "cache should remain empty when disabled")

	result2, err := input.resolve()
	require.NoError(t, err)
	assert.NotSame(t, result1, result2, "each resolve should parse fresh when cache disabled")
}

func TestSpecCache_TTLExpiry(t *testing.T) {
	synctest.Run(func() {
		c := &specCacheStore{
			entries: make(map[string]*cacheEntry),
			maxSize: 10,
		}

		result := &parser.ParseResult{}
		c.putWithTTL("key1", result, 1*time.Millisecond)
		assert.Equal(t, 1, c.size())

		// Advance fake clock past TTL.
		time.Sleep(2 * time.Millisecond)

		// get() should return nil for expired entry and remove it.
		assert.Nil(t, c.get("key1"))
		assert.Equal(t, 0, c.size())
	})
}

func TestSpecCache_TTLNotExpired(t *testing.T) {
	c := &specCacheStore{
		entries: make(map[string]*cacheEntry),
		maxSize: 10,
	}

	result := &parser.ParseResult{}
	c.putWithTTL("key1", result, 1*time.Hour)

	// Should still be valid (no time advancement needed).
	assert.Same(t, result, c.get("key1"))
}

func TestSpecCache_Sweep(t *testing.T) {
	synctest.Run(func() {
		c := &specCacheStore{
			entries: make(map[string]*cacheEntry),
			maxSize: 10,
		}

		result := &parser.ParseResult{}
		c.putWithTTL("expired", result, 1*time.Millisecond)
		c.putWithTTL("valid", result, 1*time.Hour)

		// Advance fake clock past the short TTL.
		time.Sleep(2 * time.Millisecond)
		c.sweep()

		assert.Equal(t, 1, c.size())
		assert.Nil(t, c.get("expired"))
		assert.NotNil(t, c.get("valid"))
	})
}

func TestSpecCache_Sweeper(t *testing.T) {
	synctest.Run(func() {
		c := &specCacheStore{
			entries: make(map[string]*cacheEntry),
			maxSize: 10,
		}

		result := &parser.ParseResult{}
		c.putWithTTL("sweep-me", result, 1*time.Millisecond)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		c.startSweeper(ctx, 10*time.Millisecond)

		// Advance fake clock past TTL and sweep interval.
		time.Sleep(11 * time.Millisecond)
		// Wait for sweeper goroutine to complete its sweep cycle.
		synctest.Wait()

		assert.Equal(t, 0, c.size(), "sweeper should have removed expired entry")
	})
}
