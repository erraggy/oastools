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
