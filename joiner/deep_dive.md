<a id="top"></a>

# Joiner Package Deep Dive

!!! tip "Try it Online"
    No installation required! [Try the joiner in your browser →](https://oastools.robnrob.com/join)

## Table of Contents

- [Overview](#overview)
- [Key Concepts](#key-concepts)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [Configuration Reference](#configuration-reference)
- [JoinResult Structure](#joinresult-structure)
- [Source Map Integration](#source-map-integration)
- [Best Practices](#best-practices)
- [Common Patterns](#common-patterns)

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

## Learn More

For additional examples and complete API documentation:

- 📦 [API Reference on pkg.go.dev](https://pkg.go.dev/github.com/erraggy/oastools/joiner) - Complete API documentation with all examples
- 🔗 [Basic example](https://pkg.go.dev/github.com/erraggy/oastools/joiner#example-package) - Merge two OpenAPI specifications
- ⚙️ [Custom strategies example](https://pkg.go.dev/github.com/erraggy/oastools/joiner#example-package-CustomStrategies) - Configure collision handling per component type
- 🧹 [Semantic deduplication example](https://pkg.go.dev/github.com/erraggy/oastools/joiner#example-package-SemanticDeduplication) - Consolidate identical schemas
