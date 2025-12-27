# oastools Examples

Working examples demonstrating oastools capabilities across parsing, validation, and code generation.

## Quick Start

| Time | Example | Description |
|------|---------|-------------|
| 2 min | [quickstart/](quickstart/) | Parse and validate a minimal spec |
| 5 min | [validation-pipeline/](validation-pipeline/) | Complete validation with error reporting |
| 10 min | [petstore/](petstore/) | Full code generation with multiple router variants |

## Examples Overview

### [quickstart/](quickstart/)

**Best for:** First-time users learning oastools

A minimal example with a 27-line OpenAPI 3.0.3 specification demonstrating the core parse → validate workflow in about 100 lines of Go code.

```bash
cd examples/quickstart && go run main.go
```

### [validation-pipeline/](validation-pipeline/)

**Best for:** CI/CD integration and API governance

Demonstrates a complete validation pipeline with source map integration for line numbers, severity classification, and detailed error reporting.

```bash
cd examples/validation-pipeline
go run main.go ../petstore/spec/petstore-v2.json
```

### [petstore/](petstore/)

**Best for:** Users interested in code generation

Complete client and server generation from the Swagger Petstore API (OAS 2.0), including:

| Variant | Router | Location |
|---------|--------|----------|
| stdlib | net/http | [petstore/stdlib/](petstore/stdlib/) |
| chi | go-chi/chi | [petstore/chi/](petstore/chi/) |

Both variants include OAuth2 flows, OIDC discovery, credential management, and security enforcement.

## Feature Matrix

| Feature | quickstart | validation-pipeline | petstore |
|---------|:----------:|:-------------------:|:--------:|
| Parser API | ✓ | ✓ | |
| Validator API | ✓ | ✓ | |
| Source Maps | | ✓ | |
| Code Generation | | | ✓ |
| Client Generation | | | ✓ |
| Server Generation | | | ✓ |
| OAuth2 Flows | | | ✓ |
| OIDC Discovery | | | ✓ |
| Chi Router | | | ✓ |

## OAS Version Coverage

| Version | Example |
|---------|---------|
| OAS 2.0 (Swagger) | petstore |
| OAS 3.0.3 | quickstart |
| Any version | validation-pipeline (accepts any OAS file as input) |

## Running Examples

Each example is a standalone Go module. To run any example:

```bash
cd examples/<example-name>
go run main.go
```

Or build and run:

```bash
cd examples/<example-name>
go build -o example .
./example
```

## Learn More

- [CLI Reference](https://erraggy.github.io/oastools/cli-reference/)
- [Parser Documentation](https://erraggy.github.io/oastools/packages/parser/)
- [Validator Documentation](https://erraggy.github.io/oastools/packages/validator/)
- [Generator Documentation](https://erraggy.github.io/oastools/packages/generator/)

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
