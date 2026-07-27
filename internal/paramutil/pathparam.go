// pathparam.go holds predicates over a single parameter that both the validator
// and the fixer apply, so the defect one reports is exactly the defect the other
// repairs and the two cannot drift apart.

package paramutil

import "github.com/erraggy/oastools/parser"

// NeedsRequiredTrue reports whether param is an in: path parameter that omits
// required: true. Both OAS 2.0 and 3.x demand the field be present and true for
// path parameters and permit no other value, so a parameter this returns true
// for is invalid in every version and repairable without judgment or loss.
//
// A parameter carrying a $ref is skipped. References are preserved verbatim so
// documents round-trip losslessly, which leaves every sibling field — Required
// included — empty on the reference itself; the defect, if there is one, belongs
// to the definition it names. Callers check the reusable definitions in their own
// right, so a use site skipped here is still covered exactly once.
func NeedsRequiredTrue(param *parser.Parameter) bool {
	return param != nil && !param.Required && param.Ref == "" && param.In == parser.ParamInPath
}
