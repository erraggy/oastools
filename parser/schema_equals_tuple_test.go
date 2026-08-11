// schema_equals_tuple_test.go covers Schema equality against the OAS 2.0 tuple
// form of `items`, where the field holds a list of schemas rather than one.
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
// rather than falling through to reflect.DeepEqual: elements have to get Schema
// equality, which ignores the fields Schema.Equals deliberately ignores.
// Discriminator.StringForm is spelling rather than meaning, so two tuples
// differing only in it are the same schema.
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

// TestSchemaEquals_TupleItemsBoolForm covers a bare boolean schema sitting in a
// tuple element. A tuple element is always a *Schema, so `items: [true]` arrives
// as a schema carrying BoolForm rather than as a raw bool, and equality has to
// read it as the boolean schema it is.
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
