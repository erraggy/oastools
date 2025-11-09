package validator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/internal/parser"
)

// Severity indicates the severity level of a validation issue
type Severity int

const (
	// SeverityError indicates a spec violation that makes the document invalid
	SeverityError Severity = iota
	// SeverityWarning indicates a best practice violation or recommendation
	SeverityWarning
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	default:
		return "unknown"
	}
}

// ValidationError represents a single validation issue
type ValidationError struct {
	// Path is the JSON path to the problematic field (e.g., "paths./pets.get.responses")
	Path string
	// Message is a human-readable error message
	Message string
	// SpecRef is the URL to the relevant section of the OAS specification
	SpecRef string
	// Severity indicates whether this is an error or warning
	Severity Severity
	// Field is the specific field name that has the issue
	Field string
	// Value is the problematic value (optional)
	Value interface{}
}

// String returns a formatted string representation of the validation error
func (e ValidationError) String() string {
	severity := "✗"
	if e.Severity == SeverityWarning {
		severity = "⚠"
	}

	result := fmt.Sprintf("%s %s: %s", severity, e.Path, e.Message)
	if e.SpecRef != "" {
		result += fmt.Sprintf("\n    Spec: %s", e.SpecRef)
	}
	return result
}

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
}

// Validator handles OpenAPI specification validation
type Validator struct {
	// IncludeWarnings determines whether to include best practice warnings
	IncludeWarnings bool
	// StrictMode enables stricter validation beyond the spec requirements
	StrictMode bool
	// parser instance for parsing OAS documents
	parser *parser.Parser
}

// New creates a new Validator instance with default settings
func New() *Validator {
	return &Validator{
		IncludeWarnings: true,
		StrictMode:      false,
		parser:          parser.New(),
	}
}

// Validate validates an OpenAPI specification file
func (v *Validator) Validate(specPath string) (*ValidationResult, error) {
	// Parse the document first
	parseResult, err := v.parser.Parse(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse specification: %w", err)
	}

	result := &ValidationResult{
		Version:    parseResult.Version,
		OASVersion: parseResult.OASVersion,
		Errors:     make([]ValidationError, 0),
		Warnings:   make([]ValidationError, 0),
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
	switch parseResult.OASVersion {
	case parser.OASVersion20:
		doc, ok := parseResult.Document.(*parser.OAS2Document)
		if ok {
			v.validateOAS2(doc, result)
		}
	case parser.OASVersion300, parser.OASVersion301, parser.OASVersion302, parser.OASVersion303, parser.OASVersion304,
		parser.OASVersion310, parser.OASVersion311, parser.OASVersion312, parser.OASVersion320:
		doc, ok := parseResult.Document.(*parser.OAS3Document)
		if ok {
			v.validateOAS3(doc, result)
		}
	default:
		// in reality this should never happen, since the parser's `Parse` would have errored as well
		return nil, fmt.Errorf("unsupported OAS Version: %s", parseResult.OASVersion)
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

// validateOAS2 performs OAS 2.0 specific validation
func (v *Validator) validateOAS2(doc *parser.OAS2Document, result *ValidationResult) {
	baseURL := "https://spec.openapis.org/oas/v2.0.html"

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
}

// validateOAS2Paths validates paths in OAS 2.0
func (v *Validator) validateOAS2Paths(doc *parser.OAS2Document, result *ValidationResult, baseURL string) {
	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		pathPrefix := fmt.Sprintf("paths.%s", pathPattern)

		// Validate each operation
		operations := map[string]*parser.Operation{
			"get":     pathItem.Get,
			"put":     pathItem.Put,
			"post":    pathItem.Post,
			"delete":  pathItem.Delete,
			"options": pathItem.Options,
			"head":    pathItem.Head,
			"patch":   pathItem.Patch,
		}

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
					SpecRef:  fmt.Sprintf("%s#operationObject", baseURL),
					Severity: SeverityWarning,
					Field:    "description",
				})
			}
		}
	}
}

// validateOAS2Operation validates an operation in OAS 2.0
func (v *Validator) validateOAS2Operation(op *parser.Operation, path string, result *ValidationResult, baseURL string) {
	// Validate that at least one successful response exists
	if op.Responses != nil && op.Responses.Codes != nil {
		hasSuccess := false
		for code := range op.Responses.Codes {
			if strings.HasPrefix(code, "2") || code == "default" {
				hasSuccess = true
				break
			}
		}
		if !hasSuccess && v.StrictMode {
			result.Warnings = append(result.Warnings, ValidationError{
				Path:     fmt.Sprintf("%s.responses", path),
				Message:  "Operation should define at least one successful response (2XX or default)",
				SpecRef:  fmt.Sprintf("%s#responsesObject", baseURL),
				Severity: SeverityWarning,
			})
		}
	}

	// Validate consumes/produces media types
	for i, mediaType := range op.Consumes {
		if !isValidMediaType(mediaType) {
			result.Errors = append(result.Errors, ValidationError{
				Path:     fmt.Sprintf("%s.consumes[%d]", path, i),
				Message:  fmt.Sprintf("Invalid media type: %s", mediaType),
				SpecRef:  fmt.Sprintf("%s#operationObject", baseURL),
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
				SpecRef:  fmt.Sprintf("%s#operationObject", baseURL),
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
				SpecRef:  fmt.Sprintf("%s#parameterObject", baseURL),
				Severity: SeverityError,
				Field:    "schema",
			})
		}

		// Non-body parameters must have a type
		if param.In != "body" && param.Type == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "Non-body parameter must have a type",
				SpecRef:  fmt.Sprintf("%s#parameterObject", baseURL),
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
				SpecRef:  fmt.Sprintf("%s#responseObject", baseURL),
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
					SpecRef:  fmt.Sprintf("%s#securityRequirementObject", baseURL),
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
				SpecRef:  fmt.Sprintf("%s#securitySchemeObject", baseURL),
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
					SpecRef:  fmt.Sprintf("%s#securitySchemeObject", baseURL),
					Severity: SeverityError,
					Field:    "name",
				})
			}
			if secDef.In == "" {
				result.Errors = append(result.Errors, ValidationError{
					Path:     path,
					Message:  "API key security scheme must specify 'in' (query or header)",
					SpecRef:  fmt.Sprintf("%s#securitySchemeObject", baseURL),
					Severity: SeverityError,
					Field:    "in",
				})
			}
		case "oauth2":
			if secDef.Flow == "" {
				result.Errors = append(result.Errors, ValidationError{
					Path:     path,
					Message:  "OAuth2 security scheme must have a flow",
					SpecRef:  fmt.Sprintf("%s#securitySchemeObject", baseURL),
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
						SpecRef:  fmt.Sprintf("%s#securitySchemeObject", baseURL),
						Severity: SeverityError,
						Field:    "authorizationUrl",
					})
				}
			}
			if secDef.Flow == "password" || secDef.Flow == "application" || secDef.Flow == "accessCode" {
				if secDef.TokenURL == "" {
					result.Errors = append(result.Errors, ValidationError{
						Path:     path,
						Message:  fmt.Sprintf("OAuth2 flow '%s' requires tokenUrl", secDef.Flow),
						SpecRef:  fmt.Sprintf("%s#securitySchemeObject", baseURL),
						Severity: SeverityError,
						Field:    "tokenUrl",
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
		operations := map[string]*parser.Operation{
			"get":     pathItem.Get,
			"put":     pathItem.Put,
			"post":    pathItem.Post,
			"delete":  pathItem.Delete,
			"options": pathItem.Options,
			"head":    pathItem.Head,
			"patch":   pathItem.Patch,
		}

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
						SpecRef:  fmt.Sprintf("%s#pathItemObject", baseURL),
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
						SpecRef:  fmt.Sprintf("%s#pathItemObject", baseURL),
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
			if len(varObj.Enum) > 0 {
				found := false
				for _, enumVal := range varObj.Enum {
					if enumVal == varObj.Default {
						found = true
						break
					}
				}
				if !found {
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

		// Validate each operation
		operations := map[string]*parser.Operation{
			"get":     pathItem.Get,
			"put":     pathItem.Put,
			"post":    pathItem.Post,
			"delete":  pathItem.Delete,
			"options": pathItem.Options,
			"head":    pathItem.Head,
			"patch":   pathItem.Patch,
			"trace":   pathItem.Trace,
		}

		for method, op := range operations {
			if op == nil {
				continue
			}

			opPath := fmt.Sprintf("%s.%s", pathPrefix, method)
			v.validateOAS3Operation(op, opPath, result, baseURL)

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

// validateOAS3Operation validates an operation in OAS 3.x
func (v *Validator) validateOAS3Operation(op *parser.Operation, path string, result *ValidationResult, baseURL string) {
	// Validate that at least one successful response exists
	if op.Responses != nil && op.Responses.Codes != nil {
		hasSuccess := false
		for code := range op.Responses.Codes {
			if strings.HasPrefix(code, "2") || code == "default" {
				hasSuccess = true
				break
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
		}
		if flows.AuthorizationCode.TokenURL == "" {
			result.Errors = append(result.Errors, ValidationError{
				Path:     flowPath,
				Message:  "Authorization code flow must have tokenUrl",
				SpecRef:  fmt.Sprintf("%s#oauth-flows-object", baseURL),
				Severity: SeverityError,
				Field:    "tokenUrl",
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
		operations := map[string]*parser.Operation{
			"get":     pathItem.Get,
			"put":     pathItem.Put,
			"post":    pathItem.Post,
			"delete":  pathItem.Delete,
			"options": pathItem.Options,
			"head":    pathItem.Head,
			"patch":   pathItem.Patch,
			"trace":   pathItem.Trace,
		}

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
		operations := map[string]*parser.Operation{
			"get":     pathItem.Get,
			"put":     pathItem.Put,
			"post":    pathItem.Post,
			"delete":  pathItem.Delete,
			"options": pathItem.Options,
			"head":    pathItem.Head,
			"patch":   pathItem.Patch,
			"trace":   pathItem.Trace,
		}

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

			operations := map[string]*parser.Operation{
				"get":     pathItem.Get,
				"put":     pathItem.Put,
				"post":    pathItem.Post,
				"delete":  pathItem.Delete,
				"options": pathItem.Options,
				"head":    pathItem.Head,
				"patch":   pathItem.Patch,
				"trace":   pathItem.Trace,
			}

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
	if schema == nil {
		return
	}

	// Validate type-specific constraints
	if schema.Type != "" {
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

	// Validate properties
	for propName, propSchema := range schema.Properties {
		if propSchema == nil {
			continue
		}
		propPath := fmt.Sprintf("%s.properties.%s", path, propName)
		v.validateSchema(propSchema, propPath, result)
	}

	// Validate additionalProperties (can be *Schema or bool)
	if schema.AdditionalProperties != nil {
		if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
			addPropsPath := fmt.Sprintf("%s.additionalProperties", path)
			v.validateSchema(addProps, addPropsPath, result)
		}
	}

	// Validate items (can be *Schema or bool)
	if schema.Items != nil {
		if items, ok := schema.Items.(*parser.Schema); ok {
			itemsPath := fmt.Sprintf("%s.items", path)
			v.validateSchema(items, itemsPath, result)
		}
	}

	// Validate allOf
	for i, subSchema := range schema.AllOf {
		if subSchema == nil {
			continue
		}
		subPath := fmt.Sprintf("%s.allOf[%d]", path, i)
		v.validateSchema(subSchema, subPath, result)
	}

	// Validate oneOf
	for i, subSchema := range schema.OneOf {
		if subSchema == nil {
			continue
		}
		subPath := fmt.Sprintf("%s.oneOf[%d]", path, i)
		v.validateSchema(subSchema, subPath, result)
	}

	// Validate anyOf
	for i, subSchema := range schema.AnyOf {
		if subSchema == nil {
			continue
		}
		subPath := fmt.Sprintf("%s.anyOf[%d]", path, i)
		v.validateSchema(subSchema, subPath, result)
	}

	// Validate not
	if schema.Not != nil {
		notPath := fmt.Sprintf("%s.not", path)
		v.validateSchema(schema.Not, notPath, result)
	}
}

// Helper functions

// extractPathParameters extracts parameter names from a path template
// e.g., "/pets/{petId}/owners/{ownerId}" -> {"petId": true, "ownerId": true}
func extractPathParameters(pathPattern string) map[string]bool {
	params := make(map[string]bool)
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(pathPattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params[match[1]] = true
		}
	}
	return params
}

// isValidMediaType checks if a media type string is valid
func isValidMediaType(mediaType string) bool {
	if mediaType == "" {
		return false
	}

	// Basic validation - should contain a slash
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return false
	}

	// Check for wildcard patterns
	if parts[0] == "*" || parts[1] == "*" {
		return true
	}

	// Both parts should be non-empty
	return parts[0] != "" && parts[1] != ""
}

// getJSONSchemaRef returns the JSON Schema specification reference URL
func getJSONSchemaRef() string {
	return "https://www.ietf.org/archive/id/draft-bhutton-json-schema-01.html"
}

// isValidURL performs basic URL validation
func isValidURL(s string) bool {
	if s == "" {
		return false
	}
	// Basic check - should start with http:// or https:// or be a relative URL
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "/")
}

// isValidEmail performs basic email validation
func isValidEmail(s string) bool {
	if s == "" {
		return true // Empty is valid (optional field)
	}
	// Basic check - should contain @ and domain
	parts := strings.Split(s, "@")
	if len(parts) != 2 {
		return false
	}
	return parts[0] != "" && parts[1] != "" && strings.Contains(parts[1], ".")
}

// validateSPDXLicense validates SPDX license identifier (basic validation)
func validateSPDXLicense(identifier string) bool {
	if identifier == "" {
		return true
	}
	// Basic validation - should not contain spaces and follow SPDX format
	// For a complete implementation, you'd need the full SPDX license list
	return !strings.Contains(identifier, " ")
}

// validateHTTPStatusCode validates HTTP status code format
func validateHTTPStatusCode(code string) bool {
	if code == "default" {
		return true
	}

	// Check wildcard patterns (e.g., "2XX", "4XX")
	if len(code) == 3 && code[1] == 'X' && code[2] == 'X' {
		if code[0] >= '1' && code[0] <= '5' {
			return true
		}
		return false
	}

	// Check numeric status codes
	if len(code) != 3 {
		return false
	}

	statusCode, err := strconv.Atoi(code)
	if err != nil {
		return false
	}

	return statusCode >= 100 && statusCode <= 599
}
