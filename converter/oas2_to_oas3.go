package converter

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// convertOAS2ToOAS3 converts an OAS 2.0 document to OAS 3.x
func (c *Converter) convertOAS2ToOAS3(parseResult parser.ParseResult, targetVersion parser.OASVersion, result *ConversionResult) error {
	src, ok := parseResult.OAS2Document()
	if !ok {
		return fmt.Errorf("source document is not an OAS2Document")
	}

	// Create the OAS 3.x document
	dst := &parser.OAS3Document{
		OpenAPI:    result.TargetVersion,
		OASVersion: targetVersion,
		Info:       src.Info,
		Servers:    c.convertServers(src, result),
		Paths:      make(map[string]*parser.PathItem),
		Tags:       src.Tags,
	}

	// Convert components
	if src.Definitions != nil || src.Parameters != nil || src.Responses != nil || src.SecurityDefinitions != nil {
		dst.Components = &parser.Components{
			Schemas:         make(map[string]*parser.Schema),
			Parameters:      make(map[string]*parser.Parameter),
			Responses:       make(map[string]*parser.Response),
			SecuritySchemes: make(map[string]*parser.SecurityScheme),
		}

		// Convert definitions to schemas
		for name, schema := range src.Definitions {
			dst.Components.Schemas[name] = c.convertOAS2SchemaToOAS3(schema)
		}

		// Convert parameters
		for name, param := range src.Parameters {
			path := fmt.Sprintf("parameters.%s", name)
			dst.Components.Parameters[name] = c.convertOAS2ParameterToOAS3(param, result, path)
		}

		// Convert responses
		for name, response := range src.Responses {
			dst.Components.Responses[name] = c.convertOAS2ResponseToOAS3Old(response, src.Produces)
		}

		// Convert security definitions
		c.convertSecurityDefinitions(src, dst, result)
	}

	// Convert paths
	for pathPattern, pathItem := range src.Paths {
		if pathItem == nil {
			continue
		}

		convertedPathItem := c.convertOAS2PathItemToOAS3(pathItem, src, result, fmt.Sprintf("paths.%s", pathPattern))
		dst.Paths[pathPattern] = convertedPathItem
	}

	// Handle external docs
	if src.ExternalDocs != nil {
		dst.ExternalDocs = src.ExternalDocs
	}

	// Global security is compatible
	if len(src.Security) > 0 {
		dst.Security = src.Security
	}

	// Rewrite all $ref paths from OAS 2.0 to OAS 3.x format
	c.rewriteAllRefsOAS2ToOAS3(dst)

	result.Document = dst
	return nil
}

// convertServers converts OAS 2.0 host/basePath/schemes to OAS 3.x servers
func (c *Converter) convertServers(src *parser.OAS2Document, result *ConversionResult) []*parser.Server {
	// Pre-allocate based on number of schemes (or 1 for default)
	schemeCount := len(src.Schemes)
	if schemeCount == 0 {
		schemeCount = 1
	}
	servers := make([]*parser.Server, 0, schemeCount)

	// If no host is specified, create a default server
	if src.Host == "" {
		servers = append(servers, &parser.Server{
			URL:         "/",
			Description: "Default server",
		})
		c.addIssue(result, "servers", "No host specified in OAS 2.0 document, using default server", SeverityInfo)
		return servers
	}

	schemes := src.Schemes
	if len(schemes) == 0 {
		schemes = []string{"https"}
	}

	basePath := src.BasePath
	if basePath == "" {
		basePath = "/"
	}

	// Create a server for each scheme
	for _, scheme := range schemes {
		serverURL := fmt.Sprintf("%s://%s%s", scheme, src.Host, basePath)
		servers = append(servers, &parser.Server{
			URL:         serverURL,
			Description: fmt.Sprintf("Server with %s scheme", scheme),
		})
	}

	return servers
}

// convertOAS2PathItemToOAS3 converts an OAS 2.0 path item to OAS 3.x
func (c *Converter) convertOAS2PathItemToOAS3(src *parser.PathItem, doc *parser.OAS2Document, result *ConversionResult, pathPrefix string) *parser.PathItem {
	// nil in, nil out...
	if src == nil {
		return nil
	}

	dst := &parser.PathItem{
		Summary:     src.Summary,
		Description: src.Description,
		Parameters:  c.convertParameters(src.Parameters, result, fmt.Sprintf("%s.parameters", pathPrefix)),
	}

	// Convert each standard operation using the shared helper
	convertStandardOperations(src, dst, pathPrefix, func(op *parser.Operation, path string) *parser.Operation {
		return c.convertOAS2OperationToOAS3(op, doc, result, path)
	})

	return dst
}

// convertOAS2OperationToOAS3 converts an OAS 2.0 operation to OAS 3.x
func (c *Converter) convertOAS2OperationToOAS3(src *parser.Operation, doc *parser.OAS2Document, result *ConversionResult, opPath string) *parser.Operation {
	dst := &parser.Operation{
		Tags:         src.Tags,
		Summary:      src.Summary,
		Description:  src.Description,
		ExternalDocs: src.ExternalDocs,
		OperationID:  src.OperationID,
		Parameters:   c.convertParameters(src.Parameters, result, fmt.Sprintf("%s.parameters", opPath)),
		Deprecated:   src.Deprecated,
		Security:     src.Security,
	}

	// Convert responses
	if src.Responses != nil {
		dst.Responses = &parser.Responses{
			Default: c.convertOAS2ResponseToOAS3Old(src.Responses.Default, c.getProduces(src, doc)),
			Codes:   make(map[string]*parser.Response),
		}

		for code, response := range src.Responses.Codes {
			dst.Responses.Codes[code] = c.convertOAS2ResponseToOAS3Old(response, c.getProduces(src, doc))
		}
	}

	// Convert consumes to requestBody
	hasBodyParam := false
	for _, param := range src.Parameters {
		if param != nil && param.In == "body" {
			hasBodyParam = true
			break
		}
	}

	if hasBodyParam {
		dst.RequestBody = c.convertOAS2RequestBody(src, doc)
		// Remove body parameters from the parameters list in dst
		filteredParams := make([]*parser.Parameter, 0)
		for _, param := range dst.Parameters {
			if param != nil && param.In != "body" {
				filteredParams = append(filteredParams, param)
			}
		}
		dst.Parameters = filteredParams
	}

	// Convert formData parameters to requestBody
	hasFormData := false
	for _, param := range src.Parameters {
		if param != nil && param.In == "formData" {
			hasFormData = true
			break
		}
	}

	if hasFormData {
		if dst.RequestBody != nil {
			c.addIssueWithContext(result, opPath,
				"Operation has both body and formData parameters",
				"OAS 2.0 spec forbids this; formData parameters ignored")
		} else {
			dst.RequestBody = c.convertOAS2FormDataToRequestBody(src, doc)
			filteredParams := make([]*parser.Parameter, 0, len(dst.Parameters))
			for _, param := range dst.Parameters {
				if param != nil && param.In != "formData" {
					filteredParams = append(filteredParams, param)
				}
			}
			dst.Parameters = filteredParams
		}
	}

	return dst
}

// convertOAS2RequestBody converts OAS 2.0 body parameters and consumes to OAS 3.x requestBody
func (c *Converter) convertOAS2RequestBody(src *parser.Operation, doc *parser.OAS2Document) *parser.RequestBody {
	// Find body parameter
	var bodyParam *parser.Parameter
	for _, param := range src.Parameters {
		if param != nil && param.In == "body" {
			bodyParam = param
			break
		}
	}

	if bodyParam == nil {
		return nil
	}

	requestBody := &parser.RequestBody{
		Description: bodyParam.Description,
		Required:    bodyParam.Required,
		Content:     make(map[string]*parser.MediaType),
	}

	// Get consumes media types
	consumes := c.getConsumes(src, doc)
	if len(consumes) == 0 {
		consumes = []string{getDefaultMediaType()}
	}

	// Create content for each media type
	for _, mediaType := range consumes {
		requestBody.Content[mediaType] = &parser.MediaType{
			Schema: c.convertOAS2SchemaToOAS3(bodyParam.Schema),
		}
	}

	return requestBody
}

// convertOAS2FormDataToRequestBody converts OAS 2.0 formData parameters to OAS 3.x requestBody.
func (c *Converter) convertOAS2FormDataToRequestBody(src *parser.Operation, doc *parser.OAS2Document) *parser.RequestBody {
	var formDataParams []*parser.Parameter
	hasFile := false
	for _, param := range src.Parameters {
		if param != nil && param.In == "formData" {
			formDataParams = append(formDataParams, param)
			if param.Type == "file" {
				hasFile = true
			}
		}
	}
	if len(formDataParams) == 0 {
		return nil
	}

	schema := &parser.Schema{
		Type:       "object",
		Properties: make(map[string]*parser.Schema),
	}
	var required []string
	for _, param := range formDataParams {
		propSchema := &parser.Schema{}
		switch param.Type {
		case "file":
			propSchema.Type = "string"
			propSchema.Format = "binary"
		case "array":
			propSchema.Type = "array"
			propSchema.Format = param.Format
			if param.Items != nil {
				propSchema.Items = convertOAS2ItemsToSchema(param.Items)
			}
		default:
			propSchema.Type = param.Type
			propSchema.Format = param.Format
		}
		// Transfer validation properties from OAS 2.0 parameter to schema
		propSchema.Default = param.Default
		propSchema.Enum = param.Enum
		propSchema.Maximum = param.Maximum
		propSchema.Minimum = param.Minimum
		propSchema.MaxLength = param.MaxLength
		propSchema.MinLength = param.MinLength
		propSchema.Pattern = param.Pattern
		propSchema.MaxItems = param.MaxItems
		propSchema.MinItems = param.MinItems
		propSchema.UniqueItems = param.UniqueItems
		propSchema.MultipleOf = param.MultipleOf
		if param.ExclusiveMaximum {
			propSchema.ExclusiveMaximum = true
		}
		if param.ExclusiveMinimum {
			propSchema.ExclusiveMinimum = true
		}
		if param.Description != "" {
			propSchema.Description = param.Description
		}
		schema.Properties[param.Name] = propSchema
		if param.Required {
			required = append(required, param.Name)
		}
	}
	if len(required) > 0 {
		schema.Required = required
	}

	contentType := "application/x-www-form-urlencoded"
	if hasFile {
		contentType = "multipart/form-data"
	} else {
		consumes := c.getConsumes(src, doc)
		for _, ct := range consumes {
			if strings.HasPrefix(ct, "multipart/") {
				contentType = ct
				break
			}
		}
	}

	return &parser.RequestBody{
		Required: len(required) > 0,
		Content: map[string]*parser.MediaType{
			contentType: {Schema: schema},
		},
	}
}

// convertOAS2ItemsToSchema converts an OAS 2.0 Items object to an OAS 3.x Schema.
func convertOAS2ItemsToSchema(items *parser.Items) *parser.Schema {
	if items == nil {
		return nil
	}
	s := &parser.Schema{
		Type:        items.Type,
		Format:      items.Format,
		Default:     items.Default,
		Enum:        items.Enum,
		Maximum:     items.Maximum,
		Minimum:     items.Minimum,
		MaxLength:   items.MaxLength,
		MinLength:   items.MinLength,
		Pattern:     items.Pattern,
		MaxItems:    items.MaxItems,
		MinItems:    items.MinItems,
		UniqueItems: items.UniqueItems,
		MultipleOf:  items.MultipleOf,
	}
	if items.ExclusiveMaximum {
		s.ExclusiveMaximum = true
	}
	if items.ExclusiveMinimum {
		s.ExclusiveMinimum = true
	}
	if items.Items != nil {
		s.Items = convertOAS2ItemsToSchema(items.Items)
	}
	return s
}

// convertParameters converts a list of parameters from OAS 2.0 to OAS 3.x
func (c *Converter) convertParameters(params []*parser.Parameter, result *ConversionResult, path string) []*parser.Parameter {
	return c.convertParameterSlice(params, result, path, c.convertOAS2ParameterToOAS3)
}

// convertSecurityDefinitions converts OAS 2.0 securityDefinitions to OAS 3.x components.securitySchemes
func (c *Converter) convertSecurityDefinitions(src *parser.OAS2Document, dst *parser.OAS3Document, result *ConversionResult) {
	for name, secDef := range src.SecurityDefinitions {
		path := fmt.Sprintf("securityDefinitions.%s", name)

		scheme := &parser.SecurityScheme{
			Type:        secDef.Type,
			Description: secDef.Description,
			Name:        secDef.Name,
			In:          secDef.In,
		}

		// Convert OAuth2 flows
		if secDef.Type == "oauth2" {
			scheme.Flows = &parser.OAuthFlows{}

			switch secDef.Flow {
			case "implicit":
				scheme.Flows.Implicit = &parser.OAuthFlow{
					AuthorizationURL: secDef.AuthorizationURL,
					Scopes:           secDef.Scopes,
				}
			case "password":
				scheme.Flows.Password = &parser.OAuthFlow{
					TokenURL: secDef.TokenURL,
					Scopes:   secDef.Scopes,
				}
			case "application":
				scheme.Flows.ClientCredentials = &parser.OAuthFlow{
					TokenURL: secDef.TokenURL,
					Scopes:   secDef.Scopes,
				}
			case "accessCode":
				scheme.Flows.AuthorizationCode = &parser.OAuthFlow{
					AuthorizationURL: secDef.AuthorizationURL,
					TokenURL:         secDef.TokenURL,
					Scopes:           secDef.Scopes,
				}
			default:
				c.addIssueWithContext(result, path,
					fmt.Sprintf("Unknown OAuth2 flow type: %s", secDef.Flow),
					"This may not convert correctly to OAS 3.x")
			}
		}

		// Convert basic/apiKey (these are compatible)
		if secDef.Type == "basic" {
			scheme.Type = "http"
			scheme.Scheme = "basic"
		}

		dst.Components.SecuritySchemes[name] = scheme
	}
}

// getConsumes returns the consumes array for an operation, falling back to document-level consumes
func (c *Converter) getConsumes(op *parser.Operation, doc *parser.OAS2Document) []string {
	if len(op.Consumes) > 0 {
		return op.Consumes
	}
	return doc.Consumes
}

// getProduces returns the produces array for an operation, falling back to document-level produces
func (c *Converter) getProduces(op *parser.Operation, doc *parser.OAS2Document) []string {
	if len(op.Produces) > 0 {
		return op.Produces
	}
	return doc.Produces
}
