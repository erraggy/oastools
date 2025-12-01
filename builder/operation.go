package builder

import (
	"fmt"
	"strconv"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/parser"
)

// parameterBuilder wraps a parameter with builder-specific metadata.
// This allows us to store type information and configuration without
// polluting the parser.Parameter.Extra map with non-extension fields.
type parameterBuilder struct {
	param  *parser.Parameter
	pType  any
	config *paramConfig
}

// requestBodyBuilder wraps a request body with builder-specific metadata.
type requestBodyBuilder struct {
	body  *parser.RequestBody
	bType any
}

// responseBuilder wraps a response with builder-specific metadata.
type responseBuilder struct {
	response *parser.Response
	rType    any
}

// operationConfig holds the configuration for building an operation.
type operationConfig struct {
	operationID string
	summary     string
	description string
	tags        []string
	deprecated  bool
	parameters  []*parameterBuilder
	requestBody *requestBodyBuilder
	responses   map[string]*responseBuilder
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
		cfg.parameters = append(cfg.parameters, &parameterBuilder{
			param: param,
		})
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

		// Wrap the request body with builder metadata
		cfg.requestBody = &requestBodyBuilder{
			body: &parser.RequestBody{
				Description: rbCfg.description,
				Required:    rbCfg.required,
				Content: map[string]*parser.MediaType{
					contentType: {
						// Schema will be populated by AddOperation
						Example: rbCfg.example,
					},
				},
			},
			bType: bodyType,
		}
	}
}

// WithResponse adds a response to the operation.
// The responseType is reflected to generate the schema.
// Use WithResponseContentType to specify a content type other than "application/json".
func WithResponse(statusCode int, responseType any, opts ...ResponseOption) OperationOption {
	return func(cfg *operationConfig) {
		rCfg := &responseConfig{
			description: fmt.Sprintf("%d response", statusCode),
			contentType: "application/json", // Default content type
		}
		for _, opt := range opts {
			opt(rCfg)
		}

		if cfg.responses == nil {
			cfg.responses = make(map[string]*responseBuilder)
		}

		code := strconv.Itoa(statusCode)
		cfg.responses[code] = &responseBuilder{
			response: &parser.Response{
				Description: rCfg.description,
				Headers:     rCfg.headers,
				Content: map[string]*parser.MediaType{
					rCfg.contentType: {
						// Schema will be populated by AddOperation
						Example: rCfg.example,
					},
				},
			},
			rType: responseType,
		}
	}
}

// WithResponseRef adds a response reference to the operation.
func WithResponseRef(statusCode int, ref string) OperationOption {
	return func(cfg *operationConfig) {
		if cfg.responses == nil {
			cfg.responses = make(map[string]*responseBuilder)
		}

		code := strconv.Itoa(statusCode)
		cfg.responses[code] = &responseBuilder{
			response: &parser.Response{
				Ref: ref,
			},
		}
	}
}

// WithDefaultResponse sets the default response for the operation.
// Use WithResponseContentType to specify a content type other than "application/json".
func WithDefaultResponse(responseType any, opts ...ResponseOption) OperationOption {
	return func(cfg *operationConfig) {
		rCfg := &responseConfig{
			description: "Default response",
			contentType: "application/json", // Default content type
		}
		for _, opt := range opts {
			opt(rCfg)
		}

		if cfg.responses == nil {
			cfg.responses = make(map[string]*responseBuilder)
		}

		cfg.responses["default"] = &responseBuilder{
			response: &parser.Response{
				Description: rCfg.description,
				Headers:     rCfg.headers,
				Content: map[string]*parser.MediaType{
					rCfg.contentType: {
						Example: rCfg.example,
					},
				},
			},
			rType: responseType,
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

		cfg.parameters = append(cfg.parameters, &parameterBuilder{
			param: &parser.Parameter{
				Name:        name,
				In:          parser.ParamInQuery,
				Description: pCfg.description,
				Required:    pCfg.required,
				Deprecated:  pCfg.deprecated,
				Example:     pCfg.example,
			},
			pType:  paramType,
			config: pCfg,
		})
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

		cfg.parameters = append(cfg.parameters, &parameterBuilder{
			param: &parser.Parameter{
				Name:        name,
				In:          parser.ParamInPath,
				Description: pCfg.description,
				Required:    true, // Always required for path params
				Deprecated:  pCfg.deprecated,
				Example:     pCfg.example,
			},
			pType:  paramType,
			config: pCfg,
		})
	}
}

// WithHeaderParam adds a header parameter to the operation.
func WithHeaderParam(name string, paramType any, opts ...ParamOption) OperationOption {
	return func(cfg *operationConfig) {
		pCfg := &paramConfig{}
		for _, opt := range opts {
			opt(pCfg)
		}

		cfg.parameters = append(cfg.parameters, &parameterBuilder{
			param: &parser.Parameter{
				Name:        name,
				In:          parser.ParamInHeader,
				Description: pCfg.description,
				Required:    pCfg.required,
				Deprecated:  pCfg.deprecated,
				Example:     pCfg.example,
			},
			pType:  paramType,
			config: pCfg,
		})
	}
}

// WithCookieParam adds a cookie parameter to the operation.
func WithCookieParam(name string, paramType any, opts ...ParamOption) OperationOption {
	return func(cfg *operationConfig) {
		pCfg := &paramConfig{}
		for _, opt := range opts {
			opt(pCfg)
		}

		cfg.parameters = append(cfg.parameters, &parameterBuilder{
			param: &parser.Parameter{
				Name:        name,
				In:          parser.ParamInCookie,
				Description: pCfg.description,
				Required:    pCfg.required,
				Deprecated:  pCfg.deprecated,
				Example:     pCfg.example,
			},
			pType:  paramType,
			config: pCfg,
		})
	}
}

// AddOperation adds an API operation to the specification.
// Go types passed to options are automatically converted to schemas via reflection.
//
// Note: OpenAPI requires at least one response per operation. If no responses
// are defined, the resulting spec will fail OAS validation. Use WithResponse()
// or WithDefaultResponse() to add responses. The builder package does not perform
// OAS specification validation; use the validator package to validate built documents.
func (b *Builder) AddOperation(method, path string, opts ...OperationOption) *Builder {
	// Create operation config with defaults
	cfg := &operationConfig{
		responses: make(map[string]*responseBuilder),
	}

	// Apply all options
	for _, opt := range opts {
		opt(cfg)
	}

	// Check for duplicate operation ID
	if cfg.operationID != "" {
		if b.operationIDs[cfg.operationID] {
			b.errors = append(b.errors, fmt.Errorf("duplicate operation ID: %s", cfg.operationID))
		}
		b.operationIDs[cfg.operationID] = true
	}

	// Unwrap and process request body
	var requestBody *parser.RequestBody
	if cfg.requestBody != nil {
		requestBody = cfg.requestBody.body
		if cfg.requestBody.bType != nil {
			schema := b.generateSchema(cfg.requestBody.bType)
			for contentType := range requestBody.Content {
				requestBody.Content[contentType].Schema = schema
			}
		}
	}

	// Unwrap and process responses
	responseMap := make(map[string]*parser.Response)
	for code, respBuilder := range cfg.responses {
		resp := respBuilder.response
		if respBuilder.rType != nil {
			schema := b.generateSchema(respBuilder.rType)
			for contentType := range resp.Content {
				resp.Content[contentType].Schema = schema
			}
		}
		responseMap[code] = resp
	}

	// Unwrap and process parameters
	var parameters []*parser.Parameter
	for _, paramBuilder := range cfg.parameters {
		param := paramBuilder.param

		// Generate schema if type is provided
		if paramBuilder.pType != nil {
			param.Schema = b.generateSchema(paramBuilder.pType)
		}

		// Apply constraints from config if present
		if paramBuilder.config != nil {
			pCfg := paramBuilder.config
			// Validate constraints
			if err := validateParamConstraints(pCfg); err != nil {
				b.errors = append(b.errors, err)
				continue
			}
			if b.version != parser.OASVersion20 {
				// OAS 3.x: Apply constraints to schema
				param.Schema = applyParamConstraintsToSchema(param.Schema, pCfg)
			} else {
				// OAS 2.0: Apply constraints directly to parameter
				applyParamConstraintsToParam(param, pCfg)
			}
		}

		parameters = append(parameters, param)
	}

	// Build responses object
	var responses *parser.Responses
	if len(responseMap) > 0 {
		responses = &parser.Responses{
			Codes: make(map[string]*parser.Response),
		}
		for code, resp := range responseMap {
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
		Parameters:  parameters,
		RequestBody: requestBody,
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
