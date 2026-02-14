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
5. **Filter walk results.** Walk tools support filters (method, path, tag, status, name, type) and return summaries by default. Use `detail: true` only when you need full objects.
6. **Check breaking changes.** When diffing specs, use `breaking_only: true` to focus on changes that will break API consumers.

## Common Workflows

**Validate and fix:**
1. `validate` the spec to find issues
2. `fix` with `dry_run: true` to preview fixes
3. `fix` to apply fixes
4. `validate` again to confirm the spec is now valid

**Explore an API:**
1. `parse` to get title, version, path/schema/operation counts
2. `walk_operations` to list endpoints (filter by tag, method, or path)
3. `walk_schemas` to list data models
4. `walk_operations` with `detail: true` on specific endpoints for full request/response details

**Compare API versions:**
1. `diff` with both specs to see all changes
2. Review breaking changes and their severity
3. Use `walk_operations` on the revision to understand new/modified endpoints
