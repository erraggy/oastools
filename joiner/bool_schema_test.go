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
