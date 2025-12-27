// Version Conversion example demonstrating the converter package.
//
// This example shows how to:
//   - Convert a Swagger 2.0 specification to OpenAPI 3.0.3
//   - Track conversion issues by severity
//   - Understand the key transformations applied
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/erraggy/oastools/converter"
	"go.yaml.in/yaml/v4"
)

func main() {
	specPath := findSpecPath("specs/swagger-v2.yaml")

	fmt.Println("Version Conversion Workflow")
	fmt.Println("===========================")
	fmt.Println()
	fmt.Printf("Input: %s\n", specPath)
	fmt.Println()

	// Step 1: Convert to OAS 3.0.3
	fmt.Println("[1/3] Converting OAS 2.0 -> OAS 3.0.3...")
	result, err := converter.ConvertWithOptions(
		converter.WithFilePath(specPath),
		converter.WithTargetVersion("3.0.3"),
		converter.WithIncludeInfo(true),
	)
	if err != nil {
		log.Fatalf("Conversion error: %v", err)
	}

	fmt.Printf("      Source: %s\n", result.SourceVersion)
	fmt.Printf("      Target: %s\n", result.TargetVersion)

	// Step 2: Report conversion issues by severity
	fmt.Println()
	fmt.Println("[2/3] Conversion Issues:")

	var criticalCount, warningCount, infoCount int
	for _, issue := range result.Issues {
		switch issue.Severity {
		case converter.SeverityCritical:
			criticalCount++
		case converter.SeverityWarning:
			warningCount++
		case converter.SeverityInfo:
			infoCount++
		}
	}

	fmt.Printf("      Critical: %d\n", criticalCount)
	fmt.Printf("      Warnings: %d\n", warningCount)
	fmt.Printf("      Info: %d\n", infoCount)

	if len(result.Issues) > 0 {
		fmt.Println()
		fmt.Println("      Details:")
		for _, issue := range result.Issues {
			severityLabel := severityString(issue.Severity)
			path := issue.Path
			if path == "" {
				path = "(document)"
			}
			fmt.Printf("        [%s] %s: %s\n", severityLabel, path, issue.Message)
		}
	}

	// Step 3: Show key conversions
	fmt.Println()
	fmt.Println("[3/3] Key Conversions Applied:")
	fmt.Println("      - host/basePath/schemes -> servers array")
	fmt.Println("      - definitions -> components/schemas")
	fmt.Println("      - consumes/produces -> requestBody/response content")
	fmt.Println("      - securityDefinitions -> components/securitySchemes")
	fmt.Println("      - body parameters -> requestBody objects")

	// Show converted output preview
	fmt.Println()
	fmt.Println("--- Converted Specification (excerpt) ---")

	output, err := yaml.Marshal(result.Document)
	if err != nil {
		log.Fatalf("Marshal error: %v", err)
	}

	// Print first 30 lines
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i >= 30 {
			fmt.Println("... (truncated)")
			break
		}
		fmt.Println(line)
	}

	// Summary
	fmt.Println()
	fmt.Println("---")
	if result.HasCriticalIssues() {
		fmt.Printf("Conversion completed with %d critical issue(s)\n", criticalCount)
		os.Exit(1)
	} else {
		fmt.Println("Conversion completed successfully")
	}
}

func severityString(s converter.Severity) string {
	switch s {
	case converter.SeverityCritical:
		return "CRITICAL"
	case converter.SeverityWarning:
		return "WARNING"
	case converter.SeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
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
