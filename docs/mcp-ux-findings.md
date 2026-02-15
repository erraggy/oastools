# MCP Server UX Findings

> Observations from using the oastools MCP server tools as a Claude Code client to analyze the `testdata/corpus/` specs. Focus is on what a fresh-install user would encounter without codebase knowledge.
>
> Generated 2026-02-15 from corpus analysis session.

## Resolution Status

| Finding | Status | Resolution |
|---------|--------|------------|
| `group_by` on `walk_refs` | Addressed | Added `group_by=node_type` |
| `group_by` on `walk_paths` | Addressed | Added `group_by=segment` (first path segment) |
| `group_by` on `walk_security` | Deferred | Low priority -- most specs have 0-3 schemes |
| Cross-spec aggregation | Out of scope | Would require multi-spec input; not planned |
| `walk_headers` `component` filter | Deferred | Low usage -- headers rarely exceed component count |
| Header name casing normalization | Deferred | Spec quality issue; may add as option later |
| Empty location for $ref parameters | Addressed | Labeled as `(ref)` in group_by results |
| Schema type "" ambiguity | Addressed | Labeled as `(untyped)` in group_by results |
| Component response "" status | Addressed | Labeled as `(component)` in group_by results |
| Parse description truncation | Addressed | Truncated to 200 chars in summary mode |
| Plugin rebuild after update | Addressed | SessionStart hook checks binary vs plugin version |
| `group_by` enum constraints | Deferred | Works via description; enum would be stricter |
| `walk_refs` total semantics | Addressed | Documented in MCP server docs |
| `walk_headers`/`walk_refs` missing from docs | Addressed | Added to MCP server documentation |
| Operation-level security overrides | Deferred | Would need new walk_security features |
| Spec re-parsing on every call | Addressed | Session-scoped LRU cache (max 10 entries) |

> **Note:** The sections below reflect findings as observed during the original corpus analysis session. The Resolution Status table above shows which items have since been addressed. Sections describing addressed items are preserved as historical context for the design decisions.

---

## Setup & Discovery

### Plugin rebuild required after code changes

After PR #321 merged (adding `group_by`, `walk_headers`, `walk_refs`), the **running MCP server plugin didn't expose the new tools or parameters** until the plugin binary was rebuilt. The MCP tool schema is generated at runtime from Go struct tags, so a stale binary means stale schemas.

**Impact**: A user who updates oastools via `go install` or git pull won't get new MCP features until they also rebuild the plugin. There's no version mismatch warning.

**Suggestion**: The plugin could expose its version (from `plugin.json`) in tool responses or as a resource, so clients can detect staleness. Alternatively, document the rebuild step prominently in plugin setup docs.

### Tool parameter discoverability

The `group_by` parameter's allowed values are documented in the JSON Schema `description` field (e.g., "Values: tag, method"). This works well -- Claude Code picks them up automatically from the schema. However, the allowed values are buried in prose rather than using `enum` constraints.

**Suggestion**: Consider adding `enum` to `group_by` fields so clients can validate locally and offer autocompletion. The current approach works but relies on the client reading the description text.

---

## `group_by` Aggregation

### What works well

- **Massive efficiency gain**: analyzing method distribution across 10 specs took 10 parallel calls instead of 50+ (5 methods x 10 specs). Each `group_by` call replaces N filter-and-count calls.
- **Output shape is clean**: `{"groups": [{"key": "GET", "count": 568}]}` is immediately usable. The `total` and `matched` counts alongside groups provide context.
- **Sorted by count descending**: groups come back largest-first, which is the natural reading order for analysis.
- **Composable with filters**: `group_by=method` combined with `tag=repos` works correctly to show method distribution within a specific tag.

### What could be improved

- **No `group_by` on `walk_paths`**: paths are the only walk tool without aggregation. A `group_by=depth` or `group_by=segment` (first path segment) would be valuable for understanding API structure. For example, GitHub's paths could be grouped by `/repos/**`, `/orgs/**`, `/users/**`.
- **No `group_by` on `walk_refs`**: refs already return counts by default (which is great), but there's no way to group by ref *type* (schema vs response vs parameter vs header). You can filter by `node_type`, but you can't get a single "how many schema refs vs response refs vs parameter refs" summary.
- **No `group_by` on `walk_security`**: low priority since most specs have 0-3 security schemes, but `group_by=type` would show the auth type distribution across a spec.

### Missing: cross-spec aggregation

The most labor-intensive part of the analysis was running the same `group_by` call across all 10 specs and manually assembling the results into a table. A hypothetical `walk_operations` that accepts multiple specs and returns per-spec groups would eliminate this entirely. This may be out of scope for the MCP server, but it's the dominant workflow pain point.

---

## `walk_headers` (New)

### What works well

- **Fills a genuine gap**: before this tool, finding response headers required `walk_responses` with `detail=true` and manually inspecting each response object. Now it's a single call.
- **`group_by=name` is the killer feature**: immediately reveals rate-limit patterns (Discord: 5 headers x 240 operations = 1,200 total) and HATEOAS patterns (GitHub: `Link` header on 193 responses).
- **Correctly handles both inline and `$ref`-based headers**: Discord's headers are all `$ref`s to `#/components/headers/X-RateLimit-*`, and the tool resolves them for counting.

### What could be improved

- **No `component` filter like `walk_schemas` has**: `walk_schemas` has `component=true` to show only `components/schemas/` entries. An analogous `component=true` for `walk_headers` would show only `components/headers/` definitions, useful for understanding the reusable header vocabulary.
- **Header name casing**: GitHub's spec has both `Link` and `link`, `Location` and `location` as separate header entries. Since HTTP headers are case-insensitive, `group_by=name` could optionally normalize casing (or at least note duplicates). This is arguably a spec quality issue rather than a tool issue, but the tool is in a position to surface it.

---

## `walk_refs` (New)

### What works well

- **Default behavior is perfect**: returns unique ref targets ranked by reference count, most-referenced first. This immediately shows the "gravity centers" of a spec.
- **Ref counts reveal API structure**: Discord's `SnowflakeType` (554 refs) tells you more about the API's design than any other single data point. GitHub's `owner`/`repo` parameters (480/479 refs) reveal the repo-centric architecture.
- **Mixed ref types in one view**: seeing schemas, responses, parameters, and headers ranked together shows which component *category* dominates. For Discord, 5 of the top 8 refs are headers. For GitHub, 3 of the top 10 are parameters.

### What could be improved

- **No `group_by=node_type`**: as noted above, a single call to get "schema refs: 500, response refs: 300, parameter refs: 200, header refs: 100" would be a useful overview. Currently you'd need 4 separate calls with `node_type` filter.
- **`total` count semantics**: for `walk_refs`, `total` returns the number of *unique ref targets* (e.g., 1,349 for GitHub), not the total number of `$ref` occurrences across the document. This is the right default, but the distinction isn't documented. A user might expect `total` to mean "total $ref usages" (which would be the sum of all `count` values).

---

## `walk_parameters`

### Empty location for `$ref` parameters

Parameters that use `$ref` (e.g., `$ref: '#/components/parameters/owner'`) show up with an **empty string location `""`** in `group_by=location` results. This is technically correct (the `in` field isn't on the `$ref` object itself), but it's confusing.

For GitHub, 2,832 of 3,303 parameters (86%) show empty location. The workaround is `resolve_refs=true`, but that's expensive on large specs and changes the output format.

**Suggestion**: Consider resolving just the `in` field for `$ref` parameters during grouping, or label the empty group as `"$ref (unresolved)"` instead of `""`.

---

## `walk_schemas`

### `total` vs `matched` distinction

The `total` field counts *all* schemas (inline + component), while `matched` reflects filters. With `component=true`:

| Spec | total (all) | matched (component only) | Inline % |
|------|----------:|------------------------:|---------:|
| Petstore | 49 | 33 | 33% |
| GitHub | 34,847 | 31,959 | 8% |
| Stripe | 23,286 | 8,860 | 62% |
| MS Graph | 89,434 | 29,665 | 67% |

This reveals that **Stripe and MS Graph have more inline schemas than component schemas** -- a surprising finding that `group_by=location` would surface directly.

### Schema type "(none)" ambiguity

Schemas without an explicit `type` field show as `""` in `group_by=type`. These are composition schemas (`allOf`, `anyOf`, `oneOf`), pure `$ref` wrappers, or `enum`-only schemas. The empty string is accurate but unlabeled.

**Suggestion**: Consider labeling these as `"(composition)"` or `"(untyped)"` to make the output self-documenting. Alternatively, the docs could explain what `""` means in this context.

---

## `walk_responses`

### Wildcard status codes

Discord uses `4XX` and MS Graph uses `2XX`/`4XX`/`5XX` (OAS 3.0 range wildcards). These appear correctly in `group_by=status_code` as their literal values (`"4XX"`, `"5XX"`). The `status` filter also handles them with its own wildcard syntax (`status=2xx` matches `2XX` range codes).

This works well. The only gap is that there's no way to distinguish between a spec that declares `4XX` (range wildcard) and one that declares individual `400`, `401`, `403`, `404` codes. Both are valid OAS patterns but imply different levels of client guidance.

---

## `walk_security`

### Works perfectly for its scope

Security schemes are simple enough that the current tool covers the use case completely. No `group_by` is needed since specs rarely have more than 3 schemes.

### Missing: operation-level security overrides

`walk_security` only shows schemes *defined* in `components/securitySchemes`. It doesn't show which operations *use* which schemes, or which operations override the global security. A `walk_operations` filter like `security_scheme=OAuth2` would be useful for understanding auth coverage.

---

## `parse` Tool

### Description field can be enormous

DigitalOcean's `description` field in the parse output is ~8,000 characters -- it contains their entire API introduction including rate limiting docs, curl examples, and CORS documentation. This makes the parse response very large for what's meant to be a summary.

**Suggestion**: Truncate the `description` field in parse output (e.g., first 200 chars with `...`) or move it behind a `full=true` flag. The current behavior means parsing DigitalOcean returns ~10x more text than parsing Stripe, even though both are similar in structural complexity.

---

## Cross-Tool Patterns

### Consistent output shape

All walk tools follow the same pattern: `{"total": N, "matched": N, "returned": N, "summaries|groups|refs": [...]}`. This consistency is excellent for tooling -- once you learn one walk tool, you know the shape of all of them.

### Pagination works but is rarely needed

`limit` and `offset` are available on all walk tools. In practice, `group_by` eliminates the need for pagination in analysis workflows since aggregated results are always small. Pagination is most useful for `detail=true` queries on large specs.

### `resolve_refs` is a power-user feature

Most analysis workflows don't need `resolve_refs=true`. The one exception is `walk_parameters` where unresolved refs hide the parameter location. For all other tools, the default (no resolution) is the right choice because it's faster and the output is more compact.

---

## Documentation Gaps

1. **`group_by` allowed values**: documented in schema descriptions but not in the MCP server docs (`docs/mcp-server.md`). A table of "tool -> group_by values" would help.
2. **`walk_refs` total semantics**: the `total` field means "unique ref targets", not "total $ref occurrences". This should be documented.
3. **Empty string in group keys**: `""` appears in `group_by` results for schemas without `type`, parameters without `in` (due to `$ref`), and responses without explicit status codes. What `""` means varies by context and should be explained.
4. **`walk_headers` and `walk_refs` aren't in the MCP server docs yet**: they were added in PR #321 but the documentation page hasn't been updated.
5. **Plugin rebuild after update**: the docs don't mention that updating oastools source requires rebuilding the plugin to get new MCP features.

---

## Summary of Actionable Suggestions

| Priority | Suggestion |
|----------|-----------|
| High | Add `walk_headers` and `walk_refs` to MCP server documentation |
| High | Document `group_by` values in a table in MCP docs |
| High | Document what `""` means in each group_by context |
| Medium | Add `group_by=node_type` to `walk_refs` |
| Medium | ~~Label empty parameter locations~~ — resolved: labeled as `(ref)` |
| Medium | Truncate `description` in `parse` output |
| Medium | Add `group_by` to `walk_paths` (e.g., by first segment or depth) |
| Low | Add `component` filter to `walk_headers` |
| Low | Add `enum` constraints to `group_by` fields in JSON Schema |
| Low | Add version/build info to plugin for staleness detection |
| Low | Consider case-insensitive header name grouping option |
