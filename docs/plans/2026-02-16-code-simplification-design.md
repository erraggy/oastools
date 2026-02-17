# Code Simplification and Reduction

**Date**: 2026-02-16
**Status**: Approved
**Constraint**: Zero breaking changes to public API

## Motivation

Reduce maintenance burden and improve IDE/LLM ergonomics across the codebase.
A full audit identified two categories of actionable work:

1. **Generator OAS 2/3 duplication** — 36 functions duplicated across two files, ~20 of which are identical. Changes to security, credential, or server generation must be made twice.
2. **Large monolithic files** — 4 production files over 1,000 lines that slow down IDEs and consume excessive LLM context.

## What's NOT in scope

| Item | Reason |
|------|--------|
| Deleting public exports | Zero breaking changes constraint |
| Merging `internal/maputil` + `internal/stringutil` | Already cohesive; merging adds no value |
| Merging `internal/codegen/{decode,deepcopy}` | Separate `package main` executables; Go requires separate directories |
| Deleting `internal/testutil` | Would duplicate `Ptr[T]` into 14 test files — inverse of consolidation |
| Flattening fixer helper chains | Inherent domain complexity, not poor organization (151 funcs, well-organized by concern) |
| Functional options standardization | 12 packages; pattern works fine as-is |
| Equals() codegen | High effort, low maintenance cost |

## Workstream 1: File Splitting

Pure file reorganization within existing packages. No logic changes, no API changes, no test changes.

### parser/parser.go (1,622 lines → 3 files)

| New file | Contents | ~Lines |
|----------|----------|--------|
| `parser/parser.go` | `Parser` struct, `ParseResult`, `Parse`/`ParseBytes`/`ParseReader`, `parseBytesWithBaseDir*` | ~900 |
| `parser/parser_options.go` | `parseConfig` struct, `applyOptions`, all 20 `With*` option constructors | ~350 |
| `parser/parser_format.go` | `detectFormatFromPath`, `detectFormatFromContent`, `detectFormatFromURL`, `isURL`, `fetchURL`, `FormatBytes` | ~350 |

### joiner/joiner.go (1,236 lines → 2 files)

| New file | Contents | ~Lines |
|----------|----------|--------|
| `joiner/joiner.go` | Core join logic, collision handling, merge functions | ~900 |
| `joiner/joiner_options.go` | `joinConfig` struct, `applyOptions`, all `With*` option constructors | ~350 |

### generator/security_gen_shared.go (1,194 lines → 3 files)

| New file | Contents | ~Lines |
|----------|----------|--------|
| `generator/security_oauth2.go` | OAuth2 flow generation, `buildOAuth2TemplateData` | ~450 |
| `generator/security_oidc.go` | OIDC discovery generation, well-known URL handling | ~350 |
| `generator/security_credentials.go` | `CredentialProvider` interface, credential file generation | ~400 |

### integration/harness/pipeline.go (1,294 lines → 3-4 files)

| New file | Contents | ~Lines |
|----------|----------|--------|
| `harness/pipeline.go` | `ExecuteStep` orchestrator, step registry | ~300 |
| `harness/pipeline_parse_validate.go` | parse, validate step executors | ~350 |
| `harness/pipeline_transform.go` | fix, convert, join, overlay step executors | ~350 |
| `harness/pipeline_generate.go` | generate, build step executors | ~300 |

## Workstream 2: Generator Deduplication

### Problem

`oas2_generator.go` (1,170 lines, 45 funcs) and `oas3_generator.go` (1,342 lines, 52 funcs) share 36 function names. Spot-checking reveals three categories:

| Category | ~Count | Difference |
|----------|--------|-----------|
| **Identical** | ~20 | Only receiver type differs |
| **Data-access** | ~10 | Differ in how a field is reached (e.g., `doc.SecurityDefinitions` vs `doc.Components.SecuritySchemes`) |
| **Structurally different** | ~6 | Version-specific logic (body params vs request bodies, allOf handling, etc.) |

### Approach

1. **Create `generator/generate_shared.go`** with a `sharedGenerator` base struct:
   ```go
   type sharedGenerator struct {
       g      *Generator
       result *GenerateResult
       doc    parser.DocumentAccessor
   }
   ```

2. **Move identical functions** to methods on `sharedGenerator` — written once.

3. **Move data-access functions** to `sharedGenerator`, using `DocumentAccessor` for version-agnostic field access (schemas, security schemes, parameters, responses).

4. **Keep structurally different functions** on `oas2CodeGenerator` / `oas3CodeGenerator`.

5. **Embed `sharedGenerator`** in both version-specific generators:
   ```go
   type oas3CodeGenerator struct {
       sharedGenerator
       doc *parser.OAS3Document  // version-specific access when needed
   }
   ```

### Expected result

| File | Before | After |
|------|--------|-------|
| `generate_shared.go` | — | ~800 lines |
| `oas2_generator.go` | 1,170 lines | ~400 lines |
| `oas3_generator.go` | 1,342 lines | ~500 lines |
| **Total** | **2,512 lines** | **~1,700 lines** |

Net reduction: ~800 lines. More importantly, shared logic is maintained in one place.

### Risk and mitigation

**Risk**: Some "data-access" functions may turn out to need version-specific logic not caught during sampling.

**Mitigation**: Audit every shared function body before moving. Any function needing version-specific logic stays in the version-specific file. Err on the side of keeping functions separate.

## PR Strategy

| PR | Scope | Risk | Review effort |
|----|-------|------|--------------|
| PR 1 | File splits (workstream 1) | Very low — pure file moves within packages | Low — verify no logic changes |
| PR 2 | Generator dedup (workstream 2) | Medium — logic restructuring | Higher — verify behavior preservation |

Separate PRs because file splits are trivially safe and can land first, while generator dedup needs careful review.

## Success criteria

- `make check` passes after each PR
- All existing tests pass unchanged (workstream 1) or with minimal test adjustments (workstream 2)
- No public API changes
- No file over 1,000 lines in the affected packages (except generated `zz_*` files)
