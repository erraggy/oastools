// Overlay Transformations example demonstrating the overlay package.
//
// This example shows how to:
//   - Parse and validate an OpenAPI Overlay document
//   - Preview changes with dry-run mode
//   - Apply environment-specific transformations
//   - Use JSONPath targeting for selective updates
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/erraggy/oastools/overlay"
	"github.com/erraggy/oastools/parser"
)

func main() {
	basePath := findSpecPath("specs/base.yaml")
	overlayPath := findSpecPath("specs/production.yaml")

	fmt.Println("Overlay Transformations Workflow")
	fmt.Println("=================================")
	fmt.Println()
	fmt.Printf("Base Spec: %s\n", filepath.Base(basePath))
	fmt.Printf("Overlay: %s\n", filepath.Base(overlayPath))
	fmt.Println()

	// Step 1: Parse and validate the overlay
	fmt.Println("[1/4] Validating overlay document...")
	o, err := overlay.ParseOverlayFile(overlayPath)
	if err != nil {
		log.Fatalf("Parse overlay error: %v", err)
	}

	errs := overlay.Validate(o)
	if len(errs) > 0 {
		fmt.Printf("      Validation errors: %d\n", len(errs))
		for _, e := range errs {
			fmt.Printf("        - %s\n", e.Message)
		}
		log.Fatal("Overlay validation failed")
	}
	fmt.Println("      Overlay is valid")
	fmt.Printf("      Title: %s\n", o.Info.Title)
	fmt.Printf("      Actions defined: %d\n", len(o.Actions))

	// Step 2: Parse the base spec
	fmt.Println()
	fmt.Println("[2/4] Parsing base specification...")
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath(basePath),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		log.Fatalf("Parse spec error: %v", err)
	}
	fmt.Printf("      Version: %s\n", parseResult.Version)

	// Step 3: Dry-run to preview changes
	fmt.Println()
	fmt.Println("[3/4] Previewing changes (dry-run)...")
	dryResult, err := overlay.DryRunWithOptions(
		overlay.WithSpecParsed(*parseResult),
		overlay.WithOverlayParsed(o),
	)
	if err != nil {
		log.Fatalf("Dry-run error: %v", err)
	}

	fmt.Printf("      Would apply: %d action(s)\n", dryResult.WouldApply)
	fmt.Printf("      Would skip: %d action(s)\n", dryResult.WouldSkip)
	fmt.Println()
	fmt.Println("      Changes:")
	for _, change := range dryResult.Changes {
		fmt.Printf("        - %s %d node(s) at %s\n",
			change.Operation, change.MatchCount, change.Target)
	}

	// Step 4: Apply the overlay
	fmt.Println()
	fmt.Println("[4/4] Applying overlay...")
	result, err := overlay.ApplyWithOptions(
		overlay.WithSpecParsed(*parseResult),
		overlay.WithOverlayParsed(o),
	)
	if err != nil {
		log.Fatalf("Apply error: %v", err)
	}

	fmt.Printf("      Actions applied: %d\n", result.ActionsApplied)
	fmt.Printf("      Actions skipped: %d\n", result.ActionsSkipped)

	// Show transformation results
	fmt.Println()
	fmt.Println("--- Transformation Results ---")

	// Access transformed document (using safe type assertions)
	doc, ok := result.Document.(map[string]any)
	if !ok {
		log.Fatalf("Unexpected document type: %T", result.Document)
	}

	info, ok := doc["info"].(map[string]any)
	if !ok {
		log.Fatal("Missing or invalid 'info' object in transformed document")
	}

	fmt.Printf("New Title: %s\n", info["title"])
	if env, ok := info["x-environment"]; ok {
		fmt.Printf("Environment: %s\n", env)
	}

	if serversRaw, exists := doc["servers"]; exists {
		if servers, ok := serversRaw.([]any); ok && len(servers) > 0 {
			if server, ok := servers[0].(map[string]any); ok {
				fmt.Printf("Production URL: %s\n", server["url"])
			}
		}
	}

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		log.Fatal("Missing or invalid 'paths' object in transformed document")
	}
	fmt.Printf("Paths (after removing internal): %d\n", len(paths))
	fmt.Println()
	fmt.Println("Remaining paths:")
	for path := range paths {
		fmt.Printf("  - %s\n", path)
	}

	fmt.Println()
	fmt.Println("---")
	fmt.Println("Overlay applied successfully")
}

// findSpecPath locates a file relative to the source file location.
func findSpecPath(relativePath string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Unable to get current file path")
	}
	return filepath.Join(filepath.Dir(filename), relativePath)
}
