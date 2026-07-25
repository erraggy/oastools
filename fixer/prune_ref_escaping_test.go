package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPruneKeepsEscapedNameSchema covers the highest-consequence half of issue
// #379.
//
// Pruning recovers a schema name from each collected ref to decide what is
// referenced. Without reversing the RFC 6901 escaping, the recovered name is
// "pet~1summary", which never matches the "pet/summary" key, so a schema that IS
// referenced gets deleted. That is silent data loss rather than the false
// positive the validator side of the issue produced, which is why it is pinned
// separately.
//
// OAS 2.0 places no charset constraint on definition keys, so this document is
// legitimate.
func TestPruneKeepsEscapedNameSchema(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		schemaName string
	}{
		{
			name:       "escaped slash",
			schemaName: "pet/summary",
			spec: `
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  pet/summary: {type: object, properties: {id: {type: string}}}
paths:
  /pets:
    get:
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/pet~1summary'}
`,
		},
		{
			name:       "escaped tilde",
			schemaName: "pet~summary",
			spec: `
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  pet~summary: {type: object, properties: {id: {type: string}}}
paths:
  /pets:
    get:
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/pet~0summary'}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(tt.spec)))
			require.NoError(t, err)

			f := New()
			f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
			result, err := f.FixParsed(*parseResult)
			require.NoError(t, err)

			doc := result.Document.(*parser.OAS2Document)
			assert.Contains(t, doc.Definitions, tt.schemaName,
				"a referenced schema must survive pruning even when its name needs escaping")
			assert.Zero(t, result.FixCount, "nothing should have been pruned")
		})
	}
}

// TestPruneStillRemovesUnreferencedEscapedNameSchema is the control for the test
// above: unescaping must not make every escaped-name schema look referenced.
func TestPruneStillRemovesUnreferencedEscapedNameSchema(t *testing.T) {
	spec := `
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  pet/summary: {type: object, properties: {id: {type: string}}}
  pet/orphan: {type: string}
paths:
  /pets:
    get:
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/pet~1summary'}
`

	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	doc := result.Document.(*parser.OAS2Document)
	assert.Contains(t, doc.Definitions, "pet/summary")
	assert.NotContains(t, doc.Definitions, "pet/orphan")
	assert.Equal(t, 1, result.FixCount)
}
