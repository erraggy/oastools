package validator

import (
	"strings"

	"github.com/erraggy/oastools/internal/issues"
	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// operationRef holds information about an operation that references a component.
type operationRef struct {
	Method      string
	Path        string
	OperationID string
	IsWebhook   bool
}

// refTracker tracks which operations reference which components.
type refTracker struct {
	// componentToOps maps normalized component paths to the operations that reference them.
	// e.g., "components.schemas.User" → [{Method: "GET", Path: "/users", OperationID: "getUser"}, ...]
	componentToOps map[string][]operationRef
}

// newRefTracker creates an empty reference tracker.
func newRefTracker() *refTracker {
	return &refTracker{
		componentToOps: make(map[string][]operationRef),
	}
}

// buildRefTrackerOAS3 builds a reference tracker for an OAS 3.x document.
func buildRefTrackerOAS3(doc *parser.OAS3Document) *refTracker {
	rt := newRefTracker()
	if doc == nil {
		return rt
	}

	// Track refs from paths
	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}
		rt.trackPathItemRefs(pathItem, pathPattern, false, doc.Components)
	}

	// Track refs from webhooks (OAS 3.1+)
	for name, pathItem := range doc.Webhooks {
		if pathItem == nil {
			continue
		}
		rt.trackPathItemRefs(pathItem, name, true, doc.Components)
	}

	return rt
}

// buildRefTrackerOAS2 builds a reference tracker for an OAS 2.0 document.
func buildRefTrackerOAS2(doc *parser.OAS2Document) *refTracker {
	rt := newRefTracker()
	if doc == nil {
		return rt
	}

	// Track refs from paths
	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}
		rt.trackPathItemRefsOAS2(pathItem, pathPattern, doc.Definitions)
	}

	return rt
}

// trackPathItemRefs tracks all refs in a path item for OAS3.
func (rt *refTracker) trackPathItemRefs(item *parser.PathItem, pathPattern string, isWebhook bool, components *parser.Components) {
	operations := []struct {
		method string
		op     *parser.Operation
	}{
		{"GET", item.Get},
		{"PUT", item.Put},
		{"POST", item.Post},
		{"DELETE", item.Delete},
		{"OPTIONS", item.Options},
		{"HEAD", item.Head},
		{"PATCH", item.Patch},
		{"TRACE", item.Trace},
		{"QUERY", item.Query},
	}

	for _, o := range operations {
		if o.op != nil {
			opRef := operationRef{
				Method:      o.method,
				Path:        pathPattern,
				OperationID: o.op.OperationID,
				IsWebhook:   isWebhook,
			}
			rt.trackOperationRefs(o.op, opRef, components)
		}
	}

	// Path-level parameters (tracked without method)
	for _, param := range item.Parameters {
		if param != nil {
			opRef := operationRef{Path: pathPattern, IsWebhook: isWebhook}
			rt.trackParameterRefs(param, opRef, components)
		}
	}
}

// trackPathItemRefsOAS2 tracks all refs in a path item for OAS2.
func (rt *refTracker) trackPathItemRefsOAS2(item *parser.PathItem, pathPattern string, definitions map[string]*parser.Schema) {
	operations := []struct {
		method string
		op     *parser.Operation
	}{
		{"GET", item.Get},
		{"PUT", item.Put},
		{"POST", item.Post},
		{"DELETE", item.Delete},
		{"OPTIONS", item.Options},
		{"HEAD", item.Head},
		{"PATCH", item.Patch},
	}

	for _, o := range operations {
		if o.op != nil {
			opRef := operationRef{
				Method:      o.method,
				Path:        pathPattern,
				OperationID: o.op.OperationID,
			}
			rt.trackOperationRefsOAS2(o.op, opRef, definitions)
		}
	}

	// Path-level parameters
	for _, param := range item.Parameters {
		if param != nil {
			opRef := operationRef{Path: pathPattern}
			rt.trackParameterRefsOAS2(param, opRef, definitions)
		}
	}
}

// trackOperationRefs tracks all refs in an operation for OAS3.
func (rt *refTracker) trackOperationRefs(op *parser.Operation, opRef operationRef, components *parser.Components) {
	visited := make(map[string]bool)

	// Track parameter refs
	for _, param := range op.Parameters {
		if param != nil {
			rt.trackParameterRefs(param, opRef, components)
		}
	}

	// Track request body refs
	if op.RequestBody != nil {
		rt.trackRequestBodyRefs(op.RequestBody, opRef, components, visited)
	}

	// Track response refs
	if op.Responses != nil {
		if op.Responses.Default != nil {
			rt.trackResponseRefs(op.Responses.Default, opRef, components, visited)
		}
		for _, resp := range op.Responses.Codes {
			if resp != nil {
				rt.trackResponseRefs(resp, opRef, components, visited)
			}
		}
	}
}

// trackOperationRefsOAS2 tracks all refs in an operation for OAS2.
func (rt *refTracker) trackOperationRefsOAS2(op *parser.Operation, opRef operationRef, definitions map[string]*parser.Schema) {
	visited := make(map[string]bool)

	// Track parameter refs
	for _, param := range op.Parameters {
		if param != nil {
			rt.trackParameterRefsOAS2(param, opRef, definitions)
		}
	}

	// Track response refs
	if op.Responses != nil {
		if op.Responses.Default != nil {
			rt.trackResponseRefsOAS2(op.Responses.Default, opRef, definitions, visited)
		}
		for _, resp := range op.Responses.Codes {
			if resp != nil {
				rt.trackResponseRefsOAS2(resp, opRef, definitions, visited)
			}
		}
	}
}

// trackParameterRefs tracks refs in a parameter.
func (rt *refTracker) trackParameterRefs(param *parser.Parameter, opRef operationRef, components *parser.Components) {
	if param.Ref != "" {
		rt.addRef(param.Ref, opRef)
	}
	if param.Schema != nil {
		rt.trackSchemaRefs(param.Schema, opRef, components, make(map[string]bool))
	}
}

// trackParameterRefsOAS2 tracks refs in an OAS2 parameter.
func (rt *refTracker) trackParameterRefsOAS2(param *parser.Parameter, opRef operationRef, definitions map[string]*parser.Schema) {
	if param.Ref != "" {
		rt.addRef(param.Ref, opRef)
	}
	if param.Schema != nil {
		rt.trackSchemaRefsOAS2(param.Schema, opRef, definitions, make(map[string]bool))
	}
}

// trackRequestBodyRefs tracks refs in a request body.
func (rt *refTracker) trackRequestBodyRefs(rb *parser.RequestBody, opRef operationRef, components *parser.Components, visited map[string]bool) {
	if rb.Ref != "" {
		rt.addRef(rb.Ref, opRef)
	}
	for _, mt := range rb.Content {
		if mt != nil && mt.Schema != nil {
			rt.trackSchemaRefs(mt.Schema, opRef, components, visited)
		}
	}
}

// trackResponseRefs tracks refs in a response.
func (rt *refTracker) trackResponseRefs(resp *parser.Response, opRef operationRef, components *parser.Components, visited map[string]bool) {
	if resp.Ref != "" {
		rt.addRef(resp.Ref, opRef)
	}
	for _, mt := range resp.Content {
		if mt != nil && mt.Schema != nil {
			rt.trackSchemaRefs(mt.Schema, opRef, components, visited)
		}
	}
	for _, header := range resp.Headers {
		if header != nil && header.Schema != nil {
			rt.trackSchemaRefs(header.Schema, opRef, components, visited)
		}
	}
}

// trackResponseRefsOAS2 tracks refs in an OAS2 response.
func (rt *refTracker) trackResponseRefsOAS2(resp *parser.Response, opRef operationRef, definitions map[string]*parser.Schema, visited map[string]bool) {
	if resp.Ref != "" {
		rt.addRef(resp.Ref, opRef)
	}
	if resp.Schema != nil {
		rt.trackSchemaRefsOAS2(resp.Schema, opRef, definitions, visited)
	}
}

// trackSchemaRefs tracks refs in a schema, following transitive refs.
func (rt *refTracker) trackSchemaRefs(schema *parser.Schema, opRef operationRef, components *parser.Components, visited map[string]bool) {
	if schema == nil {
		return
	}

	// Handle $ref
	if schema.Ref != "" {
		normalized := normalizeRef(schema.Ref)
		if visited[normalized] {
			return // Avoid infinite loops
		}
		visited[normalized] = true

		rt.addRef(schema.Ref, opRef)

		// Follow the ref to track transitive dependencies
		if components != nil && strings.HasPrefix(schema.Ref, pathutil.RefPrefixSchemas) {
			name := pathutil.UnescapeRefToken(strings.TrimPrefix(schema.Ref, pathutil.RefPrefixSchemas))
			if resolved, ok := components.Schemas[name]; ok {
				rt.trackSchemaRefs(resolved, opRef, components, visited)
			}
		}
		return
	}

	// Track nested schemas
	for _, items := range schemautil.SchemaOrBoolSchemas(schema.Items) {
		rt.trackSchemaRefs(items, opRef, components, visited)
	}
	for _, prop := range schema.Properties {
		rt.trackSchemaRefs(prop, opRef, components, visited)
	}
	for _, addProps := range schemautil.SchemaOrBoolSchemas(schema.AdditionalProperties) {
		rt.trackSchemaRefs(addProps, opRef, components, visited)
	}
	for _, s := range schema.AllOf {
		rt.trackSchemaRefs(s, opRef, components, visited)
	}
	for _, s := range schema.AnyOf {
		rt.trackSchemaRefs(s, opRef, components, visited)
	}
	for _, s := range schema.OneOf {
		rt.trackSchemaRefs(s, opRef, components, visited)
	}
	if schema.Not != nil {
		rt.trackSchemaRefs(schema.Not, opRef, components, visited)
	}
}

// trackSchemaRefsOAS2 tracks refs in an OAS2 schema.
func (rt *refTracker) trackSchemaRefsOAS2(schema *parser.Schema, opRef operationRef, definitions map[string]*parser.Schema, visited map[string]bool) {
	if schema == nil {
		return
	}

	if schema.Ref != "" {
		normalized := normalizeRef(schema.Ref)
		if visited[normalized] {
			return
		}
		visited[normalized] = true

		rt.addRef(schema.Ref, opRef)

		// Follow the ref
		if name, ok := strings.CutPrefix(schema.Ref, "#/definitions/"); ok {
			if resolved, ok := definitions[name]; ok {
				rt.trackSchemaRefsOAS2(resolved, opRef, definitions, visited)
			}
		}
		return
	}

	// Track nested schemas
	for _, items := range schemautil.SchemaOrBoolSchemas(schema.Items) {
		rt.trackSchemaRefsOAS2(items, opRef, definitions, visited)
	}
	for _, prop := range schema.Properties {
		rt.trackSchemaRefsOAS2(prop, opRef, definitions, visited)
	}
	for _, addProps := range schemautil.SchemaOrBoolSchemas(schema.AdditionalProperties) {
		rt.trackSchemaRefsOAS2(addProps, opRef, definitions, visited)
	}
	for _, s := range schema.AllOf {
		rt.trackSchemaRefsOAS2(s, opRef, definitions, visited)
	}
}

// addRef adds a reference mapping from a $ref to an operation.
func (rt *refTracker) addRef(ref string, opRef operationRef) {
	normalized := normalizeRef(ref)
	if normalized == "" {
		return
	}

	// Check if this operation is already recorded for this component
	existing := rt.componentToOps[normalized]
	for _, op := range existing {
		if op.Method == opRef.Method && op.Path == opRef.Path {
			return // Already tracked
		}
	}

	rt.componentToOps[normalized] = append(existing, opRef)
}

// normalizeRef converts a $ref like "#/components/schemas/User" to
// "components.schemas.User", the form issue paths use.
//
// Each pointer token is unescaped per RFC 6901 so the result matches the issue
// path the validator reports, which carries the component's real name. A
// definition named "pet/summary" is referenced as "#/definitions/pet~1summary"
// but reported at "definitions.pet/summary"; without unescaping, the two never
// match and getComponentOperationContext calls a referenced component unused.
//
// [unescapeRefBody] does that work. This function only decides whether it can be
// skipped.
func normalizeRef(ref string) string {
	if !strings.HasPrefix(ref, "#/") {
		return "" // External ref, not tracked
	}
	body := strings.TrimPrefix(ref, "#/")
	// A ref with no "~" anywhere has no token that can carry an escape sequence,
	// so unescapeRefBody would return every token unchanged and its whole round
	// trip reduces to this single replacement. That is the overwhelming majority
	// of refs, and the general path allocates a token slice and a joined string
	// for each one.
	if !strings.Contains(body, "~") {
		return strings.ReplaceAll(body, "/", ".")
	}
	return unescapeRefBody(body)
}

// unescapeRefBody converts the body of a local $ref — everything after "#/" — to
// the dotted form, unescaping each pointer token per RFC 6901.
//
// This is the general form, correct for any body including one with no escapes
// at all. [normalizeRef] shortcuts past it when it can prove the result would be
// identical, and the tests assert that equivalence by calling this directly, so
// changing what normalization means here changes it for both paths at once.
//
// Unescaping must happen per token, after splitting on "/" and before joining on
// ".". Unescaping the whole body first would turn a "~1" into a "/" that the
// join then rewrites to ".", silently renaming the component.
func unescapeRefBody(body string) string {
	tokens := strings.Split(body, "/")
	for i, token := range tokens {
		tokens[i] = pathutil.UnescapeRefToken(token)
	}
	return strings.Join(tokens, ".")
}

// getOperationsForComponent returns the operations that reference the component
// an issue path falls under, or nil when nothing references it.
//
// The path is matched against the components already known to reference
// something rather than split positionally, because a component name may itself
// contain dots — "microsoft.graph.user" is legal and accounts for every name in
// some real specs — so the boundary between the name and the nesting after it
// cannot be recovered from the path alone.
//
// Trying successively shorter prefixes yields the longest match first in one map
// lookup per dot. Longest-first matters beyond dotted names: it keeps a
// component named "User" from claiming an issue path under "UserProfile".
//
// No match means no operation references this component, which is exactly what
// componentToOps records, so returning nil here is the right answer rather than
// a failure to resolve.
func (rt *refTracker) getOperationsForComponent(issuePath string) []operationRef {
	for path := issuePath; path != ""; {
		if ops, ok := rt.componentToOps[path]; ok {
			return ops
		}
		i := strings.LastIndex(path, ".")
		if i < 0 {
			break
		}
		path = path[:i]
	}
	return nil
}

// getOperationContext builds an OperationContext for a given issue path.
// Returns nil if no operation context applies.
func (rt *refTracker) getOperationContext(issuePath string, doc any) *issues.OperationContext {
	// Check if this is under paths.*
	if strings.HasPrefix(issuePath, "paths.") {
		return rt.getPathOrWebhookOperationContext(issuePath, "paths.", false, doc)
	}

	// Check if this is under webhooks.* (OAS 3.1+)
	if strings.HasPrefix(issuePath, "webhooks.") {
		return rt.getPathOrWebhookOperationContext(issuePath, "webhooks.", true, doc)
	}

	// Check if this is a reusable component
	if isReusableComponentPath(issuePath) {
		return rt.getComponentOperationContext(issuePath)
	}

	return nil
}

// getPathOrWebhookOperationContext extracts operation context from a paths.* or webhooks.* issue path.
// The prefix parameter specifies which prefix to strip ("paths." or "webhooks.").
// The isWebhook parameter indicates whether this is a webhook (affects operationId lookup and context).
func (rt *refTracker) getPathOrWebhookOperationContext(issuePath, prefix string, isWebhook bool, doc any) *issues.OperationContext {
	// Parse: paths./users/{id}.get.parameters[0] -> path=/users/{id}, method=get
	// Parse: webhooks.orderCreated.post.requestBody -> path=orderCreated, method=post
	parts := strings.SplitN(strings.TrimPrefix(issuePath, prefix), ".", 2)
	if len(parts) == 0 {
		return nil
	}

	pathOrName := parts[0]
	if len(parts) == 1 {
		// Just the path/webhook name itself, no method
		return &issues.OperationContext{Path: pathOrName, IsWebhook: isWebhook}
	}

	remainder := parts[1]
	// Check if next part is a method
	methodPart := strings.SplitN(remainder, ".", 2)[0]
	method := parseMethod(methodPart)

	if method == "" {
		// Path-level (e.g., paths./users.parameters, webhooks.orderCreated.parameters)
		return &issues.OperationContext{Path: pathOrName, IsWebhook: isWebhook}
	}

	// Get operationId from document (only for paths, not webhooks - webhooks use different lookup)
	var operationID string
	if !isWebhook {
		operationID = getOperationID(doc, pathOrName, method)
	} else {
		operationID = getWebhookOperationID(doc, pathOrName, method)
	}

	return &issues.OperationContext{
		Method:      method,
		Path:        pathOrName,
		OperationID: operationID,
		IsWebhook:   isWebhook,
	}
}

// getComponentOperationContext builds context for a reusable component.
func (rt *refTracker) getComponentOperationContext(issuePath string) *issues.OperationContext {
	// The issue path may point inside a component — "components.schemas.User"
	// for the component itself, "components.schemas.User.properties.id" for a
	// problem nested within it — so the lookup resolves it to whichever
	// component owns it.
	ops := rt.getOperationsForComponent(issuePath)
	if len(ops) == 0 {
		// Unused component
		return &issues.OperationContext{
			IsReusableComponent: true,
			AdditionalRefs:      -1,
		}
	}

	first := ops[0]
	return &issues.OperationContext{
		Method:              first.Method,
		Path:                first.Path,
		OperationID:         first.OperationID,
		IsReusableComponent: true,
		IsWebhook:           first.IsWebhook,
		AdditionalRefs:      len(ops) - 1,
	}
}

// isReusableComponentPath returns true if the path is under a reusable component section.
func isReusableComponentPath(path string) bool {
	prefixes := []string{
		"components.schemas.",
		"components.responses.",
		"components.parameters.",
		"components.requestBodies.",
		"components.headers.",
		"components.securitySchemes.",
		"components.links.",
		"components.callbacks.",
		"components.pathItems.",
		"definitions.",
		"parameters.",
		"responses.",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// parseMethod converts a lowercase method string to uppercase, or returns "" if not a valid method.
func parseMethod(s string) string {
	methods := map[string]string{
		"get":     "GET",
		"put":     "PUT",
		"post":    "POST",
		"delete":  "DELETE",
		"options": "OPTIONS",
		"head":    "HEAD",
		"patch":   "PATCH",
		"trace":   "TRACE",
		"query":   "QUERY",
	}
	return methods[s]
}

// getOperationID looks up the operationId for a given path and method from the document.
func getOperationID(doc any, apiPath, method string) string {
	switch d := doc.(type) {
	case *parser.OAS3Document:
		if pathItem, ok := d.Paths[apiPath]; ok && pathItem != nil {
			return getOperationIDFromPathItem(pathItem, method)
		}
	case *parser.OAS2Document:
		if pathItem, ok := d.Paths[apiPath]; ok && pathItem != nil {
			return getOperationIDFromPathItem(pathItem, method)
		}
	}
	return ""
}

// getWebhookOperationID looks up operationId for a webhook operation.
// Webhooks are only available in OAS 3.1+.
func getWebhookOperationID(doc any, webhookName, method string) string {
	if d, ok := doc.(*parser.OAS3Document); ok {
		if pathItem, ok := d.Webhooks[webhookName]; ok && pathItem != nil {
			return getOperationIDFromPathItem(pathItem, method)
		}
	}
	// OAS2 doesn't have webhooks
	return ""
}

// getOperationIDFromPathItem extracts operationId for a method from a PathItem.
func getOperationIDFromPathItem(item *parser.PathItem, method string) string {
	var op *parser.Operation
	switch method {
	case "GET":
		op = item.Get
	case "PUT":
		op = item.Put
	case "POST":
		op = item.Post
	case "DELETE":
		op = item.Delete
	case "OPTIONS":
		op = item.Options
	case "HEAD":
		op = item.Head
	case "PATCH":
		op = item.Patch
	case "TRACE":
		op = item.Trace
	case "QUERY":
		op = item.Query
	}
	if op != nil {
		return op.OperationID
	}
	return ""
}
