# Plugin Corpus Bug Hunt Design

**Date**: 2026-02-14
**Goal**: Stress-test the oastools Claude plugin (MCP server + skills) against the full corpus to find bugs, crashes, incorrect output, and performance issues.

## Corpus Files

| File | Size | Format | Expected OAS |
|------|------|--------|-------------|
| petstore-swagger.json | 14KB | JSON | 2.0 |
| nws-openapi.json | 112KB | JSON | 3.0 |
| discord-openapi.json | 1MB | JSON | 3.0 |
| digitalocean-public.v2.yaml | 2.5MB | YAML | 3.0 |
| asana-oas.yaml | 2.7MB | YAML | 3.0 |
| google-maps-platform.json | 2.7MB | JSON | 3.0 |
| plaid-2020-09-14.yml | 2.9MB | YAML | 3.0 |
| stripe-spec3.json | 7.6MB | JSON | 3.0 |
| github-api.json | 11.7MB | JSON | 3.0 |
| msgraph-openapi.yaml | 36.4MB | YAML | 3.0/3.1 |

## Approach: Workflow-Based + Sweep

### Phase 1: Workflow Testing

**Workflow 1 — Explore API** (all 10 files):
1. `parse` → overview
2. `walk_operations` → list endpoints (summary)
3. `walk_schemas` → component schemas
4. `walk_operations` with tag/path filter
5. `walk_parameters`, `walk_responses`, `walk_security` for one endpoint

**Workflow 2 — Validate & Fix** (all 10 files):
1. `validate` → find errors
2. `fix` with `dry_run: true` → preview fixes

**Workflow 3 — Diff Specs** (2-3 pairs):
- Same file vs itself (zero changes expected)
- petstore (2.0) vs nws (3.0) — cross-version
- stripe vs github — large JSON

**Workflow 4 — Generate Code** (2-3 files):
- petstore (small), nws (medium), stripe or discord (large)

**Workflow 5 — Convert** (petstore OAS 2.0 → 3.x)

### Phase 2: Cleanup Sweep

Fill gaps not covered by workflows:
- `overlay_apply` and `overlay_validate`
- `join` (multiple specs)
- `validate` with `strict: true`
- `walk_*` with `detail: true` on large files
- `fix` flags: `prune`, `stub_missing_refs`, `fix_schema_names`, `fix_duplicate_operationids`

## What We Check

- **Crashes/errors**: Tool returns error or times out
- **Correctness**: Counts match, filters work, output makes sense
- **Performance**: Hangs or unreasonably slow on large files
- **Edge cases**: Missing fields, empty results, truncated output
- **MCP protocol**: JSON response parses correctly

## Deliverable

Bug report with: tool, file, input, error/unexpected behavior, severity.
