// oas32.go implements the OAS 3.2.0 constraints that are more than presence
// checks — the rules where a field's legality depends on a sibling field, on the
// document's version, or on another parameter in the same effective list.
//
// Every rule here is quoted from versions/3.2.0.md in OAI/OpenAPI-Specification
// at the site that enforces it, because several of them read as though they were
// warnings and are in fact MUST NOTs.

package validator

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/erraggy/oastools/parser"
)

// oas32SpecRef is the base URL for the 3.2.0 spec, used for every rule in this
// file. The rules only exist at 3.2, so unlike the version-varying checks
// elsewhere there is nothing to select between.
const oas32SpecRef = "https://spec.openapis.org/oas/v3.2.0.html"

// xmlNodeTypes is the closed set the XML Object's nodeType may take:
//
//	One of `element`, `attribute`, `text`, `cdata`, or `none`
//
// Quoted from the XML Object's fixed fields table. The default is deliberately
// not modeled — it is "none if $ref, $dynamicRef, or type: "array" is present in
// the Schema Object containing the XML Object, and element otherwise", which
// depends on the enclosing schema rather than on the XML Object.
var xmlNodeTypes = []string{"element", "attribute", "text", "cdata", "none"}

// oas32TraversalApplies reports whether the traversal-driven 3.2 rules are worth
// walking the document for.
//
// Gated on the version because these rules read Example.dataValue,
// Example.serializedValue, and in: "querystring" — fields OAS 3.2 introduced. A
// pre-3.2 document cannot legally carry any of them, and walking every media
// type, header, and encoding of such a document to say so is not free: the walk
// builds a JSON path string per operation, response, and media type it passes,
// which measured as roughly a 20% increase in validator allocations across every
// document size. Skipping it keeps 3.0 and 3.1 documents at exactly their
// previous cost.
//
// What that gives up is narrow. `in: "querystring"` below 3.2 is already rejected
// by the parser's structure validation, so nothing is lost there. A pre-3.2
// document carrying dataValue or serializedValue goes unreported by these rules —
// an unknown field for its version, which is the parser's business rather than a
// cross-field constraint. The schema-level 3.2 rules keep their version checks,
// because validateSchema already visits every schema and they cost a nil test.
func oas32TraversalApplies(version parser.OASVersion) bool {
	return !version.IsValid() || version >= parser.OASVersion320
}

// validateOAS32Document runs the 3.2 cross-field rules that are not reachable
// from schema validation.
//
// The schema-level rules (XML nodeType, discriminator defaultMapping) hang off
// validateSchema instead, so they reach every schema the validator already
// visits rather than needing a second traversal of their own.
//
// The Example and querystring rules do need a traversal — the validator has
// never visited Example Objects, and the querystring rules need a path item's
// parameters together with each operation's. They share one pass over the path
// items rather than taking a pass each, because [parser.GetOperations] builds a
// map per call and this package already calls it three times per path item.
// Maps are ranged directly rather than in sorted order for the same reason:
// sorting every map in the document allocates on every document to make error
// ordering deterministic in the rare document that has an error, and the
// surrounding validator ranges doc.Paths directly for exactly this reason.
func (v *Validator) validateOAS32Document(doc *parser.OAS3Document, result *ValidationResult) {
	if !oas32TraversalApplies(doc.OASVersion) || doc.Components == nil {
		return
	}
	c := doc.Components

	for name, ex := range c.Examples {
		v.validateExampleValueExclusivity(ex, "components.examples."+name, result)
	}
	for name, param := range c.Parameters {
		prefix := "components.parameters." + name
		v.visitParameterExamples(param, prefix, result)
		// A parameter definition gets its declaration-site rules only: which
		// operations reference it is not knowable from here, so the interaction
		// rules are left to the use sites.
		v.validateQueryStringParam(param, prefix, result)
	}
	for name, header := range c.Headers {
		v.visitHeaderExamples(header, "components.headers."+name, result)
	}
	for name, rb := range c.RequestBodies {
		if rb != nil {
			v.visitContentExamples(rb.Content, "components.requestBodies."+name, result)
		}
	}
	for name, resp := range c.Responses {
		v.visitResponseExamples(resp, "components.responses."+name, result)
	}
	for name, mt := range c.MediaTypes {
		v.visitMediaTypeExamples(mt, "components.mediaTypes."+name, result)
	}
}

// validateOAS32PathItem applies the traversal-driven 3.2 rules — Example Object
// exclusivity and the in: "querystring" constraints — to one path item.
//
// ops and version are passed in rather than derived here because every caller is
// a pass that already holds both. [parser.GetOperations] allocates a map per call, and this
// package already calls it three times per path item; a fourth traversal of its
// own measured as roughly a 20% increase in validator allocations on a large
// document, for rules that fire on almost none.
func (v *Validator) validateOAS32PathItem(
	item *parser.PathItem,
	prefix string,
	version parser.OASVersion,
	ops map[string]*parser.Operation,
	result *ValidationResult,
) {
	if item == nil || !oas32TraversalApplies(version) {
		return
	}

	for i, param := range item.Parameters {
		paramPath := prefix + ".parameters[" + strconv.Itoa(i) + "]"
		v.visitParameterExamples(param, paramPath, result)
		v.validateQueryStringParam(param, paramPath, result)
	}

	// Reported against the path item so a conflict wholly inside its own
	// parameter list is not repeated for every operation it contains.
	itemQueryString, itemQuery := countQueryLocations(item.Parameters)
	v.reportQueryStringConflicts(itemQueryString, itemQuery, prefix, result)

	for method, op := range ops {
		if op == nil {
			continue
		}
		opPath := prefix + "." + method

		for i, param := range op.Parameters {
			paramPath := opPath + ".parameters[" + strconv.Itoa(i) + "]"
			v.visitParameterExamples(param, paramPath, result)
			v.validateQueryStringParam(param, paramPath, result)
		}

		if opQueryString, opQuery := countQueryLocations(op.Parameters); opQueryString > 0 || opQuery > 0 {
			// Skipped when the operation declares neither, since any defect in the
			// path item's own list was already reported above.
			v.reportQueryStringConflicts(itemQueryString+opQueryString, itemQuery+opQuery, opPath, result)
		}

		if op.RequestBody != nil {
			v.visitContentExamples(op.RequestBody.Content, opPath+".requestBody", result)
		}
		if op.Responses != nil {
			for code, resp := range op.Responses.Codes {
				v.visitResponseExamples(resp, opPath+".responses."+code, result)
			}
		}
	}
}

// =============================================================================
// Example Object: dataValue / serializedValue exclusivity
// =============================================================================

// validateExampleValueExclusivity enforces the Example Object's mutual-exclusion
// rules, quoted from the 3.2.0 fixed fields table:
//
//	dataValue        — "If this field is present, `value` MUST be absent."
//	serializedValue  — "If this field is present, `value`, and `externalValue`
//	                    MUST be absent."
//	externalValue    — "If this field is present, `serializedValue` and `value`
//	                    MUST be absent."
//
// dataValue and serializedValue together are explicitly fine — the spec's own
// Example Object example sets both — so no rule pairs those two.
//
// Reached only for a 3.2+ document (or one whose version could not be
// recognized): [oas32TraversalApplies] gates the walk that finds these objects,
// so there is no pre-3.2 branch to take here.
func (v *Validator) validateExampleValueExclusivity(ex *parser.Example, path string, result *ValidationResult) {
	if ex == nil {
		return
	}
	// A pure $ref alias carries no sibling fields; the definition it names is
	// checked in its own right. Mirrors validateOAS3RequestBody.
	if ex.Ref != "" {
		return
	}

	hasData := ex.DataValue != nil
	hasSerialized := ex.SerializedValue != ""

	if !hasData && !hasSerialized {
		return
	}

	if hasData && ex.Value != nil {
		v.addError(result, path,
			"Example must not have both dataValue and value; dataValue requires value to be absent",
			withSpecRef(oas32SpecRef+"#example-object"),
			withField("dataValue"),
		)
	}
	if hasSerialized && ex.Value != nil {
		v.addError(result, path,
			"Example must not have both serializedValue and value; serializedValue requires value to be absent",
			withSpecRef(oas32SpecRef+"#example-object"),
			withField("serializedValue"),
		)
	}
	if hasSerialized && ex.ExternalValue != "" {
		v.addError(result, path,
			"Example must not have both serializedValue and externalValue; serializedValue requires externalValue to be absent",
			withSpecRef(oas32SpecRef+"#example-object"),
			withField("serializedValue"),
		)
	}
}

func (v *Validator) visitParameterExamples(param *parser.Parameter, prefix string, result *ValidationResult) {
	if param == nil {
		return
	}
	for name, ex := range param.Examples {
		v.validateExampleValueExclusivity(ex, prefix+".examples."+name, result)
	}
	v.visitContentExamples(param.Content, prefix, result)
}

func (v *Validator) visitHeaderExamples(header *parser.Header, prefix string, result *ValidationResult) {
	if header == nil {
		return
	}
	for name, ex := range header.Examples {
		v.validateExampleValueExclusivity(ex, prefix+".examples."+name, result)
	}
	v.visitContentExamples(header.Content, prefix, result)
}

func (v *Validator) visitResponseExamples(resp *parser.Response, prefix string, result *ValidationResult) {
	if resp == nil {
		return
	}
	v.visitContentExamples(resp.Content, prefix, result)
	for name, header := range resp.Headers {
		v.visitHeaderExamples(header, prefix+".headers."+name, result)
	}
}

func (v *Validator) visitContentExamples(content map[string]*parser.MediaType, prefix string, result *ValidationResult) {
	for mediaType, mt := range content {
		v.visitMediaTypeExamples(mt, prefix+".content."+mediaType, result)
	}
}

func (v *Validator) visitMediaTypeExamples(mt *parser.MediaType, prefix string, result *ValidationResult) {
	if mt == nil {
		return
	}
	for name, ex := range mt.Examples {
		v.validateExampleValueExclusivity(ex, prefix+".examples."+name, result)
	}
	// Encoding Objects carry Headers, which carry Examples. Reached through the
	// OAS 3.2 nested encodings too, since an Encoding may now contain Encodings.
	for name, enc := range mt.Encoding {
		v.visitEncodingExamples(enc, prefix+".encoding."+name, result, 0)
	}
	v.visitEncodingExamples(mt.ItemEncoding, prefix+".itemEncoding", result, 0)
	for i, enc := range mt.PrefixEncoding {
		v.visitEncodingExamples(enc, prefix+".prefixEncoding["+strconv.Itoa(i)+"]", result, 0)
	}
}

// maxEncodingNestingDepth bounds the OAS 3.2 recursive Encoding traversal.
//
// The parser decodes each level into a fresh value, so a document cannot produce
// a cyclic Encoding graph — but a depth bound costs nothing and keeps a
// hand-assembled document from recursing without end.
const maxEncodingNestingDepth = 100

func (v *Validator) visitEncodingExamples(enc *parser.Encoding, prefix string, result *ValidationResult, depth int) {
	if enc == nil || depth > maxEncodingNestingDepth {
		return
	}
	for name, header := range enc.Headers {
		v.visitHeaderExamples(header, prefix+".headers."+name, result)
	}
	for name, nested := range enc.Encoding {
		v.visitEncodingExamples(nested, prefix+".encoding."+name, result, depth+1)
	}
	v.visitEncodingExamples(enc.ItemEncoding, prefix+".itemEncoding", result, depth+1)
	for i, nested := range enc.PrefixEncoding {
		v.visitEncodingExamples(nested, prefix+".prefixEncoding["+strconv.Itoa(i)+"]", result, depth+1)
	}
}

// =============================================================================
// Parameter Object: in: "querystring"
// =============================================================================

// countQueryLocations counts the querystring and query parameters in one list.
// A $ref parameter contributes to neither: its `in` lives on the definition, and
// guessing would report a conflict that may not exist.
func countQueryLocations(params []*parser.Parameter) (queryString, query int) {
	for _, param := range params {
		if param == nil || param.Ref != "" {
			continue
		}
		switch param.In {
		case parser.ParamInQueryString:
			queryString++
		case parser.ParamInQuery:
			query++
		}
	}
	return queryString, query
}

// reportQueryStringConflicts reports the two interaction rules for one effective
// parameter list.
func (v *Validator) reportQueryStringConflicts(queryString, query int, prefix string, result *ValidationResult) {
	if queryString > 1 {
		v.addError(result, prefix,
			fmt.Sprintf("A querystring parameter must not appear more than once, but %d were found", queryString),
			withSpecRef(oas32SpecRef+"#parameter-object"),
			withField("parameters"),
		)
	}
	if queryString > 0 && query > 0 {
		v.addError(result, prefix,
			"A querystring parameter must not appear alongside any 'in: query' parameter "+
				"in the same operation or its path item",
			withSpecRef(oas32SpecRef+"#parameter-object"),
			withField("parameters"),
		)
	}
}

// validateQueryStringParam enforces the rules a single querystring parameter must
// satisfy on its own: that it is described with `content` rather than `schema`.
//
// The `schema` prohibition is stated in the spec under "Fixed Fields for use with
// `schema`" — "These fields MUST NOT be used with in: "querystring"" — and covers
// style, explode, and allowReserved alongside schema itself.
//
// The version this location was introduced in is enforced by the parser's
// structure validation, which rejects in: "querystring" outright below 3.2, so
// there is nothing to re-report here.
func (v *Validator) validateQueryStringParam(param *parser.Parameter, path string, result *ValidationResult) {
	if param == nil || param.Ref != "" || param.In != parser.ParamInQueryString {
		return
	}

	if len(param.Content) == 0 {
		v.addError(result, path,
			"A querystring parameter must be specified using the content field",
			withSpecRef(oas32SpecRef+"#parameter-object"),
			withField("content"),
		)
	}
	if param.Schema != nil {
		v.addError(result, path,
			"A querystring parameter must not use schema; the entire query string is described with content",
			withSpecRef(oas32SpecRef+"#parameter-object"),
			withField("schema"),
		)
	}
	// style, explode, and allowReserved are listed under "Fixed Fields for use
	// with `schema`" alongside schema itself, so the same prohibition covers them.
	for _, f := range []struct {
		name    string
		present bool
	}{
		{"style", param.Style != ""},
		{"explode", param.Explode != nil},
		{"allowReserved", param.AllowReserved},
	} {
		if !f.present {
			continue
		}
		v.addError(result, path,
			fmt.Sprintf("A querystring parameter must not use %s; that field is for use with schema", f.name),
			withSpecRef(oas32SpecRef+"#parameter-object"),
			withField(f.name),
		)
	}
}

// =============================================================================
// Schema-level rules: XML nodeType, Discriminator defaultMapping
// =============================================================================

// validateOAS32SchemaFields runs the 3.2 rules that belong to a Schema Object, so
// they reach every schema validateSchema already visits.
func (v *Validator) validateOAS32SchemaFields(schema *parser.Schema, path string, result *ValidationResult) {
	v.validateXMLNodeType(schema, path, result)
	v.validateDiscriminatorDefaultMapping(schema, path, result)
}

// validateXMLNodeType enforces the XML Object's nodeType rules.
//
// Both bools it supersedes are a hard conflict, not a style preference — the
// fixed fields table says of each: "If `nodeType` is present, this field MUST NOT
// be present." That is what makes modeling nodeType as an independent field
// correct rather than lazy: a document may legally set the bools *or* nodeType,
// never both, so there is no case where one has to be derived from the other.
func (v *Validator) validateXMLNodeType(schema *parser.Schema, path string, result *ValidationResult) {
	if schema.XML == nil || schema.XML.NodeType == "" {
		return
	}
	xmlPath := path + ".xml"

	if v.oasVersion.IsValid() && v.oasVersion < parser.OASVersion320 {
		v.addError(result, xmlPath,
			fmt.Sprintf("XML nodeType is only supported in OAS 3.2+, but document is version %s", v.oasVersion),
			withSpecRef(oas32SpecRef+"#xml-object"),
			withField("nodeType"),
			withValue(schema.XML.NodeType),
		)
		return
	}

	if !slices.Contains(xmlNodeTypes, schema.XML.NodeType) {
		v.addError(result, xmlPath,
			fmt.Sprintf("Invalid XML nodeType %q; must be one of element, attribute, text, cdata, none", schema.XML.NodeType),
			withSpecRef(oas32SpecRef+"#xml-object"),
			withField("nodeType"),
			withValue(schema.XML.NodeType),
		)
	}

	if schema.XML.Attribute {
		v.addError(result, xmlPath,
			"XML attribute must not be present when nodeType is present; use nodeType: attribute instead",
			withSpecRef(oas32SpecRef+"#xml-object"),
			withField("attribute"),
		)
	}
	if schema.XML.Wrapped {
		v.addError(result, xmlPath,
			"XML wrapped must not be present when nodeType is present; use nodeType: element instead",
			withSpecRef(oas32SpecRef+"#xml-object"),
			withField("wrapped"),
		)
	}
}

// validateDiscriminatorDefaultMapping enforces the conditional requirement the
// 3.2 spec places on defaultMapping:
//
//	The discriminating property MAY be defined as required or optional, but when
//	defined as an optional property the Discriminator Object MUST include a
//	`defaultMapping` field […]
//
// Reported only when this schema itself declares the discriminating property and
// omits it from `required`, which is the case where "defined as an optional
// property" is locally provable. When the property is declared in the subschemas
// of a oneOf/anyOf instead — the more common shape — its optionality is a property
// of those subschemas, and reporting from here would mean guessing. Staying
// silent loses findings; guessing invents them, and a false "must include
// defaultMapping" on a correct document is the worse failure.
//
// Below 3.2 the field does not exist, so a document carrying it is reported as a
// version error rather than checked.
func (v *Validator) validateDiscriminatorDefaultMapping(schema *parser.Schema, path string, result *ValidationResult) {
	d := schema.Discriminator
	if d == nil {
		return
	}

	if d.DefaultMapping != "" {
		if v.oasVersion.IsValid() && v.oasVersion < parser.OASVersion320 {
			v.addError(result, path+".discriminator",
				fmt.Sprintf("discriminator defaultMapping is only supported in OAS 3.2+, but document is version %s", v.oasVersion),
				withSpecRef(oas32SpecRef+"#discriminator-object"),
				withField("defaultMapping"),
				withValue(d.DefaultMapping),
			)
		}
		return
	}

	if !v.oasVersion.IsValid() || v.oasVersion < parser.OASVersion320 {
		return
	}
	if d.PropertyName == "" {
		// Missing propertyName is reported elsewhere; without it there is no
		// property whose optionality could be assessed.
		return
	}
	if _, declared := schema.Properties[d.PropertyName]; !declared {
		return
	}
	if slices.Contains(schema.Required, d.PropertyName) {
		return
	}

	v.addError(result, path+".discriminator",
		fmt.Sprintf("Discriminator must include defaultMapping because the discriminating property '%s' is optional", d.PropertyName),
		withSpecRef(oas32SpecRef+"#discriminator-object"),
		withField("defaultMapping"),
		withValue(d.PropertyName),
	)
}
