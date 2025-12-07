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
// up to 1000 documents to prevent memory exhaustion.
//
// HTTP/HTTPS $ref resolution is available via WithResolveHTTPRefs (opt-in for
// security). Use WithInsecureSkipVerify for self-signed certificates. HTTP
// responses are cached, size-limited, and protected against circular references.
// See the examples in example_test.go for more details.
//
// # ParseResult Fields
//
// ParseResult includes the detected Version, OASVersion, SourceFormat (JSON or YAML),
// and any parsing Errors or Warnings. The Document field contains the parsed OAS2Document
// or OAS3Document. The SourceFormat field can be used by conversion and joining tools to
// preserve the original file format. See the exported ParseResult and document type fields
// for complete details.
//
// # Related Packages
//
// After parsing, use these packages for additional operations:
//   - [github.com/erraggy/oastools/validator] - Validate specifications against OAS rules
//   - [github.com/erraggy/oastools/converter] - Convert between OAS versions (2.0 ↔ 3.x)
//   - [github.com/erraggy/oastools/joiner] - Join multiple specifications into one
//   - [github.com/erraggy/oastools/differ] - Compare specifications and detect breaking changes
//   - [github.com/erraggy/oastools/generator] - Generate Go code from specifications
//   - [github.com/erraggy/oastools/builder] - Programmatically build specifications
package parser
