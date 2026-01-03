# Reference Collector

Demonstrates analyzing schema references and detecting circular references using the walker package.

## What You'll Learn

- Using SchemaSkippedHandler for cycle and depth notifications
- Building reference graphs with walker
- Identifying unused components in your API specification
- Configuring WithMaxDepth for controlled schema traversal
- Detecting self-referencing schemas (circular references)

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/walker/reference-collector
go run main.go
```

## Expected Output

```
Reference Analysis Report
=========================

Schema References:
  Category (1 reference):
    - $.components.schemas['Pet'].properties['category']
  DeepLevel1 (1 reference):
    - $.components.schemas['DeepSchema'].allOf[0]
  DeepLevel2 (1 reference):
    - $.components.schemas['DeepLevel1'].allOf[0]
  DeepLevel3 (1 reference):
    - $.components.schemas['DeepLevel2'].allOf[0]
  DeepLevel4 (1 reference):
    - $.components.schemas['DeepLevel3'].allOf[0]
  DeepLevel5 (1 reference):
    - $.components.schemas['DeepLevel4'].allOf[0]
  Error (4 references):
    - $.paths['/nodes'].get.responses.default.content['application/json'].schema
    - $.paths['/pets'].get.responses.default.content['application/json'].schema
    - $.paths['/pets'].post.responses.default.content['application/json'].schema
    - $.paths['/pets/{petId}'].get.responses.default.content['application/json'].schema
  NewPet (1 reference):
    - $.paths['/pets'].post.requestBody.content['application/json'].schema
  Node (2 references):
    - $.paths['/nodes'].get.responses['200'].content['application/json'].schema.items
    - $.components.schemas['Node'].properties['children'].items
  Pet (3 references):
    - $.paths['/pets'].get.responses['200'].content['application/json'].schema.items
    - $.paths['/pets'].post.responses['201'].content['application/json'].schema
    - $.paths['/pets/{petId}'].get.responses['200'].content['application/json'].schema

Unused Schemas (3):
  - DeepSchema
  - InternalConfig
  - LegacyModel

Self-Referencing Schemas (1):
  - $.components.schemas['Node'].properties['children'].items

Walker Cycle Events (0):
  (none)

Depth-Limited Schemas: 0
```

## Files

| File | Purpose |
|------|---------|
| main.go | Collects schema references and detects cycles using walker |
| specs/complex-api.yaml | Test specification with circular refs, deep nesting, and unused schemas |
| go.mod | Module definition with local replace directive |

## Key Concepts

### SchemaSkippedHandler Reasons

The `SchemaSkippedHandler` is called when the walker skips a schema for one of two reasons:

```go
walker.WithSchemaSkippedHandler(func(reason string, schema *parser.Schema, path string) {
    switch reason {
    case "cycle":
        // Schema was already visited - circular reference detected
        collector.Cycles = append(collector.Cycles, path)
    case "depth":
        // Schema exceeds maxDepth - depth limit reached
        collector.DepthLimited = append(collector.DepthLimited, path)
    }
})
```

### Reference Tracking Pattern

Track where each schema is referenced by extracting the name from `$ref`:

```go
walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
    if schema.Ref != "" {
        schemaName := extractSchemaName(schema.Ref)
        refs[schemaName] = append(refs[schemaName], path)
    }
    return walker.Continue
})
```

### Handling Array Items with References

When an array schema has items with a `$ref`, the parser stores it as `map[string]any`:

```go
// Handle items that contain a $ref (stored as map[string]any)
if items, ok := schema.Items.(map[string]any); ok {
    if ref, ok := items["$ref"].(string); ok {
        schemaName := extractSchemaName(ref)
        collector.SchemaRefs[schemaName] = append(collector.SchemaRefs[schemaName], path+".items")
    }
}
```

### Unused Component Detection

After walking, compare defined schemas against referenced ones:

```go
for _, name := range allSchemaNames {
    if _, hasRefs := collector.SchemaRefs[name]; !hasRefs {
        unusedSchemas = append(unusedSchemas, name)
    }
}
```

### Self-Reference Detection

Detect schemas that reference themselves by checking if any reference path starts with the schema's own component path:

```go
for name, refs := range collector.SchemaRefs {
    prefix := "$.components.schemas['" + name + "']"
    for _, refPath := range refs {
        if strings.HasPrefix(refPath, prefix) {
            collector.SelfReferences = append(collector.SelfReferences, refPath)
        }
    }
}
```

### MaxDepth Configuration

Use `WithMaxDepth` to prevent infinite traversal in deeply nested schemas:

```go
walker.Walk(parseResult,
    walker.WithMaxDepth(50),  // Limit schema traversal depth
    // ... handlers
)
```

## Use Cases

- **Dead Code Detection**: Find unused schemas that can be safely removed
- **Dependency Analysis**: Build a reference graph of your API components
- **API Cleanup**: Identify legacy or orphaned schemas
- **Circular Reference Auditing**: Detect and document self-referential structures
- **Refactoring Support**: Understand schema dependencies before making changes

## Next Steps

- [Walker Deep Dive](../../../walker/deep_dive.md) - Complete walker documentation
- [API Statistics](../api-statistics/) - Collect statistics about your API
- [Security Audit](../security-audit/) - Audit security schemes and authentication

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
