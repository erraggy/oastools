package validator_test

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/erraggy/oastools/internal/validator"
)

// ExampleValidator_Validate demonstrates basic validation of an OpenAPI specification
func ExampleValidator_Validate() {
	// Create a new validator
	v := validator.New()

	// Validate a specification file
	testFile := filepath.Join("testdata", "petstore-3.0.yaml")
	result, err := v.Validate(testFile)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	// Check the results
	fmt.Printf("Valid: %v\n", result.Valid)
	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Errors: %d\n", result.ErrorCount)
	fmt.Printf("Warnings: %d\n", result.WarningCount)
}

// ExampleValidator_Validate_strictMode demonstrates validation with strict mode enabled
func ExampleValidator_Validate_strictMode() {
	// Create a validator with strict mode
	v := validator.New()
	v.StrictMode = true

	testFile := filepath.Join("testdata", "petstore-3.0.yaml")
	result, err := v.Validate(testFile)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	// Strict mode may produce additional warnings
	fmt.Printf("Valid: %v\n", result.Valid)
	fmt.Printf("Errors: %d\n", result.ErrorCount)
	fmt.Printf("Warnings: %d\n", result.WarningCount)
}

// ExampleValidator_Validate_noWarnings demonstrates validation with warnings suppressed
func ExampleValidator_Validate_noWarnings() {
	// Create a validator that suppresses warnings
	v := validator.New()
	v.IncludeWarnings = false

	testFile := filepath.Join("testdata", "petstore-3.0.yaml")
	result, err := v.Validate(testFile)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	// Warnings will not be included in the result
	fmt.Printf("Valid: %v\n", result.Valid)
	fmt.Printf("Warnings count: %d\n", result.WarningCount)
}

// ExampleValidationError_String demonstrates formatting of validation errors
func ExampleValidationError_String() {
	err := validator.ValidationError{
		Path:     "paths./pets.get.responses",
		Message:  "Missing required field 'responses'",
		SpecRef:  "https://spec.openapis.org/oas/v3.0.0.html#operation-object",
		Severity: validator.SeverityError,
		Field:    "responses",
	}

	fmt.Println(err.String())
	// Output will show formatted error with path, message, and spec reference
}

// ExampleNew demonstrates creating a new validator with default settings
func ExampleNew() {
	// Create a validator with default settings
	v := validator.New()

	// Default settings
	fmt.Printf("Include warnings: %v\n", v.IncludeWarnings)
	fmt.Printf("Strict mode: %v\n", v.StrictMode)

	// Output:
	// Include warnings: true
	// Strict mode: false
}

// ExampleValidationResult demonstrates working with validation results
func ExampleValidationResult() {
	v := validator.New()
	testFile := filepath.Join("testdata", "invalid-oas3.yaml")

	result, err := v.Validate(testFile)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	// Check if document is valid
	if !result.Valid {
		fmt.Printf("Validation failed with %d error(s):\n", result.ErrorCount)
		for i, validationErr := range result.Errors {
			fmt.Printf("%d. %s: %s\n", i+1, validationErr.Path, validationErr.Message)
			if validationErr.SpecRef != "" {
				fmt.Printf("   See: %s\n", validationErr.SpecRef)
			}
		}
	}

	// Show warnings if any
	if result.WarningCount > 0 {
		fmt.Printf("\nWarnings (%d):\n", result.WarningCount)
		for i, warning := range result.Warnings {
			fmt.Printf("%d. %s: %s\n", i+1, warning.Path, warning.Message)
		}
	}
}

// ExampleValidator_Validate_oas2 demonstrates validating an OAS 2.0 specification
func ExampleValidator_Validate_oas2() {
	v := validator.New()
	testFile := filepath.Join("testdata", "petstore-2.0.yaml")

	result, err := v.Validate(testFile)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	fmt.Printf("Valid OAS 2.0 document: %v\n", result.Valid)
	fmt.Printf("Version: %s\n", result.Version)
}

// ExampleValidator_Validate_multipleVersions demonstrates validating different OAS versions
func ExampleValidator_Validate_multipleVersions() {
	v := validator.New()

	files := []string{
		"petstore-2.0.yaml",
		"petstore-3.0.yaml",
		"petstore-3.1.yaml",
		"petstore-3.2.yaml",
	}

	for _, file := range files {
		testFile := filepath.Join("testdata", file)
		result, err := v.Validate(testFile)
		if err != nil {
			log.Printf("Error validating %s: %v", file, err)
			continue
		}

		fmt.Printf("%s: Valid=%v, Version=%s, Errors=%d, Warnings=%d\n",
			file, result.Valid, result.Version, result.ErrorCount, result.WarningCount)
	}
}
