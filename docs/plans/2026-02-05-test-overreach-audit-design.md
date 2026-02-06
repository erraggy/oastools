# Test Overreach Audit Design

**Date:** 2026-02-05
**Scope:** All 28 packages in `github.com/erraggy/oastools`
**Goal:** Identify test code that protects non-public-API code, with a prioritized report of removal candidates to improve signal-to-noise in the test suite.

## Methodology

### Three-Layer Analysis

**Layer 1 — Public API Surface Mapping**
Use `go_package_api` to enumerate every exported type, function, method, and variable per package. This defines the contract — everything else exists to serve these symbols.

**Layer 2 — Reachability Tracing**
For every unexported function/method, use `go_file_context` and `go_symbol_references` to determine if it's transitively reachable from any exported symbol. If an unexported function is only called from test code or other unreachable unexported functions, it's dead code or test infrastructure.

**Layer 3 — Test Target Classification**
Each test function is classified:
- **Category A (Essential):** Tests an exported symbol or an unexported symbol transitively reachable from an export
- **Category B (Test Infrastructure):** Tests a helper function defined in `_test.go` files, `testutil` packages, or `helpers_test.go`
- **Category C (Redundant Internal):** Tests an unexported function whose behavior is fully exercised by Category A tests
- **Category D (Dead Code):** Tests a function/type with no references from any public API path

JetBrains inspections supplement static tracing by catching unused parameters, unreachable branches, and dead declarations.

## Package Clusters (Parallel Execution)

| Cluster | Packages |
|---------|----------|
| 1 — Core Pipeline | `parser/`, `parser/internal/jsonhelpers/`, `internal/pathutil/` |
| 2 — Validation & Fixing | `validator/`, `fixer/` |
| 3 — Transformation | `joiner/`, `converter/`, `overlay/`, `differ/` |
| 4 — Runtime & Generation | `httpvalidator/`, `generator/`, `builder/`, `walker/` |
| 5 — CLI & Utilities | `cmd/oastools/`, `internal/testutil/`, `integration/`, remaining internals |

## Output Format

Per-package sections with:
- Public API surface count
- Test inventory count
- Overreach findings table (Categories B/C/D only, with test function name, category, lines, rationale)
- Package observations (prose)

Cross-package summary:
- Top highest-impact removals
- Patterns observed
- Recommendations for test hygiene
