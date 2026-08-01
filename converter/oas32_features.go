// oas32_features.go reports the OAS 3.2.0 fixed fields a document uses when the
// conversion target predates 3.2.
// https://spec.openapis.org/oas/v3.2.0.html

package converter

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/parser"
)

// detectOAS32Features reports every OAS 3.2 fixed field the document uses, for a
// conversion whose target version predates 3.2.
//
// Reporting rather than stripping is this package's convention: detectOAS3SchemaFeatures
// leaves `nullable`, `if`, and `prefixItems` in place and records an issue, and
// convertOAS3ToOAS3 removes nothing at all. A field is cleared only where the target
// cannot serialize it, as discriminatorToStringForm does for `mapping`.
//
// Severity is Warning because these are additive annotations, unlike `query` and
// `additionalOperations`, which describe an operation the target cannot express and
// are Critical where the OAS 2.0 path finds them.
func (c *Converter) detectOAS32Features(doc *parser.OAS3Document, result *ConversionResult) {
	target := result.TargetVersion

	// The section walks below range over maps, and Go randomizes that order, so the
	// same document reported these issues in a different order on each run — four
	// orderings in eight runs of the full-field fixture. Sorting what this pass
	// appended costs one sort of a short slice and leaves the walks readable, where
	// ordering every map's keys would allocate on documents that report nothing.
	// The ordered-slice comment on the Tag fields below is the same concern, caught
	// earlier and solved locally.
	first := len(result.Issues)
	defer func() {
		added := result.Issues[first:]
		slices.SortStableFunc(added, func(a, b ConversionIssue) int {
			if n := strings.Compare(a.Path, b.Path); n != 0 {
				return n
			}
			return strings.Compare(a.Message, b.Message)
		})
	}()

	// The oas32Reporter used here and passed into the downstream detectors.
	report := func(path, field string) {
		c.addIssueWithContext(result, path,
			fmt.Sprintf("'%s' is OAS 3.2+ only and has no equivalent in OAS %s", field, target),
			fmt.Sprintf("The field is preserved in the output, which is no longer a valid OAS %s document; remove it or keep the target at 3.2", target))
	}

	if doc.Self != "" {
		report("$self", "$self")
	}

	for i, server := range doc.Servers {
		if server != nil && server.Name != "" {
			report("servers["+strconv.Itoa(i)+"]", "name")
		}
	}

	for i, tag := range doc.Tags {
		if tag == nil {
			continue
		}
		tagPath := "tags[" + strconv.Itoa(i) + "]"
		// An ordered slice, not a map literal: report appends to result.Issues, and
		// Go randomizes map iteration, so a map here would reorder the output between
		// runs for no reason.
		for _, f := range []struct {
			name    string
			present bool
		}{
			{"summary", tag.Summary != ""},
			{"parent", tag.Parent != ""},
			{"kind", tag.Kind != ""},
		} {
			if f.present {
				report(tagPath, f.name)
			}
		}
	}

	for name, item := range doc.Paths {
		c.detectOAS32PathItemFeatures(item, "paths."+name, report)
	}
	for name, item := range doc.Webhooks {
		c.detectOAS32PathItemFeatures(item, "webhooks."+name, report)
	}

	if doc.Components == nil {
		return
	}
	comp := doc.Components

	if len(comp.MediaTypes) > 0 {
		report("components.mediaTypes", "mediaTypes")
	}
	for name, mt := range comp.MediaTypes {
		// The section is 3.2-only, but so is anything 3.2 added inside it, and
		// reporting only the container would understate what the target loses.
		c.detectOAS32MediaTypeFeatures(mt, "components.mediaTypes."+name, report)
	}
	for name, item := range comp.PathItems {
		c.detectOAS32PathItemFeatures(item, "components.pathItems."+name, report)
	}
	for name, schema := range comp.Schemas {
		c.detectOAS32SchemaFeatures(schema, "components.schemas."+name, report, make(map[*parser.Schema]bool))
	}
	for name, ex := range comp.Examples {
		detectOAS32ExampleFeatures(ex, "components.examples."+name, report)
	}
	for name, param := range comp.Parameters {
		c.detectOAS32ParameterFeatures(param, "components.parameters."+name, report)
	}
	for name, header := range comp.Headers {
		c.detectOAS32HeaderFeatures(header, "components.headers."+name, report)
	}
	for name, rb := range comp.RequestBodies {
		if rb != nil {
			c.detectOAS32ContentFeatures(rb.Content, "components.requestBodies."+name, report)
		}
	}
	for name, resp := range comp.Responses {
		c.detectOAS32ResponseFeatures(resp, "components.responses."+name, report)
	}
	for name, scheme := range comp.SecuritySchemes {
		detectOAS32SecuritySchemeFeatures(scheme, "components.securitySchemes."+name, report)
	}
	for name, link := range comp.Links {
		detectOAS32LinkFeatures(link, "components.links."+name, report)
	}
}

// oas32Reporter records one 3.2-only field at one location.
type oas32Reporter func(path, field string)

func (c *Converter) detectOAS32PathItemFeatures(
	item *parser.PathItem,
	prefix string,
	report oas32Reporter,
) {
	if item == nil {
		return
	}

	for i, param := range item.Parameters {
		c.detectOAS32ParameterFeatures(param, prefix+".parameters["+strconv.Itoa(i)+"]", report)
	}

	// The 3.2 methods have to be reported here, not left to the loop below: that
	// loop walks their contents, so a `query` operation carrying no 3.2-only field
	// would convert to 3.0 with no warning at all. The OAS 2.0 path reports both as
	// Critical of its own accord, since a lost operation is not a lost annotation.
	if item.Query != nil {
		report(prefix, "query")
	}
	if len(item.AdditionalOperations) > 0 {
		report(prefix, "additionalOperations")
	}

	// Listed rather than taken from [parser.GetOperations], which is version-aware
	// and omits query and additionalOperations below 3.2. This pass runs only when
	// the target is below 3.2, and on a 3.x to 3.x conversion the document already
	// carries the target version — convertOAS3ToOAS3 sets it before this runs — so
	// the accessor would drop exactly the two operations whose contents need
	// walking, and a `summary` inside a `query` would be lost with no warning. That
	// is why no version is taken here at all. Mirrors validator/oas32_gate.go.
	for _, entry := range oas32PathItemOperations {
		c.detectOAS32OperationFeatures(entry.get(item), prefix+"."+entry.method, report)
	}
	for method, op := range item.AdditionalOperations {
		c.detectOAS32OperationFeatures(op, prefix+".additionalOperations."+method, report)
	}
}

// oas32PathItemOperations pairs each of a Path Item's operation fields with its
// method name, including the 3.2-only `query`.
var oas32PathItemOperations = []struct {
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

// detectOAS32OperationFeatures walks one operation's parameters, request body and
// responses.
func (c *Converter) detectOAS32OperationFeatures(op *parser.Operation, opPath string, report oas32Reporter) {
	if op == nil {
		return
	}
	for i, param := range op.Parameters {
		c.detectOAS32ParameterFeatures(param, opPath+".parameters["+strconv.Itoa(i)+"]", report)
	}
	if op.RequestBody != nil {
		c.detectOAS32ContentFeatures(op.RequestBody.Content, opPath+".requestBody", report)
	}
	if op.Responses == nil {
		return
	}
	for code, resp := range op.Responses.Codes {
		c.detectOAS32ResponseFeatures(resp, opPath+".responses."+code, report)
	}
}

// detectOAS32ParameterFeatures reports `in: "querystring"`, new in 3.2.
// https://spec.openapis.org/oas/v3.2.0.html#parameter-in
func (c *Converter) detectOAS32ParameterFeatures(param *parser.Parameter, prefix string, report oas32Reporter) {
	if param == nil || param.Ref != "" {
		return
	}
	if param.In == parser.ParamInQueryString {
		report(prefix, `in: "querystring"`)
	}
	for name, ex := range param.Examples {
		detectOAS32ExampleFeatures(ex, prefix+".examples."+name, report)
	}
	c.detectOAS32ContentFeatures(param.Content, prefix, report)
}

func (c *Converter) detectOAS32HeaderFeatures(header *parser.Header, prefix string, report oas32Reporter) {
	if header == nil || header.Ref != "" {
		return
	}
	for name, ex := range header.Examples {
		detectOAS32ExampleFeatures(ex, prefix+".examples."+name, report)
	}
	c.detectOAS32ContentFeatures(header.Content, prefix, report)
}

// detectOAS32ResponseFeatures reports the Response Object's `summary`.
// https://spec.openapis.org/oas/v3.2.0.html#response-object
func (c *Converter) detectOAS32ResponseFeatures(resp *parser.Response, prefix string, report oas32Reporter) {
	if resp == nil || resp.Ref != "" {
		return
	}
	if resp.Summary != "" {
		report(prefix, "summary")
	}
	c.detectOAS32ContentFeatures(resp.Content, prefix, report)
	for name, header := range resp.Headers {
		c.detectOAS32HeaderFeatures(header, prefix+".headers."+name, report)
	}
	for name, link := range resp.Links {
		detectOAS32LinkFeatures(link, prefix+".links."+name, report)
	}
}

// detectOAS32LinkFeatures reports the `name` of the one Server Object that is not
// part of a servers list.
// https://spec.openapis.org/oas/v3.2.0.html#link-object
func detectOAS32LinkFeatures(link *parser.Link, prefix string, report oas32Reporter) {
	if link == nil || link.Ref != "" {
		return
	}
	if link.Server != nil && link.Server.Name != "" {
		report(prefix+".server", "name")
	}
}

func (c *Converter) detectOAS32ContentFeatures(content map[string]*parser.MediaType, prefix string, report oas32Reporter) {
	for mediaType, mt := range content {
		c.detectOAS32MediaTypeFeatures(mt, prefix+".content."+mediaType, report)
	}
}

// detectOAS32MediaTypeFeatures reports the sequential-media-type fields.
// https://spec.openapis.org/oas/v3.2.0.html#media-type-object
func (c *Converter) detectOAS32MediaTypeFeatures(mt *parser.MediaType, prefix string, report oas32Reporter) {
	if mt == nil {
		return
	}
	if mt.ItemSchema != nil {
		report(prefix, "itemSchema")
	}
	if mt.ItemEncoding != nil {
		report(prefix, "itemEncoding")
	}
	if len(mt.PrefixEncoding) > 0 {
		report(prefix, "prefixEncoding")
	}
	for name, ex := range mt.Examples {
		detectOAS32ExampleFeatures(ex, prefix+".examples."+name, report)
	}
	// The ordinary schema as well as the 3.2 itemSchema beside it: only itemSchema
	// was walked, so a 3.2 field in the schema every 3.0 document already has went
	// unreported. Issue #423.
	if mt.Schema != nil {
		c.detectOAS32SchemaFeatures(mt.Schema, prefix+".schema", report, make(map[*parser.Schema]bool))
	}
	if mt.ItemSchema != nil {
		c.detectOAS32SchemaFeatures(mt.ItemSchema, prefix+".itemSchema", report, make(map[*parser.Schema]bool))
	}
	for name, enc := range mt.Encoding {
		detectOAS32EncodingFeatures(enc, prefix+".encoding."+name, report, 0)
	}
	detectOAS32EncodingFeatures(mt.ItemEncoding, prefix+".itemEncoding", report, 0)
	for i, enc := range mt.PrefixEncoding {
		detectOAS32EncodingFeatures(enc, prefix+".prefixEncoding["+strconv.Itoa(i)+"]", report, 0)
	}
}

// maxEncodingNestingDepth bounds the recursive Encoding walk 3.2 introduced.
// https://spec.openapis.org/oas/v3.2.0.html#encoding-object
//
// A parsed document cannot build a cyclic Encoding graph, but the bound keeps a
// hand-assembled one from recursing without end.
const maxEncodingNestingDepth = 100

// detectOAS32EncodingFeatures reports the nesting fields 3.2 added to Encoding.
// https://spec.openapis.org/oas/v3.2.0.html#encoding-object
func detectOAS32EncodingFeatures(enc *parser.Encoding, prefix string, report oas32Reporter, depth int) {
	if enc == nil || depth > maxEncodingNestingDepth {
		return
	}
	if len(enc.Encoding) > 0 {
		report(prefix, "encoding")
	}
	if enc.ItemEncoding != nil {
		report(prefix, "itemEncoding")
	}
	if len(enc.PrefixEncoding) > 0 {
		report(prefix, "prefixEncoding")
	}

	for name, nested := range enc.Encoding {
		detectOAS32EncodingFeatures(nested, prefix+".encoding."+name, report, depth+1)
	}
	detectOAS32EncodingFeatures(enc.ItemEncoding, prefix+".itemEncoding", report, depth+1)
	for i, nested := range enc.PrefixEncoding {
		detectOAS32EncodingFeatures(nested, prefix+".prefixEncoding["+strconv.Itoa(i)+"]", report, depth+1)
	}
}

// detectOAS32ExampleFeatures reports `dataValue` and `serializedValue`.
// https://spec.openapis.org/oas/v3.2.0.html#fixed-fields-15
func detectOAS32ExampleFeatures(ex *parser.Example, prefix string, report oas32Reporter) {
	if ex == nil || ex.Ref != "" {
		return
	}
	if ex.DataValue != nil {
		report(prefix, "dataValue")
	}
	if ex.SerializedValue != "" {
		report(prefix, "serializedValue")
	}
}

// detectOAS32SecuritySchemeFeatures reports the scheme and OAuth flow additions.
// https://spec.openapis.org/oas/v3.2.0.html#security-scheme-object
func detectOAS32SecuritySchemeFeatures(scheme *parser.SecurityScheme, prefix string, report oas32Reporter) {
	if scheme == nil || scheme.Ref != "" {
		return
	}
	if scheme.Deprecated {
		report(prefix, "deprecated")
	}
	if scheme.OAuth2MetadataURL != "" {
		report(prefix, "oauth2MetadataUrl")
	}
	if scheme.Flows == nil {
		return
	}
	if scheme.Flows.DeviceAuthorization != nil {
		report(prefix+".flows", oas3FlowDeviceAuthorization)
	}
	// Ordered for the same reason as the tag fields above.
	for _, f := range []struct {
		name string
		flow *parser.OAuthFlow
	}{
		{oauthFlowImplicit, scheme.Flows.Implicit},
		{oauthFlowPassword, scheme.Flows.Password},
		{oas3FlowClientCredentials, scheme.Flows.ClientCredentials},
		{oas3FlowAuthorizationCode, scheme.Flows.AuthorizationCode},
		{oas3FlowDeviceAuthorization, scheme.Flows.DeviceAuthorization},
	} {
		if f.flow != nil && f.flow.DeviceAuthorizationURL != "" {
			report(prefix+".flows."+f.name, "deviceAuthorizationUrl")
		}
	}
}

// detectOAS32SchemaFeatures reports `discriminator.defaultMapping` and
// `xml.nodeType` on a Schema Object and its nested schemas.
//
// A schema with a `$ref` is skipped for the same reason walkSchemaFeatures skips
// one: the definition it names is walked at the top level.
func (c *Converter) detectOAS32SchemaFeatures(
	schema *parser.Schema,
	prefix string,
	report oas32Reporter,
	visited map[*parser.Schema]bool,
) {
	if schema == nil || visited[schema] || schema.Ref != "" {
		return
	}
	visited[schema] = true

	if schema.Discriminator != nil && schema.Discriminator.DefaultMapping != "" {
		report(prefix+".discriminator", "defaultMapping")
	}
	if schema.XML != nil && schema.XML.NodeType != "" {
		report(prefix+".xml", "nodeType")
	}

	for name, prop := range schema.Properties {
		c.detectOAS32SchemaFeatures(prop, prefix+".properties."+name, report, visited)
	}
	for pattern, prop := range schema.PatternProperties {
		c.detectOAS32SchemaFeatures(prop, prefix+".patternProperties."+pattern, report, visited)
	}
	for name, def := range schema.Defs {
		c.detectOAS32SchemaFeatures(def, prefix+".$defs."+name, report, visited)
	}
	for name, dep := range schema.DependentSchemas {
		c.detectOAS32SchemaFeatures(dep, prefix+".dependentSchemas."+name, report, visited)
	}
	for i, s := range schema.AllOf {
		c.detectOAS32SchemaFeatures(s, prefix+".allOf["+strconv.Itoa(i)+"]", report, visited)
	}
	for i, s := range schema.AnyOf {
		c.detectOAS32SchemaFeatures(s, prefix+".anyOf["+strconv.Itoa(i)+"]", report, visited)
	}
	for i, s := range schema.OneOf {
		c.detectOAS32SchemaFeatures(s, prefix+".oneOf["+strconv.Itoa(i)+"]", report, visited)
	}
	for i, s := range schema.PrefixItems {
		c.detectOAS32SchemaFeatures(s, prefix+".prefixItems["+strconv.Itoa(i)+"]", report, visited)
	}

	// Schema-or-bool fields decode to *Schema, []*Schema, or bool; only the first
	// two need walking.
	for field, value := range map[string]any{
		"additionalProperties":  schema.AdditionalProperties,
		"items":                 schema.Items,
		"additionalItems":       schema.AdditionalItems,
		"unevaluatedProperties": schema.UnevaluatedProperties,
		"unevaluatedItems":      schema.UnevaluatedItems,
	} {
		switch v := value.(type) {
		case *parser.Schema:
			c.detectOAS32SchemaFeatures(v, prefix+"."+field, report, visited)
		case []*parser.Schema:
			for i, s := range v {
				c.detectOAS32SchemaFeatures(s, prefix+"."+field+"["+strconv.Itoa(i)+"]", report, visited)
			}
		}
	}

	for field, s := range map[string]*parser.Schema{
		"not":           schema.Not,
		"contains":      schema.Contains,
		"propertyNames": schema.PropertyNames,
		"if":            schema.If,
		"then":          schema.Then,
		"else":          schema.Else,
		"contentSchema": schema.ContentSchema,
	} {
		c.detectOAS32SchemaFeatures(s, prefix+"."+field, report, visited)
	}
}
