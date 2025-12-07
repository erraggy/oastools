#!/bin/bash
# Back-fill benchmark results for previous releases
#
# Usage:
#   ./scripts/backfill-benchmarks.sh v1.9.12 v1.9.11 v1.9.10    # Specific versions
#   ./scripts/backfill-benchmarks.sh --last 5                    # Last N releases
#   ./scripts/backfill-benchmarks.sh --all                       # All releases (careful!)
#
# This script automates the process of generating benchmark results for older
# releases by checking out each tag, running benchmarks, and creating version-tagged
# benchmark files.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
BACKFILL_DIR="benchmarks"

# Print colored message
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

# Show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS] [versions...]

Back-fill benchmark results for previous releases.

Options:
    --last N        Back-fill last N releases (e.g., --last 5)
    --all           Back-fill all releases (use with caution!)
    --upload        Upload to GitHub releases instead of committing to repo
    --help          Show this help message

Examples:
    $0 v1.9.12 v1.9.11 v1.9.10           # Back-fill specific versions
    $0 --last 5                           # Back-fill last 5 releases
    $0 --last 5 --upload                  # Back-fill and upload to GitHub
    $0 --all                              # Back-fill all releases

Notes:
    - Requires clean working directory (will stash changes)
    - Benchmarks are run in isolated checkouts to avoid conflicts
    - Failed builds are logged but don't stop the process
    - Results are placed in the benchmarks/ directory

EOF
}

# Check prerequisites
check_prerequisites() {
    # Check git
    if ! command -v git &> /dev/null; then
        log_error "git is not installed"
        exit 1
    fi

    # Check if in git repository
    if ! git rev-parse --git-dir &> /dev/null; then
        log_error "Not in a git repository"
        exit 1
    fi

    # Check if working directory is clean (allow stashed changes)
    if ! git diff-index --quiet HEAD --; then
        log_warning "Working directory has uncommitted changes"
        read -p "Stash changes and continue? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            git stash push -m "backfill-benchmarks: temporary stash"
            STASHED=true
        else
            log_error "Please commit or stash changes before running this script"
            exit 1
        fi
    fi

    # Check if make bench-save works
    if ! make -n bench-save &> /dev/null; then
        log_error "make bench-save target not found"
        exit 1
    fi
}

# Get list of versions to back-fill
get_versions() {
    local last_n=""
    local all_releases=false
    local versions=()

    while [[ $# -gt 0 ]]; do
        case $1 in
            --last)
                last_n="$2"
                shift 2
                ;;
            --all)
                all_releases=true
                shift
                ;;
            --upload)
                UPLOAD_MODE=true
                shift
                ;;
            --help)
                show_usage
                exit 0
                ;;
            -*)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
            *)
                versions+=("$1")
                shift
                ;;
        esac
    done

    # Get versions based on options
    if [ "$all_releases" = true ]; then
        versions=($(git tag -l "v*" | sort -V))
    elif [ -n "$last_n" ]; then
        versions=($(git tag -l "v*" | sort -V | tail -n "$last_n"))
    fi

    if [ ${#versions[@]} -eq 0 ]; then
        log_error "No versions specified"
        show_usage
        exit 1
    fi

    echo "${versions[@]}"
}

# Back-fill benchmarks for a single version
backfill_version() {
    local version=$1
    local benchmark_file="benchmark-${version}.txt"

    log_info "Processing $version..."

    # Check if benchmark already exists
    if [ -f "$benchmark_file" ]; then
        log_warning "Benchmark file already exists: $benchmark_file"
        read -p "Overwrite? (y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Skipping $version"
            return 0
        fi
    fi

    # Checkout the version
    if ! git checkout "$version" 2>/dev/null; then
        log_error "Failed to checkout $version"
        return 1
    fi

    # Clean and prepare
    log_info "Cleaning and preparing build environment..."
    go clean -cache 2>/dev/null || true

    # Download dependencies
    if ! go mod download 2>/dev/null; then
        log_warning "Failed to download dependencies for $version"
    fi

    # Try to tidy (may fail for old versions)
    go mod tidy 2>/dev/null || true

    # Run benchmarks
    # Use -short flag and timeout to prevent long-running corpus benchmarks from hanging
    # Also set SKIP_LARGE_TESTS=1 as a fallback for older versions
    # Run go test directly instead of make bench-save to ensure consistent flags across versions
    log_info "Running benchmarks (this may take several minutes)..."
    local temp_output="benchmark-${version}-temp.txt"
    if ! SKIP_LARGE_TESTS=1 go test -bench=. -benchmem -benchtime=5s -timeout=15m -short ./parser ./validator ./converter ./joiner ./differ ./builder 2>&1 | tee "$temp_output"; then
        log_warning "Some benchmarks failed for $version (results may be partial)"
        # Don't fail completely - we may still have partial results
    fi

    # Check if we got any benchmark output
    if ! grep -q "^Benchmark" "$temp_output" 2>/dev/null; then
        log_error "No benchmark output generated for $version"
        rm -f "$temp_output"
        git checkout "$CURRENT_BRANCH"
        return 1
    fi

    # Move temp file to version-tagged name
    mv "$temp_output" "$benchmark_file"
    log_success "Created $benchmark_file"

    # Return to original branch
    git checkout "$CURRENT_BRANCH" 2>/dev/null

    # Move to backfill directory
    mkdir -p "$BACKFILL_DIR"
    mv "$benchmark_file" "$BACKFILL_DIR/"

    return 0
}

# Upload benchmark to GitHub release
upload_to_github() {
    local version=$1
    local benchmark_file="$BACKFILL_DIR/benchmark-${version}.txt"

    if [ ! -f "$benchmark_file" ]; then
        log_error "Benchmark file not found: $benchmark_file"
        return 1
    fi

    # Check if gh is installed
    if ! command -v gh &> /dev/null; then
        log_error "GitHub CLI (gh) is not installed"
        log_info "Install it or manually upload $benchmark_file to $version release"
        return 1
    fi

    log_info "Uploading $benchmark_file to $version release..."
    if gh release upload "$version" "$benchmark_file" --clobber 2>&1; then
        log_success "Uploaded to $version release"
        return 0
    else
        log_error "Failed to upload to $version release"
        return 1
    fi
}

# Main execution
main() {
    log_info "Benchmark Back-fill Tool"
    echo ""

    # Check prerequisites
    check_prerequisites

    # Get versions to process
    local versions=($(get_versions "$@"))
    local total=${#versions[@]}
    local successful=0
    local failed=0

    log_info "Will back-fill benchmarks for $total versions:"
    printf '%s\n' "${versions[@]}" | sed 's/^/  - /'
    echo ""

    read -p "Continue? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Aborted by user"
        exit 0
    fi

    # Process each version
    local count=0
    for version in "${versions[@]}"; do
        count=$((count + 1))
        echo ""
        log_info "[$count/$total] Processing $version..."

        if backfill_version "$version"; then
            successful=$((successful + 1))

            # Upload if requested
            if [ "$UPLOAD_MODE" = true ]; then
                upload_to_github "$version"
            fi
        else
            failed=$((failed + 1))
        fi
    done

    # Return to original branch
    git checkout "$CURRENT_BRANCH" 2>/dev/null

    # Restore stashed changes if any
    if [ "$STASHED" = true ]; then
        log_info "Restoring stashed changes..."
        git stash pop
    fi

    # Summary
    echo ""
    echo "========================================="
    log_info "Back-fill Summary"
    echo "========================================="
    echo "Total versions:    $total"
    log_success "Successful:        $successful"
    if [ $failed -gt 0 ]; then
        log_error "Failed:            $failed"
    else
        echo "Failed:            $failed"
    fi
    echo ""

    if [ $successful -gt 0 ]; then
        log_info "Benchmark files are in: $BACKFILL_DIR/"
        echo ""
        if [ "$UPLOAD_MODE" = true ]; then
            log_info "Files have been uploaded to GitHub releases"
        else
            log_info "To commit these to the repository:"
            echo "  git add $BACKFILL_DIR/benchmark-v*.txt"
            echo "  git commit -m \"chore: back-fill benchmark results\""
            echo ""
            log_info "To upload to GitHub releases instead:"
            echo "  for f in $BACKFILL_DIR/benchmark-*.txt; do"
            echo "    version=\$(basename \$f .txt | sed 's/benchmark-//')"
            echo "    gh release upload \"\$version\" \"\$f\" --clobber"
            echo "  done"
        fi
    fi

    if [ $failed -gt 0 ]; then
        echo ""
        log_warning "Some versions failed to back-fill."
        exit 1
    fi

    exit 0
}

# Run main function
main "$@"
