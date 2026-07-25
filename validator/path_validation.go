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
//   - A cycle and an over-long chain consist entirely of valid references, so
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
