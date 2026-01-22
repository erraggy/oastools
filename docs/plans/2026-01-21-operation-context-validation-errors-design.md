# Operation Context for Validation Errors

**Date:** 2026-01-21
**Status:** Draft
**Author:** Claude + Robbie

## Overview

Improve validation error actionability by including API operation context for errors that stem from operations. Users will be able to immediately identify which endpoint is affected without mentally parsing JSON paths.

## Problem

Current validation errors show JSON paths like:
```
✗ paths./users/{id}.get.parameters[0]: Path parameters must have required: true
```

While accurate, users must mentally parse the path to understand:
- Which API endpoint is affected (`GET /users/{id}`)
- What the operation is called (`operationId`, if they're searching code)

For errors in shared components, it's even less clear which operations are impacted:
```
✗ components.schemas.User.properties.email: Invalid email format
```

## Solution

Add operation context inline after the JSON path for operation-related errors.

### Display Format

**Operation-level issues** (under `paths.{path}.{method}.*`):
```
✗ paths./users/{id}.get.parameters[0] (operationId: getUser): Path parameters must have required: true
```

If no operationId:
```
✗ paths./users/{id}.get.parameters[0] (GET /users/{id}): Path parameters must have required: true
```

**Path-level issues** (under `paths.{path}.parameters` or `paths.{path}` directly):
```
✗ paths./users/{id}.parameters[0] (path: /users/{id}): Parameter 'id' missing schema
```

**Reusable component issues** (under `components.*`, `definitions.*`, etc.):
```
✗ components.schemas.User.properties.email (operationId: getUser, +3 operations): Invalid email format
```

If the referencing operation has no operationId:
```
✗ components.schemas.User.properties.email (GET /users, +3 operations): Invalid email format
```

If a component is **never referenced** by any operation:
```
✗ components.schemas.OrphanedSchema.type (unused component): Type must be a valid JSON Schema type
```

**No context** (top-level issues like `info.*`, `servers.*`):
```
✗ info.version: Info object must have a version
```
_(no parenthetical — context doesn't apply)_

## Data Model

### New `OperationContext` struct

```go
// OperationContext provides API operation context for validation issues.
// For issues under paths.*, it identifies the specific operation.
// For issues outside paths.*, it shows which operations reference the component.
type OperationContext struct {
    // Method is the HTTP method (GET, POST, etc.) - empty for path-level issues
    Method string
    // Path is the API path pattern (e.g., "/users/{id}")
    Path string
    // OperationID is the operationId if defined (may be empty)
    OperationID string
    // IsReusableComponent is true when the issue is in components/definitions
    IsReusableComponent bool
    // AdditionalRefs is the count of other operations referencing this component
    // (only relevant when IsReusableComponent is true)
    AdditionalRefs int
}
```

### Updated `Issue` struct

```go
type Issue struct {
    // ... existing fields ...

    // OperationContext provides API operation context when the issue relates to
    // an operation or a component referenced by operations. Nil when not applicable.
    OperationContext *OperationContext
}
```

## Implementation Approach

### Phase 1: Build Reference Map (pre-validation)

Before validation begins, walk the document to build a map of component → operations:

```go
type refTracker struct {
    // componentToOps maps component paths to the operations that reference them
    // e.g., "components.schemas.User" → [{Method: "GET", Path: "/users", OperationID: "getUsers"}, ...]
    componentToOps map[string][]OperationRef
}
```

This walk visits every `$ref` in the document and records:
- Which component path it points to (normalized, e.g., `#/components/schemas/User` → `components.schemas.User`)
- Which operation contains this reference (tracked via traversal context)

### Phase 2: Attach Context During Validation

When creating a `ValidationError`, the validator:

1. Parses the error's `Path` to determine if it's under `paths.*` or elsewhere
2. If under `paths.*`: extracts method, path, and looks up operationId from the document
3. If elsewhere: looks up the component in `refTracker.componentToOps` to find referencing operations

**Performance**: The reference map is built once per validation. For large specs (10K+ paths), this adds a small upfront cost but keeps per-error lookups O(1).

## Code Organization

| File | Purpose |
|------|---------|
| `internal/issues/operation_context.go` | `OperationContext` struct, `String()` method, formatting helpers |
| `internal/issues/issue.go` | Add `OperationContext` field, update `Issue.String()` |
| `validator/ref_tracker.go` | `refTracker` struct, `buildRefMap()`, `lookupOperationContext()` |
| `validator/validator.go` | Initialize tracker, pass to validation functions |
| `validator/context_helpers.go` | `attachOperationContext()`, path parsing helpers |
| `validator/deep_dive.md` | Documentation update with examples |

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| Circular `$ref` chains | `refTracker` uses visited set to avoid infinite loops |
| External `$ref` (file/URL) | Track as separate component; context shows "external ref" if not resolvable |
| Component referenced by 0 operations | Show `(unused component)` — helpful for detecting dead schemas |
| Component referenced by 100+ operations | Cap display at `+99 operations` to avoid clutter |
| Deeply nested `$ref` (A→B→C) | Track transitive references — if op uses A, and A refs B, B gets that op context |
| OAS2 `definitions` vs OAS3 `components` | Normalize both to same tracking logic; path prefix differs but structure is same |
| Webhooks (OAS 3.1+) | Treat like operations — `webhooks.orderCreated.post` gets context `(webhook: orderCreated)` |

## Testing Strategy

1. **Unit tests for `refTracker`** — verify map building with various ref patterns
2. **Unit tests for path parsing** — ensure correct extraction of method/path from JSON paths
3. **Unit tests for `Issue.String()`** — verify all format variations render correctly
4. **Integration tests** — validate real specs and verify errors have expected context
5. **Golden file tests** — capture expected CLI output for regression testing
6. **Benchmark** — ensure ref tracking doesn't regress validation performance on large specs (target: <5% overhead)

## Scope

- Applies to both errors and warnings (consistent experience)
- Supports OAS 2.0 and OAS 3.x
- Affects CLI output and programmatic `Issue` struct (JSON output includes `OperationContext`)

## Non-Goals

- Changing the JSON path itself (kept for tooling compatibility)
- Adding context for non-operation-related issues (e.g., `info`, `servers`)
- Making context display configurable (always shown when applicable)

---

## Implementation Plan (Orchestrator Mode)

> **Note:** Implementation uses Orchestrator Mode to protect context. The orchestrator coordinates agents; agents do the heavy lifting. No worktrees — work happens on the current feature branch.

### Step 1: Data Model (`developer` agent)

**Task:** Create `OperationContext` struct and update `Issue`

- Create `internal/issues/operation_context.go` with `OperationContext` struct
- Add `OperationContext *OperationContext` field to `Issue` in `issue.go`
- Implement `OperationContext.String()` for display formatting
- Update `Issue.String()` to include operation context when present
- Add unit tests for all formatting variations

**Acceptance criteria:**
- `go_diagnostics` passes
- Unit tests cover all display format variations from design
- Existing tests still pass

### Step 2: Reference Tracker (`developer` agent)

**Task:** Build the `refTracker` that maps components to operations

- Create `validator/ref_tracker.go` with `refTracker` struct
- Implement `buildRefMap()` that walks the document and collects `$ref` → operation mappings
- Handle OAS2 (`definitions`, `parameters`, `responses`) and OAS3 (`components.*`)
- Handle circular refs with visited set
- Handle transitive refs (A→B→C all get tracked)
- Add comprehensive unit tests

**Acceptance criteria:**
- `go_diagnostics` passes
- Unit tests cover: simple refs, circular refs, transitive refs, OAS2, OAS3
- Benchmark shows <5% overhead on large specs

### Step 3: Context Attachment (`developer` agent)

**Task:** Wire up context attachment during validation

- Create `validator/context_helpers.go` with:
  - `attachOperationContext(issue, path, tracker, doc)`
  - `parsePathComponents(path)` → extracts API path, method, isPathLevel
  - `extractOperationID(doc, apiPath, method)`
- Modify `validator/validator.go`:
  - Initialize `refTracker` in `ValidateParsed()`
  - Call `attachOperationContext()` in `addError()` and `addWarning()`
- Add integration tests with real specs

**Acceptance criteria:**
- `go_diagnostics` passes
- Integration tests verify errors have correct operation context
- All existing validator tests pass

### Step 4: Documentation & Review (`maintainer` agent)

**Task:** Update docs and review implementation

- Update `validator/deep_dive.md` with operation context section
- Review all new code for:
  - Security issues
  - Performance concerns
  - Code quality and consistency with codebase patterns
- Run full test suite and benchmarks

**Acceptance criteria:**
- Documentation is clear and includes examples
- Code review passes with no blocking issues
- `make check` passes
- Benchmarks show acceptable performance

### Step 5: Final Verification

**Task:** Orchestrator verifies everything works end-to-end

- Run `go run ./cmd/oastools validate testdata/invalid-oas3.yaml` and verify output includes operation context
- Run `make check` to ensure all tests and linting pass
- Review CLI output format matches design specification
