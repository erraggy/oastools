# sync.Pool Performance Comparison

**Branch:** `feat/sync-pool-optimization` vs `main`  
**Date:** 2026-01-12  
**Tool:** `benchstat` with `-count=6`

## Summary

| Pool | Impact | Key Metric |
|------|--------|------------|
| Marshal Buffer (parser) | ✅ **20-58% memory reduction** | `MarshalOrderedJSON` operations |
| Template Buffer (generator) | ✅ **Allocation reduction** | `Generate/Types` -0.38% allocs |
| Walker Context | ⚪ No change | Pre-existing pool |

## Key Wins: Marshal Buffer Pool

### Memory Allocation Reduction (B/op)

| Benchmark | main | feat | Change |
|-----------|------|------|--------|
| `MarshalOrderedJSON/SmallOAS3` | 4.611Ki | 3.675Ki | **-20.31%** |
| `MarshalOrderedJSON/MediumOAS3` | 61.96Ki | 39.53Ki | **-36.20%** |
| `MarshalOrderedJSON/LargeOAS3` | 616.4Ki | 472.8Ki | **-23.30%** |
| `MarshalOrderedJSONIndent` | 144.74Ki | 59.60Ki | **-58.82%** |

### Allocation Count Reduction (allocs/op)

| Benchmark | main | feat | Change |
|-----------|------|------|--------|
| `MarshalOrderedJSON/SmallOAS3` | 160 | 156 | **-2.50%** |
| `MarshalOrderedJSON/MediumOAS3` | 1,532 | 1,524 | **-0.52%** |
| `MarshalOrderedJSONIndent` | 1,535 | 1,525 | **-0.65%** |

### Speed Improvement (ns/op)

| Benchmark | main | feat | Change |
|-----------|------|------|--------|
| `MarshalOrderedJSON/MediumOAS3` | 99.56µs | 97.24µs | **-2.33%** |
| `MarshalOrderedJSON/LargeOAS3` | 1.096ms | 1.076ms | **-1.83%** |
| `MarshalOrderedJSONIndent` | 154.3µs | 141.1µs | **-8.52%** |

## Pool Microbenchmarks

Direct comparison of pooled vs non-pooled operations:

| Benchmark | Time | Memory | Allocs |
|-----------|------|--------|--------|
| `MarshalBufferPool` | 8.2ns | **0 B/op** | **0 allocs** |
| `MarshalBufferNoPool` | 641.9ns | 4.0Ki | 2 allocs |

**Result:** 78x faster, 100% allocation elimination when using the pool.

## Trade-offs Observed

Some individual `MarshalJSON` operations show slight time overhead (6-9%):

| Benchmark | Change | Notes |
|-----------|--------|-------|
| `MarshalOAS3Document/Small` | +8.99% ns/op | Pool acquire/release overhead |
| `MarshalServer/NoExtra` | +9.22% ns/op | Small payloads don't benefit as much |

This is expected — pool management has overhead. The wins come from:
1. **Aggregate operations** (MarshalOrderedJSON calls many MarshalJSON internally)
2. **GC pressure reduction** (fewer allocations = less GC work under load)
3. **Memory reuse** (hot paths benefit from warm buffers)

## Generator Template Buffer Pool

| Benchmark | main | feat | Change |
|-----------|------|------|--------|
| `Generate/Types` allocs/op | 1,317 | 1,312 | **-0.38%** |
| `Generate/Client` allocs/op | 8,511 | 8,509 | **-0.03%** |
| `Generate/Server` allocs/op | 2,479 | 2,476 | **-0.12%** |

The generator pool shows modest allocation reduction. The tiered buffer approach prevents oversized allocations.

## Methodology

```bash
# Baseline (main branch)
git checkout main
go test -bench=. -benchmem -count=6 ./parser/... ./generator/... ./walker/... \
    2>/dev/null | tee /tmp/bench-main.txt

# Feature branch
git checkout feat/sync-pool-optimization
go test -bench=. -benchmem -count=6 ./parser/... ./generator/... ./walker/... \
    2>/dev/null | tee /tmp/bench-feat.txt

# Comparison
benchstat /tmp/bench-main.txt /tmp/bench-feat.txt
```

## Conclusion

The sync.Pool optimizations deliver measurable improvements:
- **20-58% memory reduction** for JSON marshaling operations
- **78x faster** buffer acquisition in hot paths
- **Modest allocation reduction** in code generation

The pools focus on internal operations (marshal buffers, template buffers) where objects don't escape to callers, avoiding use-after-put bugs that led to removing 9 of the originally planned 12 pools.
