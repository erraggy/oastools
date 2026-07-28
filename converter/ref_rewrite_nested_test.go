package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Converting between dialects has to rewrite every $ref, and rewriteSchemaRefs
// reaches into items and additionalProperties by type-asserting the any-typed
// field to *parser.Schema. A YAML document whose nested schemas were left as
// map[string]any therefore converted into a document with dangling refs — the
// output still pointed at #/components/schemas after a downconvert to 2.0.
// See erraggy/oastools#396.

const nestedRefSpecYAML = `openapi: "3.0.3"
info:
  title: Test
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      type: object
    PetList:
      type: array
      items:
        $ref: '#/components/schemas/Pet'
    PetMap:
      type: object
      additionalProperties:
        $ref: '#/components/schemas/Pet'
`

func TestConvertRewritesRefsNestedInYAMLSchemas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.yaml")
	require.NoError(t, os.WriteFile(path, []byte(nestedRefSpecYAML), 0o644))

	result, err := New().Convert(path, "2.0")
	require.NoError(t, err)
	require.True(t, result.Success)

	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)

	items, ok := doc.Definitions["PetList"].Items.(*parser.Schema)
	require.True(t, ok, "expected items to be *parser.Schema, got %T", doc.Definitions["PetList"].Items)
	assert.Equal(t, "#/definitions/Pet", items.Ref,
		"a ref inside items must be rewritten for the target dialect")

	addProps, ok := doc.Definitions["PetMap"].AdditionalProperties.(*parser.Schema)
	require.True(t, ok, "expected additionalProperties to be *parser.Schema, got %T", doc.Definitions["PetMap"].AdditionalProperties)
	assert.Equal(t, "#/definitions/Pet", addProps.Ref,
		"a ref inside additionalProperties must be rewritten for the target dialect")
}
