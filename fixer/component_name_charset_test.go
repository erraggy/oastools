package fixer

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// componentNamePattern is the OAS 3.x Components Object key rule, restated here
// so these tests assert against the spec rather than against the fixer's own
// idea of it. Kept in sync with validator.componentNamePattern.
var componentNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// oas3SpecNamed renders a one-schema OAS 3.0.3 document whose single $ref reaches
// that schema.
//
// A builder rather than a spec per case: here the schema *name* is the variable
// under test and everything around it is noise, unlike the encoding tests in
// generic_names_slash_test.go where the literal bytes of each $ref were the
// thing being pinned. The ref is built with EscapeRefToken so a name containing
// "/" or "~" still produces a document that resolves before the fix.
func oas3SpecNamed(name string) string {
	return fmt.Sprintf(`
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    %q: {type: object, properties: {id: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: %q}
`, name, pathutil.RefPrefixSchemas+pathutil.EscapeRefToken(name))
}

// oas2SpecNamed is the OAS 2.0 counterpart of [oas3SpecNamed].
func oas2SpecNamed(name string) string {
	return fmt.Sprintf(`
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  %q: {type: object, properties: {id: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          schema: {$ref: %q}
`, name, pathutil.RefPrefixDefinitions+pathutil.EscapeRefToken(name))
}

// illegalOAS3Names are names that violate the OAS 3.x component-name charset
// without containing any character from invalidSchemaNameChars, so a denylist
// cannot see them. Issue #405 reported the first of these; the rest are the same
// gap reached by other characters.
var illegalOAS3Names = []struct {
	name string
	want string
}{
	{"pkg/Pet", "pkg.Pet"},
	{"example.com/pkg/Pet", "example.com.pkg.Pet"},
	{"pet~summary", "pet_summary"},
	{"Pet@v1", "Pet_v1"},
	{"Pet#1", "Pet_1"},
	{"Pet&Co", "Pet_Co"},
	{"Pet%20", "Pet_20"},
	{"Pet:1", "Pet_1"},
	{"Pet(1)", "Pet_1"},
	{"Pet!", "Pet"},
	{"Pet$", "Pet"},
	{"Pet*", "Pet"},
	{"Pet+", "Pet"},
	{"Pet=", "Pet"},
	{"Pet?", "Pet"},
	{"Pet;", "Pet"},
	{"Pet'", "Pet"},
	// Non-ASCII letters: legal to unicode.IsLetter, illegal to the spec.
	{"Pét", "Pet"},
	{"café.Order", "cafe.Order"},
	{"Ünïcödé", "Unicode"},
	// No ASCII base form exists, so the sanitizer replaces it and the empty
	// result falls back rather than emitting an unnamed key.
	{"宠物", "UnnamedSchema"},
}

// TestFixSchemaNamesDetectsIllegalOAS3ComponentNames is the regression for
// issue #405.
//
// Detection used a denylist of characters while OAS 3.x enforces an allowlist,
// so any name illegal for a reason not on the list was never considered for
// renaming. --fix-schema-names reported no fix and left a document the validator
// rejects.
func TestFixSchemaNamesDetectsIllegalOAS3ComponentNames(t *testing.T) {
	for _, tt := range illegalOAS3Names {
		t.Run(tt.name, func(t *testing.T) {
			spec := oas3SpecNamed(tt.name)

			// The input is invalid, or this case proves nothing.
			parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
			require.NoError(t, err)
			before, err := validator.New().ValidateParsed(*parseResult)
			require.NoError(t, err)
			require.False(t, before.Valid, "the unfixed document should fail validation")

			result := fixSchemaNamesParsed(t, parseResult, GenericNamingUnderscore)
			require.True(t, result.HasFixes(), "the illegal name should be renamed")

			names := schemaNames(t, result.Document)
			assert.Contains(t, names, tt.want)
			assert.NotContains(t, names, tt.name, "the illegal key must be gone")
			assert.Equal(t, pathutil.RefPrefixSchemas+tt.want,
				responseSchemaRef(t, result.Document),
				"the $ref must follow the rename")

			after, err := validator.New().ValidateParsed(*result.ToParseResult())
			require.NoError(t, err)
			assert.True(t, after.Valid, "errors: %v", after.Errors)
		})
	}
}

// TestFixSchemaNamesLeavesValidOAS2NamesAlone pins the version-aware half of the
// decision.
//
// OAS 2.0 places no charset constraint on definitions keys, so every name here
// is legitimate and resolves correctly once escaped per RFC 6901. Renaming them
// would rewrite a valid document, so detection must stay on the denylist for
// 2.0 even though the identical name is renamed under 3.x.
func TestFixSchemaNamesLeavesValidOAS2NamesAlone(t *testing.T) {
	for _, tt := range illegalOAS3Names {
		t.Run(tt.name, func(t *testing.T) {
			spec := oas2SpecNamed(tt.name)

			// The premise: this document is already valid as OAS 2.0.
			parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
			require.NoError(t, err)
			before, err := validator.New().ValidateParsed(*parseResult)
			require.NoError(t, err)
			require.True(t, before.Valid, "spec should be valid OAS 2.0; errors: %v", before.Errors)

			result := fixSchemaNamesParsed(t, parseResult, GenericNamingUnderscore)

			assert.False(t, result.HasFixes(), "a valid OAS 2.0 name must not be renamed")
			assert.Contains(t, schemaNames(t, result.Document), tt.name,
				"the original key must survive untouched")
		})
	}
}

// TestFixSchemaNamesOAS2StillRenamesBracketNames is the control for the test
// above: leaving valid names alone must not stop OAS 2.0 renaming the names it
// always did.
func TestFixSchemaNamesOAS2StillRenamesBracketNames(t *testing.T) {
	spec := oas2SpecNamed("Response[User]")

	result := fixSchemaNames(t, spec, GenericNamingOf)

	require.True(t, result.HasFixes(), "a bracketed name is still renamed under OAS 2.0")
	assert.Contains(t, schemaNames(t, result.Document), "ResponseOfUser")
	assert.Equal(t, pathutil.RefPrefixDefinitions+"ResponseOfUser",
		responseSchemaRef(t, result.Document))
}

// TestFixSchemaNamesOAS2KeepsNonASCII is the second control: the replacement
// side is version-aware too. Folding "Pét" to "Pet" is required under OAS 3.x
// and unwanted under 2.0, where the accent is perfectly legal.
func TestFixSchemaNamesOAS2KeepsNonASCII(t *testing.T) {
	spec := oas2SpecNamed("Pét[User]")

	result := fixSchemaNames(t, spec, GenericNamingOf)

	require.True(t, result.HasFixes(), "the bracket still triggers a rename")
	assert.Contains(t, schemaNames(t, result.Document), "PétOfUser",
		"OAS 2.0 has no reason to strip the accent")
}

// TestTransformSchemaNameIsAlwaysLegalForOAS3 asserts the honest-fix invariant
// across the whole detection surface: if the fixer decides an OAS 3.x name needs
// renaming, the name it produces must satisfy the charset it renamed for.
//
// Without it, widening detection just converts silent no-ops into fixes that are
// reported as applied and leave the document equally invalid — which is how the
// accented-name case behaved before this change.
func TestTransformSchemaNameIsAlwaysLegalForOAS3(t *testing.T) {
	bases := []string{"", "R", "Resp", "pkg/v1.R", "Pét", "宠物", "a_", "example.com/x", "Ünï"}
	frags := []string{"", "[U]", "<U>", "[a,b]", "[pkg/x.Y]", "[Pét]", "[宠物]", " ", ",", "@", "%", "~", "!", "(", "\"", "宠"}
	strategies := []GenericNamingStrategy{
		GenericNamingUnderscore, GenericNamingOf, GenericNamingFor,
		GenericNamingFlattened, GenericNamingDot,
	}

	checked := 0
	for _, b := range bases {
		for _, f := range frags {
			for _, suffix := range []string{"", "X"} {
				name := b + f + suffix
				if !hasInvalidSchemaNameChars(name, charsetComponentName) {
					continue
				}
				for _, s := range strategies {
					cfg := GenericNamingConfig{Strategy: s, Separator: "_", ParamSeparator: "_"}
					got := transformSchemaName(name, cfg, charsetComponentName)
					checked++
					assert.Regexp(t, componentNamePattern, got,
						"transform of %q under %s produced an illegal component name", name, s)
					assert.False(t, hasInvalidSchemaNameChars(got, charsetComponentName),
						"transform of %q under %s produced a name that would be renamed again", name, s)
				}
			}
		}
	}
	require.Positive(t, checked, "the table should exercise some invalid names")
}

// TestHasInvalidSchemaNameCharsByCharset covers detection directly, including
// the names whose treatment differs by version.
func TestHasInvalidSchemaNameCharsByCharset(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantOK3 bool // detected under OAS 3.x
		wantOK2 bool // detected under OAS 2.0
	}{
		{"plain name", "Pet", false, false},
		{"dots and dashes are legal in both", "pkg.v1-Pet", false, false},
		{"underscores are legal in both", "__Pet__", false, false},
		{"brackets are caught by both", "Response[User]", true, true},
		{"space is caught by both", "Res ponse", true, true},
		{"slash is 3.x only", "pkg/Pet", true, false},
		{"tilde is 3.x only", "pet~summary", true, false},
		{"at sign is 3.x only", "Pet@v1", true, false},
		{"accented letter is 3.x only", "Pét", true, false},
		{"CJK is 3.x only", "宠物", true, false},
		{"empty is caught by both", "", true, true},
		{"whitespace only is caught by both", "   ", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantOK3, hasInvalidSchemaNameChars(tt.input, charsetComponentName), "OAS 3.x")
			assert.Equal(t, tt.wantOK2, hasInvalidSchemaNameChars(tt.input, charsetUnrestricted), "OAS 2.0")
		})
	}
}

// TestCharsetForVersion pins which rule each OAS version gets. Every 3.x
// release is listed rather than a sample, so adding a version without deciding
// its charset shows up here.
func TestCharsetForVersion(t *testing.T) {
	tests := []struct {
		version parser.OASVersion
		want    nameCharset
	}{
		{parser.OASVersion20, charsetUnrestricted},
		{parser.OASVersion300, charsetComponentName},
		{parser.OASVersion301, charsetComponentName},
		{parser.OASVersion302, charsetComponentName},
		{parser.OASVersion303, charsetComponentName},
		{parser.OASVersion304, charsetComponentName},
		{parser.OASVersion310, charsetComponentName},
		{parser.OASVersion311, charsetComponentName},
		{parser.OASVersion312, charsetComponentName},
		{parser.OASVersion320, charsetComponentName},
		// An unclassified document gets the stricter rule, matching how
		// schemaPathPrefix treats anything that is not OAS 2.0.
		{parser.Unknown, charsetComponentName},
	}

	for _, tt := range tests {
		t.Run(tt.version.String(), func(t *testing.T) {
			assert.Equal(t, tt.want, charsetForVersion(tt.version))
		})
	}
}

// TestFoldToASCII covers the diacritic stripping that keeps an accented name
// readable instead of reducing it to underscores.
func TestFoldToASCII(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"ascii is returned unchanged", "Pet", "Pet"},
		{"empty is unchanged", "", ""},
		{"acute accent", "Pét", "Pet"},
		{"several accents", "Ünïcödé", "Unicode"},
		{"accent inside a qualified name", "café.Order", "cafe.Order"},
		{"cedilla", "façade", "facade"},
		{"already-decomposed input", "Pét", "Pet"},
		{"no ascii base form is left alone", "宠物", "宠物"},
		{"mixed scripts keep what cannot fold", "Pét宠", "Pet宠"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, foldToASCII(tt.input))
		})
	}
}

// TestIsComponentNameChar covers the charset predicate against the pattern it
// restates, including the ASCII-only boundary that unicode.IsLetter gets wrong.
func TestIsComponentNameChar(t *testing.T) {
	for _, r := range []rune{'a', 'z', 'A', 'Z', '0', '9', '.', '-', '_'} {
		assert.True(t, isComponentNameChar(r), "%q should be allowed", r)
		assert.Regexp(t, componentNamePattern, string(r), "%q should match the pattern", r)
	}
	for _, r := range []rune{'/', '~', '@', '!', ' ', '[', ']', 'é', '宠', '中', 'Ω'} {
		assert.False(t, isComponentNameChar(r), "%q should be rejected", r)
		assert.NotRegexp(t, componentNamePattern, string(r), "%q should not match the pattern", r)
	}
}
