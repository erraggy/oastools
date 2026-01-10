# Error Handling Standards

**IMPORTANT: Follow consistent error handling patterns across all packages.**

## Error Message Format

```go
fmt.Errorf("<package>: <action>: %w", err)
```

## Rules

1. **Always prefix with package name** - Every error returned from a public function should start with the package name
2. **Use lowercase** - Error messages should start with lowercase (except acronyms like HTTP, OAS, JSON)
3. **No trailing punctuation** - Do not end error messages with periods
4. **Use `%w` for wrapping** - Always use `%w` (not `%v` or `%s`) when wrapping errors for `errors.Is()` and `errors.Unwrap()` support
5. **Be descriptive** - Include relevant context (file paths, version numbers, counts)

## Examples

### Good - Consistent Prefixing and Wrapping
```go
return fmt.Errorf("parser: failed to parse specification: %w", err)
return fmt.Errorf("converter: invalid target version: %s", targetVersionStr)
return fmt.Errorf("validator: unsupported OAS version: %s", version)
return fmt.Errorf("joiner: %s has %d parse error(s)", path, len(errors))
return fmt.Errorf("generator: failed to generate types: %w", err)
```

### Bad - Inconsistent Patterns
```go
return fmt.Errorf("failed to parse specification: %w", err)  // Missing package prefix
return fmt.Errorf("Invalid target version: %s", version)     // Capitalized
return fmt.Errorf("parse error: %v", err)                    // Using %v instead of %w
```

## Sentinel Errors

Use the `oaserrors` package for programmatic error handling:

```go
import "github.com/erraggy/oastools/oaserrors"

// Check error types
if errors.Is(err, oaserrors.ErrParse) { ... }
if errors.Is(err, oaserrors.ErrCircularReference) { ... }
```
