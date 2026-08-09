package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeSpec writes raw spec text to a temp file and returns its path. The tests
// below convert from source text rather than from a hand-built document, since
// the loss they cover happens between parsing and writing.
func writeSpec(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

// oas2WithExtensions carries one extension on every object that survives a
// conversion to OAS 3.x, so a dropped one is attributable to a single position.
const oas2WithExtensions = `swagger: "2.0"
x-root-ext: root
info:
  title: T
  version: "1.0.0"
  x-info-ext: info
securityDefinitions:
  key:
    type: apiKey
    name: X-Key
    in: header
    x-sec-ext: sec
paths:
  /a:
    x-pathitem-ext: pathitem
    get:
      operationId: a
      x-op-ext: op
      parameters:
        - name: q
          in: query
          type: string
          x-param-ext: param
        - name: body
          in: body
          schema:
            type: object
          x-body-ext: body
      responses:
        x-responses-ext: responses
        "200":
          description: OK
          x-response-ext: response
`

// oas3WithExtensions is the OAS 3.x counterpart of oas2WithExtensions.
const oas3WithExtensions = `openapi: 3.0.3
x-root-ext: root
info:
  title: T
  version: "1.0.0"
  x-info-ext: info
components:
  securitySchemes:
    key:
      type: apiKey
      name: X-Key
      in: header
      x-sec-ext: sec
paths:
  /a:
    x-pathitem-ext: pathitem
    get:
      operationId: a
      x-op-ext: op
      parameters:
        - name: q
          in: query
          schema:
            type: string
          x-param-ext: param
      requestBody:
        x-body-ext: body
        content:
          application/json:
            schema:
              type: object
      responses:
        x-responses-ext: responses
        "200":
          description: OK
          x-response-ext: response
`

// TestOAS2ToOAS3PreservesExtensions covers #463 upgrading. Every converter that
// builds a target value field by field used to omit Extra, so only Info kept its
// extensions, and it kept them by being passed through as a whole object rather
// than rebuilt.
func TestOAS2ToOAS3PreservesExtensions(t *testing.T) {
	path := writeSpec(t, "oas2.yaml", oas2WithExtensions)

	result, err := New().Convert(path, "3.0.3")
	require.NoError(t, err)
	require.True(t, result.Success)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "converted document should be an OAS3Document")

	assert.Equal(t, "root", doc.Extra["x-root-ext"], "root document")
	assert.Equal(t, "info", doc.Info.Extra["x-info-ext"], "info")

	require.NotNil(t, doc.Components)
	scheme := doc.Components.SecuritySchemes["key"]
	require.NotNil(t, scheme)
	assert.Equal(t, "sec", scheme.Extra["x-sec-ext"], "security scheme")

	pathItem := doc.Paths["/a"]
	require.NotNil(t, pathItem)
	assert.Equal(t, "pathitem", pathItem.Extra["x-pathitem-ext"], "path item")

	op := pathItem.Get
	require.NotNil(t, op)
	assert.Equal(t, "op", op.Extra["x-op-ext"], "operation")

	require.Len(t, op.Parameters, 1, "the body parameter becomes requestBody, leaving one")
	assert.Equal(t, "param", op.Parameters[0].Extra["x-param-ext"], "parameter")

	// A body parameter has no counterpart in OAS 3.x other than requestBody, so
	// that is where its extensions belong.
	require.NotNil(t, op.RequestBody)
	assert.Equal(t, "body", op.RequestBody.Extra["x-body-ext"], "request body")

	require.NotNil(t, op.Responses)
	assert.Equal(t, "responses", op.Responses.Extra["x-responses-ext"], "responses")

	resp := op.Responses.Codes["200"]
	require.NotNil(t, resp)
	assert.Equal(t, "response", resp.Extra["x-response-ext"], "response")
}

// TestOAS3ToOAS2PreservesExtensions covers #463 downgrading.
func TestOAS3ToOAS2PreservesExtensions(t *testing.T) {
	path := writeSpec(t, "oas3.yaml", oas3WithExtensions)

	result, err := New().Convert(path, "2.0")
	require.NoError(t, err)
	require.True(t, result.Success)

	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok, "converted document should be an OAS2Document")

	assert.Equal(t, "root", doc.Extra["x-root-ext"], "root document")
	assert.Equal(t, "info", doc.Info.Extra["x-info-ext"], "info")

	scheme := doc.SecurityDefinitions["key"]
	require.NotNil(t, scheme)
	assert.Equal(t, "sec", scheme.Extra["x-sec-ext"], "security definition")

	pathItem := doc.Paths["/a"]
	require.NotNil(t, pathItem)
	assert.Equal(t, "pathitem", pathItem.Extra["x-pathitem-ext"], "path item")

	op := pathItem.Get
	require.NotNil(t, op)
	assert.Equal(t, "op", op.Extra["x-op-ext"], "operation")

	// requestBody becomes a body parameter, so its extensions ride along with it.
	var query, body *parser.Parameter
	for _, p := range op.Parameters {
		switch p.In {
		case "query":
			query = p
		case "body":
			body = p
		}
	}
	require.NotNil(t, query)
	assert.Equal(t, "param", query.Extra["x-param-ext"], "parameter")
	require.NotNil(t, body)
	assert.Equal(t, "body", body.Extra["x-body-ext"], "body parameter from requestBody")

	require.NotNil(t, op.Responses)
	assert.Equal(t, "responses", op.Responses.Extra["x-responses-ext"], "responses")

	resp := op.Responses.Codes["200"]
	require.NotNil(t, resp)
	assert.Equal(t, "response", resp.Extra["x-response-ext"], "response")
}

// TestConversionDoesNotShareExtensionMaps guards the copy itself. Assigning the
// source map instead of cloning it would leave both documents pointing at one
// map, so a write to the converted document would reach back into the source.
func TestConversionDoesNotShareExtensionMaps(t *testing.T) {
	path := writeSpec(t, "oas2.yaml", oas2WithExtensions)

	c := New()
	parsed, err := parser.ParseWithOptions(parser.WithFilePath(path))
	require.NoError(t, err)
	src, ok := parsed.OAS2Document()
	require.True(t, ok)

	result, err := c.ConvertParsed(*parsed, "3.0.3")
	require.NoError(t, err)
	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok)

	doc.Extra["x-root-ext"] = "rewritten"
	assert.Equal(t, "root", src.Extra["x-root-ext"],
		"writing to the converted document must not reach the source")
}

// TestCloneExtensionsNilStaysNil keeps an object with no extensions from gaining
// an empty map, which would serialize differently from the source.
func TestCloneExtensionsNilStaysNil(t *testing.T) {
	assert.Nil(t, parser.CloneExtensions(nil))
}
