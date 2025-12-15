# Implementation Plan: Pruning Features for the Fixer Package

This plan provides a single-phase implementation roadmap for enhancing the `fixer` package with pruning capabilities to remove orphaned schemas, empty paths, and other unreferenced definitions from OpenAPI documents. The implementation is scoped for completion in a single Claude Code session.

---

## Executive Summary

Documents produced by inferior joining tools or manual editing often contain orphaned definitions that are no longer referenced by any operation. These artifacts increase document size, create confusion during maintenance, and can cause downstream tooling issues. This enhancement introduces three interconnected pruning capabilities: schema pruning to remove unreferenced definitions, path pruning to remove empty path items, and comprehensive component pruning for all component types in OAS 3.x documents.

---

## Scope and Objectives

The implementation delivers five interconnected capabilities:

1. **Schema Pruning** (`PruneSchemas`/`prune-schemas` fix type) that identifies and removes schema definitions not referenced anywhere in the document, including transitive reference analysis.

2. **Path Pruning** (`PrunePaths`/`prune-paths` fix type) that removes path items with no operations defined.

3. **Component Pruning** (`PruneComponents`/`prune-components` fix type) for OAS 3.x that removes unreferenced parameters, responses, request bodies, headers, security schemes, links, callbacks, examples, and path items from the components object.

4. **Empty Container Cleanup** that removes empty definition containers (empty `definitions` in OAS 2.0, empty `components` sub-objects in OAS 3.x) after pruning.

5. **CLI Integration** with new flags for selective pruning and a comprehensive `--prune-all` option.

---

## Architecture Overview

The implementation introduces two new source files and modifies five existing files within the `fixer` package. The reference collection logic is extracted into a dedicated `refs.go` file to enable comprehensive reference discovery across all document locations. The pruning logic resides in `prune.go` with version-specific implementations.

```
fixer/
├── fixer.go           (modified: new fix types, routing to prune functions)
├── oas2.go            (modified: integrate prune calls for OAS 2.0)
├── oas3.go            (modified: integrate prune calls for OAS 3.x)
├── refs.go            (new: reference collection and analysis)
├── prune.go           (new: pruning implementations)
├── deep_copy.go       (existing: unchanged)
├── type_inference.go  (existing: unchanged)
├── fixer_test.go      (modified: new pruning tests)
├── refs_test.go       (new: reference collection tests)
├── prune_test.go      (new: pruning tests)
├── example_test.go    (modified: add pruning examples)
└── doc.go             (modified: document new fix types)

cmd/oastools/commands/
└── fix.go             (modified: new CLI flags)
```

---

## Detailed Implementation Specifications

### New Fix Types

The `FixType` constants gain three new values representing the pruning operations.

```go
const (
    // Existing
    FixTypeMissingPathParameter FixType = "missing-path-parameter"
    
    // New pruning fix types
    FixTypePruneSchemas    FixType = "prune-schemas"
    FixTypePrunePaths      FixType = "prune-paths"
    FixTypePruneComponents FixType = "prune-components"
)
```

### Reference Collection Engine (refs.go)

The `refs.go` file implements comprehensive reference discovery. The `RefCollector` struct tracks all `$ref` values encountered during document traversal, categorized by reference type.

```go
// RefType identifies the category of reference
type RefType int

const (
    RefTypeSchema RefType = iota
    RefTypeParameter
    RefTypeResponse
    RefTypeRequestBody
    RefTypeHeader
    RefTypeSecurityScheme
    RefTypeLink
    RefTypeCallback
    RefTypeExample
    RefTypePathItem
)

// RefCollector gathers all references in a document
type RefCollector struct {
    // Refs maps reference paths to their locations in the document
    // Key: normalized reference path (e.g., "#/components/schemas/Pet")
    // Value: list of JSON paths where the reference appears
    Refs map[string][]string
    
    // RefsByType categorizes references by their target type
    RefsByType map[RefType]map[string]bool
    
    // visited tracks processed schemas for circular reference handling
    visited map[*parser.Schema]bool
}

// NewRefCollector creates a new reference collector
func NewRefCollector() *RefCollector {
    return &RefCollector{
        Refs:       make(map[string][]string),
        RefsByType: make(map[RefType]map[string]bool),
        visited:    make(map[*parser.Schema]bool),
    }
}

// CollectOAS2 collects all references from an OAS 2.0 document
func (c *RefCollector) CollectOAS2(doc *parser.OAS2Document) {
    // Implementation collects refs from:
    // - paths (all operations, parameters, responses)
    // - definitions (schema refs including nested)
    // - parameters (top-level)
    // - responses (top-level)
    // - securityDefinitions
}

// CollectOAS3 collects all references from an OAS 3.x document
func (c *RefCollector) CollectOAS3(doc *parser.OAS3Document) {
    // Implementation collects refs from:
    // - paths (all operations, parameters, request bodies, responses, callbacks)
    // - webhooks (OAS 3.1+)
    // - components (all sub-objects)
    // - security requirements
}

// IsSchemaReferenced returns true if the schema name is referenced
func (c *RefCollector) IsSchemaReferenced(name string, version parser.OASVersion) bool {
    var prefix string
    if version == parser.OASVersion20 {
        prefix = "#/definitions/"
    } else {
        prefix = "#/components/schemas/"
    }
    _, ok := c.RefsByType[RefTypeSchema][prefix+name]
    return ok
}

// IsParameterReferenced returns true if the parameter name is referenced
func (c *RefCollector) IsParameterReferenced(name string, version parser.OASVersion) bool {
    // Similar implementation for parameters
}

// Additional Is*Referenced methods for each component type
```

#### Reference Collection Coverage

The collector must traverse all locations where `$ref` values can appear:

**Schema Locations:**
- `properties` map values
- `patternProperties` map values (OAS 3.1+)
- `additionalProperties` (when *Schema, not bool)
- `items` (when *Schema, not bool in OAS 3.1+)
- `allOf`, `anyOf`, `oneOf` arrays
- `not` schema
- `additionalItems` (when *Schema)
- `prefixItems` array (OAS 3.1+)
- `contains` schema (OAS 3.1+)
- `propertyNames` schema (OAS 3.1+)
- `dependentSchemas` map values (OAS 3.1+)
- `if`, `then`, `else` schemas (OAS 3.1+)
- `$defs` map values (OAS 3.1+)
- Discriminator `mapping` values

**Parameter Locations:**
- Path item `parameters` array
- Operation `parameters` array
- Components `parameters` map (OAS 3.x)
- Top-level `parameters` map (OAS 2.0)
- Parameter `schema` field

**Response Locations:**
- Operation `responses` (default and codes)
- Components `responses` map (OAS 3.x)
- Top-level `responses` map (OAS 2.0)
- Response `content` media types
- Response `headers` map
- Response `links` map (OAS 3.x)

**Request Body Locations:**
- Operation `requestBody` (OAS 3.x)
- Components `requestBodies` map (OAS 3.x)
- Request body `content` media types

**Header Locations:**
- Response `headers` map
- Components `headers` map (OAS 3.x)
- Encoding `headers` map (OAS 3.x)

**Callback Locations:**
- Operation `callbacks` map (OAS 3.x)
- Components `callbacks` map (OAS 3.x)
- Callback path items (recursive)

**Link Locations:**
- Response `links` map (OAS 3.x)
- Components `links` map (OAS 3.x)

**Security Scheme Locations:**
- Security requirements (operation and document level)
- Components `securitySchemes` map (OAS 3.x)
- Top-level `securityDefinitions` map (OAS 2.0)

**Example Locations:**
- Media type `examples` map (OAS 3.x)
- Parameter `examples` map (OAS 3.x)
- Components `examples` map (OAS 3.x)

**Path Item Locations:**
- Paths map values
- Webhooks map values (OAS 3.1+)
- Components `pathItems` map (OAS 3.1+)
- Callback map values

### Pruning Implementation (prune.go)

The `prune.go` file contains the pruning logic with separate functions for each fix type and OAS version.

```go
// pruneOrphanSchemasOAS2 removes unreferenced schemas from definitions
func (f *Fixer) pruneOrphanSchemasOAS2(doc *parser.OAS2Document, result *FixResult) {
    if doc.Definitions == nil || len(doc.Definitions) == 0 {
        return
    }
    
    // Collect all references
    collector := NewRefCollector()
    collector.CollectOAS2(doc)
    
    // Build transitive closure of referenced schemas
    referenced := f.buildReferencedSchemaSet(collector, doc.Definitions, parser.OASVersion20)
    
    // Remove unreferenced schemas
    for name := range doc.Definitions {
        if !referenced[name] {
            delete(doc.Definitions, name)
            result.Fixes = append(result.Fixes, Fix{
                Type:        FixTypePruneSchemas,
                Path:        fmt.Sprintf("definitions.%s", name),
                Description: fmt.Sprintf("Removed unreferenced schema '%s'", name),
                Before:      doc.Definitions[name],
                After:       nil,
            })
            f.populateFixLocation(&result.Fixes[len(result.Fixes)-1])
        }
    }
    
    // Clean up empty definitions map
    if len(doc.Definitions) == 0 {
        doc.Definitions = nil
    }
}

// pruneOrphanSchemasOAS3 removes unreferenced schemas from components.schemas
func (f *Fixer) pruneOrphanSchemasOAS3(doc *parser.OAS3Document, result *FixResult) {
    if doc.Components == nil || doc.Components.Schemas == nil || len(doc.Components.Schemas) == 0 {
        return
    }
    
    // Collect all references
    collector := NewRefCollector()
    collector.CollectOAS3(doc)
    
    // Build transitive closure of referenced schemas
    referenced := f.buildReferencedSchemaSet(collector, doc.Components.Schemas, doc.OASVersion)
    
    // Remove unreferenced schemas
    for name := range doc.Components.Schemas {
        if !referenced[name] {
            delete(doc.Components.Schemas, name)
            result.Fixes = append(result.Fixes, Fix{
                Type:        FixTypePruneSchemas,
                Path:        fmt.Sprintf("components.schemas.%s", name),
                Description: fmt.Sprintf("Removed unreferenced schema '%s'", name),
                Before:      doc.Components.Schemas[name],
                After:       nil,
            })
            f.populateFixLocation(&result.Fixes[len(result.Fixes)-1])
        }
    }
    
    // Clean up empty schemas map
    if len(doc.Components.Schemas) == 0 {
        doc.Components.Schemas = nil
    }
}

// buildReferencedSchemaSet builds the transitive closure of referenced schemas
// starting from operation-level references and following schema-to-schema references
func (f *Fixer) buildReferencedSchemaSet(collector *RefCollector, schemas map[string]*parser.Schema, version parser.OASVersion) map[string]bool {
    referenced := make(map[string]bool)
    
    // Get directly referenced schemas from collector
    prefix := "#/definitions/"
    if version >= parser.OASVersion300 {
        prefix = "#/components/schemas/"
    }
    
    // Queue for processing transitive references
    queue := make([]string, 0)
    
    for ref := range collector.RefsByType[RefTypeSchema] {
        if strings.HasPrefix(ref, prefix) {
            name := strings.TrimPrefix(ref, prefix)
            if _, exists := schemas[name]; exists {
                if !referenced[name] {
                    referenced[name] = true
                    queue = append(queue, name)
                }
            }
        }
    }
    
    // Process transitive references (schemas referencing other schemas)
    for len(queue) > 0 {
        name := queue[0]
        queue = queue[1:]
        
        schema := schemas[name]
        if schema == nil {
            continue
        }
        
        // Collect refs from this schema
        schemaRefs := collectSchemaRefs(schema, prefix)
        for _, refName := range schemaRefs {
            if _, exists := schemas[refName]; exists {
                if !referenced[refName] {
                    referenced[refName] = true
                    queue = append(queue, refName)
                }
            }
        }
    }
    
    return referenced
}

// collectSchemaRefs extracts all schema reference names from a schema
func collectSchemaRefs(schema *parser.Schema, prefix string) []string {
    if schema == nil {
        return nil
    }
    
    var refs []string
    
    // Direct ref
    if schema.Ref != "" && strings.HasPrefix(schema.Ref, prefix) {
        refs = append(refs, strings.TrimPrefix(schema.Ref, prefix))
    }
    
    // Properties
    for _, prop := range schema.Properties {
        refs = append(refs, collectSchemaRefs(prop, prefix)...)
    }
    
    // PatternProperties
    for _, prop := range schema.PatternProperties {
        refs = append(refs, collectSchemaRefs(prop, prefix)...)
    }
    
    // AdditionalProperties
    if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
        refs = append(refs, collectSchemaRefs(addProps, prefix)...)
    }
    
    // Items
    if items, ok := schema.Items.(*parser.Schema); ok {
        refs = append(refs, collectSchemaRefs(items, prefix)...)
    }
    
    // Composition keywords
    for _, sub := range schema.AllOf {
        refs = append(refs, collectSchemaRefs(sub, prefix)...)
    }
    for _, sub := range schema.AnyOf {
        refs = append(refs, collectSchemaRefs(sub, prefix)...)
    }
    for _, sub := range schema.OneOf {
        refs = append(refs, collectSchemaRefs(sub, prefix)...)
    }
    refs = append(refs, collectSchemaRefs(schema.Not, prefix)...)
    
    // Additional array keywords
    if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
        refs = append(refs, collectSchemaRefs(addItems, prefix)...)
    }
    for _, item := range schema.PrefixItems {
        refs = append(refs, collectSchemaRefs(item, prefix)...)
    }
    refs = append(refs, collectSchemaRefs(schema.Contains, prefix)...)
    
    // Object validation keywords
    refs = append(refs, collectSchemaRefs(schema.PropertyNames, prefix)...)
    for _, dep := range schema.DependentSchemas {
        refs = append(refs, collectSchemaRefs(dep, prefix)...)
    }
    
    // Conditional keywords
    refs = append(refs, collectSchemaRefs(schema.If, prefix)...)
    refs = append(refs, collectSchemaRefs(schema.Then, prefix)...)
    refs = append(refs, collectSchemaRefs(schema.Else, prefix)...)
    
    // Schema definitions
    for _, def := range schema.Defs {
        refs = append(refs, collectSchemaRefs(def, prefix)...)
    }
    
    // Discriminator mapping
    if schema.Discriminator != nil {
        for _, mapping := range schema.Discriminator.Mapping {
            if strings.HasPrefix(mapping, prefix) {
                refs = append(refs, strings.TrimPrefix(mapping, prefix))
            } else if !strings.HasPrefix(mapping, "#") {
                // Bare name reference
                refs = append(refs, mapping)
            }
        }
    }
    
    return refs
}
```

#### Path Pruning

```go
// pruneEmptyPaths removes path items that have no operations defined
func (f *Fixer) pruneEmptyPaths(paths parser.Paths, result *FixResult, version parser.OASVersion) {
    if paths == nil {
        return
    }
    
    for pattern, pathItem := range paths {
        if pathItem == nil || isPathItemEmpty(pathItem, version) {
            delete(paths, pattern)
            result.Fixes = append(result.Fixes, Fix{
                Type:        FixTypePrunePaths,
                Path:        fmt.Sprintf("paths.%s", pattern),
                Description: fmt.Sprintf("Removed empty path '%s'", pattern),
                Before:      pathItem,
                After:       nil,
            })
            f.populateFixLocation(&result.Fixes[len(result.Fixes)-1])
        }
    }
}

// isPathItemEmpty returns true if the path item has no operations
func isPathItemEmpty(pathItem *parser.PathItem, version parser.OASVersion) bool {
    if pathItem == nil {
        return true
    }
    
    // Check for $ref (not empty if referencing another path item)
    if pathItem.Ref != "" {
        return false
    }
    
    // Check all HTTP methods
    if pathItem.Get != nil || pathItem.Put != nil || pathItem.Post != nil ||
        pathItem.Delete != nil || pathItem.Options != nil || pathItem.Head != nil ||
        pathItem.Patch != nil {
        return false
    }
    
    // OAS 3.0+ methods
    if version >= parser.OASVersion300 && pathItem.Trace != nil {
        return false
    }
    
    // OAS 3.2+ methods
    if version >= parser.OASVersion320 && pathItem.Query != nil {
        return false
    }
    
    return true
}
```

#### Component Pruning (OAS 3.x)

```go
// pruneOrphanComponentsOAS3 removes unreferenced components
func (f *Fixer) pruneOrphanComponentsOAS3(doc *parser.OAS3Document, result *FixResult) {
    if doc.Components == nil {
        return
    }
    
    collector := NewRefCollector()
    collector.CollectOAS3(doc)
    
    // Prune parameters
    f.pruneUnreferencedParameters(doc.Components, collector, result, doc.OASVersion)
    
    // Prune responses
    f.pruneUnreferencedResponses(doc.Components, collector, result, doc.OASVersion)
    
    // Prune request bodies
    f.pruneUnreferencedRequestBodies(doc.Components, collector, result)
    
    // Prune headers
    f.pruneUnreferencedHeaders(doc.Components, collector, result)
    
    // Prune security schemes
    f.pruneUnreferencedSecuritySchemes(doc, collector, result)
    
    // Prune links
    f.pruneUnreferencedLinks(doc.Components, collector, result)
    
    // Prune callbacks
    f.pruneUnreferencedCallbacks(doc.Components, collector, result)
    
    // Prune examples
    f.pruneUnreferencedExamples(doc.Components, collector, result)
    
    // Prune path items (OAS 3.1+)
    if doc.OASVersion >= parser.OASVersion310 {
        f.pruneUnreferencedPathItems(doc.Components, collector, result)
    }
    
    // Clean up empty components object
    f.cleanupEmptyComponents(doc)
}

// pruneUnreferencedParameters removes unreferenced parameters from components
func (f *Fixer) pruneUnreferencedParameters(components *parser.Components, collector *RefCollector, result *FixResult, version parser.OASVersion) {
    if components.Parameters == nil || len(components.Parameters) == 0 {
        return
    }
    
    for name := range components.Parameters {
        if !collector.IsParameterReferenced(name, version) {
            delete(components.Parameters, name)
            result.Fixes = append(result.Fixes, Fix{
                Type:        FixTypePruneComponents,
                Path:        fmt.Sprintf("components.parameters.%s", name),
                Description: fmt.Sprintf("Removed unreferenced parameter '%s'", name),
                Before:      components.Parameters[name],
                After:       nil,
            })
            f.populateFixLocation(&result.Fixes[len(result.Fixes)-1])
        }
    }
    
    if len(components.Parameters) == 0 {
        components.Parameters = nil
    }
}

// Similar implementations for other component types...

// cleanupEmptyComponents removes the components object if all sub-objects are empty
func (f *Fixer) cleanupEmptyComponents(doc *parser.OAS3Document) {
    if doc.Components == nil {
        return
    }
    
    c := doc.Components
    if len(c.Schemas) == 0 && len(c.Responses) == 0 && len(c.Parameters) == 0 &&
        len(c.Examples) == 0 && len(c.RequestBodies) == 0 && len(c.Headers) == 0 &&
        len(c.SecuritySchemes) == 0 && len(c.Links) == 0 && len(c.Callbacks) == 0 &&
        len(c.PathItems) == 0 {
        doc.Components = nil
    }
}
```

### Integration with Existing Fixer (fixer.go modifications)

```go
// Add to fixOAS2 function
func (f *Fixer) fixOAS2(parseResult parser.ParseResult, result *FixResult) (*FixResult, error) {
    // ... existing code ...
    
    // Apply enabled fixes
    if f.isFixEnabled(FixTypeMissingPathParameter) {
        f.fixMissingPathParametersOAS2(doc, result)
    }
    
    // New pruning fixes
    if f.isFixEnabled(FixTypePruneSchemas) {
        f.pruneOrphanSchemasOAS2(doc, result)
    }
    if f.isFixEnabled(FixTypePrunePaths) {
        f.pruneEmptyPaths(doc.Paths, result, parser.OASVersion20)
    }
    if f.isFixEnabled(FixTypePruneComponents) {
        f.pruneOrphanComponentsOAS2(doc, result)
    }
    
    // ... existing code ...
}

// Add to fixOAS3 function
func (f *Fixer) fixOAS3(parseResult parser.ParseResult, result *FixResult) (*FixResult, error) {
    // ... existing code ...
    
    // Apply enabled fixes
    if f.isFixEnabled(FixTypeMissingPathParameter) {
        f.fixMissingPathParametersOAS3(doc, result)
    }
    
    // New pruning fixes
    if f.isFixEnabled(FixTypePruneSchemas) {
        f.pruneOrphanSchemasOAS3(doc, result)
    }
    if f.isFixEnabled(FixTypePrunePaths) {
        f.pruneEmptyPaths(doc.Paths, result, doc.OASVersion)
    }
    if f.isFixEnabled(FixTypePruneComponents) {
        f.pruneOrphanComponentsOAS3(doc, result)
    }
    
    // ... existing code ...
}
```

### OAS 2.0 Component Pruning

OAS 2.0 has top-level definition maps instead of a components object:

```go
// pruneOrphanComponentsOAS2 removes unreferenced top-level definitions
func (f *Fixer) pruneOrphanComponentsOAS2(doc *parser.OAS2Document, result *FixResult) {
    collector := NewRefCollector()
    collector.CollectOAS2(doc)
    
    // Prune parameters (top-level)
    if doc.Parameters != nil {
        for name := range doc.Parameters {
            if !collector.IsParameterReferenced(name, parser.OASVersion20) {
                delete(doc.Parameters, name)
                result.Fixes = append(result.Fixes, Fix{
                    Type:        FixTypePruneComponents,
                    Path:        fmt.Sprintf("parameters.%s", name),
                    Description: fmt.Sprintf("Removed unreferenced parameter '%s'", name),
                    Before:      doc.Parameters[name],
                    After:       nil,
                })
                f.populateFixLocation(&result.Fixes[len(result.Fixes)-1])
            }
        }
        if len(doc.Parameters) == 0 {
            doc.Parameters = nil
        }
    }
    
    // Prune responses (top-level)
    if doc.Responses != nil {
        for name := range doc.Responses {
            if !collector.IsResponseReferenced(name, parser.OASVersion20) {
                delete(doc.Responses, name)
                result.Fixes = append(result.Fixes, Fix{
                    Type:        FixTypePruneComponents,
                    Path:        fmt.Sprintf("responses.%s", name),
                    Description: fmt.Sprintf("Removed unreferenced response '%s'", name),
                    Before:      doc.Responses[name],
                    After:       nil,
                })
                f.populateFixLocation(&result.Fixes[len(result.Fixes)-1])
            }
        }
        if len(doc.Responses) == 0 {
            doc.Responses = nil
        }
    }
    
    // Prune security definitions
    f.pruneUnreferencedSecurityDefinitions(doc, collector, result)
}
```

### CLI Integration (cmd/oastools/commands/fix.go)

```go
// FixFlags contains flags for the fix command
type FixFlags struct {
    Output       string
    Infer        bool
    Quiet        bool
    SourceMap    bool
    // New pruning flags
    PruneSchemas    bool
    PrunePaths      bool
    PruneComponents bool
    PruneAll        bool
}

// SetupFixFlags creates and configures a FlagSet for the fix command
func SetupFixFlags() (*flag.FlagSet, *FixFlags) {
    fs := flag.NewFlagSet("fix", flag.ContinueOnError)
    flags := &FixFlags{}

    // Existing flags
    fs.StringVar(&flags.Output, "o", "", "output file path (default: stdout)")
    fs.StringVar(&flags.Output, "output", "", "output file path (default: stdout)")
    fs.BoolVar(&flags.Infer, "infer", false, "infer parameter types from naming conventions")
    fs.BoolVar(&flags.Quiet, "q", false, "quiet mode: only output the document, no diagnostic messages")
    fs.BoolVar(&flags.Quiet, "quiet", false, "quiet mode: only output the document, no diagnostic messages")
    fs.BoolVar(&flags.SourceMap, "source-map", false, "include line numbers in fix output")
    fs.BoolVar(&flags.SourceMap, "s", false, "include line numbers in fix output")
    
    // New pruning flags
    fs.BoolVar(&flags.PruneSchemas, "prune-schemas", false, "remove unreferenced schema definitions")
    fs.BoolVar(&flags.PrunePaths, "prune-paths", false, "remove paths with no operations")
    fs.BoolVar(&flags.PruneComponents, "prune-components", false, "remove unreferenced components (parameters, responses, etc.)")
    fs.BoolVar(&flags.PruneAll, "prune-all", false, "apply all pruning fixes (schemas, paths, components)")
    fs.BoolVar(&flags.PruneAll, "prune", false, "apply all pruning fixes (alias for --prune-all)")

    fs.Usage = func() {
        // ... existing usage ...
        cliutil.Writef(fs.Output(), "\nPruning Fixes:\n")
        cliutil.Writef(fs.Output(), "  --prune-schemas     Remove unreferenced schema definitions\n")
        cliutil.Writef(fs.Output(), "  --prune-paths       Remove paths with no operations\n")
        cliutil.Writef(fs.Output(), "  --prune-components  Remove unreferenced components\n")
        cliutil.Writef(fs.Output(), "  --prune-all, --prune  Apply all pruning fixes\n")
        cliutil.Writef(fs.Output(), "\nExamples:\n")
        cliutil.Writef(fs.Output(), "  oastools fix --prune-schemas openapi.yaml\n")
        cliutil.Writef(fs.Output(), "  oastools fix --prune-all openapi.yaml -o cleaned.yaml\n")
        cliutil.Writef(fs.Output(), "  oastools fix --prune --infer openapi.yaml\n")
    }

    return fs, flags
}

// HandleFix executes the fix command
func HandleFix(args []string) error {
    fs, flags := SetupFixFlags()
    // ... existing parsing ...
    
    // Build enabled fixes list
    var enabledFixes []fixer.FixType
    
    // Default fix is always enabled unless specific fixes are requested
    if !flags.PruneSchemas && !flags.PrunePaths && !flags.PruneComponents && !flags.PruneAll {
        // No specific fixes requested, use defaults (all fixes)
    } else {
        // Specific fixes requested
        enabledFixes = append(enabledFixes, fixer.FixTypeMissingPathParameter)
        
        if flags.PruneSchemas || flags.PruneAll {
            enabledFixes = append(enabledFixes, fixer.FixTypePruneSchemas)
        }
        if flags.PrunePaths || flags.PruneAll {
            enabledFixes = append(enabledFixes, fixer.FixTypePrunePaths)
        }
        if flags.PruneComponents || flags.PruneAll {
            enabledFixes = append(enabledFixes, fixer.FixTypePruneComponents)
        }
    }
    
    // Build fixer options
    fixOpts := []fixer.Option{
        fixer.WithFilePath(specPath),
        fixer.WithInferTypes(flags.Infer),
    }
    if len(enabledFixes) > 0 {
        fixOpts = append(fixOpts, fixer.WithEnabledFixes(enabledFixes...))
    }
    
    // ... rest of existing code ...
}
```

### Functional Options

```go
// WithPruneSchemas enables schema pruning
func WithPruneSchemas(prune bool) Option {
    return func(cfg *fixConfig) error {
        if prune {
            cfg.enabledFixes = append(cfg.enabledFixes, FixTypePruneSchemas)
        }
        return nil
    }
}

// WithPrunePaths enables path pruning
func WithPrunePaths(prune bool) Option {
    return func(cfg *fixConfig) error {
        if prune {
            cfg.enabledFixes = append(cfg.enabledFixes, FixTypePrunePaths)
        }
        return nil
    }
}

// WithPruneComponents enables component pruning
func WithPruneComponents(prune bool) Option {
    return func(cfg *fixConfig) error {
        if prune {
            cfg.enabledFixes = append(cfg.enabledFixes, FixTypePruneComponents)
        }
        return nil
    }
}

// WithPruneAll enables all pruning fixes
func WithPruneAll(prune bool) Option {
    return func(cfg *fixConfig) error {
        if prune {
            cfg.enabledFixes = append(cfg.enabledFixes,
                FixTypePruneSchemas,
                FixTypePrunePaths,
                FixTypePruneComponents,
            )
        }
        return nil
    }
}
```

---

## Testing Strategy

### Unit Tests (refs_test.go)

```go
func TestRefCollector_CollectOAS3_AllLocations(t *testing.T) {
    // Test that refs are collected from all documented locations
    doc := buildDocWithRefsInAllLocations()
    
    collector := NewRefCollector()
    collector.CollectOAS3(doc)
    
    // Verify each ref type was collected
    assert.True(t, collector.IsSchemaReferenced("Pet", parser.OASVersion303))
    assert.True(t, collector.IsParameterReferenced("PageSize", parser.OASVersion303))
    assert.True(t, collector.IsResponseReferenced("NotFound", parser.OASVersion303))
    // ... etc
}

func TestRefCollector_CircularReferences(t *testing.T) {
    // Test handling of circular schema references
    doc := buildDocWithCircularRefs()
    
    collector := NewRefCollector()
    collector.CollectOAS3(doc)
    
    // Should not panic or infinite loop
    assert.True(t, collector.IsSchemaReferenced("Node", parser.OASVersion303))
}

func TestRefCollector_TransitiveReferences(t *testing.T) {
    // Test that transitive refs are tracked
    // A -> B -> C should mark all three as referenced
    doc := buildDocWithTransitiveRefs()
    
    collector := NewRefCollector()
    collector.CollectOAS3(doc)
    
    // All schemas in chain should be marked referenced
    assert.True(t, collector.IsSchemaReferenced("A", parser.OASVersion303))
    assert.True(t, collector.IsSchemaReferenced("B", parser.OASVersion303))
    assert.True(t, collector.IsSchemaReferenced("C", parser.OASVersion303))
}
```

### Unit Tests (prune_test.go)

```go
func TestPruneOrphanSchemas_OAS2(t *testing.T) {
    doc := &parser.OAS2Document{
        Swagger: "2.0",
        Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
        Paths: parser.Paths{
            "/pets": {
                Get: &parser.Operation{
                    Responses: &parser.Responses{
                        Codes: map[string]*parser.Response{
                            "200": {Schema: &parser.Schema{Ref: "#/definitions/Pet"}},
                        },
                    },
                },
            },
        },
        Definitions: map[string]*parser.Schema{
            "Pet":      {Type: "object"},
            "Orphan":   {Type: "object"}, // Not referenced
            "AlsoOrphan": {Type: "string"}, // Not referenced
        },
    }
    
    f := New()
    result := &FixResult{Fixes: make([]Fix, 0)}
    f.pruneOrphanSchemasOAS2(doc, result)
    
    // Pet should remain, orphans should be removed
    assert.Len(t, doc.Definitions, 1)
    assert.Contains(t, doc.Definitions, "Pet")
    assert.NotContains(t, doc.Definitions, "Orphan")
    assert.NotContains(t, doc.Definitions, "AlsoOrphan")
    assert.Len(t, result.Fixes, 2)
}

func TestPruneOrphanSchemas_TransitiveRefs(t *testing.T) {
    // Schema A refs B, B refs C
    // Only A is directly referenced from operation
    // All three should be kept
    doc := &parser.OAS3Document{
        OpenAPI: "3.0.3",
        Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
        Paths: parser.Paths{
            "/test": {
                Get: &parser.Operation{
                    Responses: &parser.Responses{
                        Codes: map[string]*parser.Response{
                            "200": {
                                Content: map[string]*parser.MediaType{
                                    "application/json": {
                                        Schema: &parser.Schema{Ref: "#/components/schemas/A"},
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
        Components: &parser.Components{
            Schemas: map[string]*parser.Schema{
                "A": {
                    Type: "object",
                    Properties: map[string]*parser.Schema{
                        "b": {Ref: "#/components/schemas/B"},
                    },
                },
                "B": {
                    Type: "object",
                    Properties: map[string]*parser.Schema{
                        "c": {Ref: "#/components/schemas/C"},
                    },
                },
                "C":      {Type: "string"},
                "Orphan": {Type: "integer"}, // Not in chain
            },
        },
        OASVersion: parser.OASVersion303,
    }
    
    f := New()
    result := &FixResult{Fixes: make([]Fix, 0)}
    f.pruneOrphanSchemasOAS3(doc, result)
    
    // A, B, C should remain; Orphan should be removed
    assert.Len(t, doc.Components.Schemas, 3)
    assert.Contains(t, doc.Components.Schemas, "A")
    assert.Contains(t, doc.Components.Schemas, "B")
    assert.Contains(t, doc.Components.Schemas, "C")
    assert.NotContains(t, doc.Components.Schemas, "Orphan")
    assert.Len(t, result.Fixes, 1)
}

func TestPruneEmptyPaths(t *testing.T) {
    paths := parser.Paths{
        "/active": {
            Get: &parser.Operation{
                Responses: &parser.Responses{},
            },
        },
        "/empty": {},          // No operations
        "/nil": nil,           // Nil path item
        "/ref-only": {Ref: "#/paths/other"}, // Has ref, should keep
    }
    
    f := New()
    result := &FixResult{Fixes: make([]Fix, 0)}
    f.pruneEmptyPaths(paths, result, parser.OASVersion303)
    
    assert.Len(t, paths, 2) // /active and /ref-only
    assert.Contains(t, paths, "/active")
    assert.Contains(t, paths, "/ref-only")
    assert.Len(t, result.Fixes, 2)
}

func TestPruneComponents_AllTypes(t *testing.T) {
    // Test pruning of all component types
    doc := buildDocWithUnreferencedComponents()
    
    f := New()
    result := &FixResult{Fixes: make([]Fix, 0)}
    f.pruneOrphanComponentsOAS3(doc, result)
    
    // Verify each unreferenced component was removed
    assert.NotContains(t, doc.Components.Parameters, "UnusedParam")
    assert.NotContains(t, doc.Components.Responses, "UnusedResponse")
    // ... etc
}

func TestPruneSchemas_EmptyDefinitionsCleanup(t *testing.T) {
    // Test that empty definitions map is set to nil
    doc := &parser.OAS2Document{
        Swagger: "2.0",
        Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
        Paths:   parser.Paths{},
        Definitions: map[string]*parser.Schema{
            "Orphan": {Type: "object"},
        },
    }
    
    f := New()
    result := &FixResult{Fixes: make([]Fix, 0)}
    f.pruneOrphanSchemasOAS2(doc, result)
    
    assert.Nil(t, doc.Definitions)
}
```

### Integration Tests

```go
func TestFixer_PruneSchemas_Integration(t *testing.T) {
    spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/PetList'
components:
  schemas:
    PetList:
      type: array
      items:
        $ref: '#/components/schemas/Pet'
    Pet:
      type: object
      properties:
        name:
          type: string
    OrphanSchema:
      type: object
      description: This schema is never referenced
`
    p := parser.New()
    parseResult, err := p.ParseBytes([]byte(spec))
    require.NoError(t, err)
    
    result, err := fixer.FixWithOptions(
        fixer.WithParsed(*parseResult),
        fixer.WithEnabledFixes(fixer.FixTypePruneSchemas),
    )
    require.NoError(t, err)
    
    doc := result.Document.(*parser.OAS3Document)
    assert.Len(t, doc.Components.Schemas, 2) // PetList and Pet
    assert.Contains(t, doc.Components.Schemas, "PetList")
    assert.Contains(t, doc.Components.Schemas, "Pet")
    assert.NotContains(t, doc.Components.Schemas, "OrphanSchema")
}

func TestFixer_PruneAll_Integration(t *testing.T) {
    // Test with --prune-all equivalent
    spec := buildSpecWithOrphansAndEmptyPaths()
    
    p := parser.New()
    parseResult, err := p.ParseBytes([]byte(spec))
    require.NoError(t, err)
    
    result, err := fixer.FixWithOptions(
        fixer.WithParsed(*parseResult),
        fixer.WithPruneAll(true),
    )
    require.NoError(t, err)
    
    // Verify all pruning was applied
    assert.Greater(t, result.FixCount, 0)
}
```

### Benchmark Tests

```go
func BenchmarkPruneSchemas_Small(b *testing.B) {
    doc := buildDocWithSchemas(10, 5) // 10 schemas, 5 orphaned
    
    b.ResetTimer()
    for b.Loop() {
        f := New()
        result := &FixResult{Fixes: make([]Fix, 0)}
        docCopy, _ := deepCopyOAS3Document(doc)
        f.pruneOrphanSchemasOAS3(docCopy, result)
    }
}

func BenchmarkPruneSchemas_Large(b *testing.B) {
    doc := buildDocWithSchemas(500, 200) // 500 schemas, 200 orphaned
    
    b.ResetTimer()
    for b.Loop() {
        f := New()
        result := &FixResult{Fixes: make([]Fix, 0)}
        docCopy, _ := deepCopyOAS3Document(doc)
        f.pruneOrphanSchemasOAS3(docCopy, result)
    }
}

func BenchmarkRefCollector_Large(b *testing.B) {
    doc := buildLargeDocWithManyRefs()
    
    b.ResetTimer()
    for b.Loop() {
        collector := NewRefCollector()
        collector.CollectOAS3(doc)
    }
}
```

### Real-World Test Cases

```go
func TestPruneSchemas_RealWorldSpecs(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping real-world spec tests in short mode")
    }
    
    specs := []string{
        "testdata/discord.yaml",
        "testdata/stripe.yaml",
        "testdata/github.yaml",
    }
    
    for _, specPath := range specs {
        t.Run(filepath.Base(specPath), func(t *testing.T) {
            p := parser.New()
            parseResult, err := p.Parse(specPath)
            require.NoError(t, err)
            
            // These specs should have no orphans
            result, err := fixer.FixWithOptions(
                fixer.WithParsed(*parseResult),
                fixer.WithEnabledFixes(fixer.FixTypePruneSchemas),
            )
            require.NoError(t, err)
            
            // Well-maintained specs should have few/no orphans
            t.Logf("%s: %d orphaned schemas found", specPath, result.FixCount)
        })
    }
}
```

---

## Edge Cases and Special Handling

### Circular References

Schemas can form circular reference chains. The implementation must track visited schemas to prevent infinite loops:

```go
func (c *RefCollector) collectSchemaRefsRecursive(schema *parser.Schema, path string) {
    if schema == nil {
        return
    }
    
    // Prevent infinite loops on circular refs
    if c.visited[schema] {
        return
    }
    c.visited[schema] = true
    defer func() { delete(c.visited, schema) }()
    
    // ... collect refs ...
}
```

### Self-Referencing Schemas

A schema may reference itself (common in tree structures):

```yaml
components:
  schemas:
    TreeNode:
      type: object
      properties:
        children:
          type: array
          items:
            $ref: '#/components/schemas/TreeNode'
```

This should be handled correctly - if `TreeNode` is referenced from an operation, it should not be pruned despite the self-reference.

### Discriminator Mappings

Discriminator mappings can use either full paths or bare names:

```yaml
discriminator:
  propertyName: petType
  mapping:
    dog: '#/components/schemas/Dog'  # Full path
    cat: Cat                          # Bare name
```

Both formats must be recognized as references.

### External References

External references (file or URL) should not be followed for pruning purposes. Only local references (`#/...`) are considered:

```go
func isLocalRef(ref string) bool {
    return strings.HasPrefix(ref, "#/")
}
```

### OAS Version Differences

The implementation must handle version-specific features:

- OAS 2.0: `definitions`, `parameters`, `responses`, `securityDefinitions`
- OAS 3.0+: `components` with all sub-objects
- OAS 3.1+: `webhooks`, `components.pathItems`, JSON Schema extensions
- OAS 3.2+: `query` operation method

---

## Documentation Updates

### doc.go Updates

Add documentation for new fix types:

```go
// # Supported Fixes
//
// The fixer currently supports the following automatic fixes:
//
//   - Missing path parameters: Adds Parameter objects for path template variables
//     that are not declared in the operation's parameters list.
//
//   - Prune schemas: Removes schema definitions that are not referenced by any
//     operation, webhook, or other schema. Handles transitive references correctly.
//
//   - Prune paths: Removes path items that have no HTTP operations defined.
//     Paths with only $ref are preserved.
//
//   - Prune components: Removes unreferenced parameters, responses, request bodies,
//     headers, security schemes, links, callbacks, examples, and path items from
//     the components object (OAS 3.x) or top-level definitions (OAS 2.0).
```

### CLI Reference Updates

Update `docs/cli-reference.md`:

```markdown
### Pruning Flags

| Flag | Description |
|------|-------------|
| `--prune-schemas` | Remove unreferenced schema definitions |
| `--prune-paths` | Remove path items with no operations |
| `--prune-components` | Remove unreferenced components (parameters, responses, etc.) |
| `--prune-all`, `--prune` | Apply all pruning fixes |

### Pruning Examples

```bash
# Remove orphaned schemas
oastools fix --prune-schemas joined-api.yaml -o cleaned.yaml

# Remove all orphans and empty paths
oastools fix --prune-all messy-api.yaml -o clean.yaml

# Combine with type inference
oastools fix --prune --infer api.yaml -o fixed.yaml

# Preview what would be pruned (use validate to see document stats)
oastools fix --prune-all api.yaml | oastools validate -q -
```
```

### Developer Guide Updates

Update `docs/developer-guide.md`:

```markdown
### Pruning Orphaned Definitions

```go
// Remove unreferenced schemas
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("joined-api.yaml"),
    fixer.WithEnabledFixes(fixer.FixTypePruneSchemas),
)

// Remove all orphaned content
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("api.yaml"),
    fixer.WithPruneAll(true),
)

// Check what was pruned
for _, fix := range result.Fixes {
    if fix.Type == fixer.FixTypePruneSchemas {
        fmt.Printf("Removed: %s\n", fix.Path)
    }
}
```
```

---

## Implementation Checklist

### Phase 1: Reference Collection Engine

- [ ] Create `fixer/refs.go` with `RefCollector` struct
- [ ] Implement `CollectOAS2()` covering all OAS 2.0 reference locations
- [ ] Implement `CollectOAS3()` covering all OAS 3.x reference locations
- [ ] Add `Is*Referenced()` methods for each component type
- [ ] Create `fixer/refs_test.go` with comprehensive tests
- [ ] Test circular reference handling
- [ ] Test all reference location coverage

### Phase 2: Schema Pruning

- [ ] Add `FixTypePruneSchemas` constant
- [ ] Implement `pruneOrphanSchemasOAS2()` in `prune.go`
- [ ] Implement `pruneOrphanSchemasOAS3()` in `prune.go`
- [ ] Implement `buildReferencedSchemaSet()` for transitive closure
- [ ] Implement `collectSchemaRefs()` helper
- [ ] Add schema pruning tests
- [ ] Integrate into `fixOAS2()` and `fixOAS3()`

### Phase 3: Path Pruning

- [ ] Add `FixTypePrunePaths` constant
- [ ] Implement `pruneEmptyPaths()` in `prune.go`
- [ ] Implement `isPathItemEmpty()` helper
- [ ] Add path pruning tests
- [ ] Handle OAS version-specific methods (Trace, Query)

### Phase 4: Component Pruning

- [ ] Add `FixTypePruneComponents` constant
- [ ] Implement `pruneOrphanComponentsOAS2()` for top-level definitions
- [ ] Implement `pruneOrphanComponentsOAS3()` for components object
- [ ] Implement individual pruning methods:
  - [ ] `pruneUnreferencedParameters()`
  - [ ] `pruneUnreferencedResponses()`
  - [ ] `pruneUnreferencedRequestBodies()`
  - [ ] `pruneUnreferencedHeaders()`
  - [ ] `pruneUnreferencedSecuritySchemes()`
  - [ ] `pruneUnreferencedSecurityDefinitions()` (OAS 2.0)
  - [ ] `pruneUnreferencedLinks()`
  - [ ] `pruneUnreferencedCallbacks()`
  - [ ] `pruneUnreferencedExamples()`
  - [ ] `pruneUnreferencedPathItems()` (OAS 3.1+)
- [ ] Implement `cleanupEmptyComponents()`
- [ ] Add comprehensive component pruning tests

### Phase 5: Functional Options

- [ ] Add `WithPruneSchemas()` option
- [ ] Add `WithPrunePaths()` option
- [ ] Add `WithPruneComponents()` option
- [ ] Add `WithPruneAll()` option
- [ ] Add option tests

### Phase 6: CLI Integration

- [ ] Add new flags to `FixFlags` struct
- [ ] Update `SetupFixFlags()` with new flags
- [ ] Update `HandleFix()` to process pruning flags
- [ ] Update usage/help text
- [ ] Add CLI integration tests

### Phase 7: Documentation and Examples

- [ ] Update `fixer/doc.go` with new fix types
- [ ] Add `Example_pruneSchemas()` to `example_test.go`
- [ ] Add `Example_pruneAll()` to `example_test.go`
- [ ] Update `docs/cli-reference.md`
- [ ] Update `docs/developer-guide.md`

### Phase 8: Benchmarks and Real-World Testing

- [ ] Add benchmark tests for schema pruning
- [ ] Add benchmark tests for reference collection
- [ ] Test against real-world specs (Discord, Stripe, GitHub)
- [ ] Verify no regressions in existing functionality

### Phase 9: Final Validation

- [ ] Run `make check` (fmt, lint, test, tidy)
- [ ] Run `govulncheck ./...`
- [ ] Verify all tests pass
- [ ] Review test coverage

---

## Success Criteria

1. All pruning operations correctly identify unreferenced definitions
2. Transitive references are tracked correctly (A→B→C keeps all three)
3. Circular references do not cause infinite loops
4. Empty containers are cleaned up after pruning
5. Source location tracking works for prune fixes when SourceMap is enabled
6. CLI flags work independently and in combination
7. All existing tests continue to pass
8. New tests provide comprehensive coverage
9. Documentation is complete and accurate
10. Performance is acceptable for large documents (thousands of schemas)

---

## Risk Mitigation

### Risk: Incorrect Reference Detection

Mitigation: Comprehensive testing of all reference locations documented in OAS specification. Real-world spec testing against Discord, Stripe, GitHub APIs.

### Risk: Breaking Circular References

Mitigation: Track visited schemas during traversal. Add specific tests for circular and self-referencing schemas.

### Risk: Version-Specific Feature Mishandling

Mitigation: Separate test cases for each OAS version. Verify OAS 3.1+ specific features (webhooks, pathItems in components) are handled.

### Risk: Performance Degradation on Large Documents

Mitigation: Benchmark tests with large documents. Use efficient data structures (maps for O(1) lookup). Single-pass reference collection where possible.
