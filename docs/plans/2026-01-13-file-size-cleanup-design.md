# Code & Documentation File Size Cleanup Design

**Date:** 2026-01-13
**Status:** Approved
**Goal:** Organize all Go source files and markdown documentation to fall within a 100-2000 line target range, improving maintainability and navigation.

## Goals and Constraints

### Target Line Range

- **Minimum:** 100 lines (soft constraint—consolidate fragmented files when beneficial)
- **Maximum:** 2000 lines (hard constraint—split oversized files)

### Exemptions from Minimum

The following file types may remain under 100 lines:

- `doc.go` files (Go convention for package documentation)
- Generated files (`zz_generated_*.go`)
- `constants.go` files

### Documentation Constraints

- **Source of truth:** Package docs live in `<package>/deep_dive.md`, example docs in `examples/*/README.md`
- **mkdocs compatibility:** Any doc splits must update `mkdocs.yml` nav and maintain working links
- **Whitepaper exception:** `docs/whitepaper.md` stays as a single file (GitHub Pages requirement)

## Files Requiring Attention

### Go Files Over 2000 Lines (Must Split)

| File | Lines | Package |
|------|-------|---------|
| `validator/validator_test.go` | 2864 | validator |
| `validator/validator.go` | 2667 | validator |
| `fixer/fixer_test.go` | 2566 | fixer |
| `parser/oas_types_equals_test.go` | 2441 | parser |
| `builder/parameter_test.go` | 2311 | builder |
| `differ/unified.go` | 2176 | differ |
| `joiner/joiner_test.go` | 2039 | joiner |

### Documentation to Review

| File | Lines | Action |
|------|-------|--------|
| `docs/developer-guide.md` | 1963 | Extract CLI content to `cli-reference.md` |
| `docs/whitepaper.md` | 2038 | No action (exempted) |

## Approach: Package-by-Package Cleanup

### Package Priority Order

1. **validator/** - Largest violations (2667 + 2864 lines)
2. **fixer/** - Second largest test file (2566 lines)
3. **parser/** - Large test file (2441 lines), many small utility files
4. **builder/** - Large test file (2311 lines)
5. **differ/** - Oversized source file (2176 lines)
6. **joiner/** - Borderline test file (2039 lines)
7. **Other packages** - Consolidation review only

### Per-Package Workflow

1. **Audit** - List all files with line counts, identify split/consolidate candidates
2. **Plan splits** - Identify logical seams (by feature, by OAS version, by test category)
3. **Plan consolidations** - Propose merges for small related files
4. **Execute** - Make changes, ensure tests pass
5. **Verify** - Run `make check`, confirm no regressions

## Orchestration Strategy

### Agent Delegation Model

| Task Type | Agent | Purpose |
|-----------|-------|---------|
| File analysis & audits | `general-purpose` | Explore packages, count lines, identify split points |
| Architecture decisions | `architect` | Design file splits, determine logical boundaries |
| Implementation | `developer` | Execute splits, consolidations, move code |
| Verification | `maintainer` | Review changes, ensure quality and consistency |

### Concurrency Rules

**Sequential Within Package:** All agents working on the same package run sequentially to avoid file contention:

```
Package: validator/
    1. Architect → designs split (read-only)
    2. [Checkpoint: summarize plan for approval]
    3. Developer → executes changes (writes files)
    4. Developer → runs tests (verification)
    5. Maintainer → reviews (read-only)
    6. [Checkpoint: package complete]
```

**No Parallel Package Work:** Packages are processed one at a time to avoid merge conflicts and cascading issues.

**No Background Agents for Writes:** All file-modifying agents run in foreground for immediate visibility.

## Splitting Strategies

### Source Files

Split by logical responsibility:

- **By OAS version:** `validator_oas2.go`, `validator_oas3.go`
- **By validation category:** `validator_structural.go`, `validator_semantic.go`
- **By feature area:** `unified_output.go`, `unified_formatting.go`

### Test Files

Split by test category, mirroring source splits:

- `validator_structural_test.go` - structural validation tests
- `validator_semantic_test.go` - semantic validation tests
- `validator_integration_test.go` - end-to-end tests

### Small File Consolidation

Evaluate files under 100 lines for potential merging:

- `parser/oas2.go` (24) + `parser/oas3.go` (51) → consider combining
- Multiple tiny helpers in same package → consolidate if cohesive

**Decision Principle:** Logical cohesion takes priority over line count targets.

## Documentation Cleanup

### `docs/developer-guide.md` Cleanup

1. Audit for CLI-related content (command examples, flags, CLI workflows)
2. Move CLI content to `cli-reference.md`
3. Keep library-focused content (Go API, programmatic access)
4. Add cross-references between documents

### Other Documentation

- Package `deep_dive.md` files: Most under 1600 lines (acceptable)
- Example READMEs: Generally small, no action expected
- `docs/whitepaper.md`: Excluded (GitHub Pages constraint)

## Success Criteria

- [ ] All Go source files between 100-2000 lines (exemptions allowed)
- [ ] All Go test files between 100-2000 lines (exemptions allowed)
- [ ] `developer-guide.md` contains only library content
- [ ] `cli-reference.md` contains all CLI content
- [ ] `make check` passes after all changes
- [ ] No regressions in test coverage
