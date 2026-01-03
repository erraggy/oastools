# Walker Package Deep Dive

The `walker` package provides a document traversal API for OpenAPI specifications, enabling single-pass traversal with typed handlers for analysis and mutation.

## Overview

The walker visits all nodes in an OpenAPI document in a consistent order, calling registered handlers for each node type. This is useful for:

- **Analysis**: Collecting statistics, finding patterns, validating custom rules
- **Transformation**: Adding vendor extensions, modifying descriptions, normalizing formats
- **Code Generation**: Gathering type information across the document

## Core Concepts

### Action-Based Flow Control

Handlers return an `Action` to control traversal:

```go
type Action int

const (
    Continue     Action = iota  // Continue to children and siblings
    SkipChildren                // Skip children, continue to siblings
    Stop                        // Stop walking entirely
)
```

This provides cleaner flow control than error-based approaches.

#### Continue

`Continue` tells the walker to proceed normally—descend into children, then continue to siblings. This is the default behavior for most handlers.

```go
// Count all schemas in the document
var schemaCount int
walker.Walk(result,
    walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
        schemaCount++
        return walker.Continue  // Visit nested schemas too
    }),
)
```

Use `Continue` when you want to:
- **Visit every matching node** in the document
- **Collect comprehensive information** (all operations, all schemas, etc.)
- **Apply transformations uniformly** across the entire document

#### SkipChildren

`SkipChildren` tells the walker to skip the current node's descendants but continue to siblings. The walker moves horizontally rather than descending.

```go
// Find schemas but don't descend into their nested properties
var topLevelSchemas []string
walker.Walk(result,
    walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
        // Only capture component-level schemas, not nested ones
        if strings.HasPrefix(path, "$.components.schemas") &&
           !strings.Contains(path, ".properties") {
            topLevelSchemas = append(topLevelSchemas, path)
        }
        return walker.SkipChildren  // Don't walk into properties/items/etc.
    }),
)
```

Common use cases for `SkipChildren`:

**1. Skipping internal/private paths:**
```go
walker.Walk(result,
    walker.WithPathHandler(func(pathTemplate string, pi *parser.PathItem, path string) walker.Action {
        if strings.HasPrefix(pathTemplate, "/internal") ||
           strings.HasPrefix(pathTemplate, "/_") {
            return walker.SkipChildren  // Don't process internal endpoints
        }
        return walker.Continue
    }),
)
```

**2. Processing only top-level schemas (ignoring nested):**
```go
walker.Walk(result,
    walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
        // Process this schema...
        processSchema(schema)
        // But don't recurse into properties, items, allOf, etc.
        return walker.SkipChildren
    }),
)
```

**3. Conditional depth limiting:**
```go
walker.Walk(result,
    walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
        depth := strings.Count(path, ".properties")
        if depth >= 3 {
            return walker.SkipChildren  // Stop at 3 levels of nesting
        }
        return walker.Continue
    }),
)
```

**4. Skipping deprecated operations:**
```go
walker.Walk(result,
    walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
        if op.Deprecated {
            return walker.SkipChildren  // Skip parameters, responses of deprecated ops
        }
        return walker.Continue
    }),
)
```

#### Stop

`Stop` immediately terminates the entire walk. No more nodes are visited—the walker returns immediately.

```go
// Find the first schema with a specific title
var targetSchema *parser.Schema
walker.Walk(result,
    walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
        if schema.Title == "UserProfile" {
            targetSchema = schema
            return walker.Stop  // Found it, no need to continue
        }
        return walker.Continue
    }),
)
```

Common use cases for `Stop`:

**1. Search with early termination:**
```go
// Check if any operation uses a specific security scheme
var usesOAuth bool
walker.Walk(result,
    walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
        for _, req := range op.Security {
            if _, ok := req["oauth2"]; ok {
                usesOAuth = true
                return walker.Stop  // Found one, that's enough
            }
        }
        return walker.Continue
    }),
)
```

**2. Validation with fail-fast:**
```go
// Stop on first validation error
var firstError error
walker.Walk(result,
    walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
        if err := validateCustomRule(schema); err != nil {
            firstError = fmt.Errorf("%s: %w", path, err)
            return walker.Stop  // Fail fast
        }
        return walker.Continue
    }),
)
```

**3. Finding a specific node by path:**
```go
// Find operation at a specific path and method
var targetOp *parser.Operation
walker.Walk(result,
    walker.WithOperationHandler(func(method string, op *parser.Operation, jsonPath string) walker.Action {
        if jsonPath == "$.paths['/users/{id}'].get" {
            targetOp = op
            return walker.Stop
        }
        return walker.Continue
    }),
)
```

**4. Resource limits:**
```go
// Process at most N schemas
const maxSchemas = 1000
var processed int
walker.Walk(result,
    walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
        processed++
        if processed >= maxSchemas {
            return walker.Stop  // Resource limit reached
        }
        // Process schema...
        return walker.Continue
    }),
)
```

#### Combining Actions Across Handlers

Different handlers can return different actions to create sophisticated traversal patterns:

```go
// Analyze public APIs only, stop if we find a critical issue
var criticalIssue error
walker.Walk(result,
    walker.WithPathHandler(func(pathTemplate string, pi *parser.PathItem, path string) walker.Action {
        if strings.HasPrefix(pathTemplate, "/internal") {
            return walker.SkipChildren  // Skip internal paths
        }
        return walker.Continue
    }),
    walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
        if op.Deprecated {
            return walker.SkipChildren  // Skip deprecated operations
        }
        return walker.Continue
    }),
    walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
        if hasCriticalVulnerability(schema) {
            criticalIssue = fmt.Errorf("critical issue at %s", path)
            return walker.Stop  // Halt everything
        }
        return walker.Continue
    }),
)
```

### Handler Types

Each OAS node type has a corresponding handler type:

| Handler | Called For | OAS Version |
|---------|-----------|-------------|
| `DocumentHandler` | Root document (any type) | All |
| `OAS2DocumentHandler` | OAS 2.0 documents only | 2.0 only |
| `OAS3DocumentHandler` | OAS 3.x documents only | 3.x only |
| `InfoHandler` | API metadata | All |
| `ServerHandler` | Server definitions | 3.x only |
| `TagHandler` | Tag definitions | All |
| `PathHandler` | Path entries | All |
| `PathItemHandler` | Path items | All |
| `OperationHandler` | Operations | All |
| `ParameterHandler` | Parameters | All |
| `RequestBodyHandler` | Request bodies | 3.x only |
| `ResponseHandler` | Responses | All |
| `SchemaHandler` | Schemas (including nested) | All |
| `SecuritySchemeHandler` | Security schemes | All |
| `HeaderHandler` | Headers | All |
| `MediaTypeHandler` | Media types | 3.x only |
| `LinkHandler` | Links | 3.x only |
| `CallbackHandler` | Callbacks | 3.x only |
| `ExampleHandler` | Examples | All |
| `ExternalDocsHandler` | External docs | All |
| `SchemaSkippedHandler` | Skipped schemas (depth/cycle) | All |

### JSON Path Context

Each handler receives a JSON path string indicating the node's location:

```
$                                    # Document root
$.info                               # Info object
$.paths['/pets/{petId}']             # Path entry
$.paths['/pets'].get                 # Operation
$.paths['/pets'].get.parameters[0]   # Parameter
$.components.schemas['Pet']          # Schema
$.components.schemas['Pet'].properties['name']  # Nested schema
```

## API Reference

### Choosing an API: Walk vs WalkWithOptions

The walker package provides two complementary APIs:

| API | Best For | Input | Error Handling |
|-----|----------|-------|----------------|
| `Walk` | Pre-parsed documents | `*parser.ParseResult` | Handler registration never fails |
| `WalkWithOptions` | File paths or parsed documents | Via options | Option functions can return errors |

**Use `Walk` when:**
- You already have a `ParseResult` from parsing
- You're walking multiple documents with the same handlers
- You want simpler handler registration (no error checking)

```go
// Walk: Direct and simple
result, _ := parser.New().Parse("openapi.yaml")
walker.Walk(result,
    walker.WithSchemaHandler(handler),
    walker.WithMaxDepth(50),
)
```

**Use `WalkWithOptions` when:**
- You want to parse and walk in a single call
- You need error handling for configuration (e.g., invalid depth)
- You prefer the `On*` naming convention for handlers

```go
// WalkWithOptions: Parse and walk in one call
err := walker.WalkWithOptions(
    walker.WithFilePath("openapi.yaml"),
    walker.OnSchema(handler),
    walker.WithMaxSchemaDepth(50),  // Returns error if invalid
)
```

### Primary Functions

```go
// Walk traverses a parsed document with registered handlers
func Walk(result *parser.ParseResult, opts ...Option) error

// WalkWithOptions provides functional options for input and configuration
func WalkWithOptions(opts ...WalkInputOption) error
```

### Walk Options

```go
// Handler registration
WithDocumentHandler(fn DocumentHandler)
WithInfoHandler(fn InfoHandler)
WithServerHandler(fn ServerHandler)
WithTagHandler(fn TagHandler)
WithPathHandler(fn PathHandler)
WithPathItemHandler(fn PathItemHandler)
WithOperationHandler(fn OperationHandler)
WithParameterHandler(fn ParameterHandler)
WithRequestBodyHandler(fn RequestBodyHandler)
WithResponseHandler(fn ResponseHandler)
WithSchemaHandler(fn SchemaHandler)
WithSecuritySchemeHandler(fn SecuritySchemeHandler)
WithHeaderHandler(fn HeaderHandler)
WithMediaTypeHandler(fn MediaTypeHandler)
WithLinkHandler(fn LinkHandler)
WithCallbackHandler(fn CallbackHandler)
WithExampleHandler(fn ExampleHandler)
WithExternalDocsHandler(fn ExternalDocsHandler)
WithSchemaSkippedHandler(fn SchemaSkippedHandler)

// Configuration
WithMaxDepth(depth int)  // Default: 100
```

### WalkWithOptions Input Options

```go
WithFilePath(path string)           // Parse and walk a file
WithParsed(result *parser.ParseResult)  // Walk pre-parsed document
WithMaxSchemaDepth(depth int)       // Returns error if not positive

// On* variants for handler registration
OnDocument(fn DocumentHandler)
OnInfo(fn InfoHandler)
OnSchemaSkipped(fn SchemaSkippedHandler)
// ... etc
```

## Walk Order

### OAS 3.x Documents

1. Document root
2. Info
3. ExternalDocs (root level)
4. Servers
5. Paths → PathItems → Operations → Parameters, RequestBody, Responses, Callbacks
6. Webhooks (OAS 3.1+)
7. Components (schemas, responses, parameters, requestBodies, headers, securitySchemes, links, callbacks, examples, pathItems)
8. Tags

### OAS 2.0 Documents

1. Document root
2. Info
3. ExternalDocs (root level)
4. Paths → PathItems → Operations → Parameters, Responses
5. Definitions (schemas)
6. Parameters (reusable)
7. Responses (reusable)
8. SecurityDefinitions
9. Tags

## Schema Walking

The walker recursively visits all nested schemas:

- `properties`, `patternProperties`, `dependentSchemas`, `$defs` (maps)
- `allOf`, `anyOf`, `oneOf`, `prefixItems` (slices)
- `items`, `additionalProperties`, `additionalItems`, `unevaluatedItems`, `unevaluatedProperties` (polymorphic)
- `not`, `contains`, `propertyNames`, `contentSchema`, `if`, `then`, `else` (single)

### Cycle Detection

The walker uses pointer-based cycle detection to prevent infinite loops in circular schema references. Visited schemas are tracked and skipped on subsequent encounters.

```go
// Circular reference example
schema := &parser.Schema{Type: "object"}
schema.Properties = map[string]*parser.Schema{
    "self": schema,  // Points back to itself
}
// The walker will visit 'schema' once, then skip 'self'
// since it's already been visited
```

### Depth Limiting

Use `WithMaxDepth(n)` to limit schema recursion depth (default: 100).

```go
// Limit to 10 levels of nesting
walker.Walk(result,
    walker.WithSchemaHandler(handler),
    walker.WithMaxDepth(10),
)
```

**Behavior:**
- The depth counter starts at 0 for component/definition schemas
- Each nested schema (properties, items, allOf, etc.) increments the depth
- When depth reaches the limit, nested schemas are skipped
- The handler is not called for schemas beyond the depth limit

### Schema Skipped Callbacks

Use `WithSchemaSkippedHandler` to receive notifications when schemas are skipped due to depth limits or cycle detection:

```go
walker.Walk(result,
    walker.WithMaxDepth(10),
    walker.WithSchemaSkippedHandler(func(reason string, schema *parser.Schema, path string) {
        switch reason {
        case "depth":
            fmt.Printf("Skipped due to depth limit: %s\n", path)
        case "cycle":
            fmt.Printf("Skipped due to circular reference: %s\n", path)
        }
    }),
)
```

**Reason values:**
- `"depth"` - Schema exceeded the configured `maxDepth` limit
- `"cycle"` - Schema was already visited (circular reference detected)

This is useful for:
- **Debugging**: Understanding why certain schemas weren't processed
- **Logging**: Recording when circular references are encountered
- **Validation**: Detecting overly deep or circular schema structures

For `WalkWithOptions`, use `OnSchemaSkipped`:

```go
walker.WalkWithOptions(
    walker.WithFilePath("openapi.yaml"),
    walker.WithMaxSchemaDepth(10),
    walker.OnSchemaSkipped(func(reason string, schema *parser.Schema, path string) {
        log.Printf("Schema skipped (%s): %s", reason, path)
    }),
)
```

## Usage Patterns

### Mutation

Handlers receive pointers to the actual document nodes, allowing in-place modification:

```go
walker.Walk(result,
    walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
        // Add vendor extension to all schemas
        if schema.Extra == nil {
            schema.Extra = make(map[string]any)
        }
        schema.Extra["x-visited"] = true
        return walker.Continue
    }),
)
```

### Version-Specific Handling

For type-safe version-specific handling, use the typed document handlers:

```go
walker.Walk(result,
    walker.WithOAS2DocumentHandler(func(doc *parser.OAS2Document, path string) walker.Action {
        // Called only for OAS 2.0 documents - doc is already typed
        fmt.Printf("OAS 2.0: %s (host: %s)\n", doc.Info.Title, doc.Host)
        return walker.Continue
    }),
    walker.WithOAS3DocumentHandler(func(doc *parser.OAS3Document, path string) walker.Action {
        // Called only for OAS 3.x documents - doc is already typed
        fmt.Printf("OAS 3.x: %s (servers: %d)\n", doc.Info.Title, len(doc.Servers))
        return walker.Continue
    }),
)
```

**Handler Order:** When both typed and generic handlers are registered:
1. The typed handler (`OAS2DocumentHandler` or `OAS3DocumentHandler`) is called first
2. If it returns `Continue` or `SkipChildren`, the generic `DocumentHandler` is called
3. If it returns `Stop`, the generic handler is skipped and the walk stops

Alternatively, use a type switch with the generic handler:

```go
walker.Walk(result,
    walker.WithDocumentHandler(func(doc any, path string) walker.Action {
        switch d := doc.(type) {
        case *parser.OAS2Document:
            fmt.Printf("OAS 2.0: %s\n", d.Info.Title)
        case *parser.OAS3Document:
            fmt.Printf("OAS 3.x: %s\n", d.Info.Title)
        }
        return walker.Continue
    }),
)
```

### Multiple Handlers

Register multiple handlers to build up analysis in a single pass:

```go
var (
    pathCount      int
    operationCount int
    schemaCount    int
)

walker.Walk(result,
    walker.WithPathHandler(func(pathTemplate string, pi *parser.PathItem, path string) walker.Action {
        pathCount++
        return walker.Continue
    }),
    walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
        operationCount++
        return walker.Continue
    }),
    walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
        schemaCount++
        return walker.Continue
    }),
)
```

### Using JSON Paths

The path parameter enables location-aware processing:

```go
walker.Walk(result,
    walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
        // Different handling based on location
        switch {
        case strings.HasPrefix(path, "$.components.schemas"):
            // Component schema
        case strings.Contains(path, ".requestBody"):
            // Request body schema
        case strings.Contains(path, ".responses"):
            // Response schema
        }
        return walker.Continue
    }),
)
```

### WalkWithOptions API

For parsing and walking in one call:

```go
err := walker.WalkWithOptions(
    walker.WithFilePath("openapi.yaml"),
    walker.OnSchema(func(schema *parser.Schema, path string) walker.Action {
        fmt.Println(path)
        return walker.Continue
    }),
)
```

## Performance

- **Parse-Once Pattern**: Pass pre-parsed `ParseResult` instead of file paths
- **Minimal Allocations**: Handler function calls have minimal overhead
- **Deterministic Order**: Map keys are sorted for consistent traversal
- **Early Exit**: Use `Stop` to terminate as soon as you find what you need

## Thread Safety

⚠️ **The Walker is NOT thread-safe.** Each walk maintains internal state (visited schemas, stopped flag) that is not protected by locks.

**Safe patterns:**

```go
// ✅ Sequential walks (same or different documents)
walker.Walk(result1, opts...)
walker.Walk(result2, opts...)

// ✅ Parallel walks with separate documents
var wg sync.WaitGroup
for _, doc := range documents {
    wg.Add(1)
    go func(d *parser.ParseResult) {
        defer wg.Done()
        walker.Walk(d, opts...)  // Each goroutine has its own walk state
    }(doc)
}
wg.Wait()
```

**Unsafe patterns:**

```go
// ❌ Shared mutable state in handlers without synchronization
var count int  // Race condition!
walker.Walk(result,
    walker.WithSchemaHandler(func(s *parser.Schema, path string) walker.Action {
        count++  // Not thread-safe
        return walker.Continue
    }),
)

// ✅ Use atomic operations or mutexes for shared state
var count atomic.Int64
walker.Walk(result,
    walker.WithSchemaHandler(func(s *parser.Schema, path string) walker.Action {
        count.Add(1)  // Thread-safe
        return walker.Continue
    }),
)
```

**Document mutation:** If handlers modify the document, ensure the document is not shared across concurrent walks.

## OAS 3.2 Support

The walker supports OAS 3.2 features:

- `PathItem.Query` operation (QUERY method)
- `PathItem.AdditionalOperations` for custom methods
- `Components.MediaTypes` for reusable media types
