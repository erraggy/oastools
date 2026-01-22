# Stub Missing References - Implementation Plan

**Design:** `docs/plans/2025-01-21-stub-missing-refs-design.md`
**Branch:** `feat/stub-missing-refs`
**Execution Mode:** Orchestration (delegate to `developer` agent)

---

## Implementation Steps

### Step 1: Add Fix Type Constant

**File:** `fixer/fixer.go`

**Task:** Add the new fix type constant after existing constants (around line 24):

```go
// FixTypeDuplicateOperationId indicates a duplicate operationId was renamed
FixTypeDuplicateOperationId FixType = "duplicate-operation-id"
// FixTypeStubMissingRef indicates a stub was created for an unresolved reference
FixTypeStubMissingRef FixType = "stub-missing-ref"
```

**Task:** Add `StubConfig` field to the `Fixer` struct (around line 144):

```go
// StubConfig configures how missing reference stubs are created.
StubConfig StubConfig
```

**Task:** Initialize `StubConfig` in the `New()` function.

---

### Step 2: Create Stub Configuration

**File:** `fixer/stub_missing_refs.go` (NEW)

**Task:** Create file with:

1. `StubConfig` struct with `ResponseDescription` field
2. `DefaultStubConfig()` function
3. `isLocalRef(ref string) bool` helper
4. `extractResponseNameFromRef(ref string, version parser.OASVersion) string` helper (similar to existing `ExtractSchemaNameFromRef`)

---

### Step 3: Implement OAS 2.0 Stubbing

**File:** `fixer/stub_missing_refs.go`

**Task:** Implement:

1. `stubMissingRefsOAS2(doc *parser.OAS2Document, result *FixResult)`
   - Use `RefCollector` to collect all refs
   - Iterate `RefsByType[RefTypeSchema]`, check if exists in `doc.Definitions`
   - Iterate `RefsByType[RefTypeResponse]`, check if exists in `doc.Responses`
   - Call stub helpers for missing refs

2. `stubSchemaOAS2(doc *parser.OAS2Document, name string, result *FixResult)`
   - Initialize `doc.Definitions` if nil
   - Create empty `&parser.Schema{}`
   - Record fix

3. `stubResponseOAS2(doc *parser.OAS2Document, name string, result *FixResult)`
   - Initialize `doc.Responses` if nil
   - Create `&parser.Response{Description: config.ResponseDescription}`
   - Record fix

---

### Step 4: Implement OAS 3.x Stubbing

**File:** `fixer/stub_missing_refs.go`

**Task:** Implement:

1. `stubMissingRefsOAS3(doc *parser.OAS3Document, result *FixResult)`
   - Same pattern as OAS2, but check `doc.Components.Schemas` and `doc.Components.Responses`
   - Initialize `doc.Components` if nil

2. `stubSchemaOAS3(doc *parser.OAS3Document, name string, result *FixResult)`

3. `stubResponseOAS3(doc *parser.OAS3Document, name string, result *FixResult)`

---

### Step 5: Wire Into Fix Pipeline

**File:** `fixer/oas2.go`

**Task:** In `fixOAS2()`, add call to `stubMissingRefsOAS2()` EARLY in the pipeline (before other fixes that traverse refs):

```go
if f.isFixEnabled(FixTypeStubMissingRef) && !f.DryRun {
    f.stubMissingRefsOAS2(doc, result)
}
```

**File:** `fixer/oas3.go`

**Task:** Same pattern for `fixOAS3()`.

---

### Step 6: Add Functional Options

**File:** `fixer/pipeline.go` (or `fixer/fixer.go` where other options are)

**Task:** Add:

1. `stubConfig StubConfig` field to `fixConfig` struct
2. `WithStubConfig(config StubConfig) Option`
3. `WithStubResponseDescription(desc string) Option`
4. Wire `stubConfig` into `Fixer` in `FixWithOptions()`

---

### Step 7: Add CLI Flags

**File:** `cmd/oastools/fix.go`

**Task:** Add flags:

```go
var stubMissingRefs bool
var stubResponseDesc string

// In init():
fixCmd.Flags().BoolVar(&stubMissingRefs, "stub-missing-refs", false, "Create stubs for unresolved local $ref pointers")
fixCmd.Flags().StringVar(&stubResponseDesc, "stub-response-desc", "", "Description text for stub responses (default: auto-generated message)")
```

**Task:** Wire flags into fixer options when building `WithEnabledFixes()`.

---

### Step 8: Update Documentation

**File:** `fixer/doc.go`

**Task:** Add documentation for the new fix type in the package doc comment, following the existing pattern.

---

### Step 9: Write Tests

**File:** `fixer/stub_missing_refs_test.go` (NEW)

**Task:** Implement test cases:

1. `TestStubMissingRef_FixesValidationError` - The core acceptance test:
   - Start with invalid doc (missing ref)
   - Assert specific validation error message
   - Run fixer
   - Assert validation passes

2. `TestStubMissingSchema_OAS2` - Missing `#/definitions/Foo` gets stubbed
3. `TestStubMissingSchema_OAS3` - Missing `#/components/schemas/Foo` gets stubbed
4. `TestStubMissingResponse_OAS2` - Missing `#/responses/NotFound` gets stubbed
5. `TestStubMissingResponse_OAS3` - Missing `#/components/responses/NotFound` gets stubbed
6. `TestStubMissing_MultipleRefs` - Multiple missing refs all get stubbed
7. `TestStubMissing_ExistingNotTouched` - Existing definitions are not modified
8. `TestStubMissing_ExternalRefIgnored` - `./other.yaml#/...` is skipped
9. `TestStubMissing_NilMapsInitialized` - Nil maps get initialized
10. `TestStubMissing_CustomResponseDesc` - Custom description appears in stub
11. `TestStubMissing_DisabledByDefault` - Fix doesn't run unless enabled
12. `TestStubMissing_DryRun` - Dry run reports fixes without modifying doc

---

### Step 10: Run Verification

**Task:** Run `make check` to ensure:
- All tests pass
- No lint errors
- Build succeeds

---

## Verification Checklist

- [ ] `go build ./...` succeeds
- [ ] `go test ./fixer/...` passes
- [ ] `make lint` passes
- [ ] `make check` passes
- [ ] Manual test: `go run ./cmd/oastools fix --stub-missing-refs <test-file>` works
- [ ] Core acceptance test proves: invalid → fix → valid
