// media_type_selection_test.go covers the choice an OAS 2.0 target has to make
// when the source offers several media types and the target admits one schema.
package converter

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const multiMediaTypeSource = `openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths:
  /a:
    post:
      operationId: a
      requestBody:
        content:
          application/xml:
            schema: {type: object, title: xmlBody}
          application/json:
            schema: {type: object, title: jsonBody}
      responses:
        "200":
          description: OK
          content:
            application/atom+xml:
              schema: {type: object, title: atom}
            application/geo+json:
              schema: {type: object, title: geo}
            application/ld+json:
              schema: {type: object, title: ld}
`

func convertMulti(t *testing.T, source string) (*ConversionResult, *parser.OAS2Document) {
	t.Helper()
	parsed, err := parser.New().ParseBytes([]byte(source))
	require.NoError(t, err)
	result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("2.0"))
	require.NoError(t, err)
	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)
	return result, doc
}

func TestMultipleMediaTypesKeepTheJSONSchema(t *testing.T) {
	_, doc := convertMulti(t, multiMediaTypeSource)

	op := doc.Paths["/a"].Post
	require.NotNil(t, op)

	var body *parser.Parameter
	for _, p := range op.Parameters {
		if p.In == "body" {
			body = p
		}
	}
	require.NotNil(t, body)
	require.NotNil(t, body.Schema)
	assert.Equal(t, "jsonBody", body.Schema.Title, "application/json outranks application/xml")

	resp := op.Responses.Codes["200"]
	require.NotNil(t, resp)
	require.NotNil(t, resp.Schema)
	// No application/json here, so the two +json types rank together and the
	// name decides between them; the XML one loses on rank.
	assert.Equal(t, "geo", resp.Schema.Title)
}

// TestSchemaLessMediaTypeIsNotSelected is the data loss half. Selecting a media
// type carrying no schema leaves the response with none, and a sibling was
// offering one.
func TestSchemaLessMediaTypeIsNotSelected(t *testing.T) {
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
          content:
            application/json: {}
            application/ld+json:
              schema: {type: object, title: kept}
`

	_, doc := convertMulti(t, source)

	resp := doc.Paths["/a"].Get.Responses.Codes["200"]
	require.NotNil(t, resp)
	// application/json outranks application/ld+json and carries no schema, so
	// only the schema check can keep it from being selected. Ranking it lower
	// would mask that.
	require.NotNil(t, resp.Schema, "a media type without a schema must not be selected over one with it")
	assert.Equal(t, "kept", resp.Schema.Title)
}

// TestProducesAndConsumesAreStable pins the ordering, which a map range decided
// before.
//
// The document level merges across operations, so the fixture spreads media
// types over several paths in an order that is not sorted. One operation whose
// own media types are already sorted would pass whether the merge sorts or not.
func TestProducesAndConsumesAreStable(t *testing.T) {
	const source = `openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths:
  /z:
    post:
      operationId: z
      requestBody:
        content:
          text/plain:
            schema: {type: string}
      responses:
        "200":
          description: OK
          content:
            text/csv:
              schema: {type: string}
  /a:
    post:
      operationId: a
      requestBody:
        content:
          application/json:
            schema: {type: object}
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {type: object}
`

	_, doc := convertMulti(t, source)

	assert.Equal(t, []string{"application/json", "text/plain"}, doc.Consumes)
	assert.Equal(t, []string{"application/json", "text/csv"}, doc.Produces)
}

// TestConversionIsRepeatable converts the same document many times and requires
// one answer. Go randomizes map order per range, so a selection or an ordering
// still driven by it shows up here rather than in a single comparison.
func TestConversionIsRepeatable(t *testing.T) {
	_, first := convertMulti(t, multiMediaTypeSource)

	firstBody := bodySchemaTitle(t, first)
	for i := 0; i < 50; i++ {
		_, doc := convertMulti(t, multiMediaTypeSource)
		assert.Equal(t, firstBody, bodySchemaTitle(t, doc), "body schema changed between runs")
		assert.Equal(t, first.Produces, doc.Produces, "produces changed between runs")
		assert.Equal(t, first.Consumes, doc.Consumes, "consumes changed between runs")
		assert.Equal(t,
			first.Paths["/a"].Post.Responses.Codes["200"].Schema.Title,
			doc.Paths["/a"].Post.Responses.Codes["200"].Schema.Title,
			"response schema changed between runs")
	}
}

func bodySchemaTitle(t *testing.T, doc *parser.OAS2Document) string {
	t.Helper()
	for _, p := range doc.Paths["/a"].Post.Parameters {
		if p.In == "body" && p.Schema != nil {
			return p.Schema.Title
		}
	}
	t.Fatal("no body parameter schema")
	return ""
}
