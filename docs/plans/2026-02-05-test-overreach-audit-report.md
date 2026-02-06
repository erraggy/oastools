# Test Overreach Audit Report

**Date:** 2026-02-05
**Scope:** All 28 packages in `github.com/erraggy/oastools`
**Methodology:** Static analysis via gopls + JetBrains inspections + grep-based reference tracing
**Audited by:** Claude Opus 4.6 (5 parallel analysis agents)

---

## Executive Summary

| Metric | Value |
|--------|-------|
| Total test functions analyzed | 2,849 |
| Total test lines | 128,784 |
| Total source lines | 79,891 |
| Test-to-source ratio | 1.61:1 |
| **Category B findings (test infrastructure tests)** | **12 tests (~314 lines)** |
| **Category C findings (redundant internal tests)** | **~42 tests (~2,870 lines)** |
| **Category D findings (dead code tests)** | **0** |
| **Total removable test lines** | **~3,184 (2.5% of test code)** |
| **Infrastructure issues** | **1 (misplaced file)** |

**Bottom line:** The codebase has excellent test hygiene overall. IntelliJ inspections found zero dead code. The overreach is concentrated in two packages: **fixer/** (~2,100 lines) and **validator/** (~725 lines), both of which have comprehensive integration tests that make their helper unit tests redundant. Most packages (20 of 28) have zero overreach.

---

## Findings by Package

### Cluster 1: Core Pipeline

#### package: parser/
**Public API surface:** 83 exported functions, 45 exported types
**Test inventory:** 471 test functions, 29,666 lines
**Overreach findings:** 0 test removals

**Infrastructure issue:** `parser/schema_test_helpers.go` is a production file (no `_test.go` suffix) containing only test helper functions (`ptr()`, `intPtr()`, `boolPtr()`). These are never called from production code. Should be renamed to have a `_test.go` suffix to prevent inclusion in production binaries.

**Observations:** Despite being the largest package (39,899 production lines), parser has zero test overreach. All 13 helper test functions in `equals_helpers_test.go` (1,482 lines) test unexported helpers that ARE transitively called from exported `Equals()` methods. Test organization is exemplary.

#### package: parser/internal/jsonhelpers/
**Public API surface:** 19 exported functions, 2 exported types
**Test inventory:** 18 test functions, 734 lines
**Overreach findings:** 0

All tests target exported helper functions used throughout the parser.

#### package: internal/pathutil/
**Public API surface:** 14 exported functions, 1 exported type
**Test inventory:** 22 test functions, 320 lines
**Overreach findings:** 0

No unexported functions exist in this package. All tests target public API.

---

### Cluster 2: Validation & Fixing

#### package: validator/
**Public API surface:** 13 exported functions/types
**Test inventory:** 140 test functions, 6,730 lines
**Overreach findings:** 12 candidates (est. 725 lines removable)

| Test Function | Category | Est. Lines | Rationale |
|---------------|----------|------------|-----------|
| TestExtractPathParameters | C | 43 | Tests unexported `extractPathParameters()`. Covered by integration tests validating path templates. |
| TestIsValidMediaType | C | 25 | Tests unexported `isValidMediaType()`. Covered by media type validation integration tests. |
| TestIsValidURL | C | 22 | Tests unexported `isValidURL()`. Covered by OAuth2 flow validation tests. |
| TestIsValidEmail | C | 23 | Tests unexported `isValidEmail()`. Covered by info object validation tests. |
| TestValidateSPDXLicense | C | 22 | Tests unexported `validateSPDXLicense()`. Covered by info validation tests. |
| TestPopulateIssueLocation | C | 38 | Tests unexported `populateIssueLocation()`. Covered by error location tests. |
| TestAddError | C | 20 | Tests unexported `addError()`. Covered by every validation test that checks for errors. |
| TestAddWarning | C | 16 | Tests unexported `addWarning()`. Covered by warning-enabled tests. |
| TestNormalizeRef | C | 23 | Tests unexported `normalizeRef()`. Covered by ref tracker integration tests. |
| TestGetComponentRoot | C | 24 | Tests unexported `getComponentRoot()`. Covered by ref tracker tests. |
| TestIsReusableComponentPath | C | 35 | Tests unexported `isReusableComponentPath()`. Covered by ref validation tests. |
| TestParseMethod | C | 22 | Tests unexported `parseMethod()`. Covered by operation context tests. |

**Key observations:**
- `helpers_test.go` (264 lines) is pure overreach — every function it tests is already validated through integration tests.
- `ref_tracker_test.go` lines 346-461 (~116 lines) test 4 unexported helpers covered by `TestRefTrackerOAS3`, `TestRefTrackerOAS2`, etc.
- Removing these tests would improve refactoring freedom for internal helpers.

#### package: fixer/
**Public API surface:** 28 exported functions/types
**Test inventory:** 205 test functions, 11,499 lines
**Overreach findings:** 29 candidates (est. 2,100 lines removable)

| Test File | Category C Tests | Est. Lines | Rationale |
|-----------|-----------------|------------|-----------|
| enum_csv_test.go (lines 11-260) | TestIsCSVEnumCandidate, TestExpandCSVEnumValues, TestParseNumericValue, TestGetSchemaType | 250 | All unexported helpers covered by 15 integration tests in fixer_csvenum_test.go |
| fixer_pathparam_test.go (lines 12-117) | TestExtractPathParameters, TestInferParameterType | 106 | Covered by TestFixMissingPathParametersOAS2/OAS3 integration tests |
| generic_names_test.go (lines 17-859) | 12 tests of unexported helpers (hasInvalidSchemaNameChars, parseGenericName, splitTypeParams, transformSchemaName, etc.) | 842 | Covered by TestFixInvalidSchemaNamesOAS3, TestFixNestedGenericTypesOAS3, etc. |
| operationid_test.go (lines 17-862) | TestExpandOperationIdTemplate, TestApplyModifier, TestSanitizePath, TestGetSortedMethods | 845 | Covered by TestFixDuplicateOperationIds (467 lines, 17 integration tests) |
| stub_missing_refs_test.go (lines 674-779) | TestIsLocalRef, TestExtractResponseNameFromRef | 106 | Covered by 30+ stub missing refs integration tests |
| fixer_test.go (lines 46-91) | TestDeepCopyOAS3Document, TestDeepCopyOAS2Document | 46 | Covered by TestMutableInput_OAS3_PreservesOriginal |
| prune_transitive_test.go | TestCollectSchemaRefs_ItemsAsMap, TestIsComponentsEmpty, TestCollectRefsFromMap_AllPaths | ~200 | Covered by pruning integration tests |

**Key observations:**
- **Highest overreach in the codebase.** The fixer package has the most complex internal helper chains, and developers tested at every level of the chain.
- The integration tests in fixer are comprehensive — they exercise the full `Fix()` pipeline with real specs, which transitively covers all the helpers.
- Removing these 2,100 lines would give the most "bang for the buck" in the entire audit.

---

### Cluster 3: Transformation

#### package: joiner/
**Public API surface:** 73 exported functions, 19 exported types
**Test inventory:** 237 test functions, 13,103 lines
**Overreach findings:** 0

All 7 tests in `helpers_test.go` (590 lines) test unexported helpers that implement critical deep-copying logic called from exported joiner functions. The complexity of nested map/slice copying warrants direct testing.

#### package: converter/
**Public API surface:** 19 exported functions, 5 exported types
**Test inventory:** 89 test functions, 4,372 lines
**Overreach findings:** 0

All 8 tests in `helpers_test.go` test unexported methods called from exported `Convert()`. `TestWalkSchemaRefs` (149 lines) validates all 18+ schema reference locations — particularly valuable.

#### package: overlay/
**Public API surface:** 23 exported functions, 13 exported types
**Test inventory:** 50 test functions, 2,420 lines
**Overreach findings:** 0

Clean package with no tests of unexported helpers.

#### package: differ/
**Public API surface:** 26 exported functions, 27 exported types
**Test inventory:** 160 test functions, 8,871 lines
**Overreach findings:** 0

All 10 tests in `unified_helpers_test.go` (352 lines) test helpers implementing critical breaking-change detection logic. These helpers warrant direct testing due to their nuanced severity calculation rules.

---

### Cluster 4: Runtime & Generation

#### package: httpvalidator/
**Public API surface:** 26 exported functions, 9 exported types
**Test inventory:** 131 test functions, 6,416 lines
**Overreach findings:** 0

#### package: generator/
**Public API surface:** 79 exported functions, 21 exported types
**Test inventory:** 256 test functions, 11,909 lines
**Overreach findings:** 1 marginal candidate (est. 45 lines)

| Test Function | Category | Est. Lines | Rationale |
|---------------|----------|------------|-----------|
| TestGetOAS2ParamSchemaType | C | 45 | Tests unexported `oas2CodeGenerator.getOAS2ParamSchemaType()`. Called from production code but is internal implementation detail covered by GenerateWithOptions() tests. |

#### package: builder/
**Public API surface:** 149 exported symbols
**Test inventory:** 333 test functions, 14,070 lines
**Overreach findings:** 0

#### package: walker/
**Public API surface:** 84 exported symbols
**Test inventory:** 249 test functions, 10,764 lines
**Overreach findings:** 0

Only 2 unexported functions exist in this package — excellent API design with minimal unexported surface.

---

### Cluster 5: CLI & Utilities

#### package: cmd/oastools/
**Test inventory:** 2 test functions, 70 lines
**Overreach findings:** 1 candidate (est. 32 lines)

| Test Function | Category | Est. Lines | Rationale |
|---------------|----------|------------|-----------|
| TestLevenshteinDistance | C | 32 | Tests unexported `levenshteinDistance()`, only called from `suggestCommand()` which is itself tested by `TestSuggestCommand`. Algorithm is validated through suggestion behavior. |

#### package: internal/testutil/
**Test inventory:** 11 test functions, 282 lines
**Overreach findings:** 11 candidates (est. 282 lines removable)

| Finding | Category | Lines | Rationale |
|---------|----------|-------|-----------|
| All 11 tests (TestNewSimpleOAS2Document, TestNewDetailedOAS2Document, TestWriteTempYAML, TestDocumentFactoryConsistency, etc.) | B | 282 | Every function in testutil exists solely to support tests. Testing test infrastructure is pure Category B overreach. |

**Architectural finding:** `internal/testutil` is imported by **only one package**: `converter/converter_test.go`. An entire internal package with 150 lines of code + 282 lines of tests exists to serve 10 references in a single test file. This is an architectural consolidation opportunity — move the helpers inline into `converter/converter_test.go`.

#### package: internal/httputil/
**Test inventory:** 6 test functions, 297 lines
**Overreach findings:** 1 candidate (est. 28 lines)

| Test Function | Category | Est. Lines | Rationale |
|---------------|----------|------------|-----------|
| TestStatusCodeConstants | C | 28 | Tests that compile-time constants equal their literal values (StatusCodeLength==3, MinStatusCode==100). Tautological — if changed, function behavior tests would fail. |

#### package: internal/severity/
**Test inventory:** 3 test functions, 74 lines
**Overreach findings:** 1 candidate (est. 17 lines)

| Test Function | Category | Est. Lines | Rationale |
|---------------|----------|------------|-----------|
| TestSeverityConstants | C | 17 | Tests iota-based constant ordinal values. Tautological — the String() tests validate behavior. |

#### Remaining internal packages (jsonpath, naming, options, schemautil, corpusutil, issues, codegen/deepcopy, oaserrors, root)
**Combined overreach findings:** 0

All tests in these packages target exported API. No overreach detected.

---

## Cross-Package Summary

### Top 10 Highest-Impact Removals

| Rank | Location | Category | Lines | Impact |
|------|----------|----------|-------|--------|
| 1 | fixer/generic_names_test.go (lines 17-859) | C | 842 | Helper tests redundant with generic name integration tests |
| 2 | fixer/operationid_test.go (lines 17-862) | C | 845 | Helper tests redundant with 17 integration tests |
| 3 | internal/testutil/ (entire package) | B | 432 | Test infrastructure only used by converter; consolidate |
| 4 | fixer/enum_csv_test.go (lines 11-260) | C | 250 | Helper tests redundant with CSV enum integration tests |
| 5 | validator/helpers_test.go (entire file) | C | 264 | All 9 helpers covered by integration tests |
| 6 | fixer/prune_transitive_test.go (helpers) | C | 200 | Covered by pruning integration tests |
| 7 | validator/ref_tracker_test.go (lines 346-461) | C | 116 | 4 helpers covered by ref tracker integration tests |
| 8 | fixer/fixer_pathparam_test.go (lines 12-117) | C | 106 | Covered by path parameter integration tests |
| 9 | fixer/stub_missing_refs_test.go (lines 674-779) | C | 106 | Covered by 30+ stub integration tests |
| 10 | generator/server_extensions_test.go | C | 45 | Marginal — internal method covered by GenerateWithOptions() |

### Patterns Observed

1. **Fixer is the overreach hotspot.** Its deep helper chains (3-4 levels of unexported functions) led to testing at every level. The integration tests are comprehensive enough to make the helper tests redundant.

2. **Packages with clean API boundaries have zero overreach.** walker (2 unexported functions), overlay (all exported), builder (strong exported surface) — these packages demonstrate that good API design naturally prevents test overreach.

3. **No dead code anywhere.** IntelliJ inspections confirmed zero dead code across all scanned files. Every function is reachable from some code path. This is a well-maintained codebase.

4. **No test-tests-test anti-pattern (except testutil).** Only `internal/testutil` has tests that test test infrastructure. This is the clearest Category B case.

5. **Tautological constant tests** in httputil and severity test that constants equal their literal definitions — a common but low-value testing pattern.

### Recommendations

**Phase 1 — Quick Wins (est. 1 hour)**
- Remove `validator/helpers_test.go` entirely (264 lines)
- Remove 4 helper tests from `validator/ref_tracker_test.go` (116 lines)
- Remove `TestStatusCodeConstants` and `TestSeverityConstants` (45 lines)
- Remove `TestLevenshteinDistance` from `cmd/oastools/main_test.go` (32 lines)
- Rename `parser/schema_test_helpers.go` → add `_test.go` suffix

**Phase 2 — Fixer Cleanup (est. 2-3 hours)**
- Remove helper tests from fixer test files while preserving integration tests
- Target: ~2,100 lines across 6 files
- Verify integration tests still pass after removal

**Phase 3 — Architectural Consolidation (est. 1 hour)**
- Move `internal/testutil` functions into `converter/converter_test.go` as unexported helpers
- Delete `internal/testutil/` package entirely (432 lines)

**Expected outcome:** ~3,184 lines removed (2.5% of test code), zero reduction in public API coverage, improved refactoring freedom for internal helpers.
