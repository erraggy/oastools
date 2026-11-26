package converter

import (
	"fmt"

	"github.com/erraggy/oastools/internal/stringutil"
	"github.com/erraggy/oastools/parser"
)

// dropOAS31InfoFields removes the OAS 3.1 Info and License fields from info when
// the target predates 3.1, reporting each one removed. info must be the
// converted document's own copy, never the source's.
//
// The fields are removed rather than reported in place because the target's
// Info and License Objects are not extensible by fixed field, so keeping them
// leaves output the validator rejects. Severity is Warning, as for the 3.2
// annotations in detectOAS32Features: both fields are descriptive metadata, and
// losing them changes nothing an operation does.
//
// Gated on the target rather than the source version, because a document may
// carry a 3.1 field whatever version string it declares.
func (c *Converter) dropOAS31InfoFields(info *parser.Info, target parser.OASVersion, result *ConversionResult) {
	if info == nil || target >= parser.OASVersion310 {
		return
	}

	if info.Summary != "" {
		c.addIssueWithContext(result, "info.summary",
			fmt.Sprintf("'summary' is OAS 3.1+ only and has no equivalent in OAS %s; dropped", result.TargetVersion),
			fmt.Sprintf("The summary was %q. Move it into info.description, or keep the target at 3.1 or later", info.Summary))
		info.Summary = ""
	}

	if l := info.License; l != nil && l.Identifier != "" {
		c.addIssueWithContext(result, "info.license.identifier",
			fmt.Sprintf("'identifier' is OAS 3.1+ only and has no equivalent in OAS %s; dropped", result.TargetVersion),
			fmt.Sprintf("The SPDX identifier was %q. Set info.license.url to the license's URL, or keep the target at 3.1 or later", l.Identifier))
		l.Identifier = ""
	}
}

// dropFreeTextTermsOfService removes an info.termsOfService that is not a URL,
// reporting it. OAS 2.0 leaves the field free text and OAS 3.0+ requires a URL,
// so this applies to a conversion from 2.0 only: between 3.x versions the rule
// is the same on both sides. info must be the converted document's own copy.
func (c *Converter) dropFreeTextTermsOfService(info *parser.Info, result *ConversionResult) {
	if info == nil || info.TermsOfService == "" || stringutil.IsValidURL(info.TermsOfService) {
		return
	}
	c.addIssueWithContext(result, "info.termsOfService",
		fmt.Sprintf("'termsOfService' is not a URL, which OAS %s requires; dropped", result.TargetVersion),
		fmt.Sprintf("OAS 2.0 accepts free text here and the value was %q. Publish the terms at a URL and set termsOfService to it, or move the text into info.description", info.TermsOfService))
	info.TermsOfService = ""
}
