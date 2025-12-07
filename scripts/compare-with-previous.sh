#!/bin/bash
# Compare benchmark results with the previous version
#
# Usage:
#   ./scripts/compare-with-previous.sh v1.19.1
#
# This script finds the benchmark file for the previous version and runs
# benchstat to compare performance. If benchstat is not installed, it
# provides installation instructions.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check arguments
if [ -z "$1" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v1.19.1"
    exit 1
fi

VERSION="$1"
BENCHMARK_DIR="benchmarks"
CURRENT_FILE="$BENCHMARK_DIR/benchmark-${VERSION}.txt"

# Verify current benchmark file exists
if [ ! -f "$CURRENT_FILE" ]; then
    log_error "Benchmark file not found: $CURRENT_FILE"
    exit 1
fi

# Find previous version's benchmark file
# List all benchmark files, sort by version, find the one before current
find_previous_version() {
    local current_version="$1"
    local prev_file=""

    # Get all version-tagged benchmark files, sorted by version
    for file in $(ls -1 "$BENCHMARK_DIR"/benchmark-v*.txt 2>/dev/null | sort -V); do
        local file_version=$(basename "$file" .txt | sed 's/benchmark-//')
        if [ "$file_version" = "$current_version" ]; then
            echo "$prev_file"
            return
        fi
        prev_file="$file"
    done

    # If we get here, current version wasn't found in the list (it's new)
    # Return the last file as the previous version
    echo "$prev_file"
}

PREV_FILE=$(find_previous_version "$VERSION")

if [ -z "$PREV_FILE" ] || [ ! -f "$PREV_FILE" ]; then
    log_warning "No previous version benchmark found for comparison"
    log_info "This appears to be the first benchmark or previous version is missing"
    exit 0
fi

PREV_VERSION=$(basename "$PREV_FILE" .txt | sed 's/benchmark-//')
log_info "Comparing $VERSION with $PREV_VERSION"

# Check if benchstat is installed
if ! command -v benchstat &> /dev/null; then
    log_warning "benchstat not installed - skipping comparison"
    echo ""
    echo "To enable automatic comparisons, install benchstat:"
    echo "  go install golang.org/x/perf/cmd/benchstat@latest"
    echo ""
    echo "Manual comparison:"
    echo "  benchstat $PREV_FILE $CURRENT_FILE"
    exit 0
fi

# Run benchstat comparison
echo ""
echo "========================================="
echo "Benchmark Comparison: $PREV_VERSION → $VERSION"
echo "========================================="
echo ""

benchstat "$PREV_FILE" "$CURRENT_FILE"

echo ""
log_success "Comparison complete"
echo ""
echo "Files compared:"
echo "  Old: $PREV_FILE"
echo "  New: $CURRENT_FILE"
