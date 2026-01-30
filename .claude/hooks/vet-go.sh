#!/bin/bash
# Run gopls diagnostics on edited Go files to catch issues early
# Includes hints for performance improvements (5-15% gains per CLAUDE.md)
# Used as PostToolUse hook for Write/Edit
#
# ADVISORY ONLY: This hook reports diagnostics but does not block edits.
# Claude sees the output and addresses findings as part of the edit flow.

FILE="$1"

# Skip non-Go files silently
[[ "$FILE" != *.go ]] && exit 0

# Skip if file doesn't exist (was deleted)
[[ ! -f "$FILE" ]] && exit 0

# Normalize to repo-relative path for consistent gopls behavior
# This handles both absolute paths and paths relative to repo root
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)
if [[ -n "$REPO_ROOT" && "$FILE" == /* ]]; then
    # Convert absolute path to repo-relative
    FILE="${FILE#"$REPO_ROOT"/}"
fi

# Run gopls check with -severity=hint to include performance hints
# This catches: build errors, vet checks, static analysis, and hints
OUTPUT=$(gopls check -severity=hint "$FILE" 2>&1)

if [[ -n "$OUTPUT" ]]; then
    echo "📋 gopls diagnostics for $FILE:"
    echo "$OUTPUT"
    echo ""
    echo "💡 Address all findings above (hints improve perf 5-15%)"
fi

# Exit 0 (advisory only) - diagnostics inform but don't block
