package validator_test

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
)

// ExampleValidator_Validate demonstrates basic validation of an OpenAPI specification
func ExampleValidator_Validate() {
	v := validator.New()
	testFile := filepath.Join("testdata", "petstore-3.0.yaml")
	result, err := v.Validate(testFile)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}
	fmt.Printf("Valid: %v\n", result.Valid)
	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Errors: %d\n", result.ErrorCount)
	fmt.Printf("Warnings: %d\n", result.WarningCount)
}

// ExampleValidator_Validate_strictMode demonstrates validation with strict mode enabled
func ExampleValidator_Validate_strictMode() {
	v := validator.New()
	v.StrictMode = true
	testFile := filepath.Join("testdata", "petstore-3.0.yaml")
	result, err := v.Validate(testFile)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}
	fmt.Printf("Valid: %v\n", result.Valid)
	fmt.Printf("Errors: %d\n", result.ErrorCount)
	fmt.Printf("Warnings: %d\n", result.WarningCount)
}

// Example_customValidation demonstrates how to use the validator with
// custom options for best practice warnings and strict validation.
func Example_customValidation() {
	// Create a validator with strict mode and warnings enabled
	v := validator.New()
	v.IncludeWarnings = true // Include best-practice warnings
	v.StrictMode = true      // Enforce strict validation rules

	// Validate an OpenAPI specification
	testFile := filepath.Join("testdata", "petstore-3.0.yaml")
	result, err := v.Validate(testFile)
	if err != nil {
		log.Fatal(err)
	}

	// Separate errors from warnings by severity
	errors := 0
	warnings := 0

	for _, issue := range result.Errors {
		// Severity levels: SeverityCritical=3, SeverityError=2, SeverityWarning=1, SeverityInfo=0
		switch issue.Severity {
		case validator.SeverityCritical:
			errors++ // Critical issues are also errors
			fmt.Printf("CRITICAL [%s]: %s\n", issue.Path, issue.Message)
		case validator.SeverityError:
			errors++
			fmt.Printf("ERROR [%s]: %s\n", issue.Path, issue.Message)
		case validator.SeverityWarning:
			warnings++
			fmt.Printf("WARNING [%s]: %s\n", issue.Path, issue.Message)
		case validator.SeverityInfo:
			fmt.Printf("INFO [%s]: %s\n", issue.Path, issue.Message)
		}
	}

	// Summary
	fmt.Printf("\nValidation complete:\n")
	fmt.Printf("- Valid: %v\n", result.Valid)
	fmt.Printf("- Errors: %d\n", errors)
	fmt.Printf("- Warnings: %d\n", warnings)

	// StrictMode treats warnings as errors for result.Valid
	// IncludeWarnings populates result.Errors with warning-level issues
}

// Example_toParseResult demonstrates using ToParseResult for package chaining.
// This pattern allows validated documents to be passed to other oastools packages.
func Example_toParseResult() {
	// Validate a specification
	v := validator.New()
	testFile := filepath.Join("testdata", "petstore-3.0.yaml")
	result, err := v.Validate(testFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Valid: %v\n", result.Valid)

	// Convert to ParseResult for use with other packages
	parseResult := result.ToParseResult()
	fmt.Printf("SourcePath: %s\n", parseResult.SourcePath)
	fmt.Printf("Version: %s\n", parseResult.Version)

	// The ParseResult can now be passed to fixer, converter, joiner, etc.
	// For example:
	//   fixResult, _ := fixer.FixWithOptions(fixer.WithParsed(*parseResult))
	//   convertResult, _ := converter.ConvertWithOptions(converter.WithParsed(*parseResult), ...)
}

// ExampleValidator_ValidateStructure demonstrates controlling parser
// structure validation. When disabled, the parser is more lenient about malformed
// documents, allowing the validator to focus on semantic validation only.
func ExampleValidator_ValidateStructure() {
	// A valid OpenAPI spec to test with
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        '200':
          description: Success
`
	// Parse the spec first
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	if err != nil {
		log.Fatalf("Parse failed: %v", err)
	}

	// Using the struct-based API to control ValidateStructure
	// Validate with structure validation enabled (default)
	v1 := validator.New()
	v1.ValidateStructure = true // This is the default
	result, err := v1.ValidateParsed(*parseResult)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}
	fmt.Printf("With structure validation: Valid=%v\n", result.Valid)

	// Validate with structure validation disabled (more lenient parsing)
	v2 := validator.New()
	v2.ValidateStructure = false
	result2, err := v2.ValidateParsed(*parseResult)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}
	fmt.Printf("Without structure validation: Valid=%v\n", result2.Valid)

	// Output:
	// With structure validation: Valid=true
	// Without structure validation: Valid=true
}

// ExampleValidateWithOptions_validateStructure demonstrates the functional option
// for controlling parser structure validation.
func ExampleValidateWithOptions_validateStructure() {
	// A valid OpenAPI spec to test with
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        '200':
          description: Success
`
	// Parse the spec first
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	if err != nil {
		log.Fatal(err)
	}

	// Validate with structure validation disabled using functional options
	result, err := validator.ValidateWithOptions(
		validator.WithParsed(*parseResult),
		validator.WithValidateStructure(false),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Valid: %v\n", result.Valid)

	// Output:
	// Valid: true
}

// Example_operationContext demonstrates how validation errors include
// operation context to help identify which API operation an error relates to.
// This is especially useful for errors in shared components (schemas, responses)
// that are referenced by multiple operations.
func Example_operationContext() {
	// A spec with a validation error in an operation
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      parameters:
        - name: wrongName
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	if err != nil {
		log.Fatal(err)
	}

	result, err := validator.ValidateWithOptions(
		validator.WithParsed(*parseResult),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Validation errors include operation context
	for _, e := range result.Errors {
		// The String() method includes operation context automatically
		fmt.Println(e.String())

		// You can also access the OperationContext programmatically
		if e.OperationContext != nil {
			fmt.Printf("  Operation: %s %s\n", e.OperationContext.Method, e.OperationContext.Path)
			if e.OperationContext.OperationID != "" {
				fmt.Printf("  OperationId: %s\n", e.OperationContext.OperationID)
			}
		}
	}

	// Output:
	// ✗ paths./users/{userId}.get (operationId: getUser): Path template references parameter '{userId}' but it is not declared in parameters
	//     Spec: https://spec.openapis.org/oas/v3.0.0.html#path-item-object
	//   Operation: GET /users/{userId}
	//   OperationId: getUser
}
