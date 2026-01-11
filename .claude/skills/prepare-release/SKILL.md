---
name: prepare-release
description: Prepare a new release by running comprehensive pre-release checks, CI benchmarks, and publishing. Use when creating a new version release (e.g., "prepare release v1.44.0"). Coordinates DevOps, Architect, Maintainer, and Developer agents.
---

# prepare-release

Prepare a new release by running comprehensive pre-release checks and updates.

**Usage:** `/prepare-release <version>` (e.g., `/prepare-release v1.26.0`)

## Process

You are the DevOps Engineer coordinating a release. Execute the following steps:

### Phase 1: Launch Background Agents

Launch these agents **in background mode** (`run_in_background: true`) to run concurrently:

1. **DevOps Engineer** - Pre-release validation:
   - Check commits since last release tag
   - Run `make bench-quick` for quick local regression check (~2 min)
   - Create feature branch `chore/<version>-release-prep`

2. **Architect** - Review documentation:
   - Check if CLAUDE.md needs updates for new features
   - Check if README.md needs updates
     - Be sure to verify accuracy of all details stated. We mention things like number of packages, dependencies, etc.
   - Check if any `doc.go` files need updates
   - Check if `docs/developer-guide.md` needs updates

3. **Maintainer** - Code quality review:
   - Run `make check` to ensure all tests pass
   - Run `govulncheck` for security vulnerabilities
   - Verify gopls diagnostics are clean
   - Review new code for consistency and error handling

4. **Developer** - Check godoc examples:
   - Verify all new features have runnable examples in `example_test.go`
   - Add any missing examples
   - Ensure examples compile and pass

### Phase 2: Monitor Progress & Act Incrementally

**IMPORTANT:** Do NOT wait for all agents to complete before acting. Instead:

1. **Poll agents periodically** using `TaskOutput` with `block: false` to check status
2. **Report progress** to the user as each agent completes (use a status table)
3. **Act immediately** on completed agent results:
   - If Maintainer finds issues → fix them while other agents run
   - If DevOps completes benchmarks → report benchmark deltas
   - If Architect finds doc gaps → start fixing while waiting for others
   - If Developer finds missing examples → add them incrementally

4. **Status update format** (update after each check):
   ```
   | Agent | Status | Key Findings |
   |-------|--------|--------------|
   | DevOps | ✅ Done | Quick benchmarks clean, no regressions |
   | Architect | 🔄 Running | - |
   | Maintainer | ✅ Done | All tests pass, no vulns |
   | Developer | ✅ Done | 2 examples added |
   ```

5. **Quick benchmark check**: If `make bench-quick` shows regressions:
   - Flag prominently ⚠️ and investigate before proceeding

### Phase 3: Consolidate & Fix

After all agents complete:
1. Final summary table of all findings
2. List any remaining required changes
3. Apply fixes that couldn't be done incrementally
4. Commit all changes to the pre-release branch

### Phase 4: Trigger CI Benchmarks

After all code changes are committed to the pre-release branch:

1. **Push the branch** to origin:
   ```bash
   git push -u origin chore/<version>-release-prep
   ```

2. **Trigger the benchmark workflow** on the pre-release branch:
   ```bash
   gh workflow run benchmark.yml \
     -f version="<version>" \
     -f ref="chore/<version>-release-prep" \
     -f output_mode=commit
   ```

3. **Wait for completion** (~5 min):
   ```bash
   # Wait for run to appear
   sleep 15
   RUN_ID=$(gh run list --workflow=benchmark.yml --limit=1 --json databaseId -q '.[0].databaseId')
   gh run watch "$RUN_ID" --exit-status
   ```

4. **Pull the benchmark commit**:
   ```bash
   git pull origin chore/<version>-release-prep
   ```

The benchmark file is now included in the pre-release branch.

### Phase 5: Create Pre-Release PR

1. Verify the benchmark file exists: `ls benchmarks/benchmark-<version>.txt`
2. Push any additional changes if needed
3. Create PR with message: `chore: prepare <version> release`
4. Wait for CI checks to pass
5. Merge PR: `gh pr merge --squash --admin`

### Phase 6: Generate Release Notes

Generate comprehensive release notes that include PRs and Issues:

#### Step 1: Get GitHub's Auto-Generated Notes

```bash
# Get auto-generated release notes from GitHub API
PREV_TAG=$(git describe --tags --abbrev=0 HEAD^)
gh api repos/erraggy/oastools/releases/generate-notes \
  -f tag_name="<version>" \
  -f previous_tag_name="$PREV_TAG" \
  --jq '.body'
```

This produces PR links like:
```
* feat(api): add ToParseResult() by @erraggy in https://github.com/erraggy/oastools/pull/235
```

#### Step 2: Extract Linked Issues

```bash
# Get Issues fixed by PRs since last release
LAST_TAG_DATE=$(git log -1 --format=%cI "$PREV_TAG")
gh pr list --state merged --base main --limit 50 \
  --json number,title,mergedAt,closingIssuesReferences | \
  jq -r --arg since "$LAST_TAG_DATE" '
    [.[] | select(.mergedAt > $since and (.closingIssuesReferences | length > 0))] |
    if length == 0 then "None"
    else .[] | "- #\(.closingIssuesReferences[0].number) - Fixed by PR #\(.number)"
    end'
```

#### Step 3: Combine into Final Release Notes

Use this enhanced structure:

```markdown
## What's Changed

<!-- Copy the auto-generated PR list from Step 1 -->

## Issues Fixed

<!-- List issues from Step 2, or "None" if no linked issues -->
- #233 - Fixed by PR #234
- #227 - Fixed by PR #228

## Highlights

### Features
- Brief description of major new features

### Bug Fixes
- Brief description of significant fixes

### Performance
- Any performance improvements (reference benchmark data if available)

## Breaking Changes
- List any breaking changes (or "None" if backward compatible)

## Upgrade Notes
- Any notes for users upgrading from previous version

**Full Changelog**: https://github.com/erraggy/oastools/compare/<prev_version>...<version>
```

**Important:**
- Do NOT wrap PR numbers, issue numbers, or commit hashes in backticks - GitHub auto-links them
- Always include the "Full Changelog" link for complete diff view

### Phase 7: Tag and Publish

> ⚠️ **CRITICAL:** Use the publish script. Do NOT run release commands manually.
>
> This script exists because Claude cannot be trusted to follow skill instructions
> in fresh sessions (Claude Code v2.1.3+ bug). The script is deterministic and
> will fail-fast if anything goes wrong.

**Run the publish script:**
```bash
.claude/scripts/publish-release.sh <version>
```

The script handles all release steps:
1. Verifies on main branch
2. Creates and pushes annotated tag
3. Waits for goreleaser workflow
4. Verifies draft has 8 assets
5. Publishes with `gh release edit --draft=false`
6. Verifies published release

Do NOT improvise or run individual commands. The script enforces the correct process.

## Important Notes

- Always run on `main` branch (after merging any prep changes)
- Use `--admin` flag for PR merge if branch protection blocks
- CI benchmarks run on the pre-release branch and are included in the PR
- No separate benchmark PR needed after tagging
- Document all new public API in CLAUDE.md
- **ALWAYS use `.claude/scripts/publish-release.sh`** for Phase 7 - never run release commands manually
