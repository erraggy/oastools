#!/usr/bin/env bash
set -euo pipefail

# check-binary.sh - Verify oastools binary is installed and check version alignment.
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

# Extract binary version (e.g., "1.51.3" from "oastools v1.51.3 ...").
RAW_VERSION=$(oastools version 2>/dev/null | head -1 | grep -oE 'v[^ ]+' | sed 's/^v//' || echo "unknown")
if echo "$RAW_VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    BINARY_VERSION="$RAW_VERSION"
elif [ "$RAW_VERSION" = "dev" ]; then
    BINARY_VERSION="dev"
else
    BINARY_VERSION="unknown"
fi

# Extract plugin version from plugin.json (no jq dependency).
PLUGIN_JSON="${CLAUDE_PLUGIN_ROOT:-.}/.claude-plugin/plugin.json"
if [ -f "$PLUGIN_JSON" ]; then
    PLUGIN_VERSION=$(grep -oE '"version"[[:space:]]*:[[:space:]]*"[^"]*"' "$PLUGIN_JSON" | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || echo "unknown")
else
    PLUGIN_VERSION="unknown"
fi

# Compare versions.
if [ "$BINARY_VERSION" = "unknown" ] || [ "$BINARY_VERSION" = "dev" ]; then
    # Dev build or unparseable version — skip comparison.
    echo "oastools v${BINARY_VERSION} (binary: ${BINARY_VERSION}, plugin: ${PLUGIN_VERSION}, status: ${BINARY_VERSION})"
elif [ "$PLUGIN_VERSION" = "unknown" ]; then
    # Plugin version not found — just report binary version.
    echo "oastools v${BINARY_VERSION}"
elif [ "$BINARY_VERSION" = "$PLUGIN_VERSION" ]; then
    echo "oastools v${BINARY_VERSION} (binary: v${BINARY_VERSION}, plugin: v${PLUGIN_VERSION}, status: ok)"
else
    echo "oastools v${BINARY_VERSION} (binary: v${BINARY_VERSION}, plugin: v${PLUGIN_VERSION}, status: MISMATCH)"
    echo ""
    echo "WARNING: Binary version (v${BINARY_VERSION}) does not match plugin version (v${PLUGIN_VERSION})."
    echo "Some MCP tool parameters may be unavailable or behave unexpectedly."
    echo ""
    echo "Update the binary:"
    echo "  brew upgrade erraggy/oastools/oastools"
    echo "  # or: go install github.com/erraggy/oastools/cmd/oastools@v${PLUGIN_VERSION}"
fi
