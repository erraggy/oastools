// schema_oas3_to_oas3.go runs the per-schema passes a conversion between two
// OAS 3.x versions needs.

package converter

import (
	"fmt"

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
		c.fixSchemaExclusiveMinMaxForOAS30(schema, result, path)
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
func (c *Converter) fixSchemaExclusiveMinMaxForOAS30(schema *parser.Schema, result *ConversionResult, path string) {
	walkSchemas(schema, func(s *parser.Schema) {
		if e, ok, exact := numericBound(s.ExclusiveMaximum); ok {
			source := s.ExclusiveMaximum
			if s.Maximum != nil && *s.Maximum < e {
				s.ExclusiveMaximum = nil
			} else {
				bound := e
				s.Maximum = &bound
				s.ExclusiveMaximum = true
				c.reportInexactBound(exact, fieldExclusiveMaximum, source, e, result, path)
			}
		}
		if e, ok, exact := numericBound(s.ExclusiveMinimum); ok {
			source := s.ExclusiveMinimum
			if s.Minimum != nil && *s.Minimum > e {
				s.ExclusiveMinimum = nil
			} else {
				bound := e
				s.Minimum = &bound
				s.ExclusiveMinimum = true
				c.reportInexactBound(exact, fieldExclusiveMinimum, source, e, result, path)
			}
		}
	})
}

// reportInexactBound notes a bound that the float64 the OAS 2.0 and 3.0 models
// use cannot hold. The converted document states a different boundary from the
// source, so it is a change in meaning rather than a rounding detail.
func (c *Converter) reportInexactBound(exact bool, field string, source any, converted float64, result *ConversionResult, path string) {
	if exact || result == nil {
		return
	}
	c.addIssueWithContext(result, path,
		fmt.Sprintf("Bound '%s' is %v, which the target's number model cannot hold exactly; recorded as %.0f", field, source, converted),
		"OAS 2.0 and OAS 3.0 state a numeric bound as a JSON number, which is read as a float64 here, and integers above 2^53 are not all representable. Use a bound the target can hold exactly, or keep the document at OAS 3.1 or later")
}

// numericBound reports the value of a 2020-12 exclusive bound, and whether the
// float64 that the OAS 2.0 and 3.0 models use holds it exactly. A bool is draft
// 4's spelling and is not a numeric bound.
//
// Exactness is not a formality above 2^53: float64(math.MaxUint64) is one
// greater than math.MaxUint64, so a bound converted without a word would reject
// a value the source accepted.
func numericBound(v any) (value float64, ok, exact bool) {
	switch n := v.(type) {
	case float64:
		return n, true, true
	case float32:
		return float64(n), true, true
	case int:
		return float64(n), true, exactInt64(int64(n))
	case int64:
		return float64(n), true, exactInt64(n)
	case uint:
		return float64(n), true, exactUint64(uint64(n))
	case uint64:
		// The YAML path yields this for a bound above math.MaxInt64.
		return float64(n), true, exactUint64(n)
	default:
		return 0, false, false
	}
}

// exactUint64 reports whether float64 holds n exactly. The magnitude guard
// matters because converting a float64 at or above 2^64 back to uint64 is not
// defined, and math.MaxUint64 rounds to exactly 2^64.
func exactUint64(n uint64) bool {
	f := float64(n)
	if f >= twoTo64 {
		return false
	}
	return uint64(f) == n
}

func exactInt64(n int64) bool {
	f := float64(n)
	if f >= twoTo63 || f < -twoTo63 {
		return false
	}
	return int64(f) == n
}

const (
	twoTo63 = float64(1 << 63)
	twoTo64 = twoTo63 * 2
)
