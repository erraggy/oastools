# Benchmarks

This directory contains versioned benchmark results for the oastools project.

## Directory Structure

- **`benchmark-v{VERSION}.txt`** - Benchmark results for a specific release (committed to repo)
  - Example: `benchmark-v1.9.10.txt`, `benchmark-v1.9.12.txt`
  - These files are generated using `make bench-save` and committed for historical tracking

- **`benchmark-comparison-v{VERSION}.txt`** - Comparison reports between versions (git ignored)
  - Example: `benchmark-comparison-v1.10.0.txt`
  - Generated using `./scripts/generate-benchmark-comparison.sh`
  - Not committed to reduce repository size

## Generating Benchmarks

### For a New Release (Recommended)

Use the streamlined release benchmark command:

```bash
# Run benchmarks and save directly to versioned file
make bench-release VERSION=v1.19.1

# Commit the results
git add benchmarks/benchmark-v1.19.1.txt
git commit -m "chore: add benchmark results for v1.19.1"
```

This command:
- Runs all package benchmarks with proper timeout handling
- Saves results directly to `benchmarks/benchmark-v1.19.1.txt`
- Automatically compares with the previous version (if `benchstat` is installed)

### Manual Approach

If you prefer manual control:

1. Run all benchmarks and save the results:
   ```bash
   make bench-save
   ```

2. Copy the timestamped file to a version-tagged file:
   ```bash
   cp benchmark-YYYYMMDD-HHMMSS.txt benchmarks/benchmark-v1.10.0.txt
   ```

3. Commit the version-tagged file:
   ```bash
   git add benchmarks/benchmark-v1.10.0.txt
   git commit -m "chore: add benchmark results for v1.10.0"
   ```

### Back-filling Historical Versions

Use the automated script to generate benchmarks for previous releases:

```bash
# Back-fill specific versions
./scripts/backfill-benchmarks.sh v1.9.12 v1.9.11 v1.9.10

# Back-fill last N releases
./scripts/backfill-benchmarks.sh --last 5

# Back-fill all releases (use with caution!)
./scripts/backfill-benchmarks.sh --all
```

The script automatically:
- Checks out each version
- Runs benchmarks
- Saves results to `benchmarks/benchmark-v{VERSION}.txt`
- Cleans up temporary log files
- Returns to your original branch

### Comparing Versions

Generate a comparison report between two versions:

```bash
./scripts/generate-benchmark-comparison.sh v1.9.12 v1.10.0
```

This creates `benchmarks/benchmark-comparison-v1.10.0.txt` showing performance differences.

## File Organization

### Committed Files (tracked in git)
- Version-tagged benchmarks: `benchmarks/benchmark-v*.txt`
- These provide historical performance data

### Ignored Files (not tracked)
- Timestamped benchmarks in root: `benchmark-YYYYMMDD-HHMMSS.txt`
- Comparison reports: `benchmarks/benchmark-comparison-*.txt`
- Baseline file in root: `benchmark-baseline.txt`

## Related Documentation

- [BENCHMARK_UPDATE_PROCESS.md](../BENCHMARK_UPDATE_PROCESS.md) - Detailed benchmark update process
- [benchmarks.md](../benchmarks.md) - Performance analysis and interpretation
- [Makefile](../Makefile) - Benchmark-related make targets
