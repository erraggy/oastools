package walker

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// WithFilePath specifies a file path to parse and walk.
func WithFilePath(path string) Option {
	return func(w *Walker) {
		w.filePath = &path
	}
}

// WithParsed specifies a pre-parsed result to walk.
func WithParsed(result *parser.ParseResult) Option {
	return func(w *Walker) {
		w.parsed = result
	}
}

// WithMaxSchemaDepth sets the maximum schema recursion depth.
// The default depth is 100.
//
// Depth must be a positive integer (>= 1). Values of 0 or negative are
// silently ignored and the default of 100 is kept. There is no "unlimited"
// depth option to prevent infinite recursion in circular schemas.
//
// When the depth limit is reached, the walker skips the schema and calls
// the schema-skipped handler (if registered) with reason "depth".
func WithMaxSchemaDepth(depth int) Option {
	return func(w *Walker) {
		if depth > 0 {
			w.maxDepth = depth
		}
		// Values <= 0 are ignored; keep the default (100)
	}
}

// WithRefTracking enables tracking of $ref values during traversal.
// When enabled, the walker tracks reference information internally.
// To receive the CurrentRef value, use WithRefHandler to register a callback
// that is invoked with the populated WalkContext.CurrentRef for each $ref.
func WithRefTracking() Option {
	return func(w *Walker) {
		w.trackRefs = true
	}
}

// WithRefHandler sets a handler called when a $ref is encountered.
// Implicitly enables ref tracking.
func WithRefHandler(fn RefHandler) Option {
	return func(w *Walker) {
		w.trackRefs = true
		w.onRef = fn
	}
}

// WithParentTracking enables tracking of parent nodes during traversal.
// When enabled, WalkContext.Parent provides access to ancestor nodes,
// and helper methods like ParentSchema(), ParentOperation(), ParentPathItem(),
// ParentResponse(), ParentRequestBody(), Ancestors(), and Depth() become available.
//
// This adds some overhead (parent stack management), so only enable when needed.
// By default, parent tracking is disabled for optimal performance.
func WithParentTracking() Option {
	return func(w *Walker) {
		w.trackParent = true
	}
}

// WithMapRefTracking enables tracking of $ref values stored in map[string]any structures.
// When enabled, the walker will detect refs in polymorphic schema fields (Items, AdditionalItems,
// AdditionalProperties, UnevaluatedItems, UnevaluatedProperties) that were not parsed as *Schema.
// Implicitly enables ref tracking.
//
// Polymorphic fields may contain map[string]any instead of *Schema when:
//   - Documents are parsed from raw YAML/JSON without full schema resolution
//   - Manually constructing documents with map literals (e.g., in tests)
//   - Using external tooling that produces partially resolved documents
//
// The walker will call the ref handler for any $ref values found in these map structures.
func WithMapRefTracking() Option {
	return func(w *Walker) {
		w.trackRefs = true
		w.trackMapRefs = true
	}
}

// WalkWithOptions walks a document using functional options for input, handlers, and configuration.
// All options use the unified Option type - no adapter is needed.
//
// Example:
//
//	walker.WalkWithOptions(
//	    walker.WithFilePath("openapi.yaml"),
//	    walker.WithSchemaHandler(func(wc *walker.WalkContext, s *parser.Schema) walker.Action {
//	        fmt.Println(wc.JSONPath)
//	        return walker.Continue
//	    }),
//	)
func WalkWithOptions(opts ...Option) error {
	w := New()
	for _, opt := range opts {
		opt(w)
	}

	// Validate input
	if w.parsed == nil && w.filePath == nil {
		return fmt.Errorf("walker: no input source specified: use WithFilePath or WithParsed")
	}
	if w.parsed != nil && w.filePath != nil {
		return fmt.Errorf("walker: multiple input sources specified: use only one")
	}

	// Get or create ParseResult
	var result *parser.ParseResult
	if w.parsed != nil {
		result = w.parsed
	} else {
		p := parser.New()
		var err error
		result, err = p.Parse(*w.filePath)
		if err != nil {
			return fmt.Errorf("walker: failed to parse: %w", err)
		}
	}

	return w.walk(result)
}
