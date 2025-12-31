<a id="top"></a>

# Differ Package Deep Dive

!!! tip "Try it Online"
    No installation required! [Try the differ in your browser →](https://oastools.robnrob.com/diff)

## Table of Contents

- [Overview](#overview)
- [Key Concepts](#key-concepts)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [Schema Comparison Details](#schema-comparison-details)
- [Extension Field Coverage](#extension-field-coverage)
- [Source Map Integration](#source-map-integration)
- [Integration with Other Packages](#integration-with-other-packages)
- [Best Practices](#best-practices)
- [Configurable Breaking Change Rules](#configurable-breaking-change-rules)
- [DiffResult Structure](#diffresult-structure)

---

The [`differ`](https://pkg.go.dev/github.com/erraggy/oastools/differ) package provides OpenAPI specification comparison and breaking change detection. It enables you to identify differences between API versions, categorize changes by severity, and detect backward-incompatible modifications that could break existing clients.

## Overview

The differ supports comparing OAS 2.0 and OAS 3.x documents, offering two operational modes: simple semantic diffing and breaking change detection with severity classification. It integrates seamlessly with the parse-once pattern, delivering 81x faster performance when working with pre-parsed documents.

## Key Concepts

### Diff Modes

The differ operates in two modes that determine how changes are analyzed and reported:

**ModeSimple** reports all semantic differences between documents without categorization. Use this mode when you need a comprehensive list of what changed without severity assessment. See also: [Simple diff example](https://pkg.go.dev/github.com/erraggy/oastools/differ#example-package-Simple) on pkg.go.dev

**ModeBreaking** categorizes every change by both category (what part of the spec changed) and severity (how impactful the change is). This mode is essential for CI/CD pipelines that need to gate releases based on API compatibility. See also: [Breaking change detection example](https://pkg.go.dev/github.com/erraggy/oastools/differ#example-package-Breaking) on pkg.go.dev

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

See also: [Basic example](https://pkg.go.dev/github.com/erraggy/oastools/differ#example-package), [Breaking changes](https://pkg.go.dev/github.com/erraggy/oastools/differ#example-package-BreakingChanges) on pkg.go.dev

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

See also: [Parsed documents example](https://pkg.go.dev/github.com/erraggy/oastools/differ#example-package-Parsed) on pkg.go.dev

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

See also: [Change analysis example](https://pkg.go.dev/github.com/erraggy/oastools/differ#example-package-ChangeAnalysis) on pkg.go.dev

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

See also: [Filter by severity example](https://pkg.go.dev/github.com/erraggy/oastools/differ#example-package-FilterBySeverity) on pkg.go.dev

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

See also: [Reusable differ example](https://pkg.go.dev/github.com/erraggy/oastools/differ#example-package-ReusableDiffer) on pkg.go.dev

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

## Extension Field Coverage

The differ tracks changes to custom extension fields at commonly-used locations:

**Diffed Locations:** Document level, Info, Server, PathItem, Operation, Parameter, RequestBody, Response, Header, Link, MediaType, Schema, SecurityScheme, Tag, Components

**Not Diffed:** Contact, License, ExternalDocs, ServerVariable, Reference, Items, Example, Encoding, Discriminator, XML, OAuthFlows

All extension changes are reported with `SeverityInfo` since extensions are non-normative.

[↑ Back to top](#top)

## Source Map Integration

Source maps enable **precise change locations** by tracking line and column numbers from your YAML/JSON source files. Without source maps, changes only show JSON paths. With source maps, changes include file:line:column positions that IDEs can click to jump directly to the modification.

**Without source maps:**
```
paths./pets.get.parameters.limit: required changed from false to true
```

**With source maps:**
```
api-v2.yaml:45:11: required changed from false to true
```

The differ compares two documents, so it accepts both `WithSourceMap` (for the source/old document) and `WithTargetMap` (for the target/new document):

```go
source, _ := parser.ParseWithOptions(
    parser.WithFilePath("api-v1.yaml"),
    parser.WithSourceMap(true),  // Enable line tracking during parse
)
target, _ := parser.ParseWithOptions(
    parser.WithFilePath("api-v2.yaml"),
    parser.WithSourceMap(true),  // Enable line tracking during parse
)

result, _ := differ.DiffWithOptions(
    differ.WithSourceParsed(*source),
    differ.WithTargetParsed(*target),
    differ.WithSourceMap(source.SourceMap),   // Source document locations
    differ.WithTargetMap(target.SourceMap),   // Target document locations
    differ.WithMode(differ.ModeBreaking),
)

// Changes now include line/column/file info
for _, change := range result.Changes {
    if change.HasLocation() {
        // IDE-friendly format: file:line:column
        fmt.Printf("%s: %s\n", change.Location(), change.Description)
    } else {
        // Fallback to JSON path
        fmt.Printf("%s: %s\n", change.Path, change.Description)
    }
}
```

The `Location()` method returns the IDE-friendly `file:line:column` format pointing to where the change occurred in the target document. The `HasLocation()` method checks if line info is available (returns `true` when `Line > 0`).

[Back to top](#top)

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

## Configurable Breaking Change Rules

Organizations often have custom policies for what constitutes a breaking change. The differ supports configurable rules that let you override default severity levels or completely ignore certain change types.

### Rule Configuration

Use `BreakingRulesConfig` to customize breaking change detection:

```go
rules := &differ.BreakingRulesConfig{
    Operation: &differ.OperationRules{
        // Downgrade operationId changes to Info (not breaking for us)
        OperationIDModified: &differ.BreakingChangeRule{
            Severity: differ.SeverityPtr(differ.SeverityInfo),
        },
    },
    Schema: &differ.SchemaRules{
        // Completely ignore property removal (we handle this differently)
        PropertyRemoved: &differ.BreakingChangeRule{Ignore: true},
    },
}

result, err := differ.DiffWithOptions(
    differ.WithSourceFilePath("api-v1.yaml"),
    differ.WithTargetFilePath("api-v2.yaml"),
    differ.WithMode(differ.ModeBreaking),
    differ.WithBreakingRules(rules),
)
```

### Preset Rule Configurations

Three preset configurations are available for common use cases:

```go
// DefaultRules - uses built-in severity defaults
rules := differ.DefaultRules()

// StrictRules - elevates warnings to errors (stricter)
rules := differ.StrictRules()

// LenientRules - downgrades some errors to warnings (more permissive)
rules := differ.LenientRules()
```

### Available Rule Categories

| Category | Description | Key Rules |
|----------|-------------|-----------|
| `OperationRules` | Operation-level changes | `Removed`, `OperationIDModified`, `DeprecatedChanged`, `SummaryModified`, `DescriptionModified` |
| `ParameterRules` | Parameter changes | `Added`, `Removed`, `RequiredChanged`, `TypeChanged`, `LocationChanged`, `DescriptionModified` |
| `RequestBodyRules` | Request body changes | `Added`, `Removed`, `RequiredChanged`, `ContentTypeAdded`, `ContentTypeRemoved`, `DescriptionModified` |
| `ResponseRules` | Response changes | `Added`, `Removed`, `StatusCodeAdded`, `StatusCodeRemoved`, `ContentTypeAdded`, `ContentTypeRemoved`, `DescriptionModified` |
| `SchemaRules` | Schema changes | `TypeChanged`, `PropertyAdded`, `PropertyRemoved`, `RequiredAdded`, `RequiredRemoved`, `EnumValueAdded`, `EnumValueRemoved`, `DescriptionModified` |
| `SecurityRules` | Security scheme changes | `Added`, `Removed`, `TypeChanged`, `DescriptionModified` |
| `ServerRules` | Server changes | `Added`, `Removed`, `URLChanged`, `DescriptionModified` |
| `EndpointRules` | Endpoint changes | `Added`, `Removed`, `DescriptionModified` |
| `InfoRules` | Info object changes | `TitleChanged`, `VersionChanged`, `DescriptionModified` |
| `ExtensionRules` | Extension field changes | `Added`, `Removed`, `Modified` |

### Struct-Based Configuration

For reusable differ instances:

```go
d := differ.New()
d.Mode = differ.ModeBreaking
d.BreakingRules = &differ.BreakingRulesConfig{
    Operation: &differ.OperationRules{
        OperationIDModified: &differ.BreakingChangeRule{
            Severity: differ.SeverityPtr(differ.SeverityError), // Upgrade to error
        },
    },
}

result1, _ := d.Diff("api-v1.yaml", "api-v2.yaml")
result2, _ := d.Diff("api-v2.yaml", "api-v3.yaml")
```

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

---

## Learn More

For additional examples and complete API documentation:

- 📦 [API Reference on pkg.go.dev](https://pkg.go.dev/github.com/erraggy/oastools/differ) - Complete API documentation with all examples
- 🔍 [Simple diff example](https://pkg.go.dev/github.com/erraggy/oastools/differ#example-package-Simple) - Basic semantic diffing
- ⚠️ [Breaking changes example](https://pkg.go.dev/github.com/erraggy/oastools/differ#example-package-BreakingChanges) - Comprehensive breaking change detection
- 📊 [Change analysis example](https://pkg.go.dev/github.com/erraggy/oastools/differ#example-package-ChangeAnalysis) - Grouping changes by category
- 🔧 [Reusable differ example](https://pkg.go.dev/github.com/erraggy/oastools/differ#example-package-ReusableDiffer) - Comparing multiple API versions
