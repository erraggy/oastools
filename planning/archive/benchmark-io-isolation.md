# Benchmark I/O Isolation Plan

## Status: ✅ Complete - Merged

**Branch:** `perf/benchmark-io-isolation`
**PR:** #148 (merged 2025-12-16)

## Problem Statement

Benchmark comparisons between v1.25.0 and v1.28.1 showed apparent regressions:
- Parser: +51% (143µs → 217µs)
- Joiner: +82% (103µs → 188µs)

## Root Cause Analysis

**Finding:** No actual code regression exists. The "regression" was caused by I/O variance in saved benchmark files.

When running both versions back-to-back on the same machine:
- v1.25.0 Parse/SmallOAS3: 176 µs
- HEAD Parse/SmallOAS3: 173 µs
- **Difference: ~0%**

The saved v1.25.0 benchmark was recorded under optimal I/O conditions (cached filesystem), making it appear artificially fast.

## Solution Implemented

### New Benchmarks Added

1. **`BenchmarkParseCore`** (parser/parser_bench_test.go)
   - Pre-loads all test files in setup
   - Benchmarks only parsing logic (no file I/O in loop)
   - Covers: SmallOAS3, MediumOAS3, LargeOAS3, SmallOAS2, MediumOAS2

2. **`BenchmarkJoinParsed`** (joiner/joiner_bench_test.go)
   - Pre-parses all documents in setup
   - Benchmarks only joining logic
   - Covers: TwoDocs, ThreeDocs, FiveDocs

### Documentation Added

Both benchmark files now include detailed comments explaining:
- I/O variance can be +/- 50%
- Which benchmarks are reliable for regression detection
- Recommendation to use I/O-isolated benchmarks (`BenchmarkParseCore`, `BenchmarkJoinParsed`) for CI

## Verification

All tests pass: `make check` ✓

New benchmark results (I/O-free):
```
BenchmarkParseCore/SmallOAS3     122 µs
BenchmarkParseCore/MediumOAS3    1.1 ms
BenchmarkJoinParsed/TwoDocs      778 ns
BenchmarkJoinParsed/ThreeDocs    982 ns
```

## Remaining Work

- [x] Create feature branch
- [x] Add BenchmarkParseCore
- [x] Add FiveDocs case to BenchmarkJoinParsed
- [x] Add documentation to benchmark files
- [x] Run make check
- [x] Commit changes
- [x] Push branch to origin
- [x] Create PR (#148)
- [x] Add comprehensive documentation to CLAUDE.md, benchmarks.md, BENCHMARK_UPDATE_PROCESS.md
- [x] Merge PR after review (2025-12-16)

## Related Investigation

The initial investigation also confirmed:
- **Fixer package:** Had a real regression in v1.28.0, already fixed in v1.28.1 (commit 257b99a)
- **Parser package:** No regression, source map feature is opt-in with zero overhead when disabled
- **Converter package:** Inherits parser behavior, no code regression
- **Joiner package:** No code regression, joining logic unchanged (JoinParsed: 795ns → 796ns)
