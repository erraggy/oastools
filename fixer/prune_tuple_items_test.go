// prune_tuple_items_test.go covers pruning against the OAS 2.0 tuple form of
// `items`.
package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tupleItemsPruneSpec = `
swagger: "2.0"
info:
  title: petstore
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: successful operation
          schema:
            $ref: "#/definitions/PetTuple"
definitions:
  PetTuple:
    type: array
    items:
      - type: string
      - $ref: "#/definitions/PetDetails"
  PetDetails:
    type: object
    properties:
      name:
        type: string
  Orphan:
    type: object
`

// TestPruneKeepsSchemaReferencedFromTupleItems covers #502: a schema reachable
// only from an element of a tuple-form `items` was reported unreferenced and
// removed, leaving the $ref to it pointing at nothing.
func TestPruneKeepsSchemaReferencedFromTupleItems(t *testing.T) {
	parseResult, err := parser.New().ParseBytes([]byte(tupleItemsPruneSpec))
	require.NoError(t, err)
	doc, ok := parseResult.OAS2Document()
	require.True(t, ok)

	items, ok := doc.Definitions["PetTuple"].Items.([]*parser.Schema)
	require.True(t, ok, "the tuple form should decode to []*parser.Schema, got %T",
		doc.Definitions["PetTuple"].Items)
	require.Len(t, items, 2)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypePrunedUnusedSchema),
		WithMutableInput(true),
	)
	require.NoError(t, err)

	assert.Contains(t, doc.Definitions, "PetDetails",
		"a schema referenced from a tuple element was pruned")
	assert.NotContains(t, doc.Definitions, "Orphan",
		"a genuinely unreferenced schema should still be pruned")

	var pruned []string
	for _, fix := range result.Fixes {
		if fix.Type == "pruned-unused-schema" {
			pruned = append(pruned, fix.Path)
		}
	}
	assert.Equal(t, []string{"definitions.Orphan"}, pruned)
}

// TestCollectSchemaRefs_TupleItems covers ref collection for each shape a tuple
// can take.
func TestCollectSchemaRefs_TupleItems(t *testing.T) {
	t.Run("ref in a tuple element is collected", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "array",
			Items: []*parser.Schema{
				{Type: "string"},
				{Ref: "#/definitions/PetDetails"},
			},
		}

		assert.Contains(t, collectSchemaRefs(schema, "#/definitions/"), "PetDetails")
	})

	t.Run("refs nested inside a tuple element are collected", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "array",
			Items: []*parser.Schema{
				{
					Type:       "object",
					Properties: map[string]*parser.Schema{"owner": {Ref: "#/definitions/Owner"}},
				},
			},
		}

		assert.Contains(t, collectSchemaRefs(schema, "#/definitions/"), "Owner")
	})

	t.Run("a nil tuple element is skipped", func(t *testing.T) {
		schema := &parser.Schema{
			Type:  "array",
			Items: []*parser.Schema{nil, {Ref: "#/definitions/PetDetails"}},
		}

		assert.NotPanics(t, func() {
			assert.Contains(t, collectSchemaRefs(schema, "#/definitions/"), "PetDetails")
		})
	})

	t.Run("tuple form is also honored for additionalItems", func(t *testing.T) {
		schema := &parser.Schema{
			Type:            "array",
			AdditionalItems: []*parser.Schema{{Ref: "#/definitions/Extra"}},
		}

		assert.Contains(t, collectSchemaRefs(schema, "#/definitions/"), "Extra")
	})
}

// TestRewriteSchemaRefs_TupleItems covers the rename path: a $ref in a tuple
// element has to move with the schema it points at, or it is left dangling.
func TestRewriteSchemaRefs_TupleItems(t *testing.T) {
	schema := &parser.Schema{
		Type: "array",
		Items: []*parser.Schema{
			{Type: "string"},
			{Ref: "#/definitions/Pet[Details]"},
		},
	}

	renames := map[string]string{"#/definitions/Pet[Details]": "#/definitions/PetDetails"}
	rewriteSchemaRefs(schema, renames)

	items, ok := schema.Items.([]*parser.Schema)
	require.True(t, ok)
	assert.Equal(t, "#/definitions/PetDetails", items[1].Ref,
		"a $ref in a tuple element was left pointing at the old name")
}

// TestRewriteSchemaRefs_ContentSchema covers the contentSchema position the
// rename rewrite gained. A $ref it does not reach is left naming a schema that
// no longer exists.
func TestRewriteSchemaRefs_ContentSchema(t *testing.T) {
	schema := &parser.Schema{
		Type:          "string",
		ContentSchema: &parser.Schema{Ref: "#/definitions/Old"},
	}

	rewriteSchemaRefs(schema, map[string]string{"#/definitions/Old": "#/definitions/New"})

	assert.Equal(t, "#/definitions/New", schema.ContentSchema.Ref,
		"a $ref in contentSchema was left pointing at the old name")
}

// TestCollectSchemaOrBoolRefs_RawMapForm covers the shape decoding can leave
// untyped, which SchemaOrBoolSchemas does not yield and the collector handles
// separately.
func TestCollectSchemaOrBoolRefs_RawMapForm(t *testing.T) {
	c := NewRefCollector()
	c.collectSchemaOrBoolRefs(map[string]any{"$ref": "#/components/schemas/FromMap"}, "schema.items")

	assert.True(t, c.IsSchemaReferenced("FromMap", parser.OASVersion303),
		"a $ref in the raw map form of a schema-or-bool field was not collected")
}
