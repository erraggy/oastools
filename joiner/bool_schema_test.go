package joiner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/erraggy/oastools/parser"
)

// TestCompareBoolSchemas covers the bare-boolean schema form through the
// joiner's own comparison, which is a separate implementation from
// parser.Schema.Equals and is the one semantic deduplication actually consults.
//
// The pair that matters is `true` against `false`: they are opposite schemas,
// so calling them equivalent would let deduplication merge a schema that
// accepts everything with one that accepts nothing.
func TestCompareBoolSchemas(t *testing.T) {
	tests := []struct {
		name       string
		left       *parser.Schema
		right      *parser.Schema
		equivalent bool
	}{
		{"true is equivalent to true", parser.NewBoolSchema(true), parser.NewBoolSchema(true), true},
		{"false is equivalent to false", parser.NewBoolSchema(false), parser.NewBoolSchema(false), true},
		{"true is not equivalent to false", parser.NewBoolSchema(true), parser.NewBoolSchema(false), false},
		{"false is not equivalent to true", parser.NewBoolSchema(false), parser.NewBoolSchema(true), false},
		{
			name:  "true is not equivalent to an object schema",
			left:  parser.NewBoolSchema(true),
			right: &parser.Schema{Type: "string"},
		},
		{
			name:  "an object schema is not equivalent to false",
			left:  &parser.Schema{Type: "string"},
			right: parser.NewBoolSchema(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, mode := range []EquivalenceMode{EquivalenceModeDeep, EquivalenceModeShallow} {
				result := CompareSchemas(tt.left, tt.right, mode)
				assert.Equal(t, tt.equivalent, result.Equivalent,
					"mode %s: differences: %v", mode, result.Differences)
			}
		})
	}
}

// TestBoolSchemaIsNotEmpty guards the early return that made the comparison
// above unreachable. isEmptySchema treated a schema carrying only BoolForm as
// empty, so CompareSchemasWithOptions reported two identical `true` schemas as
// non-equivalent — and did it with an empty Differences slice, because nothing
// had been compared.
func TestBoolSchemaIsNotEmpty(t *testing.T) {
	assert.False(t, isEmptySchema(parser.NewBoolSchema(true)),
		"`true` accepts every instance deliberately; it is not an empty schema")
	assert.False(t, isEmptySchema(parser.NewBoolSchema(false)),
		"`false` rejects every instance deliberately; it is not an empty schema")
	assert.True(t, isEmptySchema(&parser.Schema{}),
		"an object schema with no keywords is still empty")
}

// TestCompareBoolSchemasReportsADifference checks the result shape, not just the
// verdict. A non-equivalent result with no differences recorded tells a caller
// nothing about why, which is exactly what the empty-schema early return
// produced before.
func TestCompareBoolSchemasReportsADifference(t *testing.T) {
	result := CompareSchemas(parser.NewBoolSchema(true), parser.NewBoolSchema(false), EquivalenceModeDeep)

	assert.False(t, result.Equivalent)
	assert.NotEmpty(t, result.Differences, "a mismatch should say what differed")
}

// TestCompareNestedBoolSchemas covers boolean schemas below the top level.
// Checking only at the entry point left every nested position unguarded:
// `{p: true}` and `{p: false}` compared equal, because the field-by-field
// comparison finds nothing set on either side and a boolean carries no fields.
func TestCompareNestedBoolSchemas(t *testing.T) {
	object := func(property *parser.Schema) *parser.Schema {
		return &parser.Schema{
			Type:       "object",
			Properties: map[string]*parser.Schema{"p": property},
		}
	}

	tests := []struct {
		name       string
		left       *parser.Schema
		right      *parser.Schema
		equivalent bool
	}{
		{"nested true and true", object(parser.NewBoolSchema(true)), object(parser.NewBoolSchema(true)), true},
		{"nested false and false", object(parser.NewBoolSchema(false)), object(parser.NewBoolSchema(false)), true},
		{"nested true and false", object(parser.NewBoolSchema(true)), object(parser.NewBoolSchema(false)), false},
		{"nested boolean and object", object(parser.NewBoolSchema(true)), object(&parser.Schema{Type: "string"}), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareSchemas(tt.left, tt.right, EquivalenceModeDeep)
			assert.Equal(t, tt.equivalent, result.Equivalent, "differences: %v", result.Differences)
		})
	}
}

// TestCompareSchemaOrBoolRepresentations covers the any-typed schema-or-bool
// fields, where a boolean can arrive either as a raw bool — what the decoders
// leave there — or as a *Schema with BoolForm set, which is what building one
// programmatically produces. The two mean the same thing.
//
// Three separate functions compare these fields (compareSchemaOrBool,
// compareItemsSchemas, comparePolymorphicSchemas), so each field below is
// covered rather than trusting one to stand for the rest.
func TestCompareSchemaOrBoolRepresentations(t *testing.T) {
	wrapped := parser.NewBoolSchema

	tests := []struct {
		name        string
		left, right *parser.Schema
		equivalent  bool
	}{
		{"items raw and wrapped true", &parser.Schema{Items: true}, &parser.Schema{Items: wrapped(true)}, true},
		{"items raw true and wrapped false", &parser.Schema{Items: true}, &parser.Schema{Items: wrapped(false)}, false},
		{"items both raw true", &parser.Schema{Items: true}, &parser.Schema{Items: true}, true},
		{"items raw true and raw false", &parser.Schema{Items: true}, &parser.Schema{Items: false}, false},
		{"items boolean and object", &parser.Schema{Items: true}, &parser.Schema{Items: &parser.Schema{Type: "string"}}, false},

		{"additionalProperties raw and wrapped", &parser.Schema{AdditionalProperties: true}, &parser.Schema{AdditionalProperties: wrapped(true)}, true},
		{"additionalProperties raw true and false", &parser.Schema{AdditionalProperties: true}, &parser.Schema{AdditionalProperties: false}, false},

		{"additionalItems raw and wrapped", &parser.Schema{AdditionalItems: true}, &parser.Schema{AdditionalItems: wrapped(true)}, true},

		{"unevaluatedProperties raw and wrapped", &parser.Schema{UnevaluatedProperties: true}, &parser.Schema{UnevaluatedProperties: wrapped(true)}, true},
		{"unevaluatedItems raw true and wrapped false", &parser.Schema{UnevaluatedItems: true}, &parser.Schema{UnevaluatedItems: wrapped(false)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareSchemas(tt.left, tt.right, EquivalenceModeDeep)
			assert.Equal(t, tt.equivalent, result.Equivalent, "differences: %v", result.Differences)
		})
	}
}
