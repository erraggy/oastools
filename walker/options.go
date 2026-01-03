package walker

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// walkConfig holds configuration for WalkWithOptions.
type walkConfig struct {
	// Input sources
	filePath *string
	parsed   *parser.ParseResult

	// Handlers
	onDocument       DocumentHandler
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

	// Configuration
	maxSchemaDepth int
}

// WalkInputOption configures the WalkWithOptions function.
// Options may return an error for invalid configuration values (e.g., non-positive maxSchemaDepth).
type WalkInputOption func(*walkConfig) error

// WithFilePath specifies a file path to parse and walk.
func WithFilePath(path string) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.filePath = &path
		return nil
	}
}

// WithParsed specifies a pre-parsed result to walk.
func WithParsed(result *parser.ParseResult) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.parsed = result
		return nil
	}
}

// WithMaxSchemaDepthOption sets the maximum schema recursion depth.
// Returns an error if depth is not positive.
func WithMaxSchemaDepthOption(depth int) WalkInputOption {
	return func(cfg *walkConfig) error {
		if depth <= 0 {
			return fmt.Errorf("maxSchemaDepth must be positive, got %d", depth)
		}
		cfg.maxSchemaDepth = depth
		return nil
	}
}

// OnDocument registers a handler for the root document.
func OnDocument(fn DocumentHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onDocument = fn
		return nil
	}
}

// OnInfo registers a handler for Info objects.
func OnInfo(fn InfoHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onInfo = fn
		return nil
	}
}

// OnServer registers a handler for Server objects.
func OnServer(fn ServerHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onServer = fn
		return nil
	}
}

// OnTag registers a handler for Tag objects.
func OnTag(fn TagHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onTag = fn
		return nil
	}
}

// OnPath registers a handler for path entries.
func OnPath(fn PathHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onPath = fn
		return nil
	}
}

// OnPathItem registers a handler for PathItem objects.
func OnPathItem(fn PathItemHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onPathItem = fn
		return nil
	}
}

// OnOperation registers a handler for Operation objects.
func OnOperation(fn OperationHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onOperation = fn
		return nil
	}
}

// OnParameter registers a handler for Parameter objects.
func OnParameter(fn ParameterHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onParameter = fn
		return nil
	}
}

// OnRequestBody registers a handler for RequestBody objects.
func OnRequestBody(fn RequestBodyHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onRequestBody = fn
		return nil
	}
}

// OnResponse registers a handler for Response objects.
func OnResponse(fn ResponseHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onResponse = fn
		return nil
	}
}

// OnSchema registers a handler for Schema objects.
func OnSchema(fn SchemaHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onSchema = fn
		return nil
	}
}

// OnSecurityScheme registers a handler for SecurityScheme objects.
func OnSecurityScheme(fn SecuritySchemeHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onSecurityScheme = fn
		return nil
	}
}

// OnHeader registers a handler for Header objects.
func OnHeader(fn HeaderHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onHeader = fn
		return nil
	}
}

// OnMediaType registers a handler for MediaType objects.
func OnMediaType(fn MediaTypeHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onMediaType = fn
		return nil
	}
}

// OnLink registers a handler for Link objects.
func OnLink(fn LinkHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onLink = fn
		return nil
	}
}

// OnCallback registers a handler for Callback objects.
func OnCallback(fn CallbackHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onCallback = fn
		return nil
	}
}

// OnExample registers a handler for Example objects.
func OnExample(fn ExampleHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onExample = fn
		return nil
	}
}

// OnExternalDocs registers a handler for ExternalDocs objects.
func OnExternalDocs(fn ExternalDocsHandler) WalkInputOption {
	return func(cfg *walkConfig) error {
		cfg.onExternalDocs = fn
		return nil
	}
}

// WalkWithOptions walks a document using functional options for input and handlers.
func WalkWithOptions(opts ...WalkInputOption) error {
	cfg := &walkConfig{
		maxSchemaDepth: 100,
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return fmt.Errorf("walker: %w", err)
		}
	}

	// Validate input
	if cfg.parsed == nil && cfg.filePath == nil {
		return fmt.Errorf("walker: no input source specified: use WithFilePath or WithParsed")
	}
	if cfg.parsed != nil && cfg.filePath != nil {
		return fmt.Errorf("walker: multiple input sources specified: use only one")
	}

	// Get or create ParseResult
	var result *parser.ParseResult
	if cfg.parsed != nil {
		result = cfg.parsed
	} else {
		p := parser.New()
		var err error
		result, err = p.Parse(*cfg.filePath)
		if err != nil {
			return fmt.Errorf("walker: failed to parse: %w", err)
		}
	}

	// Build walker with handlers
	w := &Walker{
		onDocument:       cfg.onDocument,
		onInfo:           cfg.onInfo,
		onServer:         cfg.onServer,
		onTag:            cfg.onTag,
		onPath:           cfg.onPath,
		onPathItem:       cfg.onPathItem,
		onOperation:      cfg.onOperation,
		onParameter:      cfg.onParameter,
		onRequestBody:    cfg.onRequestBody,
		onResponse:       cfg.onResponse,
		onSchema:         cfg.onSchema,
		onSecurityScheme: cfg.onSecurityScheme,
		onHeader:         cfg.onHeader,
		onMediaType:      cfg.onMediaType,
		onLink:           cfg.onLink,
		onCallback:       cfg.onCallback,
		onExample:        cfg.onExample,
		onExternalDocs:   cfg.onExternalDocs,
		maxSchemaDepth:   cfg.maxSchemaDepth,
	}

	return w.walk(result)
}
