package builder

import (
	"errors"
	"strings"
	"testing"

	"github.com/erraggy/oastools/oaserrors"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// infoVersions covers one version per distinct rule set: 2.0, the 3.0 line, and
// the two 3.1+ releases, so the version gates are exercised on both sides.
var infoVersions = []parser.OASVersion{
	parser.OASVersion20,
	parser.OASVersion300,
	parser.OASVersion304,
	parser.OASVersion310,
	parser.OASVersion311,
	parser.OASVersion320,
}

// buildAny builds with whichever method the builder's version calls for.
func buildAny(b *Builder) error {
	if b.version == parser.OASVersion20 {
		_, err := b.BuildOAS2()
		return err
	}
	_, err := b.BuildOAS3()
	return err
}

// infoFields returns the Field of every info error in err.
func infoFields(t *testing.T, err error) []string {
	t.Helper()
	var all BuilderErrors
	if !errors.As(err, &all) {
		var one *BuilderError
		require.ErrorAs(t, err, &one)
		all = BuilderErrors{one}
	}
	var fields []string
	for _, e := range all {
		if e.Component == ComponentInfo {
			fields = append(fields, e.Field)
		}
	}
	return fields
}

// validatorInfoFields validates a minimal document carrying info and returns
// the path of every error the validator reports under info.
func validatorInfoFields(t *testing.T, version parser.OASVersion, info *parser.Info) []string {
	t.Helper()
	var doc any
	if version == parser.OASVersion20 {
		doc = &parser.OAS2Document{Swagger: "2.0", OASVersion: version, Info: info, Paths: parser.Paths{}}
	} else {
		doc = &parser.OAS3Document{OpenAPI: version.String(), OASVersion: version, Info: info, Paths: parser.Paths{}}
	}
	result, err := validator.ValidateWithOptions(validator.WithParsed(parser.ParseResult{
		Version: version.String(), OASVersion: version, Document: doc,
	}))
	require.NoError(t, err)
	var fields []string
	for _, e := range result.Errors {
		if e.Path == "info" || strings.HasPrefix(e.Path, "info.") {
			fields = append(fields, e.Path)
		}
	}
	return fields
}

func TestBuilder_InfoRequiredFields(t *testing.T) {
	for _, version := range infoVersions {
		t.Run(version.String(), func(t *testing.T) {
			t.Run("nil info", func(t *testing.T) {
				err := buildAny(New(version))
				require.Error(t, err)
				assert.Equal(t, []string{"info"}, infoFields(t, err))
			})

			t.Run("title only", func(t *testing.T) {
				err := buildAny(New(version).SetTitle("API"))
				require.Error(t, err)
				assert.Equal(t, []string{"info.version"}, infoFields(t, err))
			})

			t.Run("version only", func(t *testing.T) {
				err := buildAny(New(version).SetVersion("1.0.0"))
				require.Error(t, err)
				assert.Equal(t, []string{"info.title"}, infoFields(t, err))
			})

			t.Run("description only reports both", func(t *testing.T) {
				// SetDescription alone creates a non-nil Info, which used to
				// build and then fail validation twice over.
				err := buildAny(New(version).SetDescription("d"))
				require.Error(t, err)
				assert.ElementsMatch(t, []string{"info.title", "info.version"}, infoFields(t, err))
			})

			t.Run("whitespace is a title and version", func(t *testing.T) {
				// The spec sets no minLength.
				require.NoError(t, buildAny(New(version).SetTitle("  ").SetVersion("\t")))
			})

			t.Run("empty Info struct", func(t *testing.T) {
				err := buildAny(NewWithInfo(version, &parser.Info{}))
				require.Error(t, err)
				assert.ElementsMatch(t, []string{"info.title", "info.version"}, infoFields(t, err))
			})

			t.Run("complete", func(t *testing.T) {
				require.NoError(t, buildAny(New(version).SetTitle("API").SetVersion("1.0.0")))
			})
		})
	}
}

func TestBuilder_InfoErrorsAreConfigErrors(t *testing.T) {
	err := buildAny(New(parser.OASVersion320).SetDescription("d"))
	require.Error(t, err)
	assert.ErrorIs(t, err, oaserrors.ErrConfig)
	assert.Contains(t, err.Error(), "info.title")
	assert.Contains(t, err.Error(), "info.version")
}

func TestBuilder_InfoFieldRules(t *testing.T) {
	const (
		pre31  = "pre31"
		from31 = "from31"
		from30 = "from30"
		v20    = "v20"
		all    = "all"
	)

	tests := []struct {
		name   string
		apply  func(*parser.Info)
		scope  string   // which versions the rule applies to
		fields []string // info errors expected where it applies
	}{
		{
			name: "valid contact and license url",
			apply: func(i *parser.Info) {
				i.Contact = &parser.Contact{Name: "n", URL: "https://example.com/support", Email: "a@example.com"}
				i.License = &parser.License{Name: "MIT", URL: "https://opensource.org/licenses/MIT"}
				i.TermsOfService = "https://example.com/terms"
			},
			scope: all,
		},
		{
			name: "relative references are URLs",
			apply: func(i *parser.Info) {
				i.TermsOfService = "./terms.html"
				i.Contact = &parser.Contact{URL: "support"}
				i.License = &parser.License{Name: "MIT", URL: "../LICENSE"}
			},
			scope: all,
		},
		{
			name:   "termsOfService not a URI reference",
			apply:  func(i *parser.Info) { i.TermsOfService = "terms of service" },
			scope:  from30,
			fields: []string{"info.termsOfService"},
		},
		{
			name:  "termsOfService is free text in 2.0",
			apply: func(i *parser.Info) { i.TermsOfService = "terms of service" },
			scope: v20,
		},
		{
			name:   "contact url not a URI reference",
			apply:  func(i *parser.Info) { i.Contact = &parser.Contact{URL: "support page"} },
			scope:  all,
			fields: []string{"info.contact.url"},
		},
		{
			name:   "contact email not an email",
			apply:  func(i *parser.Info) { i.Contact = &parser.Contact{Email: "not-an-email"} },
			scope:  all,
			fields: []string{"info.contact.email"},
		},
		{
			name:   "license without a name",
			apply:  func(i *parser.Info) { i.License = &parser.License{URL: "https://example.com/l"} },
			scope:  all,
			fields: []string{"info.license.name"},
		},
		{
			name:   "empty license",
			apply:  func(i *parser.Info) { i.License = &parser.License{} },
			scope:  all,
			fields: []string{"info.license.name"},
		},
		{
			name:   "license url not a url",
			apply:  func(i *parser.Info) { i.License = &parser.License{Name: "MIT", URL: "https://"} },
			scope:  all,
			fields: []string{"info.license.url"},
		},
		{
			name:   "summary does not exist before 3.1",
			apply:  func(i *parser.Info) { i.Summary = "short" },
			scope:  pre31,
			fields: []string{"info.summary"},
		},
		{
			name:  "summary is fine from 3.1",
			apply: func(i *parser.Info) { i.Summary = "short" },
			scope: from31,
		},
		{
			name:   "identifier does not exist before 3.1",
			apply:  func(i *parser.Info) { i.License = &parser.License{Name: "MIT", Identifier: "MIT"} },
			scope:  pre31,
			fields: []string{"info.license.identifier"},
		},
		{
			name:  "identifier is fine from 3.1",
			apply: func(i *parser.Info) { i.License = &parser.License{Name: "MIT", Identifier: "MIT"} },
			scope: from31,
		},
		{
			name: "identifier and url are exclusive from 3.1",
			apply: func(i *parser.Info) {
				i.License = &parser.License{Name: "MIT", Identifier: "MIT", URL: "https://example.com/l"}
			},
			scope:  from31,
			fields: []string{"info.license.identifier"},
		},
		{
			name:   "identifier with a space is not SPDX",
			apply:  func(i *parser.Info) { i.License = &parser.License{Name: "MIT", Identifier: "not spdx"} },
			scope:  from31,
			fields: []string{"info.license.identifier"},
		},
		{
			name: "identifier that is exclusive with url and malformed reports both",
			apply: func(i *parser.Info) {
				i.License = &parser.License{Name: "MIT", Identifier: "MIT OR", URL: "https://example.com/l"}
			},
			scope:  from31,
			fields: []string{"info.license.identifier", "info.license.identifier"},
		},
		{
			name: "several problems are all reported",
			apply: func(i *parser.Info) {
				i.Contact = &parser.Contact{URL: "x y", Email: "y"}
				i.License = &parser.License{}
			},
			scope: all,
			fields: []string{
				"info.contact.url", "info.contact.email", "info.license.name",
			},
		},
	}

	for _, tt := range tests {
		for _, version := range infoVersions {
			applies := tt.scope == all ||
				(tt.scope == v20 && version == parser.OASVersion20) ||
				(tt.scope == from30 && version != parser.OASVersion20) ||
				(tt.scope == pre31 && version < parser.OASVersion310) ||
				(tt.scope == from31 && version >= parser.OASVersion310)
			if !applies {
				continue
			}

			t.Run(tt.name+"/"+version.String(), func(t *testing.T) {
				info := &parser.Info{Title: "API", Version: "1.0.0"}
				tt.apply(info)

				// The validator shares the rules, so must agree (#558).
				assert.ElementsMatch(t, tt.fields, validatorInfoFields(t, version, info),
					"validator and builder disagree")

				err := buildAny(NewWithInfo(version, info))
				if len(tt.fields) == 0 {
					require.NoError(t, err)
					return
				}
				require.Error(t, err)
				assert.ElementsMatch(t, tt.fields, infoFields(t, err))
			})
		}
	}
}

// TestBuilder_InfoAcceptedIsValidated is the property the checks exist for:
// what the builder hands back, the validator accepts. It builds a spread of
// infos the builder accepts and validates each one.
func TestBuilder_InfoAcceptedIsValidated(t *testing.T) {
	infos := map[string]func(*parser.Info, parser.OASVersion){
		"minimal": func(*parser.Info, parser.OASVersion) {},
		"full": func(i *parser.Info, v parser.OASVersion) {
			i.Description = "d"
			i.TermsOfService = "https://example.com/terms"
			i.Contact = &parser.Contact{Name: "n", URL: "https://example.com", Email: "a@example.com"}
			i.License = &parser.License{Name: "Apache 2.0", URL: "https://www.apache.org/licenses/LICENSE-2.0.html"}
			if v >= parser.OASVersion310 {
				i.Summary = "s"
			}
		},
		"spdx identifier": func(i *parser.Info, v parser.OASVersion) {
			if v >= parser.OASVersion310 {
				i.License = &parser.License{Name: "Apache 2.0", Identifier: "Apache-2.0"}
			}
		},
		"spdx expression": func(i *parser.Info, v parser.OASVersion) {
			if v >= parser.OASVersion310 {
				i.License = &parser.License{Name: "Dual", Identifier: "MIT OR (Apache-2.0 AND BSD-3-Clause)"}
			}
		},
		"relative urls": func(i *parser.Info, _ parser.OASVersion) {
			i.TermsOfService = "/terms"
			i.Contact = &parser.Contact{URL: "/support"}
		},
	}

	for name, apply := range infos {
		for _, version := range infoVersions {
			t.Run(name+"/"+version.String(), func(t *testing.T) {
				info := &parser.Info{Title: "API", Version: "1.0.0"}
				apply(info, version)

				result, err := NewWithInfo(version, info).BuildResult()
				require.NoError(t, err)

				val, err := validator.ValidateWithOptions(validator.WithParsed(*result))
				require.NoError(t, err)
				assert.True(t, val.Valid, "builder accepted a document the validator rejects: %v", val.Errors)
			})
		}
	}
}

func TestBuilder_InfoFixedAfterFailedBuild(t *testing.T) {
	// SetInfo keeps the caller's pointer, so fixing it after a failed build
	// must be enough; nothing from the failed build may linger.
	info := &parser.Info{Title: "API"}
	b := NewWithInfo(parser.OASVersion320, info)

	require.Error(t, buildAny(b))

	info.Version = "1.0.0"
	require.NoError(t, buildAny(b))
}

func TestBuilder_FailedBuildLeavesSchemasAlone(t *testing.T) {
	// A build refused for its info must not run deduplication, or the retry
	// finds nothing left to alias and keeps a $ref to a removed schema.
	info := &parser.Info{Title: "API"}
	b := New(parser.OASVersion320, WithSemanticDeduplication(true)).SetInfo(info)
	shape := func() *parser.Schema {
		return &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{"name": {Type: "string"}}}
	}
	b.addSchema("Address", shape())
	b.addSchema("Location", shape())
	b.addSchema("Order", &parser.Schema{
		Type:       "object",
		Properties: map[string]*parser.Schema{"shipTo": {Ref: "#/components/schemas/Location"}},
	})

	_, err := b.BuildOAS3()
	require.Error(t, err)
	assert.Len(t, b.schemas, 3, "a refused build changed the builder's schemas")

	info.Version = "1.0.0"
	doc, err := b.BuildOAS3()
	require.NoError(t, err)
	require.Len(t, doc.Components.Schemas, 2)
	assert.Equal(t, "#/components/schemas/Address", doc.Components.Schemas["Order"].Properties["shipTo"].Ref)
}
