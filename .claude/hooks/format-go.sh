#!/bin/bash
# Auto-format Go files after Write/Edit
# Strict mode: exit 1 on failure

FILE="$1"

# Skip non-Go files silently
[[ "$FILE" != *.go ]] && exit 0

# Skip if file doesn't exist (was deleted)
[[ ! -f "$FILE" ]] && exit 0

# Run gofmt
if ! gofmt -w "$FILE" 2>&1; then
    echo "❌ gofmt failed on $FILE"
    exit 1
fi

# Run goimports
if ! goimports -w "$FILE" 2>&1; then
    echo "❌ goimports failed on $FILE"
    exit 1
fi

echo "✅ Formatted: $FILE"
