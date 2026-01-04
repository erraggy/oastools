# API Documentation Generator

Demonstrates generating Markdown API documentation from OpenAPI specifications using walker handlers.

## What You'll Learn

- Extracting documentation from multiple handler types in a single pass
- Maintaining state across nested handlers (path → operation → parameters/responses)
- Generating structured Markdown output from OpenAPI specifications
- Using walker for comprehensive documentation generation

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/walker/api-documentation
go run main.go
```

## Expected Output

```markdown
# Petstore API

**Version:** 1.0.0

A sample API for a pet store

## Servers

| Environment | URL |
|-------------|-----|
| Production server | https://petstore.example.com/v1 |
| Staging server | https://staging.petstore.example.com/v1 |

## Tags

- **pets** - Pet operations

## Endpoints

### GET /pets

**listPets**: List all pets

**Parameters:**

| Name | In | Required | Description |
|------|-----|----------|-------------|
| limit | query | No | How many items to return at one time |

**Responses:**

| Status | Description |
|--------|-------------|
| 200 | A paged array of pets |
| default | unexpected error |

---

### POST /pets

**createPet**: Create a pet

...
```

## Files

| File | Purpose |
|------|---------|
| main.go | Generates Markdown documentation using multiple walker handlers |
| go.mod | Module definition with local replace directive |

## Key Concepts

### State Management Across Handlers

The walker visits nodes in document order. To associate parameters and responses with their parent operation, we track the current context:

```go
var currentEndpoint *EndpointDoc

walker.Walk(parseResult,
    walker.WithPathHandler(func(wc *walker.WalkContext, pathItem *parser.PathItem) walker.Action {
        // wc.PathTemplate available in context
        return walker.Continue
    }),
    walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
        endpoint := EndpointDoc{Path: wc.PathTemplate, Method: wc.Method, ...}
        doc.Endpoints = append(doc.Endpoints, endpoint)
        currentEndpoint = &doc.Endpoints[len(doc.Endpoints)-1]  // Track current endpoint
        return walker.Continue
    }),
    walker.WithParameterHandler(func(wc *walker.WalkContext, param *parser.Parameter) walker.Action {
        currentEndpoint.Parameters = append(currentEndpoint.Parameters, ...)  // Add to current
        return walker.Continue
    }),
)
```

### Handler Ordering and Nesting

The walker visits nodes in this order:
1. Document root → Info → Servers → Tags
2. For each path: PathHandler → OperationHandler → ParameterHandler → ResponseHandler
3. Components (schemas, security schemes, etc.)

Parameters and responses are visited as children of their parent operation, allowing context-aware collection.

### Structured Output Generation

After walking, the collected data is sorted and formatted into Markdown:

```go
sort.Slice(doc.Endpoints, func(i, j int) bool {
    if doc.Endpoints[i].Path != doc.Endpoints[j].Path {
        return doc.Endpoints[i].Path < doc.Endpoints[j].Path
    }
    return doc.Endpoints[i].Method < doc.Endpoints[j].Method
})

generateMarkdown(doc)
```

## Use Cases

- **README Generation**: Auto-generate API documentation for repositories
- **API Guides**: Create human-readable endpoint references
- **Documentation Sites**: Generate content for static site generators
- **API Catalogs**: Build searchable API inventories

## Next Steps

- [Walker Deep Dive](../../../walker/deep_dive.md) - Complete walker documentation
- [API Statistics](../api-statistics/) - Collect statistics using multiple handlers
- [Reference Collector](../reference-collector/) - Track schema definitions and references

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
