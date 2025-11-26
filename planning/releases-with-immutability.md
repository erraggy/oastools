# Release Process with GitHub Immutable Releases

## Overview

This document describes a release workflow that is compatible with GitHub's **immutable releases** feature while maintaining a human-in-the-loop for the final publish action.

### Key Insight

With immutable releases enabled:
- **Immutable at publish time:** Git tags and release assets cannot be modified or deleted after publication
- **Always editable:** Release notes/description can be edited for the life of the release
- **Draft releases:** Remain fully mutable until published

This means we can upload all assets to a draft release, then have a human publish it when ready.

## Workflow Summary

```
Human pushes tag → Workflow creates draft + uploads assets → Human reviews → Human publishes
```

1. Human creates and pushes a semver tag
2. Tag push triggers GitHub Actions workflow
3. GoReleaser creates a draft release and uploads all assets
4. Human reviews the draft release on GitHub
5. Human publishes the release (hands-on-keyboard)
6. Release becomes immutable (assets already attached, no 422 errors)

## Prerequisites

### 1. Release Immutability (Already Enabled)

Release immutability is already enabled on this repository. No action needed.

To verify: **Settings → General → Releases** should show "Release immutability" checked.

### 2. Verify No Tag Protection Rules Block Manual Pushes

```bash
# Check for tag-targeting rulesets
gh api repos/erraggy/oastools/rulesets --jq '.[] | select(.target == "tag")'
```

If empty, manual tag pushes are allowed. If rules exist, they may need adjustment.

### 3. Ensure HOMEBREW_TAP_TOKEN Secret Exists

```bash
gh secret list --repo erraggy/oastools | grep HOMEBREW_TAP_TOKEN
```

This PAT needs `repo` scope to push the Homebrew formula to `erraggy/homebrew-oastools`.

## Implementation Changes

### File 1: `.github/workflows/release.yml`

The workflow trigger remains on tag push (not release events). GoReleaser's `draft: true` setting ensures assets are uploaded to a draft.

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v5
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v6
        with:
          go-version: '1.24'

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

**No changes required** - the current workflow is already correct.

### File 2: `.goreleaser.yaml`

Ensure `draft: true` is set in the release section. This is critical for immutable releases.

```yaml
# ... existing configuration ...

release:
  # CRITICAL: Must be true for immutable releases workflow
  # GoReleaser uploads assets to a draft, human publishes afterward
  draft: true
```

**No changes required** - the current configuration already has `draft: true`.

## Release Process Steps

### Step 1: Prepare for Release

```bash
# Ensure you're on main and up-to-date
git checkout main
git pull origin main

# Run all checks
make check

# Review changes since last release
git log $(git describe --tags --abbrev=0)..HEAD --oneline
```

### Step 2: Create and Push the Tag

```bash
# Determine the next version (follow semver)
# PATCH: bug fixes, docs, refactors without API changes
# MINOR: new features, optimizations (backward compatible)
# MAJOR: breaking changes to public APIs

# Create the tag
git tag v1.X.Y

# Push the tag (this triggers the workflow)
git push origin v1.X.Y
```

### Step 3: Monitor the Workflow

```bash
# Watch the workflow run
gh run list --workflow=release.yml --limit=1

# Get the run ID and monitor it
gh run watch <RUN_ID>
```

The workflow will:
1. Build binaries for all platforms (Darwin, Linux, Windows)
2. Create a **draft** release on GitHub
3. Upload all binary assets to the draft
4. Push the Homebrew formula to `erraggy/homebrew-oastools`

### Step 4: Verify the Draft Release

```bash
# Confirm draft status and assets
gh release view v1.X.Y --json isDraft,assets --jq '{isDraft, assetCount: (.assets | length), assets: [.assets[].name]}'
```

Expected output:
```json
{
  "isDraft": true,
  "assetCount": 8,
  "assets": [
    "checksums.txt",
    "oastools_Darwin_arm64.tar.gz",
    "oastools_Darwin_x86_64.tar.gz",
    "oastools_Linux_arm64.tar.gz",
    "oastools_Linux_i386.tar.gz",
    "oastools_Linux_x86_64.tar.gz",
    "oastools_Windows_i386.zip",
    "oastools_Windows_x86_64.zip"
  ]
}
```

### Step 5: Edit Release Notes (Optional)

You can edit the release notes before or after publishing:

```bash
# Edit release notes before publishing
gh release edit v1.X.Y --notes "$(cat <<'EOF'
## Summary
Brief description of this release.

## What's New
- Feature 1
- Feature 2

## Bug Fixes
- Fix 1

## Related PRs
- #XX - PR title
EOF
)"
```

Or edit via the GitHub web UI at: `https://github.com/erraggy/oastools/releases/tag/v1.X.Y`

### Step 6: Publish the Release (Human Action)

**This is the hands-on-keyboard step that makes the release immutable:**

```bash
gh release edit v1.X.Y --draft=false
```

After this command:
- The release is published and visible to users
- The tag and assets become **immutable**
- Release notes remain editable

### Step 7: Verify Publication

```bash
# Confirm release is published
gh release view v1.X.Y --json isDraft,publishedAt

# Test Homebrew installation
brew update
brew upgrade oastools || brew install erraggy/oastools/oastools
oastools --version
```

## Troubleshooting

### Workflow Failed Before Creating Draft

If the workflow fails before creating the release:

```bash
# Delete the tag
git push origin :refs/tags/v1.X.Y
git tag -d v1.X.Y

# Fix the issue, then start over from Step 2
```

### Workflow Failed After Creating Draft (Assets Missing)

If the workflow partially succeeded (draft exists but missing assets):

```bash
# Check what assets exist
gh release view v1.X.Y --json assets

# Delete the draft release
gh release delete v1.X.Y --yes

# Delete the tag
git push origin :refs/tags/v1.X.Y
git tag -d v1.X.Y

# Fix the issue, then start over from Step 2
```

### Accidentally Published Too Early

If you published the draft before assets were uploaded:

**With immutability enabled, this cannot be fixed.** You must:

1. Delete the release (if possible - may require temporarily disabling immutability)
2. Delete the tag
3. Increment the version number and start over

This is why the workflow is designed to upload assets first, publish last.

### 422 Error: Cannot Upload Assets to Immutable Release

This error means the release was published before assets were uploaded. See "Accidentally Published Too Early" above.

## Why This Workflow Works

### The Problem with Previous Approaches

**Approach 1: Create published release first, then upload assets**
- With immutability: Release is immediately immutable → 422 errors when uploading

**Approach 2: Create draft via `gh release create --draft`, then publish**
- Draft releases don't push tags
- Tag is only pushed when draft is published
- Workflow triggers on tag push, but release is already published/immutable → 422 errors

### The Solution: Tag-First Workflow

1. Push tag first → triggers workflow immediately
2. Workflow creates draft release (mutable)
3. Workflow uploads assets to draft (still mutable)
4. Human publishes when ready
5. Immutability only applies after publish

This sequence ensures assets are always uploaded while the release is still mutable.

## Security Benefits of Immutable Releases

With immutability enabled:

1. **Tag integrity:** Tags cannot be moved to different commits after publication
2. **Asset integrity:** Binary artifacts cannot be replaced with malicious versions
3. **Release attestations:** GitHub automatically generates cryptographically verifiable records
4. **Repository resurrection protection:** Deleted repos cannot reuse tags from immutable releases

## References

- [GitHub Docs: Immutable Releases](https://docs.github.com/en/code-security/supply-chain-security/understanding-your-software-supply-chain/immutable-releases)
- [GitHub Docs: Events that trigger workflows - Release](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#release)
- [GoReleaser: Release Customization](https://goreleaser.com/customization/release/)
- [GoReleaser: GitHub Actions](https://goreleaser.com/ci/actions/)
- [GitHub Issue #39](https://github.com/erraggy/oastools/issues/39)

## Implementation Checklist

When implementing this plan, verify the following:

- [x] Release immutability is enabled in repository settings (already enabled)
- [x] `.goreleaser.yaml` has `draft: true` in the `release` section (already configured)
- [x] `.github/workflows/release.yml` triggers on `push: tags: - 'v*'` (already configured)
- [x] `HOMEBREW_TAP_TOKEN` secret exists with `repo` scope (already configured)
- [x] No tag rulesets block manual tag pushes (verified - only branch protection exists)
- [x] Update `CLAUDE.md` "Creating a New Release" section to reflect this workflow
