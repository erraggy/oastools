package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeSchemaValue is the seam the decode generator emits for every Schema
// position, so it is exercised directly here rather than only through a parsed
// document. The generator uses it in three shapes — a singular *Schema field, a
// []*Schema slice and a map[string]*Schema — and the ResolveRefs path is the
// only one that reaches any of them.
func TestDecodeSchemaValue(t *testing.T) {
	tests := []struct {
		name       string
		input      any
		wantNil    bool
		wantBool   bool
		wantIsBool bool
		wantType   any
	}{
		{name: "boolean true", input: true, wantBool: true, wantIsBool: true},
		{name: "boolean false", input: false, wantBool: false, wantIsBool: true},
		{name: "object schema", input: map[string]any{"type": "string"}, wantType: "string"},
		{name: "empty object schema", input: map[string]any{}},
		// Anything that is neither a schema object nor a boolean is not a
		// schema. Returning nil lets the caller omit the entry rather than
		// inventing an empty one.
		{name: "string is not a schema", input: "true", wantNil: true},
		{name: "number is not a schema", input: 1, wantNil: true},
		{name: "nil", input: nil, wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeSchemaValue(tt.input)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			value, isBool := got.IsBool()
			assert.Equal(t, tt.wantIsBool, isBool)
			assert.Equal(t, tt.wantBool, value)
			assert.Equal(t, tt.wantType, got.Type)
		})
	}
}

// TestDecodeFromMapSchemaShapes drives the three generated shapes through the
// ResolveRefs path, which is the route that reaches decodeFromMap. Before
// decodeSchemaValue existed, each of these dropped a boolean silently.
func TestDecodeFromMapSchemaShapes(t *testing.T) {
	spec := `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  schemas:
    mapShape: true
    ptrShape:
      not: false
    sliceShape:
      allOf:
        - true
        - type: string
        - false
    defsShape:
      $defs:
        inner: true
`
	p := New()
	p.ResolveRefs = true

	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)
	doc, ok := result.OAS3Document()
	require.True(t, ok, "expected an OAS3 document")

	schemas := doc.Components.Schemas
	require.Len(t, schemas, 4, "a schema was dropped by decodeFromMap")

	t.Run("map value", func(t *testing.T) {
		value, isBool := schemas["mapShape"].IsBool()
		assert.True(t, isBool)
		assert.True(t, value)
	})

	t.Run("singular pointer field", func(t *testing.T) {
		require.NotNil(t, schemas["ptrShape"].Not, "not: false was dropped")
		value, isBool := schemas["ptrShape"].Not.IsBool()
		assert.True(t, isBool)
		assert.False(t, value)
	})

	t.Run("slice element", func(t *testing.T) {
		allOf := schemas["sliceShape"].AllOf
		require.Len(t, allOf, 3, "a boolean member of allOf was dropped")

		first, isBool := allOf[0].IsBool()
		assert.True(t, isBool)
		assert.True(t, first)

		_, middleIsBool := allOf[1].IsBool()
		assert.False(t, middleIsBool, "the object member should not be a boolean schema")
		assert.Equal(t, "string", allOf[1].Type)

		last, isBool := allOf[2].IsBool()
		assert.True(t, isBool)
		assert.False(t, last)
	})

	t.Run("nested map value", func(t *testing.T) {
		inner := schemas["defsShape"].Defs["inner"]
		require.NotNil(t, inner, "$defs entry was dropped")
		value, isBool := inner.IsBool()
		assert.True(t, isBool)
		assert.True(t, value)
	})
}
