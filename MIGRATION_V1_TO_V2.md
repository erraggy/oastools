# Migration Guide: oastools v1.x → v2.0

This guide helps you migrate from oastools v1.x to v2.0, which introduces a new functional options API pattern and removes deprecated convenience functions.

## Overview of Changes

v2.0 removes 11 deprecated package-level convenience functions and standardizes on the `*WithOptions` functional options API across all packages. This provides:

- **Self-documenting code**: Named options make API calls clearer
- **Flexible input sources**: File paths, io.Reader, byte slices, and parsed documents
- **Extensible configuration**: Add new options without breaking changes
- **Better IDE support**: Improved autocomplete and type safety

## Breaking Changes

### Removed Functions

All deprecated package-level convenience functions have been removed:

| Package | Removed Functions |
|---------|-------------------|
| **parser** | `Parse()`, `ParseReader()`, `ParseBytes()` |
| **validator** | `Validate()`, `ValidateParsed()` |
| **converter** | `Convert()`, `ConvertParsed()` |
| **joiner** | `Join()`, `JoinParsed()` |
| **differ** | `Diff()`, `DiffParsed()` |

**Note**: Struct methods (e.g., `parser.New().Parse()`) are **NOT deprecated** and remain unchanged.

## Migration Examples

### Parser Package

#### Basic File Parsing

**v1.x (Removed):**
```go
result, err := parser.Parse("openapi.yaml", false, true)
```

**v2.0 (New):**
```go
result, err := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithValidateStructure(true),
)
```

#### Parsing from io.Reader

**v1.x (Removed):**
```go
file, _ := os.Open("openapi.yaml")
result, err := parser.ParseReader(file, false, true)
```

**v2.0 (New):**
```go
file, _ := os.Open("openapi.yaml")
result, err := parser.ParseWithOptions(
    parser.WithReader(file),
    parser.WithValidateStructure(true),
)
```

#### Parsing from Byte Slice

**v1.x (Removed):**
```go
data := []byte(`openapi: 3.0.0...`)
result, err := parser.ParseBytes(data, false, true)
```

**v2.0 (New):**
```go
data := []byte(`openapi: 3.0.0...`)
result, err := parser.ParseWithOptions(
    parser.WithBytes(data),
    parser.WithValidateStructure(true),
)
```

#### Parser Options Reference

| v1.x Parameter | v2.0 Option | Default |
|----------------|-------------|---------|
| `resolveRefs bool` | `parser.WithResolveRefs(bool)` | `false` |
| `validateStructure bool` | `parser.WithValidateStructure(bool)` | `false` |
| N/A | `parser.WithUserAgent(string)` | `"oastools"` |

### Validator Package

#### Basic Validation

**v1.x (Removed):**
```go
result, err := validator.Validate("openapi.yaml", true, false)
```

**v2.0 (New):**
```go
result, err := validator.ValidateWithOptions(
    validator.WithFilePath("openapi.yaml"),
    validator.WithIncludeWarnings(true),
)
```

#### Validating Parsed Document

**v1.x (Removed):**
```go
parsed, _ := parser.Parse("openapi.yaml", false, true)
result, err := validator.ValidateParsed(*parsed, true, false)
```

**v2.0 (New):**
```go
parsed, _ := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithValidateStructure(true),
)
result, err := validator.ValidateWithOptions(
    validator.WithParsed(*parsed),
    validator.WithIncludeWarnings(true),
)
```

#### Validator Options Reference

| v1.x Parameter | v2.0 Option | Default |
|----------------|-------------|---------|
| `includeWarnings bool` | `validator.WithIncludeWarnings(bool)` | `false` |
| `strictMode bool` | `validator.WithStrictMode(bool)` | `false` |
| N/A | `validator.WithUserAgent(string)` | `"oastools"` |

### Converter Package

#### Basic Conversion

**v1.x (Removed):**
```go
result, err := converter.Convert("swagger.yaml", "3.0.3")
```

**v2.0 (New):**
```go
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("swagger.yaml"),
    converter.WithTargetVersion("3.0.3"),
)
```

#### Converting Parsed Document

**v1.x (Removed):**
```go
parsed, _ := parser.Parse("swagger.yaml", false, true)
result, err := converter.ConvertParsed(*parsed, "3.0.3")
```

**v2.0 (New):**
```go
parsed, _ := parser.ParseWithOptions(
    parser.WithFilePath("swagger.yaml"),
    parser.WithValidateStructure(true),
)
result, err := converter.ConvertWithOptions(
    converter.WithParsed(*parsed),
    converter.WithTargetVersion("3.0.3"),
)
```

#### Converter Options Reference

| v1.x Parameter | v2.0 Option | Default |
|----------------|-------------|---------|
| `targetVersion string` | `converter.WithTargetVersion(string)` | **Required** |
| N/A | `converter.WithStrictMode(bool)` | `false` |
| N/A | `converter.WithIncludeInfo(bool)` | `true` |
| N/A | `converter.WithUserAgent(string)` | `"oastools"` |

### Joiner Package

#### Basic Joining

**v1.x (Removed):**
```go
config := joiner.DefaultConfig()
result, err := joiner.Join([]string{"base.yaml", "ext.yaml"}, config)
```

**v2.0 (New):**
```go
result, err := joiner.JoinWithOptions(
    joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
    joiner.WithConfig(joiner.DefaultConfig()),
)
```

#### Joining with Custom Strategy

**v1.x (Removed):**
```go
config := joiner.DefaultConfig()
config.PathStrategy = joiner.StrategyAcceptLeft
result, err := joiner.Join([]string{"base.yaml", "ext.yaml"}, config)
```

**v2.0 (New):**
```go
result, err := joiner.JoinWithOptions(
    joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
    joiner.WithPathStrategy(joiner.StrategyAcceptLeft),
)
```

#### Joining Parsed Documents

**v1.x (Removed):**
```go
doc1, _ := parser.Parse("base.yaml", false, true)
doc2, _ := parser.Parse("ext.yaml", false, true)
config := joiner.DefaultConfig()
result, err := joiner.JoinParsed([]parser.ParseResult{*doc1, *doc2}, config)
```

**v2.0 (New):**
```go
doc1, _ := parser.ParseWithOptions(
    parser.WithFilePath("base.yaml"),
    parser.WithValidateStructure(true),
)
doc2, _ := parser.ParseWithOptions(
    parser.WithFilePath("ext.yaml"),
    parser.WithValidateStructure(true),
)
result, err := joiner.JoinWithOptions(
    joiner.WithParsedDocs([]parser.ParseResult{*doc1, *doc2}),
    joiner.WithConfig(joiner.DefaultConfig()),
)
```

#### Joiner Options Reference

| v1.x Parameter | v2.0 Option | Default |
|----------------|-------------|---------|
| `config JoinerConfig` | `joiner.WithConfig(JoinerConfig)` | `DefaultConfig()` |
| N/A | `joiner.WithDefaultStrategy(CollisionStrategy)` | `StrategyFailOnCollision` |
| N/A | `joiner.WithPathStrategy(CollisionStrategy)` | Uses default |
| N/A | `joiner.WithSchemaStrategy(CollisionStrategy)` | Uses default |
| N/A | `joiner.WithComponentStrategy(CollisionStrategy)` | Uses default |
| N/A | `joiner.WithDeduplicateTags(bool)` | `true` |
| N/A | `joiner.WithMergeArrays(bool)` | `true` |

### Differ Package

#### Basic Diff

**v1.x (Removed):**
```go
result, err := differ.Diff("api-v1.yaml", "api-v2.yaml")
```

**v2.0 (New):**
```go
result, err := differ.DiffWithOptions(
    differ.WithSourceFilePath("api-v1.yaml"),
    differ.WithTargetFilePath("api-v2.yaml"),
)
```

#### Diff with Breaking Change Detection

**v1.x (Removed):**
```go
d := differ.New()
d.Mode = differ.ModeBreaking
result, err := d.Diff("api-v1.yaml", "api-v2.yaml")
```

**v2.0 (New):**
```go
result, err := differ.DiffWithOptions(
    differ.WithSourceFilePath("api-v1.yaml"),
    differ.WithTargetFilePath("api-v2.yaml"),
    differ.WithMode(differ.ModeBreaking),
)
```

#### Diffing Parsed Documents

**v1.x (Removed):**
```go
source, _ := parser.Parse("api-v1.yaml", false, true)
target, _ := parser.Parse("api-v2.yaml", false, true)
result, err := differ.DiffParsed(*source, *target)
```

**v2.0 (New):**
```go
source, _ := parser.ParseWithOptions(
    parser.WithFilePath("api-v1.yaml"),
    parser.WithValidateStructure(true),
)
target, _ := parser.ParseWithOptions(
    parser.WithFilePath("api-v2.yaml"),
    parser.WithValidateStructure(true),
)
result, err := differ.DiffWithOptions(
    differ.WithSourceParsed(*source),
    differ.WithTargetParsed(*target),
)
```

#### Differ Options Reference

| v1.x Parameter | v2.0 Option | Default |
|----------------|-------------|---------|
| N/A | `differ.WithMode(DiffMode)` | `ModeSimple` |
| N/A | `differ.WithIncludeInfo(bool)` | `true` |
| N/A | `differ.WithUserAgent(string)` | `"oastools"` |

## Struct Method API (Unchanged)

The struct-based API remains **unchanged** and is the recommended approach for reusable instances:

### Parser
```go
p := parser.New()
p.ResolveRefs = false
p.ValidateStructure = true
result1, _ := p.Parse("api1.yaml")
result2, _ := p.Parse("api2.yaml")
```

### Validator
```go
v := validator.New()
v.IncludeWarnings = true
v.StrictMode = false
result1, _ := v.Validate("api1.yaml")
result2, _ := v.Validate("api2.yaml")
```

### Converter
```go
c := converter.New()
c.StrictMode = false
c.IncludeInfo = true
result1, _ := c.Convert("swagger1.yaml", "3.0.3")
result2, _ := c.Convert("swagger2.yaml", "3.0.3")
```

### Joiner
```go
j := joiner.New(joiner.DefaultConfig())
result1, _ := j.Join([]string{"base1.yaml", "ext1.yaml"})
result2, _ := j.Join([]string{"base2.yaml", "ext2.yaml"})
```

### Differ
```go
d := differ.New()
d.Mode = differ.ModeBreaking
d.IncludeInfo = false
result1, _ := d.Diff("api-v1.yaml", "api-v2.yaml")
result2, _ := d.Diff("api-v2.yaml", "api-v3.yaml")
```

## Benefits of the New API

### 1. Self-Documenting Code

**v1.x:**
```go
result, err := parser.Parse("api.yaml", false, true)  // What do these bools mean?
```

**v2.0:**
```go
result, err := parser.ParseWithOptions(
    parser.WithFilePath("api.yaml"),
    parser.WithResolveRefs(false),           // Clear intent
    parser.WithValidateStructure(true),      // Self-documenting
)
```

### 2. Flexible Input Sources

**v2.0** allows mixing input sources in a single call:
```go
// Parse from URL with custom user agent
result, err := parser.ParseWithOptions(
    parser.WithFilePath("https://api.example.com/openapi.yaml"),
    parser.WithUserAgent("myapp/1.0"),
)

// Parse from bytes
data := []byte(`openapi: 3.0.0...`)
result, err := parser.ParseWithOptions(
    parser.WithBytes(data),
)

// Parse from reader
file, _ := os.Open("spec.yaml")
result, err := parser.ParseWithOptions(
    parser.WithReader(file),
)
```

### 3. Backward-Compatible Extensions

New options can be added without breaking existing code:
```go
// v2.0
result, err := parser.ParseWithOptions(
    parser.WithFilePath("api.yaml"),
)

// v2.1+ (hypothetical) - adds new option without breaking v2.0 code
result, err := parser.ParseWithOptions(
    parser.WithFilePath("api.yaml"),
    parser.WithCacheTTL(5 * time.Minute),  // New option, old code still works
)
```

### 4. Better IDE Autocomplete

With functional options, IDEs can suggest available options as you type, making the API more discoverable.

## Migration Strategy

### For Small Codebases

1. Update all deprecated function calls to use `*WithOptions` variants
2. Run tests to ensure behavior is unchanged
3. Update to v2.0

### For Large Codebases

1. **Phase 1**: Identify all deprecated usage with:
   ```bash
   # Search for deprecated parser calls
   grep -r "parser\.Parse(" . --include="*.go"
   grep -r "parser\.ParseReader(" . --include="*.go"
   grep -r "parser\.ParseBytes(" . --include="*.go"

   # Repeat for validator, converter, joiner, differ
   ```

2. **Phase 2**: Migrate incrementally by package:
   - Start with parser (most common)
   - Then validator
   - Then converter, joiner, differ

3. **Phase 3**: Test thoroughly after each package migration

4. **Phase 4**: Update to v2.0

### Automated Migration

For simple cases, you can use sed to automate parts of the migration:

```bash
# Example: parser.Parse() → parser.ParseWithOptions()
# Note: This is a simple example and may not cover all cases
sed -i '' 's/parser\.Parse(\([^,]*\), false, true)/parser.ParseWithOptions(parser.WithFilePath(\1), parser.WithValidateStructure(true))/g' *.go
```

**Warning**: Automated migration may not handle all cases correctly. Always review and test changes.

## Staying on v1.x

If you're not ready to migrate immediately, v1.x will remain available:

```go
// Continue using v1.x
go get github.com/erraggy/oastools@v1.6.0
```

However, v1.x will no longer receive new features. Security patches may be provided for critical issues.

## Getting Help

- **GitHub Issues**: [github.com/erraggy/oastools/issues](https://github.com/erraggy/oastools/issues)
- **Documentation**: Package docs at [pkg.go.dev/github.com/erraggy/oastools](https://pkg.go.dev/github.com/erraggy/oastools)
- **Examples**: See `*_test.go` files for comprehensive usage examples

## Version Compatibility

| oastools Version | Go Version | API Style |
|------------------|------------|-----------|
| v1.x | ≥ 1.24 | Boolean parameters + struct methods |
| v2.0 | ≥ 1.24 | Functional options + struct methods |

## Summary

The v2.0 migration removes deprecated convenience functions in favor of a more flexible, maintainable functional options API. The struct method API remains unchanged, providing a smooth migration path for codebases already using that pattern.

**Key Takeaways:**
- Replace `Package.Function(args...)` with `Package.FunctionWithOptions(Package.WithOption(...), ...)`
- Struct methods (`New().Method()`) remain unchanged
- New API provides better clarity, flexibility, and extensibility
- v1.x remains available for gradual migration
