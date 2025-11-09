package parser_test

import (
	"fmt"
	"log"
	"strings"

	"github.com/erraggy/oastools/internal/parser"
)

// Example demonstrates basic usage of the parser to parse an OpenAPI specification file.
func Example() {
	p := parser.New()

	result, err := p.Parse("../../testdata/petstore-3.0.yaml")
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}

	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Has errors: %v\n", len(result.Errors) > 0)

	// Output:
	// Version: 3.0.3
	// Has errors: false
}

// Example_parseWithValidation demonstrates parsing with structure validation enabled.
func Example_parseWithValidation() {
	p := parser.New()
	p.ValidateStructure = true

	result, err := p.Parse("../../testdata/petstore-3.0.yaml")
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}

	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Validation errors: %d\n", len(result.Errors))

	// Output:
	// Version: 3.0.3
	// Validation errors: 0
}

// Example_parseWithRefs demonstrates parsing with reference resolution enabled.
func Example_parseWithRefs() {
	p := parser.New()
	p.ResolveRefs = true

	result, err := p.Parse("../../testdata/with-external-refs.yaml")
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}

	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Has warnings: %v\n", len(result.Warnings) > 0)

	// Output:
	// Version: 3.0.3
	// Has warnings: false
}

// Example_parseBytes demonstrates parsing an OpenAPI specification from a byte slice.
func Example_parseBytes() {
	specData := `
openapi: 3.0.3
info:
  title: Simple API
  version: 1.0.0
paths: {}
`

	p := parser.New()
	result, err := p.ParseBytes([]byte(specData))
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}

	fmt.Printf("Version: %s\n", result.Version)

	// Output:
	// Version: 3.0.3
}

// Example_parseReader demonstrates parsing an OpenAPI specification from an io.Reader.
func Example_parseReader() {
	specData := `
openapi: 3.0.3
info:
  title: Simple API
  version: 1.0.0
paths: {}
`

	reader := strings.NewReader(specData)

	p := parser.New()
	result, err := p.ParseReader(reader)
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}

	fmt.Printf("Version: %s\n", result.Version)

	// Output:
	// Version: 3.0.3
}

// Example_oas2 demonstrates parsing an OpenAPI 2.0 (Swagger) specification.
func Example_oas2() {
	p := parser.New()

	result, err := p.Parse("../../testdata/petstore-2.0.yaml")
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}

	fmt.Printf("Version: %s\n", result.Version)

	// Type assertion to access OAS 2.0-specific document structure
	if doc, ok := result.Document.(*parser.OAS2Document); ok {
		fmt.Printf("Swagger: %s\n", doc.Swagger)
	}

	// Output:
	// Version: 2.0
	// Swagger: 2.0
}

// Example_oas3 demonstrates parsing an OpenAPI 3.x specification and accessing version-specific fields.
func Example_oas3() {
	p := parser.New()

	result, err := p.Parse("../../testdata/petstore-3.0.yaml")
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}

	fmt.Printf("Version: %s\n", result.Version)

	// Type assertion to access OAS 3.x-specific document structure
	if doc, ok := result.Document.(*parser.OAS3Document); ok {
		fmt.Printf("OpenAPI: %s\n", doc.OpenAPI)
		fmt.Printf("Has paths: %v\n", doc.Paths != nil)
	}

	// Output:
	// Version: 3.0.3
	// OpenAPI: 3.0.3
	// Has paths: true
}
