#!/bin/bash
# Block direct use of 'gh release create' — forces use of publish-release.sh
# Used as PreToolUse hook for Bash commands
#
# stdin: JSON with tool input (has "command" field)
# exit 2 = block the tool call with message shown to agent

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.command // empty')

if echo "$COMMAND" | grep -qE '(^|[;&|])[ ]*gh release create'; then
    echo "❌ BLOCKED: Direct 'gh release create' is not allowed."
    echo "   Use /publish-release <version> instead."
    echo ""
    echo "   This ensures binaries are attached and prepared notes are used."
    exit 2
fi