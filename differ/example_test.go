package differ_test

import (
	"fmt"
	"log"

	"github.com/erraggy/oastools/differ"
	"github.com/erraggy/oastools/parser"
)

// Example demonstrates basic diff usage with functional options
func Example() {
	// Compare two OpenAPI specifications
	result, err := differ.DiffWithOptions(
		differ.WithSourceFilePath("../testdata/petstore-v1.yaml"),
		differ.WithTargetFilePath("../testdata/petstore-v2.yaml"),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d changes\n", len(result.Changes))
	fmt.Printf("Source version: %s\n", result.SourceVersion)
	fmt.Printf("Target version: %s\n", result.TargetVersion)
}

// Example_simple demonstrates simple diff mode
func Example_simple() {
	result, err := differ.DiffWithOptions(
		differ.WithSourceFilePath("../testdata/petstore-v1.yaml"),
		differ.WithTargetFilePath("../testdata/petstore-v2.yaml"),
		differ.WithMode(differ.ModeSimple),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Simple diff found %d changes\n", len(result.Changes))

	// Print first few changes
	for i, change := range result.Changes {
		if i >= 3 {
			break
		}
		fmt.Println(change.String())
	}
}

// Example_breaking demonstrates breaking change detection
func Example_breaking() {
	result, err := differ.DiffWithOptions(
		differ.WithSourceFilePath("../testdata/petstore-v1.yaml"),
		differ.WithTargetFilePath("../testdata/petstore-v2.yaml"),
		differ.WithMode(differ.ModeBreaking),
		differ.WithIncludeInfo(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	if result.HasBreakingChanges {
		fmt.Printf("⚠️  Found %d breaking change(s)\n", result.BreakingCount)
	} else {
		fmt.Println("✓ No breaking changes detected")
	}

	fmt.Printf("Summary: %d breaking, %d warnings, %d info\n",
		result.BreakingCount, result.WarningCount, result.InfoCount)
}

// Example_parsed demonstrates comparing already-parsed documents
func Example_parsed() {
	// Parse documents once
	source, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v1.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	target, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v2.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Compare parsed documents
	result, err := differ.DiffWithOptions(
		differ.WithSourceParsed(*source),
		differ.WithTargetParsed(*target),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d changes between %s and %s\n",
		len(result.Changes), result.SourceVersion, result.TargetVersion)
}

// Example_changeAnalysis demonstrates detailed change analysis
func Example_changeAnalysis() {
	d := differ.New()
	d.Mode = differ.ModeBreaking

	result, err := d.Diff("../testdata/petstore-v1.yaml", "../testdata/petstore-v2.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Group changes by category
	categories := make(map[differ.ChangeCategory]int)
	for _, change := range result.Changes {
		categories[change.Category]++
	}

	fmt.Println("Changes by category:")
	for category, count := range categories {
		fmt.Printf("  %s: %d\n", category, count)
	}
}

// Example_filterBySeverity demonstrates filtering changes by severity
func Example_filterBySeverity() {
	d := differ.New()
	d.Mode = differ.ModeBreaking
	d.IncludeInfo = false // Exclude info-level changes

	result, err := d.Diff("../testdata/petstore-v1.yaml", "../testdata/petstore-v2.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Only breaking changes and warnings remain
	fmt.Printf("Breaking and warnings only: %d changes\n", len(result.Changes))
	fmt.Printf("Breaking: %d, Warnings: %d\n", result.BreakingCount, result.WarningCount)
}

// Example_reusableDiffer demonstrates creating a reusable differ instance
func Example_reusableDiffer() {
	// Create a reusable differ with specific configuration
	d := differ.New()
	d.Mode = differ.ModeBreaking
	d.IncludeInfo = false
	d.UserAgent = "my-api-tool/1.0"

	// Use the same differ for multiple comparisons
	specs := []struct{ old, new string }{
		{"../testdata/petstore-v1.yaml", "../testdata/petstore-v2.yaml"},
	}

	for _, spec := range specs {
		result, err := d.Diff(spec.old, spec.new)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		fmt.Printf("%s → %s: ", spec.old, spec.new)
		if result.HasBreakingChanges {
			fmt.Printf("%d breaking\n", result.BreakingCount)
		} else {
			fmt.Println("compatible")
		}
	}
}
