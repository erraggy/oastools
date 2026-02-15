#!/bin/bash
# Block destructive operations on main branch
# Used as PreToolUse hook for Bash commands
#
# Reads stdin for the tool input JSON to extract the command.
# Only blocks commands that modify code or git state on main.

BRANCH=$(git branch --show-current 2>/dev/null)

# Only care about main branch
[[ "$BRANCH" != "main" ]] && exit 0

# Read the tool input to get the command
# Requires python3 or jq for JSON parsing (fail-closed: empty COMMAND is blocked)
INPUT=$(cat)
COMMAND=$(echo "$INPUT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('command',''))" 2>/dev/null)
if [[ -z "$COMMAND" ]]; then
    COMMAND=$(echo "$INPUT" | jq -r '.command // ""' 2>/dev/null)
fi

# Reject commands with shell metacharacters that could chain commands
# This prevents bypasses like: git status && rm -rf /
# Also rejects newlines to prevent multi-line bypasses
if echo "$COMMAND" | grep -qP '[;|&`$()\\n\\r]|>\s*[^&]'; then
    echo "❌ BLOCKED: Command chaining on main branch."
    echo "   Command: $(echo "$COMMAND" | head -1)"
    echo ""
    echo "   Create a feature branch first:"
    echo "   git checkout -b <type>/<description>"
    exit 1
fi

# Combined allowlist: read-only commands + hook scripts
# Every non-empty line must match (prevents multi-line bypasses)
ALLOW_RE='^(ls|cat|head|tail|wc|tree|which|pwd|date|uname|go (version|doc|list|env)|git (status|branch|log|diff|show|rev-parse|describe|ls-tree|cat-file|symbolic-ref|worktree list)|gh (pr |issue |run |release (list|view))|make (help|check|test|lint|fmt|vet|bench|build)|go test |go build |go vet |go fmt |gofmt|goimports|golangci-lint|(bash )?\.claude/hooks/[a-zA-Z0-9_-]+\.sh$)'
BLOCKED_LINE=$(echo "$COMMAND" | grep -v -E "$ALLOW_RE" | grep -v '^$' | head -1)
if [[ -z "$BLOCKED_LINE" ]]; then
    exit 0
fi

# Block everything else on main
echo "❌ BLOCKED: Destructive command on main branch."
echo "   Command: $COMMAND"
echo ""
echo "   Create a feature branch first:"
echo "   git checkout -b <type>/<description>"
echo ""
echo "   Types: feat, fix, refactor, docs, test, chore"
exit 1