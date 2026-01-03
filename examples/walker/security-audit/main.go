// Security Audit example demonstrating custom validation rules with walker.
//
// This example shows how to:
//   - Implement custom security validation rules
//   - Pattern match on field names for sensitive data detection
//   - Categorize issues by severity (ERROR, WARNING, INFO)
//   - Build security-focused linting tools
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
)

// Finding represents a security issue found during the audit.
type Finding struct {
	Severity string // ERROR, WARNING, INFO
	Path     string // JSON path from walker
	Message  string
}

// SecuritySchemeInfo holds information about a security scheme.
type SecuritySchemeInfo struct {
	Name string
	Type string
}

func main() {
	specPath := findSpecPath()

	fmt.Println("Security Audit Report")
	fmt.Println("=====================")
	fmt.Println()

	// Parse the specification
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath(specPath),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	// Patterns that indicate sensitive fields
	sensitivePatterns := []string{"password", "secret", "token", "apikey", "credential", "key"}

	// Collect findings and security schemes
	var findings []Finding
	var securitySchemes []SecuritySchemeInfo

	// Track current path template for operation handler
	var currentPathTemplate string

	// Walk the document with security audit handlers
	err = walker.Walk(parseResult,
		// Inventory security schemes
		walker.WithSecuritySchemeHandler(func(wc *walker.WalkContext, scheme *parser.SecurityScheme) walker.Action {
			securitySchemes = append(securitySchemes, SecuritySchemeInfo{
				Name: wc.Name,
				Type: scheme.Type,
			})
			return walker.Continue
		}),

		// Detect internal endpoints
		walker.WithPathHandler(func(wc *walker.WalkContext, pathItem *parser.PathItem) walker.Action {
			currentPathTemplate = wc.PathTemplate
			if strings.Contains(wc.PathTemplate, "internal") || strings.HasPrefix(wc.PathTemplate, "/_") {
				findings = append(findings, Finding{
					Severity: "INFO",
					Path:     wc.JSONPath,
					Message:  "Internal endpoint detected - verify access controls",
				})
			}
			return walker.Continue
		}),

		// Check for missing security requirements
		walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
			// Check if operation has no security requirements and is not on an internal path
			if len(op.Security) == 0 && !isInternalPath(currentPathTemplate) {
				findings = append(findings, Finding{
					Severity: "WARNING",
					Path:     wc.JSONPath,
					Message:  "Operation has no security requirements",
				})
			}
			return walker.Continue
		}),

		// Find sensitive field names in schemas
		walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
			for propName := range schema.Properties {
				propNameLower := strings.ToLower(propName)
				for _, pattern := range sensitivePatterns {
					if strings.Contains(propNameLower, pattern) {
						findings = append(findings, Finding{
							Severity: "ERROR",
							Path:     wc.JSONPath,
							Message:  fmt.Sprintf("Sensitive field '%s' found - ensure proper handling", propName),
						})
						break
					}
				}
			}
			return walker.Continue
		}),
	)
	if err != nil {
		log.Fatalf("Walk error: %v", err)
	}

	// Print the audit report
	printReport(securitySchemes, findings)
}

// isInternalPath checks if a path is an internal or system endpoint.
func isInternalPath(path string) bool {
	return strings.Contains(path, "internal") || strings.HasPrefix(path, "/_")
}

func printReport(schemes []SecuritySchemeInfo, findings []Finding) {
	// Print security schemes
	fmt.Println("Security Schemes Available:")
	if len(schemes) == 0 {
		fmt.Println("  (none)")
	} else {
		// Sort schemes by name
		sort.Slice(schemes, func(i, j int) bool {
			return schemes[i].Name < schemes[j].Name
		})
		for _, s := range schemes {
			fmt.Printf("  - %s (%s)\n", s.Name, s.Type)
		}
	}
	fmt.Println()

	// Group findings by severity
	severityOrder := []string{"ERROR", "WARNING", "INFO"}
	findingsBySeverity := make(map[string][]Finding)
	for _, f := range findings {
		findingsBySeverity[f.Severity] = append(findingsBySeverity[f.Severity], f)
	}

	// Sort findings within each severity by path
	for sev := range findingsBySeverity {
		sort.Slice(findingsBySeverity[sev], func(i, j int) bool {
			return findingsBySeverity[sev][i].Path < findingsBySeverity[sev][j].Path
		})
	}

	// Print findings by severity
	fmt.Println("Findings by Severity:")
	fmt.Println()

	errorCount := 0
	warningCount := 0
	infoCount := 0

	for _, sev := range severityOrder {
		sevFindings := findingsBySeverity[sev]
		if len(sevFindings) == 0 {
			continue
		}

		switch sev {
		case "ERROR":
			errorCount = len(sevFindings)
		case "WARNING":
			warningCount = len(sevFindings)
		case "INFO":
			infoCount = len(sevFindings)
		}

		fmt.Printf("[%s] (%d findings)\n", sev, len(sevFindings))
		for _, f := range sevFindings {
			fmt.Printf("  %s\n", f.Path)
			fmt.Printf("    %s\n", f.Message)
		}
		fmt.Println()
	}

	// Print summary
	fmt.Printf("Summary: %d errors, %d warnings, %d info\n", errorCount, warningCount, infoCount)
}

// findSpecPath locates the api-to-audit.yaml file relative to the source file.
func findSpecPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Cannot determine source file location")
	}
	return filepath.Join(filepath.Dir(filename), "specs", "api-to-audit.yaml")
}
