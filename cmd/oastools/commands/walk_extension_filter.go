package commands

import (
	"fmt"
	"slices"
	"strings"
)

// ExtensionFilter represents a parsed --extension filter.
// Groups are OR'd together; expressions within a group are AND'd.
type ExtensionFilter struct {
	Groups [][]ExtensionExpr
}

// ExtensionExpr is a single extension filter expression.
type ExtensionExpr struct {
	Key     string  // e.g., "x-audited-by"
	Value   *string // nil = existence check only
	Negated bool    // ! prefix or != operator
}

// ParseExtensionFilter parses the --extension flag value.
// Grammar: FILTER = EXPR ( ("," | "+") EXPR )*
// , = OR (separates groups), + = AND (within a group)
func ParseExtensionFilter(input string) (ExtensionFilter, error) {
	if input == "" {
		return ExtensionFilter{}, fmt.Errorf("empty extension filter")
	}

	var filter ExtensionFilter

	// Split by , for OR groups
	for orPart := range strings.SplitSeq(input, ",") {
		if orPart == "" {
			return ExtensionFilter{}, fmt.Errorf("empty expression in extension filter")
		}

		// Split by + for AND expressions within a group
		var group []ExtensionExpr

		for part := range strings.SplitSeq(orPart, "+") {
			expr, err := parseExtensionExpr(part)
			if err != nil {
				return ExtensionFilter{}, err
			}
			group = append(group, expr)
		}

		filter.Groups = append(filter.Groups, group)
	}

	return filter, nil
}

// parseExtensionExpr parses a single expression like "x-foo", "x-foo=bar", "!x-foo", "x-foo!=bar".
func parseExtensionExpr(s string) (ExtensionExpr, error) {
	if s == "" {
		return ExtensionExpr{}, fmt.Errorf("empty expression in extension filter")
	}

	var expr ExtensionExpr

	// Check for ! prefix (negation)
	if s[0] == '!' {
		expr.Negated = true
		s = s[1:]
		if s == "" {
			return ExtensionExpr{}, fmt.Errorf("bare '!' is not a valid extension filter expression")
		}
	}

	// Check for != operator
	if idx := strings.Index(s, "!="); idx > 0 {
		if expr.Negated {
			return ExtensionExpr{}, fmt.Errorf("ambiguous double negation in %q: use !x-key (negated existence) or x-key!=val (negated value), not both", "!"+s)
		}
		expr.Key = s[:idx]
		val := s[idx+2:]
		expr.Value = &val
		expr.Negated = true
	} else if idx := strings.Index(s, "="); idx > 0 {
		// Check for = operator
		expr.Key = s[:idx]
		val := s[idx+1:]
		expr.Value = &val
	} else {
		// Existence check only
		expr.Key = s
	}

	// Validate key starts with x-
	if !strings.HasPrefix(expr.Key, "x-") {
		return ExtensionExpr{}, fmt.Errorf("invalid extension key %q: must start with \"x-\"", expr.Key)
	}

	return expr, nil
}

// Match evaluates the filter against a node's extensions.
// Returns true if the filter matches.
func (f ExtensionFilter) Match(extensions map[string]any) bool {
	// OR across groups: any group matching is sufficient
	for _, group := range f.Groups {
		if matchGroup(group, extensions) {
			return true
		}
	}
	return false
}

// matchGroup evaluates AND logic: all expressions must match.
func matchGroup(group []ExtensionExpr, extensions map[string]any) bool {
	for _, expr := range group {
		if !matchExpr(expr, extensions) {
			return false
		}
	}
	return true
}

// matchExpr evaluates a single expression against extensions.
func matchExpr(expr ExtensionExpr, extensions map[string]any) bool {
	val, exists := extensions[expr.Key]

	if expr.Value == nil {
		// Existence check
		if expr.Negated {
			return !exists
		}
		return exists
	}

	// Value comparison
	valStr := fmt.Sprintf("%v", val)
	matches := exists && valStr == *expr.Value

	if expr.Negated {
		return !matches
	}
	return matches
}

// FormatExtensions formats a map of extensions as a comma-separated string for summary output.
// Keys are sorted for deterministic output.
func FormatExtensions(extra map[string]any) string {
	if len(extra) == 0 {
		return ""
	}
	var parts []string
	for k, v := range extra {
		if strings.HasPrefix(k, "x-") {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	slices.Sort(parts)
	return strings.Join(parts, ", ")
}
