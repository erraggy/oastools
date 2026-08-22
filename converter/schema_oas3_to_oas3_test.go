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

// TestExclusiveBoundConversionAcceptsTheSameValues proves the claim the
// conversion rests on: 2020-12 states an inclusive and an exclusive bound as two
// independent keywords, draft 4 has one number and a flag, and the rewrite is
// exact for every combination rather than merely close.
//
// It compares the two dialects by what they ACCEPT. For each pair of bounds it
// evaluates the 2020-12 reading of the source and the draft 4 reading of the
// converted schema over a range of values, and requires them to agree
// everywhere. A conversion that lost a constraint would accept a value the
// source rejected, and one that invented a constraint would reject a value the
// source accepted.
func TestExclusiveBoundConversionAcceptsTheSameValues(t *testing.T) {
	ptr := func(f float64) *float64 { return &f }

	for _, tc := range []struct {
		name    string
		maximum *float64
		excl    any
		minimum *float64
		exclMin any
	}{
		{name: "exclusive alone", excl: 5.0},
		{name: "inclusive alone", maximum: ptr(5)},
		{name: "exclusive tighter", maximum: ptr(10), excl: 5.0},
		{name: "inclusive tighter", maximum: ptr(3), excl: 8.0},
		{name: "bounds equal", maximum: ptr(4), excl: 4.0},
		{name: "min exclusive alone", exclMin: 5.0},
		{name: "min exclusive tighter", minimum: ptr(1), exclMin: 5.0},
		{name: "min inclusive tighter", minimum: ptr(7), exclMin: 2.0},
		{name: "min bounds equal", minimum: ptr(4), exclMin: 4.0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema := &parser.Schema{
				Type:             "number",
				Maximum:          tc.maximum,
				ExclusiveMaximum: tc.excl,
				Minimum:          tc.minimum,
				ExclusiveMinimum: tc.exclMin,
			}
			// The 2020-12 reading, taken before the schema is rewritten.
			source := func(x float64) bool {
				if tc.maximum != nil && x > *tc.maximum {
					return false
				}
				if e, ok := tc.excl.(float64); ok && x >= e {
					return false
				}
				if tc.minimum != nil && x < *tc.minimum {
					return false
				}
				if e, ok := tc.exclMin.(float64); ok && x <= e {
					return false
				}
				return true
			}

			fixSchemaExclusiveMinMaxForOAS30(schema)

			// The draft 4 reading of the result, where the flag qualifies the
			// bound beside it rather than standing on its own.
			converted := func(x float64) bool {
				if schema.Maximum != nil {
					if excl, _ := schema.ExclusiveMaximum.(bool); excl {
						if x >= *schema.Maximum {
							return false
						}
					} else if x > *schema.Maximum {
						return false
					}
				}
				if schema.Minimum != nil {
					if excl, _ := schema.ExclusiveMinimum.(bool); excl {
						if x <= *schema.Minimum {
							return false
						}
					} else if x < *schema.Minimum {
						return false
					}
				}
				return true
			}

			// Quarter steps, so a boundary is landed on exactly and straddled.
			for step := -40; step <= 40; step++ {
				x := float64(step) / 4
				assert.Equal(t, source(x), converted(x), "the two dialects disagree at x=%v", x)
			}

			// A numeric exclusive bound must not survive: 2020-12 spells it as a
			// number and draft 4 only as a bool.
			_, numericMax := schema.ExclusiveMaximum.(float64)
			_, numericMin := schema.ExclusiveMinimum.(float64)
			assert.False(t, numericMax, "a numeric exclusiveMaximum is not draft 4")
			assert.False(t, numericMin, "a numeric exclusiveMinimum is not draft 4")
		})
	}
}

// TestTupleDiagnosticsNameTheSourceSpelling pins that a diagnostic quotes the
// keywords the author wrote. A 3.1 document spells a tuple with prefixItems and
// constrains the rest with items; an OAS 2.0 one uses the draft 4 pair. Both
// reach tupleForOAS30, the second by way of prefixItemsToTuple, so without the
// source spelling the 3.1 case is told about a keyword it never used.
func TestTupleDiagnosticsNameTheSourceSpelling(t *testing.T) {
	const from31 = `openapi: 3.1.0
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
    Tup:
      type: array
      prefixItems:
        - type: string
        - type: integer
      items:
        type: boolean
`

	parsed, err := parser.New().ParseBytes([]byte(from31))
	require.NoError(t, err)
	result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("3.0.3"))
	require.NoError(t, err)

	var messages string
	for _, issue := range result.Issues {
		messages += issue.Message + "\n"
	}
	assert.Contains(t, messages, "'prefixItems'", "the tuple was written as prefixItems")
	assert.NotContains(t, messages, "additionalItems", "the source never used additionalItems")

	// The counterpart: a draft 4 source must still be told about the keywords it
	// did use, so the fix cannot be to stop naming additionalItems at all.
	const from20 = `swagger: "2.0"
info:
  title: t
  version: "1.0.0"
paths: {}
definitions:
  Tup:
    type: array
    items:
      - type: string
      - type: integer
    additionalItems:
      type: boolean
`

	parsed2, err := parser.New().ParseBytes([]byte(from20))
	require.NoError(t, err)
	result2, err := ConvertWithOptions(WithParsed(*parsed2), WithTargetVersion("3.0.3"))
	require.NoError(t, err)

	var messages2 string
	for _, issue := range result2.Issues {
		messages2 += issue.Message + "\n"
	}
	assert.Contains(t, messages2, "additionalItems")
	assert.NotContains(t, messages2, "'prefixItems'")
}
