# Release Process Fix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

> **⚠️ DO NOT use git worktrees.** Work directly on the current branch (`fix/release-process-skills`).

> **🤖 Orchestration Mode:** Each task should be delegated to a `developer` subagent via the Task tool. The orchestrator reviews results between tasks and only intervenes for corrections.

**Goal:** Prevent releases from shipping without prepared notes or pre-built binaries by adding enforcement at the hook, script, and directory layers.

**Architecture:** Three-layer defense: (1) a PreToolUse hook blocks raw `gh release create` commands, (2) the publish script hard-fails if prepared notes don't exist, (3) notes persist in a gitignored `.release/` directory instead of ephemeral `/tmp/`.

**Design Doc:** `docs/plans/2026-01-23-release-process-fix-design.md`

---

## Task 1: Create the `block-release-create.sh` Hook

**Files:**
- Create: `.claude/hooks/block-release-create.sh`

**Context:** PreToolUse hooks for the Bash tool receive the tool input as JSON on stdin. The JSON has a `command` field containing the bash command. The hook must parse this and reject commands containing `gh release create`. See existing hook at `.claude/hooks/check-branch.sh` for style reference — but note that hook doesn't read stdin (it checks git state directly). Our hook MUST read stdin.

**Step 1: Write the hook script**

```bash
#!/bin/bash
# Block direct use of 'gh release create' — forces use of publish-release.sh
# Used as PreToolUse hook for Bash commands
#
# stdin: JSON with tool input (has "command" field)
# exit 2 = block the tool call with message shown to agent

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.command // empty')

if echo "$COMMAND" | grep -q 'gh release create'; then
    echo "❌ BLOCKED: Direct 'gh release create' is not allowed."
    echo "   Use /publish-release <version> instead."
    echo ""
    echo "   This ensures binaries are attached and prepared notes are used."
    exit 2
fi
```

**Step 2: Make it executable**

Run: `chmod +x .claude/hooks/block-release-create.sh`

**Step 3: Verify script is valid bash**

Run: `shellcheck .claude/hooks/block-release-create.sh`
Expected: No errors (warnings about `cat` piping are acceptable)

**Step 4: Commit**

```bash
git add .claude/hooks/block-release-create.sh
git commit -m "feat(hooks): add block-release-create hook to prevent direct releases"
```

---

## Task 2: Register the Hook in `settings.json`

**Files:**
- Modify: `.claude/settings.json:14-24` (PreToolUse section)

**Context:** The hook should be in the SHARED `settings.json` (not `settings.local.json`) because this is a safety guardrail for all repo users. The existing PreToolUse section already has `check-branch.sh`. Add our new hook as a second entry in the same `hooks` array.

**Step 1: Add the hook registration**

The current PreToolUse section (lines 14-24) is:
```json
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/check-branch.sh"
          }
        ]
      }
    ]
```

Change it to:
```json
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/check-branch.sh"
          },
          {
            "type": "command",
            "command": "bash .claude/hooks/block-release-create.sh"
          }
        ]
      }
    ]
```

**Step 2: Add permission for the hook script**

In the `permissions.allow` array, add (near line 64-65 where similar hook permissions exist):
```
"Bash(.claude/hooks/block-release-create.sh :*)"
```

**Step 3: Validate JSON**

Run: `jq . .claude/settings.json > /dev/null`
Expected: Exit 0 (valid JSON)

**Step 4: Commit**

```bash
git add .claude/settings.json
git commit -m "feat(hooks): register block-release-create hook in shared settings"
```

---

## Task 3: Add `.release/` Directory to `.gitignore`

**Files:**
- Modify: `.gitignore:42-43`

**Context:** The existing `.gitignore` already has `releases/` (line 43). Our new `.release/` directory serves a similar purpose but is specifically for inter-session note handoff between prepare and publish skills. Add it near the existing `releases/` entry.

**Step 1: Add the gitignore entry**

After line 43 (`releases/`), add:
```
.release/
```

So lines 42-44 become:
```
# Local release notes (not tracked in git)
releases/
.release/
```

**Step 2: Verify the pattern works**

Run: `mkdir -p .release && touch .release/notes-v1.99.0.md && git check-ignore .release/notes-v1.99.0.md`
Expected: `.release/notes-v1.99.0.md` (confirmed ignored)

**Step 3: Clean up test file**

Run: `rm -rf .release`

**Step 4: Commit**

```bash
git add .gitignore
git commit -m "chore: add .release/ to gitignore for persistent release notes"
```

---

## Task 4: Update `prepare-release.sh` to Write to `.release/`

**Files:**
- Modify: `.claude/scripts/prepare-release.sh:200,246,259`

**Context:** Currently writes notes to `/tmp/release-notes-${VERSION}.md`. Change to `.release/notes-${VERSION}.md` with a `mkdir -p` guard. Three locations need updating.

**Step 1: Add mkdir and update NOTES_FILE path**

Change line 200:
```bash
NOTES_FILE="/tmp/release-notes-${VERSION}.md"
```
To:
```bash
mkdir -p .release
NOTES_FILE=".release/notes-${VERSION}.md"
```

**Step 2: Update output message at line 246**

Line 246 already says `echo "  ✓ Release notes saved to: $NOTES_FILE"` — this will automatically reflect the new path since it uses the variable. No change needed here.

**Step 3: Update output message at line 259**

Line 259 already says `echo "Notes:   $NOTES_FILE"` — same as above, uses the variable. No change needed.

**Step 4: Verify script syntax**

Run: `bash -n .claude/scripts/prepare-release.sh`
Expected: Exit 0 (no syntax errors)

**Step 5: Commit**

```bash
git add .claude/scripts/prepare-release.sh
git commit -m "fix(release): write prepared notes to .release/ instead of /tmp/"
```

---

## Task 5: Update `publish-release.sh` to Read Prepared Notes (Hard Fail)

**Files:**
- Modify: `.claude/scripts/publish-release.sh:85-94`

**Context:** Currently generates FRESH notes via the GitHub API (the root cause of discarded notes). Replace with logic that reads from `.release/notes-${VERSION}.md` and hard-fails if the file doesn't exist.

**Step 1: Replace Step 5 logic**

Replace lines 85-94:
```bash
# Step 5: Generate notes and publish
echo "Step 5: Publishing release..."
PREV_TAG=$(git describe --tags --abbrev=0 HEAD^)
NOTES=$(gh api repos/erraggy/oastools/releases/generate-notes \
    -f tag_name="$VERSION" \
    -f previous_tag_name="$PREV_TAG" \
    --jq '.body')

gh release edit "$VERSION" --notes "$NOTES" --draft=false
echo "✓ Release published"
```

With:
```bash
# Step 5: Apply prepared notes and publish
echo "Step 5: Publishing release..."
NOTES_FILE=".release/notes-${VERSION}.md"
if [[ ! -f "$NOTES_FILE" ]]; then
    echo "❌ Error: Prepared release notes not found: $NOTES_FILE" >&2
    echo "   Run /prepare-release $VERSION first to generate release notes." >&2
    exit 4
fi
NOTES=$(cat "$NOTES_FILE")
gh release edit "$VERSION" --notes "$NOTES" --draft=false
echo "✓ Release published with prepared notes from $NOTES_FILE"
```

**Step 2: Verify script syntax**

Run: `bash -n .claude/scripts/publish-release.sh`
Expected: Exit 0 (no syntax errors)

**Step 3: Run shellcheck**

Run: `shellcheck .claude/scripts/publish-release.sh`
Expected: No new errors from our changes (existing warnings are acceptable)

**Step 4: Commit**

```bash
git add .claude/scripts/publish-release.sh
git commit -m "fix(release): publish script reads prepared notes, hard-fails if missing"
```

---

## Task 6: Update `prepare-release` Skill Paths

**Files:**
- Modify: `.claude/skills/prepare-release/SKILL.md:155,224,234`

**Context:** Three references to `/tmp/release-notes-<version>.md` need updating to `.release/notes-<version>.md`.

**Step 1: Update line 155 (Phase 6.2, Step 1)**

Change:
```
1. Read the auto-generated notes at `/tmp/release-notes-<version>.md`
```
To:
```
1. Read the auto-generated notes at `.release/notes-<version>.md`
```

**Step 2: Update line 224 (Phase 6.2, Step 4)**

Change:
```
Write the enhanced notes back to `/tmp/release-notes-<version>.md`, then display them for user review.
```
To:
```
Write the enhanced notes back to `.release/notes-<version>.md`, then display them for user review.
```

**Step 3: Update line 234 (Phase 6.3 prompt)**

Change:
```
Release notes saved to: /tmp/release-notes-<version>.md
```
To:
```
Release notes saved to: .release/notes-<version>.md
```

**Step 4: Commit**

```bash
git add .claude/skills/prepare-release/SKILL.md
git commit -m "docs(skills): update prepare-release paths to .release/"
```

---

## Task 7: Update `publish-release` Skill

**Files:**
- Modify: `.claude/skills/publish-release/SKILL.md:12-17,66-67,79-80`

**Context:** The prerequisites need to mention the notes file explicitly, and Step 4's report should reference prepared notes (not "auto-generated").

**Step 1: Update Prerequisites (lines 12-17)**

Change:
```markdown
## Prerequisites

Before running this skill:
1. Run `/prepare-release <version>` to complete phases 1-6
2. Review the generated release notes
3. Ensure you're ready to publish (this is irreversible)
```
To:
```markdown
## Prerequisites

Before running this skill:
1. Run `/prepare-release <version>` to complete phases 1-6
2. Verify prepared notes exist at `.release/notes-<version>.md`
3. Review the release notes in that file
4. Ensure you're ready to publish (this is irreversible)
```

**Step 2: Update script description (lines 66-67)**

Change:
```
5. Generates release notes
6. Publishes with `gh release edit --draft=false`
```
To:
```
5. Reads prepared notes from `.release/notes-<version>.md` (fails if missing)
6. Publishes with `gh release edit --draft=false`
```

**Step 3: Update success report (lines 79-80)**

Change:
```
- Auto-generated release notes
```
To:
```
- Enhanced release notes (from prepare step)
```

**Step 4: Commit**

```bash
git add .claude/skills/publish-release/SKILL.md
git commit -m "docs(skills): update publish-release to reference .release/ notes"
```

---

## Task 8: Verify the Full Chain

**Context:** Run through the verification checklist from the design doc to confirm all layers work.

**Step 1: Test hook blocks `gh release create`**

Run: `echo '{"command": "gh release create v9.9.9 --draft"}' | bash .claude/hooks/block-release-create.sh`
Expected: Exit code 2, message containing "BLOCKED"

**Step 2: Test hook allows other gh commands**

Run: `echo '{"command": "gh release view v1.46.0"}' | bash .claude/hooks/block-release-create.sh`
Expected: Exit code 0 (allowed)

**Step 3: Test publish script fails without prepared notes**

Run: `bash -c 'VERSION=v99.99.99; NOTES_FILE=".release/notes-${VERSION}.md"; [[ ! -f "$NOTES_FILE" ]] && echo "PASS: would fail" || echo "FAIL: file exists"'`
Expected: "PASS: would fail"

**Step 4: Test prepare script creates .release/ directory**

Run: `bash -c 'mkdir -p .release && [[ -d .release ]] && echo "PASS: directory created" && rm -rf .release'`
Expected: "PASS: directory created"

**Step 5: Verify all JSON is valid**

Run: `jq . .claude/settings.json > /dev/null && echo "settings.json: valid"`
Expected: "settings.json: valid"

**Step 6: Final commit (if any verification fixes needed)**

Only if adjustments were required during verification.

---

## Summary of Changes

| # | File | Change |
|---|------|--------|
| 1 | `.claude/hooks/block-release-create.sh` | New: hook script (reads stdin JSON, blocks `gh release create`) |
| 2 | `.claude/settings.json` | Add hook registration + permission |
| 3 | `.gitignore` | Add `.release/` |
| 4 | `.claude/scripts/prepare-release.sh` | `mkdir -p .release`, write notes there |
| 5 | `.claude/scripts/publish-release.sh` | Read notes from `.release/`, hard fail if missing |
| 6 | `.claude/skills/prepare-release/SKILL.md` | Update 3 path references |
| 7 | `.claude/skills/publish-release/SKILL.md` | Update prerequisites + report text |
