# Code Simplification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce maintenance burden by splitting 4 oversized files and deduplicating the OAS 2/3 generator.

**Architecture:** Pure refactoring — move code between files (workstream 1) and extract shared base methods (workstream 2). Zero public API changes.

**Tech Stack:** Go 1.24, `goimports` for import management, `gopls` for diagnostics.

---

## PR 1: File Splits

Pure file moves within existing packages. No logic changes, no test changes.

**Branch:** `code-simplification` (already created)

**Guiding principle:** Each new file gets its own `package` declaration and imports.
The Go compiler resolves symbols across files in the same package, so moving
functions between files within a package requires zero code changes — only
ensuring the correct imports are present in each file.

---

### Task 1: Split parser/parser.go → parser_options.go

Move the functional options infrastructure out of `parser.go`.

**Files:**
- Modify: `parser/parser.go` (remove ~340 lines)
- Create: `parser/parser_options.go`

**Step 1: Create `parser/parser_options.go`**

Cut these sections from `parser/parser.go` into the new file:
- `Option` type alias (line 552-553)
- `parseConfig` struct (lines 555-583)
- `ParseWithOptions` func (lines 586-642)
- `applyOptions` func (lines 644-676)
- All 20 `With*` option constructors (lines 678-906):
  `WithFilePath`, `WithReader`, `WithBytes`, `WithResolveRefs`,
  `WithValidateStructure`, `WithUserAgent`, `WithHTTPClient`,
  `WithResolveHTTPRefs`, `WithInsecureSkipVerify`, `WithLogger`,
  `WithMaxRefDepth`, `WithMaxCachedDocuments`, `WithMaxFileSize`,
  `WithSourceMap`, `WithPreserveOrder`, `WithSourceName`

The new file needs this header:
```go
package parser

import (
	"crypto/tls"
	"io"
	"net/http"

	"github.com/erraggy/oastools/internal/options"
)
```

**Note:** Run `goimports` on both files after the split — it will add/remove imports automatically. The exact import list above is approximate; `goimports` is the source of truth.

**Step 2: Run diagnostics**

```bash
go build ./parser/...
```

Expected: clean build, zero errors.

**Step 3: Run tests**

```bash
go test ./parser/...
```

Expected: all tests pass, identical count to before.

**Step 4: Verify line counts**

```bash
wc -l parser/parser.go parser/parser_options.go
```

Expected: `parser.go` ~1,280 lines, `parser_options.go` ~340 lines. Neither over 1,000 — but `parser.go` is borderline and will be split further in Task 2.

**Step 5: Commit**

```bash
git add parser/parser.go parser/parser_options.go
git commit -m "refactor(parser): extract options to parser_options.go"
```

---

### Task 2: Split parser/parser.go → parser_format.go

Move format detection and URL fetching out of `parser.go`.

**Files:**
- Modify: `parser/parser.go` (remove ~130 lines)
- Create: `parser/parser_format.go`

**Step 1: Create `parser/parser_format.go`**

Cut these sections from `parser/parser.go` into the new file:
- `FormatBytes` func (lines 275-295)
- `detectFormatFromPath` func (lines 297-308)
- `detectFormatFromContent` func (lines 310-327)
- `isURL` func (lines 329-332)
- `fetchURL` method on `*Parser` (lines 334-397)
- `detectFormatFromURL` func (lines 398-428)

Header:
```go
package parser

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/erraggy/oastools/internal/httputil"
)
```

**Step 2: Run diagnostics**

```bash
go build ./parser/...
```

Expected: clean build.

**Step 3: Run tests**

```bash
go test ./parser/...
```

Expected: all tests pass.

**Step 4: Verify line counts**

```bash
wc -l parser/parser.go parser/parser_options.go parser/parser_format.go
```

Expected: `parser.go` ~1,150 lines or less, `parser_format.go` ~140 lines. If `parser.go` is still over 1,000, that's acceptable — the remaining code is cohesive core parsing logic (parse, resolve refs, validate structure).

**Step 5: Commit**

```bash
git add parser/parser.go parser/parser_format.go
git commit -m "refactor(parser): extract format detection to parser_format.go"
```

---

### Task 3: Split joiner/joiner.go → joiner_options.go

Move the functional options infrastructure out of `joiner.go`.

**Files:**
- Modify: `joiner/joiner.go` (remove ~340 lines)
- Create: `joiner/joiner_options.go`

**Step 1: Create `joiner/joiner_options.go`**

Cut these sections from `joiner/joiner.go` into the new file:
- `Option` type alias (lines 221-222)
- `joinConfig` struct (lines 224-262)
- `JoinWithOptions` func (lines 264-341)
- `joinWithoutOverlays` func (lines 342-368)
- `joinWithOverlays` func (lines 369-465)
- `parseOverlayList` func (lines 466-481)
- `mergeSpecOverlays` func (lines 482-506)
- `applyOptions` func (lines 507-527)
- Value helpers: `valueOrDefault`, `boolValueOrDefault`, `stringValueOrDefault`, `mapValueOrDefault` (lines 529-556)
- All `With*` option constructors (lines 558-783):
  `WithFilePaths`, `WithParsed`, `WithConfig`, `WithDefaultStrategy`,
  `WithPathStrategy`, `WithSchemaStrategy`, `WithComponentStrategy`,
  `WithDeduplicateTags`, `WithMergeArrays`, `WithRenameTemplate`,
  `WithNamespacePrefix`, `WithAlwaysApplyPrefix`, `WithEquivalenceMode`,
  `WithCollisionReport`, `WithSemanticDeduplication`, `WithOperationContext`,
  `WithPrimaryOperationPolicy`, `WithSourceMaps`, `WithCollisionHandler`,
  `WithCollisionHandlerFor`

Header:
```go
package joiner

import (
	"fmt"

	"github.com/erraggy/oastools/overlay"
	"github.com/erraggy/oastools/parser"
)
```

**Step 2: Run diagnostics**

```bash
go build ./joiner/...
```

Expected: clean build.

**Step 3: Run tests**

```bash
go test ./joiner/...
```

Expected: all tests pass.

**Step 4: Verify line counts**

```bash
wc -l joiner/joiner.go joiner/joiner_options.go
```

Expected: `joiner.go` ~900 lines, `joiner_options.go` ~340 lines.

**Step 5: Commit**

```bash
git add joiner/joiner.go joiner/joiner_options.go
git commit -m "refactor(joiner): extract options to joiner_options.go"
```

---

### Task 4: Split generator/security_gen_shared.go → server_gen_shared.go

The file has clear section dividers. Split along the "Server Generation Helpers" boundary.

**Files:**
- Modify: `generator/security_gen_shared.go` (remove ~520 lines)
- Create: `generator/server_gen_shared.go`

**Step 1: Create `generator/server_gen_shared.go`**

Cut the "Server Generation Helpers" section (lines 56-575) into the new file. This includes:
- `wrapperSuffixes` var and `resolveWrapperName` func (lines 60-83)
- `buildServerMethodSignature` func (lines 85-107)
- `clientMethodGenerator` type and `generateGroupClientMethods` func (lines 108-137)
- `writeNotImplementedError` func (lines 138-147)
- `generateServerMiddlewareShared` func (lines 148-169)
- `serverRouterContext` type and `generateServerRouterShared` func (lines 170-260)
- `serverStubsContext` type and `generateServerStubsShared` func (lines 261-336)
- `baseServerContext` type and `generateBaseServerShared` func (lines 337-471)
- "Split Server Helpers" section: `splitServerContext`, `generateSplitServerShared`, `serverGroupContext`, `generateServerGroupFileShared` (lines 472-575)

Header:
```go
package generator

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/erraggy/oastools/internal/maputil"
	"github.com/erraggy/oastools/parser"
)
```

Also move the "Client Generation Helpers" section (lines 616-651) into this file — it's the base client generation, which is a sibling concern to server generation.

**Step 2: Run diagnostics**

```bash
go build ./generator/...
```

Expected: clean build.

**Step 3: Run tests**

```bash
go test ./generator/...
```

Expected: all tests pass.

**Step 4: Verify line counts**

```bash
wc -l generator/security_gen_shared.go generator/server_gen_shared.go
```

Expected: `security_gen_shared.go` ~600 lines, `server_gen_shared.go` ~570 lines.

**Step 5: Commit**

```bash
git add generator/security_gen_shared.go generator/server_gen_shared.go
git commit -m "refactor(generator): extract server helpers to server_gen_shared.go"
```

---

### Task 5: Split integration/harness/pipeline.go

Split by step executor domain: parse/validate, transform, generate/build.

**Files:**
- Modify: `integration/harness/pipeline.go` (keep orchestrator + assertions)
- Create: `integration/harness/pipeline_parse_validate.go`
- Create: `integration/harness/pipeline_transform.go`
- Create: `integration/harness/pipeline_generate.go`

**Step 1: Create `pipeline_parse_validate.go`**

Cut these functions:
- `executeParse` (lines 111-147)
- `hasProblems` (lines 148-170)
- `executeValidate` (lines 172-211)

All new harness files need the build tag:
```go
//go:build integration

package harness
```

Imports for this file: `fmt`, `testing`, `parser`, `validator` (let `goimports` sort it out).

**Step 2: Create `pipeline_transform.go`**

Cut these functions:
- `executeFix` (lines 212-248)
- `getEnabledFixes`, `mapFixTypeName` (lines 249-298)
- `executeParseAll` (lines 300-350)
- `executeJoin`, `buildJoinerOptions` (lines 351-432)
- `executeConvert` (lines 434-481)
- `executeConvertAll` (lines 482-537)
- `containsSubstring`, `findSubstring` (lines 538-552)
- `executeFixAll` (lines 945-988)
- `executeOverlay` (lines 1221-end)
- `executeDiff` (lines 1138-1220)

Imports: `fmt`, `slices`, `strings`, `testing`, `converter`, `differ`, `fixer`, `joiner`, `overlay`, `parser`.

**Step 3: Create `pipeline_generate.go`**

Cut these functions:
- `executeGenerate` (lines 989-1071)
- `executeBuild` (lines 1072-1137)

Imports: `fmt`, `os`, `os/exec`, `path/filepath`, `strings`, `testing`, `time`, `generator`, `parser`.

**Step 4: What remains in `pipeline.go`**

- `ExecuteStep` orchestrator (lines 25-110) — the switch statement dispatching to step executors
- `checkAssertions` and `evaluateAssertion` (lines 553-816)
- `evaluateFixesApplied`, `evaluateNoFixesApplied` (lines 817-893)
- `countFixesByType` (lines 885-893)
- `checkSchemasExist`, `checkSchemasNotExist`, `getSchemaNames` (lines 894-943)

**Step 5: Run diagnostics**

```bash
go build -tags integration ./integration/harness/...
```

Expected: clean build.

**Step 6: Run tests**

Integration tests require the `integration` build tag:
```bash
go test -tags integration -count=1 -run TestPipeline ./integration/...
```

Expected: all pipeline tests pass. (If the full integration suite takes too long, run a single scenario to verify compilation.)

**Step 7: Verify line counts**

```bash
wc -l integration/harness/pipeline*.go
```

Expected: no file over 500 lines.

**Step 8: Commit**

```bash
git add integration/harness/pipeline*.go
git commit -m "refactor(harness): split pipeline.go by step executor domain"
```

---

### Task 6: Final verification for PR 1

**Step 1: Run full check suite**

```bash
make check
```

Expected: all checks pass (tests, lint, formatting).

**Step 2: Verify no file over 1,000 lines**

```bash
find parser joiner generator integration/harness -name '*.go' ! -name 'zz_*' -exec wc -l {} + | sort -rn | head -20
```

Expected: no non-generated file over 1,000 lines.

**Step 3: Create PR**

```bash
git push -u origin code-simplification
gh pr create --title "refactor: split oversized files for IDE/LLM ergonomics" --body "$(cat <<'EOF'
## Summary

- Split `parser/parser.go` (1,622 → 3 files): core, options, format detection
- Split `joiner/joiner.go` (1,236 → 2 files): core, options
- Split `generator/security_gen_shared.go` (1,194 → 2 files): security helpers, server/client helpers
- Split `integration/harness/pipeline.go` (1,294 → 4 files): orchestrator, parse/validate, transform, generate/build

## Details

Pure file moves within existing packages. No logic changes, no API changes, no test changes.
All functions remain in the same package — the Go compiler resolves symbols across files.

Design: docs/plans/2026-02-16-code-simplification-design.md

## Test plan

- [ ] `make check` passes
- [ ] All existing tests pass unchanged
- [ ] No production file over 1,000 lines (except generated `zz_*` files)
- [ ] Zero public API changes (verified by unchanged test compilation)

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## PR 2: Generator Deduplication

**Branch:** Create new branch from `main` after PR 1 merges (or from `code-simplification` if sequential).

### Key discovery: `baseCodeGenerator` already exists

The design doc proposed a new `sharedGenerator` struct, but `generator/base_code_generator.go` already has:

```go
type baseCodeGenerator struct {
    g              *Generator
    result         *GenerateResult
    schemaNames    map[string]string
    generatedTypes map[string]bool
    splitPlan      *SplitPlan
}
```

Both `oas2CodeGenerator` and `oas3CodeGenerator` embed it. We extend this existing struct with a `doc parser.DocumentAccessor` field and add shared methods to it.

---

### Task 7: Classify all 36 shared function pairs

Before writing any code, audit every function pair to classify it.

**Step 1: Generate side-by-side diff for each shared function name**

For each of the 36 shared function names between `oas2_generator.go` and `oas3_generator.go`, diff the bodies and classify as:

| Category | Criterion | Action |
|----------|-----------|--------|
| **Identical** | Bodies are character-for-character identical (ignoring receiver) | Move to `baseCodeGenerator` |
| **Data-access** | Bodies differ only in how a field is reached (e.g., `doc.SecurityDefinitions` vs `doc.Components.SecuritySchemes`) | Move to `baseCodeGenerator` using `DocumentAccessor` |
| **Structurally different** | Meaningfully different logic | Keep on version-specific generators |

**Step 2: Record classification**

Create a table in the commit message or a scratch file listing every function and its classification. This is the audit trail that every shared function was individually reviewed.

**Step 3: Commit the classification**

No code changes — just the classification document committed for review:
```bash
git commit --allow-empty -m "docs: classify 36 shared generator functions for dedup"
```

(Or include in the PR description.)

---

### Task 8: Add `DocumentAccessor` to `baseCodeGenerator`

**Files:**
- Modify: `generator/base_code_generator.go`
- Modify: `generator/oas2_generator.go` (constructor)
- Modify: `generator/oas3_generator.go` (constructor)

**Step 1: Add field to `baseCodeGenerator`**

In `generator/base_code_generator.go`, add a `doc` field:

```go
type baseCodeGenerator struct {
    g              *Generator
    result         *GenerateResult
    doc            parser.DocumentAccessor // version-agnostic document access
    schemaNames    map[string]string
    generatedTypes map[string]bool
    splitPlan      *SplitPlan
}
```

Update `initBase` to accept and store it:

```go
func (b *baseCodeGenerator) initBase(g *Generator, result *GenerateResult, doc parser.DocumentAccessor) {
    b.g = g
    b.result = result
    b.doc = doc
    // ... existing initialization
}
```

**Step 2: Update constructors**

In `newOAS2CodeGenerator`, pass `doc` (which implements `DocumentAccessor`) to `initBase`.
In `newOAS3CodeGenerator`, pass `doc` to `initBase`.

Both `OAS2Document` and `OAS3Document` already implement `DocumentAccessor`.

**Step 3: Run diagnostics**

```bash
go build ./generator/...
```

**Step 4: Run tests**

```bash
go test ./generator/...
```

**Step 5: Commit**

```bash
git add generator/base_code_generator.go generator/oas2_generator.go generator/oas3_generator.go
git commit -m "refactor(generator): add DocumentAccessor to baseCodeGenerator"
```

---

### Task 9: Move identical functions (batch 1 — security)

Move the ~8 security-related identical functions. These are confirmed identical from the prior audit in the brainstorming session.

**Files:**
- Modify: `generator/oas2_generator.go` (remove functions)
- Modify: `generator/oas3_generator.go` (remove functions)
- Modify: `generator/base_code_generator.go` (add methods)

**Step 1: Move these functions** (confirmed identical, only receiver differs):

- `generateSecurityHelpersFile` → `(b *baseCodeGenerator) generateSecurityHelpersFile`
- `generateOAuth2Files` → `(b *baseCodeGenerator) generateOAuth2Files`
- `generateCredentialsFile` → `(b *baseCodeGenerator) generateCredentialsFile`
- `generateSecurityEnforceFile` → `(b *baseCodeGenerator) generateSecurityEnforceFile`
- `generateSingleSecurityEnforce` → `(b *baseCodeGenerator) generateSingleSecurityEnforce`
- `generateSplitSecurityEnforce` → `(b *baseCodeGenerator) generateSplitSecurityEnforce`
- `generateOIDCDiscoveryFile` → `(b *baseCodeGenerator) generateOIDCDiscoveryFile`
- `generateReadmeFile` → `(b *baseCodeGenerator) generateReadmeFile`

For each function:
1. Copy the body from one version (e.g., OAS3) to `base_code_generator.go`
2. Change receiver from `(cg *oas3CodeGenerator)` to `(b *baseCodeGenerator)`
3. Replace any `cg.doc.Components.SecuritySchemes` with `b.doc.GetSecuritySchemes()`
4. Delete the function from both `oas2_generator.go` and `oas3_generator.go`

**Step 2: Run diagnostics**

```bash
go build ./generator/...
```

Fix any compilation errors. Common issues:
- Missing imports in `base_code_generator.go`
- Method calls on `cg.doc` that need to go through `b.doc` (the `DocumentAccessor` interface)

**Step 3: Run tests**

```bash
go test ./generator/...
```

Expected: all tests pass.

**Step 4: Commit**

```bash
git add generator/
git commit -m "refactor(generator): move identical security funcs to baseCodeGenerator"
```

---

### Task 10: Move identical functions (batch 2 — server/client)

**Step 1: Move these functions** (confirmed identical from audit):

- `generateServerMiddleware` → delegates to `generateServerMiddlewareShared` (already shared)
- `generateServerRouter` → delegates to `generateServerRouterShared`
- `generateServerStubs` → delegates to `generateServerStubsShared`
- `generateServerMethodSignature` → calls `buildServerMethodSignature` (already shared)
- `generateServerBinder` → builds context and calls shared func

For each: verify bodies are identical, change receiver, update field access.

**Step 2: Run diagnostics + tests**

```bash
go build ./generator/... && go test ./generator/...
```

**Step 3: Commit**

```bash
git add generator/
git commit -m "refactor(generator): move identical server/client funcs to baseCodeGenerator"
```

---

### Task 11: Move data-access functions

These functions differ only in how they access a field. With the `DocumentAccessor` interface on `baseCodeGenerator`, the version-specific access is abstracted away.

**Step 1: Move data-access functions one at a time**

For each function in this category:
1. Read both versions side-by-side
2. Identify the field access difference (e.g., `cg.doc.SecurityDefinitions` vs `cg.doc.Components.SecuritySchemes`)
3. Replace with the `DocumentAccessor` method (e.g., `b.doc.GetSecuritySchemes()`)
4. Verify no other differences exist
5. Move to `baseCodeGenerator`
6. Delete from both version-specific files

Likely candidates (verify during Task 7 classification):
- `generateSecurityHelpers` — differs only in scheme source
- `securityContext` — differs only in scheme source + global security source
- `operationsNeedTimeImport` — differs only in how operations are iterated

**Step 2: After each function, run diagnostics**

```bash
go build ./generator/...
```

**Step 3: After all moves, run tests**

```bash
go test ./generator/...
```

**Step 4: Commit**

```bash
git add generator/
git commit -m "refactor(generator): move data-access funcs to baseCodeGenerator via DocumentAccessor"
```

---

### Task 12: Final verification for PR 2

**Step 1: Run full check suite**

```bash
make check
```

**Step 2: Verify line counts**

```bash
wc -l generator/oas2_generator.go generator/oas3_generator.go generator/base_code_generator.go
```

Expected: `oas2_generator.go` ~400, `oas3_generator.go` ~500, `base_code_generator.go` grown but under 1,000.

**Step 3: Verify no structurally different functions were moved**

Review every function remaining in `oas2_generator.go` and `oas3_generator.go` — these should be the ~6 structurally different functions that legitimately need version-specific logic.

**Step 4: Run integration tests**

```bash
go test -tags integration -count=1 ./integration/...
```

**Step 5: Create PR**

```bash
gh pr create --title "refactor(generator): deduplicate OAS 2/3 generator code" --body "$(cat <<'EOF'
## Summary

Extract ~20-30 identical and data-access functions from `oas2_generator.go` and `oas3_generator.go` into shared methods on `baseCodeGenerator`, using the existing `DocumentAccessor` interface for version-agnostic field access.

## Changes

- Add `doc parser.DocumentAccessor` field to `baseCodeGenerator`
- Move identical functions (security helpers, server generation, client generation) to `baseCodeGenerator`
- Move data-access functions to `baseCodeGenerator` using `DocumentAccessor` methods
- Keep ~6 structurally different functions on version-specific generators

## Result

| File | Before | After |
|------|--------|-------|
| `base_code_generator.go` | ~30 lines | ~800 lines |
| `oas2_generator.go` | 1,170 lines | ~400 lines |
| `oas3_generator.go` | 1,342 lines | ~500 lines |

Net reduction: ~800 lines. Shared logic maintained in one place.

Design: docs/plans/2026-02-16-code-simplification-design.md

## Test plan

- [ ] `make check` passes
- [ ] All generator tests pass
- [ ] Integration tests pass
- [ ] No public API changes

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
