// This file implements path template validation and parameter consistency checks
// for OpenAPI path items and operations.

package validator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/internal/paramutil"
	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/parser"
)

// validatePathTemplate validates that a path template is well-formed
// Returns an error if the template is malformed (unclosed braces, empty parameters, etc.)
func validatePathTemplate(pathPattern string) error {
	// Check for empty braces explicitly (regex won't catch {})
	if strings.Contains(pathPattern, "{}") {
		return fmt.Errorf("empty parameter name in path template")
	}

	// Check for consecutive slashes
	if strings.Contains(pathPattern, "//") {
		return fmt.Errorf("path contains consecutive slashes")
	}

	// Check for reserved characters (fragment identifier and query string)
	if strings.Contains(pathPattern, "#") {
		return fmt.Errorf("path contains reserved character '#'")
	}
	if strings.Contains(pathPattern, "?") {
		return fmt.Errorf("path contains reserved character '?'")
	}

	// Note: Trailing slashes are handled separately as warnings, not errors
	// Empty segments in the middle are caught by the consecutive slash check above

	// Check for unclosed or unopened braces
	openCount := 0
	for i, ch := range pathPattern {
		switch ch {
		case '{':
			openCount++
			if openCount > 1 {
				return fmt.Errorf("nested braces are not allowed at position %d", i)
			}
		case '}':
			openCount--
			if openCount < 0 {
				return fmt.Errorf("unexpected closing brace at position %d", i)
			}
		}
	}
	if openCount != 0 {
		return fmt.Errorf("unclosed brace in path template")
	}

	// Check for empty or invalid parameters, and track duplicates
	paramNames := make(map[string]bool)
	matches := pathutil.PathParamRegex.FindAllStringSubmatch(pathPattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			paramName := match[1]
			if strings.TrimSpace(paramName) == "" {
				return fmt.Errorf("empty parameter name in path template")
			}
			// Check for invalid characters in parameter name
			if strings.Contains(paramName, "{") || strings.Contains(paramName, "}") {
				return fmt.Errorf("invalid parameter name '%s' contains braces", paramName)
			}
			// Check for duplicate parameter names
			if paramNames[paramName] {
				return fmt.Errorf("duplicate parameter name '%s' in path template", paramName)
			}
			paramNames[paramName] = true
		}
	}

	return nil
}

// checkTrailingSlash adds a warning if the path has a trailing slash
// Trailing slashes are discouraged by REST best practices but not forbidden by OAS spec
func checkTrailingSlash(v *Validator, pathPattern string, result *ValidationResult, baseURL string) {
	if v.IncludeWarnings && len(pathPattern) > 1 && strings.HasSuffix(pathPattern, "/") {
		v.addWarning(result, "paths."+pathPattern,
			"Path has trailing slash, which is discouraged by REST best practices",
			withSpecRef(fmt.Sprintf("%s#paths-object", baseURL)),
			withValue(pathPattern),
		)
	}
}

// validateParameterRefs reports parameter $ref values that cannot be followed
// to a parameter definition.
//
// The path-parameter consistency check suppresses its undeclared-parameter
// error whenever a reference fails to resolve, because an unresolvable
// reference may itself declare the missing name. That suppression is only sound
// if every reason a reference fails is reported somewhere, and for three of the
// four reasons nothing else reports it:
//
//   - A wrong-kind ref passes reference validation, because buildOAS2ValidRefs
//     and buildOAS3ValidRefs collect every component path into one flat set and
//     so cannot tell which kind belongs in a parameter slot.
//   - A cycle and an too-long chain consist entirely of valid references, so
//     reference validation has nothing to object to.
//
// Without this check each of those makes a broken document validate clean.
//
// Two cases are deliberately left alone: a dangling ref, already reported by
// validateRef, which would otherwise be reported twice; and an external ref,
// which is the one genuinely unknowable case.
//
// Added while addressing #374 - Root-level parameters are not applied to
// Operations contained within
func (v *Validator) validateParameterRefs(
	params []*parser.Parameter,
	prefix string,
	resolver paramutil.Resolver,
	validRefs map[string]bool,
	result *ValidationResult,
	baseURL string,
) {
	for i, param := range params {
		if param == nil || param.Ref == "" {
			continue
		}

		var message string
		switch _, reason := resolver.Classify(param); reason {
		case paramutil.ReasonNotAParameter:
			// The chain may break at a later hop than this one. Only this ref's
			// own kind is in question here: if it does name a parameter, the
			// break is downstream and is reported against the definition that
			// carries it, not against every reference that reaches it.
			if resolver.Defines(param.Ref) {
				continue
			}
			// Absent entirely means dangling, which validateRef reports.
			if !validRefs[param.Ref] {
				continue
			}
			message = fmt.Sprintf("$ref '%s' resolves to a component that is not a parameter definition", param.Ref)
		case paramutil.ReasonCycle:
			message = fmt.Sprintf("$ref '%s' leads to a reference cycle between parameter definitions", param.Ref)
		case paramutil.ReasonTooDeep:
			message = fmt.Sprintf("$ref '%s' leads to a parameter reference chain too long to follow", param.Ref)
		case paramutil.ReasonResolved, paramutil.ReasonExternal:
			continue
		}

		v.addError(result, prefix+".parameters["+strconv.Itoa(i)+"]", message,
			withSpecRef(fmt.Sprintf("%s#parameter-object", baseURL)),
			withField("$ref"),
			withValue(param.Ref),
		)
	}
}

// validateParameterDefinitionRefs reports reusable parameter definitions whose
// own $ref names a component that exists but is not a parameter.
//
// validateParameterRefs deliberately stops at the first hop: when a use-site
// $ref names a parameter, any break further down belongs to the definition
// carrying it. This is the check that makes that possible. Without it such an
// alias is invisible from every angle: reference validation accepts the target
// because it exists, the use site defers to a definition-site check, and the
// path-parameter check suppresses itself because the chain did not resolve.
//
// Cycles and too-long chains are deliberately not reported here. They are
// already reported once at the use site, and reporting them per definition
// would emit one error for every link in the chain. The gap that leaves is an
// unreferenced cyclic definition, which no operation can be affected by.
func (v *Validator) validateParameterDefinitionRefs(
	defs map[string]*parser.Parameter,
	prefix string,
	resolver paramutil.Resolver,
	validRefs map[string]bool,
	result *ValidationResult,
	baseURL string,
) {
	for name, param := range defs {
		// Only this definition's own target is in question. A ref naming
		// another parameter is that parameter's business; one naming nothing at
		// all is dangling, which validateRef reports.
		if param == nil || param.Ref == "" || resolver.Defines(param.Ref) || !validRefs[param.Ref] {
			continue
		}
		v.addError(result, prefix+"."+name,
			fmt.Sprintf("$ref '%s' resolves to a component that is not a parameter definition", param.Ref),
			withSpecRef(fmt.Sprintf("%s#parameter-object", baseURL)),
			withField("$ref"),
			withValue(param.Ref),
		)
	}
}

// validatePathParamsRequired reports inline path parameters that omit
// required: true. The rule is identical in OAS 2.0 and 3.x — Swagger 2.0 states
// that required "MUST be true" for in: path — and the check reads only fields
// both versions share, so both validators call this.
//
// Referenced parameters are deliberately skipped: a ref into the reusable
// definitions has its definition checked once by validateOAS2Parameters or
// validateOAS3Components, so checking it again at each use site would report the
// same defect repeatedly.
//
// The skip is unconditional, so a ref that does not land in those definitions
// — external, or dangling — has its required field checked nowhere. Those refs
// are reported as unresolvable in their own right, which is the more useful
// diagnostic anyway.
//
// Callers must invoke this once per path item rather than inside their
// operation loop, or a path-item parameter is reported once per operation.
//
// The defect test itself lives in [paramutil.NeedsRequiredTrue], shared with the
// fixer so what is reported here is exactly what fixer.FixTypePathParameterNotRequired
// repairs.
func (v *Validator) validatePathParamsRequired(params []*parser.Parameter, prefix string, result *ValidationResult, baseURL string) {
	for i, param := range params {
		if !paramutil.NeedsRequiredTrue(param) {
			continue
		}
		v.addError(result, prefix+".parameters["+strconv.Itoa(i)+"]",
			"Path parameters must have required: true",
			withSpecRef(fmt.Sprintf("%s#parameter-object", baseURL)),
			withField("required"),
		)
	}
}

// pathParamDeclarations records the path parameters a path item declares and
// those one of its operations declares, kept apart so each is reported where it
// is declared while the template check still sees their union.
type pathParamDeclarations struct {
	item, operation map[string]bool
	// complete is false when some $ref in either list could not be resolved, so
	// the union is only a lower bound. See [paramutil.Resolver.DeclaredPathParams].
	complete bool
}

// declaresPathParams pairs an already-resolved path item declaration set with
// an operation's. The item's set is resolved by the caller outside its
// operation loop, so a path item with several operations resolves its own
// parameters once rather than once per operation.
func declaresPathParams(
	resolver paramutil.Resolver,
	item map[string]bool,
	itemComplete bool,
	opParams []*parser.Parameter,
) pathParamDeclarations {
	operation, opComplete := resolver.DeclaredPathParams(opParams)
	return pathParamDeclarations{
		item:      item,
		operation: operation,
		complete:  itemComplete && opComplete,
	}
}

// operationOnly returns the param names the operation declares and the path item does
// not, so a name declared at both levels is not warned about twice.
func (d pathParamDeclarations) operationOnly() map[string]bool {
	only := make(map[string]bool, len(d.operation))
	for name := range d.operation {
		if !d.item[name] {
			only[name] = true
		}
	}
	return only
}

// reportUndeclaredPathParams reports path template variables that no parameter
// declares, for either the path item or the operation.
//
// Skipped entirely when some $ref could not be resolved, because an
// unresolvable reference may itself declare the missing name — reporting it
// would be the false positive this check exists to avoid.
//
// Suppressing is only safe because every reason a $ref fails to resolve is
// reported elsewhere: a dangling local ref by reference validation, and a
// wrong-kind ref, cycle, or too-long chain by validateParameterRefs and
// validateParameterDefinitionRefs. External refs are genuinely unknowable and
// are the one case that stays silent by design.
func (v *Validator) reportUndeclaredPathParams(
	decls pathParamDeclarations,
	pathParams map[string]bool,
	opPath string,
	result *ValidationResult,
	baseURL string,
) {
	if !decls.complete {
		return
	}
	for paramName := range pathParams {
		if decls.item[paramName] || decls.operation[paramName] {
			continue
		}
		v.addError(result, opPath,
			fmt.Sprintf("Path template references parameter '{%s}' but it is not declared in parameters", paramName),
			withSpecRef(fmt.Sprintf("%s#path-item-object", baseURL)),
			withValue(paramName),
		)
	}
}

// extractPathParameters extracts parameter names from a path template
// e.g., "/pets/{petId}/owners/{ownerId}" -> {"petId": true, "ownerId": true}
func extractPathParameters(pathPattern string) map[string]bool {
	params := make(map[string]bool)
	s := pathPattern
	for {
		open := strings.IndexByte(s, '{')
		if open < 0 {
			break
		}
		close := strings.IndexByte(s[open+1:], '}')
		if close < 0 {
			break
		}
		name := s[open+1 : open+1+close]
		if len(name) > 0 {
			params[name] = true
		}
		s = s[open+1+close+1:]
	}
	return params
}
