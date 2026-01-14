package validator

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/parser"
)

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
		v.addError(result, "info", "Document must have an info object",
			withSpecRef(fmt.Sprintf("%s#info-object", baseURL)),
			withField("info"),
		)
		return
	}
	v.validateInfoObject(doc.Info, result, baseURL, false)
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
		v.validateSchemaName(name, "definitions", result)
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
