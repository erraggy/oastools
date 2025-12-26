#!/bin/bash
# backfill-ci-benchmarks.sh - Backfill CI-generated benchmarks for historical versions
#
# This script triggers the benchmark workflow for multiple versions, downloads the
# artifacts, and creates a SINGLE PR with all backfilled benchmarks.
#
# Usage:
#   ./scripts/backfill-ci-benchmarks.sh [versions...]
#
# Examples:
#   ./scripts/backfill-ci-benchmarks.sh                    # Backfill default versions (v1.28.0+)
#   ./scripts/backfill-ci-benchmarks.sh v1.30.0 v1.31.0    # Backfill specific versions
#
# Prerequisites:
#   - gh CLI installed and authenticated
#   - Push access to the repository
#   - On main branch with clean working directory

set -euo pipefail

# Default versions to backfill (v1.28.0 onwards - ~10 versions)
DEFAULT_VERSIONS=(
    "v1.28.0"
    "v1.28.1"
    "v1.29.0"
    "v1.30.0"
    "v1.30.1"
    "v1.31.0"
    "v1.32.0"
    "v1.32.1"
    "v1.33.0"
    "v1.33.1"
)

# Use provided versions or defaults
if [[ $# -gt 0 ]]; then
    VERSIONS=("$@")
else
    VERSIONS=("${DEFAULT_VERSIONS[@]}")
fi

# Configuration
BRANCH="chore/benchmark-backfill-ci"
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

echo "=== CI Benchmark Backfill ==="
echo "Versions to backfill: ${VERSIONS[*]}"
echo "Temp directory: $TEMP_DIR"
echo ""

# Verify we're on main with clean working directory
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    echo "ERROR: Must be on main branch (currently on: $CURRENT_BRANCH)"
    exit 1
fi

if ! git diff --quiet || ! git diff --staged --quiet; then
    echo "ERROR: Working directory is not clean. Commit or stash changes first."
    exit 1
fi

# Create the backfill branch
echo "Creating branch: $BRANCH"
git checkout -b "$BRANCH"

# Track results
declare -A RESULTS
declare -A RUN_IDS
FAILED=0
SUCCEEDED=0

# Phase 1: Trigger all workflows
echo ""
echo "========================================"
echo "Phase 1: Triggering workflows"
echo "========================================"

for VERSION in "${VERSIONS[@]}"; do
    echo "Triggering workflow for $VERSION..."

    # Trigger the workflow with artifact-only mode (no commit/PR)
    if gh workflow run benchmark.yml -f version="$VERSION" -f output_mode=artifact; then
        echo "  ✓ Triggered $VERSION"
    else
        echo "  ✗ Failed to trigger $VERSION"
        RESULTS[$VERSION]="FAILED (trigger)"
        ((FAILED++))
    fi

    # Small delay between triggers to avoid rate limiting
    sleep 2
done

# Wait for workflows to appear
echo ""
echo "Waiting for workflows to start..."
sleep 15

# Phase 2: Wait for all workflows and download artifacts
echo ""
echo "========================================"
echo "Phase 2: Waiting for workflows"
echo "========================================"

for VERSION in "${VERSIONS[@]}"; do
    # Skip if already failed
    if [[ "${RESULTS[$VERSION]:-}" == "FAILED"* ]]; then
        continue
    fi

    echo ""
    echo "Processing $VERSION..."

    # Find the run for this version
    RUN_ID=$(gh run list --workflow=benchmark.yml --limit=20 --json databaseId,displayTitle \
        -q ".[] | select(.displayTitle | contains(\"$VERSION\")) | .databaseId" | head -1)

    if [[ -z "$RUN_ID" ]]; then
        echo "  ✗ Could not find run ID for $VERSION"
        RESULTS[$VERSION]="FAILED (no run)"
        ((FAILED++))
        continue
    fi

    RUN_IDS[$VERSION]=$RUN_ID
    echo "  Run ID: $RUN_ID"

    # Wait for the run to complete
    echo "  Waiting for completion..."
    if gh run watch "$RUN_ID" --exit-status; then
        echo "  ✓ Workflow completed successfully"

        # Download the combined artifact
        echo "  Downloading artifact..."
        ARTIFACT_DIR="$TEMP_DIR/$VERSION"
        mkdir -p "$ARTIFACT_DIR"

        if gh run download "$RUN_ID" --name "benchmark-combined-$VERSION" --dir "$ARTIFACT_DIR" 2>/dev/null; then
            # Copy benchmark file to benchmarks directory
            BENCH_FILE=$(find "$ARTIFACT_DIR" -name "benchmark-*.txt" | head -1)
            if [[ -n "$BENCH_FILE" ]]; then
                cp "$BENCH_FILE" "benchmarks/"
                echo "  ✓ Downloaded and added: $(basename "$BENCH_FILE")"
                RESULTS[$VERSION]="SUCCESS"
                ((SUCCEEDED++))
            else
                echo "  ✗ No benchmark file in artifact"
                RESULTS[$VERSION]="FAILED (no file)"
                ((FAILED++))
            fi
        else
            echo "  ✗ Failed to download artifact"
            RESULTS[$VERSION]="FAILED (download)"
            ((FAILED++))
        fi
    else
        echo "  ✗ Workflow failed"
        RESULTS[$VERSION]="FAILED (workflow)"
        ((FAILED++))
    fi
done

# Phase 3: Create single PR with all benchmarks
echo ""
echo "========================================"
echo "Phase 3: Creating PR"
echo "========================================"

# Check if we have any successful benchmarks
if [[ $SUCCEEDED -eq 0 ]]; then
    echo "ERROR: No benchmarks were successfully generated"
    git checkout main
    git branch -D "$BRANCH"
    exit 1
fi

# Commit all benchmark files
git add benchmarks/
if git diff --staged --quiet; then
    echo "No new benchmarks to commit"
    git checkout main
    git branch -D "$BRANCH"
    exit 0
fi

# Create commit with all versions
VERSIONS_LIST=$(printf '%s, ' "${VERSIONS[@]}")
VERSIONS_LIST=${VERSIONS_LIST%, }  # Remove trailing comma

git commit -m "chore: add CI benchmarks for ${VERSIONS_LIST}

Backfilled CI-generated benchmarks (linux/amd64) for:
$(for v in "${VERSIONS[@]}"; do
    status="${RESULTS[$v]:-UNKNOWN}"
    if [[ "$status" == "SUCCESS" ]]; then
        echo "  - $v ✓"
    fi
done)

Generated via scripts/backfill-ci-benchmarks.sh

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"

# Push and create PR
git push -u origin "$BRANCH"

gh pr create \
    --title "chore: add CI benchmarks for ${VERSIONS[0]}–${VERSIONS[-1]}" \
    --body "$(cat <<EOF
## Summary

Backfilled CI-generated benchmarks (linux/amd64) for ${#VERSIONS[@]} versions.

## Versions

| Version | Status |
|---------|--------|
$(for v in "${VERSIONS[@]}"; do
    status="${RESULTS[$v]:-UNKNOWN}"
    if [[ "$status" == "SUCCESS" ]]; then
        echo "| $v | ✅ Success |"
    else
        echo "| $v | ❌ $status |"
    fi
done)

## Details

- **Succeeded:** $SUCCEEDED
- **Failed:** $FAILED
- **Platform:** linux/amd64 (GitHub Actions runner)

These CI-generated benchmarks provide consistent, reproducible results for cross-version comparisons.

---
🤖 Generated via \`scripts/backfill-ci-benchmarks.sh\`
EOF
)" \
    --base main \
    --head "$BRANCH"

# Return to main
git checkout main

# Summary
echo ""
echo "========================================"
echo "=== Backfill Summary ==="
echo "========================================"
for VERSION in "${VERSIONS[@]}"; do
    STATUS="${RESULTS[$VERSION]:-UNKNOWN}"
    if [[ "$STATUS" == "SUCCESS" ]]; then
        echo "  ✅ $VERSION"
    else
        echo "  ❌ $VERSION: $STATUS"
    fi
done
echo ""
echo "Succeeded: $SUCCEEDED / ${#VERSIONS[@]}"
echo ""

if [[ $FAILED -gt 0 ]]; then
    echo "WARNING: $FAILED version(s) failed."
    echo "Re-run failed versions after merging this PR:"
    echo "  $0 <failed-versions...>"
fi

echo ""
echo "PR created! Review and merge to add benchmarks to main."
