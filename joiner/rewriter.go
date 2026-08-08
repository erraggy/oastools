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
	owns        func(entry any) bool

	// copying makes a top-level entry be replaced by a deep copy before anything
	// inside it changes. See copyOnWrite.
	copying bool
	onCopy  func(old, replacement any)
	// probing suppresses the changes themselves, and changed records whether
	// there were any. See probe.
	probing bool
	changed bool
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

// restrictTo limits the rewrite to the top-level entries the predicate accepts.
// A nil predicate rewrites everything. Used to scope renames to the entries one
// source document contributed (#478).
func (r *SchemaRewriter) restrictTo(owns func(entry any) bool) {
	r.owns = owns
}

// skipEntry reports whether a top-level entry is out of scope. Checked once per
// entry, so a skipped subtree is never descended into.
func (r *SchemaRewriter) skipEntry(entry any) bool {
	return r.owns != nil && !r.owns(entry)
}

// copyOnWrite makes the rewrite replace a top-level entry with a deep copy
// before changing anything inside it, so the documents the joiner was handed are
// left as they were (#480). Only entries that actually have a reference to
// change are copied.
//
// onCopy, if given, receives the old and new pointer. Callers keying anything on
// identity need it, since the copy is a different pointer.
func (r *SchemaRewriter) copyOnWrite(onCopy func(old, replacement any)) {
	r.copying = true
	r.onCopy = onCopy
}

// probe runs a rewrite with its changes suppressed and reports whether there
// were any, so an entry is copied only when copying buys something.
func (r *SchemaRewriter) probe(run func()) bool {
	savedVisited, savedProbing, savedChanged := r.visited, r.probing, r.changed
	r.visited = make(map[uintptr]bool)
	r.probing, r.changed = true, false

	run()

	changed := r.changed
	r.visited, r.probing, r.changed = savedVisited, savedProbing, savedChanged
	return changed
}

// mapped looks up a replacement and notes that the entry being rewritten has
// something to change, which is what probe reports.
func (r *SchemaRewriter) mapped(names map[string]string, value string) (string, bool) {
	replacement, ok := names[value]
	if ok {
		r.changed = true
	}
	return replacement, ok
}

// rewriteEntries applies fn to each in-scope entry of a top-level container.
// Every such container goes through here, so a new one cannot miss the scope
// check or the copy.
//
// clone is the entry's deep copy, passed in because parser.Callback's is not
// exported.
func rewriteEntries[T any](r *SchemaRewriter, entries map[string]*T, fn func(*T), clone func(*T) *T) {
	for name, entry := range entries {
		if r.skipEntry(entry) {
			continue
		}
		if r.copying {
			if !r.probe(func() { fn(entry) }) {
				continue
			}
			replacement := clone(entry)
			entries[name] = replacement
			if r.onCopy != nil {
				r.onCopy(entry, replacement)
			}
			entry = replacement
		}
		fn(entry)
	}
}

// cloneCallback deep copies a callback. parser.Callback is a map type and the
// parser's deep copy for it is unexported, so it is rebuilt here.
func cloneCallback(callback *parser.Callback) *parser.Callback {
	if callback == nil {
		return nil
	}
	out := make(parser.Callback, len(*callback))
	for expression, item := range *callback {
		out[expression] = item.DeepCopy()
	}
	return &out
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
		rewriteEntries(r, doc.Components.Schemas, r.rewriteSchema, (*parser.Schema).DeepCopy)
		rewriteEntries(r, doc.Components.Parameters, r.rewriteParameter, (*parser.Parameter).DeepCopy)
		rewriteEntries(r, doc.Components.Responses, r.rewriteResponse, (*parser.Response).DeepCopy)
		rewriteEntries(r, doc.Components.RequestBodies, r.rewriteRequestBody, (*parser.RequestBody).DeepCopy)
		rewriteEntries(r, doc.Components.Headers, r.rewriteHeader, (*parser.Header).DeepCopy)
		rewriteEntries(r, doc.Components.Callbacks, r.rewriteCallback, cloneCallback)
		// Links - intentionally not rewritten (don't contain schema references)
		rewriteEntries(r, doc.Components.PathItems, r.rewritePathItem, (*parser.PathItem).DeepCopy)
	}

	// Rewrite references in paths
	rewriteEntries(r, doc.Paths, r.rewritePathItem, (*parser.PathItem).DeepCopy)

	// Rewrite references in webhooks (OAS 3.1+)
	rewriteEntries(r, doc.Webhooks, r.rewritePathItem, (*parser.PathItem).DeepCopy)

	return nil
}

// rewriteOAS2Document rewrites all references in an OAS 2.0 document
func (r *SchemaRewriter) rewriteOAS2Document(doc *parser.OAS2Document) error {
	rewriteEntries(r, doc.Definitions, r.rewriteSchema, (*parser.Schema).DeepCopy)
	rewriteEntries(r, doc.Parameters, r.rewriteParameter, (*parser.Parameter).DeepCopy)
	rewriteEntries(r, doc.Responses, r.rewriteResponse, (*parser.Response).DeepCopy)
	rewriteEntries(r, doc.Paths, r.rewritePathItem, (*parser.PathItem).DeepCopy)

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
		if newRef, exists := r.mapped(r.refMap, schema.Ref); exists && !r.probing {
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
	if schema.Discriminator != nil {
		for key, value := range schema.Discriminator.Mapping {
			// Handle full $ref paths first, then bare schema names if not matched
			if newRef, exists := r.mapped(r.refMap, value); exists {
				if !r.probing {
					schema.Discriminator.Mapping[key] = newRef
				}
			} else if newName, exists := r.mapped(r.bareNameMap, value); exists {
				if !r.probing {
					schema.Discriminator.Mapping[key] = newName
				}
			}
		}
		// defaultMapping (OAS 3.2+) names a schema in the same two spellings, so
		// it is resolved the same way. A join that renames the fallback's target
		// without rewriting this produces a dangling reference.
		if value := schema.Discriminator.DefaultMapping; value != "" {
			if newRef, exists := r.mapped(r.refMap, value); exists {
				if !r.probing {
					schema.Discriminator.DefaultMapping = newRef
				}
			} else if newName, exists := r.mapped(r.bareNameMap, value); exists {
				if !r.probing {
					schema.Discriminator.DefaultMapping = newName
				}
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
		return pathutil.UnescapeRefToken(name)
	}
	// Handle "#/definitions/Name"
	if name, found := strings.CutPrefix(ref, pathutil.RefPrefixDefinitions); found {
		return pathutil.UnescapeRefToken(name)
	}
	return ""
}
