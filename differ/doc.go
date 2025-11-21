/*
Package differ provides OpenAPI specification comparison and breaking change detection.

# Overview

The differ package enables comparison of OpenAPI specifications to identify differences,
categorize changes, and detect breaking API changes. It supports both OAS 2.0 and OAS 3.x documents.

# Usage

The package provides two API styles:

 1. Package-level convenience functions for simple, one-off operations
 2. Struct-based API for reusable instances with custom configuration

# Diff Modes

The differ supports two operational modes:

  - ModeSimple: Reports all semantic differences without categorization
  - ModeBreaking: Categorizes changes by severity and identifies breaking changes

# Change Categories

Changes are categorized by the part of the specification that changed:

  - CategoryEndpoint: Path/endpoint changes
  - CategoryOperation: HTTP operation changes
  - CategoryParameter: Parameter changes
  - CategoryRequestBody: Request body changes
  - CategoryResponse: Response changes
  - CategorySchema: Schema/definition changes
  - CategorySecurity: Security scheme changes
  - CategoryServer: Server/host changes
  - CategoryInfo: Metadata changes

# Severity Levels

In ModeBreaking, changes are assigned severity levels:

  - SeverityCritical: Critical breaking changes (removed endpoints, operations)
  - SeverityError: Breaking changes (removed required parameters, type changes)
  - SeverityWarning: Potentially problematic changes (deprecated operations, new required fields)
  - SeverityInfo: Non-breaking changes (additions, relaxed constraints)

# Example (Simple Diff)

	package main

	import (
		"fmt"
		"log"

		"github.com/erraggy/oastools/differ"
	)

	func main() {
		// Simple diff using convenience function
		result, err := differ.Diff("api-v1.yaml", "api-v2.yaml")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Found %d changes\n", len(result.Changes))
		for _, change := range result.Changes {
			fmt.Println(change.String())
		}
	}

# Example (Breaking Change Detection)

	package main

	import (
		"fmt"
		"log"

		"github.com/erraggy/oastools/differ"
	)

	func main() {
		// Create differ with breaking mode
		d := differ.New()
		d.Mode = differ.ModeBreaking
		d.IncludeInfo = true

		result, err := d.Diff("api-v1.yaml", "api-v2.yaml")
		if err != nil {
			log.Fatal(err)
		}

		if result.HasBreakingChanges {
			fmt.Printf("⚠️  Found %d breaking change(s)!\n", result.BreakingCount)
		}

		fmt.Printf("Summary: %d breaking, %d warnings, %d info\n",
			result.BreakingCount, result.WarningCount, result.InfoCount)

		// Print changes grouped by severity
		for _, change := range result.Changes {
			fmt.Println(change.String())
		}
	}

# Example (Reusable Differ Instance)

	package main

	import (
		"fmt"
		"log"

		"github.com/erraggy/oastools/differ"
	)

	func main() {
		// Create a reusable differ instance
		d := differ.New()
		d.Mode = differ.ModeBreaking
		d.IncludeInfo = false // Skip informational changes

		// Compare multiple spec pairs with same configuration
		pairs := []struct{ old, new string }{
			{"api-v1.yaml", "api-v2.yaml"},
			{"api-v2.yaml", "api-v3.yaml"},
			{"api-v3.yaml", "api-v4.yaml"},
		}

		for _, pair := range pairs {
			result, err := d.Diff(pair.old, pair.new)
			if err != nil {
				log.Printf("Error comparing %s to %s: %v", pair.old, pair.new, err)
				continue
			}

			fmt.Printf("\n%s → %s:\n", pair.old, pair.new)
			if result.HasBreakingChanges {
				fmt.Printf("  ⚠️  %d breaking changes\n", result.BreakingCount)
			} else {
				fmt.Println("  ✓ No breaking changes")
			}
		}
	}

# Working with Parsed Documents

For efficiency when documents are already parsed, use DiffParsed:

	package main

	import (
		"fmt"
		"log"

		"github.com/erraggy/oastools/differ"
		"github.com/erraggy/oastools/parser"
	)

	func main() {
		// Parse documents once
		source, err := parser.Parse("api-v1.yaml", false, true)
		if err != nil {
			log.Fatal(err)
		}

		target, err := parser.Parse("api-v2.yaml", false, true)
		if err != nil {
			log.Fatal(err)
		}

		// Compare parsed documents
		result, err := differ.DiffParsed(*source, *target)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Found %d changes\n", len(result.Changes))
	}

# Change Analysis

The Change struct provides detailed information about each difference:

	for _, change := range result.Changes {
		fmt.Printf("Path: %s\n", change.Path)
		fmt.Printf("Type: %s\n", change.Type)
		fmt.Printf("Category: %s\n", change.Category)
		fmt.Printf("Severity: %s\n", change.Severity)
		fmt.Printf("Message: %s\n", change.Message)

		if change.OldValue != nil {
			fmt.Printf("Old value: %v\n", change.OldValue)
		}
		if change.NewValue != nil {
			fmt.Printf("New value: %v\n", change.NewValue)
		}
	}

# Breaking Change Examples

Common breaking changes detected in ModeBreaking:

  - Removed endpoints or operations (SeverityCritical)
  - Removed required parameters (SeverityCritical)
  - Changed parameter types (SeverityError)
  - Made optional parameters required (SeverityError)
  - Removed enum values (SeverityError)
  - Removed success response codes (SeverityError)
  - Removed schemas (SeverityError)
  - Changed authentication requirements (SeverityError)

# Non-Breaking Change Examples

Common non-breaking changes in ModeBreaking:

  - Added endpoints or operations (SeverityInfo)
  - Added optional parameters (SeverityInfo)
  - Made required parameters optional (SeverityInfo)
  - Added enum values (SeverityInfo)
  - Added response codes (SeverityInfo)
  - Documentation updates (SeverityInfo)

# Version Compatibility

The differ works with:

  - OAS 2.0 (Swagger) documents
  - OAS 3.0.x documents
  - OAS 3.1.x documents
  - OAS 3.2.x documents
  - Cross-version comparisons (with limitations)

When comparing documents of different OAS versions (e.g., 2.0 vs 3.0),
the diff is limited to common elements present in both versions.
*/
package differ
