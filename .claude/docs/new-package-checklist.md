# New Package Checklist

## Adding a New Package

When adding a new package, ensure:

1. **Implementation**
   - Package files
   - `doc.go` (package documentation)
   - `example_test.go` (godoc examples)
   - `deep_dive.md` (detailed documentation)
   - Comprehensive tests

2. **CLI** (if applicable)
   - Command in `cmd/oastools/commands/`
   - Register in `main.go`

3. **Benchmarks**
   - Create `*_bench_test.go`
   - Use `for b.Loop()` pattern (see benchmark-guide.md)

4. **Documentation**
   - Update README.md
   - Update benchmarks.md
   - Update developer-guide.md
   - Update mkdocs.yml
   - Update CLAUDE.md (Public API list)

5. **Verification**
   - Run `make check`
   - Run `make bench-<package>`
   - Verify `go doc` works

## Adding Examples

When adding new examples under `examples/`, you must update **both**:

### 1. `mkdocs.yml`
Add nav entries with explicit titles (not just file paths):
```yaml
# Good - with titles
- Overview: examples/myexample/index.md
- Feature One: examples/myexample/feature-one.md

# Bad - will show "None" in nav
- examples/myexample/index.md
```

### 2. `scripts/prepare-docs.sh`
Add copy commands for new example READMEs:
- Create target directory: `mkdir -p "$DOCS_DIR/examples/myexample"`
- Copy and fix links for each README
- Add any new sibling link patterns to `fix_example_links()`

⚠️ **Common mistake**: Updating only `mkdocs.yml` without updating `prepare-docs.sh` causes 404 errors and "None" nav entries on the docs site.
