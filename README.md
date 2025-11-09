# oastools

OpenAPI Specification (OAS) tools for validation, parsing, and code generation.

## Features

- **Validate** - Validate OpenAPI specification files for correctness
- **Parse** - Parse and analyze OpenAPI specifications
- **Generate** - Generate code from OpenAPI specifications (planned)

## Installation

### From Source

```bash
git clone https://github.com/erraggy/oastools.git
cd oastools
make install
```

### Using Go

```bash
go install github.com/erraggy/oastools/cmd/oastools@latest
```

## Usage

```bash
# Show help
oastools help

# Validate an OpenAPI spec
oastools validate openapi.yaml

# Parse an OpenAPI spec
oastools parse openapi.yaml

# Generate code from a spec
oastools generate --lang go openapi.yaml
```

## Development

### Prerequisites

- Go 1.21 or higher
- make (optional, but recommended)

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Run linter
make lint

# Format code
make fmt
```

### Project Structure

```
.
├── cmd/oastools/       # CLI entry point
├── internal/           # Private application code
│   ├── validator/      # OpenAPI validation logic
│   ├── parser/         # OpenAPI parsing logic
│   └── generator/      # Code generation logic
├── pkg/                # Public library code
└── testdata/           # Test fixtures and sample specs
```

## License

MIT
