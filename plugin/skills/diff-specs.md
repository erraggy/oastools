---
name: diff-specs
description: Compare two OpenAPI spec versions, highlight breaking changes, and suggest migration steps
---

# Diff Two OpenAPI Specifications

## Step 1: Compare the specs

Call the `diff` tool with both spec versions:

```json
{
  "base": {"file": "<old-version-path>"},
  "revision": {"file": "<new-version-path>"}
}
```

To focus only on breaking changes:

```json
{
  "base": {"file": "<old-version-path>"},
  "revision": {"file": "<new-version-path>"},
  "breaking_only": true
}
```

## Step 2: Categorize changes

Present the diff results organized by severity:

1. **Breaking (critical/error)** -- Changes that will break existing API consumers
   - Removed endpoints or operations
   - Removed or renamed required parameters
   - Changed response status codes
   - Narrowed allowed values (enum removals, stricter validation)
   - Changed authentication requirements

2. **Warnings** -- Changes that may affect consumers
   - Deprecated endpoints
   - Changed optional parameter defaults
   - Modified response schemas (new required fields)

3. **Informational** -- Non-breaking changes
   - Added endpoints or operations
   - Added optional parameters
   - Extended enum values
   - Documentation updates

## Step 3: Explain breaking changes

For each breaking change:
- Describe **what** changed (the specific path and property)
- Explain **why** it breaks consumers (e.g., "Clients sending requests to `DELETE /users/{id}` will get 404")
- Estimate the **scope of impact** (how many operations/schemas are affected)

## Step 4: Suggest migration path

For each breaking change, provide a concrete migration step:

| Change type | Migration guidance |
|------------|-------------------|
| Removed endpoint | Update client to use the replacement endpoint (if one exists) |
| Removed parameter | Remove the parameter from requests; check if behavior changed |
| Type change | Update request/response handling for the new type |
| Auth change | Update authentication configuration |
| Renamed field | Update all references to use the new name |

## Step 5: Summary

Provide a migration checklist:
- Total changes (breaking / warning / info)
- List of required client-side updates
- Recommended testing strategy for the migration
- Whether the version bump follows semver conventions (breaking changes = major bump)
