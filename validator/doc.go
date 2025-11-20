// Package validator provides validation for OpenAPI Specification documents.
//
// The validator supports OAS 2.0 through OAS 3.2.0, performing structural validation,
// format checking, and semantic analysis against the specification requirements.
// It includes best practice warnings and strict mode for enhanced validation.
//
// # Quick Start
//
// Validate a file with warnings enabled:
//
//	result, err := validator.Validate("openapi.yaml", true, false)
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
// # Validation Output
//
// ValidationResult contains:
//   - Valid: Boolean indicating if document is valid
//   - Version: Detected OpenAPI version (e.g., "3.0.3")
//   - Errors: Validation errors with JSON path locations
//   - Warnings: Best practice warnings (if IncludeWarnings is true)
//   - ErrorCount, WarningCount: Issue counts
//
// See the exported ValidationError and ValidationResult types for complete details.
package validator
