// Multi-API Merge example demonstrating the joiner package.
//
// This example shows how to:
//   - Merge multiple microservice specifications into one
//   - Configure collision resolution strategies
//   - Enable semantic deduplication for shared schemas
//   - Write the merged output to a file
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/erraggy/oastools/joiner"
)

func main() {
	usersPath := findSpecPath("specs/users-api.yaml")
	ordersPath := findSpecPath("specs/orders-api.yaml")
	outputPath := filepath.Join(os.TempDir(), "merged-api.yaml")
	defer func() { _ = os.Remove(outputPath) }()

	fmt.Println("Multi-API Merge Workflow")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("Inputs:")
	fmt.Printf("  1. %s\n", filepath.Base(usersPath))
	fmt.Printf("  2. %s\n", filepath.Base(ordersPath))
	fmt.Println()

	// Step 1: Configure the joiner
	fmt.Println("[1/4] Configuration:")
	config := joiner.DefaultConfig()
	config.PathStrategy = joiner.StrategyFailOnPaths  // Fail on path conflicts
	config.SchemaStrategy = joiner.StrategyAcceptLeft // Keep first schema on collision
	config.SemanticDeduplication = true               // Merge identical schemas
	config.DeduplicateTags = true                     // Merge duplicate tags
	config.MergeArrays = true                         // Merge servers, security, tags

	fmt.Printf("      Path Strategy: %s\n", config.PathStrategy)
	fmt.Printf("      Schema Strategy: %s\n", config.SchemaStrategy)
	fmt.Printf("      Semantic Deduplication: %t\n", config.SemanticDeduplication)
	fmt.Printf("      Deduplicate Tags: %t\n", config.DeduplicateTags)
	fmt.Printf("      Merge Arrays: %t\n", config.MergeArrays)

	// Step 2: Join the specifications
	fmt.Println()
	fmt.Println("[2/4] Joining specifications...")
	j := joiner.New(config)
	result, err := j.Join([]string{usersPath, ordersPath})
	if err != nil {
		log.Fatalf("Join error: %v", err)
	}

	fmt.Printf("      Result Version: %s\n", result.Version)
	fmt.Printf("      Collisions Resolved: %d\n", result.CollisionCount)

	// Step 3: Show warnings if any
	fmt.Println()
	if len(result.Warnings) > 0 {
		fmt.Printf("[3/4] Warnings (%d):\n", len(result.Warnings))
		for _, w := range result.Warnings {
			fmt.Printf("      - %s\n", w)
		}
	} else {
		fmt.Println("[3/4] No warnings")
	}

	// Step 4: Write and summarize
	fmt.Println()
	fmt.Println("[4/4] Writing merged specification...")
	err = j.WriteResult(result, outputPath)
	if err != nil {
		log.Fatalf("Write error: %v", err)
	}
	fmt.Printf("      Output: %s\n", outputPath)

	// Summary of merged content
	fmt.Println()
	fmt.Println("--- Merged API Summary ---")

	// Access the merged document using DocumentAccessor for version-agnostic access
	parseResult := result.ToParseResult()
	accessor := parseResult.AsAccessor()
	if accessor == nil {
		log.Fatalf("Could not access merged document")
	}

	// Display Info
	info := accessor.GetInfo()
	if info != nil {
		fmt.Printf("Title: %s\n", info.Title)
		fmt.Printf("Version: %s\n", info.Version)
	}
	fmt.Println()

	// Servers are OAS3-specific, use OAS3Document() for this
	if doc, ok := parseResult.OAS3Document(); ok {
		fmt.Printf("Servers: %d\n", len(doc.Servers))
		for _, srv := range doc.Servers {
			fmt.Printf("  - %s\n", srv.URL)
		}
		fmt.Println()
	}

	// Display Tags using accessor
	tags := accessor.GetTags()
	fmt.Printf("Tags: %d\n", len(tags))
	for _, tag := range tags {
		fmt.Printf("  - %s\n", tag.Name)
	}
	fmt.Println()

	// Display Paths using accessor
	paths := accessor.GetPaths()
	fmt.Printf("Paths: %d\n", len(paths))
	for path := range paths {
		fmt.Printf("  - %s\n", path)
	}
	fmt.Println()

	// Display Schemas using accessor
	schemas := accessor.GetSchemas()
	if schemas != nil {
		fmt.Printf("Schemas: %d\n", len(schemas))
		for name := range schemas {
			fmt.Printf("  - %s\n", name)
		}
	} else {
		fmt.Printf("Schemas: 0\n")
	}

	fmt.Println()
	fmt.Println("---")
	fmt.Println("Merge completed successfully")
}

// findSpecPath locates a file relative to the source file location.
func findSpecPath(relativePath string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Unable to get current file path")
	}
	return filepath.Join(filepath.Dir(filename), relativePath)
}
