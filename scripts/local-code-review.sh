#!/usr/bin/env bash
#
# local-code-review.sh - Run Claude Code review locally before pushing
#
# This script performs a code review using Claude Code CLI on uncommitted
# or unpushed changes, providing feedback before creating a PR.
#
# Usage:
#   ./scripts/local-code-review.sh          # Review all uncommitted changes
#   ./scripts/local-code-review.sh staged   # Review only staged changes
#   ./scripts/local-code-review.sh branch   # Review changes in current branch vs main
#   ./scripts/local-code-review.sh --help   # Show this help

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

usage() {
    cat <<EOF
Usage: $(basename "$0") [MODE]

Run Claude Code review locally before pushing changes.

MODES:
    (none)      Review all uncommitted changes (default)
    staged      Review only staged changes
    branch      Review all changes in current branch vs main
    --help      Show this help message

EXAMPLES:
    # Review all uncommitted changes
    ./scripts/local-code-review.sh

    # Review only staged changes (useful before commit)
    ./scripts/local-code-review.sh staged

    # Review all changes in current branch (useful before PR)
    ./scripts/local-code-review.sh branch

REQUIREMENTS:
    - Claude Code CLI must be installed and authenticated
    - Run 'claude --version' to verify installation

ENVIRONMENT:
    SKIP_REVIEW=1       Skip the code review (useful for emergencies)
    CLAUDE_MODEL=<id>   Use specific Claude model (default: claude-sonnet-4-5-20250929)
EOF
}

check_requirements() {
    if ! command -v claude &> /dev/null; then
        echo -e "${RED}Error: claude CLI not found${NC}"
        echo "Install from: https://code.claude.com/docs/en/quickstart#step-1:-install-claude-code"
        exit 1
    fi

    # Check authentication
    if ! claude --version &> /dev/null; then
        echo -e "${RED}Error: Claude Code not properly configured${NC}"
        echo "Run: claude auth"
        exit 1
    fi
}

get_diff() {
    local mode="$1"
    case "$mode" in
        staged)
            git diff --cached
            ;;
        branch)
            # Get the default branch (main or master)
            local default_branch
            default_branch=$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@' || echo "main")

            # Get current branch
            local current_branch
            current_branch=$(git branch --show-current)

            if [ "$current_branch" = "$default_branch" ]; then
                echo -e "${YELLOW}Warning: You're on the $default_branch branch. Showing uncommitted changes instead.${NC}" >&2
                git diff
            else
                git diff "$default_branch"..."$current_branch"
            fi
            ;;
        *)
            git diff
            ;;
    esac
}

main() {
    cd "$PROJECT_ROOT"

    # Parse arguments
    local mode=""
    if [ $# -gt 0 ]; then
        case "$1" in
            --help|-h)
                usage
                exit 0
                ;;
            staged|branch)
                mode="$1"
                ;;
            *)
                echo -e "${RED}Error: Unknown mode '$1'${NC}"
                usage
                exit 1
                ;;
        esac
    fi

    # Check for skip flag
    if [ "${SKIP_REVIEW:-0}" = "1" ]; then
        echo -e "${YELLOW}Skipping code review (SKIP_REVIEW=1)${NC}"
        exit 0
    fi

    echo -e "${BLUE}=== Local Claude Code Review ===${NC}"
    echo

    # Check requirements
    check_requirements

    # Get diff
    local diff_output
    diff_output=$(get_diff "$mode")

    if [ -z "$diff_output" ]; then
        echo -e "${YELLOW}No changes to review${NC}"
        exit 0
    fi

    # Show what we're reviewing
    case "$mode" in
        staged)
            echo -e "${BLUE}Reviewing staged changes...${NC}"
            ;;
        branch)
            echo -e "${BLUE}Reviewing branch changes...${NC}"
            ;;
        *)
            echo -e "${BLUE}Reviewing uncommitted changes...${NC}"
            ;;
    esac
    echo

    # Create temporary file with diff
    local temp_diff
    temp_diff=$(mktemp)
    trap "rm -f '$temp_diff'" EXIT

    echo "$diff_output" > "$temp_diff"

    # Run Claude Code review
    echo -e "${GREEN}Running Claude Code review...${NC}"
    echo

    local review_prompt
    review_prompt=$(cat <<'PROMPT'
Please review the following code changes and provide feedback on:

1. **Code Quality**: Are there any bugs, logic errors, or edge cases not handled?
2. **Best Practices**: Does the code follow Go best practices and project conventions?
3. **Security**: Are there any security vulnerabilities or concerns?
4. **Performance**: Are there any obvious performance issues?
5. **Testing**: Are the changes adequately tested? Are there missing test cases?
6. **Documentation**: Is the code well-documented? Are there missing comments for complex logic?

Please be concise and focus on actionable feedback. If the changes look good, say so briefly.

Changes to review:
PROMPT
)

    # Use claude CLI to review
    local model="${CLAUDE_MODEL:-claude-sonnet-4-5-20250929}"

    {
        echo "$review_prompt"
        echo '```diff'
        cat "$temp_diff"
        echo '```'
    } | claude --print --model "$model"

    echo
    echo -e "${GREEN}=== Review Complete ===${NC}"
}

main "$@"
