package walker

import "context"

// WalkContext provides contextual information about the current node being visited.
// It follows the http.Request pattern for context access.
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
}

// buildContext creates a WalkContext from the current walk state.
func (s *walkState) buildContext(jsonPath string) *WalkContext {
	return &WalkContext{
		JSONPath:     jsonPath,
		PathTemplate: s.pathTemplate,
		Method:       s.method,
		StatusCode:   s.statusCode,
		Name:         s.name,
		IsComponent:  s.isComponent,
		ctx:          s.ctx,
	}
}

// clone creates a copy of the walk state for child traversal.
func (s *walkState) clone() *walkState {
	return &walkState{
		pathTemplate: s.pathTemplate,
		method:       s.method,
		statusCode:   s.statusCode,
		name:         s.name,
		isComponent:  s.isComponent,
		ctx:          s.ctx,
	}
}
