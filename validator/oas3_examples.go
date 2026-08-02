// oas3_examples.go implements the OAS 3.x rules that one traversal reaches: the
// Example Object's mutual exclusions, the example/examples pair on the objects
// composing them, and the `querystring` interaction rules, which the same walk
// carries because it already holds the parameter lists it needs.
//
// Every rule here is stated from 3.0. What 3.2 alone states is in oas32.go, and
// the exclusions themselves are tabulated in mutual_exclusions.go, each citing
// the object that states it.
//
// Citations resolve against the document's own version through
// [Validator.specRef], so nothing in this file names a version directly.

package validator

import (
	"strconv"

	"github.com/erraggy/oastools/parser"
)

// oas3TraversalApplies reports whether the traversal-driven OAS 3.x rules apply.
//
// 3.0 is the floor because rules the walk carries are stated from there, among
// them example/examples and value/externalValue (see [mediaTypeExclusions] and
// [exampleExclusions]). A narrower walk would leave those unreachable at the
// versions stating them. Each rule carries its own version, so the walk's reach
// is not any rule's reach.
//
// The walk is not free: it builds a JSON path per operation, response and media
// type it passes. An unrecognized version counts as in scope, so a document the
// parser could not classify is still checked.
func oas3TraversalApplies(version parser.OASVersion) bool {
	return !version.IsValid() || version >= parser.OASVersion300
}

// validateOAS3TraversalComponents runs these rules over the Components sections.
// Path items go through [Validator.validateOAS3TraversalPathItem], called from
// the passes that already hold their operations map.
func (v *Validator) validateOAS3TraversalComponents(doc *parser.OAS3Document, result *ValidationResult) {
	if !oas3TraversalApplies(doc.OASVersion) || doc.Components == nil {
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
	// components.pathItems and webhooks are reached by the passes that already
	// hold their operations maps. Callbacks have no such pass.
	v.visitCallbacks(c.Callbacks, "components", doc.OASVersion, result, nil, 0)
}

// validateOAS3TraversalPathItem applies the Example and `querystring` rules to
// one path item. ops and version come from the caller because every caller
// already holds them; building another operations map here costs roughly 20%
// more validator allocations, for rules that fire on almost no document.
func (v *Validator) validateOAS3TraversalPathItem(
	item *parser.PathItem,
	prefix string,
	version parser.OASVersion,
	ops map[string]*parser.Operation,
	result *ValidationResult,
) {
	v.traversePathItem(item, prefix, version, ops, result, nil, 0)
}

// traversePathItem is [Validator.validateOAS3TraversalPathItem] carrying the
// state only the recursion through callbacks needs. visited stays nil until a
// path item inside a callback is reached, so a document with none allocates
// nothing.
func (v *Validator) traversePathItem(
	item *parser.PathItem,
	prefix string,
	version parser.OASVersion,
	ops map[string]*parser.Operation,
	result *ValidationResult,
	visited map[*parser.PathItem]bool,
	depth int,
) {
	if item == nil || !oas3TraversalApplies(version) {
		return
	}

	for i, param := range item.Parameters {
		if !paramNeedsPath(param) {
			continue
		}
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
			if !paramNeedsPath(param) {
				continue
			}
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
			// The default response is a sibling of the coded ones, not one of
			// them: it carries its own content and headers.
			if responseNeedsVisit(op.Responses.Default) {
				v.visitResponseExamples(op.Responses.Default, opPath+".responses.default", result)
			}
			for code, resp := range op.Responses.Codes {
				if !responseNeedsVisit(resp) {
					continue
				}
				v.visitResponseExamples(resp, opPath+".responses."+code, result)
			}
		}

		v.visitCallbacks(op.Callbacks, opPath, version, result, visited, depth)
	}
}

// visitCallbacks walks the path items a Callback Object holds, so the rules
// reach every position inside one. Each of those path items needs its own
// operations map, which the callers of
// [Validator.validateOAS3TraversalPathItem] cannot supply; only a document
// declaring callbacks pays for building them.
//
// visited is what makes this terminate. The depth bound alone does not contain a
// cycle, because a path item whose operations lead back to it branches, so the
// walk goes exponential in depth long before the bound is reached. Same
// reasoning as [Validator.validatePathItemSchemas], which hit exactly that.
func (v *Validator) visitCallbacks(
	callbacks map[string]*parser.Callback,
	prefix string,
	version parser.OASVersion,
	result *ValidationResult,
	visited map[*parser.PathItem]bool,
	depth int,
) {
	if len(callbacks) == 0 || depth >= maxCallbackNestingDepth {
		return
	}
	for name, cb := range callbacks {
		if cb == nil {
			continue
		}
		for expr, item := range *cb {
			// Reading a nil map is legal, so the lookup above needs no map and
			// only the write below does. A callback map holding nothing but nil
			// entries therefore allocates nothing.
			if item == nil || visited[item] {
				continue
			}
			if visited == nil {
				visited = make(map[*parser.PathItem]bool)
			}
			visited[item] = true
			v.traversePathItem(item, prefix+".callbacks."+name+"."+expr, version,
				parser.GetOperations(item, version), result, visited, depth+1)
		}
	}
}

// paramNeedsPath reports whether a parameter can produce a report from either
// rule family reached with its path, so the caller can skip building one.
// Wider than [parameterNeedsVisit] because the same path also carries the
// `querystring` rules, which read `in` rather than any example field.
func paramNeedsPath(param *parser.Parameter) bool {
	return parameterNeedsVisit(param) ||
		(param != nil && param.In == parser.ParamInQueryString)
}

// =============================================================================
// Example Object: value exclusivity
// =============================================================================

// validateExampleValueExclusivity enforces the Example Object's mutual
// exclusions. The rules and their version scoping live in [exampleExclusions].
func (v *Validator) validateExampleValueExclusivity(ex *parser.Example, path string, result *ValidationResult) {
	if ex == nil {
		return
	}
	// A pure $ref alias carries no sibling fields; the definition it names is
	// checked in its own right. Mirrors validateOAS3RequestBody.
	if ex.Ref != "" {
		return
	}

	// Value is `any`, so an explicit `value: null` is indistinguishable from an
	// absent key and goes unreported. Correcting that needs a presence flag on
	// the field, which is the same model limit recorded for #452.
	v.reportExclusions(exampleExclusions, []fieldPresence{
		{name: fieldValue, present: ex.Value != nil},
		{name: fieldExternalValue, present: ex.ExternalValue != ""},
		{name: fieldDataValue, present: ex.DataValue != nil},
		{name: fieldSerializedValue, present: ex.SerializedValue != ""},
	}, path, result)
}

func (v *Validator) visitParameterExamples(param *parser.Parameter, prefix string, result *ValidationResult) {
	if !parameterNeedsVisit(param) {
		return
	}
	if param.Ref == "" {
		v.reportExclusions(parameterExclusions, []fieldPresence{
			{name: fieldExample, present: param.Example != nil},
			{name: fieldExamples, present: param.Examples != nil},
		}, prefix, result)
	}
	for name, ex := range param.Examples {
		v.validateExampleValueExclusivity(ex, prefix+".examples."+name, result)
	}
	v.visitContentExamples(param.Content, prefix, result)
}

func (v *Validator) visitHeaderExamples(header *parser.Header, prefix string, result *ValidationResult) {
	if !headerNeedsVisit(header) {
		return
	}
	if header.Ref == "" {
		v.reportExclusions(headerExclusions, []fieldPresence{
			{name: fieldExample, present: header.Example != nil},
			{name: fieldExamples, present: header.Examples != nil},
		}, prefix, result)
	}
	for name, ex := range header.Examples {
		v.validateExampleValueExclusivity(ex, prefix+".examples."+name, result)
	}
	v.visitContentExamples(header.Content, prefix, result)
}

// The visit predicates below decide whether a node can hold anything this walk
// reports, so the caller can skip it before building its JSON path. On a
// document carrying none of these fields, that path building is the entire cost
// of the walk.
//
// Each predicate must admit every field its visit function reads. A predicate
// that is too narrow silently stops reporting, which is why the reachability
// tests assert positions rather than rules.

func mediaTypeNeedsVisit(mt *parser.MediaType) bool {
	return mt != nil && (mt.Example != nil ||
		mt.Examples != nil ||
		mt.Encoding != nil ||
		mt.ItemEncoding != nil ||
		mt.PrefixEncoding != nil)
}

func parameterNeedsVisit(param *parser.Parameter) bool {
	return param != nil && (param.Example != nil ||
		param.Examples != nil ||
		len(param.Content) > 0)
}

func headerNeedsVisit(header *parser.Header) bool {
	return header != nil && (header.Example != nil ||
		header.Examples != nil ||
		len(header.Content) > 0)
}

func responseNeedsVisit(resp *parser.Response) bool {
	return resp != nil && (len(resp.Content) > 0 || len(resp.Headers) > 0)
}

func encodingNeedsVisit(enc *parser.Encoding) bool {
	return enc != nil && (len(enc.Headers) > 0 ||
		enc.Encoding != nil ||
		enc.ItemEncoding != nil ||
		enc.PrefixEncoding != nil)
}

func (v *Validator) visitResponseExamples(resp *parser.Response, prefix string, result *ValidationResult) {
	if !responseNeedsVisit(resp) {
		return
	}
	v.visitContentExamples(resp.Content, prefix, result)
	for name, header := range resp.Headers {
		v.visitHeaderExamples(header, prefix+".headers."+name, result)
	}
}

func (v *Validator) visitContentExamples(content map[string]*parser.MediaType, prefix string, result *ValidationResult) {
	for mediaType, mt := range content {
		if !mediaTypeNeedsVisit(mt) {
			continue
		}
		v.visitMediaTypeExamples(mt, prefix+".content."+mediaType, result)
	}
}

func (v *Validator) visitMediaTypeExamples(mt *parser.MediaType, prefix string, result *ValidationResult) {
	if !mediaTypeNeedsVisit(mt) {
		return
	}
	// An empty `encoding: {}` or `prefixEncoding: []` still writes the key, and
	// every decode path keeps that distinct from an absent one, so presence is
	// the nil check rather than the length.
	v.reportExclusions(mediaTypeExclusions, []fieldPresence{
		{name: fieldExample, present: mt.Example != nil},
		{name: fieldExamples, present: mt.Examples != nil},
		{name: fieldEncoding, present: mt.Encoding != nil},
		{name: fieldPrefixEncoding, present: mt.PrefixEncoding != nil},
		{name: fieldItemEncoding, present: mt.ItemEncoding != nil},
	}, prefix, result)

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
	if !encodingNeedsVisit(enc) || depth > maxEncodingNestingDepth {
		return
	}
	v.reportExclusions(encodingExclusions, []fieldPresence{
		{name: fieldEncoding, present: enc.Encoding != nil},
		{name: fieldPrefixEncoding, present: enc.PrefixEncoding != nil},
		{name: fieldItemEncoding, present: enc.ItemEncoding != nil},
	}, prefix, result)

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
