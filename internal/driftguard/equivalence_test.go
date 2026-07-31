package driftguard

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/erraggy/oastools/joiner"
	"github.com/erraggy/oastools/parser"
)

// joiner's deep comparison is the step that verifies a hash grouping before
// deduplication merges it. A field it does not compare cannot split a false
// positive, so a hash collision becomes a merge of two different schemas.
//
// The two checks are meant to agree. [parser.Schema.Equals] compares a field or
// deliberately excludes it; the joiner should reach the same verdict for anything
// structural. This guard sets one field at a time and requires both to notice.

// equivalenceExclusions lists fields deep comparison deliberately treats as not
// affecting equivalence, with the reason.
//
// Documentation fields are absent from this list on purpose: they are handled at
// runtime by [joiner.EquivalenceDocsInclude], which is the default and makes them
// count. Anything listed here is excluded unconditionally.
var equivalenceExclusions = map[string]string{
	// A $ref schema is compared by the definition it names, which is walked at the
	// top level, so comparing the ref string here would double-report.
	"Ref": "compared via the definition it names",
}

func TestDeepComparisonReadsEveryStructuralSchemaField(t *testing.T) {
	for _, f := range fieldsOf[parser.Schema]() {
		t.Run(f.name, func(t *testing.T) {
			if reason, skipped := equivalenceExclusions[f.name]; skipped {
				t.Skipf("deliberately not compared: %s", reason)
			}

			// A type and a property keep both schemas out of the empty-schema early
			// return, which reports any two empty schemas as non-equivalent for
			// reasons unrelated to the field under test.
			base := func() *parser.Schema {
				return &parser.Schema{
					Type:       "object",
					Properties: map[string]*parser.Schema{"p": {Type: "string"}},
				}
			}

			left, right := base(), base()
			if !populate(right, f) {
				t.Skip("no distinctive value for this field's type")
			}

			result := joiner.CompareSchemas(left, right, joiner.EquivalenceModeDeep)
			assert.False(t, result.Equivalent,
				"Schema.%s differs but deep comparison called the schemas equivalent; "+
					"semantic deduplication would merge them", f.name)
		})
	}
}

// TestEqualsReadsEveryStructuralSchemaField holds parser's own equality to the
// same standard, so the two cannot drift apart from each other either.
func TestEqualsReadsEveryStructuralSchemaField(t *testing.T) {
	for _, f := range fieldsOf[parser.Schema]() {
		t.Run(f.name, func(t *testing.T) {
			if f.name == "StringForm" {
				t.Skip("not a Schema field")
			}

			left, right := &parser.Schema{}, &parser.Schema{}
			if !populate(right, f) {
				t.Skip("no distinctive value for this field's type")
			}

			assert.False(t, left.Equals(right),
				"Schema.%s differs but Equals reported the schemas equal", f.name)
		})
	}
}
