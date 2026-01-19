// Schema Deduplication example demonstrating joiner deduplication strategies.
//
// This example shows how to:
//   - Identify structurally identical schemas across documents
//   - Use deduplicate-equivalent for same-named collisions
//   - Use semantic deduplication for different-named equivalents
//   - Understand when each approach applies
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/erraggy/oastools/joiner"
	"github.com/erraggy/oastools/parser"
)

func main() {
	usersPath := findSpecPath("specs/users-api.yaml")
	productsPath := findSpecPath("specs/products-api.yaml")

	fmt.Println("Schema Deduplication Strategies")
	fmt.Println("================================")
	fmt.Println()
	fmt.Println("Scenario: Both APIs have error schemas with IDENTICAL structure")
	fmt.Println("  - users-api.yaml: UserError {code, message, details}")
	fmt.Println("  - products-api.yaml: ProductError {code, message, details}")
	fmt.Println()
	fmt.Println("These are structurally equivalent but have different names.")
	fmt.Println()

	// Demo 1: Without deduplication (baseline)
	fmt.Println("[1/3] Baseline: No deduplication")
	fmt.Println("-----------------------------------------")
	demonstrateNoDedup(usersPath, productsPath)

	// Demo 2: deduplicate-equivalent strategy
	fmt.Println()
	fmt.Println("[2/3] Strategy: deduplicate-equivalent")
	fmt.Println("-----------------------------------------")
	demonstrateDeduplicateEquivalent(usersPath, productsPath)

	// Demo 3: semantic deduplication
	fmt.Println()
	fmt.Println("[3/3] Strategy: semantic-deduplication")
	fmt.Println("-----------------------------------------")
	demonstrateSemanticDedup(usersPath, productsPath)

	fmt.Println()
	fmt.Println("=========================================")
	fmt.Println("Key Takeaway:")
	fmt.Println("  - deduplicate-equivalent: Merges SAME-named schemas if equivalent")
	fmt.Println("  - semantic-deduplication: Finds DIFFERENT-named equivalent schemas")
	fmt.Println("                            and consolidates to canonical name")
}

func demonstrateNoDedup(usersPath, productsPath string) {
	config := joiner.DefaultConfig()
	config.SchemaStrategy = joiner.StrategyAcceptLeft // Allow merge without error

	j := joiner.New(config)
	result, err := j.Join([]string{usersPath, productsPath})
	if err != nil {
		log.Printf("  Error: %v", err)
		return
	}

	accessor := result.ToParseResult().AsAccessor()
	if accessor == nil {
		log.Printf("  Could not access document")
		return
	}
	schemas := getSortedSchemaNames(accessor)

	fmt.Println("  Result: Success")
	fmt.Printf("  Schemas in merged doc: %v\n", schemas)
	fmt.Println()
	fmt.Println("  Note: Both UserError and ProductError exist in output.")
	fmt.Println("  This is wasteful since they're structurally identical!")
}

func demonstrateDeduplicateEquivalent(_, _ string) {
	// This demo is informational - our specs have different-named schemas
	fmt.Println("  This strategy handles SAME-named collisions.")
	fmt.Println("  When two schemas named 'Error' collide:")
	fmt.Println("    - If structurally equivalent -> keep one")
	fmt.Println("    - If different -> fail")
	fmt.Println()

	config := joiner.DefaultConfig()
	config.SchemaStrategy = joiner.StrategyDeduplicateEquivalent
	config.EquivalenceMode = "deep"

	fmt.Println("  Configuration:")
	fmt.Printf("    SchemaStrategy: %s\n", config.SchemaStrategy)
	fmt.Printf("    EquivalenceMode: %s\n", config.EquivalenceMode)
	fmt.Println()
	fmt.Println("  Use case: When teams independently define the same schema")
	fmt.Println("  with the same name - common with shared types like Error.")
}

func demonstrateSemanticDedup(usersPath, productsPath string) {
	config := joiner.DefaultConfig()
	config.SchemaStrategy = joiner.StrategyAcceptLeft
	config.SemanticDeduplication = true

	j := joiner.New(config)
	result, err := j.Join([]string{usersPath, productsPath})
	if err != nil {
		log.Printf("  Error: %v", err)
		return
	}

	accessor := result.ToParseResult().AsAccessor()
	if accessor == nil {
		log.Printf("  Could not access document")
		return
	}
	schemas := getSortedSchemaNames(accessor)

	fmt.Println("  Result: Success")
	fmt.Printf("  Schemas in merged doc: %v\n", schemas)
	fmt.Println()

	// Check if deduplication happened
	hasUserError := false
	hasProductError := false
	for _, name := range schemas {
		if name == "UserError" {
			hasUserError = true
		}
		if name == "ProductError" {
			hasProductError = true
		}
	}

	if hasUserError && !hasProductError {
		fmt.Println("  ProductError was deduplicated to UserError")
		fmt.Println("     (UserError < ProductError alphabetically)")
	} else if hasProductError && !hasUserError {
		fmt.Println("  UserError was deduplicated to ProductError")
		fmt.Println("     (ProductError < UserError alphabetically)")
	} else {
		fmt.Println("  Both schemas kept (no semantic equivalence detected)")
	}

	if len(result.Warnings) > 0 {
		fmt.Println()
		fmt.Println("  Warnings:")
		for _, w := range result.Warnings {
			fmt.Printf("    - %s\n", w)
		}
	}

	fmt.Println()
	fmt.Println("  Configuration:")
	fmt.Printf("    SemanticDeduplication: %t\n", config.SemanticDeduplication)
	fmt.Println()
	fmt.Println("  The joiner identified that UserError = ProductError")
	fmt.Println("  and consolidated them. All $refs are automatically rewritten.")
}

func getSortedSchemaNames(accessor parser.DocumentAccessor) []string {
	schemas := accessor.GetSchemas()
	if schemas == nil {
		return nil
	}
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func findSpecPath(relativePath string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Unable to get current file path")
	}
	return filepath.Join(filepath.Dir(filename), relativePath)
}
