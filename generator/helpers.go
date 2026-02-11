package generator

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/parser"
	"golang.org/x/tools/imports"
)

// isRequired checks if a property name is in the required list.
func isRequired(required []string, name string) bool {
	for _, r := range required {
		if r == name {
			return true
		}
	}
	return false
}

// buildDefaultUserAgent generates the default User-Agent string for generated clients.
// Format: oastools/{version}/generated/{title}
// If title is empty, it uses "API Client" as a fallback.
func buildDefaultUserAgent(info *parser.Info) string {
	version := oastools.Version()
	title := "API Client"
	if info != nil && info.Title != "" {
		title = info.Title
	}
	return fmt.Sprintf("oastools/%s/generated/%s", version, title)
}

// formatAndFixImports formats Go source code and automatically fixes imports.
// It adds missing imports and removes unused ones using goimports-equivalent processing.
// This ensures generated code is immediately compilable without requiring users to run goimports.
//
//nolint:unparam // filename kept for clarity and future flexibility, even though currently always "generated.go"
func formatAndFixImports(filename string, src []byte) ([]byte, error) {
	return imports.Process(filename, src, nil)
}

// isSelfReference checks if a schema property references its parent type.
// This is used to detect recursive type definitions that need pointer indirection.
// It handles both direct $ref and allOf compositions.
func isSelfReference(propSchema *parser.Schema, parentTypeName string) bool {
	if propSchema == nil {
		return false
	}

	// Check direct $ref
	if propSchema.Ref != "" {
		parts := strings.Split(propSchema.Ref, "/")
		if len(parts) > 0 {
			refTypeName := toTypeName(parts[len(parts)-1])
			if refTypeName == parentTypeName {
				return true
			}
		}
	}

	// Check allOf compositions for self-reference
	for _, subSchema := range propSchema.AllOf {
		if isSelfReference(subSchema, parentTypeName) {
			return true
		}
	}

	return false
}

// buildTypeGroupMaps builds the shared types set and per-group types map from a split plan.
// This is shared between OAS 2.0 and OAS 3.x generators for split type generation.
func buildTypeGroupMaps(splitPlan *SplitPlan) (sharedTypes map[string]bool, groupTypes map[string]map[string]bool) {
	sharedTypes = make(map[string]bool)
	for _, typeName := range splitPlan.SharedTypes {
		sharedTypes[typeName] = true
	}

	groupTypes = make(map[string]map[string]bool)
	for _, group := range splitPlan.Groups {
		if group.IsShared {
			continue
		}
		groupTypes[group.Name] = make(map[string]bool)
		for _, typeName := range group.Types {
			groupTypes[group.Name][typeName] = true
		}
	}

	return sharedTypes, groupTypes
}
