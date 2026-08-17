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

// generateTupleTypes runs generation over spec and returns the content of
// types.go, failing the test if generation reports any issue. An issue here
// would mean the generated Go source did not parse, since the generator formats
// every file it writes.
func generateTupleTypes(t *testing.T, spec string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tuple.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(spec), 0600))

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("tuple"),
	)
	require.NoError(t, err)
	require.Empty(t, result.Issues, "generation reported issues")

	typesFile := result.GetFile("types.go")
	require.NotNil(t, typesFile, "types.go not generated")
	return string(typesFile.Content)
}

// TestGeneratedTypeForTuple covers the struct generated for a tuple: one field
// per position, a Rest field for the positions past the end, and the JSON
// methods that keep the type marshaling as the array its schema describes.
//
// Positions are optional because draft 4 accepts an array shorter than the
// tuple, so the fields can hold nil.
func TestGeneratedTypeForTuple(t *testing.T) {
	content := generateTupleTypes(t, `swagger: "2.0"
info:
  title: Tuple
  version: "1.0.0"
paths: {}
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
`)

	assert.Contains(t, content, "type PairList struct {")
	assert.Contains(t, content, "Item0 *FirstType")
	assert.Contains(t, content, "Item1 *SecondType")
	assert.Contains(t, content, "Rest  []any")

	assert.Contains(t, content, "func (t PairList) MarshalJSON() ([]byte, error)")
	assert.Contains(t, content, "func (t *PairList) UnmarshalJSON(data []byte) error")

	// The methods need both imports, and the generated file must declare them.
	assert.Contains(t, content, `"encoding/json"`)
	assert.Contains(t, content, `"fmt"`)

	// A type reachable only from a later position is still generated.
	assert.Contains(t, content, "type FirstType struct")
	assert.Contains(t, content, "type SecondType struct")
}

// TestGeneratedTypeForTupleAdditionalItems covers what additionalItems does to
// the generated struct: false forbids the positions past the end so no field
// holds them, and a schema types the field that does.
func TestGeneratedTypeForTupleAdditionalItems(t *testing.T) {
	content := generateTupleTypes(t, `swagger: "2.0"
info:
  title: Tuple
  version: "1.0.0"
paths: {}
definitions:
  Closed:
    type: array
    items:
      - type: string
      - type: integer
    additionalItems: false
  TypedRest:
    type: array
    items:
      - type: string
    additionalItems:
      type: integer
  OpenRest:
    type: array
    items:
      - type: string
    additionalItems: true
`)

	// additionalItems false: no Rest field, and decoding a longer array fails
	// rather than dropping the elements that do not fit.
	assert.Contains(t, content, "type Closed struct {\n\tItem0 *string\n\tItem1 *int64\n}",
		"a closed tuple should have no Rest field")
	assert.Contains(t, content, "additionalItems is false so at most 2 are allowed")

	// A schema valued additionalItems types the Rest field.
	assert.Contains(t, content, "Rest  []int64")

	// additionalItems true names no schema, so those positions hold anything.
	assert.Contains(t, content, "type OpenRest struct {")
}

// TestGeneratedTypeForTupleElementForms covers the element forms a tuple can
// hold: a nil element leaves its position unconstrained, and an empty tuple
// names no position at all so the schema stays a plain slice.
func TestGeneratedTypeForTupleElementForms(t *testing.T) {
	content := generateTupleTypes(t, `swagger: "2.0"
info:
  title: Tuple
  version: "1.0.0"
paths: {}
definitions:
  Unconstrained:
    type: array
    items:
      - type: string
      - null
  Empty:
    type: array
    items: []
  EmptyTyped:
    type: array
    items: []
    additionalItems:
      type: integer
`)

	// A nil element constrains nothing, so the position holds any value.
	assert.Contains(t, content, "type Unconstrained struct {")
	assert.Contains(t, content, "Item1 any")

	// An empty tuple names no position, so it is not a struct.
	assert.Contains(t, content, "type Empty []any")

	// With no position named, additionalItems governs every element rather than
	// only the ones past the end.
	assert.Contains(t, content, "type EmptyTyped []int64")
}

// TestGeneratedTupleMarshalShape covers the shape of the generated MarshalJSON.
// Positions are written up to the last one that is set, so a single position
// needs no switch and several positions need one case each, longest first.
func TestGeneratedTupleMarshalShape(t *testing.T) {
	content := generateTupleTypes(t, `swagger: "2.0"
info:
  title: Tuple
  version: "1.0.0"
paths: {}
definitions:
  One:
    type: array
    items:
      - type: string
  Three:
    type: array
    items:
      - type: string
      - type: integer
      - type: boolean
`)

	// One position: an if, since a switch with a single case draws a lint
	// warning in the generated code.
	assert.Contains(t, content, "\tif t.Item0 != nil {\n\t\titems = append(items, t.Item0)\n\t}")

	// Rest values sit after the tuple, so an unset position before them is
	// padded. Without this a set position, an unset one and a non-empty Rest
	// write the Rest values into the unset position, and decoding the result
	// puts them in the wrong field.
	assert.Contains(t, content, "\tif len(t.Rest) > 0 {\n\t\tfor len(items) < 1 {\n\t\t\titems = append(items, nil)\n\t\t}\n\t}")

	// Three positions: the whole switch is asserted as one block, because the
	// order is the point. Ascending cases would match t.Item0 first and write
	// an array that stops there even when a later position is set.
	assert.Contains(t, content, "\tswitch {\n"+
		"\tcase t.Item2 != nil:\n\t\titems = append(items, t.Item0, t.Item1, t.Item2)\n"+
		"\tcase t.Item1 != nil:\n\t\titems = append(items, t.Item0, t.Item1)\n"+
		"\tcase t.Item0 != nil:\n\t\titems = append(items, t.Item0)\n"+
		"\t}\n")
}
