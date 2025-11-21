# Performance Benchmarks

This document provides detailed performance analysis and benchmark results for the oastools library.

## Overview

As of v1.9.1, oastools includes comprehensive performance benchmarking infrastructure covering all major operations across the parser, validator, converter, joiner, and differ packages. The library has undergone targeted optimizations to achieve significant performance improvements while maintaining correctness and code quality.

**Platform**: Apple M4, darwin/arm64, Go 1.24

## Key Performance Achievements

### Phase 2 Optimization: JSON Marshaling (v1.7.0)

The v1.7.0 release includes a major optimization to JSON marshaling that eliminates the double-marshal pattern across all 29 custom JSON marshalers in the parser package.

**Performance Improvements**:
- **25-32% faster** JSON marshaling for types with specification extensions (Extra fields)
- **29-37% fewer** memory allocations
- **Zero overhead** for types without Extra fields (fast path optimization)

**Implementation Strategy**:
- Eliminated marshal→unmarshal→marshal pattern (4 operations → 2 operations)
- Direct map building approach for types with Extra fields
- Replaced knownFields map lookups with efficient prefix checking (`x-`)
- Early return optimization when no Extra fields present

**Benchmark Results**:

| Type    | Before (baseline) | After (optimized) | Time Improvement | Alloc Improvement |
|---------|-------------------|-------------------|------------------|-------------------|
| Info    | 2,323 ns/op       | 1,707 ns/op       | **26% faster**   | **32% fewer**     |
| Contact | 2,336 ns/op       | 1,599 ns/op       | **32% faster**   | **37% fewer**     |
| Server  | 2,837 ns/op       | 2,160 ns/op       | **25% faster**   | **29% fewer**     |

## Benchmark Suite

The benchmark suite includes **70+ benchmarks** across five main packages:

### Parser Package (32 benchmarks)

**Marshaling Operations**:
- Info marshaling: no extra fields, with extra fields, varying extra field counts (1, 5, 10, 20)
- Contact marshaling: no extra fields, with extra fields
- Server marshaling: no extra fields, with extra fields
- Document marshaling: small, medium, and large OAS 2.0 and OAS 3.x documents

**Parsing Operations**:
- Small, medium, and large OAS 3.x documents
- Small and medium OAS 2.0 documents
- Parsing with and without validation
- ParseBytes vs Parse (file-based)
- Convenience function performance

**Unmarshaling Operations**:
- Info unmarshaling: no extra fields, with extra fields

### Validator Package (12 benchmarks)

**Validation Operations**:
- Small, medium, and large OAS 3.x documents
- Small and medium OAS 2.0 documents
- With and without warnings
- ValidateParsed (pre-parsed documents) vs Validate (parse + validate)
- Convenience function performance
- Strict mode validation

### Converter Package (10 benchmarks)

**Conversion Operations**:
- OAS 2.0 → OAS 3.x (small and medium documents)
- OAS 3.x → OAS 2.0 (small and medium documents)
- ConvertParsed (pre-parsed) vs Convert (parse + convert)
- Convenience function performance
- Conversion with and without info messages

### Joiner Package (8 benchmarks)

**Joining Operations**:
- Join 2 and 3 small documents
- JoinParsed (pre-parsed) vs Join (parse + join)
- Convenience function performance
- Different collision resolution strategies (AcceptLeft, AcceptRight)
- Array merge strategies
- Tag deduplication

### Differ Package (10 benchmarks)

**Diffing Operations**:
- Diff (parse + diff) vs DiffParsed (pre-parsed)
- Simple mode vs Breaking mode
- Convenience functions (Diff, DiffParsed) vs struct-based API
- Configuration options (IncludeInfo)
- Edge cases (identical specifications)
- Parse-once pattern efficiency

## Current Performance Metrics

### Parser Performance

**Document Parsing** (with validation, no ref resolution):

| Document Size | Lines | Time (μs) | Memory (KB) | Allocations |
|---------------|-------|-----------|-------------|-------------|
| Small OAS3    | ~60   | 142       | 203         | 2,128       |
| Medium OAS3   | ~570  | 1,130     | 1,465       | 17,449      |
| Large OAS3    | ~6000 | 14,131    | 16,468      | 194,777     |
| Small OAS2    | ~60   | 134       | 174         | 2,059       |
| Medium OAS2   | ~570  | 1,024     | 1,230       | 16,027      |

**Parsing without validation** provides ~3-5% improvement over validated parsing.

**JSON Marshaling** (post-optimization):

| Type         | Extra Fields | Time (ns) | Memory (bytes) | Allocations |
|--------------|--------------|-----------|----------------|-------------|
| Info         | None         | 421       | 192            | 2           |
| Info         | 5 fields     | 1,717     | 1,737          | 26          |
| Contact      | None         | 447       | 192            | 2           |
| Contact      | With extras  | 1,692     | 1,377          | 24          |
| Server       | None         | 373       | 160            | 2           |
| Server       | With extras  | 2,205     | 2,010          | 29          |
| OAS3Document | Small        | 19,750    | 7,002          | 66          |
| OAS3Document | Medium       | 220,733   | 65,744         | 471         |
| OAS3Document | Large        | 2,720,263 | 844,323        | 5,336       |

**Observation**: Marshaling performance scales linearly with document size and extra field count. The fast path (no extra fields) has minimal overhead.

### Validator Performance

**Validation** (with warnings):

| Document Size | Lines | Time (μs) | Memory (KB) | Allocations |
|---------------|-------|-----------|-------------|-------------|
| Small OAS3    | ~60   | 143       | 208         | 2,218       |
| Medium OAS3   | ~570  | 1,160     | 1,494       | 18,363      |
| Large OAS3    | ~6000 | 14,635    | 16,851      | 205,078     |
| Small OAS2    | ~60   | 137       | 180         | 2,133       |
| Medium OAS2   | ~570  | 1,042     | 1,268       | 16,849      |

**Validating pre-parsed documents** (ValidateParsed):

| Document Size | Time (μs) | Memory (KB) | Allocations |
|---------------|-----------|-------------|-------------|
| Small OAS3    | 4.7       | 5.2         | 90          |
| Medium OAS3   | 40.2      | 33.4        | 914         |
| Large OAS3    | 458       | 376         | 10,297      |

**Observation**: ValidateParsed is **30x faster** than Validate for small documents (4.7μs vs 143μs) because it skips parsing. This is ideal for workflows where documents are parsed once and validated multiple times.

### Converter Performance

**Conversion** (parse + convert):

| Conversion    | Document Size | Time (μs) | Memory (KB) | Allocations |
|---------------|---------------|-----------|-------------|-------------|
| OAS2 → OAS3   | Small         | 153       | 195         | 2,357       |
| OAS2 → OAS3   | Medium        | 1,314     | 1,496       | 19,638      |
| OAS3 → OAS2   | Small         | 168       | 221         | 2,388       |
| OAS3 → OAS2   | Medium        | 1,540     | 1,741       | 21,370      |

**Converting pre-parsed documents** (ConvertParsed):

| Conversion    | Document Size | Time (μs) | Memory (KB) | Allocations |
|---------------|---------------|-----------|-------------|-------------|
| OAS2 → OAS3   | Small         | 17.3      | 21.1        | 297         |
| OAS2 → OAS3   | Medium        | 270       | 265         | 3,608       |
| OAS3 → OAS2   | Small         | 14.4      | 17.5        | 258         |
| OAS3 → OAS2   | Medium        | 292       | 269         | 3,910       |

**Observation**: ConvertParsed is **9-10x faster** than Convert for small documents because it skips parsing. Conversion overhead is minimal compared to parsing.

### Joiner Performance

**Joining** (parse + join):

| Documents | Time (μs) | Memory (KB) | Allocations |
|-----------|-----------|-------------|-------------|
| 2 small   | 115       | 144         | 1,602       |
| 3 small   | 171       | 215         | 2,363       |

**Joining pre-parsed documents** (JoinParsed):

| Documents | Time (ns) | Memory (bytes) | Allocations |
|-----------|-----------|----------------|-------------|
| 2 small   | 747       | 1,784          | 22          |
| 3 small   | 953       | 1,912          | 23          |

**Observation**: JoinParsed is **154x faster** than Join for 2 small documents (747ns vs 115μs) because it skips parsing. The actual joining operation has minimal overhead.

### Differ Performance

**Diffing** (parse + diff):

| Mode     | Time (μs) | Memory (KB) | Allocations |
|----------|-----------|-------------|-------------|
| Simple   | 457       | 609         | 7,156       |
| Breaking | 458       | 611         | 7,157       |

**Diffing pre-parsed documents** (DiffParsed):

| Mode     | Time (μs) | Memory (KB) | Allocations |
|----------|-----------|-------------|-------------|
| Simple   | 7.82      | 6.89        | 138         |
| Breaking | 7.97      | 6.96        | 137         |

**Special cases**:

| Case               | Time (μs) | Memory (KB) | Allocations |
|--------------------|-----------|-------------|-------------|
| Identical specs    | 2.82      | 2.08        | 79          |
| With info changes  | 7.97      | 6.96        | 137         |
| Without info       | 8.10      | 8.11        | 138         |

**Observation**: DiffParsed is **58x faster** than Diff (7.8μs vs 457μs) because it skips parsing. The differ includes a fast path for identical specifications that runs in ~2.8μs. Breaking mode vs Simple mode has negligible performance difference (~0.15μs), making breaking change detection essentially free.

## Performance Best Practices

### For Library Users

1. **Reuse parser/validator/converter/joiner/differ instances** when processing multiple files with the same configuration
2. **Use ParseOnce workflows**: For operations on the same document (validate, convert, join, diff), parse once and pass the `ParseResult` to subsequent operations:
   ```go
   result, _ := parser.Parse("spec.yaml", false, true)
   validator.ValidateParsed(result, true, false)   // 30x faster than Validate
   converter.ConvertParsed(result, "3.0.3")        // 9x faster than Convert
   differ.DiffParsed(result1, result2)             // 58x faster than Diff
   ```
3. **Disable reference resolution** (`ResolveRefs=false`) when not needed
4. **Disable validation** during parsing (`ValidateStructure=false`) if you'll validate separately
5. **Minimize specification extensions**: Documents with many Extra fields (`x-*`) will be slower to marshal

### For High-Throughput Scenarios

- Use the struct-based API (e.g., `parser.New()`) instead of convenience functions for reusable instances
- Consider parallel processing of independent documents (oastools is safe for concurrent use)
- Profile your specific workload to identify bottlenecks

## Benchmark Methodology

All benchmarks follow these standards:
- Run with `-benchmem` to track memory allocations
- Use realistic test data from `testdata/bench/`
- Deterministic and repeatable
- Measure both CPU time and memory performance

**Test Data**:
- Small documents: ~60 lines
- Medium documents: ~570 lines
- Large documents: ~6000 lines

**Running Benchmarks**:

```bash
# Run all benchmarks
make bench-parser
make bench-validator
make bench-converter
make bench-joiner
make bench-differ

# Or individually
go test -bench=. -benchmem ./parser
go test -bench=. -benchmem ./validator
go test -bench=. -benchmem ./converter
go test -bench=. -benchmem ./joiner
go test -bench=. -benchmem ./differ

# Save baseline for comparison
go test -bench=. -benchmem ./... > benchmark-baseline.txt

# Compare before/after
go test -bench=. -benchmem ./... > benchmark-new.txt
benchstat benchmark-baseline.txt benchmark-new.txt
```

## Future Optimization Opportunities

Based on profiling and analysis, several low-risk optimization opportunities remain available for future releases:

### Available Low-Risk Optimizations

1. **Memory pre-allocation** (5-15% improvement)
   - Pre-allocate slices with capacity hints
   - Reduces GC pressure in high-throughput scenarios

2. **Validation early exits** (10-20% improvement for invalid documents)
   - Return early when possible for malformed inputs
   - Fast-fail scenarios

3. **Reference resolution two-pass** (20-30% improvement for docs without refs)
   - Skip resolution when no `$ref` fields present
   - Benefits simple documents

### Available Medium-Risk Optimizations

1. **Validation caching** (20-40% improvement for docs with repeated schemas)
   - Cache validation results by schema hash
   - Helps documents with many references to the same schema

2. **Parallel validation** (30-50% improvement for large documents)
   - Validate independent paths/schemas concurrently
   - Utilize multiple CPU cores

3. **Parallel reference resolution** (30-50% improvement for multi-ref docs)
   - Resolve independent references concurrently
   - Benefits documents with many external references

These optimizations are **on hold** pending real-world performance feedback. The current performance is sufficient for v1.7.0, and future optimization efforts will be prioritized based on demonstrated user needs rather than speculation.

## Historical Performance

### Baseline (Pre-Optimization)

**JSON Marshaling** (before v1.7.0):

| Type    | Time (ns) | Allocations |
|---------|-----------|-------------|
| Info    | 2,323     | 38          |
| Contact | 2,336     | 38          |
| Server  | 2,837     | 41          |

**Post-v1.7.0 Improvement**: 25-32% faster, 29-37% fewer allocations

## Related Documentation

- [Performance Planning](planning/improve-performance.md) - Detailed optimization strategy and analysis
- [CLAUDE.md](CLAUDE.md) - Development guidelines and architecture
- [README.md](README.md) - Project overview and usage

## Questions and Feedback

If you have specific performance requirements or encounter performance issues in your use case, please open an issue on GitHub. Real-world performance feedback helps prioritize future optimization efforts.
