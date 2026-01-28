package joiner

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/parser"
)

// SchemaRewriter handles rewriting of schema references throughout an OpenAPI document
type SchemaRewriter struct {
	refMap      map[string]string // Full ref path: "#/components/schemas/Old" → "#/components/schemas/New"
	bareNameMap map[string]string // Bare name: "Old" → "New" (for discriminator shorthand)
	visited     map[uintptr]bool  // Tracks visited nodes to prevent infinite loops
}

// NewSchemaRewriter creates a new rewriter instance
func NewSchemaRewriter() *SchemaRewriter {
	return &SchemaRewriter{
		refMap:      make(map[string]string),
		bareNameMap: make(map[string]string),
		visited:     make(map[uintptr]bool),
	}
}

// RegisterRename registers a schema rename operation
func (r *SchemaRewriter) RegisterRename(oldName, newName string, version parser.OASVersion) {
	oldRef := schemaRefPath(oldName, version)
	newRef := schemaRefPath(newName, version)
	r.refMap[oldRef] = newRef
	r.bareNameMap[oldName] = newName
}

// RewriteDocument traverses and rewrites all references in the document
func (r *SchemaRewriter) RewriteDocument(doc any) error {
	// Reset visited tracking for new traversal
	r.visited = make(map[uintptr]bool)

	switch d := doc.(type) {
	case *parser.OAS3Document:
		return r.rewriteOAS3Document(d)
	case *parser.OAS2Document:
		return r.rewriteOAS2Document(d)
	default:
		return fmt.Errorf("unsupported document type: %T", doc)
	}
}

// schemaRefPath returns the $ref path for a schema name based on OAS version
func schemaRefPath(name string, version parser.OASVersion) string {
	if version == parser.OASVersion20 {
		return pathutil.DefinitionRef(name)
	}
	return pathutil.SchemaRef(name)
}

// rewriteOAS3Document rewrites all references in an OAS 3.x document
func (r *SchemaRewriter) rewriteOAS3Document(doc *parser.OAS3Document) error {
	// Rewrite references in components
	if doc.Components != nil {
		// Schemas
		for _, schema := range doc.Components.Schemas {
			r.rewriteSchema(schema)
		}
		// Parameters
		for _, param := range doc.Components.Parameters {
			r.rewriteParameter(param)
		}
		// Responses
		for _, resp := range doc.Components.Responses {
			r.rewriteResponse(resp)
		}
		// Request bodies
		for _, reqBody := range doc.Components.RequestBodies {
			r.rewriteRequestBody(reqBody)
		}
		// Headers
		for _, header := range doc.Components.Headers {
			r.rewriteHeader(header)
		}
		// Callbacks
		for _, callback := range doc.Components.Callbacks {
			r.rewriteCallback(callback)
		}
		// Links - intentionally not rewritten (don't contain schema references)
		// Path items
		for _, pathItem := range doc.Components.PathItems {
			r.rewritePathItem(pathItem)
		}
	}

	// Rewrite references in paths
	for _, pathItem := range doc.Paths {
		r.rewritePathItem(pathItem)
	}

	// Rewrite references in webhooks (OAS 3.1+)
	for _, webhook := range doc.Webhooks {
		r.rewritePathItem(webhook)
	}

	return nil
}

// rewriteOAS2Document rewrites all references in an OAS 2.0 document
func (r *SchemaRewriter) rewriteOAS2Document(doc *parser.OAS2Document) error {
	// Rewrite references in definitions
	for _, schema := range doc.Definitions {
		r.rewriteSchema(schema)
	}

	// Rewrite references in parameters
	for _, param := range doc.Parameters {
		r.rewriteParameter(param)
	}

	// Rewrite references in responses
	for _, resp := range doc.Responses {
		r.rewriteResponse(resp)
	}

	// Rewrite references in paths
	for _, pathItem := range doc.Paths {
		r.rewritePathItem(pathItem)
	}

	return nil
}

// rewriteSchema traverses and rewrites references within a schema
func (r *SchemaRewriter) rewriteSchema(schema *parser.Schema) {
	if schema == nil {
		return
	}

	// Check circular references
	ptr := reflect.ValueOf(schema).Pointer()
	if r.visited[ptr] {
		return
	}
	r.visited[ptr] = true

	// Rewrite $ref
	if schema.Ref != "" {
		if newRef, exists := r.refMap[schema.Ref]; exists {
			schema.Ref = newRef
		}
	}

	// Rewrite properties
	for _, prop := range schema.Properties {
		r.rewriteSchema(prop)
	}

	// Rewrite patternProperties
	for _, prop := range schema.PatternProperties {
		r.rewriteSchema(prop)
	}

	// Rewrite additionalProperties (can be bool or Schema)
	if schema.AdditionalProperties != nil {
		if addPropSchema, ok := schema.AdditionalProperties.(*parser.Schema); ok {
			r.rewriteSchema(addPropSchema)
		}
	}

	// Rewrite items (can be bool or Schema)
	if schema.Items != nil {
		if itemsSchema, ok := schema.Items.(*parser.Schema); ok {
			r.rewriteSchema(itemsSchema)
		}
	}

	// Rewrite prefixItems (OAS 3.1+)
	for _, item := range schema.PrefixItems {
		r.rewriteSchema(item)
	}

	// Rewrite additionalItems (can be bool or Schema)
	if schema.AdditionalItems != nil {
		if addItemsSchema, ok := schema.AdditionalItems.(*parser.Schema); ok {
			r.rewriteSchema(addItemsSchema)
		}
	}

	// Rewrite contains (OAS 3.1+)
	r.rewriteSchema(schema.Contains)

	// Rewrite propertyNames (OAS 3.1+)
	r.rewriteSchema(schema.PropertyNames)

	// Rewrite dependentSchemas (OAS 3.1+)
	for _, depSchema := range schema.DependentSchemas {
		r.rewriteSchema(depSchema)
	}

	// Rewrite $defs (OAS 3.1+)
	for _, def := range schema.Defs {
		r.rewriteSchema(def)
	}

	// Rewrite composition
	for _, s := range schema.AllOf {
		r.rewriteSchema(s)
	}
	for _, s := range schema.AnyOf {
		r.rewriteSchema(s)
	}
	for _, s := range schema.OneOf {
		r.rewriteSchema(s)
	}
	r.rewriteSchema(schema.Not)

	// Rewrite conditionals (OAS 3.1+)
	r.rewriteSchema(schema.If)
	r.rewriteSchema(schema.Then)
	r.rewriteSchema(schema.Else)

	// Rewrite discriminator mappings
	if schema.Discriminator != nil && schema.Discriminator.Mapping != nil {
		for key, value := range schema.Discriminator.Mapping {
			// Handle full $ref paths first, then bare schema names if not matched
			if newRef, exists := r.refMap[value]; exists {
				schema.Discriminator.Mapping[key] = newRef
			} else if newName, exists := r.bareNameMap[value]; exists {
				schema.Discriminator.Mapping[key] = newName
			}
		}
	}
}

// rewriteParameter rewrites references in a parameter
func (r *SchemaRewriter) rewriteParameter(param *parser.Parameter) {
	if param == nil {
		return
	}

	// Rewrite $ref
	if param.Ref != "" {
		// Parameters have their own reference space, not affected by schema renames
		return
	}

	// Rewrite schema
	r.rewriteSchema(param.Schema)

	// Note: param.Items is *parser.Items, not *parser.Schema
	// Items in parameters are handled separately and don't contain $ref

	// Rewrite content (OAS 3.0+)
	for _, mediaType := range param.Content {
		r.rewriteMediaType(mediaType)
	}
}

// rewriteResponse rewrites references in a response
func (r *SchemaRewriter) rewriteResponse(resp *parser.Response) {
	if resp == nil {
		return
	}

	// Rewrite $ref
	if resp.Ref != "" {
		// Responses have their own reference space, not affected by schema renames
		return
	}

	// Rewrite schema (OAS 2.0)
	r.rewriteSchema(resp.Schema)

	// Rewrite content (OAS 3.0+)
	for _, mediaType := range resp.Content {
		r.rewriteMediaType(mediaType)
	}

	// Rewrite headers
	for _, header := range resp.Headers {
		r.rewriteHeader(header)
	}

	// Links intentionally not rewritten (don't contain schema references)
}

// rewriteRequestBody rewrites references in a request body
func (r *SchemaRewriter) rewriteRequestBody(reqBody *parser.RequestBody) {
	if reqBody == nil {
		return
	}

	// Rewrite $ref
	if reqBody.Ref != "" {
		// Request bodies have their own reference space
		return
	}

	// Rewrite content
	for _, mediaType := range reqBody.Content {
		r.rewriteMediaType(mediaType)
	}
}

// rewriteMediaType rewrites references in a media type
func (r *SchemaRewriter) rewriteMediaType(mediaType *parser.MediaType) {
	if mediaType == nil {
		return
	}

	r.rewriteSchema(mediaType.Schema)

	// Examples intentionally not rewritten (don't contain schema references)
}

// rewriteHeader rewrites references in a header
func (r *SchemaRewriter) rewriteHeader(header *parser.Header) {
	if header == nil {
		return
	}

	// Rewrite $ref
	if header.Ref != "" {
		// Headers have their own reference space
		return
	}

	r.rewriteSchema(header.Schema)

	// Rewrite content
	for _, mediaType := range header.Content {
		r.rewriteMediaType(mediaType)
	}
}

// rewriteCallback rewrites references in a callback
func (r *SchemaRewriter) rewriteCallback(callback *parser.Callback) {
	if callback == nil {
		return
	}

	// Callback is map[string]*PathItem
	for _, pathItem := range *callback {
		r.rewritePathItem(pathItem)
	}
}

// rewritePathItem rewrites references in a path item
func (r *SchemaRewriter) rewritePathItem(pathItem *parser.PathItem) {
	if pathItem == nil {
		return
	}

	// Rewrite $ref
	if pathItem.Ref != "" {
		// Path items have their own reference space
		return
	}

	// Rewrite parameters
	for _, param := range pathItem.Parameters {
		r.rewriteParameter(param)
	}

	// Rewrite operations
	r.rewriteOperation(pathItem.Get)
	r.rewriteOperation(pathItem.Put)
	r.rewriteOperation(pathItem.Post)
	r.rewriteOperation(pathItem.Delete)
	r.rewriteOperation(pathItem.Options)
	r.rewriteOperation(pathItem.Head)
	r.rewriteOperation(pathItem.Patch)
	r.rewriteOperation(pathItem.Trace)
}

// rewriteOperation rewrites references in an operation
func (r *SchemaRewriter) rewriteOperation(op *parser.Operation) {
	if op == nil {
		return
	}

	// Rewrite parameters
	for _, param := range op.Parameters {
		r.rewriteParameter(param)
	}

	// Rewrite request body (OAS 3.0+)
	r.rewriteRequestBody(op.RequestBody)

	// Rewrite responses
	if op.Responses != nil {
		r.rewriteResponse(op.Responses.Default)
		for _, resp := range op.Responses.Codes {
			r.rewriteResponse(resp)
		}
	}

	// Rewrite callbacks (OAS 3.0+)
	for _, callback := range op.Callbacks {
		r.rewriteCallback(callback)
	}
}

// extractSchemaName extracts the schema name from a $ref path
func extractSchemaName(ref string) string {
	// Handle "#/components/schemas/Name"
	if name, found := strings.CutPrefix(ref, pathutil.RefPrefixSchemas); found {
		return name
	}
	// Handle "#/definitions/Name"
	if name, found := strings.CutPrefix(ref, pathutil.RefPrefixDefinitions); found {
		return name
	}
	return ""
}
