// This file implements path template validation and parameter consistency checks
// for OpenAPI path items and operations.

package validator

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/internal/pathutil"
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
		v.addWarning(result, fmt.Sprintf("paths.%s", pathPattern),
			"Path has trailing slash, which is discouraged by REST best practices",
			withSpecRef(fmt.Sprintf("%s#paths-object", baseURL)),
			withValue(pathPattern),
		)
	}
}

// extractPathParameters extracts parameter names from a path template
// e.g., "/pets/{petId}/owners/{ownerId}" -> {"petId": true, "ownerId": true}
func extractPathParameters(pathPattern string) map[string]bool {
	params := make(map[string]bool)
	matches := pathutil.PathParamRegex.FindAllStringSubmatch(pathPattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params[match[1]] = true
		}
	}
	return params
}
