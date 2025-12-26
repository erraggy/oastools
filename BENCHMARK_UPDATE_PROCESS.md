# Benchmark Update Process

This document describes the process for updating benchmark results after making changes to the codebase. Follow this process before each release to ensure performance metrics are current.

## IMPORTANT: Detecting Performance Regressions

> **TL;DR:** File-based benchmarks can vary +/- 50% due to I/O. Use `*Core` or `*Parsed` benchmarks for reliable regression detection.

### The Problem

Saved benchmark files capture I/O conditions at the time of recording. Comparing saved benchmarks across versions can show **phantom regressions of 50%+** that are actually just I/O variance, not real code changes.

**Example from v1.28.1 investigation:**
| Benchmark | Saved v1.25.0 | Saved v1.28.1 | Live Comparison |
|-----------|---------------|---------------|-----------------|
| Parse/SmallOAS3 | 143 µs | 217 µs (+51%) | **0% actual change** |
| Join/TwoDocs | 103 µs | 188 µs (+82%) | **0% actual change** |

### The Solution: Use I/O-Isolated Benchmarks

When investigating a suspected regression, **always use these benchmarks**:

```bash
# Run ONLY reliable benchmarks (no file I/O in measurement loop)
go test -bench='ParseCore|JoinParsed|ValidateParsed|FixParsed|ConvertParsed|Diff/Parsed' -benchmem ./...
```

| Package | Recommended Benchmark | Reliable? |
|---------|----------------------|-----------|
| parser | `BenchmarkParseCore` | ✅ Yes |
| joiner | `BenchmarkJoinParsed` | ✅ Yes |
| validator | `BenchmarkValidateParsed` | ✅ Yes |
| fixer | `BenchmarkFixParsed` | ✅ Yes |
| converter | `BenchmarkConvertParsed*` | ✅ Yes |
| differ | `BenchmarkDiff/Parsed` | ✅ Yes |
| (all) | `BenchmarkParse`, `BenchmarkJoin`, etc. | ❌ No (I/O variance) |

### Quick Regression Check Workflow

```bash
# 1. Checkout the suspected "slow" version
git checkout v1.X.Y
go test -bench='ParseCore|JoinParsed' -benchmem ./parser ./joiner > /tmp/old.txt

# 2. Checkout the current version
git checkout main
go test -bench='ParseCore|JoinParsed' -benchmem ./parser ./joiner > /tmp/new.txt

# 3. Compare with benchstat
benchstat /tmp/old.txt /tmp/new.txt
```

**If benchstat shows no significant change, there is no regression—regardless of what saved benchmark files show.**

See [CLAUDE.md](CLAUDE.md#benchmark-reliability-and-performance-regression-detection) for detailed guidance.

---

## When to Update Benchmarks

### Automatic (CI)

Benchmarks are **automatically captured** when you push a version tag. The CI workflow:
- Runs 9 packages in parallel (~5 min wall time vs ~20 min sequential)
- Commits results to `benchmarks/benchmark-v1.X.Y.txt`
- Compares with the previous version using benchstat

**No manual action required for releases.**

### Manual (Development)

Update benchmarks manually in these situations:
- After making performance-related changes (to verify no regression)
- After adding new functionality that may affect performance
- When significant changes are made to core packages (parser, validator, fixer, converter, joiner, overlay, differ, generator, builder)

## Prerequisites

- Ensure all code changes are complete and tested
- Ensure `make check` passes (all tests, formatting, and linting pass)
- Close unnecessary applications to minimize system load during benchmarking
- Ensure the system is not under heavy load (for consistent results)

## Quick Start: Release Benchmarks

### Pre-Release Workflow (Recommended)

Benchmarks should be captured as part of the pre-release process, before tagging. This ensures the benchmark is included in the release PR.

```bash
# 1. Create and push pre-release branch with all changes
git checkout -b chore/v1.X.Y-release-prep
# ... make changes ...
git push -u origin chore/v1.X.Y-release-prep

# 2. Trigger benchmark workflow on the branch
gh workflow run benchmark.yml \
  -f version="v1.X.Y" \
  -f ref="chore/v1.X.Y-release-prep" \
  -f output_mode=commit

# 3. Wait for completion (~5 min)
sleep 15
RUN_ID=$(gh run list --workflow=benchmark.yml --limit=1 --json databaseId -q '.[0].databaseId')
gh run watch "$RUN_ID" --exit-status

# 4. Pull the benchmark commit
git pull origin chore/v1.X.Y-release-prep

# 5. Create PR (benchmark file is included)
gh pr create --title "chore: prepare v1.X.Y release" ...
```

The `/prepare-release` skill automates this entire process.

### Post-Tag Workflow (Fallback)

If you push a tag without running benchmarks first, the CI workflow will:
1. Run benchmarks on the tagged version
2. Create a separate PR to add the benchmark file

```bash
git tag v1.X.Y
git push origin v1.X.Y
# Workflow creates PR automatically
```

Monitor the workflow:
```bash
gh run list --workflow=benchmark.yml --limit=1
gh run watch <RUN_ID>
```

### Local Development Benchmarks

For validating performance locally before a release:

```bash
# Quick check - I/O-isolated benchmarks only (~2 min)
make bench-quick

# Fast full check - all benchmarks with 1s iterations (~5-7 min)
make bench-fast

# Parallel execution - faster wall time but interleaved output
make bench-parallel
```

### Manual Release Benchmarks (Legacy)

If you need to capture benchmarks manually:

```bash
# Capture benchmarks for upcoming release (e.g., v1.19.1)
make bench-release VERSION=v1.19.1
```

This command:
1. Runs all package benchmarks with proper timeout handling
2. Saves results directly to `benchmarks/benchmark-v1.19.1.txt`
3. Automatically compares with the previous version (if `benchstat` is installed)

After running, commit the benchmark file:
```bash
git add benchmarks/benchmark-v1.19.1.txt
git commit -m "chore: add benchmark results for v1.19.1"
```

## Detailed Process

### 1. Run All Benchmarks

For individual package benchmarks or debugging, run each separately:

```bash
# Run parser benchmarks
make bench-parser

# Run validator benchmarks
make bench-validator

# Run fixer benchmarks
make bench-fixer

# Run converter benchmarks
make bench-converter

# Run joiner benchmarks
make bench-joiner

# Run differ benchmarks
make bench-differ

# Run generator benchmarks
make bench-generator

# Run builder benchmarks
make bench-builder

# Run overlay benchmarks (includes jsonpath)
make bench-overlay
```

**Alternative:** Run all benchmarks at once:
```bash
make bench
```

### 2. Collect Benchmark Results

The benchmark output includes:
- **Iterations**: Number of times the benchmark ran (e.g., `42094`)
- **Time per operation**: In nanoseconds (e.g., `142212 ns/op`)
- **Memory per operation**: In bytes (e.g., `202678 B/op`)
- **Allocations per operation**: Number of allocations (e.g., `2128 allocs/op`)

Example output:
```
BenchmarkParseSmallOAS3-10    42094    142212 ns/op    202678 B/op    2128 allocs/op
```

### 3. Update benchmarks.md

Update the following sections in `benchmarks.md` with the new results:

#### 3.1 Parser Performance

**Document Parsing table:**
- Convert nanoseconds to microseconds (divide by 1,000)
- Convert bytes to kilobytes (divide by 1,024)
- Round to whole numbers for readability

Example:
```
BenchmarkParseSmallOAS3: 142212 ns/op, 202678 B/op, 2128 allocs/op
→ Small OAS3: 142 μs, 203 KB, 2,128 allocs
```

**JSON Marshaling table:**
- Keep time in nanoseconds (round to whole numbers)
- Keep memory in bytes (round to whole numbers)
- Keep allocations exact

Example:
```
BenchmarkMarshalInfoWithExtra: 1717 ns/op, 1737 B/op, 26 allocs/op
→ Info (5 fields): 1,717 ns, 1,737 bytes, 26 allocs
```

#### 3.2 Validator Performance

**Validation table:**
- Convert nanoseconds to microseconds
- Convert bytes to kilobytes
- Round to whole numbers

**ValidateParsed table:**
- Keep microseconds with one decimal place for small values (e.g., 4.7 μs)
- Keep kilobytes with one decimal place for small values (e.g., 5.2 KB)
- Keep allocations exact

#### 3.3 Fixer Performance

**Fixing table (parse + fix):**
- Convert nanoseconds to microseconds
- Convert bytes to kilobytes
- Round to whole numbers

**FixParsed table:**
- Keep microseconds with one decimal place for small values (e.g., 86 μs)
- Keep kilobytes with one decimal place for small values (e.g., 79 KB)
- Keep allocations exact

#### 3.4 Converter Performance

**Conversion table (parse + convert):**
- Convert nanoseconds to microseconds
- Convert bytes to kilobytes
- Round to whole numbers

**ConvertParsed table:**
- Keep microseconds with one decimal place
- Keep kilobytes with one decimal place
- Keep allocations exact

#### 3.5 Joiner Performance

**Joining table (parse + join):**
- Convert nanoseconds to microseconds
- Convert bytes to kilobytes
- Round to whole numbers

**JoinParsed table:**
- Keep time in nanoseconds (round to whole numbers)
- Keep memory in bytes (round to whole numbers)
- Keep allocations exact

#### 3.6 Differ Performance

**Diffing table (parse + diff):**
- Convert nanoseconds to microseconds
- Convert bytes to kilobytes
- Round to whole numbers

**DiffParsed table:**
- Keep microseconds with one decimal place
- Keep kilobytes with one decimal place
- Keep allocations exact

#### 3.7 Generator Performance

**Code generation table:**
- Convert nanoseconds to microseconds
- Convert bytes to kilobytes
- Round to whole numbers

#### 3.8 Builder Performance

**Builder operations table:**
- Keep time in nanoseconds (round to whole numbers)
- Keep memory in bytes (round to whole numbers)
- Keep allocations exact

### 4. Update README.md

Update the **Document Processing Performance** table in README.md:
- Use the same values from benchmarks.md's "Current Performance Metrics" section
- Ensure consistency between the two files

Example:
```
| Parse            | 142 μs            | 1,130 μs            | 14,131 μs           |
| Validate         | 143 μs            | 1,160 μs            | 14,635 μs           |
| Fix              | 220 μs            | 2,034 μs            | 24,957 μs           |
| Convert (OAS2→3) | 153 μs            | 1,314 μs            | -                   |
| Join (2 docs)    | 115 μs            | -                   | -                   |
| Diff (2 docs)    | 448 μs            | -                   | -                   |
| Generate (types) | 39 μs             | -                   | -                   |
| Generate (all)   | 249 μs            | -                   | -                   |
```

*Note: Generator benchmarks use pre-parsed documents. Builder constructs documents programmatically (~8-33 μs).*

### 5. Verify Changes

Before committing, verify:
- All tables are properly formatted (aligned columns)
- All numbers use comma separators for thousands (e.g., `2,128` not `2128`)
- Microsecond values are consistent across benchmarks.md and README.md
- Observations and commentary still make sense with the new numbers

### 6. Commit Changes

Create a commit with the updated benchmarks:

```bash
git add benchmarks.md README.md
git commit -m "docs: update benchmark results for v1.x.x release"
```

## Tips for Accurate Benchmarking

1. **Consistent Environment**: Run benchmarks on the same machine with similar system load
2. **Multiple Runs**: If results seem inconsistent, run benchmarks multiple times and use the median
3. **Benchmark Time**: Use `BENCH_TIME` to run longer benchmarks for more stable results:
   ```bash
   make bench BENCH_TIME=10s
   ```
4. **Baseline Comparison**: Save baseline benchmarks and use `benchstat` to compare:
   ```bash
   make bench-baseline          # Save current as baseline
   # Make changes...
   make bench-save              # Save new results
   make bench-compare OLD=benchmark-baseline.txt NEW=benchmark-YYYY-MM-DD-HHMMSS.txt
   ```

## Benchmark Storage

### Directory Structure

```
benchmarks/
├── local/                          # Historical local Mac benchmarks (darwin/arm64)
│   ├── benchmark-v1.9.10.txt
│   ├── benchmark-v1.10.0.txt
│   └── ... (52 files, v1.9.10 – v1.33.1)
├── benchmark-v1.28.0.txt           # CI-generated benchmarks (linux/amd64)
├── benchmark-v1.28.1.txt
└── ... (new releases)
```

### File Locations

- **CI-generated benchmarks**: `benchmarks/benchmark-v1.X.Y.txt` (committed via PR)
- **Historical local benchmarks**: `benchmarks/local/benchmark-v1.X.Y.txt` (preserved for reference)
- **Comparison reports**: `benchmarks/benchmark-comparison-v1.X.Y.txt` (ignored by git)
- **Timestamped benchmarks**: `benchmark-YYYYMMDD-HHMMSS.txt` (root, ignored by git)

### CI vs Local Benchmarks

| Aspect | CI Benchmarks | Local Benchmarks |
|--------|---------------|------------------|
| Platform | linux/amd64 | darwin/arm64 (Mac) |
| Location | `benchmarks/` | `benchmarks/local/` |
| Consistency | ✅ Reproducible | ❌ Varies with system load |
| Comparison | ✅ Apples-to-apples | ❌ Cross-platform variance |
| Creation | Pre-release workflow or tag push | Manual via `make bench-release` |

**Best Practice:** Use CI benchmarks for cross-version comparisons. Local benchmarks are useful for quick regression checks during development but should not be used for official release comparisons.

The benchmark scripts and CI workflow automatically handle file organization and cleanup of temporary files.

## Workflow Output Modes

The benchmark workflow supports three output modes via the `output_mode` parameter:

| Mode | Use Case | Behavior |
|------|----------|----------|
| `pr` | Tag push (default) | Creates PR to add benchmark to main |
| `commit` | Pre-release branch | Commits directly to the specified ref |
| `artifact` | Batch backfill | Uploads artifact only, no commit |

Example usage:
```bash
# Pre-release: commit to branch
gh workflow run benchmark.yml -f version="v1.34.0" -f ref="chore/v1.34.0-prep" -f output_mode=commit

# Backfill: artifact only
gh workflow run benchmark.yml -f version="v1.28.0" -f output_mode=artifact
```

## Backfilling Historical Benchmarks

To generate CI benchmarks for historical versions, use the backfill script:

```bash
# Backfill default versions (v1.28.0 onwards, ~10 versions)
./scripts/backfill-ci-benchmarks.sh

# Backfill specific versions
./scripts/backfill-ci-benchmarks.sh v1.30.0 v1.31.0 v1.32.0
```

The script:
1. Triggers the benchmark workflow with `output_mode=artifact` for each version
2. Waits for each workflow run to complete
3. Downloads the combined artifact from each run
4. Creates a single PR with all backfilled benchmarks

**Note:** Each version takes ~5 minutes in CI. Backfilling 10 versions takes ~50 minutes (sequential).

## Platform Information

Benchmark files include platform information at the top of each output:

**CI Benchmarks (linux/amd64):**
```
goos: linux
goarch: amd64
pkg: github.com/erraggy/oastools/parser
cpu: AMD EPYC 7763 64-Core Processor
```

**Local Benchmarks (darwin/arm64):**
```
goos: darwin
goarch: arm64
pkg: github.com/erraggy/oastools/parser
cpu: Apple M4
```

When updating `benchmarks.md`, note the platform used for the measurements.

## Troubleshooting

**Benchmarks show wildly different results:**
- Ensure system is not under load
- Close other applications
- Run benchmarks multiple times
- Consider using `make bench-baseline` and `benchstat` for comparison

**Benchmark command fails:**
- Ensure `make check` passes first
- Ensure all dependencies are installed (`make deps`)
- Check that test files exist in `testdata/bench/` directory

**Numbers don't match between packages:**
- This is expected - different packages have different overhead
- Parser is the baseline; other packages build on top of parsing

## Related Documentation

- [benchmarks.md](benchmarks.md) - Detailed performance analysis
- [CLAUDE.md](CLAUDE.md) - Development guidelines
