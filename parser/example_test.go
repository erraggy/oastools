package parser_test

import (
	"fmt"
	"log"

	"github.com/erraggy/oastools/parser"
)

// Example demonstrates basic usage of the parser to parse an OpenAPI specification file.
func Example() {
	p := parser.New()
	result, err := p.Parse("../testdata/petstore-3.0.yaml")
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}
	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Has errors: %v\n", len(result.Errors) > 0)
	// Output:
	// Version: 3.0.3
	// Has errors: false
}

// Example_parseWithRefs demonstrates parsing with external reference resolution enabled.
func Example_parseWithRefs() {
	p := parser.New()
	p.ResolveRefs = true
	result, err := p.Parse("../testdata/with-external-refs.yaml")
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}
	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Has warnings: %v\n", len(result.Warnings) > 0)
	// Output:
	// Version: 3.0.3
	// Has warnings: false
}
