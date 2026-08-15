// array_valued_guard_test.go guards a defect this package hit three times in
// one change: an array left in a schema-or-bool field that the target version
// forbids. Each instance passed the unit tests, the corpus differential and
// `validate`, because nothing asserted the property directly. This file asserts
// it over whole converted documents rather than one field at a time.
package converter

import (
	"fmt"
	"strings"
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
    post:
      operationId: addThing
      parameters:
        - name: body
          in: body
          schema:
            type: object
            additionalProperties:
              - type: string
              - type: integer
      responses:
        "200":
          description: OK
          schema:
            type: object
            unevaluatedProperties:
              - type: string
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
		fieldItems:                 s.Items,
		fieldAdditionalItems:       s.AdditionalItems,
		fieldAdditionalProperties:  s.AdditionalProperties,
		fieldUnevaluatedItems:      s.UnevaluatedItems,
		fieldUnevaluatedProperties: s.UnevaluatedProperties,
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
			legal := field == fieldItems && target == parser.OASVersion20
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

// clearedField names one place a fixture leaves a malformed value: the issue
// path it is reported under, and the field within it.
//
// kind is a fragment of the message, because a field name alone does not say
// which report fired. Two different diagnostics can name `additionalItems` at
// one path, and matching only the name lets either stand in for the other, so a
// branch can be removed with the guard still green.
type clearedField struct{ path, field, kind string }

// arrayKind is the fragment shared by every malformed-array diagnostic. It is
// what separates them from the other reports that name the same fields.
const arrayKind = "no OAS version accepts there"

// assertClearedFieldsReported fails unless each named occurrence produced
// exactly one warning. Asserting the output is clean is not enough on its own,
// because a conversion that dropped the value silently would satisfy that, and
// silent loss is the failure this package keeps producing.
//
// Occurrences are matched by path as well as field. The fixtures plant the same
// field in a component schema and in an inline one, so a check that only
// counted field names would accept a conversion that reported the first and
// dropped the second without a word. Exactly one, rather than at least one, so
// a doubled report fails too.
//
// The path is matched exactly, or as a prefix ending at a dot, rather than as a
// substring: `components.schemas.Nested` is a substring of
// `components.schemas.NestedDeep`, so a loose match would count another
// component's report as this one's.
//
// One limit worth knowing. Reports carry the path of the schema the walk
// started from, so two cleared values under one component are indistinguishable
// here and would read as a doubled report. Threading a path per subschema is
// #521; until then, keep one cleared value per component in these fixtures.
func assertClearedFieldsReported(t *testing.T, result *ConversionResult, want ...clearedField) {
	t.Helper()

	var missed bool
	for _, w := range want {
		var found int
		for _, issue := range result.Issues {
			if issue.Severity == SeverityWarning &&
				(issue.Path == w.path || strings.HasPrefix(issue.Path, w.path+".")) &&
				strings.Contains(issue.Message, "'"+w.field+"'") &&
				strings.Contains(issue.Message, w.kind) {
				found++
			}
		}
		if !assert.Equal(t, 1, found, "expected exactly one warning for %s at %s", w.field, w.path) {
			missed = true
		}
	}
	if missed {
		// Once, rather than repeated into every failing assertion.
		t.Logf("conversion issues were: %+v", result.Issues)
	}
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

			// Inline schemas reach the conversion through their own call sites,
			// so component roots alone would not prove they were cleaned.
			for name, schema := range inlineSchemasOAS3(t, doc) {
				assertNoIllegalArrays(t, name, schema, target.enum)
			}

			// Every malformed field the fixture plants must be reported, not
			// merely removed.
			assertClearedFieldsReported(t, result,
				clearedField{"components.schemas.ArrayInAdditionalProperties", "additionalProperties", arrayKind},
				clearedField{"components.schemas.ArrayInUnevaluated", "unevaluatedItems", arrayKind},
				clearedField{"components.schemas.ArrayInUnevaluated", "unevaluatedProperties", arrayKind},
				clearedField{"components.schemas.Nested", "additionalProperties", arrayKind},
				clearedField{"components.schemas.TupleWithTrailing", "additionalItems", arrayKind},
				// the two inline schemas, which reach the conversion by their own
				// call sites and report under paths of their own
				clearedField{"requestBody", "additionalProperties", arrayKind},
				clearedField{"paths./things.post.responses.200.schema", "unevaluatedProperties", arrayKind})

			// The counterpart: the guard must not pass by dropping everything.
			// The legal tuple has to arrive, spelled the way the target spells it.
			tuple := doc.Components.Schemas["Tuple"]
			require.NotNil(t, tuple, "the Tuple definition should survive conversion")
			if target.enum >= parser.OASVersion310 {
				assert.Len(t, tuple.PrefixItems, 2, "3.1 and later hold the positions in prefixItems")
			} else {
				assert.NotNil(t, tuple.Items, "OAS 3.0 keeps items present, even when the positions go")
			}

			// The interaction this change has to preserve: legal tuple positions
			// beside a malformed additionalItems. Removing the one must not take
			// the other with it.
			trailing := doc.Components.Schemas["TupleWithTrailing"]
			require.NotNil(t, trailing)
			assert.Nil(t, trailing.AdditionalItems, "the malformed array goes")
			if target.enum >= parser.OASVersion310 {
				assert.Len(t, trailing.PrefixItems, 1, "and the tuple position stays")
			} else {
				assert.NotNil(t, trailing.Items, "and items stays present for OAS 3.0")
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
paths:
  /things:
    post:
      operationId: addThing
      requestBody:
        content:
          application/json:
            schema:
              type: object
              additionalProperties:
                - type: string
                - type: integer
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
                unevaluatedProperties:
                  - type: string
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
    BareAdditionalItems:
      type: array
      items:
        - type: string
      additionalItems:
        - type: integer
        - type: boolean
    BothPresent:
      type: array
      prefixItems:
        - type: string
      items:
        type: number
      additionalItems:
        - type: integer
        - type: boolean
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
	for name, schema := range inlineSchemasOAS2(t, doc) {
		assertNoIllegalArrays(t, name, schema, parser.OASVersion20)
	}

	// The counterpart: the guard must not pass by dropping everything. The
	// legal tuple has to arrive.
	assertClearedFieldsReported(t, result,
		clearedField{"components.schemas.ArrayInAdditionalProperties", "additionalProperties", arrayKind},
		clearedField{"components.schemas.ArrayInUnevaluated", "unevaluatedItems", arrayKind},
		clearedField{"components.schemas.ArrayInUnevaluated", "unevaluatedProperties", arrayKind},
		clearedField{"components.schemas.Nested", "additionalProperties", arrayKind},
		clearedField{"paths./things.post.requestBody.content.application/json.schema", "additionalProperties", arrayKind},
		clearedField{"paths./things.post.responses.200.content.application/json.schema", "unevaluatedProperties", arrayKind},
		// no prefixItems, so the conversion takes its early return: the array in
		// items becomes the OAS 2.0 tuple and the one in additionalItems, which
		// draft 4 never accepts, is dropped and reported
		clearedField{"components.schemas.BareAdditionalItems", "additionalItems", arrayKind},
		// prefixItems present, so items becomes additionalItems: the array
		// already sitting there is discarded and must not go quietly
		clearedField{"components.schemas.BothPresent", "additionalItems", arrayKind})

	tupleSchema := doc.Definitions["Tuple"]
	require.NotNil(t, tupleSchema, "the Tuple definition should survive conversion")
	tuple, ok := tupleSchema.Items.([]*parser.Schema)
	require.True(t, ok, "the prefixItems tuple should convert, got %T", tupleSchema.Items)
	assert.Len(t, tuple, 2)
}

// inlineSchemasOAS3 gathers the schemas that live in operations rather than in
// components, which reach the conversion through their own call sites.
func inlineSchemasOAS3(t *testing.T, doc *parser.OAS3Document) map[string]*parser.Schema {
	t.Helper()

	found := make(map[string]*parser.Schema)
	for path, item := range doc.Paths {
		op := item.Post
		if op == nil {
			continue
		}
		if op.RequestBody != nil {
			for mt, media := range op.RequestBody.Content {
				if media.Schema != nil {
					found[path+".post.requestBody."+mt] = media.Schema
				}
			}
		}
		if op.Responses != nil {
			for code, resp := range op.Responses.Codes {
				for mt, media := range resp.Content {
					if media.Schema != nil {
						found[path+".post.responses."+code+"."+mt] = media.Schema
					}
				}
			}
		}
	}
	require.NotEmpty(t, found, "the fixture should carry inline schemas; it does not, so this guard proves nothing")
	return found
}

// inlineSchemasOAS2 is the same for an OAS 2.0 document, where a body parameter
// and a response each carry a schema directly.
func inlineSchemasOAS2(t *testing.T, doc *parser.OAS2Document) map[string]*parser.Schema {
	t.Helper()

	found := make(map[string]*parser.Schema)
	for path, item := range doc.Paths {
		op := item.Post
		if op == nil {
			continue
		}
		for i, param := range op.Parameters {
			if param.Schema != nil {
				found[fmt.Sprintf("%s.post.parameters[%d].schema", path, i)] = param.Schema
			}
		}
		if op.Responses != nil {
			for code, resp := range op.Responses.Codes {
				if resp.Schema != nil {
					found[path+".post.responses."+code+".schema"] = resp.Schema
				}
			}
		}
	}
	require.NotEmpty(t, found, "the fixture should carry inline schemas; it does not, so this guard proves nothing")
	return found
}
