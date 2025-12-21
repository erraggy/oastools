package validator

import (
	"fmt"
	"mime"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/erraggy/oastools/internal/httputil"
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
}

// Validator handles OpenAPI specification validation
type Validator struct {
	// IncludeWarnings determines whether to include best practice warnings
	IncludeWarnings bool
	// StrictMode enables stricter validation beyond the spec requirements
	StrictMode bool
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
		IncludeWarnings: true,
		StrictMode:      false,
	}
}

// Option is a function that configures a validation operation
type Option func(*validateConfig) error

// validateConfig holds configuration for a validation operation
type validateConfig struct {
	// Input source (exactly one must be set)
	filePath *string
	parsed   *parser.ParseResult

	// Configuration options
	includeWarnings bool
	strictMode      bool
	userAgent       string
	sourceMap       *parser.SourceMap
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
		IncludeWarnings: cfg.includeWarnings,
		StrictMode:      cfg.strictMode,
		UserAgent:       cfg.userAgent,
		SourceMap:       cfg.sourceMap,
	}

	// Route to appropriate validation method based on input source
	// Parsed input is checked first as it's the preferred high-performance path
	if cfg.parsed != nil {
		return v.ValidateParsed(*cfg.parsed)
	}
	// cfg.filePath must be non-nil here (validated by applyOptions)
	return v.Validate(*cfg.filePath)
}

// applyOptions applies option functions and validates configuration
func applyOptions(opts ...Option) (*validateConfig, error) {
	cfg := &validateConfig{
		// Set defaults to match existing behavior
		includeWarnings: true,
		strictMode:      false,
		userAgent:       "",
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Validate exactly one input source is specified
	sourceCount := 0
	if cfg.filePath != nil {
		sourceCount++
	}
	if cfg.parsed != nil {
		sourceCount++
	}

	if sourceCount == 0 {
		return nil, fmt.Errorf("must specify an input source (use WithFilePath or WithParsed)")
	}
	if sourceCount > 1 {
		return nil, fmt.Errorf("must specify exactly one input source")
	}

	return cfg, nil
}

// WithFilePath specifies a file path or URL as the input source
func WithFilePath(path string) Option {
	return func(cfg *validateConfig) error {
		cfg.filePath = &path
		return nil
	}
}

// WithParsed specifies a parsed ParseResult as the input source
func WithParsed(result parser.ParseResult) Option {
	return func(cfg *validateConfig) error {
		cfg.parsed = &result
		return nil
	}
}

// WithIncludeWarnings enables or disables best practice warnings
// Default: true
func WithIncludeWarnings(enabled bool) Option {
	return func(cfg *validateConfig) error {
		cfg.includeWarnings = enabled
		return nil
	}
}

// WithStrictMode enables or disables strict validation beyond spec requirements
// Default: false
func WithStrictMode(enabled bool) Option {
	return func(cfg *validateConfig) error {
		cfg.strictMode = enabled
		return nil
	}
}

// WithUserAgent sets the User-Agent string for HTTP requests
// Default: "" (uses parser default)
func WithUserAgent(ua string) Option {
	return func(cfg *validateConfig) error {
		cfg.userAgent = ua
		return nil
	}
}

// WithSourceMap provides a SourceMap for populating line/column information
// in validation errors. The SourceMap is typically obtained from parsing
// with parser.WithSourceMap(true).
func WithSourceMap(sm *parser.SourceMap) Option {
	return func(cfg *validateConfig) error {
		cfg.sourceMap = sm
		return nil
	}
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
		Version:    parseResult.Version,
		OASVersion: parseResult.OASVersion,
		Errors:     make([]ValidationError, 0, defaultErrorCapacity),
		Warnings:   make([]ValidationError, 0, defaultWarningCapacity),
		LoadTime:   parseResult.LoadTime,
		SourceSize: parseResult.SourceSize,
		Stats:      parseResult.Stats,
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
	// Create parser and set UserAgent if specified
	p := parser.New()
	if v.UserAgent != "" {
		p.UserAgent = v.UserAgent
	}

	// Parse the document
	parseResult, err := p.Parse(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse specification: %w", err)
	}

	return v.ValidateParsed(*parseResult)
}

// validateOAS2 performs OAS 2.0 specific validation
func (v *Validator) validateOAS2(doc *parser.OAS2Document, result *ValidationResult) {
	baseURL := "https://spec.openapis.org/oas/v2.0.html"

	// Validate required fields in info object
	v.validateOAS2Info(doc, result, baseURL)

	// Validate paths and operations
	v.validateOAS2Paths(doc, result, baseURL)

	// Validate definitions (schemas)
	v.validateOAS2Definitions(doc, result, baseURL)

	// Validate parameters
	v.validateOAS2Parameters(doc, result, baseURL)

	// Validate responses
	v.validateOAS2Responses(doc, result, baseURL)

	// Validate security definitions and requirements
	v.validateOAS2Security(doc, result, baseURL)

	// Validate path parameters match path templates
	v.validateOAS2PathParameterConsistency(doc, result, baseURL)

	// Validate duplicate operationIds
	v.validateOAS2OperationIds(doc, result, baseURL)

	// Validate all $ref values point to valid components
	v.validateOAS2Refs(doc, result, baseURL)
}

// validateOAS2Info validates the info object in OAS 2.0
func (v *Validator) validateOAS2Info(doc *parser.OAS2Document, result *ValidationResult, baseURL string) {
	if doc.Info == nil {
		result.Errors = append(result.Errors, ValidationError{
			Path:     "info",
			Message:  "Document must have an info object",
			SpecRef:  fmt.Sprintf("%s#info-object", baseURL),
			Severity: SeverityError,
			Field:    "info",
		})
		return
	}

	if doc.Info.Title == "" {
		result.Errors = append(result.Errors, ValidationError{
			Path:     "info.title",
			Message:  "Info object must have a title",
			SpecRef:  fmt.Sprintf("%s#info-object", baseURL),
			Severity: SeverityError,
			Field:    "title",
		})
	}

	if doc.Info.Version == "" {
		result.Errors = append(result.Errors, ValidationError{
			Path:     "info.version",
			Message:  "Info object must have a version",
			SpecRef:  fmt.Sprintf("%s#info-object", baseURL),
			Severity: SeverityError,
			Field:    "version",
		})
	}

	// Validate contact information if present
	if doc.Info.Contact != nil {
		if doc.Info.Contact.URL != "" && !isValidURL(doc.Info.Contact.URL) {
			result.Errors = append(result.Errors, ValidationError{
				Path:     "info.contact.url",
				Message:  fmt.Sprintf("Invalid URL format: %s", doc.Info.Contact.URL),
				SpecRef:  fmt.Sprintf("%s#contact-object", baseURL),
				Severity: SeverityError,
				Field:    "url",
				Value:    doc.Info.Contact.URL,
			})
		}
		if doc.Info.Contact.Email != "" && !isValidEmail(doc.Info.Contact.Email) {
			result.Errors = append(result.Errors, ValidationError{
				Path:     "info.contact.email",
				Message:  fmt.Sprintf("Invalid email format: %s", doc.Info.Contact.Email),
				SpecRef:  fmt.Sprintf("%s#contact-object", baseURL),
				Severity: SeverityError,
				Field:    "email",
				Value:    doc.Info.Contact.Email,
			})
		}
	}

	// Validate license information if present
	if doc.Info.License != nil {
		if doc.Info.License.URL != "" && !isValidURL(doc.Info.License.URL) {
			result.Errors = append(result.Errors, ValidationError{
				Path:     "info.license.url",
				Message:  fmt.Sprintf("Invalid URL format: %s", doc.Info.License.URL),
				SpecRef:  fmt.Sprintf("%s#license-object", baseURL),
				Severity: SeverityError,
				Field:    "url",
				Value:    doc.Info.License.URL,
			})
		}
	}
}

// validateOAS2OperationIds validates that operationIds are unique across the document
func (v *Validator) validateOAS2OperationIds(doc *parser.OAS2Document, result *ValidationResult, baseURL string) {
	operationIds := make(map[string]string) // map of operationId -> path where first seen

	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, parser.OASVersion20)

		v.checkDuplicateOperationIds(operations, "paths", pathPattern, operationIds, result, baseURL)
	}
}

// validateOAS2Paths validates paths in OAS 2.0
func (v *Validator) validateOAS2Paths(doc *parser.OAS2Document, result *ValidationResult, baseURL string) {
	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		// Validate path pattern starts with "/"
		if !strings.HasPrefix(pathPattern, "/") {
			result.Errors = append(result.Errors, ValidationError{
				Path:     fmt.Sprintf("paths.%s", pathPattern),
				Message:  "Path must start with '/'",
				SpecRef:  fmt.Sprintf("%s#paths-object", baseURL),
				Severity: SeverityError,
				Value:    pathPattern,
			})
		}

		// Validate path template is well-formed
		if err := validatePathTemplate(pathPattern); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Path:     fmt.Sprintf("paths.%s", pathPattern),
				Message:  fmt.Sprintf("Invalid path template: %s", err),
				SpecRef:  fmt.Sprintf("%s#paths-object", baseURL),
				Severity: SeverityError,
				Value:    pathPattern,
			})
		}

		// Warning: trailing slash in path (REST best practice)
		checkTrailingSlash(v, pathPattern, result, baseURL)

		pathPrefix := fmt.Sprintf("paths.%s", pathPattern)

		// Validate QUERY method is not used in OAS 2.0
		if pathItem.Query != nil {
			result.Errors = append(result.Errors, ValidationError{
				Path:     fmt.Sprintf("%s.query", pathPrefix),
				Message:  "QUERY method is only supported in OAS 3.2+, not in OAS 2.0",
				SpecRef:  fmt.Sprintf("%s#path-item-object", baseURL),
				Severity: SeverityError,
				Field:    "query",
			})
		}

		// Validate TRACE method is not used in OAS 2.0
		if pathItem.Trace != nil {
			result.Errors = append(result.Errors, ValidationError{
				Path:     fmt.Sprintf("%s.trace", pathPrefix),
				Message:  "TRACE method is only supported in OAS 3.0+, not in OAS 2.0",
				SpecRef:  fmt.Sprintf("%s#path-item-object", baseURL),
				Severity: SeverityError,
				Field:    "trace",
			})
		}

		// Validate each operation
		operations := parser.GetOperations(pathItem, parser.OASVersion20)

		for method, op := range operations {
			if op == nil {
				continue
			}

			opPath := fmt.Sprintf("%s.%s", pathPrefix, method)
			v.validateOAS2Operation(op, opPath, result, baseURL)

			// Warning: recommend description
			if v.IncludeWarnings && op.Description == "" && op.Summary == "" {
				result.Warnings = append(result.Warnings, ValidationError{
					Path:     opPath,
					Message:  "Operation should have a description or summary for better documentation",
					SpecRef:  fmt.Sprintf("%s#operation-object", baseURL),
					Severity: SeverityWarning,
					Field:    "description",
				})
			}
		}
	}
}

// validateResponseStatusCodes validates HTTP status codes in an operation's responses.
// This helper is shared by both OAS 2.0 and OAS 3.x operation validators.
func (v *Validator) validateResponseStatusCodes(responses *parser.Responses, path string, result *ValidationResult, baseURL string) {
	if responses == nil || responses.Codes == nil {
		return
	}

	hasSuccess := false
	for code := range responses.Codes {
		// Validate HTTP status code format
		if !httputil.ValidateStatusCode(code) {
			result.Errors = append(result.Errors, ValidationError{
				Path:     fmt.Sprintf("%s.responses.%s", path, code),
				Message:  fmt.Sprintf("Invalid HTTP status code: %s", code),
				SpecRef:  fmt.Sprintf("%s#responses-object", baseURL),
				Severity: SeverityError,
				Value:    code,
			})
		} else if v.StrictMode && !httputil.IsStandardStatusCode(code) {
			// In strict mode, warn about non-standard status codes
			result.Warnings = append(result.Warnings, ValidationError{
				Path:     fmt.Sprintf("%s.responses.%s", path, code),
				Message:  fmt.Sprintf("Non-standard HTTP status code: %s (not defined in HTTP RFCs)", code),
				SpecRef:  fmt.Sprintf("%s#responses-object", baseURL),
				Severity: SeverityWarning,
				Value:    code,
			})
		}

		if strings.HasPrefix(code, "2") || code == "default" {
			hasSuccess = true
		}
	}
	if !hasSuccess && v.StrictMode {
		result.Warnings = append(result.Warnings, ValidationError{
			Path:     fmt.Sprintf("%s.responses", path),
			Message:  "Operation should define at least one successful response (2XX or default)",
			SpecRef:  fmt.Sprintf("%s#responses-object", baseURL),
			Severity: SeverityWarning,
		})
	}
}

// validateOAS2Operation validates an operation in OAS 2.0
func (v *Validator) validateOAS2Operation(op *parser.Operation, path string, result *ValidationResult, baseURL string) {
	// Validate response status codes
	v.validateResponseStatusCodes(op.Responses, path, result, baseURL)

	// Validate consumes/produces media types
	for i, mediaType := range op.Consumes {
		if !isValidMediaType(mediaType) {
			result.Errors = append(result.Errors, ValidationError{
				Path:     fmt.Sprintf("%s.consumes[%d]", path, i),
				Message:  fmt.Sprintf("Invalid media type: %s", mediaType),
				SpecRef:  fmt.Sprintf("%s#operation-object", baseURL),
				Severity: SeverityError,
				Value:    mediaType,
			})
		}
	}

	for i, mediaType := range op.Produces {
		if !isValidMediaType(mediaType) {
			result.Errors = append(result.Errors, ValidationError{
				Path:     fmt.Sprintf("%s.produces[%d]", path, i),
				Message:  fmt.Sprintf("Invalid media type: %s", mediaType),
				SpecRef:  fmt.Sprintf("%s#operation-object", baseURL),
				Severity: SeverityError,
				Value:    mediaType,
			})
		}
	}
}

// validateOAS2Definitions validates schema definitions in OAS 2.0
func (v *Validator) validateOAS2Definitions(doc *parser.OAS2Document, result *ValidationResult, _ string) {
	for name, schema := range doc.Definitions {
		if schema == nil {
			continue
		}
		path := fmt.Sprintf("definitions.%s", name)
		v.validateSchema(schema, path, result)
	}
}

// validateOAS2Parameters validates parameters definitions in OAS 2.0
func (v *Validator) validateOAS2Parameters(doc *parser.OAS2Document, result *ValidationResult, baseURL string) {
	for name, param := range doc.Parameters {
		if param == nil {
			continue
		}
		path := fmt.Sprintf("parameters.%s", name)

		// Body parameters must have a schema
		if param.In == "body" && param.Schema == nil {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "Body parameter must have a schema",
				SpecRef:  fmt.Sprintf("%s#parameter-object", baseURL),
				Severity: SeverityError,
				Field:    "schema",
			})
		}

		// Non-body parameters must have a type
		if param.In != "body" && param.Type == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "Non-body parameter must have a type",
				SpecRef:  fmt.Sprintf("%s#parameter-object", baseURL),
				Severity: SeverityError,
				Field:    "type",
			})
		}
	}
}

// validateOAS2Responses validates response definitions in OAS 2.0
func (v *Validator) validateOAS2Responses(doc *parser.OAS2Document, result *ValidationResult, baseURL string) {
	for name, response := range doc.Responses {
		if response == nil {
			continue
		}
		path := fmt.Sprintf("responses.%s", name)

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
}

// validateOAS2Security validates security definitions and requirements in OAS 2.0
func (v *Validator) validateOAS2Security(doc *parser.OAS2Document, result *ValidationResult, baseURL string) {
	// Validate security requirements reference existing definitions
	for i, secReq := range doc.Security {
		for schemeName := range secReq {
			if _, exists := doc.SecurityDefinitions[schemeName]; !exists {
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

	// Validate security definitions
	for name, secDef := range doc.SecurityDefinitions {
		path := fmt.Sprintf("securityDefinitions.%s", name)

		if secDef.Type == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "Security scheme must have a type",
				SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
				Severity: SeverityError,
				Field:    "type",
			})
		}

		// Validate type-specific requirements
		switch secDef.Type {
		case "apiKey":
			if secDef.Name == "" {
				result.Errors = append(result.Errors, ValidationError{
					Path:     path,
					Message:  "API key security scheme must have a name",
					SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
					Severity: SeverityError,
					Field:    "name",
				})
			}
			if secDef.In == "" {
				result.Errors = append(result.Errors, ValidationError{
					Path:     path,
					Message:  "API key security scheme must specify 'in' (query or header)",
					SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
					Severity: SeverityError,
					Field:    "in",
				})
			}
		case "oauth2":
			if secDef.Flow == "" {
				result.Errors = append(result.Errors, ValidationError{
					Path:     path,
					Message:  "OAuth2 security scheme must have a flow",
					SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
					Severity: SeverityError,
					Field:    "flow",
				})
			}
			// Validate flow-specific requirements
			switch secDef.Flow {
			case "implicit", "accessCode":
				if secDef.AuthorizationURL == "" {
					result.Errors = append(result.Errors, ValidationError{
						Path:     path,
						Message:  fmt.Sprintf("OAuth2 flow '%s' requires authorizationUrl", secDef.Flow),
						SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
						Severity: SeverityError,
						Field:    "authorizationUrl",
					})
				} else if !isValidURL(secDef.AuthorizationURL) {
					result.Errors = append(result.Errors, ValidationError{
						Path:     path,
						Message:  fmt.Sprintf("Invalid URL format for authorizationUrl: %s", secDef.AuthorizationURL),
						SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
						Severity: SeverityError,
						Field:    "authorizationUrl",
						Value:    secDef.AuthorizationURL,
					})
				}
			}
			if secDef.Flow == "password" || secDef.Flow == "application" || secDef.Flow == "accessCode" {
				if secDef.TokenURL == "" {
					result.Errors = append(result.Errors, ValidationError{
						Path:     path,
						Message:  fmt.Sprintf("OAuth2 flow '%s' requires tokenUrl", secDef.Flow),
						SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
						Severity: SeverityError,
						Field:    "tokenUrl",
					})
				} else if !isValidURL(secDef.TokenURL) {
					result.Errors = append(result.Errors, ValidationError{
						Path:     path,
						Message:  fmt.Sprintf("Invalid URL format for tokenUrl: %s", secDef.TokenURL),
						SpecRef:  fmt.Sprintf("%s#security-scheme-object", baseURL),
						Severity: SeverityError,
						Field:    "tokenUrl",
						Value:    secDef.TokenURL,
					})
				}
			}
		}
	}
}

// validateOAS2PathParameterConsistency checks that path parameters match the path template
func (v *Validator) validateOAS2PathParameterConsistency(doc *parser.OAS2Document, result *ValidationResult, baseURL string) {
	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		// Extract parameter names from path template
		pathParams := extractPathParameters(pathPattern)

		// Check all operations in this path
		operations := parser.GetOperations(pathItem, parser.OASVersion20)

		for method, op := range operations {
			if op == nil {
				continue
			}

			// Collect declared path parameters
			declaredParams := make(map[string]bool)

			// Check path-level parameters
			for _, param := range pathItem.Parameters {
				if param != nil && param.In == "path" {
					declaredParams[param.Name] = true
				}
			}

			// Check operation-level parameters
			for _, param := range op.Parameters {
				if param != nil && param.In == "path" {
					declaredParams[param.Name] = true
				}
			}

			// Verify all path template parameters are declared
			for paramName := range pathParams {
				if !declaredParams[paramName] {
					result.Errors = append(result.Errors, ValidationError{
						Path:     fmt.Sprintf("paths.%s.%s", pathPattern, method),
						Message:  fmt.Sprintf("Path template references parameter '{%s}' but it is not declared in parameters", paramName),
						SpecRef:  fmt.Sprintf("%s#path-item-object", baseURL),
						Severity: SeverityError,
						Value:    paramName,
					})
				}
			}

			// Warn about declared path parameters not in template
			for paramName := range declaredParams {
				if !pathParams[paramName] {
					result.Warnings = append(result.Warnings, ValidationError{
						Path:     fmt.Sprintf("paths.%s.%s", pathPattern, method),
						Message:  fmt.Sprintf("Parameter '%s' is declared as path parameter but not used in path template", paramName),
						SpecRef:  fmt.Sprintf("%s#path-item-object", baseURL),
						Severity: SeverityWarning,
						Value:    paramName,
					})
				}
			}
		}
	}
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

	if doc.Info.Title == "" {
		v.addError(result, "info.title", "Info object must have a title",
			withSpecRef(fmt.Sprintf("%s#info-object", baseURL)),
			withField("title"),
		)
	}

	if doc.Info.Version == "" {
		v.addError(result, "info.version", "Info object must have a version",
			withSpecRef(fmt.Sprintf("%s#info-object", baseURL)),
			withField("version"),
		)
	}

	// Validate contact information if present
	if doc.Info.Contact != nil {
		if doc.Info.Contact.URL != "" && !isValidURL(doc.Info.Contact.URL) {
			v.addError(result, "info.contact.url", fmt.Sprintf("Invalid URL format: %s", doc.Info.Contact.URL),
				withSpecRef(fmt.Sprintf("%s#contact-object", baseURL)),
				withField("url"),
				withValue(doc.Info.Contact.URL),
			)
		}
		if doc.Info.Contact.Email != "" && !isValidEmail(doc.Info.Contact.Email) {
			v.addError(result, "info.contact.email", fmt.Sprintf("Invalid email format: %s", doc.Info.Contact.Email),
				withSpecRef(fmt.Sprintf("%s#contact-object", baseURL)),
				withField("email"),
				withValue(doc.Info.Contact.Email),
			)
		}
	}

	// Validate license information if present
	if doc.Info.License != nil {
		if doc.Info.License.URL != "" && !isValidURL(doc.Info.License.URL) {
			v.addError(result, "info.license.url", fmt.Sprintf("Invalid URL format: %s", doc.Info.License.URL),
				withSpecRef(fmt.Sprintf("%s#license-object", baseURL)),
				withField("url"),
				withValue(doc.Info.License.URL),
			)
		}
		// SPDX license identifier validation (OAS 3.1+)
		if doc.Info.License.Identifier != "" && !validateSPDXLicense(doc.Info.License.Identifier) {
			v.addError(result, "info.license.identifier", fmt.Sprintf("Invalid SPDX license identifier format: %s", doc.Info.License.Identifier),
				withSpecRef(fmt.Sprintf("%s#license-object", baseURL)),
				withField("identifier"),
				withValue(doc.Info.License.Identifier),
			)
		}
	}
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

// validateSchema performs basic schema validation
func (v *Validator) validateSchema(schema *parser.Schema, path string, result *ValidationResult) {
	v.validateSchemaWithVisited(schema, path, result, make(map[*parser.Schema]bool))
}

// validateSchemaWithVisited performs basic schema validation with cycle detection
func (v *Validator) validateSchemaWithVisited(schema *parser.Schema, path string, result *ValidationResult, visited map[*parser.Schema]bool) {
	if schema == nil {
		return
	}

	// Check for circular references
	if visited[schema] {
		return
	}
	visited[schema] = true

	// Check for excessive nesting depth to prevent resource exhaustion
	depth := strings.Count(path, ".")
	if depth > maxSchemaNestingDepth {
		result.Errors = append(result.Errors, ValidationError{
			Path:     path,
			Message:  fmt.Sprintf("Schema nesting depth (%d) exceeds maximum allowed (%d)", depth, maxSchemaNestingDepth),
			SpecRef:  getJSONSchemaRef(),
			Severity: SeverityError,
		})
		return
	}

	// Validate enum values match the schema type
	if len(schema.Enum) > 0 && schema.Type != "" {
		v.validateEnumValues(schema, path, result)
	}

	// Validate type-specific constraints
	v.validateSchemaTypeConstraints(schema, path, result)

	// Validate required fields
	v.validateRequiredFields(schema, path, result)

	// Validate nested schemas
	v.validateNestedSchemas(schema, path, result, visited)
}

// Helper functions

// validateEnumValues validates that enum values match the schema type
func (v *Validator) validateEnumValues(schema *parser.Schema, path string, result *ValidationResult) {
	for i, enumVal := range schema.Enum {
		enumPath := fmt.Sprintf("%s.enum[%d]", path, i)

		switch schema.Type {
		case "string":
			if _, ok := enumVal.(string); !ok {
				result.Errors = append(result.Errors, ValidationError{
					Path:     enumPath,
					Message:  fmt.Sprintf("Enum value must be a string (found %T)", enumVal),
					SpecRef:  getJSONSchemaRef(),
					Severity: SeverityError,
					Field:    "enum",
					Value:    enumVal,
				})
			}
		case "integer":
			// Check if it's an integer (can be int, int32, int64, or float64 with no decimal part)
			switch v := enumVal.(type) {
			case int, int32, int64:
				// Valid integer
			case float64:
				if v != float64(int64(v)) {
					result.Errors = append(result.Errors, ValidationError{
						Path:     enumPath,
						Message:  fmt.Sprintf("Enum value must be an integer (found %v)", enumVal),
						SpecRef:  getJSONSchemaRef(),
						Severity: SeverityError,
						Field:    "enum",
						Value:    enumVal,
					})
				}
			default:
				result.Errors = append(result.Errors, ValidationError{
					Path:     enumPath,
					Message:  fmt.Sprintf("Enum value must be an integer (found %T)", enumVal),
					SpecRef:  getJSONSchemaRef(),
					Severity: SeverityError,
					Field:    "enum",
					Value:    enumVal,
				})
			}
		case "number":
			// Check if it's a number (int or float)
			switch enumVal.(type) {
			case int, int32, int64, float32, float64:
				// Valid number
			default:
				result.Errors = append(result.Errors, ValidationError{
					Path:     enumPath,
					Message:  fmt.Sprintf("Enum value must be a number (found %T)", enumVal),
					SpecRef:  getJSONSchemaRef(),
					Severity: SeverityError,
					Field:    "enum",
					Value:    enumVal,
				})
			}
		case "boolean":
			if _, ok := enumVal.(bool); !ok {
				result.Errors = append(result.Errors, ValidationError{
					Path:     enumPath,
					Message:  fmt.Sprintf("Enum value must be a boolean (found %T)", enumVal),
					SpecRef:  getJSONSchemaRef(),
					Severity: SeverityError,
					Field:    "enum",
					Value:    enumVal,
				})
			}
		case "null":
			if enumVal != nil {
				result.Errors = append(result.Errors, ValidationError{
					Path:     enumPath,
					Message:  "Enum value must be null",
					SpecRef:  getJSONSchemaRef(),
					Severity: SeverityError,
					Field:    "enum",
					Value:    enumVal,
				})
			}
		}
	}
}

// validateSchemaTypeConstraints validates type-specific constraints for a schema
func (v *Validator) validateSchemaTypeConstraints(schema *parser.Schema, path string, result *ValidationResult) {
	if schema.Type == "" {
		return
	}

	switch schema.Type {
	case "array":
		if schema.Items == nil {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "Array schema must have 'items' defined",
				SpecRef:  getJSONSchemaRef(),
				Severity: SeverityError,
				Field:    "items",
			})
		}
	case "string":
		// Validate min/max length
		if schema.MinLength != nil && schema.MaxLength != nil && *schema.MinLength > *schema.MaxLength {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("minLength (%d) cannot be greater than maxLength (%d)", *schema.MinLength, *schema.MaxLength),
				SpecRef:  getJSONSchemaRef(),
				Severity: SeverityError,
			})
		}
	case "number", "integer":
		// Validate minimum/maximum
		if schema.Minimum != nil && schema.Maximum != nil && *schema.Minimum > *schema.Maximum {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("minimum (%v) cannot be greater than maximum (%v)", *schema.Minimum, *schema.Maximum),
				SpecRef:  getJSONSchemaRef(),
				Severity: SeverityError,
			})
		}
	}
}

// validateRequiredFields validates that required fields exist in properties
func (v *Validator) validateRequiredFields(schema *parser.Schema, path string, result *ValidationResult) {
	for _, reqField := range schema.Required {
		if _, exists := schema.Properties[reqField]; !exists {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("Required field '%s' not found in properties", reqField),
				SpecRef:  getJSONSchemaRef(),
				Severity: SeverityError,
				Field:    "required",
				Value:    reqField,
			})
		}
	}
}

// validateNestedSchemas validates all nested schemas (properties, items, allOf, oneOf, anyOf, not)
func (v *Validator) validateNestedSchemas(schema *parser.Schema, path string, result *ValidationResult, visited map[*parser.Schema]bool) {
	// Validate properties
	for propName, propSchema := range schema.Properties {
		if propSchema == nil {
			continue
		}
		propPath := fmt.Sprintf("%s.properties.%s", path, propName)
		v.validateSchemaWithVisited(propSchema, propPath, result, visited)
	}

	// Validate additionalProperties (can be *Schema or bool)
	if schema.AdditionalProperties != nil {
		if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
			addPropsPath := fmt.Sprintf("%s.additionalProperties", path)
			v.validateSchemaWithVisited(addProps, addPropsPath, result, visited)
		}
	}

	// Validate items (can be *Schema or bool)
	if schema.Items != nil {
		if items, ok := schema.Items.(*parser.Schema); ok {
			itemsPath := fmt.Sprintf("%s.items", path)
			v.validateSchemaWithVisited(items, itemsPath, result, visited)
		}
	}

	// Validate allOf
	for i, subSchema := range schema.AllOf {
		if subSchema == nil {
			continue
		}
		subPath := fmt.Sprintf("%s.allOf[%d]", path, i)
		v.validateSchemaWithVisited(subSchema, subPath, result, visited)
	}

	// Validate oneOf
	for i, subSchema := range schema.OneOf {
		if subSchema == nil {
			continue
		}
		subPath := fmt.Sprintf("%s.oneOf[%d]", path, i)
		v.validateSchemaWithVisited(subSchema, subPath, result, visited)
	}

	// Validate anyOf
	for i, subSchema := range schema.AnyOf {
		if subSchema == nil {
			continue
		}
		subPath := fmt.Sprintf("%s.anyOf[%d]", path, i)
		v.validateSchemaWithVisited(subSchema, subPath, result, visited)
	}

	// Validate not
	if schema.Not != nil {
		notPath := fmt.Sprintf("%s.not", path)
		v.validateSchemaWithVisited(schema.Not, notPath, result, visited)
	}
}

// Compile regex once at package level for performance
var (
	pathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)
	emailRegex     = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// checkDuplicateOperationIds checks for duplicate operationIds in a set of operations
// and reports errors when found. Updates the operationIds map as it processes operations.
func (v *Validator) checkDuplicateOperationIds(
	operations map[string]*parser.Operation,
	pathType string,
	pathPattern string,
	operationIds map[string]string,
	result *ValidationResult,
	baseURL string,
) {
	for method, op := range operations {
		if op == nil || op.OperationID == "" {
			continue
		}

		opPath := fmt.Sprintf("%s.%s.%s", pathType, pathPattern, method)

		if firstSeenAt, exists := operationIds[op.OperationID]; exists {
			// Determine the correct spec reference based on path type
			specRef := fmt.Sprintf("%s#operation-object", baseURL)
			if pathType == "webhooks" || strings.Contains(baseURL, "v3") {
				specRef = fmt.Sprintf("%s#operation-object", baseURL)
			}

			result.Errors = append(result.Errors, ValidationError{
				Path:     opPath,
				Message:  fmt.Sprintf("Duplicate operationId '%s' (first seen at %s)", op.OperationID, firstSeenAt),
				SpecRef:  specRef,
				Severity: SeverityError,
				Field:    "operationId",
				Value:    op.OperationID,
			})
		} else {
			operationIds[op.OperationID] = opPath
		}
	}
}

// validatePathTemplate validates that a path template is well-formed
// Returns an error if the template is malformed (unclosed braces, empty parameters, etc.)
func validatePathTemplate(pathPattern string) error {
	// Check for empty braces explicitly (regex won't catch {})
	if strings.Contains(pathPattern, "{}") {
		return fmt.Errorf("empty parameter name in path template")
	}

	// Check for consecutive slashes
	if strings.Contains(pathPattern, "//") {
		return fmt.Errorf("path contains consecutive slashes")
	}

	// Check for reserved characters (fragment identifier and query string)
	if strings.Contains(pathPattern, "#") {
		return fmt.Errorf("path contains reserved character '#'")
	}
	if strings.Contains(pathPattern, "?") {
		return fmt.Errorf("path contains reserved character '?'")
	}

	// Note: Trailing slashes are handled separately as warnings, not errors
	// Empty segments in the middle are caught by the consecutive slash check above

	// Check for unclosed or unopened braces
	openCount := 0
	for i, ch := range pathPattern {
		switch ch {
		case '{':
			openCount++
			if openCount > 1 {
				return fmt.Errorf("nested braces are not allowed at position %d", i)
			}
		case '}':
			openCount--
			if openCount < 0 {
				return fmt.Errorf("unexpected closing brace at position %d", i)
			}
		}
	}
	if openCount != 0 {
		return fmt.Errorf("unclosed brace in path template")
	}

	// Check for empty or invalid parameters, and track duplicates
	paramNames := make(map[string]bool)
	matches := pathParamRegex.FindAllStringSubmatch(pathPattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			paramName := match[1]
			if strings.TrimSpace(paramName) == "" {
				return fmt.Errorf("empty parameter name in path template")
			}
			// Check for invalid characters in parameter name
			if strings.Contains(paramName, "{") || strings.Contains(paramName, "}") {
				return fmt.Errorf("invalid parameter name '%s' contains braces", paramName)
			}
			// Check for duplicate parameter names
			if paramNames[paramName] {
				return fmt.Errorf("duplicate parameter name '%s' in path template", paramName)
			}
			paramNames[paramName] = true
		}
	}

	return nil
}

// checkTrailingSlash adds a warning if the path has a trailing slash
// Trailing slashes are discouraged by REST best practices but not forbidden by OAS spec
func checkTrailingSlash(v *Validator, pathPattern string, result *ValidationResult, baseURL string) {
	if v.IncludeWarnings && len(pathPattern) > 1 && strings.HasSuffix(pathPattern, "/") {
		result.Warnings = append(result.Warnings, ValidationError{
			Path:     fmt.Sprintf("paths.%s", pathPattern),
			Message:  "Path has trailing slash, which is discouraged by REST best practices",
			SpecRef:  fmt.Sprintf("%s#paths-object", baseURL),
			Severity: SeverityWarning,
			Value:    pathPattern,
		})
	}
}

// extractPathParameters extracts parameter names from a path template
// e.g., "/pets/{petId}/owners/{ownerId}" -> {"petId": true, "ownerId": true}
func extractPathParameters(pathPattern string) map[string]bool {
	params := make(map[string]bool)
	matches := pathParamRegex.FindAllStringSubmatch(pathPattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params[match[1]] = true
		}
	}
	return params
}

// isValidMediaType checks if a media type string is valid using RFC-compliant parsing
// This uses the standard library's mime.ParseMediaType which validates according to RFC 2045 and RFC 2046.
// This allows custom and vendor-specific media types (e.g., application/vnd.custom+json).
func isValidMediaType(mediaType string) bool {
	if mediaType == "" {
		return false
	}

	// Check for wildcard patterns first (mime.ParseMediaType doesn't handle these)
	// Valid: */* (both wildcards) or type/* (subtype wildcard)
	// Invalid: */subtype (type wildcard with specific subtype)
	if strings.Contains(mediaType, "*") {
		parts := strings.Split(strings.Split(mediaType, ";")[0], "/") // Remove parameters before checking
		if len(parts) != 2 {
			return false
		}
		if parts[0] == "*" {
			return parts[1] == "*" // */subtype is invalid
		}
		if parts[1] == "*" {
			return parts[0] != "" // type/* is valid if type is not empty
		}
	}

	// Use standard library for RFC-compliant validation
	_, _, err := mime.ParseMediaType(mediaType)
	return err == nil
}

// getJSONSchemaRef returns the JSON Schema specification reference URL
func getJSONSchemaRef() string {
	return "https://www.ietf.org/archive/id/draft-bhutton-json-schema-01.html"
}

// isValidURL performs URL validation using standard library's url.Parse
// Validates contact.url, externalDocs.url, license.url, and OAuth URLs
func isValidURL(s string) bool {
	if s == "" {
		return false
	}

	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	// Accept http/https schemes, or relative URLs starting with /
	// Reject bare strings without proper URL structure
	if u.Scheme == "http" || u.Scheme == "https" {
		return true
	}
	if u.Scheme == "" && strings.HasPrefix(s, "/") {
		return true
	}
	return false
}

// isValidEmail performs email validation using regex
// Validates contact.email in the info object
func isValidEmail(s string) bool {
	if s == "" {
		return true // Empty is valid (optional field)
	}
	return emailRegex.MatchString(s)
}

// validateSPDXLicense validates SPDX license identifier (basic validation)
// Used to validate license.identifier in the info object (OAS 3.1+)
func validateSPDXLicense(identifier string) bool {
	if identifier == "" {
		return true
	}
	// Basic validation - should not contain spaces and follow SPDX format
	// For a complete implementation, you'd need the full SPDX license list
	return !strings.Contains(identifier, " ")
}

// validateRef validates that a $ref string points to a valid location in the document
func (v *Validator) validateRef(ref, path string, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	if ref == "" {
		return
	}

	// Only validate local references (starting with #/)
	// External references (file paths, URLs) are handled by the parser's resolver
	if !strings.HasPrefix(ref, "#/") {
		// External reference - we don't validate these here
		return
	}

	// Check if the reference exists in the valid refs map
	if !validRefs[ref] {
		result.Errors = append(result.Errors, ValidationError{
			Path:     path,
			Message:  fmt.Sprintf("$ref '%s' does not resolve to a valid component in the document", ref),
			SpecRef:  baseURL,
			Severity: SeverityError,
			Field:    "$ref",
			Value:    ref,
		})
	}
}

// buildOAS2ValidRefs builds a map of all valid $ref paths in an OAS 2.0 document
func buildOAS2ValidRefs(doc *parser.OAS2Document) map[string]bool {
	validRefs := make(map[string]bool)

	// Add definitions
	for name := range doc.Definitions {
		validRefs[fmt.Sprintf("#/definitions/%s", name)] = true
	}

	// Add parameters
	for name := range doc.Parameters {
		validRefs[fmt.Sprintf("#/parameters/%s", name)] = true
	}

	// Add responses
	for name := range doc.Responses {
		validRefs[fmt.Sprintf("#/responses/%s", name)] = true
	}

	// Add security definitions
	for name := range doc.SecurityDefinitions {
		validRefs[fmt.Sprintf("#/securityDefinitions/%s", name)] = true
	}

	return validRefs
}

// buildOAS3ValidRefs builds a map of all valid $ref paths in an OAS 3.x document
func buildOAS3ValidRefs(doc *parser.OAS3Document) map[string]bool {
	validRefs := make(map[string]bool)

	if doc.Components == nil {
		return validRefs
	}

	// Add schemas
	for name := range doc.Components.Schemas {
		validRefs[fmt.Sprintf("#/components/schemas/%s", name)] = true
	}

	// Add responses
	for name := range doc.Components.Responses {
		validRefs[fmt.Sprintf("#/components/responses/%s", name)] = true
	}

	// Add parameters
	for name := range doc.Components.Parameters {
		validRefs[fmt.Sprintf("#/components/parameters/%s", name)] = true
	}

	// Add examples
	for name := range doc.Components.Examples {
		validRefs[fmt.Sprintf("#/components/examples/%s", name)] = true
	}

	// Add request bodies
	for name := range doc.Components.RequestBodies {
		validRefs[fmt.Sprintf("#/components/requestBodies/%s", name)] = true
	}

	// Add headers
	for name := range doc.Components.Headers {
		validRefs[fmt.Sprintf("#/components/headers/%s", name)] = true
	}

	// Add security schemes
	for name := range doc.Components.SecuritySchemes {
		validRefs[fmt.Sprintf("#/components/securitySchemes/%s", name)] = true
	}

	// Add links
	for name := range doc.Components.Links {
		validRefs[fmt.Sprintf("#/components/links/%s", name)] = true
	}

	// Add callbacks
	for name := range doc.Components.Callbacks {
		validRefs[fmt.Sprintf("#/components/callbacks/%s", name)] = true
	}

	// Add path items (OAS 3.1+)
	for name := range doc.Components.PathItems {
		validRefs[fmt.Sprintf("#/components/pathItems/%s", name)] = true
	}

	return validRefs
}

// validateSchemaRefs recursively validates all $ref values in a schema
func (v *Validator) validateSchemaRefs(schema *parser.Schema, path string, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	if schema == nil {
		return
	}

	// Validate the $ref in this schema
	if schema.Ref != "" {
		v.validateRef(schema.Ref, path, validRefs, result, baseURL)
	}

	// Recursively validate nested schemas
	// Properties
	for propName, propSchema := range schema.Properties {
		if propSchema != nil {
			propPath := fmt.Sprintf("%s.properties.%s", path, propName)
			v.validateSchemaRefs(propSchema, propPath, validRefs, result, baseURL)
		}
	}

	// Pattern properties
	for propName, propSchema := range schema.PatternProperties {
		if propSchema != nil {
			propPath := fmt.Sprintf("%s.patternProperties.%s", path, propName)
			v.validateSchemaRefs(propSchema, propPath, validRefs, result, baseURL)
		}
	}

	// Additional properties
	if schema.AdditionalProperties != nil {
		if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
			addPropsPath := fmt.Sprintf("%s.additionalProperties", path)
			v.validateSchemaRefs(addProps, addPropsPath, validRefs, result, baseURL)
		}
	}

	// Items
	if schema.Items != nil {
		if items, ok := schema.Items.(*parser.Schema); ok {
			itemsPath := fmt.Sprintf("%s.items", path)
			v.validateSchemaRefs(items, itemsPath, validRefs, result, baseURL)
		}
	}

	// AllOf, AnyOf, OneOf
	for i, subSchema := range schema.AllOf {
		if subSchema != nil {
			subPath := fmt.Sprintf("%s.allOf[%d]", path, i)
			v.validateSchemaRefs(subSchema, subPath, validRefs, result, baseURL)
		}
	}

	for i, subSchema := range schema.AnyOf {
		if subSchema != nil {
			subPath := fmt.Sprintf("%s.anyOf[%d]", path, i)
			v.validateSchemaRefs(subSchema, subPath, validRefs, result, baseURL)
		}
	}

	for i, subSchema := range schema.OneOf {
		if subSchema != nil {
			subPath := fmt.Sprintf("%s.oneOf[%d]", path, i)
			v.validateSchemaRefs(subSchema, subPath, validRefs, result, baseURL)
		}
	}

	// Not
	if schema.Not != nil {
		notPath := fmt.Sprintf("%s.not", path)
		v.validateSchemaRefs(schema.Not, notPath, validRefs, result, baseURL)
	}

	// Additional items
	if schema.AdditionalItems != nil {
		if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
			addItemsPath := fmt.Sprintf("%s.additionalItems", path)
			v.validateSchemaRefs(addItems, addItemsPath, validRefs, result, baseURL)
		}
	}

	// Prefix items (JSON Schema Draft 2020-12)
	for i, prefixItem := range schema.PrefixItems {
		if prefixItem != nil {
			prefixPath := fmt.Sprintf("%s.prefixItems[%d]", path, i)
			v.validateSchemaRefs(prefixItem, prefixPath, validRefs, result, baseURL)
		}
	}

	// Contains, PropertyNames (JSON Schema Draft 2020-12)
	if schema.Contains != nil {
		v.validateSchemaRefs(schema.Contains, fmt.Sprintf("%s.contains", path), validRefs, result, baseURL)
	}

	if schema.PropertyNames != nil {
		v.validateSchemaRefs(schema.PropertyNames, fmt.Sprintf("%s.propertyNames", path), validRefs, result, baseURL)
	}

	// Dependent schemas (JSON Schema Draft 2020-12)
	for name, depSchema := range schema.DependentSchemas {
		if depSchema != nil {
			depPath := fmt.Sprintf("%s.dependentSchemas.%s", path, name)
			v.validateSchemaRefs(depSchema, depPath, validRefs, result, baseURL)
		}
	}

	// If/Then/Else (JSON Schema Draft 2020-12, OAS 3.1+)
	if schema.If != nil {
		v.validateSchemaRefs(schema.If, fmt.Sprintf("%s.if", path), validRefs, result, baseURL)
	}
	if schema.Then != nil {
		v.validateSchemaRefs(schema.Then, fmt.Sprintf("%s.then", path), validRefs, result, baseURL)
	}
	if schema.Else != nil {
		v.validateSchemaRefs(schema.Else, fmt.Sprintf("%s.else", path), validRefs, result, baseURL)
	}

	// $defs (JSON Schema Draft 2020-12)
	for name, defSchema := range schema.Defs {
		if defSchema != nil {
			defPath := fmt.Sprintf("%s.$defs.%s", path, name)
			v.validateSchemaRefs(defSchema, defPath, validRefs, result, baseURL)
		}
	}
}

// validateParameterRef validates a parameter's $ref if present
func (v *Validator) validateParameterRef(param *parser.Parameter, path string, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	if param == nil {
		return
	}

	if param.Ref != "" {
		v.validateRef(param.Ref, path, validRefs, result, baseURL)
	}

	// Also validate schema refs within the parameter
	if param.Schema != nil {
		v.validateSchemaRefs(param.Schema, fmt.Sprintf("%s.schema", path), validRefs, result, baseURL)
	}
}

// validateResponseRef validates a response's $ref if present
func (v *Validator) validateResponseRef(response *parser.Response, path string, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	if response == nil {
		return
	}

	if response.Ref != "" {
		v.validateRef(response.Ref, path, validRefs, result, baseURL)
	}

	// Validate schema refs in the response
	if response.Schema != nil {
		v.validateSchemaRefs(response.Schema, fmt.Sprintf("%s.schema", path), validRefs, result, baseURL)
	}

	// Validate content schemas (OAS 3.x)
	for mediaType, mediaTypeObj := range response.Content {
		if mediaTypeObj != nil && mediaTypeObj.Schema != nil {
			schemaPath := fmt.Sprintf("%s.content.%s.schema", path, mediaType)
			v.validateSchemaRefs(mediaTypeObj.Schema, schemaPath, validRefs, result, baseURL)
		}
	}

	// Validate headers
	for headerName, header := range response.Headers {
		if header != nil {
			headerPath := fmt.Sprintf("%s.headers.%s", path, headerName)
			if header.Ref != "" {
				v.validateRef(header.Ref, headerPath, validRefs, result, baseURL)
			}
			if header.Schema != nil {
				v.validateSchemaRefs(header.Schema, fmt.Sprintf("%s.schema", headerPath), validRefs, result, baseURL)
			}
		}
	}

	// Validate links (OAS 3.x)
	for linkName, link := range response.Links {
		if link != nil && link.Ref != "" {
			linkPath := fmt.Sprintf("%s.links.%s", path, linkName)
			v.validateRef(link.Ref, linkPath, validRefs, result, baseURL)
		}
	}
}

// validateRequestBodyRef validates a request body's $ref if present
func (v *Validator) validateRequestBodyRef(requestBody *parser.RequestBody, path string, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	if requestBody == nil {
		return
	}

	if requestBody.Ref != "" {
		v.validateRef(requestBody.Ref, path, validRefs, result, baseURL)
	}

	// Validate content schemas
	for mediaType, mediaTypeObj := range requestBody.Content {
		if mediaTypeObj != nil && mediaTypeObj.Schema != nil {
			schemaPath := fmt.Sprintf("%s.content.%s.schema", path, mediaType)
			v.validateSchemaRefs(mediaTypeObj.Schema, schemaPath, validRefs, result, baseURL)
		}
	}
}

// validateOAS2Refs validates all $ref values in an OAS 2.0 document
func (v *Validator) validateOAS2Refs(doc *parser.OAS2Document, result *ValidationResult, baseURL string) {
	// Build the map of valid reference paths
	validRefs := buildOAS2ValidRefs(doc)

	// Validate refs in definitions
	for name, schema := range doc.Definitions {
		if schema != nil {
			path := fmt.Sprintf("definitions.%s", name)
			v.validateSchemaRefs(schema, path, validRefs, result, baseURL)
		}
	}

	// Validate refs in parameters
	for name, param := range doc.Parameters {
		if param != nil {
			path := fmt.Sprintf("parameters.%s", name)
			v.validateParameterRef(param, path, validRefs, result, baseURL)
		}
	}

	// Validate refs in responses
	for name, response := range doc.Responses {
		if response != nil {
			path := fmt.Sprintf("responses.%s", name)
			v.validateResponseRef(response, path, validRefs, result, baseURL)
		}
	}

	// Validate refs in paths
	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		pathPrefix := fmt.Sprintf("paths.%s", pathPattern)

		// Validate path-level parameters
		for i, param := range pathItem.Parameters {
			if param != nil {
				paramPath := fmt.Sprintf("%s.parameters[%d]", pathPrefix, i)
				v.validateParameterRef(param, paramPath, validRefs, result, baseURL)
			}
		}

		// Validate each operation
		operations := parser.GetOperations(pathItem, parser.OASVersion20)
		for method, op := range operations {
			if op == nil {
				continue
			}

			opPath := fmt.Sprintf("%s.%s", pathPrefix, method)

			// Validate operation parameters
			for i, param := range op.Parameters {
				if param != nil {
					paramPath := fmt.Sprintf("%s.parameters[%d]", opPath, i)
					v.validateParameterRef(param, paramPath, validRefs, result, baseURL)
				}
			}

			// Validate operation responses
			if op.Responses != nil {
				if op.Responses.Default != nil {
					responsePath := fmt.Sprintf("%s.responses.default", opPath)
					v.validateResponseRef(op.Responses.Default, responsePath, validRefs, result, baseURL)
				}

				for code, response := range op.Responses.Codes {
					if response != nil {
						responsePath := fmt.Sprintf("%s.responses.%s", opPath, code)
						v.validateResponseRef(response, responsePath, validRefs, result, baseURL)
					}
				}
			}
		}
	}
}

// validateOAS3Refs validates all $ref values in an OAS 3.x document
func (v *Validator) validateOAS3Refs(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	// Build the map of valid reference paths
	validRefs := buildOAS3ValidRefs(doc)

	// Validate refs in components
	if doc.Components != nil {
		// Validate schemas
		for name, schema := range doc.Components.Schemas {
			if schema != nil {
				path := fmt.Sprintf("components.schemas.%s", name)
				v.validateSchemaRefs(schema, path, validRefs, result, baseURL)
			}
		}

		// Validate parameters
		for name, param := range doc.Components.Parameters {
			if param != nil {
				path := fmt.Sprintf("components.parameters.%s", name)
				v.validateParameterRef(param, path, validRefs, result, baseURL)
			}
		}

		// Validate responses
		for name, response := range doc.Components.Responses {
			if response != nil {
				path := fmt.Sprintf("components.responses.%s", name)
				v.validateResponseRef(response, path, validRefs, result, baseURL)
			}
		}

		// Validate request bodies
		for name, requestBody := range doc.Components.RequestBodies {
			if requestBody != nil {
				path := fmt.Sprintf("components.requestBodies.%s", name)
				v.validateRequestBodyRef(requestBody, path, validRefs, result, baseURL)
			}
		}

		// Validate headers
		for name, header := range doc.Components.Headers {
			if header != nil {
				headerPath := fmt.Sprintf("components.headers.%s", name)
				if header.Ref != "" {
					v.validateRef(header.Ref, headerPath, validRefs, result, baseURL)
				}
				if header.Schema != nil {
					v.validateSchemaRefs(header.Schema, fmt.Sprintf("%s.schema", headerPath), validRefs, result, baseURL)
				}
			}
		}
	}

	// Validate refs in paths
	if doc.Paths != nil {
		for pathPattern, pathItem := range doc.Paths {
			if pathItem == nil {
				continue
			}

			pathPrefix := fmt.Sprintf("paths.%s", pathPattern)

			// Validate PathItem $ref
			if pathItem.Ref != "" {
				v.validateRef(pathItem.Ref, pathPrefix, validRefs, result, baseURL)
			}

			// Validate path-level parameters
			for i, param := range pathItem.Parameters {
				if param != nil {
					paramPath := fmt.Sprintf("%s.parameters[%d]", pathPrefix, i)
					v.validateParameterRef(param, paramPath, validRefs, result, baseURL)
				}
			}

			// Validate each operation
			v.validatePathItemOperationRefs(pathItem, pathPrefix, doc.OASVersion, validRefs, result, baseURL)
		}
	}

	// Validate refs in webhooks (OAS 3.1+)
	for webhookName, pathItem := range doc.Webhooks {
		if pathItem == nil {
			continue
		}

		pathPrefix := fmt.Sprintf("webhooks.%s", webhookName)

		// Validate PathItem $ref
		if pathItem.Ref != "" {
			v.validateRef(pathItem.Ref, pathPrefix, validRefs, result, baseURL)
		}

		// Validate webhook operations
		v.validatePathItemOperationRefs(pathItem, pathPrefix, doc.OASVersion, validRefs, result, baseURL)
	}
}

// validatePathItemOperationRefs validates $ref values within all operations of a PathItem.
// This is used by both paths and webhooks validation to avoid code duplication.
func (v *Validator) validatePathItemOperationRefs(pathItem *parser.PathItem, pathPrefix string, version parser.OASVersion, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	operations := parser.GetOperations(pathItem, version)
	for method, op := range operations {
		if op == nil {
			continue
		}

		opPath := fmt.Sprintf("%s.%s", pathPrefix, method)

		// Validate operation parameters
		for i, param := range op.Parameters {
			if param != nil {
				paramPath := fmt.Sprintf("%s.parameters[%d]", opPath, i)
				v.validateParameterRef(param, paramPath, validRefs, result, baseURL)
			}
		}

		// Validate request body
		if op.RequestBody != nil {
			requestBodyPath := fmt.Sprintf("%s.requestBody", opPath)
			v.validateRequestBodyRef(op.RequestBody, requestBodyPath, validRefs, result, baseURL)
		}

		// Validate operation responses
		if op.Responses != nil {
			if op.Responses.Default != nil {
				responsePath := fmt.Sprintf("%s.responses.default", opPath)
				v.validateResponseRef(op.Responses.Default, responsePath, validRefs, result, baseURL)
			}

			for code, response := range op.Responses.Codes {
				if response != nil {
					responsePath := fmt.Sprintf("%s.responses.%s", opPath, code)
					v.validateResponseRef(response, responsePath, validRefs, result, baseURL)
				}
			}
		}
	}
}
