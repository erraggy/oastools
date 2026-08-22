// schema_oas3_to_oas3.go runs the per-schema passes a conversion between two
// OAS 3.x versions needs.
//
// Two of the passes the cross-major conversions run are absent here because they
// have nothing to do inside OAS 3.x: both 3.x versions spell a schema reference
// as #/components/schemas, so there is no ref to rewrite, and both spell a
// discriminator as an object, so there is no form to flip.

package converter

import (
	"github.com/erraggy/oastools/parser"
)

// applyOAS3SchemaPasses converts a schema in place for a target OAS 3.x version.
//
// In place rather than returning a copy, because the only caller has already
// deep-copied the whole document and every position would otherwise need a
// setter as well as a getter.
//
// The passes key on the target version alone. Nothing here needs to know the
// source, because each pass either applies to every document or asks only what
// the target can spell.
func (c *Converter) applyOAS3SchemaPasses(schema *parser.Schema, targetVersion parser.OASVersion, result *ConversionResult, path string) {
	if schema == nil {
		return
	}

	dropArrayValuedSchemaOrBool(c, schema, result, path)

	if c.isOAS31OrLater(targetVersion) {
		fixSchemaExclusiveMinMaxForOAS31(c, schema, result, path, make(map[*parser.Schema]bool))
		tupleToPrefixItems(c, schema, result, path)
	} else {
		fixSchemaExclusiveMinMaxForOAS30(schema)
		prefixItemsForOAS30(c, schema, result, path)
	}
}

// prefixItemsForOAS30 rewrites a 2020-12 tuple into what OAS 3.0 can spell, by
// chaining the two existing passes rather than adding a third.
//
// prefixItemsToTuple moves the positions into the draft 4 array form of items,
// and tupleForOAS30 then collapses or drops that array. The draft 4 spelling is
// not legal OAS 3.0 either, so the intermediate must not escape: it does not,
// because tupleForOAS30 visits every schema prefixItemsToTuple can write an
// array into, and leaves items holding a single schema in every branch.
func prefixItemsForOAS30(c *Converter, schema *parser.Schema, result *ConversionResult, path string) {
	prefixItemsToTuple(c, schema, result, path, oas30Spelling)
	// The source spelled the tuple with prefixItems, so the diagnostics name
	// that rather than the draft 4 form the pass above just produced.
	tupleForOAS30(c, schema, result, path, prefixItemsTuple)
}

// fixSchemaExclusiveMinMaxForOAS30 rewrites the 2020-12 numeric exclusive bounds
// into the draft 4 pair OAS 3.0 uses: a boolean flag qualifying maximum or
// minimum. It is the counterpart of fixSchemaExclusiveMinMaxForOAS31.
//
// A schema may carry both bounds, because 2020-12 lets maximum and
// exclusiveMaximum constrain independently, and draft 4 has only one value to
// put them in. Nothing is lost by that: `x <= M and x < E` is `x < E` when
// E <= M and `x <= M` otherwise, so one of the two is always the binding bound
// and the other is implied by it. The minimum side is the mirror image.
//
// Reporting nothing is why it takes neither a Converter nor a
// ConversionResult, unlike the passes either side of it.
func fixSchemaExclusiveMinMaxForOAS30(schema *parser.Schema) {
	walkSchemas(schema, func(s *parser.Schema) {
		if e, ok := numericBound(s.ExclusiveMaximum); ok {
			if s.Maximum != nil && *s.Maximum < e {
				// The inclusive bound is the binding one, so the exclusive bound
				// says nothing the output does not already say.
				s.ExclusiveMaximum = nil
			} else {
				bound := e
				s.Maximum = &bound
				s.ExclusiveMaximum = true
			}
		}
		if e, ok := numericBound(s.ExclusiveMinimum); ok {
			if s.Minimum != nil && *s.Minimum > e {
				s.ExclusiveMinimum = nil
			} else {
				bound := e
				s.Minimum = &bound
				s.ExclusiveMinimum = true
			}
		}
	})
}

// numericBound reports the value of a 2020-12 exclusive bound. A bool is draft
// 4's spelling and is already what OAS 3.0 wants, so it is not a numeric bound.
func numericBound(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
