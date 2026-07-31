// oas32_gate.go reports OAS 3.2.0 fixed fields used by a document that predates
// them — the complement of oas32.go, which runs at 3.2 and above where the same
// fields are legal. Issue #411 has the inventory; each report below deep-links
// the object that owns the field.
//
// https://spec.openapis.org/oas/v3.2.0.html

package validator

import (
	"slices"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// oas32FieldGateApplies reports whether a document predates the 3.2 fixed fields.
// The exact complement of [oas32TraversalApplies], which keeps the unrecognized
// versions: calling a field too new needs a version to measure it against.
func oas32FieldGateApplies(version parser.OASVersion) bool {
	return version.IsValid() && version < parser.OASVersion320
}

// Each field's report links the object that defines it, so the error says where
// to read rather than restating the specification here.
const (
	refOpenAPI        = oas32SpecRef + "#openapi-object"
	refComponents     = oas32SpecRef + "#components-object"
	refPathItem       = oas32SpecRef + "#path-item-object"
	refTag            = oas32SpecRef + "#tag-object"
	refServer         = oas32SpecRef + "#server-object"
	refResponse       = oas32SpecRef + "#response-object"
	refMediaType      = oas32SpecRef + "#media-type-object"
	refEncoding       = oas32SpecRef + "#encoding-object"
	refExample        = oas32SpecRef + "#example-object"
	refSecurityScheme = oas32SpecRef + "#security-scheme-object"
	refOAuthFlows     = oas32SpecRef + "#oauth-flows-object"
	refOAuthFlow      = oas32SpecRef + "#oauth-flow-object"
)

// gateSeg is one path segment. A negative index means unindexed, so the number is
// formatted only when there is something to report.
type gateSeg struct {
	name  string
	index int
}

// gatePath accumulates a JSON path without building one. Building it eagerly cost
// roughly 20% more validator allocations — see [oas32TraversalApplies] — so
// segments push and pop by value and the join happens only where a violation is
// reported, which on a real pre-3.2 document is nowhere.
type gatePath struct {
	segs []gateSeg
}

func (p *gatePath) push(name string) {
	p.segs = append(p.segs, gateSeg{name: name, index: -1})
}

func (p *gatePath) pushIndex(name string, index int) {
	p.segs = append(p.segs, gateSeg{name: name, index: index})
}

func (p *gatePath) pop() {
	p.segs = p.segs[:len(p.segs)-1]
}

func (p *gatePath) String() string {
	var b strings.Builder
	for i, s := range p.segs {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(s.name)
		if s.index >= 0 {
			b.WriteByte('[')
			b.WriteString(strconv.Itoa(s.index))
			b.WriteByte(']')
		}
	}
	return b.String()
}

// maxPathItemNestingDepth bounds the recursive Path Item traversal, which a
// callback can close into a loop. Same reasoning as [maxEncodingNestingDepth]: a
// parsed document cannot build one, but ValidateParsed takes the caller's.
const maxPathItemNestingDepth = 100

// oas32Gate carries the walk state, so it stays off every method signature.
type oas32Gate struct {
	v       *Validator
	version parser.OASVersion
	result  *ValidationResult
	path    gatePath
	// depth counts nested path items, not path segments.
	depth int
}

// validateOAS32FieldsNotYetIntroduced reports every 3.2 fixed field found in a
// pre-3.2 document. Conversion already reports these when downgrading a target
// (`detectOAS32Features`), so validation staying silent on the result was the
// inconsistency #411 set out to close.
func (v *Validator) validateOAS32FieldsNotYetIntroduced(doc *parser.OAS3Document, result *ValidationResult) {
	if doc == nil || !oas32FieldGateApplies(doc.OASVersion) {
		return
	}
	// Sized past the deepest path the walk builds, so the segment stack does not
	// reallocate. Its growth was the walk's only measured allocation.
	g := &oas32Gate{
		v:       v,
		version: doc.OASVersion,
		result:  result,
		path:    gatePath{segs: make([]gateSeg, 0, 16)},
	}

	// Map order is randomized, so the same document reported its errors in a
	// different order each run. Sorting afterwards rather than sorting every map's
	// keys during the walk leaves a clean document sorting an empty range.
	first := len(result.Errors)
	g.document(doc)
	added := result.Errors[first:]
	slices.SortStableFunc(added, func(a, b ValidationError) int {
		if c := strings.Compare(a.Path, b.Path); c != 0 {
			return c
		}
		return strings.Compare(a.Field, b.Field)
	})
}

// report names the field, the version lacking it, and the object defining it.
func (g *oas32Gate) report(field, ref string) {
	g.v.addError(g.result, g.path.String(),
		field+" was introduced in OpenAPI 3.2.0, but this document declares "+g.version.String(),
		withSpecRef(ref),
		withField(field),
	)
}

// reportIn reports a field sitting directly under the object the walk is on.
func (g *oas32Gate) reportIn(segment, field, ref string) {
	g.path.push(segment)
	g.report(field, ref)
	g.path.pop()
}

func (g *oas32Gate) document(doc *parser.OAS3Document) {
	if doc.Self != "" {
		g.reportIn("$self", "$self", refOpenAPI)
	}

	g.servers(doc.Servers)

	for i, tag := range doc.Tags {
		if tag == nil {
			continue
		}
		g.path.pushIndex("tags", i)
		if tag.Summary != "" {
			g.reportIn("summary", "summary", refTag)
		}
		if tag.Parent != "" {
			g.reportIn("parent", "parent", refTag)
		}
		if tag.Kind != "" {
			g.reportIn("kind", "kind", refTag)
		}
		g.path.pop()
	}

	g.path.push("paths")
	for name, item := range doc.Paths {
		g.path.push(name)
		g.pathItem(item)
		g.path.pop()
	}
	g.path.pop()

	g.path.push("webhooks")
	for name, item := range doc.Webhooks {
		g.path.push(name)
		g.pathItem(item)
		g.path.pop()
	}
	g.path.pop()

	g.components(doc.Components)
}

func (g *oas32Gate) components(c *parser.Components) {
	if c == nil {
		return
	}
	g.path.push("components")
	defer g.path.pop()

	if len(c.MediaTypes) > 0 {
		g.reportIn("mediaTypes", "mediaTypes", refComponents)
	}

	walkNamedIn(g, "pathItems", c.PathItems, g.pathItem)
	walkNamedIn(g, "mediaTypes", c.MediaTypes, g.mediaType)
	walkNamedIn(g, "examples", c.Examples, g.example)
	walkNamedIn(g, "responses", c.Responses, g.response)
	walkNamedIn(g, "parameters", c.Parameters, g.parameter)
	walkNamedIn(g, "headers", c.Headers, g.header)
	walkNamedIn(g, "securitySchemes", c.SecuritySchemes, g.securityScheme)
	walkNamedIn(g, "links", c.Links, g.link)
	walkNamedIn(g, "callbacks", c.Callbacks, g.callback)

	g.path.push("requestBodies")
	for name, rb := range c.RequestBodies {
		if rb == nil {
			continue
		}
		g.path.push(name)
		g.content(rb.Content)
		g.path.pop()
	}
	g.path.pop()
}

// walkNamedIn visits a named map under one segment. Generic so the component
// sections do not each repeat the push/pop.
func walkNamedIn[T any](g *oas32Gate, section string, items map[string]*T, visit func(*T)) {
	if len(items) == 0 {
		return
	}
	g.path.push(section)
	for name, item := range items {
		if item == nil {
			continue
		}
		g.path.push(name)
		visit(item)
		g.path.pop()
	}
	g.path.pop()
}

func (g *oas32Gate) pathItem(item *parser.PathItem) {
	if item == nil || g.depth >= maxPathItemNestingDepth {
		return
	}
	g.depth++
	defer func() { g.depth-- }()

	if item.Query != nil {
		g.reportIn("query", "query", refPathItem)
	}
	if len(item.AdditionalOperations) > 0 {
		g.reportIn("additionalOperations", "additionalOperations", refPathItem)
	}

	g.servers(item.Servers)
	g.parameters(item.Parameters)

	// Listed rather than taken from [parser.GetOperations], which below 3.2 omits
	// query and additionalOperations — the two this pass exists to find. The
	// 3.2-only operations are walked, not just reported, or a field nested inside
	// one would go unseen.
	g.operationIn("get", item.Get)
	g.operationIn("put", item.Put)
	g.operationIn("post", item.Post)
	g.operationIn("delete", item.Delete)
	g.operationIn("options", item.Options)
	g.operationIn("head", item.Head)
	g.operationIn("patch", item.Patch)
	g.operationIn("trace", item.Trace)
	g.operationIn("query", item.Query)

	if len(item.AdditionalOperations) > 0 {
		g.path.push("additionalOperations")
		for method, op := range item.AdditionalOperations {
			g.operationIn(method, op)
		}
		g.path.pop()
	}
}

func (g *oas32Gate) operationIn(method string, op *parser.Operation) {
	if op == nil {
		return
	}
	g.path.push(method)
	defer g.path.pop()

	g.servers(op.Servers)
	g.parameters(op.Parameters)

	if op.RequestBody != nil {
		g.path.push("requestBody")
		g.content(op.RequestBody.Content)
		g.path.pop()
	}
	if op.Responses != nil {
		g.path.push("responses")
		if op.Responses.Default != nil {
			g.path.push("default")
			g.response(op.Responses.Default)
			g.path.pop()
		}
		for code, resp := range op.Responses.Codes {
			g.path.push(code)
			g.response(resp)
			g.path.pop()
		}
		g.path.pop()
	}

	walkNamedIn(g, "callbacks", op.Callbacks, g.callback)
}

// callback walks the path items a [Callback Object] holds.
//
// [Callback Object]: https://spec.openapis.org/oas/v3.2.0.html#callback-object
func (g *oas32Gate) callback(cb *parser.Callback) {
	if cb == nil {
		return
	}
	for expr, item := range *cb {
		g.path.push(expr)
		g.pathItem(item)
		g.path.pop()
	}
}

func (g *oas32Gate) response(resp *parser.Response) {
	if resp == nil {
		return
	}
	if resp.Summary != "" {
		g.reportIn("summary", "summary", refResponse)
	}
	g.content(resp.Content)
	g.walkHeaders(resp.Headers)
	walkNamedIn(g, "links", resp.Links, g.link)
}

// link reaches the one [Server Object] that is not part of a servers list.
//
// [Server Object]: https://spec.openapis.org/oas/v3.2.0.html#server-object
func (g *oas32Gate) link(l *parser.Link) {
	if l == nil || l.Server == nil || l.Server.Name == "" {
		return
	}
	g.path.push("server")
	g.reportIn("name", "name", refServer)
	g.path.pop()
}

func (g *oas32Gate) content(content map[string]*parser.MediaType) {
	if len(content) == 0 {
		return
	}
	g.path.push("content")
	for name, mt := range content {
		if mt == nil {
			continue
		}
		g.path.push(name)
		g.mediaType(mt)
		g.path.pop()
	}
	g.path.pop()
}

func (g *oas32Gate) mediaType(mt *parser.MediaType) {
	if mt == nil {
		return
	}
	if mt.ItemSchema != nil {
		g.reportIn("itemSchema", "itemSchema", refMediaType)
	}
	if mt.ItemEncoding != nil {
		g.reportIn("itemEncoding", "itemEncoding", refMediaType)
	}
	if len(mt.PrefixEncoding) > 0 {
		g.reportIn("prefixEncoding", "prefixEncoding", refMediaType)
	}

	g.examples(mt.Examples)
	g.encodings(mt.Encoding, 0)

	if mt.ItemEncoding != nil {
		g.path.push("itemEncoding")
		g.encoding(mt.ItemEncoding, 0)
		g.path.pop()
	}
	for i, nested := range mt.PrefixEncoding {
		g.path.pushIndex("prefixEncoding", i)
		g.encoding(nested, 0)
		g.path.pop()
	}
}

// encodings walks a named encoding map, which Media Type and Encoding both hold.
func (g *oas32Gate) encodings(encodings map[string]*parser.Encoding, depth int) {
	if len(encodings) == 0 {
		return
	}
	g.path.push("encoding")
	for name, enc := range encodings {
		g.path.push(name)
		g.encoding(enc, depth)
		g.path.pop()
	}
	g.path.pop()
}

// encoding recurses through the [Encoding Object] graph 3.2 introduced. depth
// mirrors the bound on [Validator.visitEncodingExamples].
//
// [Encoding Object]: https://spec.openapis.org/oas/v3.2.0.html#encoding-object
func (g *oas32Gate) encoding(enc *parser.Encoding, depth int) {
	if enc == nil || depth > maxEncodingNestingDepth {
		return
	}
	if len(enc.Encoding) > 0 {
		g.reportIn("encoding", "encoding", refEncoding)
	}
	if enc.ItemEncoding != nil {
		g.reportIn("itemEncoding", "itemEncoding", refEncoding)
	}
	if len(enc.PrefixEncoding) > 0 {
		g.reportIn("prefixEncoding", "prefixEncoding", refEncoding)
	}

	g.walkHeaders(enc.Headers)

	g.encodings(enc.Encoding, depth+1)

	if enc.ItemEncoding != nil {
		g.path.push("itemEncoding")
		g.encoding(enc.ItemEncoding, depth+1)
		g.path.pop()
	}
	for i, nested := range enc.PrefixEncoding {
		g.path.pushIndex("prefixEncoding", i)
		g.encoding(nested, depth+1)
		g.path.pop()
	}
}

// example tests the zero value because the parser has nothing else to test: a
// present-but-null dataValue is indistinguishable from an absent one, the same
// limit [Validator.validateExampleValueExclusivity] has. Issue #421 tracks it.
func (g *oas32Gate) example(ex *parser.Example) {
	if ex == nil {
		return
	}
	if ex.DataValue != nil {
		g.reportIn("dataValue", "dataValue", refExample)
	}
	if ex.SerializedValue != "" {
		g.reportIn("serializedValue", "serializedValue", refExample)
	}
}

func (g *oas32Gate) examples(examples map[string]*parser.Example) {
	walkNamedIn(g, "examples", examples, g.example)
}

func (g *oas32Gate) parameter(param *parser.Parameter) {
	if param == nil {
		return
	}
	g.examples(param.Examples)
	g.content(param.Content)
}

func (g *oas32Gate) parameters(params []*parser.Parameter) {
	for i, param := range params {
		if param == nil {
			continue
		}
		g.path.pushIndex("parameters", i)
		g.parameter(param)
		g.path.pop()
	}
}

func (g *oas32Gate) header(header *parser.Header) {
	if header == nil {
		return
	}
	g.examples(header.Examples)
	g.content(header.Content)
}

func (g *oas32Gate) walkHeaders(headers map[string]*parser.Header) {
	walkNamedIn(g, "headers", headers, g.header)
}

func (g *oas32Gate) servers(servers []*parser.Server) {
	for i, server := range servers {
		if server == nil || server.Name == "" {
			continue
		}
		g.path.pushIndex("servers", i)
		g.reportIn("name", "name", refServer)
		g.path.pop()
	}
}

func (g *oas32Gate) securityScheme(scheme *parser.SecurityScheme) {
	if scheme == nil {
		return
	}
	if scheme.Deprecated {
		g.reportIn("deprecated", "deprecated", refSecurityScheme)
	}
	if scheme.OAuth2MetadataURL != "" {
		g.reportIn("oauth2MetadataUrl", "oauth2MetadataUrl", refSecurityScheme)
	}
	if scheme.Flows == nil {
		return
	}

	g.path.push("flows")
	if scheme.Flows.DeviceAuthorization != nil {
		g.reportIn("deviceAuthorization", "deviceAuthorization", refOAuthFlows)
	}
	g.oauthFlowIn("implicit", scheme.Flows.Implicit)
	g.oauthFlowIn("password", scheme.Flows.Password)
	g.oauthFlowIn("clientCredentials", scheme.Flows.ClientCredentials)
	g.oauthFlowIn("authorizationCode", scheme.Flows.AuthorizationCode)
	g.oauthFlowIn("deviceAuthorization", scheme.Flows.DeviceAuthorization)
	g.path.pop()
}

func (g *oas32Gate) oauthFlowIn(name string, flow *parser.OAuthFlow) {
	if flow == nil || flow.DeviceAuthorizationURL == "" {
		return
	}
	g.path.push(name)
	g.reportIn("deviceAuthorizationUrl", "deviceAuthorizationUrl", refOAuthFlow)
	g.path.pop()
}
