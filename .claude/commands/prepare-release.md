# Prepare Release

Prepare a new release by running comprehensive pre-release checks and updates.

**Usage:** `/prepare-release <version>` (e.g., `/prepare-release v1.26.0`)

## Process

You are the DevOps Engineer coordinating a release. Execute the following steps:

### Phase 1: Parallel Agent Review

Launch these agents **in parallel** to maximize efficiency:

1. **DevOps Engineer** - Update benchmarks:
   - Check commits since last release tag
   - Run `make bench-release VERSION=<version>` and update `benchmarks.md`
   - Create feature branch `chore/<version>-release-prep`

2. **Architect** - Review documentation:
   - Check if CLAUDE.md needs updates for new features
   - Check if README.md needs updates
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

### Phase 2: Consolidate Findings

After all agents complete:
1. Summarize findings in a table format
2. List any required changes before release
3. Apply necessary fixes (doc updates, missing examples, etc.)

### Phase 3: Create Pre-Release PR

1. Commit all changes with message: `chore: prepare <version> release`
2. Push branch and create PR
3. Wait for CI checks to pass
4. Merge PR

### Phase 4: Generate Release Notes

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

### Phase 5: Tag and Publish

1. Tag the release: `git tag <version> && git push origin <version>`
2. Monitor the release workflow: `gh run watch`
3. Verify draft release is created
4. Edit release notes and publish: `gh release edit <version> --draft=false`

## Important Notes

- Always run on `main` branch (after merging any prep changes)
- Use `--admin` flag for PR merge if branch protection blocks
- Benchmark updates are required for MINOR/MAJOR releases
- Document all new public API in CLAUDE.md
