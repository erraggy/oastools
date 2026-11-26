package builder

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/internal/stringutil"
	"github.com/erraggy/oastools/parser"
)

// infoErrors returns every problem that makes the info object invalid for the
// builder's OAS version. It runs at build time and stores nothing, since
// SetInfo keeps the caller's pointer and the object can change until then.
// The rules are listed in the package documentation.
func (b *Builder) infoErrors() []*BuilderError {
	info := b.info
	if info == nil {
		return []*BuilderError{NewMissingInfoError()}
	}

	var errs []*BuilderError
	add := func(field, format string, args ...any) {
		errs = append(errs, NewInfoError(field, fmt.Sprintf(format, args...)))
	}

	if strings.TrimSpace(info.Title) == "" {
		add("info.title", "title is required; call SetTitle")
	}
	if strings.TrimSpace(info.Version) == "" {
		add("info.version", "version is required (the version of your API, not of the OpenAPI Specification); call SetVersion")
	}

	if info.Summary != "" && b.version < parser.OASVersion310 {
		add("info.summary", "summary is not part of the Info Object before OAS 3.1; builder was created with version %s", b.version)
	}

	if info.TermsOfService != "" && !stringutil.IsValidURL(info.TermsOfService) {
		add("info.termsOfService", "termsOfService must be a URL, got %q", info.TermsOfService)
	}

	if c := info.Contact; c != nil {
		if c.URL != "" && !stringutil.IsValidURL(c.URL) {
			add("info.contact.url", "contact url must be a URL, got %q", c.URL)
		}
		if c.Email != "" && !stringutil.IsValidEmail(c.Email) {
			add("info.contact.email", "contact email must be an email address, got %q", c.Email)
		}
	}

	if l := info.License; l != nil {
		if strings.TrimSpace(l.Name) == "" {
			add("info.license.name", "license name is required when a license is set")
		}
		if l.URL != "" && !stringutil.IsValidURL(l.URL) {
			add("info.license.url", "license url must be a URL, got %q", l.URL)
		}
		switch {
		case l.Identifier != "" && b.version < parser.OASVersion310:
			add("info.license.identifier", "license identifier is not part of the License Object before OAS 3.1; use a url instead (builder was created with version %s)", b.version)
		case l.Identifier != "" && l.URL != "":
			add("info.license.identifier", "license identifier and url are mutually exclusive; set only one")
		case l.Identifier != "" && !stringutil.IsValidSPDXExpression(l.Identifier):
			add("info.license.identifier", "license identifier must be an SPDX identifier, got %q", l.Identifier)
		}
	}

	return errs
}
