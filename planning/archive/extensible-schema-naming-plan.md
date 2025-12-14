# Extensible Schema Naming Implementation Plan

**Package:** `github.com/erraggy/oastools/builder`  
**Target Version:** v1.25.0  
**Estimated Complexity:** Medium-High  
**Estimated LOC:** ~1200-1500 (including tests and CLI)

## Executive Summary

This plan introduces an extensible schema naming system for the builder package, replacing the current hard-coded `package.TypeName` convention with a flexible, user-configurable approach. The implementation supports simple casing transformations (PascalCase, snake_case, kebab-case) through built-in strategies, and complex custom naming through Go text/template support.

## Current State Analysis

### Existing Implementation

The current `schemaName(t reflect.Type) string` method in `builder/reflect.go`:

1. Uses fixed format: `{package}.{TypeName}` (e.g., `models.User`)
2. Handles package conflicts by expanding to full path: `github.com_foo_models.User`
3. Sanitizes generic types: `Response[User]` → `Response_User_`
4. Names anonymous types as `AnonymousType`

### Limitations

The current approach lacks flexibility for:

- API-first design where schema names must match external specifications
- OpenAPI generators expecting specific naming conventions (e.g., camelCase for JSON Schema)
- Multi-language codegen requiring consistent casing across targets
- Integration with existing OpenAPI documents using different conventions

## Design Goals

1. **Backward Compatibility**: Default behavior remains `package.TypeName` format
2. **Progressive Complexity**: Simple use cases require minimal configuration
3. **Full Control**: Advanced users can define arbitrary naming logic via templates
4. **Type Safety**: Compile-time validation where possible, runtime validation for templates
5. **Performance**: Minimal overhead for default naming strategy

## API Design

### Option Functions

```go
// WithSchemaNaming sets a custom schema naming strategy.
// Use built-in strategies like SchemaNamingPascalCase or define custom ones.
func WithSchemaNaming(strategy SchemaNamingStrategy) BuilderOption

// WithSchemaNameTemplate sets a custom Go text/template for schema naming.
// Template receives SchemaNameContext with type metadata.
func WithSchemaNameTemplate(tmpl string) BuilderOption

// WithSchemaNameFunc sets a custom function for schema naming.
// Provides maximum flexibility for programmatic naming logic.
func WithSchemaNameFunc(fn SchemaNameFunc) BuilderOption

// WithGenericNaming sets the strategy for handling generic type names.
func WithGenericNaming(strategy GenericNamingStrategy) BuilderOption

// WithGenericNamingConfig provides fine-grained control over generic type naming.
func WithGenericNamingConfig(config GenericNamingConfig) BuilderOption
```

### Built-in Strategies

```go
type SchemaNamingStrategy int

const (
    // SchemaNamingDefault uses "package.TypeName" format (current behavior)
    SchemaNamingDefault SchemaNamingStrategy = iota
    
    // SchemaNamingPascalCase uses "PackageTypeName" format
    // Example: models.User → ModelsUser
    SchemaNamingPascalCase
    
    // SchemaNamingCamelCase uses "packageTypeName" format
    // Example: models.User → modelsUser
    SchemaNamingCamelCase
    
    // SchemaNamingSnakeCase uses "package_type_name" format
    // Example: models.User → models_user
    SchemaNamingSnakeCase
    
    // SchemaNamingKebabCase uses "package-type-name" format
    // Example: models.User → models-user
    SchemaNamingKebabCase
    
    // SchemaNamingTypeOnly uses just "TypeName" without package
    // Example: models.User → User
    // Warning: May cause conflicts with same-named types in different packages
    SchemaNamingTypeOnly
    
    // SchemaNamingFullPath uses full package path
    // Example: models.User → github.com_org_models_User
    SchemaNamingFullPath
)
```

### Generic Type Handling

Generic types require special handling because OpenAPI schema names must be valid URI fragments. The builder provides configurable strategies for how type parameters are represented in schema names.

```go
type GenericNamingStrategy int

const (
    // GenericNamingUnderscore replaces brackets with underscores (current behavior)
    // Example: Response[User] → Response_User_
    GenericNamingUnderscore GenericNamingStrategy = iota
    
    // GenericNamingOf uses "Of" separator between base type and parameters
    // Example: Response[User] → ResponseOfUser
    GenericNamingOf
    
    // GenericNamingFor uses "For" separator
    // Example: Response[User] → ResponseForUser
    GenericNamingFor
    
    // GenericNamingAngleBrackets uses angle brackets (URI-encoded in $ref)
    // Example: Response[User] → Response<User>
    // Note: Produces Response%3CUser%3E in $ref URIs
    GenericNamingAngleBrackets
    
    // GenericNamingFlattened removes brackets entirely
    // Example: Response[User] → ResponseUser
    GenericNamingFlattened
    
    // GenericNamingExpanded includes full type parameter chain
    // Example: Response[List[User]] → Response_List_User
    // With GenericNamingOf: ResponseOfListOfUser
    GenericNamingExpanded
)

// GenericNamingConfig provides fine-grained control over generic type naming.
type GenericNamingConfig struct {
    // Strategy is the primary generic naming approach
    Strategy GenericNamingStrategy
    
    // Separator is used between base type and parameters (default: "_")
    // Only applies to GenericNamingUnderscore strategy
    Separator string
    
    // ParamSeparator is used between multiple type parameters (default: "_")
    // Example with ParamSeparator="_": Map[string,int] → Map_string_int
    // Example with ParamSeparator="And": Map[string,int] → MapOfStringAndInt
    ParamSeparator string
    
    // IncludePackage includes the type parameter's package in the name
    // Example: Response[models.User] → Response_models_User (true)
    // Example: Response[models.User] → Response_User (false, default)
    IncludePackage bool
    
    // ApplyBaseCasing applies the base naming strategy to type parameters
    // Example with SchemaNamingPascalCase: Response[user_profile] → ResponseOfUserProfile
    ApplyBaseCasing bool
}
```

### Template Context

```go
// SchemaNameContext provides metadata for custom naming templates.
type SchemaNameContext struct {
    // Type is the Go type name without package (e.g., "User", "Response[T]")
    Type string
    
    // TypeSanitized is Type with generic brackets replaced per GenericNamingStrategy
    TypeSanitized string
    
    // TypeBase is the base type name without generic parameters (e.g., "Response")
    TypeBase string
    
    // Package is the package base name (e.g., "models")
    Package string
    
    // PackagePath is the full import path (e.g., "github.com/org/models")
    PackagePath string
    
    // PackagePathSanitized is PackagePath with slashes replaced (e.g., "github.com_org_models")
    PackagePathSanitized string
    
    // IsGeneric indicates if the type has type parameters
    IsGeneric bool
    
    // GenericParams contains the type parameter names if IsGeneric is true
    GenericParams []string
    
    // GenericParamsSanitized contains sanitized type parameter names
    GenericParamsSanitized []string
    
    // GenericSuffix is the formatted generic parameters portion
    // Varies based on GenericNamingStrategy (e.g., "_User_", "OfUser", "<User>")
    GenericSuffix string
    
    // IsAnonymous indicates if this is an anonymous struct type
    IsAnonymous bool
    
    // IsPointer indicates if the original type was a pointer
    IsPointer bool
    
    // Kind is the reflect.Kind as a string (e.g., "struct", "slice", "map")
    Kind string
}
```

### Custom Naming Function

```go
// SchemaNameFunc is the signature for custom schema naming functions.
type SchemaNameFunc func(ctx SchemaNameContext) string
```

### Template Functions

The following functions are available in schema name templates:

| Function | Description | Example |
|----------|-------------|---------|
| `pascal` | Convert to PascalCase | `{{pascal .Type}}` → `UserProfile` |
| `camel` | Convert to camelCase | `{{camel .Type}}` → `userProfile` |
| `snake` | Convert to snake_case | `{{snake .Type}}` → `user_profile` |
| `kebab` | Convert to kebab-case | `{{kebab .Type}}` → `user-profile` |
| `upper` | Convert to UPPERCASE | `{{upper .Type}}` → `USERPROFILE` |
| `lower` | Convert to lowercase | `{{lower .Type}}` → `userprofile` |
| `title` | Convert to Title Case | `{{title .Package}}` → `Models` |
| `sanitize` | Replace URI-unsafe chars | `{{sanitize .PackagePath}}` → `github.com_org_models` |
| `trimPrefix` | Remove prefix | `{{trimPrefix .Package "api"}}` |
| `trimSuffix` | Remove suffix | `{{trimSuffix .Type "DTO"}}` |
| `replace` | String replacement | `{{replace .Type "_" ""}}` |
| `join` | Join with separator | `{{join "_" .Package .Type}}` |

## Usage Examples

### Built-in Strategies

```go
// Default: package.TypeName
spec := builder.New(parser.OASVersion320)  // models.User

// PascalCase: PackageTypeName
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNaming(builder.SchemaNamingPascalCase),
)  // ModelsUser

// Type only (no package prefix)
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNaming(builder.SchemaNamingTypeOnly),
)  // User

// Snake case
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNaming(builder.SchemaNamingSnakeCase),
)  // models_user
```

### Generic Type Strategies

```go
// Default: underscore replacement
spec := builder.New(parser.OASVersion320)
// Response[User] → Response_User_

// "Of" separator for readable names
spec := builder.New(parser.OASVersion320,
    builder.WithGenericNaming(builder.GenericNamingOf),
)
// Response[User] → ResponseOfUser
// Map[string,int] → MapOfStringOfInt

// "For" separator
spec := builder.New(parser.OASVersion320,
    builder.WithGenericNaming(builder.GenericNamingFor),
)
// Response[User] → ResponseForUser

// Flattened (no separator)
spec := builder.New(parser.OASVersion320,
    builder.WithGenericNaming(builder.GenericNamingFlattened),
)
// Response[User] → ResponseUser

// Angle brackets (URI-encoded in $ref)
spec := builder.New(parser.OASVersion320,
    builder.WithGenericNaming(builder.GenericNamingAngleBrackets),
)
// Response[User] → Response<User>
// $ref: "#/components/schemas/Response%3CUser%3E"
```

### Fine-Grained Generic Configuration

```go
// Custom separators
spec := builder.New(parser.OASVersion320,
    builder.WithGenericNamingConfig(builder.GenericNamingConfig{
        Strategy:       builder.GenericNamingOf,
        ParamSeparator: "And",
    }),
)
// Map[string,int] → MapOfStringAndInt

// Include package names in type parameters
spec := builder.New(parser.OASVersion320,
    builder.WithGenericNamingConfig(builder.GenericNamingConfig{
        Strategy:       builder.GenericNamingUnderscore,
        IncludePackage: true,
    }),
)
// Response[models.User] → Response_models_User_

// Apply base casing to type parameters
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNaming(builder.SchemaNamingPascalCase),
    builder.WithGenericNamingConfig(builder.GenericNamingConfig{
        Strategy:        builder.GenericNamingOf,
        ApplyBaseCasing: true,
    }),
)
// models.Response[api.user_profile] → ModelsResponseOfApiUserProfile

// Custom underscore separator
spec := builder.New(parser.OASVersion320,
    builder.WithGenericNamingConfig(builder.GenericNamingConfig{
        Strategy:  builder.GenericNamingUnderscore,
        Separator: "__",
    }),
)
// Response[User] → Response__User__
```

### Combined Strategies

```go
// PascalCase naming with "Of" generics
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNaming(builder.SchemaNamingPascalCase),
    builder.WithGenericNaming(builder.GenericNamingOf),
)
// models.Response[User] → ModelsResponseOfUser

// Snake case with custom generic separator
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNaming(builder.SchemaNamingSnakeCase),
    builder.WithGenericNamingConfig(builder.GenericNamingConfig{
        Strategy:       builder.GenericNamingUnderscore,
        Separator:      "_",
        ParamSeparator: "_",
    }),
)
// models.Response[User] → models_response_user
```

### Custom Templates

```go
// Simple template: TypeName only with PascalCase
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNameTemplate(`{{pascal .Type}}`),
)  // User

// Package + Type with custom separator
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNameTemplate(`{{.Package}}+{{.Type}}`),
)  // models+User

// Full control with conditionals
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNameTemplate(`{{if .IsGeneric}}Generic{{end}}{{pascal .TypeSanitized}}`),
)  // GenericResponse_User_ (for Response[User])

// Namespace-style naming
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNameTemplate(`{{snake .Package}}.{{snake .Type}}`),
)  // models.user
```

### Custom Functions

```go
// Programmatic naming with full control
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNameFunc(func(ctx builder.SchemaNameContext) string {
        if ctx.IsAnonymous {
            return fmt.Sprintf("Inline_%d", anonymousCounter.Add(1))
        }
        if ctx.Package == "internal" {
            return ctx.Type // Hide internal package
        }
        return fmt.Sprintf("%s_%s", strings.ToUpper(ctx.Package), ctx.Type)
    }),
)
```

### RegisterTypeAs Override

The existing `RegisterTypeAs` method continues to work as an explicit override:

```go
// Strategy applies to auto-generated names
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNaming(builder.SchemaNamingSnakeCase),
)

// But RegisterTypeAs always takes precedence
spec.RegisterTypeAs("MyCustomName", models.User{})  // MyCustomName, not models_user
```

## Quick Reference Tables

### Schema Naming Strategy Transformations

| Input Type | Default | PascalCase | camelCase | snake_case | kebab-case | TypeOnly | FullPath |
|------------|---------|------------|-----------|------------|------------|----------|----------|
| `models.User` | `models.User` | `ModelsUser` | `modelsUser` | `models_user` | `models-user` | `User` | `github.com_org_models_User` |
| `api.UserProfile` | `api.UserProfile` | `ApiUserProfile` | `apiUserProfile` | `api_user_profile` | `api-user-profile` | `UserProfile` | `github.com_org_api_UserProfile` |
| `internal.APIClient` | `internal.APIClient` | `InternalAPIClient` | `internalAPIClient` | `internal_api_client` | `internal-api-client` | `APIClient` | `github.com_org_internal_APIClient` |

### Generic Naming Strategy Transformations

| Input Type | Underscore | Of | For | Flat | AngleBrackets |
|------------|------------|-----|-----|------|---------------|
| `Response[User]` | `Response_User_` | `ResponseOfUser` | `ResponseForUser` | `ResponseUser` | `Response<User>` |
| `Map[string,int]` | `Map_string_int_` | `MapOfStringOfInt` | `MapForStringForInt` | `MapStringInt` | `Map<string,int>` |
| `Result[T,E]` | `Result_T_E_` | `ResultOfTOfE` | `ResultForTForE` | `ResultTE` | `Result<T,E>` |
| `List[User]` | `List_User_` | `ListOfUser` | `ListForUser` | `ListUser` | `List<User>` |

### Combined Strategy Examples

| Base Strategy | Generic Strategy | Input | Output |
|--------------|------------------|-------|--------|
| PascalCase | Of | `models.Response[User]` | `ModelsResponseOfUser` |
| snake_case | Underscore | `models.Response[User]` | `models_response_user_` |
| TypeOnly | Flat | `models.Response[User]` | `ResponseUser` |
| camelCase | For | `api.Result[Data,Error]` | `apiResultForDataForError` |
| Default | Of + ApplyBaseCasing | `models.Response[user_profile]` | `models.ResponseOfUserProfile` |

## Implementation Details

### File Structure

```
builder/
├── naming.go           # SchemaNamingStrategy, SchemaNameContext, template funcs
├── naming_test.go      # Unit tests for naming strategies
├── options.go          # BuilderOption type and With* functions (new file)
├── reflect.go          # Modified to use naming strategy
├── reflect_test.go     # Updated tests
└── doc.go              # Updated documentation
```

### Core Types (naming.go)

```go
package builder

import (
    "path"
    "reflect"
    "strings"
    "text/template"
    "unicode"
)

// SchemaNamingStrategy defines built-in schema naming conventions.
type SchemaNamingStrategy int

const (
    SchemaNamingDefault SchemaNamingStrategy = iota
    SchemaNamingPascalCase
    SchemaNamingCamelCase  
    SchemaNamingSnakeCase
    SchemaNamingKebabCase
    SchemaNamingTypeOnly
    SchemaNamingFullPath
)

// GenericNamingStrategy defines how generic type parameters are formatted.
type GenericNamingStrategy int

const (
    GenericNamingUnderscore GenericNamingStrategy = iota
    GenericNamingOf
    GenericNamingFor
    GenericNamingAngleBrackets
    GenericNamingFlattened
)

// GenericNamingConfig provides fine-grained control over generic type naming.
type GenericNamingConfig struct {
    Strategy        GenericNamingStrategy
    Separator       string
    ParamSeparator  string
    IncludePackage  bool
    ApplyBaseCasing bool
}

// DefaultGenericNamingConfig returns the default generic naming configuration.
func DefaultGenericNamingConfig() GenericNamingConfig {
    return GenericNamingConfig{
        Strategy:        GenericNamingUnderscore,
        Separator:       "_",
        ParamSeparator:  "_",
        IncludePackage:  false,
        ApplyBaseCasing: false,
    }
}

// SchemaNameContext provides type metadata for custom naming.
type SchemaNameContext struct {
    Type                   string
    TypeSanitized          string
    TypeBase               string
    Package                string
    PackagePath            string
    PackagePathSanitized   string
    IsGeneric              bool
    GenericParams          []string
    GenericParamsSanitized []string
    GenericSuffix          string
    IsAnonymous            bool
    IsPointer              bool
    Kind                   string
}

// SchemaNameFunc is the signature for custom naming functions.
type SchemaNameFunc func(ctx SchemaNameContext) string

// schemaNamer handles schema name generation.
type schemaNamer struct {
    strategy      SchemaNamingStrategy
    genericConfig GenericNamingConfig
    template      *template.Template
    fn            SchemaNameFunc
}

// newSchemaNamer creates a namer with the default strategy.
func newSchemaNamer() *schemaNamer {
    return &schemaNamer{
        strategy:      SchemaNamingDefault,
        genericConfig: DefaultGenericNamingConfig(),
    }
}

// name generates a schema name for the given type.
func (n *schemaNamer) name(t reflect.Type) string {
    ctx := n.buildContext(t)
    
    // Custom function takes highest priority
    if n.fn != nil {
        return n.fn(ctx)
    }
    
    // Template takes second priority
    if n.template != nil {
        var buf strings.Builder
        if err := n.template.Execute(&buf, ctx); err != nil {
            // Fall back to default on template error
            return n.defaultName(ctx)
        }
        return sanitizeSchemaName(buf.String())
    }
    
    // Built-in strategy
    return n.applyStrategy(ctx)
}

// buildContext creates SchemaNameContext from a reflect.Type.
func (n *schemaNamer) buildContext(t reflect.Type) SchemaNameContext {
    // Unwrap pointer types
    isPointer := false
    for t.Kind() == reflect.Ptr {
        t = t.Elem()
        isPointer = true
    }
    
    typeName := t.Name()
    pkgPath := t.PkgPath()
    
    ctx := SchemaNameContext{
        Type:                 typeName,
        Package:              path.Base(pkgPath),
        PackagePath:          pkgPath,
        PackagePathSanitized: sanitizePath(pkgPath),
        IsAnonymous:          typeName == "",
        IsPointer:            isPointer,
        Kind:                 t.Kind().String(),
    }
    
    // Handle generic types
    if strings.Contains(typeName, "[") {
        ctx.IsGeneric = true
        ctx.TypeBase = extractBaseTypeName(typeName)
        ctx.GenericParams = extractGenericParams(typeName)
        ctx.GenericParamsSanitized = n.sanitizeGenericParams(ctx.GenericParams)
        ctx.GenericSuffix = n.formatGenericSuffix(ctx.GenericParamsSanitized)
        ctx.TypeSanitized = ctx.TypeBase + ctx.GenericSuffix
    } else {
        ctx.TypeBase = typeName
        ctx.TypeSanitized = typeName
    }
    
    return ctx
}

// sanitizeGenericParams processes generic parameters according to configuration.
func (n *schemaNamer) sanitizeGenericParams(params []string) []string {
    result := make([]string, len(params))
    for i, param := range params {
        // Remove package prefix if not including package
        if !n.genericConfig.IncludePackage {
            if idx := strings.LastIndex(param, "."); idx != -1 {
                param = param[idx+1:]
            }
        } else {
            // Sanitize package path in parameter
            param = strings.ReplaceAll(param, ".", "_")
        }
        
        // Apply base casing if configured
        if n.genericConfig.ApplyBaseCasing {
            param = n.applyCasing(param)
        }
        
        result[i] = param
    }
    return result
}

// formatGenericSuffix creates the generic portion of the name.
func (n *schemaNamer) formatGenericSuffix(params []string) string {
    if len(params) == 0 {
        return ""
    }
    
    switch n.genericConfig.Strategy {
    case GenericNamingOf:
        return "Of" + strings.Join(params, n.genericConfig.ParamSeparator+"Of")
    
    case GenericNamingFor:
        return "For" + strings.Join(params, n.genericConfig.ParamSeparator+"For")
    
    case GenericNamingAngleBrackets:
        return "<" + strings.Join(params, ",") + ">"
    
    case GenericNamingFlattened:
        return strings.Join(params, "")
    
    default: // GenericNamingUnderscore
        sep := n.genericConfig.Separator
        if sep == "" {
            sep = "_"
        }
        return sep + strings.Join(params, n.genericConfig.ParamSeparator) + sep
    }
}

// applyCasing applies the current naming strategy's casing to a string.
func (n *schemaNamer) applyCasing(s string) string {
    switch n.strategy {
    case SchemaNamingPascalCase:
        return toPascalCase(s)
    case SchemaNamingCamelCase:
        return toCamelCase(s)
    case SchemaNamingSnakeCase:
        return toSnakeCase(s)
    case SchemaNamingKebabCase:
        return toKebabCase(s)
    default:
        return s
    }
}

// extractBaseTypeName extracts the base type name from a generic type.
// Example: "Response[User]" → "Response"
func extractBaseTypeName(name string) string {
    if idx := strings.Index(name, "["); idx != -1 {
        return name[:idx]
    }
    return name
}

// applyStrategy applies a built-in naming strategy.
func (n *schemaNamer) applyStrategy(ctx SchemaNameContext) string {
    if ctx.IsAnonymous {
        return "AnonymousType"
    }
    
    switch n.strategy {
    case SchemaNamingPascalCase:
        return toPascalCase(ctx.Package) + toPascalCase(ctx.TypeSanitized)
    
    case SchemaNamingCamelCase:
        return toCamelCase(ctx.Package) + toPascalCase(ctx.TypeSanitized)
    
    case SchemaNamingSnakeCase:
        return toSnakeCase(ctx.Package) + "_" + toSnakeCase(ctx.TypeSanitized)
    
    case SchemaNamingKebabCase:
        return toKebabCase(ctx.Package) + "-" + toKebabCase(ctx.TypeSanitized)
    
    case SchemaNamingTypeOnly:
        return ctx.TypeSanitized
    
    case SchemaNamingFullPath:
        return ctx.PackagePathSanitized + "_" + ctx.TypeSanitized
    
    default: // SchemaNamingDefault
        return n.defaultName(ctx)
    }
}

// defaultName generates the default package.TypeName format.
func (n *schemaNamer) defaultName(ctx SchemaNameContext) string {
    if ctx.IsAnonymous {
        return "AnonymousType"
    }
    if ctx.Package == "" {
        return ctx.TypeSanitized
    }
    return ctx.Package + "." + ctx.TypeSanitized
}
```

### Case Conversion Functions (naming.go continued)

```go
// toPascalCase converts a string to PascalCase.
func toPascalCase(s string) string {
    if s == "" {
        return ""
    }
    
    var result strings.Builder
    capitalizeNext := true
    
    for _, r := range s {
        if r == '_' || r == '-' || r == '.' || r == '/' {
            capitalizeNext = true
            continue
        }
        if capitalizeNext {
            result.WriteRune(unicode.ToUpper(r))
            capitalizeNext = false
        } else {
            result.WriteRune(r)
        }
    }
    
    return result.String()
}

// toCamelCase converts a string to camelCase.
func toCamelCase(s string) string {
    pascal := toPascalCase(s)
    if pascal == "" {
        return ""
    }
    runes := []rune(pascal)
    runes[0] = unicode.ToLower(runes[0])
    return string(runes)
}

// toSnakeCase converts a string to snake_case.
func toSnakeCase(s string) string {
    if s == "" {
        return ""
    }
    
    var result strings.Builder
    for i, r := range s {
        if unicode.IsUpper(r) {
            if i > 0 {
                result.WriteRune('_')
            }
            result.WriteRune(unicode.ToLower(r))
        } else if r == '-' || r == '.' || r == '/' {
            result.WriteRune('_')
        } else {
            result.WriteRune(r)
        }
    }
    
    return result.String()
}

// toKebabCase converts a string to kebab-case.
func toKebabCase(s string) string {
    return strings.ReplaceAll(toSnakeCase(s), "_", "-")
}

// sanitizePath replaces path separators with underscores.
func sanitizePath(s string) string {
    return strings.ReplaceAll(s, "/", "_")
}

// extractGenericParams extracts type parameters from a generic type name.
func extractGenericParams(name string) []string {
    start := strings.Index(name, "[")
    end := strings.LastIndex(name, "]")
    if start == -1 || end == -1 || end <= start {
        return nil
    }
    
    paramStr := name[start+1 : end]
    params := strings.Split(paramStr, ",")
    for i := range params {
        params[i] = strings.TrimSpace(params[i])
    }
    return params
}
```

### Template Functions (naming.go continued)

```go
// templateFuncs returns the function map for schema name templates.
func templateFuncs() template.FuncMap {
    return template.FuncMap{
        "pascal":     toPascalCase,
        "camel":      toCamelCase,
        "snake":      toSnakeCase,
        "kebab":      toKebabCase,
        "upper":      strings.ToUpper,
        "lower":      strings.ToLower,
        "title":      strings.Title,
        "sanitize":   sanitizeSchemaName,
        "trimPrefix": strings.TrimPrefix,
        "trimSuffix": strings.TrimSuffix,
        "replace":    strings.ReplaceAll,
        "join": func(sep string, parts ...string) string {
            return strings.Join(parts, sep)
        },
    }
}

// parseSchemaNameTemplate parses and validates a schema name template.
func parseSchemaNameTemplate(tmpl string) (*template.Template, error) {
    t, err := template.New("schemaName").Funcs(templateFuncs()).Parse(tmpl)
    if err != nil {
        return nil, fmt.Errorf("invalid schema name template: %w", err)
    }
    
    // Validate template with sample context
    ctx := SchemaNameContext{
        Type:     "TestType",
        Package:  "testpkg",
    }
    var buf strings.Builder
    if err := t.Execute(&buf, ctx); err != nil {
        return nil, fmt.Errorf("schema name template execution failed: %w", err)
    }
    
    return t, nil
}
```

### Builder Options (options.go)

```go
package builder

import "text/template"

// BuilderOption configures a Builder instance.
type BuilderOption func(*builderConfig)

// builderConfig holds builder configuration.
type builderConfig struct {
    namingStrategy SchemaNamingStrategy
    namingTemplate *template.Template
    namingFunc     SchemaNameFunc
    genericConfig  GenericNamingConfig
    templateError  error // Stores template parse errors for Build() to return
}

// WithSchemaNaming sets a built-in schema naming strategy.
func WithSchemaNaming(strategy SchemaNamingStrategy) BuilderOption {
    return func(cfg *builderConfig) {
        cfg.namingStrategy = strategy
        cfg.namingTemplate = nil
        cfg.namingFunc = nil
    }
}

// WithSchemaNameTemplate sets a custom Go text/template for schema naming.
// Template parse errors are returned by Build().
func WithSchemaNameTemplate(tmpl string) BuilderOption {
    return func(cfg *builderConfig) {
        t, err := parseSchemaNameTemplate(tmpl)
        if err != nil {
            cfg.templateError = err
            return
        }
        cfg.namingTemplate = t
        cfg.namingFunc = nil
    }
}

// WithSchemaNameFunc sets a custom function for schema naming.
func WithSchemaNameFunc(fn SchemaNameFunc) BuilderOption {
    return func(cfg *builderConfig) {
        cfg.namingFunc = fn
        cfg.namingTemplate = nil
    }
}

// WithGenericNaming sets the strategy for handling generic type names.
func WithGenericNaming(strategy GenericNamingStrategy) BuilderOption {
    return func(cfg *builderConfig) {
        cfg.genericConfig.Strategy = strategy
    }
}

// WithGenericNamingConfig provides fine-grained control over generic type naming.
func WithGenericNamingConfig(config GenericNamingConfig) BuilderOption {
    return func(cfg *builderConfig) {
        cfg.genericConfig = config
    }
}

// WithGenericSeparator sets the separator used for generic type parameters.
// Only applies to GenericNamingUnderscore strategy.
func WithGenericSeparator(sep string) BuilderOption {
    return func(cfg *builderConfig) {
        cfg.genericConfig.Separator = sep
    }
}

// WithGenericParamSeparator sets the separator between multiple type parameters.
func WithGenericParamSeparator(sep string) BuilderOption {
    return func(cfg *builderConfig) {
        cfg.genericConfig.ParamSeparator = sep
    }
}

// WithGenericIncludePackage includes package names in generic type parameters.
func WithGenericIncludePackage(include bool) BuilderOption {
    return func(cfg *builderConfig) {
        cfg.genericConfig.IncludePackage = include
    }
}

// WithGenericApplyBaseCasing applies the base naming strategy to type parameters.
func WithGenericApplyBaseCasing(apply bool) BuilderOption {
    return func(cfg *builderConfig) {
        cfg.genericConfig.ApplyBaseCasing = apply
    }
}
```

### Builder Modifications (builder.go)

```go
// Builder struct additions
type Builder struct {
    // ... existing fields ...
    
    namer *schemaNamer // Schema naming configuration
}

// New signature update (backward compatible via variadic options)
func New(version parser.OASVersion, opts ...BuilderOption) *Builder {
    cfg := &builderConfig{
        namingStrategy: SchemaNamingDefault,
        genericConfig:  DefaultGenericNamingConfig(),
    }
    for _, opt := range opts {
        opt(cfg)
    }
    
    // Check for template parse errors
    // These will be returned by Build() methods
    
    namer := newSchemaNamer()
    namer.strategy = cfg.namingStrategy
    namer.genericConfig = cfg.genericConfig
    namer.template = cfg.namingTemplate
    namer.fn = cfg.namingFunc
    
    b := &Builder{
        version:       version,
        info:          &parser.Info{},
        paths:         make(parser.Paths),
        schemas:       make(map[string]*parser.Schema),
        // ... other initializations ...
        namer:         namer,
        configError:   cfg.templateError, // Store for Build() to check
    }
    
    return b
}

// BuildOAS3 additions for error handling
func (b *Builder) BuildOAS3() (*parser.OAS3Document, error) {
    // Check for configuration errors first
    if b.configError != nil {
        return nil, fmt.Errorf("builder configuration error: %w", b.configError)
    }
    
    // ... rest of existing implementation ...
}
```

### Reflect.go Modifications

Replace direct calls to `schemaName(t)` with `b.namer.name(t)`:

```go
// Before
name := b.schemaName(t)

// After  
name := b.namer.name(t)
```

Remove the old `schemaName` method and `sanitizeSchemaName` function, moving them to `naming.go`.

## Migration Path

### Phase 1: Internal Refactor (Non-Breaking)

1. Create `naming.go` with new types and functions
2. Refactor existing `schemaName` to use `schemaNamer` internally
3. Ensure all existing tests pass unchanged

### Phase 2: API Addition (Backward Compatible)

1. Add `BuilderOption` type and `With*` functions
2. Update `New()` to accept variadic options
3. Add comprehensive tests for new functionality
4. Update documentation

### Phase 3: Documentation and Examples

1. Update package documentation
2. Add examples to `example_test.go`
3. Update developer guide

## Test Plan

### Unit Tests (naming_test.go)

```go
func TestSchemaNamingStrategies(t *testing.T) {
    tests := []struct {
        name     string
        strategy SchemaNamingStrategy
        typeName string
        pkgPath  string
        want     string
    }{
        {"default", SchemaNamingDefault, "User", "models", "models.User"},
        {"pascal", SchemaNamingPascalCase, "User", "models", "ModelsUser"},
        {"camel", SchemaNamingCamelCase, "User", "models", "modelsUser"},
        {"snake", SchemaNamingSnakeCase, "User", "models", "models_user"},
        {"kebab", SchemaNamingKebabCase, "User", "models", "models-user"},
        {"type_only", SchemaNamingTypeOnly, "User", "models", "User"},
        {"full_path", SchemaNamingFullPath, "User", "github.com/org/models", "github.com_org_models_User"},
    }
    // ... test implementation
}

func TestGenericNamingStrategies(t *testing.T) {
    tests := []struct {
        name     string
        strategy GenericNamingStrategy
        typeName string
        want     string
    }{
        {"underscore_single", GenericNamingUnderscore, "Response[User]", "Response_User_"},
        {"underscore_multi", GenericNamingUnderscore, "Map[string,int]", "Map_string_int_"},
        {"underscore_nested", GenericNamingUnderscore, "Response[List[User]]", "Response_List_User_"},
        {"of_single", GenericNamingOf, "Response[User]", "ResponseOfUser"},
        {"of_multi", GenericNamingOf, "Map[string,int]", "MapOfStringOfInt"},
        {"for_single", GenericNamingFor, "Response[User]", "ResponseForUser"},
        {"angle_single", GenericNamingAngleBrackets, "Response[User]", "Response<User>"},
        {"flat_single", GenericNamingFlattened, "Response[User]", "ResponseUser"},
        {"flat_multi", GenericNamingFlattened, "Map[string,int]", "MapStringInt"},
    }
    // ... test implementation
}

func TestGenericNamingConfig(t *testing.T) {
    tests := []struct {
        name   string
        config GenericNamingConfig
        input  string
        want   string
    }{
        {
            name: "custom_separator",
            config: GenericNamingConfig{
                Strategy:  GenericNamingUnderscore,
                Separator: "__",
            },
            input: "Response[User]",
            want:  "Response__User__",
        },
        {
            name: "custom_param_separator",
            config: GenericNamingConfig{
                Strategy:       GenericNamingOf,
                ParamSeparator: "And",
            },
            input: "Map[string,int]",
            want:  "MapOfStringAndOfInt",
        },
        {
            name: "include_package",
            config: GenericNamingConfig{
                Strategy:       GenericNamingUnderscore,
                IncludePackage: true,
            },
            input: "Response[models.User]",
            want:  "Response_models_User_",
        },
        {
            name: "apply_base_casing_pascal",
            config: GenericNamingConfig{
                Strategy:        GenericNamingOf,
                ApplyBaseCasing: true,
            },
            input: "Response[user_profile]",
            want:  "ResponseOfUserProfile", // with PascalCase base strategy
        },
    }
    // ... test implementation
}

func TestCaseConversions(t *testing.T) {
    tests := []struct {
        input    string
        pascal   string
        camel    string
        snake    string
        kebab    string
    }{
        {"UserProfile", "UserProfile", "userProfile", "user_profile", "user-profile"},
        {"user_profile", "UserProfile", "userProfile", "user_profile", "user-profile"},
        {"user-profile", "UserProfile", "userProfile", "user_profile", "user-profile"},
        {"APIClient", "APIClient", "aPIClient", "api_client", "api-client"},
    }
    // ... test implementation
}

func TestSchemaNameTemplates(t *testing.T) {
    tests := []struct {
        name     string
        template string
        ctx      SchemaNameContext
        want     string
    }{
        {
            name:     "simple_type",
            template: `{{.Type}}`,
            ctx:      SchemaNameContext{Type: "User", Package: "models"},
            want:     "User",
        },
        {
            name:     "pascal_case",
            template: `{{pascal .Package}}{{pascal .Type}}`,
            ctx:      SchemaNameContext{Type: "UserProfile", Package: "api_models"},
            want:     "ApiModelsUserProfile",
        },
        {
            name:     "conditional_generic",
            template: `{{if .IsGeneric}}Generic_{{end}}{{.TypeSanitized}}`,
            ctx:      SchemaNameContext{TypeSanitized: "Response_User_", IsGeneric: true},
            want:     "Generic_Response_User_",
        },
        {
            name:     "generic_base_and_suffix",
            template: `{{.TypeBase}}{{.GenericSuffix}}`,
            ctx:      SchemaNameContext{TypeBase: "Response", GenericSuffix: "OfUser", IsGeneric: true},
            want:     "ResponseOfUser",
        },
        {
            name:     "custom_generic_format",
            template: `{{.TypeBase}}{{if .IsGeneric}}[{{range $i, $p := .GenericParams}}{{if $i}},{{end}}{{$p}}{{end}}]{{end}}`,
            ctx: SchemaNameContext{
                TypeBase:      "Map",
                IsGeneric:     true,
                GenericParams: []string{"string", "int"},
            },
            want: "Map[string,int]",
        },
    }
    // ... test implementation
}

func TestSchemaNameFunc(t *testing.T) {
    counter := 0
    fn := func(ctx SchemaNameContext) string {
        if ctx.IsAnonymous {
            counter++
            return fmt.Sprintf("Anon%d", counter)
        }
        if ctx.IsGeneric {
            return ctx.TypeBase + "Generic"
        }
        return strings.ToUpper(ctx.Type)
    }
    // ... test implementation
}

func TestExtractGenericParams(t *testing.T) {
    tests := []struct {
        input string
        want  []string
    }{
        {"Response[User]", []string{"User"}},
        {"Map[string,int]", []string{"string", "int"}},
        {"Response[List[User]]", []string{"List[User]"}},
        {"Tuple[A,B,C]", []string{"A", "B", "C"}},
        {"NoGenerics", nil},
        {"Response[models.User]", []string{"models.User"}},
    }
    // ... test implementation
}

func TestExtractBaseTypeName(t *testing.T) {
    tests := []struct {
        input string
        want  string
    }{
        {"Response[User]", "Response"},
        {"Map[string,int]", "Map"},
        {"NoGenerics", "NoGenerics"},
        {"Nested[List[User]]", "Nested"},
    }
    // ... test implementation
}
```

### Integration Tests

```go
func TestBuilderWithNamingStrategy(t *testing.T) {
    type User struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
    }
    
    spec := builder.New(parser.OASVersion320,
        builder.WithSchemaNaming(builder.SchemaNamingPascalCase),
    ).
        SetTitle("Test API").
        SetVersion("1.0.0").
        AddOperation(http.MethodGet, "/users",
            builder.WithResponse(http.StatusOK, []User{}),
        )
    
    doc, err := spec.BuildOAS3()
    require.NoError(t, err)
    
    // Verify schema name uses PascalCase
    _, exists := doc.Components.Schemas["BuilderTestUser"]
    assert.True(t, exists, "expected PascalCase schema name")
}

func TestBuilderWithGenericTypes(t *testing.T) {
    type Response[T any] struct {
        Data T `json:"data"`
    }
    type User struct {
        ID int `json:"id"`
    }
    
    tests := []struct {
        name           string
        genericNaming  builder.GenericNamingStrategy
        wantSchemaName string
    }{
        {"underscore", builder.GenericNamingUnderscore, "builder_test.Response_User_"},
        {"of", builder.GenericNamingOf, "builder_test.ResponseOfUser"},
        {"for", builder.GenericNamingFor, "builder_test.ResponseForUser"},
        {"flat", builder.GenericNamingFlattened, "builder_test.ResponseUser"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            spec := builder.New(parser.OASVersion320,
                builder.WithGenericNaming(tt.genericNaming),
            ).
                SetTitle("Test API").
                SetVersion("1.0.0").
                AddOperation(http.MethodGet, "/users",
                    builder.WithResponse(http.StatusOK, Response[User]{}),
                )
            
            doc, err := spec.BuildOAS3()
            require.NoError(t, err)
            
            _, exists := doc.Components.Schemas[tt.wantSchemaName]
            assert.True(t, exists, "expected schema name %s", tt.wantSchemaName)
        })
    }
}

func TestBuilderCombinedStrategies(t *testing.T) {
    type Response[T any] struct {
        Data T `json:"data"`
    }
    type User struct {
        ID int `json:"id"`
    }
    
    spec := builder.New(parser.OASVersion320,
        builder.WithSchemaNaming(builder.SchemaNamingPascalCase),
        builder.WithGenericNamingConfig(builder.GenericNamingConfig{
            Strategy:        builder.GenericNamingOf,
            ApplyBaseCasing: true,
        }),
    ).
        SetTitle("Test API").
        SetVersion("1.0.0").
        AddOperation(http.MethodGet, "/users",
            builder.WithResponse(http.StatusOK, Response[User]{}),
        )
    
    doc, err := spec.BuildOAS3()
    require.NoError(t, err)
    
    // PascalCase + Of generics
    _, exists := doc.Components.Schemas["BuilderTestResponseOfUser"]
    assert.True(t, exists, "expected combined strategy schema name")
}

func TestTemplateParseError(t *testing.T) {
    spec := builder.New(parser.OASVersion320,
        builder.WithSchemaNameTemplate(`{{.InvalidField}}`),
    ).
        SetTitle("Test API").
        SetVersion("1.0.0")
    
    _, err := spec.BuildOAS3()
    assert.Error(t, err, "expected template error")
    assert.Contains(t, err.Error(), "template")
}
```

### Benchmark Tests

```go
func BenchmarkSchemaNaming(b *testing.B) {
    strategies := []struct {
        name     string
        strategy SchemaNamingStrategy
    }{
        {"default", SchemaNamingDefault},
        {"pascal", SchemaNamingPascalCase},
        {"snake", SchemaNamingSnakeCase},
    }
    
    type User struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
    }
    
    for _, s := range strategies {
        b.Run(s.name, func(b *testing.B) {
            for b.Loop() {
                spec := builder.New(parser.OASVersion320,
                    builder.WithSchemaNaming(s.strategy),
                ).
                    SetTitle("Test").
                    SetVersion("1.0.0").
                    AddOperation(http.MethodGet, "/users",
                        builder.WithResponse(http.StatusOK, User{}),
                    )
                _, _ = spec.BuildOAS3()
            }
        })
    }
}

func BenchmarkGenericNaming(b *testing.B) {
    strategies := []struct {
        name     string
        strategy GenericNamingStrategy
    }{
        {"underscore", GenericNamingUnderscore},
        {"of", GenericNamingOf},
        {"flat", GenericNamingFlattened},
    }
    
    type Response[T any] struct {
        Data T `json:"data"`
    }
    type User struct {
        ID int `json:"id"`
    }
    
    for _, s := range strategies {
        b.Run(s.name, func(b *testing.B) {
            for b.Loop() {
                spec := builder.New(parser.OASVersion320,
                    builder.WithGenericNaming(s.strategy),
                ).
                    SetTitle("Test").
                    SetVersion("1.0.0").
                    AddOperation(http.MethodGet, "/users",
                        builder.WithResponse(http.StatusOK, Response[User]{}),
                    )
                _, _ = spec.BuildOAS3()
            }
        })
    }
}

func BenchmarkSchemaNameTemplate(b *testing.B) {
    templates := []struct {
        name string
        tmpl string
    }{
        {"simple", `{{.Type}}`},
        {"pascal", `{{pascal .Package}}{{pascal .Type}}`},
        {"complex", `{{if .IsGeneric}}Generic_{{end}}{{snake .Package}}_{{pascal .Type}}`},
        {"generic_aware", `{{.TypeBase}}{{if .IsGeneric}}Of{{range .GenericParams}}{{pascal .}}{{end}}{{end}}`},
    }
    
    type Response[T any] struct {
        Data T `json:"data"`
    }
    type User struct {
        ID int `json:"id"`
    }
    
    for _, tt := range templates {
        b.Run(tt.name, func(b *testing.B) {
            for b.Loop() {
                spec := builder.New(parser.OASVersion320,
                    builder.WithSchemaNameTemplate(tt.tmpl),
                ).
                    SetTitle("Test").
                    SetVersion("1.0.0").
                    AddOperation(http.MethodGet, "/users",
                        builder.WithResponse(http.StatusOK, Response[User]{}),
                    )
                _, _ = spec.BuildOAS3()
            }
        })
    }
}

func BenchmarkCaseConversions(b *testing.B) {
    input := "UserProfileSettings"
    
    b.Run("toPascalCase", func(b *testing.B) {
        for b.Loop() {
            _ = toPascalCase(input)
        }
    })
    
    b.Run("toCamelCase", func(b *testing.B) {
        for b.Loop() {
            _ = toCamelCase(input)
        }
    })
    
    b.Run("toSnakeCase", func(b *testing.B) {
        for b.Loop() {
            _ = toSnakeCase(input)
        }
    })
    
    b.Run("toKebabCase", func(b *testing.B) {
        for b.Loop() {
            _ = toKebabCase(input)
        }
    })
}

func BenchmarkExtractGenericParams(b *testing.B) {
    inputs := []struct {
        name  string
        input string
    }{
        {"simple", "Response[User]"},
        {"multi", "Map[string,int]"},
        {"nested", "Response[List[User]]"},
        {"none", "SimpleType"},
    }
    
    for _, tt := range inputs {
        b.Run(tt.name, func(b *testing.B) {
            for b.Loop() {
                _ = extractGenericParams(tt.input)
            }
        })
    }
}
```

## Documentation Updates

### Package Documentation (doc.go)

Add new section after "Schema Naming":

```go
// # Extensible Schema Naming
//
// The default schema naming uses "package.TypeName" format. For custom naming,
// use one of the WithSchemaNaming options when creating a Builder:
//
// Built-in strategies:
//
//     // PascalCase: ModelsUser
//     spec := builder.New(parser.OASVersion320,
//         builder.WithSchemaNaming(builder.SchemaNamingPascalCase),
//     )
//
//     // Type only (no package): User
//     spec := builder.New(parser.OASVersion320,
//         builder.WithSchemaNaming(builder.SchemaNamingTypeOnly),
//     )
//
// Custom templates using Go text/template:
//
//     // Custom separator: models+User
//     spec := builder.New(parser.OASVersion320,
//         builder.WithSchemaNameTemplate(`{{.Package}}+{{.Type}}`),
//     )
//
// Available template functions: pascal, camel, snake, kebab, upper, lower,
// title, sanitize, trimPrefix, trimSuffix, replace, join.
//
// For maximum flexibility, use a custom function:
//
//     spec := builder.New(parser.OASVersion320,
//         builder.WithSchemaNameFunc(func(ctx builder.SchemaNameContext) string {
//             return strings.ToUpper(ctx.Type)
//         }),
//     )
//
// Note: RegisterTypeAs always takes precedence over any naming strategy.
```

### CLI Updates (if applicable)

Consider adding CLI flags for the `build` subcommand (if one exists):

```
--schema-naming <strategy>    Schema naming strategy (default, pascal, camel, snake, kebab, type-only, full-path)
--schema-name-template <tmpl> Custom Go template for schema names
```

## CLI Integration

### New Command: `oastools build`

The `build` command enables programmatic OAS document construction from Go source files or configuration. Schema naming options apply when generating schemas from Go types.

### Global Schema Naming Flags

These flags are available on commands that generate schemas from Go types:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--schema-naming` | `-n` | Schema naming strategy | `default` |
| `--schema-name-template` | | Go text/template for custom naming | |
| `--generic-naming` | `-g` | Generic type naming strategy | `underscore` |
| `--generic-separator` | | Separator for generic parameters | `_` |
| `--generic-param-separator` | | Separator between multiple type params | `_` |
| `--generic-include-package` | | Include package in type parameter names | `false` |

### Schema Naming Strategy Values

| Value | Example Output | Description |
|-------|----------------|-------------|
| `default` | `models.User` | Package dot TypeName (current behavior) |
| `pascal` | `ModelsUser` | PascalCase concatenation |
| `camel` | `modelsUser` | camelCase concatenation |
| `snake` | `models_user` | snake_case with underscores |
| `kebab` | `models-user` | kebab-case with hyphens |
| `type-only` | `User` | TypeName only, no package |
| `full-path` | `github.com_org_models_User` | Full package path |

### Generic Naming Strategy Values

| Value | Example Output | Description |
|-------|----------------|-------------|
| `underscore` | `Response_User_` | Brackets replaced with underscores |
| `of` | `ResponseOfUser` | "Of" separator |
| `for` | `ResponseForUser` | "For" separator |
| `angle` | `Response<User>` | Angle brackets (URI-encoded) |
| `flat` | `ResponseUser` | No separator, direct concatenation |

### CLI Usage Examples

```bash
# Default naming (package.TypeName)
oastools build -o api.yaml ./api/...

# PascalCase naming
oastools build --schema-naming pascal -o api.yaml ./api/...

# Type-only naming (warning: may cause conflicts)
oastools build --schema-naming type-only -o api.yaml ./api/...

# Custom template
oastools build --schema-name-template '{{snake .Package}}_{{pascal .Type}}' -o api.yaml ./api/...

# Generic types with "Of" separator
oastools build --generic-naming of -o api.yaml ./api/...
# Result: Response[User] → ResponseOfUser

# Combined: PascalCase with "Of" generics
oastools build --schema-naming pascal --generic-naming of -o api.yaml ./api/...
# Result: Response[User] in models package → ModelsResponseOfUser

# Custom generic separator
oastools build --generic-naming underscore --generic-separator '__' -o api.yaml ./api/...
# Result: Response[User] → Response__User__

# Include package in generic parameters
oastools build --generic-include-package -o api.yaml ./api/...
# Result: Response[models.User] → Response_models_User
```

### Configuration File Support

Schema naming can also be configured via `.oastools.yaml`:

```yaml
# .oastools.yaml
build:
  schema_naming:
    strategy: pascal          # default, pascal, camel, snake, kebab, type-only, full-path
    template: ""              # Custom Go template (overrides strategy if set)
  
  generic_naming:
    strategy: of              # underscore, of, for, angle, flat
    separator: "_"            # Separator for underscore strategy
    param_separator: "And"    # Separator between multiple type params
    include_package: false    # Include package in type param names
    apply_base_casing: true   # Apply base naming strategy to type params

# Example: Generates "ModelsResponseOfUserAndRole" for Response[User, Role] in models package
```

### Environment Variables

All CLI flags can be set via environment variables:

| Environment Variable | Corresponding Flag |
|---------------------|-------------------|
| `OASTOOLS_SCHEMA_NAMING` | `--schema-naming` |
| `OASTOOLS_SCHEMA_NAME_TEMPLATE` | `--schema-name-template` |
| `OASTOOLS_GENERIC_NAMING` | `--generic-naming` |
| `OASTOOLS_GENERIC_SEPARATOR` | `--generic-separator` |
| `OASTOOLS_GENERIC_PARAM_SEPARATOR` | `--generic-param-separator` |
| `OASTOOLS_GENERIC_INCLUDE_PACKAGE` | `--generic-include-package` |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Build successful |
| 1 | Invalid schema naming strategy |
| 2 | Invalid schema name template (parse error) |
| 3 | Invalid generic naming strategy |
| 4 | Schema name collision detected |
| 5 | Build failed (other error) |

## Risk Assessment

### Low Risk

- **Backward Compatibility**: Default behavior unchanged; existing code continues to work
- **Performance**: Built-in strategies have minimal overhead; templates are cached

### Medium Risk

- **Template Errors**: Invalid templates could cause runtime panics
- **Mitigation**: Validate templates at parse time; return clear errors

### Considerations

- **Name Collisions**: `SchemaNamingTypeOnly` may cause collisions with same-named types
- **Mitigation**: Document this limitation; maintain conflict detection logic

## Timeline Estimate

| Phase | Tasks | Estimate |
|-------|-------|----------|
| 1 | Internal refactor, naming.go (base + generic) | 3-4 hours |
| 2 | API additions, options.go (all options) | 2-3 hours |
| 3 | Builder modifications | 1-2 hours |
| 4 | CLI integration (cmd/build.go) | 2-3 hours |
| 5 | Configuration file support | 1-2 hours |
| 6 | Unit tests (naming strategies + generic) | 3-4 hours |
| 7 | Integration tests | 2-3 hours |
| 8 | Benchmarks | 1-2 hours |
| 9 | Documentation | 2-3 hours |
| **Total** | | **17-26 hours** |

## Acceptance Criteria

1. All existing tests pass without modification
2. New strategies produce expected schema names for documented examples
3. Generic naming strategies produce expected output for all documented cases
4. Templates execute correctly with all template functions including generic-aware fields
5. Custom functions receive complete `SchemaNameContext` with all generic metadata
6. Performance benchmarks show <5% regression for default strategy
7. CLI flags parse correctly and produce expected naming behavior
8. Configuration file loading works for all supported options
9. Environment variables override defaults correctly
10. Error messages for invalid templates/strategies are clear and actionable
11. Documentation includes examples for all strategies, generic options, and CLI usage
12. `make check` passes (fmt, lint, test, tidy)
