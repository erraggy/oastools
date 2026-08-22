// schema_oas3_to_oas3.go runs the per-schema passes a conversion between two
// OAS 3.x versions needs.

package converter

import (
	"github.com/erraggy/oastools/parser"
)

// applyOAS3SchemaPasses converts a schema in place for a target OAS 3.x version.
// The caller owns the deep copy. Every pass keys on the target alone.
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

// prefixItemsForOAS30 rewrites a 2020-12 tuple into what OAS 3.0 can spell.
// prefixItemsToTuple produces the draft 4 array form, which is not legal OAS 3.0
// either, and tupleForOAS30 then collapses or drops it.
func prefixItemsForOAS30(c *Converter, schema *parser.Schema, result *ConversionResult, path string) {
	prefixItemsToTuple(c, schema, result, path, oas30Spelling)
	// Diagnostics name prefixItems, not the draft 4 form produced above.
	tupleForOAS30(c, schema, result, path, prefixItemsTuple)
}

// fixSchemaExclusiveMinMaxForOAS30 rewrites 2020-12's numeric exclusive bounds
// into the draft 4 pair: a boolean flag qualifying maximum or minimum. The
// counterpart of fixSchemaExclusiveMinMaxForOAS31.
//
// Where both bounds are present only the binding one survives, which is lossless:
// `x <= M and x < E` is `x < E` when E <= M and `x <= M` otherwise.
func fixSchemaExclusiveMinMaxForOAS30(schema *parser.Schema) {
	walkSchemas(schema, func(s *parser.Schema) {
		if e, ok := numericBound(s.ExclusiveMaximum); ok {
			if s.Maximum != nil && *s.Maximum < e {
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
// 4's spelling and is not one.
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
	case uint:
		return float64(n), true
	case uint64:
		// The YAML path yields this for a bound above math.MaxInt64.
		return float64(n), true
	default:
		return 0, false
	}
}
