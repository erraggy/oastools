---
name: validate-spec
description: Validate an OpenAPI spec, explain errors clearly, and suggest or apply fixes
---

# Validate an OpenAPI Specification

## Step 1: Run validation

Call the `validate` tool on the user's spec:

```json
{"spec": {"file": "<path>"}}
```

If the user wants strict validation, add `"strict": true`.

⚠️ **Tip for large specs:** Use `"no_warnings": true` for initial triage. Large specs can produce hundreds of warnings that obscure the actual errors. Get the error picture first, then run again without `no_warnings` to review warnings separately.

## Step 2: Report results

If the spec is valid:
- Confirm it passes validation
- Mention the detected OAS version
- Note any warnings (unless `no_warnings` was set)

If the spec has errors:
- List each error with its JSON path and a plain-language explanation
- Group related errors (e.g., multiple missing `$ref` targets)
- Explain **why** each error matters and what it would cause in practice (tooling failures, code generation issues, etc.)

### Paginating large results

Results are paginated (default limit: 100). When `returned < error_count`, there are more errors:

```json
{"spec": {"file": "<path>"}, "offset": 100, "limit": 100}
```

⚠️ **Strategy for large error sets:** Analyze the first page for patterns — if errors are repetitive (e.g., hundreds of missing path parameters), summarize the pattern and total count rather than paging through all of them. Only page further when errors appear diverse or the user needs specifics.

## Step 3: Suggest fixes

For each error, explain how to fix it. Common patterns:

| Error type | Suggested fix |
|-----------|---------------|
| Missing path parameter | Add the parameter to the operation's `parameters` array |
| Duplicate operationId | Rename to be unique, following a verb+resource pattern |
| Invalid `$ref` target | Fix the reference path, or stub the missing schema |
| Missing required field | Add the field with an appropriate value |

If the errors are auto-fixable, offer to run the `fix` tool:

```json
{"spec": {"file": "<path>"}, "dry_run": true}
```

Show the user the planned fixes before applying.

## Step 4: Re-validate after fixes

After any changes (manual edits or `fix` tool), run `validate` again to confirm the spec is now valid. Report the final status.
