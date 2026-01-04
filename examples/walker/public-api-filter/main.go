// Public API Filter example demonstrating subtree filtering with SkipChildren.
//
// This example shows how to:
//   - Use SkipChildren for subtree filtering
//   - Maintain context across handler calls
//   - Build filtered subsets of API documents
//   - Combine multiple filtering criteria
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
)

// FilterResult holds the results of filtering the API specification.
type FilterResult struct {
	IncludedPaths      []string
	IncludedOperations []OperationInfo
	SkippedPaths       []string // Internal/admin paths
	DeprecatedOps      []string // Deprecated operations
}

// OperationInfo holds details about an included operation.
type OperationInfo struct {
	Method      string
	Path        string
	OperationID string
	Summary     string
}

func main() {
	specPath := findSpecPath()

	fmt.Println("Public API Extraction Report")
	fmt.Println("============================")
	fmt.Println()

	// Parse the specification
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath(specPath),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	// Initialize filter result
	result := &FilterResult{
		IncludedPaths:      make([]string, 0),
		IncludedOperations: make([]OperationInfo, 0),
		SkippedPaths:       make([]string, 0),
		DeprecatedOps:      make([]string, 0),
	}

	// Track state for the operation handler
	var currentPath string
	var skipCurrentPath bool

	// Walk the document with filtering handlers
	err = walker.Walk(parseResult,
		// Filter paths based on prefix
		walker.WithPathHandler(func(wc *walker.WalkContext, pathItem *parser.PathItem) walker.Action {
			// Check if this is an internal/admin path
			if isInternalPath(wc.PathTemplate) {
				result.SkippedPaths = append(result.SkippedPaths, wc.PathTemplate)
				skipCurrentPath = true
				return walker.SkipChildren
			}

			// Include this path
			result.IncludedPaths = append(result.IncludedPaths, wc.PathTemplate)
			currentPath = wc.PathTemplate
			skipCurrentPath = false
			return walker.Continue
		}),

		// Process operations on included paths
		walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
			// Skip if the current path was filtered out
			if skipCurrentPath {
				return walker.SkipChildren
			}

			// Check if operation is deprecated
			if op.Deprecated {
				result.DeprecatedOps = append(result.DeprecatedOps,
					fmt.Sprintf("%s %s", strings.ToUpper(wc.Method), currentPath))
				return walker.SkipChildren
			}

			// Include this operation
			result.IncludedOperations = append(result.IncludedOperations, OperationInfo{
				Method:      strings.ToUpper(wc.Method),
				Path:        currentPath,
				OperationID: op.OperationID,
				Summary:     op.Summary,
			})
			return walker.Continue
		}),
	)
	if err != nil {
		log.Fatalf("Walk error: %v", err)
	}

	// Print the report
	printReport(result)
}

// isInternalPath checks if a path should be filtered out as internal/admin.
func isInternalPath(path string) bool {
	return strings.HasPrefix(path, "/internal") ||
		strings.HasPrefix(path, "/_") ||
		strings.HasPrefix(path, "/admin")
}

func printReport(result *FilterResult) {
	// Sort paths for consistent output
	sort.Strings(result.IncludedPaths)
	sort.Strings(result.SkippedPaths)
	sort.Strings(result.DeprecatedOps)

	// Sort operations by path then method
	sort.Slice(result.IncludedOperations, func(i, j int) bool {
		if result.IncludedOperations[i].Path != result.IncludedOperations[j].Path {
			return result.IncludedOperations[i].Path < result.IncludedOperations[j].Path
		}
		return result.IncludedOperations[i].Method < result.IncludedOperations[j].Method
	})

	// Included Paths
	fmt.Printf("Included Paths (%d):\n", len(result.IncludedPaths))
	for _, p := range result.IncludedPaths {
		fmt.Printf("  %s\n", p)
	}
	fmt.Println()

	// Public Operations
	fmt.Printf("Public Operations (%d):\n", len(result.IncludedOperations))
	for _, op := range result.IncludedOperations {
		fmt.Printf("  %-6s %-20s - %s: %s\n", op.Method, op.Path, op.OperationID, op.Summary)
	}
	fmt.Println()

	// Filtered Out section
	fmt.Println("Filtered Out:")

	// Internal/Admin paths
	fmt.Printf("  Internal/Admin paths skipped (%d):\n", len(result.SkippedPaths))
	for _, p := range result.SkippedPaths {
		fmt.Printf("    - %s\n", p)
	}
	fmt.Println()

	// Deprecated operations
	fmt.Printf("  Deprecated operations skipped (%d):\n", len(result.DeprecatedOps))
	for _, op := range result.DeprecatedOps {
		fmt.Printf("    - %s\n", op)
	}
}

// findSpecPath locates the full-api.yaml file relative to the source file.
func findSpecPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Cannot determine source file location")
	}
	return filepath.Join(filepath.Dir(filename), "specs", "full-api.yaml")
}
