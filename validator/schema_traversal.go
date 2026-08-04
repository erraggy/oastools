// schema_traversal.go reaches the Schema Objects the rest of the validator does
// not.
//
// [Validator.validateSchema] was called from two places: the Request Body media
// type loop and `components.schemas`. Every schema in a response, a parameter or
// a header was therefore never validated at all — by any of the rules
// validateSchema runs, not only the OAS 3.2 ones that exposed it. Issue #423 has
// the four-position reproduction.
//
// Sections are walked in map order, like every other walk in this package. The
// validator's error order is already not stable — the existing path walk alone
// produces five orderings in eight runs — so sorting keys here would allocate a
// slice per map on every document to fix a fraction of a problem. Issue #425.

package validator

import (
	"strconv"

	"github.com/erraggy/oastools/internal/httputil"

	"github.com/erraggy/oastools/parser"
)

// validateOAS3OperationSchemas validates the schemas an operation carries that
// [Validator.validateOAS3RequestBody] does not already reach.
func (v *Validator) validateOAS3OperationSchemas(op *parser.Operation, path string, result *ValidationResult) {
	// The visited set exists only for the callback traversal, and most operations
	// carry no callbacks. Allocating it unconditionally cost one map per operation
	// in the document; nil is never written, because validatePathItemSchemas is
	// reachable only through a callback.
	var visited map[*parser.PathItem]bool
	if op != nil && len(op.Callbacks) > 0 {
		visited = make(map[*parser.PathItem]bool)
	}
	v.validateOperationSchemasAtDepth(op, path, result, visited, 0)
}

// validateOperationSchemasAtDepth carries the callback traversal's cycle state,
// which only that traversal needs.
func (v *Validator) validateOperationSchemasAtDepth(op *parser.Operation, path string, result *ValidationResult, visited map[*parser.PathItem]bool, depth int) {
	if op == nil {
		return
	}
	v.validateCallbackMapSchemas(op.Callbacks, path, result, visited, depth)
	v.validateParameterListSchemas(op.Parameters, path, result)

	if op.Responses == nil {
		return
	}
	if op.Responses.Default != nil {
		v.validateResponseSchemas(op.Responses.Default, path+".responses.default", result)
	}
	for code, resp := range op.Responses.Codes {
		v.validateResponseSchemas(resp, path+".responses."+code, result)
	}
}

// pathItemOperations pairs each of a Path Item's operation fields with its method
// name.
//
// Listed rather than taken from [parser.GetOperations], which needs a version to
// decide which methods exist: a schema is a schema whatever version admits the
// operation holding it. A package-level slice rather than a map built per path
// item, so it allocates nothing and iterates in a stable order.
var pathItemOperations = []struct {
	method string
	get    func(*parser.PathItem) *parser.Operation
}{
	{httputil.MethodGet, func(p *parser.PathItem) *parser.Operation { return p.Get }},
	{httputil.MethodPut, func(p *parser.PathItem) *parser.Operation { return p.Put }},
	{httputil.MethodPost, func(p *parser.PathItem) *parser.Operation { return p.Post }},
	{httputil.MethodDelete, func(p *parser.PathItem) *parser.Operation { return p.Delete }},
	{httputil.MethodOptions, func(p *parser.PathItem) *parser.Operation { return p.Options }},
	{httputil.MethodHead, func(p *parser.PathItem) *parser.Operation { return p.Head }},
	{httputil.MethodPatch, func(p *parser.PathItem) *parser.Operation { return p.Patch }},
	{httputil.MethodTrace, func(p *parser.PathItem) *parser.Operation { return p.Trace }},
	{httputil.MethodQuery, func(p *parser.PathItem) *parser.Operation { return p.Query }},
}

// maxCallbackNestingDepth bounds the recursive Callback traversal. A callback
// holds path items whose operations hold callbacks, so the graph can close a
// loop. Same reasoning as [maxPathItemNestingDepth]: a parsed document cannot
// build one, but ValidateParsed takes the caller's.
const maxCallbackNestingDepth = 100

// validateCallbackMapSchemas validates the schemas held by a named callback map.
func (v *Validator) validateCallbackMapSchemas(callbacks map[string]*parser.Callback, path string, result *ValidationResult, visited map[*parser.PathItem]bool, depth int) {
	for name, cb := range callbacks {
		if cb == nil {
			continue
		}
		for expr, item := range *cb {
			v.validatePathItemSchemas(item, path+".callbacks."+name+"."+expr, result, visited, depth)
		}
	}
}

// validatePathItemSchemas validates the schemas a Path Item and its operations
// carry. Used for the path items inside callbacks; the ones under `paths`,
// `webhooks` and `components.pathItems` are reached by the walks that already
// visit them.
func (v *Validator) validatePathItemSchemas(item *parser.PathItem, path string, result *ValidationResult, visited map[*parser.PathItem]bool, depth int) {
	if item == nil || depth > maxCallbackNestingDepth {
		return
	}
	// The depth bound alone does not contain a cycle: a path item whose operations
	// each lead back to it branches, so the walk is exponential in depth long
	// before the bound is reached — two such operations ran for minutes without
	// returning. Visiting each path item once is what makes it terminate; the
	// bound stays as a fail-safe. Mirrors validateSchemaWithVisited.
	if visited[item] {
		return
	}
	visited[item] = true

	v.validateParameterListSchemas(item.Parameters, path, result)

	for _, entry := range pathItemOperations {
		v.validateOperationSchemasAtDepth(entry.get(item), path+"."+entry.method, result, visited, depth+1)
	}
	for method, op := range item.AdditionalOperations {
		v.validateOperationSchemasAtDepth(op, path+".additionalOperations."+method, result, visited, depth+1)
	}
}

// validateResponseSchemas validates a Response Object's content and header
// schemas.
func (v *Validator) validateResponseSchemas(resp *parser.Response, path string, result *ValidationResult) {
	if resp == nil {
		return
	}
	v.validateContentSchemas(resp.Content, path, result)
	v.validateHeaderMapSchemas(resp.Headers, path, result)
}

// validateContentSchemas validates the schema of each media type in a content map.
func (v *Validator) validateContentSchemas(content map[string]*parser.MediaType, path string, result *ValidationResult) {
	if len(content) == 0 {
		return
	}
	for mediaType, mt := range content {
		v.validateMediaTypeSchemas(mt, path+".content."+mediaType, result)
	}
}

// validateMediaTypeSchemas validates a Media Type Object's schema and, at 3.2, the
// itemSchema beside it.
func (v *Validator) validateMediaTypeSchemas(mt *parser.MediaType, path string, result *ValidationResult) {
	if mt == nil {
		return
	}
	if mt.Schema != nil {
		v.validateSchema(mt.Schema, path+".schema", result)
	}
	if mt.ItemSchema != nil {
		v.validateSchema(mt.ItemSchema, path+".itemSchema", result)
	}

	for name, enc := range mt.Encoding {
		v.validateEncodingSchemas(enc, path+".encoding."+name, result, nil, 0)
	}
	v.validateEncodingSchemas(mt.ItemEncoding, path+".itemEncoding", result, nil, 0)
	for i, enc := range mt.PrefixEncoding {
		v.validateEncodingSchemas(enc, path+".prefixEncoding["+strconv.Itoa(i)+"]", result, nil, 0)
	}
}

// validateEncodingSchemas validates the schemas an Encoding Object's headers
// carry, and recurses through the encoding nesting 3.2 added. visited and depth
// mirror the guards on [Validator.visitEncodingExamples].
func (v *Validator) validateEncodingSchemas(enc *parser.Encoding, path string, result *ValidationResult, visited map[*parser.Encoding]bool, depth int) {
	if enc == nil || depth > maxEncodingNestingDepth || visited[enc] {
		return
	}
	v.validateHeaderMapSchemas(enc.Headers, path, result)

	if !encodingNests(enc) {
		return
	}
	if visited == nil {
		visited = make(map[*parser.Encoding]bool)
	}
	visited[enc] = true

	for name, nested := range enc.Encoding {
		v.validateEncodingSchemas(nested, path+".encoding."+name, result, visited, depth+1)
	}
	v.validateEncodingSchemas(enc.ItemEncoding, path+".itemEncoding", result, visited, depth+1)
	for i, nested := range enc.PrefixEncoding {
		v.validateEncodingSchemas(nested, path+".prefixEncoding["+strconv.Itoa(i)+"]", result, visited, depth+1)
	}
}

// validateParameterListSchemas validates the schemas of a parameter list.
func (v *Validator) validateParameterListSchemas(params []*parser.Parameter, path string, result *ValidationResult) {
	for i, param := range params {
		v.validateParameterSchemas(param, path+".parameters["+strconv.Itoa(i)+"]", result)
	}
}

// validateParameterSchemas validates a Parameter Object's schema and content
// schemas. A parameter carries one or the other, never both, but which one is a
// separate rule's business.
func (v *Validator) validateParameterSchemas(param *parser.Parameter, path string, result *ValidationResult) {
	if param == nil {
		return
	}
	// Hooked here rather than at each call site so these rules inherit the
	// structural reachability this traversal exists to provide: a parameter is
	// a parameter wherever it occurs.
	v.validateParameterAllowReserved(param, path, result)
	if param.In == parser.ParamInHeader {
		v.validateHeaderName(param.Name, path, "name", result)
	}
	if param.Schema != nil {
		v.validateSchema(param.Schema, path+".schema", result)
	}
	v.validateContentSchemas(param.Content, path, result)
}

// validateHeaderMapSchemas validates the schemas of a named header map.
func (v *Validator) validateHeaderMapSchemas(headers map[string]*parser.Header, path string, result *ValidationResult) {
	if len(headers) == 0 {
		return
	}
	for name, header := range headers {
		headerPath := path + ".headers." + name
		v.validateHeaderName(name, headerPath, "headers", result)
		v.validateHeaderSchemas(header, headerPath, result)
	}
}

// validateHeaderSchemas validates a Header Object's schema and content schemas.
func (v *Validator) validateHeaderSchemas(header *parser.Header, path string, result *ValidationResult) {
	if header == nil {
		return
	}
	v.validateHeaderAllowReserved(header, path, result)
	if header.Schema != nil {
		v.validateSchema(header.Schema, path+".schema", result)
	}
	v.validateContentSchemas(header.Content, path, result)
}

// validateOAS3ComponentSchemas validates the schemas held by the Components
// sections that nothing else reaches.
//
// `schemas` and `requestBodies` are not among them: validateOAS3Components walks
// the first directly and the second through validateOAS3RequestBody, which
// validates its content schemas. Walking either here reports every error in it
// twice.
func (v *Validator) validateOAS3ComponentSchemas(c *parser.Components, result *ValidationResult) {
	if c == nil {
		return
	}
	for name, resp := range c.Responses {
		v.validateResponseSchemas(resp, "components.responses."+name, result)
	}
	for name, param := range c.Parameters {
		v.validateParameterSchemas(param, "components.parameters."+name, result)
	}
	for name, header := range c.Headers {
		headerPath := "components.headers." + name
		v.validateHeaderName(name, headerPath, "headers", result)
		v.validateHeaderSchemas(header, headerPath, result)
	}
	for name, mt := range c.MediaTypes {
		v.validateMediaTypeSchemas(mt, "components.mediaTypes."+name, result)
	}
	if len(c.Callbacks) > 0 {
		v.validateCallbackMapSchemas(c.Callbacks, "components", result, make(map[*parser.PathItem]bool), 0)
	}
}
