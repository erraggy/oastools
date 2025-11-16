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
