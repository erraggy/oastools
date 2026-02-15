#!/usr/bin/env bash
set -euo pipefail

# prepare-release.sh - Phases 4-6 of release preparation
#
# This script handles the deterministic steps of release preparation:
# - Phase 4: Trigger CI benchmarks and wait
# - Phase 5: Create and merge pre-release PR
# - Phase 6: Generate release notes
#
# Usage: prepare-release.sh <version> [--skip-benchmarks]
#
# Exit codes:
#   0 - Success
#   1 - Usage/validation error
#   2 - Prerequisite failed
#   3 - External service failed
#   4 - Verification failed

VERSION="${1:-}"
SKIP_BENCHMARKS=false
[[ "${2:-}" == "--skip-benchmarks" ]] && SKIP_BENCHMARKS=true

if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version> [--skip-benchmarks]" >&2
    echo "Example: $0 v1.46.0" >&2
    exit 1
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must match vX.Y.Z pattern (got: $VERSION)" >&2
    exit 1
fi

BRANCH="chore/${VERSION}-release-prep"
REPO="erraggy/oastools"

# Verify we're on the correct branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "$BRANCH" ]]; then
    echo "Error: Must be on branch '$BRANCH' (currently on: $CURRENT_BRANCH)" >&2
    exit 2
fi

echo "=== Prepare Release $VERSION ==="
echo "Branch: $BRANCH"
echo ""

# Strip leading 'v' for semver-only contexts (plugin.json, marketplace.json).
SEMVER="${VERSION#v}"

# =============================================================================
# Phase 3.5: Sync Plugin Version
# =============================================================================

echo "=== Phase 3.5: Sync Plugin Version ==="

PLUGIN_JSON="plugin/.claude-plugin/plugin.json"
MARKETPLACE_JSON=".claude-plugin/marketplace.json"

echo "Step 3.5.1: Updating plugin version to $SEMVER..."
if [[ -f "$PLUGIN_JSON" ]]; then
    # Use a temp file for portable sed -i behavior (macOS vs Linux).
    jq --arg v "$SEMVER" '.version = $v' "$PLUGIN_JSON" > "${PLUGIN_JSON}.tmp" \
        && mv "${PLUGIN_JSON}.tmp" "$PLUGIN_JSON"
    echo "  ✓ $PLUGIN_JSON → $SEMVER"
else
    echo "  Warning: $PLUGIN_JSON not found, skipping"
fi

echo "Step 3.5.2: Updating marketplace version to $SEMVER..."
if [[ -f "$MARKETPLACE_JSON" ]]; then
    jq --arg v "$SEMVER" '.plugins[0].version = $v' "$MARKETPLACE_JSON" > "${MARKETPLACE_JSON}.tmp" \
        && mv "${MARKETPLACE_JSON}.tmp" "$MARKETPLACE_JSON"
    echo "  ✓ $MARKETPLACE_JSON → $SEMVER"
else
    echo "  Warning: $MARKETPLACE_JSON not found, skipping"
fi

# Commit version bump if there are changes.
if ! git diff --quiet "$PLUGIN_JSON" "$MARKETPLACE_JSON" 2>/dev/null; then
    git add "$PLUGIN_JSON" "$MARKETPLACE_JSON"
    git commit -m "chore: bump plugin version to $SEMVER"
    echo "  ✓ Version bump committed"
else
    echo "  Versions already at $SEMVER, no commit needed"
fi
echo ""

# =============================================================================
# Phase 4: CI Benchmarks
# =============================================================================

echo "=== Phase 4: CI Benchmarks ==="

# Step 4.1: Push branch to origin
echo "Step 4.1: Pushing branch to origin..."
if git ls-remote --exit-code origin "$BRANCH" &>/dev/null; then
    echo "  Branch already exists on remote, pulling latest..."
    git pull origin "$BRANCH" --rebase
else
    git push -u origin "$BRANCH"
    echo "  ✓ Branch pushed"
fi
echo ""

# Step 4.2-4.4: Trigger and wait for benchmark workflow
BENCHMARK_FILE="benchmarks/benchmark-${VERSION}.txt"

if [[ "$SKIP_BENCHMARKS" == "true" ]]; then
    echo "Step 4.2-4.4: Skipping benchmarks (--skip-benchmarks flag)"
    if [[ ! -f "$BENCHMARK_FILE" ]]; then
        echo "  Warning: Benchmark file does not exist: $BENCHMARK_FILE"
    fi
elif [[ -f "$BENCHMARK_FILE" ]]; then
    echo "Step 4.2-4.4: Benchmark file already exists, skipping workflow"
    echo "  File: $BENCHMARK_FILE"
else
    echo "Step 4.2: Triggering benchmark workflow..."
    gh workflow run benchmark.yml \
        --ref "$BRANCH" \
        -f version="$VERSION" \
        -f ref="$BRANCH" \
        -f output_mode=commit
    echo "  ✓ Workflow triggered"
    echo ""

    echo "Step 4.3: Waiting for benchmark workflow..."
    RUN_ID=""
    for i in 1 2 3 4 5 6; do
        sleep $((i * 10))  # Progressive backoff: 10s, 20s, 30s, 40s, 50s, 60s
        RUN_ID=$(gh run list --workflow=benchmark.yml --branch="$BRANCH" --limit=1 --json databaseId -q '.[0].databaseId')
        if [ -n "$RUN_ID" ]; then
            break
        fi
        echo "  Waiting for run to appear (attempt $i/6)..."
    done
    if [ -z "$RUN_ID" ]; then
        echo "Error: Benchmark workflow run not found after 210s" >&2
        echo "Check manually: https://github.com/$REPO/actions" >&2
        exit 3
    fi
    echo "  Watching run $RUN_ID..."
    if ! gh run watch "$RUN_ID" --exit-status; then
        echo "Error: Benchmark workflow failed" >&2
        echo "Check: https://github.com/$REPO/actions/runs/$RUN_ID" >&2
        exit 3
    fi
    echo "  ✓ Workflow completed"
    echo ""

    echo "Step 4.4: Pulling benchmark commit..."
    git pull origin "$BRANCH"
    echo "  ✓ Benchmark commit pulled"
fi
echo ""

# =============================================================================
# Phase 5: Create Pre-Release PR
# =============================================================================

echo "=== Phase 5: Create Pre-Release PR ==="

# Step 5.1: Verify benchmark file exists
echo "Step 5.1: Verifying benchmark file..."
if [[ ! -f "$BENCHMARK_FILE" ]]; then
    echo "Error: Benchmark file not found: $BENCHMARK_FILE" >&2
    echo "Run without --skip-benchmarks or check workflow output" >&2
    exit 4
fi
echo "  ✓ Benchmark file exists"
echo ""

# Step 5.2: Create PR if it doesn't exist
echo "Step 5.2: Creating PR..."

# Get PR info including state to handle already-merged PRs on re-run
PR_INFO=$(gh pr list --head "$BRANCH" --state all --json number,state -q '.[0] // empty')
SKIP_TO_CHECKOUT=false

if [[ -n "$PR_INFO" ]]; then
    PR_STATE=$(echo "$PR_INFO" | jq -r '.state')
    PR_NUMBER=$(echo "$PR_INFO" | jq -r '.number')

    if [[ "$PR_STATE" == "MERGED" ]]; then
        echo "  PR #$PR_NUMBER already merged, skipping to checkout"
        SKIP_TO_CHECKOUT=true
    else
        echo "  PR #$PR_NUMBER already exists (state: $PR_STATE)"
    fi
else
    PR_URL=$(gh pr create \
        --title "chore: prepare $VERSION release" \
        --body "## Summary

- Pre-release preparation for $VERSION
- Includes CI benchmark results

## Checklist

- [x] All tests pass
- [x] Benchmarks recorded
- [x] Documentation reviewed

---
🤖 Generated by prepare-release.sh" \
        --base main \
        --head "$BRANCH")
    PR_NUMBER=$(echo "$PR_URL" | grep -oE '[0-9]+$')
    echo "  ✓ PR created: $PR_URL"
fi
echo ""

if [[ "$SKIP_TO_CHECKOUT" == "false" ]]; then
    # Step 5.3: Wait for CI and merge
    echo "Step 5.3: Waiting for CI checks..."
    if ! gh pr checks "$PR_NUMBER" --watch --fail-fast; then
        echo "Error: CI checks failed for PR #$PR_NUMBER" >&2
        exit 3
    fi
    echo "  ✓ CI checks passed"
    echo ""

    echo "Step 5.4: Merging PR..."
    if ! gh pr merge "$PR_NUMBER" --squash --admin --delete-branch; then
        echo "Error: Failed to merge PR #$PR_NUMBER" >&2
        exit 3
    fi
    echo "  ✓ PR merged"
    echo ""
fi

# Switch to main and pull
echo "Step 5.5: Switching to main..."
git checkout main
git pull origin main
echo "  ✓ On main with latest changes"
echo ""

# =============================================================================
# Phase 6: Generate Release Notes
# =============================================================================

echo "=== Phase 6: Generate Release Notes ==="

# Step 6.1: Generate notes via GitHub API
echo "Step 6.1: Generating release notes..."
if ! PREV_TAG=$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null); then
    echo "Error: Could not find previous tag. Is this the first release?" >&2
    exit 4
fi
echo "  Previous tag: $PREV_TAG"

mkdir -p .release
NOTES_FILE=".release/notes-${VERSION}.md"

# Get auto-generated notes
AUTO_NOTES=$(gh api "repos/$REPO/releases/generate-notes" \
    -f tag_name="$VERSION" \
    -f previous_tag_name="$PREV_TAG" \
    --jq '.body')

# Get linked issues
LAST_TAG_DATE=$(git log -1 --format=%cI "$PREV_TAG")
LINKED_ISSUES=$(gh pr list --state merged --base main --limit 50 \
    --json number,title,mergedAt,closingIssuesReferences | \
    jq -r --arg since "$LAST_TAG_DATE" '
        [.[] | select(.mergedAt > $since and (.closingIssuesReferences | length > 0))] |
        if length == 0 then "None"
        else .[] | "- #\(.closingIssuesReferences[0].number) - Fixed by PR #\(.number)"
        end')

# Step 6.2: Save to temp file
cat > "$NOTES_FILE" << EOF
$AUTO_NOTES

## Issues Fixed

$LINKED_ISSUES

## Highlights

### Features
- [Add brief description of major new features]

### Bug Fixes
- [Add brief description of significant fixes]

### Performance
- [Any performance improvements]

## Breaking Changes
- None (backward compatible)

## Upgrade Notes
- No special upgrade steps required

**Full Changelog**: https://github.com/$REPO/compare/$PREV_TAG...$VERSION
EOF

echo "  ✓ Release notes saved to: $NOTES_FILE"
echo ""

# Display notes for review
echo "=== Release Notes Preview ==="
cat "$NOTES_FILE"
echo ""
echo "================================"
echo ""

echo "=== Preparation Complete ==="
echo ""
echo "Version: $VERSION"
echo "Notes:   $NOTES_FILE"
echo ""
echo "Next step: Review the release notes above, then run:"
echo "  .claude/scripts/publish-release.sh $VERSION"
echo ""
