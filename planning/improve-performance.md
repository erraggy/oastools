# Performance Improvement Plan

> **Purpose**: This document tracks performance optimization strategies for oastools.
> It serves as a reference across multiple work sessions to maintain continuity.

## Current Status

- **Last Updated**: 2025-11-17
- **Current Phase**: Phase 1 Complete - Baseline Established
- **Benchmarking**: ✅ Implemented and baseline captured (see benchmark-baseline.txt)

---

## Overview of Performance Bottlenecks

Based on codebase analysis, the following areas have been identified as potential performance bottlenecks:

1. **Custom JSON Marshalers** - Double-marshal approach in parser/*_json.go
2. **Reference Resolution** - Recursive traversal with document caching
3. **Parser Re-marshaling** - Conditional re-marshal when ResolveRefs is enabled
4. **Validation** - Recursive validation of deeply nested schemas
5. **Memory Allocation** - Slice/map allocations without pre-sizing
6. **Deep Copying** - Future concern when deep copy utilities are added

---

## Strategy 1: Optimize Custom JSON Marshalers

### Current Implementation

Location: `parser/common_json.go`, `parser/oas2_json.go`, `parser/oas3_json.go`, etc.

```go
func (i *Info) MarshalJSON() ([]byte, error) {
    type Alias Info
    aux, err := json.Marshal((*Alias)(i))  // 1st marshal
    if err != nil {
        return nil, err
    }

    if len(i.Extra) == 0 {
        return aux, nil  // Early exit optimization (already in place)
    }

    var m map[string]interface{}
    if err := json.Unmarshal(aux, &m); err != nil {  // Unmarshal
        return nil, err
    }

    for k, v := range i.Extra {
        m[k] = v
    }

    return json.Marshal(m)  // 2nd marshal
}
```

**Impact**: For deeply nested documents with many Extra fields (specification extensions like `x-*`), this compounds significantly.

### Option A: Manual Field Serialization

**Description**: Manually write JSON bytes instead of using marshal/unmarshal/marshal pattern.

**Pros**:
- Eliminates 2 marshal + 1 unmarshal operations per struct
- Single-pass encoding directly to output
- Maximum performance gain (estimated 60-80% faster for marshal operations)
- Full control over output format

**Cons**:
- Significant implementation effort (~40+ custom marshalers to rewrite)
- More complex code that's harder to maintain
- Must handle JSON escaping manually
- Risk of subtle bugs in manual serialization
- Must keep in sync with struct field changes

**Estimated Impact**: High (60-80% improvement in marshal performance)
**Implementation Complexity**: High
**Risk**: Medium-High (correctness concerns)

### Option B: Optimize with json.RawMessage

**Description**: Pre-marshal known fields to RawMessage, merge at map level only once.

**Pros**:
- Reduces to 1 marshal + 1 merge operation
- Moderate complexity (easier than manual serialization)
- Leverages standard library for correctness
- ~40-50% performance improvement estimated

**Cons**:
- Still has some marshal overhead
- More complex than current approach
- Requires careful handling of RawMessage lifecycle

**Estimated Impact**: Medium-High (40-50% improvement)
**Implementation Complexity**: Medium
**Risk**: Low-Medium

### Option C: Document Performance Tradeoff (No Code Change)

**Description**: Add godoc comments explaining the performance tradeoff and why this approach was chosen.

**Pros**:
- Zero implementation effort
- Zero risk of introducing bugs
- Makes conscious decision explicit to users/contributors
- Allows deferring optimization until proven necessary

**Cons**:
- No performance improvement
- May disappoint users with performance-critical use cases

**Estimated Impact**: None
**Implementation Complexity**: Trivial
**Risk**: None

### Recommendation

**For v1.7.0**: Option C (Document) + establish benchmarking
**For v1.8.0+**: Option B (RawMessage optimization) if benchmarks show significant impact

---

## Strategy 2: Optimize Reference Resolution

### Current Implementation

Location: `parser/resolver.go`

**Current approach**:
- Recursive traversal of entire document tree
- Visited map to detect circular references
- Document cache (max 100 documents)
- File size limit (10MB)

**Potential bottlenecks**:
- Full document traversal even for documents with few/no refs
- Visited map allocations for each resolution
- No early exit when no refs present

### Option A: Two-Pass Reference Detection

**Description**: First pass scans for presence of `$ref` before doing full resolution.

**Pros**:
- Avoids expensive resolution when no refs present
- Simple to implement (add pre-scan step)
- No impact on documents with refs
- Safe optimization (only adds fast path)

**Cons**:
- Extra traversal for documents with refs (minimal cost)
- Marginal benefit if most documents have refs

**Estimated Impact**: Medium for docs without refs, negligible for docs with refs (20-30% improvement for no-ref case)
**Implementation Complexity**: Low
**Risk**: Very Low

### Option B: Lazy Reference Resolution

**Description**: Resolve refs on-demand during access rather than upfront.

**Pros**:
- Only resolve refs that are actually accessed
- Better for partial document processing

**Cons**:
- Significant API change (breaking change)
- Complexity in access patterns
- Harder to detect circular references
- May complicate error handling

**Estimated Impact**: Variable (depends on access patterns)
**Implementation Complexity**: Very High
**Risk**: High (breaking change)

### Option C: Parallel Reference Resolution

**Description**: Resolve independent refs in parallel using goroutines.

**Pros**:
- Can utilize multiple CPU cores
- Good for documents with many external refs
- No breaking changes to API

**Cons**:
- Complexity in coordinating goroutines
- Overhead for small documents
- Need careful locking for shared caches
- May not help much for single external doc

**Estimated Impact**: Medium-High for multi-ref docs (30-50% improvement)
**Implementation Complexity**: Medium-High
**Risk**: Medium (concurrency bugs)

### Recommendation

**For v1.7.0**: Option A (Two-pass detection) - low risk, good ROI
**For v1.8.0+**: Consider Option C (Parallel) if benchmarks show multi-ref documents are common

---

## Strategy 3: Optimize Parser Re-marshaling

### Current Implementation

Location: `parser/parser.go:213-260`

```go
var parseData []byte
if p.ResolveRefs {
    // Re-marshal the data with resolved refs
    parseData, err = yaml.Marshal(rawData)  // Expensive!
    if err != nil {
        return nil, fmt.Errorf("parser: failed to re-marshal data: %w", err)
    }
} else {
    // Use original data directly
    parseData = data
}
```

**Issue**: When `ResolveRefs=true`, we marshal the entire resolved document before parsing again.

### Option A: Direct Struct Population from Map

**Description**: Instead of re-marshaling to bytes then unmarshaling, populate structs directly from `map[string]interface{}`.

**Pros**:
- Eliminates one full marshal/unmarshal cycle
- Significant performance improvement (40-60% faster parsing with refs)
- Cleaner data flow

**Cons**:
- Complex implementation (manual struct population)
- Must handle all OAS field types correctly
- Significant refactoring required

**Estimated Impact**: High (40-60% improvement when ResolveRefs=true)
**Implementation Complexity**: Very High
**Risk**: High (correctness concerns)

### Option B: Use mapstructure Library

**Description**: Use `mitchellh/mapstructure` to decode map to struct without marshal/unmarshal.

**Pros**:
- Eliminates re-marshal step
- Well-tested library
- Moderate complexity
- Good performance improvement (30-40%)

**Cons**:
- New dependency
- Need to configure struct tags
- May not handle all edge cases

**Estimated Impact**: Medium-High (30-40% improvement)
**Implementation Complexity**: Medium
**Risk**: Low-Medium

### Option C: Keep Current Approach

**Description**: Accept the performance cost as necessary for correctness.

**Pros**:
- No implementation effort
- Current approach is proven correct
- ResolveRefs is optional (users can disable)

**Cons**:
- Performance impact remains
- May discourage use of ResolveRefs feature

**Estimated Impact**: None
**Implementation Complexity**: None
**Risk**: None

### Recommendation

**For v1.7.0**: Option C (Keep current) - optimize other areas first
**For v1.8.0+**: Option B (mapstructure) if ref resolution performance becomes issue

---

## Strategy 4: Optimize Validation

### Current Implementation

Location: `validator/validator.go`

**Current approach**:
- Recursive schema validation
- No caching of validation results
- Validates every field on every call

### Option A: Validation Result Caching

**Description**: Cache validation results for schemas by content hash.

**Pros**:
- Avoid re-validating identical schemas
- Good for documents with many repeated schema refs
- Moderate implementation complexity

**Cons**:
- Memory overhead for cache
- Need to compute hashes
- May not help if schemas are all unique
- Cache invalidation complexity

**Estimated Impact**: Medium for docs with repeated schemas (20-40% improvement)
**Implementation Complexity**: Medium
**Risk**: Low-Medium

### Option B: Early Exit Optimizations

**Description**: Return early from validation when possible (e.g., empty documents, missing required fields).

**Pros**:
- Simple to implement
- Safe optimization
- Helps with invalid documents
- No breaking changes

**Cons**:
- Limited impact on valid documents
- May mask multiple errors (but current impl already does this in some cases)

**Estimated Impact**: Low-Medium (10-20% improvement for invalid docs)
**Implementation Complexity**: Low
**Risk**: Very Low

### Option C: Parallel Validation

**Description**: Validate independent paths/schemas in parallel.

**Pros**:
- Utilize multiple cores
- Good for large documents
- No API changes

**Cons**:
- Complexity in goroutine coordination
- Need to collect errors safely
- Overhead for small documents

**Estimated Impact**: Medium-High for large docs (30-50% improvement)
**Implementation Complexity**: Medium-High
**Risk**: Medium

### Recommendation

**For v1.7.0**: Option B (Early exit) - low risk, easy wins
**For v1.8.0+**: Option A (Caching) or C (Parallel) based on benchmark data

---

## Strategy 5: Memory Allocation Optimization

### Current Implementation

**Issue**: Throughout codebase, slices and maps are allocated without pre-sizing:

```go
errors := make([]error, 0)  // No capacity hint
warnings := make([]string, 0)  // No capacity hint
```

### Option A: Pre-allocate with Reasonable Capacity

**Description**: Use capacity hints based on common document sizes.

**Example**:
```go
errors := make([]error, 0, 10)  // Reserve space for 10 errors
warnings := make([]string, 0, 10)
```

**Pros**:
- Simple to implement
- Reduces allocations and GC pressure
- Low risk
- 5-15% performance improvement estimated

**Cons**:
- Need to choose good capacity values
- May waste memory if oversized
- Minimal impact if documents are small

**Estimated Impact**: Low-Medium (5-15% improvement)
**Implementation Complexity**: Very Low
**Risk**: Very Low

### Option B: sync.Pool for Reusable Buffers

**Description**: Use sync.Pool to recycle byte buffers and slices.

**Pros**:
- Reduces GC pressure significantly
- Good for high-throughput scenarios
- Proven pattern in Go stdlib

**Cons**:
- More complex to implement correctly
- Need to reset state carefully
- May not help for single-use CLI tool
- Better for library use cases

**Estimated Impact**: Medium for high-throughput (20-30% improvement)
**Implementation Complexity**: Medium
**Risk**: Low-Medium

### Recommendation

**For v1.7.0**: Option A (Pre-allocation) - quick wins throughout codebase
**For v1.8.0+**: Consider Option B (sync.Pool) if library usage patterns show benefit

---

## Strategy 6: Benchmarking Infrastructure (PREREQUISITE)

### Priority: HIGHEST - Do This First!

**Description**: Establish comprehensive benchmarking before attempting optimizations.

### What to Benchmark

1. **Parser Operations**
   - Parse with/without ref resolution
   - Parse small/medium/large documents
   - JSON vs YAML parsing

2. **Marshaling Operations**
   - Custom marshalers with/without Extra fields
   - Deeply nested vs flat structures

3. **Validation Operations**
   - Validate small/medium/large documents
   - Valid vs invalid documents

4. **Conversion Operations**
   - OAS 2.0 → 3.0.3
   - OAS 3.0.3 → 2.0
   - Documents with/without extensions

5. **Joiner Operations**
   - Join 2, 5, 10 documents
   - With/without collisions

### Implementation

**Location**: Add benchmark files next to implementation:
- `parser/parser_bench_test.go`
- `parser/json_bench_test.go`
- `validator/validator_bench_test.go`
- `converter/converter_bench_test.go`
- `joiner/joiner_bench_test.go`

**Test Fixtures**: Create representative test files in `testdata/`:
- `testdata/bench/small-oas3.yaml` (~50 lines)
- `testdata/bench/medium-oas3.yaml` (~500 lines)
- `testdata/bench/large-oas3.yaml` (~5000 lines)
- Similar for OAS 2.0, with/without extensions, etc.

**Pros**:
- Provides objective data for optimization decisions
- Prevents premature optimization
- Tracks regressions
- Validates improvement claims

**Cons**:
- Initial time investment
- Need to maintain benchmark suite

**Estimated Impact**: None directly, but enables all other optimizations
**Implementation Complexity**: Low-Medium
**Risk**: None

### Recommendation

**MUST DO FIRST** before implementing any performance optimizations!

---

## Strategy 7: Profiling and Instrumentation

### Description

Add profiling support to identify actual bottlenecks in real-world usage.

### Option A: CPU and Memory Profiling

**Tools**: `pprof`, `go test -bench -cpuprofile -memprofile`

**Pros**:
- Identifies actual bottlenecks (not guesses)
- Standard Go tooling
- Can profile production usage

**Cons**:
- Requires representative workload
- Interpretation can be complex

**Estimated Impact**: Enables targeted optimization
**Implementation Complexity**: Low
**Risk**: None

### Option B: Trace Analysis

**Tools**: `go test -trace`, execution tracer

**Pros**:
- Visualizes goroutine execution
- Good for concurrency issues
- Identifies blocking operations

**Cons**:
- Large trace files
- Best for concurrent operations

**Estimated Impact**: Useful for parallel strategies
**Implementation Complexity**: Low
**Risk**: None

### Recommendation

Use in conjunction with benchmarking to guide optimization efforts.

---

## Implementation Roadmap

### Phase 1: Baseline (v1.7.0)
**Status**: ✅ Complete
**Goal**: Establish measurement infrastructure

1. ✅ Create this planning document
2. ✅ Implement comprehensive benchmark suite (Strategy 6)
3. ✅ Run benchmarks on current codebase (baseline)
4. ⬜ Set up profiling for real-world documents
5. ⬜ Document performance characteristics in README

**Deliverables**:
- ✅ Benchmark suite (15 test files, 60+ benchmarks)
- ✅ Baseline metrics captured (see actual results below)
- ⬜ Profiling scripts (deferred to Phase 2)

### Phase 2: Low-Risk Quick Wins (v1.7.0)
**Status**: Not started
**Dependencies**: Phase 1 complete

1. ⬜ Memory allocation optimization (Strategy 5A)
2. ⬜ Validation early exits (Strategy 4B)
3. ⬜ Reference resolution two-pass (Strategy 2A)
4. ⬜ Document marshaler tradeoffs (Strategy 1C)

**Deliverables**: 10-20% overall performance improvement, updated docs

### Phase 3: Medium-Risk Optimizations (v1.8.0)
**Status**: Not planned
**Dependencies**: Phase 2 complete, benchmarks justify effort

Choose based on benchmark data:
1. ⬜ JSON marshaler optimization (Strategy 1B or 1A)
2. ⬜ Validation caching (Strategy 4A) OR parallel validation (Strategy 4C)
3. ⬜ Parallel reference resolution (Strategy 2C)

**Deliverables**: 30-50% improvement in targeted operations

### Phase 4: Advanced Optimizations (v1.9.0+)
**Status**: Future consideration
**Dependencies**: Phase 3 complete, proven need

1. ⬜ Parser re-marshaling elimination (Strategy 3B)
2. ⬜ sync.Pool for buffers (Strategy 5B)
3. ⬜ Custom implementations based on profiling data

**Deliverables**: 50%+ improvement for high-throughput scenarios

---

## Success Metrics

### Performance Targets

Based on document size categories:

| Operation | Document Size | Baseline (actual) | Target v1.7.0 | Target v1.8.0 |
|-----------|--------------|-------------------|---------------|---------------|
| Parse (no refs) | Small (~60 lines) | 140μs | 126μs (10%) | 112μs (20%) |
| Parse (no refs) | Medium (~570 lines) | 1.1ms | 935μs (15%) | 770μs (30%) |
| Parse (no refs) | Large (~6000 lines) | *not benchmarked* | *TBD* | *TBD* |
| Validate | Small (~60 lines) | 145μs | 130μs (10%) | 101μs (30%) |
| Validate | Medium (~570 lines) | 1.2ms | 1.0ms (15%) | 840μs (30%) |
| Convert OAS2→OAS3 | Small | 155μs | 132μs (15%) | 93μs (40%) |
| Convert OAS3→OAS2 | Small | 161μs | 137μs (15%) | 96μs (40%) |
| Marshal Info (no Extra) | - | 425ns | 382ns (10%) | 340ns (20%) |
| Marshal Info (with Extra) | - | 2.4μs | 2.1μs (12%) | 1.2μs (50%) |
| Join 2 docs | Small | 108μs | 97μs (10%) | 86μs (20%) |

**Platform**: Apple M4, darwin/arm64, Go 1.24
**Source**: benchmark-baseline.txt (commit 4a61c95)

### Memory Targets

**Baseline Allocations (actual)**:
- Parse small OAS3: 202KB, 2128 allocs
- Parse medium OAS3: 1.4MB, 17448 allocs
- Validate small OAS3: 208KB, 2220 allocs
- Validate medium OAS3: 1.5MB, 18369 allocs
- Marshal Info (no Extra): 192B, 2 allocs
- Marshal Info (with Extra): 1.8KB, 38 allocs

**Targets**:
- v1.7.0: Reduce allocations by 20% across all operations
- v1.8.0: Reduce allocations by 40% across all operations
- Measure GC pressure via `GODEBUG=gctrace=1`

### Benchmarking Standards

All benchmarks should:
- Run for sufficient iterations (`-benchtime=10s` minimum)
- Report allocations (`-benchmem`)
- Use realistic test data
- Be repeatable and deterministic
- Track both CPU and memory performance

---

## Open Questions

1. **What are typical document sizes in real-world usage?**
   - Need to survey users or analyze public OpenAPI specs
   - Affects which optimizations provide best ROI

2. **How often is ResolveRefs used?**
   - If rarely used, parser re-marshaling optimization is lower priority
   - Need telemetry or user survey

3. **What percentage of documents have Extra fields?**
   - Affects priority of marshaler optimization
   - Can analyze public OpenAPI specs for data

4. **Is oastools used more as CLI or library?**
   - CLI: less benefit from sync.Pool and caching
   - Library: high-throughput optimizations more valuable
   - Affects roadmap priorities

5. **What's the acceptable memory/speed tradeoff?**
   - User preference for speed vs memory efficiency
   - Affects caching strategy decisions

---

## Notes and Lessons Learned

*This section will be updated as work progresses*

### Session 1 (2025-11-17)
- Created comprehensive performance improvement plan
- Identified 7 major optimization strategies
- Prioritized benchmarking as prerequisite
- Defined phased roadmap with clear dependencies
- Documented decision criteria for strategy selection
- **Implemented full benchmark suite** (Phase 1 complete!)
  - 60+ benchmarks across parser, validator, converter, joiner
  - Using Go 1.24's testing.B.Loop() pattern
  - Created test fixtures: small (~60 lines), medium (~550 lines), large (~6000 lines)
  - Captured baseline metrics on Apple M4
- **Key Findings**:
  - JSON marshaling with Extra fields is 5.6x slower (425ns → 2.4μs)
  - Parse/validate operations scale linearly with document size
  - Memory allocations are primary cost driver (see actual metrics above)

**Next Session TODO**:
1. Begin Phase 2: Low-risk quick wins
2. Start with memory allocation optimization (Strategy 5A)
3. Implement validation early exits (Strategy 4B)
4. Document marshaler tradeoffs (Strategy 1C)