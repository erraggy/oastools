// Package inforules states, once, the rules the OpenAPI Specification places on
// the Info, Contact and License Objects, for every OAS version.
//
// The validator reports these rules against a parsed document and the builder
// refuses to build a document that breaks them. Both consume [Check] so that
// what the builder hands back is exactly what the validator accepts, and a rule
// added for one reaches the other (#558).
//
// As an internal package, it is not part of the public API and may change
// without notice.
package inforules

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/internal/stringutil"
	"github.com/erraggy/oastools/parser"
)

// Object names the specification object a [Problem] is about. Its value is the
// anchor of that object's section in the published specification.
type Object string

const (
	ObjectInfo    Object = "info-object"
	ObjectContact Object = "contact-object"
	ObjectLicense Object = "license-object"
)

// Problem is one broken rule.
type Problem struct {
	// Path is the full path of the offending field, e.g. "info.license.name".
	Path string
	// Field is the field's own name, e.g. "name".
	Field string
	// Object is the specification object that defines the field.
	Object Object
	// Value is the offending value, or "" when the problem is a missing field.
	Value string
	// Message describes the problem without naming the path.
	Message string
}

// summaryAndIdentifierPermitted reports whether the version defines info.summary
// and license.identifier, which OAS 3.1 introduced. An unrecognized version
// permits them, matching the validator's other version gates.
func summaryAndIdentifierPermitted(version parser.OASVersion) bool {
	return !version.IsValid() || version >= parser.OASVersion310
}

// termsOfServiceIsURL reports whether the version requires termsOfService to be
// a URL. OAS 2.0 describes it as free text; 3.0 says URL and 3.1 says URI.
func termsOfServiceIsURL(version parser.OASVersion) bool {
	return version != parser.OASVersion20
}

// blank reports whether a required string field is missing. A value of only
// whitespace names nothing, so it counts as missing.
func blank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// Check returns every rule info breaks for the given OAS version, in a fixed
// order. A nil info returns nil: whether the Info Object itself is required is
// the caller's to report, since only the caller knows the path it sits at.
func Check(info *parser.Info, version parser.OASVersion) []Problem {
	if info == nil {
		return nil
	}

	var problems []Problem
	add := func(obj Object, path, field, value, format string, args ...any) {
		problems = append(problems, Problem{
			Path:    path,
			Field:   field,
			Object:  obj,
			Value:   value,
			Message: fmt.Sprintf(format, args...),
		})
	}

	if blank(info.Title) {
		add(ObjectInfo, "info.title", "title", "", "Info object must have a title")
	}
	if blank(info.Version) {
		add(ObjectInfo, "info.version", "version", "", "Info object must have a version")
	}

	idPermitted := summaryAndIdentifierPermitted(version)
	if info.Summary != "" && !idPermitted {
		add(ObjectInfo, "info.summary", "summary", info.Summary,
			"summary is not part of the Info Object before OAS 3.1 (document is %s)", version)
	}

	if info.TermsOfService != "" && termsOfServiceIsURL(version) && !stringutil.IsValidURL(info.TermsOfService) {
		add(ObjectInfo, "info.termsOfService", "termsOfService", info.TermsOfService,
			"Invalid URL format: %s", info.TermsOfService)
	}

	if c := info.Contact; c != nil {
		if c.URL != "" && !stringutil.IsValidURL(c.URL) {
			add(ObjectContact, "info.contact.url", "url", c.URL, "Invalid URL format: %s", c.URL)
		}
		if c.Email != "" && !stringutil.IsValidEmail(c.Email) {
			add(ObjectContact, "info.contact.email", "email", c.Email, "Invalid email format: %s", c.Email)
		}
	}

	if l := info.License; l != nil {
		if blank(l.Name) {
			add(ObjectLicense, "info.license.name", "name", "", "License object must have a name")
		}
		if l.URL != "" && !stringutil.IsValidURL(l.URL) {
			add(ObjectLicense, "info.license.url", "url", l.URL, "Invalid URL format: %s", l.URL)
		}
		if l.Identifier != "" {
			if !idPermitted {
				add(ObjectLicense, "info.license.identifier", "identifier", l.Identifier,
					"identifier is not part of the License Object before OAS 3.1 (document is %s); use url instead", version)
			} else {
				// Both are reported: they are independent problems, and fixing
				// only one would leave the other to surface on the next run.
				if l.URL != "" {
					add(ObjectLicense, "info.license.identifier", "identifier", l.Identifier,
						"License identifier and url are mutually exclusive; set only one")
				}
				if !stringutil.IsValidSPDXExpression(l.Identifier) {
					add(ObjectLicense, "info.license.identifier", "identifier", l.Identifier,
						"Invalid SPDX license identifier format: %s", l.Identifier)
				}
			}
		}
	}

	return problems
}
