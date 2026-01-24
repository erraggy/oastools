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
2. Verify prepared notes exist at `.release/notes-<version>.md`
3. Review the release notes in that file
4. Ensure you're ready to publish (this is irreversible)

## Process

> ⚠️ **CRITICAL:** This skill wraps `publish-release.sh`. Do NOT run release commands manually.

### Step 1: Validate Version Argument

If no version is provided, **stop and ask the user**:
```
Error: Version argument required.
Usage: /publish-release <version>
Example: /publish-release v1.46.0
```

If the version doesn't match the `vX.Y.Z` pattern, reject it:
```
Error: Version must match vX.Y.Z pattern (e.g., v1.46.0)
Got: <invalid-version>
```

### Step 2: Confirm with User

Use the **AskUserQuestion** tool to confirm before proceeding:
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

From the repository root, run:

```bash
.claude/scripts/publish-release.sh <version>
```

The script handles:
1. Verifies on main branch
2. Creates and pushes annotated tag
3. Waits for goreleaser workflow
4. Verifies draft has 8 assets (binaries + checksums)
5. Reads prepared notes from `.release/notes-<version>.md` (fails if missing)
6. Publishes with `gh release edit --draft=false`
7. Verifies published release

### Step 4: Report Results

On success, report:
```
✅ Release <version> published successfully!

View release: https://github.com/erraggy/oastools/releases/tag/<version>

The release includes:
- 8 binary assets for all platforms
- Enhanced release notes (from prepare step)
- Homebrew formula will update automatically
```

On failure, report the error and suggest recovery steps.

## Important Notes

- Version argument is **required** (no inference - you're publishing what you prepared)
- Always confirm before publishing - releases are irreversible
- If the script fails partway, check the error message for recovery steps
- **NEVER** use `gh release create` - goreleaser creates the draft
