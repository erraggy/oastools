package converter_test

import (
	"fmt"
	"log"

	"github.com/erraggy/oastools/converter"
	"github.com/erraggy/oastools/parser"
)

// Example demonstrates basic conversion using the convenience function
func Example() {
	// Convert an OAS 2.0 specification to OAS 3.0.3
	result, err := converter.Convert("testdata/petstore-2.0.yaml", "3.0.3")
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

// ExampleConverter demonstrates using a reusable Converter instance
func ExampleConverter() {
	// Create a converter with custom settings
	c := converter.New()
	c.StrictMode = false
	c.IncludeInfo = true

	// Convert multiple files with the same settings
	result1, err := c.Convert("api-v2-1.yaml", "3.0.3")
	if err != nil {
		log.Fatal(err)
	}

	result2, err := c.Convert("api-v2-2.yaml", "3.0.3")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Converted %d documents\n", 2)
	fmt.Printf("Result 1: Success=%v, Issues=%d\n", result1.Success, len(result1.Issues))
	fmt.Printf("Result 2: Success=%v, Issues=%d\n", result2.Success, len(result2.Issues))
}

// ExampleConvert demonstrates the package-level convenience function
func ExampleConvert() {
	result, err := converter.Convert("swagger.yaml", "3.0.3")
	if err != nil {
		log.Fatal(err)
	}

	if result.Success {
		fmt.Println("Conversion successful")
	} else {
		fmt.Printf("Conversion failed with %d critical issues\n", result.CriticalCount)
	}
}

// ExampleConvertParsed demonstrates converting an already-parsed document
func ExampleConvertParsed() {
	// Parse the document first
	parseResult, err := parser.Parse("openapi.yaml", false, true)
	if err != nil {
		log.Fatal(err)
	}

	// Convert the parsed document
	result, err := converter.ConvertParsed(*parseResult, "2.0")
	if err != nil {
		log.Fatal(err)
	}

	if result.HasCriticalIssues() {
		fmt.Printf("Critical issues found: %d\n", result.CriticalCount)
		for _, issue := range result.Issues {
			if issue.Severity == converter.SeverityCritical {
				fmt.Printf("  - %s: %s\n", issue.Path, issue.Message)
			}
		}
	}
}

// ExampleConverter_Convert demonstrates the instance Convert method
func ExampleConverter_Convert() {
	c := converter.New()

	result, err := c.Convert("swagger.yaml", "3.0.3")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Converted %s to %s\n", result.SourceVersion, result.TargetVersion)
}

// ExampleConverter_ConvertParsed demonstrates the instance ConvertParsed method
func ExampleConverter_ConvertParsed() {
	c := converter.New()
	c.IncludeInfo = false // Suppress info messages

	parseResult, _ := parser.Parse("openapi.yaml", false, true)

	result, err := c.ConvertParsed(*parseResult, "3.0.3")
	if err != nil {
		log.Fatal(err)
	}

	// Only warnings and critical issues will be included
	fmt.Printf("Warnings: %d, Critical: %d\n", result.WarningCount, result.CriticalCount)
}

// ExampleConverter_strictMode demonstrates strict mode behavior
func ExampleConverter_strictMode() {
	c := converter.New()
	c.StrictMode = true // Fail on any issues

	parseResult, _ := parser.Parse("openapi.yaml", false, true)

	result, err := c.ConvertParsed(*parseResult, "2.0")
	if err != nil {
		// In strict mode, errors are returned for any warnings or critical issues
		fmt.Printf("Strict mode conversion failed: %v\n", err)
		fmt.Printf("Issues: %d warnings, %d critical\n", result.WarningCount, result.CriticalCount)
		return
	}

	fmt.Println("Conversion succeeded with no issues")
}

// ExampleConversionResult_HasCriticalIssues demonstrates checking for critical issues
func ExampleConversionResult_HasCriticalIssues() {
	result, _ := converter.Convert("openapi-3.1.yaml", "2.0")

	if result.HasCriticalIssues() {
		fmt.Printf("Found %d critical issues:\n", result.CriticalCount)
		for _, issue := range result.Issues {
			if issue.Severity == converter.SeverityCritical {
				fmt.Printf("  %s\n", issue.String())
			}
		}
	} else {
		fmt.Println("No critical issues")
	}
}

// ExampleConversionResult_HasWarnings demonstrates checking for warnings
func ExampleConversionResult_HasWarnings() {
	result, _ := converter.Convert("swagger.yaml", "3.0.3")

	if result.HasWarnings() {
		fmt.Printf("Found %d warnings\n", result.WarningCount)
	} else {
		fmt.Println("No warnings")
	}
}

// ExampleConversionIssue_String demonstrates formatting conversion issues
func ExampleConversionIssue_String() {
	result, _ := converter.Convert("openapi.yaml", "2.0")

	// Print all issues with formatted output
	for _, issue := range result.Issues {
		fmt.Println(issue.String())
	}
}

// Example_convertOAS2ToOAS3 demonstrates converting from OAS 2.0 to OAS 3.x
func Example_convertOAS2ToOAS3() {
	result, err := converter.Convert("swagger-2.0.yaml", "3.0.3")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Converted OAS 2.0 to OAS 3.0.3\n")
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Issues: %d info, %d warnings, %d critical\n",
		result.InfoCount, result.WarningCount, result.CriticalCount)
}

// Example_convertOAS3ToOAS2 demonstrates converting from OAS 3.x to OAS 2.0
func Example_convertOAS3ToOAS2() {
	result, err := converter.Convert("openapi-3.0.yaml", "2.0")
	if err != nil {
		log.Fatal(err)
	}

	if result.HasCriticalIssues() {
		fmt.Println("Warning: Some OAS 3.x features could not be converted to OAS 2.0")
		fmt.Printf("Critical issues: %d\n", result.CriticalCount)
	}

	fmt.Printf("Converted OAS 3.x to OAS 2.0\n")
	fmt.Printf("Review the %d issue(s) before using the converted document\n", len(result.Issues))
}

// Example_handleConversionIssues demonstrates processing conversion issues
func Example_handleConversionIssues() {
	result, _ := converter.Convert("openapi.yaml", "2.0")

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
