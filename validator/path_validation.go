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

// validateParameterRefKinds reports parameter $ref values that point at a
// component which exists but is not a parameter definition — a schema, say,
// referenced from a parameter slot.
//
// Nothing else catches this. buildOAS2ValidRefs and buildOAS3ValidRefs collect
// every component path into one flat set, so validateRef sees a schema ref in a
// parameter position as perfectly valid. The parameter resolver keys only
// parameter definitions, so the same ref is unresolvable, which suppresses the
// undeclared-parameter check. Between them the document passes clean, so this
// check is what keeps a wrong-kind reference from becoming a silent failure.
//
// Two cases are deliberately left alone:
//   - A ref whose target is absent entirely is dangling, and validateRef
//     already reports it. Reporting it again here would duplicate the defect.
//   - A ref that does name a parameter but fails to resolve has a break further
//     down its chain, which is a different defect from being the wrong kind.
//     Defines, not Resolve, is what draws that line.
//
// Added while addressing #374 - Root-level parameters are not applied to
// Operations contained within
func (v *Validator) validateParameterRefKinds(
	params []*parser.Parameter,
	prefix string,
	resolver paramutil.Resolver,
	validRefs map[string]bool,
	result *ValidationResult,
	baseURL string,
) {
	for i, param := range params {
		if param == nil || param.Ref == "" || resolver.Defines(param.Ref) || !validRefs[param.Ref] {
			continue
		}
		v.addError(result, prefix+".parameters["+strconv.Itoa(i)+"]",
			fmt.Sprintf("$ref '%s' resolves to a component that is not a parameter definition", param.Ref),
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
