// tuple_decode_parity_test.go pins the three decode paths to the same reading of
// an OAS 2.0 tuple-form items field: YAML through promoteYAMLSchemaOrBool, JSON
// through promoteSchemaOrBool, and the map decode through decodeSchemaOrBool,
// which is what ResolveRefs uses. The form is positional, so an element a path
// declines to represent must still hold its index: dropping one renumbers every
// element after it and the document means something else (#510).
package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tupleParityYAML = `swagger: "2.0"
info:
  title: T
  version: "1.0.0"
paths: {}
definitions:
  Tuple:
    type: array
    items:
      - type: string
      - true
      - null
      - type: integer
`

const tupleParityJSON = `{
  "swagger": "2.0",
  "info": {"title": "T", "version": "1.0.0"},
  "paths": {},
  "definitions": {
    "Tuple": {
      "type": "array",
      "items": [{"type": "string"}, true, null, {"type": "integer"}]
    }
  }
}`

// assertParityTuple states the one reading all three paths must produce.
func assertParityTuple(t *testing.T, items any) {
	t.Helper()

	schemas, ok := items.([]*Schema)
	require.True(t, ok, "expected the tuple form []*Schema, got %T", items)
	require.Len(t, schemas, 4, "every element holds its index")

	assert.Equal(t, "string", schemas[0].Type)

	v, isBool := schemas[1].IsBool()
	assert.True(t, isBool, "element 1 is the bare-boolean form")
	assert.True(t, v)

	assert.Nil(t, schemas[2], "an explicit null holds its slot")

	assert.Equal(t, "integer", schemas[3].Type,
		"element 3 keeps index 3: this is what a dropped element would move")
}

func TestTupleDecodeParityAcrossPaths(t *testing.T) {
	tests := []struct {
		name string
		src  string
		opts []Option
	}{
		{name: "YAML", src: tupleParityYAML},
		{name: "JSON", src: tupleParityJSON},
		{name: "YAML with ResolveRefs, the map decode", src: tupleParityYAML,
			opts: []Option{WithResolveRefs(true)}},
		{name: "JSON with ResolveRefs, the map decode", src: tupleParityJSON,
			opts: []Option{WithResolveRefs(true)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := append([]Option{WithBytes([]byte(tt.src))}, tt.opts...)
			result, err := ParseWithOptions(opts...)
			require.NoError(t, err)

			doc, ok := result.Document.(*OAS2Document)
			require.True(t, ok, "expected an OAS 2.0 document, got %T", result.Document)

			assertParityTuple(t, doc.Definitions["Tuple"].Items)
		})
	}
}

// TestTupleDecodeParityEqualsAcrossFormats is the counterpart the parity test
// needs: the two formats agreeing element by element is only useful if the
// documents they produce also compare equal, which is what `diff` asks.
func TestTupleDecodeParityEqualsAcrossFormats(t *testing.T) {
	fromYAML, err := ParseWithOptions(WithBytes([]byte(tupleParityYAML)))
	require.NoError(t, err)
	fromJSON, err := ParseWithOptions(WithBytes([]byte(tupleParityJSON)))
	require.NoError(t, err)

	yamlSchema := fromYAML.Document.(*OAS2Document).Definitions["Tuple"]
	jsonSchema := fromJSON.Document.(*OAS2Document).Definitions["Tuple"]

	assert.True(t, yamlSchema.Equals(jsonSchema), "same document, two formats")
	assert.True(t, jsonSchema.Equals(yamlSchema), "and equality is symmetric")
}

// TestTupleDecodeLengthStillDistinguishes is the mutation counterpart: a test
// that only asserts two tuples match would pass just as well if every tuple
// decoded to the same thing.
func TestTupleDecodeLengthStillDistinguishes(t *testing.T) {
	shorter := `{"swagger":"2.0","info":{"title":"T","version":"1.0.0"},"paths":{},` +
		`"definitions":{"Tuple":{"type":"array","items":[{"type":"string"},true,null]}}}`

	full, err := ParseWithOptions(WithBytes([]byte(tupleParityJSON)))
	require.NoError(t, err)
	short, err := ParseWithOptions(WithBytes([]byte(shorter)))
	require.NoError(t, err)

	fullSchema := full.Document.(*OAS2Document).Definitions["Tuple"]
	shortSchema := short.Document.(*OAS2Document).Definitions["Tuple"]

	require.Len(t, shortSchema.Items, 3)
	assert.False(t, fullSchema.Equals(shortSchema),
		"a tuple missing its last element is a different schema")
}
