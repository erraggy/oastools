package validator_test

import (
	"fmt"
	"log"
	"path/filepath"

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
		// In validator, Severity is an int (SeverityError=2, SeverityWarning=1)
		switch issue.Severity {
		case validator.SeverityError:
			errors++
			fmt.Printf("ERROR [%s]: %s\n", issue.Path, issue.Message)
		case validator.SeverityWarning:
			warnings++
			fmt.Printf("WARNING [%s]: %s\n", issue.Path, issue.Message)
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
