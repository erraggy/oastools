# Make Commands

## Recommended Workflow

After making changes to Go source files:

```bash
make check  # Runs all quality checks (tidy, fmt, lint, lint-md, test) and shows git status
```

## Building

```bash
make build    # Build binary to bin/oastools
make install  # Install to $GOPATH/bin
```

## Testing

```bash
make test          # Run tests with coverage (parallel, fast)
make test-quick    # Run tests quickly (no coverage, for rapid iteration)
make test-full     # Run comprehensive tests with race detection
make test-coverage # Generate and view HTML coverage report
```

## Code Quality

```bash
make fmt      # Format all Go code
make vet      # Run go vet
make lint     # Run golangci-lint
make lint-md  # Lint markdown files (markdownlint-cli2)
```

## Maintenance

```bash
make deps  # Download and tidy dependencies
make clean # Remove build artifacts
```

## Documentation

```bash
make docs-prepare  # Run prepare-docs.sh (copies generated files into docs/)
make docs-serve    # Preview locally at http://127.0.0.1:8000 (blocking)
make docs-start    # Start docs server in background
make docs-stop     # Stop background docs server
make docs-build    # Build static site to site/
make docs-clean    # Remove generated docs artifacts
```

For details on source vs generated files, see [docs-website.md](docs-website.md).

## Security

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...  # Check for vulnerabilities
```

For security fix workflows and PR check commands, see [WORKFLOW.md](../../WORKFLOW.md).
