package walker

import "github.com/erraggy/oastools/parser"

// ParentInfo provides information about a parent node in the traversal.
// This enables handlers to access ancestor nodes for context-aware processing.
type ParentInfo struct {
	// Node is the parent node (*parser.Schema, *parser.Operation, etc.)
	Node any

	// JSONPath is the JSON path to this parent node
	JSONPath string

	// Parent is the grandparent, enabling ancestor chain traversal.
	// nil for the root-level parent.
	Parent *ParentInfo
}

// ParentSchema returns the nearest ancestor that is a Schema, if any.
// This is useful for detecting when a schema is nested within another schema
// (e.g., a property within an object schema).
func (wc *WalkContext) ParentSchema() (*parser.Schema, bool) {
	for p := wc.Parent; p != nil; p = p.Parent {
		if s, ok := p.Node.(*parser.Schema); ok {
			return s, true
		}
	}
	return nil, false
}

// ParentOperation returns the nearest ancestor that is an Operation, if any.
// This is useful for determining which operation context a nested node belongs to.
func (wc *WalkContext) ParentOperation() (*parser.Operation, bool) {
	for p := wc.Parent; p != nil; p = p.Parent {
		if op, ok := p.Node.(*parser.Operation); ok {
			return op, true
		}
	}
	return nil, false
}

// ParentPathItem returns the nearest ancestor that is a PathItem, if any.
// This is useful for accessing path-level configuration from nested nodes.
func (wc *WalkContext) ParentPathItem() (*parser.PathItem, bool) {
	for p := wc.Parent; p != nil; p = p.Parent {
		if pi, ok := p.Node.(*parser.PathItem); ok {
			return pi, true
		}
	}
	return nil, false
}

// ParentResponse returns the nearest ancestor that is a Response, if any.
// This is useful for determining which response a schema or header belongs to.
func (wc *WalkContext) ParentResponse() (*parser.Response, bool) {
	for p := wc.Parent; p != nil; p = p.Parent {
		if r, ok := p.Node.(*parser.Response); ok {
			return r, true
		}
	}
	return nil, false
}

// ParentRequestBody returns the nearest ancestor that is a RequestBody, if any.
// This is useful for determining if a schema is part of a request body.
func (wc *WalkContext) ParentRequestBody() (*parser.RequestBody, bool) {
	for p := wc.Parent; p != nil; p = p.Parent {
		if rb, ok := p.Node.(*parser.RequestBody); ok {
			return rb, true
		}
	}
	return nil, false
}

// Ancestors returns all ancestors from immediate parent to root.
// The first element is the immediate parent, the last is the root-level ancestor.
// Returns nil if parent tracking is not enabled or there are no ancestors.
func (wc *WalkContext) Ancestors() []*ParentInfo {
	var ancestors []*ParentInfo
	for p := wc.Parent; p != nil; p = p.Parent {
		ancestors = append(ancestors, p)
	}
	return ancestors
}

// Depth returns the number of ancestors (nesting depth).
// Returns 0 if at root level or parent tracking is not enabled.
func (wc *WalkContext) Depth() int {
	depth := 0
	for p := wc.Parent; p != nil; p = p.Parent {
		depth++
	}
	return depth
}
