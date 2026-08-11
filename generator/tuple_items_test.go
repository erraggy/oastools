// tuple_items_test.go covers the file splitter against the OAS 2.0 tuple form
// of `items`.
package generator

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
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
