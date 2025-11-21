# Homebrew Cask Migration Follow-Up

## Context

As of commit 9c737d0 (2025-11-20), we migrated the GoReleaser configuration from the deprecated `brews` section to the modern `homebrew_casks` section. This aligns with GoReleaser v2.10+ best practices for distributing pre-compiled binaries.

## What Changed

- **Before**: Used `brews` configuration, which generated Homebrew Formulas in `Formula/oastools.rb`
- **After**: Uses `homebrew_casks` configuration, which generates Homebrew Casks in `Casks/oastools.rb`
- **Added**: `url.verified` field for Homebrew audit compliance

## Required Follow-Up Actions

### 1. After Next Release (v1.9.5+)

Once the next release is published and the new Cask file has been generated in the `homebrew-oastools` repository, we need to disable the old Formula to guide users to the new Cask.

### 2. Disable Old Formula

Manually edit the old Formula file in the `homebrew-oastools` repository:

**File**: `Formula/oastools.rb` (or just `oastools.rb` if not in a subdirectory)

**Add this to the Formula class**:
```ruby
class Oastools < Formula
  # ... existing content ...
  version "1.9.4"  # Update to the last formula version
  # ... existing content ...

  # Add this disable! directive
  disable! date: "2025-12-20", because: "the cask should be used now instead", replacement_cask: "oastools"
end
```

### 3. User Migration Experience

After disabling the old Formula, when users try to upgrade, they'll see:

```shell
==> Upgrading 1 outdated package:
erraggy/oastools/oastools 1.9.4 -> 1.9.5
Error: erraggy/oastools/oastools has been disabled because the cask should be used now instead!
Replacement:
  brew install --cask oastools
```

### 4. Verify Installation Still Works

The good news is that Homebrew is smart enough to find Casks even when using `brew install` without the `--cask` flag. Both commands will work:

```bash
# These both work and install the Cask
brew install oastools
brew install --cask oastools
```

### 5. Update Documentation (if needed)

Check if any documentation references the installation process and ensure it's still accurate. The current instructions should continue to work:

```bash
brew tap erraggy/oastools
brew install oastools
```

## Technical Notes

### Why Casks for CLI Tools?

Historically, Homebrew Formulas were for building from source, and Casks were for pre-compiled binaries (especially GUI apps). However, GoReleaser's "hackyish" approach of creating Formulas that installed pre-compiled binaries has been deprecated. The modern approach is to use Casks for **all** pre-compiled binaries, whether CLI or GUI.

### File Locations

- **Old Formula**: `Formula/oastools.rb` (if directory exists) or `oastools.rb` (root)
- **New Cask**: `Casks/oastools.rb`

Both files may coexist temporarily during the migration period.

## Checklist

- [x] Update `.goreleaser.yaml` to use `homebrew_casks` (commit 9c737d0)
- [x] Test configuration with `make release-test`
- [x] Push changes to main branch
- [ ] Release next version (v1.9.5+)
- [ ] Verify new Cask file is created in `homebrew-oastools` repository
- [ ] Disable old Formula with `disable!` directive
- [ ] Test that users can install/upgrade successfully
- [ ] Consider deleting the old Formula after a grace period (optional)

## References

- GoReleaser Deprecation Notice: https://goreleaser.com/deprecations#brews
- GoReleaser Homebrew Casks Documentation: https://goreleaser.com/customization/homebrew_casks/
- Commit with migration: 9c737d0
