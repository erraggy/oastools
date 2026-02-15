#!/usr/bin/env bash
set -euo pipefail

# check-binary.sh - Verify oastools binary is installed and report version.
# Used as a SessionStart hook to give agents early awareness of the
# binary's presence and version before any MCP tool calls.

if ! command -v oastools &>/dev/null; then
    cat <<'MSG'
oastools binary is not installed. The oastools MCP server requires the
oastools CLI on your PATH.

Install via Homebrew:
  brew install erraggy/oastools/oastools

Or from source:
  go install github.com/erraggy/oastools/cmd/oastools@latest
MSG
    exit 2
fi

VERSION=$(oastools version 2>/dev/null | head -1 || echo "unknown")
echo "oastools $VERSION"
