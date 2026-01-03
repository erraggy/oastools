# oastools Examples

Complete, runnable examples demonstrating the full oastools ecosystem across parsing, validation, transformation, and code generation.

## Quick Start

| Time | Category | Example | Description |
|------|----------|---------|-------------|
| 2 min | Getting Started | [quickstart/](quickstart/) | Parse and validate a minimal spec |
| 5 min | Getting Started | [validation-pipeline/](validation-pipeline/) | Complete validation with error reporting |
| 3 min | Workflows | [validate-and-fix/](workflows/validate-and-fix/) | Auto-fix common spec errors |
| 3 min | Workflows | [version-conversion/](workflows/version-conversion/) | Convert OAS 2.0 → 3.0.3 |
| 4 min | Workflows | [multi-api-merge/](workflows/multi-api-merge/) | Merge microservice specs |
| 4 min | Workflows | [breaking-change-detection/](workflows/breaking-change-detection/) | CI/CD breaking change gates |
| 3 min | Workflows | [overlay-transformations/](workflows/overlay-transformations/) | Environment-specific customizations |
| 5 min | Workflows | [http-validation/](workflows/http-validation/) | Runtime request/response validation |
| 5 min | Programmatic API | [builder/](programmatic-api/builder/) | Build specs from Go code + ServerBuilder |
| 3 min | Walker | [api-statistics/](walker/api-statistics/) | Collect API statistics in single pass |
| 4 min | Walker | [security-audit/](walker/security-audit/) | Audit API for security issues |
| 3 min | Walker | [vendor-extensions/](walker/vendor-extensions/) | Add vendor extensions for tooling |
| 3 min | Walker | [public-api-filter/](walker/public-api-filter/) | Filter internal endpoints |
| 4 min | Walker | [api-documentation/](walker/api-documentation/) | Generate Markdown documentation |
| 4 min | Walker | [reference-collector/](walker/reference-collector/) | Analyze schema references |
| 10 min | Code Generation | [petstore/](petstore/) | Full client/server generation |

## Examples by Category

### Getting Started

Best for first-time users learning oastools.

| Example | Description |
|---------|-------------|
| [quickstart/](quickstart/) | 100-line example demonstrating parse → validate workflow |
| [validation-pipeline/](validation-pipeline/) | Complete validation with source maps and severity classification |

### [Workflow Examples](workflows/)

Common OpenAPI transformation patterns covering 6 packages.

| Example | Package | Description |
|---------|---------|-------------|
| [validate-and-fix/](workflows/validate-and-fix/) | fixer | Parse, validate, auto-fix common errors |
| [version-conversion/](workflows/version-conversion/) | converter | Convert OAS 2.0 (Swagger) → OAS 3.0.3 |
| [multi-api-merge/](workflows/multi-api-merge/) | joiner | Merge specs with collision resolution |
| [breaking-change-detection/](workflows/breaking-change-detection/) | differ | Detect breaking changes between versions |
| [overlay-transformations/](workflows/overlay-transformations/) | overlay | Apply JSONPath-based transformations |
| [http-validation/](workflows/http-validation/) | httpvalidator | Runtime HTTP request/response validation |

### [Programmatic API](programmatic-api/)

Build OpenAPI specifications from Go code.

| Example | Package | Description |
|---------|---------|-------------|
| [builder/](programmatic-api/builder/) | builder | Fluent API + ServerBuilder for runnable servers |

### [Walker Examples](walker/)

Document traversal patterns for analysis, mutation, and filtering.

| Example | Category | Description |
|---------|----------|-------------|
| [api-statistics/](walker/api-statistics/) | Analysis | Collect API statistics in a single traversal pass |
| [security-audit/](walker/security-audit/) | Validation | Audit API for security issues and compliance |
| [vendor-extensions/](walker/vendor-extensions/) | Mutation | Add vendor extensions for downstream tooling |
| [public-api-filter/](walker/public-api-filter/) | Filtering | Extract public API, filter internal endpoints |
| [api-documentation/](walker/api-documentation/) | Reporting | Generate Markdown documentation from spec |
| [reference-collector/](walker/reference-collector/) | Integration | Collect schema references and detect cycles |

### Code Generation

Generate production-ready Go client and server code.

| Example | Description |
|---------|-------------|
| [petstore/](petstore/) | Complete code generation with OAuth2, OIDC, chi router |

## Feature Matrix

| Feature | quickstart | validation-pipeline | workflows | builder | walker | petstore |
|---------|:----------:|:-------------------:|:---------:|:-------:|:------:|:--------:|
| Parser API | ✓ | ✓ | ✓ | ✓ | | |
| Validator API | ✓ | ✓ | ✓ | ✓ | | |
| Fixer API | | | ✓ | | | |
| Converter API | | | ✓ | | | |
| Joiner API | | | ✓ | | | |
| Differ API | | | ✓ | | | |
| Overlay API | | | ✓ | | | |
| HTTPValidator API | | | ✓ | | | |
| Builder API | | | | ✓ | | |
| ServerBuilder | | | | ✓ | | |
| Walker API | | | | | ✓ | |
| Source Maps | | ✓ | | | | |
| Code Generation | | | | | | ✓ |
| Client Generation | | | | | | ✓ |
| Server Generation | | | | | | ✓ |
| OAuth2 Flows | | | | | | ✓ |
| OIDC Discovery | | | | | | ✓ |

**Package Coverage:** 11/11 packages demonstrated

## OAS Version Coverage

| Version | Examples |
|---------|----------|
| OAS 2.0 (Swagger) | petstore, version-conversion |
| OAS 3.0.x | quickstart, all workflows |
| OAS 3.2.0 | builder |
| Any version | validation-pipeline (accepts any OAS file) |

## Running Examples

Each example is a standalone Go module. To run any example:

```bash
cd examples/<category>/<example-name>
go run main.go
```

Or build and run:

```bash
cd examples/<category>/<example-name>
go build -o example .
./example
```

## Common Patterns

### Parse-Once Optimization

All workflow examples demonstrate the parse-once pattern:

```go
parsed, _ := parser.ParseWithOptions(parser.WithFilePath("spec.yaml"))

// Reuse for multiple operations (9-154x faster)
fixer.FixWithOptions(fixer.WithParsed(parsed))
validator.ValidateWithOptions(validator.WithParsed(parsed))
```

### Functional Options

All packages use the functional options pattern:

```go
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("swagger.yaml"),
    converter.WithTargetVersion("3.0.3"),
)
```

## Learn More

- [CLI Reference](../cli-reference/)
- [Developer Guide](../developer-guide/)
- Package Documentation:
  - [parser](../packages/parser/)
  - [validator](../packages/validator/)
  - [fixer](../packages/fixer/)
  - [converter](../packages/converter/)
  - [joiner](../packages/joiner/)
  - [differ](../packages/differ/)
  - [overlay](../packages/overlay/)
  - [httpvalidator](../packages/httpvalidator/)
  - [builder](../packages/builder/)
  - [walker](../packages/walker/)
  - [generator](../packages/generator/)

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
