# MCP Server Cache TTL & Configuration Design

**Date**: 2026-02-19
**Status**: Approved

## Summary

Extend the MCP server's spec cache with TTL-based eviction and active background sweeping, enable caching of URL-fetched specs (with shorter TTL), and introduce a configuration system via environment variables for all tool defaults.

## Motivation

The current cache has three limitations:
1. **No TTL** — cached entries persist for the session lifetime, consuming memory for specs that may never be accessed again
2. **URLs excluded** — every URL fetch re-downloads and re-parses, even for specs that haven't changed
3. **No configuration** — all defaults (walk limits, join strategies, validation strictness, cache size, HTTP timeout) are hardcoded with no way for users to customize

## Decisions

### Configuration via Environment Variables (not MCP initializationOptions)

The MCP protocol spec defines `initializationOptions` in the `initialize` request for server configuration. However, **the Go MCP SDK (v1.3.0) does not expose this field** in its `InitializeParams` struct — it only has `Meta`, `Capabilities`, `ClientInfo`, and `ProtocolVersion`.

Environment variables are the chosen alternative because:
- MCP clients (Claude Code, Cursor, etc.) support an `env` field in server configuration
- They're simple to parse with no external dependencies
- They're set once and persist across sessions
- They work universally across all MCP client implementations

### Active TTL Eviction (not passive)

A background sweeper goroutine runs on a 60-second ticker interval. If a sweep is already in progress when the next tick fires, the tick is skipped via an `atomic.Bool` guard (avoids blocking, zero performance cost).

The sweeper exits when the server's context is cancelled — no explicit stop/cleanup needed.

### Per-call Parameters Override Config Defaults

Precedence: hardcoded default -> env var config -> per-call parameter.

For bool fields (e.g., `strict`), the zero value (`false`) is treated as "not provided," meaning the config default applies. This is acceptable because:
- LLMs omit params they don't want to set (standard MCP tool behavior)
- The config represents the user's own preference

## Configuration Reference

| Environment Variable | Type | Default | Purpose |
|---|---|---|---|
| `OASTOOLS_CACHE_ENABLED` | bool | `true` | Enable/disable spec cache |
| `OASTOOLS_CACHE_MAX_SIZE` | int | `10` | Maximum cached entries |
| `OASTOOLS_CACHE_FILE_TTL` | duration | `15m` | TTL for file-based entries |
| `OASTOOLS_CACHE_URL_TTL` | duration | `5m` | TTL for URL-based entries |
| `OASTOOLS_CACHE_CONTENT_TTL` | duration | `15m` | TTL for inline content entries |
| `OASTOOLS_CACHE_SWEEP_INTERVAL` | duration | `60s` | Background sweeper interval |
| `OASTOOLS_WALK_LIMIT` | int | `100` | Default walk result limit |
| `OASTOOLS_WALK_DETAIL_LIMIT` | int | `25` | Default detail mode limit |
| `OASTOOLS_JOIN_PATH_STRATEGY` | string | `""` | Default join path collision strategy |
| `OASTOOLS_JOIN_SCHEMA_STRATEGY` | string | `""` | Default join schema collision strategy |
| `OASTOOLS_VALIDATE_STRICT` | bool | `false` | Default strict validation mode |
| `OASTOOLS_VALIDATE_NO_WARNINGS` | bool | `false` | Default warning suppression |
| `OASTOOLS_HTTP_TIMEOUT` | duration | `30s` | HTTP fetch timeout |
| `OASTOOLS_HTTP_USER_AGENT` | string | (auto) | Custom user-agent string |

## Cache Architecture

### Cache Entry

```go
type cacheEntry struct {
    result    *parser.ParseResult
    insertAt  time.Time
    expiresAt time.Time
}
```

### Cache Keys

| Input Type | Key Format | Staleness Detection |
|---|---|---|
| File | `file:{absolutePath}:{modTimeNano}` | Mod-time in key + TTL |
| Content | `content:{sha256Hash}` | Content hash + TTL |
| URL | `url:{normalizedURL}` | TTL only |

### TTL Behavior

- **`get()`**: If `time.Now().After(e.expiresAt)`, the entry is deleted and treated as a miss
- **`put()`**: Caller passes TTL; entry sets `expiresAt = time.Now().Add(ttl)`
- **TTL of 0 or negative**: Uses the default value (not "never expire")

### Background Sweeper

```go
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
                    continue // previous sweep still running, skip
                }
                c.sweep()
                sweeping.Store(false)
            }
        }
    }()
}
```

### Cache Disable

When `cfg.CacheEnabled == false`, `resolve()` skips cache lookup and storage. The cache struct still exists but is unused.

## Tool Handler Integration

A package-level `var cfg *serverConfig` (matching the existing `var specCache` pattern) is initialized in `Run()` before `registerAllTools()`.

### Changes Per Tool

| Tool | Change |
|---|---|
| `walk_*` tools | Replace `defaultWalkLimit` / `defaultDetailLimit` constants with `cfg.WalkLimit` / `cfg.WalkDetailLimit` |
| `join` | Use `cfg.JoinPathStrategy` / `cfg.JoinSchemaStrategy` as defaults when input is empty |
| `validate` | Use `cfg.ValidateStrict` / `cfg.ValidateNoWarnings` as defaults when input is zero-value |
| `resolve()` | Pass TTL based on input type using `cfg.CacheFileTTL` / `cfg.CacheURLTTL` / `cfg.CacheContentTTL` |
| All with pagination | Use `cfg.WalkLimit` instead of hardcoded 100 |

## Documentation

### MCP Server Instructions

Set `ServerOptions.Instructions` to include:
- Available configuration via environment variables
- Default values and their meaning
- The Go SDK limitation (no `initializationOptions` support)
- Cache behavior explanation (TTL, URL caching, file mod-time invalidation)

### Tool Description Updates

Each tool's `Description` string mentions configurable defaults:
- Walk tools: "...default limit is 100 (configurable via OASTOOLS_WALK_LIMIT)"
- Join: "...default strategies are configurable via OASTOOLS_JOIN_PATH_STRATEGY / OASTOOLS_JOIN_SCHEMA_STRATEGY"
- Validate: "...strict mode default configurable via OASTOOLS_VALIDATE_STRICT"

### Documentation Site

Update `docs/mcp-server.md` with a configuration reference table and examples for common MCP clients (Claude Desktop, Claude Code `.mcp.json`, Cursor).

### Project Notes

Add a note to CLAUDE.md / AGENTS.md explaining the env var config pattern and why `initializationOptions` isn't used.

## Error Handling

- **Invalid env var values**: Log warning via `slog`, use hardcoded default. Never crash the server.
- **TTL of 0 or negative**: Use default (not "never expire")
- **Cache disabled**: `resolve()` skips cache entirely. No runtime errors.

## Testing Strategy

| What | How |
|---|---|
| Config loading | Unit tests with `t.Setenv()` for each env var, invalid values, missing values |
| Cache TTL | Unit tests with short TTLs (1ms), verify expiry behavior |
| Background sweeper | Test with short interval (10ms), verify cleanup |
| URL caching | Integration test: resolve URL twice, verify cache hit |
| Tool defaults | Unit tests per tool: verify config defaults applied when input is zero-value |
| Sweeper skip | Test atomic bool guard: verify concurrent sweeps don't block |

## Backward Compatibility

No breaking changes:
- All env vars are optional with defaults matching current behavior
- Existing tool input/output schemas are unchanged
- Cache behavior for files/content is preserved; URLs gain caching (strictly better)
- `maxSize` default stays at 10

## Files Changed

| File | Change |
|---|---|
| `internal/mcpserver/config.go` | **New** — `serverConfig` struct, `loadConfig()`, env parsing helpers |
| `internal/mcpserver/config_test.go` | **New** — config loading tests |
| `internal/mcpserver/input.go` | Evolve cache: add TTL, URL caching, sweeper, cache-disable check |
| `internal/mcpserver/input_test.go` | New tests for TTL, URL caching, sweeper |
| `internal/mcpserver/server.go` | Call `loadConfig()`, start sweeper, set `Instructions`, update tool descriptions |
| `internal/mcpserver/tools_walk_operations.go` | Replace constants with `cfg.*` |
| `internal/mcpserver/tools_walk_*.go` | Replace constants with `cfg.*` |
| `internal/mcpserver/tools_join.go` | Apply config defaults for strategies |
| `internal/mcpserver/tools_validate.go` | Apply config defaults for strict/no_warnings |
| `docs/mcp-server.md` | Configuration reference and examples |
| `CLAUDE.md` or `AGENTS.md` | Note about env var config pattern |
