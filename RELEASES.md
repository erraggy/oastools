# Release Process Guide

This document describes the release workflow for oastools, which is compatible with GitHub's **immutable releases** security feature.

## Overview

The oastools project uses a **tag-first release workflow** that maintains compatibility with GitHub's release immutability while preserving control over release quality through a human-in-the-loop publish step.

### Key Benefits

✅ **Security**: Releases are immutable after publishing (tamper-proof)
✅ **Quality**: Claude Code-generated or hand-crafted markdown release notes
✅ **Automation**: Binaries built and distributed automatically
✅ **Control**: Manual publish step allows verification before going public

### How It Works

```
Human pushes tag → Workflow creates draft + uploads assets → Human reviews → Human publishes
```

1. Human creates and pushes a semver tag
2. Tag push triggers GitHub Actions workflow
3. GoReleaser creates a draft release and uploads all assets
4. Human generates/edits release notes and reviews the draft
5. Human publishes the release (hands-on-keyboard)
6. Release becomes immutable (assets already attached, no 422 errors)

## Prerequisites

### Release Immutability (Already Enabled)

Release immutability is already enabled on this repository. This means:
- **Draft releases** remain fully mutable until published
- **Published releases** become immutable (assets and tags cannot be modified or deleted)
- **Release notes** remain editable for the life of the release

### Required Configuration

All required configuration is already in place:

- [x] Release immutability enabled in repository settings
- [x] `.goreleaser.yaml` has `draft: true` in the `release` section
- [x] `.github/workflows/release.yml` triggers on `push: tags: - 'v*'`
- [x] `HOMEBREW_TAP_TOKEN` secret exists with `repo` scope
- [x] No tag rulesets block manual tag pushes

## Release Workflow

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

### Step 2: Determine Version Number

Follow [Semantic Versioning (SemVer)](https://semver.org/):

- **PATCH** (`v1.2.3` → `v1.2.4`): Bug fixes, docs, refactors without API changes
- **MINOR** (`v1.2.3` → `v1.3.0`): New features, optimizations, new public APIs (backward compatible)
- **MAJOR** (`v1.2.3` → `v2.0.0`): Breaking changes to public APIs (rare)

### Step 3: Create and Push the Tag

```bash
# Create the tag
git tag v1.X.Y

# Push the tag (this triggers the workflow)
git push origin v1.X.Y
```

**What happens:**
- Tag is pushed to GitHub
- GitHub Actions workflow is triggered automatically
- Workflow begins building binaries and creating draft release

### Step 4: Monitor the Workflow

```bash
# Watch the workflow run
gh run list --workflow=release.yml --limit=1

# Get the run ID and monitor progress
gh run watch <RUN_ID>
```

The workflow will:
- Build binaries for all platforms (Darwin arm64/x86_64, Linux arm64/i386/x86_64, Windows i386/x86_64)
- Create a **draft** release on GitHub
- Upload all binary assets to the draft
- Push the Homebrew formula to `erraggy/homebrew-oastools`

### Step 5: Verify the Draft Release

```bash
# Confirm draft status and assets
gh release view v1.X.Y --json isDraft,assets --jq '{isDraft, assetCount: (.assets | length), assets: [.assets[].name]}'
```

**Expected output:**
- `isDraft: true`
- `assetCount: 8`
- Assets: checksums.txt + binaries for all platforms

### Step 6: Generate and Set Release Notes

Ask Claude Code to generate release notes based on commits and PRs since the last release.

**Example prompt:**
```
Generate release notes for v1.X.Y based on changes since the last release
```

Claude Code will:
1. Review git log and merged PRs
2. Categorize changes (features, bug fixes, improvements, breaking changes)
3. Generate well-formatted markdown release notes
4. Apply them to the draft release using:

```bash
gh release edit v1.X.Y --notes "$(cat <<'EOF'
## Summary
[High-level overview of what this release delivers]

## What's New
- [Feature 1: Description]
- [Feature 2: Description]

## Bug Fixes
- [Fix 1]

## Improvements
- [Improvement 1]

## Breaking Changes
- [If any]

## Related PRs
- #XX - [PR title]

## Installation

### Homebrew
\`\`\`bash
brew tap erraggy/oastools
brew install oastools
\`\`\`

### Go Module
\`\`\`bash
go get github.com/erraggy/oastools@v1.X.Y
\`\`\`

### Binary Download
Download the appropriate binary for your platform from the assets below.
EOF
)"
```

**Note:** Release notes can be edited before or after publishing via `gh release edit` or the GitHub web UI.

### Step 7: Publish the Release (Hands-on-Keyboard)

```bash
# This makes the release immutable (tags and assets are locked)
gh release edit v1.X.Y --draft=false
```

**⚠️ Once published:**
- The release becomes **immutable**
- Assets and tags cannot be modified or deleted
- Release notes can still be edited
- This action is permanent

### Step 8: Verify Published Release

```bash
# Check release is published
gh release view v1.X.Y --json isDraft,publishedAt

# Test Homebrew installation
brew update
brew upgrade oastools || brew install erraggy/oastools/oastools
oastools --version
```

## Troubleshooting

### Workflow Failed Before Creating Draft

```bash
# Delete the tag and start over
git push origin :refs/tags/v1.X.Y
git tag -d v1.X.Y

# Fix the issue, then repeat from Step 3
```

### Workflow Failed After Creating Draft (Missing Assets)

```bash
# Delete the draft release and tag
gh release delete v1.X.Y --yes
git push origin :refs/tags/v1.X.Y
git tag -d v1.X.Y

# Fix the issue, then repeat from Step 3
```

### Accidentally Published Too Early

With immutability enabled, this cannot be fixed without consequences:

1. You may need to temporarily disable immutability to delete the release
2. Delete the tag
3. Increment the version number
4. Start over from Step 3

**This is why the workflow uploads assets to a draft first, then requires manual publishing.**

### GoReleaser Can't Push to homebrew-oastools

**Possible causes:**
- `HOMEBREW_TAP_TOKEN` secret not configured or expired
- PAT doesn't have `repo` scope
- Commit author email in `.goreleaser.yaml` doesn't match verified GitHub email

**Solution:**
```bash
# Verify secret exists
gh secret list --repo erraggy/oastools

# Check .goreleaser.yaml commit_author.email matches your verified GitHub email
# Regenerate PAT if needed
```

### Build Fails

**Solution:**
- Review GitHub Actions logs: `gh run view --log-failed`
- Check CGO dependencies
- Test locally: `make release-test`

### Formula Doesn't Work

**Solution:**
- Verify formula was pushed to `erraggy/homebrew-oastools`
- Test in a clean environment
- Check formula syntax

## Configuration Reference

### GoReleaser Configuration

**File:** `.goreleaser.yaml`

```yaml
release:
  # CRITICAL: Must be true for immutable releases workflow
  # GoReleaser uploads assets to a draft, human publishes afterward
  draft: true
```

### GitHub Actions Workflow

**File:** `.github/workflows/release.yml`

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

### Repository Settings

**Required settings:**

1. **Release immutability:** ENABLED (Settings → General → Releases)
2. **Workflow permissions:** "Read and write permissions" (Settings → Actions → General)
3. **Branch protection:** Enabled on `main` (does not affect tag creation)

### Personal Access Token Setup

**Why needed:** The default `GITHUB_TOKEN` cannot push to the separate `homebrew-oastools` repository.

**Creating the PAT:**

1. GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Generate new token with `repo` scope
3. Add to repository: Settings → Secrets and variables → Actions
4. Name: `HOMEBREW_TAP_TOKEN`

**Verification:**
```bash
gh secret list --repo erraggy/oastools
```

## Why This Workflow Works

### The Tag-First Approach

**Previous problematic approaches:**

1. **Create published release first:** Release is immediately immutable → 422 errors when uploading
2. **Create draft via gh release create --draft:** Draft doesn't push tag → workflow doesn't trigger until published → release is immutable when workflow runs → 422 errors

**Current solution: Tag-first workflow**

1. Push tag first → triggers workflow immediately
2. Workflow creates draft release (mutable)
3. Workflow uploads assets to draft (still mutable)
4. Human reviews and publishes when ready
5. Immutability only applies after publish

This sequence ensures assets are always uploaded while the release is still mutable.

## Security Benefits

With immutability enabled:

- **Tag integrity:** Tags cannot be moved to different commits after publication
- **Asset integrity:** Binary artifacts cannot be replaced with malicious versions
- **Release attestations:** GitHub automatically generates cryptographically verifiable records
- **Repository resurrection protection:** Deleted repos cannot reuse tags from immutable releases

## References

### GitHub Documentation

- [Immutable Releases](https://docs.github.com/en/code-security/supply-chain-security/understanding-your-software-supply-chain/immutable-releases)
- [Events that trigger workflows - Release](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#release)
- [Managing Releases in a Repository](https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository)

### GoReleaser Documentation

- [Release Customization](https://goreleaser.com/customization/release/)
- [GitHub Actions](https://goreleaser.com/ci/actions/)
- [Homebrew Formulas](https://goreleaser.com/customization/homebrew/)

### oastools Documentation

- [CLAUDE.md](./CLAUDE.md) - Complete project documentation including release process
- [planning/releases-with-immutability.md](./planning/releases-with-immutability.md) - Detailed release workflow analysis

### Related Issues

- [GitHub Issue #39](https://github.com/erraggy/oastools/issues/39) - Release management discussion
- [planning/release-issues.md](./planning/release-issues.md) - Historical context and lessons learned

---

**Last Updated:** 2025-11-25
**Workflow Version:** Tag-first with immutability support
