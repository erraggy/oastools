package fixer

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/erraggy/oastools/parser"
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

// fixMissingPathParametersOAS2 adds missing path parameters to an OAS 2.0 document.
// Fixes are applied in sorted order (by path, method, parameter name) for deterministic output.
func (f *Fixer) fixMissingPathParametersOAS2(doc *parser.OAS2Document, result *FixResult) {
	if doc.Paths == nil {
		return
	}

	// Sort path patterns for deterministic order
	pathPatterns := make([]string, 0, len(doc.Paths))
	for pathPattern := range doc.Paths {
		pathPatterns = append(pathPatterns, pathPattern)
	}
	sort.Strings(pathPatterns)

	for _, pathPattern := range pathPatterns {
		pathItem := doc.Paths[pathPattern]
		if pathItem == nil {
			continue
		}

		// Extract parameters from path template
		pathParams := extractPathParameters(pathPattern)
		if len(pathParams) == 0 {
			continue
		}

		// Get operations for this path
		operations := parser.GetOperations(pathItem, parser.OASVersion20)

		// Sort methods for deterministic order
		methods := make([]string, 0, len(operations))
		for method := range operations {
			methods = append(methods, method)
		}
		sort.Strings(methods)

		for _, method := range methods {
			op := operations[method]
			if op == nil {
				continue
			}

			// Collect declared path parameters from PathItem and Operation
			declaredParams := make(map[string]bool)

			// PathItem-level parameters
			for _, param := range pathItem.Parameters {
				if param != nil && param.In == parser.ParamInPath {
					declaredParams[param.Name] = true
				}
			}

			// Operation-level parameters (override PathItem params)
			for _, param := range op.Parameters {
				if param != nil && param.In == parser.ParamInPath {
					declaredParams[param.Name] = true
				}
			}

			// Sort parameter names for deterministic order
			paramNames := make([]string, 0, len(pathParams))
			for paramName := range pathParams {
				paramNames = append(paramNames, paramName)
			}
			sort.Strings(paramNames)

			// Find missing parameters
			for _, paramName := range paramNames {
				if declaredParams[paramName] {
					continue
				}

				// Create the missing parameter
				paramType := "string"
				paramFormat := ""
				if f.InferTypes {
					paramType, paramFormat = inferParameterType(paramName)
				}

				newParam := &parser.Parameter{
					Name:     paramName,
					In:       parser.ParamInPath,
					Required: true, // Path parameters are always required
					Type:     paramType,
				}
				if paramFormat != "" {
					newParam.Format = paramFormat
				}

				// Add to operation parameters
				op.Parameters = append(op.Parameters, newParam)

				// Record the fix
				jsonPath := fmt.Sprintf("paths.%s.%s.parameters", pathPattern, method)
				description := fmt.Sprintf("Added missing path parameter '%s' (type: %s", paramName, paramType)
				if paramFormat != "" {
					description += fmt.Sprintf(", format: %s", paramFormat)
				}
				description += ")"

				result.Fixes = append(result.Fixes, Fix{
					Type:        FixTypeMissingPathParameter,
					Path:        jsonPath,
					Description: description,
					Before:      nil,
					After:       newParam,
				})
			}
		}
	}
}

// fixMissingPathParametersOAS3 adds missing path parameters to an OAS 3.x document.
// Fixes are applied in sorted order (by path, method, parameter name) for deterministic output.
func (f *Fixer) fixMissingPathParametersOAS3(doc *parser.OAS3Document, result *FixResult) {
	if doc.Paths == nil {
		return
	}

	// Sort path patterns for deterministic order
	pathPatterns := make([]string, 0, len(doc.Paths))
	for pathPattern := range doc.Paths {
		pathPatterns = append(pathPatterns, pathPattern)
	}
	sort.Strings(pathPatterns)

	for _, pathPattern := range pathPatterns {
		pathItem := doc.Paths[pathPattern]
		if pathItem == nil {
			continue
		}

		// Extract parameters from path template
		pathParams := extractPathParameters(pathPattern)
		if len(pathParams) == 0 {
			continue
		}

		// Get operations for this path
		operations := parser.GetOperations(pathItem, doc.OASVersion)

		// Sort methods for deterministic order
		methods := make([]string, 0, len(operations))
		for method := range operations {
			methods = append(methods, method)
		}
		sort.Strings(methods)

		for _, method := range methods {
			op := operations[method]
			if op == nil {
				continue
			}

			// Collect declared path parameters from PathItem and Operation
			declaredParams := make(map[string]bool)

			// PathItem-level parameters
			for _, param := range pathItem.Parameters {
				if param != nil && param.In == parser.ParamInPath {
					declaredParams[param.Name] = true
				}
			}

			// Operation-level parameters (override PathItem params)
			for _, param := range op.Parameters {
				if param != nil && param.In == parser.ParamInPath {
					declaredParams[param.Name] = true
				}
			}

			// Sort parameter names for deterministic order
			paramNames := make([]string, 0, len(pathParams))
			for paramName := range pathParams {
				paramNames = append(paramNames, paramName)
			}
			sort.Strings(paramNames)

			// Find missing parameters
			for _, paramName := range paramNames {
				if declaredParams[paramName] {
					continue
				}

				// Create the missing parameter
				paramType := "string"
				paramFormat := ""
				if f.InferTypes {
					paramType, paramFormat = inferParameterType(paramName)
				}

				// OAS 3.x uses Schema for type definition
				schema := &parser.Schema{
					Type: paramType,
				}
				if paramFormat != "" {
					schema.Format = paramFormat
				}

				newParam := &parser.Parameter{
					Name:     paramName,
					In:       parser.ParamInPath,
					Required: true, // Path parameters are always required
					Schema:   schema,
				}

				// Add to operation parameters
				op.Parameters = append(op.Parameters, newParam)

				// Record the fix
				jsonPath := fmt.Sprintf("paths.%s.%s.parameters", pathPattern, method)
				description := fmt.Sprintf("Added missing path parameter '%s' (type: %s", paramName, paramType)
				if paramFormat != "" {
					description += fmt.Sprintf(", format: %s", paramFormat)
				}
				description += ")"

				result.Fixes = append(result.Fixes, Fix{
					Type:        FixTypeMissingPathParameter,
					Path:        jsonPath,
					Description: description,
					Before:      nil,
					After:       newParam,
				})
			}
		}
	}
}
