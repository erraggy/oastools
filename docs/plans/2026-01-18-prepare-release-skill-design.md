# Prepare-Release Skill Redesign

**Date:** 2026-01-18
**Status:** Approved
**Goal:** Make the release process rock-solid by encapsulating deterministic steps in scripts while keeping judgment-based work with agents.

## Background

The `/prepare-release` skill has had reliability issues since Claude Code v2.1.x changes to skill/command execution. Two releases (v1.43.0, v1.44.0) failed to get homebrew binaries attached due to Claude inconsistently following prose instructions. The `publish-release.sh` script was introduced to handle the critical final phase, which succeeded. This design extends that pattern.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Script vs. agent split | Phases 1-3 agents, 4-7 scripts | Judgment work benefits from agents; mechanical steps need determinism |
| Version inference | Claude analyzes commits, prompts if ambiguous | Conventional commits are structured but context helps explain recommendations |
| Script organization | Two scripts: prepare + publish | Clear separation, simpler than phase-per-script |
| Command structure | Two commands with publish prompt | Human checkpoint before irreversible step, but frictionless in happy path |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     /prepare-release [version]                   │
├─────────────────────────────────────────────────────────────────┤
│  Version Inference (Claude)                                      │
│  └─ Parse commits → suggest version → confirm if ambiguous       │
├─────────────────────────────────────────────────────────────────┤
│  Phases 1-3 (Agents)                                             │
│  └─ DevOps, Architect, Maintainer, Developer review & fix        │
├─────────────────────────────────────────────────────────────────┤
│  Phases 4-6 (prepare-release.sh)                                 │
│  └─ Benchmarks → PR → Merge → Release Notes                      │
├─────────────────────────────────────────────────────────────────┤
│  Prompt: "Ready to publish?" ─────┬─► Yes: invoke publish-release│
│                                   └─► No: end here               │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    /publish-release <version>                    │
├─────────────────────────────────────────────────────────────────┤
│  Phase 7 (publish-release.sh)                                    │
│  └─ Tag → Push → Wait for goreleaser → Verify → Publish          │
└─────────────────────────────────────────────────────────────────┘
```

## Command Interface

### `/prepare-release [version]`

Runs phases 1-6 of release preparation.

- **With version:** `/prepare-release v1.46.0` - uses provided version
- **Without version:** `/prepare-release` - infers from conventional commits

### `/publish-release <version>`

Runs phase 7 (tag and publish). Version is required.

Exists as standalone command for:
- Publishing later after review
- Re-running after a publish failure
- Manual control when needed

## Version Inference Logic

When no version is provided, Claude analyzes commits since the last tag:

1. Get last tag: `git describe --tags --abbrev=0`
2. Get commits since: `git log <last-tag>..HEAD --oneline`
3. Parse conventional commit prefixes:
   - Any `BREAKING CHANGE:` or `!:` → MAJOR bump
   - Any `feat:` → MINOR bump
   - Only `fix:`, `chore:`, `docs:`, etc. → PATCH bump
4. If clear signal → propose version with explanation
5. If ambiguous → prompt user to choose

### Example

```
User: /prepare-release

Claude: Analyzing commits since v1.45.2...

Found 4 commits:
- feat(parser): add streaming support
- fix(validator): handle empty schemas
- chore: update dependencies
- docs: improve examples

Recommendation: **v1.46.0** (minor bump due to new feature)

Proceed with v1.46.0? [Yes / Different version]
```

## Phase Breakdown

### Phases 1-3: Agent Work (Review & Validation)

These remain as coordinated agent tasks because they require judgment:

| Phase | Agent | Tasks |
|-------|-------|-------|
| 1 | DevOps | Check commits since last tag, run `make bench-quick`, create branch `chore/<version>-release-prep` |
| 2 | Architect | Review CLAUDE.md, README.md, doc.go files, docs/ for needed updates |
| 3 | Maintainer | Run `make check`, `govulncheck`, verify gopls diagnostics clean |
| 3b | Developer | Check `example_test.go` coverage for new features |

Agents run in background with incremental progress reporting via status table.

### Phases 4-6: `prepare-release.sh` Script

Deterministic steps executed by script:

| Step | Action |
|------|--------|
| 4.1 | Push branch to origin |
| 4.2 | Trigger benchmark workflow via `gh workflow run` |
| 4.3 | Wait for workflow completion |
| 4.4 | Pull benchmark commit |
| 5.1 | Verify benchmark file exists |
| 5.2 | Create PR with `gh pr create` |
| 5.3 | Wait for CI, merge with `gh pr merge --squash --admin` |
| 6.1 | Generate release notes via GitHub API |
| 6.2 | Save notes to temp file for review |
| 6.3 | Prompt user: "Ready to publish?" → if yes, invoke `publish-release.sh` |

### Phase 7: `publish-release.sh` (Existing)

Unchanged - already rock-solid:
1. Verify on main branch
2. Create and push annotated tag
3. Wait for goreleaser workflow
4. Verify draft has 8 assets
5. Publish with `gh release edit --draft=false`
6. Verify published release

## Script Design

### `prepare-release.sh`

```bash
.claude/scripts/prepare-release.sh <version> [--skip-benchmarks]
```

- **`<version>`** - Required (e.g., `v1.46.0`)
- **`--skip-benchmarks`** - Optional, skip CI benchmarks for re-runs

### Exit Codes

| Code | Meaning | Recovery Action |
|------|---------|-----------------|
| 0 | Success | Continue |
| 1 | Usage/validation error | Fix arguments, re-run |
| 2 | Prerequisite failed | Fix prereq (e.g., wrong branch), re-run |
| 3 | External service failed | Wait/retry (e.g., GitHub API, workflow) |
| 4 | Verification failed | Manual intervention needed |

### Idempotency

Script checks state before acting:
- Don't re-push if branch already on remote
- Don't re-trigger benchmark if file exists
- Don't re-create PR if one exists for the branch

### Recovery Scenarios

| Failure Point | State After Failure | Recovery |
|---------------|---------------------|----------|
| Branch push fails | Local branch exists | Fix push issue, re-run script |
| Benchmark workflow fails | Branch pushed, no benchmark | Fix workflow, re-run with same args |
| PR creation fails | Benchmark committed | Re-run with `--skip-benchmarks` |
| PR merge fails | PR exists | Manually merge or re-run |
| Notes generation fails | PR merged | Re-run (idempotent) or generate manually |

## File Structure

```
.claude/
├── scripts/
│   ├── prepare-release.sh    # NEW - Phases 4-6
│   └── publish-release.sh    # EXISTS - Phase 7
└── skills/
    ├── prepare-release/
    │   └── SKILL.md          # UPDATE - Phases 1-6, version inference
    └── publish-release/
        └── SKILL.md          # NEW - Standalone phase 7 wrapper
```

## Implementation Checklist

- [ ] Create `.claude/scripts/prepare-release.sh` (~80-100 lines)
  - Phase 4: benchmark workflow trigger & wait
  - Phase 5: PR create & merge
  - Phase 6: release notes generation
- [ ] Create `.claude/skills/publish-release/SKILL.md` (~30 lines)
  - Simple wrapper around existing script
- [ ] Update `.claude/skills/prepare-release/SKILL.md`
  - Add version inference logic at top
  - Simplify phases 4-6 to script invocation
  - Remove phase 7, add publish prompt at end
- [ ] Test the flow
  - Dry-run with a test version
  - Verify error recovery paths
