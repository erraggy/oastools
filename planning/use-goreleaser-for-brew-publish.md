## Full Steps to Automate with GoReleaser

### Implementation Status: ✅ Complete

This guide documents how to set up automated Homebrew publishing for oastools using GoReleaser.

**Release Workflow:** Create GitHub Release first → GoReleaser adds binaries & publishes to Homebrew

This approach maintains full control over release notes while automating the build and distribution process.

### 1. Set up a Homebrew Tap Repository ✅

**Status:** Complete - Repository created at `erraggy/homebrew-oastools`

Create a new public GitHub repository specifically for your Homebrew tap. The repository name must be
prefixed with `homebrew-` (so: `erraggy/homebrew-oastools`). This allows users to tap it with
`brew tap erraggy/oastools`.

**Created:** https://github.com/erraggy/homebrew-oastools

### 2. Configure GoReleaser ✅

**Status:** Complete - GoReleaser installed and configured

**Installation:**
```bash
brew install goreleaser
```

**Initialization:**
```bash
goreleaser init
```

This generates a `.goreleaser.yaml` file with sensible defaults.

**Configuration:**

The `.goreleaser.yaml` has been configured with the following key sections:

```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin

archives:
  - formats: [tar.gz]
    name_template: >-
      {{ .ProjectName }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
    format_overrides:
      - goos: windows
        formats: [zip]

brews:
  - name: oastools
    repository:
      owner: erraggy
      name: homebrew-oastools
    commit_author:
      name: Robbie Coleman
      email: robbie@robnrob.com
    homepage: "https://github.com/erraggy/oastools"
    description: "OpenAPI Specification (OAS) tools for validation, parsing, converting, and joining specs."
    license: "MIT"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

**Key Configuration Points:**
- Cross-platform builds for Linux, Windows, and macOS
- Proper archive naming to match `uname` output
- Homebrew tap configured to push to `homebrew-oastools` repository
- Changelog automatically generated from git commits (excluding docs and test commits)
- **Release mode:** `keep-existing` - preserves your manually created release notes
  - Allows you to create detailed GitHub Releases first with `gh release create`
  - GoReleaser adds binaries to the existing release without overwriting your notes

### 3. Set up GitHub Actions Workflow ✅

**Status:** Complete - Workflow created at `.github/workflows/release.yml`

A GitHub Actions workflow has been created to automatically run GoReleaser when a new version tag is
pushed:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v5
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v6
        with:
          go-version: '1.24'

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**How it works:**
- Triggers on any tag push matching `v*` pattern (e.g., `v1.7.1`, `v2.0.0`)
- Uses the built-in `GITHUB_TOKEN` for authentication
- Fetches full git history for changelog generation
- Runs GoReleaser with the `--clean` flag to remove dist folder before building

### 4. GitHub Token Configuration

**IMPORTANT:** A Personal Access Token (PAT) is **REQUIRED** for automated releases.

**Why a PAT is Required:**

The default `GITHUB_TOKEN` provided by GitHub Actions only has permissions for the repository where
the workflow runs (`oastools`). GoReleaser needs to push the Homebrew formula to a separate repository
(`homebrew-oastools`), which requires a PAT with broader permissions.

**For GitHub Actions (automated releases) - REQUIRED:**

1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Set a descriptive name (e.g., "GoReleaser - oastools Homebrew Publishing")
4. Select scopes: **`repo`** (full control of private repositories)
5. Click "Generate token" and **copy the token immediately** (you won't be able to see it again)
6. Go to the oastools repository → Settings → Secrets and variables → Actions
7. Click "New repository secret"
8. Name: `HOMEBREW_TAP_TOKEN`
9. Value: Paste the token you copied
10. Click "Add secret"

The `.github/workflows/release.yml` file is already configured to use `${{ secrets.HOMEBREW_TAP_TOKEN }}`.

**For local releases:**

When running GoReleaser locally, you'll need to set the `GITHUB_TOKEN` environment variable:

```bash
export GITHUB_TOKEN="your_personal_access_token"
goreleaser release --clean
```

### 5. Testing GoReleaser Locally

Before creating a real release, test the configuration locally using snapshot mode:

```bash
goreleaser release --snapshot --clean
```

This will:
- Build binaries for all configured platforms
- Create archives
- Generate the Homebrew formula locally
- **NOT** push anything to GitHub or the Homebrew tap

Check the `dist/` directory to verify the output.

### 6. Creating a Release

**Recommended Workflow:**

This workflow creates the GitHub Release first with detailed notes, then GoReleaser adds binaries and publishes to Homebrew.

1. Ensure all changes are committed and pushed to `main`

2. Create the GitHub Release with detailed notes:
   ```bash
   gh release create v1.7.1 \
     --title "v1.7.1 - Brief summary within 72 chars" \
     --notes "$(cat <<'EOF'
   ## Summary

   High-level overview of what this release delivers.

   ## What's New

   - Feature 1: Description
   - Feature 2: Description

   ## Technical Details

   Additional context, benchmark results, migration notes, etc.

   ## Related PRs

   - #17 - PR title

   ## Installation

   ### Homebrew
   ```bash
   brew tap erraggy/oastools
   brew install oastools
   ```

   ### Binary Download
   Download the appropriate binary for your platform from the assets below.
   EOF
   )"
   ```

3. The `gh release create` command automatically:
   - Creates the version tag (e.g., `v1.7.1`)
   - Creates the GitHub Release with your detailed notes
   - Triggers the GitHub Actions workflow

4. GitHub Actions will automatically:
   - Build binaries for all platforms
   - Add binary archives to the GitHub Release
   - Publish the Homebrew formula to `homebrew-oastools`

5. Monitor the workflow at: https://github.com/erraggy/oastools/actions

**Key Benefits of This Workflow:**

- ✅ Complete control over release notes (write them first)
- ✅ Automated binary builds for all platforms
- ✅ Automated Homebrew formula publishing
- ✅ Single command to trigger everything (`gh release create`)
- ✅ GoReleaser preserves your release notes (configured with `mode: keep-existing`)

### 7. User Installation

Once a release is published, users can install oastools via Homebrew:

```bash
# Tap the repository (first time only)
brew tap erraggy/oastools

# Install oastools
brew install oastools

# Verify installation
oastools --version
```

**Updating:**
```bash
brew upgrade oastools
```

**Uninstalling:**
```bash
brew uninstall oastools
brew untap erraggy/oastools  # Optional: remove the tap
```

### 8. Troubleshooting

**Issue: GoReleaser can't push to homebrew-oastools**
- Check that the `homebrew-oastools` repository exists and is public
- Verify GitHub token has `repo` scope
- Ensure commit author email matches a verified email in your GitHub account

**Issue: Build fails for certain platforms**
- Check if any dependencies require CGO (we've set `CGO_ENABLED=0`)
- Review build logs in GitHub Actions

**Issue: Formula doesn't work on user's machine**
- Verify the binary architecture matches user's system
- Check that all runtime dependencies are documented
- Test installation in a clean environment

### Next Steps

- ✅ Initial setup complete
- ✅ `HOMEBREW_TAP_TOKEN` secret created and configured
- ✅ GoReleaser configured with `mode: keep-existing` to preserve release notes
- ⏳ Test with first real release (use `gh release create`)
- ⏳ Monitor homebrew-oastools repository for formula updates
- ⏳ Document Homebrew installation in main README.md

### Pre-Release Checklist

Before creating your first release, ensure:
- [x] `HOMEBREW_TAP_TOKEN` secret is created and added to repository secrets
- [x] Secret verified with: `gh secret list --repo erraggy/oastools`
- [ ] Local test successful: `make release-test`
- [x] Commit author email in `.goreleaser.yaml` matches a verified email in your GitHub account
- [ ] All changes committed and pushed to `main`
- [ ] All tests pass: `make check`
