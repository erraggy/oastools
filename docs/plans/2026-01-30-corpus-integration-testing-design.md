# Corpus Integration Testing Design

**Date**: 2026-01-30
**Status**: Draft
**Author**: Claude + Robbie

## Overview

Design for comprehensive corpus testing that exercises **overlay**, **fixer**, and **differ** packages together in realistic workflows. The goal is to test package integration rather than isolated unit behavior.

## Background

### Exploration Findings

We explored the corpus to understand what the fixer would fix in real-world specs:

| Spec | Clean | Missing Params | Unused Schemas | Empty Paths |
|------|-------|----------------|----------------|-------------|
| Petstore | ✅ | - | - | - |
| Discord | ✅ | - | - | - |
| Stripe | ✅ | - | - | - |
| DigitalOcean | - | 373 | - | - |
| Asana | - | 175 | 9 | - |
| GoogleMaps | - | - | 1 | - |
| USNWS | - | 45 | - | - |
| **Plaid** | - | - | **128** | **10** |
| GitHub | - | 2,047 | 272 | - |
| MicrosoftGraph | - | - | 44 | - |

**Key insight**: Plaid already has empty paths and 128 unused schemas, making it ideal for testing the fixer → differ pipeline without needing overlay modifications first.

### Verification

We verified fixer accuracy by examining actual spec content:
- ✅ Plaid empty paths confirmed (no operations, just path-level structure)
- ✅ Plaid unused schemas confirmed (0 references in entire spec)
- ✅ DigitalOcean missing params confirmed (path template var not declared)
- ✅ GoogleMaps unused schema confirmed (ElevationResponse has 0 refs)

## Test Architecture

### Package: `integration/corpus`

New package under `integration/` to test cross-package workflows.

```text
integration/
├── corpus/
│   ├── doc.go
│   ├── fixer_baseline_test.go      # Fixer on unmodified corpus
│   ├── overlay_fixer_test.go       # Overlay → Fixer chain
│   ├── fixer_differ_test.go        # Fixer → Differ chain
│   ├── full_pipeline_test.go       # Overlay → Fixer → Differ
│   └── helpers_test.go             # Shared test utilities
└── harness/                        # Existing integration harness
```

## Test Cases

### 1. Fixer Baseline Tests (`fixer_baseline_test.go`)

Test fixer on unmodified corpus specs to establish baselines.

```go
func TestCorpus_FixerBaseline_Plaid(t *testing.T)
// - Parse Plaid spec
// - Run fixer with pruning enabled
// - Assert: 128 unused schemas removed
// - Assert: 10 empty paths removed
// - Assert: specific schemas like AccountFilter are removed

func TestCorpus_FixerBaseline_GoogleMaps(t *testing.T)
// - Parse GoogleMaps spec
// - Run fixer with pruning enabled
// - Assert: exactly 1 unused schema removed (ElevationResponse)

func TestCorpus_FixerBaseline_CleanSpecs(t *testing.T)
// - Parse Discord, Stripe, Petstore
// - Run fixer with all fixes enabled
// - Assert: 0 fixes applied (specs are clean)
```

### 2. Fixer → Differ Chain (`fixer_differ_test.go`)

Test that differ correctly detects fixer changes.

```go
func TestCorpus_FixerDiffer_Plaid(t *testing.T)
// 1. Parse Plaid spec (original)
// 2. Run fixer → get modified spec
// 3. Run differ(original, fixed)
// 4. Assert: differ reports removed schemas as changes
// 5. Assert: differ reports removed paths as breaking changes
// 6. Assert: change count matches fix count

func TestCorpus_FixerDiffer_GoogleMaps(t *testing.T)
// - Minimal test case with single schema removal
// - Verify differ detects exactly 1 schema removal
```

### 3. Overlay → Fixer Chain (`overlay_fixer_test.go`)

Test that removing operations via overlay triggers schema pruning.

```go
func TestCorpus_OverlayFixer_RemoveOperation(t *testing.T)
// 1. Parse a spec with unique schema references
// 2. Create overlay that removes specific operation
// 3. Apply overlay → reparse to typed document
// 4. Run fixer with pruning
// 5. Assert: schemas used only by removed operation are pruned

func TestCorpus_OverlayFixer_RemovePath(t *testing.T)
// - Remove entire path via overlay
// - Verify all associated schemas are pruned if unused elsewhere
```

### 4. Full Pipeline (`full_pipeline_test.go`)

End-to-end test of overlay → fixer → differ.

```go
func TestCorpus_FullPipeline(t *testing.T)
// 1. Parse corpus spec (original)
// 2. Apply overlay to remove operations
// 3. Reparse to typed document
// 4. Run fixer to prune orphans
// 5. Run differ(original, final)
// 6. Assert: all changes properly detected
// 7. Assert: breaking changes correctly identified
```

## Test Data Strategy

### Primary Test Cases

| Spec | Use Case | Why |
|------|----------|-----|
| **Plaid** | Baseline pruning | Already has 128 unused schemas + 10 empty paths |
| **GoogleMaps** | Minimal case | Single unused schema (ElevationResponse) |
| **Discord** | Negative test | Clean spec, verify no unwanted changes |

### Overlay Test Targets

For overlay → fixer tests, we need operations that reference schemas not used elsewhere. We'll use the walker to discover these dynamically rather than hardcoding paths.

```go
// Helper to find operations with unique schema references
func findOperationsWithUniqueSchemas(doc *parser.OAS3Document) []OperationTarget {
    // Use RefCollector to build schema usage map
    // Return operations where at least one referenced schema has usage count == 1
}
```

## Implementation Plan (Orchestration Mode)

Execute via `developer` agent with `go_diagnostics` after each file write.

### Phase 1: Package Setup

**Agent**: `developer`

```text
Create integration/corpus/ package:
1. Create integration/corpus/doc.go with package documentation
2. Create integration/corpus/helpers_test.go with shared utilities:
   - parseCorpusSpec(name string) helper
   - assertFixCount(t, result, expected) helper
   - Use corpusutil.GetByName, corpusutil.SkipIfNotCached
3. Run go_diagnostics on new files
```

### Phase 2: Baseline Fixer Tests

**Agent**: `developer`

```text
Create integration/corpus/fixer_baseline_test.go:

func TestCorpus_FixerBaseline_Plaid(t *testing.T)
  - Parse Plaid, run fixer with FixTypePrunedUnusedSchema + FixTypePrunedEmptyPath
  - Assert FixCount >= 130 (128 schemas + 10 paths, with tolerance)
  - Assert specific schemas removed: AccountFilter, AccountSelectionCardinality

func TestCorpus_FixerBaseline_GoogleMaps(t *testing.T)
  - Assert exactly 1 fix (ElevationResponse schema)

func TestCorpus_FixerBaseline_CleanSpecs(t *testing.T)
  - Test Discord, Stripe, Petstore have 0 pruning fixes

Run: go test -v -run TestCorpus_FixerBaseline ./integration/corpus/...
```

### Phase 3: Fixer → Differ Chain

**Agent**: `developer`

```text
Create integration/corpus/fixer_differ_test.go:

func TestCorpus_FixerDiffer_Plaid(t *testing.T)
  1. Parse original
  2. Run fixer → fixed result
  3. differ.DiffParsed(original, fixed.ToParseResult())
  4. Assert differ.HasChanges() == true
  5. Assert removed schemas appear in diff changes

func TestCorpus_FixerDiffer_NoChanges(t *testing.T)
  - Run on Discord (clean spec)
  - Assert differ reports no changes

Run: go test -v -run TestCorpus_FixerDiffer ./integration/corpus/...
```

### Phase 4: Overlay → Fixer Chain

**Agent**: `developer`

```text
Create integration/corpus/overlay_fixer_test.go:

func TestCorpus_OverlayFixer_RemoveOperation(t *testing.T)
  1. Parse GoogleMaps (or find spec with unique refs)
  2. Create overlay removing a specific operation via JSONPath
  3. overlay.ApplyParsed() → overlay.ReparseDocument()
  4. Run fixer with pruning
  5. Assert schemas only used by removed op are pruned

Use JSONPath like: $.paths['/elevation/json'].get with remove: true
```

### Phase 5: Full Pipeline

**Agent**: `developer`

```text
Create integration/corpus/full_pipeline_test.go:

func TestCorpus_FullPipeline_OverlayFixerDiffer(t *testing.T)
  1. original := parse spec
  2. overlaid := overlay.ApplyParsed(remove operation)
  3. reparsed := overlay.ReparseDocument()
  4. fixed := fixer.FixWithOptions(pruning enabled)
  5. diff := differ.DiffParsed(original, fixed.ToParseResult())
  6. Assert diff shows removed path + removed schemas
  7. Assert HasBreakingChanges == true (removed path is breaking)
```

### Phase 6: Review & Cleanup

**Agent**: `maintainer`

```text
Review all new test files for:
- Proper error handling
- Consistent test patterns
- No hardcoded paths that could break
- Appropriate use of t.Skip for missing corpus
```

## JetBrains MCP Tools

Use these for efficient development:
- `mcp__jetbrains__get_file_problems` - Check for issues after writes
- `mcp__jetbrains__search_in_files_by_text` - Find usage patterns
- `mcp__jetbrains__execute_terminal_command` - Run tests
- `mcp__jetbrains__get_symbol_info` - Understand API signatures

## Success Criteria

1. **Accuracy**: All fix counts match exploration findings
2. **Integration**: Overlay → Fixer → Differ chain works correctly
3. **Coverage**: Tests exercise all three packages together
4. **Regression**: Tests catch regressions in any package
5. **Performance**: Tests complete in reasonable time (<30s for non-large specs)

## Open Questions

1. Should we pin corpus spec versions to prevent test flakiness from upstream changes?
2. Should overlay tests be generated dynamically or use fixed operation targets?
3. How should we handle specs that change over time (expected error counts)?

## Related Work

- `internal/corpusutil` - Corpus management utilities
- `integration/harness` - Existing integration test harness
- `fixer/fixer_corpus_test.go` - Existing (limited) fixer corpus tests
