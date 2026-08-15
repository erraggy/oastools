// schema_equals_tuple_test.go covers Schema equality against the OAS 2.0 tuple
// form of `items`.
package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaEquals_TupleItems(t *testing.T) {
	tuple := func(items ...*Schema) *Schema {
		return &Schema{Type: "array", Items: items}
	}

	tests := []struct {
		name  string
		left  *Schema
		right *Schema
		equal bool
	}{
		{
			name:  "identical tuples are equal",
			left:  tuple(&Schema{Type: "string"}, &Schema{Ref: "#/definitions/Pet"}),
			right: tuple(&Schema{Type: "string"}, &Schema{Ref: "#/definitions/Pet"}),
			equal: true,
		},
		{
			name:  "a differing element makes them unequal",
			left:  tuple(&Schema{Type: "string"}, &Schema{Ref: "#/definitions/Pet"}),
			right: tuple(&Schema{Type: "string"}, &Schema{Ref: "#/definitions/Other"}),
		},
		{
			name:  "order matters",
			left:  tuple(&Schema{Type: "string"}, &Schema{Type: "number"}),
			right: tuple(&Schema{Type: "number"}, &Schema{Type: "string"}),
		},
		{
			name:  "length matters",
			left:  tuple(&Schema{Type: "string"}),
			right: tuple(&Schema{Type: "string"}, &Schema{Type: "number"}),
		},
		{
			name:  "a tuple and a single schema are not equal",
			left:  tuple(&Schema{Type: "string"}),
			right: &Schema{Type: "array", Items: &Schema{Type: "string"}},
		},
		{
			name:  "nil elements compare equal to each other",
			left:  tuple(nil, &Schema{Type: "string"}),
			right: tuple(nil, &Schema{Type: "string"}),
			equal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, tt.left.Equals(tt.right))
			assert.Equal(t, tt.equal, tt.right.Equals(tt.left), "equality should not depend on argument order")
		})
	}
}

// TestSchemaEquals_TupleItemsUsesSchemaEquality covers why the tuple arm exists
// rather than falling through to reflect.DeepEqual. Discriminator.StringForm is
// spelling rather than meaning, so two tuples differing only in it are equal.
func TestSchemaEquals_TupleItemsUsesSchemaEquality(t *testing.T) {
	withStringForm := func(stringForm bool) *Schema {
		return &Schema{
			Type: "array",
			Items: []*Schema{{
				Type:          "object",
				Discriminator: &Discriminator{PropertyName: "petType", StringForm: stringForm},
			}},
		}
	}

	assert.True(t, withStringForm(true).Equals(withStringForm(false)),
		"tuple elements were compared without Schema equality semantics")
}

// TestSchemaEquals_TupleItemsBoolForm covers a bare boolean schema in a tuple
// element: `items: [true]` arrives as a *Schema carrying BoolForm, never as a
// raw bool.
func TestSchemaEquals_TupleItemsBoolForm(t *testing.T) {
	boolSchema := func(v bool) *Schema {
		s := NewBoolSchema(v)
		return s
	}
	tuple := func(items ...*Schema) *Schema {
		return &Schema{Type: "array", Items: items}
	}

	assert.True(t, tuple(boolSchema(true)).Equals(tuple(boolSchema(true))),
		"two tuples holding the same boolean schema were reported different")
	assert.False(t, tuple(boolSchema(true)).Equals(tuple(boolSchema(false))),
		"true and false accept opposite things and are not the same schema")
	assert.False(t, tuple(boolSchema(true)).Equals(tuple(&Schema{Type: "string"})),
		"a boolean schema and a typed schema are not the same")
}

// TestSchemaEquals_BoolSpellingsAgree pins the two spellings of a boolean
// schema to one meaning. A scalar `items: true` decodes to a bare bool while
// the same value inside a tuple decodes to a *Schema, because []*Schema cannot
// hold a bool, so both spellings occur in parsed documents. The structural
// hasher in internal/schemautil writes one encoding for both; equality has to
// match it or a hash bucket holds members its comparison rejects (#504).
func TestSchemaEquals_BoolSpellingsAgree(t *testing.T) {
	arr := func(items any) *Schema {
		return &Schema{Type: "array", Items: items}
	}

	t.Run("the spellings compare equal, in both directions", func(t *testing.T) {
		assert.True(t, arr(true).Equals(arr(NewBoolSchema(true))))
		assert.True(t, arr(NewBoolSchema(true)).Equals(arr(true)))
		assert.True(t, arr(false).Equals(arr(NewBoolSchema(false))))
		assert.True(t, arr(NewBoolSchema(false)).Equals(arr(false)))
	})

	// The counterpart. Without it, this file would pass just as well if every
	// schema-or-bool comparison returned true.
	t.Run("and the value still decides", func(t *testing.T) {
		assert.False(t, arr(true).Equals(arr(NewBoolSchema(false))),
			"true and false accept opposite things")
		assert.False(t, arr(NewBoolSchema(true)).Equals(arr(false)))
		assert.False(t, arr(true).Equals(arr(&Schema{Type: "string"})),
			"a boolean schema is not an object schema in either spelling")
		assert.False(t, arr(&Schema{Type: "string"}).Equals(arr(true)))
		assert.False(t, arr(true).Equals(arr(nil)),
			"an absent field is not the schema `true`")
	})
}
