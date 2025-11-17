package converter

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/erraggy/oastools/parser"
)

var (
	// Path parameters like: /foo/{bar}/v1
	pathParamRegx = regexp.MustCompile(`\{[^}]+}`)

	// Map OAS 3.x locations to the Regexp for the prefix of the OAS 2.0 locations
	refRegxMapWithOAS3AsNew = map[string]*regexp.Regexp{
		"#/components/schemas/":         regexp.MustCompile(`^` + regexp.QuoteMeta("#/definitions/")),
		"#/components/parameters/":      regexp.MustCompile(`^` + regexp.QuoteMeta("#/parameters/")),
		"#/components/responses/":       regexp.MustCompile(`^` + regexp.QuoteMeta("#/responses/")),
		"#/components/securitySchemes/": regexp.MustCompile(`^` + regexp.QuoteMeta("#/securityDefinitions/")),
	}

	// Map OAS 2.0 locations to the Regexp for the prefix of the OAS 3.x locations
	refRegxMapWithSwaggerAsNew = map[string]*regexp.Regexp{
		"#/definitions/":         regexp.MustCompile(`^` + regexp.QuoteMeta("#/components/schemas/")),
		"#/parameters/":          regexp.MustCompile(`^` + regexp.QuoteMeta("#/components/parameters/")),
		"#/responses/":           regexp.MustCompile(`^` + regexp.QuoteMeta("#/components/responses/")),
		"#/securityDefinitions/": regexp.MustCompile(`^` + regexp.QuoteMeta("#/components/securitySchemes/")),
	}
)

// deepCopyOAS3Document creates a deep copy of an OAS3 document
func (c *Converter) deepCopyOAS3Document(src *parser.OAS3Document) (*parser.OAS3Document, error) {
	// Use JSON marshal/unmarshal for deep copy
	data, err := json.Marshal(src)
	if err != nil {
		// This should never happen with valid documents
		return nil, fmt.Errorf("failed to marshal src document: %w", err)
	}

	var dst parser.OAS3Document
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dst document: %w", err)
	}

	// Restore OASVersion which may not round-trip through JSON
	dst.OASVersion = src.OASVersion

	return &dst, nil
}

// parseServerURL extracts host, basePath, and schemes from an OAS 3.x server URL
// Returns host, basePath, schemes, and error
func parseServerURL(serverURL string) (host, basePath string, schemes []string, err error) {
	// Handle server variables by replacing them with defaults or placeholders like:
	// http://example.com/foo/{parameter}/bar ==> http://example.com/foo/placeholder/bar
	// For simplicity, we'll strip variables for now and parse the base URL since the rest of the path is ignored here
	cleanURL := pathParamRegx.ReplaceAllString(serverURL, "placeholder")

	// Parse the URL
	u, err := url.Parse(cleanURL)
	if err != nil {
		return "", "", nil, fmt.Errorf("invalid server URL: %w", err)
	}

	// Extract components
	if u.Scheme != "" {
		schemes = []string{u.Scheme}
	}

	host = u.Host
	basePath = u.Path
	if basePath == "" {
		basePath = "/"
	}

	return host, basePath, schemes, nil
}

// convertOAS2SchemaToOAS3 converts an OAS 2.0 schema to OAS 3.x format
func (c *Converter) convertOAS2SchemaToOAS3(schema *parser.Schema) *parser.Schema {
	if schema == nil {
		return nil
	}

	// Deep copy to avoid mutations
	converted := c.deepCopySchema(schema)

	// Rewrite all $ref paths from OAS 2.0 to OAS 3.x format
	rewriteSchemaRefsOAS2ToOAS3(converted)

	return converted
}

// convertOAS3SchemaToOAS2 converts an OAS 3.x schema to OAS 2.0 format
func (c *Converter) convertOAS3SchemaToOAS2(schema *parser.Schema, result *ConversionResult, path string) *parser.Schema {
	if schema == nil {
		return nil
	}

	// Check for OAS 3.1+ features that may not be compatible with OAS 2.0
	converted := c.deepCopySchema(schema)

	// Check for nullable (OAS 3.0+)
	if schema.Nullable {
		c.addIssueWithContext(result, path, "Schema uses 'nullable' which is OAS 3.0+",
			"Consider using 'x-nullable' extension for OAS 2.0 compatibility")
	}

	// Rewrite all $ref paths from OAS 3.x to OAS 2.0 format
	rewriteSchemaRefsOAS3ToOAS2(converted)

	return converted
}

// deepCopySchema creates a deep copy of a schema
func (c *Converter) deepCopySchema(src *parser.Schema) *parser.Schema {
	if src == nil {
		return nil
	}

	// Use JSON marshal/unmarshal for deep copy
	data, err := json.Marshal(src)
	if err != nil {
		return src
	}

	var dst parser.Schema
	if err := json.Unmarshal(data, &dst); err != nil {
		return src
	}

	return &dst
}

// getDefaultMediaType returns a default media type if none is specified
func getDefaultMediaType() string {
	return "application/json"
}

// mergeStringArrays merges multiple string arrays, removing duplicates
func mergeStringArrays(arrays ...[]string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, arr := range arrays {
		for _, item := range arr {
			if !seen[item] {
				seen[item] = true
				result = append(result, item)
			}
		}
	}

	return result
}

// convertOAS2ParameterToOAS3 converts an OAS 2.0 parameter to OAS 3.x format
func (c *Converter) convertOAS2ParameterToOAS3(param *parser.Parameter, result *ConversionResult, path string) *parser.Parameter {
	if param == nil {
		return nil
	}

	converted := &parser.Parameter{
		Ref:             param.Ref, // Copy $ref field
		Name:            param.Name,
		In:              param.In,
		Description:     param.Description,
		Required:        param.Required,
		Deprecated:      param.Deprecated,
		AllowEmptyValue: param.AllowEmptyValue,
	}

	// Handle schema
	if param.Schema != nil {
		converted.Schema = c.convertOAS2SchemaToOAS3(param.Schema)
	} else if param.Type != "" {
		// Convert type/format to schema
		converted.Schema = &parser.Schema{
			Type:   param.Type,
			Format: param.Format,
		}

		// Handle collection format
		if param.CollectionFormat != "" && param.CollectionFormat != "csv" {
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Parameter uses collectionFormat '%s'", param.CollectionFormat),
				"OAS 3.x uses 'style' and 'explode' instead; 'csv' format maps to style=form")
		}
	}

	// AllowEmptyValue was removed in OAS 3.0
	if param.AllowEmptyValue {
		c.addIssueWithContext(result, path, "Parameter uses 'allowEmptyValue'",
			"This field was removed in OAS 3.0")
	}

	return converted
}

// convertOAS3ParameterToOAS2 converts an OAS 3.x parameter to OAS 2.0 format
func (c *Converter) convertOAS3ParameterToOAS2(param *parser.Parameter, result *ConversionResult, path string) *parser.Parameter {
	if param == nil {
		return nil
	}

	// Cookie parameters are not supported in OAS 2.0
	if param.In == "cookie" {
		c.addIssue(result, path, "Cookie parameters are not supported in OAS 2.0", SeverityCritical)
		return nil
	}

	converted := &parser.Parameter{
		Ref:         param.Ref, // Copy $ref field
		Name:        param.Name,
		In:          param.In,
		Description: param.Description,
		Required:    param.Required,
	}

	// Convert schema to type/format
	if param.Schema != nil {
		schema := c.convertOAS3SchemaToOAS2(param.Schema, result, fmt.Sprintf("%s.schema", path))
		converted.Schema = schema

		// Extract type and format for non-body parameters
		if param.In != "body" && schema != nil {
			// Type can be string or []string, extract as string if possible
			if typeStr, ok := schema.Type.(string); ok {
				converted.Type = typeStr
			}
			converted.Format = schema.Format
		}
	}

	// Check for OAS 3.x style/explode parameters
	if param.Style != "" {
		c.addIssueWithContext(result, path,
			fmt.Sprintf("Parameter uses style '%s'", param.Style),
			"OAS 2.0 uses 'collectionFormat' instead")
	}

	return converted
}

// convertOAS2ResponseToOAS3Old converts an OAS 2.0 response to OAS 3.x format
func (c *Converter) convertOAS2ResponseToOAS3Old(response *parser.Response, produces []string) *parser.Response {
	if response == nil {
		return nil
	}

	converted := &parser.Response{
		Description: response.Description,
		Headers:     response.Headers,
	}

	// Convert schema to content
	if response.Schema != nil {
		converted.Content = make(map[string]*parser.MediaType)

		// Use produces array or default to application/json
		mediaTypes := produces
		if len(mediaTypes) == 0 {
			mediaTypes = []string{getDefaultMediaType()}
		}

		for _, mediaType := range mediaTypes {
			converted.Content[mediaType] = &parser.MediaType{
				Schema: c.convertOAS2SchemaToOAS3(response.Schema),
			}
		}
	}

	return converted
}

// convertOAS3ResponseToOAS2 converts an OAS 3.x response to OAS 2.0 format
func (c *Converter) convertOAS3ResponseToOAS2(response *parser.Response, result *ConversionResult, path string) (*parser.Response, []string) {
	if response == nil {
		return nil, nil
	}

	converted := &parser.Response{
		Description: response.Description,
		Headers:     response.Headers,
	}

	var produces []string

	// Convert content to schema
	if len(response.Content) > 0 {
		// Take the first media type's schema
		var firstMediaType string
		var firstContent *parser.MediaType

		for mt, content := range response.Content {
			if firstMediaType == "" {
				firstMediaType = mt
				firstContent = content
			}
			produces = append(produces, mt)
		}

		if len(response.Content) > 1 {
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Response has multiple media types (%d), using first (%s)", len(response.Content), firstMediaType),
				"OAS 2.0 responses have a single schema; use 'produces' array to specify multiple content types")
		}

		if firstContent != nil && firstContent.Schema != nil {
			converted.Schema = c.convertOAS3SchemaToOAS2(firstContent.Schema, result, fmt.Sprintf("%s.content.%s.schema", path, firstMediaType))
		}
	}

	// Check for links (OAS 3.x only)
	if len(response.Links) > 0 {
		c.addIssue(result, path, "Response contains links which are not supported in OAS 2.0", SeverityCritical)
	}

	return converted, produces
}

// rewriteRefOAS2ToOAS3 rewrites an OAS 2.0 $ref to OAS 3.x format
// Only rewrites local references (starting with #/)
func rewriteRefOAS2ToOAS3(ref string) string {
	if !strings.HasPrefix(ref, "#/") {
		return ref
	}

	// iterate all regexp mappings and if found on the specified ref, replace it with the new prefix
	for newOAS3Prefix, swaggerPrefixRegX := range refRegxMapWithOAS3AsNew {
		if swaggerPrefixRegX.MatchString(ref) {
			return swaggerPrefixRegX.ReplaceAllString(ref, newOAS3Prefix)
		}
	}

	// Unknown reference format, return as-is
	return ref
}

// rewriteRefOAS3ToOAS2 rewrites an OAS 3.x $ref to OAS 2.0 format
// Only rewrites local references (starting with #/)
func rewriteRefOAS3ToOAS2(ref string) string {
	if !strings.HasPrefix(ref, "#/") {
		return ref
	}

	// iterate all regexp mappings and if found on the specified ref, replace it with the new prefix
	for newSwaggerPrefix, oas3PrefixRegX := range refRegxMapWithSwaggerAsNew {
		if oas3PrefixRegX.MatchString(ref) {
			return oas3PrefixRegX.ReplaceAllString(ref, newSwaggerPrefix)
		}
	}

	// Unknown reference format, return as-is
	return ref
}

// rewriteSchemaRefsOAS2ToOAS3 recursively rewrites all $ref values in a schema from OAS 2.0 to OAS 3.x format
func rewriteSchemaRefsOAS2ToOAS3(schema *parser.Schema) {
	if schema == nil {
		return
	}

	// Rewrite the $ref in this schema
	if schema.Ref != "" {
		schema.Ref = rewriteRefOAS2ToOAS3(schema.Ref)
	}

	// Recursively rewrite nested schemas
	for _, propSchema := range schema.Properties {
		rewriteSchemaRefsOAS2ToOAS3(propSchema)
	}

	for _, propSchema := range schema.PatternProperties {
		rewriteSchemaRefsOAS2ToOAS3(propSchema)
	}

	if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
		rewriteSchemaRefsOAS2ToOAS3(addProps)
	}

	if items, ok := schema.Items.(*parser.Schema); ok {
		rewriteSchemaRefsOAS2ToOAS3(items)
	}

	for _, subSchema := range schema.AllOf {
		rewriteSchemaRefsOAS2ToOAS3(subSchema)
	}

	for _, subSchema := range schema.AnyOf {
		rewriteSchemaRefsOAS2ToOAS3(subSchema)
	}

	for _, subSchema := range schema.OneOf {
		rewriteSchemaRefsOAS2ToOAS3(subSchema)
	}

	rewriteSchemaRefsOAS2ToOAS3(schema.Not)

	if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
		rewriteSchemaRefsOAS2ToOAS3(addItems)
	}

	for _, prefixItem := range schema.PrefixItems {
		rewriteSchemaRefsOAS2ToOAS3(prefixItem)
	}

	rewriteSchemaRefsOAS2ToOAS3(schema.Contains)
	rewriteSchemaRefsOAS2ToOAS3(schema.PropertyNames)

	for _, depSchema := range schema.DependentSchemas {
		rewriteSchemaRefsOAS2ToOAS3(depSchema)
	}

	rewriteSchemaRefsOAS2ToOAS3(schema.If)
	rewriteSchemaRefsOAS2ToOAS3(schema.Then)
	rewriteSchemaRefsOAS2ToOAS3(schema.Else)

	for _, defSchema := range schema.Defs {
		rewriteSchemaRefsOAS2ToOAS3(defSchema)
	}
}

// rewriteSchemaRefsOAS3ToOAS2 recursively rewrites all $ref values in a schema from OAS 3.x to OAS 2.0 format
func rewriteSchemaRefsOAS3ToOAS2(schema *parser.Schema) {
	if schema == nil {
		return
	}

	// Rewrite the $ref in this schema
	if schema.Ref != "" {
		schema.Ref = rewriteRefOAS3ToOAS2(schema.Ref)
	}

	// Recursively rewrite nested schemas
	for _, propSchema := range schema.Properties {
		rewriteSchemaRefsOAS3ToOAS2(propSchema)
	}

	for _, propSchema := range schema.PatternProperties {
		rewriteSchemaRefsOAS3ToOAS2(propSchema)
	}

	if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
		rewriteSchemaRefsOAS3ToOAS2(addProps)
	}

	if items, ok := schema.Items.(*parser.Schema); ok {
		rewriteSchemaRefsOAS3ToOAS2(items)
	}

	for _, subSchema := range schema.AllOf {
		rewriteSchemaRefsOAS3ToOAS2(subSchema)
	}

	for _, subSchema := range schema.AnyOf {
		rewriteSchemaRefsOAS3ToOAS2(subSchema)
	}

	for _, subSchema := range schema.OneOf {
		rewriteSchemaRefsOAS3ToOAS2(subSchema)
	}

	rewriteSchemaRefsOAS3ToOAS2(schema.Not)

	if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
		rewriteSchemaRefsOAS3ToOAS2(addItems)
	}

	for _, prefixItem := range schema.PrefixItems {
		rewriteSchemaRefsOAS3ToOAS2(prefixItem)
	}

	rewriteSchemaRefsOAS3ToOAS2(schema.Contains)
	rewriteSchemaRefsOAS3ToOAS2(schema.PropertyNames)

	for _, depSchema := range schema.DependentSchemas {
		rewriteSchemaRefsOAS3ToOAS2(depSchema)
	}

	rewriteSchemaRefsOAS3ToOAS2(schema.If)
	rewriteSchemaRefsOAS3ToOAS2(schema.Then)
	rewriteSchemaRefsOAS3ToOAS2(schema.Else)

	for _, defSchema := range schema.Defs {
		rewriteSchemaRefsOAS3ToOAS2(defSchema)
	}
}

// rewriteParameterRefsOAS2ToOAS3 rewrites $ref values in a parameter from OAS 2.0 to OAS 3.x format
func rewriteParameterRefsOAS2ToOAS3(param *parser.Parameter) {
	if param == nil {
		return
	}

	if param.Ref != "" {
		param.Ref = rewriteRefOAS2ToOAS3(param.Ref)
	}

	// Rewrite refs in the schema
	rewriteSchemaRefsOAS2ToOAS3(param.Schema)
}

// rewriteParameterRefsOAS3ToOAS2 rewrites $ref values in a parameter from OAS 3.x to OAS 2.0 format
func rewriteParameterRefsOAS3ToOAS2(param *parser.Parameter) {
	if param == nil {
		return
	}

	if param.Ref != "" {
		param.Ref = rewriteRefOAS3ToOAS2(param.Ref)
	}

	// Rewrite refs in the schema
	rewriteSchemaRefsOAS3ToOAS2(param.Schema)

	// Rewrite refs in content media types (OAS 3.x)
	for _, mediaType := range param.Content {
		if mediaType != nil {
			rewriteSchemaRefsOAS3ToOAS2(mediaType.Schema)
		}
	}
}

// rewrite ResponseRefsOAS2ToOAS3 rewrites $ref values in a response from OAS 2.0 to OAS 3.x format
func rewriteResponseRefsOAS2ToOAS3(response *parser.Response) {
	if response == nil {
		return
	}

	if response.Ref != "" {
		response.Ref = rewriteRefOAS2ToOAS3(response.Ref)
	}

	// Rewrite refs in the schema
	rewriteSchemaRefsOAS2ToOAS3(response.Schema)

	// Rewrite refs in headers
	for _, header := range response.Headers {
		if header != nil {
			if header.Ref != "" {
				header.Ref = rewriteRefOAS2ToOAS3(header.Ref)
			}
			rewriteSchemaRefsOAS2ToOAS3(header.Schema)
		}
	}
}

// rewriteResponseRefsOAS3ToOAS2 rewrites $ref values in a response from OAS 3.x to OAS 2.0 format
func rewriteResponseRefsOAS3ToOAS2(response *parser.Response) {
	if response == nil {
		return
	}

	if response.Ref != "" {
		response.Ref = rewriteRefOAS3ToOAS2(response.Ref)
	}

	// Rewrite refs in the schema
	rewriteSchemaRefsOAS3ToOAS2(response.Schema)

	// Rewrite refs in content media types (OAS 3.x)
	for _, mediaType := range response.Content {
		if mediaType != nil {
			rewriteSchemaRefsOAS3ToOAS2(mediaType.Schema)
		}
	}

	// Rewrite refs in headers
	for _, header := range response.Headers {
		if header != nil {
			if header.Ref != "" {
				header.Ref = rewriteRefOAS3ToOAS2(header.Ref)
			}
			rewriteSchemaRefsOAS3ToOAS2(header.Schema)

			// Rewrite refs in header content (OAS 3.x)
			for _, mediaType := range header.Content {
				if mediaType != nil {
					rewriteSchemaRefsOAS3ToOAS2(mediaType.Schema)
				}
			}
		}
	}

	// Rewrite refs in links (OAS 3.x)
	for _, link := range response.Links {
		if link != nil && link.Ref != "" {
			link.Ref = rewriteRefOAS3ToOAS2(link.Ref)
		}
	}
}

// rewriteRequestBodyRefsOAS2ToOAS3 rewrites $ref values in a request body from OAS 2.0 to OAS 3.x format
func rewriteRequestBodyRefsOAS2ToOAS3(requestBody *parser.RequestBody) {
	if requestBody == nil {
		return
	}

	if requestBody.Ref != "" {
		requestBody.Ref = rewriteRefOAS2ToOAS3(requestBody.Ref)
	}

	// Rewrite refs in content media types
	for _, mediaType := range requestBody.Content {
		if mediaType != nil {
			rewriteSchemaRefsOAS2ToOAS3(mediaType.Schema)
		}
	}
}

// rewriteRequestBodyRefsOAS3ToOAS2 rewrites $ref values in a request body from OAS 3.x to OAS 2.0 format
func rewriteRequestBodyRefsOAS3ToOAS2(requestBody *parser.RequestBody) {
	if requestBody == nil {
		return
	}

	if requestBody.Ref != "" {
		requestBody.Ref = rewriteRefOAS3ToOAS2(requestBody.Ref)
	}

	// Rewrite refs in content media types
	for _, mediaType := range requestBody.Content {
		if mediaType != nil {
			rewriteSchemaRefsOAS3ToOAS2(mediaType.Schema)
		}
	}
}

// rewritePathItemRefsOAS2ToOAS3 rewrites $ref values in a path item from OAS 2.0 to OAS 3.x format
func rewritePathItemRefsOAS2ToOAS3(pathItem *parser.PathItem) {
	if pathItem == nil {
		return
	}

	if pathItem.Ref != "" {
		pathItem.Ref = rewriteRefOAS2ToOAS3(pathItem.Ref)
	}

	// Rewrite refs in parameters
	for _, param := range pathItem.Parameters {
		rewriteParameterRefsOAS2ToOAS3(param)
	}

	// Rewrite refs in each operation
	operations := parser.GetOAS2Operations(pathItem)
	for _, op := range operations {
		if op == nil {
			continue
		}

		// Rewrite operation parameters
		for _, param := range op.Parameters {
			rewriteParameterRefsOAS2ToOAS3(param)
		}

		// Rewrite operation responses
		if op.Responses != nil {
			rewriteResponseRefsOAS2ToOAS3(op.Responses.Default)

			for _, response := range op.Responses.Codes {
				rewriteResponseRefsOAS2ToOAS3(response)
			}
		}
	}
}

// rewritePathItemRefsOAS3ToOAS2 rewrites $ref values in a path item from OAS 3.x to OAS 2.0 format
func rewritePathItemRefsOAS3ToOAS2(pathItem *parser.PathItem) {
	if pathItem == nil {
		return
	}

	if pathItem.Ref != "" {
		pathItem.Ref = rewriteRefOAS3ToOAS2(pathItem.Ref)
	}

	// Rewrite refs in parameters
	for _, param := range pathItem.Parameters {
		rewriteParameterRefsOAS3ToOAS2(param)
	}

	// Rewrite refs in each operation
	operations := parser.GetOAS3Operations(pathItem)
	for _, op := range operations {
		if op == nil {
			continue
		}

		// Rewrite operation parameters
		for _, param := range op.Parameters {
			rewriteParameterRefsOAS3ToOAS2(param)
		}

		// Rewrite request body
		rewriteRequestBodyRefsOAS3ToOAS2(op.RequestBody)

		// Rewrite operation responses
		if op.Responses != nil {
			rewriteResponseRefsOAS3ToOAS2(op.Responses.Default)

			for _, response := range op.Responses.Codes {
				rewriteResponseRefsOAS3ToOAS2(response)
			}
		}
	}
}

// rewriteAllRefsOAS2ToOAS3 rewrites all $ref values in an OAS 3.x document from OAS 2.0 to OAS 3.x format
func (c *Converter) rewriteAllRefsOAS2ToOAS3(doc *parser.OAS3Document) {
	if doc == nil {
		return
	}

	// Rewrite refs in components
	if doc.Components != nil {
		for _, schema := range doc.Components.Schemas {
			rewriteSchemaRefsOAS2ToOAS3(schema)
		}

		for _, param := range doc.Components.Parameters {
			rewriteParameterRefsOAS2ToOAS3(param)
		}

		for _, response := range doc.Components.Responses {
			rewriteResponseRefsOAS2ToOAS3(response)
		}

		for _, requestBody := range doc.Components.RequestBodies {
			rewriteRequestBodyRefsOAS2ToOAS3(requestBody)
		}

		for _, header := range doc.Components.Headers {
			if header != nil {
				if header.Ref != "" {
					header.Ref = rewriteRefOAS2ToOAS3(header.Ref)
				}
				rewriteSchemaRefsOAS2ToOAS3(header.Schema)
			}
		}

		for _, securityScheme := range doc.Components.SecuritySchemes {
			if securityScheme != nil && securityScheme.Ref != "" {
				securityScheme.Ref = rewriteRefOAS2ToOAS3(securityScheme.Ref)
			}
		}
	}

	// Rewrite refs in paths
	for _, pathItem := range doc.Paths {
		rewritePathItemRefsOAS2ToOAS3(pathItem)
	}

	// Rewrite refs in webhooks (OAS 3.1+)
	for _, pathItem := range doc.Webhooks {
		rewritePathItemRefsOAS2ToOAS3(pathItem)
	}
}

// rewriteAllRefsOAS3ToOAS2 rewrites all $ref values in an OAS 2.0 document from OAS 3.x to OAS 2.0 format
func (c *Converter) rewriteAllRefsOAS3ToOAS2(doc *parser.OAS2Document) {
	if doc == nil {
		return
	}

	// Rewrite refs in definitions
	for _, schema := range doc.Definitions {
		rewriteSchemaRefsOAS3ToOAS2(schema)
	}

	// Rewrite refs in parameters
	for _, param := range doc.Parameters {
		rewriteParameterRefsOAS3ToOAS2(param)
	}

	// Rewrite refs in responses
	for _, response := range doc.Responses {
		rewriteResponseRefsOAS3ToOAS2(response)
	}

	// Rewrite refs in security definitions
	for _, securityScheme := range doc.SecurityDefinitions {
		if securityScheme != nil && securityScheme.Ref != "" {
			securityScheme.Ref = rewriteRefOAS3ToOAS2(securityScheme.Ref)
		}
	}

	// Rewrite refs in paths
	for _, pathItem := range doc.Paths {
		rewritePathItemRefsOAS3ToOAS2(pathItem)
	}
}
