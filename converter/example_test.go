package converter_test

import (
	"fmt"
	"log"

	"github.com/erraggy/oastools/converter"
)

// Example demonstrates basic conversion using functional options
func Example() {
	// Convert an OAS 2.0 specification to OAS 3.0.3
	result, err := converter.ConvertWithOptions(
		converter.WithFilePath("testdata/petstore-2.0.yaml"),
		converter.WithTargetVersion("3.0.3"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Check for critical issues
	if result.HasCriticalIssues() {
		fmt.Printf("Conversion completed with %d critical issue(s)\n", result.CriticalCount)
		return
	}

	fmt.Printf("Successfully converted from %s to %s\n", result.SourceVersion, result.TargetVersion)
	fmt.Printf("Issues: %d info, %d warnings, %d critical\n",
		result.InfoCount, result.WarningCount, result.CriticalCount)
}

// Example_handleConversionIssues demonstrates processing conversion issues
func Example_handleConversionIssues() {
	result, _ := converter.ConvertWithOptions(
		converter.WithFilePath("openapi.yaml"),
		converter.WithTargetVersion("2.0"),
	)

	// Categorize issues by severity
	for _, issue := range result.Issues {
		switch issue.Severity {
		case converter.SeverityCritical:
			fmt.Printf("CRITICAL [%s]: %s\n", issue.Path, issue.Message)
			if issue.Context != "" {
				fmt.Printf("  Context: %s\n", issue.Context)
			}
		case converter.SeverityError:
			fmt.Printf("ERROR [%s]: %s\n", issue.Path, issue.Message)
		case converter.SeverityWarning:
			fmt.Printf("WARNING [%s]: %s\n", issue.Path, issue.Message)
		case converter.SeverityInfo:
			fmt.Printf("INFO [%s]: %s\n", issue.Path, issue.Message)
		}
	}

	// Summary
	fmt.Printf("\nSummary: %d critical, %d warnings, %d info\n",
		result.CriticalCount, result.WarningCount, result.InfoCount)
}

// Example_toParseResult demonstrates using ToParseResult() to chain converter
// output with other packages like validator, fixer, or differ.
func Example_toParseResult() {
	// Convert an OAS 2.0 specification to OAS 3.0.3
	convResult, err := converter.ConvertWithOptions(
		converter.WithFilePath("../testdata/petstore-2.0.yaml"),
		converter.WithTargetVersion("3.0.3"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Convert to ParseResult for use with validator, fixer, differ, etc.
	parseResult := convResult.ToParseResult()

	// The ParseResult can now be used with other packages:
	// - validator.ValidateParsed(*parseResult)
	// - fixer.FixParsed(*parseResult)
	// - differ.DiffParsed(*baseResult, *parseResult)

	fmt.Printf("Source: %s\n", parseResult.SourcePath)
	fmt.Printf("Version: %s\n", parseResult.Version)
	fmt.Printf("Has document: %v\n", parseResult.Document != nil)
	// Output:
	// Source: converter
	// Version: 3.0.3
	// Has document: true
}

// Example_complexConversion demonstrates converting a complex OAS 2.0 document
// with OAuth2 flows, custom security schemes, and polymorphic schemas to OAS 3.0.
func Example_complexConversion() {
	// Convert a complex OAS 2.0 document with strict mode disabled
	// to allow for lossy conversions (e.g., allowEmptyValue is dropped)
	result, err := converter.ConvertWithOptions(
		converter.WithFilePath("testdata/petstore-2.0.yaml"),
		converter.WithTargetVersion("3.0.3"),
		converter.WithStrictMode(false), // Allow lossy conversions
		converter.WithIncludeInfo(true), // Include informational messages
	)

	if err != nil {
		log.Fatal(err)
	}

	// Review conversion issues to understand the changes
	fmt.Printf("Conversion from %s to %s:\n", result.SourceVersion, result.TargetVersion)
	fmt.Printf("- Critical issues: %d\n", result.CriticalCount)
	fmt.Printf("- Warnings: %d\n", result.WarningCount)
	fmt.Printf("- Info messages: %d\n", result.InfoCount)

	// Important conversions in OAS 2.0 → 3.0:
	// - OAuth2 flows are restructured under components.securitySchemes
	// - `host`, `basePath`, `schemes` → `servers` array with URL templates
	// - `definitions` → `components.schemas`
	// - `consumes`/`produces` → requestBody.content / responses.*.content
	// - Body parameters → requestBody objects

	// Check if conversion was successful despite issues
	if !result.HasCriticalIssues() {
		fmt.Println("\nConversion completed successfully")
	}
}
