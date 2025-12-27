package builder

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"sync"

	"github.com/erraggy/oastools/httpvalidator"
	"github.com/erraggy/oastools/parser"
)

// ServerBuilder extends Builder to support server construction.
// It embeds Builder, inheriting all specification construction methods.
//
// Concurrency: ServerBuilder instances are not safe for concurrent use.
// Create separate ServerBuilder instances for concurrent operations.
type ServerBuilder struct {
	*Builder
	mu           sync.RWMutex
	handlers     map[string]map[string]HandlerFunc // path -> method -> handler
	middleware   []Middleware
	router       RouterStrategy
	errorHandler ErrorHandler
	config       serverBuilderConfig
}

// NewServerBuilder creates a ServerBuilder for the specified OAS version.
// This is the primary entry point for the server builder API.
//
// Example:
//
//	srv := builder.NewServerBuilder(parser.OASVersion320).
//		SetTitle("Pet Store API").
//		SetVersion("1.0.0")
//
//	srv.AddOperation(http.MethodGet, "/pets",
//		builder.WithHandler(listPetsHandler),
//		builder.WithResponse(http.StatusOK, []Pet{}),
//	)
//
//	result, err := srv.BuildServer()
func NewServerBuilder(version parser.OASVersion, opts ...ServerBuilderOption) *ServerBuilder {
	cfg := defaultServerBuilderConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &ServerBuilder{
		Builder:      New(version),
		handlers:     make(map[string]map[string]HandlerFunc),
		middleware:   make([]Middleware, 0),
		router:       cfg.router,
		errorHandler: cfg.errorHandler,
		config:       cfg,
	}
}

// FromBuilder creates a ServerBuilder from an existing Builder.
// This allows converting an existing specification into a runnable server.
//
// Example:
//
//	b := builder.New(parser.OASVersion320).SetTitle("My API")
//	srv := builder.FromBuilder(b)
//	srv.Handle(http.MethodGet, "/users", listUsersHandler)
func FromBuilder(b *Builder, opts ...ServerBuilderOption) *ServerBuilder {
	cfg := defaultServerBuilderConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &ServerBuilder{
		Builder:      b,
		handlers:     make(map[string]map[string]HandlerFunc),
		middleware:   make([]Middleware, 0),
		router:       cfg.router,
		errorHandler: cfg.errorHandler,
		config:       cfg,
	}
}

// SetTitle sets the API title. Overrides Builder.SetTitle to maintain fluent chaining.
func (s *ServerBuilder) SetTitle(title string) *ServerBuilder {
	s.Builder.SetTitle(title)
	return s
}

// SetVersion sets the API version. Overrides Builder.SetVersion to maintain fluent chaining.
func (s *ServerBuilder) SetVersion(version string) *ServerBuilder {
	s.Builder.SetVersion(version)
	return s
}

// SetDescription sets the API description. Overrides Builder.SetDescription to maintain fluent chaining.
func (s *ServerBuilder) SetDescription(desc string) *ServerBuilder {
	s.Builder.SetDescription(desc)
	return s
}

// AddServer adds a server definition. Overrides Builder.AddServer to maintain fluent chaining.
func (s *ServerBuilder) AddServer(url string, opts ...ServerOption) *ServerBuilder {
	s.Builder.AddServer(url, opts...)
	return s
}

// AddOperation adds an operation and returns the ServerBuilder for chaining.
// Overrides Builder.AddOperation to support inline handler registration via WithHandler.
//
// Example:
//
//	srv.AddOperation(http.MethodGet, "/pets",
//		builder.WithHandler(listPetsHandler),
//		builder.WithResponse(http.StatusOK, []Pet{}),
//	)
func (s *ServerBuilder) AddOperation(method, path string, opts ...OperationOption) *ServerBuilder {
	// Extract handler from options before passing to Builder
	// Initialize responses map since WithResponse writes directly to it
	cfg := &operationConfig{
		responses: make(map[string]*responseBuilder),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Register handler if provided (keyed by method+path)
	if cfg.handler != nil {
		s.registerHandler(method, path, cfg.handler)
	}

	// Call parent to build the operation
	s.Builder.AddOperation(method, path, opts...)
	return s
}

// registerHandler stores a handler for the given method and path.
func (s *ServerBuilder) registerHandler(method, path string, handler HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handlers[path] == nil {
		s.handlers[path] = make(map[string]HandlerFunc)
	}
	s.handlers[path][method] = handler
}

// AddTag adds a tag definition. Overrides Builder.AddTag to maintain fluent chaining.
func (s *ServerBuilder) AddTag(name string, opts ...TagOption) *ServerBuilder {
	s.Builder.AddTag(name, opts...)
	return s
}

// AddSecurityScheme adds a security scheme. Overrides Builder.AddSecurityScheme to maintain fluent chaining.
func (s *ServerBuilder) AddSecurityScheme(name string, scheme *parser.SecurityScheme) *ServerBuilder {
	s.Builder.AddSecurityScheme(name, scheme)
	return s
}

// SetSecurity sets global security requirements. Overrides Builder.SetSecurity to maintain fluent chaining.
func (s *ServerBuilder) SetSecurity(requirements ...parser.SecurityRequirement) *ServerBuilder {
	s.Builder.SetSecurity(requirements...)
	return s
}

// Handle registers a handler for an operation by method and path.
// This is an alternative to using WithHandler in AddOperation, useful for
// dynamic handler registration or when the handler isn't known at definition time.
//
// Example:
//
//	srv.Handle(http.MethodGet, "/pets", func(ctx context.Context, req *builder.Request) builder.Response {
//		return builder.JSON(http.StatusOK, pets)
//	})
func (s *ServerBuilder) Handle(method, path string, handler HandlerFunc) *ServerBuilder {
	s.registerHandler(method, path, handler)
	return s
}

// HandleFunc registers a handler using a standard http.HandlerFunc signature.
// This is useful for operations that don't need typed parameters.
//
// Example:
//
//	srv.HandleFunc(http.MethodGet, "/health", func(w http.ResponseWriter, r *http.Request) {
//		w.WriteHeader(http.StatusOK)
//		w.Write([]byte("OK"))
//	})
func (s *ServerBuilder) HandleFunc(method, path string, handler http.HandlerFunc) *ServerBuilder {
	return s.Handle(method, path, wrapHTTPHandler(handler))
}

// wrapHTTPHandler converts an http.HandlerFunc to a HandlerFunc.
func wrapHTTPHandler(handler http.HandlerFunc) HandlerFunc {
	return func(_ context.Context, req *Request) Response {
		rec := &responseCapture{header: make(http.Header)}
		handler(rec, req.HTTPRequest)
		return &capturedResponse{
			status:  rec.status,
			headers: rec.header,
			body:    rec.body,
		}
	}
}

// Use adds middleware to the server.
// Middleware is applied in order: first added = outermost (executes first on request).
// For example, Use(A, B) results in: A(B(handler)), so A runs first.
//
// Example:
//
//	srv.Use(loggingMiddleware, corsMiddleware)
func (s *ServerBuilder) Use(mw ...Middleware) *ServerBuilder {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.middleware = append(s.middleware, mw...)
	return s
}

// BuildServer constructs the http.Handler and related artifacts.
// Returns an error if the OAS document is invalid or the router cannot be configured.
//
// Operations without registered handlers will return 501 Not Implemented at runtime.
// To enforce that all operations have handlers, check the handlers map before calling BuildServer.
//
// Example:
//
//	result, err := srv.BuildServer()
//	if err != nil {
//		log.Fatal(err)
//	}
//	http.ListenAndServe(":8080", result.Handler)
func (s *ServerBuilder) BuildServer() (*ServerResult, error) {
	// Build the OAS document
	doc, err := s.buildDocument()
	if err != nil {
		return nil, fmt.Errorf("builder: failed to build OAS document: %w", err)
	}

	// Create ParseResult for httpvalidator
	parseResult := s.createParseResult(doc)

	// Create validator if enabled
	var validator *httpvalidator.Validator
	if s.config.enableValidation {
		validator, err = httpvalidator.New(parseResult)
		if err != nil {
			return nil, fmt.Errorf("builder: failed to create validator: %w", err)
		}
	}

	// Build route table
	routes := s.buildRoutes()

	// Create dispatcher
	dispatcher := s.buildDispatcher(routes, validator)

	// Build router
	handler, err := s.router.Build(routes, dispatcher)
	if err != nil {
		return nil, fmt.Errorf("builder: failed to build router: %w", err)
	}

	// Add validation middleware if enabled
	if s.config.enableValidation && validator != nil {
		validationMW := validationMiddleware(validator, s.config.validationConfig)
		handler = validationMW(handler)
	}

	// Add recovery middleware if enabled
	if s.config.enableRecovery {
		handler = recoveryMiddleware(s.errorHandler)(handler)
	}

	// Add logging middleware if enabled
	if s.config.requestLogger != nil {
		handler = loggingMiddleware(s.config.requestLogger)(handler)
	}

	// Apply user middleware (in reverse order so first added is outermost)
	for i := len(s.middleware) - 1; i >= 0; i-- {
		handler = s.middleware[i](handler)
	}

	return &ServerResult{
		Handler:     handler,
		Spec:        doc,
		ParseResult: parseResult,
		Validator:   validator,
	}, nil
}

// MustBuildServer is like BuildServer but panics on error.
// Useful for main() or init() where errors are fatal.
func (s *ServerBuilder) MustBuildServer() *ServerResult {
	result, err := s.BuildServer()
	if err != nil {
		panic(err)
	}
	return result
}

// buildDocument builds the OAS document from the builder state.
func (s *ServerBuilder) buildDocument() (any, error) {
	if s.version == parser.OASVersion20 {
		return s.BuildOAS2()
	}
	return s.BuildOAS3()
}

// createParseResult creates a ParseResult for compatibility with httpvalidator.
func (s *ServerBuilder) createParseResult(doc any) *parser.ParseResult {
	return &parser.ParseResult{
		SourcePath:   "builder",
		SourceFormat: parser.SourceFormatYAML,
		Version:      s.version.String(),
		OASVersion:   s.version,
		Document:     doc,
		Errors:       make([]error, 0),
		Warnings:     make([]string, 0),
	}
}

// buildRoutes builds the route table from the builder's path definitions.
func (s *ServerBuilder) buildRoutes() []operationRoute {
	routes := make([]operationRoute, 0)

	for path, pathItem := range s.paths {
		routes = append(routes, s.routesFromPathItem(path, pathItem)...)
	}

	return routes
}

// routesFromPathItem extracts routes from a PathItem.
func (s *ServerBuilder) routesFromPathItem(path string, pathItem *parser.PathItem) []operationRoute {
	routes := make([]operationRoute, 0)

	methodOps := []struct {
		method string
		op     *parser.Operation
	}{
		{http.MethodGet, pathItem.Get},
		{http.MethodPost, pathItem.Post},
		{http.MethodPut, pathItem.Put},
		{http.MethodDelete, pathItem.Delete},
		{http.MethodPatch, pathItem.Patch},
		{http.MethodHead, pathItem.Head},
		{http.MethodOptions, pathItem.Options},
		{http.MethodTrace, pathItem.Trace},
	}

	for _, mo := range methodOps {
		if mo.op != nil {
			var handler HandlerFunc
			s.mu.RLock()
			if methodHandlers, ok := s.handlers[path]; ok {
				handler = methodHandlers[mo.method]
			}
			s.mu.RUnlock()

			routes = append(routes, operationRoute{
				Method:      mo.method,
				Path:        path,
				OperationID: mo.op.OperationID,
				Handler:     handler,
			})
		}
	}

	return routes
}

// responseCapture captures the output of an http.HandlerFunc.
type responseCapture struct {
	header http.Header
	body   []byte
	status int
}

func (r *responseCapture) Header() http.Header {
	return r.header
}

func (r *responseCapture) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return len(b), nil
}

func (r *responseCapture) WriteHeader(statusCode int) {
	r.status = statusCode
}

// capturedResponse wraps a captured response from an http.HandlerFunc.
type capturedResponse struct {
	status  int
	headers http.Header
	body    []byte
}

func (r *capturedResponse) StatusCode() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *capturedResponse) Headers() http.Header {
	return r.headers
}

func (r *capturedResponse) Body() any {
	return r.body
}

func (r *capturedResponse) WriteTo(w http.ResponseWriter) error {
	maps.Copy(w.Header(), r.headers)
	w.WriteHeader(r.StatusCode())
	if len(r.body) > 0 {
		_, err := w.Write(r.body)
		return err
	}
	return nil
}

// recoveryMiddleware wraps the handler with panic recovery.
func recoveryMiddleware(errorHandler ErrorHandler) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					var err error
					switch v := rec.(type) {
					case error:
						err = fmt.Errorf("builder: panic recovered: %w", v)
					case string:
						err = fmt.Errorf("builder: panic recovered: %s", v)
					default:
						err = fmt.Errorf("builder: panic recovered: %v", v)
					}
					if errorHandler != nil {
						errorHandler(w, r, err)
					} else {
						// Don't expose internal error details to clients
						http.Error(w, "internal server error", http.StatusInternalServerError)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
