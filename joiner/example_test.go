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
	// Create a temporary output file path
	outputPath := filepath.Join(os.TempDir(), "joined-example.yaml")
	defer func() { _ = os.Remove(outputPath) }()

	// Use default configuration
	config := joiner.DefaultConfig()
	j := joiner.New(config)

	// Join two specification files
	result, err := j.Join([]string{
		"../testdata/join-base-3.0.yaml",
		"../testdata/join-extension-3.0.yaml",
	})
	if err != nil {
		log.Fatalf("failed to join: %v", err)
	}

	// Write the result
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

	// Configure custom strategies
	config := joiner.JoinerConfig{
		DefaultStrategy:   joiner.StrategyFailOnCollision,
		PathStrategy:      joiner.StrategyFailOnPaths, // Fail on path collisions
		SchemaStrategy:    joiner.StrategyAcceptLeft,  // Keep first schema definition
		ComponentStrategy: joiner.StrategyAcceptRight, // Keep last component definition
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

// Example_acceptLeft demonstrates using accept-left strategy to prefer the first document's values.
func Example_acceptLeft() {
	outputPath := filepath.Join(os.TempDir(), "joined-left.yaml")
	defer func() { _ = os.Remove(outputPath) }()

	config := joiner.DefaultConfig()
	// Accept values from the first (left) document when collisions occur
	config.PathStrategy = joiner.StrategyAcceptLeft
	config.SchemaStrategy = joiner.StrategyAcceptLeft
	config.ComponentStrategy = joiner.StrategyAcceptLeft

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

	fmt.Printf("Strategy: accept-left\n")
	fmt.Printf("Version: %s\n", result.Version)

	// Output:
	// Strategy: accept-left
	// Version: 3.0.3
}

// Example_acceptRight demonstrates using accept-right strategy to prefer the last document's values.
func Example_acceptRight() {
	outputPath := filepath.Join(os.TempDir(), "joined-right.yaml")
	defer func() { _ = os.Remove(outputPath) }()

	config := joiner.DefaultConfig()
	// Accept values from the last (right) document when collisions occur (overwrite)
	config.PathStrategy = joiner.StrategyAcceptRight
	config.SchemaStrategy = joiner.StrategyAcceptRight
	config.ComponentStrategy = joiner.StrategyAcceptRight

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

	fmt.Printf("Strategy: accept-right\n")
	fmt.Printf("Version: %s\n", result.Version)

	// Output:
	// Strategy: accept-right
	// Version: 3.0.3
}

// Example_multipleDocuments demonstrates joining more than two documents.
func Example_multipleDocuments() {
	outputPath := filepath.Join(os.TempDir(), "joined-multiple.yaml")
	defer func() { _ = os.Remove(outputPath) }()

	config := joiner.DefaultConfig()
	config.SchemaStrategy = joiner.StrategyAcceptLeft

	j := joiner.New(config)
	result, err := j.Join([]string{
		"../testdata/join-base-3.0.yaml",
		"../testdata/join-extension-3.0.yaml",
		"../testdata/join-additional-3.0.yaml",
	})
	if err != nil {
		log.Fatalf("failed to join: %v", err)
	}

	err = j.WriteResult(result, outputPath)
	if err != nil {
		log.Fatalf("failed to write result: %v", err)
	}

	fmt.Printf("Joined 3 documents\n")
	fmt.Printf("Version: %s\n", result.Version)

	// Output:
	// Joined 3 documents
	// Version: 3.0.3
}

// Example_withWarnings demonstrates handling warnings during the join process.
func Example_withWarnings() {
	outputPath := filepath.Join(os.TempDir(), "joined-warnings.yaml")
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

	// Check for warnings
	if len(result.Warnings) > 0 {
		fmt.Printf("Warnings: %d\n", len(result.Warnings))
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	} else {
		fmt.Println("No warnings")
	}

	err = j.WriteResult(result, outputPath)
	if err != nil {
		log.Fatalf("failed to write result: %v", err)
	}

	// Output:
	// No warnings
}

// Example_oas2 demonstrates joining OpenAPI 2.0 (Swagger) documents.
func Example_oas2() {
	outputPath := filepath.Join(os.TempDir(), "joined-oas2.yaml")
	defer func() { _ = os.Remove(outputPath) }()

	config := joiner.DefaultConfig()
	config.SchemaStrategy = joiner.StrategyAcceptLeft

	j := joiner.New(config)
	result, err := j.Join([]string{
		"../testdata/join-base-2.0.yaml",
		"../testdata/join-extension-2.0.yaml",
	})
	if err != nil {
		log.Fatalf("failed to join: %v", err)
	}

	err = j.WriteResult(result, outputPath)
	if err != nil {
		log.Fatalf("failed to write result: %v", err)
	}

	fmt.Printf("OAS Version: %s\n", result.Version)

	// Output:
	// OAS Version: 2.0
}

// Example_arrayMerging demonstrates array merging behavior.
func Example_arrayMerging() {
	outputPath := filepath.Join(os.TempDir(), "joined-arrays.yaml")
	defer func() { _ = os.Remove(outputPath) }()

	// Enable array merging (default)
	config := joiner.DefaultConfig()
	config.MergeArrays = true
	config.SchemaStrategy = joiner.StrategyAcceptLeft

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

	fmt.Println("Arrays merged successfully")

	// Output:
	// Arrays merged successfully
}

// Example_tagDeduplication demonstrates tag deduplication behavior.
func Example_tagDeduplication() {
	outputPath := filepath.Join(os.TempDir(), "joined-tags.yaml")
	defer func() { _ = os.Remove(outputPath) }()

	// Enable tag deduplication (default)
	config := joiner.DefaultConfig()
	config.DeduplicateTags = true
	config.SchemaStrategy = joiner.StrategyAcceptLeft

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

	fmt.Println("Tags deduplicated successfully")

	// Output:
	// Tags deduplicated successfully
}

// Example_defaultConfig demonstrates using the default configuration.
func Example_defaultConfig() {
	config := joiner.DefaultConfig()

	fmt.Printf("Default Strategy: %s\n", config.DefaultStrategy)
	fmt.Printf("Path Strategy: %s\n", config.PathStrategy)
	fmt.Printf("Schema Strategy: %s\n", config.SchemaStrategy)
	fmt.Printf("Component Strategy: %s\n", config.ComponentStrategy)
	fmt.Printf("Merge Arrays: %v\n", config.MergeArrays)
	fmt.Printf("Deduplicate Tags: %v\n", config.DeduplicateTags)

	// Output:
	// Default Strategy: fail
	// Path Strategy: fail
	// Schema Strategy: accept-left
	// Component Strategy: accept-left
	// Merge Arrays: true
	// Deduplicate Tags: true
}

// Example_collisionError demonstrates handling collision errors.
func Example_collisionError() {
	outputPath := filepath.Join(os.TempDir(), "joined-collision.yaml")
	defer func() { _ = os.Remove(outputPath) }()

	// Use fail strategy to detect collisions
	config := joiner.DefaultConfig()
	config.PathStrategy = joiner.StrategyFailOnCollision
	config.SchemaStrategy = joiner.StrategyFailOnCollision

	j := joiner.New(config)

	// This would fail if there were actual collisions
	result, err := j.Join([]string{
		"../testdata/join-base-3.0.yaml",
		"../testdata/join-extension-3.0.yaml",
	})
	if err != nil {
		// Handle collision error
		fmt.Printf("Join error: collision detected\n")
		return
	}

	err = j.WriteResult(result, outputPath)
	if err != nil {
		log.Fatalf("failed to write result: %v", err)
	}

	fmt.Println("No collisions detected")

	// Output:
	// No collisions detected
}
