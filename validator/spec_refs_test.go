package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/erraggy/oastools/parser"
)

// TestSpecBaseURL pins the version-to-URL mapping. It is a switch, so the risk
// is a version silently falling through to the fallback when a new one is added
// to parser and not here: the citation would still look plausible, which is why
// every version is listed rather than spot-checked.
func TestSpecBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		version parser.OASVersion
		raw     string
		want    string
	}{
		{"2.0", parser.OASVersion20, "2.0", "https://spec.openapis.org/oas/v2.0.html"},
		{"3.0.0", parser.OASVersion300, "3.0.0", "https://spec.openapis.org/oas/v3.0.0.html"},
		{"3.0.1", parser.OASVersion301, "3.0.1", "https://spec.openapis.org/oas/v3.0.1.html"},
		{"3.0.2", parser.OASVersion302, "3.0.2", "https://spec.openapis.org/oas/v3.0.2.html"},
		{"3.0.3", parser.OASVersion303, "3.0.3", "https://spec.openapis.org/oas/v3.0.3.html"},
		{"3.0.4", parser.OASVersion304, "3.0.4", "https://spec.openapis.org/oas/v3.0.4.html"},
		{"3.1.0", parser.OASVersion310, "3.1.0", "https://spec.openapis.org/oas/v3.1.0.html"},
		{"3.1.1", parser.OASVersion311, "3.1.1", "https://spec.openapis.org/oas/v3.1.1.html"},
		{"3.1.2", parser.OASVersion312, "3.1.2", "https://spec.openapis.org/oas/v3.1.2.html"},
		{"3.2.0", parser.OASVersion320, "3.2.0", "https://spec.openapis.org/oas/v3.2.0.html"},
		{
			// A version this build does not recognize still gets a citation,
			// built from the document's own version string.
			name:    "unrecognized version uses the raw string",
			version: parser.OASVersion(0),
			raw:     "3.3.0",
			want:    "https://spec.openapis.org/oas/v3.3.0.html",
		},
		{
			// With no raw string to fall back on, the version's own String()
			// stands in rather than producing a malformed URL.
			name:    "unrecognized version with no raw string",
			version: parser.OASVersion(0),
			raw:     "",
			want:    "https://spec.openapis.org/oas/vunknown.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, specBaseURL(tt.version, tt.raw))
		})
	}
}

// TestValidatorSpecRef covers the accessor the traversal-reached rules use. It
// takes its version from Validator.oasVersion, which exists so version-sensitive
// checks need not be plumbed through every call.
func TestValidatorSpecRef(t *testing.T) {
	tests := []struct {
		name    string
		version parser.OASVersion
		anchor  string
		want    string
	}{
		{"3.1 parameter object", parser.OASVersion310, "#parameter-object", "https://spec.openapis.org/oas/v3.1.0.html#parameter-object"},
		{"3.2 header object", parser.OASVersion320, "#header-object", "https://spec.openapis.org/oas/v3.2.0.html#header-object"},
		{"2.0 schema object", parser.OASVersion20, "#schema-object", "https://spec.openapis.org/oas/v2.0.html#schema-object"},
		{"no anchor", parser.OASVersion320, "", "https://spec.openapis.org/oas/v3.2.0.html"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Validator{oasVersion: tt.version}
			assert.Equal(t, tt.want, v.specRef(tt.anchor))
		})
	}
}
