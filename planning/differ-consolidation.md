# Differ Package Consolidation Plan

**Issue:** #35 - Consolidate code between simple.go and breaking.go in the differ package
**Status:** Planning
**Created:** 2025-11-25

## Problem Statement

The differ package has two parallel implementations (`simple.go` at 1969 lines and `breaking.go` at 2322 lines) that share nearly identical traversal logic but differ only in:

1. **Severity Assignment:** `breaking.go` assigns severity levels (`SeverityInfo`, `SeverityWarning`, `SeverityError`, `SeverityCritical`) while `simple.go` omits them (leaving zero value)
2. **Conditional Severity Logic:** `breaking.go` includes extra logic to determine severity based on context (e.g., making a parameter required vs optional)
3. **Different Handling of Edge Cases:** Some minor differences in message formatting and null checks

**Impact:**
- Any bug fix or feature addition requires changes in both files
- Risk of implementations drifting apart
- ~4300 lines of code that could be reduced by ~50%
- Cognitive overhead for maintainers

## Current Architecture

```
differ/
├── differ.go          # Core types, Differ struct, public API (268 lines)
├── simple.go          # ModeSimple implementation (1969 lines)
├── breaking.go        # ModeBreaking implementation (2322 lines)
└── schema.go          # Shared schema utilities (113 lines)
```

### Duplicated Function Pairs

| Simple Function | Breaking Function | Lines (est.) |
|----------------|-------------------|--------------|
| `diffOAS2Simple` | `diffOAS2Breaking` | ~45 each |
| `diffOAS3Simple` | `diffOAS3Breaking` | ~30 each |
| `diffCrossVersionSimple` | `diffCrossVersionBreaking` | ~40 each |
| `diffInfo` | `diffInfoBreaking` | ~65/45 |
| `diffServers` | `diffServersBreaking` | ~45 each |
| `diffServer` | `diffServerBreaking` | ~15 each |
| `diffPaths` | `diffPathsBreaking` | ~30 each |
| `diffPathItem` | `diffPathItemBreaking` | ~50 each |
| `diffOperation` | `diffOperationBreaking` | ~30/40 |
| `diffParameters` | `diffParametersBreaking` | ~45/55 |
| `diffParameter` | `diffParameterBreaking` | ~40/65 |
| `diffRequestBody` | `diffRequestBodyBreaking` | ~45/60 |
| `diffResponses` | `diffResponsesBreaking` | ~60/50 |
| `diffResponse` | `diffResponseBreaking` | ~30 each |
| `diffResponseHeaders` | `diffResponseHeadersBreaking` | ~35 each |
| `diffHeader` | `diffHeaderBreaking` | ~35 each |
| `diffMediaType` | `diffMediaTypeBreaking` | ~25 each |
| `diffResponseContent` | `diffResponseContentBreaking` | ~35 each |
| `diffResponseLinks` | `diffResponseLinksBreaking` | ~35 each |
| `diffLink` | `diffLinkBreaking` | ~35 each |
| `diffResponseExamples` | `diffResponseExamplesBreaking` | ~25 each |
| `diffSchemas` | `diffSchemasBreaking` | ~30 each |
| `diffSchema` | `diffSchemaBreaking` | ~5 each (delegates) |
| `diffSchemaRecursive` | `diffSchemaRecursiveBreaking` | ~290/70 |
| `diffSchemaProperties` | `diffSchemaPropertiesBreaking` | ~35/45 |
| `diffSchemaItems` | `diffSchemaItemsBreaking` | ~100/120 |
| `diffSchemaAdditionalProperties` | `diffSchemaAdditionalPropertiesBreaking` | ~95/125 |
| `diffSchemaAllOf` | `diffSchemaAllOfBreaking` | ~45/55 |
| `diffSchemaAnyOf` | `diffSchemaAnyOfBreaking` | ~45/50 |
| `diffSchemaOneOf` | `diffSchemaOneOfBreaking` | ~45/50 |
| `diffSchemaNot` | `diffSchemaNotBreaking` | ~30/30 |
| `diffSchemaConditional` | `diffSchemaConditionalBreaking` | ~75/80 |
| `diffSecuritySchemes` | `diffSecuritySchemesBreaking` | ~30 each |
| `diffSecurityScheme` | `diffSecuritySchemeBreaking` | ~15 each |
| `diffTags` | N/A (simple only) | ~40 |
| `diffTag` | N/A (simple only) | ~15 |
| `diffComponents` | `diffComponentsBreaking` | ~35 each |
| `diffWebhooks` | `diffWebhooksBreaking` | ~25 each |
| `diffStringSlices` | `diffStringSlicesBreaking` | ~35/40 |
| N/A | `diffEnumBreaking` | ~40 |
| `diffExtras` | `diffExtrasBreaking` | ~45 each |

### Key Differences

1. **Severity Assignment:**
   - Simple: `result.Changes = append(result.Changes, Change{Path: ..., Type: ..., Category: ..., Message: ...})`
   - Breaking: Same, plus `Severity: SeverityXxx`

2. **Conditional Logic (Breaking only):**
   - Parameter required status affects severity
   - Response code type (2xx vs 4xx/5xx) affects severity
   - Type change compatibility affects severity
   - Constraint tightening vs relaxing affects severity

3. **Helper Functions (Breaking only):**
   - `isCompatibleTypeChange()`
   - `isSuccessCode()`
   - `isErrorCode()`
   - `anyToString()`
   - Various `diffSchemaXxxConstraints()` helpers

## Proposed Solution: Unified Traversal with Severity Strategy

### Approach: Strategy Pattern with Severity Provider

Create a unified traversal implementation that accepts a severity provider interface:

```go
// SeverityProvider determines the severity for a change based on context
type SeverityProvider interface {
    // GetSeverity returns the appropriate severity for a change
    // context provides information like: isRequired, changeType, category, etc.
    GetSeverity(ctx *ChangeContext) Severity
}

// ChangeContext contains information about a change for severity determination
type ChangeContext struct {
    ChangeType ChangeType
    Category   ChangeCategory
    Path       string

    // Context-specific fields
    WasRequired    bool     // For parameters, properties
    IsRequired     bool     // For parameters, properties
    OldType        string   // For type changes
    NewType        string   // For type changes
    ResponseCode   string   // For response changes
    IsConstraint   bool     // For validation constraints
    IsTightening   bool     // True if constraint is more restrictive
}

// SimpleSeverityProvider returns zero severity for all changes
type SimpleSeverityProvider struct{}

func (s *SimpleSeverityProvider) GetSeverity(ctx *ChangeContext) Severity {
    return 0 // No severity in simple mode
}

// BreakingSeverityProvider returns appropriate severity based on context
type BreakingSeverityProvider struct{}

func (b *BreakingSeverityProvider) GetSeverity(ctx *ChangeContext) Severity {
    // Implement the existing severity logic from breaking.go
    // ...
}
```

### Alternative Approach: Unified Functions with Mode Check

A simpler approach that keeps all logic in one place:

```go
// In each diff function, check the mode and assign severity accordingly
func (d *Differ) diffParameter(source, target *parser.Parameter, path string, result *DiffResult) {
    if source.Required != target.Required {
        var sev Severity
        if d.Mode == ModeBreaking {
            if !source.Required && target.Required {
                sev = SeverityError // Making optional parameter required
            } else {
                sev = SeverityInfo // Making required parameter optional
            }
        }
        result.Changes = append(result.Changes, Change{
            Path:     path + ".required",
            Type:     ChangeTypeModified,
            Category: CategoryParameter,
            Severity: sev,
            OldValue: source.Required,
            NewValue: target.Required,
            Message:  fmt.Sprintf("required changed from %v to %v", source.Required, target.Required),
        })
    }
    // ... rest of comparison
}
```

### Recommended Approach

**Use the unified function approach** because:

1. **Simpler implementation:** No new interfaces or abstractions
2. **Clearer code path:** Mode check is explicit and visible
3. **Easier testing:** Same test can verify both modes
4. **Lower risk:** Minimal architectural changes
5. **Better performance:** No interface dispatch overhead

## Implementation Plan

### Phase 1: Create Unified Infrastructure (Est. effort: Medium)

1. **Create `unified.go`** - New file for unified comparison functions
2. **Add helper for severity-aware Change creation:**
   ```go
   func (d *Differ) newChange(path string, changeType ChangeType, category ChangeCategory,
       simpleSeverity, breakingSeverity Severity, oldValue, newValue any, message string) Change {
       sev := simpleSeverity
       if d.Mode == ModeBreaking {
           sev = breakingSeverity
       }
       return Change{
           Path:     path,
           Type:     changeType,
           Category: category,
           Severity: sev,
           OldValue: oldValue,
           NewValue: newValue,
           Message:  message,
       }
   }
   ```

### Phase 2: Migrate Non-Schema Functions (Est. effort: Medium)

Migrate functions in order of dependencies (leaf functions first):

1. `diffExtras` / `diffExtrasBreaking` -> `diffExtras`
2. `diffStringSlices` / `diffStringSlicesBreaking` -> `diffStringSlices`
3. `diffTag` (simple only - add to unified)
4. `diffTags` (simple only - add to unified)
5. `diffServer` / `diffServerBreaking` -> `diffServer`
6. `diffServers` / `diffServersBreaking` -> `diffServers`
7. `diffInfo` / `diffInfoBreaking` -> `diffInfo`
8. `diffSecurityScheme` / `diffSecuritySchemeBreaking` -> `diffSecurityScheme`
9. `diffSecuritySchemes` / `diffSecuritySchemesBreaking` -> `diffSecuritySchemes`
10. `diffWebhooks` / `diffWebhooksBreaking` -> `diffWebhooks`

### Phase 3: Migrate Response Functions (Est. effort: Medium)

1. `diffLink` / `diffLinkBreaking` -> `diffLink`
2. `diffResponseLinks` / `diffResponseLinksBreaking` -> `diffResponseLinks`
3. `diffResponseExamples` / `diffResponseExamplesBreaking` -> `diffResponseExamples`
4. `diffHeader` / `diffHeaderBreaking` -> `diffHeader`
5. `diffResponseHeaders` / `diffResponseHeadersBreaking` -> `diffResponseHeaders`
6. `diffMediaType` / `diffMediaTypeBreaking` -> `diffMediaType`
7. `diffResponseContent` / `diffResponseContentBreaking` -> `diffResponseContent`
8. `diffResponse` / `diffResponseBreaking` -> `diffResponse`
9. `diffResponses` / `diffResponsesBreaking` -> `diffResponses`

### Phase 4: Migrate Schema Functions (Est. effort: High)

Schema functions are the most complex due to recursive traversal and cycle detection:

1. Move `diffSchemaMetadata`, `diffSchemaType`, `diffSchemaNumericConstraints`, etc. to unified (these only exist in breaking.go currently)
2. `diffSchemaConditional` / `diffSchemaConditionalBreaking` -> unified
3. `diffSchemaNot` / `diffSchemaNotBreaking` -> unified
4. `diffSchemaOneOf` / `diffSchemaOneOfBreaking` -> unified
5. `diffSchemaAnyOf` / `diffSchemaAnyOfBreaking` -> unified
6. `diffSchemaAllOf` / `diffSchemaAllOfBreaking` -> unified
7. `diffSchemaAdditionalProperties` / `diffSchemaAdditionalPropertiesBreaking` -> unified
8. `diffSchemaItems` / `diffSchemaItemsBreaking` -> unified
9. `diffSchemaProperties` / `diffSchemaPropertiesBreaking` -> unified
10. `diffSchemaRecursive` / `diffSchemaRecursiveBreaking` -> unified
11. `diffSchema` / `diffSchemaBreaking` -> unified
12. `diffSchemas` / `diffSchemasBreaking` -> unified

### Phase 5: Migrate Operation Functions (Est. effort: Medium)

1. `diffParameter` / `diffParameterBreaking` -> `diffParameter`
2. `diffParameters` / `diffParametersBreaking` -> `diffParameters`
3. `diffRequestBody` / `diffRequestBodyBreaking` -> `diffRequestBody`
4. `diffOperation` / `diffOperationBreaking` -> `diffOperation`
5. `diffPathItem` / `diffPathItemBreaking` -> `diffPathItem`
6. `diffPaths` / `diffPathsBreaking` -> `diffPaths`
7. `diffComponents` / `diffComponentsBreaking` -> `diffComponents`

### Phase 6: Migrate Top-Level Functions (Est. effort: Low)

1. `diffOAS2Simple` / `diffOAS2Breaking` -> `diffOAS2`
2. `diffOAS3Simple` / `diffOAS3Breaking` -> `diffOAS3`
3. `diffCrossVersionSimple` / `diffCrossVersionBreaking` -> `diffCrossVersion`
4. `diffSimple` / `diffBreaking` -> `diff` (unified entry point)

### Phase 7: Cleanup (Est. effort: Low)

1. Delete `simple.go`
2. Delete `breaking.go`
3. Rename `unified.go` to appropriate name or merge into existing files
4. Update tests to verify both modes work correctly
5. Run benchmarks to verify no performance regression

## Testing Strategy

1. **Before migration:** Run full test suite, save results
2. **During migration:** Run tests after each phase
3. **After migration:**
   - Compare diff output for both modes against saved baseline
   - Ensure breaking change counts match
   - Ensure change messages match
   - Run benchmarks to verify performance

### Key Test Cases

- OAS2 vs OAS2 comparison (both modes)
- OAS3 vs OAS3 comparison (both modes)
- Cross-version comparison (both modes)
- Circular schema references
- All change categories (endpoint, parameter, schema, etc.)
- All severity levels in breaking mode
- Empty documents, nil values, missing fields

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Subtle behavior differences between modes | High | Comprehensive test suite, diff output comparison |
| Performance regression | Medium | Benchmark before/after each phase |
| Breaking existing consumers | High | Maintain exact same public API |
| Merge conflicts during long migration | Medium | Small, focused commits; communicate with team |

## Success Metrics

- [ ] Code reduction of ~40-50% (from ~4300 to ~2200-2600 lines)
- [ ] All existing tests pass
- [ ] Benchmark performance within 5% of baseline
- [ ] No changes to public API
- [ ] Single place to fix bugs/add features

## Open Questions

1. **Should we add context-aware severity?** Breaking mode currently doesn't know if we're diffing request vs response schemas. This affects severity (e.g., adding required field to request is Error, but to response is Info).

2. **Should we merge schema.go into unified.go?** The schema helpers are small and could be co-located.

3. **Naming convention:** `diffInfo` vs `diffInfoUnified` during migration? Recommend renaming in-place since tests will catch issues.

## Timeline Estimate

- Phase 1: 2-3 hours
- Phase 2: 3-4 hours
- Phase 3: 2-3 hours
- Phase 4: 4-6 hours (most complex)
- Phase 5: 3-4 hours
- Phase 6: 1-2 hours
- Phase 7: 1-2 hours

**Total: ~16-24 hours of focused work**

## References

- Issue #35: https://github.com/erraggy/oastools/issues/35
- Current simple.go: 1969 lines
- Current breaking.go: 2322 lines
