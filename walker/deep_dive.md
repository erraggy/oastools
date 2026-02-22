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
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
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
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        // Only capture component-level schemas, not nested ones
        if wc.IsComponent && wc.Name != "" {
            topLevelSchemas = append(topLevelSchemas, wc.JSONPath)
        }
        return walker.SkipChildren  // Don't walk into properties/items/etc.
    }),
)
```

Common use cases for `SkipChildren`:

**1. Skipping internal/private paths:**

```go
walker.Walk(result,
    walker.WithPathHandler(func(wc *walker.WalkContext, pi *parser.PathItem) walker.Action {
        if strings.HasPrefix(wc.PathTemplate, "/internal") ||
           strings.HasPrefix(wc.PathTemplate, "/_") {
            return walker.SkipChildren  // Don't process internal endpoints
        }
        return walker.Continue
    }),
)
```

**2. Processing only top-level schemas (ignoring nested):**

```go
walker.Walk(result,
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
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
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        depth := strings.Count(wc.JSONPath, ".properties")
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
    walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
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
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
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
    walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
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
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        if err := validateCustomRule(schema); err != nil {
            firstError = fmt.Errorf("%s: %w", wc.JSONPath, err)
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
    walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
        if wc.JSONPath == "$.paths['/users/{id}'].get" {
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
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
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
    walker.WithPathHandler(func(wc *walker.WalkContext, pi *parser.PathItem) walker.Action {
        if strings.HasPrefix(wc.PathTemplate, "/internal") {
            return walker.SkipChildren  // Skip internal paths
        }
        return walker.Continue
    }),
    walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
        if op.Deprecated {
            return walker.SkipChildren  // Skip deprecated operations
        }
        return walker.Continue
    }),
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        if hasCriticalVulnerability(schema) {
            criticalIssue = fmt.Errorf("critical issue at %s", wc.JSONPath)
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

Each handler type has a corresponding registration option. Register handlers using these `Option` functions:

| Option | Called For | OAS Version |
|--------|-----------|-------------|
| `WithDocumentHandler(fn)` | Root document (any type) | All |
| `WithOAS2DocumentHandler(fn)` | OAS 2.0 documents only | 2.0 only |
| `WithOAS3DocumentHandler(fn)` | OAS 3.x documents only | 3.x only |
| `WithInfoHandler(fn)` | API metadata | All |
| `WithServerHandler(fn)` | Server definitions | 3.x only |
| `WithTagHandler(fn)` | Tag definitions | All |
| `WithPathHandler(fn)` | Path entries | All |
| `WithPathItemHandler(fn)` | Path items | All |
| `WithOperationHandler(fn)` | Operations | All |
| `WithParameterHandler(fn)` | Parameters | All |
| `WithRequestBodyHandler(fn)` | Request bodies | 3.x only |
| `WithResponseHandler(fn)` | Responses | All |
| `WithSchemaHandler(fn)` | Schemas (including nested) | All |
| `WithSecuritySchemeHandler(fn)` | Security schemes | All |
| `WithHeaderHandler(fn)` | Headers | All |
| `WithMediaTypeHandler(fn)` | Media types | 3.x only |
| `WithLinkHandler(fn)` | Links | 3.x only |
| `WithCallbackHandler(fn)` | Callbacks | 3.x only |
| `WithExampleHandler(fn)` | Examples | All |
| `WithExternalDocsHandler(fn)` | External docs | All |
| `WithSchemaSkippedHandler(fn)` | Skipped schemas (depth/cycle) | All |

### WalkContext

Every handler receives a `*WalkContext` as its first parameter, providing contextual information about the current node.

> **Important: WalkContext Pooling**
>
> WalkContext instances are reused via `sync.Pool` for performance. Handlers **must not** retain references to WalkContext after returning. If you need to preserve context information, copy the needed fields:
>
> ```go
> // Wrong - retaining WalkContext reference
> var saved []*WalkContext
> WithSchemaHandler(func(wc *WalkContext, s *parser.Schema) Action {
>     saved = append(saved, wc) // Don't do this!
>     return Continue
> })
>
> // Correct - copy needed fields
> type Info struct { JSONPath, Name string }
> var saved []Info
> WithSchemaHandler(func(wc *WalkContext, s *parser.Schema) Action {
>     saved = append(saved, Info{wc.JSONPath, wc.Name})
>     return Continue
> })
> ```

| Field | Description |
|-------|-------------|
| `JSONPath` | Full JSON path to the node (always populated) |
| `PathTemplate` | URL path template when in $.paths scope |
| `Method` | HTTP method when in operation scope (e.g., "get", "post") |
| `StatusCode` | Status code when in response scope (e.g., "200", "default") |
| `Name` | Map key for named items (headers, schemas, etc.) |
| `IsComponent` | True when in components/definitions section |

#### JSON Path Examples

```
$                                    # Document root
$.info                               # Info object
$.paths['/pets/{petId}']             # Path entry
$.paths['/pets'].get                 # Operation
$.paths['/pets'].get.parameters[0]   # Parameter
$.components.schemas['Pet']          # Schema
$.components.schemas['Pet'].properties['name']  # Nested schema
```

#### Accessing Context

```go
walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
    if wc.IsComponent {
        fmt.Printf("Component schema: %s\n", wc.Name)
    } else if wc.InOperationScope() {
        fmt.Printf("Inline schema in %s %s operation\n", wc.Method, wc.PathTemplate)
    }
    return walker.Continue
})
```

#### Scope Helper Methods

The `WalkContext` provides helper methods to check the current scope:

```go
wc.InPathsScope()     // true when PathTemplate is set
wc.InOperationScope() // true when Method is set
wc.InResponseScope()  // true when StatusCode is set
```

#### Cancellation Support

Pass a `context.Context` for cancellation and timeout support:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

walker.Walk(result,
    walker.WithContext(ctx),
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        // Check if cancelled
        if wc.Context().Err() != nil {
            return walker.Stop
        }
        return walker.Continue
    }),
)
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
    walker.WithMaxSchemaDepth(50),
)
```

**Use `WalkWithOptions` when:**

- You want to parse and walk in a single call
- You need error handling for configuration (e.g., invalid depth)

```go
// WalkWithOptions: Parse and walk in one call
err := walker.WalkWithOptions(
    walker.WithFilePath("openapi.yaml"),
    walker.WithSchemaHandler(handler),
    walker.WithMaxSchemaDepth(50),  // Uses default (100) if not positive
)
```

### Primary Functions

```go
// Walk traverses a parsed document with registered handlers
func Walk(result *parser.ParseResult, opts ...Option) error

// WalkWithOptions provides functional options for input and configuration
func WalkWithOptions(opts ...Option) error
```

### Walk Options

```go
// Pre-visit handler registration
WithDocumentHandler(fn DocumentHandler)
WithOAS2DocumentHandler(fn OAS2DocumentHandler)
WithOAS3DocumentHandler(fn OAS3DocumentHandler)
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

// Post-visit handler registration
WithSchemaPostHandler(fn SchemaPostHandler)
WithOperationPostHandler(fn OperationPostHandler)
WithPathItemPostHandler(fn PathItemPostHandler)
WithResponsePostHandler(fn ResponsePostHandler)
WithRequestBodyPostHandler(fn RequestBodyPostHandler)
WithCallbackPostHandler(fn CallbackPostHandler)
WithOAS2DocumentPostHandler(fn OAS2DocumentPostHandler)
WithOAS3DocumentPostHandler(fn OAS3DocumentPostHandler)

// Reference and parent tracking
WithRefHandler(fn RefHandler)
WithRefTracking()
WithMapRefTracking()
WithParentTracking()

// Configuration
WithMaxSchemaDepth(depth int)  // Limit schema recursion depth (default: 100)
WithMaxDepth(depth int)        // Deprecated: use WithMaxSchemaDepth instead
WithContext(ctx context.Context)
```

### WalkWithOptions Input Options

```go
WithFilePath(path string)               // Parse and walk a file
WithParsed(result *parser.ParseResult)  // Walk pre-parsed document
WithMaxSchemaDepth(depth int)           // Silently ignored if not positive (uses default 100)
WithContext(ctx context.Context)        // Context for cancellation
```

| Option | Description |
|--------|-------------|
| `WithFilePath(path)` | Parse and walk a file |
| `WithParsed(result)` | Walk a pre-parsed `*parser.ParseResult` |
| `WithMaxSchemaDepth(depth)` | Maximum depth for recursive schema traversal (default: 100) |
| `WithContext(ctx)` | Set context for cancellation and deadline propagation |

All handler options (e.g., `WithSchemaHandler`, `WithOperationHandler`) work directly with both `Walk` and `WalkWithOptions`.

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

Use `WithMaxSchemaDepth(n)` to limit schema recursion depth (default: 100). The older `WithMaxDepth(n)` is deprecated but still works.

```go
// Limit to 10 levels of nesting
walker.Walk(result,
    walker.WithSchemaHandler(handler),
    walker.WithMaxSchemaDepth(10),
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
    walker.WithMaxSchemaDepth(10),
    walker.WithSchemaSkippedHandler(func(wc *walker.WalkContext, reason string, schema *parser.Schema) {
        switch reason {
        case "depth":
            fmt.Printf("Skipped due to depth limit: %s\n", wc.JSONPath)
        case "cycle":
            fmt.Printf("Skipped due to circular reference: %s\n", wc.JSONPath)
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

With `WalkWithOptions`:

```go
walker.WalkWithOptions(
    walker.WithFilePath("openapi.yaml"),
    walker.WithMaxSchemaDepth(10),
    walker.WithSchemaSkippedHandler(func(wc *walker.WalkContext, reason string, schema *parser.Schema) {
        log.Printf("Schema skipped (%s): %s", reason, wc.JSONPath)
    }),
)
```

## Usage Patterns

### Mutation

Handlers receive pointers to the actual document nodes, allowing in-place modification:

```go
walker.Walk(result,
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
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
    walker.WithOAS2DocumentHandler(func(wc *walker.WalkContext, doc *parser.OAS2Document) walker.Action {
        // Called only for OAS 2.0 documents - doc is already typed
        fmt.Printf("OAS 2.0: %s (host: %s)\n", doc.Info.Title, doc.Host)
        return walker.Continue
    }),
    walker.WithOAS3DocumentHandler(func(wc *walker.WalkContext, doc *parser.OAS3Document) walker.Action {
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
    walker.WithDocumentHandler(func(wc *walker.WalkContext, doc any) walker.Action {
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
    walker.WithPathHandler(func(wc *walker.WalkContext, pi *parser.PathItem) walker.Action {
        pathCount++
        return walker.Continue
    }),
    walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
        operationCount++
        return walker.Continue
    }),
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        schemaCount++
        return walker.Continue
    }),
)
```

### Using WalkContext for Location-Aware Processing

The `WalkContext` enables location-aware processing using both structured fields and the JSON path:

```go
walker.Walk(result,
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        // Use structured fields for cleaner logic
        if wc.IsComponent {
            // Component schema - wc.Name has the schema name
            fmt.Printf("Component: %s\n", wc.Name)
        } else if wc.InOperationScope() {
            // Inline schema in an operation
            fmt.Printf("Inline in %s %s\n", wc.Method, wc.PathTemplate)
        }

        // Or use JSON path for more specific matching
        switch {
        case strings.HasPrefix(wc.JSONPath, "$.components.schemas"):
            // Component schema
        case strings.Contains(wc.JSONPath, ".requestBody"):
            // Request body schema
        case strings.Contains(wc.JSONPath, ".responses"):
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
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        fmt.Println(wc.JSONPath)
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
    walker.WithSchemaHandler(func(wc *walker.WalkContext, s *parser.Schema) walker.Action {
        count++  // Not thread-safe
        return walker.Continue
    }),
)

// ✅ Use atomic operations or mutexes for shared state
var count atomic.Int64
walker.Walk(result,
    walker.WithSchemaHandler(func(wc *walker.WalkContext, s *parser.Schema) walker.Action {
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

## Reference Tracking

The walker provides optional `$ref` tracking to detect and process references during traversal without separate passes.

### Enabling Reference Tracking

Use `WithRefHandler` to receive callbacks when references are encountered:

```go
walker.Walk(result,
    walker.WithRefHandler(func(wc *walker.WalkContext, ref *walker.RefInfo) walker.Action {
        fmt.Printf("Found ref: %s at %s (type: %s)\n", ref.Ref, ref.SourcePath, ref.NodeType)
        return walker.Continue
    }),
)
```

### RefInfo Structure

The `RefInfo` struct contains:

| Field | Description |
|-------|-------------|
| `Ref` | The $ref value (e.g., `#/components/schemas/User`) |
| `SourcePath` | JSON path where the ref was encountered |
| `NodeType` | Type of node containing the ref |

### Supported Node Types

References are tracked in:

| Node Type | Description |
|-----------|-------------|
| `schema` | Schema references |
| `parameter` | Parameter references |
| `response` | Response references |
| `requestBody` | Request body references |
| `header` | Header references |
| `pathItem` | Path item references |
| `link` | Link references |
| `example` | Example references |
| `securityScheme` | Security scheme references |

### Use Cases

**Collecting all references:**

```go
var refs []string
walker.Walk(result,
    walker.WithRefHandler(func(wc *walker.WalkContext, ref *walker.RefInfo) walker.Action {
        refs = append(refs, ref.Ref)
        return walker.Continue
    }),
)
```

**Finding broken references:**

```go
walker.Walk(result,
    walker.WithRefHandler(func(wc *walker.WalkContext, ref *walker.RefInfo) walker.Action {
        if !isValidRef(ref.Ref) {
            fmt.Printf("Broken ref at %s: %s\n", ref.SourcePath, ref.Ref)
        }
        return walker.Continue
    }),
)
```

**Stop on first external reference:**

```go
var hasExternal bool
walker.Walk(result,
    walker.WithRefHandler(func(wc *walker.WalkContext, ref *walker.RefInfo) walker.Action {
        if strings.HasPrefix(ref.Ref, "http") {
            hasExternal = true
            return walker.Stop
        }
        return walker.Continue
    }),
)
```

### WithRefTracking Option

`WithRefTracking()` enables internal reference tracking for statistics and debugging purposes, but it does **not** populate `CurrentRef` in node handlers. The `CurrentRef` field is only set when you register a `RefHandler` via `WithRefHandler()`.

To check for references in node handlers, examine the node's `Ref` field directly:

```go
walker.Walk(result,
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        if schema.Ref != "" {
            // This schema is a $ref - check schema.Ref directly
            fmt.Printf("Schema ref at %s: %s\n", wc.JSONPath, schema.Ref)
        }
        return walker.Continue
    }),
)
```

If you need the dedicated `RefInfo` structure with `NodeType` classification, use `WithRefHandler()` instead:

```go
walker.Walk(result,
    walker.WithRefHandler(func(wc *walker.WalkContext, ref *walker.RefInfo) walker.Action {
        // RefInfo provides: Ref, SourcePath, NodeType
        fmt.Printf("Ref %s at %s (type: %s)\n", ref.Ref, ref.SourcePath, ref.NodeType)
        return walker.Continue
    }),
)
```

### Map Reference Tracking

Some polymorphic schema fields (`Items`, `AdditionalItems`, `AdditionalProperties`, `UnevaluatedItems`, `UnevaluatedProperties`) can contain either `*parser.Schema` or `map[string]any`. When a document is parsed with certain configurations or when schemas aren't fully resolved, these fields may contain raw maps with `$ref` values that the standard ref tracking wouldn't detect.

Use `WithMapRefTracking()` to enable detection of `$ref` values stored in these map structures:

```go
walker.Walk(result,
    walker.WithMapRefTracking(),
    walker.WithRefHandler(func(wc *walker.WalkContext, ref *walker.RefInfo) walker.Action {
        fmt.Printf("Found ref: %s at %s\n", ref.Ref, ref.SourcePath)
        return walker.Continue
    }),
)
```

**Key behaviors:**

- `WithMapRefTracking()` implicitly enables standard ref tracking
- The walker checks for `$ref` keys in `map[string]any` values in polymorphic fields
- Empty strings and non-string `$ref` values are ignored
- Map-stored refs receive `RefNodeSchema` as their node type

**Affected fields:**

| Field | Description |
|-------|-------------|
| `Items` | Array items schema |
| `AdditionalItems` | Additional array items schema |
| `UnevaluatedItems` | Unevaluated array items schema (OAS 3.1+) |
| `AdditionalProperties` | Additional object properties schema |
| `UnevaluatedProperties` | Unevaluated object properties schema (OAS 3.1+) |

**Example with mixed schemas:**

```go
// Schema with both *Schema and map refs
doc := &parser.OAS3Document{
    Components: &parser.Components{
        Schemas: map[string]*parser.Schema{
            "Container": {
                Type: "object",
                Properties: map[string]*parser.Schema{
                    "items": {
                        Type: "array",
                        Items: map[string]any{
                            "$ref": "#/components/schemas/Item",
                        },
                    },
                    "regular": {Ref: "#/components/schemas/Regular"},
                },
            },
        },
    },
}

// Wrap in ParseResult for walking
result := &parser.ParseResult{Document: doc, OASVersion: parser.OASVersion310}

// Both refs will be tracked with WithMapRefTracking()
walker.Walk(result,
    walker.WithMapRefTracking(),
    walker.WithRefHandler(func(wc *walker.WalkContext, ref *walker.RefInfo) walker.Action {
        // Called for both the map-stored ref and the regular ref
        return walker.Continue
    }),
)
```

**When to use:**

- Parsing documents where polymorphic fields weren't fully resolved
- Working with documents from external sources that use map representations
- Comprehensive reference analysis that needs to catch all `$ref` values

**Performance note:** Map ref tracking adds a small overhead for type assertions on polymorphic fields. Only enable when needed.

## Examples

The `examples/walker/` directory contains runnable examples demonstrating walker patterns:

| Example | Category | Description |
|---------|----------|-------------|
| [api-statistics](../examples/walker/api-statistics/) | Analysis | Collect API statistics in single pass |
| [security-audit](../examples/walker/security-audit/) | Validation | Audit for security issues |
| [vendor-extensions](../examples/walker/vendor-extensions/) | Mutation | Add vendor extensions |
| [public-api-filter](../examples/walker/public-api-filter/) | Filtering | Extract public API only |
| [api-documentation](../examples/walker/api-documentation/) | Reporting | Generate Markdown docs |
| [reference-collector](../examples/walker/reference-collector/) | Integration | Analyze schema references |

Each example includes a README with detailed explanations and expected output.

## Built-in Collectors

The walker package provides convenience functions for common collection patterns, reducing boilerplate when you need to gather spec elements.

Five collectors are available: `CollectSchemas`, `CollectOperations`, `CollectParameters`, `CollectResponses`, and `CollectSecuritySchemes`.

### When to Use Collectors vs Custom Handlers

**Use built-in collectors when:**

- You need all elements of one type in one pass
- You want ready-made lookup maps (by name, path, method, tag, status code, location)
- The standard collection fields meet your needs

**Use custom handlers when:**

- You need to filter during collection (e.g., only deprecated operations)
- You want to collect multiple node types in a single pass
- You need custom organization or aggregation logic

### SchemaCollector

`CollectSchemas` walks a document and collects all schemas:

```go
collector, err := walker.CollectSchemas(result)
if err != nil {
    return err
}

// All schemas in traversal order
for _, info := range collector.All {
    fmt.Printf("%s: %s\n", info.JSONPath, info.Schema.Type)
}

// Component schemas only
for _, info := range collector.Components {
    fmt.Printf("Component %s at %s\n", info.Name, info.JSONPath)
}

// Inline schemas only (not in components)
for _, info := range collector.Inline {
    fmt.Printf("Inline schema at %s\n", info.JSONPath)
}

// Lookup by JSON path
if schema, ok := collector.ByPath["$.components.schemas['Pet']"]; ok {
    fmt.Printf("Pet: %v\n", schema.Schema.Type)
}

// Lookup by component name
if schema, ok := collector.ByName["Pet"]; ok {
    fmt.Printf("Found Pet schema\n")
}
```

**SchemaInfo fields:**

| Field | Type | Description |
|-------|------|-------------|
| `Schema` | `*parser.Schema` | The collected schema |
| `Name` | `string` | Component name (empty for inline schemas) |
| `JSONPath` | `string` | Full JSON path to the schema |
| `IsComponent` | `bool` | True when in components/definitions section |

### OperationCollector

`CollectOperations` walks a document and collects all operations:

```go
collector, err := walker.CollectOperations(result)
if err != nil {
    return err
}

// All operations in traversal order
for _, info := range collector.All {
    fmt.Printf("%s %s (%s)\n", info.Method, info.PathTemplate, info.Operation.OperationID)
}

// Group by path template
for path, ops := range collector.ByPath {
    fmt.Printf("%s has %d operations\n", path, len(ops))
}

// Group by HTTP method
for method, ops := range collector.ByMethod {
    fmt.Printf("%s: %d operations\n", method, len(ops))
}

// Group by tag
for tag, ops := range collector.ByTag {
    fmt.Printf("Tag '%s': %d operations\n", tag, len(ops))
}
```

**OperationInfo fields:**

| Field | Type | Description |
|-------|------|-------------|
| `Operation` | `*parser.Operation` | The collected operation |
| `PathTemplate` | `string` | URL path template (e.g., "/pets/{petId}") |
| `Method` | `string` | HTTP method (e.g., "get", "post") |
| `JSONPath` | `string` | Full JSON path to the operation |

### ParameterCollector

`CollectParameters` walks a document and collects all parameters:

```go
collector, err := walker.CollectParameters(result)
if err != nil {
    return err
}

// All parameters in traversal order
for _, info := range collector.All {
    fmt.Printf("%s (%s) at %s\n", info.Name, info.In, info.JSONPath)
}

// Group by location
for location, params := range collector.ByLocation {
    fmt.Printf("%s: %d parameters\n", location, len(params))
}

// Group by path template
for path, params := range collector.ByPath {
    fmt.Printf("%s has %d parameters\n", path, len(params))
}
```

**ParameterInfo fields:**

| Field | Type | Description |
|-------|------|-------------|
| `Parameter` | `*parser.Parameter` | The collected parameter |
| `Name` | `string` | Parameter name |
| `In` | `string` | Location: query, header, path, cookie |
| `JSONPath` | `string` | Full JSON path to the parameter |
| `PathTemplate` | `string` | Owning path template |
| `Method` | `string` | Owning operation method (empty if path-level) |
| `IsComponent` | `bool` | True when in components/parameters |

### ResponseCollector

`CollectResponses` walks a document and collects all responses:

```go
collector, err := walker.CollectResponses(result)
if err != nil {
    return err
}

// All responses in traversal order
for _, info := range collector.All {
    fmt.Printf("%s %s -> %s\n", info.Method, info.PathTemplate, info.StatusCode)
}

// Group by status code
for code, responses := range collector.ByStatusCode {
    fmt.Printf("Status %s: %d responses\n", code, len(responses))
}
```

**ResponseInfo fields:**

| Field | Type | Description |
|-------|------|-------------|
| `Response` | `*parser.Response` | The collected response |
| `StatusCode` | `string` | HTTP status code (e.g., "200", "default") |
| `JSONPath` | `string` | Full JSON path to the response |
| `PathTemplate` | `string` | Owning path template |
| `Method` | `string` | Owning operation method |
| `IsComponent` | `bool` | True when in components/responses |

### SecuritySchemeCollector

`CollectSecuritySchemes` walks a document and collects all security schemes:

```go
collector, err := walker.CollectSecuritySchemes(result)
if err != nil {
    return err
}

// All security schemes
for _, info := range collector.All {
    fmt.Printf("%s: type=%s\n", info.Name, info.SecurityScheme.Type)
}

// Lookup by name
if bearer, ok := collector.ByName["bearerAuth"]; ok {
    fmt.Printf("Bearer scheme: %s\n", bearer.SecurityScheme.Scheme)
}
```

**SecuritySchemeInfo fields:**

| Field | Type | Description |
|-------|------|-------------|
| `SecurityScheme` | `*parser.SecurityScheme` | The collected security scheme |
| `Name` | `string` | Security scheme name from components map key |
| `JSONPath` | `string` | Full JSON path to the security scheme |

### Example: API Coverage Report

```go
func generateCoverageReport(result *parser.ParseResult) {
    schemas, _ := walker.CollectSchemas(result)
    ops, _ := walker.CollectOperations(result)

    fmt.Printf("API Coverage Report\n")
    fmt.Printf("==================\n\n")

    fmt.Printf("Schemas: %d total (%d component, %d inline)\n",
        len(schemas.All), len(schemas.Components), len(schemas.Inline))

    fmt.Printf("Operations: %d total\n", len(ops.All))

    fmt.Printf("\nOperations by Method:\n")
    for method, methodOps := range ops.ByMethod {
        fmt.Printf("  %s: %d\n", strings.ToUpper(method), len(methodOps))
    }

    fmt.Printf("\nOperations by Tag:\n")
    for tag, tagOps := range ops.ByTag {
        fmt.Printf("  %s: %d\n", tag, len(tagOps))
    }
}
```

## Parent Tracking

The walker supports optional parent/ancestor tracking, providing type-safe access to ancestor nodes during traversal. This is useful for context-aware processing where you need to know what contains the current node.

### Enabling Parent Tracking

Use `WithParentTracking()` to enable ancestor tracking:

```go
walker.Walk(result,
    walker.WithParentTracking(),
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        // wc.Parent is now populated
        return walker.Continue
    }),
)
```

**Note:** Parent tracking is disabled by default to avoid overhead when not needed. Enable it only when you need ancestor access.

### ParentInfo Structure

The `wc.Parent` field provides a linked list of ancestors:

```go
type ParentInfo struct {
    Node     any         // The parent node (*parser.Schema, *parser.Operation, etc.)
    JSONPath string      // JSON path to this parent
    Parent   *ParentInfo // Grandparent (or nil at root)
}
```

### Helper Methods

Type-safe helper methods make ancestor access convenient:

| Method | Returns | Description |
|--------|---------|-------------|
| `ParentSchema()` | `(*parser.Schema, bool)` | Nearest ancestor schema |
| `ParentOperation()` | `(*parser.Operation, bool)` | Nearest ancestor operation |
| `ParentPathItem()` | `(*parser.PathItem, bool)` | Nearest ancestor path item |
| `ParentResponse()` | `(*parser.Response, bool)` | Nearest ancestor response |
| `ParentRequestBody()` | `(*parser.RequestBody, bool)` | Nearest ancestor request body |
| `Ancestors()` | `[]*ParentInfo` | All ancestors (parent to root) |
| `Depth()` | `int` | Number of ancestors |

### Use Cases

**1. Determining schema context:**

```go
walker.Walk(result,
    walker.WithParentTracking(),
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        // Is this schema in a request or response?
        if _, ok := wc.ParentRequestBody(); ok {
            fmt.Printf("Request schema: %s\n", wc.JSONPath)
        } else if _, ok := wc.ParentResponse(); ok {
            fmt.Printf("Response schema: %s\n", wc.JSONPath)
        }
        return walker.Continue
    }),
)
```

**2. Finding the containing operation:**

```go
walker.Walk(result,
    walker.WithParentTracking(),
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        if op, ok := wc.ParentOperation(); ok {
            fmt.Printf("Schema in %s: %s\n", op.OperationID, wc.JSONPath)
        }
        return walker.Continue
    }),
)
```

**3. Detecting nested schemas:**

```go
walker.Walk(result,
    walker.WithParentTracking(),
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        if parentSchema, ok := wc.ParentSchema(); ok {
            // This schema is nested within another schema
            fmt.Printf("Nested in type: %v\n", parentSchema.Type)
        } else if wc.IsComponent {
            // This is a top-level component schema
            fmt.Printf("Component schema: %s\n", wc.Name)
        }
        return walker.Continue
    }),
)
```

**4. Limiting depth based on ancestor count:**

```go
walker.Walk(result,
    walker.WithParentTracking(),
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        if wc.Depth() > 5 {
            // Skip deeply nested schemas
            return walker.SkipChildren
        }
        return walker.Continue
    }),
)
```

### Performance Considerations

Parent tracking adds overhead:

- ~15-20% increase in traversal time
- Additional allocations for ParentInfo structs

Only enable `WithParentTracking()` when you need ancestor access. If you only need the current node's context (JSONPath, Method, PathTemplate, etc.), the standard `WalkContext` fields are sufficient without parent tracking.

## Post-Visit Hooks

Post-visit handlers fire after a node's children have been processed, enabling bottom-up processing patterns like aggregation and validation.

### Available Post-Visit Handlers

| Option | Called After |
|--------|-------------|
| `WithSchemaPostHandler` | Schema's children (properties, items, allOf, etc.) processed |
| `WithOperationPostHandler` | Operation's children (parameters, requestBody, responses, callbacks) processed |
| `WithPathItemPostHandler` | Path item's children (parameters, operations) processed |
| `WithResponsePostHandler` | Response's children (headers, content, links) processed |
| `WithRequestBodyPostHandler` | Request body's children (content) processed |
| `WithCallbackPostHandler` | Callback's children (path items) processed |
| `WithOAS2DocumentPostHandler` | OAS 2.0 document's children (all nodes) processed |
| `WithOAS3DocumentPostHandler` | OAS 3.x document's children (all nodes) processed |

### When Post Handlers Are Called

Post handlers are called:

- **AFTER** all children are walked
- **BEFORE** the parent is popped (if parent tracking is enabled)
- **NOT** called if the pre-visit handler returned `SkipChildren` or `Stop`

### Execution Order

For nested schemas:

```
Pre-visit A (parent)
  Pre-visit B (child)
    Pre-visit C (grandchild)
    Post-visit C
  Post-visit B
Post-visit A
```

### Use Cases

**1. Counting child nodes:**

```go
propertyCounts := make(map[string]int)

walker.Walk(result,
    walker.WithSchemaPostHandler(func(wc *walker.WalkContext, schema *parser.Schema) {
        if wc.IsComponent && wc.Name != "" {
            propertyCounts[wc.Name] = len(schema.Properties)
        }
    }),
)
```

**2. Bottom-up validation:**

```go
var issues []string

walker.Walk(result,
    walker.WithOperationPostHandler(func(wc *walker.WalkContext, op *parser.Operation) {
        // Validate after all parameters and responses are processed
        if op.OperationID == "" {
            issues = append(issues, fmt.Sprintf("%s: missing operationId", wc.JSONPath))
        }
    }),
)
```

**3. Aggregating statistics:**

```go
schemaStats := make(map[string]struct {
    PropertyCount int
    RequiredCount int
})

walker.Walk(result,
    walker.WithSchemaPostHandler(func(wc *walker.WalkContext, schema *parser.Schema) {
        if wc.IsComponent && wc.Name != "" && !strings.Contains(wc.JSONPath, ".properties") {
            schemaStats[wc.Name] = struct {
                PropertyCount int
                RequiredCount int
            }{
                PropertyCount: len(schema.Properties),
                RequiredCount: len(schema.Required),
            }
        }
    }),
)
```

**4. Building summary data after traversal:**

```go
var operationsByPath = make(map[string]int)

walker.Walk(result,
    walker.WithPathItemPostHandler(func(wc *walker.WalkContext, pathItem *parser.PathItem) {
        // Count operations in this path item
        count := 0
        if pathItem.Get != nil { count++ }
        if pathItem.Post != nil { count++ }
        if pathItem.Put != nil { count++ }
        if pathItem.Delete != nil { count++ }
        if pathItem.Patch != nil { count++ }
        operationsByPath[wc.PathTemplate] = count
    }),
)
```

### Combining Pre and Post Handlers

You can use both pre and post handlers together:

```go
walker.Walk(result,
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        // Pre-visit: mark schema as being processed
        fmt.Printf("Entering %s\n", wc.JSONPath)
        return walker.Continue
    }),
    walker.WithSchemaPostHandler(func(wc *walker.WalkContext, schema *parser.Schema) {
        // Post-visit: mark schema as complete
        fmt.Printf("Leaving %s\n", wc.JSONPath)
    }),
)
```

### Using Post Handlers Alone

Post handlers work without pre-handlers:

```go
// Only register post handler
walker.Walk(result,
    walker.WithSchemaPostHandler(func(wc *walker.WalkContext, schema *parser.Schema) {
        // Called for every schema after its children are processed
    }),
)
```

### Document Post Handlers: Single-Walk Aggregation

Document post handlers (`WithOAS2DocumentPostHandler`, `WithOAS3DocumentPostHandler`) enable single-walk patterns where you collect information from child nodes and then modify the document root based on that collection.

**Use case: Adding security definitions based on operation analysis**

Without document post handlers, you'd need two walks:

```go
// Old approach: Two walks required
needsOAuth2 := false
needsAPIKey := false

// Walk 1: Collect security requirements from operations
err := walker.Walk(result,
    walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
        authType := getAuthType(op)
        if authType == "oauth2" {
            needsOAuth2 = true
        } else if authType == "apiKey" {
            needsAPIKey = true
        }
        return walker.Continue
    }),
)

// Walk 2: Add security definitions based on collected info
err = walker.Walk(result,
    walker.WithOAS3DocumentHandler(func(_ *walker.WalkContext, doc *parser.OAS3Document) walker.Action {
        if needsOAuth2 {
            doc.Components.SecuritySchemes["oauth2"] = &parser.SecurityScheme{...}
        }
        return walker.Continue
    }),
)
```

**With document post handlers (single walk):**

```go
// New approach: Single walk with document post handler
needsOAuth2 := false
needsAPIKey := false
scopes := make(map[string][]string)

err := walker.Walk(result,
    walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
        authType := getAuthType(op)
        if authType == "oauth2" {
            needsOAuth2 = true
            scopes[op.OperationID] = buildScopes(op)
        } else if authType == "apiKey" {
            needsAPIKey = true
        }
        return walker.Continue
    }),
    walker.WithOAS3DocumentPostHandler(func(_ *walker.WalkContext, doc *parser.OAS3Document) {
        // Called AFTER all operations have been visited
        if needsOAuth2 {
            doc.Components.SecuritySchemes["oauth2"] = &parser.SecurityScheme{
                Type:  "oauth2",
                Flows: buildOAuthFlows(scopes),
            }
        }
        if needsAPIKey {
            doc.Components.SecuritySchemes["api_key"] = &parser.SecurityScheme{
                Type: "apiKey",
                In:   "header",
                Name: "X-API-Key",
            }
        }
    }),
)
```

This pattern is useful for:

- Adding security schemes based on operation requirements
- Generating documentation tags from operation tags
- Adding components discovered during traversal
- Validating document-wide constraints after seeing all nodes

### Performance Considerations

Post handlers add minimal overhead since they reuse the existing WalkContext. The primary cost is the function call itself.
