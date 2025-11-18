.PHONY: build test lint clean install tidy check help bench bench-parser bench-validator bench-converter bench-joiner bench-save bench-compare bench-baseline bench-clean

# Build variables
BINARY_NAME=oastools
BUILD_DIR=bin
MAIN_PATH=./cmd/oastools
BENCH_DIR=benchmarks
BENCH_TIME=5s

# Default target
all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

## test: Run tests
test:
	@echo "Running tests..."
ifeq ("$(shell command -v gotestsum)", "")
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
else
	gotestsum --format testname -- -v -coverprofile=coverage.txt -covermode=atomic -timeout=60m -race -failfast ./...
endif

## test-coverage: Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated at coverage.html"

## lint: Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## clean: Clean build artifacts and benchmark outputs
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.txt coverage.html
	@rm -f benchmark-*.txt

## install: Install the binary
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(MAIN_PATH)

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

## tidy: Tidy go modules
tidy:
	@echo "Tidying go modules..."
	go mod tidy

## check: Run tidy, fmt, lint, test, and git status
check: tidy fmt lint test
	@echo "Running git status..."
	@git status

## bench: Run all benchmarks
bench:
	@echo "Running all benchmarks ($(BENCH_TIME) per benchmark)..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) ./parser ./validator ./converter ./joiner

## bench-parser: Run parser benchmarks only
bench-parser:
	@echo "Running parser benchmarks..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) ./parser

## bench-validator: Run validator benchmarks only
bench-validator:
	@echo "Running validator benchmarks..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) ./validator

## bench-converter: Run converter benchmarks only
bench-converter:
	@echo "Running converter benchmarks..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) ./converter

## bench-joiner: Run joiner benchmarks only
bench-joiner:
	@echo "Running joiner benchmarks..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) ./joiner

## bench-save: Run all benchmarks and save to timestamped file
bench-save:
	@echo "Running benchmarks and saving results..."
	@TIMESTAMP=$$(date +%Y%m%d-%H%M%S); \
	OUTPUT_FILE="benchmark-$${TIMESTAMP}.txt"; \
	go test -bench=. -benchmem -benchtime=$(BENCH_TIME) ./parser ./validator ./converter ./joiner 2>&1 | tee "$${OUTPUT_FILE}"; \
	echo ""; \
	echo "Benchmark results saved to: $${OUTPUT_FILE}"

## bench-baseline: Run benchmarks and update baseline file
bench-baseline:
	@echo "Running benchmarks and updating baseline..."
	@go test -bench=. -benchmem -benchtime=$(BENCH_TIME) ./parser ./validator ./converter ./joiner 2>&1 | tee benchmark-baseline.txt
	@echo ""
	@echo "Baseline updated: benchmark-baseline.txt"

## bench-compare: Compare two benchmark files (usage: make bench-compare OLD=file1.txt NEW=file2.txt)
bench-compare:
	@if [ -z "$(OLD)" ] || [ -z "$(NEW)" ]; then \
		echo "Error: Please specify OLD and NEW benchmark files"; \
		echo "Usage: make bench-compare OLD=benchmark-baseline.txt NEW=benchmark-20251117.txt"; \
		exit 1; \
	fi
	@if command -v benchstat >/dev/null 2>&1; then \
		echo "Comparing $(OLD) vs $(NEW)..."; \
		benchstat $(OLD) $(NEW); \
	else \
		echo "benchstat not installed. Install it with:"; \
		echo "  go install golang.org/x/perf/cmd/benchstat@latest"; \
		echo ""; \
		echo "Showing simple diff instead:"; \
		echo ""; \
		diff -u $(OLD) $(NEW) || true; \
	fi

## bench-clean: Remove timestamped benchmark output files (preserves baseline)
bench-clean:
	@echo "Cleaning benchmark outputs..."
	@rm -f benchmark-[0-9]*.txt
	@echo "Benchmark outputs cleaned (baseline preserved)"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
