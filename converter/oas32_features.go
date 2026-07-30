// oas32_features.go reports the OAS 3.2.0 fixed fields a document uses when the
// conversion target predates 3.2.

package converter

import (
	"fmt"
	"strconv"

	"github.com/erraggy/oastools/parser"
)

// detectOAS32Features reports every OAS 3.2 fixed field the document uses, for a
// conversion whose target version predates 3.2.
//
// Reporting rather than stripping is this package's established convention for a
// field the target does not define. detectOAS3SchemaFeatures leaves `nullable`,
// `if`, `prefixItems` and the rest in place and records an issue; convertOAS3ToOAS3
// removes nothing at all, so even a 3.1 to 3.0 conversion keeps $defs. A field is
// cleared only where the target's serialization structurally cannot hold it: see
// discriminatorToStringForm, where the OAS 2.0 bare-string discriminator has
// nowhere to put `mapping` or `defaultMapping`.
//
// Before this ran, `query` and `additionalOperations` were the only 3.2 features
// reported on downconversion, and only on the OAS 2.0 path. The fixed fields 3.2
// added were emitted into a 3.0 or 3.1 document with no warning at all, which is
// the half of issue #397 that survives once the fields are modeled: they now round
// trip, so they reach a target that has no meaning for them.
//
// Severity is Warning rather than Critical throughout. These are additive
// annotations (a name for a server, a summary for a tag) so a consumer of the
// converted document is no worse off than before the field existed. That is
// unlike `query` or `additionalOperations`, which describe an operation the target
// version cannot express at all and are reported as Critical where they are found.
func (c *Converter) detectOAS32Features(doc *parser.OAS3Document, result *ConversionResult) {
	target := result.TargetVersion

	// define our oas32Reporter to use here and pass into downnstream functions expecting it
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
		for field, present := range map[string]bool{
			"summary": tag.Summary != "",
			"parent":  tag.Parent != "",
			"kind":    tag.Kind != "",
		} {
			if present {
				report(tagPath, field)
			}
		}
	}

	for name, item := range doc.Paths {
		c.detectOAS32PathItemFeatures(item, "paths."+name, doc.OASVersion, report)
	}
	for name, item := range doc.Webhooks {
		c.detectOAS32PathItemFeatures(item, "webhooks."+name, doc.OASVersion, report)
	}

	if doc.Components == nil {
		return
	}
	comp := doc.Components

	if len(comp.MediaTypes) > 0 {
		report("components.mediaTypes", "mediaTypes")
	}
	for name, item := range comp.PathItems {
		c.detectOAS32PathItemFeatures(item, "components.pathItems."+name, doc.OASVersion, report)
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
}

// oas32Reporter records one 3.2-only field at one location.
type oas32Reporter func(path, field string)

func (c *Converter) detectOAS32PathItemFeatures(
	item *parser.PathItem,
	prefix string,
	version parser.OASVersion,
	report oas32Reporter,
) {
	if item == nil {
		return
	}

	for i, param := range item.Parameters {
		c.detectOAS32ParameterFeatures(param, prefix+".parameters["+strconv.Itoa(i)+"]", report)
	}

	// GetOperations only surfaces query and additionalOperations at 3.2+, which is
	// exactly the case this function runs in, so the 3.2 methods are covered by the
	// loop rather than needing their own checks. They are reported as Critical
	// where the OAS 2.0 path finds them, since an operation the target cannot
	// express is a different kind of loss from an unusable annotation.
	for method, op := range parser.GetOperations(item, version) {
		if op == nil {
			continue
		}
		opPath := prefix + "." + method

		for i, param := range op.Parameters {
			c.detectOAS32ParameterFeatures(param, opPath+".parameters["+strconv.Itoa(i)+"]", report)
		}
		if op.RequestBody != nil {
			c.detectOAS32ContentFeatures(op.RequestBody.Content, opPath+".requestBody", report)
		}
		if op.Responses != nil {
			for code, resp := range op.Responses.Codes {
				c.detectOAS32ResponseFeatures(resp, opPath+".responses."+code, report)
			}
		}
	}
}

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
}

func (c *Converter) detectOAS32ContentFeatures(content map[string]*parser.MediaType, prefix string, report oas32Reporter) {
	for mediaType, mt := range content {
		c.detectOAS32MediaTypeFeatures(mt, prefix+".content."+mediaType, report)
	}
}

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

// maxEncodingNestingDepth bounds the recursive Encoding walk. The parser decodes
// each level into a fresh value, so a document cannot build a cyclic Encoding
// graph, but the bound keeps a hand-assembled one from recursing without end.
const maxEncodingNestingDepth = 100

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
		report(prefix+".flows", "deviceAuthorization")
	}
	for name, flow := range map[string]*parser.OAuthFlow{
		"implicit":            scheme.Flows.Implicit,
		"password":            scheme.Flows.Password,
		"clientCredentials":   scheme.Flows.ClientCredentials,
		"authorizationCode":   scheme.Flows.AuthorizationCode,
		"deviceAuthorization": scheme.Flows.DeviceAuthorization,
	} {
		if flow != nil && flow.DeviceAuthorizationURL != "" {
			report(prefix+".flows."+name, "deviceAuthorizationUrl")
		}
	}
}

// detectOAS32SchemaFeatures reports the 3.2 fields carried by a Schema Object and
// its nested schemas.
//
// A schema with a $ref is skipped for the same reason walkSchemaFeatures skips
// one: the definition it names is walked at the top level, so checking here would
// double-report.
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

	// Schema-or-bool fields always decode to *Schema, []*Schema, or bool; only the
	// first two need walking. See the parser's promotion note.
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
