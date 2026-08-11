// tuple_items_test.go covers conversion of the OAS 2.0 tuple form of `items`,
// where the field holds a list of schemas rather than one.
package converter

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tupleItemsOAS2Spec = `
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
`

// TestTupleItemsRefIsRewrittenOnUpconversion covers #502 from the converter's
// side: OAS 2.0 spells the pool "#/definitions", OAS 3.x spells it
// "#/components/schemas", and a $ref the rewrite never visits keeps the old
// prefix and points at nothing in the converted document.
func TestTupleItemsRefIsRewrittenOnUpconversion(t *testing.T) {
	parseResult, err := parser.New().ParseBytes([]byte(tupleItemsOAS2Spec))
	require.NoError(t, err)

	result, err := ConvertWithOptions(
		WithParsed(*parseResult),
		WithTargetVersion("3.0.3"),
	)
	require.NoError(t, err)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected an OAS 3 document, got %T", result.Document)

	tuple := doc.Components.Schemas["PetTuple"]
	require.NotNil(t, tuple)

	items, ok := tuple.Items.([]*parser.Schema)
	require.True(t, ok, "the tuple form should survive conversion, got %T", tuple.Items)
	require.Len(t, items, 2)

	assert.Equal(t, "#/components/schemas/PetDetails", items[1].Ref,
		"a $ref in a tuple element kept the OAS 2.0 prefix and now points at nothing")
}
