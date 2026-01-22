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
			v.addError(result, fmt.Sprintf("paths.%s", pathPattern),
				"Path must start with '/'",
				withSpecRef(fmt.Sprintf("%s#paths-object", baseURL)),
				withValue(pathPattern),
			)
		}

		// Validate path template is well-formed
		if err := validatePathTemplate(pathPattern); err != nil {
			v.addError(result, fmt.Sprintf("paths.%s", pathPattern),
				fmt.Sprintf("Invalid path template: %s", err),
				withSpecRef(fmt.Sprintf("%s#paths-object", baseURL)),
				withValue(pathPattern),
			)
		}

		// Warning: trailing slash in path (REST best practice)
		checkTrailingSlash(v, pathPattern, result, baseURL)

		pathPrefix := fmt.Sprintf("paths.%s", pathPattern)

		// Validate QUERY method is not used in OAS 2.0
		if pathItem.Query != nil {
			v.addError(result, fmt.Sprintf("%s.query", pathPrefix),
				"QUERY method is only supported in OAS 3.2+, not in OAS 2.0",
				withSpecRef(fmt.Sprintf("%s#path-item-object", baseURL)),
				withField("query"),
			)
		}

		// Validate TRACE method is not used in OAS 2.0
		if pathItem.Trace != nil {
			v.addError(result, fmt.Sprintf("%s.trace", pathPrefix),
				"TRACE method is only supported in OAS 3.0+, not in OAS 2.0",
				withSpecRef(fmt.Sprintf("%s#path-item-object", baseURL)),
				withField("trace"),
			)
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
				v.addWarning(result, opPath,
					"Operation should have a description or summary for better documentation",
					withSpecRef(fmt.Sprintf("%s#operation-object", baseURL)),
					withField("description"),
				)
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
			v.addError(result, fmt.Sprintf("%s.consumes[%d]", path, i),
				fmt.Sprintf("Invalid media type: %s", mediaType),
				withSpecRef(fmt.Sprintf("%s#operation-object", baseURL)),
				withValue(mediaType),
			)
		}
	}

	for i, mediaType := range op.Produces {
		if !isValidMediaType(mediaType) {
			v.addError(result, fmt.Sprintf("%s.produces[%d]", path, i),
				fmt.Sprintf("Invalid media type: %s", mediaType),
				withSpecRef(fmt.Sprintf("%s#operation-object", baseURL)),
				withValue(mediaType),
			)
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
			v.addError(result, path,
				"Body parameter must have a schema",
				withSpecRef(fmt.Sprintf("%s#parameter-object", baseURL)),
				withField("schema"),
			)
		}

		// Non-body parameters must have a type
		if param.In != "body" && param.Type == "" {
			v.addError(result, path,
				"Non-body parameter must have a type",
				withSpecRef(fmt.Sprintf("%s#parameter-object", baseURL)),
				withField("type"),
			)
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
			v.addError(result, path,
				"Response must have a description",
				withSpecRef(fmt.Sprintf("%s#response-object", baseURL)),
				withField("description"),
			)
		}
	}
}

// validateOAS2Security validates security definitions and requirements in OAS 2.0
func (v *Validator) validateOAS2Security(doc *parser.OAS2Document, result *ValidationResult, baseURL string) {
	// Validate security requirements reference existing definitions
	for i, secReq := range doc.Security {
		for schemeName := range secReq {
			if _, exists := doc.SecurityDefinitions[schemeName]; !exists {
				v.addError(result, fmt.Sprintf("security[%d].%s", i, schemeName),
					fmt.Sprintf("Security requirement references undefined security scheme: %s", schemeName),
					withSpecRef(fmt.Sprintf("%s#security-requirement-object", baseURL)),
					withValue(schemeName),
				)
			}
		}
	}

	// Validate security definitions
	for name, secDef := range doc.SecurityDefinitions {
		path := fmt.Sprintf("securityDefinitions.%s", name)

		if secDef.Type == "" {
			v.addError(result, path,
				"Security scheme must have a type",
				withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
				withField("type"),
			)
		}

		// Validate type-specific requirements
		switch secDef.Type {
		case "apiKey":
			if secDef.Name == "" {
				v.addError(result, path,
					"API key security scheme must have a name",
					withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
					withField("name"),
				)
			}
			if secDef.In == "" {
				v.addError(result, path,
					"API key security scheme must specify 'in' (query or header)",
					withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
					withField("in"),
				)
			}
		case "oauth2":
			if secDef.Flow == "" {
				v.addError(result, path,
					"OAuth2 security scheme must have a flow",
					withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
					withField("flow"),
				)
			}
			// Validate flow-specific requirements
			switch secDef.Flow {
			case "implicit", "accessCode":
				if secDef.AuthorizationURL == "" {
					v.addError(result, path,
						fmt.Sprintf("OAuth2 flow '%s' requires authorizationUrl", secDef.Flow),
						withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
						withField("authorizationUrl"),
					)
				} else if !isValidURL(secDef.AuthorizationURL) {
					v.addError(result, path,
						fmt.Sprintf("Invalid URL format for authorizationUrl: %s", secDef.AuthorizationURL),
						withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
						withField("authorizationUrl"),
						withValue(secDef.AuthorizationURL),
					)
				}
			}
			if secDef.Flow == "password" || secDef.Flow == "application" || secDef.Flow == "accessCode" {
				if secDef.TokenURL == "" {
					v.addError(result, path,
						fmt.Sprintf("OAuth2 flow '%s' requires tokenUrl", secDef.Flow),
						withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
						withField("tokenUrl"),
					)
				} else if !isValidURL(secDef.TokenURL) {
					v.addError(result, path,
						fmt.Sprintf("Invalid URL format for tokenUrl: %s", secDef.TokenURL),
						withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
						withField("tokenUrl"),
						withValue(secDef.TokenURL),
					)
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
