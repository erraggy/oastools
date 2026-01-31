# ExtractExtensions Optimization Design

**Date**: 2026-01-30
**Status**: Phase 1 Implemented, Phase 2 Deferred

## Problem Statement

The `ExtractExtensions` function in `parser/internal/jsonhelpers` is a performance hot spot, consuming ~27% of CPU time when parsing large JSON specs like Stripe (7.2MB, 193K lines). The function is called from every `UnmarshalJSON` method (30+ types) to extract specification extension fields (`x-*` properties).

### Root Cause

Every `UnmarshalJSON` does **two full JSON parses**:

```go
func (o *Operation) UnmarshalJSON(data []byte) error {
    type Alias Operation
    if err := json.Unmarshal(data, (*Alias)(o)); err != nil {  // Parse #1
        return err
    }
    o.Extra = jsonhelpers.ExtractExtensions(data)  // Parse #2 (wasteful!)
    return nil
}
```

## Solution: Two-Phase Approach

### Phase 1: Streaming Scan (Implemented ✅)

Add a `bytes.Contains` check before JSON parsing:

```go
func ExtractExtensions(data []byte) map[string]any {
    // Fast path: skip JSON parsing if no extension pattern found
    if !bytes.Contains(data, []byte(`"x-`)) {
        return nil
    }
    // ... existing implementation
}
```

**Results**:
| Scenario | Before | After | Speedup |
|----------|--------|-------|---------|
| No extensions | 2,818 ns, 65 allocs | 186 ns, 0 allocs | **15.2x** |
| With extensions | 2,290 ns | 2,347 ns | ~same |

**Why this works**:
- The pattern `"x-` (with opening quote) reliably identifies potential extension keys
- String values like `"Use x-api-key header"` don't match (no `"x-` sequence)
- URLs like `http://x-server.com` don't match (no opening quote)
- False positives (nested keys, array elements) still return correct results

**Corpus analysis** shows most specs have few/no extensions:
- Plaid: 0 extensions
- Petstore: 0 extensions
- Google Maps: 0 extensions
- Discord: 16 extensions
- Stripe: 2,433 extensions
- GitHub: 1,371 extensions

### Phase 2: Single-Parse Refactor (Deferred)

The ideal solution would eliminate the double-parse entirely by using a map-first approach:

```go
func (c *Contact) UnmarshalJSON(data []byte) error {
    var m map[string]any
    if err := json.Unmarshal(data, &m); err != nil {  // Single parse
        return err
    }

    c.Name = getString(m, "name")
    c.Email = getString(m, "email")
    c.Extra = extractFromMap(m, contactKnownFields)
    return nil
}
```

**Why this is deferred**:

1. **Nested types require re-marshaling**: For types like `Operation` with nested `[]*Parameter` or `map[string]*Response`, we must:
   - Extract the nested value from `map[string]any`
   - `json.Marshal` it back to `[]byte`
   - Call the nested type's `UnmarshalJSON`
   - This recreates the double-parse problem for nested objects

2. **High refactoring effort**: 30+ `UnmarshalJSON` methods need rewriting, each with:
   - A `knownFields` map to define
   - Type assertions for each field
   - Complex handling for slices, maps, and nested structs

3. **Marginal benefit**: Phase 1 already captures most of the performance gain for the common case. Phase 2 would only help specs with many extensions (Stripe, GitHub).

4. **Regression risk**: Manual field extraction is error-prone and harder to maintain than the current Alias pattern.

### Alternative Phase 2 Approaches (Not Recommended)

1. **Code generation**: Generate the map-first `UnmarshalJSON` implementations
   - Pro: Reduces manual effort
   - Con: Adds build complexity, still has nested object problem

2. **Lazy extension extraction**: Store raw bytes, parse on first access to `Extra`
   - Pro: Zero overhead if extensions never accessed
   - Con: Memory overhead from storing raw bytes

3. **Custom JSON decoder**: Stream-parse to extract `x-*` keys without full parse
   - Pro: Maximum performance
   - Con: Very complex implementation, edge cases

## Implementation Notes

### Files Changed (Phase 1)

- `parser/internal/jsonhelpers/helpers.go`: Added `bytes` import and streaming check
- `parser/internal/jsonhelpers/helpers_test.go`: Added comprehensive tests and benchmarks

### Testing

- All 8,068 existing tests pass
- All 15 corpus integration tests pass
- New edge case tests cover: nested keys, array elements, false positives, minimum extension names

### Benchmarks

```
BenchmarkExtractExtensions_NoExtensions     186.0 ns/op    0 B/op    0 allocs/op
BenchmarkExtractExtensions_WithExtensions  2347 ns/op   2512 B/op   57 allocs/op
BenchmarkExtractExtensions_FalsePositive    108 ns/op      0 B/op    0 allocs/op
```

## Future Work

If profiling shows extension extraction remains a bottleneck for extension-heavy specs:

1. Consider lazy extraction for types that rarely access `Extra`
2. Explore `github.com/goccy/go-json` or `github.com/bytedance/sonic` for faster parsing
3. Profile specific types to identify highest-impact refactoring targets
