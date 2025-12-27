# Quickstart Example

A minimal 5-minute introduction to oastools, demonstrating the parse and validate workflow.

## What You'll Learn

- How to parse an OpenAPI specification with oastools
- How to validate a specification against OAS schema rules
- How to access parsed document structure programmatically

## Prerequisites

- Go 1.24+
- oastools module dependency (auto-fetched)

## Quick Start

1. Clone the repository:
   ```bash
   git clone https://github.com/erraggy/oastools.git
   cd oastools/examples/quickstart
   ```

2. Run the example:
   ```bash
   go run main.go
   ```

3. Expected output:
   ```
   oastools Quickstart
   ===================

   [1/3] Parsing OpenAPI specification...
         Version: 3.0.3
         Format: yaml

   [2/3] Validating against OAS schema...
         Valid: true
         Errors: 0
         Warnings: 0

   [3/3] Accessing document structure...
         API Title: Quickstart API
         API Version: 1.0.0
         Paths: 1
         Schemas: 1

         Paths defined:
           - /hello
         Schemas defined:
           - Greeting

   ---
   Quickstart complete!
   ```

## Files

| File | Purpose |
|------|---------|
| `spec.yaml` | Minimal OpenAPI 3.0.3 specification (27 lines) |
| `main.go` | Demonstrates parse → validate → inspect workflow |
| `go.mod` | Go module definition |

## Key Concepts

**Parse-Once Pattern**: The example uses `parser.ParseWithOptions()` to parse the specification once, then passes the result to `validator.ValidateParsed()`. This avoids re-parsing and provides significant performance improvement for repeated operations.

**Version-Agnostic Access**: The `AsAccessor()` method returns a `DocumentAccessor` interface that works identically for OAS 2.0, 3.0.x, 3.1.x, and 3.2.0 documents, abstracting away version-specific structure differences.

**Validation Severity Levels**: oastools uses four severity levels:
- `Critical` - Specification cannot be processed
- `Error` - Violates OpenAPI specification requirements
- `Warning` - Best practice recommendations
- `Info` - Informational notes

## Next Steps

- [Validation Pipeline Example](../validation-pipeline/) - Detailed validation workflow with source maps
- [PetStore Generator Example](../petstore/) - Generate Go client/server code
- [Parser Deep Dive](https://erraggy.github.io/oastools/packages/parser/)
- [Validator Deep Dive](https://erraggy.github.io/oastools/packages/validator/)

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
