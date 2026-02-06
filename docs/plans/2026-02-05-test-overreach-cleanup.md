# Test Overreach Cleanup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove ~3,100 lines of redundant test code identified by the [test overreach audit](./2026-02-05-test-overreach-audit-report.md), with zero reduction in public API coverage.

**Architecture:** Four phases of surgical test removal, each followed by a full-suite checkpoint verifying test count and zero regressions. Phase 1 handles small scattered files. Phase 2 tackles the fixer package (highest overreach). Phase 3 consolidates `internal/testutil` into its sole consumer. Phase 4 renames a misplaced file.

**Tech Stack:** Go 1.24, `go test`, `git mv`

**Baseline (pre-cleanup):**
- Total passing tests: **2,650**
- Total failing tests: **0**
- Full suite command: `go test ./... -count=1`
- Affected packages: validator (152 tests), fixer (214 tests), internal/testutil (11 tests), internal/httputil, internal/severity, cmd/oastools, converter, parser

---

## Phase 1: Quick Wins — Validator + Small Packages

### Task 1: Delete `validator/helpers_test.go`

**Files:**
- Delete: `validator/helpers_test.go` (264 lines, 9 test functions)

**Context:** This file tests 9 unexported helper functions (`extractPathParameters`, `isValidMediaType`, `isValidURL`, `isValidEmail`, `validateSPDXLicense`, `populateIssueLocation`, `addError`, `addWarning`, `validateHTTPStatusCode`) that are all transitively exercised by the validator's integration tests. Every behavior path these helpers take is already covered by exported tests like `TestValidateOAS3Info`, `TestValidateOAS3Paths`, `TestRefTrackerOAS3`, etc.

**Step 1: Run validator tests to confirm baseline**

Run: `go test ./validator/ -count=1 -v 2>&1 | grep -c '^--- PASS'`
Expected: `152`

**Step 2: Delete the file**

```bash
rm validator/helpers_test.go
```

**Step 3: Run validator tests to confirm no regression**

Run: `go test ./validator/ -count=1 -v 2>&1 | grep -c '^--- PASS'`
Expected: `143` (152 minus 9 deleted tests)

Run: `go test ./validator/ -count=1`
Expected: `ok` (zero failures)

**Step 4: Commit**

```bash
git add validator/helpers_test.go
git commit -m "test(validator): remove 9 redundant unexported helper tests

These 9 functions (extractPathParameters, isValidMediaType, isValidURL,
isValidEmail, validateSPDXLicense, populateIssueLocation, addError,
addWarning, validateHTTPStatusCode) are all transitively covered by
validator integration tests.

Ref: docs/plans/2026-02-05-test-overreach-audit-report.md"
```

---

### Task 2: Remove 4 helper tests from `validator/ref_tracker_test.go`

**Files:**
- Modify: `validator/ref_tracker_test.go` — remove lines 346-461

**Context:** The bottom 116 lines of this file test 4 unexported helpers (`normalizeRef`, `getComponentRoot`, `isReusableComponentPath`, `parseMethod`). These are fully covered by the 5 integration tests above them: `TestRefTrackerOAS3`, `TestRefTrackerOAS2`, `TestRefTrackerTransitiveRefs`, `TestRefTrackerCircularRefs`, `TestRefTrackerWebhooks`.

**Step 1: Remove the helper test section**

Delete everything from the "Helper Function Tests" section header (line 344) through end of file (line 461). The file should end after `TestRefTrackerWebhooks` (line 344).

```go
// Lines to KEEP: 1-344 (package declaration, imports, 5 integration tests)
// Lines to DELETE: 345-461 (blank line + 4 helper tests: TestNormalizeRef, TestGetComponentRoot, TestIsReusableComponentPath, TestParseMethod)
```

**Step 2: Run validator tests**

Run: `go test ./validator/ -count=1 -v 2>&1 | grep -c '^--- PASS'`
Expected: `139` (143 minus 4 deleted tests)

Run: `go test ./validator/ -count=1`
Expected: `ok`

**Step 3: Commit**

```bash
git add validator/ref_tracker_test.go
git commit -m "test(validator): remove 4 redundant ref tracker helper tests

Remove TestNormalizeRef, TestGetComponentRoot, TestIsReusableComponentPath,
TestParseMethod — all covered by TestRefTrackerOAS3/OAS2/TransitiveRefs/
CircularRefs/Webhooks integration tests."
```

---

### Task 3: Remove tautological constant tests

**Files:**
- Modify: `internal/httputil/httputil_test.go` — remove `TestStatusCodeConstants` (lines 260-266)
- Modify: `internal/severity/severity_test.go` — remove `TestSeverityConstants` (lines 35-48)

**Context:** These test that compile-time constants equal their literal values (e.g., `StatusCodeLength == 3`). If anyone changed these constants, the *behavioral* tests (`TestValidateStatusCode`, `TestSeverityString`) would already fail. These are tautological — they assert that `3 == 3`.

**Step 1: Remove `TestStatusCodeConstants` (lines 259-267)**

Delete the blank line before `TestStatusCodeConstants` through its closing brace. The file should flow from `TestHTTPMethodConstants` directly to `TestStandardHTTPStatusCodesCompleteness`.

**Step 2: Remove `TestSeverityConstants` (lines 34-49)**

Delete the blank line before `TestSeverityConstants` through its closing brace. The file should flow from `TestSeverityString` directly to `TestSeverityStringConsistency`.

**Step 3: Run both packages**

Run: `go test ./internal/httputil/ ./internal/severity/ -count=1`
Expected: Both `ok`

**Step 4: Commit**

```bash
git add internal/httputil/httputil_test.go internal/severity/severity_test.go
git commit -m "test: remove tautological constant tests in httputil and severity

TestStatusCodeConstants and TestSeverityConstants assert compile-time
constants equal their literal values. Behavioral tests already validate
these constants work correctly."
```

---

### Task 4: Remove `TestLevenshteinDistance` from CLI

**Files:**
- Modify: `cmd/oastools/main_test.go` — remove `TestLevenshteinDistance` (lines 5-32)

**Context:** `levenshteinDistance()` is only called by `suggestCommand()`, which is tested by `TestSuggestCommand` (lines 34-70). The algorithm is validated through the suggestion behavior — if the distance calculation were wrong, suggestions would be wrong.

**Step 1: Remove lines 5-33**

Delete from `func TestLevenshteinDistance` through the blank line before `TestSuggestCommand`. Leave the package declaration, import, and `TestSuggestCommand` intact.

**Step 2: Run CLI tests**

Run: `go test ./cmd/oastools/ -count=1`
Expected: `ok`

**Step 3: Commit**

```bash
git add cmd/oastools/main_test.go
git commit -m "test(cmd): remove redundant TestLevenshteinDistance

The levenshtein algorithm is validated through TestSuggestCommand which
exercises the full suggestion pipeline."
```

---

### 🔒 Checkpoint 1: Phase 1 Verification

**Run full suite and verify:**

```bash
go test ./... -count=1 -v 2>&1 | grep -c '^--- PASS'
```

**Expected:** `2,636` (2,650 - 9 - 4 - 1 - 1 + 1 = 2,636... let me compute:
- Task 1: -9 (helpers_test.go)
- Task 2: -4 (ref_tracker helpers)
- Task 3: -2 (StatusCodeConstants + SeverityConstants)
- Task 4: -1 (LevenshteinDistance)
- **Total removed: 16 tests → 2,634 expected**)

```bash
go test ./... -count=1
```

**Expected:** All packages `ok`, zero failures.

**If checkpoint fails:** Do NOT proceed to Phase 2. Investigate which removed test was covering behavior not covered elsewhere. Re-read the audit report rationale for the failing package and determine if the test needs to be restored.

---

## Phase 2: Fixer Package Cleanup

### Task 5: Remove helper tests from `fixer/enum_csv_test.go`

**Files:**
- Modify: `fixer/enum_csv_test.go` — remove lines 11-260

**Context:** Lines 11-260 contain 4 helper tests (`TestIsCSVEnumCandidate`, `TestExpandCSVEnumValues`, `TestParseNumericValue`, `TestGetSchemaType`) for unexported functions. Lines 262+ contain 4 integration tests (`TestFixSchemaCSVEnums`, `TestFixSchemaCSVEnums_NestedSchemas`, etc.) that exercise the full CSV enum fix pipeline, transitively covering all helpers.

**Step 1: Remove lines 11-260**

Delete from `func TestIsCSVEnumCandidate` through the closing brace of `TestGetSchemaType`. Keep line 1-10 (package declaration + imports) and lines 261+ (integration tests).

After editing, verify the import block still compiles — the integration tests may use a subset of the imports. Run `goimports` or let the compiler tell you.

**Step 2: Run fixer tests**

Run: `go test ./fixer/ -count=1 -run 'TestFixSchemaCSVEnums'`
Expected: `ok` (integration tests still pass)

Run: `go test ./fixer/ -count=1`
Expected: `ok`

**Step 3: Commit**

```bash
git add fixer/enum_csv_test.go
git commit -m "test(fixer): remove 4 redundant CSV enum helper tests

TestIsCSVEnumCandidate, TestExpandCSVEnumValues, TestParseNumericValue,
TestGetSchemaType — all covered by TestFixSchemaCSVEnums integration tests."
```

---

### Task 6: Remove helper tests from `fixer/fixer_pathparam_test.go`

**Files:**
- Modify: `fixer/fixer_pathparam_test.go` — remove lines 12-117

**Context:** Lines 12-117 test `extractPathParameters` and `inferParameterType`. Lines 119+ contain 5 integration tests that exercise the full path parameter fix pipeline.

**Step 1: Remove lines 12-117**

Delete from `func TestExtractPathParameters` through the closing brace of `TestInferParameterType`.

**Step 2: Run fixer tests**

Run: `go test ./fixer/ -count=1 -run 'TestFixMissingPathParameters|TestFixNoChangesNeeded|TestFixPathItemLevel'`
Expected: `ok`

**Step 3: Commit**

```bash
git add fixer/fixer_pathparam_test.go
git commit -m "test(fixer): remove 2 redundant path parameter helper tests

TestExtractPathParameters and TestInferParameterType — both covered by
TestFixMissingPathParameters OAS2/OAS3 integration tests."
```

---

### Task 7: Remove helper tests from `fixer/generic_names_test.go`

**Files:**
- Modify: `fixer/generic_names_test.go` — remove lines 17-943

**Context:** Lines 17-943 contain ~19 tests of unexported helpers (`hasInvalidSchemaNameChars`, `isGenericStyleName`, `parseGenericName`, `splitTypeParams`, `transformSchemaName`, `sanitizeSchemaName`, `toPascalCase`, `rewriteSchemaRefs` variants, `extractSchemaNameFromRefPath`, `isPackageQualifiedName`, `transformTypeParam`, etc.). Lines 945-1179 contain 2 integration tests (`TestGenericSchemaFixerRefCorruption`, `TestGenericSchemaFixerRefCorruption_OAS3`) from issue #233 that exercise the full generic name pipeline.

**Step 1: Remove lines 17-943**

This is the largest single removal (927 lines). Delete from the first helper test through `TestTransformTypeParam`'s closing brace. Keep imports (lines 1-15) and integration tests (lines 945-1179).

Check imports — some may only be used by the removed tests. Let the compiler guide cleanup.

**Step 2: Run fixer tests**

Run: `go test ./fixer/ -count=1 -run 'TestGenericSchemaFixer|TestFixInvalidSchemaNames'`
Expected: `ok`

**Step 3: Commit**

```bash
git add fixer/generic_names_test.go
git commit -m "test(fixer): remove ~19 redundant generic name helper tests

Remove tests for hasInvalidSchemaNameChars, parseGenericName,
splitTypeParams, transformSchemaName, sanitizeSchemaName, toPascalCase,
rewriteSchemaRefs, extractSchemaNameFromRefPath, isPackageQualifiedName,
transformTypeParam — all covered by TestGenericSchemaFixerRefCorruption
and TestFixInvalidSchemaNamesOAS3 integration tests."
```

---

### Task 8: Remove helper tests from `fixer/operationid_test.go`

**Files:**
- Modify: `fixer/operationid_test.go` — remove lines 17-856 AND lines 1658-1718

**Context:** This file has helper tests in TWO non-contiguous regions:
1. Lines 17-856: `TestParseOperationIdNamingTemplate`, `TestExpandOperationIdTemplate`, `TestApplyModifier`, `TestSanitizePath`
2. Lines 1658-1718 (section header at 1658): `TestGetSortedMethods` and variants

Between them (lines 858-1656) are the integration tests that must be preserved: `TestFixDuplicateOperationIds` (the 467-line table-driven test with 17 subtests), OAS2/OAS3.1 variants, dry run tests, naming config tests, etc.

**Step 1: Remove the bottom helper section first (lines 1657-1718)**

Delete from the section header comment `// ===...Helper Function Tests...===` at line 1658 through `TestGetSortedMethods_Empty` closing brace at line 1718. This preserves `TestDefaultOperationIdNamingConfig` ending at line 1656.

**Step 2: Remove the top helper section (lines 17-856)**

Delete from `TestParseOperationIdNamingTemplate` through `TestSanitizePath` closing brace at line 856. Keep package/imports (lines 1-15) and the integration test section header + tests starting at line 858.

**Step 3: Run fixer tests**

Run: `go test ./fixer/ -count=1 -run 'TestFixDuplicateOperationIds|TestWithOperationIdNaming|TestDefaultOperationIdNaming'`
Expected: `ok`

Run: `go test ./fixer/ -count=1`
Expected: `ok`

**Step 4: Commit**

```bash
git add fixer/operationid_test.go
git commit -m "test(fixer): remove redundant operation ID helper tests

Remove TestParseOperationIdNamingTemplate, TestExpandOperationIdTemplate,
TestApplyModifier, TestSanitizePath, TestGetSortedMethods variants —
all covered by TestFixDuplicateOperationIds (17 integration subtests)."
```

---

### Task 9: Remove helper tests from `fixer/stub_missing_refs_test.go`

**Files:**
- Modify: `fixer/stub_missing_refs_test.go` — remove lines 669-772

**Context:** Lines 669-772 contain the section header and 2 helper tests (`TestIsLocalRef`, `TestExtractResponseNameFromRef`). Lines 1-667 contain 30+ stub integration tests. Lines 774+ contain nil safety tests (keep).

**Step 1: Remove lines 669-772**

Delete from the section header comment `// ===...Helper Function Tests...===` through `TestExtractResponseNameFromRef` closing brace. The file should flow from the last integration test directly to `// ===...Nil Safety Tests...===`.

**Step 2: Run fixer tests**

Run: `go test ./fixer/ -count=1 -run 'TestStubMissingRefs'`
Expected: `ok`

**Step 3: Commit**

```bash
git add fixer/stub_missing_refs_test.go
git commit -m "test(fixer): remove 2 redundant stub ref helper tests

TestIsLocalRef and TestExtractResponseNameFromRef — covered by 30+
stub missing refs integration tests."
```

---

### Task 10: Remove deep copy tests from `fixer/fixer_test.go`

**Files:**
- Modify: `fixer/fixer_test.go` — remove lines 45-89

**Context:** `TestDeepCopyOAS3Document` and `TestDeepCopyOAS2Document` test the unexported `deepCopyOAS3Document`/`deepCopyOAS2Document` helpers. These are fully covered by `TestMutableInput_OAS3_PreservesOriginal` and `TestMutableInput_OAS2_PreservesOriginal` which verify deep copy behavior through the public `Fix()` API.

**Step 1: Remove lines 45-89**

Delete from the blank line + comment before `TestDeepCopyOAS3Document` through `TestDeepCopyOAS2Document` closing brace. The file should flow from `TestNewFixerWithPath_EmptyPath` directly to `TestIsFixEnabled`.

**Step 2: Run fixer tests**

Run: `go test ./fixer/ -count=1 -run 'TestMutableInput|TestIsFixEnabled'`
Expected: `ok`

**Step 3: Commit**

```bash
git add fixer/fixer_test.go
git commit -m "test(fixer): remove 2 redundant deep copy unit tests

TestDeepCopyOAS3Document and TestDeepCopyOAS2Document — covered by
TestMutableInput_OAS3/OAS2_PreservesOriginal integration tests."
```

---

### Task 11: Remove helper tests from `fixer/prune_transitive_test.go`

**Files:**
- Modify: `fixer/prune_transitive_test.go` — remove 3 non-contiguous test functions

**Context:** Three isolated helper tests scattered among 1,537 lines of integration tests:
1. `TestCollectSchemaRefs_ItemsAsMap` (lines 443-520) — documents parser quirk, covered by integration tests
2. `TestIsComponentsEmpty` (lines 608-718) — tests unexported helper, covered by pruning integration tests
3. `TestCollectRefsFromMap_AllPaths` (lines 1422-1537) — tests unexported helper, redundant with `TestRefCollector_CollectRefsFromMap_AllPaths` integration test at lines 1158-1420

**Step 1: Remove in reverse order to preserve line numbers**

Remove `TestCollectRefsFromMap_AllPaths` (lines 1421-1537) first, then `TestIsComponentsEmpty` (lines 607-718), then `TestCollectSchemaRefs_ItemsAsMap` (lines 442-520).

**Step 2: Run fixer tests**

Run: `go test ./fixer/ -count=1 -run 'TestPrune|TestRefCollector'`
Expected: `ok`

**Step 3: Commit**

```bash
git add fixer/prune_transitive_test.go
git commit -m "test(fixer): remove 3 redundant prune helper tests

TestCollectSchemaRefs_ItemsAsMap, TestIsComponentsEmpty,
TestCollectRefsFromMap_AllPaths — all covered by pruning integration
tests and TestRefCollector_CollectRefsFromMap_AllPaths."
```

---

### 🔒 Checkpoint 2: Phase 2 Verification

**Run full suite:**

```bash
go test ./... -count=1 -v 2>&1 | grep -c '^--- PASS'
```

**Expected test count calculation:**
- After Phase 1: 2,634
- Task 5: -4 (enum CSV helpers)
- Task 6: -2 (path param helpers)
- Task 7: ~-19 (generic name helpers — exact count depends on subtests)
- Task 8: ~-8 (operation ID helpers + getSortedMethods × 4)
- Task 9: -2 (stub ref helpers)
- Task 10: -2 (deep copy helpers)
- Task 11: -3 (prune helpers; but some are table-driven with subtests)

**NOTE:** The exact post-Phase-2 count will depend on subtest expansion. The key verification is:

```bash
go test ./... -count=1
```

**Expected:** All packages `ok`, zero failures.

**Also verify fixer specifically:**

```bash
go test ./fixer/ -count=1 -v 2>&1 | grep -c '^--- PASS'
```

Compare to pre-cleanup count of 214. The delta should match exactly the number of test functions removed.

**If checkpoint fails:** Investigate the failing test. The most likely cause is an import that was only used by removed tests — fix with `goimports` or manual cleanup. If an actual behavioral regression occurs, the removed test was NOT redundant — restore it and update the audit report.

---

## Phase 3: Architectural Consolidation — `internal/testutil`

### Task 12: Inline testutil helpers into `converter/converter_test.go`

**Files:**
- Modify: `converter/converter_test.go` — add unexported helper functions
- Delete: `internal/testutil/fixtures.go` (151 lines)
- Delete: `internal/testutil/fixtures_test.go` (282 lines)

**Context:** `internal/testutil` is imported by exactly one file: `converter/converter_test.go`. The entire package exists to serve 10 function calls in a single test file. Moving the helpers inline eliminates a package, its tests, and the import path.

**Step 1: Add unexported helpers to converter test file**

Add the following helper functions to `converter/converter_test.go` (after the import block, before the first test function). Use unexported names since they only need package-level visibility:

```go
// Test helpers (inlined from internal/testutil)

func newSimpleOAS2Document() *parser.OAS2Document {
	return &parser.OAS2Document{
		Swagger:    "2.0",
		OASVersion: parser.OASVersion20,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Host:     "api.example.com",
		BasePath: "/v1",
		Schemes:  []string{"https"},
		Paths:    make(map[string]*parser.PathItem),
	}
}

func newDetailedOAS2Document() *parser.OAS2Document {
	doc := newSimpleOAS2Document()
	doc.Definitions = map[string]*parser.Schema{
		"Pet": {
			Type: "object",
			Properties: map[string]*parser.Schema{
				"id":   {Type: "integer"},
				"name": {Type: "string"},
			},
		},
	}
	doc.Paths = map[string]*parser.PathItem{
		"/pets": {
			Get: &parser.Operation{
				Summary:     "List pets",
				OperationID: "listPets",
			},
		},
	}
	return doc
}

func newSimpleOAS3Document() *parser.OAS3Document {
	return &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Servers: []*parser.Server{
			{
				URL:         "https://api.example.com/v1",
				Description: "Production server",
			},
		},
		Paths: make(map[string]*parser.PathItem),
	}
}

func newDetailedOAS3Document() *parser.OAS3Document {
	return &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Servers: []*parser.Server{
			{
				URL:         "https://api.example.com/v1",
				Description: "Production server",
			},
		},
		Paths: map[string]*parser.PathItem{
			"/pets": {
				Get: &parser.Operation{
					Summary:     "List pets",
					OperationID: "listPets",
				},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"id":   {Type: "integer"},
						"name": {Type: "string"},
					},
				},
			},
		},
	}
}

func writeTempYAML(t *testing.T, doc any) string {
	t.Helper()
	data, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal document to YAML: %v", err)
	}
	tmpFile := filepath.Join(t.TempDir(), "test.yaml")
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		t.Fatalf("Failed to write temporary YAML file: %v", err)
	}
	return tmpFile
}

func writeTempJSON(t *testing.T, doc any) string {
	t.Helper()
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal document to JSON: %v", err)
	}
	tmpFile := filepath.Join(t.TempDir(), "test.json")
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		t.Fatalf("Failed to write temporary JSON file: %v", err)
	}
	return tmpFile
}
```

**Step 2: Update all call sites in converter_test.go**

Replace all `testutil.FunctionName` calls with the new unexported names:
- `testutil.NewSimpleOAS2Document()` → `newSimpleOAS2Document()`
- `testutil.NewDetailedOAS2Document()` → `newDetailedOAS2Document()`
- `testutil.NewSimpleOAS3Document()` → `newSimpleOAS3Document()`
- `testutil.NewDetailedOAS3Document()` → `newDetailedOAS3Document()`
- `testutil.WriteTempYAML(t, doc)` → `writeTempYAML(t, doc)`
- `testutil.WriteTempJSON(t, doc)` → `writeTempJSON(t, doc)`

**Step 3: Remove the testutil import**

Remove `"github.com/erraggy/oastools/internal/testutil"` from the import block.

**Step 4: Add required imports**

If not already present, add to the import block:
- `"encoding/json"`
- `"os"`
- `"path/filepath"`
- `"go.yaml.in/yaml/v4"`

**Step 5: Run converter tests**

Run: `go test ./converter/ -count=1`
Expected: `ok`

**Step 6: Delete `internal/testutil/` package**

```bash
rm -r internal/testutil/
```

**Step 7: Verify no dangling references**

Run: `go build ./...`
Expected: Build succeeds (no import of `internal/testutil` anywhere)

**Step 8: Commit**

```bash
git add converter/converter_test.go
git rm -r internal/testutil/
git commit -m "refactor(converter): inline testutil helpers, delete internal/testutil

internal/testutil was imported by exactly one file (converter_test.go).
Inlined the 6 helper functions as unexported test helpers, eliminating
the package, its 282 lines of tests-for-test-infrastructure, and the
indirection."
```

---

### 🔒 Checkpoint 3: Phase 3 Verification

**Run full suite:**

```bash
go test ./... -count=1
```

**Expected:** All packages `ok`. The `internal/testutil` package no longer exists.

**Verify converter coverage is unchanged:**

```bash
go test ./converter/ -count=1 -v 2>&1 | grep -c '^--- PASS'
```

**Expected:** Same count as before Phase 3 (converter tests should be unchanged — same test functions, just different helper source).

---

## Phase 4: Infrastructure Fix — Parser Test Helpers

### Task 13: Rename `parser/schema_test_helpers.go`

**Files:**
- Rename: `parser/schema_test_helpers.go` → `parser/schema_test_helpers_test.go`

**Context:** This file contains 3 helper functions (`ptr()`, `intPtr()`, `boolPtr()`) used exclusively by parser test files. It has no `_test.go` suffix, so it's included in production binaries — adding ~20 lines of dead code to every consumer of the parser package.

**Step 1: Rename the file**

```bash
git mv parser/schema_test_helpers.go parser/schema_test_helpers_test.go
```

**Step 2: Run parser tests**

Run: `go test ./parser/ -count=1`
Expected: `ok` (test files can still see the helpers since they're in the same package)

**Step 3: Verify production build excludes the file**

Run: `go build ./parser/`
Expected: Build succeeds (the helpers have no production callers)

**Step 4: Commit**

```bash
git add parser/schema_test_helpers.go parser/schema_test_helpers_test.go
git commit -m "fix(parser): rename schema_test_helpers.go to _test.go suffix

This file contains only test helper functions (ptr, intPtr, boolPtr)
never called from production code. Without the _test.go suffix, these
functions are included in production binaries."
```

---

### 🔒 Checkpoint 4: Final Verification

**Run the complete test suite one final time:**

```bash
go test ./... -count=1 -v 2>&1 | grep -c '^--- PASS'
```

**Also run with race detector:**

```bash
go test ./... -count=1 -race 2>&1 | tail -30
```

**Expected:** All packages `ok`, zero failures, zero data races.

**Verify build:**

```bash
go build ./...
```

**Expected:** Clean build, no errors.

**Run `make check` (project standard pre-commit check):**

```bash
make check
```

**Expected:** All checks pass.

**Summary of changes:**
- Validator: ~380 lines removed (13 test functions)
- Fixer: ~2,500 lines removed (~40 test functions)
- Utilities: ~45 lines removed (3 test functions)
- internal/testutil: ~433 lines removed (entire package)
- parser: 1 file renamed (0 lines changed)
- **Total: ~3,358 lines removed, 0 reduction in public API coverage**
