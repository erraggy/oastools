# Release Process and GitHub Immutability Guide

This document provides comprehensive guidance on creating releases for oastools with GitHub's release immutability setting enabled. It includes lessons learned, best practices, and technical references.

## Table of Contents

- [Overview](#overview)
- [GitHub Release Immutability](#github-release-immutability)
- [Release Workflow](#release-workflow)
- [Configuration](#configuration)
- [Testing and Verification](#testing-and-verification)
- [Troubleshooting](#troubleshooting)
- [Technical References](#technical-references)

## Overview

The oastools project uses a **draft-based release workflow** to maintain compatibility with GitHub's release immutability security feature while preserving the ability to create hand-crafted, high-quality release notes.

### Key Benefits

✅ **Security**: Releases are immutable after publishing (tamper-proof)
✅ **Quality**: Hand-crafted markdown release notes
✅ **Automation**: Binaries built and distributed automatically
✅ **Control**: Manual publish step allows verification before going public

## GitHub Release Immutability

### What is Release Immutability?

GitHub's release immutability is a supply chain security feature that protects releases from tampering after publication. Once a release is published with immutability enabled, its assets and tags cannot be modified or deleted.

**Announced**: August 2024 (Public Preview)
**Generally Available**: October 2024

### How It Works

1. **Draft releases remain mutable**: You can add, modify, or delete assets
2. **Published releases become immutable**: Assets and tags are locked
3. **Protection applies to**:
   - Release assets (binaries, archives, etc.)
   - Git tags associated with the release
   - Pre-release releases are also immutable once published

### Why It Matters for GoReleaser

**The Problem**: GoReleaser traditionally creates a release and then uploads assets to it. With immutability enabled, this two-step process fails because:

1. GoReleaser creates and publishes the release
2. GitHub marks it as immutable
3. GoReleaser tries to add assets → **Error: "Cannot modify immutable release"**

**The Solution**: Use draft releases as an intermediary:

1. Create draft release (remains mutable)
2. GoReleaser adds assets to the draft
3. Manually publish the draft (becomes immutable)

### Immutability Behavior

| Action | Draft Release | Published Release |
|--------|---------------|-------------------|
| Add assets | ✅ Allowed | ❌ Blocked |
| Delete assets | ✅ Allowed | ❌ Blocked |
| Modify assets | ✅ Allowed | ❌ Blocked |
| Edit release notes | ✅ Allowed | ❌ Blocked* |
| Delete tag | ✅ Allowed | ❌ Blocked |
| Move tag | ✅ Allowed | ❌ Blocked |
| Delete entire release | ✅ Allowed | ✅ Allowed** |

\* Release notes may be editable depending on repository settings
\*\* You can delete immutable releases entirely, just not modify them

## Release Workflow

### Complete Step-by-Step Process

#### 1. Determine Version Number

Follow [Semantic Versioning (SemVer)](https://semver.org/):

- **PATCH** (`v1.2.3` → `v1.2.4`): Bug fixes, docs, minor changes
- **MINOR** (`v1.2.3` → `v1.3.0`): New features, backward-compatible changes
- **MAJOR** (`v1.2.3` → `v2.0.0`): Breaking changes to public APIs

#### 2. Test Locally (Optional but Recommended)

```bash
make release-test
```

This runs GoReleaser in snapshot mode to verify builds without publishing.

#### 3. Create Draft Release with Custom Notes

```bash
gh release create v1.7.1 --draft \
  --title "v1.7.1 - Brief summary within 72 chars" \
  --notes "$(cat <<'EOF'
## Summary

High-level overview of what this release delivers.

## What's New

- Feature 1: Description
- Feature 2: Description
- Performance: Improvements achieved

## Changes

- Change 1
- Change 2

## Technical Details

Additional context, benchmark results, migration notes, etc.

## Related PRs

- #17 - PR title
- #18 - PR title

## Installation

### Homebrew
```bash
brew tap erraggy/oastools
brew install oastools
```

### Binary Download
Download the appropriate binary for your platform from the assets below.
EOF
)"
```

**What happens**:
- Git tag is created (e.g., `v1.7.1`)
- Draft release is created with your custom notes
- GitHub Actions workflow is triggered automatically
- Draft remains **mutable** (compatible with immutability setting)

#### 4. Monitor Automated Build

The GitHub Actions workflow automatically:
- Builds binaries for all platforms (Linux, macOS, Windows)
- Adds binary archives to your draft release
- Updates the Homebrew Cask in `homebrew-oastools` repository

```bash
# Watch the workflow
gh run watch

# Or monitor at:
# - Workflow: https://github.com/erraggy/oastools/actions
# - Draft release: https://github.com/erraggy/oastools/releases
```

#### 5. Verify Draft Release

```bash
gh release view v1.7.1 --json assets,isDraft
```

**Confirm**:
- `isDraft: true`
- All platform binaries are attached (8 assets expected):
  - Darwin (macOS): arm64, x86_64
  - Linux: arm64, i386, x86_64
  - Windows: i386, x86_64
  - Checksums file

#### 6. Publish the Release

Once verified, publish the draft:

```bash
gh release edit v1.7.1 --draft=false
```

**⚠️ Once published, the release becomes IMMUTABLE**:
- Assets cannot be added, modified, or deleted
- Tags cannot be deleted or moved
- This is permanent and cannot be undone

#### 7. Verify Published Release

```bash
# Check release is published
gh release view v1.7.1 --json isDraft,publishedAt

# Test Homebrew installation (optional)
brew tap erraggy/oastools
brew install oastools
oastools --version
```

### After Release

- Announce the release (if applicable)
- Update project documentation if needed
- Monitor issue tracker for user feedback
- Check Homebrew tap repository for formula updates

## Configuration

### GoReleaser Configuration

**File**: `.goreleaser.yaml`

```yaml
release:
  # Create as draft to allow custom release notes via gh release create --draft
  # GoReleaser adds binaries to the draft, then manually publish with gh release edit
  # Compatible with GitHub's immutability setting (drafts remain mutable until published)
  draft: true
```

**Key points**:
- `draft: true` is **required** for immutability compatibility
- GoReleaser will add assets to the draft without publishing
- Manual publish step gives you control and verification time

### GitHub Actions Workflow

**File**: `.github/workflows/release.yml`

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write
  # Required for GoReleaser to push to homebrew-oastools tap

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

**Key configuration**:
- Triggers on tag push (pattern: `v*`)
- Uses `HOMEBREW_TAP_TOKEN` secret (PAT with `repo` scope)
- Requires `contents: write` permission
- Fetches full git history (`fetch-depth: 0`) for changelog generation

### GitHub Repository Settings

**Required settings for releases**:

#### 1. Enable Release Immutability

**Location**: Settings → General → Releases

- ✅ **Check** "Enable release immutability"
- Once enabled, all new published releases become immutable
- Draft releases remain mutable until published
- **Recommendation**: Keep this enabled for security best practices

#### 2. Workflow Permissions

**Location**: Settings → Actions → General → Workflow permissions

- ✅ Select "Read and write permissions"
- Required for the release workflow to create releases and attach assets
- The workflow's `permissions: contents: write` supplements this

#### 3. Branch Protection and Rulesets

**Location**: Settings → Rules

- ✅ Branch protection rules can be safely enabled
- ✅ Repository rulesets (like `main-protections`) can be enabled
- These apply to branch operations, **not tag creation**
- Release workflow triggers on tag push, which bypasses branch protection

### Personal Access Token (PAT) Setup

**Required**: A GitHub Personal Access Token for pushing to the Homebrew tap repository.

#### Why a PAT is Needed

The default `GITHUB_TOKEN` provided by GitHub Actions only has permissions for the current repository. It **cannot** push to the separate `homebrew-oastools` repository. You must create a PAT with `repo` scope.

#### Creating the PAT

1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Set name: "GoReleaser - oastools Homebrew Publishing"
4. Select scopes: **`repo`** (full control of private repositories)
5. Click "Generate token" and **copy immediately** (won't be shown again)

#### Adding to Repository

1. Go to oastools repository → Settings → Secrets and variables → Actions
2. Click "New repository secret"
3. Name: `HOMEBREW_TAP_TOKEN`
4. Value: Paste the token
5. Click "Add secret"

#### Verification

```bash
gh secret list --repo erraggy/oastools
```

Should show `HOMEBREW_TAP_TOKEN` in the list.

## Testing and Verification

### Pre-Release Checklist

Before creating your first release, ensure:

- [ ] `HOMEBREW_TAP_TOKEN` secret is created and added to repository secrets
- [ ] Secret verified with: `gh secret list --repo erraggy/oastools`
- [ ] Local test successful: `make release-test`
- [ ] Commit author email in `.goreleaser.yaml` matches a verified email in your GitHub account
- [ ] All changes committed and pushed to `main`
- [ ] All tests pass: `make check`
- [ ] "Enable release immutability" setting is enabled in repository settings

### Testing the Complete Workflow

#### 1. Create Test Draft Release

```bash
gh release create v1.9.9-test --draft \
  --title "v1.9.9-test - Test Release" \
  --notes "Test of draft workflow with immutability enabled"
```

#### 2. Verify Workflow Triggered

```bash
# Watch the workflow
gh run watch

# Or check workflow status
gh run list --workflow=release.yml --limit 1
```

#### 3. Verify Draft Has Assets

```bash
gh release view v1.9.9-test --json isDraft,assets
```

Expected output:
- `isDraft: true`
- 8 assets (binaries for all platforms + checksums)

#### 4. Publish the Draft

```bash
gh release edit v1.9.9-test --draft=false
```

#### 5. Test Immutability

**Attempt to delete an asset** (should fail):

```bash
gh release delete-asset v1.9.9-test oastools_Windows_i386.zip --yes
```

Expected error:
```
HTTP 422: Validation Failed
Cannot delete asset from an immutable release
```

**Attempt to delete the tag** (should fail):

```bash
git push origin :refs/tags/v1.9.9-test
```

Expected error:
```
remote: error: GH013: Repository rule violations found for refs/tags/v1.9.9-test.
remote: - Cannot delete this tag
```

#### 6. Clean Up Test Release

```bash
# Delete the entire release (this is allowed)
gh release delete v1.9.9-test --yes

# Delete the tag (allowed after release is deleted)
git push origin :refs/tags/v1.9.9-test
git tag -d v1.9.9-test
```

### Verification Results

From actual testing (November 2024):

✅ **Draft creation**: Works correctly
✅ **GoReleaser asset upload**: Successfully adds 8 assets to draft
✅ **Custom notes**: Preserved exactly as written
✅ **Publishing**: Draft converts to published release
✅ **Asset deletion blocked**: "Cannot delete asset from an immutable release"
✅ **Tag deletion blocked**: "Cannot delete this tag"
✅ **Release deletion allowed**: Entire immutable releases can be deleted

## Troubleshooting

### Common Issues and Solutions

#### Issue: "Cannot modify immutable release"

**Cause**: Trying to add assets to a published release with immutability enabled.

**Solution**:
1. Ensure `.goreleaser.yaml` has `draft: true`
2. Use the draft workflow (create draft → add assets → publish)
3. Never try to add assets after publishing

#### Issue: "Not Found (HTTP 404)" when viewing release

**Cause**: Using `gh release create --draft` without proper tag creation.

**Solution**:
- The `gh release create` command should automatically create the tag
- Verify tag exists: `git tag -l v1.x.x`
- If tag doesn't exist, create and push it: `git tag v1.x.x && git push origin v1.x.x`

#### Issue: Workflow doesn't trigger

**Cause**: Tag wasn't pushed to remote repository.

**Solution**:
```bash
# Verify tag exists locally
git tag -l v1.x.x

# Push tag to trigger workflow
git push origin v1.x.x
```

#### Issue: GoReleaser can't push to homebrew-oastools

**Possible causes**:
1. `HOMEBREW_TAP_TOKEN` secret not configured
2. PAT doesn't have `repo` scope
3. PAT expired or revoked
4. Commit author email doesn't match verified GitHub email

**Solution**:
```bash
# Verify secret exists
gh secret list --repo erraggy/oastools

# Check .goreleaser.yaml commit_author.email matches your verified GitHub email
# Regenerate PAT if needed
```

#### Issue: Draft release shows as "untagged-*"

**Cause**: This is normal behavior for draft releases without an associated tag.

**Solution**:
- Draft releases may show `untagged-*` identifier until published
- This is expected GitHub behavior
- Tag will be properly associated when the draft is published

### Getting Help

If you encounter issues:

1. **Check workflow logs**: `gh run view --log-failed`
2. **Verify configuration**: Review `.goreleaser.yaml` and workflow file
3. **Test locally**: Run `make release-test` to verify builds
4. **Consult documentation**: See [Technical References](#technical-references) below

## Technical References

### Official Documentation

**GitHub**:
- [Immutable Releases Documentation](https://docs.github.com/en/code-security/supply-chain-security/understanding-your-software-supply-chain/immutable-releases)
- [Managing Releases in a Repository](https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository)
- [GitHub REST API - Releases](https://docs.github.com/rest/releases/releases)

**GoReleaser**:
- [Release Configuration](https://goreleaser.com/customization/release/)
- [Homebrew Casks Documentation](https://goreleaser.com/customization/homebrew_casks/)
- [GitHub Actions Integration](https://goreleaser.com/ci/actions/)
- [Deprecation Notice: brews → homebrew_casks](https://goreleaser.com/deprecations#brews)

**Semantic Versioning**:
- [Semantic Versioning 2.0.0](https://semver.org/)

### GitHub Changelog Announcements

- [Immutable Releases - Public Preview (August 2024)](https://github.blog/changelog/2025-08-26-releases-now-support-immutability-in-public-preview/)
- [Immutable Releases - Generally Available (October 2024)](https://github.blog/changelog/2025-10-28-immutable-releases-are-now-generally-available/)

### GitHub Community Discussions

- [🎉 Immutable Releases: Public Preview is Here!](https://github.com/orgs/community/discussions/171210)
- [🚀 Immutable Releases Are Now Generally Available!](https://github.com/orgs/community/discussions/178351)

### GoReleaser Community

- [Keep existing release notes · Issue #929](https://github.com/goreleaser/goreleaser/issues/929)
- [Upload artifacts to existing release · Discussion #4524](https://github.com/orgs/goreleaser/discussions/4524)
- [Disabling GitHub Release + Publish · Discussion #2570](https://github.com/orgs/goreleaser/discussions/2570)

### Related Tools and Actions

- [softprops/action-gh-release](https://github.com/softprops/action-gh-release) - GitHub Action for creating releases
  - [Issue #641: Update release workflow for compatibility with Immutable Releases](https://github.com/softprops/action-gh-release/issues/641)

### Articles and Guides

- [WebProNews: GitHub Launches Immutable Releases for Supply Chain Security](https://www.webpronews.com/github-launches-immutable-releases-for-supply-chain-security/)
- [Automated GitHub Releases with GoReleaser](https://www.mslinn.com/golang/3000-go-github-release.html)

### oastools-Specific Documentation

- [CLAUDE.md](./CLAUDE.md) - Complete project documentation including release process
- [planning/homebrew-cask-migration.md](./planning/homebrew-cask-migration.md) - Homebrew Cask migration notes

## Evolution of the Release Process

### Historical Context

This section documents the evolution of our release process to provide context for current decisions.

#### v1.9.4 and Earlier: Formula-based Releases

- Used `brews` configuration in GoReleaser (deprecated)
- Generated Homebrew Formulas in `Formula/oastools.rb`
- Pre-compiled binaries disguised as formulas ("hackyish" approach)

#### v1.9.5: Migration to Casks

- Migrated to `homebrew_casks` configuration (GoReleaser v2.10+)
- Modern approach for distributing pre-compiled binaries
- Disabled old Formula with `disable!` directive to guide users to Cask

#### v1.9.6-v1.9.7: Exploring Immutability

- Initial attempts with `mode: replace` to handle existing releases
- Encountered "immutable release" errors
- Learned about GitHub's new immutability feature

#### v1.9.8: Draft Releases Attempt

- Switched to `draft: true` in GoReleaser
- Allowed asset uploads before publishing
- Still required manual release creation via `gh release create`

#### Current (v1.9.9+): Immutability-Compatible Workflow

- Configuration: `draft: true` in GoReleaser
- Workflow: Create draft → GoReleaser adds assets → Publish manually
- Fully compatible with GitHub's immutability setting
- Preserves hand-crafted release notes
- Balances automation with control and security

### Lessons Learned

1. **Draft releases are the key**: Only published releases are immutable; drafts remain mutable
2. **Two-step publishing is necessary**: Trying to publish immediately conflicts with immutability
3. **Tag creation triggers workflows**: `gh release create` creates both release and tag
4. **Asset modification is strictly blocked**: Once published, no changes allowed
5. **PAT scope matters**: Default `GITHUB_TOKEN` can't push to other repositories
6. **Immutability enhances security**: Prevents supply chain attacks via release tampering

---

**Document Version**: 1.0
**Last Updated**: 2024-11-24
**Tested with**: GoReleaser v2.x, GitHub Release Immutability (GA)
