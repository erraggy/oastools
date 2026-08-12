// tuple_items_test.go covers walking the OAS 2.0 tuple form of `items`.
package walker

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tupleItemsDocument(t *testing.T) *parser.ParseResult {
	t.Helper()
	const spec = `
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
	parseResult, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)
	return parseResult
}

// TestTupleItemsRefsReachTheRefHandler covers #502 from the walker's side:
// anything collecting references reads them from [RefHandler], so a $ref held
// in a tuple element that never reaches it is a reference nothing accounts for.
func TestTupleItemsRefsReachTheRefHandler(t *testing.T) {
	parseResult := tupleItemsDocument(t)

	var paths []string
	err := Walk(parseResult,
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			if ref.Ref == "#/definitions/PetDetails" {
				paths = append(paths, ref.SourcePath)
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	require.Len(t, paths, 1, "the $ref in a tuple element was not reported")
	assert.Contains(t, paths[0], "items[1]",
		"a tuple element should be addressed by position, not as the field itself")
}

// TestSchemaOrBoolFieldsReachTheRefHandler covers every field walkSchemaOrBool
// serves, in both the single and tuple shapes. A ref that does not reach
// [RefHandler] is one nothing collecting references accounts for.
func TestSchemaOrBoolFieldsReachTheRefHandler(t *testing.T) {
	result := &parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.1.0",
			Info:    &parser.Info{Title: "t", Version: "1.0.0"},
			Components: &parser.Components{
				Schemas: map[string]*parser.Schema{
					"Holder": {
						Items:                 []*parser.Schema{{Ref: "#/components/schemas/A"}},
						AdditionalItems:       &parser.Schema{Ref: "#/components/schemas/B"},
						AdditionalProperties:  []*parser.Schema{{Ref: "#/components/schemas/C"}},
						UnevaluatedItems:      &parser.Schema{Ref: "#/components/schemas/D"},
						UnevaluatedProperties: []*parser.Schema{{Ref: "#/components/schemas/E"}},
					},
				},
			},
		},
	}

	seen := map[string]string{}
	err := Walk(result, WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
		seen[ref.Ref] = ref.SourcePath
		return Continue
	}))
	require.NoError(t, err)

	for ref, wantPathFragment := range map[string]string{
		"#/components/schemas/A": "items[0]",
		"#/components/schemas/B": "additionalItems",
		"#/components/schemas/C": "additionalProperties[0]",
		"#/components/schemas/D": "unevaluatedItems",
		"#/components/schemas/E": "unevaluatedProperties[0]",
	} {
		path, ok := seen[ref]
		require.True(t, ok, "%s never reached the ref handler", ref)
		assert.Contains(t, path, wantPathFragment)
	}
}

// TestWalkSchemaOrBoolHonorsStop covers the Stop action across the several
// schema-or-bool fields the caller walks in a row. A handler that stops during
// one field must not be reached again through the next.
func TestWalkSchemaOrBoolHonorsStop(t *testing.T) {
	rawRef := func(name string) map[string]any {
		return map[string]any{"$ref": "#/components/schemas/" + name}
	}

	result := &parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.1.0",
			Info:    &parser.Info{Title: "t", Version: "1.0.0"},
			Components: &parser.Components{
				Schemas: map[string]*parser.Schema{
					"Holder": {
						Items:                 rawRef("A"),
						AdditionalItems:       rawRef("B"),
						AdditionalProperties:  rawRef("C"),
						UnevaluatedItems:      rawRef("D"),
						UnevaluatedProperties: rawRef("E"),
					},
				},
			},
		},
	}

	var seen []string
	err := Walk(result,
		WithMapRefTracking(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			seen = append(seen, ref.Ref)
			return Stop
		}),
	)
	require.NoError(t, err)

	assert.Len(t, seen, 1,
		"the handler stopped on the first ref but kept being called: %v", seen)
}
