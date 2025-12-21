# Implementation Plan: Auto-Update GitHub Pages Version Display

## Problem Summary

The GitHub Pages site at https://erraggy.github.io/oastools/ displays the repository release version in the header (via MkDocs Material's built-in GitHub integration). This version is stuck at v1.28.3 even though newer releases (v1.30.0) have been published.

### Root Cause

MkDocs Material fetches the **latest published release** from GitHub's API (`GET /repos/{owner}/{repo}/releases/latest`) and displays it in the header. However, the docs workflow (`.github/workflows/docs.yml`) only triggers on:
- Push to `main` branch
- Manual workflow dispatch

It does **not** trigger when a release is published. This means:
1. You publish v1.29.0, v1.30.0, etc.
2. No changes are pushed to `main`
3. Docs workflow never runs
4. GitHub Pages site is not redeployed
5. Version display remains stale

**Note**: The version is fetched client-side by JavaScript in MkDocs Material, so even though the API returns the correct version, browser/CDN caching may cause staleness. Redeploying the docs helps ensure fresh content.

---

## Solution

Add a `release: published` trigger to the docs workflow so it automatically redeploys when releases are published.

---

## Implementation Steps

### Step 1: Update `.github/workflows/docs.yml`

Add the `release` event trigger with `types: [published]`:

```yaml
name: Publish Docs

on:
  push:
    branches:
      - main
  release:
    types: [published]
  workflow_dispatch:

permissions:
  contents: write

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6

      - name: Configure Git Credentials
        run: |
          git config user.name github-actions[bot]
          git config user.email 41898282+github-actions[bot]@users.noreply.github.com

      - name: Setup Python
        uses: actions/setup-python@v5
        with:
          python-version: 3.x

      - name: Install MkDocs and Material Theme
        run: pip install mkdocs-material

      - name: Make prepare-docs executable
        run: chmod +x scripts/prepare-docs.sh

      - name: Prepare Documentation
        run: ./scripts/prepare-docs.sh

      - name: Deploy to GitHub Pages
        run: mkdocs gh-deploy --force
```

### Step 2: Verify the Change

After merging this change:

1. The next release you publish will automatically trigger a docs deployment
2. The docs site will be rebuilt and deployed to GitHub Pages
3. The version in the header will update to reflect the latest published release

---

## Why This Works

1. **Release Event Trigger**: The `release: published` event fires when you run `gh release edit v1.X.Y --draft=false` (your current final publish step)

2. **MkDocs Material Version Fetch**: The theme makes a client-side request to GitHub's API for the latest release. By redeploying after each release, we ensure:
   - The site reflects the latest content
   - Any edge caching is invalidated
   - The JavaScript that fetches the version runs against fresh content

3. **No Breaking Changes**: This is purely additive—existing triggers (`push` to `main`, `workflow_dispatch`) continue to work

---

## Testing the Fix

After deploying this change:

1. Publish your next release as usual:
   ```bash
   gh release edit v1.31.0 --draft=false
   ```

2. Watch for the docs workflow to trigger:
   ```bash
   gh run list --workflow=docs.yml --limit=3
   ```

3. After the workflow completes, visit https://erraggy.github.io/oastools/ and verify the version in the header shows the new release

---

## Immediate Fix (Before Merging)

To update the site immediately (before this workflow change is merged), manually trigger the docs workflow:

```bash
gh workflow run docs.yml
```

Or via the GitHub UI: Actions → Publish Docs → Run workflow

---

## Files to Modify

| File | Change |
|------|--------|
| `.github/workflows/docs.yml` | Add `release: types: [published]` trigger |

---

## Execution Commands for Claude Code

```bash
# 1. Edit the docs workflow file
# File: .github/workflows/docs.yml
# Add the release trigger after the push trigger

# 2. Commit the change
git add .github/workflows/docs.yml
git commit -m "ci(docs): trigger docs deployment on release publish

Add release:published event trigger to docs workflow so the GitHub Pages
site automatically redeploys when a release is published. This ensures
the version display in the header stays current with the latest release.

Fixes stale version display (was stuck at v1.28.3 despite newer releases)."

# 3. Push to main (or create PR if preferred)
git push origin main

# 4. Optionally, trigger immediate redeploy to fix the current stale version
gh workflow run docs.yml
```

---

## Success Criteria

- [ ] `.github/workflows/docs.yml` includes `release: types: [published]` trigger
- [ ] Next release publish triggers automatic docs deployment
- [ ] Version in site header reflects latest published release
- [ ] Existing docs workflow triggers (push to main, manual dispatch) still work

---

## Estimated Time

5-10 minutes to implement and verify.
