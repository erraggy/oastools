# Build variables
BINARY_NAME=oastools
BUILD_DIR=bin
MAIN_PATH=./cmd/oastools
BENCH_DIR=benchmarks
BENCH_TIME=5s

# Default target
all: build

# =============================================================================
# Build Targets
# =============================================================================

## build: Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

## install: Install the binary
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(MAIN_PATH)

## clean: Clean build artifacts and benchmark outputs
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf dist/
	@rm -f coverage.txt coverage.html
	@rm -f benchmark-*.txt

# =============================================================================
# Test Targets
# =============================================================================

## test: Run tests with coverage (parallel execution for speed)
## Note: Fuzz tests are skipped in regular test runs. Use 'make test-fuzz-parse' to run them separately.
.PHONY: test
test:
	@echo "Running tests..."
ifeq ("$(shell command -v gotestsum)", "")
	go test -coverprofile=coverage.txt -covermode=atomic -timeout=5m -skip='^Fuzz' ./...
else
	gotestsum --format testname -- -coverprofile=coverage.txt -covermode=atomic -timeout=5m -failfast -skip='^Fuzz' ./...
endif

## test-quick: Run tests quickly for rapid iteration (no coverage, short mode)
.PHONY: test-quick
test-quick:
	@echo "Running quick tests..."
	go test -short -skip='^Fuzz' ./...

## test-race: Run tests with race detector (slower, thorough race detection)
.PHONY: test-race
test-race:
	@echo "Running tests with race detector (this may take several minutes)..."
ifeq ("$(shell command -v gotestsum)", "")
	GORACE="halt_on_error=1" GOMAXPROCS=1 go test -v -race -short -timeout=10m -p=1 -parallel=1 -skip='^Fuzz' ./...
else
	GORACE="halt_on_error=1" GOMAXPROCS=1 gotestsum --format testname -- -v -race -short -timeout=10m -failfast -p=1 -parallel=1 -skip='^Fuzz' ./...
endif

## test-full: Run comprehensive tests including race detection and all test modes
.PHONY: test-full
test-full:
	@echo "Running comprehensive tests (this may take several minutes)..."
	@echo ""
	@echo "Phase 1: Unit tests with race detection..."
	GORACE="halt_on_error=1" go test -race -short -timeout=10m -skip='^Fuzz' ./...
	@echo ""
	@echo "Phase 2: Full test suite with coverage..."
	go test -v -coverprofile=coverage.txt -covermode=atomic -timeout=10m -skip='^Fuzz' ./...
	@echo ""
	@echo "Comprehensive tests complete."

## test-coverage: Run tests with coverage report
.PHONY: test-coverage
test-coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated at coverage.html"

## test-fuzz-parse: Run fuzz tests for parser (default: 1m30s, override with FUZZ_TIME, optionally set FUZZ_LOG=1 to save output)
.PHONY: test-fuzz-parse
test-fuzz-parse:
	@echo "Running fuzz tests for ParseBytes..."
	@FUZZ_TIME=$${FUZZ_TIME:-1m30s}; \
	FUZZ_LOG=$${FUZZ_LOG:-0}; \
	echo "Fuzz time: $${FUZZ_TIME}"; \
	if [ "$$FUZZ_LOG" = "1" ]; then \
		TIMESTAMP=$$(date +%Y%m%d-%H%M%S); \
		LOG_FILE="fuzz-parse-$${TIMESTAMP}.log"; \
		echo "Saving output to: $${LOG_FILE}"; \
		go test -v ./parser -run=^$$ -fuzz=FuzzParseBytes -fuzztime=$${FUZZ_TIME} -fuzzminimizetime=30s -parallel=4 2>&1 | tee "$${LOG_FILE}"; \
		echo ""; \
		echo "Fuzz log saved to: $${LOG_FILE}"; \
	else \
		go test -v ./parser -run=^$$ -fuzz=FuzzParseBytes -fuzztime=$${FUZZ_TIME} -fuzzminimizetime=30s -parallel=4; \
	fi
	@echo ""
	@echo "Fuzz corpus stored in: parser/testdata/fuzz/FuzzParseBytes"
	@echo ""
	@echo "To re-run a specific failing input: go test ./parser -run=FuzzParseBytes/<hash>"
	@echo "To save fuzz output to a log file: FUZZ_LOG=1 make test-fuzz-parse"

# =============================================================================
# Code Quality Targets
# =============================================================================

## lint: Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

## fmt: Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

## vet: Run go vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

## check: Run tidy, fmt, lint, test, and git status
.PHONY: check
check: tidy fmt lint test
	@echo "Running git status..."
	@git status

# =============================================================================
# Dependency Targets
# =============================================================================

## deps: Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

## tidy: Tidy go modules
.PHONY: tidy
tidy:
	@echo "Tidying go modules..."
	go mod tidy

# =============================================================================
# Benchmark Targets
# =============================================================================

## bench: Run all benchmarks
.PHONY: bench
bench:
	@echo "Running all benchmarks ($(BENCH_TIME) per benchmark)..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -timeout=15m ./parser ./validator ./fixer ./converter ./joiner ./differ ./generator ./builder

## bench-parser: Run parser benchmarks only
.PHONY: bench-parser
bench-parser:
	@echo "Running parser benchmarks..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -timeout=15m ./parser

## bench-validator: Run validator benchmarks only
.PHONY: bench-validator
bench-validator:
	@echo "Running validator benchmarks..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -timeout=15m ./validator

## bench-fixer: Run fixer benchmarks only
.PHONY: bench-fixer
bench-fixer:
	@echo "Running fixer benchmarks..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -timeout=15m ./fixer

## bench-converter: Run converter benchmarks only
.PHONY: bench-converter
bench-converter:
	@echo "Running converter benchmarks..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -timeout=15m ./converter

## bench-joiner: Run joiner benchmarks only
.PHONY: bench-joiner
bench-joiner:
	@echo "Running joiner benchmarks..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -timeout=15m ./joiner

## bench-differ: Run differ benchmarks only
.PHONY: bench-differ
bench-differ:
	@echo "Running differ benchmarks..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -timeout=15m ./differ

## bench-generator: Run generator benchmarks only
.PHONY: bench-generator
bench-generator:
	@echo "Running generator benchmarks..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -timeout=15m ./generator

## bench-builder: Run builder benchmarks only
.PHONY: bench-builder
bench-builder:
	@echo "Running builder benchmarks..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -timeout=15m ./builder

## bench-overlay: Run overlay and jsonpath benchmarks only
.PHONY: bench-overlay
bench-overlay:
	@echo "Running overlay benchmarks..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -timeout=15m ./overlay ./internal/jsonpath

## bench-save: Run all benchmarks and save to timestamped file
.PHONY: bench-save
bench-save:
	@echo "Running benchmarks and saving results..."
	@TIMESTAMP=$$(date +%Y%m%d-%H%M%S); \
	OUTPUT_FILE="benchmark-$${TIMESTAMP}.txt"; \
	go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -timeout=15m ./parser ./validator ./fixer ./converter ./joiner ./differ ./generator ./builder ./overlay ./internal/jsonpath 2>&1 | tee "$${OUTPUT_FILE}"; \
	echo ""; \
	echo "Benchmark results saved to: $${OUTPUT_FILE}"

## bench-baseline: Run benchmarks and update baseline file
.PHONY: bench-baseline
bench-baseline:
	@echo "Running benchmarks and updating baseline..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -timeout=15m ./parser ./validator ./fixer ./converter ./joiner ./differ ./generator ./builder ./overlay ./internal/jsonpath 2>&1 | tee benchmark-baseline.txt
	@echo ""
	@echo "Baseline updated: benchmark-baseline.txt"

## bench-release: Run benchmarks for upcoming release (usage: make bench-release VERSION=v1.19.1)
## Note: Corpus benchmarks require -tags=corpus and are excluded by default.
## Use 'make bench-corpus' to run corpus benchmarks separately.
.PHONY: bench-release
bench-release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION required"; \
		echo "Usage: make bench-release VERSION=v1.19.1"; \
		exit 1; \
	fi
	@echo "Running benchmarks for $(VERSION)..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -timeout=15m ./parser ./validator ./fixer ./converter ./joiner ./differ ./generator ./builder 2>&1 | tee "$(BENCH_DIR)/benchmark-$(VERSION).txt"
	@echo ""
	@echo "Benchmark saved to: $(BENCH_DIR)/benchmark-$(VERSION).txt"
	@echo ""
	@./scripts/compare-with-previous.sh "$(VERSION)" || true

## bench-compare: Compare two benchmark files (usage: make bench-compare OLD=file1.txt NEW=file2.txt)
.PHONY: bench-compare
bench-compare:
	@if [ -z "$(OLD)" ] || [ -z "$(NEW)" ]; then \
		echo "Error: Please specify OLD and NEW benchmark files"; \
		echo "Usage: make bench-compare OLD=benchmark-baseline.txt NEW=benchmark-20251117.txt"; \
		exit 1; \
	fi
	@if command -v benchstat >/dev/null 2>&1; then \
		echo "Comparing $(OLD) vs $(NEW)..."; \
		benchstat "$(OLD)" "$(NEW)"; \
	else \
		echo "benchstat not installed. Install it with:"; \
		echo "  go install golang.org/x/perf/cmd/benchstat@latest"; \
		echo ""; \
		echo "Showing simple diff instead:"; \
		echo ""; \
		diff -u "$(OLD)" "$(NEW)" || true; \
	fi

## bench-clean: Remove timestamped benchmark and fuzz output files (preserves baseline and corpus)
.PHONY: bench-clean
bench-clean:
	@echo "Cleaning benchmark and fuzz outputs..."
	@rm -f benchmark-[0-9]*.txt
	@rm -f cpu-profile-*.prof
	@rm -f mem-profile-*.prof
	@rm -f fuzz-parse-*.log
	@echo "Benchmark and fuzz outputs cleaned (baseline and corpus preserved)"

# =============================================================================
# Profiling Targets
# =============================================================================

## bench-cpu: Run benchmarks with CPU profiling
.PHONY: bench-cpu
bench-cpu:
	@echo "Running benchmarks with CPU profiling..."
	@TIMESTAMP=$$(date +%Y%m%d-%H%M%S); \
	PROFILE_FILE="cpu-profile-$${TIMESTAMP}.prof"; \
	go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -cpuprofile="$${PROFILE_FILE}" ./parser ./validator ./converter ./joiner ./differ ./builder; \
	echo ""; \
	echo "CPU profile saved to: $${PROFILE_FILE}"; \
	echo "Analyze with: go tool pprof $${PROFILE_FILE}"

## bench-mem: Run benchmarks with memory profiling
.PHONY: bench-mem
bench-mem:
	@echo "Running benchmarks with memory profiling..."
	@TIMESTAMP=$$(date +%Y%m%d-%H%M%S); \
	PROFILE_FILE="mem-profile-$${TIMESTAMP}.prof"; \
	go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -memprofile="$${PROFILE_FILE}" ./parser ./validator ./converter ./joiner ./differ ./builder; \
	echo ""; \
	echo "Memory profile saved to: $${PROFILE_FILE}"; \
	echo "Analyze with: go tool pprof $${PROFILE_FILE}"

## bench-profile: Run benchmarks with both CPU and memory profiling
.PHONY: bench-profile
bench-profile:
	@echo "Running benchmarks with CPU and memory profiling..."
	@TIMESTAMP=$$(date +%Y%m%d-%H%M%S); \
	CPU_PROFILE="cpu-profile-$${TIMESTAMP}.prof"; \
	MEM_PROFILE="mem-profile-$${TIMESTAMP}.prof"; \
	go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -cpuprofile="$${CPU_PROFILE}" -memprofile="$${MEM_PROFILE}" ./parser ./validator ./converter ./joiner ./differ ./builder; \
	echo ""; \
	echo "CPU profile saved to: $${CPU_PROFILE}"; \
	echo "Memory profile saved to: $${MEM_PROFILE}"; \
	echo "Analyze with: go tool pprof <profile-file>"

# =============================================================================
# Release Targets
# =============================================================================

## release-test: Test GoReleaser configuration locally (creates dist/ without publishing)
.PHONY: release-test
release-test:
	@echo "Testing GoReleaser configuration (snapshot mode)..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "Error: goreleaser not installed. Install it with:"; \
		echo "  brew install goreleaser"; \
		exit 1; \
	fi
	@goreleaser release --snapshot --clean
	@echo ""
	@echo "Test successful! Check dist/ directory for generated artifacts."
	@echo "To clean up: make release-clean"
	@echo ""
	@echo "To create a real release, use:"
	@echo "  gh release create vX.Y.Z --title \"vX.Y.Z - Description\" --notes \"...\""

## release-clean: Clean GoReleaser artifacts from local testing
.PHONY: release-clean
release-clean:
	@echo "Cleaning release artifacts..."
	@rm -rf dist/
	@echo "Release artifacts cleaned"

# =============================================================================
# Corpus Testing Targets
# =============================================================================

## corpus-download: Download public OpenAPI specifications for integration testing
.PHONY: corpus-download
corpus-download:
	@echo "Downloading corpus specifications..."
	@mkdir -p testdata/corpus
	@echo "  Downloading Petstore (OAS 2.0)..."
	@curl -sL -o testdata/corpus/petstore-swagger.json "https://petstore.swagger.io/v2/swagger.json"
	@echo "  Downloading DigitalOcean (OAS 3.0.0, bundled)..."
	@curl -sL -o testdata/corpus/digitalocean-public.v2.yaml "https://api-engineering.nyc3.digitaloceanspaces.com/spec-ci/DigitalOcean-public.v2.yaml"
	@echo "  Downloading Asana (OAS 3.0.0)..."
	@curl -sL -o testdata/corpus/asana-oas.yaml "https://raw.githubusercontent.com/Asana/openapi/master/defs/asana_oas.yaml"
	@echo "  Downloading Google Maps (OAS 3.0.3)..."
	@curl -sL -o testdata/corpus/google-maps-platform.json "https://raw.githubusercontent.com/googlemaps/openapi-specification/main/dist/google-maps-platform-openapi3.json"
	@echo "  Downloading US NWS (OAS 3.0.3)..."
	@curl -sL -o testdata/corpus/nws-openapi.json "https://api.weather.gov/openapi.json"
	@echo "  Downloading Plaid (OAS 3.0.0)..."
	@curl -sL -o testdata/corpus/plaid-2020-09-14.yml "https://raw.githubusercontent.com/plaid/plaid-openapi/master/2020-09-14.yml"
	@echo "  Downloading Discord (OAS 3.1.0)..."
	@curl -sL -o testdata/corpus/discord-openapi.json "https://raw.githubusercontent.com/discord/discord-api-spec/main/specs/openapi.json"
	@echo "  Downloading GitHub (OAS 3.0.3)..."
	@curl -sL -o testdata/corpus/github-api.json "https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json"
	@echo "  Downloading Stripe (OAS 3.0.0, large)..."
	@curl -sL -o testdata/corpus/stripe-spec3.json "https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.json"
	@echo "  Downloading Microsoft Graph (OAS 3.0.4, large)..."
	@curl -sL -o testdata/corpus/msgraph-openapi.yaml "https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml"
	@echo "Corpus download complete!"
	@echo ""
	@ls -lh testdata/corpus/

## corpus-clean: Remove downloaded corpus files
.PHONY: corpus-clean
corpus-clean:
	@echo "Cleaning corpus files..."
	@rm -f testdata/corpus/*.json testdata/corpus/*.yaml testdata/corpus/*.yml
	@echo "Corpus files removed (README.md preserved)"

## test-corpus: Run corpus integration tests (requires corpus-download)
.PHONY: test-corpus
test-corpus:
	@echo "Running corpus integration tests..."
	@go test -v -count=1 ./... -run 'TestCorpus_'

## test-corpus-short: Run corpus integration tests excluding large specs
.PHONY: test-corpus-short
test-corpus-short:
	@echo "Running corpus integration tests (short mode, excludes large specs)..."
	@go test -v -short -count=1 ./... -run 'TestCorpus_'

## bench-corpus: Run corpus benchmarks (requires -tags=corpus)
## Note: Corpus benchmarks require large specs and more memory.
## These are excluded from regular benchmark runs to prevent memory exhaustion.
.PHONY: bench-corpus
bench-corpus:
	@echo "Running corpus benchmarks (requires -tags=corpus)..."
	@go test -tags=corpus -bench='BenchmarkCorpus' -benchmem -benchtime=$(BENCH_TIME) ./parser ./validator ./fixer ./differ

# =============================================================================
# Help Target
# =============================================================================

## help: Show this help message
.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
	@echo ""
	@echo "Benchmark Configuration:"
	@echo "  BENCH_TIME=<duration>  Benchmark run time per test (default: 5s)"
	@echo "                         Example: make bench BENCH_TIME=10s"
	@echo ""
	@echo "Corpus Testing:"
	@echo "  1. make corpus-download    # Download all specs (one-time)"
	@echo "  2. make test-corpus-short  # Run tests (excludes large specs)"
	@echo "  3. make test-corpus        # Run all corpus tests"
