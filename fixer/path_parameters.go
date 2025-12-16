package fixer

import (
	"regexp"
	"strings"
)

// pathParamRegex matches path template parameters like {paramName}
var pathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// extractPathParameters extracts parameter names from a path template.
// e.g., "/pets/{petId}/owners/{ownerId}" -> {"petId": true, "ownerId": true}
func extractPathParameters(pathPattern string) map[string]bool {
	params := make(map[string]bool)
	matches := pathParamRegex.FindAllStringSubmatch(pathPattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params[match[1]] = true
		}
	}
	return params
}

// inferParameterType returns the inferred type and format for a parameter name.
// Returns (type, format) where format may be empty.
//
// Inference rules:
//   - Names ending in "id", "Id", or "ID" -> ("integer", "")
//   - Names containing "uuid" or "guid" (case-insensitive) -> ("string", "uuid")
//   - All other names -> ("string", "")
func inferParameterType(paramName string) (string, string) {
	nameLower := strings.ToLower(paramName)

	// Check for UUID/GUID pattern
	if strings.Contains(nameLower, "uuid") || strings.Contains(nameLower, "guid") {
		return "string", "uuid"
	}

	// Check for ID suffix (case-sensitive patterns)
	if strings.HasSuffix(paramName, "id") ||
		strings.HasSuffix(paramName, "Id") ||
		strings.HasSuffix(paramName, "ID") {
		return "integer", ""
	}

	// Default to string
	return "string", ""
}
