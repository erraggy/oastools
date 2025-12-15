<a id="top"></a>

# Differ Package Deep Dive

## Table of Contents

- [Overview](#overview)
- [Key Concepts](#key-concepts)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [Schema Comparison Details](#schema-comparison-details)
- [Extension (x-*) Field Coverage](#extension-x--field-coverage)
- [Integration with Other Packages](#integration-with-other-packages)
- [Best Practices](#best-practices)
- [DiffResult Structure](#diffresult-structure)

---

The `differ` package provides OpenAPI specification comparison and breaking change detection. It enables you to identify differences between API versions, categorize changes by severity, and detect backward-incompatible modifications that could break existing clients.

## Overview

The differ supports comparing OAS 2.0 and OAS 3.x documents, offering two operational modes: simple semantic diffing and breaking change detection with severity classification. It integrates seamlessly with the parse-once pattern, delivering 81x faster performance when working with pre-parsed documents.

## Key Concepts

### Diff Modes

The differ operates in two modes that determine how changes are analyzed and reported:

**ModeSimple** reports all semantic differences between documents without categorization. Use this mode when you need a comprehensive list of what changed without severity assessment.

**ModeBreaking** categorizes every change by both category (what part of the spec changed) and severity (how impactful the change is). This mode is essential for CI/CD pipelines that need to gate releases based on API compatibility.

### Change Categories

Changes are organized by the specification element that was modified:

| Category | Description |
|----------|-------------|
| `CategoryEndpoint` | Path/endpoint additions, removals, or modifications |
| `CategoryOperation` | HTTP method changes (GET, POST, etc.) |
| `CategoryParameter` | Query, path, header, or cookie parameter changes |
| `CategoryRequestBody` | Request body schema or content type changes |
| `CategoryResponse` | Response schema, status code, or header changes |
| `CategorySchema` | Component schema/definition changes |
| `CategorySecurity` | Security scheme modifications |
| `CategoryServer` | Server URL or variable changes |
| `CategoryInfo` | Metadata changes (title, version, description) |

### Severity Levels

In `ModeBreaking`, each change receives a severity level indicating its impact on API consumers:

| Severity | Impact | Examples |
|----------|--------|----------|
| `SeverityCritical` | Breaking - immediate client failure | Removed endpoint, removed operation |
| `SeverityError` | Breaking - client code changes required | Removed required parameter, type changes |
| `SeverityWarning` | Potentially problematic | Deprecated operations, new required fields |
| `SeverityInfo` | Non-breaking | Additions, relaxed constraints |

[↑ Back to top](#top)

## API Styles

The differ provides two complementary API patterns:

### Functional Options API

Best for one-off comparisons with inline configuration:

```go
result, err := differ.DiffWithOptions(
    differ.WithSourceFilePath("api-v1.yaml"),
    differ.WithTargetFilePath("api-v2.yaml"),
    differ.WithMode(differ.ModeBreaking),
    differ.WithIncludeInfo(true),
)
```

### Struct-Based API

Best for comparing multiple document pairs with consistent configuration:

```go
d := differ.New()
d.Mode = differ.ModeBreaking
d.IncludeInfo = false

// Compare multiple API versions
result1, _ := d.Diff("api-v1.yaml", "api-v2.yaml")
result2, _ := d.Diff("api-v2.yaml", "api-v3.yaml")
```

[↑ Back to top](#top)

## Practical Examples

### Basic Difference Detection

The simplest use case compares two specifications and reports all changes:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/erraggy/oastools/differ"
)

func main() {
    result, err := differ.DiffWithOptions(
        differ.WithSourceFilePath("api-v1.yaml"),
        differ.WithTargetFilePath("api-v2.yaml"),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d changes between versions\n", len(result.Changes))
    fmt.Printf("Source version: %s\n", result.SourceVersion)
    fmt.Printf("Target version: %s\n", result.TargetVersion)
    
    for _, change := range result.Changes {
        fmt.Println(change.String())
    }
}
```

**Example Input (api-v1.yaml):**
```yaml
openapi: 3.0.3
info:
  title: Pet Store API
  version: 1.0.0
paths:
  /pets:
    get:
      operationId: listPets
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        '200':
          description: Success
```

**Example Input (api-v2.yaml):**
```yaml
openapi: 3.0.3
info:
  title: Pet Store API
  version: 2.0.0
paths:
  /pets:
    get:
      operationId: listPets
      parameters:
        - name: limit
          in: query
          required: true
          schema:
            type: integer
        - name: offset
          in: query
          schema:
            type: integer
      responses:
        '200':
          description: Success
```

**Example Output:**
```
Found 2 changes between versions
Source version: 3.0.3
Target version: 3.0.3
paths./pets.get.parameters.limit: required changed from false to true
paths./pets.get.parameters: added parameter 'offset'
```

### Breaking Change Detection for CI/CD

Use breaking change detection to gate deployments:

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/erraggy/oastools/differ"
)

func main() {
    result, err := differ.DiffWithOptions(
        differ.WithSourceFilePath("current-api.yaml"),
        differ.WithTargetFilePath("proposed-api.yaml"),
        differ.WithMode(differ.ModeBreaking),
        differ.WithIncludeInfo(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Summary: %d breaking, %d warnings, %d info\n",
        result.BreakingCount, result.WarningCount, result.InfoCount)
    
    if result.HasBreakingChanges {
        fmt.Println("\n⚠️  Breaking changes detected!")
        for _, change := range result.Changes {
            if change.Severity == differ.SeverityCritical || 
               change.Severity == differ.SeverityError {
                fmt.Printf("  [%s] %s: %s\n", 
                    change.Severity, change.Path, change.Description)
            }
        }
        os.Exit(1)
    }
    
    fmt.Println("✓ No breaking changes detected")
}
```

**Example Output (with breaking changes):**
```
Summary: 2 breaking, 1 warnings, 3 info

⚠️  Breaking changes detected!
  [critical] paths./users/{id}: endpoint removed
  [error] paths./pets.get.parameters.limit: changed from optional to required
```

### High-Performance Diffing with Pre-Parsed Documents

When processing multiple comparisons or integrating with other oastools packages, use pre-parsed documents for 81x faster performance:

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/erraggy/oastools/differ"
    "github.com/erraggy/oastools/parser"
)

func main() {
    // Parse documents once
    source, err := parser.ParseWithOptions(
        parser.WithFilePath("api-v1.yaml"),
        parser.WithValidateStructure(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    target, err := parser.ParseWithOptions(
        parser.WithFilePath("api-v2.yaml"),
        parser.WithValidateStructure(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Compare using parsed documents (skips parsing overhead)
    start := time.Now()
    result, err := differ.DiffWithOptions(
        differ.WithSourceParsed(*source),
        differ.WithTargetParsed(*target),
        differ.WithMode(differ.ModeBreaking),
    )
    elapsed := time.Since(start)
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Diff completed in %v\n", elapsed)
    fmt.Printf("Changes: %d\n", len(result.Changes))
}
```

### Grouping Changes by Category

Organize diff results for better reporting:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/erraggy/oastools/differ"
)

func main() {
    result, err := differ.DiffWithOptions(
        differ.WithSourceFilePath("api-v1.yaml"),
        differ.WithTargetFilePath("api-v2.yaml"),
        differ.WithMode(differ.ModeBreaking),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Group changes by category
    categories := make(map[differ.ChangeCategory][]differ.Change)
    for _, change := range result.Changes {
        categories[change.Category] = append(
            categories[change.Category], change)
    }
    
    // Report in logical order
    categoryOrder := []differ.ChangeCategory{
        differ.CategoryEndpoint,
        differ.CategoryOperation,
        differ.CategoryParameter,
        differ.CategoryRequestBody,
        differ.CategoryResponse,
        differ.CategorySchema,
        differ.CategorySecurity,
        differ.CategoryServer,
        differ.CategoryInfo,
    }
    
    for _, category := range categoryOrder {
        changes := categories[category]
        if len(changes) > 0 {
            fmt.Printf("\n%s Changes (%d):\n", category, len(changes))
            for _, change := range changes {
                fmt.Printf("  [%s] %s\n", change.Severity, change.String())
            }
        }
    }
}
```

**Example Output:**
```
parameter Changes (2):
  [error] paths./pets.get.parameters.limit: required changed from false to true
  [info] paths./pets.get.parameters: added parameter 'offset'

schema Changes (1):
  [warning] components.schemas.Pet.properties.status: deprecated
```

### Filtering Changes by Severity

Focus on specific severity levels:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/erraggy/oastools/differ"
)

func main() {
    result, err := differ.DiffWithOptions(
        differ.WithSourceFilePath("api-v1.yaml"),
        differ.WithTargetFilePath("api-v2.yaml"),
        differ.WithMode(differ.ModeBreaking),
        differ.WithIncludeInfo(false), // Exclude informational changes
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Only process breaking changes (Critical + Error)
    var breaking []differ.Change
    for _, change := range result.Changes {
        switch change.Severity {
        case differ.SeverityCritical, differ.SeverityError:
            breaking = append(breaking, change)
        }
    }
    
    if len(breaking) == 0 {
        fmt.Println("No breaking changes found")
        return
    }
    
    fmt.Printf("Found %d breaking changes:\n", len(breaking))
    for _, change := range breaking {
        fmt.Printf("  %s\n", change.String())
    }
}
```

### Comparing Multiple API Version Pairs

When you need to analyze an API's evolution across multiple versions:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/erraggy/oastools/differ"
)

func main() {
    // Create a reusable differ instance
    d := differ.New()
    d.Mode = differ.ModeBreaking
    d.IncludeInfo = false
    
    // Compare multiple version pairs
    pairs := []struct{ old, new string }{
        {"api-v1.yaml", "api-v2.yaml"},
        {"api-v2.yaml", "api-v3.yaml"},
        {"api-v3.yaml", "api-v4.yaml"},
    }
    
    for _, pair := range pairs {
        result, err := d.Diff(pair.old, pair.new)
        if err != nil {
            log.Printf("Error comparing %s to %s: %v", pair.old, pair.new, err)
            continue
        }
        
        fmt.Printf("\n%s → %s:\n", pair.old, pair.new)
        if result.HasBreakingChanges {
            fmt.Printf("  ⚠️  %d breaking changes\n", result.BreakingCount)
            fmt.Printf("  ⚠️  %d warnings\n", result.WarningCount)
        } else {
            fmt.Printf("  ✓ No breaking changes\n")
            if result.WarningCount > 0 {
                fmt.Printf("  ℹ️  %d warnings\n", result.WarningCount)
            }
        }
    }
}
```

[↑ Back to top](#top)

## Schema Comparison Details

The differ performs comprehensive schema comparison including:

**Type Information:** type, format

**Numeric Constraints:** multipleOf, maximum, exclusiveMaximum, minimum, exclusiveMinimum

**String Constraints:** maxLength, minLength, pattern

**Array Constraints:** maxItems, minItems, uniqueItems

**Object Constraints:** maxProperties, minProperties, required fields

**OAS-specific Fields:** nullable, readOnly, writeOnly, deprecated

### Smart Severity Assignment for Schema Changes

The differ uses intelligent severity assignment for constraint changes:

```
ERROR severity (stricter = breaking):
  - Adding required fields
  - Lowering maximum values
  - Raising minimum values
  - Reducing maxLength/maxItems/maxProperties

WARNING severity (potentially problematic):
  - Type changes
  - Format changes
  - Pattern modifications

INFO severity (relaxations = non-breaking):
  - Removing required fields
  - Raising maximum values
  - Lowering minimum values
  - Increasing maxLength/maxItems/maxProperties
```

[↑ Back to top](#top)

## Extension (x-*) Field Coverage

The differ tracks changes to custom extension fields at commonly-used locations:

**Diffed Locations:** Document level, Info, Server, PathItem, Operation, Parameter, RequestBody, Response, Header, Link, MediaType, Schema, SecurityScheme, Tag, Components

**Not Diffed:** Contact, License, ExternalDocs, ServerVariable, Reference, Items, Example, Encoding, Discriminator, XML, OAuthFlows

All extension changes are reported with `SeverityInfo` since extensions are non-normative.

[↑ Back to top](#top)

## Integration with Other Packages

The differ integrates naturally with the oastools ecosystem:

```go
// Parse → Validate → Diff workflow
source, _ := parser.ParseWithOptions(parser.WithFilePath("api-v1.yaml"))
target, _ := parser.ParseWithOptions(parser.WithFilePath("api-v2.yaml"))

// Validate both documents
sourceVal, _ := validator.ValidateWithOptions(validator.WithParsed(*source))
targetVal, _ := validator.ValidateWithOptions(validator.WithParsed(*target))

if !sourceVal.Valid || !targetVal.Valid {
    log.Fatal("Documents must be valid before comparison")
}

// Compare validated documents
result, _ := differ.DiffWithOptions(
    differ.WithSourceParsed(*source),
    differ.WithTargetParsed(*target),
    differ.WithMode(differ.ModeBreaking),
)
```

[↑ Back to top](#top)

## Best Practices

**Always use ModeBreaking for production workflows** to get severity classifications that enable automated decision-making.

**Document all breaking changes in release notes** with migration guides for each Critical or Error severity change.

**Consider deprecation first** before removing features. Deprecation appears as Warning severity, giving consumers time to adapt.

**Pin to specific major versions** based on severity levels—Critical and Error changes warrant major version bumps.

**Use the parse-once pattern** when comparing multiple documents or integrating with other packages for 81x performance improvement.

[↑ Back to top](#top)

## DiffResult Structure

```go
type DiffResult struct {
    // SourceVersion is the OAS version of the source document
    SourceVersion string
    // TargetVersion is the OAS version of the target document
    TargetVersion string
    // Changes contains all detected changes
    Changes []Change
    // BreakingCount is the number of breaking changes (Critical + Error)
    BreakingCount int
    // WarningCount is the number of warnings
    WarningCount int
    // InfoCount is the number of informational changes
    InfoCount int
    // HasBreakingChanges is true if any breaking changes were detected
    HasBreakingChanges bool
}

type Change struct {
    Type        ChangeType     // added, removed, modified
    Category    ChangeCategory // endpoint, operation, parameter, etc.
    Severity    Severity       // critical, error, warning, info
    Path        string         // JSON path to changed element
    Description string         // Human-readable description
    OldValue    any            // Previous value (for modifications)
    NewValue    any            // New value (for modifications)
}
```
