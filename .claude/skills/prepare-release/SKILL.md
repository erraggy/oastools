---
name: prepare-release
description: Prepare a release (phases 1-6). Usage: /prepare-release [version]. If version omitted, infers from conventional commits. Coordinates agents for review, runs prepare-release.sh, then enhances release notes with rich formatting.
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
- Any `BREAKING CHANGE:` footer or type with exclamation suffix (e.g., `feat!:`, `fix!:`) → **MAJOR** bump
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

After all agent work is committed, from the repository root run:

```bash
.claude/scripts/prepare-release.sh <version>
```

The script handles:
- **Phase 4:** Push branch, trigger CI benchmarks, wait for completion, pull results
- **Phase 5:** Create PR, wait for CI checks, merge with `--admin`, switch to main
- **Phase 6:** Generate release notes, save to temp file, display for review

If the script fails partway through, check the error message. You can re-run safely:
- `--skip-benchmarks` flag if benchmarks already completed
- The script auto-detects already-merged PRs and skips to checkout

### Phase 6.2: Enhance Release Notes

After the script completes, enhance the auto-generated release notes with richer content:

#### Step 1: Analyze the Release

1. Read the auto-generated notes at `.release/notes-<version>.md`
2. Get commits since the previous tag:
   ```bash
   PREV_TAG=$(git describe --tags --abbrev=0 HEAD^)
   git log "$PREV_TAG"..HEAD --oneline
   ```
3. For each significant commit, understand what changed by reading relevant files

#### Step 2: Reference Previous Releases

Look at 1-2 previous releases for style consistency:
```bash
gh release view --json body | jq -r '.body' | head -100
```

#### Step 3: Create Enhanced Notes

Structure the enhanced notes following this format:

```markdown
## [Emoji] [Theme]: [Headline Summary]

[1-2 sentence overview of what this release accomplishes and its impact]

### [Feature/Change Category 1]

[Context about why this matters]

- **[Benefit 1]**: [Specific detail]
- **[Benefit 2]**: [Specific detail]
- **[Benefit 3]**: [Specific detail]

[Optional: Code example if helpful]

### [Feature/Change Category 2]

[Similar structure...]

### 🔒 [User Impact Section]

- [Breaking changes if any, or "No breaking changes"]
- [API changes if any, or "No public API changes"]
- [Dependency changes if any]
- [Backward compatibility statement]

### 📊 Quality Metrics

- ✅ All tests passing ([count]+ unit tests, [other suites])
- ✅ Zero vulnerabilities (`govulncheck` clean)
- ✅ All benchmarks passing with no regressions
- ✅ Documentation verified accurate

## What's Changed

[Keep the auto-generated PR list from original notes]

**Full Changelog**: [Keep the auto-generated link]
```

#### Guidelines for Enhancement

1. **Headline emoji + theme**: Match the release character (🚀 Features, 🐛 Bug Fixes, 🔧 Infrastructure, ⚡ Performance)
2. **User-focused language**: Explain benefits, not just what changed
3. **Code examples**: Include for new features when they clarify usage
4. **Honest impact assessment**: Clearly state if there are no user-facing changes
5. **Preserve auto-generated content**: Keep the "What's Changed" PR list and changelog link at the bottom

#### Step 4: Update the Notes File

Write the enhanced notes back to `.release/notes-<version>.md`, then display them for user review.

### Phase 6.3: Prompt for Publishing

After the enhanced notes are ready, prompt the user:

```
✅ Release preparation complete!

Version: <version>
Release notes saved to: .release/notes-<version>.md

Ready to publish <version>?
- [Yes, publish now] → Runs publish-release.sh <version>
- [Not yet] → End here (run /publish-release <version> later)
```

If user chooses "Yes", from the repository root run:
```bash
.claude/scripts/publish-release.sh <version>
```

---

## Important Notes

- **Orchestration Mode**: Delegate agent tasks, don't do the work yourself
- Use `--admin` flag for PR merge if branch protection blocks
- CI benchmarks run on the pre-release branch and are included in the PR
- Document all new public API in CLAUDE.md
- **ALWAYS use scripts** for phases 4-6 - never run release commands manually
- **ALWAYS enhance release notes** (Phase 6.2) - never skip to publishing with auto-generated notes only
