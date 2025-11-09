# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`oastools` is a Go-based command-line tool for working with OpenAPI Specification (OAS) files. The primary goals are:
- Validating OpenAPI specification files
- Parsing and analyzing OAS documents
- Generating code from OpenAPI specifications

## Development Commands

### Building and Running
```bash
# Build the binary (output: bin/oastools)
make build

# Install to $GOPATH/bin
make install

# Run the binary directly
./bin/oastools [command]
```

### Testing
```bash
# Run all tests with race detection and coverage
make test

# Generate and view HTML coverage report
make test-coverage
```

### Code Quality
```bash
# Format all Go code
make fmt

# Run go vet
make vet

# Run golangci-lint (requires golangci-lint to be installed)
make lint
```

### Dependency Management
```bash
# Download and tidy dependencies
make deps
```

### Cleanup
```bash
# Remove build artifacts and coverage reports
make clean
```

## Architecture

### Directory Structure

- **cmd/oastools/** - CLI entry point with command routing and user interface
  - `main.go` contains the command dispatcher and usage information

- **internal/** - Private application code not importable by other projects
  - `validator/` - Logic for validating OpenAPI specifications against the spec schema
  - `parser/` - Logic for parsing YAML/JSON OAS files into Go structures
  - `generator/` - Logic for generating code (clients, servers, models) from OAS files

- **pkg/** - Public library code that could be imported by external projects
  - Currently unused, but reserved for any public APIs

- **testdata/** - Test fixtures including sample OpenAPI specification files

### Design Patterns

- **Internal packages**: All core logic is in `internal/` to maintain encapsulation and prevent external dependencies on unstable APIs
- **Separation of concerns**: Each package has a single, well-defined responsibility
- **CLI structure**: Simple command dispatcher in main.go that delegates to internal packages

### Extension Points

When adding new commands:
1. Add the command case to the switch statement in `cmd/oastools/main.go`
2. Create corresponding logic in the appropriate `internal/` package
3. Update the `printUsage()` function to document the new command
4. Add test files in the same package as the implementation

### Testing Strategy

- Unit tests live alongside implementation files (e.g., `validator.go` → `validator_test.go`)
- Integration tests should use fixtures from `testdata/`
- Run tests with race detection enabled to catch concurrency issues
- Aim for high test coverage, especially for validation and parsing logic

## Go Module

- Module path: `github.com/erraggy/oastools`
- Minimum Go version: 1.21
