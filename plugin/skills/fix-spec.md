---
name: fix-spec
description: Auto-fix common OpenAPI spec issues with a preview-first workflow
---

# Fix an OpenAPI Specification

## Step 1: Preview fixes with dry run

Always start with a dry run to see what would change:

```json
{
  "spec": {"file": "<path>"},
  "dry_run": true
}
```

Include any specific fix flags the user requested:
- `fix_schema_names` -- Rename generic names like Object1, Model2 to meaningful names
- `fix_duplicate_operationids` -- Deduplicate operationId values
- `prune` -- Remove empty paths and unused schemas
- `stub_missing_refs` -- Create stub schemas for missing `$ref` targets

If no specific flags are given, the tool applies the default fix (missing path parameters).

## Step 2: Present the preview

Show the user a clear summary of planned fixes:
- Number of fixes that will be applied
- For each fix: what it changes, where (JSON path), and why

Ask the user to confirm before proceeding. If they want to exclude certain fixes, adjust the flags accordingly.

## Step 3: Apply fixes

Run the `fix` tool without `dry_run`:

```json
{
  "spec": {"file": "<path>"},
  "fix_schema_names": true
}
```

Add `"include_document": true` if the user wants to see the full corrected spec in the response.

## Step 4: Validate the result

Run the `validate` tool on the fixed spec to confirm it is now valid:

```json
{"spec": {"file": "<path>"}}
```

Report the final validation status. If new issues were introduced, investigate and explain them.

## Step 5: Summary

Provide a final summary:
- Total fixes applied
- Remaining issues (if any)
- Suggested next steps (e.g., manual review of renamed schemas, re-running code generation)
