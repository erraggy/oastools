package schemautil

import (
	"iter"
	"strconv"

	"github.com/erraggy/oastools/parser"
)

// SingleForm is the index [SchemaOrBoolSchemas] yields for a field holding one
// schema rather than a list. See [IndexSuffix].
const SingleForm = -1

// SchemaOrBoolSchemas iterates the schemas held by a schema-or-bool field:
// Items, AdditionalProperties, AdditionalItems, UnevaluatedItems or
// UnevaluatedProperties.
//
// These fields are typed `any` because OAS gives them three shapes, and every
// decode path (JSON, YAML, decodeFromMap) yields one of them:
//
//   - *parser.Schema, yielded once with index [SingleForm]
//   - []*parser.Schema, the OAS 2.0 tuple form, yielded with indices 0..n-1
//   - bool, which holds no schema and yields nothing
//
// A traversal that omits the tuple form cannot see a $ref held in it (#502).
//
// Elements that are nil are skipped, so callers need no nil check of their own.
func SchemaOrBoolSchemas(field any) iter.Seq2[int, *parser.Schema] {
	return func(yield func(int, *parser.Schema) bool) {
		switch v := field.(type) {
		case *parser.Schema:
			if v != nil {
				yield(SingleForm, v)
			}
		case []*parser.Schema:
			for i, s := range v {
				if s == nil {
					continue
				}
				if !yield(i, s) {
					return
				}
			}
		}
	}
}

// IndexSuffix renders the JSON path suffix for an index from
// [SchemaOrBoolSchemas]. Appended to a field path it gives "items" for the
// single-schema form and "items[0]", "items[1]" for tuple elements.
func IndexSuffix(i int) string {
	if i == SingleForm {
		return ""
	}
	return "[" + strconv.Itoa(i) + "]"
}

// SchemaTuple returns the OAS 2.0 tuple form held by a schema-or-bool field,
// one schema per position, and whether the field holds that form at all.
//
// An empty tuple is still the tuple form. Draft 4 gives it a meaning: it names
// no position, so additionalItems governs every element and additionalItems
// false admits only an empty array. Callers must therefore tell it apart from a
// field holding a single schema or a bool, which is what the second return
// value is for. Testing the slice's length cannot do that.
//
// The slice is returned rather than iterated as [SchemaOrBoolSchemas] does,
// because a caller mapping an array position to the schema constraining it
// needs indexed access, and a nil element means that position is unconstrained
// rather than absent.
func SchemaTuple(field any) ([]*parser.Schema, bool) {
	tuple, ok := field.([]*parser.Schema)
	return tuple, ok
}
