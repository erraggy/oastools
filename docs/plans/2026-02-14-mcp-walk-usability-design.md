# MCP Walk Tool Usability Improvements

**Date:** 2026-02-14
**Status:** Approved
**Motivation:** AI agent friction when navigating large OpenAPI specs (MS Graph: 16K operations, 4K schemas)

## Problem Statement

The oastools MCP walk tools have three categories of usability issues that cause AI agents to waste tool calls:

1. **Path glob matching is broken for real-world use** — `*` matches exactly one segment with no `**` support, so agents can't search across path depths
2. **No cross-referencing capability** — no way to count `$ref` references or map schemas to operations
3. **Tool descriptions mislead agents** — descriptions underspecify behavior, causing trial-and-error

## Design

### 1. Path Glob Matching Upgrade

**File:** `internal/mcpserver/tools_walk_operations.go` — `matchWalkPath()`

**Current behavior:** `*` matches exactly one path segment. Pattern must have same segment count as path.

**New behavior:**

| Pattern | Meaning | Example |
|---------|---------|---------|
| `*` | One segment (unchanged) | `/users/*` matches `/users/{id}` |
| `**` | Zero or more segments (new) | `/drives/**/workbook/**` matches `/drives/{id}/items/{id}/workbook/functions/abs` |

**Implementation:** Replace hand-rolled matcher with `**`-aware recursive matching. When a `**` segment is encountered, try matching the remaining pattern against every possible suffix of the remaining path segments.

**Impact:** All 4 tools using `matchWalkPath` (walk_operations, walk_paths, walk_parameters, walk_responses) get the upgrade.

### 2. Schema Name Glob Matching

**File:** `internal/mcpserver/tools_walk_schemas.go` — `filterWalkSchemas()`

**Current behavior:** `strings.EqualFold` — exact match only.

**New behavior:** If name contains `*` or `?`, treat as case-insensitive glob pattern. Otherwise exact match (backwards-compatible).

**Implementation:** New `matchName(name, pattern string) bool` helper that checks for glob characters and branches to `filepath.Match` (lowercased) or `strings.EqualFold`.

### 3. New `walk_refs` Tool

**File:** New `internal/mcpserver/tools_walk_refs.go`

Walk all `$ref` occurrences using the existing `walker.WithRefHandler` + `walker.WithMapRefTracking` infrastructure.

**Two modes:**

**Summary mode (default):** Returns unique ref targets ranked by count (most-referenced first).

```json
{
  "total": 15234,
  "matched": 15234,
  "returned": 5,
  "refs": [
    {"ref": "#/components/schemas/BaseCollectionPaginationCountResponse", "count": 1478},
    {"ref": "#/components/schemas/microsoft.graph.entity", "count": 669}
  ]
}
```

**Detail mode (detail=true):** Returns individual source locations for a specific target.

```json
{
  "total": 15234,
  "matched": 618,
  "returned": 5,
  "refs": [
    {
      "ref": "#/components/schemas/microsoft.graph.workbookRange",
      "source_path": "$.paths['/drives/...'].get.responses['2XX'].content['application/json'].schema.anyOf[0]",
      "node_type": "schema"
    }
  ]
}
```

**Input parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `spec` | object | Required. The OAS document (file/url/content) |
| `target` | string | Filter by ref target (supports `*` glob, e.g. `*schemas/microsoft.graph.workbook*`) |
| `node_type` | string | Filter by: schema, parameter, response, requestBody, header, pathItem |
| `detail` | bool | Show source locations instead of aggregated counts |
| `limit` | int | Max results (default 100) |
| `offset` | int | Pagination offset |

### 4. Tool Description Audit

All tool descriptions and parameter descriptions updated for AI agent optimization. Principles:

1. Lead with "when to use" signal
2. Include examples in parameter descriptions
3. Encode strategy hints ("filter by tag first for large APIs")
4. State limitations explicitly

#### Core Tool Description Changes

**validate:**
> Validate an OpenAPI Specification document against its version schema. Returns errors and warnings with JSON path locations. For large specs, use no_warnings to focus on errors first. Use offset/limit to paginate through results.

**parse:**
> Parse an OpenAPI Specification document. Returns a structural summary: title, version, OAS version, path/operation/schema counts, servers, and tags. Use full=true only for small specs; for large specs use walk_* tools to explore specific sections.

- `full` parameter: Add `"WARNING: produces very large output for big specs — prefer walk_* tools instead."`

**fix:**
> Automatically fix common issues in an OpenAPI Specification document. Fix types: generic schema names, duplicate operationIds, missing path parameters, unused schemas/empty paths (prune), missing $ref targets (stub). Use dry_run=true to preview fixes before applying. Use output to write to a file instead of returning inline.

**diff:**
> Compare two versions of the same OpenAPI Specification document and report differences. Detects breaking changes, additions, removals, and modifications with severity levels. Use breaking_only=true to focus on breaking changes first. Both base and revision must be provided.

**join:**
> Join multiple OpenAPI Specification documents into a single merged document. Requires at least 2 specs via the specs array. Collision strategies: accept_left, accept_right, fail (paths/schemas), rename (schemas only). Use semantic_dedup to merge equivalent schemas.

**generate:**
> Generate Go code from an OpenAPI Specification document. Set exactly one of: types (type definitions only), client (HTTP client), or server (server interfaces and handlers). Requires output_dir. Returns a manifest of generated files.

**convert, overlay_apply, overlay_validate:** No changes needed.

#### Walk Tool Description Changes

**walk_operations:**
> Walk and query operations in an OpenAPI Specification document. Filter by method, path, tag, operationId, deprecated status, or extension. Returns summaries (method, path, operationId, tags) by default or full operation objects with detail=true. For large APIs, filter by tag first (most selective), then narrow with path or method. Path patterns support * (one segment) and ** (zero or more segments).

**walk_schemas:**
> Walk and query schemas in an OpenAPI Specification document. Filter by name, type, component/inline location, or extension. Returns summaries (name, type, JSON path, component status) by default or full schema objects with detail=true. Use component=true to see only named component schemas (skips inline schemas, reducing results 3-5x). Avoid detail=true without filters on large specs.

**walk_parameters:**
> Walk and query parameters in an OpenAPI Specification document. Filter by location (in), name, path pattern, method, or extension. Returns summaries (name, location, path, method) by default or full parameter objects with detail=true.

**walk_responses:**
> Walk and query responses in an OpenAPI Specification document. Filter by status code, path pattern, method, or extension. Returns summaries (status code, path, method, description) by default or full response objects with detail=true.

**walk_security:**
> Walk and query security schemes defined in components. Filter by name or type (apiKey, http, oauth2, openIdConnect). Returns summaries (name, type, location) by default or full security scheme objects with detail=true.

**walk_paths:**
> Walk and query path items in an OpenAPI Specification document. Filter by path pattern or extension. Returns summaries (path, method count) by default or full path item objects with detail=true. Path patterns support * (one segment) and ** (zero or more segments), e.g. /users/** matches all paths under /users.

**walk_refs (new):**
> Walk and count $ref references in an OpenAPI Specification document. By default, returns unique ref targets ranked by reference count (most-referenced first). Use target to filter to a specific ref (supports * glob, e.g. *schemas/microsoft.graph.*). Use detail=true to see individual source locations instead of counts. Filter by node_type to narrow to schema, parameter, response, requestBody, header, or pathItem refs.

#### Parameter Description Standardization

**Path filter (4 tools):**
> Filter by path pattern (* = one segment, ** = zero or more segments, e.g. /users/* or /drives/**/workbook/**)

**Schema name filter:**
> Filter by schema name (exact match, or glob with * and ? for pattern matching, e.g. *workbook* or microsoft.graph.*)

**Tag filter:**
> Filter by tag name (exact match, case-sensitive)

**operation_id filter:**
> Select a single operation by operationId (exact match)

**Parameter name filter:**
> Filter by parameter name (case-insensitive exact match)

**Status code filter:**
> Filter by status code: exact (200, 404), wildcard (2xx, 4xx, 5xx), or default (case-insensitive)

**resolve_refs (standardize across all):**
> Resolve $ref pointers in output. Inlines referenced objects instead of showing $ref strings.

**component (walk_schemas):**
> Only show component schemas (defined in components/schemas or definitions). Mutually exclusive with inline.

**inline (walk_schemas):**
> Only show inline schemas (embedded in operations, not in components). Mutually exclusive with component.

**detail (walk_schemas, add warning):**
> Return full schema objects. WARNING: produces large output without name/type filters on big specs.

**full (parse, add warning):**
> Return full parsed document instead of summary. WARNING: produces very large output for big specs — prefer walk_* tools instead.

## Priority Order

1. **P0 — Path glob `**` support** (fixes the #1 friction, affects 4 tools)
2. **P0 — Tool description audit** (zero-code-change, high-impact for agent usability)
3. **P1 — Schema name glob** (small fix, one tool)
4. **P2 — `walk_refs` tool** (new capability, medium effort)

## Testing Strategy

- Path glob: Unit tests for `matchWalkPath` covering `**` with 0, 1, N segment matches, edge cases (trailing `**`, leading `**`, `**` in middle)
- Schema name glob: Unit tests for `matchName` with exact, glob, case-insensitive cases
- `walk_refs`: Integration tests against petstore + MS Graph corpus fixtures for both summary and detail modes
- Description changes: Manual verification via MCP tool listing
