# Walk Tool Aggregation & walk_headers Design

## Goal

Add `group_by` aggregation to the four collector-based walk tools and introduce a new `walk_headers` tool, improving the MCP server's ability to answer distribution questions ("how many operations per tag?") without paging through thousands of items.

## Motivation

The walker's collectors (`CollectOperations`, `CollectSchemas`, `CollectParameters`, `CollectResponses`) already compute groupings (`.ByTag`, `.ByMethod`, `.ByStatusCode`, `.ByLocation`), but the MCP tools only use these for pre-filtering, never for aggregation output. The data is already grouped in memory -- we're just not exposing it.

For headers, there's no way to directly query response headers or component headers. They're only visible deeply nested inside `walk_responses` or `walk_parameters` with `detail: true`.

## Feature A: `group_by` for Walk Tools

### Scope

All four collector-based walk tools get a `group_by` parameter:

| Tool | `group_by` values |
|------|-------------------|
| `walk_operations` | `tag`, `method` |
| `walk_schemas` | `type`, `location` |
| `walk_parameters` | `location`, `name` |
| `walk_responses` | `status_code`, `method` |

### Semantics

- **Filters apply before grouping** (SQL WHERE + GROUP BY). Example: `walk_operations(group_by="method", tag="Users")` returns method distribution *within* the Users tag.
- **Mutually exclusive with `detail`**. Setting both returns an error: `"cannot use both group_by and detail"`.
- **Invalid values return an error** with the tool-specific allowed values listed.
- **Groups sorted by count descending**, ties broken alphabetically by key.
- **Pagination applies to groups** (`offset`/`limit`).

### Output Shape

When `group_by` is set, the `groups` array replaces `summaries` and detail arrays:

```go
type groupCount struct {
    Key   string `json:"key"`
    Count int    `json:"count"`
}
```

```json
{
  "total": 16000,
  "matched": 16000,
  "returned": 4,
  "groups": [
    {"key": "GET", "count": 8500},
    {"key": "POST", "count": 4200},
    {"key": "DELETE", "count": 2100},
    {"key": "PATCH", "count": 1200}
  ]
}
```

### Grouping Implementation

Filters first, then group from the filtered slice (don't use collector indexes directly when filters are active):

```
collect -> filter -> group -> sort -> paginate
```

### Tool-Specific Notes

- **walk_operations `group_by=tag`**: Untagged operations are excluded from groups (matching `collector.ByTag` behavior). The difference `matched - sum(groups[*].count)` reveals the untagged count.
- **walk_schemas `group_by=location`**: Two groups: "component" and "inline".
- **walk_schemas `group_by=type`**: Groups by schema type string ("object", "array", "string", etc.). Schemas with no type get key "".

### jsonschema Descriptions

Allowed values are embedded directly in the jsonschema description string for vanilla MCP client discoverability:

```
"Group results and return counts instead of individual items. Values: tag\\, method"
```

## Feature B: `walk_headers` Tool

### Purpose

Expose response headers and component headers as first-class walkable items.

### Implementation Pattern

Uses raw `Walk()` + `WithHeaderHandler` (no collector exists for headers). Same pattern as `walk_paths` and `walk_refs`.

### Input

```go
type walkHeadersInput struct {
    Spec        specInput `json:"spec"                     jsonschema:"..."`
    Name        string    `json:"name,omitempty"           jsonschema:"Filter by header name (exact match\\, or glob with * and ?)"`
    Path        string    `json:"path,omitempty"           jsonschema:"Filter by path pattern (* = one segment\\, ** = zero or more segments)"`
    Method      string    `json:"method,omitempty"         jsonschema:"Filter by HTTP method"`
    Status      string    `json:"status,omitempty"         jsonschema:"Filter by status code: exact (200)\\, wildcard (2xx)\\, or default"`
    Component   bool      `json:"component,omitempty"      jsonschema:"Only show component headers (defined in components/headers)"`
    ResolveRefs bool      `json:"resolve_refs,omitempty"   jsonschema:"Resolve $ref pointers in output"`
    Detail      bool      `json:"detail,omitempty"         jsonschema:"Return full header objects instead of summaries"`
    GroupBy     string    `json:"group_by,omitempty"       jsonschema:"Group results and return counts. Values: name\\, status_code"`
    Limit       int       `json:"limit,omitempty"          jsonschema:"Maximum results (default 100)"`
    Offset      int       `json:"offset,omitempty"         jsonschema:"Skip the first N results (for pagination)"`
}
```

### Output

Summary mode:

```go
type headerSummary struct {
    Name        string `json:"name"`
    Path        string `json:"path,omitempty"`
    Method      string `json:"method,omitempty"`
    Status      string `json:"status,omitempty"`
    Description string `json:"description,omitempty"`
    Required    bool   `json:"required,omitempty"`
    Deprecated  bool   `json:"deprecated,omitempty"`
}
```

Detail mode: full `*parser.Header` object.

Group-by: `name` (header distribution across API) and `status_code` (header distribution across status codes).

### WalkContext Fields Used

- `wc.Name` -> header name
- `wc.JSONPath` -> full path
- `wc.IsComponent` -> component vs inline
- `wc.PathTemplate` -> owning path (empty for component headers)
- `wc.Method` -> owning method (empty for component headers)
- `wc.StatusCode` -> owning response status (empty for component headers)

### OAS 2.0 Compatibility

OAS 2.0 has response headers too. The walker's `WithHeaderHandler` handles both versions. No special casing needed.

### Tool Registration

Tool count: 16 -> 17. Description:

> Walk and query response headers and component headers in an OpenAPI Specification document. Filter by name, path, method, status code, or component location. Returns summaries (name, path, method, status, description) by default or full header objects with detail=true. Use group_by=name to find the most commonly used headers across the API.

## Plugin Guidance Updates

### plugin/CLAUDE.md

- Tool count: 16 -> 17
- Add `walk_headers` to Walk tool list
- Add best practice for `group_by`: "Use `group_by` to get distributions in a single call"

### explore-api Skill

- Step 2 (endpoints): Add `group_by` examples for tag and method distribution
- Step 3 (schemas): Add `group_by` example for type distribution
- Step 4 (specifics): Add walk_headers guidance
- All examples use jsonschema-documented values

## Edge Cases

- `group_by` + `detail` -> error
- Invalid `group_by` value -> error with tool-specific allowed values
- Empty results with `group_by` -> `{"total": N, "matched": 0, "returned": 0, "groups": []}`
- walk_headers on OAS 2.0 -> works (walker handles both versions)
- Operations without tags in `group_by=tag` -> excluded from groups, detectable via matched vs sum
