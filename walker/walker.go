package walker

import (
	"context"
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// Action controls the walker's behavior after visiting a node.
type Action int

const (
	// Continue continues walking normally, visiting children and siblings.
	Continue Action = iota

	// SkipChildren skips all children of the current node but continues with siblings.
	SkipChildren

	// Stop stops the walk immediately. No more nodes will be visited.
	Stop
)

// IsValid returns true if the action is one of the defined constants.
func (a Action) IsValid() bool {
	return a >= Continue && a <= Stop
}

// String returns a string representation of the action.
func (a Action) String() string {
	switch a {
	case Continue:
		return "Continue"
	case SkipChildren:
		return "SkipChildren"
	case Stop:
		return "Stop"
	default:
		return fmt.Sprintf("Action(%d)", a)
	}
}

// Handler types for each OAS node type.
// Each handler receives a WalkContext with contextual information and the node,
// and returns an Action to control traversal.

// DocumentHandler is called for the root document (OAS2Document or OAS3Document).
// The JSON path is available in wc.JSONPath.
type DocumentHandler func(wc *WalkContext, doc any) Action

// OAS2DocumentHandler is called for OAS 2.0 (Swagger) documents.
// The JSON path is available in wc.JSONPath.
type OAS2DocumentHandler func(wc *WalkContext, doc *parser.OAS2Document) Action

// OAS3DocumentHandler is called for OAS 3.x documents.
// The JSON path is available in wc.JSONPath.
type OAS3DocumentHandler func(wc *WalkContext, doc *parser.OAS3Document) Action

// InfoHandler is called for the Info object.
// The JSON path is available in wc.JSONPath.
type InfoHandler func(wc *WalkContext, info *parser.Info) Action

// ServerHandler is called for each Server (OAS 3.x only).
// The JSON path is available in wc.JSONPath.
type ServerHandler func(wc *WalkContext, server *parser.Server) Action

// TagHandler is called for each Tag.
// The JSON path is available in wc.JSONPath.
type TagHandler func(wc *WalkContext, tag *parser.Tag) Action

// PathHandler is called for each path entry.
// The path template is available in wc.PathTemplate. The JSON path is in wc.JSONPath.
type PathHandler func(wc *WalkContext, pathItem *parser.PathItem) Action

// PathItemHandler is called for each PathItem.
// The path template is available in wc.PathTemplate. The JSON path is in wc.JSONPath.
type PathItemHandler func(wc *WalkContext, pathItem *parser.PathItem) Action

// OperationHandler is called for each Operation.
// The HTTP method is available in wc.Method. The JSON path is in wc.JSONPath.
type OperationHandler func(wc *WalkContext, op *parser.Operation) Action

// ParameterHandler is called for each Parameter.
// The JSON path is available in wc.JSONPath.
type ParameterHandler func(wc *WalkContext, param *parser.Parameter) Action

// RequestBodyHandler is called for each RequestBody (OAS 3.x only).
// The JSON path is available in wc.JSONPath.
type RequestBodyHandler func(wc *WalkContext, reqBody *parser.RequestBody) Action

// ResponseHandler is called for each Response.
// The status code is available in wc.StatusCode. The JSON path is in wc.JSONPath.
type ResponseHandler func(wc *WalkContext, resp *parser.Response) Action

// SchemaHandler is called for each Schema, including nested schemas.
// The JSON path is available in wc.JSONPath. For named schemas, wc.Name contains the name.
type SchemaHandler func(wc *WalkContext, schema *parser.Schema) Action

// SecuritySchemeHandler is called for each SecurityScheme.
// The scheme name is available in wc.Name. The JSON path is in wc.JSONPath.
type SecuritySchemeHandler func(wc *WalkContext, scheme *parser.SecurityScheme) Action

// HeaderHandler is called for each Header.
// The header name is available in wc.Name. The JSON path is in wc.JSONPath.
type HeaderHandler func(wc *WalkContext, header *parser.Header) Action

// MediaTypeHandler is called for each MediaType (OAS 3.x only).
// The media type name is available in wc.Name. The JSON path is in wc.JSONPath.
type MediaTypeHandler func(wc *WalkContext, mt *parser.MediaType) Action

// LinkHandler is called for each Link (OAS 3.x only).
// The link name is available in wc.Name. The JSON path is in wc.JSONPath.
type LinkHandler func(wc *WalkContext, link *parser.Link) Action

// CallbackHandler is called for each Callback (OAS 3.x only).
// The callback name is available in wc.Name. The JSON path is in wc.JSONPath.
type CallbackHandler func(wc *WalkContext, callback parser.Callback) Action

// ExampleHandler is called for each Example.
// The example name is available in wc.Name. The JSON path is in wc.JSONPath.
type ExampleHandler func(wc *WalkContext, example *parser.Example) Action

// ExternalDocsHandler is called for each ExternalDocs.
// The JSON path is available in wc.JSONPath.
type ExternalDocsHandler func(wc *WalkContext, extDocs *parser.ExternalDocs) Action

// SchemaSkippedHandler is called when a schema is skipped due to depth limit or cycle detection.
// The reason parameter is either "depth" when the schema exceeds maxDepth,
// or "cycle" when the schema was already visited (circular reference detected).
// The schema parameter is the schema that was skipped. The JSON path is in wc.JSONPath.
type SchemaSkippedHandler func(wc *WalkContext, reason string, schema *parser.Schema)

// Post-visit handler types.
// These handlers are called after a node's children have been processed.
// They do not return an Action since children are already visited.
// Post handlers are not called if the pre-visit handler returned SkipChildren or Stop.

// SchemaPostHandler is called after a schema's children have been processed.
type SchemaPostHandler func(wc *WalkContext, schema *parser.Schema)

// OperationPostHandler is called after an operation's children have been processed.
type OperationPostHandler func(wc *WalkContext, op *parser.Operation)

// PathItemPostHandler is called after a path item's children have been processed.
type PathItemPostHandler func(wc *WalkContext, pathItem *parser.PathItem)

// ResponsePostHandler is called after a response's children have been processed.
type ResponsePostHandler func(wc *WalkContext, resp *parser.Response)

// RequestBodyPostHandler is called after a request body's children have been processed.
type RequestBodyPostHandler func(wc *WalkContext, reqBody *parser.RequestBody)

// CallbackPostHandler is called after a callback's children have been processed.
type CallbackPostHandler func(wc *WalkContext, callback parser.Callback)

// Walker traverses OpenAPI documents and calls handlers for each node type.
type Walker struct {
	// Input sources (mutually exclusive)
	filePath *string
	parsed   *parser.ParseResult

	// Handlers
	onDocument       DocumentHandler
	onOAS2Document   OAS2DocumentHandler
	onOAS3Document   OAS3DocumentHandler
	onInfo           InfoHandler
	onServer         ServerHandler
	onTag            TagHandler
	onPath           PathHandler
	onPathItem       PathItemHandler
	onOperation      OperationHandler
	onParameter      ParameterHandler
	onRequestBody    RequestBodyHandler
	onResponse       ResponseHandler
	onSchema         SchemaHandler
	onSecurityScheme SecuritySchemeHandler
	onHeader         HeaderHandler
	onMediaType      MediaTypeHandler
	onLink           LinkHandler
	onCallback       CallbackHandler
	onExample        ExampleHandler
	onExternalDocs   ExternalDocsHandler
	onSchemaSkipped  SchemaSkippedHandler
	onRef            RefHandler

	// Post-visit handlers
	onSchemaPost      SchemaPostHandler
	onOperationPost   OperationPostHandler
	onPathItemPost    PathItemPostHandler
	onResponsePost    ResponsePostHandler
	onRequestBodyPost RequestBodyPostHandler
	onCallbackPost    CallbackPostHandler

	// Configuration
	maxDepth    int
	trackRefs   bool
	trackParent bool
	userCtx     context.Context

	// Internal state
	visitedSchemas map[*parser.Schema]bool
	stopped        bool
}

// New creates a new Walker with default settings.
func New() *Walker {
	return &Walker{
		maxDepth: 100,
	}
}

// Option configures the Walker.
type Option func(*Walker)

// WithDocumentHandler sets the handler for the root document.
func WithDocumentHandler(fn DocumentHandler) Option {
	return func(w *Walker) { w.onDocument = fn }
}

// WithOAS2DocumentHandler sets the handler for OAS 2.0 (Swagger) documents.
// This handler is called before the generic DocumentHandler.
func WithOAS2DocumentHandler(fn OAS2DocumentHandler) Option {
	return func(w *Walker) { w.onOAS2Document = fn }
}

// WithOAS3DocumentHandler sets the handler for OAS 3.x documents.
// This handler is called before the generic DocumentHandler.
func WithOAS3DocumentHandler(fn OAS3DocumentHandler) Option {
	return func(w *Walker) { w.onOAS3Document = fn }
}

// WithInfoHandler sets the handler for Info objects.
func WithInfoHandler(fn InfoHandler) Option {
	return func(w *Walker) { w.onInfo = fn }
}

// WithServerHandler sets the handler for Server objects (OAS 3.x only).
func WithServerHandler(fn ServerHandler) Option {
	return func(w *Walker) { w.onServer = fn }
}

// WithTagHandler sets the handler for Tag objects.
func WithTagHandler(fn TagHandler) Option {
	return func(w *Walker) { w.onTag = fn }
}

// WithPathHandler sets the handler for path entries.
func WithPathHandler(fn PathHandler) Option {
	return func(w *Walker) { w.onPath = fn }
}

// WithPathItemHandler sets the handler for PathItem objects.
func WithPathItemHandler(fn PathItemHandler) Option {
	return func(w *Walker) { w.onPathItem = fn }
}

// WithOperationHandler sets the handler for Operation objects.
func WithOperationHandler(fn OperationHandler) Option {
	return func(w *Walker) { w.onOperation = fn }
}

// WithParameterHandler sets the handler for Parameter objects.
func WithParameterHandler(fn ParameterHandler) Option {
	return func(w *Walker) { w.onParameter = fn }
}

// WithRequestBodyHandler sets the handler for RequestBody objects (OAS 3.x only).
func WithRequestBodyHandler(fn RequestBodyHandler) Option {
	return func(w *Walker) { w.onRequestBody = fn }
}

// WithResponseHandler sets the handler for Response objects.
func WithResponseHandler(fn ResponseHandler) Option {
	return func(w *Walker) { w.onResponse = fn }
}

// WithSchemaHandler sets the handler for Schema objects.
func WithSchemaHandler(fn SchemaHandler) Option {
	return func(w *Walker) { w.onSchema = fn }
}

// WithSecuritySchemeHandler sets the handler for SecurityScheme objects.
func WithSecuritySchemeHandler(fn SecuritySchemeHandler) Option {
	return func(w *Walker) { w.onSecurityScheme = fn }
}

// WithHeaderHandler sets the handler for Header objects.
func WithHeaderHandler(fn HeaderHandler) Option {
	return func(w *Walker) { w.onHeader = fn }
}

// WithMediaTypeHandler sets the handler for MediaType objects (OAS 3.x only).
func WithMediaTypeHandler(fn MediaTypeHandler) Option {
	return func(w *Walker) { w.onMediaType = fn }
}

// WithLinkHandler sets the handler for Link objects (OAS 3.x only).
func WithLinkHandler(fn LinkHandler) Option {
	return func(w *Walker) { w.onLink = fn }
}

// WithCallbackHandler sets the handler for Callback objects (OAS 3.x only).
func WithCallbackHandler(fn CallbackHandler) Option {
	return func(w *Walker) { w.onCallback = fn }
}

// WithExampleHandler sets the handler for Example objects.
func WithExampleHandler(fn ExampleHandler) Option {
	return func(w *Walker) { w.onExample = fn }
}

// WithExternalDocsHandler sets the handler for ExternalDocs objects.
func WithExternalDocsHandler(fn ExternalDocsHandler) Option {
	return func(w *Walker) { w.onExternalDocs = fn }
}

// WithSchemaSkippedHandler sets the handler called when schemas are skipped.
// This handler is invoked when a schema is skipped due to depth limit ("depth")
// or cycle detection ("cycle").
func WithSchemaSkippedHandler(fn SchemaSkippedHandler) Option {
	return func(w *Walker) { w.onSchemaSkipped = fn }
}

// Post-visit handler options.
// These handlers are called after a node's children have been processed,
// enabling bottom-up processing patterns like aggregation.

// WithSchemaPostHandler sets a handler called after a schema's children are processed.
// Not called if the pre-visit handler returns SkipChildren or Stop.
func WithSchemaPostHandler(fn SchemaPostHandler) Option {
	return func(w *Walker) { w.onSchemaPost = fn }
}

// WithOperationPostHandler sets a handler called after an operation's children are processed.
// Not called if the pre-visit handler returns SkipChildren or Stop.
func WithOperationPostHandler(fn OperationPostHandler) Option {
	return func(w *Walker) { w.onOperationPost = fn }
}

// WithPathItemPostHandler sets a handler called after a path item's children are processed.
// Not called if the pre-visit handler returns SkipChildren or Stop.
func WithPathItemPostHandler(fn PathItemPostHandler) Option {
	return func(w *Walker) { w.onPathItemPost = fn }
}

// WithResponsePostHandler sets a handler called after a response's children are processed.
// Not called if the pre-visit handler returns SkipChildren or Stop.
func WithResponsePostHandler(fn ResponsePostHandler) Option {
	return func(w *Walker) { w.onResponsePost = fn }
}

// WithRequestBodyPostHandler sets a handler called after a request body's children are processed.
// Not called if the pre-visit handler returns SkipChildren or Stop.
func WithRequestBodyPostHandler(fn RequestBodyPostHandler) Option {
	return func(w *Walker) { w.onRequestBodyPost = fn }
}

// WithCallbackPostHandler sets a handler called after a callback's children are processed.
// Not called if the pre-visit handler returns SkipChildren or Stop.
func WithCallbackPostHandler(fn CallbackPostHandler) Option {
	return func(w *Walker) { w.onCallbackPost = fn }
}

// WithMaxDepth sets the maximum schema recursion depth.
// The default depth is 100.
//
// Depth must be a positive integer (>= 1). Values of 0 or negative are
// silently ignored and the default of 100 is kept. There is no "unlimited"
// depth option to prevent infinite recursion in circular schemas.
//
// When the depth limit is reached, the walker skips the schema and calls
// the schema-skipped handler (if registered) with reason "depth".
//
// Deprecated: Use WithMaxSchemaDepth instead for clarity.
func WithMaxDepth(depth int) Option {
	return func(w *Walker) {
		if depth > 0 {
			w.maxDepth = depth
		}
		// Values <= 0 are ignored; keep the default (100)
	}
}

// WithContext sets the context for cancellation and deadline propagation.
// The context is available to handlers via wc.Context().
func WithContext(ctx context.Context) Option {
	return func(w *Walker) {
		w.userCtx = ctx
	}
}

// Walk traverses the parsed document and calls registered handlers for each node.
func Walk(result *parser.ParseResult, opts ...Option) error {
	if result == nil {
		return fmt.Errorf("walker: nil ParseResult")
	}
	if result.Document == nil {
		return fmt.Errorf("walker: nil Document in ParseResult")
	}

	w := New()
	for _, opt := range opts {
		opt(w)
	}

	return w.walk(result)
}

// walk performs the actual traversal.
func (w *Walker) walk(result *parser.ParseResult) error {
	w.visitedSchemas = make(map[*parser.Schema]bool)
	w.stopped = false

	// Create initial walk state with user context and walker reference
	state := &walkState{
		ctx:    w.userCtx,
		walker: w,
	}

	// Initialize parent stack if tracking is enabled
	if w.trackParent {
		stack := make([]*ParentInfo, 0, 16)
		state.parentStack = &stack
	}

	switch doc := result.Document.(type) {
	case *parser.OAS2Document:
		return w.walkOAS2(doc, state)
	case *parser.OAS3Document:
		return w.walkOAS3(doc, state)
	default:
		return fmt.Errorf("walker: unsupported document type: %T", result.Document)
	}
}

// handleAction processes the action returned by a handler.
// Returns true if walking should continue to children.
//
// Action values are handled as follows:
//   - Continue (0): continue walking to children and siblings
//   - SkipChildren (1): skip children but continue with siblings
//   - Stop (2): halt traversal immediately
//   - Any other value: treated as Continue (this includes the zero value,
//     which is intentionally Continue for ergonomic default behavior)
//
// Invalid Action values (e.g., Action(42)) are treated as Continue.
// Use Action.IsValid() to check if an action is one of the defined constants.
func (w *Walker) handleAction(action Action) bool {
	switch action {
	case Stop:
		w.stopped = true
		return false
	case SkipChildren:
		return false
	default:
		// Continue and any invalid action values continue walking.
		// The zero value of Action is Continue (intentional design).
		return true
	}
}
