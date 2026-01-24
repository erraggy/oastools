# Release Process Fix Design

**Date:** 2026-01-23
**Problem:** 3 failed releases without homebrew binaries + prepared release notes discarded on publish

## Background

The release process has two skills (`prepare-release`, `publish-release`) backed by shell scripts. Three releases have shipped without pre-built binaries because the agent bypassed the publish script and used `gh release create` directly. Additionally, the publish script generates its own auto-notes instead of using the enhanced notes prepared by the prepare skill.

## Root Causes

1. **No enforcement**: Nothing prevents the agent from running `gh release create` directly
2. **Notes handoff broken**: `publish-release.sh` generates fresh notes via GitHub API instead of reading the prepared file
3. **Ephemeral storage**: Prepared notes saved to `/tmp/` which doesn't persist across sessions

## Design

### Change 1: Pre-tool-use Hook to Block `gh release create`

**File:** `.claude/hooks/block-release-create.sh`

A hook that intercepts Bash tool invocations and blocks any command containing `gh release create`. This operates below the LLM decision layer — the agent physically cannot execute the dangerous command.

**Hook registration** in `.claude/settings.local.json`:
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "command": "bash .claude/hooks/block-release-create.sh"
      }
    ]
  }
}
```

### Change 2: Git-ignored `.release/` Directory

A `.release/` directory at the repo root stores prepared release notes locally. Added to `.gitignore` so notes never get committed but persist across sessions (unlike `/tmp/`).

**Structure:**
```
.release/
  notes-v1.46.2.md
  notes-v1.47.0.md
```

**`.gitignore` addition:**
```
# Release notes (local-only, used by publish-release.sh)
.release/
```

### Change 3: Publish Script Reads Prepared Notes (Hard Fail)

Replace the auto-note generation in `publish-release.sh` (current Step 5) with:

```bash
NOTES_FILE=".release/notes-${VERSION}.md"
if [[ ! -f "$NOTES_FILE" ]]; then
    echo "Error: Prepared release notes not found: $NOTES_FILE" >&2
    echo "Run /prepare-release $VERSION first to generate release notes." >&2
    exit 4
fi
NOTES=$(cat "$NOTES_FILE")
gh release edit "$VERSION" --notes "$NOTES" --draft=false
```

The script **refuses to publish** if the notes file doesn't exist, forcing the prepare step to always run first.

### Change 4: Skill and Script Path Updates

Update all references from `/tmp/release-notes-<version>.md` to `.release/notes-<version>.md`:

- `prepare-release.sh`: Line 200, add `mkdir -p .release`
- `prepare-release/SKILL.md`: Phase 6.2 Step 4, Phase 6.3 prompt text
- `publish-release/SKILL.md`: Prerequisites section, Step 4 report text

## Files Changed

| File | Change |
|------|--------|
| `.claude/hooks/block-release-create.sh` | New: hook script |
| `.claude/settings.local.json` | Add hook registration |
| `.gitignore` | Add `.release/` |
| `.claude/scripts/publish-release.sh` | Read notes from `.release/`, hard fail if missing |
| `.claude/scripts/prepare-release.sh` | Write notes to `.release/` |
| `.claude/skills/prepare-release/SKILL.md` | Update notes file paths |
| `.claude/skills/publish-release/SKILL.md` | Update prerequisites and report |

## Verification

After implementation, the following should be true:
1. `gh release create` in any Bash command → blocked by hook with clear error
2. `/publish-release v1.x.x` without prior `/prepare-release` → hard fail (no notes file)
3. `/prepare-release v1.x.x` → writes enhanced notes to `.release/notes-v1.x.x.md`
4. `/publish-release v1.x.x` after prepare → publishes with the enhanced notes
