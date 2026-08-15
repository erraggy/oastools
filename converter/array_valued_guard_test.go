// array_valued_guard_test.go guards a defect this package hit three times in
// one change: an array left in a schema-or-bool field that the target version
// forbids. Each instance passed the unit tests, the corpus differential and
// `validate`, because nothing asserted the property directly. This file asserts
// it over whole converted documents rather than one field at a time.
package converter

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// arrayValuedSourceOAS2 puts an array in every schema-or-bool field, nested and
// top level. Only `items` is legal that way in OAS 2.0; the rest are malformed
// but parseable, which is exactly the shape that produced the defects.
const arrayValuedSourceOAS2 = `swagger: "2.0"
info:
  title: rakes
  version: "1.0.0"
paths:
  /things:
    get:
      operationId: listThings
      responses:
        "200":
          description: OK
          schema:
            $ref: "#/definitions/Nested"
definitions:
  Tuple:
    type: array
    items:
      - type: string
      - type: integer
  TupleWithTrailing:
    type: array
    items:
      - type: string
    additionalItems:
      - type: integer
      - type: boolean
  ArrayInAdditionalProperties:
    type: object
    additionalProperties:
      - type: string
      - type: integer
  ArrayInUnevaluated:
    type: object
    unevaluatedProperties:
      - type: string
    unevaluatedItems:
      - type: integer
  Nested:
    type: object
    properties:
      inner:
        type: array
        items:
          - type: string
          - type: integer
    allOf:
      - type: object
        additionalProperties:
          - type: string
`

// schemaOrBoolFields is every field parser models as *Schema, []*Schema or bool.
// Keep this list in step with the one CLAUDE.md names; a field missing here is a
// field this guard does not watch.
func schemaOrBoolFields(s *parser.Schema) map[string]any {
	return map[string]any{
		"items":                 s.Items,
		"additionalItems":       s.AdditionalItems,
		"additionalProperties":  s.AdditionalProperties,
		"unevaluatedItems":      s.UnevaluatedItems,
		"unevaluatedProperties": s.UnevaluatedProperties,
	}
}

// assertNoIllegalArrays fails for each schema-or-bool field holding an array
// where the target version forbids one. OAS 2.0 spells a tuple as an array in
// `items`, and that is the only position any version allows: 3.0 forbids arrays
// there outright, and 3.1 and later put the positions in prefixItems, which is
// a typed slice rather than one of these fields.
//
// It walks with walkSchemas, the same traversal the conversions use, rather
// than a second one written for the test. A guard that visits fewer positions
// than the code it guards is the defect it exists to catch, and the first
// version of this function had exactly that gap: it never reached Contains,
// PropertyNames, DependentSchemas, If, Then, Else, ContentSchema or Defs.
//
// The trade is that a report names the component rather than the field inside
// it. Locating it is #521's subject; detecting it is this function's.
func assertNoIllegalArrays(t *testing.T, name string, root *parser.Schema, target parser.OASVersion) {
	t.Helper()

	walkSchemas(root, func(s *parser.Schema) {
		for field, value := range schemaOrBoolFields(s) {
			arr, isArray := value.([]*parser.Schema)
			if !isArray {
				continue
			}
			legal := field == "items" && target == parser.OASVersion20
			assert.True(t, legal,
				"%s: %s holds a %d element array, which OAS %s does not accept there",
				name, field, len(arr), target)
		}

		if target < parser.OASVersion310 && len(s.PrefixItems) > 0 {
			assert.Fail(t, "prefixItems before OAS 3.1",
				"%s: prefixItems is a JSON Schema 2020-12 keyword and has no place in OAS %s", name, target)
		}
	})
}

func TestConvertedOutputHoldsNoIllegalArrays(t *testing.T) {
	for _, target := range []struct {
		spec string
		enum parser.OASVersion
	}{
		{"3.0.3", parser.OASVersion303},
		{"3.1.0", parser.OASVersion310},
		{"3.2.0", parser.OASVersion320},
	} {
		t.Run(target.spec, func(t *testing.T) {
			parsed, err := parser.New().ParseBytes([]byte(arrayValuedSourceOAS2))
			require.NoError(t, err)

			result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion(target.spec))
			require.NoError(t, err)

			doc, ok := result.Document.(*parser.OAS3Document)
			require.True(t, ok)
			require.NotNil(t, doc.Components)

			for name, schema := range doc.Components.Schemas {
				assertNoIllegalArrays(t, "components.schemas."+name, schema, target.enum)
			}

			// The counterpart: the guard must not pass by dropping everything.
			// The legal tuple has to arrive, spelled the way the target spells it.
			tuple := doc.Components.Schemas["Tuple"]
			require.NotNil(t, tuple, "the Tuple definition should survive conversion")
			if target.enum >= parser.OASVersion310 {
				assert.Len(t, tuple.PrefixItems, 2, "3.1 and later hold the positions in prefixItems")
			} else {
				assert.NotNil(t, tuple.Items, "OAS 3.0 keeps items present, even when the positions go")
			}
		})
	}
}

// arrayValuedSourceOAS31 is the same shape from the other side. The positions
// live in prefixItems, which is how 2020-12 spells a tuple, so the guard expects
// them to arrive as an OAS 2.0 array-form items while the malformed arrays in
// the other fields are dropped.
const arrayValuedSourceOAS31 = `openapi: 3.1.0
info:
  title: rakes
  version: "1.0.0"
paths: {}
components:
  schemas:
    Tuple:
      type: array
      prefixItems:
        - type: string
        - type: integer
    ArrayInAdditionalProperties:
      type: object
      additionalProperties:
        - type: string
        - type: integer
    ArrayInUnevaluated:
      type: object
      unevaluatedProperties:
        - type: string
      unevaluatedItems:
        - type: integer
    Nested:
      type: object
      properties:
        inner:
          type: object
          additionalProperties:
            - type: string
`

func TestDownconvertedOutputHoldsNoIllegalArrays(t *testing.T) {
	parsed, err := parser.New().ParseBytes([]byte(arrayValuedSourceOAS31))
	require.NoError(t, err)

	result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("2.0"))
	require.NoError(t, err)

	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)

	for name, schema := range doc.Definitions {
		assertNoIllegalArrays(t, "definitions."+name, schema, parser.OASVersion20)
	}

	// The counterpart: the guard must not pass by dropping everything. The
	// legal tuple has to arrive.
	tupleSchema := doc.Definitions["Tuple"]
	require.NotNil(t, tupleSchema, "the Tuple definition should survive conversion")
	tuple, ok := tupleSchema.Items.([]*parser.Schema)
	require.True(t, ok, "the prefixItems tuple should convert, got %T", tupleSchema.Items)
	assert.Len(t, tuple, 2)
}
