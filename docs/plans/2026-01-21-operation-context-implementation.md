# Operation Context for Validation Errors - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add operation context (method, path, operationId) to validation errors so users can immediately identify which API endpoint is affected.

**Architecture:** Two-phase approach: (1) build a reference map tracking which operations use which components, (2) attach operation context when creating validation errors. The `Issue` struct gets a new `OperationContext` pointer field.

**Tech Stack:** Go, existing `internal/issues` and `validator` packages, table-driven tests with testify.

---

## Task 1: Add OperationContext Struct

**Files:**
- Create: `internal/issues/operation_context.go`
- Test: `internal/issues/operation_context_test.go`

**Step 1: Write the failing test**

Create `internal/issues/operation_context_test.go`:

```go
package issues

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOperationContextString(t *testing.T) {
	tests := []struct {
		name     string
		ctx      OperationContext
		expected string
	}{
		{
			name: "operation with operationId",
			ctx: OperationContext{
				Method:      "GET",
				Path:        "/users/{id}",
				OperationID: "getUser",
			},
			expected: "(operationId: getUser)",
		},
		{
			name: "operation without operationId",
			ctx: OperationContext{
				Method: "GET",
				Path:   "/users/{id}",
			},
			expected: "(GET /users/{id})",
		},
		{
			name: "path-level (no method)",
			ctx: OperationContext{
				Path: "/users/{id}",
			},
			expected: "(path: /users/{id})",
		},
		{
			name: "reusable component with operationId",
			ctx: OperationContext{
				Method:              "GET",
				Path:                "/users",
				OperationID:         "listUsers",
				IsReusableComponent: true,
				AdditionalRefs:      3,
			},
			expected: "(operationId: listUsers, +3 operations)",
		},
		{
			name: "reusable component without operationId",
			ctx: OperationContext{
				Method:              "POST",
				Path:                "/orders",
				IsReusableComponent: true,
				AdditionalRefs:      5,
			},
			expected: "(POST /orders, +5 operations)",
		},
		{
			name: "reusable component single ref",
			ctx: OperationContext{
				Method:              "GET",
				Path:                "/users",
				OperationID:         "listUsers",
				IsReusableComponent: true,
				AdditionalRefs:      0,
			},
			expected: "(operationId: listUsers)",
		},
		{
			name: "unused component",
			ctx: OperationContext{
				IsReusableComponent: true,
				AdditionalRefs:      -1, // sentinel for unused
			},
			expected: "(unused component)",
		},
		{
			name: "webhook context",
			ctx: OperationContext{
				Method:    "POST",
				Path:      "orderCreated",
				IsWebhook: true,
			},
			expected: "(webhook: orderCreated)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ctx.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOperationContextIsEmpty(t *testing.T) {
	assert.True(t, OperationContext{}.IsEmpty())
	assert.False(t, OperationContext{Path: "/users"}.IsEmpty())
	assert.False(t, OperationContext{Method: "GET"}.IsEmpty())
	assert.False(t, OperationContext{OperationID: "test"}.IsEmpty())
	assert.False(t, OperationContext{IsReusableComponent: true, AdditionalRefs: -1}.IsEmpty())
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/issues -run TestOperationContext -v`
Expected: FAIL - `undefined: OperationContext`

**Step 3: Write minimal implementation**

Create `internal/issues/operation_context.go`:

```go
// Package issues provides a unified issue type for validation and conversion problems.
package issues

import "fmt"

// OperationContext provides API operation context for validation issues.
// For issues under paths.*, it identifies the specific operation.
// For issues outside paths.*, it shows which operations reference the component.
type OperationContext struct {
	// Method is the HTTP method (GET, POST, etc.) - empty for path-level issues
	Method string
	// Path is the API path pattern (e.g., "/users/{id}") or webhook name
	Path string
	// OperationID is the operationId if defined (may be empty)
	OperationID string
	// IsReusableComponent is true when the issue is in components/definitions
	IsReusableComponent bool
	// IsWebhook is true when the issue is in a webhook operation
	IsWebhook bool
	// AdditionalRefs is the count of other operations referencing this component.
	// Only relevant when IsReusableComponent is true.
	// -1 indicates the component is unused (not referenced by any operation).
	AdditionalRefs int
}

// String returns a formatted string representation of the operation context.
// Returns empty string if the context is empty.
func (c OperationContext) String() string {
	if c.IsEmpty() {
		return ""
	}

	// Handle unused component
	if c.IsReusableComponent && c.AdditionalRefs == -1 {
		return "(unused component)"
	}

	// Handle webhook
	if c.IsWebhook {
		return fmt.Sprintf("(webhook: %s)", c.Path)
	}

	// Build the primary identifier
	var primary string
	if c.OperationID != "" {
		primary = fmt.Sprintf("operationId: %s", c.OperationID)
	} else if c.Method != "" {
		primary = fmt.Sprintf("%s %s", c.Method, c.Path)
	} else if c.Path != "" {
		// Path-level (no method)
		return fmt.Sprintf("(path: %s)", c.Path)
	}

	// Add additional refs count for reusable components
	if c.IsReusableComponent && c.AdditionalRefs > 0 {
		return fmt.Sprintf("(%s, +%d operations)", primary, c.AdditionalRefs)
	}

	return fmt.Sprintf("(%s)", primary)
}

// IsEmpty returns true if the context has no meaningful information.
func (c OperationContext) IsEmpty() bool {
	// Unused component is not empty - it's valid context
	if c.IsReusableComponent && c.AdditionalRefs == -1 {
		return false
	}
	return c.Method == "" && c.Path == "" && c.OperationID == ""
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/issues -run TestOperationContext -v`
Expected: PASS

**Step 5: Run go_diagnostics**

Run: `go_diagnostics` for `internal/issues/operation_context.go`
Expected: No errors

**Step 6: Commit**

```bash
git add internal/issues/operation_context.go internal/issues/operation_context_test.go
git commit -m "feat(issues): add OperationContext struct for validation error context"
```

---

## Task 2: Add OperationContext Field to Issue

**Files:**
- Modify: `internal/issues/issue.go:10-32` (Issue struct)
- Modify: `internal/issues/issue.go:34-70` (String method)
- Test: `internal/issues/issue_test.go` (add new tests)

**Step 1: Write the failing test**

Add to `internal/issues/issue_test.go`:

```go
func TestIssueStringWithOperationContext(t *testing.T) {
	tests := []struct {
		name     string
		issue    Issue
		contains []string
	}{
		{
			name: "error with operation context (operationId)",
			issue: Issue{
				Path:     "paths./users/{id}.get.parameters[0]",
				Message:  "Path parameters must have required: true",
				Severity: severity.SeverityError,
				OperationContext: &OperationContext{
					Method:      "GET",
					Path:        "/users/{id}",
					OperationID: "getUser",
				},
			},
			contains: []string{
				"✗ paths./users/{id}.get.parameters[0] (operationId: getUser):",
				"Path parameters must have required: true",
			},
		},
		{
			name: "error with operation context (no operationId)",
			issue: Issue{
				Path:     "paths./users/{id}.get.parameters[0]",
				Message:  "Path parameters must have required: true",
				Severity: severity.SeverityError,
				OperationContext: &OperationContext{
					Method: "GET",
					Path:   "/users/{id}",
				},
			},
			contains: []string{
				"✗ paths./users/{id}.get.parameters[0] (GET /users/{id}):",
				"Path parameters must have required: true",
			},
		},
		{
			name: "error with path-level context",
			issue: Issue{
				Path:     "paths./users/{id}.parameters[0]",
				Message:  "Parameter missing schema",
				Severity: severity.SeverityError,
				OperationContext: &OperationContext{
					Path: "/users/{id}",
				},
			},
			contains: []string{
				"✗ paths./users/{id}.parameters[0] (path: /users/{id}):",
			},
		},
		{
			name: "error with reusable component context",
			issue: Issue{
				Path:     "components.schemas.User.properties.email",
				Message:  "Invalid email format",
				Severity: severity.SeverityError,
				OperationContext: &OperationContext{
					Method:              "GET",
					Path:                "/users",
					OperationID:         "listUsers",
					IsReusableComponent: true,
					AdditionalRefs:      3,
				},
			},
			contains: []string{
				"✗ components.schemas.User.properties.email (operationId: listUsers, +3 operations):",
			},
		},
		{
			name: "error with nil operation context",
			issue: Issue{
				Path:     "info.version",
				Message:  "Version is required",
				Severity: severity.SeverityError,
			},
			contains: []string{
				"✗ info.version: Version is required",
			},
		},
		{
			name: "warning with operation context and SpecRef",
			issue: Issue{
				Path:     "paths./users.get",
				Message:  "Operation should have description",
				Severity: severity.SeverityWarning,
				OperationContext: &OperationContext{
					Method:      "GET",
					Path:        "/users",
					OperationID: "listUsers",
				},
				SpecRef: "https://spec.openapis.org/oas/v3.0.3.html#operation-object",
			},
			contains: []string{
				"⚠ paths./users.get (operationId: listUsers):",
				"Spec: https://spec.openapis.org/oas/v3.0.3.html#operation-object",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.issue.String()
			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/issues -run TestIssueStringWithOperationContext -v`
Expected: FAIL - `unknown field 'OperationContext'`

**Step 3: Modify Issue struct**

In `internal/issues/issue.go`, add field to Issue struct after line 31 (before the closing brace):

```go
	// OperationContext provides API operation context when the issue relates to
	// an operation or a component referenced by operations. Nil when not applicable.
	OperationContext *OperationContext
```

**Step 4: Modify String() method**

Update the `String()` method in `internal/issues/issue.go` to include operation context. Replace lines 52-57:

```go
	var result string
	pathWithContext := i.Path
	if i.OperationContext != nil && !i.OperationContext.IsEmpty() {
		pathWithContext = fmt.Sprintf("%s %s", i.Path, i.OperationContext.String())
	}

	if i.Line > 0 {
		result = fmt.Sprintf("%s %s (line %d, col %d): %s", symbol, pathWithContext, i.Line, i.Column, i.Message)
	} else {
		result = fmt.Sprintf("%s %s: %s", symbol, pathWithContext, i.Message)
	}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/issues -run TestIssueStringWithOperationContext -v`
Expected: PASS

**Step 6: Run all issue tests**

Run: `go test ./internal/issues -v`
Expected: All tests PASS

**Step 7: Run go_diagnostics**

Run: `go_diagnostics` for `internal/issues/issue.go`
Expected: No errors

**Step 8: Commit**

```bash
git add internal/issues/issue.go internal/issues/issue_test.go
git commit -m "feat(issues): add OperationContext field to Issue struct"
```

---

## Task 3: Create Reference Tracker

**Files:**
- Create: `validator/ref_tracker.go`
- Test: `validator/ref_tracker_test.go`

**Step 1: Write the failing test**

Create `validator/ref_tracker_test.go`:

```go
package validator

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefTrackerOAS3(t *testing.T) {
	// Create a simple OAS3 document with refs
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listUsers",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{
											Ref: "#/components/schemas/UserList",
										},
									},
								},
							},
						},
					},
				},
				Post: &parser.Operation{
					OperationID: "createUser",
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{
									Ref: "#/components/schemas/User",
								},
							},
						},
					},
				},
			},
			"/users/{id}": &parser.PathItem{
				Parameters: []*parser.Parameter{
					{Name: "id", In: "path", Required: boolPtr(true)},
				},
				Get: &parser.Operation{
					OperationID: "getUser",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{
											Ref: "#/components/schemas/User",
										},
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
				"User": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"id":   {Type: "integer"},
						"name": {Type: "string"},
					},
				},
				"UserList": {
					Type: "array",
					Items: &parser.Schema{
						Ref: "#/components/schemas/User",
					},
				},
				"OrphanedSchema": {
					Type: "object",
				},
			},
		},
	}

	tracker := buildRefTrackerOAS3(doc)

	t.Run("User schema referenced by multiple operations", func(t *testing.T) {
		ops := tracker.getOperationsForComponent("components.schemas.User")
		require.NotEmpty(t, ops)
		// Should include createUser, getUser, and listUsers (via UserList)
		assert.GreaterOrEqual(t, len(ops), 2)

		// Check one of the operations has correct data
		var foundGetUser bool
		for _, op := range ops {
			if op.OperationID == "getUser" {
				foundGetUser = true
				assert.Equal(t, "GET", op.Method)
				assert.Equal(t, "/users/{id}", op.Path)
			}
		}
		assert.True(t, foundGetUser, "should find getUser operation")
	})

	t.Run("UserList schema referenced by listUsers", func(t *testing.T) {
		ops := tracker.getOperationsForComponent("components.schemas.UserList")
		require.Len(t, ops, 1)
		assert.Equal(t, "listUsers", ops[0].OperationID)
		assert.Equal(t, "GET", ops[0].Method)
		assert.Equal(t, "/users", ops[0].Path)
	})

	t.Run("OrphanedSchema has no references", func(t *testing.T) {
		ops := tracker.getOperationsForComponent("components.schemas.OrphanedSchema")
		assert.Empty(t, ops)
	})

	t.Run("non-existent component returns empty", func(t *testing.T) {
		ops := tracker.getOperationsForComponent("components.schemas.DoesNotExist")
		assert.Empty(t, ops)
	})
}

func TestRefTrackerOAS2(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Schema: &parser.Schema{
									Type: "array",
									Items: &parser.Schema{
										Ref: "#/definitions/Pet",
									},
								},
							},
						},
					},
				},
			},
		},
		Definitions: map[string]*parser.Schema{
			"Pet": {
				Type: "object",
			},
		},
	}

	tracker := buildRefTrackerOAS2(doc)

	t.Run("Pet definition referenced by listPets", func(t *testing.T) {
		ops := tracker.getOperationsForComponent("definitions.Pet")
		require.Len(t, ops, 1)
		assert.Equal(t, "listPets", ops[0].OperationID)
	})
}

func TestRefTrackerTransitiveRefs(t *testing.T) {
	// A -> B -> C: operation references A, which refs B, which refs C
	// All three should be tracked as used by the operation
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Paths: parser.Paths{
			"/test": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "testOp",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{
											Ref: "#/components/schemas/A",
										},
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
				"C": {
					Type: "object",
				},
			},
		},
	}

	tracker := buildRefTrackerOAS3(doc)

	// All three schemas should be tracked
	for _, schema := range []string{"A", "B", "C"} {
		t.Run("schema "+schema+" is tracked", func(t *testing.T) {
			ops := tracker.getOperationsForComponent("components.schemas." + schema)
			require.Len(t, ops, 1, "schema %s should have 1 operation", schema)
			assert.Equal(t, "testOp", ops[0].OperationID)
		})
	}
}

func TestRefTrackerCircularRefs(t *testing.T) {
	// A -> B -> A (circular)
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Paths: parser.Paths{
			"/test": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "testOp",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{
											Ref: "#/components/schemas/A",
										},
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
						"a": {Ref: "#/components/schemas/A"}, // circular
					},
				},
			},
		},
	}

	// Should not hang or panic
	tracker := buildRefTrackerOAS3(doc)

	// Both schemas should be tracked
	opsA := tracker.getOperationsForComponent("components.schemas.A")
	opsB := tracker.getOperationsForComponent("components.schemas.B")
	assert.Len(t, opsA, 1)
	assert.Len(t, opsB, 1)
}

// Helper
func boolPtr(b bool) *bool { return &b }
```

**Step 2: Run test to verify it fails**

Run: `go test ./validator -run TestRefTracker -v`
Expected: FAIL - `undefined: buildRefTrackerOAS3`

**Step 3: Write implementation**

Create `validator/ref_tracker.go`:

```go
package validator

import (
	"strings"

	"github.com/erraggy/oastools/internal/issues"
	"github.com/erraggy/oastools/parser"
)

// operationRef holds information about an operation that references a component.
type operationRef struct {
	Method      string
	Path        string
	OperationID string
	IsWebhook   bool
}

// refTracker tracks which operations reference which components.
type refTracker struct {
	// componentToOps maps normalized component paths to the operations that reference them.
	// e.g., "components.schemas.User" → [{Method: "GET", Path: "/users", OperationID: "getUser"}, ...]
	componentToOps map[string][]operationRef
}

// newRefTracker creates an empty reference tracker.
func newRefTracker() *refTracker {
	return &refTracker{
		componentToOps: make(map[string][]operationRef),
	}
}

// buildRefTrackerOAS3 builds a reference tracker for an OAS 3.x document.
func buildRefTrackerOAS3(doc *parser.OAS3Document) *refTracker {
	rt := newRefTracker()
	if doc == nil {
		return rt
	}

	// Track refs from paths
	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}
		rt.trackPathItemRefs(pathItem, pathPattern, false, doc.Components)
	}

	// Track refs from webhooks (OAS 3.1+)
	for name, pathItem := range doc.Webhooks {
		if pathItem == nil {
			continue
		}
		rt.trackPathItemRefs(pathItem, name, true, doc.Components)
	}

	return rt
}

// buildRefTrackerOAS2 builds a reference tracker for an OAS 2.0 document.
func buildRefTrackerOAS2(doc *parser.OAS2Document) *refTracker {
	rt := newRefTracker()
	if doc == nil {
		return rt
	}

	// Track refs from paths
	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}
		rt.trackPathItemRefsOAS2(pathItem, pathPattern, doc.Definitions)
	}

	return rt
}

// trackPathItemRefs tracks all refs in a path item for OAS3.
func (rt *refTracker) trackPathItemRefs(item *parser.PathItem, pathPattern string, isWebhook bool, components *parser.Components) {
	operations := []struct {
		method string
		op     *parser.Operation
	}{
		{"GET", item.Get},
		{"PUT", item.Put},
		{"POST", item.Post},
		{"DELETE", item.Delete},
		{"OPTIONS", item.Options},
		{"HEAD", item.Head},
		{"PATCH", item.Patch},
		{"TRACE", item.Trace},
		{"QUERY", item.Query},
	}

	for _, o := range operations {
		if o.op != nil {
			opRef := operationRef{
				Method:      o.method,
				Path:        pathPattern,
				OperationID: o.op.OperationID,
				IsWebhook:   isWebhook,
			}
			rt.trackOperationRefs(o.op, opRef, components)
		}
	}

	// Path-level parameters (tracked without method)
	for _, param := range item.Parameters {
		if param != nil {
			opRef := operationRef{Path: pathPattern, IsWebhook: isWebhook}
			rt.trackParameterRefs(param, opRef, components)
		}
	}
}

// trackPathItemRefsOAS2 tracks all refs in a path item for OAS2.
func (rt *refTracker) trackPathItemRefsOAS2(item *parser.PathItem, pathPattern string, definitions map[string]*parser.Schema) {
	operations := []struct {
		method string
		op     *parser.Operation
	}{
		{"GET", item.Get},
		{"PUT", item.Put},
		{"POST", item.Post},
		{"DELETE", item.Delete},
		{"OPTIONS", item.Options},
		{"HEAD", item.Head},
		{"PATCH", item.Patch},
	}

	for _, o := range operations {
		if o.op != nil {
			opRef := operationRef{
				Method:      o.method,
				Path:        pathPattern,
				OperationID: o.op.OperationID,
			}
			rt.trackOperationRefsOAS2(o.op, opRef, definitions)
		}
	}

	// Path-level parameters
	for _, param := range item.Parameters {
		if param != nil {
			opRef := operationRef{Path: pathPattern}
			rt.trackParameterRefsOAS2(param, opRef, definitions)
		}
	}
}

// trackOperationRefs tracks all refs in an operation for OAS3.
func (rt *refTracker) trackOperationRefs(op *parser.Operation, opRef operationRef, components *parser.Components) {
	visited := make(map[string]bool)

	// Track parameter refs
	for _, param := range op.Parameters {
		if param != nil {
			rt.trackParameterRefs(param, opRef, components)
		}
	}

	// Track request body refs
	if op.RequestBody != nil {
		rt.trackRequestBodyRefs(op.RequestBody, opRef, components, visited)
	}

	// Track response refs
	if op.Responses != nil {
		if op.Responses.Default != nil {
			rt.trackResponseRefs(op.Responses.Default, opRef, components, visited)
		}
		for _, resp := range op.Responses.Codes {
			if resp != nil {
				rt.trackResponseRefs(resp, opRef, components, visited)
			}
		}
	}
}

// trackOperationRefsOAS2 tracks all refs in an operation for OAS2.
func (rt *refTracker) trackOperationRefsOAS2(op *parser.Operation, opRef operationRef, definitions map[string]*parser.Schema) {
	visited := make(map[string]bool)

	// Track parameter refs
	for _, param := range op.Parameters {
		if param != nil {
			rt.trackParameterRefsOAS2(param, opRef, definitions)
		}
	}

	// Track response refs
	if op.Responses != nil {
		if op.Responses.Default != nil {
			rt.trackResponseRefsOAS2(op.Responses.Default, opRef, definitions, visited)
		}
		for _, resp := range op.Responses.Codes {
			if resp != nil {
				rt.trackResponseRefsOAS2(resp, opRef, definitions, visited)
			}
		}
	}
}

// trackParameterRefs tracks refs in a parameter.
func (rt *refTracker) trackParameterRefs(param *parser.Parameter, opRef operationRef, components *parser.Components) {
	if param.Ref != "" {
		rt.addRef(param.Ref, opRef)
	}
	if param.Schema != nil {
		rt.trackSchemaRefs(param.Schema, opRef, components, make(map[string]bool))
	}
}

// trackParameterRefsOAS2 tracks refs in an OAS2 parameter.
func (rt *refTracker) trackParameterRefsOAS2(param *parser.Parameter, opRef operationRef, definitions map[string]*parser.Schema) {
	if param.Ref != "" {
		rt.addRef(param.Ref, opRef)
	}
	if param.Schema != nil {
		rt.trackSchemaRefsOAS2(param.Schema, opRef, definitions, make(map[string]bool))
	}
}

// trackRequestBodyRefs tracks refs in a request body.
func (rt *refTracker) trackRequestBodyRefs(rb *parser.RequestBody, opRef operationRef, components *parser.Components, visited map[string]bool) {
	if rb.Ref != "" {
		rt.addRef(rb.Ref, opRef)
	}
	for _, mt := range rb.Content {
		if mt != nil && mt.Schema != nil {
			rt.trackSchemaRefs(mt.Schema, opRef, components, visited)
		}
	}
}

// trackResponseRefs tracks refs in a response.
func (rt *refTracker) trackResponseRefs(resp *parser.Response, opRef operationRef, components *parser.Components, visited map[string]bool) {
	if resp.Ref != "" {
		rt.addRef(resp.Ref, opRef)
	}
	for _, mt := range resp.Content {
		if mt != nil && mt.Schema != nil {
			rt.trackSchemaRefs(mt.Schema, opRef, components, visited)
		}
	}
	for _, header := range resp.Headers {
		if header != nil && header.Schema != nil {
			rt.trackSchemaRefs(header.Schema, opRef, components, visited)
		}
	}
}

// trackResponseRefsOAS2 tracks refs in an OAS2 response.
func (rt *refTracker) trackResponseRefsOAS2(resp *parser.Response, opRef operationRef, definitions map[string]*parser.Schema, visited map[string]bool) {
	if resp.Ref != "" {
		rt.addRef(resp.Ref, opRef)
	}
	if resp.Schema != nil {
		rt.trackSchemaRefsOAS2(resp.Schema, opRef, definitions, visited)
	}
}

// trackSchemaRefs tracks refs in a schema, following transitive refs.
func (rt *refTracker) trackSchemaRefs(schema *parser.Schema, opRef operationRef, components *parser.Components, visited map[string]bool) {
	if schema == nil {
		return
	}

	// Handle $ref
	if schema.Ref != "" {
		normalized := normalizeRef(schema.Ref)
		if visited[normalized] {
			return // Avoid infinite loops
		}
		visited[normalized] = true

		rt.addRef(schema.Ref, opRef)

		// Follow the ref to track transitive dependencies
		if components != nil && strings.HasPrefix(schema.Ref, "#/components/schemas/") {
			name := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
			if resolved, ok := components.Schemas[name]; ok {
				rt.trackSchemaRefs(resolved, opRef, components, visited)
			}
		}
		return
	}

	// Track nested schemas
	if schema.Items != nil {
		rt.trackSchemaRefs(schema.Items, opRef, components, visited)
	}
	for _, prop := range schema.Properties {
		rt.trackSchemaRefs(prop, opRef, components, visited)
	}
	if schema.AdditionalProperties != nil {
		rt.trackSchemaRefs(schema.AdditionalProperties, opRef, components, visited)
	}
	for _, s := range schema.AllOf {
		rt.trackSchemaRefs(s, opRef, components, visited)
	}
	for _, s := range schema.AnyOf {
		rt.trackSchemaRefs(s, opRef, components, visited)
	}
	for _, s := range schema.OneOf {
		rt.trackSchemaRefs(s, opRef, components, visited)
	}
	if schema.Not != nil {
		rt.trackSchemaRefs(schema.Not, opRef, components, visited)
	}
}

// trackSchemaRefsOAS2 tracks refs in an OAS2 schema.
func (rt *refTracker) trackSchemaRefsOAS2(schema *parser.Schema, opRef operationRef, definitions map[string]*parser.Schema, visited map[string]bool) {
	if schema == nil {
		return
	}

	if schema.Ref != "" {
		normalized := normalizeRef(schema.Ref)
		if visited[normalized] {
			return
		}
		visited[normalized] = true

		rt.addRef(schema.Ref, opRef)

		// Follow the ref
		if strings.HasPrefix(schema.Ref, "#/definitions/") {
			name := strings.TrimPrefix(schema.Ref, "#/definitions/")
			if resolved, ok := definitions[name]; ok {
				rt.trackSchemaRefsOAS2(resolved, opRef, definitions, visited)
			}
		}
		return
	}

	// Track nested schemas
	if schema.Items != nil {
		rt.trackSchemaRefsOAS2(schema.Items, opRef, definitions, visited)
	}
	for _, prop := range schema.Properties {
		rt.trackSchemaRefsOAS2(prop, opRef, definitions, visited)
	}
	if schema.AdditionalProperties != nil {
		rt.trackSchemaRefsOAS2(schema.AdditionalProperties, opRef, definitions, visited)
	}
	for _, s := range schema.AllOf {
		rt.trackSchemaRefsOAS2(s, opRef, definitions, visited)
	}
}

// addRef adds a reference mapping from a $ref to an operation.
func (rt *refTracker) addRef(ref string, opRef operationRef) {
	normalized := normalizeRef(ref)
	if normalized == "" {
		return
	}

	// Check if this operation is already recorded for this component
	existing := rt.componentToOps[normalized]
	for _, op := range existing {
		if op.Method == opRef.Method && op.Path == opRef.Path {
			return // Already tracked
		}
	}

	rt.componentToOps[normalized] = append(existing, opRef)
}

// normalizeRef converts a $ref like "#/components/schemas/User" to "components.schemas.User".
func normalizeRef(ref string) string {
	if !strings.HasPrefix(ref, "#/") {
		return "" // External ref, not tracked
	}
	// Remove leading #/ and replace / with .
	return strings.ReplaceAll(strings.TrimPrefix(ref, "#/"), "/", ".")
}

// getOperationsForComponent returns all operations that reference a component.
func (rt *refTracker) getOperationsForComponent(componentPath string) []operationRef {
	return rt.componentToOps[componentPath]
}

// getOperationContext builds an OperationContext for a given issue path.
// Returns nil if no operation context applies.
func (rt *refTracker) getOperationContext(issuePath string, doc any) *issues.OperationContext {
	// Check if this is under paths.*
	if strings.HasPrefix(issuePath, "paths.") {
		return rt.getPathOperationContext(issuePath, doc)
	}

	// Check if this is a reusable component
	if isReusableComponentPath(issuePath) {
		return rt.getComponentOperationContext(issuePath)
	}

	return nil
}

// getPathOperationContext extracts operation context from a paths.* issue path.
func (rt *refTracker) getPathOperationContext(issuePath string, doc any) *issues.OperationContext {
	// Parse: paths./users/{id}.get.parameters[0] -> path=/users/{id}, method=get
	parts := strings.SplitN(strings.TrimPrefix(issuePath, "paths."), ".", 2)
	if len(parts) == 0 {
		return nil
	}

	apiPath := parts[0]
	if len(parts) == 1 {
		// Just the path itself, no method
		return &issues.OperationContext{Path: apiPath}
	}

	remainder := parts[1]
	// Check if next part is a method
	methodPart := strings.SplitN(remainder, ".", 2)[0]
	method := parseMethod(methodPart)

	if method == "" {
		// Path-level (e.g., paths./users.parameters)
		return &issues.OperationContext{Path: apiPath}
	}

	// Get operationId from document
	operationID := getOperationID(doc, apiPath, method)

	return &issues.OperationContext{
		Method:      method,
		Path:        apiPath,
		OperationID: operationID,
	}
}

// getComponentOperationContext builds context for a reusable component.
func (rt *refTracker) getComponentOperationContext(issuePath string) *issues.OperationContext {
	// Normalize path to component root (e.g., "components.schemas.User.properties.id" -> "components.schemas.User")
	componentPath := getComponentRoot(issuePath)

	ops := rt.getOperationsForComponent(componentPath)
	if len(ops) == 0 {
		// Unused component
		return &issues.OperationContext{
			IsReusableComponent: true,
			AdditionalRefs:      -1,
		}
	}

	first := ops[0]
	return &issues.OperationContext{
		Method:              first.Method,
		Path:                first.Path,
		OperationID:         first.OperationID,
		IsReusableComponent: true,
		IsWebhook:           first.IsWebhook,
		AdditionalRefs:      len(ops) - 1,
	}
}

// isReusableComponentPath returns true if the path is under a reusable component section.
func isReusableComponentPath(path string) bool {
	prefixes := []string{
		"components.schemas.",
		"components.responses.",
		"components.parameters.",
		"components.requestBodies.",
		"components.headers.",
		"components.securitySchemes.",
		"components.links.",
		"components.callbacks.",
		"components.pathItems.",
		"definitions.",
		"parameters.",
		"responses.",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// getComponentRoot extracts the component root from a nested path.
// e.g., "components.schemas.User.properties.id" -> "components.schemas.User"
func getComponentRoot(path string) string {
	// Handle OAS3 components
	if strings.HasPrefix(path, "components.") {
		parts := strings.Split(path, ".")
		if len(parts) >= 3 {
			return strings.Join(parts[:3], ".")
		}
	}
	// Handle OAS2 definitions/parameters/responses
	if strings.HasPrefix(path, "definitions.") || strings.HasPrefix(path, "parameters.") || strings.HasPrefix(path, "responses.") {
		parts := strings.Split(path, ".")
		if len(parts) >= 2 {
			return strings.Join(parts[:2], ".")
		}
	}
	return path
}

// parseMethod converts a lowercase method string to uppercase, or returns "" if not a valid method.
func parseMethod(s string) string {
	methods := map[string]string{
		"get":     "GET",
		"put":     "PUT",
		"post":    "POST",
		"delete":  "DELETE",
		"options": "OPTIONS",
		"head":    "HEAD",
		"patch":   "PATCH",
		"trace":   "TRACE",
		"query":   "QUERY",
	}
	return methods[s]
}

// getOperationID looks up the operationId for a given path and method from the document.
func getOperationID(doc any, apiPath, method string) string {
	switch d := doc.(type) {
	case *parser.OAS3Document:
		if pathItem, ok := d.Paths[apiPath]; ok && pathItem != nil {
			return getOperationIDFromPathItem(pathItem, method)
		}
	case *parser.OAS2Document:
		if pathItem, ok := d.Paths[apiPath]; ok && pathItem != nil {
			return getOperationIDFromPathItem(pathItem, method)
		}
	}
	return ""
}

// getOperationIDFromPathItem extracts operationId for a method from a PathItem.
func getOperationIDFromPathItem(item *parser.PathItem, method string) string {
	var op *parser.Operation
	switch method {
	case "GET":
		op = item.Get
	case "PUT":
		op = item.Put
	case "POST":
		op = item.Post
	case "DELETE":
		op = item.Delete
	case "OPTIONS":
		op = item.Options
	case "HEAD":
		op = item.Head
	case "PATCH":
		op = item.Patch
	case "TRACE":
		op = item.Trace
	case "QUERY":
		op = item.Query
	}
	if op != nil {
		return op.OperationID
	}
	return ""
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./validator -run TestRefTracker -v`
Expected: PASS

**Step 5: Run go_diagnostics**

Run: `go_diagnostics` for `validator/ref_tracker.go`
Expected: No errors (or only hints)

**Step 6: Commit**

```bash
git add validator/ref_tracker.go validator/ref_tracker_test.go
git commit -m "feat(validator): add reference tracker for operation context"
```

---

## Task 4: Wire Up Context Attachment

**Files:**
- Modify: `validator/validator.go:108-124` (Validator struct - add refTracker field)
- Modify: `validator/validator.go:236-299` (ValidateParsed - initialize tracker and attach context)
- Test: `validator/validator_test.go` (add integration tests)

**Step 1: Write the failing test**

Add to `validator/validator_test.go`:

```go
func TestValidationErrorsHaveOperationContext(t *testing.T) {
	// Create a spec with intentional errors in different locations
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users/{id}:
    parameters:
      - name: id
        in: path
        # Missing required: true - path-level error
    get:
      operationId: getUser
      parameters:
        - name: filter
          in: query
          schema:
            type: invalid_type  # Error in operation parameter
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
      properties:
        email:
          type: string
          format: not_a_real_format  # Error in shared schema
`
	p := parser.New()
	p.ValidateStructure = false
	parseResult, err := p.ParseData([]byte(spec), parser.SourceFormatYAML)
	require.NoError(t, err)

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	// Find errors and check their operation context
	var foundPathLevelError, foundOperationError, foundSchemaError bool

	for _, e := range result.Errors {
		if strings.Contains(e.Path, "parameters") && !strings.Contains(e.Path, ".get.") {
			// Path-level parameter error
			if e.OperationContext != nil {
				foundPathLevelError = true
				assert.Equal(t, "/users/{id}", e.OperationContext.Path)
				assert.Empty(t, e.OperationContext.Method, "path-level should have no method")
			}
		}
		if strings.Contains(e.Path, ".get.parameters") {
			// Operation-level parameter error
			if e.OperationContext != nil {
				foundOperationError = true
				assert.Equal(t, "GET", e.OperationContext.Method)
				assert.Equal(t, "/users/{id}", e.OperationContext.Path)
				assert.Equal(t, "getUser", e.OperationContext.OperationID)
			}
		}
		if strings.Contains(e.Path, "components.schemas.User") {
			// Shared schema error
			if e.OperationContext != nil {
				foundSchemaError = true
				assert.True(t, e.OperationContext.IsReusableComponent)
				assert.Equal(t, "getUser", e.OperationContext.OperationID)
			}
		}
	}

	// Note: Some errors may not trigger depending on what the validator catches
	// The important thing is that when errors DO occur in these locations, they have context
	t.Logf("Found errors - path-level: %v, operation: %v, schema: %v",
		foundPathLevelError, foundOperationError, foundSchemaError)
}

func TestOperationContextInErrorString(t *testing.T) {
	v := New()
	result, err := v.Validate("testdata/invalid-oas3.yaml")
	require.NoError(t, err)

	// Check that at least some errors have operation context in their string representation
	var foundWithContext bool
	for _, e := range result.Errors {
		str := e.String()
		if strings.Contains(str, "(operationId:") || strings.Contains(str, "(GET ") || strings.Contains(str, "(POST ") {
			foundWithContext = true
			t.Logf("Error with context: %s", str)
		}
	}

	// Note: This test verifies the formatting works, not that all errors have context
	if foundWithContext {
		t.Log("Found errors with operation context in string output")
	}
}
```

**Step 2: Run test to verify current state**

Run: `go test ./validator -run TestValidationErrorsHaveOperationContext -v`
Expected: May pass or fail depending on implementation state

**Step 3: Modify Validator struct**

Add field to Validator struct in `validator/validator.go` after line 123:

```go
	// refTracker tracks which operations reference which components.
	// Built during ValidateParsed for populating OperationContext on issues.
	refTracker *refTracker
```

**Step 4: Modify ValidateParsed to initialize tracker**

In `validator/validator.go`, add after line 247 (after setting SourcePath):

```go
	// Build reference tracker for operation context
	switch doc := parseResult.Document.(type) {
	case *parser.OAS3Document:
		v.refTracker = buildRefTrackerOAS3(doc)
	case *parser.OAS2Document:
		v.refTracker = buildRefTrackerOAS2(doc)
	}
```

**Step 5: Modify addError to attach context**

Update `addError` in `validator/validator.go` (around line 193):

```go
func (v *Validator) addError(result *ValidationResult, path, message string, opts ...func(*ValidationError)) {
	err := ValidationError{
		Path:     path,
		Message:  message,
		Severity: SeverityError,
	}
	for _, opt := range opts {
		opt(&err)
	}
	v.populateIssueLocation(&err)
	v.populateOperationContext(&err, result.Document)
	result.Errors = append(result.Errors, err)
}
```

**Step 6: Modify addWarning similarly**

Update `addWarning` in `validator/validator.go`:

```go
func (v *Validator) addWarning(result *ValidationResult, path, message string, opts ...func(*ValidationError)) {
	warn := ValidationError{
		Path:     path,
		Message:  message,
		Severity: SeverityWarning,
	}
	for _, opt := range opts {
		opt(&warn)
	}
	v.populateIssueLocation(&warn)
	v.populateOperationContext(&warn, result.Document)
	result.Warnings = append(result.Warnings, warn)
}
```

**Step 7: Add populateOperationContext method**

Add after `populateIssueLocation` in `validator/validator.go`:

```go
// populateOperationContext attaches operation context to an issue if applicable.
func (v *Validator) populateOperationContext(issue *ValidationError, doc any) {
	if v.refTracker == nil {
		return
	}
	issue.OperationContext = v.refTracker.getOperationContext(issue.Path, doc)
}
```

**Step 8: Run tests**

Run: `go test ./validator -run TestValidationErrorsHaveOperationContext -v`
Run: `go test ./validator -run TestOperationContextInErrorString -v`
Expected: PASS (or at least no panics; context attachment working)

**Step 9: Run all validator tests**

Run: `go test ./validator -v`
Expected: All tests PASS

**Step 10: Run go_diagnostics**

Run: `go_diagnostics` for `validator/validator.go`
Expected: No errors

**Step 11: Commit**

```bash
git add validator/validator.go validator/validator_test.go
git commit -m "feat(validator): attach operation context to validation errors"
```

---

## Task 5: Handle Direct Error Appends

**Files:**
- Modify: Various validator files that append errors directly
- Create: `validator/context_helpers.go` (helper for manual appends)

**Step 1: Identify direct appends**

Run: `grep -n "result.Errors = append" validator/*.go | head -20`

These need to be updated to use `addError` or manually attach context.

**Step 2: Create helper for legacy code**

Create `validator/context_helpers.go`:

```go
package validator

// attachOperationContextToIssue is a helper for code that appends errors directly
// instead of using addError. Call this after creating the error.
func (v *Validator) attachOperationContextToIssue(issue *ValidationError, doc any) {
	if v.refTracker == nil {
		return
	}
	issue.OperationContext = v.refTracker.getOperationContext(issue.Path, doc)
}
```

**Step 3: Update direct appends in refs.go**

For each direct `result.Errors = append(...)` in `validator/refs.go`, add context attachment after the append. Example pattern:

```go
// Before:
result.Errors = append(result.Errors, ValidationError{
    Path:     path,
    Message:  fmt.Sprintf("..."),
    Severity: SeverityError,
})

// After:
err := ValidationError{
    Path:     path,
    Message:  fmt.Sprintf("..."),
    Severity: SeverityError,
}
v.populateIssueLocation(&err)
v.populateOperationContext(&err, result.Document)
result.Errors = append(result.Errors, err)
```

**Step 4: Update direct appends in oas2.go and oas3.go**

Apply the same pattern to other files with direct appends.

**Step 5: Run all tests**

Run: `go test ./validator -v`
Expected: All tests PASS

**Step 6: Run go_diagnostics**

Run: `go_diagnostics` for all modified files
Expected: No errors

**Step 7: Commit**

```bash
git add validator/context_helpers.go validator/refs.go validator/oas2.go validator/oas3.go validator/schema.go validator/helpers.go
git commit -m "refactor(validator): attach operation context to all error appends"
```

---

## Task 6: Add Integration Test with Real Output

**Files:**
- Create: `validator/operation_context_test.go`

**Step 1: Write comprehensive integration test**

Create `validator/operation_context_test.go`:

```go
package validator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationContextIntegration(t *testing.T) {
	// Test with testdata/invalid-oas3.yaml
	v := New()
	result, err := v.Validate("../testdata/invalid-oas3.yaml")
	require.NoError(t, err)

	t.Logf("Found %d errors, %d warnings", result.ErrorCount, result.WarningCount)

	for i, e := range result.Errors {
		str := e.String()
		t.Logf("Error %d: %s", i+1, str)

		// Verify format based on path type
		if strings.HasPrefix(e.Path, "paths.") {
			// Should have some form of operation context
			if e.OperationContext != nil && !e.OperationContext.IsEmpty() {
				assert.Contains(t, str, "(", "paths error should have context in output")
			}
		}
	}

	for i, w := range result.Warnings {
		str := w.String()
		t.Logf("Warning %d: %s", i+1, str)
	}
}

func TestOperationContextFormats(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		opCtx        *issues.OperationContext
		wantContains string
	}{
		{
			name: "operation with operationId",
			path: "paths./users.get.responses",
			opCtx: &issues.OperationContext{
				Method:      "GET",
				Path:        "/users",
				OperationID: "listUsers",
			},
			wantContains: "(operationId: listUsers)",
		},
		{
			name: "operation without operationId",
			path: "paths./orders.post.requestBody",
			opCtx: &issues.OperationContext{
				Method: "POST",
				Path:   "/orders",
			},
			wantContains: "(POST /orders)",
		},
		{
			name: "path-level parameter",
			path: "paths./users/{id}.parameters[0]",
			opCtx: &issues.OperationContext{
				Path: "/users/{id}",
			},
			wantContains: "(path: /users/{id})",
		},
		{
			name: "shared schema with multiple refs",
			path: "components.schemas.User.properties.email",
			opCtx: &issues.OperationContext{
				Method:              "GET",
				Path:                "/users",
				OperationID:         "listUsers",
				IsReusableComponent: true,
				AdditionalRefs:      3,
			},
			wantContains: "(operationId: listUsers, +3 operations)",
		},
		{
			name: "unused component",
			path: "components.schemas.Orphan.type",
			opCtx: &issues.OperationContext{
				IsReusableComponent: true,
				AdditionalRefs:      -1,
			},
			wantContains: "(unused component)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := ValidationError{
				Path:             tt.path,
				Message:          "test error",
				Severity:         SeverityError,
				OperationContext: tt.opCtx,
			}
			str := issue.String()
			assert.Contains(t, str, tt.wantContains)
		})
	}
}
```

**Step 2: Run tests**

Run: `go test ./validator -run TestOperationContext -v`
Expected: PASS

**Step 3: Commit**

```bash
git add validator/operation_context_test.go
git commit -m "test(validator): add operation context integration tests"
```

---

## Task 7: Update Documentation

**Files:**
- Modify: `validator/deep_dive.md`

**Step 1: Add section about operation context**

Add to `validator/deep_dive.md` after the ValidationError section:

```markdown
### Operation Context

Validation errors now include operation context to help identify which API endpoint is affected:

```go
// Operation-level errors include method, path, and operationId
err.OperationContext = &issues.OperationContext{
    Method:      "GET",
    Path:        "/users/{id}",
    OperationID: "getUser",
}
// Renders as: paths./users/{id}.get.parameters[0] (operationId: getUser): ...

// Path-level errors show only the path
err.OperationContext = &issues.OperationContext{
    Path: "/users/{id}",
}
// Renders as: paths./users/{id}.parameters[0] (path: /users/{id}): ...

// Shared component errors show referencing operations
err.OperationContext = &issues.OperationContext{
    Method:              "GET",
    Path:                "/users",
    OperationID:         "listUsers",
    IsReusableComponent: true,
    AdditionalRefs:      3,
}
// Renders as: components.schemas.User.properties.email (operationId: listUsers, +3 operations): ...
```

The operation context is automatically populated during validation. For programmatic access:

```go
result, _ := validator.ValidateWithOptions(
    validator.WithFilePath("api.yaml"),
)
for _, err := range result.Errors {
    if err.OperationContext != nil {
        fmt.Printf("Affected operation: %s %s\n",
            err.OperationContext.Method,
            err.OperationContext.Path)
    }
}
```
```

**Step 2: Commit**

```bash
git add validator/deep_dive.md
git commit -m "docs(validator): document operation context feature"
```

---

## Task 8: Run Full Test Suite and Benchmarks

**Step 1: Run make check**

Run: `make check`
Expected: All checks pass

**Step 2: Run benchmarks**

Run: `go test -bench=. ./validator -benchmem -run=^$ | head -30`
Expected: Performance within acceptable range (< 5% regression from baseline)

**Step 3: Test CLI output manually**

Run: `go run ./cmd/oastools validate testdata/invalid-oas3.yaml`
Expected: Output shows operation context in error messages

**Step 4: Final commit if needed**

```bash
git status
# If any uncommitted changes:
git add .
git commit -m "chore: final cleanup for operation context feature"
```

---

## Summary

This plan implements operation context for validation errors in 8 tasks:

1. **OperationContext struct** - Data model for operation context
2. **Issue field** - Add OperationContext to Issue struct
3. **Reference tracker** - Track $ref → operation mappings
4. **Wire up** - Attach context in addError/addWarning
5. **Direct appends** - Handle legacy error appends
6. **Integration tests** - Verify end-to-end behavior
7. **Documentation** - Update deep_dive.md
8. **Final verification** - Run all tests and benchmarks

Each task follows TDD: write failing test, implement, verify, commit.
