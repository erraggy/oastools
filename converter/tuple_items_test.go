// tuple_items_test.go covers conversion of the OAS 2.0 tuple form of `items`.
package converter

import (
	"strings"
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
// It asserts the $ref only. That the tuple reaches OAS 3 output at all is a
// separate defect, since OAS 3.0 says of items "Value MUST be an object and not
// an array".
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
// items, whichever shape the field holds, so it keeps checking the reference
// once the tuple's own conversion is settled.
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

// TestSchemaOrBoolFieldsAreConvertedForOAS31 covers the fields the OAS 3.1
// exclusiveMinimum/exclusiveMaximum conversion gained alongside the tuple form.
// OAS 2.0 spells those keywords as booleans paired with minimum/maximum, 3.1 as
// numbers, so a field the conversion skips keeps the old spelling and is invalid
// in the converted document.
func TestSchemaOrBoolFieldsAreConvertedForOAS31(t *testing.T) {
	const spec = `
swagger: "2.0"
info:
  title: t
  version: "1.0.0"
paths: {}
definitions:
  Holder:
    type: array
    items:
      - type: number
        minimum: 5
        exclusiveMinimum: true
    additionalItems:
      type: number
      minimum: 5
      exclusiveMinimum: true
    additionalProperties:
      type: number
      minimum: 5
      exclusiveMinimum: true
    unevaluatedItems:
      type: number
      minimum: 5
      exclusiveMinimum: true
    unevaluatedProperties:
      type: number
      minimum: 5
      exclusiveMinimum: true
`
	parsed, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)
	parseResult := *parsed

	result, err := ConvertWithOptions(WithParsed(parseResult), WithTargetVersion("3.1.0"))
	require.NoError(t, err)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok)
	converted := doc.Components.Schemas["Holder"]

	check := func(name string, field any) {
		t.Helper()
		var s *parser.Schema
		switch v := field.(type) {
		case *parser.Schema:
			s = v
		case []*parser.Schema:
			require.NotEmpty(t, v, name)
			s = v[0]
		default:
			t.Fatalf("%s held no schema, got %T", name, field)
		}
		assert.Equal(t, 5.0, s.ExclusiveMinimum,
			"%s kept the OAS 3.0 boolean spelling instead of the 3.1 numeric one", name)
		assert.Nil(t, s.Minimum, "%s should have moved minimum into exclusiveMinimum", name)
	}

	check("items", converted.Items)
	check("additionalItems", converted.AdditionalItems)
	check("additionalProperties", converted.AdditionalProperties)
	check("unevaluatedItems", converted.UnevaluatedItems)
	check("unevaluatedProperties", converted.UnevaluatedProperties)
}

// TestSchemaOrBoolFieldsAreWalkedOnDowngrade covers the fields the feature walk
// gained alongside the tuple form. The walk reports OAS 3.x features that cannot
// be expressed in OAS 2.0, so a field it skips downgrades silently.
func TestSchemaOrBoolFieldsAreWalkedOnDowngrade(t *testing.T) {
	const spec = `
openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths: {}
components:
  schemas:
    Holder:
      type: array
      additionalItems:
        type: object
        deprecated: true
      unevaluatedItems:
        type: object
        deprecated: true
      unevaluatedProperties:
        type: object
        deprecated: true
`
	parsed, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("2.0"))
	require.NoError(t, err)

	paths := make([]string, 0, len(result.Issues))
	for _, issue := range result.Issues {
		paths = append(paths, issue.Path)
	}
	joined := strings.Join(paths, "\n")

	for _, field := range []string{"additionalItems", "unevaluatedItems", "unevaluatedProperties"} {
		assert.Contains(t, joined, "."+field,
			"the deprecated schema under %s was not reached; issue paths:\n%s", field, joined)
	}
}

// TestCheckSchemaNullableReachesTupleAdditionalProperties covers the tuple form
// in the nullable check, which reports the OAS 3.1 deprecation of 'nullable'.
func TestCheckSchemaNullableReachesTupleAdditionalProperties(t *testing.T) {
	const spec = `
openapi: 3.0.3
info:
  title: t
  version: "1.0.0"
paths: {}
components:
  schemas:
    Holder:
      type: object
      additionalProperties:
        - type: string
          nullable: true
`
	parsed, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("3.1.0"))
	require.NoError(t, err)

	paths := make([]string, 0, len(result.Issues))
	for _, issue := range result.Issues {
		paths = append(paths, issue.Path)
	}
	assert.Contains(t, strings.Join(paths, "\n"), "additionalProperties[0]",
		"the nullable schema in a tuple additionalProperties element was not reached")
}
