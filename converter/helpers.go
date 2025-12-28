package converter

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/erraggy/oastools/internal/schemautil"
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

// deepCopyOAS3Document creates a deep copy of an OAS3 document.
// Uses generated DeepCopy methods for type-safe, efficient copying.
func (c *Converter) deepCopyOAS3Document(src *parser.OAS3Document) (*parser.OAS3Document, error) {
	if src == nil {
		return nil, fmt.Errorf("cannot copy nil document")
	}
	return src.DeepCopy(), nil
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

// deepCopySchema creates a deep copy of a schema.
// Uses generated DeepCopy methods for type-safe, efficient copying.
func (c *Converter) deepCopySchema(src *parser.Schema) *parser.Schema {
	if src == nil {
		return nil
	}
	return src.DeepCopy()
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
			// Type can be string or []string (OAS 3.1+), extract primary type
			converted.Type = schemautil.GetPrimaryType(schema)
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

// refRewriter is a function that rewrites a $ref string to a different format.
// It is used by [walkSchemaRefs] to apply version-specific reference transformations.
// The function receives the original $ref value and returns the rewritten value.
type refRewriter func(ref string) string

// walkSchemaRefs recursively walks a schema and rewrites all $ref values using the provided rewriter function.
// This is a generic traversal function that handles all nested schema locations.
func walkSchemaRefs(schema *parser.Schema, rewrite refRewriter) {
	if schema == nil {
		return
	}

	// Rewrite the $ref in this schema
	if schema.Ref != "" {
		schema.Ref = rewrite(schema.Ref)
	}

	// Recursively rewrite nested schemas in properties
	for _, propSchema := range schema.Properties {
		walkSchemaRefs(propSchema, rewrite)
	}

	for _, propSchema := range schema.PatternProperties {
		walkSchemaRefs(propSchema, rewrite)
	}

	// Handle interface{} typed fields with type assertion.
	// These can be bool (OAS 3.1+) or *Schema - only *Schema needs traversal.
	if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
		walkSchemaRefs(addProps, rewrite)
	}

	if items, ok := schema.Items.(*parser.Schema); ok {
		walkSchemaRefs(items, rewrite)
	}

	// Composition keywords
	for _, subSchema := range schema.AllOf {
		walkSchemaRefs(subSchema, rewrite)
	}

	for _, subSchema := range schema.AnyOf {
		walkSchemaRefs(subSchema, rewrite)
	}

	for _, subSchema := range schema.OneOf {
		walkSchemaRefs(subSchema, rewrite)
	}

	walkSchemaRefs(schema.Not, rewrite)

	// Array-related keywords
	if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
		walkSchemaRefs(addItems, rewrite)
	}

	for _, prefixItem := range schema.PrefixItems {
		walkSchemaRefs(prefixItem, rewrite)
	}

	walkSchemaRefs(schema.Contains, rewrite)

	// Object validation keywords
	walkSchemaRefs(schema.PropertyNames, rewrite)

	for _, depSchema := range schema.DependentSchemas {
		walkSchemaRefs(depSchema, rewrite)
	}

	// JSON Schema 2020-12 unevaluated keywords (can be bool or *Schema)
	if unevalProps, ok := schema.UnevaluatedProperties.(*parser.Schema); ok {
		walkSchemaRefs(unevalProps, rewrite)
	}

	if unevalItems, ok := schema.UnevaluatedItems.(*parser.Schema); ok {
		walkSchemaRefs(unevalItems, rewrite)
	}

	// JSON Schema 2020-12 content keywords
	walkSchemaRefs(schema.ContentSchema, rewrite)

	// Conditional keywords
	walkSchemaRefs(schema.If, rewrite)
	walkSchemaRefs(schema.Then, rewrite)
	walkSchemaRefs(schema.Else, rewrite)

	// Schema definitions
	for _, defSchema := range schema.Defs {
		walkSchemaRefs(defSchema, rewrite)
	}
}

// rewriteSchemaRefsOAS2ToOAS3 recursively rewrites all $ref values in a schema from OAS 2.0 to OAS 3.x format.
func rewriteSchemaRefsOAS2ToOAS3(schema *parser.Schema) {
	walkSchemaRefs(schema, rewriteRefOAS2ToOAS3)
}

// rewriteSchemaRefsOAS3ToOAS2 recursively rewrites all $ref values in a schema from OAS 3.x to OAS 2.0 format.
func rewriteSchemaRefsOAS3ToOAS2(schema *parser.Schema) {
	walkSchemaRefs(schema, rewriteRefOAS3ToOAS2)
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
	operations := parser.GetOperations(pathItem, parser.OASVersion20)
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
	// Note: We use OASVersion300 here as a representative OAS 3.x version since this function
	// is only called during OAS3→OAS2 conversion and the QUERY method (OAS 3.2+) cannot be
	// converted to OAS 2.0 anyway (handled separately in convertOAS3PathItemToOAS2).
	operations := parser.GetOperations(pathItem, parser.OASVersion300)
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

// httpMethod defines an HTTP method with accessors for a PathItem.
// This enables table-driven operation conversion without repetitive if-statements.
type httpMethod struct {
	name   string
	getter func(*parser.PathItem) *parser.Operation
	setter func(*parser.PathItem, *parser.Operation)
}

// standardHTTPMethods are HTTP methods common to OAS 2.0 and OAS 3.x.
// TRACE (OAS 3.0+), QUERY (OAS 3.2+), and AdditionalOperations are handled separately.
var standardHTTPMethods = []httpMethod{
	{"get", func(p *parser.PathItem) *parser.Operation { return p.Get }, func(p *parser.PathItem, op *parser.Operation) { p.Get = op }},
	{"put", func(p *parser.PathItem) *parser.Operation { return p.Put }, func(p *parser.PathItem, op *parser.Operation) { p.Put = op }},
	{"post", func(p *parser.PathItem) *parser.Operation { return p.Post }, func(p *parser.PathItem, op *parser.Operation) { p.Post = op }},
	{"delete", func(p *parser.PathItem) *parser.Operation { return p.Delete }, func(p *parser.PathItem, op *parser.Operation) { p.Delete = op }},
	{"options", func(p *parser.PathItem) *parser.Operation { return p.Options }, func(p *parser.PathItem, op *parser.Operation) { p.Options = op }},
	{"head", func(p *parser.PathItem) *parser.Operation { return p.Head }, func(p *parser.PathItem, op *parser.Operation) { p.Head = op }},
	{"patch", func(p *parser.PathItem) *parser.Operation { return p.Patch }, func(p *parser.PathItem, op *parser.Operation) { p.Patch = op }},
}

// convertStandardOperations converts all standard HTTP method operations from src to dst.
// The convert function is called for each non-nil operation with its path prefix.
// This is the shared implementation for both OAS2→OAS3 and OAS3→OAS2 path item conversion.
func convertStandardOperations(src, dst *parser.PathItem, pathPrefix string, convert func(*parser.Operation, string) *parser.Operation) {
	for _, method := range standardHTTPMethods {
		if op := method.getter(src); op != nil {
			method.setter(dst, convert(op, fmt.Sprintf("%s.%s", pathPrefix, method.name)))
		}
	}
}

// paramConvertFunc is the signature for parameter conversion functions.
type paramConvertFunc func(param *parser.Parameter, result *ConversionResult, path string) *parser.Parameter

// convertParameterSlice converts a slice of parameters using the provided conversion function.
// This helper reduces duplication between OAS2→OAS3 and OAS3→OAS2 parameter list conversion.
func (c *Converter) convertParameterSlice(params []*parser.Parameter, result *ConversionResult, path string, convert paramConvertFunc) []*parser.Parameter {
	if len(params) == 0 {
		return nil
	}

	converted := make([]*parser.Parameter, 0, len(params))
	for i, param := range params {
		if param == nil {
			continue
		}
		paramPath := fmt.Sprintf("%s[%d]", path, i)
		convertedParam := convert(param, result, paramPath)
		if convertedParam != nil {
			converted = append(converted, convertedParam)
		}
	}

	return converted
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
