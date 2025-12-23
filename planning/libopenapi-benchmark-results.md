# Comparative Benchmark Results

**Date**: December 2025
**Platform**: Apple M4, darwin/arm64, Go 1.24
**oastools Version**: v1.30.1
**libopenapi Version**: v0.30.4
**libopenapi-validator Version**: v0.9.4

---

## Purpose

These benchmarks were conducted to understand performance characteristics and identify optimization opportunities for oastools. The results inform development priorities.

## Summary

| Operation | Observation | Notes |
|-----------|-------------|-------|
| **Parsing** | oastools faster in benchmarks | Consistent across spec sizes |
| **Validation** | Different approaches | oastools validates spec structure; libopenapi-validator validates data against schemas |
| **Diff** | oastools faster in benchmarks | Both produce comparable output |
| **Circular Refs** | Different handling | Stripe spec causes issues in libopenapi; oastools handles it |

---

## Test Specifications

| Spec | Size | OAS Version | Format | Description |
|------|------|-------------|--------|-------------|
| Petstore | 13.8 KB | 2.0 | JSON | Small, classic example API |
| Discord | 1.0 MB | 3.1.0 | JSON | Medium, OAS 3.1 with modern features |
| Stripe | 7.5 MB | 3.0.0 | JSON | Large, complex with circular references |

---

## Parsing Benchmarks

Both libraries parse documents from in-memory byte slices to eliminate I/O variance.

### Results

| Spec | oastools | libopenapi | Ratio | Notes |
|------|----------|------------|-------|-------|
| **Petstore** | 1.14 ms | 1.49 ms | 1.3x | OAS 2.0 |
| **Discord** | 52.6 ms | 94.5 ms | 1.8x | OAS 3.1 |
| **Stripe** | 286 ms | ❌ Failed | N/A | libopenapi circular ref error |

### Memory Allocation

| Spec | oastools | libopenapi | Ratio |
|------|----------|------------|-------|
| **Petstore** | 1.6 MB / 18K allocs | 2.2 MB / 36K allocs | 1.4x / 2.0x less |
| **Discord** | 53 MB / 621K allocs | 111 MB / 1.6M allocs | 2.1x / 2.7x less |
| **Stripe** | 228 MB / 3.3M allocs | ❌ Failed | N/A |

### Throughput (MB/s)

| Spec | oastools | libopenapi |
|------|----------|------------|
| **Petstore** | 12.1 MB/s | 9.3 MB/s |
| **Discord** | 19.5 MB/s | 10.8 MB/s |
| **Stripe** | 26.3 MB/s | N/A |

---

## Validation Benchmarks (Pre-parsed)

Tests validation of already-parsed documents, isolating validation logic from parsing.

### Results

| Spec | oastools | libopenapi | Ratio | Notes |
|------|----------|------------|-------|-------|
| **Petstore** | 37.3 µs | ❌ N/A | N/A | libopenapi validator requires OAS 3+ |
| **Discord** | 1.86 ms | 4,187 ms | 2,250x | Different validation scope (see analysis) |
| **Stripe** | 10.4 ms | ❌ Failed | N/A | libopenapi circular ref error |

### Analysis: Understanding the Difference

The performance gap stems from fundamentally different purposes:

**oastools validation** checks the OpenAPI specification structure:
- Validates structural correctness against OAS schema
- Checks required fields, formats, and references
- Designed for spec validation (is the YAML/JSON valid OpenAPI?)

**libopenapi-validator** validates data against schemas:
- Builds JSON Schema validators for every schema in the spec
- Creates deep validation chains for allOf/oneOf/anyOf
- Designed for HTTP traffic validation (does this request match the spec?)

These are complementary use cases. Spec validation (oastools) answers "is my OpenAPI file correct?" while data validation (libopenapi-validator) answers "does this HTTP request/response conform to the spec?"

### Memory Allocation (Validation)

| Spec | oastools | libopenapi |
|------|----------|------------|
| **Petstore** | 28 KB / 772 allocs | N/A |
| **Discord** | 1.8 MB / 38K allocs | 4.2 GB / 41M allocs |
| **Stripe** | 10.4 MB / 166K allocs | N/A |

---

## Diff Benchmarks (Pre-parsed)

Tests comparison of two already-parsed documents for breaking changes.

### Results

| Spec | oastools | libopenapi | Ratio |
|------|----------|------------|-------|
| **Discord** | 3.89 ms | 14.2 ms | 3.6x |
| **Stripe** | 23.3 ms | ❌ Failed | N/A |

### Memory Allocation (Diff)

| Spec | oastools | libopenapi |
|------|----------|------------|
| **Discord** | 7.1 MB / 92K allocs | 22.1 MB / 281K allocs |
| **Stripe** | 47.2 MB / 418K allocs | N/A |

---

## Memory Profile: Large Spec (Stripe)

End-to-end memory usage for parsing the 7.5MB Stripe specification:

| Metric | oastools | libopenapi | Ratio |
|--------|----------|------------|-------|
| **Parse Time** | 296 ms | 537 ms | 1.8x |
| **Memory Used** | 228 MB | 616 MB | 2.7x |
| **Allocations** | 3.25 M | 8.20 M | 2.5x |
| **Throughput** | 25.3 MB/s | 14.0 MB/s | 1.8x |

---

## Circular Reference Handling

The Stripe OpenAPI specification contains circular references. The libraries handle these differently:

**libopenapi** reports these as errors:
```
infinite circular reference detected: payment_intent:
  payment_intent -> api_errors -> payment_intent [2097:23]

infinite circular reference detected: transfer_reversal:
  payment_intent -> api_errors -> payment_method ->
  payment_method_sepa_debit -> sepa_debit_generated_from ->
  charge -> application_fee -> balance_transaction -> ... ->
  email_sent [58564:23]
```

**oastools** uses a different circular reference handling strategy that allows processing to complete. Both approaches have trade-offs: strict detection catches potential issues early, while permissive handling allows processing of specs with intentional circular references.

---

## Raw Benchmark Output

```
goos: darwin
goarch: arm64
pkg: libopenapi-comparison
cpu: Apple M4

# Parsing
BenchmarkParse_oastools_Petstore-10        1059    1135417 ns/op   12.19 MB/s   1637463 B/op   18289 allocs/op
BenchmarkParse_libopenapi_Petstore-10       801    1495334 ns/op    9.26 MB/s   2165886 B/op   36297 allocs/op
BenchmarkParse_oastools_Discord-10           22   52595739 ns/op   19.46 MB/s  53134258 B/op  621507 allocs/op
BenchmarkParse_libopenapi_Discord-10         12   94030656 ns/op   10.89 MB/s 110732661 B/op 1656081 allocs/op
BenchmarkParse_oastools_Stripe-10             4  285449927 ns/op   26.33 MB/s 227823516 B/op 3254220 allocs/op

# Validation (pre-parsed)
BenchmarkValidateParsed_oastools_Petstore-10   32101     37346 ns/op     28348 B/op      772 allocs/op
BenchmarkValidateParsed_oastools_Discord-10      639   1861025 ns/op   1803212 B/op    37965 allocs/op
BenchmarkValidateParsed_libopenapi_Discord-10      1 4177193667 ns/op 4150414656 B/op 41448640 allocs/op
BenchmarkValidateParsed_oastools_Stripe-10       100  10487046 ns/op  10388235 B/op   166248 allocs/op

# Diff (pre-parsed)
BenchmarkDiffParsed_oastools_Discord-10        306   3897102 ns/op   7052913 B/op    91748 allocs/op
BenchmarkDiffParsed_libopenapi_Discord-10       76  14223845 ns/op  22112695 B/op   281104 allocs/op
BenchmarkDiffParsed_oastools_Stripe-10          49  23576666 ns/op  47155403 B/op   418190 allocs/op

# Memory profile (Stripe)
BenchmarkMemory_oastools_Stripe-10              4  297854531 ns/op   25.23 MB/s 227822438 B/op 3254217 allocs/op
BenchmarkMemory_libopenapi_Stripe-10            2  535883771 ns/op   14.02 MB/s 615718376 B/op 8197274 allocs/op
```

---

## Observations

### Performance Characteristics

| Aspect | oastools | libopenapi |
|--------|----------|------------|
| **Parsing Speed** | 24-80% faster across test specs | Baseline |
| **Spec Validation** | Structural validation only | Full schema validation (slower) |
| **Memory Usage** | ~2.7x less for large specs | Higher due to validator caches |
| **Circular References** | Handles complex cases | Fails on some patterns |
| **Diff Speed** | ~3.6x faster | Baseline |

### Feature Differences

| Feature | oastools | libopenapi |
|---------|----------|------------|
| **HTTP Request/Response Validation** | Not implemented | Available via libopenapi-validator |
| **Lossless Parsing** | Not implemented | Available via GoLow() API |
| **Ecosystem Tools** | Standalone | Powers vacuum, openapi-changes, wiretap |

### Use Case Guidance

| Use Case | Considerations |
|----------|----------------|
| **Spec validation in CI/CD** | oastools validates structure faster; libopenapi-validator validates data against schemas |
| **Large specifications (>5MB)** | oastools handles circular refs that cause issues in libopenapi |
| **Breaking change detection** | Both libraries offer this; oastools is faster in benchmarks |
| **HTTP traffic validation** | Only libopenapi-validator supports this currently |
| **Building linters/IDE tooling** | libopenapi's GoLow() preserves source positions |

---

## Methodology

- All benchmarks run with `-benchmem -count=3`
- Specifications pre-loaded into memory to eliminate I/O variance
- Go 1.24+ `for b.Loop()` pattern for accurate iteration measurement
- Each benchmark isolated to measure only the target operation
- Platform: Apple M4, darwin/arm64

## Files

- Benchmark code: `planning/libopenapi-benchmark/benchmark_test.go`
- Raw results: `planning/libopenapi-benchmark/results.txt`
- Analysis: This document
