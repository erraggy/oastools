package builder

import (
	"fmt"
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
	handlers     map[string]HandlerFunc
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
//		builder.WithOperationID("listPets"),
//		builder.WithResponse(http.StatusOK, []Pet{}),
//	).Handle("listPets", listPetsHandler)
//
//	result, err := srv.BuildServer()
func NewServerBuilder(version parser.OASVersion, opts ...ServerBuilderOption) *ServerBuilder {
	cfg := defaultServerBuilderConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &ServerBuilder{
		Builder:      New(version),
		handlers:     make(map[string]HandlerFunc),
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
//	srv.Handle("listUsers", listUsersHandler)
func FromBuilder(b *Builder, opts ...ServerBuilderOption) *ServerBuilder {
	cfg := defaultServerBuilderConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &ServerBuilder{
		Builder:      b,
		handlers:     make(map[string]HandlerFunc),
		middleware:   make([]Middleware, 0),
		router:       cfg.router,
		errorHandler: cfg.errorHandler,
		config:       cfg,
	}
}

// Handle registers a handler for an operation by operationID.
// The operation must have been added via AddOperation with an operationID.
//
// Example:
//
//	srv.AddOperation(http.MethodGet, "/pets",
//		builder.WithOperationID("listPets"),
//		builder.WithResponse(http.StatusOK, []Pet{}),
//	)
//	srv.Handle("listPets", func(ctx context.Context, req *builder.Request) builder.Response {
//		return builder.JSON(http.StatusOK, pets)
//	})
func (s *ServerBuilder) Handle(operationID string, handler HandlerFunc) *ServerBuilder {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[operationID] = handler
	return s
}

// HandleFunc registers a handler using a standard http.HandlerFunc signature.
// This is useful for operations that don't need typed parameters.
//
// Example:
//
//	srv.HandleFunc("healthCheck", func(w http.ResponseWriter, r *http.Request) {
//		w.WriteHeader(http.StatusOK)
//		w.Write([]byte("OK"))
//	})
func (s *ServerBuilder) HandleFunc(operationID string, handler http.HandlerFunc) *ServerBuilder {
	return s.Handle(operationID, func(_ context.Context, req *Request) Response {
		// Create a response recorder to capture the http.HandlerFunc output
		rec := &responseCapture{header: make(http.Header)}
		handler(rec, req.HTTPRequest)
		return &capturedResponse{
			status:  rec.status,
			headers: rec.header,
			body:    rec.body,
		}
	})
}

// Use adds middleware to the server.
// Middleware is applied in order: first added = outermost.
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
// Returns an error if required handlers are missing or the spec is invalid.
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
	parseResult, err := s.createParseResult(doc)
	if err != nil {
		return nil, fmt.Errorf("builder: failed to create parse result: %w", err)
	}

	// Create validator if enabled
	var validator *httpvalidator.Validator
	if s.config.enableValidation {
		validator, err = httpvalidator.New(parseResult)
		if err != nil {
			return nil, fmt.Errorf("builder: failed to create validator: %w", err)
		}
	}

	// Build route table
	routes, err := s.buildRoutes()
	if err != nil {
		return nil, fmt.Errorf("builder: failed to build routes: %w", err)
	}

	// Create dispatcher
	dispatcher := s.buildDispatcher(routes, validator)

	// Build router
	handler := s.router.Build(routes, dispatcher)

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
func (s *ServerBuilder) createParseResult(doc any) (*parser.ParseResult, error) {
	return &parser.ParseResult{
		SourcePath:   "builder",
		SourceFormat: parser.SourceFormatYAML,
		Version:      s.version.String(),
		OASVersion:   s.version,
		Document:     doc,
		Errors:       make([]error, 0),
		Warnings:     make([]string, 0),
	}, nil
}

// buildRoutes builds the route table from the builder's path definitions.
func (s *ServerBuilder) buildRoutes() ([]operationRoute, error) {
	routes := make([]operationRoute, 0)

	for path, pathItem := range s.paths {
		routes = append(routes, s.routesFromPathItem(path, pathItem)...)
	}

	return routes, nil
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
			if mo.op.OperationID != "" {
				s.mu.RLock()
				handler = s.handlers[mo.op.OperationID]
				s.mu.RUnlock()
			}

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
	for k, v := range r.headers {
		w.Header()[k] = v
	}
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
						err = v
					case string:
						err = fmt.Errorf("%s", v)
					default:
						err = fmt.Errorf("panic: %v", v)
					}
					if errorHandler != nil {
						errorHandler(w, r, err)
					} else {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
