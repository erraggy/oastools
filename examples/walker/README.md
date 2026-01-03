# Walker Examples

This directory contains examples demonstrating document traversal capabilities using the walker package for analysis, mutation, validation, filtering, and reporting.

## Available Examples

| Workflow | Category | Description | Time |
|----------|----------|-------------|------|
| [api-statistics](api-statistics/) | Analysis | Collect API statistics in single traversal pass | 3 min |
| [security-audit](security-audit/) | Validation | Audit API for security issues and compliance | 4 min |
| [vendor-extensions](vendor-extensions/) | Mutation | Add vendor extensions for downstream tooling | 3 min |
| [public-api-filter](public-api-filter/) | Filtering | Extract public API, filter internal endpoints | 3 min |
| [api-documentation](api-documentation/) | Reporting | Generate Markdown documentation from spec | 4 min |
| [reference-collector](reference-collector/) | Integration | Collect schema references and detect cycles | 4 min |

## Quick Start

Each example is a standalone Go module. To run any example:

```bash
cd examples/walker/<example-name>
go run main.go
```

## Workflow Overview

### API Statistics

The [api-statistics](api-statistics/) example demonstrates collecting comprehensive API metrics in a single traversal:

1. Register handlers for paths, operations, schemas, and parameters
2. Walk the document once
3. Aggregate counts, categorize endpoints, and compute statistics

**Use cases:** API complexity analysis, documentation metrics, governance reports

### Security Audit

The [security-audit](security-audit/) example shows how to audit APIs for security compliance:

1. Register handlers for security schemes, operations, and parameters
2. Detect missing authentication, sensitive data exposure, insecure patterns
3. Generate compliance reports with severity levels

**Use cases:** Security reviews, compliance checks, CI/CD security gates

### Vendor Extensions

The [vendor-extensions](vendor-extensions/) example demonstrates adding custom metadata:

1. Walk the document with mutation enabled
2. Add vendor extensions (x-*) to operations, schemas, and paths
3. Enrich specs for downstream tooling (code generators, gateways)

**Use cases:** Gateway configuration, code generator hints, documentation metadata

### Public API Filter

The [public-api-filter](public-api-filter/) example shows how to extract a subset of the API:

1. Walk the document and identify internal vs public endpoints
2. Use flow control to skip internal paths
3. Collect only public operations and their dependencies

**Use cases:** Public API documentation, partner API exports, SDK generation

### API Documentation

The [api-documentation](api-documentation/) example generates human-readable documentation:

1. Walk paths, operations, parameters, and response schemas
2. Build structured documentation model during traversal
3. Render to Markdown with proper formatting

**Use cases:** Auto-generated docs, README generation, API portals

### Reference Collector

The [reference-collector](reference-collector/) example demonstrates schema reference tracking:

1. Walk schemas and track $ref usage
2. Build dependency graphs between components
3. Detect circular references and unused schemas

**Use cases:** Dependency analysis, dead code detection, refactoring preparation

## Common Patterns

### Handler Registration

The walker uses functional options to register typed handlers:

```go
err := walker.Walk(parseResult,
    walker.WithPathHandler(func(pathTemplate string, pathItem *parser.PathItem, path string) walker.Action {
        fmt.Printf("Path: %s\n", pathTemplate)
        return walker.Continue
    }),
    walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
        fmt.Printf("  %s %s\n", method, path)
        return walker.Continue
    }),
)
```

### Flow Control

Control traversal with return values:

```go
walker.WithPathHandler(func(pathTemplate string, pathItem *parser.PathItem, path string) walker.Action {
    if strings.HasPrefix(pathTemplate, "/internal") {
        return walker.SkipChildren  // Skip operations under this path
    }
    if pathTemplate == "/admin" {
        return walker.Stop  // Stop entire traversal
    }
    return walker.Continue  // Process children normally
})
```

| Return Value | Behavior |
|--------------|----------|
| `walker.Continue` | Process this node and its children |
| `walker.SkipChildren` | Process this node, skip its children |
| `walker.Stop` | Stop traversal immediately |

### Mutation via Pointer Receivers

Handlers receive pointers, enabling in-place modification:

```go
walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
    if op.Extra == nil {
        op.Extra = make(map[string]any)
    }
    op.Extra["x-generated"] = time.Now().Format(time.RFC3339)
    return walker.Continue
})
```

## Next Steps

- [Walker Deep Dive](../../walker/deep_dive.md) - Complete package documentation
- [Workflow Examples](../workflows/) - Common OpenAPI transformation patterns
- [Getting Started](../quickstart/) - Basic parser and validator usage

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
