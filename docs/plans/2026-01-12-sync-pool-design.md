# sync.Pool Implementation Design

**Date:** 2026-01-12
**Status:** Ready for Implementation
**Scope:** 12 pools (8 high-impact, 4 medium-impact)
**Approach:** Single PR with comprehensive benchmarks

---

## Overview

Implement Go's `sync.Pool` across the oastools codebase to reduce GC pressure and improve performance. All capacity decisions are data-driven from corpus analysis of 10 real-world OpenAPI specs (19,147 operations, 360,577 schemas).

### Expected Impact

| Metric | Current | Target | Improvement |
|--------|---------|--------|-------------|
| Large doc parse allocs | 194,712 | ~160,000 | 18% fewer |
| Marshal allocs (large) | 5,336 | ~4,000 | 25% fewer |
| HTTP request allocs | 8 | ~2 | 75% fewer |
| DeepCopy allocs (large) | 4,877 | ~4,000 | 18% fewer |
| Diff allocs (identical) | 115 | ~90 | 22% fewer |

---

## Architecture

### Pool Pattern

Every pool follows this template:

```go
var thingPool = sync.Pool{
    New: func() interface{} { return newThing() },
}

func getThing() *Thing {
    t := thingPool.Get().(*Thing)
    t.Reset()  // Always reset on get
    return t
}

func putThing(t *Thing) {
    if t == nil || t.isOversized() {
        return  // Size guard
    }
    thingPool.Put(t)
}
```

### Key Principles

1. **Reset-on-Get** — Always reset pooled objects when retrieving
2. **Size guards** — Prevent memory leaks from oversized objects
3. **No instrumentation** — Keep simple, use standard Go profiling when needed

---

## High-Impact Pools (1-8)

### 1. Marshal Buffer Pool

**Package:** `parser/pool.go`

| Setting | Value | Source |
|---------|-------|--------|
| Initial capacity | 4KB | Covers most fields |
| Max pooled size | 1MB | Prevent memory leaks |

**Used by:** `MarshalJSON()`, `MarshalYAML()`, ordered marshal functions

**Expected impact:** 20-30% fewer marshal allocations

---

### 2. HTTP Validator Context Pool

**Package:** `httpvalidator/pool.go`

| Setting | Value | Source |
|---------|-------|--------|
| Errors slice | 8 | Typical max errors |
| Warnings slice | 4 | Typical max warnings |

**Used by:** `ValidateRequest()`, `ValidateResponse()`

**Expected impact:** 50-75% fewer allocations per request (8 → ~2)

---

### 3. String Builder Pool

**Package:** `internal/issues/pool.go`

| Setting | Value | Source |
|---------|-------|--------|
| No size limit | — | Builders are lightweight |

**Used by:** `formatPath()`, issue message construction

**Expected impact:** 5-10% fewer string allocations

---

### 4. Slice Pre-allocation Pools

**Package:** `parser/pool.go`

| Slice Type | Capacity | Corpus Source |
|------------|----------|---------------|
| `[]*Parameter` | 4 | p75=2, p90=8 |
| `map[string]*Response` | 4 | p95=4 |
| `[]*Server` | 2 | max=2 across all specs |
| `[]string` (tags) | 2 | p99=1 |

**Used by:** Various `UnmarshalJSON()` methods

**Expected impact:** 10-15% fewer slice allocations

---

### 5. DeepCopy Work Slice Pool

**Package:** `parser/pool.go`

| Setting | Value | Corpus Source |
|---------|-------|---------------|
| Initial capacity | 16 | Schema depth: P99=3, max=9 |
| Max pooled size | 256 | — |

**Used by:** Generated `DeepCopy()` methods

**Expected impact:** 15-20% fewer DeepCopy allocations

---

### 6. JSONPath Expression Pool

**Package:** `internal/jsonpath/pool.go`

| Pool | Capacity | Corpus Source |
|------|----------|---------------|
| Node slice | 8 | Overlay patterns 3-4 tokens |
| Stack | 8 | Same as nodes |
| Results | 32 | Varies by filter |

**Used by:** Overlay `Apply()`, JSONPath `Evaluate()`

**Expected impact:** 20-30% fewer overlay allocations

---

### 7. Walker Context Pool

**Package:** `walker/pool.go`

| Setting | Value | Corpus Source |
|---------|-------|---------------|
| Path capacity | 16 | P99=14, max=14 |
| Ancestors capacity | 16 | Same as path |

**Used by:** `Walk()`, `WalkSchema()`, typed handler traversals

**Expected impact:** 10-15% fewer walk allocations

---

### 8. Conversion Map Pool

**Package:** `converter/pool.go`

| Setting | Value | Corpus Source |
|---------|-------|---------------|
| Initial capacity | 8192 | P75=6,319 refs |
| Max pooled size | 16384 | msgraph has 73K |

**Used by:** `ConvertToOAS3()`, `ConvertToOAS2()`

**Expected impact:** 5-8% fewer conversion allocations

---

## Medium-Impact Pools (9-12)

### 9. Differ Change Slice Pool

**Package:** `differ/pool.go`

| Setting | Value | Corpus Source |
|---------|-------|---------------|
| Capacity | 16 | median=12, p95=13 |

**Used by:** `Diff()`, `DiffParsed()`

**Expected impact:** 20-30% fewer differ allocations

---

### 10. Generator Template Buffer Pool

**Package:** `generator/pool.go`

| Tier | Capacity | Condition |
|------|----------|-----------|
| Small | 8KB | <10 operations |
| Medium | 32KB | 10-50 operations |
| Large | 64KB | 50+ operations |

**Formula:** `(ops × 350) + (schemas × 150) + (paths × 75)` bytes

**Used by:** Code generation templates

**Expected impact:** 5-10% improvement for full generation

---

### 11. Builder Component Map Pool

**Package:** `builder/pool.go`

| Map Type | Capacity | Source |
|----------|----------|--------|
| Schema map | 8 | Small incremental builds |
| Path map | 4 | — |
| Operations slice | 8 | — |

**Used by:** Programmatic spec construction

**Expected impact:** 2-5% improvement

---

### 12. Fixer Issue Collection Pool

**Package:** `fixer/pool.go`

| Setting | Value | Corpus Source |
|---------|-------|---------------|
| Capacity | 4 | p95=3 fixes |
| Initialization | Lazy | median=0 fixes |

**Used by:** Fix analysis

**Expected impact:** 3-5% improvement (when fixes exist)

---

## File Structure

### New Files (20 total)

```
parser/pool.go                    # ~150 lines
parser/pool_test.go
internal/issues/pool.go           # ~40 lines
internal/issues/pool_test.go
internal/jsonpath/pool.go         # ~60 lines
internal/jsonpath/pool_test.go
walker/pool.go                    # ~50 lines
walker/pool_test.go
converter/pool.go                 # ~50 lines
converter/pool_test.go
httpvalidator/pool.go             # ~50 lines
httpvalidator/pool_test.go
differ/pool.go                    # ~40 lines
differ/pool_test.go
generator/pool.go                 # ~70 lines
generator/pool_test.go
builder/pool.go                   # ~50 lines
builder/pool_test.go
fixer/pool.go                     # ~40 lines
fixer/pool_test.go
```

### Files to Modify (16 total)

| File | Changes |
|------|---------|
| `parser/marshal.go` | Use buffer pool |
| `parser/ordered_marshal.go` | Use buffer pool |
| `parser/zz_generated_deepcopy.go` | Use work pool |
| `parser/oas3_types.go` | Use slice pools |
| `parser/oas2_types.go` | Use slice pools |
| `internal/issues/issues.go` | Use string builder pool |
| `internal/jsonpath/jsonpath.go` | Use expression pools |
| `walker/walker.go` | Use context pool |
| `converter/convert.go` | Use map pool |
| `converter/oas2_to_oas3.go` | Use map pool |
| `converter/oas3_to_oas2.go` | Use map pool |
| `httpvalidator/validate.go` | Use request context pool |
| `differ/diff.go` | Use change slice pool |
| `generator/generate.go` | Use tiered buffer pool |
| `builder/builder.go` | Use component map pools |
| `fixer/fixer.go` | Use issue collection pool |

---

## Testing Strategy

### Short-Term (Analysis Phase)

1. **Before/After Allocation Profiles**
   ```bash
   go test -bench=. -benchmem -memprofile=before.prof
   # Implement pools
   go test -bench=. -benchmem -memprofile=after.prof
   ```

2. **Corpus Validation**
   - Parse all 10 corpus specs with/without pools
   - Measure: allocs/op, bytes/op, ns/op
   - Target: 15-30% allocation reduction for large specs

3. **Pool Effectiveness Analysis**
   - Temporarily add hit/miss counters during development
   - Validate capacities match corpus predictions
   - Remove instrumentation before merge

### Long-Term (CI Maintained)

| Benchmark | Location | Purpose |
|-----------|----------|---------|
| `BenchmarkMarshalJSON_Pool` | `parser/pool_test.go` | Buffer pool |
| `BenchmarkValidateRequest_Pool` | `httpvalidator/pool_test.go` | HTTP context |
| `BenchmarkDeepCopy_Pool` | `parser/pool_test.go` | Work pool |
| `BenchmarkWalk_Pool` | `walker/pool_test.go` | Context pool |
| `BenchmarkConvert_Pool` | `converter/pool_test.go` | Map pool |
| `BenchmarkDiff_Pool` | `differ/pool_test.go` | Change slice |

### Race Detection

```bash
go test -race ./parser ./walker ./converter ./differ \
    ./httpvalidator ./fixer ./generator ./builder ./internal/...
```

---

## Implementation Order

| Priority | Pool | Risk | Effort |
|----------|------|------|--------|
| 1 | Marshal Buffer Pool | Low | Low |
| 2 | HTTP Validator Context Pool | Low | Low |
| 3 | String Builder Pool | Low | Low |
| 4 | Slice Pre-allocation Pools | Medium | Medium |
| 5 | JSONPath Expression Pool | Low | Low |
| 6 | Conversion Map Pool | Low | Low |
| 7 | Walker Context Pool | Low | Low |
| 8 | DeepCopy Work Pool | Medium | Medium |
| 9 | Differ Change Slice Pool | Low | Low |
| 10 | Generator Template Buffer Pool | Low | Medium |
| 11 | Builder Component Map Pool | Low | Low |
| 12 | Fixer Issue Collection Pool | Low | Low |

---

## Corpus Analysis Reference

Analysis performed on 10 real-world specs:
- asana, digitalocean, discord, github, google-maps, msgraph, nws, petstore, plaid, stripe

### Key Statistics

| Metric | P50 | P75 | P90 | P99 | Max |
|--------|-----|-----|-----|-----|-----|
| Parameters/operation | 1 | 2 | 8 | 15 | 127 |
| Responses/operation | 3 | 3 | 4 | 4 | 7 |
| Tags/operation | 1 | 1 | 1 | 1 | 3 |
| Servers/document | 1 | 1 | 2 | 2 | 2 |
| Document nesting depth | 12 | 13 | 14 | 14 | 14 |
| Schema nesting depth | 0 | 0 | 1 | 3 | 9 |
| $refs/document | 3,539 | 6,319 | 9,309 | 73,511 | 73,511 |
| Diff changes | 12 | 13 | 13 | 13 | 13 |
| Fixer issues | 0 | 1 | 2 | 3 | 10 |

---

## Risk Mitigation

### Gotchas to Avoid

| Gotcha | Mitigation |
|--------|------------|
| Forgetting to reset | Always wrap Get() with Reset() call |
| Returning oversized objects | Cap maximum pooled size |
| Use-after-put bugs | Clear references after Put() |
| Pool contention | Profile under concurrent load |
| Memory leaks | Don't pool objects with external references |

### Areas Where sync.Pool Is NOT Used

1. **Parsed document structures** — Long-lived, user-retained
2. **Component definitions** — Part of document tree
3. **Validation results** — Returned to caller

---

## Appendix: Scripts Created

| Script | Purpose |
|--------|---------|
| `scripts/analyze_corpus.go` | Initial capacity analysis |
| `scripts/analyze_corpus_depth.go` | Nesting depth analysis |
| `scripts/analyze_corpus_medium.go` | Differ/fixer statistics |

---

*Design validated through collaborative brainstorming with corpus-driven capacity analysis.*
