// Version Migration example demonstrating OAS version conversions.
//
// This example shows how to:
//   - Upgrade specs from OAS 3.0 to 3.1 or 3.2
//   - Downgrade specs (with feature loss warnings)
//   - Handle lossy conversions gracefully
//   - Understand what features change between versions
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/erraggy/oastools/converter"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
)

func main() {
	modernPath := findSpecPath("specs/modern-api-31.yaml")
	classicPath := findSpecPath("specs/classic-api-30.yaml")

	fmt.Println("Version Migration: OAS 3.0, 3.1, and 3.2")
	fmt.Println("=========================================")
	fmt.Println()
	fmt.Println("OAS Version Features:")
	fmt.Println("  - 3.0.x: Stable, widely supported")
	fmt.Println("  - 3.1.x: JSON Schema 2020-12, webhooks, type arrays")
	fmt.Println("  - 3.2.x: Latest features and refinements")
	fmt.Println()

	// Demo 1: Upgrade 3.0 -> 3.1
	fmt.Println("[1/4] Upgrade: OAS 3.0 -> 3.1")
	fmt.Println("--------------------------------")
	demonstrateConversion(classicPath, "3.1.0", false)

	// Demo 2: Upgrade 3.0 -> 3.2 (latest)
	fmt.Println()
	fmt.Println("[2/4] Upgrade: OAS 3.0 -> 3.2 (latest)")
	fmt.Println("--------------------------------")
	demonstrateConversion(classicPath, "3.2.0", false)

	// Demo 3: Downgrade 3.1 -> 3.0 (potentially lossy)
	fmt.Println()
	fmt.Println("[3/4] Downgrade: OAS 3.1 -> 3.0 (potentially lossy)")
	fmt.Println("--------------------------------")
	demonstrateConversion(modernPath, "3.0.3", true)

	// Demo 4: Downgrade 3.1 -> 2.0 (lossy - webhooks lost!)
	fmt.Println()
	fmt.Println("[4/4] Downgrade: OAS 3.1 -> 2.0 (lossy!)")
	fmt.Println("--------------------------------")
	demonstrateConversion(modernPath, "2.0", true)

	fmt.Println()
	fmt.Println("=============================================")
	fmt.Println("Version Conversion Summary:")
	fmt.Println()
	fmt.Println("  Upgrades (safe):")
	fmt.Println("    3.0 -> 3.1: Gains webhooks, type arrays, JSON Schema 2020-12")
	fmt.Println("    3.0 -> 3.2: Gains all 3.1 features plus 3.2 refinements")
	fmt.Println("    3.1 -> 3.2: Minor refinements")
	fmt.Println()
	fmt.Println("  Downgrades (may lose features):")
	fmt.Println("    3.1 -> 3.0: Loses type arrays, some JSON Schema features")
	fmt.Println("    3.1 -> 2.0: Loses webhooks, components, many features")
	fmt.Println("    3.0 -> 2.0: Loses links, callbacks, components structure")
	fmt.Println()
	fmt.Println("  Tip: Always validate after conversion!")
}

func demonstrateConversion(specPath, targetVersion string, expectLossy bool) {
	// Step 1: Parse
	parsed, err := parser.ParseWithOptions(parser.WithFilePath(specPath))
	if err != nil {
		log.Printf("  Parse error: %v", err)
		return
	}
	fmt.Printf("  Source: %s (OAS %s)\n", filepath.Base(specPath), parsed.Version)
	fmt.Printf("  Target: OAS %s\n", targetVersion)
	fmt.Println()

	// Show source features if it's the modern spec
	if parsed.Version == "3.1.0" {
		doc, ok := parsed.Document.(*parser.OAS3Document)
		if ok {
			if doc.Webhooks != nil && len(doc.Webhooks) > 0 {
				fmt.Printf("  Source has %d webhook(s)\n", len(doc.Webhooks))
			}
			if doc.JSONSchemaDialect != "" {
				fmt.Printf("  Source uses JSON Schema dialect: %s\n", truncate(doc.JSONSchemaDialect, 40))
			}
			fmt.Println()
		}
	}

	// Step 2: Convert
	c := converter.New()
	result, err := c.ConvertParsed(*parsed, targetVersion)
	if err != nil {
		log.Printf("  [x] Convert error: %v", err)
		return
	}
	fmt.Printf("  [ok] Converted to OAS %s\n", result.TargetVersion)

	// Step 3: Check for issues by severity
	if len(result.Issues) > 0 {
		// Count by severity
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

		fmt.Printf("  Conversion issues: %d critical, %d warnings, %d info\n",
			criticalCount, warningCount, infoCount)

		// Show critical issues (lossy conversions)
		if criticalCount > 0 {
			fmt.Println("  Critical issues (features lost):")
			shown := 0
			for _, issue := range result.Issues {
				if issue.Severity == converter.SeverityCritical {
					fmt.Printf("      [!] %s: %s\n", issue.Path, truncate(issue.Message, 50))
					shown++
					if shown >= 3 {
						if criticalCount > 3 {
							fmt.Printf("      ... and %d more\n", criticalCount-3)
						}
						break
					}
				}
			}
		}

		// Show warnings
		if warningCount > 0 && !expectLossy {
			fmt.Println("  Warnings:")
			shown := 0
			for _, issue := range result.Issues {
				if issue.Severity == converter.SeverityWarning {
					fmt.Printf("      [!] %s\n", truncate(issue.Message, 55))
					shown++
					if shown >= 2 {
						break
					}
				}
			}
		}
	}

	// Check what was lost in downgrade
	if expectLossy {
		checkFeatureLoss(parsed, result)
	}

	// Step 4: Validate converted spec
	fmt.Println()
	fmt.Println("  Validating converted spec:")
	v := validator.New()
	validation, err := v.ValidateParsed(*result.ToParseResult())
	if err != nil {
		log.Printf("    Validate error: %v", err)
		return
	}

	if validation.Valid {
		fmt.Println("    [ok] Valid!")
	} else {
		fmt.Printf("    [x] %d validation errors\n", len(validation.Errors))
		for i, e := range validation.Errors {
			if i >= 2 {
				break
			}
			fmt.Printf("      - %s\n", truncate(e.Message, 55))
		}
	}
}

func checkFeatureLoss(source *parser.ParseResult, result *converter.ConversionResult) {
	// Check for webhook loss
	sourceDoc, ok := source.Document.(*parser.OAS3Document)
	if !ok {
		return
	}

	sourceWebhooks := 0
	if sourceDoc.Webhooks != nil {
		sourceWebhooks = len(sourceDoc.Webhooks)
	}

	// Check target based on version
	if result.TargetVersion == "2.0" {
		if sourceWebhooks > 0 {
			fmt.Printf("  [!] LOST: %d webhook(s) (not supported in OAS 2.0)\n", sourceWebhooks)
		}
		fmt.Println("  [!] LOST: components structure (converted to definitions)")
		return
	}

	// For 3.x targets
	targetDoc, ok := result.Document.(*parser.OAS3Document)
	if !ok {
		return
	}

	targetWebhooks := 0
	if targetDoc.Webhooks != nil {
		targetWebhooks = len(targetDoc.Webhooks)
	}

	if sourceWebhooks > 0 && targetWebhooks == 0 {
		fmt.Printf("  [!] LOST: %d webhook(s)\n", sourceWebhooks)
	} else if sourceWebhooks > 0 && targetWebhooks > 0 {
		fmt.Printf("  [ok] Preserved: %d webhook(s)\n", targetWebhooks)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func findSpecPath(relativePath string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Unable to get current file path")
	}
	return filepath.Join(filepath.Dir(filename), relativePath)
}
