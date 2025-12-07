// Package fixer provides automatic fixes for common OpenAPI Specification validation errors.
//
// The fixer analyzes OAS documents and applies fixes for issues that would cause
// validation failures. It supports both OAS 2.0 and OAS 3.x documents. The fixer
// preserves the input file format (JSON or YAML) in the FixResult.SourceFormat
// field, allowing tools to maintain format consistency when writing output.
//
// # Quick Start
//
// Fix a file using functional options:
//
//	result, err := fixer.FixWithOptions(
//		fixer.WithFilePath("openapi.yaml"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Applied %d fixes\n", result.FixCount)
//
// Or use a reusable Fixer instance:
//
//	f := fixer.New()
//	f.InferTypes = true // Infer parameter types from naming conventions
//	result1, _ := f.Fix("api1.yaml")
//	result2, _ := f.Fix("api2.yaml")
//
// # Supported Fixes
//
// The fixer currently supports the following automatic fixes:
//
//   - Missing path parameters: Adds Parameter objects for path template variables
//     that are not declared in the operation's parameters list. For example, if a
//     path is "/users/{userId}" but the operation doesn't declare a "userId" path
//     parameter, the fixer adds one with type "string" (or inferred type if enabled).
//
// # Type Inference
//
// When InferTypes is enabled (--infer flag in CLI), the fixer uses naming conventions
// to determine parameter types:
//
//   - Names ending in "id", "Id", or "ID" -> integer
//   - Names containing "uuid" or "guid" -> string with format "uuid"
//   - All other names -> string
//
// # Pipeline Usage
//
// The fixer is designed to work in a pipeline with other oastools commands:
//
//	# Fix and validate
//	oastools fix api.yaml | oastools validate -q -
//
//	# Fix and save
//	oastools fix api.yaml -o fixed.yaml
//
// # Related Packages
//
// The fixer integrates with other oastools packages:
//   - [github.com/erraggy/oastools/parser] - Parse specifications before fixing
//   - [github.com/erraggy/oastools/validator] - Validate specifications (use to see errors)
//   - [github.com/erraggy/oastools/converter] - Convert between OAS versions
//   - [github.com/erraggy/oastools/joiner] - Join multiple specifications
//   - [github.com/erraggy/oastools/differ] - Compare specifications
//   - [github.com/erraggy/oastools/generator] - Generate code from specifications
//   - [github.com/erraggy/oastools/builder] - Programmatically build specifications
package fixer
