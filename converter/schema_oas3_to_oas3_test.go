// schema_oas3_to_oas3_test.go covers the per-schema passes on a conversion
// between two OAS 3.x versions, and the positions that conversion has to reach.
package converter

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func convertOAS3(t *testing.T, source, target string) (*ConversionResult, *parser.OAS3Document) {
	t.Helper()

	parsed, err := parser.New().ParseBytes([]byte(source))
	require.NoError(t, err)

	result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion(target))
	require.NoError(t, err)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected an OAS3Document, got %T", result.Document)
	return result, doc
}

// TestOAS31ToOAS30RunsTheSchemaPasses pins that a conversion between two OAS 3.x
// versions runs the per-schema passes, so no construct the target forbids
// survives in the output.
func TestOAS31ToOAS30RunsTheSchemaPasses(t *testing.T) {
	const source = `openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        "200":
          description: OK
components:
  schemas:
    Row:
      type: object
      additionalProperties:
        - type: string
    Tup:
      type: array
      prefixItems:
        - type: string
        - type: integer
    Bound:
      type: integer
      exclusiveMinimum: 3
`

	result, doc := convertOAS3(t, source, "3.0.3")

	row := doc.Components.Schemas["Row"]
	require.NotNil(t, row)
	assert.Nil(t, row.AdditionalProperties, "an array is not a legal additionalProperties in any dialect")
	assertIssueMentioning(t, result, "additionalProperties")

	tup := doc.Components.Schemas["Tup"]
	require.NotNil(t, tup)
	assert.Empty(t, tup.PrefixItems, "OAS 3.0 has no prefixItems")
	items, ok := tup.Items.(*parser.Schema)
	require.True(t, ok, "OAS 3.0 requires items to be a single schema, got %T", tup.Items)
	assert.NotNil(t, items)
	assertIssueMentioning(t, result, "tuple")

	bound := doc.Components.Schemas["Bound"]
	require.NotNil(t, bound)
	assert.Equal(t, true, bound.ExclusiveMinimum, "OAS 3.0 spells an exclusive bound as a boolean")
	require.NotNil(t, bound.Minimum)
	assert.InDelta(t, 3.0, *bound.Minimum, 0.0001)
}

// TestExclusiveBoundsPickTheBindingOne covers the case where a 2020-12 schema
// carries both an inclusive and an exclusive bound. Draft 4 has one value to put
// them in, and nothing is lost by that: one of the two always implies the other.
func TestExclusiveBoundsPickTheBindingOne(t *testing.T) {
	const source = `openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        "200":
          description: OK
components:
  schemas:
    ExclusiveIsTighter:
      type: integer
      maximum: 10
      exclusiveMaximum: 5
    InclusiveIsTighter:
      type: integer
      maximum: 3
      exclusiveMaximum: 8
`

	_, doc := convertOAS3(t, source, "3.0.3")

	// x <= 10 and x < 5 is x < 5.
	tighter := doc.Components.Schemas["ExclusiveIsTighter"]
	require.NotNil(t, tighter)
	require.NotNil(t, tighter.Maximum)
	assert.InDelta(t, 5.0, *tighter.Maximum, 0.0001)
	assert.Equal(t, true, tighter.ExclusiveMaximum)

	// x <= 3 and x < 8 is x <= 3, so the exclusive bound says nothing more.
	inclusive := doc.Components.Schemas["InclusiveIsTighter"]
	require.NotNil(t, inclusive)
	require.NotNil(t, inclusive.Maximum)
	assert.InDelta(t, 3.0, *inclusive.Maximum, 0.0001)
	assert.Nil(t, inclusive.ExclusiveMaximum)
}

func TestOAS30ToOAS31RunsTheSchemaPasses(t *testing.T) {
	const source = `openapi: 3.0.3
info:
  title: t
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        "200":
          description: OK
components:
  schemas:
    Bound:
      type: integer
      minimum: 3
      exclusiveMinimum: true
`

	_, doc := convertOAS3(t, source, "3.1.0")

	bound := doc.Components.Schemas["Bound"]
	require.NotNil(t, bound)
	assert.Nil(t, bound.Minimum, "the bound moves into exclusiveMinimum in 2020-12")
	assert.InDelta(t, 3.0, bound.ExclusiveMinimum, 0.0001)
}

// TestOAS3ToOAS3ReachesEverySchemaPosition pins the positional walk rather than
// the passes. Each of these positions holds the same illegal array, so a
// position the walk misses leaves one behind.
func TestOAS3ToOAS3ReachesEverySchemaPosition(t *testing.T) {
	const source = `openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      parameters:
        - name: q
          in: query
          schema:
            type: object
            additionalProperties:
              - type: string
      responses:
        "200":
          description: OK
          headers:
            X-H:
              schema:
                type: object
                additionalProperties:
                  - type: string
          content:
            application/json:
              schema:
                type: object
                additionalProperties:
                  - type: string
              itemSchema:
                type: object
                additionalProperties:
                  - type: string
              encoding:
                field:
                  headers:
                    X-E:
                      schema:
                        type: object
                        additionalProperties:
                          - type: string
      callbacks:
        onEvent:
          '{$request.body#/u}':
            post:
              operationId: cb
              requestBody:
                content:
                  application/json:
                    schema:
                      type: object
                      additionalProperties:
                        - type: string
              responses:
                "200":
                  description: OK
components:
  headers:
    Shared:
      schema:
        type: object
        additionalProperties:
          - type: string
  requestBodies:
    Body:
      content:
        application/json:
          schema:
            type: object
            additionalProperties:
              - type: string
`

	_, doc := convertOAS3(t, source, "3.0.3")

	// Navigated by hand rather than through forEachOAS3Schema. Checking the
	// walk with the walk is no check at all: a position deleted from it would
	// be skipped by the conversion and by the assertion together, and the test
	// would still pass.
	op := doc.Paths["/a"].Get
	require.NotNil(t, op)
	media := op.Responses.Codes["200"].Content["application/json"]
	require.NotNil(t, media)
	callback := op.Callbacks["onEvent"]
	require.NotNil(t, callback)
	callbackItem := (*callback)["{$request.body#/u}"]
	require.NotNil(t, callbackItem)

	for name, schema := range map[string]*parser.Schema{
		"operation parameter":     op.Parameters[0].Schema,
		"response header":         op.Responses.Codes["200"].Headers["X-H"].Schema,
		"response content":        media.Schema,
		"media type itemSchema":   media.ItemSchema,
		"encoding header":         media.Encoding["field"].Headers["X-E"].Schema,
		"callback request body":   callbackItem.Post.RequestBody.Content["application/json"].Schema,
		"components header":       doc.Components.Headers["Shared"].Schema,
		"components request body": doc.Components.RequestBodies["Body"].Content["application/json"].Schema,
	} {
		require.NotNil(t, schema, "the fixture should place a schema at the %s position", name)
		assert.Nil(t, schema.AdditionalProperties, "the %s position was not reached by the conversion", name)
	}
}
