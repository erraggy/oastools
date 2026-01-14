package validator

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/erraggy/oastools/internal/issues"
	"github.com/erraggy/oastools/internal/severity"
	"github.com/erraggy/oastools/parser"
)

// Severity indicates the severity level of a validation issue
type Severity = severity.Severity

const (
	// SeverityError indicates a spec violation that makes the document invalid
	SeverityError = severity.SeverityError
	// SeverityWarning indicates a best practice violation or recommendation
	SeverityWarning = severity.SeverityWarning
	// SeverityInfo indicates informational messages
	SeverityInfo = severity.SeverityInfo
	// SeverityCritical indicates critical issues
	SeverityCritical = severity.SeverityCritical
)

const (
	// defaultErrorCapacity is the initial capacity for error slices
	defaultErrorCapacity = 10
	// defaultWarningCapacity is the initial capacity for warning slices
	defaultWarningCapacity = 10

	// Resource exhaustion protection
	maxSchemaNestingDepth = 100 // Maximum depth for nested schemas to prevent stack overflow
)

// ValidationError represents a single validation issue
type ValidationError = issues.Issue

// ValidationResult contains the results of validating an OpenAPI specification
type ValidationResult struct {
	// Valid is true if no errors were found (warnings are allowed)
	Valid bool
	// Version is the detected OAS version string
	Version string
	// OASVersion is the enumerated OAS version
	OASVersion parser.OASVersion
	// Errors contains all validation errors
	Errors []ValidationError
	// Warnings contains all validation warnings
	Warnings []ValidationError
	// ErrorCount is the total number of errors
	ErrorCount int
	// WarningCount is the total number of warnings
	WarningCount int
	// LoadTime is the time taken to load the source data
	LoadTime time.Duration
	// SourceSize is the size of the source data in bytes
	SourceSize int64
	// Stats contains statistical information about the document
	Stats parser.DocumentStats
	// Document contains the validated document (*parser.OAS2Document or *parser.OAS3Document).
	// Added to enable ToParseResult() for package chaining.
	Document any
	// SourceFormat is the format of the source file (JSON or YAML)
	SourceFormat parser.SourceFormat
	// SourcePath is the original source path from the parsed document.
	// Used by ToParseResult() to preserve the original path for package chaining.
	SourcePath string
}

// ToParseResult converts the ValidationResult to a ParseResult for use with
// other packages like fixer, converter, joiner, and differ.
// The returned ParseResult has Document populated but Data is nil
// (consumers use Document, not Data).
// Validation errors and warnings are converted to string warnings with
// severity prefixes for programmatic filtering:
// "[error] path: message", "[warning] path: message", etc.
func (r *ValidationResult) ToParseResult() *parser.ParseResult {
	// Convert validation errors/warnings to string warnings with severity prefix
	warnings := make([]string, 0, len(r.Errors)+len(r.Warnings))
	for _, e := range r.Errors {
		warnings = append(warnings, "["+e.Severity.String()+"] "+e.String())
	}
	for _, w := range r.Warnings {
		warnings = append(warnings, "["+w.Severity.String()+"] "+w.String())
	}

	// Use original source path, falling back to "validator" if not set
	sourcePath := r.SourcePath
	if sourcePath == "" {
		sourcePath = "validator"
	}

	return &parser.ParseResult{
		SourcePath:   sourcePath,
		SourceFormat: r.SourceFormat,
		Version:      r.Version,
		OASVersion:   r.OASVersion,
		Document:     r.Document,
		Errors:       make([]error, 0),
		Warnings:     warnings,
		Stats:        r.Stats,
		LoadTime:     r.LoadTime,
		SourceSize:   r.SourceSize,
	}
}

// Validator handles OpenAPI specification validation
type Validator struct {
	// IncludeWarnings determines whether to include best practice warnings
	IncludeWarnings bool
	// StrictMode enables stricter validation beyond the spec requirements
	StrictMode bool
	// ValidateStructure controls whether the parser performs basic structure validation.
	// When true (default), the parser validates required fields and correct types.
	// When false, parsing is more lenient and skips structure validation.
	ValidateStructure bool
	// UserAgent is the User-Agent string used when fetching URLs
	// Defaults to "oastools" if not set
	UserAgent string
	// SourceMap provides source location lookup for validation errors.
	// When set, validation errors will include Line, Column, and File fields.
	SourceMap *parser.SourceMap
}

// New creates a new Validator instance with default settings
func New() *Validator {
	return &Validator{
		IncludeWarnings:   true,
		StrictMode:        false,
		ValidateStructure: true,
	}
}

// ValidateWithOptions validates an OpenAPI specification using functional options.
// This provides a flexible, extensible API that combines input source selection
// and configuration in a single function call.
//
// Example:
//
//	result, err := validator.ValidateWithOptions(
//	    validator.WithFilePath("openapi.yaml"),
//	    validator.WithStrictMode(true),
//	)
func ValidateWithOptions(opts ...Option) (*ValidationResult, error) {
	cfg, err := applyOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("validator: invalid options: %w", err)
	}

	v := &Validator{
		IncludeWarnings:   cfg.includeWarnings,
		StrictMode:        cfg.strictMode,
		ValidateStructure: cfg.validateStructure,
		UserAgent:         cfg.userAgent,
		SourceMap:         cfg.sourceMap,
	}

	// Route to appropriate validation method based on input source
	// Parsed input is checked first as it's the preferred high-performance path
	if cfg.parsed != nil {
		return v.ValidateParsed(*cfg.parsed)
	}
	// cfg.filePath must be non-nil here (validated by applyOptions)
	return v.Validate(*cfg.filePath)
}

// populateIssueLocation looks up the source location for an issue's path
// and populates the Line, Column, and File fields if found.
func (v *Validator) populateIssueLocation(issue *ValidationError) {
	if v.SourceMap == nil {
		return
	}
	// Convert validation path to JSON path format (add $ prefix if needed)
	jsonPath := issue.Path
	if !hasJSONPathPrefix(jsonPath) {
		jsonPath = "$." + issue.Path
	}
	loc := v.SourceMap.Get(jsonPath)
	if loc.IsKnown() {
		issue.Line = loc.Line
		issue.Column = loc.Column
		issue.File = loc.File
	}
}

// hasJSONPathPrefix returns true if the path already has a JSON path prefix.
func hasJSONPathPrefix(path string) bool {
	return len(path) > 0 && path[0] == '$'
}

// addError appends a validation error and populates its source location.
func (v *Validator) addError(result *ValidationResult, path, message string, opts ...func(*ValidationError)) {
	err := ValidationError{
		Path:     path,
		Message:  message,
		Severity: SeverityError,
	}
	for _, opt := range opts {
		opt(&err)
	}
	v.populateIssueLocation(&err)
	result.Errors = append(result.Errors, err)
}

// addWarning appends a validation warning and populates its source location.
func (v *Validator) addWarning(result *ValidationResult, path, message string, opts ...func(*ValidationError)) {
	warn := ValidationError{
		Path:     path,
		Message:  message,
		Severity: SeverityWarning,
	}
	for _, opt := range opts {
		opt(&warn)
	}
	v.populateIssueLocation(&warn)
	result.Warnings = append(result.Warnings, warn)
}

// withField sets the Field on a ValidationError.
func withField(field string) func(*ValidationError) {
	return func(e *ValidationError) { e.Field = field }
}

// withValue sets the Value on a ValidationError.
func withValue(value any) func(*ValidationError) {
	return func(e *ValidationError) { e.Value = value }
}

// withSpecRef sets the SpecRef on a ValidationError.
func withSpecRef(ref string) func(*ValidationError) {
	return func(e *ValidationError) { e.SpecRef = ref }
}

// ValidateParsed validates an already parsed OpenAPI specification
func (v *Validator) ValidateParsed(parseResult parser.ParseResult) (*ValidationResult, error) {
	result := &ValidationResult{
		Version:      parseResult.Version,
		OASVersion:   parseResult.OASVersion,
		Errors:       make([]ValidationError, 0, defaultErrorCapacity),
		Warnings:     make([]ValidationError, 0, defaultWarningCapacity),
		LoadTime:     parseResult.LoadTime,
		SourceSize:   parseResult.SourceSize,
		Stats:        parseResult.Stats,
		Document:     parseResult.Document,
		SourceFormat: parseResult.SourceFormat,
		SourcePath:   parseResult.SourcePath,
	}

	// Add parser errors to validation result
	for _, parseErr := range parseResult.Errors {
		result.Errors = append(result.Errors, ValidationError{
			Path:     "document",
			Message:  parseErr.Error(),
			Severity: SeverityError,
		})
	}

	// Add parser warnings to validation result
	for _, warning := range parseResult.Warnings {
		result.Warnings = append(result.Warnings, ValidationError{
			Path:     "document",
			Message:  warning,
			Severity: SeverityWarning,
		})
	}

	// Perform additional validation based on OAS version
	// Check both version and document type to ensure consistency
	if parseResult.IsOAS2() {
		if doc, ok := parseResult.OAS2Document(); ok {
			v.validateOAS2(doc, result)
		} else {
			return nil, fmt.Errorf("validator: failed to cast document to OAS2Document")
		}
	} else if parseResult.IsOAS3() {
		if doc, ok := parseResult.OAS3Document(); ok {
			v.validateOAS3(doc, result)
		} else {
			return nil, fmt.Errorf("validator: failed to cast document to OAS3Document")
		}
	} else {
		// in reality this should never happen, since the parser's `Parse` would have errored as well
		return nil, fmt.Errorf("validator: unsupported OAS version: %s", parseResult.OASVersion)
	}

	// Calculate counts
	result.ErrorCount = len(result.Errors)
	result.WarningCount = len(result.Warnings)
	result.Valid = result.ErrorCount == 0

	// Filter warnings if not included
	if !v.IncludeWarnings {
		result.Warnings = nil
		result.WarningCount = 0
	}

	return result, nil
}

// Validate validates an OpenAPI specification file
func (v *Validator) Validate(specPath string) (*ValidationResult, error) {
	// Create parser and configure it
	p := parser.New()
	p.ValidateStructure = v.ValidateStructure
	if v.UserAgent != "" {
		p.UserAgent = v.UserAgent
	}

	// Parse the document
	parseResult, err := p.Parse(specPath)
	if err != nil {
		return nil, fmt.Errorf("validator: failed to parse specification: %w", err)
	}

	return v.ValidateParsed(*parseResult)
}

// validateOAS3 performs OAS 3.x specific validation
func (v *Validator) validateOAS3(doc *parser.OAS3Document, result *ValidationResult) {
	version := doc.OpenAPI
	var baseURL string

	// Determine the correct spec URL based on version
	switch doc.OASVersion {
	case parser.OASVersion300:
		baseURL = "https://spec.openapis.org/oas/v3.0.0.html"
	case parser.OASVersion301:
		baseURL = "https://spec.openapis.org/oas/v3.0.1.html"
	case parser.OASVersion302:
		baseURL = "https://spec.openapis.org/oas/v3.0.2.html"
	case parser.OASVersion303:
		baseURL = "https://spec.openapis.org/oas/v3.0.3.html"
	case parser.OASVersion304:
		baseURL = "https://spec.openapis.org/oas/v3.0.4.html"
	case parser.OASVersion310:
		baseURL = "https://spec.openapis.org/oas/v3.1.0.html"
	case parser.OASVersion311:
		baseURL = "https://spec.openapis.org/oas/v3.1.1.html"
	case parser.OASVersion312:
		baseURL = "https://spec.openapis.org/oas/v3.1.2.html"
	case parser.OASVersion320:
		baseURL = "https://spec.openapis.org/oas/v3.2.0.html"
	default:
		baseURL = fmt.Sprintf("https://spec.openapis.org/oas/v%s.html", version)
	}

	// Validate required fields in info object
	v.validateOAS3Info(doc, result, baseURL)

	// Validate servers
	v.validateOAS3Servers(doc, result, baseURL)

	// Validate paths and operations
	v.validateOAS3Paths(doc, result, baseURL)

	// Validate components
	v.validateOAS3Components(doc, result, baseURL)

	// Validate webhooks (OAS 3.1+)
	v.validateOAS3Webhooks(doc, result, baseURL)

	// Validate path parameters match path templates
	v.validateOAS3PathParameterConsistency(doc, result, baseURL)

	// Validate security requirements reference existing schemes
	v.validateOAS3SecurityRequirements(doc, result, baseURL)

	// Validate duplicate operationIds
	v.validateOAS3OperationIds(doc, result, baseURL)

	// Validate all $ref values point to valid components
	v.validateOAS3Refs(doc, result, baseURL)
}

// validateOAS3Info validates the info object in OAS 3.x
func (v *Validator) validateOAS3Info(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	if doc.Info == nil {
		v.addError(result, "info", "Document must have an info object",
			withSpecRef(fmt.Sprintf("%s#info-object", baseURL)),
			withField("info"),
		)
		return
	}
	v.validateInfoObject(doc.Info, result, baseURL, true)
}

// validateOAS3OperationIds validates that operationIds are unique across the document
func (v *Validator) validateOAS3OperationIds(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	operationIds := make(map[string]string) // map of operationId -> path where first seen

	// Check paths
	if doc.Paths != nil {
		for pathPattern, pathItem := range doc.Paths {
			if pathItem == nil {
				continue
			}

			operations := parser.GetOperations(pathItem, doc.OASVersion)

			v.checkDuplicateOperationIds(operations, "paths", pathPattern, operationIds, result, baseURL)
		}
	}

	// Check webhooks (OAS 3.1+)
	for webhookName, pathItem := range doc.Webhooks {
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, doc.OASVersion)

		v.checkDuplicateOperationIds(operations, "webhooks", webhookName, operationIds, result, baseURL)
	}
}

// validateOAS3Servers validates server objects in OAS 3.x
func (v *Validator) validateOAS3Servers(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	for i, server := range doc.Servers {
		path := fmt.Sprintf("servers[%d]", i)

		if server.URL == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "Server must have a url",
				SpecRef:  fmt.Sprintf("%s#server-object", baseURL),
				Severity: SeverityError,
				Field:    "url",
			})
		}

		// Validate server variables
		for varName, varObj := range server.Variables {
			varPath := fmt.Sprintf("%s.variables.%s", path, varName)

			if varObj.Default == "" {
				result.Errors = append(result.Errors, ValidationError{
					Path:     varPath,
					Message:  "Server variable must have a default value",
					SpecRef:  fmt.Sprintf("%s#server-variable-object", baseURL),
					Severity: SeverityError,
					Field:    "default",
				})
			}

			// If enum is specified, default must be in enum
			if len(varObj.Enum) > 0 && !slices.Contains(varObj.Enum, varObj.Default) {
				result.Errors = append(result.Errors, ValidationError{
					Path:     varPath,
					Message:  fmt.Sprintf("Server variable default value '%s' must be one of the enum values", varObj.Default),
					SpecRef:  fmt.Sprintf("%s#server-variable-object", baseURL),
					Severity: SeverityError,
					Field:    "default",
					Value:    varObj.Default,
				})
			}
		}
	}
}

// validateOAS3Paths validates paths in OAS 3.x
func (v *Validator) validateOAS3Paths(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	if doc.Paths == nil {
		return
	}

	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		pathPrefix := fmt.Sprintf("paths.%s", pathPattern)

		// Validate path pattern starts with "/"
		if !strings.HasPrefix(pathPattern, "/") {
			v.addError(result, pathPrefix, "Path must start with '/'",
				withSpecRef(fmt.Sprintf("%s#paths-object", baseURL)),
				withValue(pathPattern),
			)
		}

		// Validate path template is well-formed
		if err := validatePathTemplate(pathPattern); err != nil {
			v.addError(result, pathPrefix, fmt.Sprintf("Invalid path template: %s", err),
				withSpecRef(fmt.Sprintf("%s#paths-object", baseURL)),
				withValue(pathPattern),
			)
		}

		// Warning: trailing slash in path (REST best practice)
		checkTrailingSlash(v, pathPattern, result, baseURL)

		// Validate QUERY method is only used in OAS 3.2+
		if pathItem.Query != nil && doc.OASVersion < parser.OASVersion320 {
			v.addError(result, fmt.Sprintf("%s.query", pathPrefix),
				fmt.Sprintf("QUERY method is only supported in OAS 3.2+, but document is version %s", doc.OASVersion),
				withSpecRef(fmt.Sprintf("%s#path-item-object", baseURL)),
				withField("query"),
			)
		}

		// Validate each operation
		operations := parser.GetOperations(pathItem, doc.OASVersion)

		for method, op := range operations {
			if op == nil {
				continue
			}

			opPath := fmt.Sprintf("%s.%s", pathPrefix, method)
			v.validateOAS3Operation(op, opPath, result, baseURL)

			// Warning: recommend description
			if v.IncludeWarnings && op.Description == "" && op.Summary == "" {
				v.addWarning(result, opPath, "Operation should have a description or summary for better documentation",
					withSpecRef(fmt.Sprintf("%s#operation-object", baseURL)),
					withField("description"),
				)
			}
		}
	}
}

// validateOAS3Operation validates an operation in OAS 3.x
func (v *Validator) validateOAS3Operation(op *parser.Operation, path string, result *ValidationResult, baseURL string) {
	// Validate request body if present
	if op.RequestBody != nil {
		v.validateOAS3RequestBody(op.RequestBody, fmt.Sprintf("%s.requestBody", path), result, baseURL)
	}

	// Validate response status codes
	v.validateResponseStatusCodes(op.Responses, path, result, baseURL)
}

// validateOAS3RequestBody validates a request body in OAS 3.x
func (v *Validator) validateOAS3RequestBody(requestBody *parser.RequestBody, path string, result *ValidationResult, baseURL string) {
	if requestBody == nil {
		return
	}

	// Skip validation if this is a $ref
	if requestBody.Ref != "" {
		return
	}

	// RequestBody must have content
	if len(requestBody.Content) == 0 {
		v.addError(result, path, "RequestBody must have a content object with at least one media type",
			withSpecRef(fmt.Sprintf("%s#request-body-object", baseURL)),
			withField("content"),
		)
		return
	}

	// Validate each media type
	for mediaType, mediaTypeObj := range requestBody.Content {
		mediaTypePath := fmt.Sprintf("%s.content.%s", path, mediaType)

		// Validate media type format
		if !isValidMediaType(mediaType) {
			v.addError(result, mediaTypePath, fmt.Sprintf("Invalid media type: %s", mediaType),
				withSpecRef(fmt.Sprintf("%s#request-body-object", baseURL)),
				withValue(mediaType),
			)
		}

		// Validate that media type has a schema
		if mediaTypeObj != nil && mediaTypeObj.Schema != nil {
			schemaPath := fmt.Sprintf("%s.schema", mediaTypePath)
			v.validateSchema(mediaTypeObj.Schema, schemaPath, result)
		}
	}
}

// validateOAS3Components validates components in OAS 3.x
func (v *Validator) validateOAS3Components(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	if doc.Components == nil {
		return
	}

	// Validate schemas
	for name, schema := range doc.Components.Schemas {
		v.validateSchemaName(name, "components.schemas", result)
		if schema == nil {
			continue
		}
		path := fmt.Sprintf("components.schemas.%s", name)
		v.validateSchema(schema, path, result)
	}

	// Validate responses
	for name, response := range doc.Components.Responses {
		if response == nil {
			continue
		}
		path := fmt.Sprintf("components.responses.%s", name)

		if response.Description == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "Response must have a description",
				SpecRef:  fmt.Sprintf("%s#response-object", baseURL),
				Severity: SeverityError,
				Field:    "description",
			})
		}
	}

	// Validate request bodies
	for name, requestBody := range doc.Components.RequestBodies {
		if requestBody == nil {
			continue
		}
		path := fmt.Sprintf("components.requestBodies.%s", name)
		v.validateOAS3RequestBody(requestBody, path, result, baseURL)
	}

	// Validate parameters
	for name, param := range doc.Components.Parameters {
		if param == nil {
			continue
		}
		path := fmt.Sprintf("components.parameters.%s", name)

		// Parameters must have either schema or content (but not both)
		hasSchema := param.Schema != nil
		hasContent := len(param.Content) > 0

		if !hasSchema && !hasContent {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "Parameter must have either a schema or content",
				SpecRef:  fmt.Sprintf("%s#parameter-object", baseURL),
				Severity: SeverityError,
			})
		}

		if hasSchema && hasContent {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "Parameter must not have both schema and content",
				SpecRef:  fmt.Sprintf("%s#parameter-object", baseURL),
				Severity: SeverityError,
			})
		}

		// Path parameters must have required: true
		if param.In == "path" && !param.Required {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "Path parameters must have required: true",
				SpecRef:  fmt.Sprintf("%s#parameter-object", baseURL),
				Severity: SeverityError,
				Field:    "required",
			})
		}
	}

	// Validate security schemes
	for name, secScheme := range doc.Components.SecuritySchemes {
		if secScheme == nil {
			continue
		}
		path := fmt.Sprintf("components.securitySchemes.%s", name)
		v.validateOAS3SecurityScheme(secScheme, path, result, baseURL)
	}
}

// validateOAS3SecurityScheme validates a security scheme in OAS 3.x
func (v *Validator) validateOAS3SecurityScheme(scheme *parser.SecurityScheme, path string, result *ValidationResult, baseURL string) {
	if scheme.Type == "" {
		result.Errors = append(result.Errors, ValidationError{
			Path:     path,
			Message:  "Security scheme must have a type",
			SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
			Severity: SeverityError,
			Field:    "type",
		})
		return
	}

	switch scheme.Type {
	case "apiKey":
		if scheme.Name == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "API key security scheme must have a name",
				SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
				Severity: SeverityError,
				Field:    "name",
			})
		}
		if scheme.In == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "API key security scheme must specify 'in' (query, header, or cookie)",
				SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
				Severity: SeverityError,
				Field:    "in",
			})
		}
	case "http":
		if scheme.Scheme == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "HTTP security scheme must have a scheme (e.g., 'basic', 'bearer')",
				SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
				Severity: SeverityError,
				Field:    "scheme",
			})
		}
	case "oauth2":
		if scheme.Flows == nil {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "OAuth2 security scheme must have flows",
				SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
				Severity: SeverityError,
				Field:    "flows",
			})
		} else {
			v.validateOAuth2Flows(scheme.Flows, path, result, baseURL)
		}
	case "openIdConnect":
		if scheme.OpenIDConnectURL == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "OpenID Connect security scheme must have openIdConnectUrl",
				SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
				Severity: SeverityError,
				Field:    "openIdConnectUrl",
			})
		}
	}
}

// validateOAuth2Flows validates OAuth2 flows in OAS 3.x
func (v *Validator) validateOAuth2Flows(flows *parser.OAuthFlows, path string, result *ValidationResult, baseURL string) {
	// Validate implicit flow
	if flows.Implicit != nil {
		flowPath := fmt.Sprintf("%s.flows.implicit", path)
		if flows.Implicit.AuthorizationURL == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     flowPath,
				Message:  "Implicit flow must have authorizationUrl",
				SpecRef:  fmt.Sprintf("%s#oauth-flows-object", baseURL),
				Severity: SeverityError,
				Field:    "authorizationUrl",
			})
		} else if !isValidURL(flows.Implicit.AuthorizationURL) {
			result.Errors = append(result.Errors, ValidationError{
				Path:     flowPath,
				Message:  fmt.Sprintf("Invalid URL format for authorizationUrl: %s", flows.Implicit.AuthorizationURL),
				SpecRef:  fmt.Sprintf("%s#oauth-flows-object", baseURL),
				Severity: SeverityError,
				Field:    "authorizationUrl",
				Value:    flows.Implicit.AuthorizationURL,
			})
		}
	}

	// Validate password flow
	if flows.Password != nil {
		flowPath := fmt.Sprintf("%s.flows.password", path)
		if flows.Password.TokenURL == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     flowPath,
				Message:  "Password flow must have tokenUrl",
				SpecRef:  fmt.Sprintf("%s#oauth-flows-object", baseURL),
				Severity: SeverityError,
				Field:    "tokenUrl",
			})
		} else if !isValidURL(flows.Password.TokenURL) {
			result.Errors = append(result.Errors, ValidationError{
				Path:     flowPath,
				Message:  fmt.Sprintf("Invalid URL format for tokenUrl: %s", flows.Password.TokenURL),
				SpecRef:  fmt.Sprintf("%s#oauth-flows-object", baseURL),
				Severity: SeverityError,
				Field:    "tokenUrl",
				Value:    flows.Password.TokenURL,
			})
		}
	}

	// Validate clientCredentials flow
	if flows.ClientCredentials != nil {
		flowPath := fmt.Sprintf("%s.flows.clientCredentials", path)
		if flows.ClientCredentials.TokenURL == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     flowPath,
				Message:  "Client credentials flow must have tokenUrl",
				SpecRef:  fmt.Sprintf("%s#oauth-flows-object", baseURL),
				Severity: SeverityError,
				Field:    "tokenUrl",
			})
		} else if !isValidURL(flows.ClientCredentials.TokenURL) {
			result.Errors = append(result.Errors, ValidationError{
				Path:     flowPath,
				Message:  fmt.Sprintf("Invalid URL format for tokenUrl: %s", flows.ClientCredentials.TokenURL),
				SpecRef:  fmt.Sprintf("%s#oauth-flows-object", baseURL),
				Severity: SeverityError,
				Field:    "tokenUrl",
				Value:    flows.ClientCredentials.TokenURL,
			})
		}
	}

	// Validate authorizationCode flow
	if flows.AuthorizationCode != nil {
		flowPath := fmt.Sprintf("%s.flows.authorizationCode", path)
		if flows.AuthorizationCode.AuthorizationURL == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     flowPath,
				Message:  "Authorization code flow must have authorizationUrl",
				SpecRef:  fmt.Sprintf("%s#oauth-flows-object", baseURL),
				Severity: SeverityError,
				Field:    "authorizationUrl",
			})
		} else if !isValidURL(flows.AuthorizationCode.AuthorizationURL) {
			result.Errors = append(result.Errors, ValidationError{
				Path:     flowPath,
				Message:  fmt.Sprintf("Invalid URL format for authorizationUrl: %s", flows.AuthorizationCode.AuthorizationURL),
				SpecRef:  fmt.Sprintf("%s#oauth-flows-object", baseURL),
				Severity: SeverityError,
				Field:    "authorizationUrl",
				Value:    flows.AuthorizationCode.AuthorizationURL,
			})
		}
		if flows.AuthorizationCode.TokenURL == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     flowPath,
				Message:  "Authorization code flow must have tokenUrl",
				SpecRef:  fmt.Sprintf("%s#oauth-flows-object", baseURL),
				Severity: SeverityError,
				Field:    "tokenUrl",
			})
		} else if !isValidURL(flows.AuthorizationCode.TokenURL) {
			result.Errors = append(result.Errors, ValidationError{
				Path:     flowPath,
				Message:  fmt.Sprintf("Invalid URL format for tokenUrl: %s", flows.AuthorizationCode.TokenURL),
				SpecRef:  fmt.Sprintf("%s#oauth-flows-object", baseURL),
				Severity: SeverityError,
				Field:    "tokenUrl",
				Value:    flows.AuthorizationCode.TokenURL,
			})
		}
	}
}

// validateOAS3Webhooks validates webhooks in OAS 3.1+
func (v *Validator) validateOAS3Webhooks(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	if len(doc.Webhooks) == 0 {
		return
	}

	for webhookName, pathItem := range doc.Webhooks {
		if pathItem == nil {
			continue
		}

		pathPrefix := fmt.Sprintf("webhooks.%s", webhookName)

		// Validate each operation in the webhook
		operations := parser.GetOperations(pathItem, doc.OASVersion)

		for method, op := range operations {
			if op == nil {
				continue
			}

			opPath := fmt.Sprintf("%s.%s", pathPrefix, method)
			v.validateOAS3Operation(op, opPath, result, baseURL)
		}
	}
}

// validateOAS3PathParameterConsistency checks that path parameters match the path template
func (v *Validator) validateOAS3PathParameterConsistency(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	if doc.Paths == nil {
		return
	}

	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		// Extract parameter names from path template
		pathParams := extractPathParameters(pathPattern)

		// Check all operations in this path
		operations := parser.GetOperations(pathItem, doc.OASVersion)

		for method, op := range operations {
			if op == nil {
				continue
			}

			// Collect declared path parameters
			declaredParams := make(map[string]bool)

			// Check path-level parameters
			for i, param := range pathItem.Parameters {
				if param != nil && param.In == "path" {
					declaredParams[param.Name] = true

					// Path parameters must have required: true
					if !param.Required {
						v.addError(result, fmt.Sprintf("paths.%s.parameters[%d]", pathPattern, i),
							"Path parameters must have required: true",
							withSpecRef(fmt.Sprintf("%s#parameter-object", baseURL)),
							withField("required"),
						)
					}
				}
			}

			// Check operation-level parameters
			for i, param := range op.Parameters {
				if param != nil && param.In == "path" {
					declaredParams[param.Name] = true

					// Path parameters must have required: true
					if !param.Required {
						v.addError(result, fmt.Sprintf("paths.%s.%s.parameters[%d]", pathPattern, method, i),
							"Path parameters must have required: true",
							withSpecRef(fmt.Sprintf("%s#parameter-object", baseURL)),
							withField("required"),
						)
					}
				}
			}

			// Verify all path template parameters are declared
			for paramName := range pathParams {
				if !declaredParams[paramName] {
					v.addError(result, fmt.Sprintf("paths.%s.%s", pathPattern, method),
						fmt.Sprintf("Path template references parameter '{%s}' but it is not declared in parameters", paramName),
						withSpecRef(fmt.Sprintf("%s#path-item-object", baseURL)),
						withValue(paramName),
					)
				}
			}

			// Warn about declared path parameters not in template
			for paramName := range declaredParams {
				if !pathParams[paramName] {
					v.addWarning(result, fmt.Sprintf("paths.%s.%s", pathPattern, method),
						fmt.Sprintf("Parameter '%s' is declared as path parameter but not used in path template", paramName),
						withSpecRef(fmt.Sprintf("%s#path-item-object", baseURL)),
						withValue(paramName),
					)
				}
			}
		}
	}
}

// validateOAS3SecurityRequirements validates security requirements reference existing schemes
func (v *Validator) validateOAS3SecurityRequirements(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	// Get available security schemes
	availableSchemes := make(map[string]bool)
	if doc.Components != nil {
		for name := range doc.Components.SecuritySchemes {
			availableSchemes[name] = true
		}
	}

	// Validate root-level security requirements
	for i, secReq := range doc.Security {
		for schemeName := range secReq {
			if !availableSchemes[schemeName] {
				result.Errors = append(result.Errors, ValidationError{
					Path:     fmt.Sprintf("security[%d].%s", i, schemeName),
					Message:  fmt.Sprintf("Security requirement references undefined security scheme: %s", schemeName),
					SpecRef:  fmt.Sprintf("%s#security-requirement-object", baseURL),
					Severity: SeverityError,
					Value:    schemeName,
				})
			}
		}
	}

	// Validate operation-level security requirements
	if doc.Paths != nil {
		for pathPattern, pathItem := range doc.Paths {
			if pathItem == nil {
				continue
			}

			operations := parser.GetOperations(pathItem, doc.OASVersion)

			for method, op := range operations {
				if op == nil {
					continue
				}

				for i, secReq := range op.Security {
					for schemeName := range secReq {
						if !availableSchemes[schemeName] {
							result.Errors = append(result.Errors, ValidationError{
								Path:     fmt.Sprintf("paths.%s.%s.security[%d].%s", pathPattern, method, i, schemeName),
								Message:  fmt.Sprintf("Security requirement references undefined security scheme: %s", schemeName),
								SpecRef:  fmt.Sprintf("%s#security-requirement-object", baseURL),
								Severity: SeverityError,
								Value:    schemeName,
							})
						}
					}
				}
			}
		}
	}
}
