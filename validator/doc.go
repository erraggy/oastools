// Package validator provides validation for OpenAPI Specification documents.
//
// The validator supports OAS 2.0 through OAS 3.2.0, performing structural validation,
// format checking, and semantic analysis against the specification requirements.
// It includes best practice warnings and strict mode for enhanced validation.
//
// # Quick Start
//
// Validate a file with warnings enabled using functional options:
//
//	result, err := validator.ValidateWithOptions(
//		validator.WithFilePath("openapi.yaml"),
//		validator.WithIncludeWarnings(true),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	if !result.Valid {
//		fmt.Printf("Found %d error(s)\n", result.ErrorCount)
//	}
//
// Or create a reusable validator instance:
//
//	v := validator.New()
//	v.StrictMode = true
//	result, _ := v.Validate("api1.yaml")
//	result, _ := v.Validate("api2.yaml")
//
// # Features
//
// The validator checks document structure, required fields, format validation
// (URLs, emails, media types), semantic constraints (operation ID uniqueness,
// parameter consistency), and JSON schemas. Path validation includes checking
// for malformed templates (unclosed braces, reserved characters, consecutive slashes)
// and REST best practices (trailing slashes generate warnings when IncludeWarnings
// is enabled). See the examples in example_test.go for more usage patterns.
//
// Schema validation reaches every position a Schema Object can occupy, including
// schemas in responses, parameters, headers, encoding headers and callbacks, not
// only those under components.
//
// # Version-Specific Fields
//
// A field is reported when the document's declared version does not define it.
// A document declaring 3.0.x or 3.1.x that uses an OAS 3.2 fixed field gets an
// error, naming the field and the version that introduced it. The gate covers
// the 3.2 additions across Path Item, Operation, Response, Media Type, Encoding,
// Example, Tag, Server, Security Scheme and the Schema Object's dialect fields.
//
// This is deliberately an error rather than a warning: the document declares a
// version whose consumers have no way to interpret the field, so the two
// statements in it contradict each other.
//
// Some rules also stop applying as versions advance, and those are gated the
// same way. An Operation without a responses field is an error at 3.0.x and
// permitted from 3.1, because 3.1 removed the requirement.
//
// # Validation Output
//
// ValidationResult contains:
//   - Valid: Boolean indicating if document is valid
//   - Version: Detected OpenAPI version (e.g., "3.0.3")
//   - Errors: Validation errors with JSON path locations and optional OperationContext
//   - Warnings: Best practice warnings (if IncludeWarnings is true)
//   - ErrorCount, WarningCount: Issue counts
//   - Document: The validated document for chaining with other packages
//
// Each ValidationError includes an OperationContext field that identifies the affected
// API operation (HTTP method, path, operationId). This helps pinpoint which endpoint
// contains the error. See Example_operationContext for usage.
//
// See the exported ValidationError and ValidationResult types for complete details.
//
// # Package Chaining with ToParseResult
//
// The ValidationResult.ToParseResult() method enables chaining validation with other
// oastools packages. This converts the validation result back to a ParseResult that
// can be passed to fixer, converter, joiner, or differ:
//
//	// Parse and validate, then fix any issues
//	parseResult, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
//	valResult, _ := validator.ValidateWithOptions(validator.WithParsed(*parseResult))
//	if !valResult.Valid {
//	    fixResult, _ := fixer.FixWithOptions(fixer.WithParsed(*valResult.ToParseResult()))
//	    // Use fixResult...
//	}
//
// Validation errors and warnings are converted to string warnings in the ParseResult
// with severity prefixes for programmatic filtering:
//   - "[error] path: message" for validation errors
//   - "[warning] path: message" for validation warnings
//
// # Related Packages
//
// Validation typically follows parsing:
//   - [github.com/erraggy/oastools/parser] - Parse specifications before validation
//   - [github.com/erraggy/oastools/fixer] - Fix common validation errors automatically
//   - [github.com/erraggy/oastools/converter] - Convert validated specs between OAS versions
//   - [github.com/erraggy/oastools/joiner] - Join validated specs into one document
//   - [github.com/erraggy/oastools/differ] - Compare specifications and detect breaking changes
//   - [github.com/erraggy/oastools/generator] - Generate Go code from validated specifications
//   - [github.com/erraggy/oastools/builder] - Programmatically build specifications
package validator
