package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/internal/schemautil"
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

// oas2FormDataWithExtensions exercises the other position with no same-named
// counterpart: formData parameters collapse into a single request body schema,
// one property each, so the property schema is the only place a parameter's
// extensions can land.
const oas2FormDataWithExtensions = `swagger: "2.0"
info:
  title: T
  version: "1.0.0"
consumes:
  - application/x-www-form-urlencoded
paths:
  /upload:
    post:
      operationId: upload
      parameters:
        - name: label
          in: formData
          type: string
          required: true
          x-label-ext: label
        - name: size
          in: formData
          type: integer
          x-size-ext: size
      responses:
        "200":
          description: OK
`

// TestOAS2FormDataExtensionsFollowTheProperty covers the formData half of the
// conversion. Nothing in OAS 3.x corresponds to a formData parameter, so the
// extensions travel with the object into the shape it becomes.
func TestOAS2FormDataExtensionsFollowTheProperty(t *testing.T) {
	path := writeSpec(t, "oas2-formdata.yaml", oas2FormDataWithExtensions)

	result, err := New().Convert(path, "3.0.3")
	require.NoError(t, err)
	require.True(t, result.Success)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok)

	op := doc.Paths["/upload"].Post
	require.NotNil(t, op)
	require.NotNil(t, op.RequestBody)

	media := op.RequestBody.Content["application/x-www-form-urlencoded"]
	require.NotNil(t, media, "formData converts to a urlencoded body")
	require.NotNil(t, media.Schema)

	label := media.Schema.Properties["label"]
	require.NotNil(t, label)
	assert.Equal(t, "label", label.Extra["x-label-ext"])

	size := media.Schema.Properties["size"]
	require.NotNil(t, size)
	assert.Equal(t, "size", size.Extra["x-size-ext"])

	// The property schema keeps its own meaning too: the extension rides along
	// with the converted type, it does not replace it.
	assert.Equal(t, "string", schemautil.GetPrimaryType(label))
	assert.Equal(t, "integer", schemautil.GetPrimaryType(size))
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

// TestUpgradeDoesNotShareStateWithSource guards the copies. Assigning a source
// field straight across leaves both documents holding one object, so a write to
// the converted document reaches back into the caller's source. Info was the
// worst of these: it was passed through whole, so this covers the title as well
// as the extensions.
func TestUpgradeDoesNotShareStateWithSource(t *testing.T) {
	path := writeSpec(t, "oas2.yaml", oas2WithExtensions)

	parsed, err := parser.ParseWithOptions(parser.WithFilePath(path))
	require.NoError(t, err)
	src, ok := parsed.OAS2Document()
	require.True(t, ok)

	result, err := New().ConvertParsed(*parsed, "3.0.3")
	require.NoError(t, err)
	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok)

	doc.Extra["x-root-ext"] = "rewritten"
	assert.Equal(t, "root", src.Extra["x-root-ext"], "root extensions")

	require.NotSame(t, src.Info, doc.Info, "Info must not be the same object")
	doc.Info.Extra["x-info-ext"] = "rewritten"
	doc.Info.Title = "rewritten"
	assert.Equal(t, "info", src.Info.Extra["x-info-ext"], "info extensions")
	assert.Equal(t, "T", src.Info.Title, "info title")

	op := doc.Paths["/a"].Get
	require.NotNil(t, op)
	op.Extra["x-op-ext"] = "rewritten"
	assert.Equal(t, "op", src.Paths["/a"].Get.Extra["x-op-ext"], "operation extensions")
}

// TestDowngradeDoesNotShareStateWithSource is the same guard in the other
// direction, which rebuilds its target through different functions.
func TestDowngradeDoesNotShareStateWithSource(t *testing.T) {
	path := writeSpec(t, "oas3.yaml", oas3WithExtensions)

	parsed, err := parser.ParseWithOptions(parser.WithFilePath(path))
	require.NoError(t, err)
	src, ok := parsed.OAS3Document()
	require.True(t, ok)

	result, err := New().ConvertParsed(*parsed, "2.0")
	require.NoError(t, err)
	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)

	doc.Extra["x-root-ext"] = "rewritten"
	assert.Equal(t, "root", src.Extra["x-root-ext"], "root extensions")

	require.NotSame(t, src.Info, doc.Info, "Info must not be the same object")
	doc.Info.Extra["x-info-ext"] = "rewritten"
	doc.Info.Title = "rewritten"
	assert.Equal(t, "info", src.Info.Extra["x-info-ext"], "info extensions")
	assert.Equal(t, "T", src.Info.Title, "info title")
}
