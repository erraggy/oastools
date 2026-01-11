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

// Example_documentAccessor demonstrates using the DocumentAccessor interface
// for version-agnostic access to OpenAPI documents. This allows writing code
// that works identically for both OAS 2.0 and OAS 3.x without type switches.
func Example_documentAccessor() {
	result, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Get version-agnostic accessor
	accessor := result.AsAccessor()
	if accessor == nil {
		log.Fatal("unsupported document type")
	}

	// Works identically for both OAS 2.0 and OAS 3.x
	fmt.Printf("API: %s\n", accessor.GetInfo().Title)
	fmt.Printf("Paths: %d\n", len(accessor.GetPaths()))
	fmt.Printf("Schemas: %d\n", len(accessor.GetSchemas()))
	fmt.Printf("Schema ref prefix: %s\n", accessor.SchemaRefPrefix())
	// Output:
	// API: Petstore API
	// Paths: 2
	// Schemas: 4
	// Schema ref prefix: #/components/schemas/
}

// Example_withSourceName demonstrates setting a meaningful source name when parsing
// from bytes or io.Reader. This is important when later joining documents, as the
// source name appears in collision reports and warnings.
func Example_withSourceName() {
	// When parsing from bytes (e.g., fetched from HTTP), the default source name
	// is "ParseBytes.yaml" which isn't helpful for collision reports.
	specData := []byte(`openapi: "3.0.0"
info:
  title: Users API
  version: "1.0"
paths:
  /users:
    get:
      summary: List users
      responses:
        '200':
          description: OK
`)

	// Use WithSourceName to set a meaningful identifier
	result, err := parser.ParseWithOptions(
		parser.WithBytes(specData),
		parser.WithSourceName("users-api"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// The source name is now "users-api" instead of "ParseBytes.yaml"
	fmt.Printf("Source: %s\n", result.SourcePath)
	fmt.Printf("Version: %s\n", result.Version)
	// Output:
	// Source: users-api
	// Version: 3.0.0
}

// Example_documentTypeHelpers demonstrates using the type assertion helper methods
// to safely extract version-specific documents and check document versions.
func Example_documentTypeHelpers() {
	// Parse an OAS 3.x document
	result, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Use IsOAS2/IsOAS3 for version checking without type assertions
	fmt.Printf("Is OAS 2.0: %v\n", result.IsOAS2())
	fmt.Printf("Is OAS 3.x: %v\n", result.IsOAS3())

	// Use OAS3Document for safe type assertion
	if doc, ok := result.OAS3Document(); ok {
		fmt.Printf("API Title: %s\n", doc.Info.Title)
	}
	// Output:
	// Is OAS 2.0: false
	// Is OAS 3.x: true
	// API Title: Petstore API
}

// ExampleParseResult_Equals demonstrates comparing two ParseResults for semantic equality.
// This is useful for testing, caching, or detecting specification changes.
func ExampleParseResult_Equals() {
	// Parse the same specification twice
	result1, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
	)
	if err != nil {
		log.Fatal(err)
	}

	result2, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Compare for semantic equality (ignores metadata like LoadTime, SourcePath)
	fmt.Printf("Same content: %v\n", result1.Equals(result2))

	// Parse a different specification
	result3, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-2.0.yaml"),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Different specs: %v\n", result1.Equals(result3))
	// Output:
	// Same content: true
	// Different specs: false
}

// ExampleParseResult_DocumentEquals demonstrates comparing documents ignoring version metadata.
// This is useful when comparing specifications that may have been converted between versions.
func ExampleParseResult_DocumentEquals() {
	result1, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Create a copy with the same document
	result2, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// DocumentEquals compares only the document content
	fmt.Printf("Documents equal: %v\n", result1.DocumentEquals(result2))
	// Output:
	// Documents equal: true
}

// ExampleSchema_Equals demonstrates comparing two Schema objects for structural equality.
// This is useful for detecting schema changes or deduplicating identical schemas.
func ExampleSchema_Equals() {
	// Create two identical schemas
	schema1 := &parser.Schema{
		Type:        "object",
		Description: "A pet in the store",
		Properties: map[string]*parser.Schema{
			"id":   {Type: "integer", Format: "int64"},
			"name": {Type: "string"},
		},
		Required: []string{"id", "name"},
	}

	schema2 := &parser.Schema{
		Type:        "object",
		Description: "A pet in the store",
		Properties: map[string]*parser.Schema{
			"id":   {Type: "integer", Format: "int64"},
			"name": {Type: "string"},
		},
		Required: []string{"id", "name"},
	}

	fmt.Printf("Schemas equal: %v\n", schema1.Equals(schema2))

	// Modify one schema
	schema2.Description = "A different description"
	fmt.Printf("After modification: %v\n", schema1.Equals(schema2))
	// Output:
	// Schemas equal: true
	// After modification: false
}

// ExampleOAS3Document_Equals demonstrates comparing two OAS 3.x documents for equality.
func ExampleOAS3Document_Equals() {
	result, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
	)
	if err != nil {
		log.Fatal(err)
	}

	doc, ok := result.OAS3Document()
	if !ok {
		log.Fatal("expected OAS3 document")
	}

	// DeepCopy creates an identical document
	docCopy := doc.DeepCopy()
	fmt.Printf("Copy equals original: %v\n", doc.Equals(docCopy))

	// Modify the copy
	docCopy.Info.Title = "Modified API"
	fmt.Printf("After modification: %v\n", doc.Equals(docCopy))
	// Output:
	// Copy equals original: true
	// After modification: false
}

// ExampleOAS2Document_Equals demonstrates comparing two OAS 2.0 (Swagger) documents for equality.
func ExampleOAS2Document_Equals() {
	result, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-2.0.yaml"),
	)
	if err != nil {
		log.Fatal(err)
	}

	doc, ok := result.OAS2Document()
	if !ok {
		log.Fatal("expected OAS2 document")
	}

	// DeepCopy creates an identical document
	docCopy := doc.DeepCopy()
	fmt.Printf("Copy equals original: %v\n", doc.Equals(docCopy))

	// Modify the copy
	docCopy.Info.Title = "Modified Swagger API"
	fmt.Printf("After modification: %v\n", doc.Equals(docCopy))
	// Output:
	// Copy equals original: true
	// After modification: false
}
