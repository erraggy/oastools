// Fixer Showcase example demonstrating all available fix types.
//
// This example shows how to:
//   - Identify common OpenAPI spec issues
//   - Apply specific fixes individually
//   - Apply all fixes at once
//   - Use dry-run mode to preview changes
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/erraggy/oastools/fixer"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
)

func main() {
	specPath := findSpecPath("specs/problematic-api.yaml")

	fmt.Println("Fixer Showcase: All Available Fix Types")
	fmt.Println("=======================================")
	fmt.Println()
	fmt.Println("This spec intentionally contains common issues:")
	fmt.Println("  - CSV enum values (should be array)")
	fmt.Println("  - Duplicate operationIds")
	fmt.Println("  - Empty path items")
	fmt.Println("  - Generic schema names like Response[Pet]")
	fmt.Println("  - Missing path parameter definitions")
	fmt.Println("  - Unused/unreferenced schemas")
	fmt.Println()

	// First, show the validation errors
	fmt.Println("[0/7] Initial Validation")
	fmt.Println("------------------------")
	showValidationStatus(specPath)

	// Demo each fix type
	fmt.Println()
	fmt.Println("[1/7] Fix: CSV Enums")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypeEnumCSVExpanded, "CSV enum values -> proper arrays")

	fmt.Println()
	fmt.Println("[2/7] Fix: Duplicate OperationIds")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypeDuplicateOperationId, "Duplicate IDs -> unique suffixed IDs")

	fmt.Println()
	fmt.Println("[3/7] Fix: Empty Paths")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypePrunedEmptyPath, "Empty path items -> removed")

	fmt.Println()
	fmt.Println("[4/7] Fix: Generic Schema Names")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypeRenamedGenericSchema, "Response[Pet] -> Response_Pet_")

	fmt.Println()
	fmt.Println("[5/7] Fix: Missing Path Parameters")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypeMissingPathParameter, "Missing {petId} param -> added")

	fmt.Println()
	fmt.Println("[6/7] Fix: Unused Schemas")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypePrunedUnusedSchema, "Unreferenced schemas -> removed")

	// Demo all fixes combined
	fmt.Println()
	fmt.Println("[7/7] Apply ALL Fixes")
	fmt.Println("------------------------")
	demonstrateAllFixes(specPath)

	fmt.Println()
	fmt.Println("=======================================")
	fmt.Println("Available Fix Types:")
	fmt.Println("  fixer.FixTypeEnumCSVExpanded       - Convert CSV enums to arrays")
	fmt.Println("  fixer.FixTypeDuplicateOperationId  - Make operation IDs unique")
	fmt.Println("  fixer.FixTypePrunedEmptyPath       - Remove empty path items")
	fmt.Println("  fixer.FixTypeRenamedGenericSchema  - Sanitize generic names")
	fmt.Println("  fixer.FixTypeMissingPathParameter  - Add missing path params")
	fmt.Println("  fixer.FixTypePrunedUnusedSchema    - Remove unreferenced schemas")
}

func showValidationStatus(specPath string) {
	parsed, err := parser.ParseWithOptions(parser.WithFilePath(specPath))
	if err != nil {
		log.Printf("  Parse error: %v", err)
		return
	}

	v := validator.New()
	result, err := v.ValidateParsed(*parsed)
	if err != nil {
		log.Printf("  Validate error: %v", err)
		return
	}

	if result.Valid {
		fmt.Println("  [OK] Spec is valid (surprisingly!)")
	} else {
		fmt.Printf("  [X] Found %d validation errors:\n", len(result.Errors))
		// Show first few errors
		maxShow := 5
		for i, e := range result.Errors {
			if i >= maxShow {
				fmt.Printf("    ... and %d more\n", len(result.Errors)-maxShow)
				break
			}
			// Truncate long messages
			msg := e.Message
			if len(msg) > 60 {
				msg = msg[:57] + "..."
			}
			fmt.Printf("    - %s\n", msg)
		}
	}
}

func demonstrateFix(specPath string, fixType fixer.FixType, description string) {
	parsed, err := parser.ParseWithOptions(parser.WithFilePath(specPath))
	if err != nil {
		log.Printf("  Parse error: %v", err)
		return
	}

	// Apply single fix type using the Fixer struct
	f := fixer.New()
	f.EnabledFixes = []fixer.FixType{fixType}
	result, err := f.FixParsed(*parsed)
	if err != nil {
		log.Printf("  Fix error: %v", err)
		return
	}

	if result.FixCount == 0 {
		fmt.Printf("  -> No fixes needed for this type\n")
	} else {
		fmt.Printf("  -> %s\n", description)
		fmt.Printf("  [OK] Applied %d fix(es):\n", result.FixCount)
		for _, fix := range result.Fixes {
			// Clean up the description for display
			desc := fix.Description
			if len(desc) > 70 {
				desc = desc[:67] + "..."
			}
			fmt.Printf("    - %s\n", desc)
		}
	}
}

func demonstrateAllFixes(specPath string) {
	parsed, err := parser.ParseWithOptions(parser.WithFilePath(specPath))
	if err != nil {
		log.Printf("  Parse error: %v", err)
		return
	}

	// First, dry-run to preview
	fmt.Println("  Dry-run preview:")
	preview, err := fixer.FixWithOptions(
		fixer.WithParsed(*parsed),
		fixer.WithEnabledFixes(
			fixer.FixTypeEnumCSVExpanded,
			fixer.FixTypeDuplicateOperationId,
			fixer.FixTypePrunedEmptyPath,
			fixer.FixTypeRenamedGenericSchema,
			fixer.FixTypeMissingPathParameter,
			fixer.FixTypePrunedUnusedSchema,
		),
		fixer.WithDryRun(true),
	)
	if err != nil {
		log.Printf("  Dry-run error: %v", err)
		return
	}
	fmt.Printf("    Would apply %d fixes\n", preview.FixCount)

	// Group fixes by type for summary
	fixCounts := make(map[fixer.FixType]int)
	for _, fix := range preview.Fixes {
		fixCounts[fix.Type]++
	}
	for fixType, count := range fixCounts {
		fmt.Printf("    - %s: %d\n", fixType, count)
	}

	// Now apply all fixes
	fmt.Println()
	fmt.Println("  Applying all fixes:")
	f := fixer.New()
	f.EnabledFixes = []fixer.FixType{
		fixer.FixTypeEnumCSVExpanded,
		fixer.FixTypeDuplicateOperationId,
		fixer.FixTypePrunedEmptyPath,
		fixer.FixTypeRenamedGenericSchema,
		fixer.FixTypeMissingPathParameter,
		fixer.FixTypePrunedUnusedSchema,
	}
	fixed, err := f.FixParsed(*parsed)
	if err != nil {
		log.Printf("  Fix error: %v", err)
		return
	}
	fmt.Printf("  [OK] Applied %d total fixes\n", fixed.FixCount)

	// Validate after fixes
	fmt.Println()
	fmt.Println("  Validation after fixes:")
	v := validator.New()
	validation, err := v.ValidateParsed(*fixed.ToParseResult())
	if err != nil {
		log.Printf("  Validate error: %v", err)
		return
	}

	if validation.Valid {
		fmt.Println("  [OK] Spec is now VALID!")
	} else {
		fmt.Printf("  [X] Still have %d errors (may need manual fixes)\n", len(validation.Errors))
	}

	// Show schema count change
	doc := fixed.Document.(*parser.OAS3Document)
	schemaCount := 0
	if doc.Components != nil && doc.Components.Schemas != nil {
		schemaCount = len(doc.Components.Schemas)
	}
	fmt.Printf("  -> Final schema count: %d\n", schemaCount)

	// List remaining schemas
	if doc.Components != nil && doc.Components.Schemas != nil {
		var names []string
		for name := range doc.Components.Schemas {
			names = append(names, name)
		}
		slices.Sort(names)
		fmt.Printf("  -> Schemas: %s\n", strings.Join(names, ", "))
	}
}

func findSpecPath(relativePath string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Unable to get current file path")
	}
	return filepath.Join(filepath.Dir(filename), relativePath)
}
