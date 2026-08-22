package converter

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// getDefaultMediaType returns a default media type if none is specified
func getDefaultMediaType() string {
	return mediaTypeJSON
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
		Extra:           parser.DeepCopyExtensions(param.Extra),
	}

	// Handle schema
	if param.Schema != nil {
		converted.Schema = c.convertOAS2SchemaToOAS3(param.Schema, result.TargetOASVersion, result, path+".schema")
	} else if param.Type != "" {
		converted.Schema = c.oas2TypedValueToSchema(oas2TypedValueOfParameter(param), "Parameter", result, path)
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
		Extra:       parser.DeepCopyExtensions(param.Extra),
	}

	// Convert schema to type/format
	if param.Schema != nil {
		schema := c.convertOAS3SchemaToOAS2(param.Schema, result, fmt.Sprintf("%s.schema", path))

		if param.In == "body" {
			// The one position OAS 2.0 describes with a Schema Object.
			converted.Schema = schema
		} else if schema != nil {
			// Elsewhere OAS 2.0 defines no 'schema' field, so it is read into
			// the type declaration rather than carried alongside it.
			c.oas2TypedValueFromSchema(schema, "Parameter", result, path).applyToParameter(converted)
		}
	}

	// Fallback: infer type from composite schemas
	if converted.Type == "" && param.In != "body" {
		inferred := inferTypeFromSchema(param.Schema)
		if inferred != "" {
			converted.Type = c.oas2PrimitiveType(inferred, "Parameter", result, path)
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Inferred type '%s' from composite schema", inferred),
				"OAS 2.0 requires explicit type for non-body parameters")
		} else {
			converted.Type = "string"
			c.addIssueWithContext(result, path,
				"Could not infer type from schema, defaulting to 'string'",
				"OAS 2.0 requires explicit type for non-body parameters")
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

// inferTypeFromSchema walks allOf/oneOf/anyOf to find a concrete type. A branch
// OAS 2.0 can name wins over an earlier one it cannot, since the alternatives
// are interchangeable and only one survives the demotion.
func inferTypeFromSchema(schema *parser.Schema) string {
	if schema == nil {
		return ""
	}

	var fallback string
	for _, branch := range [][]*parser.Schema{schema.AllOf, schema.OneOf, schema.AnyOf} {
		for _, sub := range branch {
			t := schemautil.GetPrimaryType(sub)
			if t == "" {
				continue
			}
			if oas2PrimitiveTypes[t] {
				return t
			}
			if fallback == "" {
				fallback = t
			}
		}
	}
	return fallback
}

// resolveHeaderRef resolves a #/components/headers/* ref by inlining the definition.
func (c *Converter) resolveHeaderRef(ref string, result *ConversionResult, path string) *parser.Header {
	if !strings.HasPrefix(ref, componentHeadersPrefix) {
		return nil
	}
	if c.sourceHeaders == nil {
		return nil
	}
	name := ref[len(componentHeadersPrefix):]
	header, ok := c.sourceHeaders[name]
	if !ok {
		c.addIssueWithContext(result, path,
			fmt.Sprintf("Unresolved header ref: %s", ref),
			"Header not found in components.headers")
		return nil
	}
	c.addIssue(result, path,
		fmt.Sprintf("Inlined component header ref %s", ref), SeverityInfo)
	inlined := header.DeepCopy()
	inlined.Ref = ""
	return inlined
}

// convertOAS2ResponseToOAS3Old converts an OAS 2.0 response to OAS 3.x format
func (c *Converter) convertOAS2ResponseToOAS3Old(response *parser.Response, produces []string, targetVersion parser.OASVersion, result *ConversionResult, path string) *parser.Response {
	if response == nil {
		return nil
	}

	converted := &parser.Response{
		Description: response.Description,
		Headers:     c.convertHeadersToOAS3(response.Headers, result, path),
		Extra:       parser.DeepCopyExtensions(response.Extra),
	}

	// Convert schema to content
	if response.Schema != nil {
		converted.Content = make(map[string]*parser.MediaType)

		// Use produces array or default to application/json
		mediaTypes := produces
		if len(mediaTypes) == 0 {
			mediaTypes = []string{getDefaultMediaType()}
		}

		// Convert schema once; deep-copy per media type to avoid shared mutation.
		convertedSchema := c.convertOAS2SchemaToOAS3(response.Schema, targetVersion, result, path+".schema")
		for _, mediaType := range mediaTypes {
			converted.Content[mediaType] = &parser.MediaType{
				Schema: convertedSchema.DeepCopy(),
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
		Extra:       parser.DeepCopyExtensions(response.Extra),
	}

	if len(response.Headers) > 0 {
		converted.Headers = c.convertHeadersToOAS2(response.Headers, result, path)
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
	{httputil.MethodGet, func(p *parser.PathItem) *parser.Operation { return p.Get }, func(p *parser.PathItem, op *parser.Operation) { p.Get = op }},
	{httputil.MethodPut, func(p *parser.PathItem) *parser.Operation { return p.Put }, func(p *parser.PathItem, op *parser.Operation) { p.Put = op }},
	{httputil.MethodPost, func(p *parser.PathItem) *parser.Operation { return p.Post }, func(p *parser.PathItem, op *parser.Operation) { p.Post = op }},
	{httputil.MethodDelete, func(p *parser.PathItem) *parser.Operation { return p.Delete }, func(p *parser.PathItem, op *parser.Operation) { p.Delete = op }},
	{httputil.MethodOptions, func(p *parser.PathItem) *parser.Operation { return p.Options }, func(p *parser.PathItem, op *parser.Operation) { p.Options = op }},
	{httputil.MethodHead, func(p *parser.PathItem) *parser.Operation { return p.Head }, func(p *parser.PathItem, op *parser.Operation) { p.Head = op }},
	{httputil.MethodPatch, func(p *parser.PathItem) *parser.Operation { return p.Patch }, func(p *parser.PathItem, op *parser.Operation) { p.Patch = op }},
}

// convertStandardOperations converts all standard HTTP method operations from src to dst.
// The convert function is called for each non-nil operation with its path prefix.
// This is the shared implementation for both OAS2->OAS3 and OAS3->OAS2 path item conversion.
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
// This helper reduces duplication between OAS2->OAS3 and OAS3->OAS2 parameter list conversion.
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
