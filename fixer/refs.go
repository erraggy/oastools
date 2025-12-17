// refs.go implements reference collection and analysis for the fixer package.
// It provides utilities to traverse OpenAPI documents and collect all $ref values,
// which is needed for both generic schema name fixing and schema/component pruning.
package fixer

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// RefType categorizes reference targets by their component type.
type RefType int

const (
	// RefTypeSchema represents references to schema definitions
	RefTypeSchema RefType = iota
	// RefTypeParameter represents references to parameter definitions
	RefTypeParameter
	// RefTypeResponse represents references to response definitions
	RefTypeResponse
	// RefTypeRequestBody represents references to request body definitions (OAS 3.x only)
	RefTypeRequestBody
	// RefTypeHeader represents references to header definitions
	RefTypeHeader
	// RefTypeSecurityScheme represents references to security scheme definitions
	RefTypeSecurityScheme
	// RefTypeLink represents references to link definitions (OAS 3.x only)
	RefTypeLink
	// RefTypeCallback represents references to callback definitions (OAS 3.x only)
	RefTypeCallback
	// RefTypeExample represents references to example definitions (OAS 3.x only)
	RefTypeExample
	// RefTypePathItem represents references to path item definitions (OAS 3.1+ only)
	RefTypePathItem
)

// String returns the string representation of a RefType.
func (rt RefType) String() string {
	switch rt {
	case RefTypeSchema:
		return "schema"
	case RefTypeParameter:
		return "parameter"
	case RefTypeResponse:
		return "response"
	case RefTypeRequestBody:
		return "requestBody"
	case RefTypeHeader:
		return "header"
	case RefTypeSecurityScheme:
		return "securityScheme"
	case RefTypeLink:
		return "link"
	case RefTypeCallback:
		return "callback"
	case RefTypeExample:
		return "example"
	case RefTypePathItem:
		return "pathItem"
	default:
		return "unknown"
	}
}

// RefCollector traverses OpenAPI documents to collect all $ref values.
// It tracks where references appear in the document and categorizes them by type.
type RefCollector struct {
	// Refs maps reference paths to their locations in the document.
	// Key: normalized reference path (e.g., "#/components/schemas/Pet")
	// Value: list of JSON paths where the reference appears
	Refs map[string][]string

	// RefsByType categorizes references by their target type.
	// Key: RefType (schema, parameter, etc.)
	// Value: set of reference paths of that type
	RefsByType map[RefType]map[string]bool

	// visited tracks processed schemas for circular reference handling.
	visited map[*parser.Schema]bool
}

// NewRefCollector creates a new RefCollector instance.
func NewRefCollector() *RefCollector {
	return &RefCollector{
		Refs:       make(map[string][]string),
		RefsByType: make(map[RefType]map[string]bool),
		visited:    make(map[*parser.Schema]bool),
	}
}

// addRef records a reference at the given location.
func (c *RefCollector) addRef(ref, location string, refType RefType) {
	if ref == "" {
		return
	}

	// Record the location where this ref appears
	c.Refs[ref] = append(c.Refs[ref], location)

	// Record by type
	if c.RefsByType[refType] == nil {
		c.RefsByType[refType] = make(map[string]bool)
	}
	c.RefsByType[refType][ref] = true
}

// CollectOAS2 collects all references from an OAS 2.0 document.
func (c *RefCollector) CollectOAS2(doc *parser.OAS2Document) {
	if doc == nil {
		return
	}

	// Collect from paths
	for pathKey, pathItem := range doc.Paths {
		c.collectPathItemRefs(pathItem, fmt.Sprintf("paths.%s", pathKey), parser.OASVersion20)
	}

	// Collect from definitions (schemas)
	for name, schema := range doc.Definitions {
		c.collectSchemaRefs(schema, fmt.Sprintf("definitions.%s", name))
	}

	// Collect from global parameters
	for name, param := range doc.Parameters {
		c.collectParameterRefs(param, fmt.Sprintf("parameters.%s", name), parser.OASVersion20)
	}

	// Collect from global responses
	for name, resp := range doc.Responses {
		c.collectResponseRefs(resp, fmt.Sprintf("responses.%s", name), parser.OASVersion20)
	}

	// Collect from security definitions
	for name, scheme := range doc.SecurityDefinitions {
		if scheme != nil && scheme.Ref != "" {
			c.addRef(scheme.Ref, fmt.Sprintf("securityDefinitions.%s", name), RefTypeSecurityScheme)
		}
	}
}

// CollectOAS3 collects all references from an OAS 3.x document.
func (c *RefCollector) CollectOAS3(doc *parser.OAS3Document) {
	if doc == nil {
		return
	}

	// Collect from paths
	for pathKey, pathItem := range doc.Paths {
		c.collectPathItemRefs(pathItem, fmt.Sprintf("paths.%s", pathKey), doc.OASVersion)
	}

	// Collect from webhooks (OAS 3.1+)
	for name, pathItem := range doc.Webhooks {
		c.collectPathItemRefs(pathItem, fmt.Sprintf("webhooks.%s", name), doc.OASVersion)
	}

	// Collect from components
	if doc.Components != nil {
		c.collectComponentsRefs(doc.Components, doc.OASVersion)
	}
}

// collectComponentsRefs collects references from the components object.
func (c *RefCollector) collectComponentsRefs(comp *parser.Components, version parser.OASVersion) {
	if comp == nil {
		return
	}

	// Schemas
	for name, schema := range comp.Schemas {
		c.collectSchemaRefs(schema, fmt.Sprintf("components.schemas.%s", name))
	}

	// Parameters
	for name, param := range comp.Parameters {
		c.collectParameterRefs(param, fmt.Sprintf("components.parameters.%s", name), version)
	}

	// Responses
	for name, resp := range comp.Responses {
		c.collectResponseRefs(resp, fmt.Sprintf("components.responses.%s", name), version)
	}

	// Request bodies
	for name, reqBody := range comp.RequestBodies {
		c.collectRequestBodyRefs(reqBody, fmt.Sprintf("components.requestBodies.%s", name))
	}

	// Headers
	for name, header := range comp.Headers {
		c.collectHeaderRefs(header, fmt.Sprintf("components.headers.%s", name), version)
	}

	// Security schemes
	for name, scheme := range comp.SecuritySchemes {
		if scheme != nil && scheme.Ref != "" {
			c.addRef(scheme.Ref, fmt.Sprintf("components.securitySchemes.%s", name), RefTypeSecurityScheme)
		}
	}

	// Links
	for name, link := range comp.Links {
		c.collectLinkRefs(link, fmt.Sprintf("components.links.%s", name))
	}

	// Callbacks
	for name, callback := range comp.Callbacks {
		c.collectCallbackRefs(callback, fmt.Sprintf("components.callbacks.%s", name), version)
	}

	// Examples
	for name, example := range comp.Examples {
		if example != nil && example.Ref != "" {
			c.addRef(example.Ref, fmt.Sprintf("components.examples.%s", name), RefTypeExample)
		}
	}

	// Path items (OAS 3.1+)
	for name, pathItem := range comp.PathItems {
		c.collectPathItemRefs(pathItem, fmt.Sprintf("components.pathItems.%s", name), version)
	}
}

// collectPathItemRefs collects references from a path item and its operations.
func (c *RefCollector) collectPathItemRefs(pathItem *parser.PathItem, path string, version parser.OASVersion) {
	if pathItem == nil {
		return
	}

	// PathItem can have its own $ref
	if pathItem.Ref != "" {
		c.addRef(pathItem.Ref, path, RefTypePathItem)
	}

	// Collect from path-level parameters
	for i, param := range pathItem.Parameters {
		c.collectParameterRefs(param, fmt.Sprintf("%s.parameters[%d]", path, i), version)
	}

	// Collect from all operations
	ops := parser.GetOperations(pathItem, version)
	for method, op := range ops {
		if op != nil {
			c.collectOperationRefs(op, fmt.Sprintf("%s.%s", path, method), version)
		}
	}
}

// collectOperationRefs collects references from an operation.
func (c *RefCollector) collectOperationRefs(op *parser.Operation, path string, version parser.OASVersion) {
	if op == nil {
		return
	}

	// Collect from parameters
	for i, param := range op.Parameters {
		c.collectParameterRefs(param, fmt.Sprintf("%s.parameters[%d]", path, i), version)
	}

	// Collect from request body (OAS 3.x)
	if op.RequestBody != nil {
		c.collectRequestBodyRefs(op.RequestBody, fmt.Sprintf("%s.requestBody", path))
	}

	// Collect from responses
	if op.Responses != nil {
		c.collectResponsesRefs(op.Responses, fmt.Sprintf("%s.responses", path), version)
	}

	// Collect from callbacks (OAS 3.x)
	for name, callback := range op.Callbacks {
		c.collectCallbackRefs(callback, fmt.Sprintf("%s.callbacks.%s", path, name), version)
	}
}

// collectResponsesRefs collects references from a responses container.
func (c *RefCollector) collectResponsesRefs(responses *parser.Responses, path string, version parser.OASVersion) {
	if responses == nil {
		return
	}

	// Default response
	if responses.Default != nil {
		c.collectResponseRefs(responses.Default, fmt.Sprintf("%s.default", path), version)
	}

	// Status code responses
	for code, resp := range responses.Codes {
		c.collectResponseRefs(resp, fmt.Sprintf("%s.%s", path, code), version)
	}
}

// collectResponseRefs collects references from a response.
func (c *RefCollector) collectResponseRefs(resp *parser.Response, path string, version parser.OASVersion) {
	if resp == nil {
		return
	}

	// Response can have $ref
	if resp.Ref != "" {
		c.addRef(resp.Ref, path, RefTypeResponse)
	}

	// Headers
	for name, header := range resp.Headers {
		c.collectHeaderRefs(header, fmt.Sprintf("%s.headers.%s", path, name), version)
	}

	// Content (OAS 3.x)
	for mediaType, mt := range resp.Content {
		c.collectMediaTypeRefs(mt, fmt.Sprintf("%s.content.%s", path, mediaType))
	}

	// Links (OAS 3.x)
	for name, link := range resp.Links {
		c.collectLinkRefs(link, fmt.Sprintf("%s.links.%s", path, name))
	}

	// Schema (OAS 2.0)
	if resp.Schema != nil {
		c.collectSchemaRefs(resp.Schema, fmt.Sprintf("%s.schema", path))
	}
}

// collectParameterRefs collects references from a parameter.
func (c *RefCollector) collectParameterRefs(param *parser.Parameter, path string, version parser.OASVersion) {
	if param == nil {
		return
	}

	// Parameter can have $ref
	if param.Ref != "" {
		c.addRef(param.Ref, path, RefTypeParameter)
	}

	// Schema (OAS 3.x)
	if param.Schema != nil {
		c.collectSchemaRefs(param.Schema, fmt.Sprintf("%s.schema", path))
	}

	// Content (OAS 3.x)
	for mediaType, mt := range param.Content {
		c.collectMediaTypeRefs(mt, fmt.Sprintf("%s.content.%s", path, mediaType))
	}

	// Examples (OAS 3.x)
	for name, example := range param.Examples {
		if example != nil && example.Ref != "" {
			c.addRef(example.Ref, fmt.Sprintf("%s.examples.%s", path, name), RefTypeExample)
		}
	}

	// Items for OAS 2.0 array parameters
	if version == parser.OASVersion20 && param.Items != nil {
		c.collectItemsRefs(param.Items, fmt.Sprintf("%s.items", path))
	}
}

// collectItemsRefs collects references from OAS 2.0 Items.
func (c *RefCollector) collectItemsRefs(items *parser.Items, path string) {
	if items == nil {
		return
	}
	// Items can contain nested items for array of arrays
	if items.Items != nil {
		c.collectItemsRefs(items.Items, fmt.Sprintf("%s.items", path))
	}
}

// collectRequestBodyRefs collects references from a request body.
func (c *RefCollector) collectRequestBodyRefs(reqBody *parser.RequestBody, path string) {
	if reqBody == nil {
		return
	}

	// RequestBody can have $ref
	if reqBody.Ref != "" {
		c.addRef(reqBody.Ref, path, RefTypeRequestBody)
	}

	// Content
	for mediaType, mt := range reqBody.Content {
		c.collectMediaTypeRefs(mt, fmt.Sprintf("%s.content.%s", path, mediaType))
	}
}

// collectHeaderRefs collects references from a header.
func (c *RefCollector) collectHeaderRefs(header *parser.Header, path string, version parser.OASVersion) {
	if header == nil {
		return
	}

	// Header can have $ref
	if header.Ref != "" {
		c.addRef(header.Ref, path, RefTypeHeader)
	}

	// Schema (OAS 3.x)
	if header.Schema != nil {
		c.collectSchemaRefs(header.Schema, fmt.Sprintf("%s.schema", path))
	}

	// Content (OAS 3.x)
	for mediaType, mt := range header.Content {
		c.collectMediaTypeRefs(mt, fmt.Sprintf("%s.content.%s", path, mediaType))
	}

	// Examples (OAS 3.x)
	for name, example := range header.Examples {
		if example != nil && example.Ref != "" {
			c.addRef(example.Ref, fmt.Sprintf("%s.examples.%s", path, name), RefTypeExample)
		}
	}

	// Items for OAS 2.0 headers
	if version == parser.OASVersion20 && header.Items != nil {
		c.collectItemsRefs(header.Items, fmt.Sprintf("%s.items", path))
	}
}

// collectMediaTypeRefs collects references from a media type.
func (c *RefCollector) collectMediaTypeRefs(mt *parser.MediaType, path string) {
	if mt == nil {
		return
	}

	// Schema
	if mt.Schema != nil {
		c.collectSchemaRefs(mt.Schema, fmt.Sprintf("%s.schema", path))
	}

	// Examples
	for name, example := range mt.Examples {
		if example != nil && example.Ref != "" {
			c.addRef(example.Ref, fmt.Sprintf("%s.examples.%s", path, name), RefTypeExample)
		}
	}

	// Encoding headers
	for encName, encoding := range mt.Encoding {
		if encoding != nil {
			for headerName, header := range encoding.Headers {
				c.collectHeaderRefs(header, fmt.Sprintf("%s.encoding.%s.headers.%s", path, encName, headerName), parser.OASVersion300)
			}
		}
	}
}

// collectLinkRefs collects references from a link.
func (c *RefCollector) collectLinkRefs(link *parser.Link, path string) {
	if link == nil {
		return
	}

	// Link can have $ref
	if link.Ref != "" {
		c.addRef(link.Ref, path, RefTypeLink)
	}

	// OperationRef is a runtime expression, not a JSON reference
	// but we should track operationRef for completeness
}

// collectCallbackRefs collects references from a callback.
func (c *RefCollector) collectCallbackRefs(callback *parser.Callback, path string, version parser.OASVersion) {
	if callback == nil {
		return
	}

	// Callback is a map of expressions to path items
	for expr, pathItem := range *callback {
		c.collectPathItemRefs(pathItem, fmt.Sprintf("%s.%s", path, expr), version)
	}
}

// collectSchemaRefs recursively collects references from a schema.
func (c *RefCollector) collectSchemaRefs(schema *parser.Schema, path string) {
	if schema == nil {
		return
	}

	// Circular reference protection
	if c.visited[schema] {
		return
	}
	c.visited[schema] = true
	defer delete(c.visited, schema)

	// Schema can have $ref
	if schema.Ref != "" {
		c.addRef(schema.Ref, path, RefTypeSchema)
	}

	// Properties
	for propName, propSchema := range schema.Properties {
		c.collectSchemaRefs(propSchema, fmt.Sprintf("%s.properties.%s", path, propName))
	}

	// AdditionalProperties (can be *Schema or bool)
	if schema.AdditionalProperties != nil {
		if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
			c.collectSchemaRefs(addProps, fmt.Sprintf("%s.additionalProperties", path))
		} else if addPropsMap, ok := schema.AdditionalProperties.(map[string]any); ok {
			// Fallback: extract refs from raw map (polymorphic field may remain as map)
			c.collectRefsFromMap(addPropsMap, fmt.Sprintf("%s.additionalProperties", path))
		}
	}

	// Items (can be *Schema or bool in OAS 3.1+)
	if schema.Items != nil {
		if items, ok := schema.Items.(*parser.Schema); ok {
			c.collectSchemaRefs(items, fmt.Sprintf("%s.items", path))
		} else if itemsMap, ok := schema.Items.(map[string]any); ok {
			// Fallback: extract refs from raw map (polymorphic field may remain as map)
			c.collectRefsFromMap(itemsMap, fmt.Sprintf("%s.items", path))
		}
	}

	// AdditionalItems (can be *Schema or bool)
	if schema.AdditionalItems != nil {
		if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
			c.collectSchemaRefs(addItems, fmt.Sprintf("%s.additionalItems", path))
		} else if addItemsMap, ok := schema.AdditionalItems.(map[string]any); ok {
			// Fallback: extract refs from raw map (polymorphic field may remain as map)
			c.collectRefsFromMap(addItemsMap, fmt.Sprintf("%s.additionalItems", path))
		}
	}

	// Schema composition
	for i, s := range schema.AllOf {
		c.collectSchemaRefs(s, fmt.Sprintf("%s.allOf[%d]", path, i))
	}
	for i, s := range schema.AnyOf {
		c.collectSchemaRefs(s, fmt.Sprintf("%s.anyOf[%d]", path, i))
	}
	for i, s := range schema.OneOf {
		c.collectSchemaRefs(s, fmt.Sprintf("%s.oneOf[%d]", path, i))
	}
	if schema.Not != nil {
		c.collectSchemaRefs(schema.Not, fmt.Sprintf("%s.not", path))
	}

	// OAS 3.1+ / JSON Schema Draft 2020-12 fields
	for i, s := range schema.PrefixItems {
		c.collectSchemaRefs(s, fmt.Sprintf("%s.prefixItems[%d]", path, i))
	}
	if schema.Contains != nil {
		c.collectSchemaRefs(schema.Contains, fmt.Sprintf("%s.contains", path))
	}
	if schema.PropertyNames != nil {
		c.collectSchemaRefs(schema.PropertyNames, fmt.Sprintf("%s.propertyNames", path))
	}
	for name, depSchema := range schema.DependentSchemas {
		c.collectSchemaRefs(depSchema, fmt.Sprintf("%s.dependentSchemas.%s", path, name))
	}

	// Conditional schemas (OAS 3.1+)
	if schema.If != nil {
		c.collectSchemaRefs(schema.If, fmt.Sprintf("%s.if", path))
	}
	if schema.Then != nil {
		c.collectSchemaRefs(schema.Then, fmt.Sprintf("%s.then", path))
	}
	if schema.Else != nil {
		c.collectSchemaRefs(schema.Else, fmt.Sprintf("%s.else", path))
	}

	// $defs (OAS 3.1+)
	for name, defSchema := range schema.Defs {
		c.collectSchemaRefs(defSchema, fmt.Sprintf("%s.$defs.%s", path, name))
	}

	// Pattern properties
	for pattern, propSchema := range schema.PatternProperties {
		c.collectSchemaRefs(propSchema, fmt.Sprintf("%s.patternProperties.%s", path, pattern))
	}

	// Discriminator mapping values are references
	if schema.Discriminator != nil {
		for key, ref := range schema.Discriminator.Mapping {
			c.addRef(ref, fmt.Sprintf("%s.discriminator.mapping.%s", path, key), RefTypeSchema)
		}
	}
}

// maxRefCollectionDepth is the maximum recursion depth for collectRefsFromMap.
// This prevents stack overflow from malformed or circular map structures.
const maxRefCollectionDepth = 100

// collectRefsFromMap extracts schema references from a raw map[string]any.
// This handles polymorphic schema fields (Items, AdditionalProperties, etc.) that may
// remain as untyped maps after YAML/JSON unmarshaling. These fields are declared as
// `any` in parser.Schema to support both *Schema and bool values per the OAS spec.
func (c *RefCollector) collectRefsFromMap(m map[string]any, path string) {
	c.collectRefsFromMapWithDepth(m, path, 0)
}

// collectRefsFromMapWithDepth is the internal implementation with depth tracking.
func (c *RefCollector) collectRefsFromMapWithDepth(m map[string]any, path string, depth int) {
	if depth > maxRefCollectionDepth {
		// Extremely deep nesting or circular structure - stop to prevent stack overflow
		return
	}

	// Check for direct $ref
	if refStr, ok := m["$ref"].(string); ok && refStr != "" {
		c.addRef(refStr, path, RefTypeSchema)
	}

	// Check nested properties
	if props, ok := m["properties"].(map[string]any); ok {
		for propName, propVal := range props {
			if propMap, ok := propVal.(map[string]any); ok {
				c.collectRefsFromMapWithDepth(propMap, fmt.Sprintf("%s.properties.%s", path, propName), depth+1)
			}
		}
	}

	// Check items
	if items, ok := m["items"].(map[string]any); ok {
		c.collectRefsFromMapWithDepth(items, fmt.Sprintf("%s.items", path), depth+1)
	}

	// Check additionalProperties
	if addProps, ok := m["additionalProperties"].(map[string]any); ok {
		c.collectRefsFromMapWithDepth(addProps, fmt.Sprintf("%s.additionalProperties", path), depth+1)
	}

	// Check additionalItems
	if addItems, ok := m["additionalItems"].(map[string]any); ok {
		c.collectRefsFromMapWithDepth(addItems, fmt.Sprintf("%s.additionalItems", path), depth+1)
	}

	// Check allOf, anyOf, oneOf
	for _, key := range []string{"allOf", "anyOf", "oneOf"} {
		if arr, ok := m[key].([]any); ok {
			for i, item := range arr {
				if itemMap, ok := item.(map[string]any); ok {
					c.collectRefsFromMapWithDepth(itemMap, fmt.Sprintf("%s.%s[%d]", path, key, i), depth+1)
				}
			}
		}
	}

	// Check not
	if notSchema, ok := m["not"].(map[string]any); ok {
		c.collectRefsFromMapWithDepth(notSchema, fmt.Sprintf("%s.not", path), depth+1)
	}

	// Check conditional schemas (OAS 3.1+)
	for _, key := range []string{"if", "then", "else"} {
		if condSchema, ok := m[key].(map[string]any); ok {
			c.collectRefsFromMapWithDepth(condSchema, fmt.Sprintf("%s.%s", path, key), depth+1)
		}
	}

	// Check prefixItems (OAS 3.1+)
	if prefixItems, ok := m["prefixItems"].([]any); ok {
		for i, item := range prefixItems {
			if itemMap, ok := item.(map[string]any); ok {
				c.collectRefsFromMapWithDepth(itemMap, fmt.Sprintf("%s.prefixItems[%d]", path, i), depth+1)
			}
		}
	}

	// Check contains (OAS 3.1+)
	if contains, ok := m["contains"].(map[string]any); ok {
		c.collectRefsFromMapWithDepth(contains, fmt.Sprintf("%s.contains", path), depth+1)
	}

	// Check propertyNames (OAS 3.1+)
	if propNames, ok := m["propertyNames"].(map[string]any); ok {
		c.collectRefsFromMapWithDepth(propNames, fmt.Sprintf("%s.propertyNames", path), depth+1)
	}

	// Check dependentSchemas (OAS 3.1+)
	if depSchemas, ok := m["dependentSchemas"].(map[string]any); ok {
		for name, schemaVal := range depSchemas {
			if schemaMap, ok := schemaVal.(map[string]any); ok {
				c.collectRefsFromMapWithDepth(schemaMap, fmt.Sprintf("%s.dependentSchemas.%s", path, name), depth+1)
			}
		}
	}

	// Check patternProperties
	if patternProps, ok := m["patternProperties"].(map[string]any); ok {
		for pattern, propVal := range patternProps {
			if propMap, ok := propVal.(map[string]any); ok {
				c.collectRefsFromMapWithDepth(propMap, fmt.Sprintf("%s.patternProperties.%s", path, pattern), depth+1)
			}
		}
	}

	// Check $defs (OAS 3.1+)
	if defs, ok := m["$defs"].(map[string]any); ok {
		for name, defVal := range defs {
			if defMap, ok := defVal.(map[string]any); ok {
				c.collectRefsFromMapWithDepth(defMap, fmt.Sprintf("%s.$defs.%s", path, name), depth+1)
			}
		}
	}

	// Check discriminator.mapping
	if disc, ok := m["discriminator"].(map[string]any); ok {
		if mapping, ok := disc["mapping"].(map[string]any); ok {
			for key, ref := range mapping {
				if refStr, ok := ref.(string); ok && refStr != "" {
					c.addRef(refStr, fmt.Sprintf("%s.discriminator.mapping.%s", path, key), RefTypeSchema)
				}
			}
		}
	}
}

// IsSchemaReferenced returns true if a schema with the given name is referenced.
func (c *RefCollector) IsSchemaReferenced(name string, version parser.OASVersion) bool {
	ref := schemaRefPath(name, version)
	return c.isRefReferenced(ref, RefTypeSchema)
}

// IsParameterReferenced returns true if a parameter with the given name is referenced.
func (c *RefCollector) IsParameterReferenced(name string, version parser.OASVersion) bool {
	ref := parameterRefPath(name, version)
	return c.isRefReferenced(ref, RefTypeParameter)
}

// IsResponseReferenced returns true if a response with the given name is referenced.
func (c *RefCollector) IsResponseReferenced(name string, version parser.OASVersion) bool {
	ref := responseRefPath(name, version)
	return c.isRefReferenced(ref, RefTypeResponse)
}

// IsRequestBodyReferenced returns true if a request body with the given name is referenced.
// This is only applicable for OAS 3.x documents.
func (c *RefCollector) IsRequestBodyReferenced(name string) bool {
	ref := "#/components/requestBodies/" + name
	return c.isRefReferenced(ref, RefTypeRequestBody)
}

// IsSecuritySchemeReferenced returns true if a security scheme with the given name is referenced.
// Note: Security schemes are typically referenced by name in the security array, not via $ref.
// This method checks for explicit $ref usage which is rare but valid.
func (c *RefCollector) IsSecuritySchemeReferenced(name string, version parser.OASVersion) bool {
	ref := securitySchemeRefPath(name, version)
	return c.isRefReferenced(ref, RefTypeSecurityScheme)
}

// IsHeaderReferenced returns true if a header with the given name is referenced.
func (c *RefCollector) IsHeaderReferenced(name string, version parser.OASVersion) bool {
	var ref string
	if version == parser.OASVersion20 {
		// OAS 2.0 doesn't have a global headers definition
		return false
	}
	ref = "#/components/headers/" + name
	return c.isRefReferenced(ref, RefTypeHeader)
}

// IsLinkReferenced returns true if a link with the given name is referenced.
func (c *RefCollector) IsLinkReferenced(name string) bool {
	ref := "#/components/links/" + name
	return c.isRefReferenced(ref, RefTypeLink)
}

// IsCallbackReferenced returns true if a callback with the given name is referenced.
func (c *RefCollector) IsCallbackReferenced(name string) bool {
	ref := "#/components/callbacks/" + name
	return c.isRefReferenced(ref, RefTypeCallback)
}

// IsExampleReferenced returns true if an example with the given name is referenced.
func (c *RefCollector) IsExampleReferenced(name string) bool {
	ref := "#/components/examples/" + name
	return c.isRefReferenced(ref, RefTypeExample)
}

// IsPathItemReferenced returns true if a path item with the given name is referenced.
func (c *RefCollector) IsPathItemReferenced(name string) bool {
	ref := "#/components/pathItems/" + name
	return c.isRefReferenced(ref, RefTypePathItem)
}

// GetSchemaRefs returns all schema reference paths that were collected.
func (c *RefCollector) GetSchemaRefs() []string {
	return c.getRefsByType(RefTypeSchema)
}

// GetUnreferencedSchemas returns the names of schemas that are defined but not referenced.
// For OAS 2.0, it checks definitions. For OAS 3.x, it checks components/schemas.
func (c *RefCollector) GetUnreferencedSchemas(doc any) []string {
	var schemaNames []string
	var version parser.OASVersion

	switch d := doc.(type) {
	case *parser.OAS2Document:
		version = parser.OASVersion20
		for name := range d.Definitions {
			schemaNames = append(schemaNames, name)
		}
	case *parser.OAS3Document:
		version = d.OASVersion
		if d.Components != nil {
			for name := range d.Components.Schemas {
				schemaNames = append(schemaNames, name)
			}
		}
	default:
		return nil
	}

	var unreferenced []string
	for _, name := range schemaNames {
		if !c.IsSchemaReferenced(name, version) {
			unreferenced = append(unreferenced, name)
		}
	}
	return unreferenced
}

// isRefReferenced checks if a reference path is in the collected refs.
func (c *RefCollector) isRefReferenced(ref string, refType RefType) bool {
	if refs, ok := c.RefsByType[refType]; ok {
		return refs[ref]
	}
	return false
}

// getRefsByType returns all reference paths of a given type.
func (c *RefCollector) getRefsByType(refType RefType) []string {
	refs, ok := c.RefsByType[refType]
	if !ok {
		return nil
	}
	result := make([]string, 0, len(refs))
	for ref := range refs {
		result = append(result, ref)
	}
	return result
}

// schemaRefPath returns the reference path for a schema name.
func schemaRefPath(name string, version parser.OASVersion) string {
	if version == parser.OASVersion20 {
		return "#/definitions/" + name
	}
	return "#/components/schemas/" + name
}

// parameterRefPath returns the reference path for a parameter name.
func parameterRefPath(name string, version parser.OASVersion) string {
	if version == parser.OASVersion20 {
		return "#/parameters/" + name
	}
	return "#/components/parameters/" + name
}

// responseRefPath returns the reference path for a response name.
func responseRefPath(name string, version parser.OASVersion) string {
	if version == parser.OASVersion20 {
		return "#/responses/" + name
	}
	return "#/components/responses/" + name
}

// securitySchemeRefPath returns the reference path for a security scheme name.
func securitySchemeRefPath(name string, version parser.OASVersion) string {
	if version == parser.OASVersion20 {
		return "#/securityDefinitions/" + name
	}
	return "#/components/securitySchemes/" + name
}

// ExtractSchemaNameFromRef extracts the schema name from a reference path.
// Returns empty string if the reference is not a schema reference.
func ExtractSchemaNameFromRef(ref string, version parser.OASVersion) string {
	var prefix string
	if version == parser.OASVersion20 {
		prefix = "#/definitions/"
	} else {
		prefix = "#/components/schemas/"
	}

	if strings.HasPrefix(ref, prefix) {
		return strings.TrimPrefix(ref, prefix)
	}
	return ""
}

// ExtractComponentNameFromRef extracts the component name from a reference path.
// Returns the component type and name, or empty strings if not a valid component reference.
func ExtractComponentNameFromRef(ref string) (componentType, name string) {
	// OAS 2.0 patterns
	oas2Prefixes := map[string]string{
		"#/definitions/":         "schema",
		"#/parameters/":          "parameter",
		"#/responses/":           "response",
		"#/securityDefinitions/": "securityScheme",
	}

	// OAS 3.x patterns
	oas3Prefixes := map[string]string{
		"#/components/schemas/":         "schema",
		"#/components/parameters/":      "parameter",
		"#/components/responses/":       "response",
		"#/components/requestBodies/":   "requestBody",
		"#/components/headers/":         "header",
		"#/components/securitySchemes/": "securityScheme",
		"#/components/links/":           "link",
		"#/components/callbacks/":       "callback",
		"#/components/examples/":        "example",
		"#/components/pathItems/":       "pathItem",
	}

	// Try OAS 3.x patterns first (more specific)
	for prefix, compType := range oas3Prefixes {
		if strings.HasPrefix(ref, prefix) {
			return compType, strings.TrimPrefix(ref, prefix)
		}
	}

	// Try OAS 2.0 patterns
	for prefix, compType := range oas2Prefixes {
		if strings.HasPrefix(ref, prefix) {
			return compType, strings.TrimPrefix(ref, prefix)
		}
	}

	return "", ""
}
