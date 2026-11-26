// Package inforules holds the spec's Info, Contact and License Object rules,
// shared by the validator and the builder so the two cannot disagree.
package inforules

import (
	"fmt"

	"github.com/erraggy/oastools/internal/stringutil"
	"github.com/erraggy/oastools/parser"
)

// Object is the spec section anchor of the object a [Problem] is about.
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

// summaryAndIdentifierPermitted reports whether the version has the 3.1 fields
// info.summary and license.identifier. An unrecognized version permits them.
func summaryAndIdentifierPermitted(version parser.OASVersion) bool {
	return !version.IsValid() || version >= parser.OASVersion310
}

// termsOfServiceIsURL reports whether termsOfService must be a URL (3.0+; 2.0
// leaves it free text).
func termsOfServiceIsURL(version parser.OASVersion) bool {
	return version != parser.OASVersion20
}

// blank reports whether a required string is missing. The spec sets no
// minLength, so only "" (indistinguishable from absent) counts.
func blank(s string) bool {
	return s == ""
}

// Check returns every rule info breaks for version, in a fixed order. A nil
// info returns nil; reporting a missing info is the caller's job.
func Check(info *parser.Info, version parser.OASVersion) []Problem {
	if info == nil {
		return nil
	}

	var problems []Problem
	add := func(obj Object, path, field, value, format string, args ...any) {
		var msg string
		if len(args) > 0 {
			msg = fmt.Sprintf(format, args...)
		} else {
			msg = format
		}
		problems = append(problems, Problem{
			Path:    path,
			Field:   field,
			Object:  obj,
			Value:   value,
			Message: msg,
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
				// Independent problems, so both are reported.
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
