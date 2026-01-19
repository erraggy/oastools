# Pipeline Compositions

Demonstrates multi-step oastools workflows by chaining parser, fixer, converter, validator, joiner, and generator operations.

## What You'll Learn

- Chaining multiple oastools operations together
- Reusing parsed documents for efficiency (parse once, use many times)
- Common pipeline patterns for real-world scenarios
- Using `ToParseResult()` for seamless package chaining

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/workflows/pipeline-compositions
go run main.go
```

## Expected Output

```
Pipeline Compositions
=====================

Demonstrating multi-step oastools workflows

[1/3] Pipeline: Convert Legacy -> Validate -> Generate
-------------------------------------------------------
  Step 1: Parse OAS 2.0 spec
    ✓ Parsed: legacy-api.yaml (OAS 2.0)
  Step 2: Convert to OAS 3.0.3
    ✓ Converted to OAS 3.0.3
  Step 3: Validate converted spec
    ✓ Validation passed
  Step 4: Generate Go types
    ✓ Generated 1 files

  Result: Legacy OAS 2.0 -> OAS 3.0.3 -> Go types ✓

[2/3] Pipeline: Fix -> Validate
-------------------------------------------------------
  Step 1: Parse and identify issues
    ✓ Parsed: service-a.yaml
  Step 2: Validate (before fix)
    ✗ Found 2 validation errors
      - oas 3.0.3: duplicate operationId 'getItems' at 'paths./items/{itemId}.get'...
      - Duplicate operationId 'getItems' (first seen at paths./items.get)
  Step 3: Apply fixes
    ✓ Applied 1 fixes
      - renamed duplicate operationId "getItems" to "getItems2" (first occurrence at GET /items)
  Step 4: Validate (after fix)
    ✓ Validation passed

  Result: Spec with issues -> Fixed -> Valid ✓

[3/3] Pipeline: Fix All -> Join -> Validate -> Generate
-------------------------------------------------------
  Step 1: Parse all specs
    ✓ Parsed: service-a.yaml, service-b.yaml
  Step 2: Fix all specs
    ✓ Service A: 1 fixes applied
    ✓ Service B: 0 fixes applied
  Step 3: Join fixed specs
    ✓ Joined: OAS 3.0.3
  Step 4: Validate joined spec
    ✓ Validation passed
  Step 5: Generate Go code
    ✓ Generated 1 files

  Result: Multiple specs -> Fixed -> Joined -> Validated -> Generated ✓

=======================================================
Key Takeaways:
  - Chain operations for complex workflows
  - Parse once, reuse for multiple operations
  - Fix before join to ensure clean merge
  - Convert legacy specs before code generation
```

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates three pipeline patterns |
| specs/legacy-api.yaml | OAS 2.0 (Swagger) spec for conversion demo |
| specs/service-a.yaml | OAS 3.0.3 spec with duplicate operationId |
| specs/service-b.yaml | OAS 3.0.3 spec for joining demo |

## Pipeline Patterns

### Pattern 1: Convert Legacy -> Validate -> Generate

Upgrades a legacy Swagger 2.0 specification to OpenAPI 3.0.3, validates the converted result, and generates Go types.

```go
// Parse OAS 2.0
parsed, _ := parser.ParseWithOptions(parser.WithFilePath("swagger.yaml"))

// Convert to OAS 3.0.3
c := converter.New()
converted, _ := c.ConvertParsed(*parsed, "3.0.3")

// Validate converted spec
v := validator.New()
validation, _ := v.ValidateParsed(*converted.ToParseResult())

// Generate Go types
genResult, _ := generator.GenerateWithOptions(
    generator.WithParsed(*converted.ToParseResult()),
    generator.WithPackageName("api"),
)
```

### Pattern 2: Fix -> Validate

Identifies and automatically fixes issues in a specification, then validates the result.

```go
// Parse spec
parsed, _ := parser.ParseWithOptions(parser.WithFilePath("spec.yaml"))

// Fix issues
fixResult, _ := fixer.FixWithOptions(
    fixer.WithParsed(*parsed),
    fixer.WithEnabledFixes(fixer.FixTypeDuplicateOperationId),
)

// Validate fixed spec
v := validator.New()
validation, _ := v.ValidateParsed(*fixResult.ToParseResult())
```

### Pattern 3: Fix All -> Join -> Validate -> Generate

Fixes multiple specifications independently, joins them into a unified API, validates, and generates code.

```go
// Parse all specs
parsedA, _ := parser.ParseWithOptions(parser.WithFilePath("service-a.yaml"))
parsedB, _ := parser.ParseWithOptions(parser.WithFilePath("service-b.yaml"))

// Fix each spec
fixedA, _ := fixer.FixWithOptions(fixer.WithParsed(*parsedA), ...)
fixedB, _ := fixer.FixWithOptions(fixer.WithParsed(*parsedB), ...)

// Join fixed specs
joinResult, _ := joiner.JoinWithOptions(
    joiner.WithParsed(*fixedA.ToParseResult(), *fixedB.ToParseResult()),
    joiner.WithSchemaStrategy(joiner.StrategyAcceptLeft),
)

// Validate and generate
v := validator.New()
validation, _ := v.ValidateParsed(*joinResult.ToParseResult())
genResult, _ := generator.GenerateWithOptions(
    generator.WithParsed(*joinResult.ToParseResult()),
    generator.WithPackageName("unified"),
)
```

## Key Concepts

### ToParseResult() for Chaining

Each package result type provides a `ToParseResult()` method that converts the result to a `parser.ParseResult` for use with downstream packages:

| Package | Result Type | ToParseResult() |
|---------|-------------|-----------------|
| converter | `*ConversionResult` | ✓ |
| fixer | `*FixResult` | ✓ |
| joiner | `*JoinResult` | ✓ |

```go
// Chain converter -> validator
converted, _ := c.ConvertParsed(*parsed, "3.0.3")
validation, _ := v.ValidateParsed(*converted.ToParseResult())

// Chain fixer -> generator
fixResult, _ := fixer.FixWithOptions(...)
genResult, _ := generator.GenerateWithOptions(
    generator.WithParsed(*fixResult.ToParseResult()),
    ...
)
```

### Parse-Once Optimization

Parsing is expensive. For pipelines that need the same spec multiple times, parse once and reuse:

```go
// Parse once
parsed, _ := parser.ParseWithOptions(parser.WithFilePath("spec.yaml"))

// Reuse for multiple operations
v.ValidateParsed(*parsed)
f.FixParsed(*parsed)
c.ConvertParsed(*parsed, "3.1.0")
```

## Use Cases

- **CI/CD Pipelines**: Validate -> Fix -> Generate in automated builds
- **Legacy Migration**: Convert OAS 2.0 -> 3.x before generating modern clients
- **Microservice Consolidation**: Fix individual specs -> Join -> Generate unified SDK
- **Pre-commit Hooks**: Parse -> Validate -> Fix (dry-run) -> Report

## Next Steps

- [Validate and Fix](../validate-and-fix/) - Deep dive into the fixer package
- [Version Conversion](../version-conversion/) - Converting between OAS versions
- [Multi-API Merge](../multi-api-merge/) - Advanced joining strategies

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
