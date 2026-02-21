# Documentation Accuracy Audit & CI Prevention Strategy

**Date:** 2026-02-21
**Scope:** whitepaper, deep_dive.md files (x11), cli-reference.md, breaking-changes.md
**Status:** Audit complete. Remediation pending.

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Audit Findings: Whitepaper](#audit-findings-whitepaper)
3. [Audit Findings: Deep Dive Files](#audit-findings-deep-dive-files)
4. [Audit Findings: CLI Reference](#audit-findings-cli-reference)
5. [CI Prevention Strategy](#ci-prevention-strategy)
6. [Implementation Priority](#implementation-priority)

---

## Executive Summary

An audit of all documentation files against the actual Go source code found **~20 confirmed discrepancies** across three categories:

| Category | Count | Severity |
|----------|:-----:|----------|
| Whitepaper: fabricated/wrong API calls | 5 | CRITICAL — code blocks won't compile |
| Whitepaper: incomplete/wrong struct docs | 2 | HIGH — misleads library users |
| CLI reference: missing flags | 10+ | HIGH — users can't discover features |
| CLI reference: wrong descriptions | 4 | MEDIUM |
| Deep dive files: wrong function signatures | 2 | MEDIUM — builder security APIs |

The deep_dive files are in surprisingly good shape overall (only 2 issues found across 11 files). The whitepaper's "Package Chaining" section and the CLI reference's flag tables are the two highest-priority remediation targets.

---

## Audit Findings: Whitepaper

File: `docs/whitepaper.md`

### CRITICAL: Package Chaining Example (lines 230-238)

The entire code block uses functions that **do not exist**:

```go
// WHITEPAPER SHOWS (line 231-237):
result, _ := parser.Parse("spec.yaml", false, true)
validated := validator.ValidateParsed(result, true, false)
fixed := fixer.FixParsed(validated.ToParseResult())
converted := converter.ConvertParsed(fixed.ToParseResult(), "3.1.0")
joined := joiner.JoinParsed(converted.ToParseResult(), other.ToParseResult())
```

**None of these functions exist.** The actual APIs are:

| Whitepaper Function | Actual Function | Source Location |
|---|---|---|
| `parser.Parse("spec.yaml", false, true)` | `parser.ParseWithOptions(opts ...Option)` | `parser/parser_options.go:58` |
| `validator.ValidateParsed(result, ...)` | `validator.ValidateWithOptions(opts ...Option)` | `validator/validator.go:148` |
| `fixer.FixParsed(result)` | `fixer.FixWithOptions(opts ...Option)` | `fixer/fixer.go:203` |
| `converter.ConvertParsed(result, "3.1.0")` | `converter.ConvertWithOptions(opts ...Option)` | `converter/converter.go` |
| `joiner.JoinParsed(a, b)` | `joiner.JoinWithOptions(opts ...Option)` | `joiner/joiner_options.go:73` |

**Fix:** Rewrite the chaining example to use the actual `*WithOptions` + functional option pattern. The `ToParseResult()` methods on result types do exist (added in v1.40-v1.41), so the chaining concept is valid — just the function names are wrong.

### HIGH: ValidationError Struct Definition (lines 499-508)

The whitepaper defines `ValidationError` as a standalone struct with explicit fields. In reality:

```go
// validator/validator.go:37
type ValidationError = issues.Issue   // type ALIAS, not a struct definition
```

The `issues.Issue` struct does have the documented fields, so the field list is correct. But presenting it as a `validator.ValidationError` struct definition is misleading — it's a type alias to `issues.Issue`.

**Fix:** Either document it as `issues.Issue` (the canonical type) or note that `ValidationError` is a type alias.

### HIGH: CollisionContext Missing Fields (lines 862-872)

The whitepaper shows 8 fields. The actual struct (`joiner/collision_handler.go:120-147`) has **11 fields**. Missing from docs:

| Missing Field | Type | Purpose |
|---|---|---|
| `LeftLocation` | `*SourceLocation` | Line/column info for left value |
| `RightLocation` | `*SourceLocation` | Line/column info for right value |
| `RenameInfo` | `*RenameContext` | Operation context for renaming |

**Fix:** Add the three missing fields to the whitepaper's struct definition.

### MEDIUM: Functional Options vs Struct Pattern (lines 322-343)

The "Dual Configuration Patterns" section (4.5) shows both patterns correctly, but the struct-based pattern at line 337-343 uses direct field assignment (`p.ResolveRefs = true`), which works but is not the recommended pattern — the functional options pattern (`WithResolveRefs(true)`) is canonical.

**Fix:** Consider noting that the struct pattern is supported but the functional options pattern is preferred.

---

## Audit Findings: Deep Dive Files

Overall status: **11 files audited, 2 issues found (builder package only)**. The parser, validator, fixer, converter, joiner, differ, walker, generator, overlay, and httpvalidator deep_dive.md files are accurate.

### MEDIUM: Builder `WithSecurity` Wrong Parameter Type

**File:** `builder/deep_dive.md`, line 680 and line 1184

Doc shows:
```go
builder.WithSecurity([]string{"bearerAuth"})
```

Actual signature (`builder/operation.go`):
```go
func WithSecurity(requirements ...parser.SecurityRequirement) OperationOption
```

Correct usage:
```go
builder.WithSecurity(parser.SecurityRequirement{"bearerAuth": []string{}})
```

**Fix:** Update the code example and the configuration table entry to use `parser.SecurityRequirement` instead of `[]string`.

### MEDIUM: Builder Security Methods Misclassified

**File:** `builder/deep_dive.md`, lines 1176-1187

The "Security Options" table lists three items as if they're all the same kind of option:

| Doc Entry | Actual Kind |
|---|---|
| `WithSecurity([]string)` | `OperationOption` (correct placement, wrong type) |
| `AddSecurityScheme(name, scheme)` | `Builder` method, NOT an option |
| `SetSecurity(requirements...)` | `Builder` method, NOT an option |

**Fix:** Split the table — `WithSecurity` stays in Operation Options (with corrected signature), while `AddSecurityScheme` and `SetSecurity` should be documented as Builder methods.

---

## Audit Findings: CLI Reference

File: `docs/cli-reference.md`

### HIGH: Missing Flags (10+ undocumented flags)

#### JOIN Command — 6 missing flags (most impactful)

Source: `cmd/oastools/commands/join.go`, lines 115-127

| Flag | Description | Line in source |
|---|---|---|
| `--equivalence-mode` | Schema equivalence mode for dedup | join.go:115 |
| `--collision-report` | Generate collision report file | join.go:116 |
| `--semantic-dedup` | Enable semantic deduplication | join.go:117 |
| `--namespace-prefix` | Prefix for namespaced components | join.go:120 |
| `--always-prefix` | Always add prefix even without collisions | join.go:121 |
| `--operation-context` | Include operation context in collisions | join.go:124 |
| `--primary-operation-policy` | Policy for primary file operations | join.go:126 |

**Fix:** Add a dedicated "Advanced Flags" table for the JOIN command.

#### PARSE Command — 2 missing flags

Source: `cmd/oastools/commands/parse.go`, lines 28-31

| Flag | Description |
|---|---|
| `--resolve-http-refs` | Resolve HTTP/HTTPS `$ref` references |
| `--insecure` | Skip TLS verification for HTTP refs |

**Fix:** Add to the PARSE flags table.

#### VALIDATE Command — 1 missing flag

Source: `cmd/oastools/commands/validate.go`, line 39

| Flag | Description |
|---|---|
| `--include-document` | Include full OAS document in JSON/YAML output |

**Fix:** Add to the VALIDATE flags table.

#### FIX Command — 2 missing flags

Source: `cmd/oastools/commands/fix.go`, lines 79-80

| Flag | Description |
|---|---|
| `--operationid-path-sep` | Path separator for operation ID templates |
| `--operationid-tag-sep` | Tag separator for operation ID templates |

**Fix:** Add to the FIX flags table.

#### WALK Command — Missing subcommand flag tables

The `walk operations`, `walk schemas`, etc. subcommands have specific filter flags (`--method`, `--path`, `--tag`, `--deprecated`, `--operationId`) documented in examples but not in a formal flags table.

**Fix:** Add subcommand-specific flags tables.

### MEDIUM: Wrong/Incomplete Descriptions

| Location | Flag | Issue |
|---|---|---|
| Line 477 | `--source-map` (convert) | Says "in output" but source says "in conversion issues" |
| Line 848 | `--source-map` (diff) | Minor wording difference vs source |
| Line 995 | `--types` (generate) | Default `true` is confusing — needs note about behavior with `--client`/`--server` |
| Line 613 | collision strategy name | Doc says `dedup-equivalent`, code says `deduplicate` |

### LOW: breaking-changes.md

**File:** `docs/breaking-changes.md`, line 309

The CI/CD example uses `grep -q "CRITICAL\|ERROR"` which won't work with `--format json/yaml`. Should note this assumes text output format.

---

## CI Prevention Strategy

Each layer targets a specific drift pattern, ranked by effort-to-impact ratio.

### Layer 1: Link Checking with `lychee`

**Catches:** Dead links, broken anchors, missing files (e.g., the `benchmarks.md` reference)
**Effort:** ~1 hour
**Tooling:** [lychee](https://github.com/lycheeverse/lychee) — fast Rust-based link checker

#### Why this needs `docs-prepare` first

The `docs/` directory has a two-phase assembly:
- **Source files** (`docs/*.md`, `*/deep_dive.md`) are checked into git
- **Generated files** (`docs/packages/`, `docs/examples/`) are created by `scripts/prepare-docs.sh` and gitignored

Source files link to the *assembled* paths — e.g., `docs/whitepaper.md` links to `packages/parser.md`, which only exists after `prepare-docs.sh` copies `parser/deep_dive.md` → `docs/packages/parser.md`. A naive `lychee docs/*.md` would false-positive on every `packages/` and `examples/` link.

**Solution:** Run lychee after `docs-prepare`, scoped to the assembled `docs/` directory:

```makefile
## lint-links: Check for broken links in assembled docs
.PHONY: lint-links
lint-links: docs-prepare
	@echo "Checking links..."
	@if command -v lychee >/dev/null 2>&1; then \
		lychee --no-progress --base docs/ \
			--exclude 'localhost|127\.0\.0\.1' \
			--exclude 'pkg\.go\.dev' \
			--exclude 'oastools\.robnrob\.com' \
			docs/; \
	else \
		echo "lychee not found. Install: cargo install lychee (or brew install lychee). Skipping link check."; \
	fi
```

Key flags:
- **`--base docs/`** — resolves relative links from the `docs/` directory (matching mkdocs' resolution)
- **Depends on `docs-prepare`** — ensures `docs/packages/` and `docs/examples/` are populated
- **External URL exclusions** — `pkg.go.dev` rate-limits aggressively; `oastools.robnrob.com` is the live site (would cause flaky CI)

Deep dive source files (`*/deep_dive.md`) contain **no cross-file links** — they're self-contained. They get checked through their copies in `docs/packages/`.

GitHub Actions integration (add to `.github/workflows/docs.yml`):

```yaml
- uses: lycheeverse/lychee-action@v1
  with:
    args: --no-progress --base docs/ --exclude 'localhost|127\.0\.0\.1|pkg\.go\.dev|oastools\.robnrob\.com' docs/
    fail: true
```

Configuration via `.lychee.toml` for caching and additional exclude patterns:

```toml
# .lychee.toml
cache = true
max_cache_age = "1d"
exclude = ["localhost", "127.0.0.1", "pkg.go.dev", "oastools.robnrob.com"]
exclude_path = ["node_modules", "vendor", "site"]
```

Do NOT add `lint-links` to `make check` — it depends on `docs-prepare` (which requires the prepare script and has side effects on `docs/`). Instead, add it to the docs workflow in CI only:

```makefile
## docs-check: Full docs validation (prepare + link check)
.PHONY: docs-check
docs-check: docs-prepare lint-links
```

**Key point:** This is complementary to `markdownlint-cli2` (already in `make lint-md`). markdownlint checks *formatting*; lychee checks *where links point*.

---

### Layer 2: CLI Flag Table Verification via Test

**Catches:** Flag drift — the single largest category of issues found (10+ missing flags)
**Effort:** 2-4 hours
**Tooling:** stdlib only (`flag`, `go/ast`, `strings`)

The CLI uses stdlib `flag.FlagSet` (no Cobra), so `cobra/doc` is not applicable. Instead, write a test that extracts registered flags from each command's `flag.FlagSet` and compares against the `cli-reference.md` flag tables.

Implementation approach:

1. Each `Handle*` function in `cmd/oastools/commands/` creates a `flag.FlagSet` and registers flags via `fs.BoolVar`, `fs.StringVar`, etc.
2. A `TestCLIFlagsDocumented` test can call `flag.FlagSet.VisitAll()` to enumerate all registered flags, then parse `docs/cli-reference.md` for the corresponding flag table rows.

```go
// cmd/oastools/commands/cli_doc_test.go
func TestCLIFlagsDocumented(t *testing.T) {
    // Build map of command -> registered flags by calling each command's
    // flag setup function (extract into a helper if needed)
    commands := map[string]*flag.FlagSet{
        "validate": setupValidateFlags(),
        "parse":    setupParseFlags(),
        "join":     setupJoinFlags(),
        // ... etc
    }

    // Parse cli-reference.md for documented flags per command
    docFlags := parseFlagTablesFromMarkdown(t, "docs/cli-reference.md")

    for cmd, fs := range commands {
        fs.VisitAll(func(f *flag.Flag) {
            if !docFlags[cmd].Contains(f.Name) {
                t.Errorf("Flag --%s on command %q is registered but not documented in cli-reference.md", f.Name, cmd)
            }
        })
    }
}
```

This requires refactoring flag setup into callable helpers (currently inline in `Handle*` functions). The alternative is AST-based extraction of `fs.BoolVar`/`fs.StringVar` calls from the source files — no refactoring needed, but more brittle.

**Pattern:** Same "source is truth, test verifies docs match" pattern as Layer 3, but focused on CLI flags instead of Go API options.

---

### Layer 3: AST-Based Struct Table Verification

**Catches:** Option/field table drift in deep_dive.md files — when `With*` functions are added/removed/renamed without updating doc tables
**Effort:** 1-2 days
**Tooling:** `go/ast` + `go/doc` (stdlib)

Write a `TestDocOptionsTablesInSync` test per package:

```go
func TestDocOptionsTablesInSync(t *testing.T) {
    // 1. Use go/ast to enumerate all exported With* functions
    exportedOpts := extractWithFunctions(t, ".")

    // 2. Parse deep_dive.md and extract the Configuration Reference table rows
    docOpts := extractMarkdownTableColumn(t, "deep_dive.md", "Configuration Reference", 0)

    // 3. Every exported With* function must appear in the table
    for _, opt := range exportedOpts {
        if !slices.Contains(docOpts, opt) {
            t.Errorf("Option %s exists in source but missing from deep_dive.md table", opt)
        }
    }

    // 4. Every table row must reference an existing With* function
    for _, doc := range docOpts {
        if !slices.Contains(exportedOpts, doc) {
            t.Errorf("deep_dive.md documents %s but no such function exists", doc)
        }
    }
}
```

This runs in `go test ./...` — no external tooling needed. When someone adds a `WithFoo()` option but forgets the table row, the test fails.

---

### Layer 4: `mdcode` for Code Block Sync

**Catches:** Wrong API names in code examples — the pattern where prose is written from memory and drifts from actual source
**Effort:** 1 day to annotate all files, then ongoing discipline
**Tooling:** [`mdcode`](https://github.com/szkiba/mdcode)

Workflow:

1. In `example_test.go`, mark regions:
   ```go
   // #region parse-basic
   result, err := parser.ParseWithOptions(
       parser.WithFilePath("api.yaml"),
   )
   // #endregion
   ```

2. In `deep_dive.md`, link code fences to those regions:
   ````markdown
   ```go file=example_test.go region=parse-basic
   result, err := parser.ParseWithOptions(
       parser.WithFilePath("api.yaml"),
   )
   ```
   ````

3. CI check: `mdcode check parser/deep_dive.md` — fails if markdown diverges from source

This creates a single source of truth chain: `example_test.go` → `go test` verifies it compiles and runs → `mdcode check` verifies the markdown matches.

---

### Layer 5: `testscript` for CLI Bash Examples

**Catches:** Broken CLI examples in cli-reference.md and breaking-changes.md
**Effort:** 2-3 days
**Tooling:** [`github.com/rogpeppe/go-internal/testscript`](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript)

Example `.txtar` file:

```
# testdata/script/validate_basic.txtar
exec oastools validate petstore.yaml
stdout 'valid'
! stderr .

-- petstore.yaml --
openapi: "3.0.0"
info:
  title: Petstore
  version: "1.0.0"
paths: {}
```

This is the bash-example equivalent of Go's `Example*` functions — it verifies CLI behavior matches documented examples.

---

### Layer 6: Existing Infrastructure (Already Working)

These mechanisms are already in place and should be preserved:

| Mechanism | What It Catches | Location |
|---|---|---|
| 151 `Example*` functions | Broken API examples compile & run | `*/example_test.go` |
| `make lint-md` | Markdown formatting issues | `markdownlint-cli2` |
| `make check` | Comprehensive pre-commit gate | `Makefile:192` |
| 70% patch coverage requirement | New code without tests | CI |

The gap: **none of these connect prose documentation to source code**. The code examples in `example_test.go` are correct, but the *copies* of those examples in markdown drift silently. Layers 1-5 close this gap.

---

## Implementation Priority

| Priority | Layer | What to Add | Effort | Catches |
|:--------:|:-----:|---|---|---|
| **P0** | — | Fix the 5 fabricated functions in whitepaper lines 230-238 | 30 min | Critical broken examples |
| **P0** | — | Add 10+ missing CLI flags to cli-reference.md | 1-2 hours | Undiscoverable features |
| **P1** | 1 | `lychee` link checking in `make docs-check` + CI | 1 hour | Dead links forever |
| **P1** | 2 | CLI flag table verification test | 2-4 hours | CLI flag drift forever |
| **P2** | 3 | AST-based option table tests | 1-2 days | Struct/option table drift |
| **P2** | — | Fix whitepaper CollisionContext (add 3 fields) | 15 min | Incomplete struct docs |
| **P2** | — | Fix builder deep_dive WithSecurity signature | 15 min | Wrong parameter type |
| **P3** | 4 | `mdcode` annotations for deep_dive code blocks | 1 day + ongoing | Code example drift |
| **P3** | 5 | `testscript` for CLI bash examples | 2-3 days | Broken CLI examples |

### Recommended Execution Order

1. **Immediate (this PR or next):** Fix the P0 content errors — the whitepaper's fabricated functions and the CLI's missing flags. These are factual errors that mislead users right now.

2. **Short-term (1-2 PRs):** Add lychee (runs after `docs-prepare`, not in `make check`) and CLI flag verification test (Layers 1-2). These prevent the two most common drift patterns from recurring.

3. **Medium-term:** Add AST-based table tests (Layer 3). This prevents the option table drift pattern — the second largest category found in this audit.

4. **Ongoing:** Adopt mdcode annotations incrementally as deep_dive files are updated. No big-bang migration needed.

---

## References

- [Testable Examples in Go](https://go.dev/blog/examples)
- [mdcode — Testable Markdown Code Blocks](https://github.com/szkiba/mdcode)
- [lychee — Fast link checker](https://github.com/lycheeverse/lychee)
- [lychee-action for GitHub](https://github.com/lycheeverse/lychee-action)
- [flag.FlagSet.VisitAll](https://pkg.go.dev/flag#FlagSet.VisitAll) — enumerate registered flags programmatically
- [testscript — Script-based integration testing](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript)
- [go/ast package](https://pkg.go.dev/go/ast)
- [go/doc package](https://pkg.go.dev/go/doc)
