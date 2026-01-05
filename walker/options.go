package walker

import (
	"context"
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
// If depth is not positive, it is silently ignored and the default (100) is kept.
func WithMaxSchemaDepth(depth int) Option {
	return func(w *Walker) {
		if depth > 0 {
			w.maxDepth = depth
		}
		// If depth <= 0, keep the default (100)
	}
}

// WithUserContext sets the context for cancellation and deadline propagation.
// The context is available to handlers via wc.Context().
func WithUserContext(ctx context.Context) Option {
	return func(w *Walker) {
		w.userCtx = ctx
	}
}

// WithRefTracking enables tracking of $ref values during traversal.
// When enabled, WalkContext.CurrentRef is populated for nodes with refs.
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
