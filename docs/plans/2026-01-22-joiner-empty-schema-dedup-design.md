# Design: Skip Empty Schemas in Joiner Semantic Deduplication

**Issue**: #270
**Date**: 2026-01-22
**Status**: Implemented

## Problem Statement

The joiner's semantic deduplication was incorrectly treating empty schemas `{}` as structurally identical and consolidating them, losing their semantic distinction.

Empty schemas serve multiple semantic purposes despite being structurally identical:
- **Future placeholders**: `store.Product.structure = {}` (to be defined later)
- **Open types**: `metadata = {}` (intentionally accepts any type, like Go's `any`)
- **Context-specific wildcards**: Different properties using `{}` to mean "anything valid in this context"

Merging these loses the semantic distinction of *where* and *why* each empty schema exists.

## Solution

When semantic deduplication is enabled, automatically detect and skip empty schemas from the deduplication process. An "empty schema" is one with **no structural constraints** - no type, properties, format, enum, or any validation rules, even if it has metadata like title or description.

## Design Decisions

### What Qualifies as "Empty"?

A schema is empty if it has **no structural constraints**. Metadata fields (title, description, example, deprecated) are ignored because they don't affect validation behavior.

**Examples of empty schemas:**
- `{}` - truly empty
- `{title: "Placeholder"}` - has metadata but no constraints
- `{description: "To be defined"}` - has metadata but no constraints

**Examples of non-empty schemas:**
- `{type: "object"}` - has structural constraint (validates type)
- `{properties: {id: {type: "string"}}}` - has structural constraint
- `{format: "uuid"}` - has structural constraint

### Configuration Approach

**Always enabled when semantic deduplication is on** - no separate configuration flag. This is simpler and preserves correct semantics by default.

### Comparison Behavior

When `CompareSchemas()` encounters an empty schema (left or right), it returns:
```go
EquivalenceResult{
    Equivalent: false,
    Differences: []SchemaDifference{}, // Empty slice = no structural differences
}
```

The empty `Differences` slice signals: "structurally identical but semantically distinct". The `String()` method interprets this case specially.

## Implementation

### Core Function: `isEmptySchema()`

Checks for absence of ALL constraint fields:
- **Basic**: Type, Format, Enum, Const, Pattern, Required
- **OAS-specific**: Nullable, ReadOnly, WriteOnly, CollectionFormat
- **Properties**: Properties, AdditionalProperties, Items
- **Object**: MinProperties, MaxProperties, PatternProperties, DependentRequired
- **Array**: MinItems, MaxItems, UniqueItems, AdditionalItems, MaxContains, MinContains
- **Numeric**: Minimum, Maximum, MultipleOf, ExclusiveMinimum, ExclusiveMaximum
- **String**: MinLength, MaxLength
- **Composition**: AllOf, AnyOf, OneOf, Not
- **Conditional**: If, Then, Else
- **JSON Schema 2020-12**: UnevaluatedProperties, UnevaluatedItems, ContentEncoding, ContentMediaType, ContentSchema, PrefixItems, Contains, PropertyNames, DependentSchemas

Returns `false` for nil schemas, `true` only if no constraints exist.

### Modified `CompareSchemas()`

Added early return after nil checks:
```go
// Empty schemas are semantically distinct - never equivalent
if isEmptySchema(left) || isEmptySchema(right) {
    return EquivalenceResult{
        Equivalent:  false,
        Differences: []SchemaDifference{},
    }
}
```

### `EquivalenceResult.String()` Method

Provides human-readable output:
```go
func (r EquivalenceResult) String() string {
    if r.Equivalent {
        return "Schemas are equivalent"
    }
    if len(r.Differences) == 0 {
        return "Schemas are non-equivalent (empty schemas are semantically distinct)"
    }
    // Format differences list...
}
```

## Edge Cases Handled

1. **Mixed Empty/Non-Empty**: Correctly returns non-equivalent
2. **Nested Empty Schemas**: Parent schema's emptiness determines behavior
3. **Empty in Compositions**: Parent schema with allOf/anyOf is NOT empty
4. **JSON Schema 2020-12 Fields**: All new constraint fields included in check
5. **Performance**: O(1) early return improves performance by skipping deep comparison

## Testing

### Unit Tests (`equivalence_test.go`)
- `TestIsEmptySchema`: 56 sub-tests covering all constraint types
- `TestCompareSchemas_EmptySchemasNonEquivalent`: Verifies empty schema comparison
- `TestCompareSchemas_EmptySchemaShallowMode`: Works in both shallow and deep modes
- `TestEquivalenceResult_String`: Verifies string formatting for all cases

### Integration Tests (`joiner_dedupe_test.go`)
- `TestJoiner_SemanticDeduplication_EmptySchemasPreserved` (OAS3)
- `TestJoiner_SemanticDeduplication_EmptySchemasPreserved_OAS2`
- `TestJoiner_SemanticDeduplication_EmptyWithMetadataPreserved`

All 7892 tests pass. Coverage on new code: 100%.

## Backward Compatibility

**Impact Level**: Bug Fix (Non-Breaking)

**What Changes**:
- Users with `SemanticDeduplication: true` will see empty schemas preserved instead of consolidated
- Join results will have more schemas in output (the previously incorrectly merged empty schemas)
- No API changes, no configuration changes required

**Migration**: None needed - this fixes incorrect behavior. Users who want the old (incorrect) behavior can disable semantic deduplication entirely.

## Documentation Updates

- `joiner/doc.go`: Added section explaining empty schema behavior
- `joiner/deep_dive.md`: Added "Empty Schemas Are Preserved" subsection with examples
- Godoc comments on `isEmptySchema()` with detailed examples

## Performance Impact

`isEmptySchema()` is O(1) - just checking fields. Called once per comparison before expensive deep traversal, so it's actually a **performance win** (early exit for empty schemas).

## Key Insights

1. **Semantic vs Structural**: This highlights the difference between structural equivalence (do the fields match?) and semantic equivalence (do they mean the same thing?). Empty schemas are structurally identical but semantically distinct.

2. **Data-Driven String Formatting**: Using empty `Differences` slice to signal "semantically distinct but structurally same" lets the data structure tell the story, and `String()` interprets it for humans.

3. **Early Returns for Performance**: Checking `isEmptySchema()` before deep traversal is both correct (we want different behavior) and faster (we skip unnecessary work).

## Files Modified

- `joiner/equivalence.go`: Added `isEmptySchema()`, modified `CompareSchemas()`, added `String()` method
- `joiner/equivalence_test.go`: Added comprehensive unit tests
- `joiner/joiner_dedupe_test.go`: Added integration tests
- `joiner/doc.go`: Updated documentation
- `joiner/deep_dive.md`: Added examples and explanation
