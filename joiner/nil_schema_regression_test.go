package joiner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// TestSemanticDedup_NilPropertyValue is the regression for issue #417, at the
// level it was reported: a valid document, parsed, joined with semantic
// deduplication. That issue covers why both documents have to carry the same nil
// before the comparison ever reaches it.
//
// Parsed rather than hand-built, since the point is that an ordinary document
// produces this shape without the caller doing anything unusual.
func TestSemanticDedup_NilPropertyValue(t *testing.T) {
	specA := []byte(`
openapi: 3.1.0
info:
  title: API a
  version: "1.0.0"
paths: {}
components:
  schemas:
    Thinga:
      type: object
      properties:
        name:
`)
	specB := []byte(`
openapi: 3.1.0
info:
  title: API b
  version: "1.0.0"
paths: {}
components:
  schemas:
    Thingb:
      type: object
      properties:
        name:
`)

	// A nil error does not mean a clean parse: collected errors ride on Errors,
	// which JoinParsed rejects. Checked here so a bad fixture reports as one.
	p := parser.New()
	docA, err := p.ParseBytes(specA)
	require.NoError(t, err)
	require.Empty(t, docA.Errors)
	docB, err := p.ParseBytes(specB)
	require.NoError(t, err)
	require.Empty(t, docB.Errors)

	// Distinct source paths, so the join does not warn about generic names and
	// bury the thing under test.
	docA.SourcePath = "crash-a.yaml"
	docB.SourcePath = "crash-b.yaml"

	parsedA, ok := docA.OAS3Document()
	require.True(t, ok)
	schemas := parsedA.Components.Schemas
	require.Contains(t, schemas, "Thinga")
	require.Contains(t, schemas["Thinga"].Properties, "name",
		"the parse must keep the empty property as a present key, or this test proves nothing")
	require.Nil(t, schemas["Thinga"].Properties["name"],
		"the empty property must parse to a nil schema, or this test proves nothing")

	j := New(JoinerConfig{SemanticDeduplication: true})

	// Before the guard moved into compareDeep this panicked rather than failing.
	var result *JoinResult
	require.NotPanics(t, func() {
		result, err = j.JoinParsed([]parser.ParseResult{*docA, *docB})
	})
	require.NoError(t, err)
	joinedDoc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok)

	// The two schemas are genuinely equivalent, so deduplication should still do
	// its job: surviving the nil is not the same as treating it as a difference.
	joined := joinedDoc.Components.Schemas
	assert.Len(t, joined, 1,
		"two identical schemas differing only in name should deduplicate to one")
}
