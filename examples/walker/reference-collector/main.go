// Reference Collector example demonstrating schema reference analysis and cycle detection.
//
// This example shows how to:
//   - Use SchemaSkippedHandler for cycle and depth notifications
//   - Build reference graphs with walker
//   - Identify unused components
//   - Configure WithMaxSchemaDepth for schema traversal
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

// RefCollector tracks schema references, cycles, and depth-limited schemas.
type RefCollector struct {
	SchemaRefs     map[string][]string // schema name -> list of JSON paths where referenced
	Cycles         []string            // paths where cycles detected
	DepthLimited   []string            // paths where depth limit hit
	SelfReferences []string            // schemas that reference themselves (circular)
}

// extractSchemaName extracts the schema name from a $ref string.
// For example, "#/components/schemas/Pet" returns "Pet".
func extractSchemaName(ref string) string {
	const prefix = "#/components/schemas/"
	if strings.HasPrefix(ref, prefix) {
		return strings.TrimPrefix(ref, prefix)
	}
	return ""
}

func main() {
	specPath := findSpecPath()

	fmt.Println("Reference Analysis Report")
	fmt.Println("=========================")
	fmt.Println()

	// Parse the specification with ref resolution enabled (default)
	// Note: When refs are resolved, $ref fields are replaced with the target schema,
	// so we need to handle both $ref strings AND check Items for nested refs
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath(specPath),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	// Initialize reference collector
	collector := &RefCollector{
		SchemaRefs: make(map[string][]string),
	}

	// Walk the document with handlers for reference tracking
	err = walker.Walk(parseResult,
		// Set max depth for schema traversal
		walker.WithMaxSchemaDepth(50),

		// Track schema references
		walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
			// Track direct $ref
			if schema.Ref != "" {
				schemaName := extractSchemaName(schema.Ref)
				if schemaName != "" {
					collector.SchemaRefs[schemaName] = append(collector.SchemaRefs[schemaName], wc.JSONPath)
				}
			}

			// Handle items that contain a $ref (stored as map[string]any)
			if items, ok := schema.Items.(map[string]any); ok {
				if ref, ok := items["$ref"].(string); ok {
					schemaName := extractSchemaName(ref)
					if schemaName != "" {
						collector.SchemaRefs[schemaName] = append(collector.SchemaRefs[schemaName], wc.JSONPath+".items")
					}
				}
			}

			return walker.Continue
		}),

		// Track skipped schemas (cycles and depth limits)
		walker.WithSchemaSkippedHandler(func(wc *walker.WalkContext, reason string, schema *parser.Schema) {
			switch reason {
			case "cycle":
				collector.Cycles = append(collector.Cycles, wc.JSONPath)
			case "depth":
				collector.DepthLimited = append(collector.DepthLimited, wc.JSONPath)
			}
		}),
	)
	if err != nil {
		log.Fatalf("Walk error: %v", err)
	}

	// Get all schema names from the document to find unused ones
	allSchemas := getAllSchemaNames(parseResult)
	unusedSchemas := findUnusedSchemas(allSchemas, collector.SchemaRefs)

	// Detect self-referencing schemas (circular references)
	for name, refs := range collector.SchemaRefs {
		prefix := "$.components.schemas['" + name + "']"
		for _, refPath := range refs {
			if strings.HasPrefix(refPath, prefix) {
				collector.SelfReferences = append(collector.SelfReferences, refPath)
			}
		}
	}
	sort.Strings(collector.SelfReferences)

	// Print the report
	printReport(collector, unusedSchemas)
}

// getAllSchemaNames returns all schema names defined in components.schemas.
func getAllSchemaNames(result *parser.ParseResult) []string {
	doc, ok := result.Document.(*parser.OAS3Document)
	if !ok || doc.Components == nil {
		return nil
	}

	names := make([]string, 0, len(doc.Components.Schemas))
	for name := range doc.Components.Schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// findUnusedSchemas returns schema names that have no references.
func findUnusedSchemas(allSchemas []string, refs map[string][]string) []string {
	var unused []string
	for _, name := range allSchemas {
		if _, hasRefs := refs[name]; !hasRefs {
			unused = append(unused, name)
		}
	}
	return unused
}

func printReport(collector *RefCollector, unusedSchemas []string) {
	// Schema references
	fmt.Println("Schema References:")
	schemaNames := sortedKeys(collector.SchemaRefs)
	for _, name := range schemaNames {
		refs := collector.SchemaRefs[name]
		fmt.Printf("  %s (%d reference%s):\n", name, len(refs), plural(len(refs)))
		for _, ref := range refs {
			fmt.Printf("    - %s\n", ref)
		}
	}
	fmt.Println()

	// Unused schemas
	fmt.Printf("Unused Schemas (%d):\n", len(unusedSchemas))
	if len(unusedSchemas) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, name := range unusedSchemas {
			fmt.Printf("  - %s\n", name)
		}
	}
	fmt.Println()

	// Self-referencing schemas (circular)
	fmt.Printf("Self-Referencing Schemas (%d):\n", len(collector.SelfReferences))
	if len(collector.SelfReferences) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, path := range collector.SelfReferences {
			fmt.Printf("  - %s\n", path)
		}
	}
	fmt.Println()

	// Walker-detected cycles (from SchemaSkippedHandler)
	fmt.Printf("Walker Cycle Events (%d):\n", len(collector.Cycles))
	if len(collector.Cycles) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, path := range collector.Cycles {
			fmt.Printf("  - %s\n", path)
		}
	}
	fmt.Println()

	// Depth-limited schemas
	fmt.Printf("Depth-Limited Schemas: %d\n", len(collector.DepthLimited))
	if len(collector.DepthLimited) > 0 {
		for _, path := range collector.DepthLimited {
			fmt.Printf("  - %s\n", path)
		}
	}
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// findSpecPath locates the complex-api.yaml file relative to the source file.
func findSpecPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Cannot determine source file location")
	}
	return filepath.Join(filepath.Dir(filename), "specs", "complex-api.yaml")
}
