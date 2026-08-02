package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Schema.Type is `any`: a schema may declare one type or several, and the two
// forms decode to different Go types.
//
// The forms are asserted concretely rather than normalized. Which one a document
// carries is meaning rather than spelling, so an assertion accepting either would
// let a scalar become a sequence without failing anything. Reading code belongs in
// internal/schemautil; these tests pin what the decoders hand it.
const (
	typeFormsYAML = `
openapi: 3.1.0
info:
  title: T
  version: "1.0.0"
paths: {}
components:
  schemas:
    Plural:
      type: [string, "null"]
    Singular:
      type: string
`

	typeFormsJSON = `{
  "openapi": "3.1.0",
  "info": {"title": "T", "version": "1.0.0"},
  "paths": {},
  "components": {
    "schemas": {
      "Plural": {"type": ["string", "null"]},
      "Singular": {"type": "string"}
    }
  }
}`
)

// assertTypeForms pins both forms on one parsed document.
//
// The plural form is []any rather than []string: nothing in the decoders builds a
// []string, so a test accepting one would be asserting a shape the parser never
// produces. Construction by hand can still yield []string, which is why the
// reading helpers accept it.
func assertTypeForms(t *testing.T, doc *OAS3Document, label string) {
	t.Helper()

	require.NotNil(t, doc.Components, label)

	plural := doc.Components.Schemas["Plural"]
	require.NotNil(t, plural, "%s: Plural schema", label)
	pluralType, ok := plural.Type.([]any)
	require.True(t, ok, "%s: plural type should decode to []any, got %T", label, plural.Type)
	assert.Equal(t, []any{"string", "null"}, pluralType, "%s: plural type values", label)

	singular := doc.Components.Schemas["Singular"]
	require.NotNil(t, singular, "%s: Singular schema", label)
	singularType, ok := singular.Type.(string)
	require.True(t, ok, "%s: singular type should stay a string, got %T", label, singular.Type)
	assert.Equal(t, "string", singularType, "%s: singular type value", label)
}

// parseTypeForms parses one of the fixtures above by the requested route.
func parseTypeForms(t *testing.T, spec string, resolveRefs bool) *OAS3Document {
	t.Helper()

	p := New()
	p.ValidateStructure = false
	p.ResolveRefs = resolveRefs

	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	doc, ok := result.OAS3Document()
	require.True(t, ok)
	return doc
}

// TestSchemaTypeFormsSurviveEveryDecodePath covers the three decode paths parser
// keeps: YAML, JSON, and the generated decodeFromMap that ResolveRefs uses. They
// are separate implementations, so a test covering one covers about a third of
// the surface. Issue #397 is a case where two of them disagreed.
func TestSchemaTypeFormsSurviveEveryDecodePath(t *testing.T) {
	tests := []struct {
		name        string
		spec        string
		resolveRefs bool
	}{
		{name: "YAML", spec: typeFormsYAML},
		{name: "JSON", spec: typeFormsJSON},
		{name: "YAML through decodeFromMap", spec: typeFormsYAML, resolveRefs: true},
		{name: "JSON through decodeFromMap", spec: typeFormsJSON, resolveRefs: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTypeForms(t, parseTypeForms(t, tt.spec, tt.resolveRefs), tt.name)
		})
	}
}

// TestSchemaTypeFormsSurviveMarshalling re-parses the marshalled output, so a
// marshaller that flattened a sequence to its first element, or promoted a scalar
// to a sequence, fails here rather than silently rewriting a user's document.
func TestSchemaTypeFormsSurviveMarshalling(t *testing.T) {
	for _, tt := range []struct {
		name string
		spec string
	}{
		{name: "from YAML", spec: typeFormsYAML},
		{name: "from JSON", spec: typeFormsJSON},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, err := parseTypeForms(t, tt.spec, false).MarshalJSON()
			require.NoError(t, err)
			assertTypeForms(t, parseTypeForms(t, string(out), false), tt.name+" re-parsed")
		})
	}
}

// TestSchemaTypeFormsSurviveDeepCopy covers the generated DeepCopy, whose
// Schema.Type helper switches on the concrete form. A copy that dropped to the
// default arm would alias the original's slice rather than copy it.
func TestSchemaTypeFormsSurviveDeepCopy(t *testing.T) {
	doc := parseTypeForms(t, typeFormsYAML, false)
	cp := doc.DeepCopy()
	require.NotNil(t, cp)
	assertTypeForms(t, cp, "deep copy")

	// Copied, not aliased: mutating the copy must not reach the original.
	copied, ok := cp.Components.Schemas["Plural"].Type.([]any)
	require.True(t, ok)
	copied[0] = "mutated"

	original, ok := doc.Components.Schemas["Plural"].Type.([]any)
	require.True(t, ok)
	assert.Equal(t, "string", original[0], "deep copy should not alias the original's type slice")
}

// TestSchemaTypeFormsCompareEqual pins that the two documents are the same
// document, which is what makes the JSON and YAML fixtures above a pair rather
// than two unrelated specs.
func TestSchemaTypeFormsCompareEqual(t *testing.T) {
	fromYAML := parseTypeForms(t, typeFormsYAML, false)
	fromJSON := parseTypeForms(t, typeFormsJSON, false)

	for _, name := range []string{"Plural", "Singular"} {
		t.Run(name, func(t *testing.T) {
			assert.True(t, fromYAML.Components.Schemas[name].Equals(fromJSON.Components.Schemas[name]),
				"the YAML and JSON forms of %s should compare equal", name)
		})
	}
}
