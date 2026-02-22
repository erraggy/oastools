// This file implements $ref rewriting between OAS 2.0 and OAS 3.x formats.
// It provides functions to traverse documents and update all reference paths
// when converting between specification versions.

package converter

import (
	"strings"

	"github.com/erraggy/oastools/parser"
)

// refMapping defines a prefix substitution for $ref rewriting.
type refMapping struct {
	from string
	to   string
}

// oas2ToOAS3Mappings maps OAS 2.0 $ref prefixes to their OAS 3.x equivalents.
var oas2ToOAS3Mappings = []refMapping{
	{"#/definitions/", "#/components/schemas/"},
	{"#/parameters/", "#/components/parameters/"},
	{"#/responses/", "#/components/responses/"},
	{"#/securityDefinitions/", "#/components/securitySchemes/"},
}

// oas3ToOAS2Mappings maps OAS 3.x $ref prefixes to their OAS 2.0 equivalents.
var oas3ToOAS2Mappings = []refMapping{
	{"#/components/schemas/", "#/definitions/"},
	{"#/components/parameters/", "#/parameters/"},
	{"#/components/responses/", "#/responses/"},
	{"#/components/securitySchemes/", "#/securityDefinitions/"},
}

// rewriteRefOAS2ToOAS3 rewrites an OAS 2.0 $ref to OAS 3.x format
// Only rewrites local references (starting with #/)
func rewriteRefOAS2ToOAS3(ref string) string {
	if !strings.HasPrefix(ref, "#/") {
		return ref
	}

	for _, m := range oas2ToOAS3Mappings {
		if strings.HasPrefix(ref, m.from) {
			return m.to + ref[len(m.from):]
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

	for _, m := range oas3ToOAS2Mappings {
		if strings.HasPrefix(ref, m.from) {
			return m.to + ref[len(m.from):]
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

	// Handle polymorphic fields with type assertion.
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

	// Discriminator mapping contains schema refs
	if schema.Discriminator != nil {
		for key, ref := range schema.Discriminator.Mapping {
			schema.Discriminator.Mapping[key] = rewrite(ref)
		}
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

// rewriteResponseRefsOAS2ToOAS3 rewrites $ref values in a response from OAS 2.0 to OAS 3.x format
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
