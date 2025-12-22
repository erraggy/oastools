package converter

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// convertOAS3ToOAS2 converts an OAS 3.x document to OAS 2.0
func (c *Converter) convertOAS3ToOAS2(parseResult parser.ParseResult, result *ConversionResult) error {
	src, ok := parseResult.OAS3Document()
	if !ok {
		return fmt.Errorf("source document is not an OAS3Document")
	}

	// Create the OAS 2.0 document
	dst := &parser.OAS2Document{
		Swagger:    "2.0",
		OASVersion: parser.OASVersion20,
		Info:       src.Info,
		Paths:      make(map[string]*parser.PathItem),
		Tags:       src.Tags,
	}

	// Convert servers to host/basePath/schemes
	c.convertServersToHostBasePath(src, dst, result)

	// Convert components
	if src.Components != nil {
		// Convert schemas
		if len(src.Components.Schemas) > 0 {
			dst.Definitions = make(map[string]*parser.Schema)
			for name, schema := range src.Components.Schemas {
				path := fmt.Sprintf("components.schemas.%s", name)
				dst.Definitions[name] = c.convertOAS3SchemaToOAS2(schema, result, path)
			}
		}

		// Convert parameters
		if len(src.Components.Parameters) > 0 {
			dst.Parameters = make(map[string]*parser.Parameter)
			for name, param := range src.Components.Parameters {
				path := fmt.Sprintf("components.parameters.%s", name)
				convertedParam := c.convertOAS3ParameterToOAS2(param, result, path)
				if convertedParam != nil {
					dst.Parameters[name] = convertedParam
				}
			}
		}

		// Convert responses
		if len(src.Components.Responses) > 0 {
			dst.Responses = make(map[string]*parser.Response)
			for name, response := range src.Components.Responses {
				path := fmt.Sprintf("components.responses.%s", name)
				convertedResponse, produces := c.convertOAS3ResponseToOAS2(response, result, path)
				if convertedResponse != nil {
					dst.Responses[name] = convertedResponse
					// Merge produces into document-level produces
					dst.Produces = mergeStringArrays(dst.Produces, produces)
				}
			}
		}

		// Convert security schemes
		c.convertSecuritySchemes(src, dst, result)
	}

	// Convert paths
	for pathPattern, pathItem := range src.Paths {
		if pathItem == nil {
			continue
		}

		convertedPathItem := c.convertOAS3PathItemToOAS2(pathItem, dst, result, fmt.Sprintf("paths.%s", pathPattern))
		dst.Paths[pathPattern] = convertedPathItem
	}

	// Handle webhooks (OAS 3.1+)
	if len(src.Webhooks) > 0 {
		c.addIssue(result, "webhooks", "Webhooks are OAS 3.1+ only and cannot be converted to OAS 2.0", SeverityCritical)
	}

	// Handle external docs
	if src.ExternalDocs != nil {
		dst.ExternalDocs = src.ExternalDocs
	}

	// Global security is compatible
	if len(src.Security) > 0 {
		dst.Security = src.Security
	}

	// Rewrite all $ref paths from OAS 3.x to OAS 2.0 format
	c.rewriteAllRefsOAS3ToOAS2(dst)

	result.Document = dst
	return nil
}

// convertServersToHostBasePath converts OAS 3.x servers to OAS 2.0 host/basePath/schemes
func (c *Converter) convertServersToHostBasePath(src *parser.OAS3Document, dst *parser.OAS2Document, result *ConversionResult) {
	if len(src.Servers) == 0 {
		// No servers defined, use defaults
		dst.Host = "localhost"
		dst.BasePath = "/"
		dst.Schemes = []string{"https"}
		c.addIssue(result, "servers", "No servers defined in OAS 3.x document, using defaults", SeverityInfo)
		return
	}

	// Use the first server
	firstServer := src.Servers[0]

	// Parse the server URL
	host, basePath, schemes, err := parseServerURL(firstServer.URL)
	if err != nil {
		c.addIssueWithContext(result, "servers[0]",
			fmt.Sprintf("Failed to parse server URL: %v", err),
			"Using default values")
		dst.Host = "localhost"
		dst.BasePath = "/"
		dst.Schemes = []string{"https"}
		return
	}

	dst.Host = host
	dst.BasePath = basePath
	dst.Schemes = schemes

	// Warn about multiple servers
	if len(src.Servers) > 1 {
		c.addIssueWithContext(result, "servers",
			fmt.Sprintf("Multiple servers defined (%d), using only the first one", len(src.Servers)),
			"OAS 2.0 supports only a single host/basePath combination")
	}

	// Warn about server variables
	if len(firstServer.Variables) > 0 {
		c.addIssueWithContext(result, "servers[0].variables",
			"Server variables are not supported in OAS 2.0",
			"Variables have been removed from the server URL")
	}
}

// convertOAS3PathItemToOAS2 converts an OAS 3.x path item to OAS 2.0
func (c *Converter) convertOAS3PathItemToOAS2(src *parser.PathItem, doc *parser.OAS2Document, result *ConversionResult, pathPrefix string) *parser.PathItem {
	dst := &parser.PathItem{
		Summary:     src.Summary,
		Description: src.Description,
		Parameters:  c.convertParametersToOAS2(src.Parameters, result, fmt.Sprintf("%s.parameters", pathPrefix)),
	}

	// Convert each operation
	if src.Get != nil {
		dst.Get = c.convertOAS3OperationToOAS2(src.Get, doc, result, fmt.Sprintf("%s.get", pathPrefix))
	}
	if src.Put != nil {
		dst.Put = c.convertOAS3OperationToOAS2(src.Put, doc, result, fmt.Sprintf("%s.put", pathPrefix))
	}
	if src.Post != nil {
		dst.Post = c.convertOAS3OperationToOAS2(src.Post, doc, result, fmt.Sprintf("%s.post", pathPrefix))
	}
	if src.Delete != nil {
		dst.Delete = c.convertOAS3OperationToOAS2(src.Delete, doc, result, fmt.Sprintf("%s.delete", pathPrefix))
	}
	if src.Options != nil {
		dst.Options = c.convertOAS3OperationToOAS2(src.Options, doc, result, fmt.Sprintf("%s.options", pathPrefix))
	}
	if src.Head != nil {
		dst.Head = c.convertOAS3OperationToOAS2(src.Head, doc, result, fmt.Sprintf("%s.head", pathPrefix))
	}
	if src.Patch != nil {
		dst.Patch = c.convertOAS3OperationToOAS2(src.Patch, doc, result, fmt.Sprintf("%s.patch", pathPrefix))
	}

	// Trace is OAS 3.x only
	if src.Trace != nil {
		c.addIssue(result, fmt.Sprintf("%s.trace", pathPrefix),
			"TRACE method is OAS 3.x only and cannot be converted to OAS 2.0", SeverityCritical)
	}

	// Query is OAS 3.2+ only
	if src.Query != nil {
		c.addIssue(result, fmt.Sprintf("%s.query", pathPrefix),
			"QUERY method is OAS 3.2+ only and cannot be converted to OAS 2.0", SeverityCritical)
	}

	// AdditionalOperations (custom HTTP methods) are OAS 3.2+ only
	for method := range src.AdditionalOperations {
		c.addIssue(result, fmt.Sprintf("%s.additionalOperations.%s", pathPrefix, method),
			fmt.Sprintf("Custom HTTP method %s is OAS 3.2+ only and cannot be converted to OAS 2.0", method), SeverityCritical)
	}

	return dst
}

// convertOAS3OperationToOAS2 converts an OAS 3.x operation to OAS 2.0
func (c *Converter) convertOAS3OperationToOAS2(src *parser.Operation, doc *parser.OAS2Document, result *ConversionResult, opPath string) *parser.Operation {
	dst := &parser.Operation{
		Tags:         src.Tags,
		Summary:      src.Summary,
		Description:  src.Description,
		ExternalDocs: src.ExternalDocs,
		OperationID:  src.OperationID,
		Parameters:   c.convertParametersToOAS2(src.Parameters, result, fmt.Sprintf("%s.parameters", opPath)),
		Deprecated:   src.Deprecated,
		Security:     src.Security,
	}

	// Convert requestBody to body parameter and consumes
	if src.RequestBody != nil {
		bodyParam, consumes := c.convertOAS3RequestBodyToOAS2(src.RequestBody, result, opPath)
		if bodyParam != nil {
			dst.Parameters = append(dst.Parameters, bodyParam)
		}
		if len(consumes) > 0 {
			dst.Consumes = consumes
			// Merge into document-level consumes
			doc.Consumes = mergeStringArrays(doc.Consumes, consumes)
		}
	}

	// Convert responses
	if src.Responses != nil {
		dst.Responses = &parser.Responses{
			Codes: make(map[string]*parser.Response),
		}

		if src.Responses.Default != nil {
			convertedDefault, produces := c.convertOAS3ResponseToOAS2(src.Responses.Default, result, fmt.Sprintf("%s.responses.default", opPath))
			dst.Responses.Default = convertedDefault
			if len(produces) > 0 {
				dst.Produces = mergeStringArrays(dst.Produces, produces)
				doc.Produces = mergeStringArrays(doc.Produces, produces)
			}
		}

		for code, response := range src.Responses.Codes {
			convertedResponse, produces := c.convertOAS3ResponseToOAS2(response, result, fmt.Sprintf("%s.responses.%s", opPath, code))
			dst.Responses.Codes[code] = convertedResponse
			if len(produces) > 0 {
				dst.Produces = mergeStringArrays(dst.Produces, produces)
				doc.Produces = mergeStringArrays(doc.Produces, produces)
			}
		}
	}

	// Check for callbacks (OAS 3.x only)
	if len(src.Callbacks) > 0 {
		c.addIssue(result, opPath, "Operation contains callbacks which are not supported in OAS 2.0", SeverityCritical)
	}

	return dst
}

// convertOAS3RequestBodyToOAS2 converts an OAS 3.x requestBody to OAS 2.0 body parameter
func (c *Converter) convertOAS3RequestBodyToOAS2(requestBody *parser.RequestBody, result *ConversionResult, opPath string) (*parser.Parameter, []string) {
	if requestBody == nil || len(requestBody.Content) == 0 {
		return nil, nil
	}

	consumes := make([]string, 0, len(requestBody.Content))
	var firstMediaType string
	var firstContent *parser.MediaType

	// Collect all media types
	for mediaType, content := range requestBody.Content {
		consumes = append(consumes, mediaType)
		if firstMediaType == "" {
			firstMediaType = mediaType
			firstContent = content
		}
	}

	// Warn about multiple media types
	if len(requestBody.Content) > 1 {
		c.addIssueWithContext(result, fmt.Sprintf("%s.requestBody", opPath),
			fmt.Sprintf("RequestBody has multiple media types (%d), using first (%s)", len(requestBody.Content), firstMediaType),
			"OAS 2.0 body parameters have a single schema; use 'consumes' array to specify multiple content types")
	}

	// Create body parameter
	bodyParam := &parser.Parameter{
		Name:        "body",
		In:          "body",
		Description: requestBody.Description,
		Required:    requestBody.Required,
	}

	if firstContent != nil && firstContent.Schema != nil {
		schemaPath := fmt.Sprintf("%s.requestBody.content.%s.schema", opPath, firstMediaType)
		bodyParam.Schema = c.convertOAS3SchemaToOAS2(firstContent.Schema, result, schemaPath)
	}

	return bodyParam, consumes
}

// convertParametersToOAS2 converts a list of parameters from OAS 3.x to OAS 2.0
func (c *Converter) convertParametersToOAS2(params []*parser.Parameter, result *ConversionResult, path string) []*parser.Parameter {
	return c.convertParameterSlice(params, result, path, c.convertOAS3ParameterToOAS2)
}

// convertSecuritySchemes converts OAS 3.x components.securitySchemes to OAS 2.0 securityDefinitions
func (c *Converter) convertSecuritySchemes(src *parser.OAS3Document, dst *parser.OAS2Document, result *ConversionResult) {
	if src.Components == nil || len(src.Components.SecuritySchemes) == 0 {
		return
	}

	dst.SecurityDefinitions = make(map[string]*parser.SecurityScheme)

	for name, scheme := range src.Components.SecuritySchemes {
		path := fmt.Sprintf("components.securitySchemes.%s", name)

		secDef := &parser.SecurityScheme{
			Type:        scheme.Type,
			Description: scheme.Description,
			Name:        scheme.Name,
			In:          scheme.In,
		}

		// Convert HTTP to basic
		if scheme.Type == "http" {
			if scheme.Scheme == "basic" {
				secDef.Type = "basic"
			} else {
				c.addIssueWithContext(result, path,
					fmt.Sprintf("HTTP scheme '%s' may not be fully compatible with OAS 2.0", scheme.Scheme),
					"OAS 2.0 only supports 'basic' HTTP authentication natively")
				secDef.Type = "basic"
			}
		}

		// Convert OAuth2 flows
		if scheme.Type == "oauth2" && scheme.Flows != nil {
			// OAS 2.0 only supports a single flow
			flowCount := 0
			if scheme.Flows.Implicit != nil {
				flowCount++
				secDef.Flow = "implicit"
				secDef.AuthorizationURL = scheme.Flows.Implicit.AuthorizationURL
				secDef.Scopes = scheme.Flows.Implicit.Scopes
			}
			if scheme.Flows.Password != nil {
				flowCount++
				secDef.Flow = "password"
				secDef.TokenURL = scheme.Flows.Password.TokenURL
				secDef.Scopes = scheme.Flows.Password.Scopes
			}
			if scheme.Flows.ClientCredentials != nil {
				flowCount++
				secDef.Flow = "application"
				secDef.TokenURL = scheme.Flows.ClientCredentials.TokenURL
				secDef.Scopes = scheme.Flows.ClientCredentials.Scopes
			}
			if scheme.Flows.AuthorizationCode != nil {
				flowCount++
				secDef.Flow = "accessCode"
				secDef.AuthorizationURL = scheme.Flows.AuthorizationCode.AuthorizationURL
				secDef.TokenURL = scheme.Flows.AuthorizationCode.TokenURL
				secDef.Scopes = scheme.Flows.AuthorizationCode.Scopes
			}

			if flowCount > 1 {
				c.addIssueWithContext(result, path,
					fmt.Sprintf("Multiple OAuth2 flows defined (%d), using only one", flowCount),
					"OAS 2.0 supports only a single OAuth2 flow per security definition")
			}
			if flowCount == 0 {
				c.addIssue(result, path, "OAuth2 security scheme has no flows defined", SeverityWarning)
			}
		}

		// OpenID Connect is OAS 3.x only
		if scheme.Type == "openIdConnect" {
			c.addIssue(result, path, "OpenID Connect is OAS 3.x only and cannot be converted to OAS 2.0", SeverityCritical)
			continue
		}

		dst.SecurityDefinitions[name] = secDef
	}
}
