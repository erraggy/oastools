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

// Example_functionalOptions demonstrates parsing using functional options.
func Example_functionalOptions() {
	result, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
		parser.WithValidateStructure(true),
		parser.WithResolveRefs(false),
	)
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}
	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Format: %s\n", result.SourceFormat)
	// Output:
	// Version: 3.0.3
	// Format: yaml
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

// Example_parseWithHTTPRefs demonstrates parsing with HTTP/HTTPS $ref resolution.
// This is useful for specifications that reference external schemas via URLs.
// HTTP resolution is opt-in for security (prevents SSRF attacks).
func Example_parseWithHTTPRefs() {
	// Enable HTTP reference resolution (opt-in for security)
	result, err := parser.ParseWithOptions(
		parser.WithFilePath("spec-with-http-refs.yaml"),
		parser.WithResolveRefs(true),
		parser.WithResolveHTTPRefs(true), // Enable HTTP $ref resolution
		// parser.WithInsecureSkipVerify(true), // For self-signed certs (dev only)
	)
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}

	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Errors: %d\n", len(result.Errors))

	// HTTP responses are cached, size-limited, and protected against circular refs
}

// Example_parseFromURL demonstrates parsing a specification directly from a URL.
func Example_parseFromURL() {
	result, err := parser.ParseWithOptions(
		parser.WithFilePath("https://petstore.swagger.io/v2/swagger.json"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}

	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Format: %s\n", result.SourceFormat)
}

// Example_reusableParser demonstrates creating a reusable parser instance
// for processing multiple files with the same configuration.
func Example_reusableParser() {
	// Configure parser once
	p := parser.New()
	p.ResolveRefs = true
	p.ValidateStructure = true
	p.ResolveHTTPRefs = false // Keep HTTP refs disabled for security

	// Parse multiple files with same config
	files := []string{
		"../testdata/petstore-3.0.yaml",
		"../testdata/petstore-2.0.yaml",
	}

	for _, file := range files {
		result, err := p.Parse(file)
		if err != nil {
			log.Printf("Error parsing %s: %v", file, err)
			continue
		}
		fmt.Printf("%s: version=%s, errors=%d\n",
			file, result.Version, len(result.Errors))
	}
}

// Example_deepCopy demonstrates using DeepCopy to create independent copies
// of parsed documents. This is useful when you need to modify a document
// without affecting the original (e.g., in fixers or converters).
func Example_deepCopy() {
	result, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Type assert to get the OAS3 document
	original, ok := result.Document.(*parser.OAS3Document)
	if !ok {
		log.Fatal("expected OAS3 document")
	}

	// Create a deep copy of the document
	docCopy := original.DeepCopy()

	// Modify the copy without affecting the original
	docCopy.Info.Title = "Modified Petstore API"

	fmt.Printf("Original title: %s\n", original.Info.Title)
	fmt.Printf("Copy title: %s\n", docCopy.Info.Title)
	// Output:
	// Original title: Petstore API
	// Copy title: Modified Petstore API
}
