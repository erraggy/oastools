// tuple_items_test.go covers joining against the OAS 2.0 tuple form of `items`,
// where the field holds a list of schemas rather than one.
package joiner

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTupleItemsRefFollowsARename covers #502 from the joiner's side: a schema
// renamed to settle a collision has to take every $ref pointing at it along,
// and a $ref the rewrite never visits is left naming a schema that is gone.
func TestTupleItemsRefFollowsARename(t *testing.T) {
	schema := &parser.Schema{
		Type: "array",
		Items: []*parser.Schema{
			{Type: "string"},
			{Ref: "#/definitions/PetDetails"},
		},
	}

	r := NewSchemaRewriter()
	r.RegisterRename("PetDetails", "PetDetails2", parser.OASVersion20)
	r.rewriteSchema(schema)

	items, ok := schema.Items.([]*parser.Schema)
	require.True(t, ok)
	assert.Equal(t, "#/definitions/PetDetails2", items[1].Ref,
		"a $ref in a tuple element was left pointing at the pre-rename name")
}

// TestTupleItemsAreComparedElementwise covers schema equivalence, which decides
// whether two same-named schemas collide or are the same schema twice. Tuples
// were reported as a type mismatch and never actually compared, so two
// identical ones could not be recognized as identical.
func TestTupleItemsAreComparedElementwise(t *testing.T) {
	tuple := func(second *parser.Schema) *parser.Schema {
		return &parser.Schema{
			Type:  "array",
			Items: []*parser.Schema{{Type: "string"}, second},
		}
	}

	t.Run("identical tuples are equivalent", func(t *testing.T) {
		result := CompareSchemas(
			tuple(&parser.Schema{Ref: "#/definitions/PetDetails"}),
			tuple(&parser.Schema{Ref: "#/definitions/PetDetails"}),
			EquivalenceModeDeep,
		)
		assert.True(t, result.Equivalent,
			"two identical tuples were reported different: %v", result.Differences)
	})

	t.Run("a differing element is reported at its position", func(t *testing.T) {
		result := CompareSchemas(
			tuple(&parser.Schema{Ref: "#/definitions/PetDetails"}),
			tuple(&parser.Schema{Ref: "#/definitions/Other"}),
			EquivalenceModeDeep,
		)
		require.False(t, result.Equivalent, "tuples differing in an element were reported the same")
		require.NotEmpty(t, result.Differences)
		assert.Contains(t, result.Differences[0].Path, "items.[1]",
			"the difference should name the element it is in, the way allOf.[0] does")
	})

	t.Run("tuples of differing length are not equivalent", func(t *testing.T) {
		short := &parser.Schema{Type: "array", Items: []*parser.Schema{{Type: "string"}}}
		result := CompareSchemas(short, tuple(&parser.Schema{Type: "number"}), EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})
}
