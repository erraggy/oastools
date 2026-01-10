#!/bin/bash
# Block destructive operations on main branch
# Used as PreToolUse hook for Bash commands

BRANCH=$(git branch --show-current 2>/dev/null)

if [[ "$BRANCH" == "main" ]]; then
    echo "❌ BLOCKED: On main branch. Create a feature branch first:"
    echo "   git checkout -b <type>/<description>"
    echo ""
    echo "   Types: feat, fix, refactor, docs, test, chore"
    exit 1
fi
