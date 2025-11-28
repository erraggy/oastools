package builder

import (
	"fmt"
	"strconv"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/parser"
)

// operationConfig holds the configuration for building an operation.
type operationConfig struct {
	operationID string
	summary     string
	description string
	tags        []string
	deprecated  bool
	parameters  []*parser.Parameter
	requestBody *parser.RequestBody
	responses   map[string]*parser.Response
	security    []parser.SecurityRequirement
	noSecurity  bool
}

// OperationOption configures an operation.
type OperationOption func(*operationConfig)

// WithOperationID sets the operation ID.
func WithOperationID(id string) OperationOption {
	return func(cfg *operationConfig) {
		cfg.operationID = id
	}
}

// WithSummary sets the operation summary.
func WithSummary(summary string) OperationOption {
	return func(cfg *operationConfig) {
		cfg.summary = summary
	}
}

// WithDescription sets the operation description.
func WithDescription(desc string) OperationOption {
	return func(cfg *operationConfig) {
		cfg.description = desc
	}
}

// WithTags sets the operation tags.
func WithTags(tags ...string) OperationOption {
	return func(cfg *operationConfig) {
		cfg.tags = tags
	}
}

// WithDeprecated marks the operation as deprecated.
func WithDeprecated(deprecated bool) OperationOption {
	return func(cfg *operationConfig) {
		cfg.deprecated = deprecated
	}
}

// WithParameter adds a pre-built parameter to the operation.
func WithParameter(param *parser.Parameter) OperationOption {
	return func(cfg *operationConfig) {
		cfg.parameters = append(cfg.parameters, param)
	}
}

// WithSecurity sets the security requirements for the operation.
func WithSecurity(requirements ...parser.SecurityRequirement) OperationOption {
	return func(cfg *operationConfig) {
		cfg.security = requirements
	}
}

// WithNoSecurity explicitly marks the operation as requiring no security.
func WithNoSecurity() OperationOption {
	return func(cfg *operationConfig) {
		cfg.noSecurity = true
	}
}

// requestBodyConfig holds configuration for request body building.
type requestBodyConfig struct {
	description string
	required    bool
	example     any
}

// RequestBodyOption configures a request body.
type RequestBodyOption func(*requestBodyConfig)

// WithRequired sets whether the request body is required.
func WithRequired(required bool) RequestBodyOption {
	return func(cfg *requestBodyConfig) {
		cfg.required = required
	}
}

// WithRequestDescription sets the request body description.
func WithRequestDescription(desc string) RequestBodyOption {
	return func(cfg *requestBodyConfig) {
		cfg.description = desc
	}
}

// WithRequestExample sets the request body example.
func WithRequestExample(example any) RequestBodyOption {
	return func(cfg *requestBodyConfig) {
		cfg.example = example
	}
}

// responseConfig holds configuration for response building.
type responseConfig struct {
	description string
	example     any
	headers     map[string]*parser.Header
}

// ResponseOption configures a response.
type ResponseOption func(*responseConfig)

// WithResponseDescription sets the response description.
func WithResponseDescription(desc string) ResponseOption {
	return func(cfg *responseConfig) {
		cfg.description = desc
	}
}

// WithResponseExample sets the response example.
func WithResponseExample(example any) ResponseOption {
	return func(cfg *responseConfig) {
		cfg.example = example
	}
}

// WithResponseHeader adds a header to the response.
func WithResponseHeader(name string, header *parser.Header) ResponseOption {
	return func(cfg *responseConfig) {
		if cfg.headers == nil {
			cfg.headers = make(map[string]*parser.Header)
		}
		cfg.headers[name] = header
	}
}

// paramConfig holds configuration for parameter building.
type paramConfig struct {
	description string
	required    bool
	deprecated  bool
	example     any
}

// ParamOption configures a parameter.
type ParamOption func(*paramConfig)

// WithParamDescription sets the parameter description.
func WithParamDescription(desc string) ParamOption {
	return func(cfg *paramConfig) {
		cfg.description = desc
	}
}

// WithParamRequired sets whether the parameter is required.
func WithParamRequired(required bool) ParamOption {
	return func(cfg *paramConfig) {
		cfg.required = required
	}
}

// WithParamExample sets the parameter example.
func WithParamExample(example any) ParamOption {
	return func(cfg *paramConfig) {
		cfg.example = example
	}
}

// WithParamDeprecated marks the parameter as deprecated.
func WithParamDeprecated(deprecated bool) ParamOption {
	return func(cfg *paramConfig) {
		cfg.deprecated = deprecated
	}
}

// WithRequestBody sets the request body for the operation.
// The bodyType is reflected to generate the schema.
func WithRequestBody(contentType string, bodyType any, opts ...RequestBodyOption) OperationOption {
	return func(cfg *operationConfig) {
		rbCfg := &requestBodyConfig{
			required: false, // Default to false
		}
		for _, opt := range opts {
			opt(rbCfg)
		}

		// We'll need to generate the schema in the context of the builder
		// This is a marker that will be processed by AddOperation
		cfg.requestBody = &parser.RequestBody{
			Description: rbCfg.description,
			Required:    rbCfg.required,
			Content: map[string]*parser.MediaType{
				contentType: {
					// Schema will be populated by AddOperation
					Example: rbCfg.example,
				},
			},
		}
		// Store the body type for later schema generation
		cfg.requestBody.Extra = map[string]any{
			"_bodyType": bodyType,
		}
	}
}

// WithResponse adds a response to the operation.
// The responseType is reflected to generate the schema.
func WithResponse(statusCode int, responseType any, opts ...ResponseOption) OperationOption {
	return func(cfg *operationConfig) {
		rCfg := &responseConfig{
			description: fmt.Sprintf("%d response", statusCode),
		}
		for _, opt := range opts {
			opt(rCfg)
		}

		if cfg.responses == nil {
			cfg.responses = make(map[string]*parser.Response)
		}

		code := strconv.Itoa(statusCode)
		cfg.responses[code] = &parser.Response{
			Description: rCfg.description,
			Headers:     rCfg.headers,
			Content: map[string]*parser.MediaType{
				"application/json": {
					// Schema will be populated by AddOperation
					Example: rCfg.example,
				},
			},
			Extra: map[string]any{
				"_responseType": responseType,
			},
		}
	}
}

// WithResponseRef adds a response reference to the operation.
func WithResponseRef(statusCode int, ref string) OperationOption {
	return func(cfg *operationConfig) {
		if cfg.responses == nil {
			cfg.responses = make(map[string]*parser.Response)
		}

		code := strconv.Itoa(statusCode)
		cfg.responses[code] = &parser.Response{
			Ref: ref,
		}
	}
}

// WithDefaultResponse sets the default response for the operation.
func WithDefaultResponse(responseType any, opts ...ResponseOption) OperationOption {
	return func(cfg *operationConfig) {
		rCfg := &responseConfig{
			description: "Default response",
		}
		for _, opt := range opts {
			opt(rCfg)
		}

		if cfg.responses == nil {
			cfg.responses = make(map[string]*parser.Response)
		}

		cfg.responses["default"] = &parser.Response{
			Description: rCfg.description,
			Headers:     rCfg.headers,
			Content: map[string]*parser.MediaType{
				"application/json": {
					Example: rCfg.example,
				},
			},
			Extra: map[string]any{
				"_responseType": responseType,
			},
		}
	}
}

// WithQueryParam adds a query parameter to the operation.
func WithQueryParam(name string, paramType any, opts ...ParamOption) OperationOption {
	return func(cfg *operationConfig) {
		pCfg := &paramConfig{}
		for _, opt := range opts {
			opt(pCfg)
		}

		param := &parser.Parameter{
			Name:        name,
			In:          parser.ParamInQuery,
			Description: pCfg.description,
			Required:    pCfg.required,
			Deprecated:  pCfg.deprecated,
			Example:     pCfg.example,
			Extra: map[string]any{
				"_paramType": paramType,
			},
		}
		cfg.parameters = append(cfg.parameters, param)
	}
}

// WithPathParam adds a path parameter to the operation.
// Note: Path parameters are always required per the OAS spec.
func WithPathParam(name string, paramType any, opts ...ParamOption) OperationOption {
	return func(cfg *operationConfig) {
		pCfg := &paramConfig{
			required: true, // Path parameters are always required
		}
		for _, opt := range opts {
			opt(pCfg)
		}

		param := &parser.Parameter{
			Name:        name,
			In:          parser.ParamInPath,
			Description: pCfg.description,
			Required:    true, // Always required for path params
			Deprecated:  pCfg.deprecated,
			Example:     pCfg.example,
			Extra: map[string]any{
				"_paramType": paramType,
			},
		}
		cfg.parameters = append(cfg.parameters, param)
	}
}

// WithHeaderParam adds a header parameter to the operation.
func WithHeaderParam(name string, paramType any, opts ...ParamOption) OperationOption {
	return func(cfg *operationConfig) {
		pCfg := &paramConfig{}
		for _, opt := range opts {
			opt(pCfg)
		}

		param := &parser.Parameter{
			Name:        name,
			In:          parser.ParamInHeader,
			Description: pCfg.description,
			Required:    pCfg.required,
			Deprecated:  pCfg.deprecated,
			Example:     pCfg.example,
			Extra: map[string]any{
				"_paramType": paramType,
			},
		}
		cfg.parameters = append(cfg.parameters, param)
	}
}

// WithCookieParam adds a cookie parameter to the operation.
func WithCookieParam(name string, paramType any, opts ...ParamOption) OperationOption {
	return func(cfg *operationConfig) {
		pCfg := &paramConfig{}
		for _, opt := range opts {
			opt(pCfg)
		}

		param := &parser.Parameter{
			Name:        name,
			In:          parser.ParamInCookie,
			Description: pCfg.description,
			Required:    pCfg.required,
			Deprecated:  pCfg.deprecated,
			Example:     pCfg.example,
			Extra: map[string]any{
				"_paramType": paramType,
			},
		}
		cfg.parameters = append(cfg.parameters, param)
	}
}

// AddOperation adds an API operation to the specification.
// Go types passed to options are automatically converted to schemas via reflection.
func (b *Builder) AddOperation(method, path string, opts ...OperationOption) *Builder {
	// Create operation config with defaults
	cfg := &operationConfig{
		responses: make(map[string]*parser.Response),
	}

	// Apply all options
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate operation ID uniqueness
	if cfg.operationID != "" {
		if b.operationIDs[cfg.operationID] {
			b.errors = append(b.errors, fmt.Errorf("duplicate operation ID: %s", cfg.operationID))
		}
		b.operationIDs[cfg.operationID] = true
	}

	// Process request body schema
	if cfg.requestBody != nil && cfg.requestBody.Extra != nil {
		if bodyType, ok := cfg.requestBody.Extra["_bodyType"]; ok {
			schema := b.generateSchema(bodyType)
			for contentType := range cfg.requestBody.Content {
				cfg.requestBody.Content[contentType].Schema = schema
			}
		}
		// Clear the extra field to avoid serialization issues
		cfg.requestBody.Extra = nil
	}

	// Process response schemas
	for code, resp := range cfg.responses {
		if resp.Extra != nil {
			if respType, ok := resp.Extra["_responseType"]; ok {
				schema := b.generateSchema(respType)
				for contentType := range resp.Content {
					resp.Content[contentType].Schema = schema
				}
			}
			// Clear the extra field
			resp.Extra = nil
		}
		cfg.responses[code] = resp
	}

	// Process parameter schemas
	for _, param := range cfg.parameters {
		if param.Extra != nil {
			if paramType, ok := param.Extra["_paramType"]; ok {
				param.Schema = b.generateSchema(paramType)
			}
			// Clear the extra field
			param.Extra = nil
		}
	}

	// Build responses object
	var responses *parser.Responses
	if len(cfg.responses) > 0 {
		responses = &parser.Responses{
			Codes: make(map[string]*parser.Response),
		}
		for code, resp := range cfg.responses {
			if code == "default" {
				responses.Default = resp
			} else {
				responses.Codes[code] = resp
			}
		}
	}

	// Build Operation struct
	var security []parser.SecurityRequirement
	if cfg.noSecurity {
		// Explicitly empty security requirement
		security = []parser.SecurityRequirement{{}}
	} else if len(cfg.security) > 0 {
		security = cfg.security
	}

	op := &parser.Operation{
		OperationID: cfg.operationID,
		Summary:     cfg.summary,
		Description: cfg.description,
		Tags:        cfg.tags,
		Parameters:  cfg.parameters,
		RequestBody: cfg.requestBody,
		Responses:   responses,
		Security:    security,
		Deprecated:  cfg.deprecated,
	}

	// Get or create PathItem
	pathItem := b.getOrCreatePathItem(path)

	// Assign operation to method
	b.setOperation(pathItem, method, op)

	return b
}

// setOperation assigns an operation to a path item based on HTTP method.
func (b *Builder) setOperation(pathItem *parser.PathItem, method string, op *parser.Operation) {
	switch method {
	case httputil.MethodGet, "GET":
		pathItem.Get = op
	case httputil.MethodPut, "PUT":
		pathItem.Put = op
	case httputil.MethodPost, "POST":
		pathItem.Post = op
	case httputil.MethodDelete, "DELETE":
		pathItem.Delete = op
	case httputil.MethodOptions, "OPTIONS":
		pathItem.Options = op
	case httputil.MethodHead, "HEAD":
		pathItem.Head = op
	case httputil.MethodPatch, "PATCH":
		pathItem.Patch = op
	case httputil.MethodTrace, "TRACE":
		pathItem.Trace = op
	default:
		b.errors = append(b.errors, fmt.Errorf("unsupported HTTP method: %s", method))
	}
}
