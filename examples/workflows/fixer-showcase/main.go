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

// allFixTypes lists every fixer.FixType. Declared once so the individual demos
// below and the combined run below them cannot drift out of sync -- the drift
// that let two fix types go undemonstrated for several releases.
var allFixTypes = []fixer.FixType{
	fixer.FixTypeEnumCSVExpanded,
	fixer.FixTypeDuplicateOperationId,
	fixer.FixTypePrunedEmptyPath,
	fixer.FixTypeRenamedGenericSchema,
	fixer.FixTypeMissingPathParameter,
	fixer.FixTypePathParameterNotRequired,
	fixer.FixTypePrunedUnusedSchema,
	fixer.FixTypeStubMissingRef,
}

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
	fmt.Println("  - Declared path parameters missing required: true")
	fmt.Println("  - Unused/unreferenced schemas")
	fmt.Println("  - $refs pointing at schemas that do not exist")
	fmt.Println()

	// First, show the validation errors
	fmt.Println("[0/9] Initial Validation")
	fmt.Println("------------------------")
	showValidationStatus(specPath)

	// Demo each fix type
	fmt.Println()
	fmt.Println("[1/9] Fix: CSV Enums")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypeEnumCSVExpanded, "CSV enum values -> proper arrays")

	fmt.Println()
	fmt.Println("[2/9] Fix: Duplicate OperationIds")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypeDuplicateOperationId, "Duplicate IDs -> unique suffixed IDs")

	fmt.Println()
	fmt.Println("[3/9] Fix: Empty Paths")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypePrunedEmptyPath, "Empty path items -> removed")

	fmt.Println()
	fmt.Println("[4/9] Fix: Generic Schema Names")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypeRenamedGenericSchema, "Response[Pet] -> Response_Pet_")

	fmt.Println()
	fmt.Println("[5/9] Fix: Missing Path Parameters")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypeMissingPathParameter, "Missing {petId} param -> added")

	fmt.Println()
	fmt.Println("[6/9] Fix: Path Parameters Missing required")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypePathParameterNotRequired, "Declared {orderId} param -> required: true")

	fmt.Println()
	fmt.Println("[7/9] Fix: Unused Schemas")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypePrunedUnusedSchema, "Unreferenced schemas -> removed")

	fmt.Println()
	fmt.Println("[8/9] Fix: Stub Missing Refs")
	fmt.Println("------------------------")
	demonstrateFix(specPath, fixer.FixTypeStubMissingRef, "$ref to undefined Order -> stub created")

	// Demo all fixes combined
	fmt.Println()
	fmt.Println("[9/9] Apply ALL Fixes")
	fmt.Println("------------------------")
	demonstrateAllFixes(specPath)

	fmt.Println()
	fmt.Println("=======================================")
	fmt.Println("Available Fix Types:")
	fmt.Println("  fixer.FixTypeEnumCSVExpanded            - Convert CSV enums to arrays")
	fmt.Println("  fixer.FixTypeDuplicateOperationId       - Make operation IDs unique")
	fmt.Println("  fixer.FixTypePrunedEmptyPath            - Remove empty path items")
	fmt.Println("  fixer.FixTypeRenamedGenericSchema       - Sanitize generic names")
	fmt.Println("  fixer.FixTypeMissingPathParameter       - Add missing path params")
	fmt.Println("  fixer.FixTypePathParameterNotRequired   - Set required: true on path params")
	fmt.Println("  fixer.FixTypePrunedUnusedSchema         - Remove unreferenced schemas")
	fmt.Println("  fixer.FixTypeStubMissingRef             - Stub out unresolved $refs")
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
		fixer.WithEnabledFixes(allFixTypes...),
		fixer.WithDryRun(true),
	)
	if err != nil {
		log.Printf("  Dry-run error: %v", err)
		return
	}
	fmt.Printf("    Would apply %d fixes\n", preview.FixCount)

	// Group fixes by type for summary. Map iteration order is random, so sort
	// the types before printing -- otherwise this block reorders itself run to
	// run, which is exactly the noise a showcase should not produce.
	fixCounts := make(map[fixer.FixType]int)
	for _, fix := range preview.Fixes {
		fixCounts[fix.Type]++
	}
	types := make([]fixer.FixType, 0, len(fixCounts))
	for fixType := range fixCounts {
		types = append(types, fixType)
	}
	slices.Sort(types)
	for _, fixType := range types {
		fmt.Printf("    - %s: %d\n", fixType, fixCounts[fixType])
	}
	fmt.Println("    (stub-missing-ref is absent above: stubbing is skipped in")
	fmt.Println("     dry-run mode, so the applied count below is higher)")

	// Now apply all fixes
	fmt.Println()
	fmt.Println("  Applying all fixes:")
	f := fixer.New()
	f.EnabledFixes = slices.Clone(allFixTypes)
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

	// Show schema count change using accessor pattern
	accessor := fixed.ToParseResult().AsAccessor()
	if accessor == nil {
		log.Printf("  Could not access document")
		return
	}
	schemas := accessor.GetSchemas()
	schemaCount := 0
	if schemas != nil {
		schemaCount = len(schemas)
	}
	fmt.Printf("  -> Final schema count: %d\n", schemaCount)

	// List remaining schemas
	if schemas != nil {
		var names []string
		for name := range schemas {
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
