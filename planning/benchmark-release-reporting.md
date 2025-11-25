# Benchmark Reporting for Releases

**Status:** Proposed
**Author:** Claude (AI Assistant)
**Created:** 2025-11-24
**Purpose:** Define strategy for including benchmark performance comparisons in release notes

## Overview

This document outlines a strategy for automatically including performance metrics in GitHub releases, allowing users to understand not just what features and fixes are included, but also how performance has changed compared to the previous version.

## Goals

1. **Transparency**: Users can see actual performance characteristics of each release
2. **Regression Detection**: Performance regressions are immediately visible
3. **Improvement Tracking**: Performance improvements are highlighted and celebrated
4. **Historical Context**: Users can compare performance across multiple versions

## Proposed Solution

### 1. Benchmark Comparison Files

**Current State:**
- Benchmark results are saved to timestamped files: `benchmark-YYYYMMDD-HHMMSS.txt`
- Baseline file: `benchmark-baseline.txt`
- Comparison is manual using `make bench-compare`

**Proposed Enhancement:**
- Add `benchmark-v{VERSION}.txt` for each release (e.g., `benchmark-v1.10.0.txt`)
- Keep these version-tagged files in the repository
- Use GoReleaser to generate comparison reports automatically

### 2. Release Notes Template

Add a **Performance Metrics** section to release notes that includes:

```markdown
## Performance Metrics

Benchmark results on Apple M4 (darwin/arm64) with Go 1.24:

### Parser Performance
| Operation | Small OAS3 | Medium OAS3 | Large OAS3 | vs v1.9.12 |
|-----------|------------|-------------|------------|------------|
| Parse     | 142 μs     | 1,130 μs    | 14,131 μs  | +2.1% ⚠️   |
| Marshal   | 20 μs      | 222 μs      | 2,723 μs   | -5.3% ✅   |

### Validator Performance
| Operation | Small OAS3 | Medium OAS3 | Large OAS3 | vs v1.9.12 |
|-----------|------------|-------------|------------|------------|
| Validate  | 143 μs     | 1,160 μs    | 14,635 μs  | +0.1%      |

### Converter Performance
| Operation      | Small | Medium | vs v1.9.12 |
|----------------|-------|--------|------------|
| OAS2→OAS3      | 153 μs| 1,314 μs| -1.2% ✅  |
| Parsed Convert | 16 μs | 252 μs | -0.5%     |

### Joiner Performance
| Operation | 2 docs | 3 docs | vs v1.9.12 |
|-----------|--------|--------|------------|
| Join      | 115 μs | 161 μs | +0.3%      |
| Parsed    | 712 ns | 901 ns | +1.1%      |

**Legend:**
- ✅ = Performance improvement (>5%)
- ⚠️ = Performance regression (>5%)
- No emoji = Performance within 5% (acceptable variance)

<details>
<summary>Full Benchmark Results</summary>

[Full benchmark output attached as release asset: `benchmark-v1.10.0.txt`]

</details>
```

### 3. Automated Generation Script

Create a new script `scripts/generate-benchmark-comparison.sh`:

```bash
#!/bin/bash
# Generate benchmark comparison for releases

PREV_VERSION=$1
CURR_VERSION=$2

if [ -z "$PREV_VERSION" ] || [ -z "$CURR_VERSION" ]; then
    echo "Usage: $0 <prev_version> <curr_version>"
    echo "Example: $0 v1.9.12 v1.10.0"
    exit 1
fi

PREV_FILE="benchmark-${PREV_VERSION}.txt"
CURR_FILE="benchmark-${CURR_VERSION}.txt"

if [ ! -f "$PREV_FILE" ]; then
    echo "Error: Previous benchmark file not found: $PREV_FILE"
    exit 1
fi

if [ ! -f "$CURR_FILE" ]; then
    echo "Error: Current benchmark file not found: $CURR_FILE"
    exit 1
fi

# Use benchstat for comparison
benchstat "$PREV_FILE" "$CURR_FILE" > "benchmark-comparison-${CURR_VERSION}.txt"

echo "Comparison saved to: benchmark-comparison-${CURR_VERSION}.txt"
```

### 4. Integration with Release Process

Update the release process (documented in CLAUDE.md):

**Before creating release:**
```bash
# 1. Update benchmarks
make bench-save

# 2. Copy timestamped benchmark to version-tagged file
cp benchmark-YYYYMMDD-HHMMSS.txt benchmark-v1.10.0.txt

# 3. Generate comparison (if previous version benchmark exists)
./scripts/generate-benchmark-comparison.sh v1.9.12 v1.10.0

# 4. Commit benchmark files
git add benchmark-v1.10.0.txt benchmark-comparison-v1.10.0.txt
git commit -m "chore: add benchmark results for v1.10.0"
```

**During release creation:**
```bash
gh release create v1.10.0 \
  --title "v1.10.0 - ..." \
  --notes "$(cat release-notes.md)" \
  benchmark-v1.10.0.txt \
  benchmark-comparison-v1.10.0.txt
```

### 5. GoReleaser Integration (Future Enhancement)

Modify `.goreleaser.yml` to include benchmark files as release assets:

```yaml
release:
  extra_files:
    - glob: benchmark-v*.txt
    - glob: benchmark-comparison-v*.txt
```

## Implementation Plan

### Phase 1: Manual Process (v1.10.0)

1. ✅ Document the manual process in CLAUDE.md
2. Create `scripts/generate-benchmark-comparison.sh`
3. Generate benchmark files for v1.10.0
4. Include performance section in release notes
5. Attach benchmark files to GitHub release

### Phase 2: Semi-Automated (Future)

1. Add benchmark file detection to release script
2. Auto-generate comparison if previous version exists
3. Template-based release notes with benchmark placeholders

### Phase 3: Fully Automated (Future)

1. GoReleaser workflow integration
2. Automatic benchstat comparison in CI
3. Performance regression detection in PR checks
4. Historical performance graphs in documentation

## Benefits

### For Users

- **Informed Decisions**: Users can assess performance impact before upgrading
- **Trust**: Transparency builds confidence in the project
- **Planning**: Users can plan capacity based on performance characteristics

### For Maintainers

- **Quality Gate**: Forces attention to performance with each release
- **Documentation**: Performance history is preserved
- **Motivation**: Performance improvements are celebrated

### For Contributors

- **Feedback**: Contributors see the impact of their optimizations
- **Standards**: Sets expectations for performance-sensitive changes

## Considerations

### Storage and Repository Size

- **Benchmark files are ~15-20 KB each**
- Two files per release (benchmark + comparison) = ~35-40 KB
- 10 releases = ~350-400 KB (negligible for Git)
- Consider moving to GitHub Releases exclusively after 1 year

### Benchmark Stability

- **Environment**: Benchmarks are platform-specific (Apple M4 in this case)
- **Variance**: Small changes (<5%) should not be highlighted
- **Consistency**: Always run on same hardware for valid comparisons

### Maintenance Burden

- **Phase 1 (Manual)**: ~5-10 minutes per release
- **Phase 2 (Semi)**: ~2-3 minutes per release
- **Phase 3 (Auto)**: Near zero after initial setup

## Alternative Approaches Considered

### 1. Separate Performance Report Repository

**Pros:**
- Keeps main repo clean
- Dedicated space for performance analysis

**Cons:**
- Requires separate maintenance
- Disconnected from release process
- Users might not discover it

**Decision:** Rejected - Benefits don't outweigh complexity

### 2. GitHub Actions Dashboard

**Pros:**
- Automated tracking
- Visual graphs over time

**Cons:**
- Requires CI infrastructure
- Not visible in release notes
- More complex setup

**Decision:** Consider for Phase 3

### 3. Benchmark-as-Documentation

**Pros:**
- Integrated with godoc
- Always up-to-date

**Cons:**
- No historical comparison
- Not release-specific

**Decision:** Complement, not replacement

## Examples from Other Projects

### 1. Go Standard Library

- Uses `golang.org/x/benchmarks` repository
- Continuous performance tracking
- Not directly in release notes

### 2. Rust Compiler

- Performance tracking at https://perf.rust-lang.org
- Detailed per-commit analysis
- Comprehensive but complex

### 3. SQLite

- Performance numbers in release notes
- Simple tables with key metrics
- Historical comparisons

**Inspiration:** SQLite's approach is most aligned with our goals

## Success Metrics

How we'll know this is working:

1. **Adoption**: Users reference performance metrics when discussing releases
2. **Discovery**: Performance regressions caught before release
3. **Engagement**: GitHub release page views increase
4. **Feedback**: User comments about performance (positive or questions)

## Future Enhancements

### Performance Regression Tests

Add CI check that fails if performance degrades >10%:

```yaml
- name: Check for performance regressions
  run: |
    make bench-save
    ./scripts/check-performance-regression.sh benchmark-baseline.txt benchmark-latest.txt
```

### Interactive Performance Dashboard

Host a simple static site showing:
- Performance trends over versions
- Interactive charts
- Comparison tools

### Platform-Specific Benchmarks

Expand to include:
- Linux/amd64 benchmarks
- Windows benchmarks
- Highlight platform-specific differences

## Conclusion

Starting with a simple, manual process (Phase 1) for v1.10.0 allows us to:

1. Establish the practice without significant investment
2. Gather feedback from users
3. Iterate based on actual needs
4. Build automation incrementally

The immediate value is transparency and performance awareness, with room to grow into more sophisticated tooling as the project matures.

## Back-filling Historical Benchmarks

For establishing historical baselines, see:

- [planning/benchmark-backfill-process.md](benchmark-backfill-process.md) - How to back-fill benchmarks for older releases
- [scripts/backfill-benchmarks.sh](../scripts/backfill-benchmarks.sh) - Automated back-fill tool

This allows generating benchmark comparisons against older releases even if they weren't benchmarked at the time.

## Related Documentation

- [BENCHMARK_UPDATE_PROCESS.md](../BENCHMARK_UPDATE_PROCESS.md) - How to update benchmarks
- [benchmarks.md](../benchmarks.md) - Detailed performance analysis
- [CLAUDE.md](../CLAUDE.md) - Release process documentation
- [planning/benchmark-backfill-process.md](benchmark-backfill-process.md) - Back-filling older versions
