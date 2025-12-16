# Implementation Plan: Generic Schema Name Fixing for the Fixer Package

This plan provides a single-phase implementation roadmap for enhancing the `fixer` package with the ability to fix invalid schema names containing unencoded special characters (particularly square brackets from generic types). The implementation mirrors the `GenericNamingStrategy` options from the `builder` package for consistency across the toolkit.

---

## Executive Summary

Third-party code generators often produce OpenAPI specifications with schema names containing unencoded square brackets (e.g., `Response[User]`). While the `$ref` URLs pointing to these schemas are correctly URL-encoded (`#/components/schemas/Response%5BUser%5D`), the actual map keys in `definitions` or `components.schemas` retain the raw brackets. This mismatch causes reference resolution failures in most OpenAPI tooling. This enhancement provides automatic detection and renaming of problematic schema names with configurable naming strategies matching the `builder` package.

---

## Problem Statement

### The Mismatch

```yaml
# What third-party generators produce:
components:
  schemas:
    Response[User]:           # ← Raw brackets in map key
      type: object
      properties:
        data:
          $ref: '#/components/schemas/User'
    
paths:
  /users:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Response%5BUser%5D'  # ← Encoded in $ref
```

The `$ref` value `Response%5BUser%5D` (URL-encoded) doesn't match the schema key `Response[User]` (unencoded), causing broken references.

### Characters That Cause Problems

| Character | URL Encoded | Common Source |
|-----------|-------------|---------------|
| `[` | `%5B` | Generic type parameters |
| `]` | `%5D` | Generic type parameters |
| `<` | `%3C` | C#/Java style generics |
| `>` | `%3E` | C#/Java style generics |
| `,` | `%2C` | Multiple type parameters |
| ` ` | `%20` | Poorly sanitized names |

---

## Scope and Objectives

The implementation delivers four interconnected capabilities:

1. **Schema Name Detection** that identifies schema names containing problematic characters that would require URL encoding in `$ref` values.

2. **Configurable Naming Strategies** matching the `builder` package's `GenericNamingStrategy` options for consistency across the toolkit.

3. **Automatic Reference Rewriting** that updates all `$ref` values throughout the document to point to the renamed schemas.

4. **CLI Integration** with flags for strategy selection and customization.

---

## Architecture Overview

The implementation adds one new source file and modifies existing files within the `fixer` package. The naming logic is extracted into a dedicated `generic_names.go` file to maintain separation of concerns and enable reuse of the `builder` package's naming strategies.

```
fixer/
├── fixer.go              (modified: new fix type, routing)
├── oas2.go               (modified: integrate fix calls)
├── oas3.go               (modified: integrate fix calls)
├── generic_names.go      (new: detection and renaming logic)
├── refs.go               (from pruning plan: reference collection)
├── prune.go              (from pruning plan)
├── fixer_test.go         (modified: new tests)
├── generic_names_test.go (new: naming strategy tests)
├── example_test.go       (modified: add examples)
└── doc.go                (modified: document new fix type)

cmd/oastools/commands/
└── fix.go                (modified: new CLI flags)
```

---

## Detailed Implementation Specifications

### New Fix Type

```go
const (
    // Existing fix types
    FixTypeMissingPathParameter FixType = "missing-path-parameter"
    FixTypePruneSchemas         FixType = "prune-schemas"
    FixTypePrunePaths           FixType = "prune-paths"
    FixTypePruneComponents      FixType = "prune-components"
    
    // New fix type
    FixTypeInvalidSchemaName    FixType = "invalid-schema-name"
)
```

### Generic Naming Strategies

Mirror the `builder` package's strategies for consistency:

```go
// GenericNamingStrategy defines how generic type parameters are reformatted
// when fixing invalid schema names.
type GenericNamingStrategy int

const (
    // GenericNamingUnderscore replaces brackets with underscores.
    // Example: Response[User] → Response_User_
    // Example: Map[string,int] → Map_string_int_
    GenericNamingUnderscore GenericNamingStrategy = iota
    
    // GenericNamingOf uses "Of" separator between base type and parameters.
    // Example: Response[User] → ResponseOfUser
    // Example: Map[string,int] → MapOfStringOfInt
    GenericNamingOf
    
    // GenericNamingFor uses "For" separator.
    // Example: Response[User] → ResponseForUser
    // Example: Map[string,int] → MapForStringForInt
    GenericNamingFor
    
    // GenericNamingFlattened removes brackets entirely.
    // Example: Response[User] → ResponseUser
    // Example: Map[string,int] → MapStringInt
    GenericNamingFlattened
    
    // GenericNamingDot uses dots as separator.
    // Example: Response[User] → Response.User
    // Example: Map[string,int] → Map.string.int
    GenericNamingDot
)

// String returns the string representation of the strategy for CLI/config use.
func (s GenericNamingStrategy) String() string {
    switch s {
    case GenericNamingUnderscore:
        return "underscore"
    case GenericNamingOf:
        return "of"
    case GenericNamingFor:
        return "for"
    case GenericNamingFlattened:
        return "flat"
    case GenericNamingDot:
        return "dot"
    default:
        return "underscore"
    }
}

// ParseGenericNamingStrategy converts a string to GenericNamingStrategy.
func ParseGenericNamingStrategy(s string) (GenericNamingStrategy, error) {
    switch strings.ToLower(s) {
    case "underscore", "_":
        return GenericNamingUnderscore, nil
    case "of":
        return GenericNamingOf, nil
    case "for":
        return GenericNamingFor, nil
    case "flat", "flattened":
        return GenericNamingFlattened, nil
    case "dot", ".":
        return GenericNamingDot, nil
    default:
        return GenericNamingUnderscore, fmt.Errorf("unknown generic naming strategy: %s", s)
    }
}
```

### Generic Naming Configuration

```go
// GenericNamingConfig provides fine-grained control over schema name fixing.
type GenericNamingConfig struct {
    // Strategy is the primary naming approach for generic types.
    Strategy GenericNamingStrategy
    
    // Separator is used between base type and parameters.
    // Only applies to GenericNamingUnderscore strategy.
    // Default: "_"
    Separator string
    
    // ParamSeparator is used between multiple type parameters.
    // Example with ParamSeparator="_": Map[string,int] → Map_string_int
    // Example with ParamSeparator="And": Map[string,int] → MapOfStringAndInt
    // Default: "_" for underscore, matches word for of/for strategies
    ParamSeparator string
    
    // PreserveCasing keeps the original casing of type parameters.
    // When false (default), type parameters are converted to PascalCase
    // for "of" and "for" strategies.
    // Example with PreserveCasing=false: Response[user_data] → ResponseOfUserData
    // Example with PreserveCasing=true: Response[user_data] → ResponseOfuser_data
    PreserveCasing bool
}

// DefaultGenericNamingConfig returns the default configuration.
func DefaultGenericNamingConfig() GenericNamingConfig {
    return GenericNamingConfig{
        Strategy:       GenericNamingUnderscore,
        Separator:      "_",
        ParamSeparator: "_",
        PreserveCasing: false,
    }
}
```

### Detection Logic (generic_names.go)

```go
// invalidSchemaNameChars contains characters that require URL encoding in $ref values
// and thus should not appear in schema names.
var invalidSchemaNameChars = []rune{'[', ']', '<', '>', ',', ' ', '{', '}', '|', '\\', '^', '`'}

// hasInvalidSchemaNameChars returns true if the name contains characters that
// would require URL encoding in a $ref value.
func hasInvalidSchemaNameChars(name string) bool {
    for _, r := range name {
        for _, invalid := range invalidSchemaNameChars {
            if r == invalid {
                return true
            }
        }
    }
    return false
}

// isGenericStyleName returns true if the name appears to be a generic type name
// (contains brackets indicating type parameters).
func isGenericStyleName(name string) bool {
    return strings.ContainsAny(name, "[]<>")
}

// parseGenericName extracts the base name and type parameters from a generic-style name.
// Returns the base name, slice of type parameters, and the bracket style used.
//
// Examples:
//   - "Response[User]" → ("Response", ["User"], '[')
//   - "Map[string,int]" → ("Map", ["string", "int"], '[')
//   - "Result<T,E>" → ("Result", ["T", "E"], '<')
//   - "SimpleType" → ("SimpleType", nil, 0)
func parseGenericName(name string) (base string, params []string, bracketStyle rune) {
    // Try square brackets first
    if idx := strings.IndexRune(name, '['); idx != -1 {
        base = name[:idx]
        bracketStyle = '['
        
        // Find matching closing bracket
        end := strings.LastIndexRune(name, ']')
        if end > idx {
            paramStr := name[idx+1 : end]
            params = splitTypeParams(paramStr)
        }
        return
    }
    
    // Try angle brackets
    if idx := strings.IndexRune(name, '<'); idx != -1 {
        base = name[:idx]
        bracketStyle = '<'
        
        end := strings.LastIndexRune(name, '>')
        if end > idx {
            paramStr := name[idx+1 : end]
            params = splitTypeParams(paramStr)
        }
        return
    }
    
    // Not a generic name
    return name, nil, 0
}

// splitTypeParams splits a type parameter string by commas, handling nested brackets.
// Example: "User,List[Item],int" → ["User", "List[Item]", "int"]
func splitTypeParams(s string) []string {
    var params []string
    var current strings.Builder
    depth := 0
    
    for _, r := range s {
        switch r {
        case '[', '<':
            depth++
            current.WriteRune(r)
        case ']', '>':
            depth--
            current.WriteRune(r)
        case ',':
            if depth == 0 {
                param := strings.TrimSpace(current.String())
                if param != "" {
                    params = append(params, param)
                }
                current.Reset()
            } else {
                current.WriteRune(r)
            }
        default:
            current.WriteRune(r)
        }
    }
    
    // Don't forget the last parameter
    param := strings.TrimSpace(current.String())
    if param != "" {
        params = append(params, param)
    }
    
    return params
}
```

### Name Transformation Logic

```go
// transformSchemaName applies the naming strategy to generate a valid schema name.
func transformSchemaName(name string, config GenericNamingConfig) string {
    base, params, _ := parseGenericName(name)
    
    if len(params) == 0 {
        // Not a generic name, just sanitize any remaining invalid chars
        return sanitizeSchemaName(name)
    }
    
    // Recursively transform nested generic parameters
    transformedParams := make([]string, len(params))
    for i, param := range params {
        transformedParams[i] = transformSchemaName(param, config)
    }
    
    // Apply casing to parameters if configured
    if !config.PreserveCasing {
        for i, param := range transformedParams {
            transformedParams[i] = toPascalCase(param)
        }
    }
    
    // Apply the naming strategy
    switch config.Strategy {
    case GenericNamingOf:
        return base + "Of" + strings.Join(transformedParams, config.ParamSeparator+"Of")
        
    case GenericNamingFor:
        return base + "For" + strings.Join(transformedParams, config.ParamSeparator+"For")
        
    case GenericNamingFlattened:
        return base + strings.Join(transformedParams, "")
        
    case GenericNamingDot:
        return base + "." + strings.Join(transformedParams, ".")
        
    default: // GenericNamingUnderscore
        sep := config.Separator
        if sep == "" {
            sep = "_"
        }
        paramSep := config.ParamSeparator
        if paramSep == "" {
            paramSep = "_"
        }
        return base + sep + strings.Join(transformedParams, paramSep) + sep
    }
}

// sanitizeSchemaName removes or replaces any remaining invalid characters.
func sanitizeSchemaName(name string) string {
    var result strings.Builder
    for _, r := range name {
        if isValidSchemaNameChar(r) {
            result.WriteRune(r)
        } else {
            result.WriteRune('_')
        }
    }
    return result.String()
}

// isValidSchemaNameChar returns true if the character is valid in a schema name.
func isValidSchemaNameChar(r rune) bool {
    // Allow alphanumeric, underscore, hyphen, and dot
    return (r >= 'a' && r <= 'z') ||
        (r >= 'A' && r <= 'Z') ||
        (r >= '0' && r <= '9') ||
        r == '_' || r == '-' || r == '.'
}

// toPascalCase converts a string to PascalCase.
func toPascalCase(s string) string {
    // Handle snake_case and kebab-case
    words := strings.FieldsFunc(s, func(r rune) bool {
        return r == '_' || r == '-' || r == ' '
    })
    
    var result strings.Builder
    caser := cases.Title(language.English)
    for _, word := range words {
        result.WriteString(caser.String(strings.ToLower(word)))
    }
    
    if result.Len() == 0 {
        return s
    }
    return result.String()
}
```

### Fix Implementation

```go
// fixInvalidSchemaNames detects and renames schemas with invalid characters.
func (f *Fixer) fixInvalidSchemaNamesOAS2(doc *parser.OAS2Document, result *FixResult) {
    if doc.Definitions == nil || len(doc.Definitions) == 0 {
        return
    }
    
    // Build rename map: old name → new name
    renames := make(map[string]string)
    
    for name := range doc.Definitions {
        if hasInvalidSchemaNameChars(name) {
            newName := transformSchemaName(name, f.GenericNamingConfig)
            
            // Handle collision with existing schema
            newName = f.resolveNameCollision(newName, doc.Definitions, renames)
            
            renames[name] = newName
        }
    }
    
    if len(renames) == 0 {
        return
    }
    
    // Apply renames to the definitions map
    for oldName, newName := range renames {
        schema := doc.Definitions[oldName]
        delete(doc.Definitions, oldName)
        doc.Definitions[newName] = schema
        
        result.Fixes = append(result.Fixes, Fix{
            Type:        FixTypeInvalidSchemaName,
            Path:        fmt.Sprintf("definitions.%s", oldName),
            Description: fmt.Sprintf("Renamed schema '%s' to '%s' (invalid characters)", oldName, newName),
            Before:      oldName,
            After:       newName,
        })
        f.populateFixLocation(&result.Fixes[len(result.Fixes)-1])
    }
    
    // Rewrite all $refs to use the new names
    f.rewriteSchemaRefsOAS2(doc, renames, result)
}

// fixInvalidSchemaNames detects and renames schemas with invalid characters.
func (f *Fixer) fixInvalidSchemaNamesOAS3(doc *parser.OAS3Document, result *FixResult) {
    if doc.Components == nil || doc.Components.Schemas == nil || len(doc.Components.Schemas) == 0 {
        return
    }
    
    // Build rename map: old name → new name
    renames := make(map[string]string)
    
    for name := range doc.Components.Schemas {
        if hasInvalidSchemaNameChars(name) {
            newName := transformSchemaName(name, f.GenericNamingConfig)
            
            // Handle collision with existing schema
            newName = f.resolveNameCollision(newName, doc.Components.Schemas, renames)
            
            renames[name] = newName
        }
    }
    
    if len(renames) == 0 {
        return
    }
    
    // Apply renames to the schemas map
    for oldName, newName := range renames {
        schema := doc.Components.Schemas[oldName]
        delete(doc.Components.Schemas, oldName)
        doc.Components.Schemas[newName] = schema
        
        result.Fixes = append(result.Fixes, Fix{
            Type:        FixTypeInvalidSchemaName,
            Path:        fmt.Sprintf("components.schemas.%s", oldName),
            Description: fmt.Sprintf("Renamed schema '%s' to '%s' (invalid characters)", oldName, newName),
            Before:      oldName,
            After:       newName,
        })
        f.populateFixLocation(&result.Fixes[len(result.Fixes)-1])
    }
    
    // Rewrite all $refs to use the new names
    f.rewriteSchemaRefsOAS3(doc, renames, result)
}

// resolveNameCollision ensures the new name doesn't conflict with existing schemas.
func (f *Fixer) resolveNameCollision(newName string, schemas map[string]*parser.Schema, pendingRenames map[string]string) string {
    originalName := newName
    counter := 1
    
    for {
        // Check if name exists in schemas (excluding names being renamed away)
        _, existsInSchemas := schemas[newName]
        isBeingRenamed := false
        for oldName := range pendingRenames {
            if oldName == newName {
                isBeingRenamed = true
                break
            }
        }
        
        // Check if another rename is already targeting this name
        alreadyTargeted := false
        for _, targetName := range pendingRenames {
            if targetName == newName {
                alreadyTargeted = true
                break
            }
        }
        
        if (!existsInSchemas || isBeingRenamed) && !alreadyTargeted {
            return newName
        }
        
        // Append counter to resolve collision
        counter++
        newName = fmt.Sprintf("%s%d", originalName, counter)
    }
}
```

### Reference Rewriting

```go
// rewriteSchemaRefsOAS2 updates all $refs to use the new schema names.
func (f *Fixer) rewriteSchemaRefsOAS2(doc *parser.OAS2Document, renames map[string]string, result *FixResult) {
    // Build ref rewrite map: old ref → new ref
    refRenames := make(map[string]string)
    for oldName, newName := range renames {
        // Handle both encoded and unencoded refs
        oldRef := "#/definitions/" + oldName
        oldRefEncoded := "#/definitions/" + url.PathEscape(oldName)
        newRef := "#/definitions/" + newName
        
        refRenames[oldRef] = newRef
        if oldRefEncoded != oldRef {
            refRenames[oldRefEncoded] = newRef
        }
    }
    
    // Rewrite refs in definitions (schemas can reference each other)
    for _, schema := range doc.Definitions {
        rewriteSchemaRefs(schema, refRenames)
    }
    
    // Rewrite refs in parameters
    for _, param := range doc.Parameters {
        if param != nil && param.Schema != nil {
            rewriteSchemaRefs(param.Schema, refRenames)
        }
    }
    
    // Rewrite refs in responses
    for _, response := range doc.Responses {
        if response != nil && response.Schema != nil {
            rewriteSchemaRefs(response.Schema, refRenames)
        }
    }
    
    // Rewrite refs in paths
    for _, pathItem := range doc.Paths {
        if pathItem == nil {
            continue
        }
        
        rewritePathItemRefs(pathItem, refRenames, parser.OASVersion20)
    }
}

// rewriteSchemaRefsOAS3 updates all $refs to use the new schema names.
func (f *Fixer) rewriteSchemaRefsOAS3(doc *parser.OAS3Document, renames map[string]string, result *FixResult) {
    // Build ref rewrite map: old ref → new ref
    refRenames := make(map[string]string)
    for oldName, newName := range renames {
        // Handle both encoded and unencoded refs
        oldRef := "#/components/schemas/" + oldName
        oldRefEncoded := "#/components/schemas/" + url.PathEscape(oldName)
        newRef := "#/components/schemas/" + newName
        
        refRenames[oldRef] = newRef
        if oldRefEncoded != oldRef {
            refRenames[oldRefEncoded] = newRef
        }
    }
    
    // Rewrite refs in components
    if doc.Components != nil {
        for _, schema := range doc.Components.Schemas {
            rewriteSchemaRefs(schema, refRenames)
        }
        
        for _, param := range doc.Components.Parameters {
            rewriteParameterRefs(param, refRenames)
        }
        
        for _, response := range doc.Components.Responses {
            rewriteResponseRefs(response, refRenames)
        }
        
        for _, requestBody := range doc.Components.RequestBodies {
            rewriteRequestBodyRefs(requestBody, refRenames)
        }
        
        for _, header := range doc.Components.Headers {
            if header != nil && header.Schema != nil {
                rewriteSchemaRefs(header.Schema, refRenames)
            }
        }
        
        for _, callback := range doc.Components.Callbacks {
            rewriteCallbackRefs(callback, refRenames, doc.OASVersion)
        }
    }
    
    // Rewrite refs in paths
    for _, pathItem := range doc.Paths {
        rewritePathItemRefs(pathItem, refRenames, doc.OASVersion)
    }
    
    // Rewrite refs in webhooks (OAS 3.1+)
    for _, pathItem := range doc.Webhooks {
        rewritePathItemRefs(pathItem, refRenames, doc.OASVersion)
    }
}

// rewriteSchemaRefs recursively rewrites $ref values in a schema.
func rewriteSchemaRefs(schema *parser.Schema, renames map[string]string) {
    if schema == nil {
        return
    }
    
    // Rewrite direct ref
    if schema.Ref != "" {
        if newRef, ok := renames[schema.Ref]; ok {
            schema.Ref = newRef
        }
    }
    
    // Rewrite in properties
    for _, prop := range schema.Properties {
        rewriteSchemaRefs(prop, renames)
    }
    
    // Rewrite in patternProperties
    for _, prop := range schema.PatternProperties {
        rewriteSchemaRefs(prop, renames)
    }
    
    // Rewrite in additionalProperties
    if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
        rewriteSchemaRefs(addProps, renames)
    }
    
    // Rewrite in items
    if items, ok := schema.Items.(*parser.Schema); ok {
        rewriteSchemaRefs(items, renames)
    }
    
    // Rewrite in composition keywords
    for _, sub := range schema.AllOf {
        rewriteSchemaRefs(sub, renames)
    }
    for _, sub := range schema.AnyOf {
        rewriteSchemaRefs(sub, renames)
    }
    for _, sub := range schema.OneOf {
        rewriteSchemaRefs(sub, renames)
    }
    rewriteSchemaRefs(schema.Not, renames)
    
    // Rewrite in additional array keywords
    if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
        rewriteSchemaRefs(addItems, renames)
    }
    for _, item := range schema.PrefixItems {
        rewriteSchemaRefs(item, renames)
    }
    rewriteSchemaRefs(schema.Contains, renames)
    
    // Rewrite in object validation keywords
    rewriteSchemaRefs(schema.PropertyNames, renames)
    for _, dep := range schema.DependentSchemas {
        rewriteSchemaRefs(dep, renames)
    }
    
    // Rewrite in conditional keywords
    rewriteSchemaRefs(schema.If, renames)
    rewriteSchemaRefs(schema.Then, renames)
    rewriteSchemaRefs(schema.Else, renames)
    
    // Rewrite in $defs
    for _, def := range schema.Defs {
        rewriteSchemaRefs(def, renames)
    }
    
    // Rewrite discriminator mappings
    if schema.Discriminator != nil {
        for key, mapping := range schema.Discriminator.Mapping {
            if newRef, ok := renames[mapping]; ok {
                schema.Discriminator.Mapping[key] = newRef
            }
            // Also check for bare name references
            for oldName, newName := range extractBareNames(renames) {
                if mapping == oldName {
                    schema.Discriminator.Mapping[key] = newName
                }
            }
        }
    }
}

// extractBareNames extracts old→new mappings for bare names (without #/path/).
func extractBareNames(renames map[string]string) map[string]string {
    result := make(map[string]string)
    for oldRef, newRef := range renames {
        oldName := extractSchemaNameFromRef(oldRef)
        newName := extractSchemaNameFromRef(newRef)
        if oldName != "" && newName != "" {
            result[oldName] = newName
        }
    }
    return result
}

// extractSchemaNameFromRef extracts the schema name from a $ref path.
func extractSchemaNameFromRef(ref string) string {
    if strings.HasPrefix(ref, "#/components/schemas/") {
        return strings.TrimPrefix(ref, "#/components/schemas/")
    }
    if strings.HasPrefix(ref, "#/definitions/") {
        return strings.TrimPrefix(ref, "#/definitions/")
    }
    return ""
}
```

### Fixer Struct Updates

```go
// Fixer handles automatic fixing of OAS validation issues
type Fixer struct {
    // Existing fields
    InferTypes   bool
    EnabledFixes []FixType
    UserAgent    string
    SourceMap    *parser.SourceMap
    
    // New field for generic naming configuration
    GenericNamingConfig GenericNamingConfig
}

// New creates a new Fixer instance with default settings
func New() *Fixer {
    return &Fixer{
        InferTypes:          false,
        EnabledFixes:        nil, // all fixes enabled
        GenericNamingConfig: DefaultGenericNamingConfig(),
    }
}
```

### Functional Options

```go
// fixConfig updates
type fixConfig struct {
    // Existing fields...
    
    // Generic naming configuration
    genericNamingConfig GenericNamingConfig
}

// WithGenericNaming sets the naming strategy for fixing invalid schema names.
func WithGenericNaming(strategy GenericNamingStrategy) Option {
    return func(cfg *fixConfig) error {
        cfg.genericNamingConfig.Strategy = strategy
        return nil
    }
}

// WithGenericNamingConfig sets the full generic naming configuration.
func WithGenericNamingConfig(config GenericNamingConfig) Option {
    return func(cfg *fixConfig) error {
        cfg.genericNamingConfig = config
        return nil
    }
}

// WithGenericSeparator sets the separator for underscore strategy.
func WithGenericSeparator(sep string) Option {
    return func(cfg *fixConfig) error {
        cfg.genericNamingConfig.Separator = sep
        return nil
    }
}

// WithGenericParamSeparator sets the separator between multiple type parameters.
func WithGenericParamSeparator(sep string) Option {
    return func(cfg *fixConfig) error {
        cfg.genericNamingConfig.ParamSeparator = sep
        return nil
    }
}

// WithPreserveCasing disables automatic PascalCase conversion for type parameters.
func WithPreserveCasing(preserve bool) Option {
    return func(cfg *fixConfig) error {
        cfg.genericNamingConfig.PreserveCasing = preserve
        return nil
    }
}
```

### Integration with Fix Routing

```go
// In fixOAS2
func (f *Fixer) fixOAS2(parseResult parser.ParseResult, result *FixResult) (*FixResult, error) {
    // ... existing code ...
    
    // Apply enabled fixes
    if f.isFixEnabled(FixTypeMissingPathParameter) {
        f.fixMissingPathParametersOAS2(doc, result)
    }
    
    // Invalid schema names should be fixed BEFORE pruning
    // (so refs can be properly resolved for pruning analysis)
    if f.isFixEnabled(FixTypeInvalidSchemaName) {
        f.fixInvalidSchemaNamesOAS2(doc, result)
    }
    
    if f.isFixEnabled(FixTypePruneSchemas) {
        f.pruneOrphanSchemasOAS2(doc, result)
    }
    // ... etc
}

// Similar for fixOAS3
```

### CLI Integration

```go
// FixFlags updates
type FixFlags struct {
    // Existing flags...
    Output          string
    Infer           bool
    Quiet           bool
    SourceMap       bool
    PruneSchemas    bool
    PrunePaths      bool
    PruneComponents bool
    PruneAll        bool
    
    // New flags for generic naming
    FixSchemaNames       bool
    GenericNaming        string
    GenericSeparator     string
    GenericParamSeparator string
    PreserveCasing       bool
}

// SetupFixFlags updates
func SetupFixFlags() (*flag.FlagSet, *FixFlags) {
    fs := flag.NewFlagSet("fix", flag.ContinueOnError)
    flags := &FixFlags{}
    
    // Existing flags...
    
    // New generic naming flags
    fs.BoolVar(&flags.FixSchemaNames, "fix-schema-names", false, 
        "fix invalid schema names (brackets, special characters)")
    fs.StringVar(&flags.GenericNaming, "generic-naming", "underscore",
        "strategy for renaming generic types: underscore, of, for, flat, dot")
    fs.StringVar(&flags.GenericSeparator, "generic-separator", "_",
        "separator for underscore strategy")
    fs.StringVar(&flags.GenericParamSeparator, "generic-param-separator", "_",
        "separator between multiple type parameters")
    fs.BoolVar(&flags.PreserveCasing, "preserve-casing", false,
        "preserve original casing of type parameters")
    
    fs.Usage = func() {
        // ... existing usage ...
        cliutil.Writef(fs.Output(), "\nSchema Name Fixes:\n")
        cliutil.Writef(fs.Output(), "  --fix-schema-names       Fix invalid schema names (brackets, etc.)\n")
        cliutil.Writef(fs.Output(), "  --generic-naming <strat> Naming strategy: underscore, of, for, flat, dot\n")
        cliutil.Writef(fs.Output(), "  --generic-separator      Separator for underscore strategy (default: _)\n")
        cliutil.Writef(fs.Output(), "  --generic-param-separator Separator between type params (default: _)\n")
        cliutil.Writef(fs.Output(), "  --preserve-casing        Keep original type parameter casing\n")
        cliutil.Writef(fs.Output(), "\nGeneric Naming Strategy Examples:\n")
        cliutil.Writef(fs.Output(), "  underscore: Response[User] → Response_User_\n")
        cliutil.Writef(fs.Output(), "  of:         Response[User] → ResponseOfUser\n")
        cliutil.Writef(fs.Output(), "  for:        Response[User] → ResponseForUser\n")
        cliutil.Writef(fs.Output(), "  flat:       Response[User] → ResponseUser\n")
        cliutil.Writef(fs.Output(), "  dot:        Response[User] → Response.User\n")
    }
    
    return fs, flags
}

// HandleFix updates
func HandleFix(args []string) error {
    fs, flags := SetupFixFlags()
    // ... existing parsing ...
    
    // Build enabled fixes list
    var enabledFixes []fixer.FixType
    
    // ... existing fix type logic ...
    
    if flags.FixSchemaNames {
        enabledFixes = append(enabledFixes, fixer.FixTypeInvalidSchemaName)
    }
    
    // Parse generic naming strategy
    strategy, err := fixer.ParseGenericNamingStrategy(flags.GenericNaming)
    if err != nil {
        return fmt.Errorf("invalid generic naming strategy: %w", err)
    }
    
    genericConfig := fixer.GenericNamingConfig{
        Strategy:       strategy,
        Separator:      flags.GenericSeparator,
        ParamSeparator: flags.GenericParamSeparator,
        PreserveCasing: flags.PreserveCasing,
    }
    
    // Build fixer options
    fixOpts := []fixer.Option{
        fixer.WithFilePath(specPath),
        fixer.WithInferTypes(flags.Infer),
        fixer.WithGenericNamingConfig(genericConfig),
    }
    if len(enabledFixes) > 0 {
        fixOpts = append(fixOpts, fixer.WithEnabledFixes(enabledFixes...))
    }
    
    // ... rest of existing code ...
}
```

---

## Testing Strategy

### Unit Tests (generic_names_test.go)

```go
func TestHasInvalidSchemaNameChars(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected bool
    }{
        {"simple name", "User", false},
        {"with underscore", "User_Profile", false},
        {"with hyphen", "user-profile", false},
        {"with dot", "api.User", false},
        {"square brackets", "Response[User]", true},
        {"angle brackets", "Response<User>", true},
        {"comma", "Map[string,int]", true},
        {"space", "User Profile", true},
        {"curly braces", "Response{User}", true},
        {"encoded brackets", "Response%5BUser%5D", false}, // encoded is valid
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.expected, hasInvalidSchemaNameChars(tt.input))
        })
    }
}

func TestParseGenericName(t *testing.T) {
    tests := []struct {
        name          string
        input         string
        expectedBase  string
        expectedParams []string
        expectedStyle rune
    }{
        {"simple", "User", "User", nil, 0},
        {"single param square", "Response[User]", "Response", []string{"User"}, '['},
        {"multiple params square", "Map[string,int]", "Map", []string{"string", "int"}, '['},
        {"nested square", "Response[List[User]]", "Response", []string{"List[User]"}, '['},
        {"single param angle", "Result<T>", "Result", []string{"T"}, '<'},
        {"multiple params angle", "Either<L,R>", "Either", []string{"L", "R"}, '<'},
        {"complex nested", "Response[Map[string,List[User]]]", "Response", []string{"Map[string,List[User]]"}, '['},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            base, params, style := parseGenericName(tt.input)
            assert.Equal(t, tt.expectedBase, base)
            assert.Equal(t, tt.expectedParams, params)
            assert.Equal(t, tt.expectedStyle, style)
        })
    }
}

func TestTransformSchemaName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        strategy GenericNamingStrategy
        expected string
    }{
        // Underscore strategy
        {"underscore simple", "Response[User]", GenericNamingUnderscore, "Response_User_"},
        {"underscore multiple", "Map[string,int]", GenericNamingUnderscore, "Map_String_Int_"},
        {"underscore nested", "Response[List[User]]", GenericNamingUnderscore, "Response_List_User__"},
        
        // Of strategy
        {"of simple", "Response[User]", GenericNamingOf, "ResponseOfUser"},
        {"of multiple", "Map[string,int]", GenericNamingOf, "MapOfStringOfInt"},
        {"of nested", "Response[List[User]]", GenericNamingOf, "ResponseOfListOfUser"},
        
        // For strategy
        {"for simple", "Response[User]", GenericNamingFor, "ResponseForUser"},
        {"for multiple", "Map[string,int]", GenericNamingFor, "MapForStringForInt"},
        
        // Flat strategy
        {"flat simple", "Response[User]", GenericNamingFlattened, "ResponseUser"},
        {"flat multiple", "Map[string,int]", GenericNamingFlattened, "MapStringInt"},
        
        // Dot strategy
        {"dot simple", "Response[User]", GenericNamingDot, "Response.User"},
        {"dot multiple", "Map[string,int]", GenericNamingDot, "Map.String.Int"},
        
        // Non-generic names
        {"plain name", "User", GenericNamingOf, "User"},
        
        // Angle brackets
        {"angle brackets", "Result<T>", GenericNamingOf, "ResultOfT"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            config := GenericNamingConfig{
                Strategy:       tt.strategy,
                Separator:      "_",
                ParamSeparator: "_",
                PreserveCasing: false,
            }
            assert.Equal(t, tt.expected, transformSchemaName(tt.input, config))
        })
    }
}

func TestTransformSchemaName_PreserveCasing(t *testing.T) {
    config := GenericNamingConfig{
        Strategy:       GenericNamingOf,
        PreserveCasing: true,
    }
    
    result := transformSchemaName("Response[user_data]", config)
    assert.Equal(t, "ResponseOfuser_data", result)
    
    config.PreserveCasing = false
    result = transformSchemaName("Response[user_data]", config)
    assert.Equal(t, "ResponseOfUserData", result)
}

func TestTransformSchemaName_CustomSeparators(t *testing.T) {
    config := GenericNamingConfig{
        Strategy:       GenericNamingUnderscore,
        Separator:      "__",
        ParamSeparator: "_",
    }
    
    result := transformSchemaName("Map[string,int]", config)
    assert.Equal(t, "Map__String_Int__", result)
}
```

### Integration Tests

```go
func TestFixInvalidSchemaNames_OAS3(t *testing.T) {
    spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Response%5BUser%5D'
components:
  schemas:
    Response[User]:
      type: object
      properties:
        data:
          $ref: '#/components/schemas/User'
    User:
      type: object
`
    p := parser.New()
    parseResult, err := p.ParseBytes([]byte(spec))
    require.NoError(t, err)
    
    result, err := fixer.FixWithOptions(
        fixer.WithParsed(*parseResult),
        fixer.WithEnabledFixes(fixer.FixTypeInvalidSchemaName),
        fixer.WithGenericNaming(fixer.GenericNamingOf),
    )
    require.NoError(t, err)
    
    doc := result.Document.(*parser.OAS3Document)
    
    // Schema should be renamed
    assert.NotContains(t, doc.Components.Schemas, "Response[User]")
    assert.Contains(t, doc.Components.Schemas, "ResponseOfUser")
    
    // User should remain unchanged
    assert.Contains(t, doc.Components.Schemas, "User")
    
    // Check that ref was rewritten
    responseSchema := doc.Paths["/users"].Get.Responses.Codes["200"].Content["application/json"].Schema
    assert.Equal(t, "#/components/schemas/ResponseOfUser", responseSchema.Ref)
}

func TestFixInvalidSchemaNames_EncodedRef(t *testing.T) {
    // Test that encoded refs are properly handled
    spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: 1.0.0
paths:
  /data:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Map%5Bstring%2Cint%5D'
components:
  schemas:
    Map[string,int]:
      type: object
`
    p := parser.New()
    parseResult, err := p.ParseBytes([]byte(spec))
    require.NoError(t, err)
    
    result, err := fixer.FixWithOptions(
        fixer.WithParsed(*parseResult),
        fixer.WithEnabledFixes(fixer.FixTypeInvalidSchemaName),
    )
    require.NoError(t, err)
    
    doc := result.Document.(*parser.OAS3Document)
    
    // Schema should be renamed
    assert.NotContains(t, doc.Components.Schemas, "Map[string,int]")
    assert.Contains(t, doc.Components.Schemas, "Map_String_Int_")
    
    // Encoded ref should be rewritten
    responseSchema := doc.Paths["/data"].Get.Responses.Codes["200"].Content["application/json"].Schema
    assert.Equal(t, "#/components/schemas/Map_String_Int_", responseSchema.Ref)
}

func TestFixInvalidSchemaNames_Collision(t *testing.T) {
    // Test name collision handling
    spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Response[User]:
      type: object
    ResponseOfUser:
      type: object
      description: Already exists
`
    p := parser.New()
    parseResult, err := p.ParseBytes([]byte(spec))
    require.NoError(t, err)
    
    result, err := fixer.FixWithOptions(
        fixer.WithParsed(*parseResult),
        fixer.WithEnabledFixes(fixer.FixTypeInvalidSchemaName),
        fixer.WithGenericNaming(fixer.GenericNamingOf),
    )
    require.NoError(t, err)
    
    doc := result.Document.(*parser.OAS3Document)
    
    // Both should exist with unique names
    assert.Contains(t, doc.Components.Schemas, "ResponseOfUser")  // Original
    assert.Contains(t, doc.Components.Schemas, "ResponseOfUser2") // Renamed with suffix
}

func TestFixInvalidSchemaNames_TransitiveRefs(t *testing.T) {
    // Test that schemas referencing renamed schemas are updated
    spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Response[User]:
      type: object
      properties:
        data:
          $ref: '#/components/schemas/List[User]'
    List[User]:
      type: array
      items:
        $ref: '#/components/schemas/User'
    User:
      type: object
`
    p := parser.New()
    parseResult, err := p.ParseBytes([]byte(spec))
    require.NoError(t, err)
    
    result, err := fixer.FixWithOptions(
        fixer.WithParsed(*parseResult),
        fixer.WithEnabledFixes(fixer.FixTypeInvalidSchemaName),
        fixer.WithGenericNaming(fixer.GenericNamingOf),
    )
    require.NoError(t, err)
    
    doc := result.Document.(*parser.OAS3Document)
    
    // All refs should be updated
    responseSchema := doc.Components.Schemas["ResponseOfUser"]
    assert.Equal(t, "#/components/schemas/ListOfUser", responseSchema.Properties["data"].Ref)
}
```

### Benchmark Tests

```go
func BenchmarkTransformSchemaName(b *testing.B) {
    config := DefaultGenericNamingConfig()
    names := []string{
        "Response[User]",
        "Map[string,int]",
        "Response[List[Map[string,User]]]",
    }
    
    b.ResetTimer()
    for b.Loop() {
        for _, name := range names {
            transformSchemaName(name, config)
        }
    }
}

func BenchmarkFixInvalidSchemaNames_Large(b *testing.B) {
    // Build document with many generic schemas
    doc := buildDocWithGenericSchemas(100)
    
    b.ResetTimer()
    for b.Loop() {
        f := New()
        f.GenericNamingConfig = DefaultGenericNamingConfig()
        result := &FixResult{Fixes: make([]Fix, 0)}
        docCopy, _ := deepCopyOAS3Document(doc)
        f.fixInvalidSchemaNamesOAS3(docCopy, result)
    }
}
```

---

## Edge Cases and Special Handling

### Nested Generic Types

Handle deeply nested generics:

```
Response[List[Map[string,User]]]
→ ResponseOfListOfMapOfStringOfUser (with "of" strategy)
→ Response_List_Map_string_User__ (with underscore strategy)
```

### Mixed Bracket Styles

Some generators might mix styles:

```
Response<List[User]>
```

The implementation should handle this by processing the outermost brackets first.

### URL-Encoded References

Both encoded and unencoded refs must be handled:

```yaml
# These should all resolve to the same schema:
$ref: '#/components/schemas/Response[User]'
$ref: '#/components/schemas/Response%5BUser%5D'
```

### Discriminator Bare Names

Discriminator mappings can use bare names:

```yaml
discriminator:
  propertyName: type
  mapping:
    user: Response[User]       # bare name
    admin: '#/components/schemas/Response[Admin]'  # full ref
```

Both must be rewritten.

### Name Collisions

When the transformed name already exists:

```yaml
Response[User]:     # Would become ResponseOfUser
ResponseOfUser:     # Already exists!
```

Resolution: Append numeric suffix (`ResponseOfUser2`, `ResponseOfUser3`, etc.)

---

## Documentation Updates

### doc.go Updates

```go
// # Supported Fixes
//
// ...existing fixes...
//
//   - Invalid schema names: Renames schema definitions containing characters
//     that require URL encoding (brackets, spaces, etc.) to valid names.
//     This commonly occurs with generic types from third-party generators.
//     Configurable naming strategies match the builder package:
//       * underscore: Response[User] → Response_User_
//       * of:         Response[User] → ResponseOfUser
//       * for:        Response[User] → ResponseForUser
//       * flat:       Response[User] → ResponseUser
//       * dot:        Response[User] → Response.User
```

### CLI Reference Updates

```markdown
### Schema Name Fixes

| Flag | Description |
|------|-------------|
| `--fix-schema-names` | Fix invalid schema names containing brackets or special characters |
| `--generic-naming <strategy>` | Naming strategy: `underscore`, `of`, `for`, `flat`, `dot` |
| `--generic-separator` | Separator for underscore strategy (default: `_`) |
| `--generic-param-separator` | Separator between type parameters (default: `_`) |
| `--preserve-casing` | Keep original casing of type parameters |

### Generic Naming Examples

```bash
# Fix generic type names with default strategy (underscore)
oastools fix --fix-schema-names api.yaml -o fixed.yaml
# Response[User] → Response_User_

# Use "Of" naming convention
oastools fix --fix-schema-names --generic-naming of api.yaml
# Response[User] → ResponseOfUser

# Use "For" naming convention
oastools fix --fix-schema-names --generic-naming for api.yaml
# Response[User] → ResponseForUser

# Flatten brackets completely
oastools fix --fix-schema-names --generic-naming flat api.yaml
# Response[User] → ResponseUser

# Custom separators
oastools fix --fix-schema-names --generic-separator "__" api.yaml
# Response[User] → Response__User__

# Preserve original casing
oastools fix --fix-schema-names --preserve-casing api.yaml
# Response[user_data] → Response_user_data_ (not Response_UserData_)
```
```

---

## Implementation Checklist

### Phase 1: Core Detection and Transformation

- [ ] Create `fixer/generic_names.go`
- [ ] Implement `GenericNamingStrategy` enum with String() and Parse()
- [ ] Implement `GenericNamingConfig` struct with defaults
- [ ] Implement `hasInvalidSchemaNameChars()`
- [ ] Implement `parseGenericName()`
- [ ] Implement `splitTypeParams()` with nesting support
- [ ] Implement `transformSchemaName()`
- [ ] Implement `sanitizeSchemaName()`
- [ ] Implement `toPascalCase()`
- [ ] Add comprehensive unit tests

### Phase 2: Fix Implementation

- [ ] Add `FixTypeInvalidSchemaName` constant
- [ ] Add `GenericNamingConfig` field to `Fixer` struct
- [ ] Implement `fixInvalidSchemaNamesOAS2()`
- [ ] Implement `fixInvalidSchemaNamesOAS3()`
- [ ] Implement `resolveNameCollision()`
- [ ] Add integration tests

### Phase 3: Reference Rewriting

- [ ] Implement `rewriteSchemaRefsOAS2()`
- [ ] Implement `rewriteSchemaRefsOAS3()`
- [ ] Implement `rewriteSchemaRefs()` recursive helper
- [ ] Handle both encoded and unencoded refs
- [ ] Handle discriminator bare names
- [ ] Add reference rewriting tests

### Phase 4: Functional Options

- [ ] Add `WithGenericNaming()` option
- [ ] Add `WithGenericNamingConfig()` option
- [ ] Add `WithGenericSeparator()` option
- [ ] Add `WithGenericParamSeparator()` option
- [ ] Add `WithPreserveCasing()` option
- [ ] Update `applyOptions()` to copy config to Fixer
- [ ] Add option tests

### Phase 5: CLI Integration

- [ ] Add new flags to `FixFlags` struct
- [ ] Update `SetupFixFlags()` with new flags and usage
- [ ] Update `HandleFix()` to process flags
- [ ] Add CLI integration tests

### Phase 6: Integration with Fix Routing

- [ ] Update `fixOAS2()` to call `fixInvalidSchemaNamesOAS2()`
- [ ] Update `fixOAS3()` to call `fixInvalidSchemaNamesOAS3()`
- [ ] Ensure fix order: invalid names before pruning
- [ ] Add end-to-end tests

### Phase 7: Documentation and Examples

- [ ] Update `fixer/doc.go`
- [ ] Add `Example_fixSchemaNames()` to `example_test.go`
- [ ] Update `docs/cli-reference.md`
- [ ] Update `docs/developer-guide.md`

### Phase 8: Benchmarks and Validation

- [ ] Add benchmark tests
- [ ] Test against real-world specs with generic types
- [ ] Run `make check`
- [ ] Verify all tests pass

---

## Success Criteria

1. Detection correctly identifies all invalid schema name characters
2. All naming strategies produce valid, URL-safe schema names
3. Nested generic types are handled correctly
4. Both encoded and unencoded `$ref` values are rewritten
5. Name collisions are resolved with numeric suffixes
6. Discriminator mappings (both bare and full path) are updated
7. Naming strategies match `builder` package behavior for consistency
8. All existing tests continue to pass
9. Performance is acceptable for documents with many generic schemas

---

## Risk Mitigation

### Risk: Missed Reference Locations

Mitigation: Comprehensive reference rewriting covering all documented `$ref` locations. Reuse patterns from converter and joiner packages.

### Risk: Breaking Valid Schemas

Mitigation: Only rename schemas that actually contain invalid characters. Schemas with URL-encoded names (already valid) are not modified.

### Risk: Inconsistent Naming with Builder

Mitigation: Use identical strategy names and default behaviors. Consider extracting shared naming logic to internal package in future.

### Risk: Circular Reference Handling

Mitigation: Process renames in a single pass (collect all renames, then apply). Reference rewriting doesn't need cycle detection since it's a simple string replacement.
