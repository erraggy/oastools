// This file implements detection and fixing of duplicate operationId values.
// The OpenAPI specification requires operationId to be unique across all operations.
// This fixer detects duplicates and renames them using configurable templates.

package fixer

import (
	"fmt"
	"slices"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// operationLocation tracks where an operationId was first seen
type operationLocation struct {
	path   string
	method string
}

// fixDuplicateOperationIdsOAS2 fixes duplicate operationIds in an OAS 2.0 document.
func (f *Fixer) fixDuplicateOperationIdsOAS2(doc *parser.OAS2Document, result *FixResult) {
	if doc == nil || doc.Paths == nil {
		return
	}
	f.fixDuplicateOperationIds(doc.Paths, parser.OASVersion20, result)
}

// fixDuplicateOperationIdsOAS3 fixes duplicate operationIds in an OAS 3.x document.
// Per the OpenAPI spec, operationId must be unique across ALL operations in the document,
// including both paths and webhooks (OAS 3.1+).
func (f *Fixer) fixDuplicateOperationIdsOAS3(doc *parser.OAS3Document, result *FixResult) {
	if doc == nil {
		return
	}

	// Collect all path items (paths + webhooks) into a unified namespace
	// Per OAS spec: "The id MUST be unique among all operations described in the API"
	allPathItems := make(map[string]*parser.PathItem)
	pathTypes := make(map[string]string) // tracks whether key is from "paths" or "webhooks"

	// Add paths
	if doc.Paths != nil {
		for path, item := range doc.Paths {
			key := "paths:" + path
			allPathItems[key] = item
			pathTypes[key] = "paths"
		}
	}

	// Add webhooks (OAS 3.1+)
	if doc.OASVersion >= parser.OASVersion310 && len(doc.Webhooks) > 0 {
		for name, item := range doc.Webhooks {
			key := "webhooks:" + name
			allPathItems[key] = item
			pathTypes[key] = "webhooks"
		}
	}

	if len(allPathItems) == 0 {
		return
	}

	// Fix duplicates across the unified namespace
	f.fixDuplicateOperationIdsUnified(allPathItems, pathTypes, doc.OASVersion, result)
}

// fixDuplicateOperationIds is the shared implementation for fixing duplicates in paths (OAS 2.0).
func (f *Fixer) fixDuplicateOperationIds(paths parser.Paths, version parser.OASVersion, result *FixResult) {
	f.fixDuplicateOperationIdsInPathItems(paths, version, "paths", result)
}

// fixDuplicateOperationIdsUnified fixes duplicates across paths and webhooks in a unified namespace.
// This ensures operationId uniqueness across ALL operations per the OAS spec.
func (f *Fixer) fixDuplicateOperationIdsUnified(
	allPathItems map[string]*parser.PathItem,
	pathTypes map[string]string,
	version parser.OASVersion,
	result *FixResult,
) {
	if allPathItems == nil {
		return
	}

	seen := make(map[string]operationLocation)
	assigned := make(map[string]bool)

	// Sort keys for deterministic output
	sortedKeys := make([]string, 0, len(allPathItems))
	for key := range allPathItems {
		sortedKeys = append(sortedKeys, key)
	}
	slices.Sort(sortedKeys)

	// First pass: collect all existing operationIds
	for _, key := range sortedKeys {
		pathItem := allPathItems[key]
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, version)
		for _, op := range operations {
			if op != nil && op.OperationID != "" {
				assigned[op.OperationID] = true
			}
		}
	}

	// Second pass: find and fix duplicates
	for _, key := range sortedKeys {
		pathItem := allPathItems[key]
		if pathItem == nil {
			continue
		}

		// Extract the actual path/webhook name from the prefixed key
		pathType := pathTypes[key]
		actualPath := strings.TrimPrefix(key, pathType+":")

		operations := parser.GetOperations(pathItem, version)
		methods := getSortedMethods(operations)

		for _, method := range methods {
			op := operations[method]
			if op == nil || op.OperationID == "" {
				continue
			}

			opId := op.OperationID

			if loc, exists := seen[opId]; exists {
				// This is a duplicate - need to rename
				ctx := OperationContext{
					OperationId: opId,
					Method:      strings.ToLower(method),
					Path:        actualPath,
					Tags:        op.Tags,
				}

				newName := f.resolveOperationIdCollision(ctx, assigned)

				if !f.DryRun {
					op.OperationID = newName
				}

				fix := Fix{
					Type: FixTypeDuplicateOperationId,
					Path: fmt.Sprintf("%s.%s.%s.operationId", pathType, actualPath, method),
					Description: fmt.Sprintf(
						"renamed duplicate operationId %q to %q (first occurrence at %s %s)",
						opId, newName, strings.ToUpper(loc.method), loc.path,
					),
					Before: opId,
					After:  newName,
				}
				f.populateFixLocation(&fix)
				result.Fixes = append(result.Fixes, fix)

				assigned[newName] = true
			} else {
				// First occurrence - record it
				seen[opId] = operationLocation{
					path:   actualPath,
					method: method,
				}
			}
		}
	}
}

// fixDuplicateOperationIdsInPathItems is the unified implementation for fixing duplicate
// operationIds in a map of path items. The pathType parameter ("paths" or "webhooks")
// determines the JSON path prefix used in fix descriptions.
func (f *Fixer) fixDuplicateOperationIdsInPathItems(
	pathItems map[string]*parser.PathItem,
	version parser.OASVersion,
	pathType string,
	result *FixResult,
) {
	if pathItems == nil {
		return
	}

	// Initialize tracking maps and get sorted keys for deterministic output
	seen, assigned, sortedKeys := initOperationIdTracking(pathItems, version)

	// Find and fix duplicates
	for _, key := range sortedKeys {
		pathItem := pathItems[key]
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, version)
		methods := getSortedMethods(operations)

		for _, method := range methods {
			op := operations[method]
			if op == nil || op.OperationID == "" {
				continue
			}

			opId := op.OperationID

			if loc, exists := seen[opId]; exists {
				// This is a duplicate - need to rename
				ctx := OperationContext{
					OperationId: opId,
					Method:      strings.ToLower(method), // Normalize to lowercase
					Path:        key,
					Tags:        op.Tags,
				}

				newName := f.resolveOperationIdCollision(ctx, assigned)

				if !f.DryRun {
					op.OperationID = newName
				}

				fix := Fix{
					Type: FixTypeDuplicateOperationId,
					Path: fmt.Sprintf("%s.%s.%s.operationId", pathType, key, method),
					Description: fmt.Sprintf(
						"renamed duplicate operationId %q to %q (first occurrence at %s %s)",
						opId, newName, strings.ToUpper(loc.method), loc.path,
					),
					Before: opId,
					After:  newName,
				}
				f.populateFixLocation(&fix)
				result.Fixes = append(result.Fixes, fix)

				assigned[newName] = true
			} else {
				// First occurrence - record it
				seen[opId] = operationLocation{
					path:   key,
					method: method,
				}
			}
		}
	}
}

// initOperationIdTracking initializes the tracking maps for duplicate operationId detection.
// It returns:
//   - seen: map tracking operationId -> first location (for duplicate detection)
//   - assigned: map tracking all operationIds (for collision avoidance when renaming)
//   - sortedKeys: sorted keys from pathItems for deterministic processing order
func initOperationIdTracking(
	pathItems map[string]*parser.PathItem,
	version parser.OASVersion,
) (seen map[string]operationLocation, assigned map[string]bool, sortedKeys []string) {
	seen = make(map[string]operationLocation)
	assigned = make(map[string]bool)

	// Sort keys for deterministic output
	sortedKeys = make([]string, 0, len(pathItems))
	for key := range pathItems {
		sortedKeys = append(sortedKeys, key)
	}
	slices.Sort(sortedKeys)

	// First pass: collect all existing operationIds into the assigned map
	for _, key := range sortedKeys {
		pathItem := pathItems[key]
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, version)
		methods := getSortedMethods(operations)

		for _, method := range methods {
			op := operations[method]
			if op == nil || op.OperationID == "" {
				continue
			}
			assigned[op.OperationID] = true
		}
	}

	return seen, assigned, sortedKeys
}

// resolveOperationIdCollision generates a unique operationId using the template.
// If the template still produces a collision, appends a numeric suffix.
// Includes a maximum iteration guard to prevent infinite loops.
func (f *Fixer) resolveOperationIdCollision(ctx OperationContext, assigned map[string]bool) string {
	config := f.OperationIdNamingConfig
	const maxIterations = 10000

	// Start with n=2 since the original (n=1) is already taken
	for n := 2; n <= maxIterations; n++ {
		candidate := expandOperationIdTemplate(config.Template, ctx, n, config)

		// If template doesn't include {n} and we get a collision, need to force numeric suffix
		if !strings.Contains(config.Template, "{n}") && n > 2 {
			candidate = fmt.Sprintf("%s%d", candidate, n)
		}

		if !assigned[candidate] {
			return candidate
		}
	}

	// Fallback: use operationId + maxIterations to guarantee uniqueness
	return fmt.Sprintf("%s%d", ctx.OperationId, maxIterations)
}

// getSortedMethods returns the methods from an operations map in a deterministic order.
// Standard HTTP methods come first in a fixed order, followed by any custom methods sorted alphabetically.
func getSortedMethods(operations map[string]*parser.Operation) []string {
	// Standard method order for consistency
	// Note: "query" is an OAS 3.2+ method
	standardOrder := []string{"get", "put", "post", "delete", "options", "head", "patch", "trace", "query"}

	var methods []string
	seen := make(map[string]bool)

	// Add standard methods in order (if they exist)
	for _, method := range standardOrder {
		if op, exists := operations[method]; exists && op != nil {
			methods = append(methods, method)
			seen[method] = true
		}
	}

	// Collect any additional/custom methods
	var customMethods []string
	for method, op := range operations {
		if op != nil && !seen[method] {
			customMethods = append(customMethods, method)
		}
	}

	// Sort custom methods alphabetically
	slices.Sort(customMethods)
	methods = append(methods, customMethods...)

	return methods
}
