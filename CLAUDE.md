# CLAUDE.md

> ⚠️ **BRANCH PROTECTION**: Never commit directly to main. A PreToolUse hook enforces this automatically.

## Project Overview

`oastools` is a Go CLI for OpenAPI Specification files. Validates, fixes, joins, converts, diffs, walks, generates, and builds OAS 2.0-3.2.

## Style

- **Emojis welcome** in PR descriptions and release notes, but not required in code or docs
- **GitHub formatting**: Bare hashes/issues auto-link; backticks break linking
  - Good: `Fixed in commit 1f3eb93` → clickable
  - Bad: `Fixed in commit \`1f3eb93\`` → not clickable

## Quick Reference

- `make check` before committing
- Conventional commits: `feat(parser): add feature`
- **Never amend a pushed PR commit once review has begun** — add a new commit. PRs squash on merge, so amending only breaks review diffs and comment anchors
- See [WORKFLOW.md](WORKFLOW.md) for PR/release process
- See [AGENTS.md](AGENTS.md) for agent workflow

## Architecture

| Package | Purpose |
|---------|---------|
| cmd/oastools/ | CLI entry point |
| parser/ | Parse YAML/JSON OAS, resolve refs, detect versions |
| validator/ | Validate against spec schema |
| fixer/ | Auto-fix common errors |
| joiner/ | Join multiple OAS files |
| converter/ | Convert between OAS versions |
| differ/ | Compare specs, detect breaking changes |
| httpvalidator/ | Runtime HTTP validation |
| generator/ | Generate Go client/server |
| builder/ | Programmatic spec construction |
| overlay/ | Apply Overlay transformations |
| walker/ | Traverse with typed handlers |

## Key Patterns

- **Format preserved**: JSON/YAML auto-detected from extension or content
- **Use constants**: `httputil.MethodGet`, `severity.SeverityError`
- **Always run `go_diagnostics`** after edits—hints improve perf 5-15%
- **Favor fixing immediately** over deferring issues
- **Deep copy**: Use generated `doc.DeepCopy()` methods, **never** JSON marshal/unmarshal (loses `interface{}` types, drops `json:"-"` fields)
- **"No conversion needed" is not "same object"**: A field needing no per-version work is still a field two documents will share if you assign it straight across. `converter` hit this at 23 sites (Info, tags, externalDocs, security, response headers, OAuth scope maps) after `joiner` hit it in 8c55687, and Info was passed through whole, so a write to the converted document changed the source's title. When building an object field by field, copy: `parser.DeepCopyExtensions` for `Extra`, `parser.DeepCopySecurityRequirements` for `Security`, `DeepCopy()` for anything that has one, and `converter/deepcopy.go` for the shapes with neither
- **A deep copy is not a conversion**: the inverse of the bullet above, and #532 was both halves of it at different scales. `parser.Header` was copied across whole, so its Schema reached no pass, and `convertOAS3ToOAS3` copied the whole document, so a 3.0 to 3.1 conversion ran no per-schema pass at all. Both emitted the source version's spelling into a document declaring the target, and `validate` accepted the result, which is why neither showed up as a failure. When a field's type is the same in both versions, ask whether its *contents* are version-spelled before reaching for a copy
- **Choosing one media type out of a content map has one owner**: OAS 2.0 admits a single schema where OAS 3.x offers several, and ranging the map to pick one makes the output depend on Go's map order. `internal/httputil.MediaTypeRank` and `PreferredMediaType` own the order (`application/json`, then a `+json` suffix, then everything else, with the name breaking a tie), and `internal/schemautil.SortedContentTypes` applies it to a content map. #533 and #535 fixed the same defect in `converter`, `generator` and `builder` across two PRs, so a fourth package writing `for ct := range content { ...; break }` is the shape to catch in review. Each caller still applies its own predicate: `converter.selectContentSchema` skips a media type carrying no schema, since selecting it would lose the schema a sibling was offering
- **Put a refusal at the shared entry point, not the convenience wrapper**: `converter`'s parse-error guard sat in `Convert`, which reads a file, while every other caller arrives through `ConvertParsed`. The same document was refused from a file and converted from stdin
- **A new `parser` field has five homes, and `internal/driftguard` is what catches the ones you miss**: the hand-built `MarshalJSON` path, the structural hasher in `internal/schemautil`, the type's equality function, the joiner's schema comparison, and the deepcopy generator's field list. The guard is test-only and reflects over the struct, so a missed home is a test failure rather than a silent bug. Read its failure message before assuming the guard is wrong: it names the field and the home
- **Two passes that rank the same names must share the ranking function**: `joiner` decides which of several equivalent schema names survives in two places, the `StrategyDeduplicateOrRename` collapse and the `SemanticDeduplication` pass. The collapse ranked by `outranks` (a name no rename generated beats one a rename produced), the second pass sorted alphabetically, and with both enabled the second consolidated into the generated alias and dropped the name every document wrote (#498). The fix is one function reached from two places: `internal/schemautil.DeduplicationConfig.Outranks`, which `joiner` fills from `outranksGenerated` and `builder` leaves nil for the alphabetical default. Order matters too: take the generated-name set before the collapse, since `renameScope.redirect` points a rename at the name the class kept, so a set taken afterwards reports that surviving name as generated
- **Semantic deduplication compares shapes, so a name's meaning has to come from somewhere else**: two equal-comparing schemas have only their names telling them apart, and a schema referencing both relies on that: merging `OriginAddress` and `DestinationAddress` under a `Shipment` requiring both gives a document that validates and says a shipment's origin is its destination (#501). `internal/schemautil.DeduplicationConfig.Split` partitions a group of equivalent names into the parts that may each collapse, and `joiner` and `builder` fill it from `internal/schemarefs.Collect`. The unit is the schema tree (a schema with no schema above it), so depth does not matter, `oneOf` alternatives are held apart, and an inline parent counts like a named one. Sharing an operation is not sharing a tree, so a matching request body and response still merge
- **Schema-or-bool fields are always promoted**: `Items`, `AdditionalProperties`, `AdditionalItems`, `UnevaluatedItems` and `UnevaluatedProperties` are `any`, but every decode path (JSON, YAML, `decodeFromMap`) yields `*Schema`, `[]*Schema` (OAS 2.0 tuple form), or `bool`. A `map[string]any` arm is dead for parsed documents. Never type-assert to `*parser.Schema` alone: that drops the tuple form, and #502 was ~50 sites across 7 packages doing exactly that, so `-prune` deleted a schema a tuple element referenced and `convert` left the same $ref pointing at `#/definitions` in an OAS 3 document. Iterate with `internal/schemautil.SchemaOrBoolSchemas`, which yields each contained schema with its index, and build paths with its `IndexSuffix` so `items` and `items[0]` stay distinct in error messages. The structural hasher, `parser`'s `equalSchemaOrBoolWithVisited` and `joiner`'s `compareSchemaOrBool` carry the tuple arm too, since a hash that ignores tuples buckets schemas the comparison then calls different. `parser/equals.go`'s `equalSchemaOrBool` deliberately has none: its only caller intercepts `[]*Schema` first, so the arm is unreachable. `differ` carries it as of #511: a tuple classified as unknown drops every change inside it (both an element edit and a length change), and the `%T` fallback message it used to print named a Go type for a perfectly legal OAS 2.0 document
- **`internal/schemautil.SchemaTuple` owns whether a field IS the tuple form**: it returns the positions and a bool, and the bool is the answer, never `len(tuple) == 0`. An empty tuple is still the tuple form, and draft 4 gives it a meaning: it names no position, so `additionalItems` governs every element and `additionalItems: false` admits only an empty array. `httpvalidator` and `generator` each had their own `tupleItemSchemas` and they disagreed on exactly that case, so `items: []` with `additionalItems: false` rejected a one element request while the generated type constrained nothing (#529). A package may still decide what to DO with the form, under a name that says so: `generator.structTupleSchemas` reports no struct for an empty tuple because there is no position to give a field to, which is a decision about output rather than about what the schema means
- **`SchemaOrBoolSchemas` skips nil elements, so a paired walk must not use it**: comparing two tuples position by position (`differ.diffSchemaTupleUnified`) indexes the slices directly, because a nil present on one side alone would shift every later index and misreport `items[i]`. Nil elements are real: YAML decodes `- null` to one, while the JSON path drops it and changes the tuple's length (#510)
- **An OAS 2.0 constraint is often on the object, not in a schema**: only a body parameter has a `schema`. Every other `in` puts `type`, `enum` and the rest on the parameter object itself, as does a response header, as does either one's `Items` chain, and OAS 2.0 offers nowhere else to put them. A pass reaching a constraint through `Parameter.Schema` alone expanded a body parameter's CSV enum and left the identical value on a query parameter (#513). The fix shape is to make the core take the constraint rather than the container: `fixer`'s expansion takes `(type string, enum *[]any)`, so `parser.Parameter`, `parser.Header`, `parser.Items` and `parser.Schema` all reach it despite sharing no type. Before reaching for `.Schema`, ask whether the OAS 2.0 form declares the thing inline
- **`parser.GetOperations` flattens `additionalOperations`, and a reported path has to re-separate it**: the map it returns keys a custom OAS 3.2 method by its own name, beside the standard ones, so a caller building a path from the key writes `paths.{p}.PURGE` where the document says `paths.{p}.additionalOperations.PURGE`. `walker/walk_oas3.go`, `validator/schema_traversal.go` and `converter/oas3_schema_positions.go` all spell it the long way; `fixer.operationPathSegment` is the fourth. The version matters too, since `GetOperations` returns a custom method only at 3.2 and TRACE only at 3.0, and every `fixer` pass reads the version from the document rather than from the `ParseResult` it routed on. A document built in Go carries none, so `fixOAS3` adopts the parse result's version at the entry point rather than leaving each pass to treat it as the oldest OAS 3
- **Discriminator has two dialects**: OAS 2.0 spells `discriminator` as a bare string, OAS 3.0+ as an object. Both decode into `parser.Discriminator`; `StringForm bool` records which. The parser accepts either (it cannot see the document version), the validator rejects the wrong one for the version, and the converter flips the flag in both directions. `StringForm` is excluded from JSON, YAML, and `equalDiscriminator` — it is spelling, not meaning
- **`make check` before pushing** — not just `go test`; catches lint, formatting, and trailing whitespace
- **`docs/` is mixed source + generated**: Source files (`index.md`, `mcp-server.md`, `cli-reference.md`, etc.) are edited directly in `docs/`. Generated files (`docs/packages/`, `docs/examples/`) come from `{package}/deep_dive.md` and `examples/*/README.md` — see `.claude/docs/docs-website.md`
- **MCP config via env vars**: The MCP server reads `OASTOOLS_*` env vars for configuration (cache TTLs, walk limits, join strategies, etc.). The Go MCP SDK doesn't support `initializationOptions`, so env vars are used instead. MCP clients set these via their `env` field in server config.
- **Component-name charset has one definition**: `internal/naming` owns it — `ComponentNamePattern` (string, for error messages), `IsComponentNameChar` (per-rune, for the fixer building replacements), `IsValidComponentName` (whole-name, for the validator). Never restate the pattern elsewhere; both `validator` and `fixer` consume `internal/naming`. A drift-guard test compares the compiled pattern against the predicate over every rune through U+0700
- **OAS versions disagree on schema-name legality, deliberately**: OAS 3.x Components keys are an *allowlist* (`^[a-zA-Z0-9._-]+$`), so a denylist can never keep up — `pkg/Pet`, `Pet@v1`, `pet~summary`, `Pét` are all illegal without sharing a character. OAS 2.0 places *no* charset constraint on `definitions` keys, so those names are valid and renaming them would rewrite a valid document; the fixer's character denylist applies there only. `fixer.charsetForVersion` is the switch
- **`$ref` tokens: look up exact-first, decoded-second**: Generators mix escaping conventions (e.g. percent-encoding brackets while leaving slashes raw — neither pure RFC 6901 nor pure percent-encoding). `pathutil.DecodeRefToken` reverses both, but it's lossy: a component genuinely named `Foo%20Bar` decodes to `Foo Bar` and stops matching itself. Index rename maps via `fixer.lookupRenamedRef` (checks the exact ref first, decoded second), and register decoded keys in sorted order — two names can share a decoded form without either being it, making map-range order-dependent

## Orchestrator Mode

**Default behavior**: Act as an orchestrator, not an implementer.

### When to Delegate

| Task Type | Agent |
|-----------|-------|
| Research/exploration | `general-purpose` |
| Architecture/planning | `architect` |
| Implementation/coding | `developer` |
| Code review/security | `maintainer` |
| Release/deployment | `devops-engineer` |

### When to Handle Directly

- Simple questions answerable from context
- Clarifying user intent
- Synthesizing agent results
- Coordinating multi-agent workflows

## Deep Dives (read when needed)

| Topic | File |
|-------|------|
| OAS concepts & pitfalls | `.claude/docs/oas-concepts.md` |
| Error handling patterns | `.claude/docs/error-handling.md` |
| Testing requirements | `.claude/docs/testing-requirements.md` |
| Benchmark guide | `.claude/docs/benchmark-guide.md` |
| gopls workflow | `.claude/docs/gopls-workflow.md` |
| New package checklist | `.claude/docs/new-package-checklist.md` |
| Make commands | `.claude/docs/make-commands.md` |
| Docs website | `.claude/docs/docs-website.md` |

## Go Module

- **Module**: `github.com/erraggy/oastools`
- **Minimum Go**: 1.25
