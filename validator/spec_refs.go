package validator

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// Specification citations follow one of two forms, and which one a rule uses is
// decided by the rule, not by convenience:
//
//   - A rule whose wording or applicability varies by version cites the
//     document's own version, via [Validator.specRef] or a threaded baseURL.
//     Pointing a 3.1 document at the 3.2 text describes a rule it is not being
//     held to.
//   - A rule that exists at exactly one version cites that version directly.
//     [oas32SpecRef] is the 3.2 case; see the comment on it.
//
// Most of the validator threads baseURL down from [Validator.validateOAS3],
// which is fine where the parameter already exists. specRef covers the rules
// reached through traversals that carry no baseURL, using the version
// [Validator.oasVersion] already records for exactly this purpose.

// specBaseURL returns the specification URL for an OAS version.
//
// raw is the document's own version string, used only when the version is not
// one this build recognizes, so a citation for a future 3.x release still points
// somewhere plausible rather than nowhere.
func specBaseURL(version parser.OASVersion, raw string) string {
	switch version {
	case parser.OASVersion20:
		return "https://spec.openapis.org/oas/v2.0.html"
	case parser.OASVersion300:
		return "https://spec.openapis.org/oas/v3.0.0.html"
	case parser.OASVersion301:
		return "https://spec.openapis.org/oas/v3.0.1.html"
	case parser.OASVersion302:
		return "https://spec.openapis.org/oas/v3.0.2.html"
	case parser.OASVersion303:
		return "https://spec.openapis.org/oas/v3.0.3.html"
	case parser.OASVersion304:
		return "https://spec.openapis.org/oas/v3.0.4.html"
	case parser.OASVersion310:
		return "https://spec.openapis.org/oas/v3.1.0.html"
	case parser.OASVersion311:
		return "https://spec.openapis.org/oas/v3.1.1.html"
	case parser.OASVersion312:
		return "https://spec.openapis.org/oas/v3.1.2.html"
	case parser.OASVersion320:
		return "https://spec.openapis.org/oas/v3.2.0.html"
	default:
		if raw == "" {
			raw = version.String()
		}
		return fmt.Sprintf("https://spec.openapis.org/oas/v%s.html", raw)
	}
}

// specRef returns a citation for the document under validation, anchored at the
// given fragment (including its leading "#").
//
// For rules reached through a traversal that carries no baseURL. The version
// comes from [Validator.oasVersion], which exists so version-sensitive checks
// need not be plumbed through every call.
func (v *Validator) specRef(anchor string) string {
	return specBaseURL(v.oasVersion, "") + anchor
}
