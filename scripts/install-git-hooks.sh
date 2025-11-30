#!/usr/bin/env bash
#
# install-git-hooks.sh - Install Git hooks for local code review
#
# This script installs the pre-push hook that runs Claude Code review
# before pushing changes.

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GIT_HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Installing Git hooks for local code review...${NC}"
echo

# Check if .git directory exists
if [ ! -d "$PROJECT_ROOT/.git" ]; then
    echo -e "${YELLOW}Warning: .git directory not found. Are you in a Git repository?${NC}"
    exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p "$GIT_HOOKS_DIR"

# Install pre-push hook
HOOK_SOURCE="$SCRIPT_DIR/pre-push-hook"
HOOK_TARGET="$GIT_HOOKS_DIR/pre-push"

if [ -f "$HOOK_TARGET" ] && [ ! -L "$HOOK_TARGET" ]; then
    echo -e "${YELLOW}Warning: Existing pre-push hook found (not a symlink)${NC}"
    echo "Backing up to: $HOOK_TARGET.backup"
    mv "$HOOK_TARGET" "$HOOK_TARGET.backup"
fi

# Create symlink
ln -sf "../../scripts/pre-push-hook" "$HOOK_TARGET"

echo -e "${GREEN}✓ Pre-push hook installed${NC}"
echo
echo "The hook will run before each 'git push' and perform a code review."
echo
echo "To skip the review for a specific push:"
echo "  SKIP_REVIEW=1 git push"
echo
echo "To bypass the hook entirely (not recommended):"
echo "  git push --no-verify"
echo
echo "To manually run code review:"
echo "  ./scripts/local-code-review.sh          # Review uncommitted changes"
echo "  ./scripts/local-code-review.sh staged   # Review staged changes"
echo "  ./scripts/local-code-review.sh branch   # Review branch changes"
