# Building an MCP Server for OpenAPI in 48 Hours

*How corpus-driven development turned a prototype into a production-grade tool server across 6 releases.*

**Published:** February 2026 | **Releases:** v1.51.0 – v1.51.5 | **Pull Requests:** 8 PRs, 8,274 lines of Go

---

## The Pitch

OpenAPI specs are everywhere, but working with them from AI coding assistants is painful. You can ask an LLM to read a YAML file, but it has no understanding of what `$ref: '#/components/schemas/Pet'` actually resolves to, whether a spec is valid, or how two API versions differ. The LLM sees text; it needs structure.

[Model Context Protocol](https://modelcontextprotocol.io/) (MCP) changes this equation. Instead of asking the LLM to interpret raw YAML, you give it *tools* — validate this spec, walk its operations, diff these two versions — and it calls them like functions, receiving typed JSON back. The LLM reasons about structure instead of parsing syntax.

oastools already had all the capabilities: parsing, validation, fixing, diffing, joining, converting, overlays, code generation, and deep traversal with filtering. They just needed to be exposed over MCP's JSON-RPC protocol.

What followed was an unexpectedly intense 48-hour development sprint.

---

## Day 0: The Foundation (v1.51.0)

**February 13, 2026 — PR [#310](https://github.com/erraggy/oastools/pull/310)**

The initial implementation landed as a single PR: 15 MCP tools covering every oastools capability, a CLI subcommand (`oastools mcp`), a Claude Code plugin with guided skills, and documentation.

### Architecture

The MCP server lives in `internal/mcpserver/` — deliberately internal, since the wire protocol is the public API, not the Go types. Each tool follows the same pattern:

```
Input struct (JSON Schema) → Resolve spec → Call library package → Output struct (JSON)
```

The [Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk) uses generics for type-safe tool registration. You define an input struct with `jsonschema` tags, and the SDK handles JSON Schema generation, validation, and deserialization:

```go
type validateInput struct {
    Spec       specInput `json:"spec"       jsonschema:"required,description=The OAS document"`
    Strict     bool      `json:"strict"     jsonschema:"description=Enable stricter validation"`
    NoWarnings bool      `json:"no_warnings" jsonschema:"description=Suppress warnings"`
}
```

A shared `specInput` type handles the three input modes (file path, URL, inline content) with mutual exclusivity validation. This one type serves all 15 tools (and later, the 2 added in v1.51.3).

### The 15 tools

- **Spec lifecycle:** `parse`, `validate`, `fix`, `convert`
- **Multi-spec:** `diff`, `join`
- **Overlays:** `overlay_apply`, `overlay_validate`
- **Code generation:** `generate`
- **Walk tools:** `walk_paths`, `walk_operations`, `walk_schemas`, `walk_parameters`, `walk_responses`, `walk_security`

Each walk tool supports filtering by relevant dimensions (path pattern, HTTP method, status code, schema type, parameter location) and a `detail` flag for full objects vs. summaries.

### Design decisions

**Format preservation.** If you feed in JSON, you get JSON back. YAML in, YAML out. This seems obvious but matters enormously for LLM workflows where the agent writes a fixed spec back to disk — it shouldn't silently convert your YAML project to JSON.

**Structured summaries by default.** The `parse` tool returns counts and metadata, not the entire parsed document. Walk tools return compact summaries. An LLM working with a 36MB spec (MS Graph has 16,098 operations) needs aggregated intelligence, not a firehose.

**Consistent error handling.** Every error returns as an MCP tool error (`IsError: true`) rather than crashing the server. The agent sees the error, adjusts its input, and retries. This is critical for autonomous workflows.

At this point, 8,000+ tests passed, and the server worked. Ship it.

---

## Day 1, Morning: The Overflow Problem (v1.51.1)

**February 14, 2026 — PR [#315](https://github.com/erraggy/oastools/pull/315)**

We immediately ran the MCP tools against our corpus of 10 real-world OpenAPI specs: Petstore, Google Maps, NWS Weather, Asana, Discord, Plaid, DigitalOcean, Stripe, GitHub, and Microsoft Graph. The results were... large:

| Tool | Spec | Output size | Items |
|------|------|------------|-------|
| `validate` | GitHub API | 367 KB | 2,158 errors |
| `fix` | GitHub API | 381 KB | 2,051 fixes |
| `diff` | Stripe vs GitHub | 591 KB | 3,445 changes |

Half a megabyte of validation errors in a single tool response. LLM context windows are precious — we were burning them on pagination that should happen server-side.

### The solution: `paginate[T]`

A generic helper that all array-returning tools share:

```go
func paginate[T any](items []T, offset, limit int) []T
```

Nine tools gained `offset` and `limit` parameters. Default limit: 100. Totals (`error_count`, `fix_count`, `breaking_count`) always reflect the full result set — the agent knows there are 2,158 errors even when only seeing 100 at a time. A `returned` field tells the agent how many items are in the current page.

With `limit=5`, that 367KB validate response drops to ~0.8KB. The agent can triage the first page, apply filters (`no_warnings: true`), and drill deeper as needed.

---

## Day 1, Afternoon: Completing the Pipeline (v1.51.2)

**February 14, 2026 — PR [#318](https://github.com/erraggy/oastools/pull/318)**

Corpus testing also revealed a workflow gap: the `fix` tool could apply fixes but couldn't write the result to disk. An agent using the MCP server had to fall back to CLI commands for persistence. Meanwhile, `convert`, `join`, and `overlay_apply` all had `output` parameters.

This was a quick fix — add `output` to `fixInput`, add `written_to` to `fixOutput` — but it completed a crucial pattern: **every transform tool can now persist its results**, enabling multi-tool pipelines entirely through MCP:

```
fix(spec, output="/tmp/fixed.yaml")
  → validate(file="/tmp/fixed.yaml")
  → convert(file="/tmp/fixed.yaml", target="3.1", output="/tmp/converted.yaml")
```

### Plugin maturation

This release also restructured the Claude Code plugin:

- **Version coupling**: Plugin version now tracks the binary version instead of independent semver. A `SessionStart` hook warns when they diverge.
- **Skills auto-discovery**: Moved from flat `skills/*.md` to `skills/*/SKILL.md` subdirectories, matching Claude Code's auto-discovery convention.
- **Workflow docs**: Added "Persisting Results" and "Pipelining Tools" sections to teach agents multi-tool chaining.

### Corpus bugs found (and fixed)

Running the tools against real specs also uncovered three bugs in the underlying library:

1. **Generator schema collision** (Discord): Schema names ending in "Request" collided with generated server wrapper structs. Fixed with a suffix cascade (`Request` → `Input` → `Req` → numeric fallback).
2. **Converter formData passthrough** (Petstore 2.0→3.x): `in: "formData"` parameters were passed through unconverted instead of building a `requestBody`.
3. **Converter downconversion loss** (NWS 3.0→2.0): Composite schemas lost their `type` field, and header refs went unresolved.

None of these were found by unit tests. They needed real-world specs with real-world complexity.

---

## Day 1, Night: The Aggregation Leap (v1.51.3)

**February 15, 2026 — PR [#321](https://github.com/erraggy/oastools/pull/321)**

This was the largest single feature release. We'd been using the walk tools in a loop — "group operations by tag" meant calling `walk_operations` once per tag and counting results. For MS Graph with 457 tags, that's 457 API calls to answer one question.

### `group_by`

Every walk tool gained a `group_by` parameter that returns `{key, count}` groups sorted by count:

```json
{"spec": {"file": "github-api.json"}, "group_by": "method"}
```

```json
{
  "total": 1078,
  "groups": [
    {"key": "GET", "count": 568},
    {"key": "POST", "count": 171},
    {"key": "DELETE", "count": 166},
    {"key": "PUT", "count": 112},
    {"key": "PATCH", "count": 61}
  ]
}
```

One call instead of five. For the corpus analysis of 10 specs across multiple dimensions, `group_by` reduced hundreds of calls to dozens.

The implementation uses a generic `groupAndSort[T any]` helper with filter-before-group semantics (WHERE then GROUP BY, like SQL). Pagination applies to grouped results too.

### Two new walk tools

**`walk_headers`** fills a genuine gap — before this, finding response headers required `walk_responses` with `detail=true` and manual inspection. Headers are crucial for understanding rate limiting, CORS, and pagination patterns. Discord's spec has 5 rate-limit headers across 240 operations (1,200 header instances).

**`walk_refs`** shows dependency graphs — which schemas reference which, with counts. Discord's `SnowflakeType` (554 references) reveals more about the API's design than any documentation. GitHub's `owner`/`repo` parameters (480/479 refs) reveal the repo-centric architecture.

### Glob-style filtering

Path filters gained `**` for multi-segment matching:

- `/users/*` matches `/users/{id}` but not `/users/{id}/posts`
- `/drives/**/workbook/**` matches any depth under both segments

Schema name filters gained glob support too: `Pet*` matches `Pet`, `PetStore`, `PetResponse`.

Tool count: 15 → 17.

---

## Day 2, Afternoon: Performance and Polish (v1.51.4)

**February 15, 2026 — PR [#324](https://github.com/erraggy/oastools/pull/324)**

With `group_by` in hand, we ran a systematic corpus analysis — all 10 specs, every dimension. The full results are in the [Corpus Analysis](../corpus-analysis.md), but here are some highlights that surprised us:

!!! info "Corpus Analysis Highlights"

    The analysis covers **19,173 operations** across 10 real-world specs spanning 3 orders of magnitude (20 to 16,098 operations). Some of the most interesting patterns:

    - **Plaid is 99.4% POST** — pure RPC-over-HTTP, every endpoint is a command
    - **MS Graph uses 4 integers total** across 4,294 schemas — their OData convention prefers `number` for everything
    - **Discord's `SnowflakeType` is referenced 554 times** — one schema reveals their entire ID architecture
    - **GitHub has 20,123 string schemas** (63% of all component schemas) — enums and scalars expanded individually
    - **Stripe documents zero response headers** despite having rate limits in practice
    - **NWS uses User-Agent as a security scheme** — creative abuse tracking via a required header

    Each pattern tells a story about API design philosophy. [Read the full analysis →](../corpus-analysis.md)

But the analysis also exposed pain points. Every `group_by` call on the same spec meant re-parsing it. MS Graph (36MB YAML) takes noticeable time to parse. Multiply by 8 walk tools and it's frustrating.

### Session-scoped spec cache

An LRU cache (max 10 entries) stores parsed specs for the session duration:

- **File inputs**: keyed by absolute path + modification time — invalidates automatically on file change
- **Content inputs**: keyed by SHA-256 hash — identical inline specs reuse cached results
- **URL inputs**: bypass cache (remote content may change)

Read-only tools (`parse`, `validate`, `walk_*`, `diff`, `generate`) use the cache. Mutating tools (`fix`, `convert`, `join`, `overlay_apply`) bypass it since they modify the document.

### Better labels

Empty group keys were confusing. `""` in `group_by=type` results means "schema without an explicit type" — compositions, `$ref` wrappers. Now these display meaningful labels:

- `(untyped)` for typeless schemas
- `(ref)` for unresolved `$ref` parameters
- `(component)` for component-level responses

### New aggregation dimensions

- `walk_refs` gained `group_by=node_type` (schema vs. parameter vs. response)
- `walk_paths` gained `group_by=segment` (first path segment for API structure overview)

### Parse truncation

DigitalOcean's `info.description` was 8,000 characters — their entire API introduction with curl examples and CORS docs. The `parse` tool now truncates to 200 characters in summary mode, with rune-safe truncation so multi-byte UTF-8 characters are never split mid-codepoint.

---

## Day 2, Evening: Final Edge Cases (v1.51.5)

**February 15, 2026 — PRs [#327](https://github.com/erraggy/oastools/pull/327), [#329](https://github.com/erraggy/oastools/pull/329), [#330](https://github.com/erraggy/oastools/pull/330)**

The final release addressed three distinct issues found during continued corpus stress-testing.

### Detail mode was too verbose

Walk tools with `detail=true` returned up to 100 full objects — each 2-10KB of JSON. For MS Graph, that's potentially 1MB per call. A shared `detailLimit()` helper now defaults detail mode to 25 results (explicitly overridable).

### Parameter filters silently failed

`walk_parameters` with `in=query` returned 0 results on GitHub's spec, even though it has hundreds of query parameters. The problem: 86% of GitHub's parameters use `$ref`, and unresolved refs don't have an `in` field. The fix: auto-resolve refs when `in` or `name` filters are specified, so users don't need to know about `resolve_refs=true`.

### A naming mismatch caused 100% first-attempt failures

The `join` tool documented `accept_left`/`accept_right` (underscored) as collision strategies, but the underlying joiner package uses `accept-left`/`accept-right` (hyphenated). Every LLM agent passed invalid values on first attempt, got an error, and had to retry. A one-line fix in the `jsonschema` tags, but it eliminated a guaranteed round-trip for every join call.

### Circular reference resolution (bonus: parser fix)

While testing `walk_parameters` auto-resolution, we discovered that the parser's `resolve_refs` mode had a nuclear fallback: detecting *any* circular `$ref` caused it to discard *all* resolution work. The fix was surgical — only truly circular nodes remain as `$ref` pointers; everything else resolves normally. As a bonus, the new `decodeFromMap` approach eliminated an intermediate `[]byte` allocation that was inflating memory 10x on large specs (GitHub API: 11.6MB input → 108MB intermediate → now eliminated).

---

## The Feedback Loop

Looking back, the most interesting aspect isn't any single feature — it's the development cadence. Six releases in 48 hours, each driven by the same cycle:

```
Build → Test against real specs → Discover gap → Fix → Ship
```

The corpus of 10 specs acted as a forcing function. Every new feature immediately met reality:

- **Pagination** was born from 367KB validate responses
- **`group_by`** was born from looping `walk_operations` 457 times for MS Graph
- **Spec caching** was born from re-parsing 36MB of YAML on every call
- **Auto-resolution** was born from `in=query` returning 0 results
- **Detail limits** were born from 1MB detail responses
- **Strategy name fix** was born from 100% first-attempt failures

The corpus didn't just find bugs — it shaped the API surface. Features like `group_by`, auto-resolution, and meaningful group labels all emerged from watching the tools fail in practice rather than from upfront design.

## By the Numbers

| Metric | Value |
|--------|-------|
| Development window | 48 hours (Feb 13–15, 2026) |
| Releases | 6 (v1.51.0 – v1.51.5) |
| Pull requests | 8 MCP-focused PRs |
| MCP tools | 15 → 17 |
| Lines of Go | 8,274 across 37 files |
| Test count | 8,000 → 8,200+ |
| Corpus specs tested | 10 (19,173 total operations) |
| Corpus size range | 20 operations (Petstore) to 16,098 (MS Graph) |
| Features added | Pagination, group_by, caching, walk_headers, walk_refs, glob matching, auto-resolution, detail limits |
| Bugs found via corpus | 6 (3 library, 3 MCP-specific) |

## Getting Started

```bash
# Install
brew install erraggy/oastools/oastools

# Start the MCP server
oastools mcp

# Or add to Claude Code
claude mcp add --transport stdio oastools -- oastools mcp
```

The full tool reference is in the [MCP Server documentation](../mcp-server.md).
