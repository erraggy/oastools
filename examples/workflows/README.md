# Workflow Examples

This directory contains examples demonstrating common OpenAPI workflows using the oastools packages.

## Available Workflows

| Workflow | Package | Description | Time |
|----------|---------|-------------|------|
| [validate-and-fix](validate-and-fix/) | fixer | Parse, validate, auto-fix common errors | 3 min |
| [version-conversion](version-conversion/) | converter | Convert OAS 2.0 (Swagger) → OAS 3.0.3 | 3 min |
| [multi-api-merge](multi-api-merge/) | joiner | Merge multiple specs with collision resolution | 4 min |
| [breaking-change-detection](breaking-change-detection/) | differ | Detect breaking changes between API versions | 4 min |
| [overlay-transformations](overlay-transformations/) | overlay | Apply environment-specific customizations | 3 min |
| [http-validation](http-validation/) | httpvalidator | Runtime HTTP request/response validation | 5 min |

## Quick Start

Each example is a standalone Go module. To run any example:

```bash
cd examples/workflows/<workflow-name>
go run main.go
```

## Workflow Overview

### Validate and Fix

The [validate-and-fix](validate-and-fix/) workflow shows how to automatically repair common OpenAPI spec issues:

1. Parse the specification
2. Validate and identify errors
3. Preview fixes with dry-run mode
4. Apply fixes automatically
5. Re-validate to confirm resolution

**Use cases:** CI/CD pre-commit hooks, spec cleanup automation

### Version Conversion

The [version-conversion](version-conversion/) workflow demonstrates OAS version migration:

1. Parse OAS 2.0 (Swagger) specification
2. Convert to OAS 3.0.3
3. Track conversion issues and warnings
4. Access the converted document

**Use cases:** Legacy API migration, spec modernization

### Multi-API Merge

The [multi-api-merge](multi-api-merge/) workflow shows how to combine microservice specs:

1. Parse multiple OpenAPI specs
2. Configure collision resolution strategies
3. Merge with semantic deduplication
4. Handle path and schema conflicts

**Use cases:** API gateway specs, unified documentation, monorepo builds

### Breaking Change Detection

The [breaking-change-detection](breaking-change-detection/) workflow implements CI/CD quality gates:

1. Parse base and target specifications
2. Compare for breaking changes
3. Categorize by severity (CRITICAL, ERROR, WARNING, INFO)
4. Generate reports for PR reviews

**Use cases:** CI/CD gates, release validation, API governance

### Overlay Transformations

The [overlay-transformations](overlay-transformations/) workflow applies environment-specific changes:

1. Parse base specification
2. Load overlay document with JSONPath actions
3. Preview changes in dry-run mode
4. Apply transformations

**Use cases:** Multi-environment configs, security additions, filtering internal endpoints

### HTTP Validation

The [http-validation](http-validation/) workflow validates runtime HTTP traffic:

1. Parse specification
2. Create HTTP validator
3. Validate requests (path, query, body)
4. Extract typed path parameters
5. Validate responses

**Use cases:** Request validation middleware, API testing, contract compliance

## Common Patterns

### Parse-Once Optimization

All workflows demonstrate the parse-once pattern for maximum performance:

```go
// Parse once
parsed, _ := parser.ParseWithOptions(parser.WithFilePath("spec.yaml"))

// Reuse for multiple operations
fixer.FixWithOptions(fixer.WithParsed(parsed))
validator.ValidateWithOptions(validator.WithParsed(parsed))
```

This avoids re-parsing the same spec, providing 9-154x performance improvements.

### Functional Options

All packages use the functional options pattern for clean, extensible configuration:

```go
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("swagger.yaml"),
    converter.WithTargetVersion("3.0.3"),
)
```

### Error Handling

All workflows include proper error handling with rich error types:

```go
result, err := differ.DiffWithOptions(...)
if err != nil {
    log.Fatal(err)
}

// Iterate over changes and filter by breaking status
for _, change := range result.Changes {
    if result.HasBreakingChanges {
        fmt.Printf("[%s] %s: %s\n",
            change.Category,
            change.Severity,
            change.Message)
    }
}
```

## Next Steps

- [Getting Started Examples](../quickstart/) - Basic parser and validator usage
- [Programmatic API](../programmatic-api/) - Build specs from Go code
- [Code Generation](../petstore/) - Generate client/server code

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
