# Codebase Polish: DRY, Dead Code, and Structural Cleanup

**Date**: 2026-02-10
**Status**: Design
**Constraint**: Public API must NOT break. All changes are behind-the-scenes.

## Motivation

oastools has reached feature completion. This cleanup removes code duplication,
dead code, over-abstraction, and test waste — while preserving the entire public API.

## Findings Summary

| Category | Items | Est. Lines Removed |
|----------|-------|--------------------|
| Dead code (internal only) | 2 symbols | ~15 |
| Stdlib replacements + wrapper removal | 7 functions | ~60 |
| Shared utility extraction | 3 clusters (17 duplicates) | ~65 |
| Over-abstraction flattening | 2 interfaces | ~160 |
| Validation consolidation + file splits | 3 helpers.go files | ~20 + readability |
| Test cleanup (coverage-gated) | 2 test files + shared fixtures | ~500-1,000 |
| **Total** | | **~820-1,320** |

## Execution Plan

Three waves of parallel agents, sequenced so each wave's outputs feed the next.

---

### Wave 1 — Independent Foundations (6 parallel agents)

All agents touch completely separate files. No conflicts possible.

#### Agent 1: Dead Code Removal (Phase 1)
**Files**: `internal/jsonpath/jsonpath.go`, `internal/corpusutil/corpus.go`
**Actions**:
- Delete `Segment.segmentType()` interface method and all 6 implementations from `internal/jsonpath/jsonpath.go`
- Delete `SkipIfEnvSet()` from `internal/corpusutil/corpus.go`
- Run `go_diagnostics` on both files
- Run `go test ./internal/jsonpath/ ./internal/corpusutil/`

#### Agent 2: Flatten builder constraintTarget (Phase 4a)
**Files**: `builder/constraints.go`, `builder/builder.go` (callers)
**Actions**:
- Remove `constraintTarget` interface (12 methods)
- Remove `schemaConstraintAdapter` struct and its 12 methods
- Remove `paramConstraintAdapter` struct and its 12 methods
- Remove `applyConstraintsToTarget()` function
- Create `applyConstraintsToSchema(schema *parser.Schema, cfg *paramConfig)` with direct field assignment
- Create `applyConstraintsToParam(param *parser.Parameter, cfg *paramConfig)` with direct field assignment
- Update the 2 call sites in builder
- Run `go_diagnostics` on changed files
- Run `go test ./builder/`

#### Agent 3: Flatten fixer refFieldSource + remove fixer deepCopy wrappers (Phase 4b + partial Phase 2b)
**Files**: `fixer/refs.go`, `fixer/helpers.go`, fixer callers
**Actions**:
- Remove `refFieldSource` interface and `parameterRefAdapter`/`headerRefAdapter` from `fixer/refs.go`
- Replace with direct functions or type switch in `collectRefFieldSourceRefs`
- Remove `deepCopyOAS2Document()` and `deepCopyOAS3Document()` from `fixer/helpers.go`
- Replace all call sites with direct `.DeepCopy()` calls
- Run `go_diagnostics` on changed files
- Run `go test ./fixer/`

#### Agent 4: Extract sortedMapKeys (Phase 3b)
**Files**: `internal/maputil/keys.go` (new), `walker/walk_shared.go`, `joiner/rename_context.go`, `generator/security_gen_shared.go`
**Actions**:
- Create `internal/maputil/keys.go` with `func SortedKeys[V any](m map[string]V) []string`
- Replace `sortedMapKeys` in `walker/walk_shared.go` with import
- Replace `sortedMapKeys` in `joiner/rename_context.go` with import
- Replace `sortedPathKeys` in `generator/security_gen_shared.go` with import (verify `parser.Paths` is `map[string]*PathItem`)
- Run `go_diagnostics` on all changed files
- Run `go test ./internal/maputil/ ./walker/ ./joiner/ ./generator/`

#### Agent 5: Create testutil.Ptr[T] (Phase 3c)
**Files**: `internal/testutil/ptr.go` (new), 8 test files
**Actions**:
- Create `internal/testutil/ptr.go`:
  ```go
  package testutil

  func Ptr[T any](v T) *T { return &v }
  ```
- Replace all 12 pointer helper variants across test files:
  - `parser/schema_test_helpers_test.go` — `ptr()`, `intPtr()`, `boolPtr()`
  - `builder/parameter_constraints_test.go` — `ptrFloat64()`, `ptrInt()`
  - `builder/parameter_inline_test.go` — `ptrIntInline()`
  - `differ/schema_test.go` — `ptrInt()`
  - `differ/unified_schema_constraints_test.go` — `ptrIntTest()`, `ptrFloat64Test()`
  - `httpvalidator/response_test.go` — `intPtr()`
  - `httpvalidator/params_test.go` — `boolPtr()`
  - `internal/schemautil/hash_bench_test.go` — `intPtr()`
- Run `go_diagnostics` on all changed files
- Run `go test` for each affected package

#### Agent 6 (background): Coverage Analysis (Phase 6a)
**Files**: None modified (read-only)
**Actions**:
- Run: `go test -coverprofile=conv.out ./converter/ && go tool cover -func=conv.out`
- Run: `go test -coverprofile=join.out ./joiner/ && go tool cover -func=join.out`
- Capture output for Wave 3 decision-making
- Specifically check coverage of functions tested in `converter/helpers_test.go` and `joiner/helpers_test.go`

---

### Wave 2 — Cross-Package Consolidation (4 parallel agents)

Depends on Wave 1 completing. Each agent touches a distinct set of packages.

#### Agent 7: Converter cleanup + split (Phase 2b partial + Phase 5d)
**Files**: `converter/helpers.go` → split into multiple files
**Depends on**: Wave 1 (Agent 3 removed the pattern from fixer; converter deepCopy wrappers still need removal here)
**Actions**:
- Remove `deepCopyOAS3Document()` method and `deepCopySchema()` method from `converter/helpers.go`
- Replace call sites with direct `.DeepCopy()` calls
- Split remaining `converter/helpers.go` into:
  - `converter/ref_rewrite.go` — all `rewriteRef*`, `walkSchemaRefs`, `rewriteAllRefs*` functions (~430 lines)
  - `converter/server_url.go` — `parseServerURL` (~25 lines)
  - `converter/schema_convert.go` — `convertOAS2SchemaToOAS3`, `convertOAS3SchemaToOAS2` (~40 lines)
  - `converter/helpers.go` — remaining functions (~240 lines)
- Run `go_diagnostics` on all new/changed files
- Run `go test ./converter/`

#### Agent 8: Validator consolidation + split (Phase 3a + Phase 5a + Phase 5f)
**Files**: `internal/pathutil/` (add regex), `internal/stringutil/validate.go` (new), `validator/helpers.go` → split, `fixer/path_parameters.go`, `httpvalidator/schema.go`
**Actions**:
- Add `PathParamRegex` to `internal/pathutil/path.go` (or new file)
- Update `validator/helpers.go` and `fixer/path_parameters.go` to import from `internal/pathutil`
- Create `internal/stringutil/validate.go` with canonical `IsValidEmail()` (single regex)
- Update `validator/helpers.go` and `httpvalidator/schema.go` to use shared email validation
- Split `validator/helpers.go` into:
  - `validator/path_validation.go` — `validatePathTemplate`, `checkTrailingSlash`, `extractPathParameters` (~90 lines)
  - `validator/format_validation.go` — `isValidURL`, `isValidEmail` (now delegates), `isValidMediaType`, `validateSPDXLicense`, `getJSONSchemaRef` (~70 lines)
  - `validator/helpers.go` — `validateInfoObject`, `validateResponseStatusCodes`, `checkDuplicateOperationIds` (~150 lines)
- Run `go_diagnostics` on all changed files
- Run `go test ./validator/ ./fixer/ ./httpvalidator/ ./internal/pathutil/ ./internal/stringutil/`

#### Agent 9: Generator helpers split (Phase 5e)
**Files**: `generator/helpers.go` → split into multiple files
**Actions**:
- Split `generator/helpers.go` into:
  - `generator/naming.go` — `escapeReservedWord`, `toTypeName`, `toFieldName`, `toParamName`, `operationToMethodName`, `operationInfoToMethodName`, `generateMethodNameFromPathMethod`, `cleanDescription`, `formatMultilineComment` (~160 lines)
  - `generator/type_mapping.go` — `getSchemaType`, `stringFormatToGoType`, `integerFormatToGoType`, `numberFormatToGoType`, `paramTypeToGoType`, `needsTimeImport`, `zeroValue`, `schemaTypeFromMap` (~140 lines)
  - `generator/helpers.go` — `isRequired`, `buildDefaultUserAgent`, `formatAndFixImports`, `isSelfReference`, `buildTypeGroupMaps` (~110 lines)
- Run `go_diagnostics` on all new/changed files
- Run `go test ./generator/`

#### Agent 10: Replace contains with slices.Contains (Phase 2a)
**Files**: `builder/reflect.go`, `integration/harness/pipeline.go`, `parser/oas2_json_test.go`
**Actions**:
- In `builder/reflect.go`: delete `contains()` function, replace calls with `slices.Contains()`, add `slices` import
- In `integration/harness/pipeline.go`: delete `containsString()`, replace calls with `slices.Contains()`
- In `parser/oas2_json_test.go`: inspect `contains()` — this is substring matching (`strings.Contains`), NOT slice membership. Replace with `strings.Contains()` if not already, or leave if it's already correct. Do NOT replace with `slices.Contains`.
- Run `go_diagnostics` on all changed files
- Run `go test ./builder/ ./parser/`

---

### Wave 3 — Test Cleanup (1-2 agents, depends on coverage data)

Depends on Wave 1 Agent 6 (coverage analysis) and Wave 2 completing.

#### Agent 11: Test file cleanup (Phase 6b + 6c + 6d)
**Files**: `converter/helpers_test.go`, `joiner/helpers_test.go`, `internal/testutil/fixtures.go` (new), test files in converter/generator/joiner/differ
**Actions**:
- Review coverage data from Agent 6
- If converter internal functions have >95% coverage without `helpers_test.go`:
  - Delete `converter/helpers_test.go` (502 lines)
- If joiner internal functions have >95% coverage without `helpers_test.go`:
  - Delete `joiner/helpers_test.go` (591 lines)
- If coverage gaps exist, keep only the tests that fill gaps, delete the rest
- Create `internal/testutil/fixtures.go` with:
  - `MinimalOAS2() *parser.OAS2Document`
  - `MinimalOAS3() *parser.OAS3Document`
- Replace duplicated test document builders in converter, generator, joiner, differ tests
- Run `go test` for all affected packages
- Run `make check` as final validation

---

## Final Validation

After all waves complete:
- `make check` must pass (tests + lint + formatting)
- `go test ./...` must pass with same test count (minus intentionally deleted tests)
- No exported symbols removed (public API preserved)

## Files Created (New)

| File | Purpose |
|------|---------|
| `internal/maputil/keys.go` | Generic `SortedKeys[V any]` |
| `internal/testutil/ptr.go` | Generic `Ptr[T any]` |
| `internal/testutil/fixtures.go` | Shared test document builders |
| `internal/stringutil/validate.go` | Canonical email validation |
| `converter/ref_rewrite.go` | Ref transformation functions (moved from helpers.go) |
| `converter/server_url.go` | URL parsing (moved from helpers.go) |
| `converter/schema_convert.go` | Schema conversion (moved from helpers.go) |
| `validator/path_validation.go` | Path validation (moved from helpers.go) |
| `validator/format_validation.go` | Format validation (moved from helpers.go) |
| `generator/naming.go` | Name transformation (moved from helpers.go) |
| `generator/type_mapping.go` | Type mapping (moved from helpers.go) |

## Files Deleted

| File | Lines | Condition |
|------|-------|-----------|
| `converter/helpers_test.go` | 502 | Coverage >95% without it |
| `joiner/helpers_test.go` | 591 | Coverage >95% without it |

## Risk Assessment

- **Wave 1**: Zero risk — internal dead code, self-contained refactors, test-only changes
- **Wave 2**: Low risk — file splits are mechanical moves, validation consolidation is straightforward
- **Wave 3**: Low risk — coverage-gated, only delete what's provably redundant
- **Public API**: No exported symbols added, removed, or changed in any phase
