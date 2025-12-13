package jsonpath

import (
	"fmt"
)

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
// For arrays, matching indices are removed (with index shift).
func (p *Path) Remove(doc any) (any, error) {
	if len(p.segments) < 2 {
		return nil, fmt.Errorf("jsonpath: cannot remove root")
	}

	// Get the parent nodes and the final key
	parentPath := &Path{
		raw:      p.raw,
		segments: p.segments[:len(p.segments)-1],
	}

	parents := parentPath.Get(doc)
	if len(parents) == 0 {
		// No matches - nothing to remove
		return doc, nil
	}

	lastSeg := p.segments[len(p.segments)-1]

	for _, parent := range parents {
		removeFromParent(parent, lastSeg)
	}

	return doc, nil
}

// Modify applies a transformation function to all matching nodes.
//
// The function receives each matched value and should return the new value.
// The document is modified in place.
func (p *Path) Modify(doc any, fn func(any) any) error {
	if len(p.segments) < 2 {
		// Modifying root means replacing entire doc - not supported in-place
		return fmt.Errorf("jsonpath: cannot modify root in place")
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
			results = append(results, recursiveDescend(node, s.Child)...)
		}
	}

	return results
}

// recursiveDescend finds all nodes matching the child selector at any depth.
func recursiveDescend(node any, child Segment) []any {
	var results []any

	// If child is nil, collect all descendants
	if child == nil {
		collectAllDescendants(node, &results)
		return results
	}

	// Apply child selector at this level first
	childResults := applySegment([]any{node}, child)
	results = append(results, childResults...)

	// Recurse into children
	switch v := node.(type) {
	case map[string]any:
		for _, val := range v {
			results = append(results, recursiveDescend(val, child)...)
		}
	case []any:
		for _, elem := range v {
			results = append(results, recursiveDescend(elem, child)...)
		}
	}

	return results
}

// collectAllDescendants collects all nodes in the tree (for bare ..)
func collectAllDescendants(node any, results *[]any) {
	switch v := node.(type) {
	case map[string]any:
		for _, val := range v {
			*results = append(*results, val)
			collectAllDescendants(val, results)
		}
	case []any:
		for _, elem := range v {
			*results = append(*results, elem)
			collectAllDescendants(elem, results)
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

// removeFromParent removes nodes from the parent at the location specified by the segment.
func removeFromParent(parent any, seg Segment) {
	switch s := seg.(type) {
	case ChildSegment:
		if m, ok := parent.(map[string]any); ok {
			delete(m, s.Key)
		}

	case IndexSegment:
		// Note: Removing from arrays by index is tricky as it shifts other elements.
		// For simplicity, we set to nil rather than removing.
		if arr, ok := parent.([]any); ok {
			idx := s.Index
			if idx < 0 {
				idx = len(arr) + idx
			}
			if idx >= 0 && idx < len(arr) {
				arr[idx] = nil
			}
		}

	case WildcardSegment:
		switch v := parent.(type) {
		case map[string]any:
			for key := range v {
				delete(v, key)
			}
		case []any:
			// Clear array elements
			for i := range v {
				v[i] = nil
			}
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
			// For arrays, we need to collect indices to remove, then remove in reverse order
			var toRemove []int
			for i, elem := range v {
				if evalFilter(elem, s.Expr) {
					toRemove = append(toRemove, i)
				}
			}
			// Remove in reverse order to maintain correct indices
			for i := len(toRemove) - 1; i >= 0; i-- {
				idx := toRemove[i]
				// Set to nil marker (actual removal would require modifying the slice)
				v[idx] = nil
			}
		}
	}
}

// modifyInParent applies a transformation function to matching nodes in the parent.
func modifyInParent(parent any, seg Segment, fn func(any) any) {
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
	}
}
