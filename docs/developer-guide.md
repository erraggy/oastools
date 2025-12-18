# Developer Guide

This guide provides comprehensive documentation for developers using oastools as both a library and a command-line tool.

## Table of Contents

- [Installation](#installation)
- [CLI Usage](#cli-usage)
  - [Quick Reference](#quick-reference)
  - [Pipeline Support](#pipeline-support)
- [Library Usage](#library-usage)
  - [Parser Package](#parser-package)
  - [Validator Package](#validator-package)
  - [Fixer Package](#fixer-package)
  - [Converter Package](#converter-package)
  - [Joiner Package](#joiner-package)
  - [Differ Package](#differ-package)
  - [Generator Package](#generator-package)
  - [Builder Package](#builder-package)
  - [Overlay Package](#overlay-package)
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

This section provides quick examples of common CLI operations. For complete documentation including all flags, options, and output formats, see the **[CLI Reference](cli-reference.md)**.

### Quick Reference

```bash
# Validate
oastools validate openapi.yaml
oastools validate --strict --format json openapi.yaml

# Fix common errors
oastools fix openapi.yaml -o fixed.yaml
oastools fix --infer openapi.yaml -o fixed.yaml  # Type inference

# Parse and inspect
oastools parse openapi.yaml
oastools parse --resolve-refs openapi.yaml

# Convert between versions
oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml
oastools convert -t 2.0 openapi.yaml -o swagger.yaml

# Join multiple specs
oastools join -o merged.yaml base.yaml ext.yaml
oastools join --path-strategy accept-left -o merged.yaml base.yaml ext.yaml

# Compare specs (breaking change detection)
oastools diff api-v1.yaml api-v2.yaml
oastools diff --breaking api-v1.yaml api-v2.yaml

# Generate Go code
oastools generate --client -o ./client -p petstore openapi.yaml
oastools generate --server -o ./server -p petstore openapi.yaml

# Apply overlay transformations
oastools overlay apply --spec openapi.yaml --overlay changes.yaml -o result.yaml
oastools overlay validate overlay.yaml
oastools overlay apply --spec openapi.yaml --overlay changes.yaml --dry-run
```

### Pipeline Support

All commands support stdin (`-`) and quiet mode (`-q`) for shell pipelines:

```bash
# Fix then validate
oastools fix api.yaml | oastools validate -q -

# Convert via pipeline
cat swagger.yaml | oastools convert -q -t 3.0.3 - > openapi.yaml

# Chain operations
curl -s https://example.com/swagger.yaml | oastools convert -q -t 3.0.3 - | oastools validate -q -
```

For detailed documentation on each command including all flags, output formats, exit codes, and examples, see the **[CLI Reference](cli-reference.md)**.

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

> 📚 **Deep Dive:** For comprehensive examples and advanced patterns, see the [Validator Deep Dive](packages/validator.md).

### Fixer Package

The fixer package automatically corrects common validation errors in OpenAPI Specification documents.

**Basic Fixing:**

```go
import "github.com/erraggy/oastools/fixer"

// Fix with functional options
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("openapi.yaml"),
    fixer.WithInferTypes(true),
)
if err != nil {
    log.Fatal(err)
}

if result.HasFixes() {
    fmt.Printf("Applied %d fixes\n", result.FixCount)
    for _, fix := range result.Fixes {
        fmt.Printf("  %s: %s\n", fix.Type, fix.Description)
    }
}
```

**Reusable Fixer Instance:**

```go
// Create a reusable fixer for processing multiple files
f := fixer.New()
f.InferTypes = true

// Process multiple files with same configuration
result1, _ := f.Fix("api1.yaml")
result2, _ := f.Fix("api2.yaml")
result3, _ := f.Fix("api3.yaml")
```

**Fixing Pre-Parsed Documents:**

```go
// Parse once, fix multiple times
parseResult, err := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithValidateStructure(true),
)

result, err := fixer.FixWithOptions(
    fixer.WithParsed(*parseResult),
    fixer.WithInferTypes(true),
)
```

**Processing Fix Results:**

```go
result, _ := fixer.FixWithOptions(
    fixer.WithFilePath("openapi.yaml"),
)

for _, fix := range result.Fixes {
    fmt.Printf("Type: %s\n", fix.Type)
    fmt.Printf("Path: %s\n", fix.Path)
    fmt.Printf("Description: %s\n", fix.Description)
}

// Access the fixed document
switch doc := result.Document.(type) {
case *parser.OAS2Document:
    // Work with fixed OAS 2.0 document
case *parser.OAS3Document:
    // Work with fixed OAS 3.x document
}
```

**Fixing Invalid Schema Names:**

> **Note:** Schema renaming is not enabled by default. You must use `WithEnabledFixes()` to opt-in.

```go
// Fix schemas with invalid names (e.g., Response[User])
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("openapi.yaml"),
    fixer.WithEnabledFixes(fixer.FixTypeRenamedGenericSchema), // Enable schema renaming
    fixer.WithGenericNaming(fixer.GenericNamingOf),            // Response[User] → ResponseOfUser
)

// Available naming strategies:
// - fixer.GenericNamingUnderscore: Response[User] → Response_User_
// - fixer.GenericNamingOf:         Response[User] → ResponseOfUser
// - fixer.GenericNamingFor:        Response[User] → ResponseForUser
// - fixer.GenericNamingFlattened:  Response[User] → ResponseUser
// - fixer.GenericNamingDot:        Response[User] → Response.User
```

**Pruning Unused Schemas:**

```go
// Remove unreferenced schemas
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("openapi.yaml"),
    fixer.WithEnabledFixes(fixer.FixTypePrunedUnusedSchema),
)

for _, fix := range result.Fixes {
    fmt.Printf("Removed unused schema: %s\n", fix.Before)
}
```

**Dry-Run Mode:**

```go
// Preview changes without applying
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("openapi.yaml"),
    fixer.WithDryRun(true),
)

fmt.Printf("Would apply %d fix(es)\n", result.FixCount)
for _, fix := range result.Fixes {
    fmt.Printf("  %s: %s\n", fix.Type, fix.Description)
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
    "go.yaml.in/yaml/v4"
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

**Semantic Deduplication:**

Semantic deduplication identifies structurally identical schemas across documents and consolidates them to reduce duplication:

```go
// Enable semantic deduplication to consolidate identical schemas
config := joiner.JoinerConfig{
    DefaultStrategy:       joiner.StrategyAcceptLeft,
    SemanticDeduplication: true,   // Enable schema deduplication
    EquivalenceMode:       "deep", // Use deep structural comparison
    DeduplicateTags:       true,
    MergeArrays:           true,
}

j := joiner.New(config)
result, err := j.Join([]string{"api1.yaml", "api2.yaml", "api3.yaml"})
if err != nil {
    log.Fatal(err)
}

// Semantic deduplication consolidates identical schemas:
// - Schemas with identical structure are detected via FNV-1a hashing
// - The alphabetically-first name becomes the canonical schema
// - All $ref references are automatically rewritten
// - Warnings indicate how many duplicates were consolidated
```

> 📚 **Deep Dive:** For comprehensive examples and advanced patterns, see the [Joiner Deep Dive](packages/joiner.md).

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

> 📚 **Deep Dive:** For comprehensive examples and advanced patterns, see the [Differ Deep Dive](packages/differ.md).

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

**Security Code Generation:**

```go
// Generate client with security helpers
result, err := generator.GenerateWithOptions(
    generator.WithFilePath("openapi.yaml"),
    generator.WithPackageName("api"),
    generator.WithClient(true),
    generator.WithSecurity(true),  // Generate security helpers
)

// The generated security_helpers.go contains ClientOption functions:
// - WithApiKeyAPIKey(key string) for apiKey (header) schemes
// - WithApiKeyAPIKeyQuery(key string) for apiKey (query) schemes
// - WithApiKeyAPIKeyCookie(key string) for apiKey (cookie) schemes
// - WithBasicAuthBasicAuth(username, password string) for HTTP basic auth
// - WithBearerTokenBearerToken(token string) for HTTP bearer auth
// - WithOAuth2OAuth2Token(token string) for OAuth2
// - WithOidcToken(token string) for OpenID Connect
```

**OAuth2 Flow Generation:**

```go
// Generate full OAuth2 client implementations
result, err := generator.GenerateWithOptions(
    generator.WithFilePath("openapi.yaml"),
    generator.WithPackageName("api"),
    generator.WithClient(true),
    generator.WithOAuth2Flows(true),
)

// Generated OAuth2 code includes:
// - {SchemeName}OAuth2Config struct
// - {SchemeName}OAuth2Client with flow methods
// - GetAuthorizationURL() for authorization code flow
// - ExchangeCode() to exchange codes for tokens
// - GeneratePKCEChallenge() for PKCE challenge generation
// - GetAuthorizationURLWithPKCE() for secure authorization with PKCE
// - ExchangeCodeWithPKCE() for token exchange with PKCE
// - GetClientCredentialsToken() for client credentials
// - RefreshToken() for token refresh
// - WithOAuth2AutoRefresh() ClientOption
```

**Credential Management:**

```go
// Generate credential provider interfaces
result, err := generator.GenerateWithOptions(
    generator.WithFilePath("openapi.yaml"),
    generator.WithPackageName("api"),
    generator.WithClient(true),
    generator.WithCredentialMgmt(true),
)

// Generated credential code includes:
// - CredentialProvider interface
// - MemoryCredentialProvider for testing
// - EnvCredentialProvider for environment variables
// - CredentialChain for fallback providers
// - WithCredentialProvider() ClientOption
```

**Security Enforcement (Server-Side):**

```go
// Generate security validation middleware
result, err := generator.GenerateWithOptions(
    generator.WithFilePath("openapi.yaml"),
    generator.WithPackageName("api"),
    generator.WithServer(true),
    generator.WithSecurityEnforce(true),
)

// Generated enforcement code includes:
// - SecurityRequirement struct
// - OperationSecurityRequirements map
// - SecurityValidator for validating requests
// - RequireSecurityMiddleware for enforcement
```

**OpenID Connect Discovery:**

```go
// Generate OIDC discovery client
result, err := generator.GenerateWithOptions(
    generator.WithFilePath("openapi.yaml"),
    generator.WithPackageName("api"),
    generator.WithClient(true),
    generator.WithOIDCDiscovery(true),
)

// Generated OIDC code includes:
// - OIDCConfiguration struct
// - OIDCDiscoveryClient for .well-known discovery
// - NewOAuth2ClientFromOIDC() helper
```

**File Splitting for Large APIs:**

```go
// Configure file splitting for large APIs
result, err := generator.GenerateWithOptions(
    generator.WithFilePath("large-api.yaml"),
    generator.WithPackageName("api"),
    generator.WithClient(true),
    generator.WithMaxLinesPerFile(2000),
    generator.WithMaxTypesPerFile(200),
    generator.WithMaxOperationsPerFile(100),
    generator.WithSplitByTag(true),
    generator.WithSplitByPathPrefix(true),
)

// Files will be split based on:
// 1. Operation tags (e.g., users_client.go, orders_client.go)
// 2. Path prefixes (e.g., api_v1_client.go, api_v2_client.go)
// 3. Shared types in types.go
```

**README Generation:**

```go
// Generate documentation with the code
result, err := generator.GenerateWithOptions(
    generator.WithFilePath("openapi.yaml"),
    generator.WithPackageName("api"),
    generator.WithClient(true),
    generator.WithReadme(true),
)

// README.md includes:
// - API overview and version
// - Generated file descriptions
// - Security configuration examples
// - Regeneration command
```

> 📚 **Deep Dive:** For comprehensive examples and advanced patterns, see the [Generator Deep Dive](packages/generator.md).

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

**Semantic Deduplication:**

Enable semantic deduplication to automatically consolidate structurally identical schemas:

```go
import (
    "net/http"
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

// Types that are structurally identical
type UserID struct {
    Value int64 `json:"value"`
}
type CustomerID struct {
    Value int64 `json:"value"`
}

// Enable semantic deduplication
spec := builder.New(parser.OASVersion320,
    builder.WithSemanticDeduplication(true),
).
    SetTitle("API").
    SetVersion("1.0.0").
    AddOperation(http.MethodGet, "/users/{id}",
        builder.WithResponse(http.StatusOK, UserID{}),
    ).
    AddOperation(http.MethodGet, "/customers/{id}",
        builder.WithResponse(http.StatusOK, CustomerID{}),
    )

doc, err := spec.BuildOAS3()
// Result: Only 1 schema instead of 2, with $refs automatically rewritten
```

> 📚 **Deep Dive:** For comprehensive examples and advanced patterns, see the [Builder Deep Dive](packages/builder.md).

### Overlay Package

The overlay package applies OpenAPI Overlay Specification v1.0.0 transformations to OpenAPI documents using JSONPath targeting.

**Basic Usage:**

```go
import "github.com/erraggy/oastools/overlay"

// Apply overlay with functional options
result, err := overlay.ApplyWithOptions(
    overlay.WithSpecFilePath("openapi.yaml"),
    overlay.WithOverlayFilePath("changes.yaml"),
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Applied %d actions\n", result.ActionsApplied)
```

**Creating Overlays Programmatically:**

```go
// Create an overlay to update API metadata
o := &overlay.Overlay{
    Version: "1.0.0",
    Info: overlay.Info{
        Title:   "Update API Title",
        Version: "1.0.0",
    },
    Actions: []overlay.Action{
        {
            Target: "$.info",
            Update: map[string]any{
                "title":         "Production API",
                "x-environment": "production",
            },
        },
        {
            Target: "$.paths[?@.x-internal==true]",
            Remove: true,
        },
    },
}

result, err := overlay.ApplyWithOptions(
    overlay.WithSpecFilePath("openapi.yaml"),
    overlay.WithOverlayParsed(o),
)
```

**Dry-Run Mode (Preview Changes):**

```go
// Preview changes without applying
dryResult, err := overlay.DryRunWithOptions(
    overlay.WithSpecFilePath("openapi.yaml"),
    overlay.WithOverlayFilePath("changes.yaml"),
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Would apply: %d actions\n", dryResult.WouldApply)
fmt.Printf("Would skip: %d actions\n", dryResult.WouldSkip)
for _, change := range dryResult.Changes {
    fmt.Printf("  - %s %d nodes at %s\n", change.Operation, change.MatchCount, change.Target)
}
```

**Validating Overlay Documents:**

```go
o, err := overlay.ParseOverlayFile("overlay.yaml")
if err != nil {
    log.Fatal(err)
}

errs := overlay.Validate(o)
if len(errs) > 0 {
    for _, err := range errs {
        fmt.Printf("Validation error: %s\n", err.Message)
    }
}
```

**Advanced JSONPath Targeting:**

```go
// The overlay package supports various JSONPath expressions:

// Recursive descent - find all descriptions at any depth
{Target: "$..description", Update: "Updated description"}

// Compound filters with AND
{Target: "$.paths[?@.deprecated==true && @.x-internal==true]", Remove: true}

// Compound filters with OR
{Target: "$.paths[?@.deprecated==true || @.x-obsolete==true]", Remove: true}

// Wildcard selectors
{Target: "$.paths.*.get", Update: map[string]any{"x-cached": true}}
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
- **Performance Details**: See [benchmarks.md](benchmarks.md)
