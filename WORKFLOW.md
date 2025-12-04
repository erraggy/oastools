# Development Workflow

This document defines the standard workflow for the oastools repository, from commit to pull request to release.

## Table of Contents

- [Development Workflow](#development-workflow)
  - [Table of Contents](#table-of-contents)
  - [Commit-to-PR Workflow](#commit-to-pr-workflow)
    - [Pre-Commit Checklist](#pre-commit-checklist)
    - [Commit Message Format](#commit-message-format)
    - [Local Code Review](#local-code-review)
    - [Pushing Changes](#pushing-changes)
  - [PR Workflow](#pr-workflow)
    - [Creating a Pull Request](#creating-a-pull-request)
    - [PR Title and Description](#pr-title-and-description)
    - [PR Review Process](#pr-review-process)
    - [Merging](#merging)
  - [PR-to-Release Workflow](#pr-to-release-workflow)
    - [Prerequisites](#prerequisites)
    - [Semantic Versioning Guidelines](#semantic-versioning-guidelines)
    - [Release Process](#release-process)
    - [Post-Release Verification](#post-release-verification)
    - [Troubleshooting](#troubleshooting)
  - [Quick Reference Commands](#quick-reference-commands)

## Commit-to-PR Workflow

### Pre-Commit Checklist

Before committing code changes, ensure:

1. **All tests pass:**
   ```bash
   make check  # Runs tidy, fmt, lint, test, and shows git status
   ```

2. **Code coverage is adequate:**
   ```bash
   make test-coverage  # Generate and view HTML coverage report
   ```
   - All exported functions, methods, and types must have tests
   - Include positive cases, negative cases, and edge cases
   - Integration tests for components working together

3. **Benchmarks updated (if applicable):**
   ```bash
   make bench-save  # If changes affect performance
   ```

4. **Security vulnerabilities checked:**
   ```bash
   go run golang.org/x/vuln/cmd/govulncheck@latest ./...
   ```

5. **No violations of project standards:**
   - Use package-level constants (e.g., `httputil.MethodGet`, `severity.SeverityError`)
   - Follow Go 1.24+ benchmark pattern (`for b.Loop()`)
   - Avoid over-engineering (only implement what's requested)
   - No backwards-compatibility hacks for unused code

### Commit Message Format

**Structure:**
```
<type>(<scope>): <subject>

<body>

<footer>
```

**First Line (72 characters max):**
- Type: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`
- Scope: package name or component (optional)
- Subject: imperative mood, lowercase, no period

**Body (100 columns max):**
- Explain what and why, not how
- Simple formatting, basic reasoning

**Examples:**
```
feat(converter): add OAS 3.2.0 support

Add conversion support for OpenAPI Specification 3.2.0, including new
schema types and webhook improvements. This enables users to convert
between all OAS versions from 2.0 through 3.2.0.

Closes #123
```

```
fix(parser): handle nil schema.Type in OAS 3.1 documents

Use type assertions to safely handle schema.Type which can be either
string or []string in OAS 3.1+. Prevents nil pointer dereference when
processing documents with missing type fields.

Fixes #456
```

### Local Code Review

**One-time setup:**
```bash
./scripts/install-git-hooks.sh  # Installs pre-push hook
```

**Manual review (optional but recommended):**
```bash
# Review all uncommitted changes
./scripts/local-code-review.sh

# Review only staged changes (useful before commit)
./scripts/local-code-review.sh staged

# Review all changes in current branch (useful before PR)
./scripts/local-code-review.sh branch
```

### Pushing Changes

**Standard push:**
```bash
git push origin <branch-name>
```

The pre-push hook will automatically run local code review.

**Skip review (use sparingly):**
```bash
SKIP_REVIEW=1 git push origin <branch-name>
```

**Bypass hook entirely (not recommended):**
```bash
git push origin <branch-name> --no-verify
```

## PR Workflow

### Creating a Pull Request

**Method 1: Using GitHub CLI (recommended):**
```bash
# Ensure changes are committed and pushed
git push origin <branch-name> -u

# Create PR using gh cli
gh pr create --title "Your PR Title" --body "$(cat <<'EOF'
## Summary
[1-3 bullet points summarizing the changes]

## Changes
- [Detailed change 1]
- [Detailed change 2]

## Context
[Why these changes were made, any relevant background]

## Testing
- [How changes were tested]
- [ ] All tests pass (`make check`)
- [ ] Coverage reviewed (`make test-coverage`)
- [ ] Security scan clean (`govulncheck`)

## Related Issues
Closes #XXX
EOF
)"
```

**Method 2: Using GitHub Web UI:**
1. Push your branch to GitHub
2. Navigate to repository on GitHub
3. Click "Compare & pull request"
4. Fill in title and description using template below

### PR Title and Description

**Title Format:**
Same as commit message first line (conventional commit format)

**Description Template:**
```markdown
## Summary
[1-3 bullet points summarizing the changes]

## Changes
- [Detailed change 1]
- [Detailed change 2]
- [Detailed change 3]

## Context
[Why these changes were made, relevant background, design decisions]

## Testing
- [How changes were tested]
- [ ] All tests pass (`make check`)
- [ ] Coverage reviewed (`make test-coverage`)
- [ ] Security scan clean (`govulncheck`)
- [ ] Local code review passed

## Related Issues
Closes #XXX
Fixes #YYY
Related to #ZZZ
```

### PR Review Process

**Monitoring PR checks:**
```bash
# Check status of all PR checks
gh pr checks <PR_NUMBER>

# View workflow run details
gh run view <RUN_ID>

# Monitor running workflow
gh run watch <RUN_ID>

# Get all PR comments (including bot reviews)
gh pr view <PR_NUMBER> --comments
```

**Addressing review feedback:**
1. Make requested changes in your local branch
2. Run `make check` to ensure quality
3. Commit changes with descriptive message
4. Push to update the PR
5. Respond to review comments

**Never submit a PR with:**
- Untested exported functions, methods, or types
- Tests that only cover the "happy path" without error cases
- Performance regressions without documented justification
- Security vulnerabilities
- Failing lints or tests

### Merging

PRs are merged by maintainers once:
1. All checks pass
2. Code review approved
3. No outstanding change requests
4. Branch is up-to-date with main

## PR-to-Release Workflow

### Prerequisites

Before creating a release:

1. **On main branch, up-to-date:**
   ```bash
   git checkout main
   git pull origin main
   ```

2. **All tests pass:**
   ```bash
   make check
   ```

3. **Benchmarks updated:**
   Follow [BENCHMARK_UPDATE_PROCESS.md](BENCHMARK_UPDATE_PROCESS.md) to update benchmark results

4. **Review merged PRs:**
   ```bash
   git log $(git describe --tags --abbrev=0)..HEAD --oneline
   gh pr list --state merged --limit 20
   ```

### Semantic Versioning Guidelines

**PATCH** (`v1.6.0` → `v1.6.1`):
- Bug fixes
- Documentation updates
- Small refactors without API changes
- Performance improvements without API changes

**MINOR** (`v1.6.0` → `v1.7.0`):
- New features
- New public APIs (backward compatible)
- Significant optimizations
- Deprecations (without removal)

**MAJOR** (`v1.6.0` → `v2.0.0`):
- Breaking changes to public APIs
- Removal of deprecated functionality
- Incompatible API changes

**Note:** See CLAUDE.md "Semantic Versioning" section for context on v1.13.0 exception.

### Release Process

This process uses GitHub's **immutable releases** feature. The workflow creates a draft release that must be manually published.

**Workflow:** Push tag → CI creates draft + uploads assets → Review → Publish

**Step 1: Prepare for release**
```bash
# Ensure you're on main and up-to-date
git checkout main
git pull origin main

# Run all checks
make check

# Review changes since last release
git log $(git describe --tags --abbrev=0)..HEAD --oneline
```

**Step 2: Create and push the tag**
```bash
# Determine the next version following semver
# Create and push the tag (this triggers the workflow)
git tag v1.X.Y
git push origin v1.X.Y
```

**Step 3: Monitor the workflow**
```bash
# Watch the workflow run
gh run list --workflow=release.yml --limit=1

# Get the run ID and monitor progress
gh run watch <RUN_ID>
```

The workflow will:
- Build binaries for all platforms (Darwin, Linux, Windows)
- Create a **draft** release on GitHub
- Upload all binary assets to the draft (8 files total)
- Push the Homebrew formula to `erraggy/homebrew-oastools`

**Step 4: Verify the draft release**
```bash
# Confirm draft status and assets
gh release view v1.X.Y --json isDraft,assets --jq '{isDraft, assetCount: (.assets | length), assets: [.assets[].name]}'
```

Expected output:
- `isDraft: true`
- 8 assets: checksums + binaries for Darwin (amd64/arm64), Linux (amd64/arm64), Windows (amd64/arm64)

**Step 5: Generate and set release notes**

Use Claude Code to generate release notes:

```
Prompt: "Generate release notes for v1.X.Y based on changes since the last release"
```

Claude will:
- Review `git log` and merged PRs
- Categorize changes (features, bug fixes, improvements, breaking changes)
- Generate well-formatted markdown release notes
- Apply them to the draft release using `gh release edit`

**Manual alternative:**
```bash
gh release edit v1.X.Y --notes "$(cat <<'EOF'
## Summary
[High-level overview of what this release delivers]

## What's New
- [Feature 1: Description]
- [Feature 2: Description]

## Bug Fixes
- [Fix 1]
- [Fix 2]

## Improvements
- [Improvement 1]

## Breaking Changes
- [If any - rare, would typically trigger major version]

## Related PRs
- #XX - [PR title]
- #YY - [PR title]

## Installation

### Homebrew
\`\`\`bash
brew tap erraggy/oastools
brew install oastools
\`\`\`

### Go Module
\`\`\`bash
go get github.com/erraggy/oastools@v1.X.Y
\`\`\`

### Binary Download
Download the appropriate binary for your platform from the assets below.
EOF
)"
```

**Step 6: Publish the release (hands-on-keyboard)**
```bash
# Review the draft one final time on GitHub web UI (optional)
gh release view v1.X.Y --web

# Publish the release (makes it immutable)
gh release edit v1.X.Y --draft=false
```

⚠️ **WARNING:** Once published with immutability enabled, releases cannot be modified or deleted without admin intervention.

### Post-Release Verification

**Test installation:**
```bash
# Homebrew installation
brew update
brew upgrade oastools || brew install erraggy/oastools/oastools
oastools --version

# Expected output: oastools version v1.X.Y
```

**Go module:**
```bash
go get github.com/erraggy/oastools@v1.X.Y
```

**Verify release on GitHub:**
```bash
gh release view v1.X.Y
```

### Troubleshooting

**Workflow failed before creating draft:**
```bash
# Delete the tag and start over
git push origin :refs/tags/v1.X.Y
git tag -d v1.X.Y

# Fix the issue, then repeat from Step 2
```

**Workflow failed after creating draft (missing assets):**
```bash
# Delete the draft release and tag
gh release delete v1.X.Y --yes
git push origin :refs/tags/v1.X.Y
git tag -d v1.X.Y

# Fix the issue, then repeat from Step 2
```

**Accidentally published too early:**

With immutability enabled, this cannot be fixed easily. Options:
1. Contact GitHub support to delete the release (requires admin)
2. Temporarily disable immutability in repository settings, delete release, re-enable
3. Delete the release and tag, increment the version (v1.X.Y → v1.X.Y+1), start over

Prevention: Always review the draft thoroughly before publishing.

**Other issues:**

- **GoReleaser can't push to Homebrew repo:**
  - Verify `homebrew-oastools` repository exists
  - Check `HOMEBREW_TAP_TOKEN` secret has `repo` scope
  - Verify commit author email is verified on GitHub

- **Build fails:**
  - Review GitHub Actions logs
  - Check for CGO dependencies
  - Test locally with `make release-test`

- **Formula doesn't work:**
  - Verify formula in `homebrew-oastools` repository
  - Test installation in clean environment
  - Check formula URL and SHA256 checksums

**Retrieving security alerts:**
```bash
# List all code scanning alerts
gh api /repos/erraggy/oastools/code-scanning/alerts

# Check for Go vulnerabilities
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## Quick Reference Commands

**Development:**
```bash
make check              # Run all quality checks
make test-coverage      # View test coverage
make bench-save         # Update benchmarks
govulncheck ./...       # Check for vulnerabilities
```

**Git workflow:**
```bash
git checkout -b feature-branch          # Create feature branch
# Make changes
make check                              # Verify quality
./scripts/local-code-review.sh branch   # Review changes
git add .
git commit -m "feat: your message"      # Commit
git push origin feature-branch -u       # Push
```

**Pull requests:**
```bash
gh pr create --title "..." --body "..." # Create PR
gh pr checks <NUMBER>                   # Check PR status
gh pr view <NUMBER>                     # View PR details
gh pr merge <NUMBER>                    # Merge PR (maintainers)
```

**Releases:**
```bash
git tag v1.X.Y                          # Create tag
git push origin v1.X.Y                  # Push tag (triggers workflow)
gh run watch <RUN_ID>                   # Monitor workflow
gh release view v1.X.Y                  # Verify draft
gh release edit v1.X.Y --notes "..."    # Add release notes
gh release edit v1.X.Y --draft=false    # Publish
```

**Monitoring:**
```bash
gh run list --workflow=release.yml      # List release workflows
gh run view <RUN_ID>                    # View workflow details
gh run watch <RUN_ID>                   # Watch workflow
gh pr checks <NUMBER>                   # Check PR status
```
