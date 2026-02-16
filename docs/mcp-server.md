# MCP Server

oastools includes a built-in [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server that exposes all capabilities as tools over stdio transport. This allows LLM agents and AI-powered editors to validate, fix, convert, diff, join, parse, overlay, generate, and walk OpenAPI specs programmatically.

## Prerequisites

The MCP server is built into the `oastools` binary. Install it first:

```bash
brew install erraggy/oastools/oastools                   # Homebrew (macOS/Linux)
go install github.com/erraggy/oastools/cmd/oastools@latest  # Go install (requires Go 1.24+)
```

Or download a pre-built binary from the [Releases page](https://github.com/erraggy/oastools/releases/latest).

## Quick Start

```bash
oastools mcp
```

The server communicates over stdin/stdout using JSON-RPC, following the MCP specification. It is designed to be launched by an MCP client (such as Claude Code, Cursor, or any MCP-compatible host) rather than used interactively.

### Claude Code

The easiest way to use the MCP server with Claude Code is to install the [oastools plugin](claude-code-plugin.md), which configures the server and registers guided skills automatically. Run inside Claude Code:

```text
/plugin marketplace add erraggy/oastools
/plugin install oastools
```

Alternatively, add it manually to your project's `.mcp.json`:

```json
{
  "mcpServers": {
    "oastools": {
      "type": "stdio",
      "command": "oastools",
      "args": ["mcp"]
    }
  }
}
```

### Other MCP Clients

Any MCP-compatible client can launch the server. The transport is stdio — the client spawns `oastools mcp` as a subprocess and communicates via stdin/stdout.

---

## Tools (17)

The server registers 17 tools organized into two categories: 9 core tools and 8 walk tools.

### Core Tools

| Tool | Description |
|------|-------------|
| `validate` | Validate an OAS document against its version schema |
| `parse` | Parse an OAS document and return a structural summary |
| `fix` | Auto-fix common OAS issues (duplicate IDs, missing parameters, etc.) |
| `convert` | Convert between OAS versions (2.0, 3.0, 3.1) |
| `diff` | Compare two specs and detect breaking changes |
| `join` | Merge multiple specs with collision strategies |
| `overlay_apply` | Apply an Overlay document to a spec |
| `overlay_validate` | Validate an Overlay document structure |
| `generate` | Generate Go client/server/types from a spec |

### Walk Tools

| Tool | Description |
|------|-------------|
| `walk_operations` | Query operations by method, path, tag, operationId, or deprecated status |
| `walk_schemas` | Query schemas by name, type, or component/inline location |
| `walk_parameters` | Query parameters by location, name, path, or method |
| `walk_responses` | Query responses by status code, path, or method |
| `walk_headers` | Query headers by name, path, method, status code, or component location |
| `walk_security` | Query security schemes by name or type |
| `walk_paths` | Query path items by path pattern (supports `*` glob) |
| `walk_refs` | Query `$ref` references by target pattern or node type |

---

## Input Model

Every tool accepts an OpenAPI spec through a `spec` object with three mutually exclusive input modes:

| Field | Description |
|-------|-------------|
| `file` | Path to an OAS file on disk |
| `url` | URL to fetch an OAS document from |
| `content` | Inline OAS document content (JSON or YAML string) |

Exactly one must be provided.

```json
{"spec": {"file": "openapi.yaml"}}
```

```json
{"spec": {"url": "https://example.com/api/openapi.yaml"}}
```

```json
{"spec": {"content": "{\"openapi\": \"3.1.0\", ...}"}}
```

### Special Input Patterns

Some tools use different input structures:

| Tool | Input Pattern |
|------|---------------|
| `diff` | `base` + `revision` (two separate spec objects) |
| `join` | `specs` array (minimum 2) |
| `overlay_apply` | `spec` + `overlay` (spec object + overlay object) |
| All others | Single `spec` object |

---

## Tool Reference

### validate

Validate an OpenAPI spec against its declared version schema.

**Input:**

| Field | Type | Description |
|-------|------|-------------|
| `spec` | object | The OAS document (file/url/content) |
| `strict` | boolean | Enable stricter validation beyond spec requirements |
| `no_warnings` | boolean | Suppress warning messages |

**Output:**

| Field | Type | Description |
|-------|------|-------------|
| `valid` | boolean | Whether the spec is valid |
| `version` | string | Detected OAS version |
| `error_count` | number | Number of validation errors |
| `warning_count` | number | Number of warnings |
| `errors` | array | Error details (path, message, severity) |
| `warnings` | array | Warning details |

**Example:**

```json
{
  "spec": {"file": "openapi.yaml"},
  "strict": true
}
```

---

### parse

Parse an OAS document and return a structural summary.

**Input:**

| Field | Type | Description |
|-------|------|-------------|
| `spec` | object | The OAS document |
| `full` | boolean | Return the full parsed document instead of a summary |
| `resolve_refs` | boolean | Resolve `$ref` pointers before returning |

**Output (summary mode):**

| Field | Type | Description |
|-------|------|-------------|
| `version` | string | OAS version |
| `title` | string | API title |
| `description` | string | API description |
| `path_count` | number | Number of paths |
| `operation_count` | number | Number of operations |
| `schema_count` | number | Number of schemas |
| `servers` | array | Server URLs |
| `tags` | array | Tag names |

---

### fix

Auto-fix common issues in an OAS document.

**Input:**

| Field | Type | Description |
|-------|------|-------------|
| `spec` | object | The OAS document |
| `dry_run` | boolean | Preview fixes without applying |
| `include_document` | boolean | Include the fixed document in the response |
| `output` | string | File path to write the fixed document |
| `fix_duplicate_operationids` | boolean | Deduplicate operationId values |
| `fix_schema_names` | boolean | Rename generic schema names |
| `prune` | boolean | Remove empty paths and unused schemas |
| `stub_missing_refs` | boolean | Create stub schemas for missing `$ref` targets |

**Output:**

| Field | Type | Description |
|-------|------|-------------|
| `version` | string | OAS version |
| `fix_count` | number | Number of fixes applied |
| `fixes` | array | Fix details (type, path, description) |
| `written_to` | string | File path where the fixed document was written |
| `document` | string | Fixed document (when `include_document` is true) |

---

### convert

Convert an OAS document between versions.

**Input:**

| Field | Type | Description |
|-------|------|-------------|
| `spec` | object | The OAS document |
| `target` | string | Target version (`2.0`, `3.0`, or `3.1`) — **required** |
| `output` | string | File path to write the converted document |

**Output:**

| Field | Type | Description |
|-------|------|-------------|
| `source_version` | string | Original OAS version |
| `target_version` | string | Target OAS version |
| `success` | boolean | Whether conversion succeeded |
| `issue_count` | number | Number of conversion issues |
| `issues` | array | Issue details (severity, path, message) |
| `document` | string | Converted document (when no output file specified) |

---

### diff

Compare two OAS documents and report differences.

**Input:**

| Field | Type | Description |
|-------|------|-------------|
| `base` | object | The base/original OAS document |
| `revision` | object | The revised OAS document |
| `breaking_only` | boolean | Only show breaking changes |
| `no_info` | boolean | Suppress informational changes |

**Output:**

| Field | Type | Description |
|-------|------|-------------|
| `total_changes` | number | Number of changes displayed |
| `breaking_count` | number | Number of breaking changes |
| `warning_count` | number | Number of warnings |
| `info_count` | number | Number of informational changes |
| `changes` | array | Change details (severity, type, path, message) |
| `summary` | string | Human-readable summary |

---

### join

Merge multiple OAS documents into one.

**Input:**

| Field | Type | Description |
|-------|------|-------------|
| `specs` | array | Array of spec objects (minimum 2) |
| `path_strategy` | string | Collision strategy for paths |
| `schema_strategy` | string | Collision strategy for schemas |
| `output` | string | File path to write the merged document |

Collision strategies: `accept-left`, `accept-right`, `fail`

---

### overlay_apply

Apply an Overlay document to an OAS spec.

**Input:**

| Field | Type | Description |
|-------|------|-------------|
| `spec` | object | The OAS document |
| `overlay` | object | The Overlay document (file/url/content) |
| `dry_run` | boolean | Preview changes without applying |
| `strict` | boolean | Fail if any target matches nothing |
| `output` | string | File path to write the result |

---

### overlay_validate

Validate an Overlay document structure.

**Input:**

| Field | Type | Description |
|-------|------|-------------|
| `spec` | object | The Overlay document (file/url/content) |

**Output:**

| Field | Type | Description |
|-------|------|-------------|
| `valid` | boolean | Whether the overlay is valid |
| `version` | string | Overlay specification version |
| `title` | string | Overlay title |
| `action_count` | number | Number of actions |
| `errors` | array | Validation errors |

---

### generate

Generate Go code from an OAS document.

**Input:**

| Field | Type | Description |
|-------|------|-------------|
| `spec` | object | The OAS document |
| `output_dir` | string | Directory for generated files — **required** |
| `package_name` | string | Go package name (default: `api`) |
| `client` | boolean | Generate HTTP client |
| `server` | boolean | Generate server interface |
| `types` | boolean | Generate type definitions (default: true) |

---

### Walk Tools

All walk tools share common input fields:

| Field | Type | Description |
|-------|------|-------------|
| `spec` | object | The OAS document |
| `detail` | boolean | Return full objects instead of summaries |
| `resolve_refs` | boolean | Resolve `$ref` pointers before output |
| `limit` | number | Max results to return (default: 100; 25 in detail mode) |
| `offset` | number | Skip the first N results (for pagination) |
| `group_by` | string | Group results and return `{key, count}` aggregates instead of individual items |
| `extension` | string | Filter by extension (e.g., `x-internal=true`) |

**Tool-specific filters:**

| Tool | Filter Fields | `group_by` Values |
|------|---------------|-------------------|
| `walk_operations` | `method`, `path`, `tag`, `operation_id`, `deprecated` | `tag`, `method` |
| `walk_schemas` | `name`, `type`, `component`, `inline` | `type`, `location` |
| `walk_parameters` | `name`, `in`, `path`, `method` | `location`, `name` |
| `walk_responses` | `status`, `path`, `method` | `status_code`, `method` |
| `walk_headers` | `name`, `path`, `method`, `status`, `component` | `name`, `status_code` |
| `walk_security` | `name`, `type` | — |
| `walk_paths` | `path` (supports `*` glob) | `segment` |
| `walk_refs` | `target`, `node_type` | `node_type` |

---

## Spec Caching

The MCP server maintains a session-scoped cache of parsed specs. Once a spec is parsed, subsequent tool calls referencing the same spec reuse the cached parse result instead of re-parsing.

**Cache keys:**

| Input Mode | Key Strategy | Invalidation |
|------------|-------------|--------------|
| `file` | Absolute path + file modification time | Automatic on file change (mtime) |
| `content` | SHA-256 hash of the content string | Automatic (different content = different key) |
| `url` | Not cached | N/A (remote content may change between calls) |

The cache holds a maximum of 10 entries and uses LRU (least recently used) eviction when at capacity.

**Which tools use the cache:**

- Most tools use the cache: `parse`, `validate`, all `walk_*` tools, `generate`, `diff`, `join`, `overlay_apply`
- `fix` and `convert` bypass the cache and parse independently (they use their own internal parsing pipelines)

---

## Behavioral Notes

### Parse Truncation

In summary mode (the default), `parse` truncates long description fields to 200 characters. This reduces token usage for LLM consumers. Set `full=true` to retrieve complete, untruncated descriptions.

### walk_refs Count Semantics

In `walk_refs` summary mode (the default), the `total` field represents the number of unique `$ref` targets in the spec, not the total number of `$ref` occurrences. A single target referenced from 5 locations counts as 1 toward `total`.

In `detail` mode and `group_by` mode, `total` counts individual `$ref` occurrences (a target referenced 5 times counts as 5). When using `group_by=node_type`, each group's `count` is the number of `$ref` occurrences pointing to targets of that node type.

### Group Key Labels

Some walk tools use labeled group keys to represent edge cases when using `group_by` aggregation:

| Tool | `group_by` | Label | Meaning |
|------|-----------|-------|---------|
| `walk_schemas` | `type` | `(untyped)` | Schemas without an explicit `type` field -- typically compositions (`allOf`/`anyOf`/`oneOf`) or `$ref` wrappers |
| `walk_parameters` | `location` | `(ref)` | Parameters defined as a `$ref` that have not been resolved -- the `in` field is not available until the reference is followed |
| `walk_responses` | `status_code` | `(component)` | Component-level responses (defined in `components/responses`) that are not associated with a specific operation or status code |
| `walk_responses` | `method` | `(component)` | Component-level responses that are not associated with any HTTP method |

These labels appear as the `key` field in the `groups` array when the corresponding condition is met.

### Detail Mode Defaults

Walk tools in `detail=true` mode use a default limit of **25** (vs 100 for summaries) to keep output manageable. Each detail object can be 2-10KB of JSON, so 100 detail results could produce 200KB-1MB+ of output. Set `limit` explicitly to override the default.

### Filter Behaviors

- **Parameter auto-resolution:** `walk_parameters` automatically resolves `$ref` parameters when the `in` or `name` filter is specified. Without this, `$ref` parameters have empty `in` and `name` fields, causing filters to silently return 0 results. The `path` and `method` filters don't need this since they operate on location context available even for unresolved refs. **Note:** Specs with circular `$ref`s fall back to original data during resolution (#326), so parameter auto-resolution may not take effect for those specs.
- **OAS 3.1 type filter:** `walk_schemas` `type=string` uses exact match. OAS 3.1 nullable types (`type: [string, null]`) won't match `type=string`. Use `component=true` with `detail=true` to inspect multi-type schemas.
- **Schema name filter scope:** `walk_schemas` `name` matches both component schema names and nested property names. Use `component=true` to restrict to component schemas only.

### Cross-Tool Limitations

- **Cross-version diff:** Comparing specs across OAS versions (e.g., 2.0 vs 3.0) with `diff` reports structural format changes as breaking. This is technically correct but reflects version differences, not API changes.
- **Fixer coverage:** `fix` handles structural issues (duplicate operationIds, missing path parameters, unused schemas, missing `$ref` targets). Semantic validation errors (invalid compositions, type mismatches) are not auto-fixable.
- **Validator strictness:** Specs using `allOf`/`anyOf`/`oneOf` compositions may produce high error counts in strict mode if required properties are distributed across composed schemas.

---

## Error Handling

When a tool encounters an error (invalid input, parse failure, etc.), the response uses the MCP error convention:

- `IsError` is set to `true`
- The `content` array contains a `TextContent` item with the error message

This is a **tool-level** error — the MCP protocol call itself succeeds. This allows agents to detect the error and retry with corrected input.

---

## Architecture

The MCP server is implemented in the `internal/mcpserver` package:

```text
internal/mcpserver/
├── server.go              # Entry point and tool registration
├── input.go               # Shared specInput type
├── tools_validate.go      # validate tool
├── tools_parse.go         # parse tool
├── tools_fix.go           # fix tool
├── tools_convert.go       # convert tool
├── tools_diff.go          # diff tool
├── tools_join.go          # join tool
├── tools_overlay.go       # overlay_apply + overlay_validate
├── tools_generate.go      # generate tool
├── tools_walk_*.go        # 8 walk tools
└── integration_test.go    # In-process integration tests
```

Each tool handler follows a consistent pattern:

1. Parse the typed input struct (auto-deserialized by the SDK via JSON Schema)
2. Resolve the spec input (file, URL, or inline content)
3. Call the underlying oastools library package
4. Return a typed output struct (auto-serialized by the SDK)

The server uses the [Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk) with generics for type-safe tool registration.
