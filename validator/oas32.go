// oas32.go implements the OAS 3.2.0 constraints that depend on more than a
// field's presence: a sibling field, the document version, or another parameter
// in the same effective list.
//
// Each rule links to the section of the specification that states it rather than
// restating it here. https://spec.openapis.org/oas/v3.2.0.html

package validator

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/erraggy/oastools/parser"
)

// oas32SpecRef is the OAS 3.2.0 specification. Every rule in this file exists
// only at 3.2, so there is no version to select between.
const oas32SpecRef = "https://spec.openapis.org/oas/v3.2.0.html"

// xmlNodeTypes is the closed set `nodeType` accepts.
// https://spec.openapis.org/oas/v3.2.0.html#xml-node-type
//
// Its default is not modeled: the default depends on the enclosing Schema Object,
// which an XML Object cannot see.
var xmlNodeTypes = []string{"element", "attribute", "text", "cdata", "none"}

// oas32TraversalApplies reports whether the traversal-driven 3.2 rules apply.
//
// They read fields 3.2 introduced, and the walk is not free: it builds a JSON path
// per operation, response, and media type it passes, which ungated cost roughly
// 20% more validator allocations on every document. An unrecognized version counts
// as in scope, so a document the parser could not classify is still checked.
func oas32TraversalApplies(version parser.OASVersion) bool {
	return !version.IsValid() || version >= parser.OASVersion320
}

// validateOAS32Document runs the 3.2 rules over the Components sections. Path
// items go through [Validator.validateOAS32PathItem], called from the passes that
// already hold their operations map.
//
// The schema-level rules hang off validateSchema instead, which already visits
// every schema.
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
		// Declaration-site rules only: which operations reference this definition is
		// not knowable here, so the interaction rules are left to the use sites.
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

// validateOAS32PathItem applies the Example and `querystring` rules to one path
// item. ops and version come from the caller because every caller already holds
// them; building another operations map here cost roughly 20% more validator
// allocations, for rules that fire on almost no document.
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

// validateExampleValueExclusivity enforces the Example Object's mutual exclusions:
// `dataValue` and `serializedValue` each forbid `value`, and `serializedValue`
// also forbids `externalValue`.
// https://spec.openapis.org/oas/v3.2.0.html#fixed-fields-15
//
// `dataValue` with `serializedValue` is legal, so no rule pairs those two.
// [oas32TraversalApplies] gates the walk that reaches here, so there is no
// pre-3.2 branch to take.
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
			withSpecRef(oas32SpecRef+"#fixed-fields-15"),
			withField("dataValue"),
		)
	}
	if hasSerialized && ex.Value != nil {
		v.addError(result, path,
			"Example must not have both serializedValue and value; serializedValue requires value to be absent",
			withSpecRef(oas32SpecRef+"#fixed-fields-15"),
			withField("serializedValue"),
		)
	}
	if hasSerialized && ex.ExternalValue != "" {
		v.addError(result, path,
			"Example must not have both serializedValue and externalValue; serializedValue requires externalValue to be absent",
			withSpecRef(oas32SpecRef+"#fixed-fields-15"),
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
	// Encoding Objects carry Headers, which carry Examples, including through the
	// nested encodings 3.2 added.
	for name, enc := range mt.Encoding {
		v.visitEncodingExamples(enc, prefix+".encoding."+name, result, 0)
	}
	v.visitEncodingExamples(mt.ItemEncoding, prefix+".itemEncoding", result, 0)
	for i, enc := range mt.PrefixEncoding {
		v.visitEncodingExamples(enc, prefix+".prefixEncoding["+strconv.Itoa(i)+"]", result, 0)
	}
}

// maxEncodingNestingDepth bounds the recursive Encoding traversal 3.2 introduced.
// A parsed document cannot build a cyclic Encoding graph, but the bound keeps a
// hand-assembled one from recursing without end.
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

// countQueryLocations counts the `querystring` and `query` parameters in one list.
// A `$ref` parameter counts as neither: its `in` lives on the definition, and
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

// reportQueryStringConflicts reports the two `querystring` interaction rules for
// one effective parameter list: at most one, and never alongside `in: "query"`.
// https://spec.openapis.org/oas/v3.2.0.html#parameter-in
func (v *Validator) reportQueryStringConflicts(queryString, query int, prefix string, result *ValidationResult) {
	if queryString > 1 {
		v.addError(result, prefix,
			fmt.Sprintf("A querystring parameter must not appear more than once, but %d were found", queryString),
			withSpecRef(oas32SpecRef+"#parameter-in"),
			withField("parameters"),
		)
	}
	if queryString > 0 && query > 0 {
		v.addError(result, prefix,
			"A querystring parameter must not appear alongside any 'in: query' parameter "+
				"in the same operation or its path item",
			withSpecRef(oas32SpecRef+"#parameter-in"),
			withField("parameters"),
		)
	}
}

// validateQueryStringParam enforces what an `in: "querystring"` parameter must
// satisfy on its own: described with `content`, and none of the schema-form fields
// present.
// https://spec.openapis.org/oas/v3.2.0.html#fixed-fields-for-use-with-schema
//
// The version gate belongs to the parser, which rejects the location below 3.2.
func (v *Validator) validateQueryStringParam(param *parser.Parameter, path string, result *ValidationResult) {
	if param == nil || param.Ref != "" || param.In != parser.ParamInQueryString {
		return
	}

	if len(param.Content) == 0 {
		v.addError(result, path,
			"A querystring parameter must be specified using the content field",
			withSpecRef(oas32SpecRef+"#fixed-fields-for-use-with-schema"),
			withField("content"),
		)
	}
	if param.Schema != nil {
		v.addError(result, path,
			"A querystring parameter must not use schema; the entire query string is described with content",
			withSpecRef(oas32SpecRef+"#fixed-fields-for-use-with-schema"),
			withField("schema"),
		)
	}
	// Listed alongside `schema` in the section linked above, so the same
	// prohibition covers them.
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
			withSpecRef(oas32SpecRef+"#fixed-fields-for-use-with-schema"),
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

// validateXMLNodeType enforces the XML Object's `nodeType` rules: a value from the
// closed set, and neither `attribute` nor `wrapped` present alongside it.
// https://spec.openapis.org/oas/v3.2.0.html#xml-node-type
//
// The two bools are a MUST NOT rather than a preference, which is why the parser
// models `nodeType` beside them instead of deriving one from the other.
func (v *Validator) validateXMLNodeType(schema *parser.Schema, path string, result *ValidationResult) {
	if schema.XML == nil || schema.XML.NodeType == "" {
		return
	}
	xmlPath := path + ".xml"

	if v.oasVersion.IsValid() && v.oasVersion < parser.OASVersion320 {
		v.addError(result, xmlPath,
			fmt.Sprintf("XML nodeType is only supported in OAS 3.2+, but document is version %s", v.oasVersion),
			withSpecRef(oas32SpecRef+"#xml-node-type"),
			withField("nodeType"),
			withValue(schema.XML.NodeType),
		)
		return
	}

	if !slices.Contains(xmlNodeTypes, schema.XML.NodeType) {
		v.addError(result, xmlPath,
			fmt.Sprintf("Invalid XML nodeType %q; must be one of element, attribute, text, cdata, none", schema.XML.NodeType),
			withSpecRef(oas32SpecRef+"#xml-node-type"),
			withField("nodeType"),
			withValue(schema.XML.NodeType),
		)
	}

	if schema.XML.Attribute {
		v.addError(result, xmlPath,
			"XML attribute must not be present when nodeType is present; use nodeType: attribute instead",
			withSpecRef(oas32SpecRef+"#xml-attribute"),
			withField("attribute"),
		)
	}
	if schema.XML.Wrapped {
		v.addError(result, xmlPath,
			"XML wrapped must not be present when nodeType is present; use nodeType: element instead",
			withSpecRef(oas32SpecRef+"#xml-wrapped"),
			withField("wrapped"),
		)
	}
}

// validateDiscriminatorDefaultMapping enforces that an optional discriminating
// property requires `defaultMapping`.
// https://spec.openapis.org/oas/v3.2.0.html#discriminator-default-mapping
//
// Reported only where "optional" is locally provable: this schema declares the
// property and omits it from `required`. When the property lives in `oneOf` or
// `anyOf` subschemas instead, reporting from here would be a guess, and a false
// report on a correct document is the worse failure.
//
// Below 3.2 the field does not exist, so its presence is a version error.
func (v *Validator) validateDiscriminatorDefaultMapping(schema *parser.Schema, path string, result *ValidationResult) {
	d := schema.Discriminator
	if d == nil {
		return
	}

	if d.DefaultMapping != "" {
		if v.oasVersion.IsValid() && v.oasVersion < parser.OASVersion320 {
			v.addError(result, path+".discriminator",
				fmt.Sprintf("discriminator defaultMapping is only supported in OAS 3.2+, but document is version %s", v.oasVersion),
				withSpecRef(oas32SpecRef+"#discriminator-default-mapping"),
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
		withSpecRef(oas32SpecRef+"#discriminator-default-mapping"),
		withField("defaultMapping"),
		withValue(d.PropertyName),
	)
}
