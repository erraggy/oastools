# Developer Guide

This guide provides comprehensive documentation for developers using oastools as both a library and a command-line tool.

## Table of Contents

- [Installation](#installation)
- [CLI Usage](#cli-usage)
  - [Validate Command](#validate-command)
  - [Parse Command](#parse-command)
  - [Convert Command](#convert-command)
  - [Join Command](#join-command)
  - [Diff Command](#diff-command)
  - [Generate Command](#generate-command)
- [Library Usage](#library-usage)
  - [Parser Package](#parser-package)
  - [Validator Package](#validator-package)
  - [Converter Package](#converter-package)
  - [Joiner Package](#joiner-package)
  - [Differ Package](#differ-package)
  - [Generator Package](#generator-package)
  - [Builder Package](#builder-package)
- [Advanced Patterns](#advanced-patterns)
  - [Parse-Once Pattern](#parse-once-pattern)
  - [Error Handling](#error-handling)
  - [Working with Different OAS Versions](#working-with-different-oas-versions)
- [Troubleshooting](#troubleshooting)

## Installation

### CLI Tool

```bash
# Homebrew (macOS and Linux)
brew tap erraggy/oastools
brew install oastools

# Go install
go install github.com/erraggy/oastools/cmd/oastools@latest

# From source
git clone https://github.com/erraggy/oastools.git
cd oastools
make install
```

### Library

```bash
go get github.com/erraggy/oastools@latest
```

## CLI Usage

### Validate Command

The validate command checks an OpenAPI specification for correctness against the OpenAPI Specification.

**Basic Usage:**

```bash
# Validate a local file
oastools validate openapi.yaml

# Validate from a URL
oastools validate https://example.com/api/openapi.yaml
```

**Options:**

```bash
# Enable strict mode (treats warnings as errors)
oastools validate --strict openapi.yaml

# Suppress warnings (show only errors)
oastools validate --no-warnings openapi.yaml
```

**Understanding Output:**

```
OpenAPI Specification Validator
================================

Specification: openapi.yaml
OAS Version: 3.0.3
Source Size: 2.5 KB
Paths: 5
Operations: 12
Schemas: 8
Load Time: 125ms
Total Time: 140ms

Errors (2):
  ✗ paths./users.get.responses: missing required field '200' or 'default': at least one response is required
    Spec: https://spec.openapis.org/oas/v3.0.3.html#responses-object
  ✗ components.schemas.User.properties.id: missing required 'type' field
    Spec: https://spec.openapis.org/oas/v3.0.3.html#schema-object

Warnings (1):
  ⚠ paths./users/{id}.get: Operation should have a description or summary for better documentation
    Spec: https://spec.openapis.org/oas/v3.0.3.html#operation-object

✗ Validation failed: 2 error(s), 1 warning(s)
```

**Common Validation Errors and Fixes:**

| Error | Solution |
|-------|----------|
| `missing required field 'info.version'` | Add `version: "1.0.0"` to the info section |
| `path must begin with '/'` | Ensure all paths start with `/` (e.g., `/users` not `users`) |
| `missing required field 'responses'` | Add at least one response (e.g., `200` or `default`) to each operation |
| `missing required 'type' field` | Add `type: string`, `type: object`, etc. to schema definitions |
| `Invalid $ref format` | Use `#/components/schemas/Name` for OAS 3.x or `#/definitions/Name` for OAS 2.0 |

### Parse Command

The parse command reads and analyzes an OpenAPI specification, displaying its structure and metadata.

**Basic Usage:**

```bash
# Parse a file
oastools parse openapi.yaml

# Parse with external reference resolution
oastools parse --resolve-refs openapi.yaml

# Parse with structure validation
oastools parse --validate-structure openapi.yaml
```

**Output Explanation:**

```
OpenAPI Specification Parser
============================

Specification: petstore.yaml
OAS Version: 3.0.3
Source Size: 15.2 KB
Paths: 8
Operations: 15
Schemas: 12
Load Time: 45ms

Document Type: OpenAPI 3.x
Title: Petstore API
Description: A sample API for a pet store
Version: 1.0.0
Servers: 2
Paths: 8
```

### Convert Command

The convert command transforms OpenAPI specifications between different versions.

**Basic Usage:**

```bash
# Convert OAS 2.0 (Swagger) to OAS 3.0.3
oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml

# Convert from URL
oastools convert -t 3.0.3 https://example.com/swagger.yaml -o openapi.yaml

# Convert OAS 3.x back to OAS 2.0
oastools convert -t 2.0 openapi.yaml -o swagger.yaml

# Strict mode (fail on any conversion issues)
oastools convert --strict -t 3.0.3 swagger.yaml -o openapi.yaml

# Suppress info messages
oastools convert --no-warnings -t 3.0.3 swagger.yaml -o openapi.yaml
```

**Understanding Conversion Issues:**

```
Conversion Issues (3):
  [INFO] servers: Converted host 'api.example.com' with basePath '/v1' to server URL 'https://api.example.com/v1'
  [WARNING] parameters.filter.allowEmptyValue: OAS 3.x does not support allowEmptyValue for query parameters; value dropped
  [CRITICAL] paths./webhook: Webhooks cannot be converted to OAS 2.0; webhook removed from output
```

| Severity | Meaning | Action |
|----------|---------|--------|
| INFO | Conversion choice or transformation | Review for correctness |
| WARNING | Lossy conversion, some data may be lost | Verify output meets requirements |
| CRITICAL | Feature cannot be converted | Manual intervention required |

**OAS 2.0 → 3.x Conversion Notes:**

- `host`, `basePath`, `schemes` → `servers` array
- `definitions` → `components.schemas`
- `parameters` → `components.parameters`
- `responses` → `components.responses`
- `securityDefinitions` → `components.securitySchemes`
- `consumes`/`produces` → `requestBody.content` / `responses.*.content`
- Body parameters → `requestBody` objects

**OAS 3.x → 2.0 Conversion Notes:**

These features cannot be converted:
- Webhooks (OAS 3.1+)
- Callbacks
- Links
- TRACE HTTP method
- Cookie parameters (partial support)
- Multiple servers (only first is used)

### Join Command

The join command merges multiple OpenAPI specifications into a single document.

**Basic Usage:**

```bash
# Join two specifications
oastools join -o merged.yaml base.yaml extensions.yaml

# Join multiple specifications
oastools join -o api.yaml users.yaml posts.yaml comments.yaml
```

**Collision Strategies:**

```bash
# Keep first value on collision (default behavior)
oastools join --path-strategy accept-left -o merged.yaml base.yaml ext.yaml

# Keep last value on collision (overwrite)
oastools join --path-strategy accept-right -o merged.yaml base.yaml ext.yaml

# Fail on any collision
oastools join --path-strategy fail -o merged.yaml base.yaml ext.yaml

# Fail only on path collisions, allow schema merging
oastools join --path-strategy fail-on-paths -o merged.yaml base.yaml ext.yaml
```

**Per-Component Strategies:**

```bash
# Different strategies for different component types
oastools join \
  --path-strategy fail \
  --schema-strategy accept-left \
  --component-strategy accept-right \
  -o merged.yaml base.yaml ext.yaml
```

**Array Handling:**

```bash
# Don't merge arrays (servers, security, tags)
oastools join --no-merge-arrays -o merged.yaml base.yaml ext.yaml

# Don't deduplicate tags
oastools join --no-dedup-tags -o merged.yaml base.yaml ext.yaml
```

**Example Scenario: Merging Microservice APIs**

```bash
# Combine APIs from multiple microservices
oastools join \
  --path-strategy fail \
  --schema-strategy accept-left \
  -o gateway-api.yaml \
  users-service/openapi.yaml \
  orders-service/openapi.yaml \
  products-service/openapi.yaml
```

### Diff Command

The diff command compares two OpenAPI specifications and reports differences.

**Basic Usage:**

```bash
# Simple diff (all changes)
oastools diff api-v1.yaml api-v2.yaml

# Breaking change detection
oastools diff --breaking api-v1.yaml api-v2.yaml

# Exclude informational changes
oastools diff --breaking --no-info api-v1.yaml api-v2.yaml

# Compare from URLs
oastools diff https://example.com/api/v1.yaml https://example.com/api/v2.yaml
```

**Understanding Breaking Change Severity:**

| Severity | Impact | Examples |
|----------|--------|----------|
| CRITICAL | API consumers WILL break | Removed endpoints, removed operations |
| ERROR | API consumers MAY break | Type changes, new required parameters |
| WARNING | Consumers SHOULD be aware | Deprecated operations, new optional fields |
| INFO | Non-breaking changes | Added endpoints, improved documentation |

**Example Output:**

```
OpenAPI Specification Diff
==========================

Source: api-v1.yaml (3.0.3)
Target: api-v2.yaml (3.0.3)
Total Time: 125ms

Endpoint Changes (2):
  [CRITICAL] /users/{id}: Endpoint removed
  [INFO] /posts: Endpoint added

Operation Changes (1):
  [WARNING] GET /users: Operation deprecated

Parameter Changes (2):
  [ERROR] GET /users: Required parameter 'limit' added
  [INFO] GET /users: Optional parameter 'filter' added

Summary:
  Total changes: 5
  ⚠️  Breaking changes: 2
  Warnings: 1
  Info: 2
```

**Exit Codes:**

| Code | Meaning |
|------|---------|
| 0 | No differences (or no breaking changes in `--breaking` mode) |
| 1 | Differences found (or breaking changes in `--breaking` mode) |

### Generate Command

The generate command creates idiomatic Go code for API clients and server stubs from an OpenAPI specification.

**Basic Usage:**

```bash
# Generate client code
oastools generate --client -o ./client -p petstore openapi.yaml

# Generate server interface
oastools generate --server -o ./server -p petstore openapi.yaml

# Generate both client and server
oastools generate --client --server -o ./generated -p myapi openapi.yaml

# Generate types only
oastools generate --types -o ./models -p models openapi.yaml

# From a URL
oastools generate --client -o ./generated https://example.com/api/openapi.yaml
```

**Options:**

```bash
-o, --output string         Output directory for generated files (required)
-p, --package string        Go package name for generated code (default: "api")
--client                    Generate HTTP client code
--server                    Generate server interface code
--types                     Generate type definitions from schemas (default: true)
--no-pointers              Don't use pointer types for optional fields
--no-validation            Don't include validation tags in generated code
--strict                   Fail on any generation issues (even warnings)
--no-warnings              Suppress warning and info messages
```

**Generated Files:**

The command produces:
- `types.go` - Model structs from schema definitions
- `client.go` - HTTP client with methods for each operation (when `--client` is used)
- `server.go` - Server interface for implementing endpoints (when `--server` is used)

**Understanding Output:**

```
Generating code from OpenAPI specification...
==============================

File: openapi.yaml
Version: 3.0.3
Package: petstore

Generated Files:
  types.go (8.2 KB)
  client.go (12.5 KB)
  server.go (5.8 KB)

Statistics:
  Types: 15
  Operations: 8

Issues (3):
  INFO [paths./pets.post]: Consider adding a requestBody description
  WARNING [paths./pets/{id}.get]: Parameter 'limit' should have a description
  INFO [components.schemas.Pet]: Consider adding examples

Generation complete in 245ms
```

## Library Usage

### Parser Package

The parser package provides parsing for OpenAPI Specification documents.

**Basic Parsing:**

```go
import "github.com/erraggy/oastools/parser"

// Using functional options (recommended for simple cases)
result, err := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithValidateStructure(true),
)
if err != nil {
    log.Fatal(err)
}

// Access parsed document
switch doc := result.Document.(type) {
case *parser.OAS2Document:
    fmt.Printf("Swagger %s: %s\n", doc.Swagger, doc.Info.Title)
case *parser.OAS3Document:
    fmt.Printf("OpenAPI %s: %s\n", doc.OpenAPI, doc.Info.Title)
}
```

**Reusable Parser Instance:**

```go
// Create a reusable parser for processing multiple files
p := parser.New()
p.ResolveRefs = false
p.ValidateStructure = true

// Process multiple files with same configuration
result1, _ := p.Parse("api1.yaml")
result2, _ := p.Parse("api2.yaml")
result3, _ := p.Parse("api3.yaml")
```

**Parsing from Different Sources:**

```go
// From file
result, err := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
)

// From URL
result, err := parser.ParseWithOptions(
    parser.WithFilePath("https://example.com/openapi.yaml"),
)

// From bytes
yamlContent := []byte(`openapi: "3.0.0"
info:
  title: My API
  version: "1.0"
paths: {}`)
result, err := parser.ParseWithOptions(
    parser.WithBytes(yamlContent),
)

// From io.Reader
file, _ := os.Open("openapi.yaml")
result, err := parser.ParseWithOptions(
    parser.WithReader(file),
)
```

**Working with External References:**

```go
// Enable reference resolution
result, err := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithResolveRefs(true),
)

// Security: References are restricted to base directory and subdirectories
// HTTP(S) references are NOT supported for security reasons
```

### Validator Package

The validator package provides validation for OpenAPI Specification documents.

**Basic Validation:**

```go
import "github.com/erraggy/oastools/validator"

// Validate with functional options
result, err := validator.ValidateWithOptions(
    validator.WithFilePath("openapi.yaml"),
    validator.WithIncludeWarnings(true),
)
if err != nil {
    log.Fatal(err)
}

if !result.Valid {
    fmt.Printf("Validation failed with %d errors\n", result.ErrorCount)
    for _, err := range result.Errors {
        fmt.Printf("  %s: %s\n", err.Path, err.Message)
    }
}
```

**Strict Mode:**

```go
// Strict mode treats warnings as errors
result, err := validator.ValidateWithOptions(
    validator.WithFilePath("openapi.yaml"),
    validator.WithIncludeWarnings(true),
    validator.WithStrictMode(true),
)

// In strict mode, result.Valid will be false if there are warnings
```

**Validating Pre-Parsed Documents:**

```go
// Parse once, validate multiple times (30x faster)
parseResult, err := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithValidateStructure(true),
)

result, err := validator.ValidateWithOptions(
    validator.WithParsed(*parseResult),
    validator.WithIncludeWarnings(true),
)
```

**Processing Validation Errors:**

```go
for _, err := range result.Errors {
    fmt.Printf("Path: %s\n", err.Path)
    fmt.Printf("Message: %s\n", err.Message)
    fmt.Printf("Severity: %s\n", err.Severity)
    if err.SpecRef != "" {
        fmt.Printf("Spec Reference: %s\n", err.SpecRef)
    }
}
```

### Converter Package

The converter package provides version conversion for OpenAPI Specification documents.

**Basic Conversion:**

```go
import "github.com/erraggy/oastools/converter"

// Convert OAS 2.0 to 3.0.3
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("swagger.yaml"),
    converter.WithTargetVersion("3.0.3"),
)
if err != nil {
    log.Fatal(err)
}

// Check for critical issues
if result.HasCriticalIssues() {
    fmt.Printf("Conversion had %d critical issues\n", result.CriticalCount)
    for _, issue := range result.Issues {
        if issue.Severity == converter.SeverityCritical {
            fmt.Printf("  [CRITICAL] %s: %s\n", issue.Path, issue.Message)
        }
    }
}
```

**Handling Conversion Issues:**

```go
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("swagger.yaml"),
    converter.WithTargetVersion("3.0.3"),
    converter.WithIncludeInfo(true), // Include informational messages
)

// Process issues by severity
for _, issue := range result.Issues {
    switch issue.Severity {
    case converter.SeverityCritical:
        fmt.Printf("CRITICAL: %s - %s\n", issue.Path, issue.Message)
        if issue.Context != "" {
            fmt.Printf("  Context: %s\n", issue.Context)
        }
    case converter.SeverityWarning:
        fmt.Printf("WARNING: %s - %s\n", issue.Path, issue.Message)
    case converter.SeverityInfo:
        fmt.Printf("INFO: %s - %s\n", issue.Path, issue.Message)
    }
}
```

**Writing Converted Output:**

```go
import (
    "os"
    "encoding/json"
    "gopkg.in/yaml.v3"
)

result, err := converter.ConvertWithOptions(
    converter.WithFilePath("swagger.yaml"),
    converter.WithTargetVersion("3.0.3"),
)

// Write as YAML (preserves input format by default)
var output []byte
if result.SourceFormat == parser.SourceFormatJSON {
    output, _ = json.MarshalIndent(result.Document, "", "  ")
} else {
    output, _ = yaml.Marshal(result.Document)
}

os.WriteFile("openapi.yaml", output, 0600)
```

### Joiner Package

The joiner package provides joining for multiple OpenAPI Specification documents.

**Basic Joining:**

```go
import "github.com/erraggy/oastools/joiner"

// Join with default configuration
result, err := joiner.JoinWithOptions(
    joiner.WithFilePaths([]string{"base.yaml", "extension.yaml"}),
)
if err != nil {
    log.Fatal(err)
}

// Write result to file
joiner.WriteResult(result, "merged.yaml")
```

**Custom Collision Strategies:**

```go
// Different strategies for different component types
config := joiner.JoinerConfig{
    DefaultStrategy:   joiner.StrategyFailOnCollision,
    PathStrategy:      joiner.StrategyFailOnPaths,      // Fail on path collisions
    SchemaStrategy:    joiner.StrategyAcceptLeft,       // Keep first schema
    ComponentStrategy: joiner.StrategyAcceptRight,      // Keep last component
    DeduplicateTags:   true,
    MergeArrays:       true,
}

result, err := joiner.JoinWithOptions(
    joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
    joiner.WithConfig(config),
)
```

**Joining Pre-Parsed Documents (154x faster):**

```go
// Parse documents once
docs := make([]parser.ParseResult, 0)
for _, path := range []string{"api1.yaml", "api2.yaml", "api3.yaml"} {
    result, err := parser.ParseWithOptions(
        parser.WithFilePath(path),
        parser.WithValidateStructure(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    docs = append(docs, *result)
}

// Join parsed documents
config := joiner.DefaultConfig()
j := joiner.New(config)
result, err := j.JoinParsed(docs)
```

**Processing Join Warnings:**

```go
result, err := joiner.JoinWithOptions(
    joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
    joiner.WithConfig(joiner.DefaultConfig()),
)

if len(result.Warnings) > 0 {
    fmt.Printf("Join completed with %d warnings:\n", len(result.Warnings))
    for _, warning := range result.Warnings {
        fmt.Printf("  - %s\n", warning)
    }
}

if result.CollisionCount > 0 {
    fmt.Printf("Resolved %d collisions\n", result.CollisionCount)
}
```

### Differ Package

The differ package provides OpenAPI specification comparison and breaking change detection.

**Basic Diff:**

```go
import "github.com/erraggy/oastools/differ"

// Simple diff
result, err := differ.DiffWithOptions(
    differ.WithSourceFilePath("api-v1.yaml"),
    differ.WithTargetFilePath("api-v2.yaml"),
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d changes\n", len(result.Changes))
for _, change := range result.Changes {
    fmt.Println(change.String())
}
```

**Breaking Change Detection:**

```go
// Enable breaking change detection
result, err := differ.DiffWithOptions(
    differ.WithSourceFilePath("api-v1.yaml"),
    differ.WithTargetFilePath("api-v2.yaml"),
    differ.WithMode(differ.ModeBreaking),
    differ.WithIncludeInfo(true),
)

if result.HasBreakingChanges {
    fmt.Printf("⚠️  Found %d breaking changes!\n", result.BreakingCount)
}

fmt.Printf("Summary: %d breaking, %d warnings, %d info\n",
    result.BreakingCount, result.WarningCount, result.InfoCount)
```

**Diffing Pre-Parsed Documents (81x faster):**

```go
// Parse documents once
source, _ := parser.ParseWithOptions(
    parser.WithFilePath("api-v1.yaml"),
    parser.WithValidateStructure(true),
)
target, _ := parser.ParseWithOptions(
    parser.WithFilePath("api-v2.yaml"),
    parser.WithValidateStructure(true),
)

// Compare parsed documents
result, err := differ.DiffWithOptions(
    differ.WithSourceParsed(*source),
    differ.WithTargetParsed(*target),
    differ.WithMode(differ.ModeBreaking),
)
```

**Grouping Changes by Category:**

```go
result, _ := differ.DiffWithOptions(
    differ.WithSourceFilePath("api-v1.yaml"),
    differ.WithTargetFilePath("api-v2.yaml"),
    differ.WithMode(differ.ModeBreaking),
)

// Group changes by category
categories := make(map[differ.ChangeCategory][]differ.Change)
for _, change := range result.Changes {
    categories[change.Category] = append(categories[change.Category], change)
}

// Process each category
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
            fmt.Printf("  %s\n", change.String())
        }
    }
}
```

### Generator Package

The generator package creates idiomatic Go code for API clients and server stubs from OpenAPI specifications.

**Basic Code Generation:**

```go
import "github.com/erraggy/oastools/generator"

// Generate client and server code
result, err := generator.GenerateWithOptions(
    generator.WithFilePath("openapi.yaml"),
    generator.WithPackageName("petstore"),
    generator.WithClient(true),
    generator.WithServer(true),
)
if err != nil {
    log.Fatal(err)
}

// Write generated files to output directory
if err := result.WriteFiles("./generated"); err != nil {
    log.Fatal(err)
}

fmt.Printf("Generated %d files\n", len(result.Files))
fmt.Printf("Types: %d, Operations: %d\n", result.GeneratedTypes, result.GeneratedOperations)
```

**Types-Only Generation:**

```go
// Generate only type definitions from schemas
result, err := generator.GenerateWithOptions(
    generator.WithFilePath("openapi.yaml"),
    generator.WithPackageName("models"),
    generator.WithTypes(true),
    generator.WithClient(false),
    generator.WithServer(false),
)
```

**Configuration Options:**

```go
g := generator.New()
g.PackageName = "api"
g.GenerateClient = true
g.GenerateServer = true
g.GenerateTypes = true      // Always true when client or server enabled
g.UsePointers = true        // Use pointers for optional fields
g.IncludeValidation = true  // Add validation tags
g.StrictMode = false        // Fail on generation issues
g.IncludeInfo = true        // Include info messages

result, err := g.Generate("openapi.yaml")
if err != nil {
    log.Fatal(err)
}

// Access generated files
for _, file := range result.Files {
    fmt.Printf("%s: %d bytes\n", file.Name, len(file.Content))
}

// Check for critical issues
if result.HasCriticalIssues() {
    fmt.Printf("Warning: %d critical issue(s)\n", result.CriticalCount)
}
```

**Handling Generation Issues:**

```go
result, err := generator.GenerateWithOptions(
    generator.WithFilePath("openapi.yaml"),
    generator.WithPackageName("api"),
    generator.WithClient(true),
)

if err != nil {
    log.Fatal(err)
}

// Process issues by severity
for _, issue := range result.Issues {
    switch issue.Severity {
    case generator.SeverityCritical:
        fmt.Printf("CRITICAL [%s]: %s\n", issue.Path, issue.Message)
    case generator.SeverityWarning:
        fmt.Printf("WARNING [%s]: %s\n", issue.Path, issue.Message)
    case generator.SeverityInfo:
        fmt.Printf("INFO [%s]: %s\n", issue.Path, issue.Message)
    }
}
```

### Builder Package

The builder package enables programmatic construction of OpenAPI specifications with reflection-based schema generation from Go types.

**Basic Construction:**

```go
import (
    "net/http"
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

// Define Go types for your API
type User struct {
    ID   int64  `json:"id" oas:"description=Unique user identifier"`
    Name string `json:"name" oas:"minLength=1,maxLength=100"`
    Email string `json:"email" oas:"format=email"`
}

type Error struct {
    Code    int    `json:"code" oas:"description=HTTP status code"`
    Message string `json:"message" oas:"description=Error message"`
}

// Build OAS 3.0 specification
spec := builder.New(parser.OASVersion300).
    SetTitle("User API").
    SetVersion("1.0.0").
    SetDescription("Simple user management API").
    AddOperation(http.MethodGet, "/users/{id}",
        builder.WithOperationID("getUserByID"),
        builder.WithOperationDescription("Get a user by ID"),
        builder.WithParameter("id", "path", "string", "User ID"),
        builder.WithResponse(http.StatusOK, User{}),
        builder.WithResponse(http.StatusNotFound, Error{}),
    ).
    AddOperation(http.MethodPost, "/users",
        builder.WithOperationID("createUser"),
        builder.WithRequestBody(User{}, "Create user request"),
        builder.WithResponse(http.StatusCreated, User{}),
        builder.WithResponse(http.StatusBadRequest, Error{}),
    )

// Build the document
doc, err := spec.BuildOAS3()
if err != nil {
    log.Fatal(err)
}

// Convert to YAML/JSON
data, _ := parser.ToYAML(doc)
fmt.Println(string(data))
```

**OAS Version Selection:**

```go
// Build for OAS 3.2.0 (latest)
spec := builder.New(parser.OASVersion320)

// Build for OAS 3.1.x
spec := builder.New(parser.OASVersion310)

// Build for OAS 3.0.x
spec := builder.New(parser.OASVersion300)

// Build for OAS 2.0 (Swagger)
spec := builder.New(parser.OASVersion20)
```

**Schema Generation from Go Types:**

```go
// Builder automatically generates JSON Schema from Go types
// Struct tags control schema properties:

type Product struct {
    ID          int64     `json:"id" oas:"description=Product ID"`
    Name        string    `json:"name" oas:"minLength=1,maxLength=255"`
    Price       float64   `json:"price" oas:"minimum=0,exclusiveMinimum=true"`
    Description string    `json:"description" oas:"maxLength=1000"`
    Tags        []string  `json:"tags" oas:"maxItems=10"`
    Active      bool      `json:"active" oas:"description=Is product active"`
    CreatedAt   time.Time `json:"created_at" oas:"format=date-time"`
}

// Builder generates appropriate OpenAPI 3.0 schema:
// - Infers types from struct field types
// - Applies constraints from oas tags
// - Handles nested structs, arrays, and time.Time
// - Generates descriptions and format hints
```

## Advanced Patterns

### Parse-Once Pattern

For workflows that process the same document multiple times, parse once and reuse the result:

```go
// Parse once
parseResult, err := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithValidateStructure(true),
)
if err != nil {
    log.Fatal(err)
}

// Validate (30x faster than validator.Validate)
valResult, _ := validator.ValidateWithOptions(
    validator.WithParsed(*parseResult),
    validator.WithIncludeWarnings(true),
)

// Convert (9x faster than converter.Convert)
convResult, _ := converter.ConvertWithOptions(
    converter.WithParsed(*parseResult),
    converter.WithTargetVersion("3.0.3"),
)

// Diff against another parsed document (81x faster than differ.Diff)
targetResult, _ := parser.ParseWithOptions(parser.WithFilePath("api-v2.yaml"))
diffResult, _ := differ.DiffWithOptions(
    differ.WithSourceParsed(*parseResult),
    differ.WithTargetParsed(*targetResult),
)
```

### Error Handling

**Handling Parse Errors:**

```go
result, err := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
)

if err != nil {
    // File not found, network error, or YAML/JSON syntax error
    log.Fatalf("Failed to parse: %v", err)
}

if len(result.Errors) > 0 {
    // Document parsed but has structural errors
    fmt.Printf("Document has %d validation errors:\n", len(result.Errors))
    for _, e := range result.Errors {
        fmt.Printf("  - %s\n", e)
    }
}
```

**Handling Conversion Errors:**

```go
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("openapi.yaml"),
    converter.WithTargetVersion("2.0"),
)

if err != nil {
    // Parse error or unsupported conversion
    log.Fatalf("Conversion failed: %v", err)
}

// Check for critical issues (features that couldn't be converted)
for _, issue := range result.Issues {
    if issue.Severity == converter.SeverityCritical {
        fmt.Printf("CRITICAL: %s\n", issue.Message)
        // Decide whether to proceed or abort based on your requirements
    }
}
```

**Handling Join Errors:**

```go
result, err := joiner.JoinWithOptions(
    joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
    joiner.WithConfig(joiner.DefaultConfig()),
)

if err != nil {
    switch {
    case strings.Contains(err.Error(), "collision"):
        fmt.Println("Path or schema collision detected")
        fmt.Println("Use a different collision strategy or resolve conflicts manually")
    case strings.Contains(err.Error(), "version mismatch"):
        fmt.Println("Cannot join OAS 2.0 with OAS 3.x documents")
    default:
        log.Fatalf("Join failed: %v", err)
    }
}
```

### Working with Different OAS Versions

**Detecting OAS Version:**

```go
result, _ := parser.ParseWithOptions(
    parser.WithFilePath("spec.yaml"),
)

fmt.Printf("Version: %s\n", result.Version)      // e.g., "3.0.3"
fmt.Printf("OAS Version: %d\n", result.OASVersion) // 2 or 3

switch doc := result.Document.(type) {
case *parser.OAS2Document:
    // Swagger 2.0 specific handling
    fmt.Printf("Host: %s\n", doc.Host)
    fmt.Printf("BasePath: %s\n", doc.BasePath)
case *parser.OAS3Document:
    // OpenAPI 3.x specific handling
    for _, server := range doc.Servers {
        fmt.Printf("Server: %s\n", server.URL)
    }
}
```

**Version-Specific Validation:**

```go
result, _ := validator.ValidateWithOptions(
    validator.WithFilePath("spec.yaml"),
    validator.WithIncludeWarnings(true),
)

// Validation automatically applies version-specific rules
// OAS 2.0: validates against Swagger 2.0 specification
// OAS 3.x: validates against OpenAPI 3.x specification
```

## Troubleshooting

### Common Issues

**"missing required field 'openapi' or 'swagger'"**

The document doesn't specify a version. Add either:
- `openapi: "3.0.0"` (or another 3.x version) for OpenAPI 3.x
- `swagger: "2.0"` for Swagger 2.0

**"$ref resolution failed: access denied"**

External references are restricted to the base directory and subdirectories. Ensure referenced files are within the allowed path.

**"cannot join OAS 2.0 with OAS 3.x documents"**

All documents in a join operation must be the same major version. Convert documents to a common version first.

**"collision at path '/users'"**

Two documents define the same path. Choose a collision strategy:
- `accept-left`: Keep the first definition
- `accept-right`: Keep the last definition
- `fail`: Abort the operation
- `fail-on-paths`: Allow schema collisions but fail on path collisions

### Performance Tips

1. **Use the Parse-Once pattern** for workflows that process the same document multiple times
2. **Disable reference resolution** when not needed: `parser.WithResolveRefs(false)`
3. **Disable validation during parsing** if you'll validate separately: `parser.WithValidateStructure(false)`
4. **Reuse instances** (Parser, Validator, Converter, Joiner, Differ) for processing multiple files

### Getting Help

- **API Documentation**: [pkg.go.dev/github.com/erraggy/oastools](https://pkg.go.dev/github.com/erraggy/oastools)
- **GitHub Issues**: [https://github.com/erraggy/oastools/issues](https://github.com/erraggy/oastools/issues)
- **Breaking Change Semantics**: See [breaking-changes.md](breaking-changes.md)
- **Performance Details**: See [benchmarks.md](../benchmarks.md)
