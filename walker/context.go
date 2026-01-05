package walker

import (
	"context"
	"sync"
)

// contextPool provides reusable WalkContext instances to reduce allocations.
var contextPool = sync.Pool{
	New: func() any { return &WalkContext{} },
}

// WalkContext provides contextual information about the current node being visited.
// It follows the http.Request pattern for context access.
//
// IMPORTANT: WalkContext instances are reused via sync.Pool. Handlers must not
// retain references to WalkContext after returning from the handler function.
// If you need to preserve context information, copy the relevant fields to your
// own data structures before the handler returns.
type WalkContext struct {
	// JSONPath is the full JSON path to the current node.
	// Always populated. Example: "$.paths['/pets'].get.responses['200']"
	JSONPath string

	// PathTemplate is the URL path template when walking within $.paths scope.
	// Empty when not in paths scope. Example: "/pets/{petId}"
	PathTemplate string

	// Method is the HTTP method when walking within an operation scope.
	// Empty when not in operation scope. Example: "get", "post"
	Method string

	// StatusCode is the HTTP status code when walking within a response scope.
	// Empty when not in response scope. Example: "200", "default"
	StatusCode string

	// Name is the map key for named items like headers, schemas, examples, etc.
	// Empty for array items or root-level objects. Example: "Pet", "X-Rate-Limit"
	Name string

	// IsComponent is true when the current node is within the components section
	// (OAS 3.x) or definitions/parameters/responses at document root (OAS 2.x).
	IsComponent bool

	// CurrentRef is populated when the current node has a $ref.
	// Only available when WithRefTracking() is enabled.
	CurrentRef *RefInfo

	// Parent provides access to the parent node, or nil at root.
	// Only populated when WithParentTracking() option is used.
	// Use helper methods like ParentSchema(), ParentOperation(), etc.
	Parent *ParentInfo

	ctx context.Context
}

// Context returns the context.Context for cancellation and deadline propagation.
// Returns context.Background() if no context was set.
func (wc *WalkContext) Context() context.Context {
	if wc.ctx == nil {
		return context.Background()
	}
	return wc.ctx
}

// WithContext returns a shallow copy of WalkContext with the new context.
func (wc *WalkContext) WithContext(ctx context.Context) *WalkContext {
	wc2 := *wc
	wc2.ctx = ctx
	return &wc2
}

// InPathsScope returns true if currently walking within $.paths.
func (wc *WalkContext) InPathsScope() bool {
	return wc.PathTemplate != ""
}

// InOperationScope returns true if currently walking within an operation.
func (wc *WalkContext) InOperationScope() bool {
	return wc.Method != ""
}

// InResponseScope returns true if currently walking within a response.
func (wc *WalkContext) InResponseScope() bool {
	return wc.StatusCode != ""
}

// walkState tracks context as we descend through the document.
// This is internal to the walker and used to build WalkContext instances.
type walkState struct {
	pathTemplate string
	method       string
	statusCode   string
	name         string
	isComponent  bool
	ctx          context.Context

	// Parent tracking state (only used when trackParent is enabled)
	// Uses a pointer to slice to ensure all cloned states share the same stack.
	walker      *Walker
	parentStack *[]*ParentInfo
}

// buildContext creates a WalkContext from the current walk state.
// The returned WalkContext is obtained from a sync.Pool and must be
// released with releaseContext after use.
func (s *walkState) buildContext(jsonPath string) *WalkContext {
	wc := contextPool.Get().(*WalkContext)
	wc.JSONPath = jsonPath
	wc.PathTemplate = s.pathTemplate
	wc.Method = s.method
	wc.StatusCode = s.statusCode
	wc.Name = s.name
	wc.IsComponent = s.isComponent
	wc.Parent = s.currentParent()
	wc.ctx = s.ctx
	return wc
}

// releaseContext returns a WalkContext to the pool for reuse.
// The WalkContext must not be used after this call.
func releaseContext(wc *WalkContext) {
	*wc = WalkContext{} // Clear all fields to prevent data leakage
	contextPool.Put(wc)
}

// pushParent adds a parent to the stack. No-op if parent tracking is disabled.
func (s *walkState) pushParent(node any, jsonPath string) {
	if s.walker == nil || !s.walker.trackParent || s.parentStack == nil {
		return
	}
	var parent *ParentInfo
	if len(*s.parentStack) > 0 {
		parent = (*s.parentStack)[len(*s.parentStack)-1]
	}
	*s.parentStack = append(*s.parentStack, &ParentInfo{
		Node:     node,
		JSONPath: jsonPath,
		Parent:   parent,
	})
}

// popParent removes the most recent parent from the stack. No-op if parent tracking is disabled.
func (s *walkState) popParent() {
	if s.walker == nil || !s.walker.trackParent || s.parentStack == nil || len(*s.parentStack) == 0 {
		return
	}
	*s.parentStack = (*s.parentStack)[:len(*s.parentStack)-1]
}

// currentParent returns the current parent, or nil if none.
func (s *walkState) currentParent() *ParentInfo {
	if s.parentStack == nil || len(*s.parentStack) == 0 {
		return nil
	}
	return (*s.parentStack)[len(*s.parentStack)-1]
}

// clone creates a copy of the walk state for child traversal.
// The parentStack is shared (not deep copied) since children push/pop on the same stack.
func (s *walkState) clone() *walkState {
	return &walkState{
		pathTemplate: s.pathTemplate,
		method:       s.method,
		statusCode:   s.statusCode,
		name:         s.name,
		isComponent:  s.isComponent,
		ctx:          s.ctx,
		walker:       s.walker,
		parentStack:  s.parentStack,
	}
}
