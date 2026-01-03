package walker

import (
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
// Each handler receives the node and its JSON path, and returns an Action.

// DocumentHandler is called for the root document (OAS2Document or OAS3Document).
type DocumentHandler func(doc any, path string) Action

// OAS2DocumentHandler is called for OAS 2.0 (Swagger) documents.
type OAS2DocumentHandler func(doc *parser.OAS2Document, path string) Action

// OAS3DocumentHandler is called for OAS 3.x documents.
type OAS3DocumentHandler func(doc *parser.OAS3Document, path string) Action

// InfoHandler is called for the Info object.
type InfoHandler func(info *parser.Info, path string) Action

// ServerHandler is called for each Server (OAS 3.x only).
type ServerHandler func(server *parser.Server, path string) Action

// TagHandler is called for each Tag.
type TagHandler func(tag *parser.Tag, path string) Action

// PathHandler is called for each path entry with the path template string.
type PathHandler func(pathTemplate string, pathItem *parser.PathItem, path string) Action

// PathItemHandler is called for each PathItem.
type PathItemHandler func(pathItem *parser.PathItem, path string) Action

// OperationHandler is called for each Operation.
type OperationHandler func(method string, op *parser.Operation, path string) Action

// ParameterHandler is called for each Parameter.
type ParameterHandler func(param *parser.Parameter, path string) Action

// RequestBodyHandler is called for each RequestBody (OAS 3.x only).
type RequestBodyHandler func(reqBody *parser.RequestBody, path string) Action

// ResponseHandler is called for each Response.
type ResponseHandler func(statusCode string, resp *parser.Response, path string) Action

// SchemaHandler is called for each Schema, including nested schemas.
type SchemaHandler func(schema *parser.Schema, path string) Action

// SecuritySchemeHandler is called for each SecurityScheme.
type SecuritySchemeHandler func(name string, scheme *parser.SecurityScheme, path string) Action

// HeaderHandler is called for each Header.
type HeaderHandler func(name string, header *parser.Header, path string) Action

// MediaTypeHandler is called for each MediaType (OAS 3.x only).
type MediaTypeHandler func(mediaTypeName string, mt *parser.MediaType, path string) Action

// LinkHandler is called for each Link (OAS 3.x only).
type LinkHandler func(name string, link *parser.Link, path string) Action

// CallbackHandler is called for each Callback (OAS 3.x only).
type CallbackHandler func(name string, callback parser.Callback, path string) Action

// ExampleHandler is called for each Example.
type ExampleHandler func(name string, example *parser.Example, path string) Action

// ExternalDocsHandler is called for each ExternalDocs.
type ExternalDocsHandler func(extDocs *parser.ExternalDocs, path string) Action

// SchemaSkippedHandler is called when a schema is skipped due to depth limit or cycle detection.
// The reason parameter is either "depth" when the schema exceeds maxDepth,
// or "cycle" when the schema was already visited (circular reference detected).
// The schema parameter is the schema that was skipped, and path is its JSON path.
type SchemaSkippedHandler func(reason string, schema *parser.Schema, path string)

// Walker traverses OpenAPI documents and calls handlers for each node type.
type Walker struct {
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

	// Configuration
	maxDepth int

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

// WithMaxDepth sets the maximum recursion depth for schema traversal.
// Default is 100. If depth is <= 0, the default is kept.
func WithMaxDepth(depth int) Option {
	return func(w *Walker) {
		if depth > 0 {
			w.maxDepth = depth
		}
		// If depth <= 0, keep the default (100)
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

	switch doc := result.Document.(type) {
	case *parser.OAS2Document:
		return w.walkOAS2(doc)
	case *parser.OAS3Document:
		return w.walkOAS3(doc)
	default:
		return fmt.Errorf("walker: unsupported document type: %T", result.Document)
	}
}

// handleAction processes the action returned by a handler.
// Returns true if walking should continue to children.
func (w *Walker) handleAction(action Action) bool {
	switch action {
	case Stop:
		w.stopped = true
		return false
	case SkipChildren:
		return false
	default:
		return true
	}
}
