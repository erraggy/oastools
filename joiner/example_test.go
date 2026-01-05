package joiner_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/erraggy/oastools/joiner"
)

// Example demonstrates basic usage of the joiner to combine two OpenAPI specifications.
func Example() {
	outputPath := filepath.Join(os.TempDir(), "joined-example.yaml")
	defer func() { _ = os.Remove(outputPath) }()
	config := joiner.DefaultConfig()
	j := joiner.New(config)
	result, err := j.Join([]string{
		"../testdata/join-base-3.0.yaml",
		"../testdata/join-extension-3.0.yaml",
	})
	if err != nil {
		log.Fatalf("failed to join: %v", err)
	}
	err = j.WriteResult(result, outputPath)
	if err != nil {
		log.Fatalf("failed to write result: %v", err)
	}
	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Warnings: %d\n", len(result.Warnings))
	// Output:
	// Version: 3.0.3
	// Warnings: 0
}

// Example_customStrategies demonstrates using custom collision strategies for different component types.
func Example_customStrategies() {
	outputPath := filepath.Join(os.TempDir(), "joined-custom.yaml")
	defer func() { _ = os.Remove(outputPath) }()
	config := joiner.JoinerConfig{
		DefaultStrategy:   joiner.StrategyFailOnCollision,
		PathStrategy:      joiner.StrategyFailOnPaths,
		SchemaStrategy:    joiner.StrategyAcceptLeft,
		ComponentStrategy: joiner.StrategyAcceptRight,
		DeduplicateTags:   true,
		MergeArrays:       true,
	}
	j := joiner.New(config)
	result, err := j.Join([]string{
		"../testdata/join-base-3.0.yaml",
		"../testdata/join-extension-3.0.yaml",
	})
	if err != nil {
		log.Fatalf("failed to join: %v", err)
	}
	err = j.WriteResult(result, outputPath)
	if err != nil {
		log.Fatalf("failed to write result: %v", err)
	}
	fmt.Printf("Joined successfully\n")
	fmt.Printf("Collisions resolved: %d\n", result.CollisionCount)
	// Output:
	// Joined successfully
	// Collisions resolved: 0
}

// Example_semanticDeduplication demonstrates automatic consolidation of identical schemas
// across multiple OpenAPI documents. When documents share structurally identical schemas
// (even if named differently), semantic deduplication identifies these duplicates and
// consolidates them to a single canonical schema.
func Example_semanticDeduplication() {
	outputPath := filepath.Join(os.TempDir(), "joined-dedup.yaml")
	defer func() { _ = os.Remove(outputPath) }()

	// Enable semantic deduplication in the joiner configuration
	config := joiner.JoinerConfig{
		DefaultStrategy:       joiner.StrategyAcceptLeft,
		SemanticDeduplication: true,   // Enable schema deduplication
		EquivalenceMode:       "deep", // Use deep structural comparison
		DeduplicateTags:       true,
		MergeArrays:           true,
	}

	j := joiner.New(config)
	result, err := j.Join([]string{
		"../testdata/join-base-3.0.yaml",
		"../testdata/join-extension-3.0.yaml",
	})
	if err != nil {
		log.Fatalf("failed to join: %v", err)
	}

	err = j.WriteResult(result, outputPath)
	if err != nil {
		log.Fatalf("failed to write result: %v", err)
	}

	// Semantic deduplication identifies structurally equivalent schemas
	// across documents and consolidates them, reducing duplication in the
	// merged output. The alphabetically-first name becomes canonical.
	fmt.Printf("Joined successfully\n")
	fmt.Printf("Version: %s\n", result.Version)
	// Output:
	// Joined successfully
	// Version: 3.0.3
}

// Example_operationContext demonstrates operation-aware schema renaming.
// When OperationContext is enabled, the joiner traces schemas back to their
// originating operations, allowing rename templates to use fields like
// OperationID, Path, Method, and Tags.
func Example_operationContext() {
	// Join two specs that both define a "Response" schema.
	// Without operation context, collisions are resolved using Source/Index.
	// With operation context, we can use operationId for meaningful names.
	result, err := joiner.JoinWithOptions(
		joiner.WithFilePaths(
			"../testdata/join-operation-context-users-3.0.yaml",
			"../testdata/join-operation-context-orders-3.0.yaml",
		),
		joiner.WithOperationContext(true),
		joiner.WithSchemaStrategy(joiner.StrategyRenameRight),
		// Template uses OperationID with PascalCase conversion
		joiner.WithRenameTemplate("{{.OperationID | pascalCase}}{{.Name}}"),
	)
	if err != nil {
		log.Fatalf("failed to join: %v", err)
	}

	// The colliding "Response" schema from the orders API is renamed using
	// the operationId "createOrder" -> "CreateOrderResponse"
	fmt.Printf("Collisions resolved: %d\n", result.CollisionCount)
	fmt.Printf("Version: %s\n", result.Version)
	// Output:
	// Collisions resolved: 1
	// Version: 3.0.0
}

// Example_templateFunctions demonstrates the template functions available
// for operation-aware renaming: pathResource, pathClean, case conversions,
// and fallback functions like default and coalesce.
func Example_templateFunctions() {
	// Template using pathResource to extract "orders" from "/orders"
	// and pascalCase to convert it to "Orders"
	result, err := joiner.JoinWithOptions(
		joiner.WithFilePaths(
			"../testdata/join-operation-context-users-3.0.yaml",
			"../testdata/join-operation-context-orders-3.0.yaml",
		),
		joiner.WithOperationContext(true),
		joiner.WithSchemaStrategy(joiner.StrategyRenameRight),
		// pathResource extracts the first path segment (e.g., "/orders" -> "orders")
		// pascalCase converts to PascalCase (e.g., "orders" -> "Orders")
		joiner.WithRenameTemplate("{{pathResource .Path | pascalCase}}{{.Name}}"),
	)
	if err != nil {
		log.Fatalf("failed to join: %v", err)
	}

	// The colliding "Response" schema from /orders becomes "OrdersResponse"
	fmt.Printf("Collisions resolved: %d\n", result.CollisionCount)
	fmt.Printf("Version: %s\n", result.Version)
	// Output:
	// Collisions resolved: 1
	// Version: 3.0.0
}

// Example_primaryOperationPolicy demonstrates how to select which operation
// provides the primary context when a schema is referenced by multiple operations.
// PolicyMostSpecific prefers operations with operationId over those without.
func Example_primaryOperationPolicy() {
	// When a schema is referenced by multiple operations, the policy determines
	// which operation's context is used for the primary fields (Path, Method,
	// OperationID). PolicyMostSpecific chooses the operation with the richest
	// metadata: first preferring those with operationId, then those with tags.
	result, err := joiner.JoinWithOptions(
		joiner.WithFilePaths(
			"../testdata/join-operation-context-users-3.0.yaml",
			"../testdata/join-operation-context-orders-3.0.yaml",
		),
		joiner.WithOperationContext(true),
		joiner.WithPrimaryOperationPolicy(joiner.PolicyMostSpecific),
		joiner.WithSchemaStrategy(joiner.StrategyRenameRight),
		// Use operationId if available; fallback to path resource
		joiner.WithRenameTemplate("{{coalesce .OperationID (pathResource .Path) .Source | pascalCase}}{{.Name}}"),
	)
	if err != nil {
		log.Fatalf("failed to join: %v", err)
	}

	// With PolicyMostSpecific, operations with operationId are preferred
	fmt.Printf("Collisions resolved: %d\n", result.CollisionCount)
	fmt.Printf("Version: %s\n", result.Version)
	// Output:
	// Collisions resolved: 1
	// Version: 3.0.0
}
