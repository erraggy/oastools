package pathutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeRefToken(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no escaping needed", input: "Pet", want: "Pet"},
		{name: "empty", input: "", want: ""},
		{name: "slash", input: "pet/id", want: "pet~1id"},
		{name: "tilde", input: "pet~id", want: "pet~0id"},
		{
			// Escaping "/" first would leave "~1", which escaping "~" would then
			// turn into "~01" — a token that unescapes to "~1", not "~/".
			name:  "tilde before slash",
			input: "pet~/id",
			want:  "pet~0~1id",
		},
		{
			// The literal text "~1" must survive as a name, distinct from an
			// escaped "/".
			name:  "literal escape sequence is itself escaped",
			input: "pet~1id",
			want:  "pet~01id",
		},
		{name: "multiple of each", input: "a/b~c/d", want: "a~1b~0c~1d"},
		{name: "only separators", input: "/~", want: "~1~0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EscapeRefToken(tt.input))
		})
	}
}

func TestUnescapeRefToken(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no escaping present", input: "Pet", want: "Pet"},
		{name: "empty", input: "", want: ""},
		{name: "escaped slash", input: "pet~1id", want: "pet/id"},
		{name: "escaped tilde", input: "pet~0id", want: "pet~id"},
		{name: "both", input: "pet~0~1id", want: "pet~/id"},
		{
			// "~01" must decode to the literal "~1" rather than to "/", which is
			// what unescaping "~0" first would produce.
			name:  "escaped tilde followed by one",
			input: "pet~01id",
			want:  "pet~1id",
		},
		{name: "only separators", input: "~1~0", want: "/~"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, UnescapeRefToken(tt.input))
		})
	}
}

// TestEscapeUnescapeRoundTrip is the property that matters: every name must
// survive being turned into a pointer token and back, or renaming and pruning
// silently corrupt it.
func TestEscapeUnescapeRoundTrip(t *testing.T) {
	names := []string{
		"Pet",
		"",
		"pet/id",
		"pet~id",
		"pet~/id",
		"pet~1id",
		"pet~0id",
		"~",
		"/",
		"a/b~c/d",
		"microsoft.graph.user",
		"Pet-Summary_v2",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, name, UnescapeRefToken(EscapeRefToken(name)))
		})
	}
}

// TestRefBuildersEscape checks that the builders produce the pointer a document
// legitimately carries, rather than one assembled by raw concatenation.
func TestRefBuildersEscape(t *testing.T) {
	assert.Equal(t, "#/components/schemas/pet~1summary", SchemaRef("pet/summary"))
	assert.Equal(t, "#/definitions/pet~1summary", DefinitionRef("pet/summary"))
	assert.Equal(t, "#/parameters/pet~1id", ParameterRef("pet/id", true))
	assert.Equal(t, "#/components/parameters/pet~1id", ParameterRef("pet/id", false))
	assert.Equal(t, "#/responses/not~1found", ResponseRef("not/found", true))
	assert.Equal(t, "#/components/responses/not~1found", ResponseRef("not/found", false))
	assert.Equal(t, "#/securityDefinitions/o~1auth", SecuritySchemeRef("o/auth", true))
	assert.Equal(t, "#/components/securitySchemes/o~1auth", SecuritySchemeRef("o/auth", false))
	assert.Equal(t, "#/components/headers/x~1rate", HeaderRef("x/rate"))
	assert.Equal(t, "#/components/requestBodies/pet~1body", RequestBodyRef("pet/body"))
	assert.Equal(t, "#/components/examples/pet~1example", ExampleRef("pet/example"))
	assert.Equal(t, "#/components/links/pet~1link", LinkRef("pet/link"))
	assert.Equal(t, "#/components/callbacks/pet~1cb", CallbackRef("pet/cb"))
	assert.Equal(t, "#/components/pathItems/pet~1item", PathItemRef("pet/item"))

	// Names needing no escaping are unchanged, so ordinary specs are unaffected.
	assert.Equal(t, "#/components/schemas/Pet", SchemaRef("Pet"))
	assert.Equal(t, "#/definitions/Pet", DefinitionRef("Pet"))
}

// TestDecodeRefToken covers both escaping conventions and the mixed spellings
// that satisfy neither, which is what the function exists for.
func TestDecodeRefToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"plain name is unchanged", "Pet", "Pet"},
		{"empty is unchanged", "", ""},
		{"percent-encoded brackets", "Response%5BUser%5D", "Response[User]"},
		{"JSON Pointer slash", "pet~1summary", "pet/summary"},
		{"JSON Pointer tilde", "pet~0summary", "pet~summary"},
		{"percent-encoded slash", "pkg%2FPet", "pkg/Pet"},
		{"mixed: encoded brackets, raw slashes", "Paged%5Bexample.com/pkg.Pet%5D", "Paged[example.com/pkg.Pet]"},
		{"mixed: encoded brackets, pointer slashes", "Paged%5Bexample.com~1pkg.Pet%5D", "Paged[example.com/pkg.Pet]"},
		{"fully percent-encoded", "Paged%5Bexample.com%2Fpkg.Pet%5D", "Paged[example.com/pkg.Pet]"},
		{"percent-encoded tilde becomes a pointer escape", "pet%7E1summary", "pet/summary"},
		{"invalid percent sequence is left alone", "Discount%OFF", "Discount%OFF"},
		{"lone percent is left alone", "100%", "100%"},
		{"plus is not a space", "a+b", "a+b"},
		{"escaped tilde-one survives", "x~01y", "x~1y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DecodeRefToken(tt.token))
		})
	}
}

// TestCutRefPrefix covers the percent-encoded pointer separators the raw
// strings.CutPrefix path cannot see, and the boundary cases where a partially
// decodable ref must be rejected rather than half-matched.
func TestCutRefPrefix(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		prefix   string
		wantRest string
		wantOK   bool
	}{
		{
			name:     "raw prefix",
			ref:      "#/components/schemas/Pet",
			prefix:   RefPrefixSchemas,
			wantRest: "Pet",
			wantOK:   true,
		},
		{
			name:     "fully percent-encoded pointer",
			ref:      "#%2Fcomponents%2Fschemas%2FPaged%5BPet%5D",
			prefix:   RefPrefixSchemas,
			wantRest: "Paged%5BPet%5D",
			wantOK:   true,
		},
		{
			name:     "lowercase hex decodes the same as uppercase",
			ref:      "#%2fcomponents%2fschemas%2fPet",
			prefix:   RefPrefixSchemas,
			wantRest: "Pet",
			wantOK:   true,
		},
		{
			name:     "mixed raw and encoded separators",
			ref:      "#/components%2Fschemas/Pet",
			prefix:   RefPrefixSchemas,
			wantRest: "Pet",
			wantOK:   true,
		},
		{
			name:     "the name keeps its own escaping, so DecodeRefToken stays the one decoder",
			ref:      "#%2Fdefinitions%2FPaged%255BPet%255D",
			prefix:   RefPrefixDefinitions,
			wantRest: "Paged%255BPet%255D",
			wantOK:   true,
		},
		{
			name:     "encoded prefix for the other OAS version does not match",
			ref:      "#%2Fdefinitions%2FPet",
			prefix:   RefPrefixSchemas,
			wantRest: "",
			wantOK:   false,
		},
		{
			name:     "a percent sequence that decodes to the wrong byte does not match",
			ref:      "#%2Bcomponents%2Fschemas%2FPet",
			prefix:   RefPrefixSchemas,
			wantRest: "",
			wantOK:   false,
		},
		{
			name:     "an invalid percent sequence does not match",
			ref:      "#%ZZcomponents%2Fschemas%2FPet",
			prefix:   RefPrefixSchemas,
			wantRest: "",
			wantOK:   false,
		},
		{
			name:     "a truncated percent sequence does not run off the end",
			ref:      "#%2Fcomponents%2Fschemas%2",
			prefix:   RefPrefixSchemas,
			wantRest: "",
			wantOK:   false,
		},
		{
			name:     "a ref shorter than its prefix does not match",
			ref:      "#/components/",
			prefix:   RefPrefixSchemas,
			wantRest: "",
			wantOK:   false,
		},
		{
			name:     "the bare prefix matches with an empty remainder",
			ref:      "#%2Fcomponents%2Fschemas%2F",
			prefix:   RefPrefixSchemas,
			wantRest: "",
			wantOK:   true,
		},
		{
			name:     "a ref with no percent at all takes the reject path",
			ref:      "#/definitions/Pet",
			prefix:   RefPrefixSchemas,
			wantRest: "",
			wantOK:   false,
		},
		{
			name:     "an external ref does not match a local prefix",
			ref:      "other.yaml#/components/schemas/Pet",
			prefix:   RefPrefixSchemas,
			wantRest: "",
			wantOK:   false,
		},
		{
			name:     "an external ref with an encoded pointer still does not match",
			ref:      "other.yaml#%2Fcomponents%2Fschemas%2FPet",
			prefix:   RefPrefixSchemas,
			wantRest: "",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rest, ok := CutRefPrefix(tt.ref, tt.prefix)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantRest, rest)
		})
	}
}
