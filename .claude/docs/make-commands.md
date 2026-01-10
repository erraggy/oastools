# Make Commands

## Recommended Workflow

After making changes to Go source files:
```bash
make check  # Runs all quality checks (tidy, fmt, lint, test) and shows git status
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
make fmt   # Format all Go code
make vet   # Run go vet
make lint  # Run golangci-lint
```

## Maintenance

```bash
make deps  # Download and tidy dependencies
make clean # Remove build artifacts
```

## Security

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...  # Check for vulnerabilities
```

For security fix workflows and PR check commands, see [WORKFLOW.md](../../WORKFLOW.md).
