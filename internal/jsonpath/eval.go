package jsonpath

import (
	"fmt"
	"log/slog"
)

// jsonpathLogger is used for warning-level log output (e.g., depth limit
// truncation). Tests swap it with a discard logger to suppress expected noise.
var jsonpathLogger = slog.Default()

// Get evaluates the path against the document and returns all matching values.
//
// The document should be a map[string]any or []any structure (typically from
// JSON/YAML unmarshaling). Returns an empty slice if no matches are found.
func (p *Path) Get(doc any) []any {
	if len(p.segments) == 0 {
		return nil
	}

	// Start with root
	current := []any{doc}

	// Apply each segment after root
	for i := 1; i < len(p.segments); i++ {
		seg := p.segments[i]
		current = applySegment(current, seg)
		if len(current) == 0 {
			return nil
		}
	}

	return current
}

// Set sets the value at all matching locations in the document.
//
// Returns an error if no matches are found or if the path cannot be traversed.
// The document is modified in place.
func (p *Path) Set(doc any, value any) error {
	if len(p.segments) < 2 {
		return fmt.Errorf("jsonpath: cannot set on root path")
	}

	// Get the parent nodes and the final key
	parentPath := &Path{
		raw:      p.raw,
		segments: p.segments[:len(p.segments)-1],
	}

	parents := parentPath.Get(doc)
	if len(parents) == 0 {
		return fmt.Errorf("jsonpath: no matches for parent path")
	}

	lastSeg := p.segments[len(p.segments)-1]

	for _, parent := range parents {
		if err := setInParent(parent, lastSeg, value); err != nil {
			return err
		}
	}

	return nil
}

// Remove removes all matching nodes from the document.
//
// Returns the modified document. For maps, matching keys are deleted.
// For arrays, matching elements are spliced out (not set to nil).
// Returns the original document unmodified (not an error) if the path matches
// no nodes — callers that need to distinguish no-match from success should
// pre-check with Get.
func (p *Path) Remove(doc any) (any, error) {
	if len(p.segments) < 2 {
		return nil, fmt.Errorf("jsonpath: cannot remove root")
	}

	lastSeg := p.segments[len(p.segments)-1]
	parentPath := &Path{
		raw:      p.raw,
		segments: p.segments[:len(p.segments)-1],
	}

	// When the parent is the root document, apply removal directly.
	if len(parentPath.segments) <= 1 {
		return removeFromParent(doc, lastSeg), nil
	}

	// For deeper paths, go one level further to the grandparent. This lets us
	// update the grandparent's reference to the parent slice when splicing,
	// since a new slice header cannot be reflected through a plain any parameter.
	grandParentPath := &Path{
		raw:      p.raw,
		segments: p.segments[:len(p.segments)-2],
	}
	parentSeg := p.segments[len(p.segments)-2]

	grandParents := grandParentPath.Get(doc)
	if len(grandParents) == 0 {
		return doc, nil
	}

	for _, gp := range grandParents {
		modifyInParent(gp, parentSeg, func(v any) any {
			return removeFromParent(v, lastSeg)
		})
	}

	return doc, nil
}

// Modify applies a transformation function to all matching nodes.
//
// The function receives each matched value and should return the new value.
// The document is modified in place. For the root path ("$"), the document
// must be a map[string]any; fn is expected to mutate it in place (e.g.
// mergeDeep). Root replacement via fn's return value is not supported since
// the caller's variable cannot be reassigned.
func (p *Path) Modify(doc any, fn func(any) any) error {
	if len(p.segments) < 2 {
		if _, ok := doc.(map[string]any); !ok {
			return fmt.Errorf("jsonpath: root path Modify requires a map document; got %T", doc)
		}
		// fn is expected to mutate the map in place; return value is ignored.
		fn(doc)
		return nil
	}

	// Get the parent nodes and the final segment
	parentPath := &Path{
		raw:      p.raw,
		segments: p.segments[:len(p.segments)-1],
	}

	parents := parentPath.Get(doc)
	if len(parents) == 0 {
		return nil // No matches - nothing to modify
	}

	lastSeg := p.segments[len(p.segments)-1]

	for _, parent := range parents {
		modifyInParent(parent, lastSeg, fn)
	}

	return nil
}

// applySegment applies a segment to a list of current nodes and returns the results.
func applySegment(current []any, seg Segment) []any {
	var results []any

	for _, node := range current {
		switch s := seg.(type) {
		case ChildSegment:
			if m, ok := node.(map[string]any); ok {
				if val, exists := m[s.Key]; exists {
					results = append(results, val)
				}
			}

		case WildcardSegment:
			switch v := node.(type) {
			case map[string]any:
				for _, val := range v {
					results = append(results, val)
				}
			case []any:
				results = append(results, v...)
			}

		case IndexSegment:
			if arr, ok := node.([]any); ok {
				idx := s.Index
				if idx < 0 {
					idx = len(arr) + idx // Negative indexing
				}
				if idx >= 0 && idx < len(arr) {
					results = append(results, arr[idx])
				}
			}

		case FilterSegment:
			// Filter selector iterates into a collection and selects matching children.
			// For arrays: iterate elements, include those matching filter
			// For maps: iterate values, include those matching filter
			switch v := node.(type) {
			case []any:
				for _, elem := range v {
					if evalFilter(elem, s.Expr) {
						results = append(results, elem)
					}
				}
			case map[string]any:
				for _, val := range v {
					if evalFilter(val, s.Expr) {
						results = append(results, val)
					}
				}
			}

		case RecursiveSegment:
			// Recursive descent searches all descendants
			results = append(results, recursiveDescend(node, s.Child, 0)...)
		}
	}

	return results
}

// maxRecursionDepth caps how deep recursive descent will traverse to prevent
// stack overflow on pathologically nested structures.
const maxRecursionDepth = 500

// recursiveDescend finds all nodes matching the child selector at any depth.
func recursiveDescend(node any, child Segment, depth int) []any {
	if depth > maxRecursionDepth {
		jsonpathLogger.Warn("jsonpath recursive descent truncated at depth limit",
			"depth", depth,
			"maxDepth", maxRecursionDepth)
		return nil
	}

	var results []any

	// If child is nil, collect all descendants
	if child == nil {
		collectAllDescendants(node, &results, depth)
		return results
	}

	// Apply child selector at this level first
	childResults := applySegment([]any{node}, child)
	results = append(results, childResults...)

	// Recurse into children
	switch v := node.(type) {
	case map[string]any:
		for _, val := range v {
			results = append(results, recursiveDescend(val, child, depth+1)...)
		}
	case []any:
		for _, elem := range v {
			results = append(results, recursiveDescend(elem, child, depth+1)...)
		}
	}

	return results
}

// collectAllDescendants collects all nodes in the tree (for bare ..)
func collectAllDescendants(node any, results *[]any, depth int) {
	if depth > maxRecursionDepth {
		jsonpathLogger.Warn("jsonpath descendant collection truncated at depth limit",
			"depth", depth,
			"maxDepth", maxRecursionDepth)
		return
	}

	switch v := node.(type) {
	case map[string]any:
		for _, val := range v {
			*results = append(*results, val)
			collectAllDescendants(val, results, depth+1)
		}
	case []any:
		for _, elem := range v {
			*results = append(*results, elem)
			collectAllDescendants(elem, results, depth+1)
		}
	}
}

// setInParent sets a value in the parent at the location specified by the segment.
func setInParent(parent any, seg Segment, value any) error {
	switch s := seg.(type) {
	case ChildSegment:
		if m, ok := parent.(map[string]any); ok {
			m[s.Key] = value
			return nil
		}
		return fmt.Errorf("jsonpath: cannot set child on non-object")

	case IndexSegment:
		if arr, ok := parent.([]any); ok {
			idx := s.Index
			if idx < 0 {
				idx = len(arr) + idx
			}
			if idx >= 0 && idx < len(arr) {
				arr[idx] = value
				return nil
			}
			return fmt.Errorf("jsonpath: index %d out of bounds", s.Index)
		}
		return fmt.Errorf("jsonpath: cannot set index on non-array")

	case WildcardSegment:
		switch v := parent.(type) {
		case map[string]any:
			for key := range v {
				v[key] = value
			}
			return nil
		case []any:
			for i := range v {
				v[i] = value
			}
			return nil
		}
		return fmt.Errorf("jsonpath: cannot set wildcard on non-collection")

	case FilterSegment:
		switch v := parent.(type) {
		case map[string]any:
			for key, val := range v {
				if evalFilter(val, s.Expr) {
					v[key] = value
				}
			}
			return nil
		case []any:
			for i, elem := range v {
				if evalFilter(elem, s.Expr) {
					v[i] = value
				}
			}
			return nil
		}
		return fmt.Errorf("jsonpath: cannot apply filter on non-collection")

	default:
		return fmt.Errorf("jsonpath: unsupported segment type for set")
	}
}

// removeFromParent removes nodes from the parent at the location specified by the
// segment and returns the (possibly new) parent value. Callers must use the
// return value when the parent is a slice, since splicing produces a new header.
func removeFromParent(parent any, seg Segment) any {
	return removeFromParentAt(parent, seg, 0)
}

func removeFromParentAt(parent any, seg Segment, depth int) any {
	switch s := seg.(type) {
	case ChildSegment:
		if m, ok := parent.(map[string]any); ok {
			delete(m, s.Key)
		}

	case IndexSegment:
		if arr, ok := parent.([]any); ok {
			idx := s.Index
			if idx < 0 {
				idx = len(arr) + idx
			}
			if idx >= 0 && idx < len(arr) {
				return append(arr[:idx:idx], arr[idx+1:]...)
			}
		}

	case WildcardSegment:
		switch v := parent.(type) {
		case map[string]any:
			for key := range v {
				delete(v, key)
			}
		case []any:
			return v[:0]
		}

	case FilterSegment:
		switch v := parent.(type) {
		case map[string]any:
			for key, val := range v {
				if evalFilter(val, s.Expr) {
					delete(v, key)
				}
			}
		case []any:
			result := v[:0]
			for _, elem := range v {
				if !evalFilter(elem, s.Expr) {
					result = append(result, elem)
				}
			}
			return result
		}

	case RecursiveSegment:
		if s.Child != nil {
			// Remove child at this level, then recurse into all descendants.
			parent = removeFromParentAt(parent, s.Child, depth)
			if depth > maxRecursionDepth {
				jsonpathLogger.Warn("jsonpath recursive remove truncated at depth limit",
					"depth", depth, "maxDepth", maxRecursionDepth)
				return parent
			}
			switch v := parent.(type) {
			case map[string]any:
				for key, val := range v {
					v[key] = removeFromParentAt(val, seg, depth+1)
				}
			case []any:
				for i, elem := range v {
					v[i] = removeFromParentAt(elem, seg, depth+1)
				}
			}
		}
	}

	return parent
}

// modifyInParent applies a transformation function to matching nodes in the parent.
func modifyInParent(parent any, seg Segment, fn func(any) any) {
	modifyInParentAt(parent, seg, fn, 0)
}

func modifyInParentAt(parent any, seg Segment, fn func(any) any, depth int) {
	switch s := seg.(type) {
	case ChildSegment:
		if m, ok := parent.(map[string]any); ok {
			if val, exists := m[s.Key]; exists {
				m[s.Key] = fn(val)
			}
		}

	case IndexSegment:
		if arr, ok := parent.([]any); ok {
			idx := s.Index
			if idx < 0 {
				idx = len(arr) + idx
			}
			if idx >= 0 && idx < len(arr) {
				arr[idx] = fn(arr[idx])
			}
		}

	case WildcardSegment:
		switch v := parent.(type) {
		case map[string]any:
			for key, val := range v {
				v[key] = fn(val)
			}
		case []any:
			for i, elem := range v {
				v[i] = fn(elem)
			}
		}

	case FilterSegment:
		switch v := parent.(type) {
		case map[string]any:
			for key, val := range v {
				if evalFilter(val, s.Expr) {
					v[key] = fn(val)
				}
			}
		case []any:
			for i, elem := range v {
				if evalFilter(elem, s.Expr) {
					v[i] = fn(elem)
				}
			}
		}

	case RecursiveSegment:
		if s.Child != nil {
			// Apply transform at this level, then recurse into all descendants.
			modifyInParentAt(parent, s.Child, fn, depth)
			if depth > maxRecursionDepth {
				jsonpathLogger.Warn("jsonpath recursive modify truncated at depth limit",
					"depth", depth, "maxDepth", maxRecursionDepth)
				return
			}
			switch v := parent.(type) {
			case map[string]any:
				for _, val := range v {
					modifyInParentAt(val, seg, fn, depth+1)
				}
			case []any:
				for _, elem := range v {
					modifyInParentAt(elem, seg, fn, depth+1)
				}
			}
		}
	}
}
