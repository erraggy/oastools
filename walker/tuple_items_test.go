// tuple_items_test.go covers walking the OAS 2.0 tuple form of `items`, where
// the field holds a list of schemas rather than one.
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
