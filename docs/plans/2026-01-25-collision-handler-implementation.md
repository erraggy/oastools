# Collision Handler Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add collision handler callbacks to the joiner package, enabling callers to intercept, observe, or override collision resolution.

**Architecture:** Handler intercepts collisions before strategy dispatch. Single handler per joiner (caller composes). Configurable collision type filtering. Handler errors fall back gracefully to configured strategy with warning.

**Tech Stack:** Go 1.24, testify for assertions, existing joiner patterns (functional options, JoinWarning).

---

## Task 1: Create Collision Handler Types

**Files:**
- Create: `joiner/collision_handler.go`
- Test: `joiner/collision_handler_test.go`

**Step 1: Write the failing test for CollisionType constants**

```go
// joiner/collision_handler_test.go
package joiner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollisionType_Constants(t *testing.T) {
	// Verify all collision types are defined
	types := []CollisionType{
		CollisionTypeSchema,
		CollisionTypePath,
		CollisionTypeWebhook,
		CollisionTypeResponse,
		CollisionTypeParameter,
		CollisionTypeExample,
		CollisionTypeRequestBody,
		CollisionTypeHeader,
		CollisionTypeSecurityScheme,
		CollisionTypeLink,
		CollisionTypeCallback,
	}

	assert.Len(t, types, 11, "should have 11 collision types")

	// Verify each has a non-empty string value
	for _, ct := range types {
		assert.NotEmpty(t, string(ct), "collision type should have non-empty value")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./joiner -run TestCollisionType_Constants`
Expected: FAIL with "undefined: CollisionTypeSchema"

**Step 3: Write minimal implementation for CollisionType**

```go
// joiner/collision_handler.go
package joiner

// CollisionType identifies what kind of component collided.
type CollisionType string

const (
	// CollisionTypeSchema indicates a schema collision in components.schemas or definitions.
	CollisionTypeSchema CollisionType = "schema"
	// CollisionTypePath indicates a path collision in paths.
	CollisionTypePath CollisionType = "path"
	// CollisionTypeWebhook indicates a webhook collision.
	CollisionTypeWebhook CollisionType = "webhook"
	// CollisionTypeResponse indicates a response collision in components.responses.
	CollisionTypeResponse CollisionType = "response"
	// CollisionTypeParameter indicates a parameter collision in components.parameters.
	CollisionTypeParameter CollisionType = "parameter"
	// CollisionTypeExample indicates an example collision in components.examples.
	CollisionTypeExample CollisionType = "example"
	// CollisionTypeRequestBody indicates a request body collision in components.requestBodies.
	CollisionTypeRequestBody CollisionType = "requestBody"
	// CollisionTypeHeader indicates a header collision in components.headers.
	CollisionTypeHeader CollisionType = "header"
	// CollisionTypeSecurityScheme indicates a security scheme collision.
	CollisionTypeSecurityScheme CollisionType = "securityScheme"
	// CollisionTypeLink indicates a link collision in components.links.
	CollisionTypeLink CollisionType = "link"
	// CollisionTypeCallback indicates a callback collision in components.callbacks.
	CollisionTypeCallback CollisionType = "callback"
)
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./joiner -run TestCollisionType_Constants`
Expected: PASS

**Step 5: Write the failing test for ResolutionAction constants**

```go
// Add to joiner/collision_handler_test.go

func TestResolutionAction_Constants(t *testing.T) {
	// Verify resolution actions are defined with correct iota values
	assert.Equal(t, ResolutionAction(0), ResolutionContinue)
	assert.Equal(t, ResolutionAction(1), ResolutionAcceptLeft)
	assert.Equal(t, ResolutionAction(2), ResolutionAcceptRight)
	assert.Equal(t, ResolutionAction(3), ResolutionRename)
	assert.Equal(t, ResolutionAction(4), ResolutionDeduplicate)
	assert.Equal(t, ResolutionAction(5), ResolutionFail)
	assert.Equal(t, ResolutionAction(6), ResolutionCustom)
}
```

**Step 6: Run test to verify it fails**

Run: `go test -v ./joiner -run TestResolutionAction_Constants`
Expected: FAIL with "undefined: ResolutionAction"

**Step 7: Add ResolutionAction to implementation**

```go
// Add to joiner/collision_handler.go

// ResolutionAction specifies what the joiner should do after a collision.
type ResolutionAction int

const (
	// ResolutionContinue delegates to the configured strategy (observe-only).
	ResolutionContinue ResolutionAction = iota
	// ResolutionAcceptLeft keeps the left (base) value.
	ResolutionAcceptLeft
	// ResolutionAcceptRight keeps the right (incoming) value.
	ResolutionAcceptRight
	// ResolutionRename renames the right value to avoid collision.
	ResolutionRename
	// ResolutionDeduplicate treats colliding values as equivalent.
	ResolutionDeduplicate
	// ResolutionFail aborts the join with an error.
	ResolutionFail
	// ResolutionCustom uses the CustomValue provided in CollisionResolution.
	ResolutionCustom
)
```

**Step 8: Run test to verify it passes**

Run: `go test -v ./joiner -run TestResolutionAction_Constants`
Expected: PASS

**Step 9: Write the failing test for CollisionContext and CollisionResolution structs**

```go
// Add to joiner/collision_handler_test.go

func TestCollisionContext_Fields(t *testing.T) {
	ctx := CollisionContext{
		Type:               CollisionTypeSchema,
		Name:               "User",
		JSONPath:           "$.components.schemas.User",
		LeftSource:         "base.yaml",
		LeftValue:          "left-schema",
		RightSource:        "overlay.yaml",
		RightValue:         "right-schema",
		ConfiguredStrategy: StrategyAcceptLeft,
	}

	assert.Equal(t, CollisionTypeSchema, ctx.Type)
	assert.Equal(t, "User", ctx.Name)
	assert.Equal(t, "$.components.schemas.User", ctx.JSONPath)
	assert.Equal(t, "base.yaml", ctx.LeftSource)
	assert.Equal(t, "left-schema", ctx.LeftValue)
	assert.Equal(t, "overlay.yaml", ctx.RightSource)
	assert.Equal(t, "right-schema", ctx.RightValue)
	assert.Equal(t, StrategyAcceptLeft, ctx.ConfiguredStrategy)
}

func TestCollisionResolution_Fields(t *testing.T) {
	res := CollisionResolution{
		Action:      ResolutionCustom,
		CustomValue: "merged-value",
		Message:     "custom merge applied",
	}

	assert.Equal(t, ResolutionCustom, res.Action)
	assert.Equal(t, "merged-value", res.CustomValue)
	assert.Equal(t, "custom merge applied", res.Message)
}
```

**Step 10: Run test to verify it fails**

Run: `go test -v ./joiner -run 'TestCollisionContext_Fields|TestCollisionResolution_Fields'`
Expected: FAIL with "undefined: CollisionContext"

**Step 11: Add CollisionContext, CollisionResolution, and CollisionHandler to implementation**

```go
// Add to joiner/collision_handler.go

import "github.com/erraggy/oastools/parser"

// CollisionContext provides full details about a detected collision.
type CollisionContext struct {
	// Type identifies what kind of component collided.
	Type CollisionType
	// Name is the colliding name (e.g., "User", "/pets").
	Name string
	// JSONPath is the full path (e.g., "$.components.schemas.User").
	JSONPath string

	// LeftSource is the source file/identifier for left document.
	LeftSource string
	// LeftLocation is the line/column in left document (nil if unknown).
	LeftLocation *parser.Location
	// LeftValue is the left component (*parser.Schema, *parser.PathItem, etc.).
	LeftValue any

	// RightSource is the source file/identifier for right document.
	RightSource string
	// RightLocation is the line/column in right document (nil if unknown).
	RightLocation *parser.Location
	// RightValue is the right component.
	RightValue any

	// RenameInfo provides operation context if available (nil otherwise).
	RenameInfo *RenameContext

	// ConfiguredStrategy is the strategy that would apply without handler.
	ConfiguredStrategy CollisionStrategy
}

// CollisionResolution is returned by the handler to indicate desired action.
type CollisionResolution struct {
	// Action specifies what the joiner should do.
	Action ResolutionAction
	// CustomValue is used when Action is ResolutionCustom.
	CustomValue any
	// Message is an optional message for logging/warnings.
	Message string
}

// CollisionHandler is called when a collision is detected.
// Return an error to log a warning and fall back to configured strategy.
type CollisionHandler func(collision CollisionContext) (CollisionResolution, error)
```

**Step 12: Run test to verify it passes**

Run: `go test -v ./joiner -run 'TestCollisionContext_Fields|TestCollisionResolution_Fields'`
Expected: PASS

**Step 13: Run go_diagnostics**

Run: `gopls-mcp go_diagnostics` with files `["joiner/collision_handler.go", "joiner/collision_handler_test.go"]`
Expected: No errors

**Step 14: Commit**

```bash
git add joiner/collision_handler.go joiner/collision_handler_test.go
git commit -m "$(cat <<'EOF'
feat(joiner): add collision handler types

Add core types for collision handler callbacks:
- CollisionType enum for all component types
- ResolutionAction enum for handler responses
- CollisionContext struct with full collision details
- CollisionResolution struct for handler return value
- CollisionHandler function type

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Create Resolution Helper Functions

**Files:**
- Modify: `joiner/collision_handler.go`
- Test: `joiner/collision_handler_test.go`

**Step 1: Write the failing test for helper functions**

```go
// Add to joiner/collision_handler_test.go

func TestResolutionHelpers(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() CollisionResolution
		expected CollisionResolution
	}{
		{
			name: "ContinueWithStrategy",
			fn:   ContinueWithStrategy,
			expected: CollisionResolution{
				Action: ResolutionContinue,
			},
		},
		{
			name: "AcceptLeft",
			fn:   AcceptLeft,
			expected: CollisionResolution{
				Action: ResolutionAcceptLeft,
			},
		},
		{
			name: "AcceptRight",
			fn:   AcceptRight,
			expected: CollisionResolution{
				Action: ResolutionAcceptRight,
			},
		},
		{
			name: "Rename",
			fn:   Rename,
			expected: CollisionResolution{
				Action: ResolutionRename,
			},
		},
		{
			name: "Deduplicate",
			fn:   Deduplicate,
			expected: CollisionResolution{
				Action: ResolutionDeduplicate,
			},
		},
		{
			name: "Fail",
			fn:   Fail,
			expected: CollisionResolution{
				Action: ResolutionFail,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			assert.Equal(t, tt.expected.Action, got.Action)
			assert.Empty(t, got.Message)
			assert.Nil(t, got.CustomValue)
		})
	}
}

func TestResolutionHelpersWithMessage(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string) CollisionResolution
		message  string
		expected ResolutionAction
	}{
		{"AcceptLeftWithMessage", AcceptLeftWithMessage, "kept base", ResolutionAcceptLeft},
		{"AcceptRightWithMessage", AcceptRightWithMessage, "overlay wins", ResolutionAcceptRight},
		{"RenameWithMessage", RenameWithMessage, "renamed to avoid conflict", ResolutionRename},
		{"FailWithMessage", FailWithMessage, "intentional failure", ResolutionFail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.message)
			assert.Equal(t, tt.expected, got.Action)
			assert.Equal(t, tt.message, got.Message)
		})
	}
}

func TestUseCustomValue(t *testing.T) {
	customSchema := map[string]string{"type": "merged"}

	got := UseCustomValue(customSchema)
	assert.Equal(t, ResolutionCustom, got.Action)
	assert.Equal(t, customSchema, got.CustomValue)
	assert.Empty(t, got.Message)

	gotWithMsg := UseCustomValueWithMessage(customSchema, "custom merge")
	assert.Equal(t, ResolutionCustom, gotWithMsg.Action)
	assert.Equal(t, customSchema, gotWithMsg.CustomValue)
	assert.Equal(t, "custom merge", gotWithMsg.Message)
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./joiner -run 'TestResolutionHelpers|TestUseCustomValue'`
Expected: FAIL with "undefined: ContinueWithStrategy"

**Step 3: Add helper functions**

```go
// Add to joiner/collision_handler.go

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

**Step 4: Run test to verify it passes**

Run: `go test -v ./joiner -run 'TestResolutionHelpers|TestUseCustomValue'`
Expected: PASS

**Step 5: Run go_diagnostics**

Run: `gopls-mcp go_diagnostics` with files `["joiner/collision_handler.go"]`
Expected: No errors

**Step 6: Commit**

```bash
git add joiner/collision_handler.go joiner/collision_handler_test.go
git commit -m "$(cat <<'EOF'
feat(joiner): add resolution helper functions

Add convenience functions for building CollisionResolution values:
- ContinueWithStrategy, AcceptLeft, AcceptRight, Rename, Deduplicate, Fail
- WithMessage variants for logging context
- UseCustomValue for custom merge results

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Add Warning Categories for Handler Events

**Files:**
- Modify: `joiner/warnings.go`
- Test: `joiner/warnings_test.go`

**Step 1: Write the failing test for new warning categories**

```go
// Add to joiner/warnings_test.go (create new test function)

func TestHandlerWarningCategories(t *testing.T) {
	// Test WarnHandlerError
	assert.Equal(t, WarningCategory("handler_error"), WarnHandlerError)

	// Test WarnHandlerResolution
	assert.Equal(t, WarningCategory("handler_resolution"), WarnHandlerResolution)
}

func TestNewHandlerErrorWarning(t *testing.T) {
	warn := NewHandlerErrorWarning("$.components.schemas.User", "handler failed: timeout", "overlay.yaml", 42, 5)

	assert.Equal(t, WarnHandlerError, warn.Category)
	assert.Equal(t, "$.components.schemas.User", warn.Path)
	assert.Equal(t, "handler failed: timeout", warn.Message)
	assert.Equal(t, "overlay.yaml", warn.SourceFile)
	assert.Equal(t, 42, warn.Line)
	assert.Equal(t, 5, warn.Column)
	assert.Equal(t, severity.SeverityWarning, warn.Severity)
}

func TestNewHandlerResolutionWarning(t *testing.T) {
	warn := NewHandlerResolutionWarning("$.components.schemas.User", "custom merge applied", "overlay.yaml", 42, 5)

	assert.Equal(t, WarnHandlerResolution, warn.Category)
	assert.Equal(t, "$.components.schemas.User", warn.Path)
	assert.Equal(t, "custom merge applied", warn.Message)
	assert.Equal(t, "overlay.yaml", warn.SourceFile)
	assert.Equal(t, 42, warn.Line)
	assert.Equal(t, 5, warn.Column)
	assert.Equal(t, severity.SeverityInfo, warn.Severity)
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./joiner -run 'TestHandlerWarningCategories|TestNewHandlerErrorWarning|TestNewHandlerResolutionWarning'`
Expected: FAIL with "undefined: WarnHandlerError"

**Step 3: Add warning categories and constructors**

```go
// Add to joiner/warnings.go, in the const block after WarnGenericSourceName:
	// WarnHandlerError indicates a collision handler returned an error.
	WarnHandlerError WarningCategory = "handler_error"
	// WarnHandlerResolution indicates a collision handler resolved with a message.
	WarnHandlerResolution WarningCategory = "handler_resolution"
```

```go
// Add to joiner/warnings.go, after NewGenericSourceNameWarning function:

// NewHandlerErrorWarning creates a warning when a collision handler returns an error.
func NewHandlerErrorWarning(jsonPath, message, source string, line, col int) *JoinWarning {
	return &JoinWarning{
		Category:   WarnHandlerError,
		Path:       jsonPath,
		Message:    message,
		SourceFile: source,
		Line:       line,
		Column:     col,
		Severity:   severity.SeverityWarning,
	}
}

// NewHandlerResolutionWarning creates a warning when a handler provides a resolution message.
func NewHandlerResolutionWarning(jsonPath, message, source string, line, col int) *JoinWarning {
	return &JoinWarning{
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

**Step 4: Run test to verify it passes**

Run: `go test -v ./joiner -run 'TestHandlerWarningCategories|TestNewHandlerErrorWarning|TestNewHandlerResolutionWarning'`
Expected: PASS

**Step 5: Run go_diagnostics**

Run: `gopls-mcp go_diagnostics` with files `["joiner/warnings.go"]`
Expected: No errors

**Step 6: Commit**

```bash
git add joiner/warnings.go joiner/warnings_test.go
git commit -m "$(cat <<'EOF'
feat(joiner): add handler warning categories

Add warning types for collision handler events:
- WarnHandlerError: handler returned error, fell back to strategy
- WarnHandlerResolution: handler resolved with message

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Add Handler Fields to joinConfig and Options

**Files:**
- Modify: `joiner/joiner.go`
- Test: `joiner/joiner_test.go`

**Step 1: Write the failing test for WithCollisionHandler option**

```go
// Add to joiner/joiner_test.go (find existing options tests section)

func TestWithCollisionHandler(t *testing.T) {
	called := false
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		called = true
		return ContinueWithStrategy(), nil
	}

	// Create two docs that will collide
	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	assert.True(t, called, "handler should have been called")
}

func TestWithCollisionHandler_NilReturnsError(t *testing.T) {
	_, err := JoinWithOptions(
		WithFilePaths("testdata/valid-oas3.yaml", "testdata/valid-oas3-2.yaml"),
		WithCollisionHandler(nil),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "collision handler cannot be nil")
}

func TestWithCollisionHandlerFor(t *testing.T) {
	schemaHandlerCalled := false
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		if collision.Type == CollisionTypeSchema {
			schemaHandlerCalled = true
		}
		return ContinueWithStrategy(), nil
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandlerFor(handler, CollisionTypeSchema),
	)

	assert.NoError(t, err)
	assert.True(t, schemaHandlerCalled, "handler should have been called for schema collision")
}

func TestWithCollisionHandlerFor_EmptyTypesReturnsError(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return ContinueWithStrategy(), nil
	}

	_, err := JoinWithOptions(
		WithFilePaths("testdata/valid-oas3.yaml", "testdata/valid-oas3-2.yaml"),
		WithCollisionHandlerFor(handler),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one collision type must be specified")
}

// Helper function for creating test docs
func createTestOAS3Doc(sourcePath string, schemas map[string]string) parser.ParseResult {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(parser.Paths),
		Components: &parser.Components{
			Schemas: make(map[string]*parser.Schema),
		},
	}
	for name, desc := range schemas {
		doc.Components.Schemas[name] = &parser.Schema{
			Type:        "object",
			Description: desc,
		}
	}
	return parser.ParseResult{
		SourcePath: sourcePath,
		Version:    "3.0.0",
		OASVersion: parser.OASVersion30,
		Document:   doc,
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./joiner -run 'TestWithCollisionHandler'`
Expected: FAIL with "undefined: WithCollisionHandler"

**Step 3: Add collisionHandler fields to joinConfig**

```go
// Modify joiner/joiner.go - add to joinConfig struct (around line 240, after sourceMaps field):

	// Collision handler configuration
	collisionHandler      CollisionHandler
	collisionHandlerTypes map[CollisionType]bool // empty means all types
```

**Step 4: Add WithCollisionHandler and WithCollisionHandlerFor options**

```go
// Add to joiner/joiner.go after WithSourceMaps function (around line 728):

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
			return fmt.Errorf("collision handler cannot be nil")
		}
		cfg.collisionHandler = handler
		cfg.collisionHandlerTypes = nil // nil/empty means all types
		return nil
	}
}

// WithCollisionHandlerFor registers a handler for specific collision types only.
// Collisions of other types use the configured strategy without invoking the handler.
func WithCollisionHandlerFor(handler CollisionHandler, types ...CollisionType) Option {
	return func(cfg *joinConfig) error {
		if handler == nil {
			return fmt.Errorf("collision handler cannot be nil")
		}
		if len(types) == 0 {
			return fmt.Errorf("at least one collision type must be specified")
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

**Step 5: Run test to verify it passes**

Run: `go test -v ./joiner -run 'TestWithCollisionHandler'`
Expected: PASS (options are defined, but handler won't be invoked yet - that's Task 5)

**Step 6: Run go_diagnostics**

Run: `gopls-mcp go_diagnostics` with files `["joiner/joiner.go"]`
Expected: No errors

**Step 7: Commit**

```bash
git add joiner/joiner.go joiner/joiner_test.go
git commit -m "$(cat <<'EOF'
feat(joiner): add collision handler options

Add WithCollisionHandler and WithCollisionHandlerFor functional options:
- WithCollisionHandler registers handler for all collision types
- WithCollisionHandlerFor registers handler for specific types only
- Validation for nil handlers and empty type lists

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Wire Handler into Joiner Struct and JoinWithOptions

**Files:**
- Modify: `joiner/joiner.go`

**Step 1: Add handler fields to Joiner struct**

```go
// Modify joiner/joiner.go - add to Joiner struct (around line 129, after SourceMaps field):

	// collisionHandler is called when collisions are detected (nil if not configured).
	collisionHandler CollisionHandler
	// collisionHandlerTypes specifies which collision types invoke the handler.
	// Empty map means all types.
	collisionHandlerTypes map[CollisionType]bool
```

**Step 2: Wire handler from joinConfig to Joiner in JoinWithOptions**

```go
// Modify joiner/joiner.go - in JoinWithOptions function (around line 304, after setting SourceMaps):

	// Set collision handler if provided
	if cfg.collisionHandler != nil {
		j.collisionHandler = cfg.collisionHandler
		j.collisionHandlerTypes = cfg.collisionHandlerTypes
	}
```

**Step 3: Add shouldInvokeHandler method**

```go
// Add to joiner/joiner.go (after getNamespacePrefix function, around line 1116):

// shouldInvokeHandler checks if the handler wants this collision type.
func (j *Joiner) shouldInvokeHandler(collisionType CollisionType) bool {
	if j.collisionHandler == nil {
		return false
	}
	if len(j.collisionHandlerTypes) == 0 {
		return true // empty means all types
	}
	return j.collisionHandlerTypes[collisionType]
}
```

**Step 4: Run go_diagnostics**

Run: `gopls-mcp go_diagnostics` with files `["joiner/joiner.go"]`
Expected: No errors

**Step 5: Commit**

```bash
git add joiner/joiner.go
git commit -m "$(cat <<'EOF'
feat(joiner): wire handler to Joiner struct

Add collision handler fields to Joiner struct and wire from joinConfig:
- collisionHandler and collisionHandlerTypes fields on Joiner
- Copy from joinConfig in JoinWithOptions
- Add shouldInvokeHandler method for type filtering

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Integrate Handler into Schema Collision Handling (OAS3)

**Files:**
- Modify: `joiner/oas3.go`
- Test: `joiner/collision_handler_test.go`

**Step 1: Write the failing test for handler invocation on schema collision**

```go
// Add to joiner/collision_handler_test.go

func TestCollisionHandler_InvokedOnSchemaCollision(t *testing.T) {
	var receivedCollision CollisionContext
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		receivedCollision = collision
		return AcceptLeft(), nil
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	assert.Equal(t, CollisionTypeSchema, receivedCollision.Type)
	assert.Equal(t, "User", receivedCollision.Name)
	assert.Equal(t, "$.components.schemas.User", receivedCollision.JSONPath)
	assert.Equal(t, "base.yaml", receivedCollision.LeftSource)
	assert.Equal(t, "overlay.yaml", receivedCollision.RightSource)
	assert.NotNil(t, receivedCollision.LeftValue)
	assert.NotNil(t, receivedCollision.RightValue)
	assert.Equal(t, StrategyAcceptLeft, receivedCollision.ConfiguredStrategy)

	// Verify the resolution was applied (left was kept)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "base-user", oas3Doc.Components.Schemas["User"].Description)
}

func TestCollisionHandler_AcceptRightResolution(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return AcceptRight(), nil
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft), // Would keep left, but handler overrides
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "overlay-user", oas3Doc.Components.Schemas["User"].Description)
}

func TestCollisionHandler_ContinueWithStrategy(t *testing.T) {
	handlerCalled := false
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		handlerCalled = true
		return ContinueWithStrategy(), nil // Defer to configured strategy
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptRight), // Should take effect
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "overlay-user", oas3Doc.Components.Schemas["User"].Description)
}

func TestCollisionHandler_ErrorFallsBackToStrategy(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return CollisionResolution{}, fmt.Errorf("simulated handler error")
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft), // Fallback
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err, "join should succeed despite handler error")

	// Verify fallback to AcceptLeft occurred
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "base-user", oas3Doc.Components.Schemas["User"].Description)

	// Verify warning was recorded
	var foundWarning bool
	for _, warn := range result.StructuredWarnings {
		if warn.Category == WarnHandlerError {
			foundWarning = true
			assert.Contains(t, warn.Message, "simulated handler error")
		}
	}
	assert.True(t, foundWarning, "should have handler error warning")
}

func TestCollisionHandler_FailResolution(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return FailWithMessage("intentional failure from handler"), nil
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "intentional failure from handler")
}

func TestCollisionHandler_TypeFiltering(t *testing.T) {
	schemaCallCount := 0
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		schemaCallCount++
		return ContinueWithStrategy(), nil
	}

	// Create docs with both schema and path collisions
	base := createTestOAS3DocWithPaths("base.yaml",
		map[string]string{"User": "base-user"},
		[]string{"/users"},
	)
	overlay := createTestOAS3DocWithPaths("overlay.yaml",
		map[string]string{"User": "overlay-user"},
		[]string{"/users"},
	)

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandlerFor(handler, CollisionTypeSchema), // Only schemas
	)

	assert.NoError(t, err)
	assert.Equal(t, 1, schemaCallCount, "should only call for schema collision, not path")
}

// Helper that creates doc with paths
func createTestOAS3DocWithPaths(sourcePath string, schemas map[string]string, paths []string) parser.ParseResult {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(parser.Paths),
		Components: &parser.Components{
			Schemas: make(map[string]*parser.Schema),
		},
	}
	for name, desc := range schemas {
		doc.Components.Schemas[name] = &parser.Schema{
			Type:        "object",
			Description: desc,
		}
	}
	for _, path := range paths {
		doc.Paths[path] = &parser.PathItem{
			Get: &parser.Operation{
				OperationID: "get" + path,
				Responses:   make(map[string]*parser.Response),
			},
		}
	}
	return parser.ParseResult{
		SourcePath: sourcePath,
		Version:    "3.0.0",
		OASVersion: parser.OASVersion30,
		Document:   doc,
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./joiner -run 'TestCollisionHandler_InvokedOnSchemaCollision'`
Expected: FAIL (handler not yet integrated into mergeSchemas)

**Step 3: Modify mergeSchemas to invoke handler**

This is the key integration. Modify `joiner/oas3.go` `mergeSchemas` function (around line 238, before the `switch strategy` block):

```go
// Add this block before "switch strategy {" in mergeSchemas (around line 242):

			// Invoke collision handler if configured
			if j.shouldInvokeHandler(CollisionTypeSchema) {
				collision := CollisionContext{
					Type:               CollisionTypeSchema,
					Name:               effectiveName,
					JSONPath:           fmt.Sprintf("$.components.schemas.%s", effectiveName),
					LeftSource:         result.firstFilePath,
					LeftLocation:       j.getLocationPtr(result.firstFilePath, fmt.Sprintf("$.components.schemas.%s", effectiveName)),
					LeftValue:          target[effectiveName],
					RightSource:        ctx.filePath,
					RightLocation:      j.getLocationPtr(ctx.filePath, fmt.Sprintf("$.components.schemas.%s", name)),
					RightValue:         schema,
					RenameInfo:         buildRenameContextPtr(effectiveName, ctx.filePath, ctx.docIndex, sourceGraph, j.config.PrimaryOperationPolicy),
					ConfiguredStrategy: strategy,
				}

				resolution, handlerErr := j.collisionHandler(collision)
				if handlerErr != nil {
					// Log warning and fall back to configured strategy
					line, col := j.getLocation(ctx.filePath, collision.JSONPath)
					result.AddWarning(NewHandlerErrorWarning(
						collision.JSONPath,
						fmt.Sprintf("collision handler error: %v; using %s strategy", handlerErr, strategy),
						ctx.filePath, line, col,
					))
					// Fall through to strategy switch below
				} else {
					// Apply the resolution
					applied, err := j.applySchemaResolution(collision, resolution, target, result, ctx, sourceGraph)
					if err != nil {
						return err
					}
					if applied {
						continue // Resolution handled, skip strategy switch
					}
					// ResolutionContinue falls through to strategy switch
				}
			}
```

**Step 4: Add helper methods for handler integration**

```go
// Add to joiner/joiner.go (after getLocation function):

// getLocationPtr returns a *parser.Location for the given file and JSON path.
// Returns nil if no SourceMap is available or path not found.
func (j *Joiner) getLocationPtr(filePath, jsonPath string) *parser.Location {
	line, col := j.getLocation(filePath, jsonPath)
	if line == 0 {
		return nil
	}
	return &parser.Location{Line: line, Column: col}
}
```

```go
// Add to joiner/rename_context.go (after buildRenameContext function):

// buildRenameContextPtr is like buildRenameContext but returns a pointer.
// Returns nil if the context would have no useful information.
func buildRenameContextPtr(
	schemaName string,
	sourcePath string,
	docIndex int,
	graph *RefGraph,
	policy PrimaryOperationPolicy,
) *RenameContext {
	ctx := buildRenameContext(schemaName, sourcePath, docIndex, graph, policy)
	return &ctx
}
```

**Step 5: Add applySchemaResolution method**

```go
// Add to joiner/oas3.go (after mergeSchemas function):

// applySchemaResolution applies a CollisionResolution to a schema collision.
// Returns true if the resolution was fully handled, false if strategy should still be applied.
func (j *Joiner) applySchemaResolution(
	collision CollisionContext,
	resolution CollisionResolution,
	target map[string]*parser.Schema,
	result *JoinResult,
	ctx documentContext,
	sourceGraph *RefGraph,
) (bool, error) {
	// Record message as warning if provided
	if resolution.Message != "" {
		line, col := j.getLocation(ctx.filePath, collision.JSONPath)
		result.AddWarning(NewHandlerResolutionWarning(collision.JSONPath, resolution.Message, ctx.filePath, line, col))
	}

	schema, ok := collision.RightValue.(*parser.Schema)
	if !ok {
		return false, fmt.Errorf("collision handler: RightValue is not a *parser.Schema")
	}

	switch resolution.Action {
	case ResolutionContinue:
		// Delegate to configured strategy
		return false, nil

	case ResolutionAcceptLeft:
		// Keep existing (left), discard incoming (right)
		j.recordCollisionEvent(result, collision.Name, collision.LeftSource, collision.RightSource, collision.ConfiguredStrategy, "kept-left", "")
		return true, nil

	case ResolutionAcceptRight:
		// Replace with incoming (right)
		target[collision.Name] = schema
		j.recordCollisionEvent(result, collision.Name, collision.LeftSource, collision.RightSource, collision.ConfiguredStrategy, "kept-right", "")
		return true, nil

	case ResolutionRename:
		// Rename right schema
		newName := j.generateRenamedSchemaName(collision.Name, ctx.filePath, ctx.docIndex, sourceGraph)
		target[newName] = schema
		if result.rewriter == nil {
			result.rewriter = NewSchemaRewriter()
		}
		result.rewriter.RegisterRename(collision.Name, newName, result.OASVersion)
		line, col := j.getLocation(ctx.filePath, collision.JSONPath)
		result.AddWarning(NewSchemaRenamedWarning(collision.Name, newName, "schema", ctx.filePath, line, col, false))
		j.recordCollisionEvent(result, collision.Name, collision.LeftSource, collision.RightSource, collision.ConfiguredStrategy, "renamed", newName)
		return true, nil

	case ResolutionDeduplicate:
		// Keep left, discard right (treat as equivalent)
		line, col := j.getLocation(ctx.filePath, collision.JSONPath)
		result.AddWarning(NewSchemaDedupWarning(collision.Name, "schema", ctx.filePath, line, col))
		j.recordCollisionEvent(result, collision.Name, collision.LeftSource, collision.RightSource, collision.ConfiguredStrategy, "deduplicated", "")
		return true, nil

	case ResolutionFail:
		msg := "collision handler requested failure"
		if resolution.Message != "" {
			msg = resolution.Message
		}
		return true, &CollisionError{
			Section:    "components.schemas",
			Key:        collision.Name,
			FirstFile:  collision.LeftSource,
			SecondFile: collision.RightSource,
			Strategy:   collision.ConfiguredStrategy,
			Message:    msg,
		}

	case ResolutionCustom:
		if resolution.CustomValue == nil {
			return true, fmt.Errorf("collision handler: ResolutionCustom requires CustomValue")
		}
		customSchema, ok := resolution.CustomValue.(*parser.Schema)
		if !ok {
			return true, fmt.Errorf("collision handler: CustomValue must be *parser.Schema for schema collisions")
		}
		target[collision.Name] = customSchema
		j.recordCollisionEvent(result, collision.Name, collision.LeftSource, collision.RightSource, collision.ConfiguredStrategy, "custom", "")
		return true, nil

	default:
		return true, fmt.Errorf("collision handler: unknown resolution action: %d", resolution.Action)
	}
}
```

**Step 6: Add Message field to CollisionError**

```go
// Modify joiner/joiner.go - add Message field to CollisionError struct (find CollisionError struct):
	// Message is an optional custom error message (from handler).
	Message string
```

```go
// Modify joiner/joiner.go - update CollisionError.Error() method to include Message:
func (e *CollisionError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("collision error in %s.%s: %s", e.Section, e.Key, e.Message)
	}
	// ... existing error formatting
}
```

**Step 7: Run tests to verify they pass**

Run: `go test -v ./joiner -run 'TestCollisionHandler_'`
Expected: PASS

**Step 8: Run go_diagnostics**

Run: `gopls-mcp go_diagnostics` with files `["joiner/oas3.go", "joiner/joiner.go", "joiner/rename_context.go"]`
Expected: No errors

**Step 9: Commit**

```bash
git add joiner/oas3.go joiner/joiner.go joiner/rename_context.go joiner/collision_handler_test.go
git commit -m "$(cat <<'EOF'
feat(joiner): integrate collision handler into schema merging

Invoke collision handler before strategy dispatch in mergeSchemas:
- Build CollisionContext with full details
- Apply resolution or fall back to strategy on error
- Support all resolution actions (accept, rename, deduplicate, fail, custom)
- Add getLocationPtr and buildRenameContextPtr helpers
- Add Message field to CollisionError for handler messages

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Integrate Handler into Other Component Collision Handling (OAS3)

**Files:**
- Modify: `joiner/oas3.go`
- Test: `joiner/collision_handler_test.go`

**Step 1: Write the failing test for path collision handler**

```go
// Add to joiner/collision_handler_test.go

func TestCollisionHandler_PathCollision(t *testing.T) {
	var receivedCollision CollisionContext
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		receivedCollision = collision
		return AcceptRight(), nil
	}

	base := createTestOAS3DocWithPaths("base.yaml", nil, []string{"/users"})
	overlay := createTestOAS3DocWithPaths("overlay.yaml", nil, []string{"/users"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	assert.Equal(t, CollisionTypePath, receivedCollision.Type)
	assert.Equal(t, "/users", receivedCollision.Name)
	assert.Contains(t, receivedCollision.JSONPath, "paths")

	// Verify overlay path was used (AcceptRight)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "get/users", oas3Doc.Paths["/users"].Get.OperationID) // overlay has this ID
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./joiner -run 'TestCollisionHandler_PathCollision'`
Expected: FAIL (path collisions don't invoke handler yet)

**Step 3: Modify mergePathsMap to invoke handler**

Find `mergePathsMap` in `joiner/oas3.go` and add handler invocation similar to mergeSchemas. The collision handling is in the `if _, exists := target[path]` block.

**Step 4: Modify mergeMap generic function for other components**

The `mergeMap` function in `joiner/oas3.go` handles responses, parameters, examples, etc. Add handler invocation there.

**Step 5: Modify webhook collision handling in joinOAS3Documents**

The webhook collision handling is inline in `joinOAS3Documents`. Extract and add handler support.

**Step 6: Run tests to verify they pass**

Run: `go test -v ./joiner -run 'TestCollisionHandler_'`
Expected: PASS

**Step 7: Run go_diagnostics**

Run: `gopls-mcp go_diagnostics` with files `["joiner/oas3.go"]`
Expected: No errors

**Step 8: Commit**

```bash
git add joiner/oas3.go joiner/collision_handler_test.go
git commit -m "$(cat <<'EOF'
feat(joiner): integrate collision handler for all OAS3 components

Extend collision handler support to all component types:
- Paths (mergePathsMap)
- Webhooks (joinOAS3Documents inline)
- Responses, Parameters, Examples, RequestBodies, Headers,
  SecuritySchemes, Links, Callbacks (mergeMap)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Integrate Handler into OAS2 Collision Handling

**Files:**
- Modify: `joiner/oas2.go`
- Test: `joiner/collision_handler_test.go`

**Step 1: Write the failing test for OAS2 collision handler**

```go
// Add to joiner/collision_handler_test.go

func TestCollisionHandler_OAS2SchemaCollision(t *testing.T) {
	var receivedCollision CollisionContext
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		receivedCollision = collision
		return AcceptLeft(), nil
	}

	base := createTestOAS2Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS2Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	assert.Equal(t, CollisionTypeSchema, receivedCollision.Type)
	assert.Equal(t, "User", receivedCollision.Name)
	assert.Contains(t, receivedCollision.JSONPath, "definitions")

	// Verify the resolution was applied
	oas2Doc, _ := result.Document.(*parser.OAS2Document)
	assert.Equal(t, "base-user", oas2Doc.Definitions["User"].Description)
}

func createTestOAS2Doc(sourcePath string, definitions map[string]string) parser.ParseResult {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths:       make(parser.Paths),
		Definitions: make(map[string]*parser.Schema),
	}
	for name, desc := range definitions {
		doc.Definitions[name] = &parser.Schema{
			Type:        "object",
			Description: desc,
		}
	}
	return parser.ParseResult{
		SourcePath: sourcePath,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./joiner -run 'TestCollisionHandler_OAS2'`
Expected: FAIL (OAS2 doesn't invoke handler yet)

**Step 3: Add handler integration to OAS2 collision handling**

Modify `joiner/oas2.go` to invoke handler similar to OAS3 integration.

**Step 4: Run tests to verify they pass**

Run: `go test -v ./joiner -run 'TestCollisionHandler_OAS2'`
Expected: PASS

**Step 5: Run go_diagnostics**

Run: `gopls-mcp go_diagnostics` with files `["joiner/oas2.go"]`
Expected: No errors

**Step 6: Commit**

```bash
git add joiner/oas2.go joiner/collision_handler_test.go
git commit -m "$(cat <<'EOF'
feat(joiner): integrate collision handler for OAS2 documents

Extend collision handler support to OAS 2.0 documents:
- Definitions (schema equivalents)
- Paths
- Parameters, Responses (global)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Add Custom Value Test and Documentation

**Files:**
- Modify: `joiner/collision_handler_test.go`
- Modify: `joiner/collision_handler.go` (add package docs)

**Step 1: Write comprehensive test for custom value resolution**

```go
// Add to joiner/collision_handler_test.go

func TestCollisionHandler_CustomValue(t *testing.T) {
	customSchema := &parser.Schema{
		Type:        "object",
		Description: "custom-merged-schema",
		Properties: map[string]*parser.Schema{
			"merged": {Type: "boolean"},
		},
	}

	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return UseCustomValueWithMessage(customSchema, "merged both schemas"), nil
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "custom-merged-schema", oas3Doc.Components.Schemas["User"].Description)
	assert.Contains(t, oas3Doc.Components.Schemas["User"].Properties, "merged")

	// Verify warning was recorded
	var foundWarning bool
	for _, warn := range result.StructuredWarnings {
		if warn.Category == WarnHandlerResolution && warn.Message == "merged both schemas" {
			foundWarning = true
		}
	}
	assert.True(t, foundWarning)
}

func TestCollisionHandler_CustomValueWrongType(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return UseCustomValue("not a schema"), nil // Wrong type
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithCollisionHandler(handler),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CustomValue must be *parser.Schema")
}

func TestCollisionHandler_CustomValueNil(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return CollisionResolution{Action: ResolutionCustom, CustomValue: nil}, nil
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithCollisionHandler(handler),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ResolutionCustom requires CustomValue")
}
```

**Step 2: Run tests to verify they pass**

Run: `go test -v ./joiner -run 'TestCollisionHandler_CustomValue'`
Expected: PASS

**Step 3: Add package documentation to collision_handler.go**

```go
// Add to top of joiner/collision_handler.go, after package statement:

/*
Collision Handler Support

The joiner package supports collision handlers for custom collision resolution.
A collision handler is called when two documents being joined have conflicting
components (schemas, paths, webhooks, etc.).

Basic usage:

	result, err := joiner.JoinWithOptions(
	    joiner.WithFilePaths("base.yaml", "overlay.yaml"),
	    joiner.WithCollisionHandler(func(collision joiner.CollisionContext) (joiner.CollisionResolution, error) {
	        // Log all collisions
	        log.Printf("Collision: %s %s", collision.Type, collision.Name)
	        // Defer to configured strategy
	        return joiner.ContinueWithStrategy(), nil
	    }),
	)

Handler capabilities:

1. Observe-only: Return ContinueWithStrategy() to log/observe and defer to strategy
2. Decision-only: Return AcceptLeft(), AcceptRight(), Rename(), etc. to override strategy
3. Custom resolution: Return UseCustomValue(mergedSchema) to provide a custom merged value

Error handling:

If the handler returns an error, the joiner logs a warning and falls back to the
configured strategy. This ensures handlers cannot break the join operation.

Type filtering:

Use WithCollisionHandlerFor to handle only specific collision types:

	joiner.WithCollisionHandlerFor(handler, joiner.CollisionTypeSchema, joiner.CollisionTypePath)

See the CollisionContext, CollisionResolution, and helper function documentation
for complete details.
*/
```

**Step 4: Run tests to verify they pass**

Run: `go test -v ./joiner -run 'TestCollisionHandler'`
Expected: PASS

**Step 5: Run go_diagnostics**

Run: `gopls-mcp go_diagnostics` with files `["joiner/collision_handler.go"]`
Expected: No errors

**Step 6: Commit**

```bash
git add joiner/collision_handler.go joiner/collision_handler_test.go
git commit -m "$(cat <<'EOF'
docs(joiner): add collision handler documentation and tests

Add comprehensive tests for custom value resolution:
- Valid custom schema replacement
- Error on wrong custom value type
- Error on nil custom value

Add package-level documentation explaining:
- Basic usage patterns
- Three capability levels (observe, decide, custom)
- Error handling behavior
- Type filtering

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Run Full Test Suite and Final Verification

**Files:**
- All joiner package files

**Step 1: Run all joiner tests**

Run: `go test -v ./joiner/...`
Expected: All tests PASS

**Step 2: Run go_diagnostics on all modified files**

Run: `gopls-mcp go_diagnostics` with files from joiner package
Expected: No errors or warnings

**Step 3: Run linter**

Run: `make lint` or `golangci-lint run ./joiner/...`
Expected: No lint errors

**Step 4: Run full test suite**

Run: `make test` or `go test ./...`
Expected: All tests PASS

**Step 5: Final commit if any cleanup needed**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore(joiner): final cleanup for collision handler feature

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Summary

This implementation plan adds collision handler support to the joiner package through 10 incremental tasks:

1. **Core types** - CollisionType, ResolutionAction, CollisionContext, CollisionResolution, CollisionHandler
2. **Helper functions** - ContinueWithStrategy, AcceptLeft, AcceptRight, etc.
3. **Warning categories** - WarnHandlerError, WarnHandlerResolution
4. **Options API** - WithCollisionHandler, WithCollisionHandlerFor
5. **Joiner wiring** - Handler fields on Joiner struct, shouldInvokeHandler method
6. **OAS3 schema integration** - Handler invocation in mergeSchemas
7. **OAS3 other components** - Paths, webhooks, responses, parameters, etc.
8. **OAS2 integration** - Definitions, paths, parameters
9. **Documentation and edge cases** - Custom value tests, package docs
10. **Final verification** - Full test suite, linting, cleanup

Each task follows TDD with explicit test-first steps, run commands, and expected outputs.
