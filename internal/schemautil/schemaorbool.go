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
