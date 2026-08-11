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
//
// The assertion is about the $ref only. Where that $ref sits in the converted
// document is a separate question this test deliberately does not settle: OAS
// 3.0 says of items "Value MUST be an object and not an array", so the tuple
// reaching the output at all is its own defect, tracked apart from #502.
// Reading the elements by index here records current behavior so the ref can be
// checked, not that the shape is right.
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

	refs := collectRefsUnderItems(t, tuple)
	assert.Contains(t, refs, "#/components/schemas/PetDetails",
		"a $ref reachable through items kept the OAS 2.0 prefix and now points at nothing")
	assert.NotContains(t, refs, "#/definitions/PetDetails")
}

// collectRefsUnderItems gathers the $ref values reachable through a schema's
// items, whichever shape the field holds. Written shape-agnostically so it keeps
// checking the reference once the tuple's own conversion is settled.
func collectRefsUnderItems(t *testing.T, schema *parser.Schema) []string {
	t.Helper()

	var refs []string
	switch items := schema.Items.(type) {
	case *parser.Schema:
		refs = append(refs, items.Ref)
	case []*parser.Schema:
		for _, s := range items {
			refs = append(refs, s.Ref)
		}
	default:
		t.Fatalf("items held no schema to check, got %T", schema.Items)
	}

	for _, s := range schema.PrefixItems {
		refs = append(refs, s.Ref)
	}

	return refs
}
