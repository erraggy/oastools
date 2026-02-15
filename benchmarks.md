# Performance Benchmarks

This document provides detailed performance analysis and benchmark results for the oastools library.

## Overview

As of v1.18.0, oastools includes comprehensive performance benchmarking infrastructure covering all major operations across the parser, validator, fixer, converter, joiner, differ, generator, builder, and overlay packages. The library has undergone targeted optimizations to achieve significant performance improvements while maintaining correctness and code quality.

**CI Platform (for cross-version comparisons)**: linux/amd64, AMD EPYC 7763, Go 1.24
**Local Platform (historical reference)**: darwin/arm64, Apple M4, Go 1.24

> **Note:** CI-generated benchmarks (`benchmarks/benchmark-v*.txt`) provide consistent, reproducible results for cross-version comparisons. Local benchmarks (`benchmarks/local/`) are preserved for historical reference but have ±50% I/O variance.

## ⚠️ Benchmark Reliability Notice

### I/O Variance Can Mask Reality

**IMPORTANT:** File-based benchmarks (e.g., `BenchmarkParse`, `BenchmarkJoin`) include file I/O in every iteration. This makes them **unreliable for detecting code-level performance regressions** because:

- Filesystem caching state varies between benchmark runs
- System load affects disk access times
- I/O variance can be **+/- 50%** or more

**Example:** An investigation in v1.28.1 found that comparing saved benchmarks showed apparent 51-82% regressions. However, running both versions live on the same machine showed **0% actual code performance difference**.

### Which Benchmarks to Trust

| Benchmark Type | Pattern | Use For | Reliable? |
|----------------|---------|---------|-----------|
| I/O-Isolated | `*Core`, `*Bytes` | Regression detection | ✅ Yes |
| Pre-Parsed | `*Parsed` | Regression detection | ✅ Yes |
| End-to-End | `BenchmarkParse`, `BenchmarkJoin`, etc. | User documentation | ⚠️ Informational only |

### Recommended Benchmarks for Regression Detection

```bash
# Parser: Use BenchmarkParseCore (not BenchmarkParse)
go test -bench=BenchmarkParseCore -benchmem ./parser

# Joiner: Use BenchmarkJoinParsed (not BenchmarkJoin)
go test -bench=BenchmarkJoinParsed -benchmem ./joiner

# All packages: Filter to reliable benchmarks only
go test -bench='ParseCore|JoinParsed|ValidateParsed|FixParsed|ConvertParsed|Diff/Parsed' -benchmem ./...
```

See [CLAUDE.md](https://github.com/erraggy/oastools/blob/main/CLAUDE.md#benchmark-reliability-and-performance-regression-detection) and [BENCHMARK_UPDATE_PROCESS.md](https://github.com/erraggy/oastools/blob/main/BENCHMARK_UPDATE_PROCESS.md#important-detecting-performance-regressions) for detailed guidance on investigating suspected regressions.

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

The benchmark suite includes **250+ benchmark runs** across ten packages:

### Parser Package (44 benchmarks)

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
- ParseWithOptions convenience API (file path, reader, reference resolution)
- ParseReader I/O performance
- ParseResult.Copy() deep copy performance
- Reference resolution overhead measurement
- FormatBytes utility function

**DeepCopy Operations**:

- OAS3Document.DeepCopy() for small, medium, and large documents
- OAS2Document.DeepCopy() for small and medium documents
- Type-aware copying that preserves nil vs empty distinction

**Unmarshaling Operations**:

- Info unmarshaling: no extra fields, with extra fields

**Order-Preserving Marshaling**:

- MarshalOrderedJSON: small, medium, and large OAS 3.x documents
- MarshalOrderedYAML: small, medium, and large OAS 3.x documents
- MarshalOrderedJSONIndent: indented JSON output
- Order-preserving vs standard marshaling comparison
- PreserveOrder parsing overhead measurement

### Validator Package (16 benchmarks)

**Validation Operations**:

- Small, medium, and large OAS 3.x documents
- Small and medium OAS 2.0 documents
- With and without warnings
- ValidateParsed (pre-parsed documents) vs Validate (parse + validate)
- ValidateWithOptions convenience API (basic and pre-parsed variants)
- Strict mode validation (small, medium, and large documents)
- Large document validation without warnings

### Fixer Package (13 benchmarks)

**Fixing Operations**:

- Small, medium, and large OAS 3.x documents
- Small and medium OAS 2.0 documents
- With and without type inference
- FixParsed (pre-parsed documents) vs Fix (parse + fix)
- FixWithOptions convenience API (basic and pre-parsed variants)

### Converter Package (13 benchmarks)

**Conversion Operations**:

- OAS 2.0 → OAS 3.x (small and medium documents)
- OAS 3.x → OAS 2.0 (small and medium documents)
- OAS 3.0 → OAS 3.1 version updates
- ConvertParsed (pre-parsed) vs Convert (parse + convert)
- ConvertWithOptions convenience API (basic and pre-parsed variants)
- Conversion with and without info messages

### Joiner Package (14 benchmarks)

**Joining Operations**:

- Join 2, 3, and 5 small documents
- JoinParsed (pre-parsed) vs Join (parse + join)
- JoinWithOptions convenience API (basic and pre-parsed variants)
- Different collision resolution strategies (AcceptLeft, AcceptRight)
- Array merge strategies
- Tag deduplication
- WriteResult I/O performance
- Configuration utilities (DefaultConfig, IsValidStrategy, ValidStrategies)

### Differ Package (10 benchmarks)

**Diffing Operations**:

- Diff (parse + diff) vs DiffParsed (pre-parsed)
- DiffWithOptions convenience API
- Simple mode vs Breaking mode
- Configuration options (IncludeInfo)
- Edge cases (identical specifications)
- Parse-once pattern efficiency
- Change.String() formatting performance

### Generator Package (4 benchmarks)

**Code Generation Operations**:

- Types-only generation
- Client code generation
- Server code generation
- Full generation (types + client + server)

### Overlay Package (18 benchmarks)

**Overlay Application Operations**:

- Apply overlays to small, medium, and large documents
- Apply with in-memory overlays vs file-based
- ApplyParsed performance (pre-parsed documents)
- ApplyWithOptions functional API
- DryRun preview functionality

**JSONPath Operations**:

- Parse expressions of varying complexity
- Get operations with wildcards, filters, recursive descent
- Modify operations for document transformation
- Compound filter evaluation (&&, ||)

### Builder Package (17 benchmarks)

**Builder Construction Operations**:

- Builder instantiation (New)
- Info configuration (SetTitle, SetVersion, SetDescription)
- Schema generation from reflection (primitives, structs, nested structs, slices, maps)
- Operation building (simple operations, with parameters, with request bodies)
- Full document building (Build)
- Serialization (MarshalYAML, MarshalJSON)
- OAS tag parsing and application

### HTTP Validator Package (9 benchmarks)

**Request/Response Validation**:

- Request validation (path, query, header, body)
- Response validation (status code, headers, body)
- Path matching performance (single path, multiple paths)
- Schema validation for request/response bodies
- Parameter deserialization benchmarks
- Strict mode validation overhead
- Pre-parsed validator reuse performance

## Current Performance Metrics

### Parser Performance

**Document Parsing** (with validation, no ref resolution):

| Document Size | Lines | Time (μs) | Memory (KB) | Allocations |
|---------------|-------|-----------|-------------|-------------|
| Small OAS3    | ~60   | 138       | 197         | 2,070       |
| Medium OAS3   | ~570  | 1,119     | 1,447       | 17,389      |
| Large OAS3    | ~6000 | 13,880    | 16,425      | 194,712     |
| Small OAS2    | ~60   | 134       | 174         | 2,059       |
| Medium OAS2   | ~570  | 1,016     | 1,230       | 16,027      |

**Parsing without validation** provides ~3-5% improvement over validated parsing.

**JSON Marshaling** (post-optimization):

| Type         | Extra Fields | Time (ns) | Memory (bytes) | Allocations |
|--------------|--------------|-----------|----------------|-------------|
| Info         | None         | 432       | 192            | 2           |
| Info         | 5 fields     | 1,762     | 1,705          | 26          |
| Contact      | None         | 449       | 192            | 2           |
| Contact      | With extras  | 1,686     | 1,377          | 24          |
| Server       | None         | 371       | 160            | 2           |
| Server       | With extras  | 2,275     | 2,010          | 29          |
| OAS3Document | Small        | 19,891    | 7,003          | 66          |
| OAS3Document | Medium       | 221,137   | 65,753         | 471         |
| OAS3Document | Large        | 2,724,839 | 840,948        | 5,336       |

**Observation**: Marshaling performance scales linearly with document size and extra field count. The fast path (no extra fields) has minimal overhead.

**DeepCopy Performance** (code-generated, type-aware copying):

| Document Size | Time (ns) | Memory (KB) | Allocations |
|---------------|-----------|-------------|-------------|
| Small OAS3    | 1,769     | 7.2         | 44          |
| Medium OAS3   | 24,596    | 106.01      | 466         |
| Large OAS3    | 342,732   | 1,136       | 4,877       |
| Small OAS2    | 1,499     | 6.2         | 38          |
| Medium OAS2   | 20,699    | 92.5        | 387         |

**Observation**: DeepCopy uses code-generated methods that are type-aware, preserving nil vs empty distinction and handling OAS-specific polymorphic fields (Schema.Type, AdditionalProperties, etc.) correctly. This replaces the previous JSON marshal/unmarshal approach which was ~20-40x slower and corrupted `$ref` parameters.

**Order-Preserving Marshaling** (pre-parsed with PreserveOrder=true):

| Format | Document Size | Time | Memory | Allocations |
|--------|---------------|------|--------|-------------|
| JSON   | Small OAS3    | 9.0 us | 4.7 KB | 160 |
| JSON   | Medium OAS3   | 98.8 us | 63.4 KB | 1,532 |
| JSON   | Large OAS3    | 1.08 ms | 631 KB | 17,433 |
| YAML   | Small OAS3    | 46.1 us | 187 KB | 390 |
| YAML   | Medium OAS3   | 544 us | 1.79 MB | 3,495 |
| YAML   | Large OAS3    | 5.11 ms | 22.1 MB | 38,850 |
| JSON Indent | Medium OAS3 | 148 us | 148 KB | 1,535 |

**Order-Preserving vs Standard Marshaling** (Medium OAS3):

| Method | Time | Memory | Allocations |
|--------|------|--------|-------------|
| Order-Preserving JSON | 99.5 us | 63.4 KB | 1,532 |
| Standard JSON | 222 us | 65.9 KB | 471 |

**PreserveOrder Parsing Overhead** (Medium OAS3):

| Mode | Time | Memory | Allocations |
|------|------|--------|-------------|
| With PreserveOrder | 1.50 ms | 1.71 MB | 22,119 |
| Without PreserveOrder | 1.10 ms | 1.45 MB | 17,389 |

**Observation**: Order-preserving JSON marshaling is **2.2x faster** than standard JSON marshaling (99.5us vs 222us) because it writes directly to a buffer using the source node order, avoiding the overhead of reflection-based encoding. YAML marshaling requires building an intermediate node tree, making it ~5.5x slower than JSON. Enabling PreserveOrder during parsing adds ~37% overhead (storing the yaml.Node tree) but this is amortized when the document is marshaled multiple times with consistent field ordering.

### Validator Performance

**Validation** (with warnings):

| Document Size | Lines | Time (μs) | Memory (KB) | Allocations |
|---------------|-------|-----------|-------------|-------------|
| Small OAS3    | ~60   | 139       | 204         | 2,162       |
| Medium OAS3   | ~570  | 1,133     | 1,492       | 18,307      |
| Large OAS3    | ~6000 | 14,409    | 16,844      | 205,022     |
| Small OAS2    | ~60   | 134       | 181         | 2,135       |
| Medium OAS2   | ~570  | 1,058     | 1,268       | 16,851      |

**Validating pre-parsed documents** (ValidateParsed):

| Document Size | Time (μs) | Memory (KB) | Allocations |
|---------------|-----------|-------------|-------------|
| Small OAS3    | 4.7       | 5.1         | 90          |
| Medium OAS3   | 40.4      | 32.6        | 914         |
| Large OAS3    | 462       | 367         | 10,297      |

**Observation**: ValidateParsed is **31x faster** than Validate for small documents (4.7μs vs 147μs) because it skips parsing. This is ideal for workflows where documents are parsed once and validated multiple times.

### Fixer Performance

**Fixing** (parse + fix):

| Document Size | Lines | Time (μs) | Memory (KB) | Allocations |
|---------------|-------|-----------|-------------|-------------|
| Small OAS3    | ~60   | 220       | 279         | 3,252       |
| Medium OAS3   | ~570  | 2,034     | 2,208       | 28,422      |
| Large OAS3    | ~6000 | 24,957    | 25,028      | 320,120     |
| Small OAS2    | ~60   | 209       | 239         | 3,100       |
| Medium OAS2   | ~570  | 1,733     | 1,797       | 24,946      |

**Fixing pre-parsed documents** (FixParsed):

| Document Size | Time (μs) | Memory (KB) | Allocations |
|---------------|-----------|-------------|-------------|
| Small OAS3    | 2.4       | 8.17        | 52          |
| Medium OAS3   | 28.6      | 115.1       | 530         |
| Large OAS3    | 354       | 1,161       | 5,021       |

**Observation**: FixParsed is **60x faster** than Fix for small documents (2.3μs vs 141μs) because it skips parsing. The v1.20.0 release dramatically improved FixParsed performance by replacing JSON marshal/unmarshal with code-generated DeepCopy methods, resulting in ~37x faster deep copying operations. Type inference has negligible overhead (~3% slower). The fixer is I/O and parse-bound for most workflows.

### Converter Performance

**Conversion** (parse + convert):

| Conversion    | Document Size | Time (μs) | Memory (KB) | Allocations |
|---------------|---------------|-----------|-------------|-------------|
| OAS2 → OAS3   | Small         | 152       | 191         | 2,359       |
| OAS2 → OAS3   | Medium        | 1,258     | 1,461       | 19,640      |
| OAS3 → OAS2   | Small         | 160       | 216         | 2,390       |
| OAS3 → OAS2   | Medium        | 1,438     | 1,696       | 21,369      |

**Converting pre-parsed documents** (ConvertParsed):

| Conversion    | Document Size | Time (μs) | Memory (KB) | Allocations |
|---------------|---------------|-----------|-------------|-------------|
| OAS2 → OAS3   | Small         | 3.2       | 10.0        | 83          |
| OAS2 → OAS3   | Medium        | 37.9      | 119         | 687         |
| OAS3 → OAS2   | Small         | 3.6       | 8.4         | 89          |
| OAS3 → OAS2   | Medium        | 44.8      | 114         | 821         |

**Observation**: ConvertParsed is **42x faster** than Convert for small documents (3.2μs vs 135μs) because it skips parsing. The v1.20.0 release improved ConvertParsed by ~5x through code-generated DeepCopy methods replacing JSON marshal/unmarshal. Conversion overhead is minimal compared to parsing.

### Joiner Performance

**Joining** (parse + join):

| Documents | Time (μs) | Memory (KB) | Allocations |
|-----------|-----------|-------------|-------------|
| 2 small   | 110       | 141         | 1,602       |
| 3 small   | 162       | 210         | 2,363       |

**Joining pre-parsed documents** (JoinParsed):

| Documents | Time (ns) | Memory (bytes) | Allocations |
|-----------|-----------|----------------|-------------|
| 2 small   | 732       | 1,816          | 22          |
| 3 small   | 934       | 1,960          | 23          |

**Observation**: JoinParsed is **150x faster** than Join for 2 small documents (732ns vs 110μs) because it skips parsing. The actual joining operation has minimal overhead.

### Differ Performance

**Diffing** (parse + diff):

| Mode     | Time (μs) | Memory (KB) | Allocations |
|----------|-----------|-------------|-------------|
| Simple   | 463       | 594         | 7,182       |
| Breaking | 467       | 597         | 7,183       |

**Diffing pre-parsed documents** (DiffParsed):

| Mode     | Time (μs) | Memory (KB) | Allocations |
|----------|-----------|-------------|-------------|
| Simple   | 5.7       | 7.6         | 162         |
| Breaking | 6.9       | 7.9         | 177         |

**Special cases**:

| Case               | Time (μs) | Memory (KB) | Allocations |
|--------------------|-----------|-------------|-------------|
| Identical specs    | 3.8       | 3.2         | 115         |
| With info changes  | 6.8       | 7.9         | 177         |
| Without info       | 6.9       | 9.0         | 178         |

**Observation**: DiffParsed is **81x faster** than Diff (5.7μs vs 463μs) because it skips parsing. The differ includes a fast path for identical specifications that runs in ~3.8μs. Breaking mode vs Simple mode has negligible performance difference (~1.2μs), making breaking change detection essentially free.

### Generator Performance

**Code Generation** (pre-parsed documents):

| Generation Mode | Time (μs) | Memory (KB) | Allocations |
|-----------------|-----------|-------------|-------------|
| Types only      | 39        | 28          | 724         |
| Client          | 272       | 187         | 4,088       |
| Server          | 57        | 48          | 1,040       |
| All (full)      | 249       | 182         | 3,882       |

**Observation**: Types-only generation is fastest at 39μs. Client generation is most expensive due to HTTP client method generation. Full generation (all modes) is comparable to client-only because client code dominates the generation time.

### Builder Performance

**Builder Construction**:

| Operation                | Time (ns) | Memory (bytes) | Allocations |
|--------------------------|-----------|----------------|-------------|
| New                      | 203       | 736            | 13          |
| SetInfo (fluent chain)   | 221       | 848            | 14          |

**Schema Generation** (reflection-based):

| Type             | Time (ns) | Memory (bytes) | Allocations |
|------------------|-----------|----------------|-------------|
| Primitive        | 166       | 768            | 1           |
| Struct           | 3,229     | 15,280         | 75          |
| Nested struct    | 4,684     | 22,960         | 95          |
| Slice            | 3,356     | 16,048         | 76          |
| Map              | 3,389     | 16,048         | 76          |

**Operation Building**:

| Operation Type          | Time (ns) | Memory (bytes) | Allocations |
|-------------------------|-----------|----------------|-------------|
| Simple operation        | 4,087     | 18,633         | 99          |
| With parameters         | 5,741     | 26,925         | 140         |
| With request body       | 6,835     | 31,504         | 151         |

**Document Building**:

| Operation        | Time (ns) | Memory (bytes) | Allocations |
|------------------|-----------|----------------|-------------|
| Build (3 ops)    | 8,013     | 35,490         | 211         |
| MarshalYAML      | 32,841    | 93,870         | 482         |
| MarshalJSON      | 18,704    | 8,429          | 38          |

**Tag Processing**:

| Operation      | Time (ns) | Memory (bytes) | Allocations |
|----------------|-----------|----------------|-------------|
| Parse OAS tag  | 181       | 432            | 3           |
| Apply OAS tag  | 267       | 1,184          | 6           |

**Observation**: Builder provides efficient programmatic construction of OAS documents. Schema generation from reflection is ~3-5μs for typical structs, making it suitable for runtime use. JSON marshaling is ~2x faster than YAML marshaling (18.7μs vs 32.8μs). Tag parsing is highly optimized at ~181ns per tag.

### HTTP Validator Performance

**Request Validation (Pre-parsed)**:

| Document Size | Time (ns) | Memory (bytes) | Allocations |
|---------------|-----------|----------------|-------------|
| Small (3 paths) | 210 | 528 | 8 |
| Medium (20 paths) | 268 | 528 | 8 |
| Large (100+ paths) | 502 | 528 | 8 |

**Validation Components**:

| Component | Time (ns) | Memory (bytes) | Allocations |
|-----------|-----------|----------------|-------------|
| Path matching only | 210-1017 | 528 | 8 |
| With parameters | 212 | 528 | 8 |
| With JSON body | 455 | 1,521 | 16 |
| Response validation | 110 | 256 | 2 |
| Strict mode | 212 | 528 | 8 |

**API Comparison**:

| API | Time (ns) | Memory (bytes) | Allocations |
|-----|-----------|----------------|-------------|
| ValidateRequestWithOptions (file path) | 217,277 | 207,185 | 2,203 |
| ValidateRequestWithOptions (parsed) | 3,250 | 9,153 | 127 |
| Validator.ValidateRequest | 210 | 528 | 8 |

**Observation**: HTTP validation is extremely fast when using a pre-created Validator instance (~210ns per request). The functional options API with file path includes parsing overhead (217μs), while using pre-parsed results is ~67x faster (3.2μs). Path matching scales sub-linearly with path count. Response validation is even faster at ~110ns. This makes the httpvalidator package suitable for high-throughput API gateway use cases.

## Performance Best Practices

### For Library Users

1. **Reuse parser/validator/fixer/converter/joiner/differ/httpvalidator instances** when processing multiple files with the same configuration
2. **Use ParseOnce workflows**: For operations on the same document (validate, fix, convert, join, diff), parse once and pass the `ParseResult` to subsequent operations:

   ```go
   result, _ := parser.Parse("spec.yaml", false, true)
   validator.ValidateParsed(result, true, false)   // 30x faster than Validate
   fixer.FixParsed(result)                         // 2.6x faster than Fix
   converter.ConvertParsed(result, "3.0.3")        // 9x faster than Convert
   differ.DiffParsed(result1, result2)             // 81x faster than Diff
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
make bench-fixer
make bench-converter
make bench-joiner
make bench-differ
make bench-generator
make bench-builder

# Or individually
go test -bench=. -benchmem ./parser
go test -bench=. -benchmem ./validator
go test -bench=. -benchmem ./fixer
go test -bench=. -benchmem ./converter
go test -bench=. -benchmem ./joiner
go test -bench=. -benchmem ./differ
go test -bench=. -benchmem ./generator
go test -bench=. -benchmem ./builder

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

### v1.29.0 - OAS 3.2.0 and JSON Schema 2020-12 Support

**Parser Performance** (I/O-Isolated):

| Benchmark | Time | Memory | Allocations |
|-----------|------|--------|-------------|
| ParseCore/SmallOAS3 | 134 µs | 196 KB | 2,065 |
| ParseCore/MediumOAS3 | 1.18 ms | 1.4 MB | 17,384 |
| ParseCore/LargeOAS3 | 14.0 ms | 16.4 MB | 194,707 |

**Pre-Parsed Benchmarks** (Core Operations):

| Package | Benchmark | Time | Memory | Allocations |
|---------|-----------|------|--------|-------------|
| Validator | ValidateParsed | 4.8 µs | 5.8 KB | 90 |
| Fixer | FixParsed | 2.8 µs | 9.1 KB | 54 |
| Converter | ConvertParsed | 3.3 µs | 11.1 KB | 83 |
| Joiner | JoinParsed/TwoDocs | 803 ns | 1.9 KB | 22 |
| Differ | Diff/Parsed | 12.7 µs | 21.7 KB | 369 |

**Changes from v1.28.4**: Added support for OAS 3.2.0 fields (`$self`, `Query`, `additionalOperations`, `mediaTypes`) and JSON Schema 2020-12 keywords (`unevaluatedProperties`, `unevaluatedItems`, `contentEncoding`, `contentMediaType`, `contentSchema`). Performance impact: ~0% (new fields are only processed when present).

### Baseline (Pre-Optimization)

**JSON Marshaling** (before v1.7.0):

| Type    | Time (ns) | Allocations |
|---------|-----------|-------------|
| Info    | 2,323     | 38          |
| Contact | 2,336     | 38          |
| Server  | 2,837     | 41          |

**Post-v1.7.0 Improvement**: 25-32% faster, 29-37% fewer allocations

## Related Documentation

- [CLAUDE.md](https://github.com/erraggy/oastools/blob/main/CLAUDE.md) - Development guidelines and architecture
- [BENCHMARK_UPDATE_PROCESS.md](https://github.com/erraggy/oastools/blob/main/BENCHMARK_UPDATE_PROCESS.md) - How to update benchmarks for releases

## Questions and Feedback

If you have specific performance requirements or encounter performance issues in your use case, please open an issue on GitHub. Real-world performance feedback helps prioritize future optimization efforts.
