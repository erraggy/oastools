# GEMINI.md

This document provides a comprehensive overview of the `oastools` project, designed to facilitate understanding and contribution for Gemini agents.

## Project Overview

`oastools` is a Go-based command-line interface (CLI) and library for working with OpenAPI specifications (OAS). It provides a rich set of features for parsing, validating, converting, diffing, joining, and building OAS documents, as well as generating Go code from them. The project supports all OpenAPI versions from 2.0 to 3.2.0.

The project is structured as a Go module with several public packages, each corresponding to a major feature:

*   `parser`: For parsing and analyzing OpenAPI specifications.
*   `validator`: For validating the correctness of OpenAPI specifications.
*   `fixer`: For automatically fixing common validation errors in OpenAPI specifications.
*   `converter`: For converting between different OpenAPI versions (e.g., 2.0 to 3.x).
*   `joiner`: For merging multiple OpenAPI specifications.
*   `differ`: For comparing OpenAPI specifications and detecting breaking changes.
*   `generator`: For generating idiomatic Go code (clients, servers, types) from OpenAPI specifications.
*   `builder`: For programmatically constructing OpenAPI specifications.

The project also includes a CLI tool, `oastools`, which exposes the library's functionality on the command line.

## Building and Running

The project uses a `Makefile` to streamline common development tasks.

### Key Commands

*   **Build the CLI:**
    ```bash
    make build
    ```
    This command compiles the `oastools` CLI and places the binary in the `bin/` directory.

*   **Run tests:**
    ```bash
    make test
    ```
    This command runs the test suite for all packages.

*   **Run tests with race detection:**
    ```bash
    make test-race
    ```

*   **Generate a test coverage report:**
    ```bash
    make test-coverage
    ```
    This will generate an HTML coverage report named `coverage.html`.

*   **Lint the code:**
    ```bash
    make lint
    ```
    This command runs `golangci-lint` to check for style and formatting issues.

*   **Format the code:**
    ```bash
    make fmt
    ```

*   **Run all checks:**
    ```bash
    make check
    ```
    This is a convenience target that runs `tidy`, `fmt`, `lint`, and `test`.

*   **Install the CLI:**
    ```bash
    make install
    ```
    This command installs the `oastools` binary in your `$GOPATH/bin`.

## Development Conventions

*   **Code Style:** The project follows standard Go formatting and style. Use `make fmt` to format the code before committing.
*   **Testing:** All new features and bug fixes should be accompanied by tests. The project uses the standard Go testing framework and `stretchr/testify` for assertions.
*   **Commit Messages:** Follow conventional commit message formats (e.g., `feat(parser): add support for OAS 3.1.0`).
*   **Dependencies:** The project has minimal external dependencies, which are managed using Go modules.
*   **Documentation:** The project has extensive documentation in the `docs/` directory and in the `README.md` file. The `Makefile` also provides a `help` target for discovering common commands.

## Architecture

*   **`cmd/oastools/`**: CLI entry point.
*   **`parser/`**: OpenAPI parsing library.
*   **`validator/`**: OpenAPI validation library.
*   **`fixer/`**: OpenAPI fixer library (auto-fix common errors).
*   **`converter/`**: OpenAPI conversion library.
*   **`joiner/`**: OpenAPI joining library.
*   **`differ/`**: OpenAPI diffing library.
*   **`generator/`**: OpenAPI code generation library.
*   **`builder/`**: OpenAPI builder library.
*   **`internal/`**: Internal shared utilities.
*   **`testdata/`**: Test fixtures and sample specs.

## Boundaries

Do not modify the following files/directories unless explicitly asked:

*   `.github/workflows/`
*   `testdata/` (except for adding new test cases)
*   `vendor/`, `bin/`, `dist/`
*   `go.mod`, `go.sum` (unless adding/removing dependencies)
*   `.goreleaser.yaml`
*   `benchmarks/`

## Acceptance Criteria

A task is complete when:

1. All required functionality is implemented.
2. New/modified exported functions have comprehensive tests.
3. `make build` succeeds without errors.
4. `make test` passes with no failures.
5. Code is formatted and follows existing patterns.
6. Public APIs have godoc comments.
7. No regressions in existing tests.
8. No new security vulnerabilities (`govulncheck`).
