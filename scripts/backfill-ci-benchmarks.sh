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
#
# Note: This script is compatible with bash 3.2 (macOS default)

set -euo pipefail

# Default versions to backfill (v1.28.0 onwards - ~10 versions)
DEFAULT_VERSIONS="v1.28.0 v1.28.1 v1.29.0 v1.30.0 v1.30.1 v1.31.0 v1.32.0 v1.32.1 v1.33.0 v1.33.1"

# Use provided versions or defaults
if [[ $# -gt 0 ]]; then
    VERSIONS="$*"
else
    VERSIONS="$DEFAULT_VERSIONS"
fi

# Configuration
BRANCH="chore/benchmark-backfill-ci"
TEMP_DIR=$(mktemp -d)
STATUS_FILE="$TEMP_DIR/status.txt"
trap 'rm -rf "$TEMP_DIR"' EXIT

echo "=== CI Benchmark Backfill ==="
echo "Versions to backfill: $VERSIONS"
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

# Initialize counters
FAILED=0
SUCCEEDED=0

# Initialize status file
touch "$STATUS_FILE"

# Helper to get status
get_status() {
    local version="$1"
    grep "^$version:" "$STATUS_FILE" 2>/dev/null | cut -d: -f2 || echo "PENDING"
}

# Helper to set status
set_status() {
    local version="$1"
    local status="$2"
    # Remove old status and add new
    grep -v "^$version:" "$STATUS_FILE" > "$STATUS_FILE.tmp" 2>/dev/null || true
    echo "$version:$status" >> "$STATUS_FILE.tmp"
    mv "$STATUS_FILE.tmp" "$STATUS_FILE"
}

# Process each version sequentially
# (Trigger, wait, download - one at a time to reliably track run IDs)
echo ""
echo "========================================"
echo "Processing versions (trigger → wait → download)"
echo "========================================"

for VERSION in $VERSIONS; do
    echo ""
    echo "----------------------------------------"
    echo "Processing $VERSION..."
    echo "----------------------------------------"

    # Get the latest run ID before triggering (to detect new run)
    BEFORE_RUN_ID=$(gh run list --workflow=benchmark.yml --limit=1 --json databaseId -q '.[0].databaseId' 2>/dev/null || echo "0")

    # Trigger the workflow
    echo "  Triggering workflow..."
    if ! gh workflow run benchmark.yml -f version="$VERSION" -f output_mode=artifact; then
        echo "  ✗ Failed to trigger workflow"
        set_status "$VERSION" "FAILED (trigger)"
        FAILED=$((FAILED + 1))
        continue
    fi
    echo "  ✓ Workflow triggered"

    # Wait for the new run to appear
    echo "  Waiting for run to start..."
    RUN_ID=""
    for i in 1 2 3 4 5 6 7 8 9 10; do
        sleep 3
        NEW_RUN_ID=$(gh run list --workflow=benchmark.yml --limit=1 --json databaseId -q '.[0].databaseId' 2>/dev/null || echo "0")
        if [[ "$NEW_RUN_ID" != "$BEFORE_RUN_ID" && "$NEW_RUN_ID" != "0" ]]; then
            RUN_ID="$NEW_RUN_ID"
            break
        fi
    done

    if [[ -z "$RUN_ID" ]]; then
        echo "  ✗ Could not find new run ID"
        set_status "$VERSION" "FAILED (no run)"
        FAILED=$((FAILED + 1))
        continue
    fi

    echo "  Run ID: $RUN_ID"

    # Wait for the run to complete
    echo "  Waiting for completion (~5 min)..."
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
                set_status "$VERSION" "SUCCESS"
                SUCCEEDED=$((SUCCEEDED + 1))
            else
                echo "  ✗ No benchmark file in artifact"
                set_status "$VERSION" "FAILED (no file)"
                FAILED=$((FAILED + 1))
            fi
        else
            echo "  ✗ Failed to download artifact"
            set_status "$VERSION" "FAILED (download)"
            FAILED=$((FAILED + 1))
        fi
    else
        echo "  ✗ Workflow failed"
        set_status "$VERSION" "FAILED (workflow)"
        FAILED=$((FAILED + 1))
    fi
done

# Phase 3: Create single PR with all benchmarks
echo ""
echo "========================================"
echo "Creating PR"
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

# Build success list for commit message
SUCCESS_LIST=""
for v in $VERSIONS; do
    status=$(get_status "$v")
    if [[ "$status" == "SUCCESS" ]]; then
        SUCCESS_LIST="$SUCCESS_LIST  - $v ✓
"
    fi
done

# Create commit
git commit -m "chore: add CI benchmarks for multiple versions

Backfilled CI-generated benchmarks (linux/amd64) for:
$SUCCESS_LIST
Generated via scripts/backfill-ci-benchmarks.sh

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"

# Push and create PR
git push -u origin "$BRANCH"

# Build version table for PR body
VERSION_TABLE=""
for v in $VERSIONS; do
    status=$(get_status "$v")
    if [[ "$status" == "SUCCESS" ]]; then
        VERSION_TABLE="$VERSION_TABLE| $v | ✅ Success |
"
    else
        VERSION_TABLE="$VERSION_TABLE| $v | ❌ $status |
"
    fi
done

# Get first and last version for title
FIRST_VERSION=$(echo "$VERSIONS" | awk '{print $1}')
LAST_VERSION=$(echo "$VERSIONS" | awk '{print $NF}')
VERSION_COUNT=$(echo "$VERSIONS" | wc -w | tr -d ' ')

gh pr create \
    --title "chore: add CI benchmarks for ${FIRST_VERSION}–${LAST_VERSION}" \
    --body "## Summary

Backfilled CI-generated benchmarks (linux/amd64) for $VERSION_COUNT versions.

## Versions

| Version | Status |
|---------|--------|
$VERSION_TABLE
## Details

- **Succeeded:** $SUCCEEDED
- **Failed:** $FAILED
- **Platform:** linux/amd64 (GitHub Actions runner)

These CI-generated benchmarks provide consistent, reproducible results for cross-version comparisons.

---
🤖 Generated via \`scripts/backfill-ci-benchmarks.sh\`" \
    --base main \
    --head "$BRANCH"

# Return to main
git checkout main

# Summary
echo ""
echo "========================================"
echo "=== Backfill Summary ==="
echo "========================================"
for VERSION in $VERSIONS; do
    STATUS=$(get_status "$VERSION")
    if [[ "$STATUS" == "SUCCESS" ]]; then
        echo "  ✅ $VERSION"
    else
        echo "  ❌ $VERSION: $STATUS"
    fi
done
echo ""
echo "Succeeded: $SUCCEEDED / $VERSION_COUNT"
echo ""

if [[ $FAILED -gt 0 ]]; then
    echo "WARNING: $FAILED version(s) failed."
    echo "Re-run failed versions after merging this PR:"
    echo "  $0 <failed-versions...>"
fi

echo ""
echo "PR created! Review and merge to add benchmarks to main."
