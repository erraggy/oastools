// Quickstart example demonstrating oastools parser and validator packages.
//
// This example shows the fundamental workflow of parsing and validating
// an OpenAPI specification, then accessing its structure programmatically.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
)

func main() {
	// Find spec.yaml relative to this source file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("failed to get current file path")
	}
	specPath := filepath.Join(filepath.Dir(filename), "spec.yaml")

	fmt.Println("oastools Quickstart")
	fmt.Println("===================")
	fmt.Println()

	// Step 1: Parse the specification
	fmt.Println("[1/3] Parsing OpenAPI specification...")
	result, err := parser.ParseWithOptions(
		parser.WithFilePath(specPath),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}
	fmt.Printf("      Version: %s\n", result.Version)
	fmt.Printf("      Format: %s\n", result.SourceFormat)

	// Report any parse-time issues
	if len(result.Errors) > 0 {
		fmt.Printf("      Parse errors: %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("        - %v\n", e)
		}
	}

	// Step 2: Validate the specification against OAS schema rules
	fmt.Println()
	fmt.Println("[2/3] Validating against OAS schema...")
	v := validator.New()
	v.IncludeWarnings = true
	valResult, err := v.ValidateParsed(*result)
	if err != nil {
		log.Fatalf("validation failed: %v", err)
	}
	fmt.Printf("      Valid: %t\n", valResult.Valid)
	fmt.Printf("      Errors: %d\n", valResult.ErrorCount)
	fmt.Printf("      Warnings: %d\n", valResult.WarningCount)

	// Report validation issues if any
	if len(valResult.Errors) > 0 {
		fmt.Println()
		fmt.Println("      Issues:")
		for _, issue := range valResult.Errors {
			fmt.Printf("        [%s] %s: %s\n", issue.Severity, issue.Path, issue.Message)
		}
	}

	// Step 3: Access the parsed document structure
	fmt.Println()
	fmt.Println("[3/3] Accessing document structure...")
	accessor := result.AsAccessor()
	info := accessor.GetInfo()
	paths := accessor.GetPaths()
	schemas := accessor.GetSchemas()

	fmt.Printf("      API Title: %s\n", info.Title)
	fmt.Printf("      API Version: %s\n", info.Version)
	fmt.Printf("      Paths: %d\n", len(paths))
	fmt.Printf("      Schemas: %d\n", len(schemas))

	// List the paths and schemas
	fmt.Println()
	fmt.Println("      Paths defined:")
	for path := range paths {
		fmt.Printf("        - %s\n", path)
	}
	fmt.Println("      Schemas defined:")
	for name := range schemas {
		fmt.Printf("        - %s\n", name)
	}

	fmt.Println()
	fmt.Println("---")
	if valResult.Valid {
		fmt.Println("Quickstart complete!")
	} else {
		fmt.Println("Quickstart complete (with validation errors)")
		os.Exit(1)
	}
}
