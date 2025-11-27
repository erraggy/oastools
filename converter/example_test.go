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
