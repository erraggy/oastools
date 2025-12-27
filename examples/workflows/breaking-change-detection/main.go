// Breaking Change Detection example demonstrating the differ package.
//
// This example shows how to:
//   - Compare two API versions to detect changes
//   - Identify breaking changes by severity
//   - Use diff results for CI/CD gate checks
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/erraggy/oastools/differ"
)

func main() {
	v1Path := findSpecPath("specs/v1.yaml")
	v2Path := findSpecPath("specs/v2.yaml")

	fmt.Println("Breaking Change Detection Workflow")
	fmt.Println("===================================")
	fmt.Println()
	fmt.Println("Comparing:")
	fmt.Printf("  Source (old): %s\n", filepath.Base(v1Path))
	fmt.Printf("  Target (new): %s\n", filepath.Base(v2Path))
	fmt.Println()

	// Step 1: Perform diff in breaking mode
	fmt.Println("[1/3] Analyzing changes...")
	result, err := differ.DiffWithOptions(
		differ.WithSourceFilePath(v1Path),
		differ.WithTargetFilePath(v2Path),
		differ.WithMode(differ.ModeBreaking),
		differ.WithIncludeInfo(true),
	)
	if err != nil {
		log.Fatalf("Diff error: %v", err)
	}

	fmt.Printf("      Source Version: %s\n", result.SourceVersion)
	fmt.Printf("      Target Version: %s\n", result.TargetVersion)
	fmt.Printf("      Total Changes: %d\n", len(result.Changes))

	// Step 2: Summary by severity
	fmt.Println()
	fmt.Println("[2/3] Change Summary:")
	fmt.Printf("      Breaking (Error+Critical): %d\n", result.BreakingCount)
	fmt.Printf("      Warnings: %d\n", result.WarningCount)
	fmt.Printf("      Info: %d\n", result.InfoCount)

	// Step 3: Detailed breakdown by category
	fmt.Println()
	fmt.Println("[3/3] Detailed Changes:")

	// Group by category for readability
	categories := make(map[differ.ChangeCategory][]differ.Change)
	for _, change := range result.Changes {
		categories[change.Category] = append(categories[change.Category], change)
	}

	for category, changes := range categories {
		fmt.Println()
		fmt.Printf("      %s:\n", category)
		for _, c := range changes {
			icon := severityIcon(c.Severity)
			fmt.Printf("        %s %s\n", icon, c.Message)
			if c.Path != "" {
				fmt.Printf("             at %s\n", c.Path)
			}
		}
	}

	// CI/CD guidance
	fmt.Println()
	fmt.Println("---")
	if result.HasBreakingChanges {
		fmt.Printf("BREAKING CHANGES DETECTED: %d\n", result.BreakingCount)
		fmt.Println()
		fmt.Println("Recommendations:")
		fmt.Println("  - Consider incrementing major version")
		fmt.Println("  - Update API documentation")
		fmt.Println("  - Notify API consumers")
		os.Exit(1) // Fail CI
	} else {
		fmt.Println("No breaking changes detected - safe to deploy")
	}
}

func severityIcon(s differ.Severity) string {
	switch s {
	case differ.SeverityCritical:
		return "[CRITICAL]"
	case differ.SeverityError:
		return "[ERROR]"
	case differ.SeverityWarning:
		return "[WARNING]"
	default:
		return "[INFO]"
	}
}

// findSpecPath locates a file relative to the source file location.
func findSpecPath(relativePath string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Unable to get current file path")
	}
	return filepath.Join(filepath.Dir(filename), relativePath)
}
