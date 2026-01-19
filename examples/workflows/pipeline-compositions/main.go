// Pipeline Compositions example demonstrating multi-step oastools workflows.
//
// This example shows how to:
//   - Chain multiple oastools operations together
//   - Convert legacy specs before generating code
//   - Fix issues across multiple specs then join
//   - Reuse parsed documents for efficiency
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/erraggy/oastools/converter"
	"github.com/erraggy/oastools/fixer"
	"github.com/erraggy/oastools/generator"
	"github.com/erraggy/oastools/joiner"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
)

func main() {
	legacyPath := findSpecPath("specs/legacy-api.yaml")
	serviceAPath := findSpecPath("specs/service-a.yaml")
	serviceBPath := findSpecPath("specs/service-b.yaml")

	fmt.Println("Pipeline Compositions")
	fmt.Println("=====================")
	fmt.Println()
	fmt.Println("Demonstrating multi-step oastools workflows")
	fmt.Println()

	// Pipeline 1: Convert -> Validate -> Generate
	fmt.Println("[1/3] Pipeline: Convert Legacy -> Validate -> Generate")
	fmt.Println("-------------------------------------------------------")
	demonstrateConvertPipeline(legacyPath)

	// Pipeline 2: Fix -> Validate (single spec)
	fmt.Println()
	fmt.Println("[2/3] Pipeline: Fix -> Validate")
	fmt.Println("-------------------------------------------------------")
	demonstrateFixPipeline(serviceAPath)

	// Pipeline 3: Fix -> Join -> Validate -> Generate
	fmt.Println()
	fmt.Println("[3/3] Pipeline: Fix All -> Join -> Validate -> Generate")
	fmt.Println("-------------------------------------------------------")
	demonstrateFixJoinPipeline(serviceAPath, serviceBPath)

	fmt.Println()
	fmt.Println("=======================================================")
	fmt.Println("Key Takeaways:")
	fmt.Println("  - Chain operations for complex workflows")
	fmt.Println("  - Parse once, reuse for multiple operations")
	fmt.Println("  - Fix before join to ensure clean merge")
	fmt.Println("  - Convert legacy specs before code generation")
}

func demonstrateConvertPipeline(legacyPath string) {
	// Step 1: Parse OAS 2.0
	fmt.Println("  Step 1: Parse OAS 2.0 spec")
	parsed, err := parser.ParseWithOptions(parser.WithFilePath(legacyPath))
	if err != nil {
		log.Printf("    Parse error: %v", err)
		return
	}
	fmt.Printf("    ✓ Parsed: %s (OAS %s)\n", filepath.Base(legacyPath), parsed.Version)

	// Step 2: Convert to OAS 3.0.3
	fmt.Println("  Step 2: Convert to OAS 3.0.3")
	c := converter.New()
	converted, err := c.ConvertParsed(*parsed, "3.0.3")
	if err != nil {
		log.Printf("    Convert error: %v", err)
		return
	}
	fmt.Printf("    ✓ Converted to OAS %s\n", converted.TargetVersion)

	// Step 3: Validate converted spec
	fmt.Println("  Step 3: Validate converted spec")
	v := validator.New()
	validation, err := v.ValidateParsed(*converted.ToParseResult())
	if err != nil {
		log.Printf("    Validate error: %v", err)
		return
	}
	if validation.Valid {
		fmt.Println("    ✓ Validation passed")
	} else {
		fmt.Printf("    ✗ Validation failed: %d errors\n", len(validation.Errors))
		return
	}

	// Step 4: Generate code (types only for speed)
	fmt.Println("  Step 4: Generate Go types")
	tmpDir, err := os.MkdirTemp("", "pipeline-demo-*")
	if err != nil {
		log.Printf("    Temp dir error: %v", err)
		return
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	genResult, err := generator.GenerateWithOptions(
		generator.WithParsed(*converted.ToParseResult()),
		generator.WithPackageName("legacyapi"),
		generator.WithTypes(true),
	)
	if err != nil {
		log.Printf("    Generate error: %v", err)
		return
	}
	fmt.Printf("    ✓ Generated %d files\n", len(genResult.Files))

	fmt.Println()
	fmt.Println("  Result: Legacy OAS 2.0 -> OAS 3.0.3 -> Go types ✓")
}

func demonstrateFixPipeline(specPath string) {
	// Step 1: Parse and show issues
	fmt.Println("  Step 1: Parse and identify issues")
	parsed, err := parser.ParseWithOptions(parser.WithFilePath(specPath))
	if err != nil {
		log.Printf("    Parse error: %v", err)
		return
	}
	fmt.Printf("    ✓ Parsed: %s\n", filepath.Base(specPath))

	// Step 2: Validate (will show errors)
	fmt.Println("  Step 2: Validate (before fix)")
	v := validator.New()
	validation, err := v.ValidateParsed(*parsed)
	if err != nil {
		log.Printf("    Validate error: %v", err)
		return
	}
	if !validation.Valid {
		fmt.Printf("    ✗ Found %d validation errors\n", len(validation.Errors))
		for _, e := range validation.Errors {
			fmt.Printf("      - %s\n", e.Message)
		}
	} else {
		fmt.Println("    ✓ No validation errors (checking for fixable issues)")
	}

	// Step 3: Fix issues
	fmt.Println("  Step 3: Apply fixes")
	fixResult, err := fixer.FixWithOptions(
		fixer.WithParsed(*parsed),
		fixer.WithInferTypes(true),
		fixer.WithEnabledFixes(
			fixer.FixTypeDuplicateOperationId,
		),
	)
	if err != nil {
		log.Printf("    Fix error: %v", err)
		return
	}
	fmt.Printf("    ✓ Applied %d fixes\n", fixResult.FixCount)
	for _, fix := range fixResult.Fixes {
		fmt.Printf("      - %s\n", fix.Description)
	}

	// Step 4: Re-validate
	fmt.Println("  Step 4: Validate (after fix)")
	validation2, err := v.ValidateParsed(*fixResult.ToParseResult())
	if err != nil {
		log.Printf("    Validate error: %v", err)
		return
	}
	if validation2.Valid {
		fmt.Println("    ✓ Validation passed")
	} else {
		fmt.Printf("    ✗ Still have %d errors\n", len(validation2.Errors))
	}

	fmt.Println()
	fmt.Println("  Result: Spec with issues -> Fixed -> Valid ✓")
}

func demonstrateFixJoinPipeline(serviceAPath, serviceBPath string) {
	// Step 1: Parse all specs
	fmt.Println("  Step 1: Parse all specs")
	parsedA, err := parser.ParseWithOptions(parser.WithFilePath(serviceAPath))
	if err != nil {
		log.Printf("    Parse error: %v", err)
		return
	}
	parsedB, err := parser.ParseWithOptions(parser.WithFilePath(serviceBPath))
	if err != nil {
		log.Printf("    Parse error: %v", err)
		return
	}
	fmt.Printf("    ✓ Parsed: %s, %s\n", filepath.Base(serviceAPath), filepath.Base(serviceBPath))

	// Step 2: Fix all specs
	fmt.Println("  Step 2: Fix all specs")
	fixedA, err := fixer.FixWithOptions(
		fixer.WithParsed(*parsedA),
		fixer.WithEnabledFixes(fixer.FixTypeDuplicateOperationId),
	)
	if err != nil {
		log.Printf("    Fix error: %v", err)
		return
	}
	fixedB, err := fixer.FixWithOptions(
		fixer.WithParsed(*parsedB),
		fixer.WithEnabledFixes(fixer.FixTypeDuplicateOperationId),
	)
	if err != nil {
		log.Printf("    Fix error: %v", err)
		return
	}
	fmt.Printf("    ✓ Service A: %d fixes applied\n", fixedA.FixCount)
	fmt.Printf("    ✓ Service B: %d fixes applied\n", fixedB.FixCount)

	// Step 3: Join fixed specs
	fmt.Println("  Step 3: Join fixed specs")
	joinResult, err := joiner.JoinWithOptions(
		joiner.WithParsed(*fixedA.ToParseResult(), *fixedB.ToParseResult()),
		joiner.WithSchemaStrategy(joiner.StrategyAcceptLeft),
		joiner.WithSemanticDeduplication(true),
	)
	if err != nil {
		log.Printf("    Join error: %v", err)
		return
	}
	fmt.Printf("    ✓ Joined: OAS %s\n", joinResult.Version)

	// Step 4: Validate joined spec
	fmt.Println("  Step 4: Validate joined spec")
	v := validator.New()
	validation, err := v.ValidateParsed(*joinResult.ToParseResult())
	if err != nil {
		log.Printf("    Validate error: %v", err)
		return
	}
	if validation.Valid {
		fmt.Println("    ✓ Validation passed")
	} else {
		fmt.Printf("    ✗ Validation failed: %d errors\n", len(validation.Errors))
	}

	// Step 5: Generate code
	fmt.Println("  Step 5: Generate Go code")
	tmpDir, err := os.MkdirTemp("", "pipeline-demo-*")
	if err != nil {
		log.Printf("    Temp dir error: %v", err)
		return
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	genResult, err := generator.GenerateWithOptions(
		generator.WithParsed(*joinResult.ToParseResult()),
		generator.WithPackageName("unified"),
		generator.WithTypes(true),
	)
	if err != nil {
		log.Printf("    Generate error: %v", err)
		return
	}
	fmt.Printf("    ✓ Generated %d files\n", len(genResult.Files))

	fmt.Println()
	fmt.Println("  Result: Multiple specs -> Fixed -> Joined -> Validated -> Generated ✓")
}

func findSpecPath(relativePath string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Unable to get current file path")
	}
	return filepath.Join(filepath.Dir(filename), relativePath)
}
