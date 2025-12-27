// Validation Pipeline example demonstrating parse → validate → report workflow.
//
// This example shows a complete validation pipeline with source map integration
// for line numbers in error messages, making it suitable for CI/CD integration.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <spec-file>")
		fmt.Println("Example: go run main.go ../petstore/spec/petstore-v2.json")
		os.Exit(1)
	}
	specPath := os.Args[1]

	fmt.Println("Validation Pipeline")
	fmt.Println("===================")
	fmt.Println()
	fmt.Printf("Input: %s\n\n", specPath)

	// Step 1: Parse with source map for line numbers
	fmt.Println("[1/3] Parsing specification...")
	result, err := parser.ParseWithOptions(
		parser.WithFilePath(specPath),
		parser.WithValidateStructure(true),
		parser.WithSourceMap(true), // Enable line number tracking
	)
	if err != nil {
		log.Fatalf("Parse failed: %v", err)
	}

	fmt.Printf("      OAS Version: %s\n", result.Version)
	fmt.Printf("      Format: %s\n", result.SourceFormat)
	fmt.Printf("      Size: %s\n", formatBytes(result.SourceSize))

	// Report parse warnings/errors if any
	if len(result.Errors) > 0 {
		fmt.Printf("\n      Parse Errors (%d):\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("        - %v\n", e)
		}
	}
	if len(result.Warnings) > 0 {
		fmt.Printf("\n      Parse Warnings (%d):\n", len(result.Warnings))
		for _, w := range result.Warnings {
			fmt.Printf("        - %s\n", w)
		}
	}

	// Step 2: Validate against OAS schema
	fmt.Println()
	fmt.Println("[2/3] Validating against OpenAPI schema...")
	v := validator.New()
	v.IncludeWarnings = true
	v.SourceMap = result.SourceMap // Use source map for line numbers

	valResult, err := v.ValidateParsed(*result)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	// Step 3: Report results
	fmt.Println()
	fmt.Println("[3/3] Validation Results")
	fmt.Printf("      Valid: %t\n", valResult.Valid)
	fmt.Printf("      Errors: %d\n", valResult.ErrorCount)
	fmt.Printf("      Warnings: %d\n", valResult.WarningCount)

	if len(valResult.Errors) > 0 {
		fmt.Println()
		fmt.Println("      Issues:")
		for _, issue := range valResult.Errors {
			severity := severityLabel(issue.Severity)
			location := ""
			if issue.Line > 0 {
				location = fmt.Sprintf(" (line %d)", issue.Line)
			}
			fmt.Printf("        [%s] %s%s: %s\n",
				severity, issue.Path, location, issue.Message)
		}
	}

	// Summary
	fmt.Println()
	fmt.Println("---")
	if valResult.Valid {
		fmt.Println("Validation PASSED")
	} else {
		fmt.Println("Validation FAILED")
		os.Exit(1)
	}
}

// severityLabel returns a human-readable label for the severity level.
func severityLabel(s validator.Severity) string {
	switch s {
	case validator.SeverityCritical:
		return "CRITICAL"
	case validator.SeverityError:
		return "ERROR"
	case validator.SeverityWarning:
		return "WARNING"
	case validator.SeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

// formatBytes formats a byte count as a human-readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
