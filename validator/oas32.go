// oas32.go implements the constraints OAS 3.2 alone states: the `querystring`
// parameter location and its interaction rules, and the two Schema Object fields
// 3.2 added, `nodeType` on the XML Object and `defaultMapping` on the
// Discriminator Object.
//
// Rules stated from 3.0 that depend on a sibling field are in oas3_examples.go.
// oas32_gate.go is this file's complement: it reports these same fields when a
// document predates them.
//
// Each rule links the section stating it rather than restating it here.
// https://spec.openapis.org/oas/v3.2.0.html

package validator

import (
	"fmt"
	"slices"

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
	if !queryStringRulesApply(v.oasVersion) {
		return
	}
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

// queryStringRulesApply reports whether the `in: "querystring"` rules are in
// force. 3.2 introduced the location, so a document below it describes something
// else by that name and is left alone.
//
// Gated here rather than at the walk: [oas3TraversalApplies] admits 3.0 and 3.1
// for the rules stated at those versions, and a parameter reaching this function
// carries no proof the parser refused the location.
func queryStringRulesApply(version parser.OASVersion) bool {
	return !version.IsValid() || version >= parser.OASVersion320
}

// validateQueryStringParam enforces what an `in: "querystring"` parameter must
// satisfy on its own: described with `content`, and none of the schema-form fields
// present.
// https://spec.openapis.org/oas/v3.2.0.html#fixed-fields-for-use-with-schema
func (v *Validator) validateQueryStringParam(param *parser.Parameter, path string, result *ValidationResult) {
	if param == nil || param.Ref != "" || param.In != parser.ParamInQueryString {
		return
	}
	if !queryStringRulesApply(v.oasVersion) {
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
