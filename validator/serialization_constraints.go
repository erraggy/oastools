package validator

import (
	"fmt"
	"regexp"

	"github.com/erraggy/oastools/parser"
)

// rfc9110Token matches the `token` production of RFC 9110 §5.6.2, which is what
// a field name must be. The official OAS 3.2 schema states it as a `$defs/token`
// referenced from `propertyNames` on every header map and from a header
// parameter's `name`.
//
// https://www.rfc-editor.org/rfc/rfc9110.html#section-5.6.2
var rfc9110Token = regexp.MustCompile(`^[0-9A-Za-z!#$%&'*+.^_` + "`" + `|~-]+$`)

// headerNameRulesApply reports whether the RFC 9110 token constraint on header
// names is in force.
//
// OAS 3.2 introduced it: the `token` definition does not exist in the 3.1 schema
// at all, so enforcing it there would reject documents 3.1 considers valid. An
// unrecognized version counts as in scope, matching oas32TraversalApplies.
func headerNameRulesApply(version parser.OASVersion) bool {
	return !version.IsValid() || version >= parser.OASVersion320
}

// emptyServerEnumApplies reports whether the non-empty Server Variable enum
// constraint is in force. OAS 3.1 added `minItems: 1`; 3.0's schema has no such
// constraint.
//
// Declared beside the other version gates so all four can be compared in one
// place. An unrecognized version counts as in scope, as it does for the rest.
func emptyServerEnumApplies(version parser.OASVersion) bool {
	return !version.IsValid() || version >= parser.OASVersion310
}

// validateHeaderName checks one header name against the RFC 9110 token rule.
// The name is a map key for a Header Object and the `name` field for a header
// parameter; both are field names on the wire, so both carry the constraint.
func (v *Validator) validateHeaderName(name, path, field string, result *ValidationResult) {
	if !headerNameRulesApply(v.oasVersion) {
		return
	}
	if name == "" || rfc9110Token.MatchString(name) {
		return
	}
	v.addError(result, path,
		fmt.Sprintf("Header name %q is not a valid HTTP field name; RFC 9110 allows only token characters (alphanumerics and !#$%%&'*+.^_`|~-)", name),
		withSpecRef(oas32SpecRef+"#header-object"),
		withField(field),
		withValue(name),
	)
}

// allowReservedPermitted reports whether `allowReserved` may appear on a
// parameter with the given `in` and `style`, for the document's version.
//
// The permitted set widened in 3.2, so this cannot be a single rule:
//
//   - 3.0: the schema lists allowReserved as a plain Parameter property with no
//     conditional, and the prose says it "only applies to" query parameters,
//     which is a statement about effect rather than validity. Not enforced.
//   - 3.1: the schema evaluates allowReserved only inside the `in: query`
//     branch, and `unevaluatedProperties: false` makes it invalid anywhere else.
//   - 3.2+: widened to the `in` and `style` combinations that percent-encode:
//     `in: path`, `in: query`, and `in: cookie` with `style: form`.
//
// Applying one version's rule to all of them fails in both directions: 3.1's
// would reject valid 3.2 documents, and 3.2's would accept invalid 3.1 ones.
func allowReservedPermitted(version parser.OASVersion, in, style string) bool {
	// 3.0 and earlier: structurally permitted, so nothing to enforce.
	if version.IsValid() && version < parser.OASVersion310 {
		return true
	}

	if version.IsValid() && version < parser.OASVersion320 {
		return in == parser.ParamInQuery
	}

	switch in {
	case parser.ParamInPath, parser.ParamInQuery:
		return true
	case parser.ParamInCookie:
		// `form` is the default style for a cookie parameter, so an unset
		// style is the permitted case rather than the forbidden one.
		return style == "" || style == "form"
	default:
		return false
	}
}

// validateParameterAllowReserved rejects `allowReserved` where the parameter's
// `in` and `style` do not permit it. See allowReservedPermitted for the
// per-version table.
func (v *Validator) validateParameterAllowReserved(param *parser.Parameter, path string, result *ValidationResult) {
	if !param.AllowReserved || allowReservedPermitted(v.oasVersion, param.In, param.Style) {
		return
	}
	v.addError(result, path,
		fmt.Sprintf("allowReserved is not permitted on a parameter with in: %q%s", param.In, styleSuffix(param.Style)),
		withSpecRef(v.specRef("#parameter-object")),
		withField("allowReserved"),
		withValue(true),
	)
}

// validateHeaderAllowReserved rejects `allowReserved` on a Header Object, where
// no OAS 3.x version permits it: the field appears nowhere in the Header
// Object's schema, and `unevaluatedProperties: false` closes the object.
//
// Only enforced for 3.1+, which is where the schema makes it structural; 3.0's
// draft-04 schema does not close the object the same way.
func (v *Validator) validateHeaderAllowReserved(header *parser.Header, path string, result *ValidationResult) {
	// parser.Header has no AllowReserved field, because no OAS version defines
	// one for a Header Object, so the key lands in Extra: the inline decoder
	// fills it with every unmatched field, not only x- extensions.
	//
	// This detects the one field rather than every unmatched one. A general
	// check for `unevaluatedProperties: false` needs the field/version matrix
	// (#439).
	if _, present := header.Extra["allowReserved"]; !present {
		return
	}
	if v.oasVersion.IsValid() && v.oasVersion < parser.OASVersion310 {
		return
	}
	v.addError(result, path,
		"allowReserved is not permitted on a Header Object; it applies only to parameters whose in and style percent-encode",
		withSpecRef(v.specRef("#header-object")),
		withField("allowReserved"),
		withValue(true),
	)
}

// styleSuffix renders the style clause of an allowReserved message, which is
// only informative when a style is actually set.
func styleSuffix(style string) string {
	if style == "" {
		return ""
	}
	return fmt.Sprintf(" and style: %q", style)
}
