// Vendor Extensions example demonstrating document mutation with walker.
//
// This example shows how to:
//   - Mutate documents in-place through pointer receivers
//   - Add vendor extensions (x-*) to schemas, operations, and paths
//   - Use SkipChildren to exclude deprecated operations
//   - Apply conditional mutations based on node properties
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"go.yaml.in/yaml/v4"
)

// MutationStats tracks modifications made during the walk.
type MutationStats struct {
	SchemasProcessed    int
	OperationsEnhanced  int
	OperationsSkipped   int
	PathsMarkedInternal int
}

func main() {
	specPath := findSpecPath()

	fmt.Println("Vendor Extensions Processor")
	fmt.Println("===========================")
	fmt.Println()

	// Parse the specification
	result, err := parser.ParseWithOptions(
		parser.WithFilePath(specPath),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	// Fixed timestamp for reproducible output
	timestamp := "2024-01-15T10:30:00Z"

	// Initialize statistics
	stats := &MutationStats{}

	// Walk the document with mutation handlers
	err = walker.Walk(result,
		// Add vendor extensions to ALL schemas
		walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
			if schema.Extra == nil {
				schema.Extra = make(map[string]any)
			}
			schema.Extra["x-processed"] = true
			schema.Extra["x-processed-at"] = timestamp
			stats.SchemasProcessed++
			return walker.Continue
		}),

		// Add rate limiting based on HTTP method; skip deprecated operations
		walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
			// Skip deprecated operations - don't enhance them
			if op.Deprecated {
				stats.OperationsSkipped++
				return walker.SkipChildren
			}

			if op.Extra == nil {
				op.Extra = make(map[string]any)
			}

			// Apply rate limits based on HTTP method
			switch strings.ToUpper(method) {
			case "GET":
				op.Extra["x-rate-limit"] = 100
				op.Extra["x-cache-ttl"] = 60
			case "POST", "PUT", "PATCH":
				op.Extra["x-rate-limit"] = 20
			case "DELETE":
				op.Extra["x-rate-limit"] = 10
			}

			stats.OperationsEnhanced++
			return walker.Continue
		}),

		// Mark internal paths
		walker.WithPathHandler(func(pathTemplate string, pi *parser.PathItem, path string) walker.Action {
			// Mark paths starting with /admin or /_ as internal
			if strings.HasPrefix(pathTemplate, "/admin") || strings.HasPrefix(pathTemplate, "/_") {
				if pi.Extra == nil {
					pi.Extra = make(map[string]any)
				}
				pi.Extra["x-internal"] = true
				stats.PathsMarkedInternal++
			}
			return walker.Continue
		}),
	)
	if err != nil {
		log.Fatalf("Walk error: %v", err)
	}

	// Output the modified document as YAML
	output, err := yaml.Marshal(result.Document)
	if err != nil {
		log.Fatalf("Marshal error: %v", err)
	}

	fmt.Println("Modified Specification:")
	fmt.Println("-----------------------")
	fmt.Println(string(output))

	// Print summary
	fmt.Println()
	fmt.Println("Modification Summary")
	fmt.Println("--------------------")
	fmt.Printf("Schemas processed:     %d\n", stats.SchemasProcessed)
	fmt.Printf("Operations enhanced:   %d (with rate limits)\n", stats.OperationsEnhanced)
	fmt.Printf("Operations skipped:    %d (deprecated)\n", stats.OperationsSkipped)
	fmt.Printf("Paths marked internal: %d\n", stats.PathsMarkedInternal)
}

// findSpecPath locates the petstore-3.0.yaml file relative to the source file.
func findSpecPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Cannot determine source file location")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "testdata", "petstore-3.0.yaml")
}
