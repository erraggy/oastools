---
name: explore-api
description: Explore an OpenAPI spec's structure using parse and walk tools
---

# Explore an API

## Step 1: Get the high-level overview

Call `parse` to understand the API at a glance:

```json
{"spec": {"file": "<path>"}}
```

Report:
- API title and version
- OAS version (2.0, 3.0, 3.1)
- Number of paths, operations, and schemas
- Server URLs
- Tags (these organize operations into groups)

## Handling Large APIs

If `parse` shows 100+ operations or 200+ schemas, adjust your strategy:

- ⚠️ **Filter first, page second.** Use `tag`, `method`, or `path` filters on walk tools rather than paging through all results.
- ✅ **Use `component: true` for schemas.** Without it, `walk_schemas` returns ALL schemas including inline ones (request bodies, response wrappers, etc.), which can be 3-5x the number of named component schemas. Start with `component: true` to see the data model, then omit it only when hunting for inline schema issues.
- ✅ **Walk by tag.** If the API has tags, use them. `walk_operations` with `tag` filter is the fastest way to understand a specific area.
- ❌ **Avoid `detail: true` on unfiltered walks.** Full operation objects can be very large. Get summaries first, then drill into specific operations.

## Step 2: List endpoints

Call `walk_operations` to list all API endpoints:

```json
{"spec": {"file": "<path>"}}
```

Present them grouped by tag or by path prefix. For each operation show the method, path, and summary.

For a quick overview of a large API, use `group_by` first:

```json
{"spec": {"file": "<path>"}, "group_by": "tag"}
```

This returns operation counts per tag — the fastest way to understand API scope.

```json
{"spec": {"file": "<path>"}, "group_by": "method"}
```

This shows the HTTP method distribution (e.g., 60% GET, 25% POST, etc.).

⚠️ If the API is large (more endpoints than the default page of 100), prefer **filtering** over paging:

```json
{"spec": {"file": "<path>"}, "tag": "Users"}
```

```json
{"spec": {"file": "<path>"}, "path": "/users/*"}
```

For deeply nested APIs, use `**` to match across path depths:

```json
{"spec": {"file": "<path>"}, "path": "/drives/**/workbook/**"}
```

✅ When `returned < matched`, use `offset` to page through remaining results:

```json
{"spec": {"file": "<path>"}, "offset": 100, "limit": 100}
```

## Step 3: List data models

Call `walk_schemas` to list the API's data models:

```json
{"spec": {"file": "<path>"}}
```

✅ Summarize the schemas by name and type. **Always start with `component: true`** for large APIs — this shows only the named schemas from `components/schemas` (or `definitions` in OAS 2.0), filtering out inline schemas that clutter the results:

```json
{"spec": {"file": "<path>"}, "component": true}
```

Get schema type distribution:

```json
{"spec": {"file": "<path>"}, "group_by": "type", "component": true}
```

⚠️ Omit `component` only when you need to find inline schemas (e.g., hunting for unnamed request body schemas).

## Step 4: Drill into specifics

Based on what the user is interested in, drill deeper:

**Specific endpoint details:**

```json
{"spec": {"file": "<path>"}, "operation_id": "getUser", "detail": true}
```

**Parameters for a path:**

```json
{"spec": {"file": "<path>"}, "path": "/users/{id}", "detail": true}
```

(using `walk_parameters`)

**Responses for an endpoint:**

```json
{"spec": {"file": "<path>"}, "path": "/users", "method": "get", "detail": true}
```

(using `walk_responses`)

**Security schemes:**

```json
{"spec": {"file": "<path>"}}
```

(using `walk_security`)

**Response headers:**

```json
{"spec": {"file": "<path>"}, "group_by": "name"}
```

(using `walk_headers` — shows which headers are most common across the API)

**Headers for a specific endpoint:**

```json
{"spec": {"file": "<path>"}, "path": "/users", "method": "get"}
```

(using `walk_headers`)

## Step 5: Analyze references

Call `walk_refs` to understand which schemas and responses are most referenced:

```json
{"spec": {"file": "<path>"}}
```

This returns unique `$ref` targets ranked by count (most-referenced first). Use this to identify the API's core data models.

**Trace a specific schema's usage:**

```json
{"spec": {"file": "<path>"}, "target": "*schemas/User*", "detail": true}
```

This shows every location that references the schema — useful for understanding impact before modifying a schema.

**Filter by ref type:**

```json
{"spec": {"file": "<path>"}, "node_type": "response"}
```

Narrow to schema, parameter, response, requestBody, header, or pathItem refs.

## Step 6: Summarize findings

Provide a structured summary of the API:
- Purpose and scope (from title/description)
- Authentication methods (from security schemes)
- Key resource groups (from tags)
- Notable patterns (versioning, pagination, common response shapes)
- Any concerns (deprecated endpoints, missing descriptions, inconsistencies)
