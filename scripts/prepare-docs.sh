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
# Convert examples/ relative links to local docs paths
# First handle petstore specifically (directory with subdirs, needs index.md)
sed 's|](examples/petstore/)|](examples/petstore/index.md)|g' "$DOCS_DIR/index.md" > "$DOCS_DIR/index.md.tmp" && mv "$DOCS_DIR/index.md.tmp" "$DOCS_DIR/index.md"
# Handle nested paths: examples/workflows/validate-and-fix/ -> examples/workflows/validate-and-fix.md
sed 's|](examples/workflows/\([^/)]*\)/)|](examples/workflows/\1.md)|g' "$DOCS_DIR/index.md" > "$DOCS_DIR/index.md.tmp" && mv "$DOCS_DIR/index.md.tmp" "$DOCS_DIR/index.md"
sed 's|](examples/programmatic-api/\([^/)]*\)/)|](examples/programmatic-api/\1.md)|g' "$DOCS_DIR/index.md" > "$DOCS_DIR/index.md.tmp" && mv "$DOCS_DIR/index.md.tmp" "$DOCS_DIR/index.md"
# Handle top-level examples: examples/quickstart/ -> examples/quickstart.md
sed 's|](examples/\([^/)]*\)/)|](examples/\1.md)|g' "$DOCS_DIR/index.md" > "$DOCS_DIR/index.md.tmp" && mv "$DOCS_DIR/index.md.tmp" "$DOCS_DIR/index.md"
# Fix examples/README.md -> examples/index.md
sed 's|examples/README.md)|examples/index.md)|g' "$DOCS_DIR/index.md" > "$DOCS_DIR/index.md.tmp" && mv "$DOCS_DIR/index.md.tmp" "$DOCS_DIR/index.md"
# Fix examples/ directory link -> examples/index.md
sed 's|](examples/)|](examples/index.md)|g' "$DOCS_DIR/index.md" > "$DOCS_DIR/index.md.tmp" && mv "$DOCS_DIR/index.md.tmp" "$DOCS_DIR/index.md"

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

# 4. Copy Examples documentation
echo "Copying Examples documentation..."
mkdir -p "$DOCS_DIR/examples/workflows" "$DOCS_DIR/examples/programmatic-api" "$DOCS_DIR/examples/petstore"

# Helper function to copy and fix links in example READMEs
copy_example_readme() {
    local src="$1"
    local dest="$2"
    cp "$src" "$dest"
    # Fix relative links to specs/ files - point to GitHub since we don't copy specs
    sed -i.bak 's|](specs/|](https://github.com/erraggy/oastools/blob/main/'"$(dirname "$src" | sed 's|^\./||')"'/specs/|g' "$dest"
    # Fix relative links to main.go - point to GitHub
    sed -i.bak 's|](main.go)|](https://github.com/erraggy/oastools/blob/main/'"$(dirname "$src" | sed 's|^\./||')"'/main.go)|g' "$dest"
    # Fix relative links to spec/ files - point to GitHub
    sed -i.bak 's|](spec/|](https://github.com/erraggy/oastools/blob/main/'"$(dirname "$src" | sed 's|^\./||')"'/spec/|g' "$dest"
    rm -f "$dest.bak"
}

# Helper function to fix directory-style links to .md files in copied docs
# The tricky part: original links like ../sibling/ become same-level links when
# README.md is flattened to sibling.md
fix_example_links() {
    local file="$1"

    # From top-level examples (quickstart.md, validation-pipeline.md):
    # ../sibling/ -> sibling.md (was going up from quickstart/README.md, now same level)
    # ../petstore/ -> petstore/index.md (folder with children)
    # ../workflows/... -> workflows/...

    # From workflow files (workflows/validate-and-fix.md):
    # ../sibling/ -> sibling.md (sibling workflow)
    # ../../sibling/ -> ../sibling.md (up to examples level)
    # ../../petstore/ -> ../petstore/index.md

    # Specific directory -> index.md mappings (folders that have subdirs)
    sed -i.bak 's|](petstore/)|](petstore/index.md)|g' "$file"
    sed -i.bak 's|](\.\./petstore/)|](petstore/index.md)|g' "$file"
    sed -i.bak 's|](\.\./\.\./petstore/)|](../petstore/index.md)|g' "$file"
    sed -i.bak 's|](workflows/)|](workflows/index.md)|g' "$file"
    sed -i.bak 's|](\.\./workflows/)|](workflows/index.md)|g' "$file"
    sed -i.bak 's|](programmatic-api/)|](programmatic-api/index.md)|g' "$file"
    sed -i.bak 's|](\.\./programmatic-api/)|](programmatic-api/index.md)|g' "$file"
    sed -i.bak 's|](\.\./\.\./programmatic-api/)|](../programmatic-api/index.md)|g' "$file"

    # From top-level: ../sibling/ -> sibling.md
    sed -i.bak 's|](\.\./quickstart/)|](quickstart.md)|g' "$file"
    sed -i.bak 's|](\.\./validation-pipeline/)|](validation-pipeline.md)|g' "$file"

    # From workflows: ../../top-level/ -> ../top-level.md
    sed -i.bak 's|](\.\./\.\./quickstart/)|](../quickstart.md)|g' "$file"
    sed -i.bak 's|](\.\./\.\./validation-pipeline/)|](../validation-pipeline.md)|g' "$file"

    # From workflows: ../sibling-workflow/ -> sibling-workflow.md
    sed -i.bak 's|](\.\./validate-and-fix/)|](validate-and-fix.md)|g' "$file"
    sed -i.bak 's|](\.\./version-conversion/)|](version-conversion.md)|g' "$file"
    sed -i.bak 's|](\.\./multi-api-merge/)|](multi-api-merge.md)|g' "$file"
    sed -i.bak 's|](\.\./breaking-change-detection/)|](breaking-change-detection.md)|g' "$file"
    sed -i.bak 's|](\.\./overlay-transformations/)|](overlay-transformations.md)|g' "$file"
    sed -i.bak 's|](\.\./http-validation/)|](http-validation.md)|g' "$file"

    # Nested paths like ../workflows/http-validation/ -> workflows/http-validation.md
    sed -i.bak 's|](\.\./workflows/\([a-z-]*\)/)|](workflows/\1.md)|g' "$file"
    sed -i.bak 's|](\.\./\.\./workflows/\([a-z-]*\)/)|](../workflows/\1.md)|g' "$file"

    # programmatic-api paths
    sed -i.bak 's|](\.\./programmatic-api/builder/)|](programmatic-api/builder.md)|g' "$file"
    sed -i.bak 's|](\.\./\.\./programmatic-api/builder/)|](../programmatic-api/builder.md)|g' "$file"
    sed -i.bak 's|](builder/)|](builder.md)|g' "$file"
    sed -i.bak 's|](\.\./builder/)|](builder.md)|g' "$file"

    # Simple same-level directories: dir/ -> dir.md (for child links in indices)
    sed -i.bak 's|](validate-and-fix/)|](validate-and-fix.md)|g' "$file"
    sed -i.bak 's|](version-conversion/)|](version-conversion.md)|g' "$file"
    sed -i.bak 's|](multi-api-merge/)|](multi-api-merge.md)|g' "$file"
    sed -i.bak 's|](breaking-change-detection/)|](breaking-change-detection.md)|g' "$file"
    sed -i.bak 's|](overlay-transformations/)|](overlay-transformations.md)|g' "$file"
    sed -i.bak 's|](http-validation/)|](http-validation.md)|g' "$file"
    sed -i.bak 's|](quickstart/)|](quickstart.md)|g' "$file"
    sed -i.bak 's|](validation-pipeline/)|](validation-pipeline.md)|g' "$file"
    sed -i.bak 's|](stdlib/)|](stdlib.md)|g' "$file"
    sed -i.bak 's|](chi/)|](chi.md)|g' "$file"

    rm -f "$file.bak"
}

# Fix links specifically for subdirectory context (workflows/, petstore/, programmatic-api/)
fix_subdir_links() {
    local file="$1"
    # From a subdirectory, sibling directories need ../
    sed -i.bak 's|](quickstart\.md)|](../quickstart.md)|g' "$file"
    sed -i.bak 's|](validation-pipeline\.md)|](../validation-pipeline.md)|g' "$file"
    sed -i.bak 's|](workflows/|](../workflows/|g' "$file"
    sed -i.bak 's|](petstore/|](../petstore/|g' "$file"
    sed -i.bak 's|](programmatic-api/|](../programmatic-api/|g' "$file"
    rm -f "$file.bak"
}

# Root examples README
cp examples/README.md "$DOCS_DIR/examples/index.md"
fix_example_links "$DOCS_DIR/examples/index.md"
echo "  - examples/README.md -> $DOCS_DIR/examples/index.md"

# Getting Started examples
copy_example_readme "examples/quickstart/README.md" "$DOCS_DIR/examples/quickstart.md"
fix_example_links "$DOCS_DIR/examples/quickstart.md"
echo "  - examples/quickstart/README.md -> $DOCS_DIR/examples/quickstart.md"
copy_example_readme "examples/validation-pipeline/README.md" "$DOCS_DIR/examples/validation-pipeline.md"
fix_example_links "$DOCS_DIR/examples/validation-pipeline.md"
echo "  - examples/validation-pipeline/README.md -> $DOCS_DIR/examples/validation-pipeline.md"

# Workflow examples
copy_example_readme "examples/workflows/README.md" "$DOCS_DIR/examples/workflows/index.md"
fix_example_links "$DOCS_DIR/examples/workflows/index.md"
fix_subdir_links "$DOCS_DIR/examples/workflows/index.md"
echo "  - examples/workflows/README.md -> $DOCS_DIR/examples/workflows/index.md"
for workflow in validate-and-fix version-conversion multi-api-merge breaking-change-detection overlay-transformations http-validation; do
    if [ -f "examples/workflows/$workflow/README.md" ]; then
        copy_example_readme "examples/workflows/$workflow/README.md" "$DOCS_DIR/examples/workflows/$workflow.md"
        fix_example_links "$DOCS_DIR/examples/workflows/$workflow.md"
        fix_subdir_links "$DOCS_DIR/examples/workflows/$workflow.md"
        echo "  - examples/workflows/$workflow/README.md -> $DOCS_DIR/examples/workflows/$workflow.md"
    fi
done

# Programmatic API examples
copy_example_readme "examples/programmatic-api/README.md" "$DOCS_DIR/examples/programmatic-api/index.md"
fix_example_links "$DOCS_DIR/examples/programmatic-api/index.md"
fix_subdir_links "$DOCS_DIR/examples/programmatic-api/index.md"
echo "  - examples/programmatic-api/README.md -> $DOCS_DIR/examples/programmatic-api/index.md"
copy_example_readme "examples/programmatic-api/builder/README.md" "$DOCS_DIR/examples/programmatic-api/builder.md"
fix_example_links "$DOCS_DIR/examples/programmatic-api/builder.md"
fix_subdir_links "$DOCS_DIR/examples/programmatic-api/builder.md"
echo "  - examples/programmatic-api/builder/README.md -> $DOCS_DIR/examples/programmatic-api/builder.md"

# Petstore examples
copy_example_readme "examples/petstore/README.md" "$DOCS_DIR/examples/petstore/index.md"
fix_example_links "$DOCS_DIR/examples/petstore/index.md"
fix_subdir_links "$DOCS_DIR/examples/petstore/index.md"
echo "  - examples/petstore/README.md -> $DOCS_DIR/examples/petstore/index.md"
copy_example_readme "examples/petstore/stdlib/README.md" "$DOCS_DIR/examples/petstore/stdlib.md"
fix_example_links "$DOCS_DIR/examples/petstore/stdlib.md"
fix_subdir_links "$DOCS_DIR/examples/petstore/stdlib.md"
echo "  - examples/petstore/stdlib/README.md -> $DOCS_DIR/examples/petstore/stdlib.md"
copy_example_readme "examples/petstore/chi/README.md" "$DOCS_DIR/examples/petstore/chi.md"
fix_example_links "$DOCS_DIR/examples/petstore/chi.md"
fix_subdir_links "$DOCS_DIR/examples/petstore/chi.md"
echo "  - examples/petstore/chi/README.md -> $DOCS_DIR/examples/petstore/chi.md"

echo "Documentation preparation complete."
