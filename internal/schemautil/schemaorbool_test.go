package schemautil

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaOrBoolSchemas(t *testing.T) {
	single := &parser.Schema{Ref: "#/definitions/Single"}
	first := &parser.Schema{Type: "string"}
	second := &parser.Schema{Ref: "#/definitions/Second"}

	type yielded struct {
		index  int
		schema *parser.Schema
	}

	tests := []struct {
		name  string
		field any
		want  []yielded
	}{
		{
			name:  "nil field yields nothing",
			field: nil,
		},
		{
			name:  "bool yields nothing",
			field: true,
		},
		{
			name:  "map is not a typed schema and yields nothing",
			field: map[string]any{"$ref": "#/definitions/Raw"},
		},
		{
			name:  "typed nil schema yields nothing",
			field: (*parser.Schema)(nil),
		},
		{
			name:  "single schema yields once with the single-form index",
			field: single,
			want:  []yielded{{SingleForm, single}},
		},
		{
			name:  "tuple yields each element with its position",
			field: []*parser.Schema{first, second},
			want:  []yielded{{0, first}, {1, second}},
		},
		{
			name:  "nil tuple elements are skipped but do not shift the others",
			field: []*parser.Schema{nil, second},
			want:  []yielded{{1, second}},
		},
		{
			name:  "empty tuple yields nothing",
			field: []*parser.Schema{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []yielded
			for i, s := range SchemaOrBoolSchemas(tt.field) {
				got = append(got, yielded{i, s})
			}
			assert.Len(t, got, len(tt.want))
			for i := range tt.want {
				assert.Equal(t, tt.want[i].index, got[i].index)
				assert.Same(t, tt.want[i].schema, got[i].schema)
			}
		})
	}
}

// TestSchemaOrBoolSchemas_EarlyBreak covers the yield-returned-false path, which
// a caller reaches by breaking out of the range loop.
func TestSchemaOrBoolSchemas_EarlyBreak(t *testing.T) {
	tuple := []*parser.Schema{{Type: "string"}, {Type: "number"}, {Type: "boolean"}}

	var visited int
	for range SchemaOrBoolSchemas(tuple) {
		visited++
		break
	}

	assert.Equal(t, 1, visited)
}

func TestIndexSuffix(t *testing.T) {
	assert.Empty(t, IndexSuffix(SingleForm), "single form addresses the field itself")
	assert.Equal(t, "[0]", IndexSuffix(0))
	assert.Equal(t, "[12]", IndexSuffix(12))
}

// TestHashDistinguishesTupleElements covers the structural hasher, which buckets
// candidates before anything compares them.
func TestHashDistinguishesTupleElements(t *testing.T) {
	tuple := func(items ...*parser.Schema) *parser.Schema {
		return &parser.Schema{Type: "array", Items: items}
	}
	str := func() *parser.Schema { return &parser.Schema{Type: "string"} }
	ref := func(name string) *parser.Schema { return &parser.Schema{Ref: "#/definitions/" + name} }

	h := NewSchemaHasher()

	assert.NotEqual(t, h.Hash(tuple(str(), ref("PetDetails"))), h.Hash(tuple(str(), ref("Other"))),
		"tuples with different elements hashed the same")
	assert.NotEqual(t, h.Hash(tuple(str(), ref("PetDetails"))), h.Hash(tuple(ref("PetDetails"), str())),
		"position is part of a tuple's meaning, so a reordering should not hash the same")
	assert.NotEqual(t, h.Hash(tuple(str())), h.Hash(tuple(str(), str())),
		"tuples of different length hashed the same")
	assert.Equal(t, h.Hash(tuple(str(), ref("PetDetails"))), h.Hash(tuple(str(), ref("PetDetails"))),
		"two equal tuples hashed differently")
}

// TestSchemaTuple covers the accessor that tells the OAS 2.0 tuple form apart
// from the other shapes a schema-or-bool field can hold. The empty tuple is the
// case with one meaning and two plausible readings: it is the tuple form, and it
// names no position, so the ok result carries what the length cannot.
func TestSchemaTuple(t *testing.T) {
	first := &parser.Schema{Type: "string"}

	tests := []struct {
		name      string
		field     any
		wantOK    bool
		wantCount int
	}{
		{
			name:      "tuple with positions",
			field:     []*parser.Schema{first, {Type: "integer"}},
			wantOK:    true,
			wantCount: 2,
		},
		{
			name:   "empty tuple is still the tuple form",
			field:  []*parser.Schema{},
			wantOK: true,
		},
		{
			name:      "a nil element keeps its position",
			field:     []*parser.Schema{first, nil, {Type: "integer"}},
			wantOK:    true,
			wantCount: 3,
		},
		{
			name:   "single schema form",
			field:  first,
			wantOK: false,
		},
		{
			name:   "boolean form",
			field:  true,
			wantOK: false,
		},
		{
			name:   "absent",
			field:  nil,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuple, ok := SchemaTuple(tt.field)
			assert.Equal(t, tt.wantOK, ok)
			assert.Len(t, tuple, tt.wantCount)
		})
	}

	// The positions come back indexed, nil elements included, which is what
	// separates this from SchemaOrBoolSchemas.
	tuple, ok := SchemaTuple([]*parser.Schema{first, nil})
	require.True(t, ok)
	assert.Same(t, first, tuple[0])
	assert.Nil(t, tuple[1])
}
