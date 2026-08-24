// tuple_items_test.go covers joining against the OAS 2.0 tuple form of `items`.
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
// were reported as a type mismatch and never compared.
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

// TestTupleItemsRefsAreRecordedInTheRefGraph covers the graph the joiner builds
// to answer which schema points at which. A tuple element missing from it makes
// a referenced schema look unreferenced, the same way #502 did in the fixer.
func TestTupleItemsRefsAreRecordedInTheRefGraph(t *testing.T) {
	g := newRefGraph()
	g.recordSchemaRefs("PetTuple", &parser.Schema{
		Type: "array",
		Items: []*parser.Schema{
			{Ref: "#/definitions/First"},
			{Type: "string"},
			{Ref: "#/definitions/Third"},
		},
	})

	locationsFor := func(name string) []string {
		out := make([]string, 0, len(g.schemaRefs[name]))
		for _, ref := range g.schemaRefs[name] {
			assert.Equal(t, "PetTuple", ref.FromSchema)
			out = append(out, ref.RefLocation)
		}
		return out
	}

	assert.Equal(t, []string{"items[0]"}, locationsFor("First"),
		"a $ref in the first tuple element was not recorded at its position")
	assert.Equal(t, []string{"items[2]"}, locationsFor("Third"),
		"a $ref in a later tuple element was not recorded at its position")
}

// TestTupleItemsRefsAreRecordedForAdditionalItems covers the other schema-or-bool
// field the tuple form reaches through in an OAS 2.0 document.
func TestTupleItemsRefsAreRecordedForAdditionalItems(t *testing.T) {
	g := newRefGraph()
	g.recordSchemaRefs("PetTuple", &parser.Schema{
		Type:            "array",
		AdditionalItems: []*parser.Schema{{Ref: "#/definitions/Extra"}},
	})

	require.Len(t, g.schemaRefs["Extra"], 1)
	assert.Equal(t, "additionalItems[0]", g.schemaRefs["Extra"][0].RefLocation)
}

// TestNewlyTraversedFieldsAreRecordedInTheRefGraph covers the positions the ref
// graph gained alongside the tuple form. A ref the graph misses makes a
// referenced schema look unreferenced.
func TestNewlyTraversedFieldsAreRecordedInTheRefGraph(t *testing.T) {
	g := newRefGraph()
	g.recordSchemaRefs("Holder", &parser.Schema{
		AdditionalProperties:  []*parser.Schema{{Ref: "#/definitions/AddProps"}},
		AdditionalItems:       []*parser.Schema{{Ref: "#/definitions/AddItems"}},
		UnevaluatedProperties: &parser.Schema{Ref: "#/definitions/UnevProps"},
		UnevaluatedItems:      []*parser.Schema{{Ref: "#/definitions/UnevItems"}},
	})

	for name, wantLocation := range map[string]string{
		"AddProps":  "additionalProperties[0]",
		"AddItems":  "additionalItems[0]",
		"UnevProps": "unevaluatedProperties",
		"UnevItems": "unevaluatedItems[0]",
	} {
		refs := g.schemaRefs[name]
		require.Len(t, refs, 1, "%s was not recorded", name)
		assert.Equal(t, wantLocation, refs[0].RefLocation)
		assert.Equal(t, "Holder", refs[0].FromSchema)
	}
}

// TestNewlyTraversedFieldsFollowARename covers the same positions in the
// rewriter. A $ref it does not reach is left naming a schema that is gone.
func TestNewlyTraversedFieldsFollowARename(t *testing.T) {
	schema := &parser.Schema{
		AdditionalProperties:  []*parser.Schema{{Ref: "#/definitions/Old"}},
		AdditionalItems:       []*parser.Schema{{Ref: "#/definitions/Old"}},
		UnevaluatedProperties: &parser.Schema{Ref: "#/definitions/Old"},
		UnevaluatedItems:      []*parser.Schema{{Ref: "#/definitions/Old"}},
		ContentSchema:         &parser.Schema{Ref: "#/definitions/Old"},
	}

	r := NewSchemaRewriter()
	r.RegisterRename("Old", "New", parser.OASVersion20)
	r.rewriteSchema(schema)

	const want = "#/definitions/New"
	assert.Equal(t, want, schema.AdditionalProperties.([]*parser.Schema)[0].Ref, "additionalProperties")
	assert.Equal(t, want, schema.AdditionalItems.([]*parser.Schema)[0].Ref, "additionalItems")
	assert.Equal(t, want, schema.UnevaluatedProperties.(*parser.Schema).Ref, "unevaluatedProperties")
	assert.Equal(t, want, schema.UnevaluatedItems.([]*parser.Schema)[0].Ref, "unevaluatedItems")
	assert.Equal(t, want, schema.ContentSchema.Ref, "contentSchema")
}
