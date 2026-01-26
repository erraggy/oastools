# Collision Handler Design

## Overview

Extend the `joiner` package to accept collision handler functions, enabling callers to receive context about identified collisions and either handle them directly, log/observe them, or delegate to configured strategies.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Context detail level | Full context | Maximum flexibility for callers |
| Handler capabilities | All 3 levels | Resolution + replacement, decision only, observe with veto |
| Multiple handlers | Single handler | Caller composes their own logic internally |
| Collision types | Configurable subset | Power users get full control, common case stays simple |
| Sync vs async | Synchronous only | Keeps API simple; callers can wrap async internally |
| Error handling | Fall back to strategy | Resilience - handlers can't break the join |

## Core Types

```go
// CollisionType identifies what kind of component collided.
type CollisionType string

const (
    CollisionTypeSchema         CollisionType = "schema"
    CollisionTypePath           CollisionType = "path"
    CollisionTypeWebhook        CollisionType = "webhook"
    CollisionTypeResponse       CollisionType = "response"
    CollisionTypeParameter      CollisionType = "parameter"
    CollisionTypeExample        CollisionType = "example"
    CollisionTypeRequestBody    CollisionType = "requestBody"
    CollisionTypeHeader         CollisionType = "header"
    CollisionTypeSecurityScheme CollisionType = "securityScheme"
    CollisionTypeLink           CollisionType = "link"
    CollisionTypeCallback       CollisionType = "callback"
)

// CollisionContext provides full details about a detected collision.
type CollisionContext struct {
    Type           CollisionType          // What kind of component collided
    Name           string                 // The colliding name (e.g., "User", "/pets")
    JSONPath       string                 // Full path (e.g., "$.components.schemas.User")

    LeftSource     string                 // Source file/identifier for left document
    LeftLocation   *parser.Location       // Line/column in left document (nil if unknown)
    LeftValue      any                    // The left component (*parser.Schema, *parser.PathItem, etc.)

    RightSource    string                 // Source file/identifier for right document
    RightLocation  *parser.Location       // Line/column in right document (nil if unknown)
    RightValue     any                    // The right component

    RenameInfo     *RenameContext         // Operation context if available (nil otherwise)

    ConfiguredStrategy CollisionStrategy  // Strategy that would apply without handler
}

// ResolutionAction specifies what the joiner should do.
type ResolutionAction int

const (
    ResolutionContinue      ResolutionAction = iota // Use configured strategy (observe-only)
    ResolutionAcceptLeft                            // Keep left value
    ResolutionAcceptRight                           // Keep right value (overwrite)
    ResolutionRename                                // Rename right, keep both
    ResolutionDeduplicate                           // Treat as equivalent, deduplicate
    ResolutionFail                                  // Fail with error
    ResolutionCustom                                // Use CustomValue provided
)

// CollisionResolution is returned by the handler to indicate desired action.
type CollisionResolution struct {
    Action      ResolutionAction
    CustomValue any    // Used when Action is ResolutionCustom
    Message     string // Optional message for logging/warnings
}

// CollisionHandler is called when a collision is detected.
// Return an error to log a warning and fall back to configured strategy.
type CollisionHandler func(collision CollisionContext) (CollisionResolution, error)
```

## Registration API

```go
// WithCollisionHandler registers a handler called when collisions are detected.
// The handler receives full context and can resolve, observe, or delegate.
// If the handler returns an error, it's logged as a warning and the configured
// strategy is used instead.
//
// By default, the handler is called for all collision types. Use
// WithCollisionHandlerFor to handle specific types only.
func WithCollisionHandler(handler CollisionHandler) Option {
    return func(cfg *joinConfig) error {
        if handler == nil {
            return errors.New("collision handler cannot be nil")
        }
        cfg.collisionHandler = handler
        cfg.collisionHandlerTypes = nil // nil means all types
        return nil
    }
}

// WithCollisionHandlerFor registers a handler for specific collision types only.
// Collisions of other types use the configured strategy without invoking the handler.
func WithCollisionHandlerFor(handler CollisionHandler, types ...CollisionType) Option {
    return func(cfg *joinConfig) error {
        if handler == nil {
            return errors.New("collision handler cannot be nil")
        }
        if len(types) == 0 {
            return errors.New("at least one collision type must be specified")
        }
        cfg.collisionHandler = handler
        cfg.collisionHandlerTypes = make(map[CollisionType]bool, len(types))
        for _, t := range types {
            cfg.collisionHandlerTypes[t] = true
        }
        return nil
    }
}
```

New fields in `joinConfig`:

```go
type joinConfig struct {
    // ... existing fields ...

    collisionHandler      CollisionHandler
    collisionHandlerTypes map[CollisionType]bool // empty means all types
}
```

## Integration with Existing Collision Handling

Handler intercepts collisions *before* strategy dispatch:

```go
// handleCollision processes a collision, optionally invoking a registered handler.
// Returns the value to use (left or right), a new name if renamed, and any error.
func (j *Joiner) handleCollision(
    collisionType CollisionType,
    name string,
    jsonPath string,
    leftSource, rightSource string,
    leftValue, rightValue any,
    renameInfo *RenameContext,
) (useValue any, newName string, err error) {

    // Determine which strategy applies for this collision type
    strategy := j.strategyFor(collisionType, name)

    // Check if handler should be invoked
    if j.config.collisionHandler != nil && j.shouldInvokeHandler(collisionType) {
        collision := CollisionContext{
            Type:               collisionType,
            Name:               name,
            JSONPath:           jsonPath,
            LeftSource:         leftSource,
            LeftLocation:       j.lookupLocation(leftSource, jsonPath),
            LeftValue:          leftValue,
            RightSource:        rightSource,
            RightLocation:      j.lookupLocation(rightSource, jsonPath),
            RightValue:         rightValue,
            RenameInfo:         renameInfo,
            ConfiguredStrategy: strategy,
        }

        resolution, handlerErr := j.config.collisionHandler(collision)
        if handlerErr != nil {
            // Log warning and fall back to configured strategy
            j.addWarning(WarnHandlerError, jsonPath,
                fmt.Sprintf("collision handler error: %v; using %s strategy", handlerErr, strategy),
                rightSource)
        } else {
            // Apply the resolution
            return j.applyResolution(collision, resolution)
        }
    }

    // No handler or handler deferred - use configured strategy
    return j.applyStrategy(collisionType, name, jsonPath, leftSource, rightSource,
        leftValue, rightValue, renameInfo, strategy)
}

// shouldInvokeHandler checks if the handler wants this collision type.
func (j *Joiner) shouldInvokeHandler(collisionType CollisionType) bool {
    if len(j.config.collisionHandlerTypes) == 0 {
        return true // empty means all types
    }
    return j.config.collisionHandlerTypes[collisionType]
}

// applyResolution converts a CollisionResolution into the appropriate action.
func (j *Joiner) applyResolution(
    collision CollisionContext,
    resolution CollisionResolution,
) (useValue any, newName string, err error) {

    // Record message as warning if provided
    if resolution.Message != "" {
        j.addWarning(WarnHandlerResolution, collision.JSONPath,
            resolution.Message, collision.RightSource)
    }

    switch resolution.Action {
    case ResolutionContinue:
        // Delegate to configured strategy
        return j.applyStrategy(collision.Type, collision.Name, collision.JSONPath,
            collision.LeftSource, collision.RightSource,
            collision.LeftValue, collision.RightValue,
            collision.RenameInfo, collision.ConfiguredStrategy)

    case ResolutionAcceptLeft:
        return collision.LeftValue, "", nil

    case ResolutionAcceptRight:
        return collision.RightValue, "", nil

    case ResolutionRename:
        newName, err := j.generateRenameName(collision)
        if err != nil {
            return nil, "", err
        }
        return collision.RightValue, newName, nil

    case ResolutionDeduplicate:
        // Verify schemas are equivalent before deduplicating
        if collision.Type == CollisionTypeSchema {
            return collision.LeftValue, "", nil // Keep left, discard right
        }
        return collision.LeftValue, "", nil

    case ResolutionFail:
        return nil, "", j.buildCollisionError(collision, resolution.Message)

    case ResolutionCustom:
        if resolution.CustomValue == nil {
            return nil, "", errors.New("ResolutionCustom requires CustomValue")
        }
        return resolution.CustomValue, "", nil

    default:
        return nil, "", fmt.Errorf("unknown resolution action: %d", resolution.Action)
    }
}
```

## Warning Types

```go
// New warning categories
const (
    WarnHandlerError      WarningCategory = "handler_error"      // Handler returned error, fell back to strategy
    WarnHandlerResolution WarningCategory = "handler_resolution" // Handler resolved collision with message
)

// NewHandlerErrorWarning creates a warning for handler errors.
func NewHandlerErrorWarning(jsonPath, message, source string, line, col int) JoinWarning {
    return JoinWarning{
        Category:   WarnHandlerError,
        Path:       jsonPath,
        Message:    message,
        SourceFile: source,
        Line:       line,
        Column:     col,
        Severity:   severity.SeverityWarning,
    }
}

// NewHandlerResolutionWarning creates a warning for handler-provided messages.
func NewHandlerResolutionWarning(jsonPath, message, source string, line, col int) JoinWarning {
    return JoinWarning{
        Category:   WarnHandlerResolution,
        Path:       jsonPath,
        Message:    message,
        SourceFile: source,
        Line:       line,
        Column:     col,
        Severity:   severity.SeverityInfo,
    }
}
```

## Helper Functions

Convenience functions for common handler patterns:

```go
// handlers.go - convenience functions for common handler patterns

// ContinueWithStrategy returns a resolution that defers to the configured strategy.
// Use this for observe-only handlers that just want to log collisions.
func ContinueWithStrategy() CollisionResolution {
    return CollisionResolution{Action: ResolutionContinue}
}

// AcceptLeft returns a resolution that keeps the left (base) value.
func AcceptLeft() CollisionResolution {
    return CollisionResolution{Action: ResolutionAcceptLeft}
}

// AcceptLeftWithMessage returns a resolution that keeps the left value with a log message.
func AcceptLeftWithMessage(message string) CollisionResolution {
    return CollisionResolution{Action: ResolutionAcceptLeft, Message: message}
}

// AcceptRight returns a resolution that keeps the right (incoming) value.
func AcceptRight() CollisionResolution {
    return CollisionResolution{Action: ResolutionAcceptRight}
}

// AcceptRightWithMessage returns a resolution that keeps the right value with a log message.
func AcceptRightWithMessage(message string) CollisionResolution {
    return CollisionResolution{Action: ResolutionAcceptRight, Message: message}
}

// Rename returns a resolution that renames the right value to avoid collision.
func Rename() CollisionResolution {
    return CollisionResolution{Action: ResolutionRename}
}

// RenameWithMessage returns a resolution that renames with a log message.
func RenameWithMessage(message string) CollisionResolution {
    return CollisionResolution{Action: ResolutionRename, Message: message}
}

// Deduplicate returns a resolution that treats colliding values as equivalent.
func Deduplicate() CollisionResolution {
    return CollisionResolution{Action: ResolutionDeduplicate}
}

// Fail returns a resolution that aborts the join with an error.
func Fail() CollisionResolution {
    return CollisionResolution{Action: ResolutionFail}
}

// FailWithMessage returns a resolution that aborts with a custom error message.
func FailWithMessage(message string) CollisionResolution {
    return CollisionResolution{Action: ResolutionFail, Message: message}
}

// UseCustomValue returns a resolution that uses a caller-provided merged value.
func UseCustomValue(value any) CollisionResolution {
    return CollisionResolution{Action: ResolutionCustom, CustomValue: value}
}

// UseCustomValueWithMessage returns a resolution with custom value and log message.
func UseCustomValueWithMessage(value any, message string) CollisionResolution {
    return CollisionResolution{Action: ResolutionCustom, CustomValue: value, Message: message}
}
```

## Usage Examples

### Example 1: Observe-only handler (logging)

```go
result, err := joiner.JoinWithOptions(
    joiner.WithFilePaths("base.yaml", "overlay.yaml"),
    joiner.WithDefaultStrategy(joiner.StrategyFailOnCollision),
    joiner.WithCollisionHandler(func(collision joiner.CollisionContext) (joiner.CollisionResolution, error) {
        log.Printf("Collision detected: %s %q (left: %s, right: %s)",
            collision.Type, collision.Name, collision.LeftSource, collision.RightSource)

        // Defer to configured strategy
        return joiner.ContinueWithStrategy(), nil
    }),
)
```

### Example 2: Decision-only handler (conditional logic)

```go
result, err := joiner.JoinWithOptions(
    joiner.WithFilePaths("base.yaml", "overlay.yaml"),
    joiner.WithCollisionHandler(func(collision joiner.CollisionContext) (joiner.CollisionResolution, error) {
        // Always accept right for paths (overlay wins)
        if collision.Type == joiner.CollisionTypePath {
            return joiner.AcceptRightWithMessage("overlay path takes precedence"), nil
        }

        // For schemas, check if they're from a "canonical" source
        if collision.Type == joiner.CollisionTypeSchema {
            if strings.Contains(collision.LeftSource, "canonical") {
                return joiner.AcceptLeftWithMessage("canonical schema preserved"), nil
            }
        }

        // Everything else uses configured strategy
        return joiner.ContinueWithStrategy(), nil
    }),
)
```

### Example 3: Custom resolution handler (schema merging)

```go
result, err := joiner.JoinWithOptions(
    joiner.WithFilePaths("base.yaml", "overlay.yaml"),
    joiner.WithCollisionHandlerFor(
        func(collision joiner.CollisionContext) (joiner.CollisionResolution, error) {
            leftSchema, ok := collision.LeftValue.(*parser.Schema)
            if !ok {
                return joiner.ContinueWithStrategy(), nil
            }
            rightSchema, ok := collision.RightValue.(*parser.Schema)
            if !ok {
                return joiner.ContinueWithStrategy(), nil
            }

            // Custom merge: combine properties from both schemas
            merged := mergeSchemaProperties(leftSchema, rightSchema)
            return joiner.UseCustomValueWithMessage(merged,
                fmt.Sprintf("merged %d properties from both schemas", len(merged.Properties))), nil
        },
        joiner.CollisionTypeSchema, // Only handle schema collisions
    ),
)
```

### Example 4: Error handling with graceful fallback

```go
result, err := joiner.JoinWithOptions(
    joiner.WithFilePaths("base.yaml", "overlay.yaml"),
    joiner.WithDefaultStrategy(joiner.StrategyAcceptLeft),
    joiner.WithCollisionHandler(func(collision joiner.CollisionContext) (joiner.CollisionResolution, error) {
        // Try to consult external schema registry
        resolution, err := consultSchemaRegistry(collision.Name)
        if err != nil {
            // Return error - joiner will log warning and use StrategyAcceptLeft
            return joiner.CollisionResolution{}, fmt.Errorf("registry unavailable: %w", err)
        }
        return resolution, nil
    }),
)

// Check warnings for any handler fallbacks
for _, warn := range result.Warnings {
    if warn.Category == joiner.WarnHandlerError {
        log.Printf("Handler fell back: %s", warn.Message)
    }
}
```

## Testing Strategy

Tests should verify:

1. **Handler invocation** - Handler is called when collisions occur
2. **Context population** - All fields in `CollisionContext` are correctly populated
3. **Resolution actions** - Each `ResolutionAction` behaves correctly
4. **Error fallback** - Handler errors fall back to strategy with warning
5. **Type filtering** - `WithCollisionHandlerFor` only invokes for specified types
6. **Custom values** - `ResolutionCustom` uses provided value in result
7. **Source locations** - Locations populated when source maps enabled

## Implementation Files

| File | Changes |
|------|---------|
| `joiner/collision_types.go` | New file: `CollisionType`, `CollisionContext`, `ResolutionAction`, `CollisionResolution`, `CollisionHandler` |
| `joiner/collision_helpers.go` | New file: Helper functions (`AcceptLeft`, `Rename`, etc.) |
| `joiner/joiner.go` | Add `collisionHandler` and `collisionHandlerTypes` to `joinConfig`; add `WithCollisionHandler` and `WithCollisionHandlerFor` options |
| `joiner/oas3.go` | Modify collision handling to invoke handler before strategy dispatch |
| `joiner/oas2.go` | Same modifications for OAS 2.0 path |
| `joiner/warnings.go` | Add `WarnHandlerError` and `WarnHandlerResolution` categories |
| `joiner/collision_handler_test.go` | New file: Comprehensive tests |
