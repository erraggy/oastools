// API Statistics example demonstrating multi-handler analysis with walker.
//
// This example shows how to:
//   - Use multiple handlers to collect statistics in a single pass
//   - Access Info, Operation, Schema, Parameter, and Tag nodes
//   - Handle type as string or []string for OAS 3.1 compatibility
//   - Build statistics using closure state
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

// APIStats holds collected statistics about an OpenAPI specification.
type APIStats struct {
	Title   string
	Version string

	OperationsByMethod map[string]int
	SchemasByType      map[string]int
	ParametersByIn     map[string]int
	Tags               []string

	TotalOperations int
	TotalSchemas    int
	TotalParameters int
}

func main() {
	specPath := findSpecPath()

	fmt.Println("API Statistics Report")
	fmt.Println("=====================")
	fmt.Println()

	// Parse the specification
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath(specPath),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	// Initialize statistics
	stats := &APIStats{
		OperationsByMethod: make(map[string]int),
		SchemasByType:      make(map[string]int),
		ParametersByIn:     make(map[string]int),
	}

	// Walk the document with multiple handlers
	err = walker.Walk(parseResult,
		// Extract API info
		walker.WithInfoHandler(func(wc *walker.WalkContext, info *parser.Info) walker.Action {
			stats.Title = info.Title
			stats.Version = info.Version
			return walker.Continue
		}),

		// Count operations by HTTP method
		walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
			stats.TotalOperations++
			stats.OperationsByMethod[strings.ToUpper(wc.Method)]++
			return walker.Continue
		}),

		// Count schemas by type
		walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
			stats.TotalSchemas++
			// Handle type as string or []string (OAS 3.1 compatibility)
			switch t := schema.Type.(type) {
			case string:
				if t != "" {
					stats.SchemasByType[t]++
				}
			case []string:
				for _, typeName := range t {
					stats.SchemasByType[typeName]++
				}
			case []any:
				for _, v := range t {
					if typeName, ok := v.(string); ok {
						stats.SchemasByType[typeName]++
					}
				}
			}
			return walker.Continue
		}),

		// Count parameters by location
		walker.WithParameterHandler(func(wc *walker.WalkContext, param *parser.Parameter) walker.Action {
			stats.TotalParameters++
			if param.In != "" {
				stats.ParametersByIn[param.In]++
			}
			return walker.Continue
		}),

		// Collect tag names
		walker.WithTagHandler(func(wc *walker.WalkContext, tag *parser.Tag) walker.Action {
			stats.Tags = append(stats.Tags, tag.Name)
			return walker.Continue
		}),
	)
	if err != nil {
		log.Fatalf("Walk error: %v", err)
	}

	// Print the report
	printReport(stats)
}

func printReport(stats *APIStats) {
	fmt.Printf("API: %s v%s\n", stats.Title, stats.Version)
	fmt.Println()

	// Operations
	fmt.Printf("Operations (%d total):\n", stats.TotalOperations)
	methods := sortedKeys(stats.OperationsByMethod)
	for _, method := range methods {
		fmt.Printf("  %-8s %d\n", method+":", stats.OperationsByMethod[method])
	}
	fmt.Println()

	// Schemas by type
	fmt.Printf("Schemas by Type (%d total):\n", stats.TotalSchemas)
	types := sortedKeys(stats.SchemasByType)
	for _, t := range types {
		fmt.Printf("  %-10s %d\n", t+":", stats.SchemasByType[t])
	}
	fmt.Println()

	// Parameters by location
	fmt.Println("Parameters by Location:")
	locations := sortedKeys(stats.ParametersByIn)
	for _, loc := range locations {
		fmt.Printf("  %-10s %d\n", loc+":", stats.ParametersByIn[loc])
	}
	fmt.Println()

	// Tags
	fmt.Println("Tags:")
	if len(stats.Tags) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, tag := range stats.Tags {
			fmt.Printf("  - %s\n", tag)
		}
	}
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// findSpecPath locates the petstore-3.0.yaml file relative to the source file.
func findSpecPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Cannot determine source file location")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "testdata", "petstore-3.0.yaml")
}
