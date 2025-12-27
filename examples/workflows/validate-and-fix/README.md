# Validate and Fix

Demonstrates automatic fixing of common OpenAPI validation errors using the fixer package.

## What You'll Learn

- How to identify validation errors in an OpenAPI specification
- Using dry-run mode to preview fixes before applying
- Applying fixes with type inference from naming conventions
- Selective fix application with `WithEnabledFixes()`

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/workflows/validate-and-fix
go run main.go
```

## Expected Output

```
Validate-and-Fix Workflow
=========================

[1/5] Parsing specification...
      Version: 3.0.3
      Format: yaml

[2/5] Validating (before fix)...
      Valid: false
      Errors: 3
        - Path template references parameter '{taskId}' but it is not declared in parameters
        - Path template references parameter '{projectId}' but it is not declared in parameters
        - Path template references parameter '{taskId}' but it is not declared in parameters

[3/5] Previewing fixes (dry-run)...
      Would apply 3 fix(es):
        - [missing-path-parameter] Added missing path parameter 'projectId' (type: integer)
        - [missing-path-parameter] Added missing path parameter 'taskId' (type: integer)
        - [missing-path-parameter] Added missing path parameter 'taskId' (type: integer)

[4/5] Applying fixes...
      Applied 4 fix(es)

[5/5] Validating (after fix)...
      Valid: true
      Errors: 0

---
Summary: Applied 4 fixes
  - [missing-path-parameter] Added missing path parameter 'projectId' (type: integer)
  - [missing-path-parameter] Added missing path parameter 'taskId' (type: integer)
  - [missing-path-parameter] Added missing path parameter 'taskId' (type: integer)
  - [pruned-unused-schema] removed unreferenced schema 'UnusedModel'

Specification is now valid!
```

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates the complete validate → fix → validate workflow |
| specs/invalid.yaml | OpenAPI spec with intentional validation issues |

## Key Concepts

### Type Inference

The fixer uses naming conventions to infer parameter types:
- `*Id` suffix → `integer` type
- `*Uuid` suffix → `string` with `uuid` format
- Other parameters → `string` type (default)

### Fix Types

| Fix Type | Description |
|----------|-------------|
| `missing-path-parameter` | Adds parameters declared in path but missing from operation |
| `pruned-unused-schema` | Removes schemas not referenced anywhere in the specification |

### Dry-Run Mode

Use `WithDryRun(true)` to preview what fixes would be applied without modifying the document. This is useful for:
- CI/CD pipelines that need to report issues
- Interactive tools that want user confirmation

## Next Steps

- [Fixer Deep Dive](https://erraggy.github.io/oastools/packages/fixer/) - Complete fixer documentation
- [Validation Pipeline](../../validation-pipeline/) - Validation with severity reporting
- [Version Conversion](../version-conversion/) - Convert between OAS versions

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
