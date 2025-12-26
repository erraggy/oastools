# Prepare Release

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
   - Note: Full benchmarks run automatically via CI when tag is pushed

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
   - Full benchmarks will run in CI after tag push

### Phase 3: Consolidate & Fix

After all agents complete:
1. Final summary table of all findings
2. List any remaining required changes
3. Apply fixes that couldn't be done incrementally

### Phase 4: Create Pre-Release PR

1. Commit all changes with message: `chore: prepare <version> release`
2. Push branch and create PR
3. Wait for CI checks to pass
4. Merge PR

### Phase 5: Generate Release Notes

Create release notes with this structure:

```markdown
## What's New

### Features
- List new features with brief descriptions

### Improvements
- List improvements and enhancements

### Bug Fixes
- List bug fixes

### Documentation
- List documentation updates

## Breaking Changes
- List any breaking changes (or "None" if backward compatible)

## Upgrade Notes
- Any notes for users upgrading from previous version
```

### Phase 6: Tag and Publish

1. Tag the release: `git tag <version> && git push origin <version>`
2. Monitor both workflows (they run in parallel):
   - Release workflow: `gh run list --workflow=release.yml --limit=1`
   - Benchmark workflow: `gh run list --workflow=benchmark.yml --limit=1`
   - Watch with: `gh run watch <RUN_ID>`
3. Verify draft release is created with all assets
4. Verify benchmark workflow creates PR with results (merge the PR)
5. Edit release notes and publish: `gh release edit <version> --draft=false`

## Important Notes

- Always run on `main` branch (after merging any prep changes)
- Use `--admin` flag for PR merge if branch protection blocks
- Full benchmarks run automatically in CI when tag is pushed (~5 min parallel vs ~20 min local)
- Benchmark workflow creates a PR that needs to be merged after release
- Document all new public API in CLAUDE.md
