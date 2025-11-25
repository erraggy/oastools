#!/bin/bash
# Generate benchmark comparison for releases
#
# Usage: ./scripts/generate-benchmark-comparison.sh <prev_version> <curr_version>
# Example: ./scripts/generate-benchmark-comparison.sh v1.9.12 v1.10.0
#
# This script uses benchstat to compare benchmark results between two versions.
# Benchmark files must exist as benchmark-<version>.txt in the repository root.

set -e

PREV_VERSION=$1
CURR_VERSION=$2

# Validate arguments
if [ -z "$PREV_VERSION" ] || [ -z "$CURR_VERSION" ]; then
    echo "Usage: $0 <prev_version> <curr_version>"
    echo "Example: $0 v1.9.12 v1.10.0"
    exit 1
fi

PREV_FILE="benchmark-${PREV_VERSION}.txt"
CURR_FILE="benchmark-${CURR_VERSION}.txt"
OUTPUT_FILE="benchmark-comparison-${CURR_VERSION}.txt"

# Check if previous benchmark file exists
if [ ! -f "$PREV_FILE" ]; then
    echo "Error: Previous benchmark file not found: $PREV_FILE"
    echo ""
    echo "Available benchmark files:"
    ls -1 benchmark-v*.txt 2>/dev/null || echo "  (none found)"
    exit 1
fi

# Check if current benchmark file exists
if [ ! -f "$CURR_FILE" ]; then
    echo "Error: Current benchmark file not found: $CURR_FILE"
    echo ""
    echo "You may need to run:"
    echo "  make bench-save"
    echo "  cp benchmark-YYYYMMDD-HHMMSS.txt $CURR_FILE"
    exit 1
fi

# Check if benchstat is installed
if ! command -v benchstat &> /dev/null; then
    echo "Error: benchstat is not installed"
    echo ""
    echo "Install it with:"
    echo "  go install golang.org/x/perf/cmd/benchstat@latest"
    exit 1
fi

# Generate comparison
echo "Comparing benchmarks:"
echo "  Previous: $PREV_FILE"
echo "  Current:  $CURR_FILE"
echo ""

benchstat "$PREV_FILE" "$CURR_FILE" > "$OUTPUT_FILE"

echo "Comparison saved to: $OUTPUT_FILE"
echo ""
echo "Summary of significant changes:"
benchstat "$PREV_FILE" "$CURR_FILE" | grep -E "~|±" | head -20
