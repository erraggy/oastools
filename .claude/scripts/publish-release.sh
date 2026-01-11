#!/usr/bin/env bash
set -euo pipefail

# publish-release.sh - Deterministic release publishing
#
# This script exists because Claude cannot be trusted to follow
# skill instructions in fresh sessions (Claude Code v2.1.3+ bug).
# See: https://github.com/anthropics/claude-code/releases/tag/v2.1.3
#
# NEVER use `gh release create` - goreleaser creates the draft.
# Using `gh release create` makes the release immutable before
# assets can be uploaded, breaking brew/installer distribution.

VERSION="${1:-}"

if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version>" >&2
    echo "Example: $0 v1.44.1" >&2
    exit 1
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must match vX.Y.Z pattern" >&2
    exit 1
fi

echo "=== Phase 7: Publish Release $VERSION ==="
echo ""

# Step 1: Verify on main
echo "Step 1: Verifying on main branch..."
BRANCH=$(git branch --show-current)
if [[ "$BRANCH" != "main" ]]; then
    echo "Error: Must be on main branch (currently on: $BRANCH)" >&2
    exit 1
fi
git pull origin main
echo "✓ On main branch, up to date"
echo ""

# Step 2: Create and push tag
echo "Step 2: Creating and pushing tag..."
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo "Error: Tag $VERSION already exists" >&2
    exit 1
fi
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"
echo "✓ Tag $VERSION pushed"
echo ""

# Step 3: Wait for goreleaser workflow
echo "Step 3: Waiting for release workflow..."
sleep 15
RUN_ID=$(gh run list --workflow=release.yml --limit=1 --json databaseId -q '.[0].databaseId')
echo "Watching workflow run $RUN_ID..."
if ! gh run watch "$RUN_ID" --exit-status; then
    echo "Error: Release workflow failed" >&2
    exit 1
fi
echo "✓ Release workflow completed"
echo ""

# Step 4: Verify draft has assets
echo "Step 4: Verifying draft release..."
RELEASE_INFO=$(gh release view "$VERSION" --json isDraft,assets \
    --jq '{isDraft, assetCount: (.assets | length)}')
IS_DRAFT=$(echo "$RELEASE_INFO" | jq -r '.isDraft')
ASSET_COUNT=$(echo "$RELEASE_INFO" | jq -r '.assetCount')

if [[ "$IS_DRAFT" != "true" ]]; then
    echo "Error: Release is not a draft (isDraft=$IS_DRAFT)" >&2
    echo "This means someone used 'gh release create' - release is broken" >&2
    exit 1
fi

if [[ "$ASSET_COUNT" -lt 8 ]]; then
    echo "Error: Expected 8 assets, got $ASSET_COUNT" >&2
    echo "Goreleaser may have failed to upload binaries" >&2
    exit 1
fi
echo "✓ Draft release verified: $ASSET_COUNT assets"
echo ""

# Step 5: Generate notes and publish
echo "Step 5: Publishing release..."
PREV_TAG=$(git describe --tags --abbrev=0 HEAD^)
NOTES=$(gh api repos/erraggy/oastools/releases/generate-notes \
    -f tag_name="$VERSION" \
    -f previous_tag_name="$PREV_TAG" \
    --jq '.body')

gh release edit "$VERSION" --notes "$NOTES" --draft=false
echo "✓ Release published"
echo ""

# Step 6: Verify
echo "Step 6: Verifying published release..."
gh release view "$VERSION" --json name,isDraft,assets \
    --jq '{name, isDraft, assetCount: (.assets | length), assets: [.assets[].name]}'
echo ""
echo "=== Release $VERSION complete ==="
