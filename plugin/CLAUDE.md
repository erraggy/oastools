# oastools MCP Plugin

You have access to the oastools MCP server, which provides 17 tools for working with OpenAPI Specification (OAS 2.0-3.2) documents.

## Prerequisites

This plugin requires the `oastools` CLI binary on your PATH. The MCP server runs as `oastools mcp`.

**Install via Homebrew:**

```bash
brew install erraggy/oastools/oastools
```

**Or from source (requires Go 1.24+):**

```bash
go install github.com/erraggy/oastools/cmd/oastools@latest
```

**Check version:** `oastools version`

If MCP tools are unavailable or returning errors, verify the binary is installed and up to date. New tool parameters (like `output` on `fix`) require the corresponding binary version.

**Version mismatch detection:** On session start, the plugin checks whether the installed `oastools` binary version matches the plugin version. If a mismatch is detected, a warning is surfaced so you can update accordingly. Dev and unknown builds skip this check.

## Available Tools

**Core (9):** `validate`, `parse`, `fix`, `convert`, `diff`, `join`, `overlay_apply`, `overlay_validate`, `generate`

**Walk (8):** `walk_operations`, `walk_schemas`, `walk_parameters`, `walk_responses`, `walk_headers`, `walk_security`, `walk_paths`, `walk_refs`

## Input Model

Every tool accepts a spec via one of three methods inside a `spec` object:

- `file` -- Path to an OAS file on disk (preferred for large specs)
- `url` -- URL to fetch an OAS document from
- `content` -- Inline OAS content as a string (JSON or YAML)

Exactly one must be provided. Example: `{"spec": {"file": "openapi.yaml"}}`

Special cases:

- `diff` uses `base` and `revision` instead of `spec`
- `join` uses a `specs` array
- `overlay_apply` uses `spec` and `overlay`

## Configuration

The MCP server reads `OASTOOLS_*` environment variables for default behavior. These are set by the user in their MCP client config (e.g., the `env` field in `.mcp.json`). You don't need to configure these yourself — just be aware that defaults may differ from the documented values.

**Key settings that affect tool behavior:**

| Variable | Default | Effect |
|----------|---------|--------|
| `OASTOOLS_VALIDATE_STRICT` | `false` | When `true`, validate uses strict mode unless the call explicitly sets `strict: false` |
| `OASTOOLS_VALIDATE_NO_WARNINGS` | `false` | When `true`, validate suppresses warnings unless the call explicitly sets `no_warnings: false` |
| `OASTOOLS_WALK_LIMIT` | `100` | Default result limit for walk tools (override with `limit` per call) |
| `OASTOOLS_WALK_DETAIL_LIMIT` | `25` | Default limit in detail mode |
| `OASTOOLS_JOIN_PATH_STRATEGY` | *(none)* | Default path collision strategy for join |
| `OASTOOLS_JOIN_SCHEMA_STRATEGY` | *(none)* | Default schema collision strategy for join |
| `OASTOOLS_CACHE_ENABLED` | `true` | Disable spec caching entirely with `false` |
| `OASTOOLS_CACHE_MAX_SIZE` | `10` | Maximum number of parsed specs held in the LRU cache |
| `OASTOOLS_CACHE_FILE_TTL` | `15m` | How long file-based specs are cached |
| `OASTOOLS_CACHE_URL_TTL` | `5m` | How long URL-fetched specs are cached |
| `OASTOOLS_CACHE_CONTENT_TTL` | `15m` | How long inline content specs are cached |
| `OASTOOLS_CACHE_SWEEP_INTERVAL` | `60s` | How often the background cache sweeper runs |
| `OASTOOLS_ALLOW_PRIVATE_IPS` | `false` | Allow URL resolution to private/loopback/link-local IPs (for internal APIs) |
| `OASTOOLS_MAX_INLINE_SIZE` | `10485760` | Max inline content size in bytes (10 MiB) |
| `OASTOOLS_MAX_LIMIT` | `1000` | Max pagination limit for walk tools |
| `OASTOOLS_MAX_JOIN_SPECS` | `20` | Max number of specs in a join operation |

Tool-level parameters (e.g., `strict`, `no_warnings`, `limit`) always override the env var defaults when explicitly provided.

## Best Practices

1. **Prefer `file` over `content`** for specs already on disk. Avoids copying large documents into tool calls.
2. **Explore before modifying.** Use `parse` for a high-level overview and `walk_*` tools to drill into specific parts before running `fix` or `convert`.
3. **Validate after changes.** Always run `validate` after `fix`, `convert`, or `overlay_apply` to confirm the result is valid.
4. **Use `dry_run` for fix.** Preview what the `fix` tool will change before applying.
5. 🔍 **Filter before paging.** All walk tools and `validate`, `fix`, `diff` support filters that reduce result size more effectively than pagination. Key filters:
   - `walk_*`: `tag`, `method`, `path`, `status`, `name`, `type`, `component` (schemas only)
   - `validate`: `no_warnings: true` — suppresses warnings for error-focused triage
   - `diff`: `breaking_only: true` — shows only breaking changes (usually fewest and most important)
   - Use `detail: true` only after filtering to specific items — full objects can be very large
6. **Check breaking changes.** When diffing specs, use `breaking_only: true` to focus on changes that will break API consumers.
7. 📄 **Page through large results.** Tools that return arrays (`validate`, `fix`, `diff`, `walk_*`) support `offset` and `limit` params (default limit: 100). When `returned` is less than the total count, use `offset` to page through. Prefer filtering over paging when possible.
8. 📊 **Aggregate with `group_by`.** Most walk tools (`walk_operations`, `walk_schemas`, `walk_parameters`, `walk_responses`, `walk_headers`, `walk_paths`, `walk_refs`) support a `group_by` parameter (`walk_security` does not, since specs rarely have more than 3 schemes) that returns `{key, count}` groups instead of individual items. `walk_paths` supports `group_by: "segment"` (group by first path segment) and `walk_refs` supports `group_by: "node_type"` (group by reference target type). Use this for distribution questions ("how many operations per tag?") instead of paging through all results.
9. 🔍 **Parameter filters auto-resolve refs.** `walk_parameters` automatically resolves `$ref` parameters when filtering by `in` or `name`. No need to set `resolve_refs: true` manually for these filters.
10. 📏 **Detail mode has a lower default limit (25).** `detail=true` defaults to 25 results instead of 100 to keep output manageable. Set `limit` explicitly for more results.

## 💾 Persisting Results

Tools that transform documents (`fix`, `convert`, `join`, `overlay_apply`) do **not** modify input files in-place. The `file` input is read-only. To persist results, use the `output` parameter:

```json
{"spec": {"file": "openapi.yaml"}, "output": "/tmp/fixed.yaml"}
```

The response includes `written_to` confirming the file path. Both `output` and `include_document` can be used together when you need to write to disk AND inspect the result inline.

When `output` is omitted, the document is returned inline (automatically for `convert`/`join`/`overlay_apply`; only when `include_document: true` for `fix`). **For large specs, prefer `output` over inline** to avoid excessive token usage.

## 🔗 Pipelining Tools

Chain tools by writing intermediate results to files and referencing them in subsequent calls:

**Fix → Validate:**

```
fix(spec.file="api.yaml", output="/tmp/api-fixed.yaml")
validate(spec.file="/tmp/api-fixed.yaml")
```

**Fix → Convert → Validate:**

```
fix(spec.file="api.yaml", output="/tmp/api-fixed.yaml")
convert(spec.file="/tmp/api-fixed.yaml", target="3.1", output="/tmp/api-3.1.yaml")
validate(spec.file="/tmp/api-3.1.yaml")
```

**Fix → Generate:**

```
fix(spec.file="api.yaml", output="/tmp/api-fixed.yaml")
generate(spec.file="/tmp/api-fixed.yaml", client=true, output_dir="./generated")
```

Use a temp directory for intermediate files (e.g., `/tmp/`) and copy the final result to the desired location when the pipeline succeeds.

## Common Workflows

**Validate and fix:**

1. `validate` the spec to find issues
2. `fix` with `dry_run: true` to preview fixes
3. `fix` with `output` to apply fixes and persist the result
4. `validate` the output file to confirm the spec is now valid

**Explore an API:**

1. `parse` to get title, version, path/schema/operation counts
2. `walk_operations` to list endpoints (filter by tag, method, or path)
3. `walk_schemas` to list data models
4. `walk_operations` with `detail: true` on specific endpoints for full request/response details

**Compare API versions:**

1. `diff` with both specs to see all changes
2. Review breaking changes and their severity
3. Use `walk_operations` on the revision to understand new/modified endpoints

**Explore a large API (100+ operations):**

1. 📊 `parse` to get counts and tag list
2. 📊 `walk_operations` with `group_by: "tag"` — operation count per tag at a glance
3. 🏷️ `walk_operations` with `tag` filter — work through one tag at a time
4. 📋 `walk_schemas` with `component: true` — named schemas only (skip inline)
5. 🔗 `walk_refs` to find most-referenced schemas and trace usage patterns
6. 🔍 Drill into specifics with `operation_id` or `path` + `detail: true`
7. ✅ Use `validate` with `no_warnings: true` for error-focused triage

## Troubleshooting

### Private/internal URLs blocked

When `spec.url` returns an error containing `"blocked request to private/loopback IP"` or `"redirect to private/loopback IP blocked"`, this is SSRF protection preventing URL resolution to private, loopback, or link-local IP addresses. The error message itself includes the fix (`set OASTOOLS_ALLOW_PRIVATE_IPS=true to allow`). This is expected for internal/company APIs.

**Fix:** Add `OASTOOLS_ALLOW_PRIVATE_IPS=true` to the MCP server's `env` config:

```json
{
  "mcpServers": {
    "oastools": {
      "command": "oastools",
      "args": ["mcp"],
      "env": {
        "OASTOOLS_ALLOW_PRIVATE_IPS": "true"
      }
    }
  }
}
```

Do NOT work around this by downloading the spec with `curl`, `fetch`, or other tools — use the proper env var so the MCP tools can fetch and cache the spec directly.

### Inline content too large

When content exceeds 10 MiB, the server returns an error containing `"use file input instead, or set OASTOOLS_MAX_INLINE_SIZE to increase"`. Follow the error's guidance: use `file` input instead of `content`. If the spec is only available as a string (e.g., from an API response), write it to a temp file first, then reference that file.

### Env var changes require restart

The MCP server reads environment variables at startup. After changing `env` values in `.mcp.json` (or project settings), the MCP server must be restarted. In Claude Code: use `/mcp` and restart the server, or restart the session.
