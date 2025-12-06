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
		// Simple diff using functional options
		result, err := differ.DiffWithOptions(
			differ.WithSourceFilePath("api-v1.yaml"),
			differ.WithTargetFilePath("api-v2.yaml"),
		)
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
		// Diff with breaking mode using functional options
		result, err := differ.DiffWithOptions(
			differ.WithSourceFilePath("api-v1.yaml"),
			differ.WithTargetFilePath("api-v2.yaml"),
			differ.WithMode(differ.ModeBreaking),
			differ.WithIncludeInfo(true),
		)
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

For efficiency when documents are already parsed, use WithSourceParsed and WithTargetParsed:

	package main

	import (
		"fmt"
		"log"

		"github.com/erraggy/oastools/differ"
		"github.com/erraggy/oastools/parser"
	)

	func main() {
		// Parse documents once
		source, err := parser.ParseWithOptions(
			parser.WithFilePath("api-v1.yaml"),
			parser.WithValidateStructure(true),
		)
		if err != nil {
			log.Fatal(err)
		}

		target, err := parser.ParseWithOptions(
			parser.WithFilePath("api-v2.yaml"),
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

# Coverage Details

The differ provides comprehensive comparison of OpenAPI specification elements:

# Response Comparison

Response objects are fully compared including:
  - Headers: All header properties (description, required, deprecated, type, style, schema)
  - Content/MediaTypes: Media type objects and their schemas
  - Links: Link objects (operationRef, operationId, description)
  - Examples: Example map keys (not deep value comparison)
  - Extensions: All x-* fields on Response objects

Header comparison includes:
  - Description and deprecation status
  - Required flag changes
  - Type and style modifications
  - Schema changes (delegates to schema comparison)
  - Extensions on Header objects

MediaType comparison includes:
  - Schema changes (delegates to comprehensive schema comparison)
  - Extensions on MediaType objects

Link comparison includes:
  - Operation references (operationRef, operationId)
  - Description changes
  - Extensions on Link objects

# Schema Comparison

Schema objects are comprehensively compared including all fields:

Metadata:
  - title, description

Type information:
  - type, format

Numeric constraints:
  - multipleOf, maximum, exclusiveMaximum, minimum, exclusiveMinimum

String constraints:
  - maxLength, minLength, pattern

Array constraints:
  - maxItems, minItems, uniqueItems

Object constraints:
  - maxProperties, minProperties
  - required fields (with smart severity: adding required=ERROR, removing=INFO)

OAS-specific fields:
  - nullable, readOnly, writeOnly, deprecated

Schema comparison uses smart severity assignment in breaking mode:
  - ERROR: Stricter constraints (adding required fields, lowering max values, raising min values)
  - WARNING: Changes that might affect consumers (type changes, constraint modifications)
  - INFO: Relaxations and non-breaking changes (removing required, raising max, lowering min)

Note: Recursive schema properties (properties, items, allOf, oneOf, anyOf, not) are
compared separately to avoid cyclic comparison issues.

# Extension (x-*) Field Coverage

The OpenAPI Specification allows custom extension fields (starting with "x-")
at many levels of the document. The differ detects and reports changes to
extensions at commonly-used locations:

Extensions ARE diffed for these types:
  - Document level (OAS2Document, OAS3Document)
  - Info object
  - Server objects
  - PathItem objects
  - Operation objects
  - Parameter objects
  - RequestBody objects
  - Response objects
  - Header objects (response headers)
  - Link objects (response links)
  - MediaType objects (content types)
  - Schema objects
  - SecurityScheme objects
  - Tag objects
  - Components object

Extensions are NOT currently diffed for these less commonly-used types:
  - Contact, License, ExternalDocs (nested within Info)
  - ServerVariable (nested within Server)
  - Reference objects
  - Items (OAS 2.0 array item definitions)
  - Example, Encoding (response-related nested objects)
  - Discriminator, XML (schema-related nested objects)
  - OAuthFlows, OAuthFlow (security-related nested objects)

The rationale for this selective coverage is that extensions are most commonly
placed at document, path, operation, parameter, response, and schema levels where
they provide cross-cutting metadata. Extensions in deeply nested objects like
ServerVariable, Discriminator, or Example are rare in practice.

If your use case requires extension diffing for the uncovered types, please
open an issue at https://github.com/erraggy/oastools/issues

All extension changes are reported with CategoryExtension and are assigned
SeverityInfo in breaking mode, as specification extensions are non-normative
and optional according to the OpenAPI Specification.

# Related Packages

The differ integrates with other oastools packages:
  - [github.com/erraggy/oastools/parser] - Parse specifications before diffing
  - [github.com/erraggy/oastools/validator] - Validate specifications before comparison
  - [github.com/erraggy/oastools/converter] - Convert versions before comparing different OAS versions
  - [github.com/erraggy/oastools/joiner] - Join specifications before comparison
  - [github.com/erraggy/oastools/generator] - Generate code from compared specifications
  - [github.com/erraggy/oastools/builder] - Programmatically build specifications to compare
*/
package differ
