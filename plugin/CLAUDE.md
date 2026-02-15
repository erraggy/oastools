# oastools MCP Plugin

You have access to the oastools MCP server, which provides 15 tools for working with OpenAPI Specification (OAS 2.0-3.2) documents.

## Available Tools

**Core (9):** `validate`, `parse`, `fix`, `convert`, `diff`, `join`, `overlay_apply`, `overlay_validate`, `generate`

**Walk (6):** `walk_operations`, `walk_schemas`, `walk_parameters`, `walk_responses`, `walk_security`, `walk_paths`

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
2. 🏷️ `walk_operations` with `tag` filter — work through one tag at a time
3. 📋 `walk_schemas` with `component: true` — named schemas only (skip inline)
4. 🔍 Drill into specifics with `operation_id` or `path` + `detail: true`
5. ✅ Use `validate` with `no_warnings: true` for error-focused triage
