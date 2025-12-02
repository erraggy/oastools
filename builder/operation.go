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
	formParams  []*formParamBuilder // Form parameters (handled differently in OAS 2.0 vs 3.x)
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

// WithRequestBodyRawSchema sets the request body for the operation with a pre-built schema.
// This is useful when you need full control over the schema structure or when working with
// schemas that cannot be easily represented with Go types (e.g., file uploads, oneOf/anyOf).
//
// Example:
//
//	schema := &parser.Schema{
//		Type: "string",
//		Format: "binary",
//	}
//	WithRequestBodyRawSchema("application/octet-stream", schema)
func WithRequestBodyRawSchema(contentType string, schema *parser.Schema, opts ...RequestBodyOption) OperationOption {
	return func(cfg *operationConfig) {
		rbCfg := &requestBodyConfig{
			required: false, // Default to false
		}
		for _, opt := range opts {
			opt(rbCfg)
		}

		// Create request body with pre-built schema
		cfg.requestBody = &requestBodyBuilder{
			body: &parser.RequestBody{
				Description: rbCfg.description,
				Required:    rbCfg.required,
				Content: map[string]*parser.MediaType{
					contentType: {
						Schema:  schema,
						Example: rbCfg.example,
					},
				},
			},
			bType: nil, // No type reflection needed
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

// WithResponseRawSchema adds a response to the operation with a pre-built schema.
// This is useful when you need full control over the schema structure or when working with
// schemas that cannot be easily represented with Go types (e.g., file downloads, oneOf/anyOf).
//
// Example:
//
//	schema := &parser.Schema{
//		Type: "string",
//		Format: "binary",
//	}
//	WithResponseRawSchema(200, "application/octet-stream", schema,
//		WithResponseDescription("File download"))
func WithResponseRawSchema(statusCode int, contentType string, schema *parser.Schema, opts ...ResponseOption) OperationOption {
	return func(cfg *operationConfig) {
		rCfg := &responseConfig{
			description: fmt.Sprintf("%d response", statusCode),
			contentType: contentType,
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
					contentType: {
						Schema:  schema,
						Example: rCfg.example,
					},
				},
			},
			rType: nil, // No type reflection needed
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

// WithFileParam adds a file upload parameter to the operation.
// This is primarily for OAS 2.0 file uploads using formData parameters with type="file".
// For OAS 3.x, consider using WithRequestBodyRawSchema with a binary/octet-stream schema instead.
//
// Example (OAS 2.0):
//
//	WithFileParam("file", WithParamDescription("File to upload"), WithParamRequired(true))
//
// Example (OAS 3.x alternative):
//
//	schema := &parser.Schema{Type: "string", Format: "binary"}
//	WithRequestBodyRawSchema("multipart/form-data", schema)
func WithFileParam(name string, opts ...ParamOption) OperationOption {
	return func(cfg *operationConfig) {
		pCfg := &paramConfig{}
		for _, opt := range opts {
			opt(pCfg)
		}

		// For OAS 2.0: Create a formData parameter with type="file"
		// For OAS 3.x: This will be handled as a formData parameter, but the builder
		// will need to convert it to a request body with multipart/form-data
		cfg.formParams = append(cfg.formParams, &formParamBuilder{
			name:   name,
			pType:  "file", // Special marker for file type
			config: pCfg,
		})
	}
}

// formParamBuilder tracks form parameter metadata for later processing.
// Form parameters are handled differently in OAS 2.0 vs 3.x:
// - OAS 2.0: parameters with in="formData"
// - OAS 3.x: properties in request body schema with application/x-www-form-urlencoded
type formParamBuilder struct {
	name   string
	pType  any
	config *paramConfig
}

// WithFormParam adds a form parameter to the operation.
// The handling differs based on OAS version:
//   - OAS 2.0: Adds a parameter with in="formData"
//   - OAS 3.x: Adds to request body with content-type application/x-www-form-urlencoded
//
// Form parameters support all standard parameter options including constraints,
// description, required flag, default values, and format specifications.
func WithFormParam(name string, paramType any, opts ...ParamOption) OperationOption {
	return func(cfg *operationConfig) {
		pCfg := &paramConfig{}
		for _, opt := range opts {
			opt(pCfg)
		}

		// Store form parameter metadata for processing in AddOperation
		cfg.formParams = append(cfg.formParams, &formParamBuilder{
			name:   name,
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

	// Process form parameters based on OAS version
	if len(cfg.formParams) > 0 {
		if b.version == parser.OASVersion20 {
			// OAS 2.0: Add form parameters as parameters with in="formData"
			for _, formParam := range cfg.formParams {
				// Validate constraints
				if err := validateParamConstraints(formParam.config); err != nil {
					b.errors = append(b.errors, err)
					continue
				}

				param := &parser.Parameter{
					Name:        formParam.name,
					In:          parser.ParamInFormData,
					Description: formParam.config.description,
					Required:    formParam.config.required,
					Deprecated:  formParam.config.deprecated,
					Example:     formParam.config.example,
				}

				// Handle file type specially for OAS 2.0
				if formParam.pType == "file" {
					// For OAS 2.0, file uploads use type="file"
					param.Type = "file"
					// File parameters don't need schema or constraints
				} else {
					// Generate schema from type
					if formParam.pType != nil {
						param.Schema = b.generateSchema(formParam.pType)
					}

					// Apply constraints directly to parameter for OAS 2.0
					applyParamConstraintsToParam(param, formParam.config)
				}

				parameters = append(parameters, param)
			}
		} else {
			// OAS 3.x: Add form parameters to request body with application/x-www-form-urlencoded
			formSchema := b.buildFormParamSchema(cfg.formParams)
			requestBody = addFormParamsToRequestBody(requestBody, formSchema)
		}
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

// buildFormParamSchema builds a schema for form parameters in OAS 3.x.
// Form parameters are represented as an object schema where each form parameter
// becomes a property. The schema supports all parameter constraints.
func (b *Builder) buildFormParamSchema(formParams []*formParamBuilder) *parser.Schema {
	properties := make(map[string]*parser.Schema)
	var required []string

	for _, formParam := range formParams {
		// Validate constraints
		if err := validateParamConstraints(formParam.config); err != nil {
			b.errors = append(b.errors, err)
			continue
		}

		var propSchema *parser.Schema

		// Handle file type specially for OAS 3.x
		if formParam.pType == "file" {
			// For OAS 3.x, file uploads use type="string" with format="binary"
			propSchema = &parser.Schema{
				Type:   "string",
				Format: "binary",
			}
		} else {
			// Generate schema from type
			propSchema = b.generateSchema(formParam.pType)

			// Apply constraints to the property schema
			propSchema = applyParamConstraintsToSchema(propSchema, formParam.config)
		}

		// Set description if provided
		if formParam.config.description != "" {
			propSchema.Description = formParam.config.description
		}

		// Set deprecated if specified
		if formParam.config.deprecated {
			propSchema.Deprecated = formParam.config.deprecated
		}

		properties[formParam.name] = propSchema

		// Track required fields
		if formParam.config.required {
			required = append(required, formParam.name)
		}
	}

	schema := &parser.Schema{
		Type:       "object",
		Properties: properties,
	}

	if len(required) > 0 {
		schema.Required = required
	}

	return schema
}
