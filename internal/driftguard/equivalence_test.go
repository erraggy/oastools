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

// TestDeepComparisonSurvivesNilNestedSchemas covers the class behind issue #417
// rather than the individual sites that issue lists. Populating every
// nested-schema field with a nil in turn is what keeps the next nested walk added
// from reintroducing it.
func TestDeepComparisonSurvivesNilNestedSchemas(t *testing.T) {
	// Both schemas carry a type and a property so neither is the empty schema,
	// which compareDeep rejects for unrelated reasons.
	base := func() *parser.Schema {
		return &parser.Schema{
			Type:       "object",
			Properties: map[string]*parser.Schema{"p": {Type: "string"}},
		}
	}
	present := &parser.Schema{Type: "string"}

	// Every Schema field that can hold a nested schema, by the shape that holds it.
	nilIn := map[string]func(*parser.Schema, bool){
		"properties":            func(s *parser.Schema, nilled bool) { s.Properties = schemaMapEntry(nilled, present) },
		"patternProperties":     func(s *parser.Schema, nilled bool) { s.PatternProperties = schemaMapEntry(nilled, present) },
		"dependentSchemas":      func(s *parser.Schema, nilled bool) { s.DependentSchemas = schemaMapEntry(nilled, present) },
		"$defs":                 func(s *parser.Schema, nilled bool) { s.Defs = schemaMapEntry(nilled, present) },
		"allOf":                 func(s *parser.Schema, nilled bool) { s.AllOf = schemaSliceEntry(nilled, present) },
		"anyOf":                 func(s *parser.Schema, nilled bool) { s.AnyOf = schemaSliceEntry(nilled, present) },
		"oneOf":                 func(s *parser.Schema, nilled bool) { s.OneOf = schemaSliceEntry(nilled, present) },
		"prefixItems":           func(s *parser.Schema, nilled bool) { s.PrefixItems = schemaSliceEntry(nilled, present) },
		"not":                   func(s *parser.Schema, nilled bool) { s.Not = schemaPointer(nilled, present) },
		"contains":              func(s *parser.Schema, nilled bool) { s.Contains = schemaPointer(nilled, present) },
		"propertyNames":         func(s *parser.Schema, nilled bool) { s.PropertyNames = schemaPointer(nilled, present) },
		"if":                    func(s *parser.Schema, nilled bool) { s.If = schemaPointer(nilled, present) },
		"then":                  func(s *parser.Schema, nilled bool) { s.Then = schemaPointer(nilled, present) },
		"else":                  func(s *parser.Schema, nilled bool) { s.Else = schemaPointer(nilled, present) },
		"contentSchema":         func(s *parser.Schema, nilled bool) { s.ContentSchema = schemaPointer(nilled, present) },
		"items":                 func(s *parser.Schema, nilled bool) { s.Items = schemaOrBool(nilled, present) },
		"additionalItems":       func(s *parser.Schema, nilled bool) { s.AdditionalItems = schemaOrBool(nilled, present) },
		"additionalProperties":  func(s *parser.Schema, nilled bool) { s.AdditionalProperties = schemaOrBool(nilled, present) },
		"unevaluatedItems":      func(s *parser.Schema, nilled bool) { s.UnevaluatedItems = schemaOrBool(nilled, present) },
		"unevaluatedProperties": func(s *parser.Schema, nilled bool) { s.UnevaluatedProperties = schemaOrBool(nilled, present) },
	}

	for field, set := range nilIn {
		t.Run(field, func(t *testing.T) {
			nilled, populated := base(), base()
			set(nilled, true)
			set(populated, false)

			// The assertion is that these return at all. A panic here is the bug.
			assert.NotPanics(t, func() {
				result := joiner.CompareSchemas(nilled, populated, joiner.EquivalenceModeDeep)
				assert.False(t, result.Equivalent,
					"a nil %s against a populated one is a difference", field)
			})

			assert.NotPanics(t, func() {
				bothNil := base()
				set(bothNil, true)
				other := base()
				set(other, true)
				assert.True(t, joiner.CompareSchemas(bothNil, other, joiner.EquivalenceModeDeep).Equivalent,
					"two schemas both holding a nil %s are equivalent", field)
			})
		})
	}
}

// schemaMapEntry returns a one-entry schema map whose value is nil or present.
func schemaMapEntry(nilled bool, present *parser.Schema) map[string]*parser.Schema {
	if nilled {
		return map[string]*parser.Schema{"a": nil}
	}
	return map[string]*parser.Schema{"a": present}
}

// schemaSliceEntry returns a one-element schema slice holding nil or present.
func schemaSliceEntry(nilled bool, present *parser.Schema) []*parser.Schema {
	if nilled {
		return []*parser.Schema{nil}
	}
	return []*parser.Schema{present}
}

// schemaPointer returns nil or present.
func schemaPointer(nilled bool, present *parser.Schema) *parser.Schema {
	if nilled {
		return nil
	}
	return present
}

// schemaOrBool returns a typed nil or a present schema, in the `any` a
// schema-or-bool field is declared as. A typed nil is the shape that passes an
// interface nil check and still dereferences to nothing.
func schemaOrBool(nilled bool, present *parser.Schema) any {
	if nilled {
		return (*parser.Schema)(nil)
	}
	return present
}
