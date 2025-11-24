# Release Process Issues and Solutions

## Current Problem (2025-11-24)

### What Happened - v1.9.11 Release Attempt

**Goal**: Release v1.9.11 to fix Homebrew code signing issues by reverting from Cask to Formula

**Actions Taken**:
1. ✅ Updated `.goreleaser.yaml` to use `brews` instead of `homebrew_casks`
2. ✅ Tested locally with `make release-test` - worked perfectly
3. ✅ Committed and pushed changes
4. ✅ Created draft release using:
   ```bash
   gh release create v1.9.11 --draft \
     --title "v1.9.11 - Fix Homebrew installation code signing errors" \
     --notes "..."
   ```
5. ❌ **MISTAKE**: Published the draft release with `gh release edit v1.9.11 --draft=false`
6. ❌ **RESULT**: Tag was created, workflow triggered, but release was already published (immutable)
7. ❌ GoReleaser tried to upload assets to immutable release → 422 errors

**Error Messages**:
```
upload failed: POST https://uploads.github.com/repos/erraggy/oastools/releases/264694648/assets
422 Cannot upload assets to an immutable release.
```

### Root Cause Analysis

**The Fundamental Misunderstanding**:

With "Enable release immutability" enabled on GitHub, the workflow is:

1. `gh release create v1.9.11 --draft` creates a draft release **and creates the tag**
2. Creating the tag triggers the `release.yml` workflow (triggers on `push: tags: v*`)
3. GoReleaser runs and tries to upload assets to the release
4. `.goreleaser.yaml` has `draft: true`, which tells GoReleaser to:
   - Find the existing draft release at tag v1.9.11
   - Upload assets to that draft release
5. Once GoReleaser completes, manually publish with `gh release edit v1.9.11 --draft=false`

**What I Did Wrong**:
- I published the draft (`--draft=false`) BEFORE the workflow completed
- This made the release immutable immediately
- When GoReleaser tried to upload assets, it failed because the release was already published

**What I Thought Was Supposed to Happen**:
- I thought publishing the draft would CREATE the tag and trigger the workflow
- This was wrong - the tag is created when you run `gh release create --draft`

### Current State

- Release v1.9.11 exists but has no assets (deleted)
- Tag v1.9.11 was deleted
- Repository has tag protection rules that prevent manual tag creation
- We're back to square one

## The Correct Release Process

### Prerequisites

1. Ensure you're on `main` branch, up-to-date with origin
2. Run `make check` to ensure all tests pass
3. Update benchmarks if needed (skip for bug fixes)
4. Review changes since last release

### Step-by-Step Process

#### Step 1: Create Draft Release (Creates Tag)

```bash
gh release create v1.9.11 --draft \
  --title "v1.9.11 - Brief description" \
  --notes "$(cat <<'EOF'
## Summary
...
EOF
)"
```

**What happens**:
- ✅ Draft release is created on GitHub
- ✅ Tag `v1.9.11` is created and pushed to GitHub
- ✅ This triggers the `release.yml` workflow (because tag was pushed)
- ✅ Draft release remains mutable (compatible with immutability setting)

#### Step 2: Wait for Workflow to Complete

```bash
# Watch the workflow
gh run list --workflow=release.yml --limit=1

# Monitor progress
gh run watch <RUN_ID>
```

**What happens**:
- GoReleaser builds binaries for all platforms
- `.goreleaser.yaml` has `draft: true`, so GoReleaser:
  - Finds the existing draft release at tag v1.9.11
  - Uploads all binary assets to the **draft** release
  - Pushes the Formula to homebrew-oastools repository
- Draft release remains mutable, so assets can be added

#### Step 3: Verify Draft Release Has Assets

```bash
gh release view v1.9.11 --json assets,isDraft
```

Confirm:
- `isDraft: true`
- All platform binaries are present (Darwin arm64/x86_64, Linux arm64/x86_64/i386, Windows x86_64/i386)
- Checksums file is present
- Formula was pushed to homebrew-oastools

#### Step 4: Publish the Release

**ONLY AFTER WORKFLOW COMPLETES AND ASSETS ARE VERIFIED**:

```bash
gh release edit v1.9.11 --draft=false
```

**What happens**:
- Release becomes published
- With immutability enabled, release becomes immutable
- Users can now see the release

#### Step 5: Verify Installation

```bash
# Test Homebrew installation
brew tap erraggy/oastools
brew install oastools
oastools --version
```

## Common Mistakes

### ❌ Mistake 1: Publishing Draft Too Early
**What**: Running `gh release edit --draft=false` before workflow completes
**Result**: Release becomes immutable before assets are uploaded → 422 errors
**Solution**: Wait for workflow to complete, verify assets, THEN publish

### ❌ Mistake 2: Trying to Manually Push Tags
**What**: Running `git tag v1.9.11 && git push origin v1.9.11`
**Result**: Repository rules prevent direct tag creation
**Solution**: Use `gh release create --draft` which is allowed to create tags

### ❌ Mistake 3: Thinking Tag is Created on Publish
**What**: Believing the tag doesn't exist until release is published
**Reality**: Tag is created when you run `gh release create --draft`
**Solution**: Understand that draft creation = tag creation = workflow trigger

## Recovery from Failed Release

If a release fails with 422 errors:

1. **Delete the release**:
   ```bash
   gh release delete v1.9.11 --yes
   ```

2. **Delete the tag** (if immutability is disabled temporarily):
   ```bash
   git push origin :refs/tags/v1.9.11
   ```

3. **Clean up local tags**:
   ```bash
   git tag -d v1.9.11
   git fetch --tags
   ```

4. **Start over** with Step 1 of the correct process

## Repository Settings

### Required Settings for This Process

1. **Enable release immutability** (Settings → General → Releases)
   - ✅ Recommended for security
   - Requires the workflow above
   - Draft releases remain mutable

2. **Workflow permissions** (Settings → Actions → General)
   - ✅ "Read and write permissions" required
   - Allows workflow to create releases and upload assets

3. **Tag protection rules**
   - ✅ Can be enabled
   - Prevents manual tag creation
   - `gh release create` is allowed through this restriction

### Alternative: Disable Immutability Temporarily

If issues occur, immutability can be disabled temporarily to fix state:

1. Settings → General → Releases
2. Uncheck "Enable release immutability"
3. Fix the broken release/tag state
4. Re-enable immutability
5. Proceed with corrected process

## GoReleaser Configuration

### Critical `.goreleaser.yaml` Setting

```yaml
release:
  # CRITICAL: Must be true for immutable releases
  draft: true
```

**Why this matters**:
- With `draft: true`, GoReleaser uploads to existing draft release
- With `draft: false`, GoReleaser would try to create/publish release
- For immutable releases, we need draft to remain mutable during upload

### Brews vs Casks

```yaml
# ✅ Correct for CLI tools (no code signing required)
brews:
  - name: oastools
    repository:
      owner: erraggy
      name: homebrew-oastools

# ❌ Wrong for CLI tools (requires Apple Developer code signing)
homebrew_casks:
  - name: oastools
    # Casks need notarization
```

## Resolution (2025-11-24)

### What Actually Worked

With release immutability **disabled temporarily**:

1. ✅ Deleted broken v1.9.11 draft release
2. ✅ Deleted local v1.9.11 tag
3. ✅ Created v1.9.12 release (published, not draft) using:
   ```bash
   gh release create v1.9.12 --title "..." --notes "..."
   ```
4. ✅ Tag was created and pushed automatically
5. ✅ Workflow triggered and completed successfully
6. ✅ All binaries uploaded to release
7. ✅ Formula pushed to homebrew-oastools

**Key insight**: With immutability disabled, creating a published release (not draft) works because GoReleaser can update it. The `draft: true` in `.goreleaser.yaml` doesn't prevent this when immutability is disabled.

### Remaining Tasks

1. Re-enable release immutability
2. Delete the old Cask from homebrew-oastools (Casks/oastools.rb)
3. Update the Formula to disable or note that Cask is deprecated
4. Document the **correct** process for future releases with immutability enabled

## Questions to Resolve

1. **Does `gh release create --draft` actually create and push the tag?**
   - Need to verify this behavior
   - Test by creating a draft and checking if workflow triggers

2. **What's the exact timing of tag creation?**
   - When draft is created? Or when draft is published?
   - This determines when workflow triggers

3. **Can we see workflow logs for v1.9.11 failure?**
   - Already reviewed - confirmed 422 errors
   - Release was immutable when GoReleaser tried to upload

## References

- CLAUDE.md "Creating a New Release" section
- `.goreleaser.yaml` configuration
- GitHub Actions `release.yml` workflow
- Previous planning doc: `planning/homebrew-cask-migration.md` (deleted after completion)
