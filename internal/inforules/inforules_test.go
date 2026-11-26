package inforules

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
)

func paths(problems []Problem) []string {
	if len(problems) == 0 {
		return nil
	}
	out := make([]string, 0, len(problems))
	for _, p := range problems {
		out = append(out, p.Path)
	}
	return out
}

func valid() *parser.Info {
	return &parser.Info{Title: "API", Version: "1.0.0"}
}

func TestCheck_NilInfo(t *testing.T) {
	assert.Nil(t, Check(nil, parser.OASVersion320))
}

func TestCheck_VersionScopedRules(t *testing.T) {
	tests := []struct {
		name    string
		apply   func(*parser.Info)
		version parser.OASVersion
		want    []string
	}{
		{"minimal", func(*parser.Info) {}, parser.OASVersion20, nil},
		{"empty title and version", func(i *parser.Info) { i.Title, i.Version = "", "" }, parser.OASVersion304,
			[]string{"info.title", "info.version"}},
		// The spec requires a string and constrains it no further.
		{"whitespace title and version are present", func(i *parser.Info) { i.Title, i.Version = " ", "\t" }, parser.OASVersion304, nil},
		{"whitespace license name is present", func(i *parser.Info) { i.License = &parser.License{Name: " "} }, parser.OASVersion20, nil},

		// license.name, every version
		{"license without name 2.0", func(i *parser.Info) { i.License = &parser.License{URL: "https://e.com/l"} }, parser.OASVersion20,
			[]string{"info.license.name"}},
		{"empty license 3.2", func(i *parser.Info) { i.License = &parser.License{} }, parser.OASVersion320,
			[]string{"info.license.name"}},

		// termsOfService, from 3.0
		{"tos free text 2.0", func(i *parser.Info) { i.TermsOfService = "terms of service" }, parser.OASVersion20, nil},
		{"tos not a URI 3.0", func(i *parser.Info) { i.TermsOfService = "terms of service" }, parser.OASVersion300,
			[]string{"info.termsOfService"}},
		{"tos not a URI 3.1", func(i *parser.Info) { i.TermsOfService = "terms of service" }, parser.OASVersion311,
			[]string{"info.termsOfService"}},
		{"tos relative 3.0", func(i *parser.Info) { i.TermsOfService = "./terms.html" }, parser.OASVersion300, nil},

		// summary, from 3.1
		{"summary 3.0.4", func(i *parser.Info) { i.Summary = "s" }, parser.OASVersion304, []string{"info.summary"}},
		{"summary 2.0", func(i *parser.Info) { i.Summary = "s" }, parser.OASVersion20, []string{"info.summary"}},
		{"summary 3.1.0", func(i *parser.Info) { i.Summary = "s" }, parser.OASVersion310, nil},

		// identifier, from 3.1; the SPDX check does not run before it
		{"identifier 3.0.4", func(i *parser.Info) { i.License = &parser.License{Name: "MIT", Identifier: "MIT"} }, parser.OASVersion304,
			[]string{"info.license.identifier"}},
		{"malformed identifier 3.0 reported once", func(i *parser.Info) { i.License = &parser.License{Name: "MIT", Identifier: "not spdx"} },
			parser.OASVersion300, []string{"info.license.identifier"}},
		{"identifier 3.1.0", func(i *parser.Info) { i.License = &parser.License{Name: "MIT", Identifier: "MIT"} }, parser.OASVersion310, nil},
		{"malformed identifier 3.1", func(i *parser.Info) { i.License = &parser.License{Name: "MIT", Identifier: "not spdx"} },
			parser.OASVersion311, []string{"info.license.identifier"}},

		// identifier and url, exclusive from 3.1
		{"identifier and url 3.1", func(i *parser.Info) {
			i.License = &parser.License{Name: "MIT", Identifier: "MIT", URL: "https://e.com/l"}
		}, parser.OASVersion311, []string{"info.license.identifier"}},
		{"identifier and url 3.0 reports only the field", func(i *parser.Info) {
			i.License = &parser.License{Name: "MIT", Identifier: "MIT", URL: "https://e.com/l"}
		}, parser.OASVersion303, []string{"info.license.identifier"}},
		{"exclusive and malformed reports both", func(i *parser.Info) {
			i.License = &parser.License{Name: "MIT", Identifier: "MIT OR", URL: "https://e.com/l"}
		}, parser.OASVersion320, []string{"info.license.identifier", "info.license.identifier"}},

		// an unrecognized version permits the 3.1 fields, as the other gates do
		{"unknown version permits summary", func(i *parser.Info) { i.Summary = "s" }, parser.Unknown, nil},

		{"contact", func(i *parser.Info) { i.Contact = &parser.Contact{URL: "x y", Email: "y"} }, parser.OASVersion20,
			[]string{"info.contact.url", "info.contact.email"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := valid()
			tt.apply(info)
			assert.Equal(t, tt.want, paths(Check(info, tt.version)))
		})
	}
}

func TestCheck_ProblemDetails(t *testing.T) {
	info := valid()
	info.License = &parser.License{URL: "https://"}
	got := Check(info, parser.OASVersion320)
	assert.Equal(t, []Problem{
		{Path: "info.license.name", Field: "name", Object: ObjectLicense, Message: "License object must have a name"},
		{Path: "info.license.url", Field: "url", Object: ObjectLicense, Value: "https://", Message: "Invalid URL format: https://"},
	}, got)
}
