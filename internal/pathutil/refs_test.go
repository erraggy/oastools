// Copyright 2024 Erraggy
// SPDX-License-Identifier: MIT

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
