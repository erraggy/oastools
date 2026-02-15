package mcpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/erraggy/oastools/parser"
)

// specInput represents the three ways an OAS spec can be provided to a tool.
// Exactly one of File, URL, or Content must be set.
type specInput struct {
	File    string `json:"file,omitempty"    jsonschema:"Path to an OAS file on disk"`
	URL     string `json:"url,omitempty"     jsonschema:"URL to fetch an OAS document from"`
	Content string `json:"content,omitempty" jsonschema:"Inline OAS document content (JSON or YAML)"`
}

// cacheEntry holds a cached parse result with its insertion order for LRU eviction.
type cacheEntry struct {
	result   *parser.ParseResult
	insertAt time.Time
}

// specCacheStore provides a session-scoped cache for parsed specs.
// File inputs are keyed by (absolutePath, modTime). Content inputs are keyed
// by a SHA-256 hash. URL inputs are never cached (the remote may change).
type specCacheStore struct {
	mu      sync.Mutex
	entries map[string]*cacheEntry
	maxSize int
}

var specCache = &specCacheStore{
	entries: make(map[string]*cacheEntry),
	maxSize: 10,
}

// get returns a cached result or nil.
func (c *specCacheStore) get(key string) *parser.ParseResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[key]; ok {
		// Touch entry for LRU.
		e.insertAt = time.Now()
		return e.result
	}
	return nil
}

// put stores a result in the cache, evicting the oldest entry if at capacity.
func (c *specCacheStore) put(key string, result *parser.ParseResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If already cached, just update.
	if _, ok := c.entries[key]; ok {
		c.entries[key] = &cacheEntry{result: result, insertAt: time.Now()}
		return
	}

	// Evict oldest if at capacity.
	if len(c.entries) >= c.maxSize {
		var oldestKey string
		var oldestTime time.Time
		for k, e := range c.entries {
			if oldestKey == "" || e.insertAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = e.insertAt
			}
		}
		if oldestKey != "" {
			delete(c.entries, oldestKey)
		}
	}

	c.entries[key] = &cacheEntry{result: result, insertAt: time.Now()}
}

// reset clears all cached entries. Used in tests.
func (c *specCacheStore) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

// size returns the number of cached entries.
func (c *specCacheStore) size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// makeCacheKey creates a cache key for the given spec input.
// Returns empty string if the input should not be cached (URLs, or when
// extra parser options are provided since we cannot distinguish option sets).
func makeCacheKey(s specInput, extraOpts []parser.Option) string {
	if len(extraOpts) > 0 {
		return ""
	}

	switch {
	case s.File != "":
		absPath, err := filepath.Abs(s.File)
		if err != nil {
			return ""
		}
		info, err := os.Stat(absPath)
		if err != nil {
			return "" // Can't stat, don't cache.
		}
		return fmt.Sprintf("file:%s:%d", absPath, info.ModTime().UnixNano())
	case s.Content != "":
		h := sha256.Sum256([]byte(s.Content))
		return fmt.Sprintf("content:%s", hex.EncodeToString(h[:]))
	default:
		return "" // URL inputs are not cached.
	}
}

// resolve parses the spec from whichever input was provided, using the cache
// for file and content inputs. Additional parser options can be passed to
// customize parsing behavior.
func (s specInput) resolve(extraOpts ...parser.Option) (*parser.ParseResult, error) {
	count := 0
	if s.File != "" {
		count++
	}
	if s.URL != "" {
		count++
	}
	if s.Content != "" {
		count++
	}
	if count != 1 {
		return nil, fmt.Errorf("exactly one of file, url, or content must be provided (got %d)", count)
	}

	// Check cache for file and content inputs.
	key := makeCacheKey(s, extraOpts)
	if key != "" {
		if cached := specCache.get(key); cached != nil {
			return cached, nil
		}
	}

	var opts []parser.Option
	switch {
	case s.File != "":
		opts = append(opts, parser.WithFilePath(s.File))
	case s.URL != "":
		opts = append(opts, parser.WithFilePath(s.URL))
	case s.Content != "":
		opts = append(opts, parser.WithReader(strings.NewReader(s.Content)))
	}
	opts = append(opts, extraOpts...)

	result, err := parser.ParseWithOptions(opts...)
	if err != nil {
		return nil, err
	}

	// Cache the result for future calls.
	if key != "" {
		specCache.put(key, result)
	}

	return result, nil
}
