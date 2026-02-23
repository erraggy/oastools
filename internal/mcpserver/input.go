package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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

// cacheEntry holds a cached parse result with LRU ordering and TTL expiry.
type cacheEntry struct {
	result    *parser.ParseResult
	insertAt  time.Time
	expiresAt time.Time
}

// specCacheStore provides a session-scoped cache for parsed specs.
// File inputs are keyed by (absolutePath, modTime). Content inputs are keyed
// by a SHA-256 hash. URL inputs are keyed by URL string.
// Entries have per-type TTLs and a background sweeper removes expired entries.
type specCacheStore struct {
	mu             sync.Mutex
	entries        map[string]*cacheEntry
	maxSize        int
	sweeperStarted atomic.Bool
}

var specCache = &specCacheStore{
	entries: make(map[string]*cacheEntry),
	maxSize: cfg.CacheMaxSize,
}

// get returns a cached result or nil. Expired entries are lazily removed.
func (c *specCacheStore) get(key string) *parser.ParseResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[key]; ok {
		if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
			delete(c.entries, key)
			return nil
		}
		// Touch entry for LRU.
		e.insertAt = time.Now()
		return e.result
	}
	return nil
}

// putWithTTL stores a result with a specific TTL, evicting the oldest entry if at capacity.
func (c *specCacheStore) putWithTTL(key string, result *parser.ParseResult, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	entry := &cacheEntry{result: result, insertAt: now, expiresAt: now.Add(ttl)}

	// If already cached, just update.
	if _, ok := c.entries[key]; ok {
		c.entries[key] = entry
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

	c.entries[key] = entry
}

// sweep removes all expired entries from the cache.
func (c *specCacheStore) sweep() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, e := range c.entries {
		if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
			delete(c.entries, k)
		}
	}
}

// startSweeper launches a background goroutine that periodically removes expired entries.
// It is safe to call multiple times; only the first call spawns a sweeper.
// It stops when ctx is cancelled.
func (c *specCacheStore) startSweeper(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}
	if !c.sweeperStarted.CompareAndSwap(false, true) {
		return
	}
	var sweeping atomic.Bool
	go func() {
		defer c.sweeperStarted.Store(false)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !sweeping.CompareAndSwap(false, true) {
					continue
				}
				c.sweep()
				sweeping.Store(false)
			}
		}
	}()
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
// Returns empty string when extra parser options are provided since we cannot
// distinguish option sets.
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
	case s.URL != "":
		return fmt.Sprintf("url:%s", s.URL)
	default:
		return ""
	}
}

// resolve parses the spec from whichever input was provided, using the cache
// for file, URL, and content inputs. Additional parser options can be passed
// to customize parsing behavior.
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

	// Enforce inline content size limit.
	if s.Content != "" && int64(len(s.Content)) > cfg.MaxInlineSize {
		return nil, fmt.Errorf("inline content size %d bytes exceeds maximum %d bytes; use file input instead, or set OASTOOLS_MAX_INLINE_SIZE to increase",
			len(s.Content), cfg.MaxInlineSize)
	}

	// Determine cache key and TTL (skip when caching is disabled).
	var key string
	var ttl time.Duration
	if cfg.CacheEnabled {
		key = makeCacheKey(s, extraOpts)
		switch {
		case s.File != "":
			ttl = cfg.CacheFileTTL
		case s.URL != "":
			ttl = cfg.CacheURLTTL
		default:
			ttl = cfg.CacheContentTTL
		}
	}

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
		// Inject SSRF-safe HTTP client for URL resolution unless private IPs are allowed.
		if !cfg.AllowPrivateIPs {
			opts = append(opts, parser.WithHTTPClient(newSafeHTTPClient()))
		}
	case s.Content != "":
		opts = append(opts, parser.WithReader(strings.NewReader(s.Content)))
	}
	opts = append(opts, extraOpts...)

	result, err := parser.ParseWithOptions(opts...)
	if err != nil {
		return nil, err
	}

	// Cache the result for future calls (key is empty when caching is disabled).
	if key != "" {
		specCache.putWithTTL(key, result, ttl)
	}

	return result, nil
}
