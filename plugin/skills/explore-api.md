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

## Step 2: List endpoints

Call `walk_operations` to list all API endpoints:

```json
{"spec": {"file": "<path>"}}
```

Present them grouped by tag or by path prefix. For each operation show the method, path, and summary.

If the API is large, filter by tag or path to focus:

```json
{"spec": {"file": "<path>"}, "tag": "Users"}
```

```json
{"spec": {"file": "<path>"}, "path": "/users/*"}
```

## Step 3: List data models

Call `walk_schemas` to list the API's data models:

```json
{"spec": {"file": "<path>"}}
```

Summarize the schemas by name and type. To focus on component schemas (not inline ones):

```json
{"spec": {"file": "<path>"}, "component": true}
```

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

## Step 5: Summarize findings

Provide a structured summary of the API:
- Purpose and scope (from title/description)
- Authentication methods (from security schemes)
- Key resource groups (from tags)
- Notable patterns (versioning, pagination, common response shapes)
- Any concerns (deprecated endpoints, missing descriptions, inconsistencies)
