# Validator Package File Split Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Split `validator.go` (2667 lines) and `validator_test.go` (2864 lines) into logical files under 2000 lines each.

**Architecture:** Extract code by responsibility—options handling, OAS2-specific validation, OAS3-specific validation, schema validation, reference validation, and shared helpers—into separate files. Test files mirror source files.

**Tech Stack:** Go 1.24, same internal dependencies (parser, internal/httputil, internal/issues, internal/options, internal/severity)

---

## Target File Structure

| Source File | Purpose | Est. Lines |
|-------------|---------|------------|
| `validator.go` | Core types, entry points, orchestrators | ~550 |
| `options.go` | Option type, With* functions, config | ~220 |
| `oas2.go` | OAS 2.0 validators | ~450 |
| `oas3.go` | OAS 3.x validators | ~700 |
| `schema.go` | Schema validation | ~280 |
| `refs.go` | Reference validation | ~550 |
| `helpers.go` | Shared utilities | ~200 |

| Test File | Purpose | Est. Lines |
|-----------|---------|------------|
| `validator_test.go` | Core tests | ~350 |
| `options_test.go` | Options tests | ~400 |
| `oas2_test.go` | OAS2 tests | ~350 |
| `oas3_test.go` | OAS3 tests | ~400 |
| `schema_test.go` | Schema tests | ~200 |
| `refs_test.go` | Ref validation tests | ~350 |
| `path_test.go` | Path template tests | ~450 |
| `helpers_test.go` | Helper tests | ~200 |
| `result_test.go` | ToParseResult tests | ~450 |

---

## Task 1: Create helpers.go

**Files:**
- Create: `validator/helpers.go`
- Modify: `validator/validator.go` (remove moved code)

**Step 1: Create helpers.go with package header and imports**

Create `validator/helpers.go` with the helper functions. Extract these from `validator.go`:
- `pathParamRegex`, `emailRegex` (package-level vars, ~lines 1907-1911)
- `validateInfoObject` (~lines 467-521)
- `validateResponseStatusCodes` (~lines 632-672)
- `checkDuplicateOperationIds` (~lines 1907-1949)
- `validatePathTemplate` (~lines 1951-2017)
- `checkTrailingSlash` (~lines 2019-2031)
- `extractPathParameters` (~lines 2033-2044)
- `isValidMediaType` (~lines 2046-2073)
- `getJSONSchemaRef` (~lines 2075-2078)
- `isValidURL` (~lines 2080-2101)
- `isValidEmail` (~lines 2103-2110)
- `validateSPDXLicense` (~lines 2112-2121)

**Step 2: Verify compilation**

Run: `go build ./validator/...`
Expected: Success (no errors)

**Step 3: Run tests**

Run: `go test ./validator/... -count=1`
Expected: All tests pass

**Step 4: Remove moved code from validator.go**

Delete the helper functions from `validator.go` that now exist in `helpers.go`.

**Step 5: Verify compilation again**

Run: `go build ./validator/...`
Expected: Success

**Step 6: Run tests again**

Run: `go test ./validator/... -count=1`
Expected: All tests pass

**Step 7: Commit**

```bash
git add validator/helpers.go validator/validator.go
git commit -m "refactor(validator): extract helpers to separate file"
```

---

## Task 2: Create options.go

**Files:**
- Create: `validator/options.go`
- Modify: `validator/validator.go` (remove moved code)

**Step 1: Create options.go**

Extract from `validator.go` (~lines 142-283):
- `Option` type
- `validateConfig` struct
- `ValidateWithOptions` function
- `applyOptions` function
- All `With*` functions: `WithFilePath`, `WithParsed`, `WithIncludeWarnings`, `WithStrictMode`, `WithValidateStructure`, `WithUserAgent`, `WithSourceMap`

**Step 2: Verify compilation**

Run: `go build ./validator/...`
Expected: Success

**Step 3: Run tests**

Run: `go test ./validator/... -count=1`
Expected: All tests pass

**Step 4: Remove moved code from validator.go**

**Step 5: Verify and commit**

```bash
git add validator/options.go validator/validator.go
git commit -m "refactor(validator): extract options to separate file"
```

---

## Task 3: Create schema.go

**Files:**
- Create: `validator/schema.go`
- Modify: `validator/validator.go` (remove moved code)

**Step 1: Create schema.go**

Extract from `validator.go`:
- `validateSchemaName` (~lines 705-726)
- `validateSchema` (~lines 1660-1664)
- `validateSchemaWithVisited` (~lines 1666-1704)
- `validateEnumValues` (~lines 1706-1790)
- `validateSchemaTypeConstraints` (~lines 1791-1828)
- `validateRequiredFields` (~lines 1830-1844)
- `validateNestedSchemas` (~lines 1846-1905)

**Step 2: Verify compilation**

Run: `go build ./validator/...`
Expected: Success

**Step 3: Run tests**

Run: `go test ./validator/... -count=1`
Expected: All tests pass

**Step 4: Remove moved code from validator.go**

**Step 5: Verify and commit**

```bash
git add validator/schema.go validator/validator.go
git commit -m "refactor(validator): extract schema validation to separate file"
```

---

## Task 4: Create refs.go

**Files:**
- Create: `validator/refs.go`
- Modify: `validator/validator.go` (remove moved code)

**Step 1: Create refs.go**

Extract from `validator.go` (~lines 2123-2667):
- `validateRef`
- `buildOAS2ValidRefs`
- `buildOAS3ValidRefs`
- `validateSchemaRefs`
- `validateParameterRef`
- `validateOperationResponses`
- `validateResponseRef`
- `validateRequestBodyRef`
- `validateOAS2Refs`
- `validateOAS3Refs`
- `validatePathItemOperationRefs`

**Step 2: Verify compilation**

Run: `go build ./validator/...`
Expected: Success

**Step 3: Run tests**

Run: `go test ./validator/... -count=1`
Expected: All tests pass

**Step 4: Remove moved code from validator.go**

**Step 5: Verify and commit**

```bash
git add validator/refs.go validator/validator.go
git commit -m "refactor(validator): extract ref validation to separate file"
```

---

## Task 5: Create oas2.go

**Files:**
- Create: `validator/oas2.go`
- Modify: `validator/validator.go` (remove moved code)

**Step 1: Create oas2.go**

Extract from `validator.go` (~lines 522-961):
- `validateOAS2Info`
- `validateOAS2OperationIds`
- `validateOAS2Paths`
- `validateOAS2Operation`
- `validateOAS2Definitions`
- `validateOAS2Parameters`
- `validateOAS2Responses`
- `validateOAS2Security`
- `validateOAS2PathParameterConsistency`

**Step 2: Verify compilation**

Run: `go build ./validator/...`
Expected: Success

**Step 3: Run tests**

Run: `go test ./validator/... -count=1`
Expected: All tests pass

**Step 4: Remove moved code from validator.go**

**Step 5: Verify and commit**

```bash
git add validator/oas2.go validator/validator.go
git commit -m "refactor(validator): extract OAS2 validators to separate file"
```

---

## Task 6: Create oas3.go

**Files:**
- Create: `validator/oas3.go`
- Modify: `validator/validator.go` (remove moved code)

**Step 1: Create oas3.go**

Extract from `validator.go` (~lines 1020-1658):
- `validateOAS3Info`
- `validateOAS3OperationIds`
- `validateOAS3Servers`
- `validateOAS3Paths`
- `validateOAS3Operation`
- `validateOAS3RequestBody`
- `validateOAS3Components`
- `validateOAS3SecurityScheme`
- `validateOAuth2Flows`
- `validateOAS3Webhooks`
- `validateOAS3PathParameterConsistency`
- `validateOAS3SecurityRequirements`

**Step 2: Verify compilation**

Run: `go build ./validator/...`
Expected: Success

**Step 3: Run tests**

Run: `go test ./validator/... -count=1`
Expected: All tests pass

**Step 4: Remove moved code from validator.go**

**Step 5: Verify and commit**

```bash
git add validator/oas3.go validator/validator.go
git commit -m "refactor(validator): extract OAS3 validators to separate file"
```

---

## Task 7: Split validator_test.go

After source files are split, reorganize tests to mirror source structure.

**Files to create:**
- `validator/options_test.go` - Tests for With* functions
- `validator/schema_test.go` - Tests for schema validation
- `validator/refs_test.go` - Tests for ref validation
- `validator/path_test.go` - Tests for path template validation
- `validator/helpers_test.go` - Tests for helper functions
- `validator/result_test.go` - Tests for ToParseResult

**Step 1: Create each test file by moving relevant tests**

Move tests that directly test functions in each source file to the corresponding test file.

**Step 2: Verify all tests pass**

Run: `go test ./validator/... -count=1 -v`
Expected: All tests pass

**Step 3: Commit test reorganization**

```bash
git add validator/*_test.go
git commit -m "refactor(validator): reorganize tests to mirror source files"
```

---

## Task 8: Final Verification

**Step 1: Run full test suite**

Run: `make check`
Expected: All checks pass

**Step 2: Verify line counts**

Run: `wc -l validator/*.go | sort -n`
Expected: No file exceeds 2000 lines

**Step 3: Verify test line counts**

Run: `wc -l validator/*_test.go | sort -n`
Expected: No test file exceeds 2000 lines

---

## Success Criteria

- [ ] All source files under 2000 lines
- [ ] All test files under 2000 lines
- [ ] `go build ./validator/...` succeeds
- [ ] `go test ./validator/...` passes (100%)
- [ ] `make check` passes
- [ ] No changes to public API
- [ ] No changes to test coverage
