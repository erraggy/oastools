// Validate-and-Fix example demonstrating the fixer package.
//
// This example shows how to:
//   - Parse a specification with validation issues
//   - Validate and identify errors
//   - Preview fixes with dry-run mode
//   - Apply fixes automatically
//   - Validate the fixed specification
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/erraggy/oastools/fixer"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
)

func main() {
	specPath := findSpecPath("specs/invalid.yaml")

	fmt.Println("Validate-and-Fix Workflow")
	fmt.Println("=========================")
	fmt.Println()

	// Step 1: Parse the specification
	fmt.Println("[1/5] Parsing specification...")
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath(specPath),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}
	fmt.Printf("      Version: %s\n", parseResult.Version)
	fmt.Printf("      Format: %s\n", parseResult.SourceFormat)

	// Step 2: Validate BEFORE fixing (show errors)
	fmt.Println()
	fmt.Println("[2/5] Validating (before fix)...")
	v := validator.New()
	v.IncludeWarnings = true
	valResult, err := v.ValidateParsed(*parseResult)
	if err != nil {
		log.Fatalf("Validation error: %v", err)
	}
	fmt.Printf("      Valid: %t\n", valResult.Valid)
	fmt.Printf("      Errors: %d\n", len(valResult.Errors))
	for _, e := range valResult.Errors {
		fmt.Printf("        - %s\n", e.Message)
	}

	// Step 3: Preview fixes with dry-run
	fmt.Println()
	fmt.Println("[3/5] Previewing fixes (dry-run)...")
	dryRunResult, err := fixer.FixWithOptions(
		fixer.WithParsed(*parseResult),
		fixer.WithInferTypes(true),
		fixer.WithDryRun(true),
	)
	if err != nil {
		log.Fatalf("Dry-run error: %v", err)
	}
	fmt.Printf("      Would apply %d fix(es):\n", dryRunResult.FixCount)
	for _, fix := range dryRunResult.Fixes {
		fmt.Printf("        - [%s] %s\n", fix.Type, fix.Description)
	}

	// Step 4: Apply fixes for real
	fmt.Println()
	fmt.Println("[4/5] Applying fixes...")
	fixResult, err := fixer.FixWithOptions(
		fixer.WithParsed(*parseResult),
		fixer.WithInferTypes(true),
		fixer.WithEnabledFixes(
			fixer.FixTypeMissingPathParameter,
			fixer.FixTypePrunedUnusedSchema,
		),
	)
	if err != nil {
		log.Fatalf("Fix error: %v", err)
	}
	fmt.Printf("      Applied %d fix(es)\n", fixResult.FixCount)

	// Step 5: Validate AFTER fixing
	fmt.Println()
	fmt.Println("[5/5] Validating (after fix)...")
	// Create a new parse result with the fixed document
	fixedParseResult := parser.ParseResult{
		Version:      parseResult.Version,
		OASVersion:   parseResult.OASVersion,
		SourceFormat: parseResult.SourceFormat,
		Document:     fixResult.Document,
	}
	valResultAfter, err := v.ValidateParsed(fixedParseResult)
	if err != nil {
		log.Fatalf("Validation error: %v", err)
	}
	fmt.Printf("      Valid: %t\n", valResultAfter.Valid)
	fmt.Printf("      Errors: %d\n", len(valResultAfter.Errors))

	// Summary
	fmt.Println()
	fmt.Println("---")
	fmt.Printf("Summary: Applied %d fixes\n", fixResult.FixCount)
	for _, fix := range fixResult.Fixes {
		fmt.Printf("  - [%s] %s\n", fix.Type, fix.Description)
	}

	if valResultAfter.Valid {
		fmt.Println()
		fmt.Println("Specification is now valid!")
		os.Exit(0)
	} else {
		os.Exit(1)
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
