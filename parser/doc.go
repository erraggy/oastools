// Package parser provides parsing for OpenAPI Specification documents.
//
// The parser supports OAS 2.0 through OAS 3.2.0 in YAML and JSON formats. It can
// resolve external references ($ref), validate structure, and preserve unknown
// fields for forward compatibility and extension properties. The parser can load
// specifications from local files or remote URLs (http:// or https://).
//
// # Quick Start
//
// Parse a file using functional options:
//
//	result, err := parser.ParseWithOptions(
//		parser.WithFilePath("openapi.yaml"),
//		parser.WithValidateStructure(true),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	if len(result.Errors) > 0 {
//		fmt.Printf("Parse errors: %d\n", len(result.Errors))
//	}
//
// Parse from a URL:
//
//	result, err := parser.ParseWithOptions(
//		parser.WithFilePath("https://example.com/api/openapi.yaml"),
//		parser.WithValidateStructure(true),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Or create a reusable Parser instance:
//
//	p := parser.New()
//	p.ResolveRefs = false
//	result1, _ := p.Parse("api1.yaml")
//	result2, _ := p.Parse("https://example.com/api2.yaml")
//
// # Features and Security
//
// The parser validates operation IDs, status codes, and HTTP status codes. For
// external references, it prevents path traversal attacks by restricting file
// access to the base directory and subdirectories. Reference resolution caches
// up to 100 documents by default to prevent memory exhaustion.
//
// HTTP/HTTPS $ref resolution is available via WithResolveHTTPRefs (opt-in for
// security). Use WithInsecureSkipVerify for self-signed certificates. HTTP
// responses are cached, size-limited, and protected against circular references.
// See the examples in example_test.go for more details.
//
// # Circular Reference Handling
//
// When the parser detects circular references during $ref resolution, it uses a
// "silent fallback" strategy: the affected $ref nodes remain unresolved, but
// parsing continues successfully. This ensures documents with circular references
// can still be used while maintaining safety.
//
// Circular references are detected when:
//   - A $ref points to an ancestor in the current resolution path
//   - The resolution depth exceeds MaxRefDepth (default: 100)
//
// When circular references are detected:
//   - The $ref node is left unresolved (preserves the "$ref" key)
//   - A warning is added to result.Warnings
//   - The document remains valid for most operations
//
// To check for circular reference warnings after parsing:
//
//	result, err := parser.ParseWithOptions(
//		parser.WithFilePath("openapi.yaml"),
//		parser.WithResolveRefs(true),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	for _, warning := range result.Warnings {
//		if strings.Contains(warning, "circular") {
//			fmt.Println("Document contains circular references")
//		}
//	}
//
// # Resource Limits
//
// The parser enforces configurable resource limits to prevent denial-of-service:
//
//   - MaxRefDepth: Maximum depth for nested $ref resolution (default: 100)
//   - MaxCachedDocuments: Maximum external documents to cache (default: 100)
//   - MaxFileSize: Maximum file size for external references (default: 10MB)
//
// Configure limits using functional options:
//
//	result, err := parser.ParseWithOptions(
//		parser.WithFilePath("openapi.yaml"),
//		parser.WithMaxRefDepth(50),           // Reduce max depth
//		parser.WithMaxCachedDocuments(200),   // Allow more cached docs
//		parser.WithMaxFileSize(20*1024*1024), // 20MB limit
//	)
//
// # OAS 3.2.0 Features
//
// OAS 3.2.0 introduces several new capabilities:
//   - $self: Document identity/base URI (OAS3Document.Self)
//   - Query method: New HTTP method (PathItem.Query)
//   - additionalOperations: Custom HTTP methods (PathItem.AdditionalOperations)
//   - mediaTypes: Reusable media type definitions (Components.MediaTypes)
//
// # JSON Schema 2020-12 Keywords
//
// The parser supports JSON Schema Draft 2020-12 keywords for OAS 3.1+:
//   - unevaluatedProperties/unevaluatedItems: Strict validation (can be bool, *Schema, or map)
//   - contentEncoding/contentMediaType/contentSchema: Encoded content validation
//   - prefixItems, contains, propertyNames, dependentSchemas, $defs: Advanced schema features
//
// # Array Index References
//
// JSON Pointer references support array indices per RFC 6901:
//
//	$ref: '#/paths/~1users/get/parameters/0/schema'
//
// # ParseResult Fields
//
// ParseResult includes the detected Version, OASVersion, SourceFormat (JSON or YAML),
// and any parsing Errors or Warnings. The Document field contains the parsed OAS2Document
// or OAS3Document. The SourceFormat field can be used by conversion and joining tools to
// preserve the original file format. See the exported ParseResult and document type fields
// for complete details.
//
// # Document Type Helpers
//
// ParseResult provides convenient methods for version checking and type assertion:
//
//	result, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
//
//	// Version checking
//	if result.IsOAS2() {
//		fmt.Println("This is a Swagger 2.0 document")
//	}
//	if result.IsOAS3() {
//		fmt.Println("This is an OAS 3.x document")
//	}
//
//	// Safe type assertion
//	if doc, ok := result.OAS3Document(); ok {
//		fmt.Printf("API: %s v%s\n", doc.Info.Title, doc.Info.Version)
//	}
//	if doc, ok := result.OAS2Document(); ok {
//		fmt.Printf("Swagger: %s v%s\n", doc.Info.Title, doc.Info.Version)
//	}
//
// These helpers eliminate the need for manual type switches on the Document field.
//
// # Related Packages
//
// After parsing, use these packages for additional operations:
//   - [github.com/erraggy/oastools/validator] - Validate specifications against OAS rules
//   - [github.com/erraggy/oastools/fixer] - Fix common validation errors automatically
//   - [github.com/erraggy/oastools/converter] - Convert between OAS versions (2.0 ↔ 3.x)
//   - [github.com/erraggy/oastools/joiner] - Join multiple specifications into one
//   - [github.com/erraggy/oastools/differ] - Compare specifications and detect breaking changes
//   - [github.com/erraggy/oastools/generator] - Generate Go code from specifications
//   - [github.com/erraggy/oastools/builder] - Programmatically build specifications
//   - [github.com/erraggy/oastools/overlay] - Apply overlay transformations
package parser
