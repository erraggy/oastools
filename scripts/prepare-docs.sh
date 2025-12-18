#!/bin/bash
set -e

# Directory where mkdocs expects the documentation
DOCS_DIR="docs"

echo "Preparing documentation in $DOCS_DIR..."

# Ensure the target directories exist
mkdir -p "$DOCS_DIR/packages"

# 1. Copy README.md to index.md (Home page)
# We filter out badges or tweak content if needed, but a direct copy is usually fine.
echo "Copying README.md to $DOCS_DIR/index.md"
cp README.md "$DOCS_DIR/index.md"

# Fix links in index.md
# 1. Change docs/filename.md to filename.md (since index.md is now in docs/)
# 2. Change package/deep_dive.md to packages/package.md
# 3. Change src="docs/ to src=" (for HTML img tags)
sed 's|](docs/|](|g' "$DOCS_DIR/index.md" > "$DOCS_DIR/index.md.tmp" && mv "$DOCS_DIR/index.md.tmp" "$DOCS_DIR/index.md"
sed 's|\([a-z]*\)/deep_dive.md|packages/\1.md|g' "$DOCS_DIR/index.md" > "$DOCS_DIR/index.md.tmp" && mv "$DOCS_DIR/index.md.tmp" "$DOCS_DIR/index.md"
sed 's|src="docs/|src="|g' "$DOCS_DIR/index.md" > "$DOCS_DIR/index.md.tmp" && mv "$DOCS_DIR/index.md.tmp" "$DOCS_DIR/index.md"

# 2. Copy root level docs (user-facing only; internal dev docs stay in GitHub)
echo "Copying root level markdown files..."
cp CONTRIBUTORS.md "$DOCS_DIR/" || true
cp LICENSE "$DOCS_DIR/LICENSE.md" || true
cp benchmarks.md "$DOCS_DIR/" || true

# 3. Gather Deep Dive documentation from subdirectories
# We look for folders containing a 'deep_dive.md' and copy it to docs/packages/<dirname>.md
echo "Gathering Deep Dive documentation..."

for dir in */ ; do
    dirname=$(basename "$dir")
    deep_dive_file="${dirname}/deep_dive.md"
    
    if [ -f "$deep_dive_file" ]; then
        target_file="$DOCS_DIR/packages/${dirname}.md"
        echo "  - Found $deep_dive_file -> $target_file"
        cp "$deep_dive_file" "$target_file"
        
        # Optional: Add a note at the top linking back to source? 
        # For now, we just copy.
    fi
done

echo "Documentation preparation complete."
