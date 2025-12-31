<a id="top"></a>

# Validator Package Deep Dive

!!! tip "Try it Online"
    No installation required! [Try the validator in your browser →](https://oastools.robnrob.com/validate)

## Table of Contents

- [Overview](#overview)
- [Key Concepts](#key-concepts)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [Validation Coverage](#validation-coverage)
- [Validation Result Structure](#validation-result-structure)
- [Configuration Reference](#configuration-reference)
- [Source Map Integration](#source-map-integration)
- [Best Practices](#best-practices)

---

The [`validator`](https://pkg.go.dev/github.com/erraggy/oastools/validator) package performs comprehensive validation of OpenAPI Specification documents against their declared version. It checks structural correctness, semantic constraints, format compliance, and best practices, producing detailed error reports with specification references for every issue found.

## Overview

Validation ensures your OpenAPI documents are correct before using them for code generation, documentation, or API gateway configuration. The validator catches issues ranging from missing required fields to invalid reference targets, malformed URLs, and inconsistent parameter declarations.

The validator supports OAS 2.0 through OAS 3.2.0, automatically adapting its rules to match the declared specification version. Each validation error includes a reference to the relevant section of the OpenAPI Specification, making it easy to understand why something is flagged and how to fix it.

[↑ Back to top](#top)

## Key Concepts

### Validation vs Parsing

Understanding the distinction between parsing and validation is important. The parser converts YAML or JSON into structured Go types, performing basic syntax checking but minimal semantic validation. The validator then examines the parsed structure for specification compliance.

This separation allows you to parse a document (which might have validation errors) to understand its structure, then validate it to identify issues. It also enables the 30x performance improvement when using `ValidateParsed` on pre-parsed documents.

### Severity Levels

Validation issues are categorized by severity, helping you prioritize fixes and configure appropriate behavior for your workflow.

**SeverityError** indicates specification violations that make the document invalid according to the OpenAPI Specification. These must be fixed for the document to be compliant.

**SeverityWarning** indicates best practice violations or recommendations that don't prevent the document from being technically valid but may cause issues with tooling or API consumers. Examples include operations without descriptions, trailing slashes in paths, or deprecated patterns.

**SeverityInfo** indicates informational messages that may be useful for debugging or understanding validator behavior but don't require action.

**SeverityCritical** indicates severe issues that prevent further processing. These are rare and typically indicate fundamental document problems.

### Strict Mode

By default, the validator reports errors separately from warnings, and a document is considered valid if it has no errors (warnings are allowed). Strict mode changes this behavior, treating warnings as errors and requiring the document to be warning-free to be considered valid.

Use strict mode in CI/CD pipelines where you want to enforce best practices, not just specification compliance.

[↑ Back to top](#top)

## API Styles

See also: [Basic example](https://pkg.go.dev/github.com/erraggy/oastools/validator#example-Validator_Validate), [Strict mode example](https://pkg.go.dev/github.com/erraggy/oastools/validator#example-Validator_Validate_strictMode), [Custom validation example](https://pkg.go.dev/github.com/erraggy/oastools/validator#example-package-CustomValidation) on pkg.go.dev

### Functional Options API

Best for one-off validations with inline configuration:

```go
result, err := validator.ValidateWithOptions(
    validator.WithFilePath("openapi.yaml"),
    validator.WithIncludeWarnings(true),
    validator.WithStrictMode(false),
)
```

### Struct-Based API

Best for validating multiple documents with consistent configuration:

```go
v := validator.New()
v.IncludeWarnings = true
v.StrictMode = true

result1, _ := v.Validate("api1.yaml")
result2, _ := v.Validate("api2.yaml")
```

[↑ Back to top](#top)

## Practical Examples

### Basic Validation

The simplest use case validates a single specification file:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/erraggy/oastools/validator"
)

func main() {
    result, err := validator.ValidateWithOptions(
        validator.WithFilePath("openapi.yaml"),
        validator.WithIncludeWarnings(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("OAS Version: %s\n", result.Version)
    fmt.Printf("Valid: %v\n", result.Valid)
    fmt.Printf("Errors: %d\n", result.ErrorCount)
    fmt.Printf("Warnings: %d\n", result.WarningCount)
    
    if !result.Valid {
        fmt.Println("\nValidation Errors:")
        for _, e := range result.Errors {
            fmt.Printf("  [%s] %s: %s\n", e.Severity, e.Path, e.Message)
            if e.SpecRef != "" {
                fmt.Printf("       Spec: %s\n", e.SpecRef)
            }
        }
    }
    
    if result.WarningCount > 0 {
        fmt.Println("\nWarnings:")
        for _, w := range result.Warnings {
            fmt.Printf("  [%s] %s: %s\n", w.Severity, w.Path, w.Message)
        }
    }
}
```

**Example Input (with issues):**
```yaml
openapi: 3.0.3
info:
  title: Test API
  # Missing required 'version' field
paths:
  /users/:           # Trailing slash (warning)
    get:
      # Missing operationId (warning)
      responses:
        '200':
          # Missing description (error)
  /users/{userId}:
    get:
      operationId: getUser
      # Missing declaration for 'userId' path parameter (error)
      responses:
        '200':
          description: Success
```

**Example Output:**
```
OAS Version: 3.0.3
Valid: false
Errors: 3
Warnings: 2

Validation Errors:
  [error] info.version: missing required field 'version'
       Spec: https://spec.openapis.org/oas/v3.0.3.html#info-object
  [error] paths./users/.get.responses.200: missing required field 'description'
       Spec: https://spec.openapis.org/oas/v3.0.3.html#response-object
  [error] paths./users/{userId}.get: Path template references parameter '{userId}' but it is not declared in parameters
       Spec: https://spec.openapis.org/oas/v3.0.3.html#path-item-object

Warnings:
  [warning] paths./users/: Path should not have a trailing slash
  [warning] paths./users/.get: Operation should have an operationId
```

### Strict Mode for CI/CD

Enforce both specification compliance and best practices:

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/erraggy/oastools/validator"
)

func main() {
    result, err := validator.ValidateWithOptions(
        validator.WithFilePath("openapi.yaml"),
        validator.WithIncludeWarnings(true),
        validator.WithStrictMode(true),  // Treat warnings as errors
    )
    if err != nil {
        log.Fatalf("Validation failed: %v", err)
    }
    
    if !result.Valid {
        fmt.Fprintf(os.Stderr, "❌ Validation failed\n")
        fmt.Fprintf(os.Stderr, "   Errors: %d\n", result.ErrorCount)
        fmt.Fprintf(os.Stderr, "   Warnings (treated as errors): %d\n", result.WarningCount)
        
        // Print all issues
        allIssues := append(result.Errors, result.Warnings...)
        for _, issue := range allIssues {
            fmt.Fprintf(os.Stderr, "\n[%s] %s\n", issue.Severity, issue.Path)
            fmt.Fprintf(os.Stderr, "  %s\n", issue.Message)
        }
        
        os.Exit(1)
    }
    
    fmt.Println("✓ Validation passed (including best practices)")
}
```

### High-Performance Validation with Pre-Parsed Documents

When integrating validation into a pipeline that also uses other oastools packages, parse once and reuse the result for 30x faster validation:

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/erraggy/oastools/parser"
    "github.com/erraggy/oastools/validator"
)

func main() {
    // Parse once
    parseResult, err := parser.ParseWithOptions(
        parser.WithFilePath("openapi.yaml"),
        parser.WithValidateStructure(true),
    )
    if err != nil {
        log.Fatalf("Parse failed: %v", err)
    }
    
    // Validate using pre-parsed document (30x faster)
    start := time.Now()
    valResult, err := validator.ValidateWithOptions(
        validator.WithParsed(*parseResult),
        validator.WithIncludeWarnings(true),
    )
    elapsed := time.Since(start)
    
    if err != nil {
        log.Fatalf("Validation failed: %v", err)
    }
    
    fmt.Printf("Validation completed in %v\n", elapsed)
    fmt.Printf("Valid: %v\n", valResult.Valid)
    
    // parseResult can now be used with other packages
    // (fixer, converter, joiner, generator) without re-parsing
}
```

### Validating Multiple Documents

For batch validation scenarios:

```go
package main

import (
    "fmt"
    "log"
    "path/filepath"
    
    "github.com/erraggy/oastools/validator"
)

func main() {
    // Create reusable validator
    v := validator.New()
    v.IncludeWarnings = true
    v.StrictMode = false
    
    // Find all OpenAPI files
    files, err := filepath.Glob("specs/*.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    var failed []string
    
    for _, file := range files {
        result, err := v.Validate(file)
        if err != nil {
            log.Printf("Error validating %s: %v", file, err)
            failed = append(failed, file)
            continue
        }
        
        if result.Valid {
            fmt.Printf("✓ %s (warnings: %d)\n", file, result.WarningCount)
        } else {
            fmt.Printf("✗ %s (errors: %d, warnings: %d)\n", 
                file, result.ErrorCount, result.WarningCount)
            failed = append(failed, file)
        }
    }
    
    if len(failed) > 0 {
        fmt.Printf("\n%d/%d files failed validation\n", len(failed), len(files))
    } else {
        fmt.Printf("\nAll %d files passed validation\n", len(files))
    }
}
```

### Processing Validation Errors Programmatically

Extract and categorize validation issues for custom reporting:

```go
package main

import (
    "fmt"
    "log"
    "strings"
    
    "github.com/erraggy/oastools/validator"
)

func main() {
    result, err := validator.ValidateWithOptions(
        validator.WithFilePath("openapi.yaml"),
        validator.WithIncludeWarnings(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Categorize errors by location
    bySection := make(map[string][]validator.ValidationError)
    
    for _, e := range result.Errors {
        // Extract top-level section from path
        section := "document"
        if parts := strings.SplitN(e.Path, ".", 2); len(parts) > 0 {
            section = parts[0]
        }
        bySection[section] = append(bySection[section], e)
    }
    
    // Report by section
    for section, errors := range bySection {
        fmt.Printf("\n%s issues (%d):\n", strings.ToUpper(section), len(errors))
        for _, e := range errors {
            fmt.Printf("  • %s: %s\n", e.Path, e.Message)
        }
    }
    
    // Identify patterns
    missingFields := 0
    invalidRefs := 0
    formatErrors := 0
    
    for _, e := range result.Errors {
        switch {
        case strings.Contains(e.Message, "missing required"):
            missingFields++
        case strings.Contains(e.Message, "does not resolve"):
            invalidRefs++
        case strings.Contains(e.Message, "Invalid"):
            formatErrors++
        }
    }
    
    fmt.Printf("\nError patterns:\n")
    fmt.Printf("  Missing required fields: %d\n", missingFields)
    fmt.Printf("  Invalid references: %d\n", invalidRefs)
    fmt.Printf("  Format errors: %d\n", formatErrors)
}
```

### Validation with Custom User Agent

When validating specifications from URLs, set a custom User-Agent:

```go
package main

import (
    "log"
    
    "github.com/erraggy/oastools/validator"
)

func main() {
    result, err := validator.ValidateWithOptions(
        validator.WithFilePath("https://api.example.com/openapi.yaml"),
        validator.WithUserAgent("MyValidationTool/1.0"),
        validator.WithIncludeWarnings(true),
    )
    if err != nil {
        log.Fatalf("Validation failed: %v", err)
    }
    
    log.Printf("Valid: %v, Load time: %v", result.Valid, result.LoadTime)
}
```

### Integration with Fixer

A common pattern validates, identifies fixable issues, applies fixes, and re-validates:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/erraggy/oastools/fixer"
    "github.com/erraggy/oastools/parser"
    "github.com/erraggy/oastools/validator"
)

func main() {
    // Parse once
    parseResult, err := parser.ParseWithOptions(
        parser.WithFilePath("openapi.yaml"),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Initial validation
    valResult, _ := validator.ValidateWithOptions(
        validator.WithParsed(*parseResult),
    )
    
    fmt.Printf("Initial validation: %d errors\n", valResult.ErrorCount)
    
    if !valResult.Valid {
        // Attempt automatic fixes
        fixResult, err := fixer.FixWithOptions(
            fixer.WithParsed(*parseResult),
            fixer.WithInferTypes(true),
        )
        if err != nil {
            log.Fatalf("Fixer failed: %v", err)
        }
        
        if fixResult.HasFixes() {
            fmt.Printf("Applied %d fixes\n", fixResult.FixCount)
            
            // Re-validate fixed document
            // Create a new ParseResult from the fixed document
            fixedParse := &parser.ParseResult{
                Document:   fixResult.Document,
                Version:    parseResult.Version,
                OASVersion: parseResult.OASVersion,
            }
            
            valResult, _ = validator.ValidateWithOptions(
                validator.WithParsed(*fixedParse),
            )
            
            fmt.Printf("After fixes: %d errors\n", valResult.ErrorCount)
        }
    }
    
    if valResult.Valid {
        fmt.Println("✓ Document is valid")
    } else {
        fmt.Println("✗ Manual fixes required:")
        for _, e := range valResult.Errors {
            fmt.Printf("  %s: %s\n", e.Path, e.Message)
        }
    }
}
```

[↑ Back to top](#top)

## Validation Coverage

The validator performs comprehensive checks across all document sections. Understanding what's validated helps you interpret results and know what to expect.

### Info Object Validation

The validator checks that required fields are present and formats are correct.

**Checked Fields:**
- `title` (required)
- `version` (required)
- `termsOfService` (URL format when present)
- `contact.url` (URL format when present)
- `contact.email` (email format when present)
- `license.url` (URL format when present)

### Path Validation

Paths receive thorough structural and semantic validation.

**Path Template Checks:**
- Properly formed parameter placeholders (`{paramName}`)
- No unclosed braces
- No reserved characters in parameter names
- No consecutive slashes (`//`)
- Trailing slash warnings (best practice)

**Path Parameter Consistency:**
- Every `{param}` in the path template must have a corresponding parameter definition with `in: path`
- Path parameters must be marked as `required: true`
- Declared path parameters should be used in the template

### Operation Validation

Each operation (GET, POST, PUT, etc.) is validated for completeness and correctness.

**Required Checks:**
- `responses` object must be present
- At least one response code or `default` response
- Each response must have a `description`

**Best Practice Warnings:**
- Missing `operationId`
- Missing `summary` or `description`
- Duplicate `operationId` values across operations

### Parameter Validation

Parameters are checked for correct structure and usage.

**Structural Checks:**
- `name` is required
- `in` is required and must be one of: path, query, header, cookie
- Path parameters must have `required: true`
- Schema or content must be defined (OAS 3.x)

**Format Checks:**
- Valid parameter types and formats
- Consistent use of serialization styles

### Schema/Definition Validation

Component schemas undergo structural validation.

**Checked Items:**
- Valid `type` values when present
- Correct `format` for the specified `type`
- `$ref` targets must resolve to existing schemas
- Nested schema structures (allOf, oneOf, anyOf, properties)

**Reference Validation:**
- `$ref` uses correct path format for the OAS version
- Referenced schemas exist in components/definitions
- No circular references that would cause infinite loops

### Security Scheme Validation

Security definitions are validated for completeness.

**Checked by Type:**
- `apiKey`: `name` and `in` are required
- `http`: `scheme` is required
- `oauth2`: `flows` object with required flow properties
- `openIdConnect`: `openIdConnectUrl` is required and valid URL

### Server Validation (OAS 3.x)

Server objects are checked for correct structure.

**Checked Items:**
- `url` is required and valid format
- Server variables have `default` values
- Variable placeholders in URL match defined variables

[↑ Back to top](#top)

## Validation Result Structure

```go
type ValidationResult struct {
    // Valid is true if no errors were found (warnings allowed)
    Valid bool
    
    // Version is the detected OAS version string (e.g., "3.0.3")
    Version string
    
    // OASVersion is the enumerated version
    OASVersion parser.OASVersion
    
    // Errors contains all validation errors
    Errors []ValidationError
    
    // Warnings contains all validation warnings (if IncludeWarnings is true)
    Warnings []ValidationError
    
    // Counts
    ErrorCount   int
    WarningCount int
    
    // Performance metrics
    LoadTime   time.Duration
    SourceSize int64
    
    // Document statistics
    Stats parser.DocumentStats
}

type ValidationError struct {
    // Path is the JSON path to the issue location
    Path string
    
    // Message describes the issue
    Message string
    
    // Severity indicates the issue level
    Severity Severity
    
    // SpecRef is a URL to the relevant specification section
    SpecRef string
    
    // Field is the specific field that has the issue (optional)
    Field string
    
    // Value is the problematic value (optional)
    Value string
}
```

[↑ Back to top](#top)

## Configuration Reference

### Validator Fields

```go
type Validator struct {
    // IncludeWarnings determines whether to include best practice warnings
    IncludeWarnings bool  // Default: true
    
    // StrictMode treats warnings as errors
    StrictMode bool  // Default: false
    
    // UserAgent for HTTP requests when fetching URLs
    UserAgent string
}
```

### Available Options

| Option | Description |
|--------|-------------|
| `WithFilePath(string)` | Input file path or URL |
| `WithParsed(ParseResult)` | Pre-parsed document (30x faster) |
| `WithIncludeWarnings(bool)` | Include best practice warnings |
| `WithStrictMode(bool)` | Treat warnings as errors |
| `WithUserAgent(string)` | Custom User-Agent for HTTP requests |

[↑ Back to top](#top)

## Source Map Integration

When you need line numbers for IDE-friendly error reporting, enable source maps during parsing and pass them to the validator:

```go
parseResult, _ := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithSourceMap(true),
)
result, _ := validator.ValidateWithOptions(
    validator.WithParsed(*parseResult),
    validator.WithSourceMap(parseResult.SourceMap),
)
for _, err := range result.Errors {
    fmt.Printf("%s: %s\n", err.Location(), err.Message)
}
```

[Back to top](#top)

## Best Practices

**Always validate before using specifications** for code generation, documentation, or gateway configuration. Invalid specifications can produce incorrect or broken outputs.

**Use strict mode in CI/CD pipelines** to enforce both specification compliance and best practices from the start. It's easier to maintain quality than to fix accumulated issues later.

**Leverage the parse-once pattern** when combining validation with other operations. Parse once, then pass the `ParseResult` to validator, fixer, converter, and other packages for significant performance improvements.

**Include spec references in error reports** when building developer-facing tooling. The `SpecRef` field provides direct links to documentation that helps developers understand and fix issues.

**Categorize issues by severity** in your reporting. Critical and Error issues must be fixed; Warnings should be addressed but don't block validity; Info messages are informational only.

**Consider validation as part of your API design workflow.** Validating specifications early catches issues before they propagate to generated code, documentation, or runtime systems.

**Use warnings as guidance, not just rules.** The best practice warnings reflect common patterns that improve API usability and tooling compatibility. Understanding why something is flagged helps you make informed decisions about whether to address it.

---

## Learn More

For additional examples and complete API documentation:

- 📦 [API Reference on pkg.go.dev](https://pkg.go.dev/github.com/erraggy/oastools/validator) - Complete API documentation with all examples
- ✅ [Basic example](https://pkg.go.dev/github.com/erraggy/oastools/validator#example-Validator_Validate) - Validate an OpenAPI specification
- 🔒 [Strict mode example](https://pkg.go.dev/github.com/erraggy/oastools/validator#example-Validator_Validate_strictMode) - Enforce best practices
- ⚙️ [Custom validation example](https://pkg.go.dev/github.com/erraggy/oastools/validator#example-package-CustomValidation) - Process validation issues by severity
