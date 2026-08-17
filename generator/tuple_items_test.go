// tuple_items_test.go covers code generation against the OAS 2.0 tuple form of
// `items`, both at the file splitter and end to end.
package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileSplitterCollectsTypesUsedFromTupleElements covers the set the splitter
// builds to decide a generated file's imports. A type reachable only from a
// tuple element that is missing from the set yields code that does not compile.
func TestFileSplitterCollectsTypesUsedFromTupleElements(t *testing.T) {
	fs := &FileSplitter{}

	used := map[string]bool{}
	fs.collectSchemaRefs(&parser.Schema{
		Type:                 "array",
		Items:                []*parser.Schema{{Type: "string"}, {Ref: "#/definitions/FromItems"}},
		AdditionalProperties: []*parser.Schema{{Ref: "#/definitions/FromAdditionalProperties"}},
	}, used)

	assert.True(t, used["FromItems"],
		"a type used only from a tuple element was left out of the import set")
	assert.True(t, used["FromAdditionalProperties"],
		"a type used only from a tuple additionalProperties element was left out")
}

// TestGeneratedTypesForTupleItems pins what a tuple maps to in generated code.
// Go has no tuple type, so an array whose items is a tuple becomes []any, and
// every type its elements reference is generated, including one reachable only
// from a later element (#507).
func TestGeneratedTypesForTupleItems(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Tuple
  version: "1.0.0"
paths:
  /pairs:
    get:
      operationId: getPairs
      responses:
        "200":
          description: OK
          schema:
            $ref: "#/definitions/PairList"
definitions:
  PairList:
    type: array
    items:
      - $ref: "#/definitions/FirstType"
      - $ref: "#/definitions/SecondType"
  FirstType:
    type: object
    properties:
      a:
        type: string
  SecondType:
    type: object
    properties:
      b:
        type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tuple.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(spec), 0600))

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("tuple"),
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	require.NotNil(t, typesFile, "types.go not generated")
	content := string(typesFile.Content)

	assert.Contains(t, content, "type PairList []any",
		"a tuple should generate as []any, since Go has no tuple type")
	assert.Contains(t, content, "type FirstType struct",
		"a type referenced by the first tuple element should be generated")
	assert.Contains(t, content, "type SecondType struct",
		"a type referenced by a later tuple element should be generated")
}
