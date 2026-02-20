# MCP Cache TTL & Configuration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extend the MCP server with TTL-based cache eviction, URL spec caching, and user-configurable defaults via environment variables.

**Architecture:** A new `config.go` file holds the `serverConfig` struct and `loadConfig()`. The existing `specCacheStore` in `input.go` evolves to support TTL + background sweeping + URL caching. Tool handlers reference a package-level `cfg` variable for defaults. All configuration is delivered via `OASTOOLS_*` environment variables.

**Tech Stack:** Go 1.24, Go MCP SDK v1.3.0, `sync/atomic` for sweeper guard, `time.Ticker` for background sweeping, `t.Setenv()` for testing.

**Design doc:** `docs/plans/2026-02-19-mcp-cache-config-design.md`

---

### Task 1: Configuration System — Tests

**Files:**
- Create: `internal/mcpserver/config_test.go`

**Context:** The `serverConfig` struct doesn't exist yet. We write the tests first, describing the behavior we want. Tests use `t.Setenv()` to inject env vars and verify `loadConfig()` returns correct values.

**Step 1: Write the config test file**

Create `internal/mcpserver/config_test.go` with these tests:

```go
package mcpserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// No env vars set — all defaults should apply.
	c := loadConfig()

	assert.True(t, c.CacheEnabled)
	assert.Equal(t, 10, c.CacheMaxSize)
	assert.Equal(t, 15*time.Minute, c.CacheFileTTL)
	assert.Equal(t, 5*time.Minute, c.CacheURLTTL)
	assert.Equal(t, 15*time.Minute, c.CacheContentTTL)
	assert.Equal(t, 60*time.Second, c.CacheSweepInterval)
	assert.Equal(t, 100, c.WalkLimit)
	assert.Equal(t, 25, c.WalkDetailLimit)
	assert.Empty(t, c.JoinPathStrategy)
	assert.Empty(t, c.JoinSchemaStrategy)
	assert.False(t, c.ValidateStrict)
	assert.False(t, c.ValidateNoWarnings)
	assert.Equal(t, 30*time.Second, c.HTTPTimeout)
	assert.Empty(t, c.HTTPUserAgent)
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	t.Setenv("OASTOOLS_CACHE_ENABLED", "false")
	t.Setenv("OASTOOLS_CACHE_MAX_SIZE", "50")
	t.Setenv("OASTOOLS_CACHE_FILE_TTL", "30m")
	t.Setenv("OASTOOLS_CACHE_URL_TTL", "2m")
	t.Setenv("OASTOOLS_CACHE_CONTENT_TTL", "10m")
	t.Setenv("OASTOOLS_CACHE_SWEEP_INTERVAL", "30s")
	t.Setenv("OASTOOLS_WALK_LIMIT", "200")
	t.Setenv("OASTOOLS_WALK_DETAIL_LIMIT", "50")
	t.Setenv("OASTOOLS_JOIN_PATH_STRATEGY", "accept-left")
	t.Setenv("OASTOOLS_JOIN_SCHEMA_STRATEGY", "rename")
	t.Setenv("OASTOOLS_VALIDATE_STRICT", "true")
	t.Setenv("OASTOOLS_VALIDATE_NO_WARNINGS", "true")
	t.Setenv("OASTOOLS_HTTP_TIMEOUT", "10s")
	t.Setenv("OASTOOLS_HTTP_USER_AGENT", "my-agent/1.0")

	c := loadConfig()

	assert.False(t, c.CacheEnabled)
	assert.Equal(t, 50, c.CacheMaxSize)
	assert.Equal(t, 30*time.Minute, c.CacheFileTTL)
	assert.Equal(t, 2*time.Minute, c.CacheURLTTL)
	assert.Equal(t, 10*time.Minute, c.CacheContentTTL)
	assert.Equal(t, 30*time.Second, c.CacheSweepInterval)
	assert.Equal(t, 200, c.WalkLimit)
	assert.Equal(t, 50, c.WalkDetailLimit)
	assert.Equal(t, "accept-left", c.JoinPathStrategy)
	assert.Equal(t, "rename", c.JoinSchemaStrategy)
	assert.True(t, c.ValidateStrict)
	assert.True(t, c.ValidateNoWarnings)
	assert.Equal(t, 10*time.Second, c.HTTPTimeout)
	assert.Equal(t, "my-agent/1.0", c.HTTPUserAgent)
}

func TestLoadConfig_InvalidValues_UseDefaults(t *testing.T) {
	t.Setenv("OASTOOLS_CACHE_MAX_SIZE", "banana")
	t.Setenv("OASTOOLS_CACHE_FILE_TTL", "not-a-duration")
	t.Setenv("OASTOOLS_CACHE_ENABLED", "maybe")
	t.Setenv("OASTOOLS_WALK_LIMIT", "-5")

	c := loadConfig()

	// Invalid values should fall back to defaults.
	assert.True(t, c.CacheEnabled)
	assert.Equal(t, 10, c.CacheMaxSize)
	assert.Equal(t, 15*time.Minute, c.CacheFileTTL)
	assert.Equal(t, 100, c.WalkLimit)
}

func TestLoadConfig_PartialOverrides(t *testing.T) {
	// Only override some values; others stay at defaults.
	t.Setenv("OASTOOLS_WALK_LIMIT", "42")
	t.Setenv("OASTOOLS_CACHE_URL_TTL", "10m")

	c := loadConfig()

	assert.Equal(t, 42, c.WalkLimit)
	assert.Equal(t, 10*time.Minute, c.CacheURLTTL)
	// Unchanged defaults:
	assert.Equal(t, 25, c.WalkDetailLimit)
	assert.Equal(t, 15*time.Minute, c.CacheFileTTL)
	assert.True(t, c.CacheEnabled)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcpserver/ -run TestLoadConfig -v`
Expected: FAIL — `loadConfig` is not defined.

**Step 3: Commit the failing tests**

```bash
git add internal/mcpserver/config_test.go
git commit -m "test(mcp): add config loading tests (red phase)"
```

---

### Task 2: Configuration System — Implementation

**Files:**
- Create: `internal/mcpserver/config.go`

**Context:** Implement `serverConfig` struct and `loadConfig()` to make the tests from Task 1 pass. Uses `os.Getenv`, `strconv`, and `time.ParseDuration` — no external dependencies.

**Step 1: Write the config implementation**

Create `internal/mcpserver/config.go`:

```go
package mcpserver

import (
	"log/slog"
	"os"
	"strconv"
	"time"
)

// serverConfig holds all configurable MCP server defaults.
// Loaded once at startup from environment variables via loadConfig().
type serverConfig struct {
	// Cache settings.
	CacheEnabled       bool
	CacheMaxSize       int
	CacheFileTTL       time.Duration
	CacheURLTTL        time.Duration
	CacheContentTTL    time.Duration
	CacheSweepInterval time.Duration

	// Walk tool defaults.
	WalkLimit       int
	WalkDetailLimit int

	// Join tool defaults.
	JoinPathStrategy   string
	JoinSchemaStrategy string

	// Validate tool defaults.
	ValidateStrict     bool
	ValidateNoWarnings bool

	// HTTP settings.
	HTTPTimeout   time.Duration
	HTTPUserAgent string
}

// cfg is the active server configuration, initialized in Run().
var cfg = loadConfig()

// loadConfig reads configuration from OASTOOLS_* environment variables.
// Invalid values log a warning and fall back to the hardcoded default.
func loadConfig() *serverConfig {
	return &serverConfig{
		CacheEnabled:       envBool("OASTOOLS_CACHE_ENABLED", true),
		CacheMaxSize:       envInt("OASTOOLS_CACHE_MAX_SIZE", 10),
		CacheFileTTL:       envDuration("OASTOOLS_CACHE_FILE_TTL", 15*time.Minute),
		CacheURLTTL:        envDuration("OASTOOLS_CACHE_URL_TTL", 5*time.Minute),
		CacheContentTTL:    envDuration("OASTOOLS_CACHE_CONTENT_TTL", 15*time.Minute),
		CacheSweepInterval: envDuration("OASTOOLS_CACHE_SWEEP_INTERVAL", 60*time.Second),
		WalkLimit:          envInt("OASTOOLS_WALK_LIMIT", 100),
		WalkDetailLimit:    envInt("OASTOOLS_WALK_DETAIL_LIMIT", 25),
		JoinPathStrategy:   os.Getenv("OASTOOLS_JOIN_PATH_STRATEGY"),
		JoinSchemaStrategy: os.Getenv("OASTOOLS_JOIN_SCHEMA_STRATEGY"),
		ValidateStrict:     envBool("OASTOOLS_VALIDATE_STRICT", false),
		ValidateNoWarnings: envBool("OASTOOLS_VALIDATE_NO_WARNINGS", false),
		HTTPTimeout:        envDuration("OASTOOLS_HTTP_TIMEOUT", 30*time.Second),
		HTTPUserAgent:      os.Getenv("OASTOOLS_HTTP_USER_AGENT"),
	}
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		slog.Warn("invalid bool env var, using default", "key", key, "value", v, "default", fallback)
		return fallback
	}
	return b
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		slog.Warn("invalid int env var, using default", "key", key, "value", v, "default", fallback)
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		slog.Warn("invalid duration env var, using default", "key", key, "value", v, "default", fallback)
		return fallback
	}
	return d
}
```

**Step 2: Run tests to verify they pass**

Run: `go test ./internal/mcpserver/ -run TestLoadConfig -v`
Expected: All 4 tests PASS.

**Step 3: Run gopls diagnostics**

Run: `go_diagnostics` on `internal/mcpserver/config.go`

**Step 4: Commit**

```bash
git add internal/mcpserver/config.go
git commit -m "feat(mcp): add serverConfig and loadConfig from env vars"
```

---

### Task 3: Cache TTL Evolution — Tests

**Files:**
- Modify: `internal/mcpserver/input_test.go`

**Context:** Add tests for TTL-based expiry, the `sweep()` method, and the background sweeper. These tests will fail because the cache doesn't support TTL yet.

**Step 1: Add TTL and sweeper tests**

Append to `internal/mcpserver/input_test.go`:

```go
func TestSpecCache_TTLExpiry(t *testing.T) {
	c := &specCacheStore{
		entries: make(map[string]*cacheEntry),
		maxSize: 10,
	}

	// Insert with a very short TTL.
	result := &parser.ParseResult{}
	c.putWithTTL("key1", result, 1*time.Millisecond)
	assert.Equal(t, 1, c.size())

	// Wait for expiry.
	time.Sleep(5 * time.Millisecond)

	// get() should return nil for expired entry and remove it.
	assert.Nil(t, c.get("key1"))
	assert.Equal(t, 0, c.size())
}

func TestSpecCache_TTLNotExpired(t *testing.T) {
	c := &specCacheStore{
		entries: make(map[string]*cacheEntry),
		maxSize: 10,
	}

	result := &parser.ParseResult{}
	c.putWithTTL("key1", result, 1*time.Hour)

	// Should still be valid.
	assert.Same(t, result, c.get("key1"))
}

func TestSpecCache_Sweep(t *testing.T) {
	c := &specCacheStore{
		entries: make(map[string]*cacheEntry),
		maxSize: 10,
	}

	result := &parser.ParseResult{}
	c.putWithTTL("expired", result, 1*time.Millisecond)
	c.putWithTTL("valid", result, 1*time.Hour)

	time.Sleep(5 * time.Millisecond)
	c.sweep()

	assert.Equal(t, 1, c.size())
	assert.Nil(t, c.get("expired"))
	assert.NotNil(t, c.get("valid"))
}

func TestSpecCache_Sweeper(t *testing.T) {
	c := &specCacheStore{
		entries: make(map[string]*cacheEntry),
		maxSize: 10,
	}

	result := &parser.ParseResult{}
	c.putWithTTL("sweep-me", result, 1*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.startSweeper(ctx, 10*time.Millisecond)

	// Wait for at least one sweep cycle.
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, c.size(), "sweeper should have removed expired entry")
}
```

**Note:** You'll need to add `"context"` to the import block.

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcpserver/ -run "TestSpecCache_TTL|TestSpecCache_Sweep" -v`
Expected: FAIL — `putWithTTL`, `sweep`, `startSweeper` are not defined.

**Step 3: Commit**

```bash
git add internal/mcpserver/input_test.go
git commit -m "test(mcp): add cache TTL and sweeper tests (red phase)"
```

---

### Task 4: Cache TTL Evolution — Implementation

**Files:**
- Modify: `internal/mcpserver/input.go`

**Context:** Evolve the cache to support TTL. Add `expiresAt` to `cacheEntry`, add `putWithTTL()`, add TTL checking in `get()`, add `sweep()`, add `startSweeper()`. Update `put()` to delegate to `putWithTTL()` with a default TTL. Update `makeCacheKey()` to support URLs. Update `resolve()` to pass per-type TTLs and cache URL results.

**Step 1: Modify `cacheEntry` struct**

In `internal/mcpserver/input.go`, change:

```go
// cacheEntry holds a cached parse result with its insertion order for LRU eviction.
type cacheEntry struct {
	result   *parser.ParseResult
	insertAt time.Time
}
```

to:

```go
// cacheEntry holds a cached parse result with LRU ordering and TTL expiry.
type cacheEntry struct {
	result    *parser.ParseResult
	insertAt  time.Time
	expiresAt time.Time
}
```

**Step 2: Add `putWithTTL`, update `get()` for TTL, add `sweep` and `startSweeper`**

Add `sync/atomic` and `context` to imports.

Update `get()` to check TTL:

```go
func (c *specCacheStore) get(key string) *parser.ParseResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[key]; ok {
		if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
			delete(c.entries, key)
			return nil
		}
		e.insertAt = time.Now()
		return e.result
	}
	return nil
}
```

Add `putWithTTL()`:

```go
// putWithTTL stores a result with a specific TTL, evicting the oldest entry if at capacity.
func (c *specCacheStore) putWithTTL(key string, result *parser.ParseResult, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	entry := &cacheEntry{result: result, insertAt: now, expiresAt: now.Add(ttl)}

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
```

Update the existing `put()` to delegate to `putWithTTL` with the file TTL (backward compat for any callers):

```go
func (c *specCacheStore) put(key string, result *parser.ParseResult) {
	c.putWithTTL(key, result, cfg.CacheFileTTL)
}
```

Add `sweep()` and `startSweeper()`:

```go
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
// It stops when ctx is cancelled.
func (c *specCacheStore) startSweeper(ctx context.Context, interval time.Duration) {
	var sweeping atomic.Bool
	go func() {
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
```

**Step 3: Update `makeCacheKey` to support URLs**

Change the `default` case in `makeCacheKey`:

```go
	case s.URL != "":
		return fmt.Sprintf("url:%s", s.URL)
	default:
		return "" // No input provided.
	}
```

**Step 4: Update `resolve()` to use per-type TTLs and handle cache-disabled**

```go
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

	// Determine cache key and TTL.
	key := makeCacheKey(s, extraOpts)
	var ttl time.Duration
	switch {
	case s.File != "":
		ttl = cfg.CacheFileTTL
	case s.URL != "":
		ttl = cfg.CacheURLTTL
	default:
		ttl = cfg.CacheContentTTL
	}

	// Check cache (skip if disabled).
	if cfg.CacheEnabled && key != "" {
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
	if cfg.CacheEnabled && key != "" {
		specCache.putWithTTL(key, result, ttl)
	}

	return result, nil
}
```

**Step 5: Update `specCache` var to use config**

Change:

```go
var specCache = &specCacheStore{
	entries: make(map[string]*cacheEntry),
	maxSize: 10,
}
```

to:

```go
var specCache = &specCacheStore{
	entries: make(map[string]*cacheEntry),
	maxSize: cfg.CacheMaxSize,
}
```

**Step 6: Run tests**

Run: `go test ./internal/mcpserver/ -run "TestSpecCache" -v`
Expected: ALL PASS (both old LRU tests and new TTL/sweeper tests).

**Step 7: Run gopls diagnostics**

Run: `go_diagnostics` on `internal/mcpserver/input.go`

**Step 8: Commit**

```bash
git add internal/mcpserver/input.go
git commit -m "feat(mcp): add TTL-based cache eviction, URL caching, and background sweeper"
```

---

### Task 5: Wire Config Into Server Startup

**Files:**
- Modify: `internal/mcpserver/server.go`

**Context:** Update `Run()` to initialize config, apply cache max-size, start the sweeper, and pass `ServerOptions.Instructions`. Remove the hardcoded `defaultWalkLimit`/`defaultDetailLimit` constants and use `cfg.*` instead.

**Step 1: Update `Run()` function**

In `internal/mcpserver/server.go`, change:

```go
func Run(ctx context.Context) error {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "oastools", Version: oastools.Version()},
		nil,
	)
	registerAllTools(server)
	return server.Run(ctx, &mcp.StdioTransport{})
}
```

to:

```go
func Run(ctx context.Context) error {
	// Apply config to cache.
	specCache.maxSize = cfg.CacheMaxSize
	if cfg.CacheEnabled {
		specCache.startSweeper(ctx, cfg.CacheSweepInterval)
	}

	server := mcp.NewServer(
		&mcp.Implementation{Name: "oastools", Version: oastools.Version()},
		&mcp.ServerOptions{
			Instructions: serverInstructions,
		},
	)
	registerAllTools(server)
	return server.Run(ctx, &mcp.StdioTransport{})
}
```

**Step 2: Add `serverInstructions` constant**

Add to `server.go` (after the import block):

```go
const serverInstructions = `oastools MCP server — validates, fixes, converts, diffs, joins, walks, and generates OpenAPI specs.

Configuration: All defaults are configurable via OASTOOLS_* environment variables set in your MCP client config. The Go MCP SDK does not support initializationOptions; use env vars instead.

Key settings:
- OASTOOLS_CACHE_FILE_TTL (default: 15m) — cache TTL for local file specs
- OASTOOLS_CACHE_URL_TTL (default: 5m) — cache TTL for URL-fetched specs
- OASTOOLS_CACHE_ENABLED (default: true) — disable spec caching entirely
- OASTOOLS_WALK_LIMIT (default: 100) — default result limit for walk tools
- OASTOOLS_WALK_DETAIL_LIMIT (default: 25) — default limit in detail mode
- OASTOOLS_VALIDATE_STRICT (default: false) — enable strict validation by default
- OASTOOLS_JOIN_PATH_STRATEGY — default path collision strategy for join
- OASTOOLS_JOIN_SCHEMA_STRATEGY — default schema collision strategy for join
- OASTOOLS_HTTP_TIMEOUT (default: 30s) — timeout for URL fetches

Caching: Parsed specs are cached per session. File entries use path+mtime as key (auto-invalidated on change). URL entries are cached with a shorter TTL. A background sweeper removes expired entries every 60s.`
```

**Step 3: Update `paginate()` and `detailLimit()` to use `cfg`**

Change:

```go
func paginate[T any](items []T, offset, limit int) []T {
	if limit <= 0 {
		limit = defaultWalkLimit
	}
```

to:

```go
func paginate[T any](items []T, offset, limit int) []T {
	if limit <= 0 {
		limit = cfg.WalkLimit
	}
```

Change:

```go
func detailLimit(limit int) int {
	if limit <= 0 {
		return defaultDetailLimit
	}
	return limit
}
```

to:

```go
func detailLimit(limit int) int {
	if limit <= 0 {
		return cfg.WalkDetailLimit
	}
	return limit
}
```

Update the `paginate` doc comment to reference `cfg.WalkLimit` instead of `defaultWalkLimit`.

**Step 4: Run tests**

Run: `go test ./internal/mcpserver/ -v`
Expected: ALL PASS.

**Step 5: Run gopls diagnostics**

Run: `go_diagnostics` on `internal/mcpserver/server.go`

**Step 6: Commit**

```bash
git add internal/mcpserver/server.go
git commit -m "feat(mcp): wire config into server startup, sweeper, and pagination"
```

---

### Task 6: Remove Hardcoded Constants

**Files:**
- Modify: `internal/mcpserver/tools_walk_operations.go`

**Context:** Remove the `defaultWalkLimit` and `defaultDetailLimit` constants since they're now replaced by `cfg.WalkLimit` and `cfg.WalkDetailLimit`. The constants are only defined in `tools_walk_operations.go` and referenced in `server.go` (already updated in Task 5).

**Step 1: Remove the constants**

In `internal/mcpserver/tools_walk_operations.go`, delete lines 53-57:

```go
const defaultWalkLimit = 100

// defaultDetailLimit is lower because detail mode returns full objects,
// which are significantly larger than summaries (2-10KB each).
const defaultDetailLimit = 25
```

**Step 2: Run tests**

Run: `go test ./internal/mcpserver/ -v`
Expected: ALL PASS.

**Step 3: Run gopls diagnostics**

Run: `go_diagnostics` on `internal/mcpserver/tools_walk_operations.go`

**Step 4: Commit**

```bash
git add internal/mcpserver/tools_walk_operations.go
git commit -m "refactor(mcp): remove hardcoded walk limit constants in favor of config"
```

---

### Task 7: Apply Config Defaults to Validate Tool

**Files:**
- Modify: `internal/mcpserver/tools_validate.go`
- Modify: `internal/mcpserver/tools_validate_test.go`

**Context:** Apply `cfg.ValidateStrict` and `cfg.ValidateNoWarnings` defaults when the per-call input fields are zero-value. Add a test verifying this behavior.

**Step 1: Write the failing test**

Add to `internal/mcpserver/tools_validate_test.go`:

```go
func TestHandleValidate_ConfigDefaults(t *testing.T) {
	specCache.reset()
	origCfg := cfg
	cfg = &serverConfig{
		CacheEnabled:       true,
		CacheMaxSize:       10,
		CacheFileTTL:       15 * time.Minute,
		CacheURLTTL:        5 * time.Minute,
		CacheContentTTL:    15 * time.Minute,
		CacheSweepInterval: 60 * time.Second,
		WalkLimit:          100,
		WalkDetailLimit:    25,
		ValidateStrict:     true,
		ValidateNoWarnings: true,
		HTTPTimeout:        30 * time.Second,
	}
	t.Cleanup(func() { cfg = origCfg })

	// Call validate without setting strict or no_warnings in input.
	// The config defaults (strict=true, no_warnings=true) should apply.
	input := validateInput{
		Spec: specInput{File: "../../testdata/petstore-3.0.yaml"},
	}
	_, output, err := handleValidate(context.Background(), nil, input)
	require.NoError(t, err)
	// With no_warnings=true from config, warnings should be suppressed.
	assert.Empty(t, output.Warnings)
	assert.Equal(t, 0, output.WarningCount)
}
```

**Note:** Add `"time"` and `"context"` to the test file imports if not already present, and `require` from testify.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcpserver/ -run TestHandleValidate_ConfigDefaults -v`
Expected: FAIL — config defaults are not applied yet.

**Step 3: Apply config defaults in `handleValidate`**

In `internal/mcpserver/tools_validate.go`, add config default application at the start of `handleValidate`:

```go
func handleValidate(_ context.Context, _ *mcp.CallToolRequest, input validateInput) (*mcp.CallToolResult, validateOutput, error) {
	// Apply config defaults.
	if !input.Strict {
		input.Strict = cfg.ValidateStrict
	}
	if !input.NoWarnings {
		input.NoWarnings = cfg.ValidateNoWarnings
	}

	parseResult, err := input.Spec.resolve()
	// ... rest unchanged
```

**Step 4: Run tests**

Run: `go test ./internal/mcpserver/ -run TestHandleValidate -v`
Expected: ALL PASS.

**Step 5: Run gopls diagnostics**

Run: `go_diagnostics` on `internal/mcpserver/tools_validate.go`

**Step 6: Commit**

```bash
git add internal/mcpserver/tools_validate.go internal/mcpserver/tools_validate_test.go
git commit -m "feat(mcp): apply config defaults to validate tool (strict, no_warnings)"
```

---

### Task 8: Apply Config Defaults to Join Tool

**Files:**
- Modify: `internal/mcpserver/tools_join.go`
- Modify: `internal/mcpserver/tools_join_test.go`

**Context:** Apply `cfg.JoinPathStrategy` and `cfg.JoinSchemaStrategy` defaults when the per-call input fields are empty.

**Step 1: Write the failing test**

Add to `internal/mcpserver/tools_join_test.go`:

```go
func TestHandleJoin_ConfigDefaults(t *testing.T) {
	specCache.reset()
	origCfg := cfg
	cfg = &serverConfig{
		CacheEnabled:       true,
		CacheMaxSize:       10,
		CacheFileTTL:       15 * time.Minute,
		CacheURLTTL:        5 * time.Minute,
		CacheContentTTL:    15 * time.Minute,
		CacheSweepInterval: 60 * time.Second,
		WalkLimit:          100,
		WalkDetailLimit:    25,
		JoinPathStrategy:   "accept-left",
		JoinSchemaStrategy: "accept-right",
		HTTPTimeout:        30 * time.Second,
	}
	t.Cleanup(func() { cfg = origCfg })

	// When input doesn't specify strategies, config defaults should be used.
	// We just verify the input is modified — the actual join behavior is tested elsewhere.
	input := joinInput{
		Specs: []specInput{
			{File: "../../testdata/petstore-3.0.yaml"},
			{File: "../../testdata/petstore-3.0.yaml"},
		},
	}

	// The handler should apply config defaults to empty strategy fields.
	_, output, _ := handleJoin(context.Background(), nil, input)
	// If join succeeds without error, the strategies were accepted.
	assert.Greater(t, output.PathCount, 0)
}
```

**Step 2: Apply config defaults in `handleJoin`**

In `internal/mcpserver/tools_join.go`, add config default application at the start of `handleJoin`:

```go
func handleJoin(_ context.Context, _ *mcp.CallToolRequest, input joinInput) (*mcp.CallToolResult, joinOutput, error) {
	// Apply config defaults.
	if input.PathStrategy == "" {
		input.PathStrategy = cfg.JoinPathStrategy
	}
	if input.SchemaStrategy == "" {
		input.SchemaStrategy = cfg.JoinSchemaStrategy
	}

	if len(input.Specs) < 2 {
	// ... rest unchanged
```

**Step 3: Run tests**

Run: `go test ./internal/mcpserver/ -run TestHandleJoin -v`
Expected: ALL PASS.

**Step 4: Run gopls diagnostics and commit**

```bash
git add internal/mcpserver/tools_join.go internal/mcpserver/tools_join_test.go
git commit -m "feat(mcp): apply config defaults to join tool (path/schema strategies)"
```

---

### Task 9: Update Tool Descriptions

**Files:**
- Modify: `internal/mcpserver/server.go`

**Context:** Add mentions of configurable defaults to tool `Description` strings so LLM agents know about them.

**Step 1: Update descriptions**

In `internal/mcpserver/server.go`, update these tool descriptions in `registerAllTools`:

- **validate**: Append ` Strict mode and warning suppression defaults are configurable via OASTOOLS_VALIDATE_STRICT and OASTOOLS_VALIDATE_NO_WARNINGS env vars.`
- **join**: Append ` Default collision strategies are configurable via OASTOOLS_JOIN_PATH_STRATEGY and OASTOOLS_JOIN_SCHEMA_STRATEGY env vars.`
- **walk_operations** (and representative for all walk tools, or just the first one since the Instructions cover this): Append ` Default limit is configurable via OASTOOLS_WALK_LIMIT (default 100, 25 in detail mode).`

Only update a few representative tools — the `Instructions` field covers all configuration comprehensively. Avoid making every description verbose.

**Step 2: Run tests**

Run: `go test ./internal/mcpserver/ -v`
Expected: ALL PASS.

**Step 3: Commit**

```bash
git add internal/mcpserver/server.go
git commit -m "docs(mcp): mention configurable defaults in tool descriptions"
```

---

### Task 10: Update Documentation Site

**Files:**
- Modify: `docs/mcp-server.md`

**Context:** Add a Configuration section to the MCP server docs page with the full env var reference table and MCP client configuration examples.

**Step 1: Update `docs/mcp-server.md`**

Add a new `## Configuration` section after the `## Quick Start` section. Include:

1. An intro paragraph explaining that defaults are configurable via `OASTOOLS_*` environment variables
2. A note about why env vars are used instead of MCP `initializationOptions` (Go SDK limitation)
3. The full env var reference table from the design doc
4. Configuration examples for Claude Code (`.mcp.json` with `env` field) and Claude Desktop (`claude_desktop_config.json`)

Update the **Spec Caching** section:
- Change "URL: Not cached" to "URL: Cached with TTL (default 5m)"
- Add mention of TTL for files (default 15m)
- Add mention of background sweeper
- Update max entries description to mention configurability

**Step 2: Verify docs build** (if applicable)

Run: `make docs` or verify the markdown renders correctly.

**Step 3: Commit**

```bash
git add docs/mcp-server.md
git commit -m "docs(mcp): add configuration reference and update caching docs"
```

---

### Task 11: Update Project Notes

**Files:**
- Modify: `CLAUDE.md`

**Context:** Add a note about the env var config pattern and the Go SDK limitation so agents working on oastools itself understand the architecture.

**Step 1: Add MCP config note**

In `CLAUDE.md`, add to the Key Patterns section:

```markdown
- **MCP config via env vars**: The MCP server reads `OASTOOLS_*` env vars for configuration (cache TTLs, walk limits, join strategies, etc.). The Go MCP SDK doesn't support `initializationOptions`, so env vars are used instead. MCP clients set these via their `env` field in server config.
```

**Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add MCP env var config pattern to CLAUDE.md"
```

---

### Task 12: Quality Gate

**Context:** Run the full validation suite to verify nothing is broken.

**Step 1: Run `make check`**

Run: `make check`
Expected: ALL PASS — no lint errors, no test failures, no formatting issues.

**Step 2: Run gopls diagnostics on all changed files**

Run: `go_diagnostics` on:
- `internal/mcpserver/config.go`
- `internal/mcpserver/input.go`
- `internal/mcpserver/server.go`
- `internal/mcpserver/tools_validate.go`
- `internal/mcpserver/tools_join.go`
- `internal/mcpserver/tools_walk_operations.go`

**Step 3: Fix any issues found**

If any issues are found, fix them and re-run the quality gate.

**Step 4: Final commit if any fixes were needed**

```bash
git add -A
git commit -m "fix(mcp): address quality gate findings"
```
