package validator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/erraggy/oastools/parser"
)

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
			v.addError(result, path, "Server must have a url",
				withSpecRef(fmt.Sprintf("%s#server-object", baseURL)),
				withField("url"),
			)
		}

		// Validate server variables
		for varName, varObj := range server.Variables {
			varPath := fmt.Sprintf("%s.variables.%s", path, varName)

			if varObj.Default == "" {
				v.addError(result, varPath, "Server variable must have a default value",
					withSpecRef(fmt.Sprintf("%s#server-variable-object", baseURL)),
					withField("default"),
				)
			}

			// If enum is specified, default must be in enum
			if len(varObj.Enum) > 0 && !slices.Contains(varObj.Enum, varObj.Default) {
				v.addError(result, varPath,
					fmt.Sprintf("Server variable default value '%s' must be one of the enum values", varObj.Default),
					withSpecRef(fmt.Sprintf("%s#server-variable-object", baseURL)),
					withField("default"),
					withValue(varObj.Default),
				)
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
			v.addError(result, path, "Response must have a description",
				withSpecRef(fmt.Sprintf("%s#response-object", baseURL)),
				withField("description"),
			)
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
			v.addError(result, path, "Parameter must have either a schema or content",
				withSpecRef(fmt.Sprintf("%s#parameter-object", baseURL)),
			)
		}

		if hasSchema && hasContent {
			v.addError(result, path, "Parameter must not have both schema and content",
				withSpecRef(fmt.Sprintf("%s#parameter-object", baseURL)),
			)
		}

		// Path parameters must have required: true
		if param.In == "path" && !param.Required {
			v.addError(result, path, "Path parameters must have required: true",
				withSpecRef(fmt.Sprintf("%s#parameter-object", baseURL)),
				withField("required"),
			)
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
		v.addError(result, path, "Security scheme must have a type",
			withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
			withField("type"),
		)
		return
	}

	switch scheme.Type {
	case "apiKey":
		if scheme.Name == "" {
			v.addError(result, path, "API key security scheme must have a name",
				withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
				withField("name"),
			)
		}
		if scheme.In == "" {
			v.addError(result, path, "API key security scheme must specify 'in' (query, header, or cookie)",
				withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
				withField("in"),
			)
		}
	case "http":
		if scheme.Scheme == "" {
			v.addError(result, path, "HTTP security scheme must have a scheme (e.g., 'basic', 'bearer')",
				withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
				withField("scheme"),
			)
		}
	case "oauth2":
		if scheme.Flows == nil {
			v.addError(result, path, "OAuth2 security scheme must have flows",
				withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
				withField("flows"),
			)
		} else {
			v.validateOAuth2Flows(scheme.Flows, path, result, baseURL)
		}
	case "openIdConnect":
		if scheme.OpenIDConnectURL == "" {
			v.addError(result, path, "OpenID Connect security scheme must have openIdConnectUrl",
				withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
				withField("openIdConnectUrl"),
			)
		}
	}
}

// validateOAuth2Flows validates OAuth2 flows in OAS 3.x
func (v *Validator) validateOAuth2Flows(flows *parser.OAuthFlows, path string, result *ValidationResult, baseURL string) {
	// Validate implicit flow
	if flows.Implicit != nil {
		flowPath := fmt.Sprintf("%s.flows.implicit", path)
		if flows.Implicit.AuthorizationURL == "" {
			v.addError(result, flowPath, "Implicit flow must have authorizationUrl",
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("authorizationUrl"),
			)
		} else if !isValidURL(flows.Implicit.AuthorizationURL) {
			v.addError(result, flowPath,
				fmt.Sprintf("Invalid URL format for authorizationUrl: %s", flows.Implicit.AuthorizationURL),
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("authorizationUrl"),
				withValue(flows.Implicit.AuthorizationURL),
			)
		}
	}

	// Validate password flow
	if flows.Password != nil {
		flowPath := fmt.Sprintf("%s.flows.password", path)
		if flows.Password.TokenURL == "" {
			v.addError(result, flowPath, "Password flow must have tokenUrl",
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("tokenUrl"),
			)
		} else if !isValidURL(flows.Password.TokenURL) {
			v.addError(result, flowPath,
				fmt.Sprintf("Invalid URL format for tokenUrl: %s", flows.Password.TokenURL),
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("tokenUrl"),
				withValue(flows.Password.TokenURL),
			)
		}
	}

	// Validate clientCredentials flow
	if flows.ClientCredentials != nil {
		flowPath := fmt.Sprintf("%s.flows.clientCredentials", path)
		if flows.ClientCredentials.TokenURL == "" {
			v.addError(result, flowPath, "Client credentials flow must have tokenUrl",
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("tokenUrl"),
			)
		} else if !isValidURL(flows.ClientCredentials.TokenURL) {
			v.addError(result, flowPath,
				fmt.Sprintf("Invalid URL format for tokenUrl: %s", flows.ClientCredentials.TokenURL),
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("tokenUrl"),
				withValue(flows.ClientCredentials.TokenURL),
			)
		}
	}

	// Validate authorizationCode flow
	if flows.AuthorizationCode != nil {
		flowPath := fmt.Sprintf("%s.flows.authorizationCode", path)
		if flows.AuthorizationCode.AuthorizationURL == "" {
			v.addError(result, flowPath, "Authorization code flow must have authorizationUrl",
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("authorizationUrl"),
			)
		} else if !isValidURL(flows.AuthorizationCode.AuthorizationURL) {
			v.addError(result, flowPath,
				fmt.Sprintf("Invalid URL format for authorizationUrl: %s", flows.AuthorizationCode.AuthorizationURL),
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("authorizationUrl"),
				withValue(flows.AuthorizationCode.AuthorizationURL),
			)
		}
		if flows.AuthorizationCode.TokenURL == "" {
			v.addError(result, flowPath, "Authorization code flow must have tokenUrl",
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("tokenUrl"),
			)
		} else if !isValidURL(flows.AuthorizationCode.TokenURL) {
			v.addError(result, flowPath,
				fmt.Sprintf("Invalid URL format for tokenUrl: %s", flows.AuthorizationCode.TokenURL),
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("tokenUrl"),
				withValue(flows.AuthorizationCode.TokenURL),
			)
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
				v.addError(result, fmt.Sprintf("security[%d].%s", i, schemeName),
					fmt.Sprintf("Security requirement references undefined security scheme: %s", schemeName),
					withSpecRef(fmt.Sprintf("%s#security-requirement-object", baseURL)),
					withValue(schemeName),
				)
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
							v.addError(result, fmt.Sprintf("paths.%s.%s.security[%d].%s", pathPattern, method, i, schemeName),
								fmt.Sprintf("Security requirement references undefined security scheme: %s", schemeName),
								withSpecRef(fmt.Sprintf("%s#security-requirement-object", baseURL)),
								withValue(schemeName),
							)
						}
					}
				}
			}
		}
	}
}
