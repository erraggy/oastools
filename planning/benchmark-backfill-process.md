# Benchmark Back-filling Process

**Purpose:** Document how to generate benchmark results for previous releases to establish historical performance baselines.

**Status:** Documented
**Created:** 2025-11-24

## Overview

Back-filling benchmark results allows us to:
1. Create historical performance baselines for older releases
2. Generate comparative reports showing performance trends over time
3. Validate that recent changes haven't introduced regressions vs. older versions

## Prerequisites

- Git repository with tagged releases
- Go toolchain (ideally same major version as was used for the release)
- Clean working directory
- Sufficient disk space for checking out multiple versions

### Required Tooling

**benchstat** (for generating benchmark comparisons):
```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Verify installation
benchstat -h
```

**GitHub CLI** (optional, for uploading to releases):
```bash
# macOS
brew install gh

# Or download from: https://cli.github.com/

# Verify installation
gh --version

# Authenticate (first time only)
gh auth login
```

**Make** (should already be installed):
```bash
# Verify make is available
make --version
```

The `scripts/backfill-benchmarks.sh` script will check for these prerequisites and provide installation instructions if any are missing.

## Manual Back-fill Process

### Step 1: Identify Releases to Back-fill

List all releases:
```bash
git tag -l "v*" | sort -V
```

Or get recent releases only:
```bash
git tag -l "v*" | sort -V | tail -n 10
```

### Step 2: Back-fill for a Single Version

For each version you want to back-fill:

```bash
# 1. Save your current state
git stash

# 2. Checkout the release tag
git checkout v1.9.12

# 3. Ensure dependencies are correct for that version
go mod download
go mod tidy

# 4. Run benchmarks
make bench-save

# 5. Rename the timestamped file to version-tagged file
cp benchmark-YYYYMMDD-HHMMSS.txt benchmark-v1.9.12.txt

# 6. Return to main branch
git checkout main

# 7. Copy the benchmark file to main branch
git checkout v1.9.12 -- benchmark-v1.9.12.txt

# 8. Restore your working state
git stash pop
```

### Step 3: Commit Back-filled Benchmarks

```bash
git add benchmark-v1.9.12.txt
git commit -m "chore: back-fill benchmark results for v1.9.12"
```

## Automated Back-fill Process

Use the provided script for multiple versions:

```bash
# Back-fill specific versions
./scripts/backfill-benchmarks.sh v1.9.12 v1.9.11 v1.9.10

# Back-fill last N releases
./scripts/backfill-benchmarks.sh --last 5

# Back-fill all releases (use with caution!)
./scripts/backfill-benchmarks.sh --all
```

## Important Considerations

### 1. Go Version Compatibility

**Problem:** Older releases may not compile with newer Go versions.

**Solutions:**
- Use the same Go version that was current when the release was made
- Use `go mod edit -go=1.XX` to adjust go.mod if needed
- Skip versions that fail to build (document which ones)

**Example:**
```bash
# For v1.9.12 released in Nov 2024, use Go 1.24
go1.24 download  # if not already installed
go1.24 test -bench=. -benchmem ./...
```

### 2. Benchmark Stability

**Problem:** Benchmarks vary based on system load, CPU, Go version.

**Best Practices:**
- Run on the same machine used for current benchmarks
- Close other applications
- Run multiple times and take median if results vary significantly
- Document any anomalies in commit message

**Example:**
```bash
# Run 3 times and manually select the most representative
make bench-save  # Run 1
make bench-save  # Run 2
make bench-save  # Run 3
# Compare and choose the middle result
```

### 3. Changed Benchmark Tests

**Problem:** Benchmark tests may have changed between versions.

**Solution:**
- Only back-fill versions where benchmarks are comparable
- Document any major benchmark test changes in CHANGELOG
- Focus on recent releases (last 1-2 years)

### 4. Repository State

**Problem:** Checking out old tags can leave artifacts.

**Solution:**
```bash
# Always use a clean state
git clean -fdx  # WARNING: Removes all untracked files
go mod download
go mod tidy
```

## Storage Considerations

### Repository Size

- Each benchmark file: ~15-20 KB
- 10 versions: ~150-200 KB
- 50 versions: ~750 KB - 1 MB

**Recommendation:** Back-fill last 10-20 releases only. Older versions provide diminishing value.

### Alternative: GitHub Releases Only

Instead of committing to the repository, attach benchmark files to existing GitHub releases:

```bash
# Back-fill v1.9.12
git checkout v1.9.12
make bench-save
cp benchmark-YYYYMMDD-HHMMSS.txt benchmark-v1.9.12.txt

# Attach to existing release
gh release upload v1.9.12 benchmark-v1.9.12.txt

# Clean up
git checkout main
rm benchmark-v1.9.12.txt
```

**Pros:**
- No impact on repository size
- Benchmarks still associated with releases
- Easy to download via GitHub UI or API

**Cons:**
- Not in version control
- Can't diff easily with local git tools
- Requires GitHub CLI or manual upload

## Back-fill Strategy Recommendations

### For oastools Project

Given the current state:

1. **Priority: Last 3-5 releases**
   - Most relevant for users considering upgrades
   - Establishes recent trend line
   - Manageable effort

2. **Medium Priority: Major version milestones**
   - v1.0.0, v2.0.0 (if they exist)
   - First release of significant features
   - Useful for long-term trend analysis

3. **Low Priority: All other releases**
   - Diminishing returns
   - May not compile with current Go version
   - Historical interest only

### Recommended Workflow

```bash
# Back-fill last 5 releases
./scripts/backfill-benchmarks.sh --last 5

# Commit to repository
git add benchmark-v*.txt
git commit -m "chore: back-fill benchmark results for last 5 releases

Back-filled versions:
- v1.9.12
- v1.9.11
- v1.9.10
- v1.9.9
- v1.9.8

Benchmarks run on:
- Platform: darwin/arm64
- CPU: Apple M4
- Go: 1.24

All benchmarks ran successfully with no build issues."

# Alternative: Attach to existing releases instead
for version in v1.9.12 v1.9.11 v1.9.10 v1.9.9 v1.9.8; do
    if [ -f "benchmark-${version}.txt" ]; then
        gh release upload "$version" "benchmark-${version}.txt"
    fi
done
```

## Troubleshooting

### Build Fails for Old Version

```bash
# Try with older Go version
go1.23 download
go1.23 test -bench=. ./...

# Or skip that version
echo "Skipping v1.8.0 - build compatibility issues"
```

### Benchmarks Show Anomalies

```bash
# Document in commit message
git commit -m "chore: back-fill benchmarks for v1.9.12

Note: v1.9.12 shows 20% slower performance than v1.9.13.
This may be due to:
- Different Go version (1.23 vs 1.24)
- System load during benchmark run
- Known performance regression that was fixed in v1.9.13

These results should be interpreted with caution."
```

### Missing Dependencies

```bash
# Clean and re-download
go clean -modcache
go mod download
go mod tidy
```

## Validation

After back-filling, validate the results:

```bash
# Check file sizes are reasonable
ls -lh benchmark-v*.txt

# Check file format (should start with "goos: ")
head -5 benchmark-v1.9.12.txt

# Compare with newer version to ensure format consistency
diff -u <(head -20 benchmark-v1.9.12.txt) <(head -20 benchmark-v1.10.0.txt)

# Generate comparison to verify results make sense
./scripts/generate-benchmark-comparison.sh v1.9.12 v1.10.0
cat benchmark-comparison-v1.10.0.txt
```

## Related Documentation

- [BENCHMARK_UPDATE_PROCESS.md](../BENCHMARK_UPDATE_PROCESS.md) - How to update benchmarks for new releases
- [planning/benchmark-release-reporting.md](benchmark-release-reporting.md) - How to include benchmarks in release notes
- [scripts/backfill-benchmarks.sh](../scripts/backfill-benchmarks.sh) - Automated back-fill script
