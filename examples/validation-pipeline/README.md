# Validation Pipeline Example

Demonstrates a complete validation pipeline with source map integration for IDE-friendly line numbers in error messages.

## What You'll Learn

- How to parse and validate any OpenAPI specification
- How to enable source maps for line numbers in errors
- How to classify and report validation issues by severity
- The parse-once optimization pattern for better performance

## Prerequisites

- Go 1.24+
- An OpenAPI specification file to validate

## Quick Start

1. Run against the petstore spec:
   ```bash
   cd examples/validation-pipeline
   go run main.go ../petstore/spec/petstore-v2.json
   ```

2. Expected output:
   ```
   Validation Pipeline
   ===================

   Input: ../petstore/spec/petstore-v2.json

   [1/3] Parsing specification...
         OAS Version: 2.0
         Format: json
         Size: 13.5 KiB

   [2/3] Validating against OpenAPI schema...

   [3/3] Validation Results
         Valid: true
         Errors: 0
         Warnings: 0

   ---
   Validation PASSED
   ```

3. Try with your own specification:
   ```bash
   go run main.go /path/to/your/openapi.yaml
   ```

## Files

| File | Purpose |
|------|---------|
| `main.go` | Complete validation pipeline with error reporting |
| `go.mod` | Go module definition |

## Key Concepts

**Source Map Integration**: By enabling `parser.WithSourceMap(true)` and passing it to the validator via `v.SourceMap = result.SourceMap`, validation errors include line numbers. This is essential for IDE integration and debugging.

**Severity Levels**: oastools uses four severity levels:
- `CRITICAL` - Specification cannot be processed
- `ERROR` - Violates OpenAPI specification requirements
- `WARNING` - Best practice recommendations
- `INFO` - Informational notes

**Parse-Once Pattern**: The example parses once with `ParseWithOptions()` then validates with `ValidateParsed()`. This avoids re-parsing the document and provides 9-154x performance improvement for multi-step workflows.

**Exit Codes**: The example exits with code 0 for success and code 1 for validation failures, making it suitable for CI/CD pipelines.

## CLI Equivalent

This example replicates the functionality of:

```bash
oastools validate --source-map openapi.yaml
```

## Next Steps

- [Quickstart Example](../quickstart/) - Minimal introduction
- [PetStore Example](../petstore/) - Code generation
- [Validator Deep Dive](https://erraggy.github.io/oastools/packages/validator/)

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
