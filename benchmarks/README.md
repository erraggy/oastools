# Benchmarks

This directory contains versioned benchmark results for the oastools project.

## Directory Structure

```
benchmarks/
├── benchmark-v1.28.0.txt           # CI-generated benchmarks (linux/amd64)
├── benchmark-v1.28.1.txt           # Consistent, reproducible results
├── ... (new releases)
├── local/                          # Historical local Mac benchmarks (darwin/arm64)
│   ├── benchmark-v1.9.10.txt
│   ├── benchmark-v1.10.0.txt
│   └── ... (52 files, v1.9.10 – v1.33.1)
└── README.md
```

### CI Benchmarks (Recommended for Comparisons)

- **Location:** `benchmarks/benchmark-v*.txt`
- **Platform:** linux/amd64 (GitHub Actions runner, AMD EPYC 7763)
- **Generated:** Automatically when version tags are pushed
- **Use for:** Cross-version performance comparisons (consistent environment)

### Local Benchmarks (Historical Reference)

- **Location:** `benchmarks/local/benchmark-v*.txt`
- **Platform:** darwin/arm64 (Apple Silicon Macs)
- **Generated:** Manually via `make bench-release`
- **Use for:** Historical reference only (±50% I/O variance between runs)

## Generating Benchmarks

### Automatic (CI) - Recommended

Benchmarks are **automatically generated** when you push a version tag:

```bash
git tag v1.X.Y
git push origin v1.X.Y
```

The CI workflow:
- Runs 9 packages in parallel (~5 min wall time)
- Commits results to `benchmarks/benchmark-v1.X.Y.txt`
- Compares with the previous version using benchstat

### Pre-Release Workflow

For capturing benchmarks before tagging (included in release PR):

```bash
# Trigger benchmark workflow on a pre-release branch
gh workflow run benchmark.yml \
  -f version="v1.X.Y" \
  -f ref="chore/v1.X.Y-prep" \
  -f output_mode=commit
```

The `/prepare-release` skill automates this process.

### Backfilling Historical Versions (CI)

Use the CI backfill script for consistent results:

```bash
# Backfill default versions (v1.28.0+)
./scripts/backfill-ci-benchmarks.sh

# Backfill specific versions
./scripts/backfill-ci-benchmarks.sh v1.25.0 v1.26.0 v1.27.0
```

The script:
- Triggers the benchmark workflow for each version
- Waits for completion (~5 min per version)
- Downloads artifacts and creates a single PR

### Local Development Benchmarks

For quick validation during development:

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
make bench-release VERSION=v1.X.Y
```

Results are saved to `benchmarks/local/benchmark-v1.X.Y.txt` (requires manual move to `local/` subdirectory).

### Comparing Versions

```bash
# Using benchstat directly
benchstat benchmarks/benchmark-v1.32.0.txt benchmarks/benchmark-v1.33.0.txt

# Using comparison script
./scripts/generate-benchmark-comparison.sh v1.32.0 v1.33.0
```

## File Organization

### Committed Files (tracked in git)
- CI benchmarks: `benchmarks/benchmark-v*.txt`
- Local benchmarks: `benchmarks/local/benchmark-v*.txt`

### Ignored Files (not tracked)
- Timestamped benchmarks in root: `benchmark-YYYYMMDD-HHMMSS.txt`
- Comparison reports: `benchmarks/benchmark-comparison-*.txt`
- Baseline file in root: `benchmark-baseline.txt`

## CI vs Local Benchmarks

| Aspect | CI Benchmarks | Local Benchmarks |
|--------|---------------|------------------|
| Platform | linux/amd64 | darwin/arm64 (Mac) |
| Location | `benchmarks/` | `benchmarks/local/` |
| Consistency | ✅ Reproducible | ❌ ±50% I/O variance |
| Comparison | ✅ Apples-to-apples | ❌ Cross-platform variance |
| Creation | Automatic on tag push | Manual |

**Best Practice:** Use CI benchmarks for cross-version comparisons. Local benchmarks are useful for quick regression checks during development.

## Related Documentation

- [BENCHMARK_UPDATE_PROCESS.md](../BENCHMARK_UPDATE_PROCESS.md) - Detailed benchmark update process
- [benchmarks.md](../benchmarks.md) - Performance analysis and interpretation
- [Makefile](../Makefile) - Benchmark-related make targets
