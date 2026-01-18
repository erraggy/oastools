# Prepare-Release Skill Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the redesigned `/prepare-release` and `/publish-release` skills with bash scripts for deterministic execution.

**Architecture:** Two skills (`prepare-release`, `publish-release`) backed by two scripts (`prepare-release.sh`, `publish-release.sh`). Agent coordination for judgment phases 1-3, scripts for mechanical phases 4-7.

**Tech Stack:** Bash scripts, Markdown skill files, GitHub CLI (`gh`)

**Design Doc:** `docs/plans/2026-01-18-prepare-release-skill-design.md`

---

## Task 1: Create `prepare-release.sh` Script

**Files:**
- Create: `.claude/scripts/prepare-release.sh`

**Step 1: Create the script file with header and validation**

```bash
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
```

**Step 2: Run shellcheck to verify syntax**

Run: `shellcheck .claude/scripts/prepare-release.sh`
Expected: No errors (may have info-level suggestions)

**Step 3: Add Phase 4 - CI Benchmarks**

Append to `.claude/scripts/prepare-release.sh`:

```bash
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
        -f version="$VERSION" \
        -f ref="$BRANCH" \
        -f output_mode=commit
    echo "  ✓ Workflow triggered"
    echo ""

    echo "Step 4.3: Waiting for benchmark workflow..."
    sleep 15  # Wait for run to appear
    RUN_ID=$(gh run list --workflow=benchmark.yml --limit=1 --json databaseId -q '.[0].databaseId')
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
```

**Step 4: Run shellcheck again**

Run: `shellcheck .claude/scripts/prepare-release.sh`
Expected: No errors

**Step 5: Add Phase 5 - Create Pre-Release PR**

Append to `.claude/scripts/prepare-release.sh`:

```bash
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
EXISTING_PR=$(gh pr list --head "$BRANCH" --json number -q '.[0].number // empty')

if [[ -n "$EXISTING_PR" ]]; then
    echo "  PR #$EXISTING_PR already exists"
    PR_NUMBER="$EXISTING_PR"
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

# Switch to main and pull
echo "Step 5.5: Switching to main..."
git checkout main
git pull origin main
echo "  ✓ On main with latest changes"
echo ""
```

**Step 6: Run shellcheck again**

Run: `shellcheck .claude/scripts/prepare-release.sh`
Expected: No errors

**Step 7: Add Phase 6 - Generate Release Notes**

Append to `.claude/scripts/prepare-release.sh`:

```bash
# =============================================================================
# Phase 6: Generate Release Notes
# =============================================================================

echo "=== Phase 6: Generate Release Notes ==="

# Step 6.1: Generate notes via GitHub API
echo "Step 6.1: Generating release notes..."
PREV_TAG=$(git describe --tags --abbrev=0 HEAD^)
echo "  Previous tag: $PREV_TAG"

NOTES_FILE="/tmp/release-notes-${VERSION}.md"

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
```

**Step 8: Make script executable**

Run: `chmod +x .claude/scripts/prepare-release.sh`

**Step 9: Final shellcheck**

Run: `shellcheck .claude/scripts/prepare-release.sh`
Expected: No errors

**Step 10: Commit**

```bash
git add .claude/scripts/prepare-release.sh
git commit -m "feat(release): add prepare-release.sh script

Deterministic script for phases 4-6 of release preparation:
- Phase 4: Push branch, trigger CI benchmarks, wait for completion
- Phase 5: Create PR, wait for CI, merge with --admin
- Phase 6: Generate release notes via GitHub API

Includes idempotency checks and --skip-benchmarks flag for re-runs.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Create `publish-release` Skill

**Files:**
- Create: `.claude/skills/publish-release/SKILL.md`

**Step 1: Create the skill directory**

Run: `mkdir -p .claude/skills/publish-release`

**Step 2: Write the skill file**

Create `.claude/skills/publish-release/SKILL.md`:

```markdown
---
name: publish-release
description: Publish a prepared release (phase 7). Usage: /publish-release <version>. Requires version argument. Wraps publish-release.sh for deterministic execution.
---

# publish-release

Publish a prepared release to GitHub. This is phase 7 of the release process.

**Usage:** `/publish-release <version>` (e.g., `/publish-release v1.46.0`)

## Prerequisites

Before running this skill:
1. Run `/prepare-release <version>` to complete phases 1-6
2. Review the generated release notes
3. Ensure you're ready to publish (this is irreversible)

## Process

> ⚠️ **CRITICAL:** This skill wraps `publish-release.sh`. Do NOT run release commands manually.

### Step 1: Validate Version Argument

If no version is provided, **stop and ask the user**:
```
Error: Version argument required.
Usage: /publish-release <version>
Example: /publish-release v1.46.0
```

### Step 2: Confirm with User

Before proceeding, confirm:
```
Ready to publish release **<version>**?

This will:
1. Create and push git tag <version>
2. Trigger goreleaser workflow
3. Wait for draft release with binaries
4. Publish the release (irreversible)

Proceed? [Yes / Cancel]
```

### Step 3: Run the Publish Script

```bash
.claude/scripts/publish-release.sh <version>
```

The script handles:
1. Verifies on main branch
2. Creates and pushes annotated tag
3. Waits for goreleaser workflow
4. Verifies draft has 8 assets (binaries + checksums)
5. Generates release notes
6. Publishes with `gh release edit --draft=false`
7. Verifies published release

### Step 4: Report Results

On success, report:
```
✅ Release <version> published successfully!

View release: https://github.com/erraggy/oastools/releases/tag/<version>

The release includes:
- 8 binary assets for all platforms
- Auto-generated release notes
- Homebrew formula will update automatically
```

On failure, report the error and suggest recovery steps.

## Important Notes

- Version argument is **required** (no inference - you're publishing what you prepared)
- Always confirm before publishing - releases are irreversible
- If the script fails partway, check the error message for recovery steps
- **NEVER** use `gh release create` - goreleaser creates the draft
```

**Step 3: Verify file exists and is readable**

Run: `head -20 .claude/skills/publish-release/SKILL.md`
Expected: Shows the frontmatter and title

**Step 4: Commit**

```bash
git add .claude/skills/publish-release/SKILL.md
git commit -m "feat(release): add publish-release skill

Standalone skill for phase 7 (tag and publish).
Wraps publish-release.sh with confirmation prompt.
Separated from prepare-release for explicit human checkpoint.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Update `prepare-release` Skill

**Files:**
- Modify: `.claude/skills/prepare-release/SKILL.md` (complete rewrite)

**Step 1: Rewrite the skill file**

Replace entire contents of `.claude/skills/prepare-release/SKILL.md`:

```markdown
---
name: prepare-release
description: Prepare a release (phases 1-6). Usage: /prepare-release [version]. If version omitted, infers from conventional commits. Coordinates agents for review, then runs prepare-release.sh for deterministic steps.
---

# prepare-release

Prepare a new release by running comprehensive pre-release checks and updates.

**Usage:**
- `/prepare-release <version>` - Use specified version (e.g., `/prepare-release v1.46.0`)
- `/prepare-release` - Infer version from conventional commits

## Version Inference

If no version is provided, analyze commits since the last tag:

### Step 1: Get Commits Since Last Release

```bash
LAST_TAG=$(git describe --tags --abbrev=0)
echo "Last release: $LAST_TAG"
git log "$LAST_TAG"..HEAD --oneline
```

### Step 2: Determine Version Bump

Parse conventional commit prefixes:
- Any `BREAKING CHANGE:` or `!:` in commit message → **MAJOR** bump
- Any `feat:` or `feat(scope):` → **MINOR** bump
- Only `fix:`, `chore:`, `docs:`, `refactor:`, `test:`, `perf:` → **PATCH** bump

### Step 3: Propose or Prompt

**If clear signal:** Propose the version with explanation:
```
Analyzing commits since v1.45.2...

Found 4 commits:
- feat(parser): add streaming support
- fix(validator): handle empty schemas
- chore: update dependencies
- docs: improve examples

Recommendation: **v1.46.0** (minor bump due to new feature)

Proceed with v1.46.0? [Yes / Different version]
```

**If ambiguous:** Prompt user to choose:
```
Analyzing commits since v1.45.2...

Found 3 commits with unclear versioning signal:
- refactor(core): reorganize internal structure
- perf: optimize memory usage
- chore: update dependencies

What version bump is appropriate?
- v1.45.3 (patch) - Bug fixes and minor improvements
- v1.46.0 (minor) - Notable improvements worth a minor bump
- Other - Specify a different version
```

---

## Process

You are the DevOps Engineer coordinating a release. Execute the following phases:

### Phase 1: Launch Background Agents

Launch these agents **in background mode** (`run_in_background: true`) to run concurrently:

1. **DevOps Engineer** - Pre-release validation:
   - Check commits since last release tag
   - Run `make bench-quick` for quick local regression check (~2 min)
   - Create feature branch `chore/<version>-release-prep`

2. **Architect** - Review documentation:
   - Check if CLAUDE.md needs updates for new features
   - Check if README.md needs updates (verify accuracy of stated details)
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

### Phases 4-6: Run Preparation Script

> ⚠️ **CRITICAL:** Use the prepare script. Do NOT run these commands manually.

After all agent work is committed, run:

```bash
.claude/scripts/prepare-release.sh <version>
```

The script handles:
- **Phase 4:** Push branch, trigger CI benchmarks, wait for completion, pull results
- **Phase 5:** Create PR, wait for CI checks, merge with `--admin`, switch to main
- **Phase 6:** Generate release notes, save to temp file, display for review

If the script fails partway through, check the error message. You can re-run with:
- `--skip-benchmarks` flag if benchmarks already completed

### Phase 6.3: Prompt for Publishing

After the script completes successfully, prompt the user:

```
✅ Release preparation complete!

Version: <version>
Release notes saved to: /tmp/release-notes-<version>.md

Ready to publish <version>?
- [Yes, publish now] → Runs publish-release.sh <version>
- [Not yet] → End here (run /publish-release <version> later)
```

If user chooses "Yes", run:
```bash
.claude/scripts/publish-release.sh <version>
```

---

## Important Notes

- **Orchestration Mode**: Delegate agent tasks, don't do the work yourself
- Use `--admin` flag for PR merge if branch protection blocks
- CI benchmarks run on the pre-release branch and are included in the PR
- Document all new public API in CLAUDE.md
- **ALWAYS use scripts** for phases 4-7 - never run release commands manually
```

**Step 2: Verify the changes look correct**

Run: `head -50 .claude/skills/prepare-release/SKILL.md`
Expected: Shows new frontmatter with updated description and version inference section

**Step 3: Commit**

```bash
git add .claude/skills/prepare-release/SKILL.md
git commit -m "feat(release): update prepare-release skill with version inference

- Add version inference from conventional commits
- Simplify phases 4-6 to use prepare-release.sh script
- Remove phase 7 (moved to separate publish-release skill)
- Add phase 6.3 prompt for publishing
- Update description to reflect optional version argument

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Test the Implementation

**Files:**
- None (manual testing)

**Step 1: Verify shellcheck passes on all scripts**

Run: `shellcheck .claude/scripts/*.sh`
Expected: No errors

**Step 2: Verify skill files have valid frontmatter**

Run: `head -5 .claude/skills/*/SKILL.md`
Expected: Each shows valid YAML frontmatter with `name:` and `description:`

**Step 3: Dry-run test of prepare-release.sh validation**

Run: `.claude/scripts/prepare-release.sh`
Expected: Usage error (no version provided)

Run: `.claude/scripts/prepare-release.sh invalid`
Expected: Version format error

Run: `.claude/scripts/prepare-release.sh v1.99.0`
Expected: Branch prerequisite error (not on chore/v1.99.0-release-prep)

**Step 4: Final commit with test verification**

```bash
git add -A
git status  # Should show all changes
git log --oneline -3  # Should show the 3 commits from tasks 1-3
```

**Step 5: Push branch for review**

```bash
git push -u origin docs/prepare-release-skill-redesign
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Create prepare-release.sh | `.claude/scripts/prepare-release.sh` |
| 2 | Create publish-release skill | `.claude/skills/publish-release/SKILL.md` |
| 3 | Update prepare-release skill | `.claude/skills/prepare-release/SKILL.md` |
| 4 | Test implementation | Manual verification |

**Total commits:** 4 (1 per task, including design doc already committed)
