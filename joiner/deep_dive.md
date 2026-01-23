<a id="top"></a>

# Joiner Package Deep Dive

!!! tip "Try it Online"
    No installation required! [Try the joiner in your browser →](https://oastools.robnrob.com/join)

## Table of Contents

- [Overview](#overview)
- [Key Concepts](#key-concepts)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [Operation-Aware Schema Renaming](#operation-aware-schema-renaming)
- [Limitations](#limitations)
- [Configuration Reference](#configuration-reference)
- [JoinResult Structure](#joinresult-structure)
- [Source Map Integration](#source-map-integration)
- [Package Chaining](#package-chaining)
- [Best Practices](#best-practices)
- [Common Patterns](#common-patterns)
- [CLI Usage](#cli-usage)

---

The [`joiner`](https://pkg.go.dev/github.com/erraggy/oastools/joiner) package merges multiple OpenAPI Specification documents into a single unified document. It provides sophisticated collision handling strategies, automatic reference rewriting, and semantic deduplication for large-scale API consolidation scenarios.

## Overview

When organizations maintain multiple API specifications—whether from different teams, microservices, or API modules—the joiner enables consolidation into a single document. This is particularly valuable for generating unified documentation, client SDKs, or gateway configurations from distributed API definitions.

The joiner supports OAS 2.0 documents merging with other 2.0 documents, and all OAS 3.x versions together (3.0.x, 3.1.x, 3.2.x). It uses the version and format (JSON or YAML) from the first document as the result format, ensuring consistency in the output.

[↑ Back to top](#top)

## Key Concepts

### Collision Handling

When merging multiple documents, name collisions are inevitable—two documents might define different schemas with the same name, or contain overlapping paths. The joiner provides seven collision strategies to handle these situations:

| Strategy | Behavior |
|----------|----------|
| `StrategyFailOnCollision` | Return error on any collision (default, safest) |
| `StrategyAcceptLeft` | Keep value from first/left document |
| `StrategyAcceptRight` | Keep value from last/right document (overwrite) |
| `StrategyFailOnPaths` | Fail only on path collisions, allow schema merging |
| `StrategyRenameLeft` | Rename left schema, keep right under original name |
| `StrategyRenameRight` | Rename right schema, keep left under original name |
| `StrategyDeduplicateEquivalent` | Merge structurally identical schemas |

Strategies can be set globally or per-component type (paths, schemas, other components), giving fine-grained control over merge behavior.

### Semantic Deduplication

Beyond handling same-named collisions, the joiner can identify and consolidate schemas that are structurally identical but have different names. When your Users API and Orders API both define equivalent `Address` and `Location` schemas, semantic deduplication recognizes they're identical and consolidates them.

### Reference Rewriting

When schemas are renamed or deduplicated, all `$ref` references throughout the merged document are automatically updated. This ensures the resulting document maintains valid internal references without manual intervention.

[↑ Back to top](#top)

## API Styles

See also: [Basic example](https://pkg.go.dev/github.com/erraggy/oastools/joiner#example-package), [Custom strategies example](https://pkg.go.dev/github.com/erraggy/oastools/joiner#example-package-CustomStrategies), [Semantic deduplication example](https://pkg.go.dev/github.com/erraggy/oastools/joiner#example-package-SemanticDeduplication) on pkg.go.dev

### Functional Options API

Best for single merge operations with inline configuration:

```go
result, err := joiner.JoinWithOptions(
    joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
    joiner.WithPathStrategy(joiner.StrategyFailOnCollision),
    joiner.WithSchemaStrategy(joiner.StrategyAcceptLeft),
)
```

### Struct-Based API

Best for multiple merge operations or complex configuration:

```go
config := joiner.DefaultConfig()
config.PathStrategy = joiner.StrategyFailOnPaths
config.SchemaStrategy = joiner.StrategyDeduplicateEquivalent
config.EquivalenceMode = "deep"

j := joiner.New(config)
result1, _ := j.Join([]string{"api1-base.yaml", "api1-ext.yaml"})
result2, _ := j.Join([]string{"api2-base.yaml", "api2-ext.yaml"})
```

[↑ Back to top](#top)

## Practical Examples

### Basic Document Joining

The simplest use case merges two or more documents with default settings:

```go
package main

import (
    "fmt"
    "log"
    "os"
    "path/filepath"
    
    "github.com/erraggy/oastools/joiner"
)

func main() {
    outputPath := filepath.Join(os.TempDir(), "merged.yaml")
    
    config := joiner.DefaultConfig()
    j := joiner.New(config)
    
    result, err := j.Join([]string{
        "users-api.yaml",
        "orders-api.yaml",
        "products-api.yaml",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    if err := j.WriteResult(result, outputPath); err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Successfully merged %d documents\n", 3)
    fmt.Printf("Output version: %s\n", result.Version)
    fmt.Printf("Total paths: %d\n", result.Stats.PathCount)
    fmt.Printf("Total schemas: %d\n", result.Stats.SchemaCount)
    fmt.Printf("Collisions resolved: %d\n", result.CollisionCount)
}
```

**Example Input (users-api.yaml):**
```yaml
openapi: 3.0.3
info:
  title: Users API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
```

**Example Input (orders-api.yaml):**
```yaml
openapi: 3.0.3
info:
  title: Orders API
  version: 1.0.0
paths:
  /orders:
    get:
      operationId: listOrders
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Order'
components:
  schemas:
    Order:
      type: object
      properties:
        id:
          type: integer
        userId:
          type: integer
```

**Example Output (merged.yaml):**
```yaml
openapi: 3.0.3
info:
  title: Users API  # Info from first document
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/User'
  /orders:
    get:
      operationId: listOrders
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Order'
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
    Order:
      type: object
      properties:
        id:
          type: integer
        userId:
          type: integer
```

### Handling Schema Collisions with Rename Strategies

When different APIs define schemas with the same name but different structures, use rename strategies to preserve both:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/erraggy/oastools/joiner"
)

func main() {
    config := joiner.DefaultConfig()
    
    // Keep left schema, rename right schema
    config.SchemaStrategy = joiner.StrategyRenameRight
    // Template for renamed schemas: "User_orders-api" format
    config.RenameTemplate = "{{.Name}}_{{.Source}}"
    
    j := joiner.New(config)
    
    result, err := j.Join([]string{
        "users-api.yaml",    // Has User schema (id, name, email)
        "orders-api.yaml",   // Has User schema (id, customerId) - different structure!
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Result will have:
    // - User (from users-api.yaml, original name)
    // - User_orders-api (from orders-api.yaml, renamed)
    // All $refs in orders-api paths are rewritten to User_orders-api
    
    fmt.Printf("Collisions resolved: %d\n", result.CollisionCount)
    for _, warning := range result.Warnings {
        fmt.Printf("  %s\n", warning)
    }
}
```

**Example Output:**
```
Collisions resolved: 1
  schema 'User' collision: right renamed to 'User_orders-api'
```

[Back to top](#top)

## Operation-Aware Schema Renaming

### The Problem

When joining OpenAPI specifications from different services, you often encounter generic schema names that collide. Consider two microservices:

**users-service.yaml:**
```yaml
openapi: 3.0.3
info:
  title: Users Service
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      tags: [users]
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Response'
components:
  schemas:
    Response:
      type: object
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/User'
        total:
          type: integer
    User:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
```

**orders-service.yaml:**
```yaml
openapi: 3.0.3
info:
  title: Orders Service
  version: 1.0.0
paths:
  /orders:
    get:
      operationId: listOrders
      tags: [orders]
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Response'
components:
  schemas:
    Response:
      type: object
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/Order'
        count:
          type: integer
    Order:
      type: object
      properties:
        id:
          type: integer
        userId:
          type: integer
```

Both services define a `Response` schema with different structures. A basic rename template like `{{.Name}}_{{.Source}}` would produce `Response_orders_service`—functional but not descriptive. For programmatically generated specs or code generation, you want names like `ListUsersResponse` and `ListOrdersResponse`.

### The Solution

Operation-aware renaming traces schemas back to their originating operations, enabling semantic names based on paths, methods, operation IDs, and tags:

```go
package main

import (
    "fmt"
    "log"

    "github.com/erraggy/oastools/joiner"
)

func main() {
    config := joiner.DefaultConfig()
    config.SchemaStrategy = joiner.StrategyRenameRight

    // Enable operation context for rich rename templates
    config.OperationContext = true

    // Use operation-derived naming
    config.RenameTemplate = "{{pascalCase .OperationID}}{{.Name}}"

    // Select how to pick the primary operation when a schema
    // is referenced by multiple operations
    config.PrimaryOperationPolicy = joiner.PolicyMostSpecific

    j := joiner.New(config)

    result, err := j.Join([]string{
        "users-service.yaml",
        "orders-service.yaml",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Result will have:
    // - Response (from users-service, kept original)
    // - ListOrdersResponse (from orders-service, renamed with context)

    fmt.Printf("Schemas renamed with operation context\n")
    fmt.Printf("Collisions: %d\n", result.CollisionCount)
}
```

### How It Works

The joiner builds a **reference graph** that maps each schema to the operations that use it. This graph captures both direct references (operation → schema) and indirect references (operation → schema → nested schema).

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   GET /users    │────▶│    Response     │────▶│      User       │
│   listUsers     │     │   (schema)      │     │   (schema)      │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       ▲                       ▲
        │                       │                       │
        └───────────────────────┴───────────────────────┘
                      Reference Graph
```

The reference graph construction:

1. **Traverses all paths and operations** - Records which schemas are referenced in request bodies, responses, parameters, and headers
2. **Tracks schema-to-schema references** - Records `$ref` chains through properties, items, allOf/anyOf/oneOf, and other composition keywords
3. **Resolves lineage** - For any schema, walks up the reference chain to find all operations that ultimately use it
4. **Caches results** - Lineage is computed once and cached for efficient template evaluation

When a collision occurs and renaming is needed, the joiner:
1. Retrieves the operation lineage for the schema
2. Selects a primary operation based on the configured policy
3. Builds a `RenameContext` with all available operation metadata
4. Executes the rename template with this rich context

### RenameContext Reference

The `RenameContext` provides comprehensive metadata for rename template evaluation:

#### Core Fields (Always Available)

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `Name` | string | Original schema name | `"Response"` |
| `Source` | string | Source file name (sanitized, no extension) | `"orders_service"` |
| `Index` | int | Document index (0-based) | `1` |

#### Operation Context Fields (When `OperationContext` is true)

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `Path` | string | API path from primary operation | `"/orders"` |
| `Method` | string | HTTP method (lowercase) | `"get"` |
| `OperationID` | string | Operation ID if defined | `"listOrders"` |
| `Tags` | []string | Tags from primary operation | `["orders"]` |
| `UsageType` | string | Where schema is used | `"response"` |
| `StatusCode` | string | Response status code | `"200"` |
| `ParamName` | string | Parameter name (for parameter usage) | `"filter"` |
| `MediaType` | string | Content media type | `"application/json"` |
| `PrimaryResource` | string | First path segment (resource name) | `"orders"` |

#### Aggregate Context Fields (Multi-Operation Schemas)

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `AllPaths` | []string | All paths referencing this schema | `["/orders", "/orders/{id}"]` |
| `AllMethods` | []string | All HTTP methods (deduplicated) | `["get", "post"]` |
| `AllOperationIDs` | []string | All operation IDs (non-empty only) | `["listOrders", "getOrder"]` |
| `AllTags` | []string | All tags (deduplicated, sorted) | `["admin", "orders"]` |
| `RefCount` | int | Total operation references | `3` |
| `IsShared` | bool | True if used by multiple operations | `true` |

#### UsageType Values

| Value | Description |
|-------|-------------|
| `"request"` | Schema used in request body |
| `"response"` | Schema used in response body |
| `"parameter"` | Schema used in parameter definition |
| `"header"` | Schema used in header definition |
| `"callback"` | Schema used in callback definition |

### Template Functions Reference

The joiner provides built-in template functions for transforming context values:

#### Path Functions

| Function | Description | Example Input | Example Output |
|----------|-------------|---------------|----------------|
| `pathSegment` | Extract nth segment (0-indexed, negative from end) | `pathSegment "/users/{id}/orders" 0` | `"users"` |
| `pathSegment` | Negative index | `pathSegment "/users/{id}/orders" -1` | `"orders"` |
| `pathResource` | First non-parameter segment | `pathResource "/users/{id}/orders"` | `"users"` |
| `pathLast` | Last non-parameter segment | `pathLast "/users/{id}/orders"` | `"orders"` |
| `pathClean` | Sanitize path for naming | `pathClean "/users/{id}"` | `"users_id"` |

Path functions automatically skip path parameters (segments like `{id}` or `{userId}`).

#### Tag Functions

| Function | Description | Example Input | Example Output |
|----------|-------------|---------------|----------------|
| `firstTag` | First tag or empty string | `firstTag .Tags` | `"orders"` |
| `joinTags` | Join tags with separator | `joinTags .Tags "_"` | `"admin_orders"` |
| `hasTag` | Check if tag exists | `hasTag .Tags "admin"` | `true` |

#### Case Functions

| Function | Description | Example Input | Example Output |
|----------|-------------|---------------|----------------|
| `pascalCase` | PascalCase conversion | `pascalCase "list_orders"` | `"ListOrders"` |
| `camelCase` | camelCase conversion | `camelCase "list_orders"` | `"listOrders"` |
| `snakeCase` | snake_case conversion | `snakeCase "ListOrders"` | `"list_orders"` |
| `kebabCase` | kebab-case conversion | `kebabCase "ListOrders"` | `"list-orders"` |

Case functions handle various input formats: `snake_case`, `kebab-case`, `camelCase`, `PascalCase`, and space-separated words.

#### Conditional Helpers

| Function | Description | Example |
|----------|-------------|---------|
| `default` | Return fallback if value empty | `default .OperationID "Unknown"` |
| `coalesce` | First non-empty value | `coalesce .OperationID .Path .Name` |

### Primary Operation Policy

When a schema is referenced by multiple operations, the joiner must select one as the "primary" operation for template context. Three policies are available:

| Policy | Behavior | Best For |
|--------|----------|----------|
| `PolicyFirstEncountered` | Uses the first operation found during graph traversal | Deterministic results based on document order |
| `PolicyMostSpecific` | Prefers operations with operationId, then those with tags | Well-documented APIs with operation IDs |
| `PolicyAlphabetical` | Sorts by path+method, uses alphabetically first | Reproducible builds regardless of traversal order |

**Example: Policy behavior with a shared schema**

Consider an `Address` schema used by three operations:

```yaml
paths:
  /users/{id}:
    get:
      operationId: getUser
      tags: [users]
  /orders:
    post:
      operationId: createOrder
      tags: [orders]
  /shipping:
    get:
      # No operationId
      tags: [shipping]
```

| Policy | Selected Operation | Reason |
|--------|-------------------|--------|
| `PolicyFirstEncountered` | GET /users/{id} | First in document order |
| `PolicyMostSpecific` | GET /users/{id} | Has operationId (both GET /users and POST /orders do, but GET comes first) |
| `PolicyAlphabetical` | POST /orders | "/orders" + "post" comes before "/shipping" + "get" and "/users/{id}" + "get" |

Configure the policy in your joiner config:

```go
config := joiner.DefaultConfig()
config.OperationContext = true
config.PrimaryOperationPolicy = joiner.PolicyMostSpecific
```

### Example Template Patterns

Common rename template patterns for different scenarios:

| Scenario | Template | Example Output |
|----------|----------|----------------|
| Operation ID prefix | `{{pascalCase .OperationID}}{{.Name}}` | `ListOrdersResponse` |
| Resource-based | `{{pascalCase (pathResource .Path)}}{{.Name}}` | `OrdersResponse` |
| Tag-based | `{{pascalCase (firstTag .Tags)}}{{.Name}}` | `OrdersResponse` |
| Method + resource | `{{pascalCase .Method}}{{pascalCase (pathResource .Path)}}{{.Name}}` | `GetOrdersResponse` |
| Full path | `{{pascalCase (pathClean .Path)}}{{.Name}}` | `OrdersIdResponse` |
| With fallback | `{{pascalCase (coalesce .OperationID (pathResource .Path) .Source)}}{{.Name}}` | `ListOrdersResponse` |
| Versioned API | `{{.Name}}_{{pathSegment .Path 0}}_{{.Source}}` | `Response_v2_orders` |
| Response codes | `{{pascalCase .OperationID}}{{.StatusCode}}{{.Name}}` | `ListOrders200Response` |
| Shared indicator | `{{if .IsShared}}Shared{{end}}{{.Name}}_{{.Source}}` | `SharedResponse_orders` |

### Handling Shared Schemas

Schemas referenced by multiple operations require special consideration. Use the `IsShared` field to detect and handle these cases:

```go
// Template that indicates shared schemas
config.RenameTemplate = `{{if .IsShared}}Common{{else}}{{pascalCase .OperationID}}{{end}}{{.Name}}`

// Results:
// - Schema used by one operation: "ListOrdersResponse"
// - Schema used by multiple operations: "CommonResponse"
```

For more granular control, use aggregate fields:

```go
// Use all operation IDs for shared schemas
config.RenameTemplate = `{{if .IsShared}}{{range $i, $id := .AllOperationIDs}}{{if $i}}_{{end}}{{$id}}{{end}}_{{.Name}}{{else}}{{.OperationID}}_{{.Name}}{{end}}`

// Shared schema used by listOrders and getOrder: "listOrders_getOrder_Response"
// Single-use schema: "listOrders_Response"
```

## Limitations

### Operation Context for Base Document Schemas

When using `WithOperationContext(true)`, only schemas from the RIGHT (incoming) documents receive operation-derived context. The LEFT (base) document's schemas do not have their operation references traced.

This means for base document schemas, the following `RenameContext` fields will be empty:
- `Path`, `Method`, `OperationID`, `Tags`
- `UsageType`, `StatusCode`, `ParamName`, `MediaType`
- `AllPaths`, `AllMethods`, `AllOperationIDs`, `AllTags`
- `RefCount`, `PrimaryResource`, `IsShared`

Only the core fields (`Name`, `Source`, `Index`) are populated for base document schemas.

**Workaround**: If you need operation context for all schemas, consider restructuring your join order so the document with schemas requiring operation context is joined as the RIGHT document.

### OAS 2.0 Support

Operation-aware renaming works with both OAS 2.0 and OAS 3.x documents. The reference graph construction adapts to each version's structure:

| OAS Version | Request Body Detection | Schema Reference Path |
|-------------|----------------------|----------------------|
| OAS 2.0 | Body parameter with `in: body` | `#/definitions/SchemaName` |
| OAS 3.x | `requestBody.content.*.schema` | `#/components/schemas/SchemaName` |

For OAS 2.0 documents:

```go
config := joiner.DefaultConfig()
config.SchemaStrategy = joiner.StrategyRenameRight
config.OperationContext = true
config.RenameTemplate = "{{pascalCase .OperationID}}{{.Name}}"

j := joiner.New(config)

// Works with OAS 2.0 (Swagger) documents
result, err := j.Join([]string{
    "swagger-users.yaml",  // OAS 2.0
    "swagger-orders.yaml", // OAS 2.0
})
```

### Webhook Support (OAS 3.1+)

For OAS 3.1+ documents with webhooks, the reference graph includes webhook operations:

```yaml
webhooks:
  orderCreated:
    post:
      operationId: handleOrderCreated
      tags: [webhooks]
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/OrderEvent'
```

The `Path` field for webhook operations uses the format `webhook:<name>`:

| Template | Webhook Path | Result |
|----------|--------------|--------|
| `{{.Path}}` | orderCreated webhook | `webhook:orderCreated` |
| `{{pathResource .Path}}` | orderCreated webhook | `webhook` |

### Callback Support

Callbacks in OAS 3.0+ are also tracked in the reference graph. The path includes the parent operation and callback name:

```yaml
paths:
  /orders:
    post:
      operationId: createOrder
      callbacks:
        orderStatus:
          '{$request.body#/callbackUrl}':
            post:
              requestBody:
                content:
                  application/json:
                    schema:
                      $ref: '#/components/schemas/StatusUpdate'
```

For callback operations, the `Path` field uses the format `<parent_path>-><callback_name>:<callback_path>`:

```
/orders->orderStatus:{$request.body#/callbackUrl}
```

The `UsageType` will be `"callback"` for schemas referenced within callbacks.

### Debugging Rename Templates

To debug rename templates, use a template that outputs all available fields:

```go
// Debug template that shows all context
config.RenameTemplate = `DEBUG_{{.Name}}_path={{.Path}}_method={{.Method}}_op={{.OperationID}}_usage={{.UsageType}}_shared={{.IsShared}}`
```

This produces names like:
```
DEBUG_Response_path=/orders_method=get_op=listOrders_usage=response_shared=false
```

Once you've identified the available fields, simplify to your production template.

### Performance Considerations

Building the reference graph adds a traversal pass over the document. For most specifications, this overhead is negligible:

| Document Size | Typical Overhead |
|---------------|-----------------|
| Small (< 50 paths) | < 1ms |
| Medium (50-500 paths) | 1-5ms |
| Large (500+ paths) | 5-20ms |

The reference graph is built once per document and cached. Lineage resolution is also cached, so multiple schema renames reuse the same graph traversal results.

To minimize overhead when operation context is not needed:

```go
config := joiner.DefaultConfig()
config.OperationContext = false  // Default - skip graph building
config.RenameTemplate = "{{.Name}}_{{.Source}}"  // Core fields only
```

### Integration with Semantic Deduplication

Operation-aware renaming and semantic deduplication are complementary features:

| Feature | Purpose | When to Use |
|---------|---------|-------------|
| **Semantic Deduplication** | Consolidates structurally identical schemas | When schemas are duplicated across services |
| **Operation-Aware Renaming** | Creates meaningful names for colliding schemas | When schemas have the same name but different structures |

Use both together for comprehensive schema management:

```go
config := joiner.DefaultConfig()

// Consolidate identical schemas first
config.SemanticDeduplication = true

// For remaining collisions (same name, different structure),
// use operation-aware renaming
config.SchemaStrategy = joiner.StrategyRenameRight
config.OperationContext = true
config.RenameTemplate = "{{pascalCase .OperationID}}{{.Name}}"
config.PrimaryOperationPolicy = joiner.PolicyMostSpecific

j := joiner.New(config)
result, err := j.Join(files)
```

With this configuration:
1. Structurally identical schemas (e.g., `Address` in users and orders) are consolidated
2. Structurally different schemas with the same name (e.g., different `Response` schemas) are renamed with operation context

### Common Pitfalls

**Empty OperationID:** Not all APIs define operation IDs. Use fallbacks:

```go
// Bad - empty OperationID produces "Response"
config.RenameTemplate = "{{.OperationID}}{{.Name}}"

// Good - falls back to path resource
config.RenameTemplate = "{{pascalCase (coalesce .OperationID (pathResource .Path) .Source)}}{{.Name}}"
```

**Orphaned Schemas:** Schemas not referenced by any operation will have empty operation context. These typically include:

- Base schemas used only via `allOf`/`anyOf`/`oneOf`
- Schemas defined but never referenced
- Nested `$defs` schemas

For orphaned schemas, the template receives only core fields (`Name`, `Source`, `Index`). Design templates with fallbacks:

```go
// Handles orphaned schemas gracefully
config.RenameTemplate = `{{if .Path}}{{pascalCase .OperationID}}{{.Name}}{{else}}{{.Name}}_{{.Source}}{{end}}`
```

**Path Parameters in Templates:** Path functions skip parameters, but `pathClean` converts them:

```go
pathClean("/users/{id}")       // "users_id" - includes parameter name
pathResource("/users/{id}")    // "users" - excludes parameter
pathLast("/users/{id}/orders") // "orders" - excludes parameter
```

[Back to top](#top)

### Namespace Prefixes for Team-Based APIs

When consolidating APIs from different teams, namespace prefixes prevent collisions while maintaining clarity about schema origins:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/erraggy/oastools/joiner"
)

func main() {
    config := joiner.DefaultConfig()
    config.SchemaStrategy = joiner.StrategyAcceptLeft
    
    // Map source files to namespace prefixes
    config.NamespacePrefix = map[string]string{
        "users-api.yaml":   "Users",
        "billing-api.yaml": "Billing",
        "orders-api.yaml":  "Orders",
    }
    
    // Apply prefix to ALL schemas, not just collisions
    config.AlwaysApplyPrefix = true
    
    j := joiner.New(config)
    
    result, err := j.Join([]string{
        "users-api.yaml",
        "billing-api.yaml",
        "orders-api.yaml",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Schemas will be named:
    // Users_User, Users_Profile
    // Billing_Invoice, Billing_Payment
    // Orders_Order, Orders_LineItem
    
    fmt.Printf("Merged with namespace prefixes\n")
    fmt.Printf("Schema count: %d\n", result.Stats.SchemaCount)
}
```

### Semantic Deduplication Across Documents

When multiple APIs define structurally identical schemas with different names, semantic deduplication consolidates them:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/erraggy/oastools/joiner"
)

func main() {
    result, err := joiner.JoinWithOptions(
        joiner.WithFilePaths([]string{
            "users-api.yaml",    // Has Address schema
            "orders-api.yaml",   // Has ShippingAddress schema (identical structure)
            "billing-api.yaml",  // Has BillingAddress schema (identical structure)
        }),
        joiner.WithSemanticDeduplication(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // If Address, ShippingAddress, and BillingAddress are structurally identical,
    // they'll be consolidated to "Address" (alphabetically first)
    // All references are rewritten automatically
    
    fmt.Printf("Schema count after deduplication: %d\n", result.Stats.SchemaCount)
    for _, warning := range result.Warnings {
        fmt.Printf("  %s\n", warning)
    }
}
```

**Example Input Documents:**

users-api.yaml:
```yaml
components:
  schemas:
    Address:
      type: object
      properties:
        street:
          type: string
        city:
          type: string
        zip:
          type: string
```

orders-api.yaml:
```yaml
components:
  schemas:
    ShippingAddress:
      type: object
      properties:
        street:
          type: string
        city:
          type: string
        zip:
          type: string
```

**Example Output:**
```
Schema count after deduplication: 1
  semantic deduplication: consolidated 3 duplicate definition(s)
```

#### Empty Schemas Are Preserved

Empty schemas (those with no structural constraints) are automatically excluded from deduplication, even when they appear structurally identical. This is because empty schemas serve different semantic purposes depending on context:

- **Placeholders** for schemas to be defined later
- **"Any type" markers** that accept any value
- **Context-specific wildcards** with meaning derived from their name or position

A schema is considered "empty" if it has no type, format, properties, validation rules, or composition keywords. Metadata fields (title, description, example, deprecated) are NOT considered constraints.

```yaml
# users-api.yaml
components:
  schemas:
    AnyPayload: {}           # "Accept any request body"
    User:
      type: object
      properties:
        name:
          type: string

# events-api.yaml
components:
  schemas:
    DynamicData: {}           # "Event data can be anything"
    User:                     # Identical to users-api User
      type: object
      properties:
        name:
          type: string
```

After joining with semantic deduplication enabled:
- `AnyPayload` and `DynamicData` are **both preserved** (empty schemas are never consolidated)
- The two `User` schemas are consolidated into one (structurally identical, non-empty)

### Schema Equivalence Detection

For collision handling with `StrategyDeduplicateEquivalent`, configure the depth of structural comparison:

```go
package main

import (
    "log"
    
    "github.com/erraggy/oastools/joiner"
)

func main() {
    config := joiner.DefaultConfig()
    
    // Use deduplication for same-named schemas
    config.SchemaStrategy = joiner.StrategyDeduplicateEquivalent
    
    // Configure comparison depth:
    // "none"    - No comparison, always treat as collision
    // "shallow" - Compare top-level properties only
    // "deep"    - Full recursive structural comparison
    config.EquivalenceMode = "deep"
    
    j := joiner.New(config)
    
    // If both files have User schema with identical structure,
    // they'll be merged without error
    // If structures differ, join fails with collision error
    result, err := j.Join([]string{"api1.yaml", "api2.yaml"})
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Merged successfully, %d collisions resolved", result.CollisionCount)
}
```

### High-Performance Joining with Pre-Parsed Documents

For integration with other oastools packages, use pre-parsed documents for 154x faster performance:

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/erraggy/oastools/joiner"
    "github.com/erraggy/oastools/parser"
    "github.com/erraggy/oastools/validator"
)

func main() {
    // Parse and validate documents
    files := []string{"api1.yaml", "api2.yaml", "api3.yaml"}
    var parsed []parser.ParseResult
    
    for _, file := range files {
        p, err := parser.ParseWithOptions(
            parser.WithFilePath(file),
            parser.WithValidateStructure(true),
        )
        if err != nil {
            log.Fatalf("Failed to parse %s: %v", file, err)
        }
        
        // Validate before joining (required)
        v, err := validator.ValidateWithOptions(
            validator.WithParsed(*p),
        )
        if err != nil {
            log.Fatalf("Failed to validate %s: %v", file, err)
        }
        if !v.Valid {
            log.Fatalf("%s has validation errors", file)
        }
        
        parsed = append(parsed, *p)
    }
    
    // Join using pre-parsed documents (154x faster)
    start := time.Now()
    result, err := joiner.JoinWithOptions(
        joiner.WithParsed(parsed...),
        joiner.WithSchemaStrategy(joiner.StrategyAcceptLeft),
    )
    elapsed := time.Since(start)
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Join completed in %v\n", elapsed)
    fmt.Printf("Paths: %d, Schemas: %d\n", 
        result.Stats.PathCount, result.Stats.SchemaCount)
}
```

### Collision Report Generation

For debugging complex merges, enable collision reporting:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/erraggy/oastools/joiner"
)

func main() {
    config := joiner.DefaultConfig()
    config.SchemaStrategy = joiner.StrategyAcceptLeft
    config.CollisionReport = true  // Enable detailed reporting
    
    j := joiner.New(config)
    
    result, err := j.Join([]string{
        "api1.yaml",
        "api2.yaml",
        "api3.yaml",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    if result.CollisionDetails != nil {
        fmt.Printf("Collision Report:\n")
        fmt.Printf("  Total collisions: %d\n", result.CollisionDetails.TotalCount)
        fmt.Printf("  Schema collisions: %d\n", result.CollisionDetails.SchemaCount)
        fmt.Printf("  Path collisions: %d\n", result.CollisionDetails.PathCount)
        
        for _, collision := range result.CollisionDetails.Collisions {
            fmt.Printf("\n  %s collision: %s\n", collision.Type, collision.Name)
            fmt.Printf("    Sources: %v\n", collision.Sources)
            fmt.Printf("    Resolution: %s\n", collision.Resolution)
        }
    }
}
```

### Overlay Integration During Join

Apply transformations during the join process:

```go
package main

import (
    "log"
    
    "github.com/erraggy/oastools/joiner"
)

func main() {
    result, err := joiner.JoinWithOptions(
        joiner.WithFilePaths([]string{"api1.yaml", "api2.yaml"}),
        
        // Apply overlay to each input before merging
        joiner.WithPreJoinOverlayFile("normalize.yaml"),
        
        // Apply overlay to final result
        joiner.WithPostJoinOverlayFile("enhance.yaml"),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Join with overlays completed: %d paths", result.Stats.PathCount)
}
```

**Example Pre-Join Overlay (normalize.yaml):**
```yaml
overlay: 1.0.0
info:
  title: Normalization Overlay
actions:
  - target: $.paths.*.*.responses.*.description
    update:
      description: Standardized response
```

### Different Strategies per Component Type

Fine-grained control over collision handling for different specification elements:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/erraggy/oastools/joiner"
)

func main() {
    config := joiner.JoinerConfig{
        // Fail on path collisions - paths must be unique
        PathStrategy: joiner.StrategyFailOnCollision,
        
        // For schemas, rename collisions from the right document
        SchemaStrategy: joiner.StrategyRenameRight,
        RenameTemplate: "{{.Name}}_v{{.Index}}",
        
        // For other components (parameters, responses), keep left
        ComponentStrategy: joiner.StrategyAcceptLeft,
        
        // Merge arrays (servers, security requirements)
        MergeArrays: true,
        
        // Remove duplicate tags by name
        DeduplicateTags: true,
    }
    
    j := joiner.New(config)
    
    result, err := j.Join([]string{"base.yaml", "extension.yaml"})
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Merged with custom strategies\n")
    fmt.Printf("Warnings: %d\n", len(result.Warnings))
}
```

[↑ Back to top](#top)

## Configuration Reference

### JoinerConfig Fields

```go
type JoinerConfig struct {
    // Global default strategy for all collisions
    DefaultStrategy CollisionStrategy
    
    // Per-component type strategies (override DefaultStrategy)
    PathStrategy      CollisionStrategy
    SchemaStrategy    CollisionStrategy
    ComponentStrategy CollisionStrategy
    
    // Tag and array handling
    DeduplicateTags bool  // Remove duplicate tags by name
    MergeArrays     bool  // Merge servers, security, tags arrays
    
    // Rename strategy configuration
    RenameTemplate string              // Go template: "{{.Name}}_{{.Source}}"
    NamespacePrefix map[string]string  // Source file → prefix mapping
    AlwaysApplyPrefix bool             // Apply prefix to all schemas, not just collisions
    
    // Equivalence detection configuration
    EquivalenceMode string  // "none", "shallow", or "deep"
    
    // Reporting
    CollisionReport bool  // Generate detailed collision analysis
    
    // Post-processing
    SemanticDeduplication bool  // Consolidate identical schemas across documents
}
```

### Available Options

| Option | Description |
|--------|-------------|
| `WithFilePaths([]string)` | Input file paths or URLs |
| `WithParsed(docs ...ParseResult)` | Pre-parsed documents (154x faster) |
| `WithConfig(JoinerConfig)` | Full configuration object |
| `WithPathStrategy(CollisionStrategy)` | Strategy for path collisions |
| `WithSchemaStrategy(CollisionStrategy)` | Strategy for schema collisions |
| `WithComponentStrategy(CollisionStrategy)` | Strategy for other components |
| `WithSemanticDeduplication(bool)` | Enable cross-document deduplication |
| `WithPreJoinOverlayFile(string)` | Overlay applied to each input |
| `WithPostJoinOverlayFile(string)` | Overlay applied to merged result |

[↑ Back to top](#top)

## JoinResult Structure

```go
type JoinResult struct {
    // Document contains the merged document
    // (*parser.OAS2Document or *parser.OAS3Document)
    Document any
    
    // Version is the OpenAPI version string (e.g., "3.0.3")
    Version string
    
    // OASVersion is the enumerated version
    OASVersion parser.OASVersion
    
    // SourceFormat is the format of the first source file
    SourceFormat parser.SourceFormat
    
    // Warnings contains non-fatal issues encountered
    Warnings []string
    
    // CollisionCount tracks resolved collisions
    CollisionCount int
    
    // Stats contains document statistics
    Stats parser.DocumentStats
    
    // CollisionDetails contains detailed analysis
    // (when CollisionReport is enabled)
    CollisionDetails *CollisionReport
}
```

### JoinResult Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `ToParseResult()` | `*parser.ParseResult` | Converts result for package chaining |

[↑ Back to top](#top)

## Source Map Integration

Source maps enable **precise collision and warning locations** by tracking line and column numbers from your YAML/JSON source files. Without source maps, collision errors only show JSON paths. With source maps, collision errors include file:line:column positions that IDEs can click to jump directly to the conflict.

**Without source maps:**
```
schema collision: 'User' defined in users-api.yaml (components.schemas.User) and orders-api.yaml (components.schemas.User)
```

**With source maps:**
```
schema collision: 'User' defined in users-api.yaml:45:5 and orders-api.yaml:62:5
```

When joining multiple files, use `WithSourceMaps` (plural) to pass source maps for all input documents:

```go
sourceMaps := make(map[string]*parser.SourceMap)
var docs []parser.ParseResult

for _, path := range []string{"users-api.yaml", "orders-api.yaml"} {
    p, _ := parser.ParseWithOptions(
        parser.WithFilePath(path),
        parser.WithSourceMap(true),  // Enable line tracking during parse
    )
    sourceMaps[path] = p.SourceMap
    docs = append(docs, *p)
}

result, _ := joiner.JoinWithOptions(
    joiner.WithParsed(docs...),
    joiner.WithSourceMaps(sourceMaps),  // Pass all source maps (keyed by file path)
)

// Warnings and collision details now include line/column/file info
for _, warning := range result.Warnings {
    fmt.Println(warning)  // Includes file:line:column when available
}
```

The joiner uses `WithSourceMaps` (plural, with a map) because it needs source maps from multiple input files to track collision locations across documents.

[Back to top](#top)

## Package Chaining

The `ToParseResult()` method enables seamless chaining with other oastools packages by converting `JoinResult` to a `parser.ParseResult`:

```go
// Join then validate
joinResult, err := joiner.JoinWithOptions(
    joiner.WithFilePaths([]string{"users-api.yaml", "orders-api.yaml"}),
)
if err != nil {
    log.Fatal(err)
}

// Chain to validator
v := validator.New()
valResult, _ := v.ValidateParsed(*joinResult.ToParseResult())
fmt.Printf("Valid: %v\n", valResult.Valid)

// Or chain to converter
c := converter.New()
convResult, _ := c.ConvertParsed(*joinResult.ToParseResult(), "3.1.0")

// Or chain to fixer
fixResult, _ := fixer.FixWithOptions(
    fixer.WithParsed(*joinResult.ToParseResult()),
)
```

This enables workflows like: `parse → join → validate → convert → diff`

Note: Join warnings are converted to string warnings in the resulting ParseResult.

[Back to top](#top)

## Best Practices

**Always validate input documents before joining.** The joiner requires documents with no validation errors. Use the validator package first.

**Use StrategyFailOnCollision initially** to understand what collisions exist in your documents before choosing a resolution strategy.

**Choose collision strategies based on your use case:**
- **Fail strategies** for strict merging where collisions indicate problems
- **Accept strategies** when you have a clear "primary" document
- **Rename strategies** when you need to preserve both versions
- **Deduplicate strategies** when schemas should be consolidated

**Use namespace prefixes for team-based consolidation** to maintain clarity about schema origins in large merged documents.

**Enable semantic deduplication** when consolidating APIs that likely share common schemas (addresses, pagination, error responses).

**Use the parse-once pattern** when integrating with validation or other processing for 154x performance improvement.

[↑ Back to top](#top)

## Common Patterns

### Microservices Consolidation

```go
// Collect all service specs
services := []string{
    "auth-service/openapi.yaml",
    "user-service/openapi.yaml",
    "order-service/openapi.yaml",
    "payment-service/openapi.yaml",
}

config := joiner.DefaultConfig()
config.PathStrategy = joiner.StrategyFailOnCollision
config.SchemaStrategy = joiner.StrategyRenameRight
config.RenameTemplate = "{{.Name}}_{{.Source}}"
config.SemanticDeduplication = true

j := joiner.New(config)
result, err := j.Join(services)
```

### API Gateway Aggregation

```go
// Prefix schemas by team for gateway configuration
config.NamespacePrefix = map[string]string{
    "team-a/api.yaml": "TeamA",
    "team-b/api.yaml": "TeamB",
    "team-c/api.yaml": "TeamC",
}
config.AlwaysApplyPrefix = true
```

### Extension Document Pattern

```go
// Base API with extensions
config.PathStrategy = joiner.StrategyAcceptRight  // Extensions override base
config.SchemaStrategy = joiner.StrategyAcceptLeft  // Base schemas take priority

result, _ := j.Join([]string{
    "base-api.yaml",       // Core API
    "custom-extension.yaml", // Customer-specific additions
})
```

---

## CLI Usage

The joiner's operation-aware schema renaming and overlay integration features are fully accessible from the command line.

### Basic Schema Renaming

```bash
# Rename colliding schemas with source file suffix
oastools join --schema-strategy rename-right \
  --rename-template "{{.Name}}_{{.Source}}" \
  -o merged.yaml users-api.yaml orders-api.yaml
```

### Operation-Aware Renaming

Enable `--operation-context` to access path, method, operation ID, and tag information in templates:

```bash
# Use operation ID as prefix (e.g., "ListUsersResponse")
oastools join --schema-strategy rename-right --operation-context \
  --rename-template "{{.OperationID | pascalCase}}{{.Name}}" \
  -o merged.yaml api1.yaml api2.yaml

# Use path resource as prefix (e.g., "OrdersResponse")
oastools join --schema-strategy rename-right --operation-context \
  --rename-template "{{pathResource .Path | pascalCase}}{{.Name}}" \
  -o merged.yaml api1.yaml api2.yaml

# Full method + resource naming (e.g., "GetOrdersResponse")
oastools join --schema-strategy rename-right --operation-context \
  --rename-template "{{.Method | pascalCase}}{{pathResource .Path | pascalCase}}{{.Name}}" \
  -o merged.yaml api1.yaml api2.yaml
```

### Primary Operation Policy

Control which operation provides context when a schema is used by multiple operations:

```bash
# Prefer operations with operationId defined
oastools join --schema-strategy rename-right --operation-context \
  --primary-operation-policy most-specific \
  --rename-template "{{.OperationID | default .Name}}" \
  -o merged.yaml api1.yaml api2.yaml

# Alphabetical for reproducible builds
oastools join --schema-strategy rename-right --operation-context \
  --primary-operation-policy alphabetical \
  --rename-template "{{.OperationID | pascalCase}}{{.Name}}" \
  -o merged.yaml api1.yaml api2.yaml
```

### Template Patterns

Common template patterns for different use cases:

```bash
# With fallback for schemas without operationId
oastools join --schema-strategy rename-right --operation-context \
  --rename-template "{{coalesce .OperationID (pathResource .Path) .Source | pascalCase}}{{.Name}}" \
  -o merged.yaml api1.yaml api2.yaml

# Different handling for shared vs single-use schemas
oastools join --schema-strategy rename-right --operation-context \
  --rename-template "{{if .IsShared}}Shared{{else}}{{.OperationID | pascalCase}}{{end}}{{.Name}}" \
  -o merged.yaml api1.yaml api2.yaml

# Include response status code
oastools join --schema-strategy rename-right --operation-context \
  --rename-template "{{.OperationID | pascalCase}}{{.StatusCode}}{{.Name}}" \
  -o merged.yaml api1.yaml api2.yaml

# Tag-based naming
oastools join --schema-strategy rename-right --operation-context \
  --rename-template "{{firstTag .Tags | pascalCase}}{{.Name}}" \
  -o merged.yaml api1.yaml api2.yaml
```

### Overlay Integration

Apply overlays during the join process:

```bash
# Pre-overlay: applied to each input before merging
# Post-overlay: applied to the final merged result
oastools join \
  --pre-overlay normalize.yaml \
  --post-overlay enhance.yaml \
  -o merged.yaml api1.yaml api2.yaml

# Multiple overlays (applied in order)
oastools join \
  --pre-overlay strip-internal.yaml \
  --pre-overlay standardize-responses.yaml \
  --post-overlay add-security.yaml \
  --post-overlay add-metadata.yaml \
  -o merged.yaml api1.yaml api2.yaml
```

**Example pre-overlay (normalize.yaml):**
```yaml
overlay: 1.0.0
info:
  title: Normalization Overlay
  version: 1.0.0
actions:
  - target: $..description
    update:
      description: ""  # Clear all descriptions before merge
```

**Example post-overlay (enhance.yaml):**
```yaml
overlay: 1.0.0
info:
  title: Enhancement Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: "Unified API"
      version: "1.0.0"
  - target: $.servers
    update:
      - url: https://api.example.com/v1
        description: Production
```

### Combined Features

Use multiple features together for comprehensive join operations:

```bash
# Full-featured join with all options
oastools join \
  --schema-strategy rename-right \
  --operation-context \
  --primary-operation-policy most-specific \
  --rename-template "{{coalesce .OperationID (pathResource .Path) | pascalCase}}{{.Name}}" \
  --semantic-dedup \
  --pre-overlay normalize.yaml \
  --post-overlay finalize.yaml \
  -o merged.yaml \
  users-service.yaml orders-service.yaml billing-service.yaml
```

This command:
1. Applies `normalize.yaml` to each input document
2. Joins the documents with semantic deduplication
3. Renames colliding schemas using operation context
4. Uses the most-specific operation policy for context selection
5. Applies `finalize.yaml` to the merged result

For complete API documentation and programmatic usage, see the sections above or the [Go package documentation](https://pkg.go.dev/github.com/erraggy/oastools/joiner).

---

## Learn More

For additional examples and complete API documentation:

- 📦 [API Reference on pkg.go.dev](https://pkg.go.dev/github.com/erraggy/oastools/joiner) - Complete API documentation with all examples
- 🔗 [Basic example](https://pkg.go.dev/github.com/erraggy/oastools/joiner#example-package) - Merge two OpenAPI specifications
- ⚙️ [Custom strategies example](https://pkg.go.dev/github.com/erraggy/oastools/joiner#example-package-CustomStrategies) - Configure collision handling per component type
- 🧹 [Semantic deduplication example](https://pkg.go.dev/github.com/erraggy/oastools/joiner#example-package-SemanticDeduplication) - Consolidate identical schemas
