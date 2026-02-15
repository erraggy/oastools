# Benchmark Guide

## Benchmark Test Requirements

**CRITICAL: Use the Go 1.24+ `for b.Loop()` pattern for all benchmarks.**

### Correct Pattern

```go
func BenchmarkOperation(b *testing.B) {
    // Setup (parsing, creating instances, etc.)
    source, _ := parser.ParseWithOptions(
        parser.WithFilePath("file.yaml"),
        parser.WithValidateStructure(true),
    )

    for b.Loop() {  // ✅ Modern Go 1.24+ pattern
        _, err := Operation(source)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### DO NOT

- Use `for i := 0; i < b.N; i++` (old pattern)
- Call `b.ReportAllocs()` manually (handled by `b.Loop()`)
- Call `b.ResetTimer()` for trivial setup

## Benchmark Reliability

**File-based benchmarks can vary ±50% due to I/O.** Use `*Core` or `*Parsed` benchmarks for reliable regression detection:

| Package | Reliable Benchmark |
|---------|-------------------|
| parser | `BenchmarkParseCore` |
| joiner | `BenchmarkJoinParsed` |
| validator | `BenchmarkValidateParsed` |
| fixer | `BenchmarkFixParsed` |
| converter | `BenchmarkConvertParsed*` |
| differ | `BenchmarkDiff/Parsed` |
| walker | `BenchmarkWalk/Parsed` |

See [BENCHMARK_UPDATE_PROCESS.md](../../BENCHMARK_UPDATE_PROCESS.md) for detailed guidance.
