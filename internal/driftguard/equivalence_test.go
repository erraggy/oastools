package driftguard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// Deep comparison excludes nothing: every field of a Schema affects what a
// document means, documentation included, since [joiner.EquivalenceDocsInclude]
// is the default and decides at runtime whether the documentation fields count.
//
// There is deliberately no exclusion list here. If one becomes necessary, the
// entry belongs beside an assertion that the field really is ignored, not beside
// a skip: a skipped case checks nothing and reads as though it did.

func TestDeepComparisonReadsEveryStructuralSchemaField(t *testing.T) {
	for _, f := range fieldsOf[parser.Schema]() {
		t.Run(f.name, func(t *testing.T) {
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
			require.True(t, populate(right, f),
				"populate cannot produce a value for Schema.%s; extend it rather than "+
					"leaving the field unchecked", f.name)

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
			left, right := &parser.Schema{}, &parser.Schema{}
			require.True(t, populate(right, f),
				"populate cannot produce a value for Schema.%s; extend it rather than "+
					"leaving the field unchecked", f.name)

			assert.False(t, left.Equals(right),
				"Schema.%s differs but Equals reported the schemas equal", f.name)
		})
	}
}
