package schemautil

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/erraggy/oastools/parser"
)

// A boolean schema has two representations in the parser model. In a
// schema-or-bool field such as Items it stays a raw bool, because the promotion
// step passes bools through untouched; in a *Schema-typed field it arrives as a
// Schema with BoolForm set. They mean the same thing, so they must hash the
// same — otherwise deduplication sorts equivalent schemas into different
// buckets and never compares them.
func TestHashBoolSchemaRepresentationsAgree(t *testing.T) {
	tests := []struct {
		name string
		raw  *parser.Schema
		wrap *parser.Schema
	}{
		{
			name: "items true",
			raw:  &parser.Schema{Type: "array", Items: true},
			wrap: &parser.Schema{Type: "array", Items: parser.NewBoolSchema(true)},
		},
		{
			name: "items false",
			raw:  &parser.Schema{Type: "array", Items: false},
			wrap: &parser.Schema{Type: "array", Items: parser.NewBoolSchema(false)},
		},
		{
			name: "additionalProperties true",
			raw:  &parser.Schema{Type: "object", AdditionalProperties: true},
			wrap: &parser.Schema{Type: "object", AdditionalProperties: parser.NewBoolSchema(true)},
		},
		{
			name: "additionalProperties false",
			raw:  &parser.Schema{Type: "object", AdditionalProperties: false},
			wrap: &parser.Schema{Type: "object", AdditionalProperties: parser.NewBoolSchema(false)},
		},
		{
			name: "additionalItems true",
			raw:  &parser.Schema{Type: "array", AdditionalItems: true},
			wrap: &parser.Schema{Type: "array", AdditionalItems: parser.NewBoolSchema(true)},
		},
		{
			name: "additionalItems false",
			raw:  &parser.Schema{Type: "array", AdditionalItems: false},
			wrap: &parser.Schema{Type: "array", AdditionalItems: parser.NewBoolSchema(false)},
		},
		{
			name: "unevaluatedProperties true",
			raw:  &parser.Schema{Type: "object", UnevaluatedProperties: true},
			wrap: &parser.Schema{Type: "object", UnevaluatedProperties: parser.NewBoolSchema(true)},
		},
		{
			name: "unevaluatedItems false",
			raw:  &parser.Schema{Type: "array", UnevaluatedItems: false},
			wrap: &parser.Schema{Type: "array", UnevaluatedItems: parser.NewBoolSchema(false)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, NewSchemaHasher().Hash(tt.raw), NewSchemaHasher().Hash(tt.wrap),
				"the raw bool and BoolForm representations must hash alike")
		})
	}
}

// The two boolean values are opposite schemas, so they must not collide — in
// either representation, and at the top level as well as nested.
func TestHashBoolSchemaValuesDiffer(t *testing.T) {
	tests := []struct {
		name string
		a, b *parser.Schema
	}{
		{
			name: "top-level true and false",
			a:    parser.NewBoolSchema(true),
			b:    parser.NewBoolSchema(false),
		},
		{
			name: "raw items true and false",
			a:    &parser.Schema{Type: "array", Items: true},
			b:    &parser.Schema{Type: "array", Items: false},
		},
		{
			name: "a boolean schema and an empty object schema",
			a:    parser.NewBoolSchema(true),
			b:    &parser.Schema{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEqual(t, NewSchemaHasher().Hash(tt.a), NewSchemaHasher().Hash(tt.b),
				"distinct schemas must not share a deduplication bucket")
		})
	}
}

// TestHashAndEqualsAgreeOnBoolSpellings asserts the joint invariant rather than
// either half of it: things that hash alike must compare alike. The hasher was
// taught the two spellings first, which left the contract broken in the other
// direction, with both spellings sharing a bucket that the comparison then
// split (#504). A test on the hasher alone cannot see that.
func TestHashAndEqualsAgreeOnBoolSpellings(t *testing.T) {
	arr := func(items any) *parser.Schema {
		return &parser.Schema{Type: "array", Items: items}
	}

	tests := []struct {
		name  string
		left  *parser.Schema
		right *parser.Schema
		same  bool
	}{
		{"bare true and BoolForm true", arr(true), arr(parser.NewBoolSchema(true)), true},
		{"bare false and BoolForm false", arr(false), arr(parser.NewBoolSchema(false)), true},
		{"bare true and BoolForm false", arr(true), arr(parser.NewBoolSchema(false)), false},
		{"bare true and an object schema", arr(true), arr(&parser.Schema{Type: "string"}), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewSchemaHasher()
			hashesAgree := h.Hash(tt.left) == h.Hash(tt.right)
			equal := tt.left.Equals(tt.right)

			assert.Equal(t, tt.same, equal, "Equals")
			assert.Equal(t, tt.same, hashesAgree, "Hash")

			// The contract itself, stated once: a shared bucket that the
			// comparison rejects is the failure mode deduplication cannot see.
			if hashesAgree {
				assert.True(t, equal,
					"hash equal implies compare equal, or deduplication buckets what it will not merge")
			}
		})
	}
}
