// Package parser provides OpenAPI Specification (OAS) parsing functionality.
//
// This package supports parsing and validating OpenAPI specifications across
// multiple versions, from OAS 2.0 (Swagger) through OAS 3.2.0. It handles
// YAML and JSON formats, resolves external references, and performs structural
// validation.
//
// # Supported Versions
//
// The parser supports all official OpenAPI Specification releases:
//   - OAS 2.0 (Swagger): https://spec.openapis.org/oas/v2.0.html
//   - OAS 3.0.x (3.0.0 through 3.0.4): https://spec.openapis.org/oas/v3.0.0.html
//   - OAS 3.1.x (3.1.0 through 3.1.2): https://spec.openapis.org/oas/v3.1.0.html
//   - OAS 3.2.0: https://spec.openapis.org/oas/v3.2.0.html
//
// All schema definitions use JSON Schema Specification Draft 2020-12:
// https://www.ietf.org/archive/id/draft-bhutton-json-schema-01.html
//
// Release candidate versions (e.g., 3.0.0-rc0) are detected but not officially supported.
//
// # Features
//
//   - Multi-format parsing (YAML, JSON)
//   - External reference resolution ($ref)
//   - Path traversal protection for file references
//   - Operation ID uniqueness validation
//   - Status code format validation
//   - Memory-efficient caching with limits
//
// # Security Considerations
//
// The parser implements several security protections:
//
//   - Path traversal prevention: External file references are restricted to the
//     base directory and its subdirectories using filepath.Rel validation
//
//   - Cache limits: A maximum of MaxCachedDocuments (default: 1000) external
//     documents can be loaded to prevent memory exhaustion
//
//   - HTTP(S) references: Remote URL references are not currently supported,
//     limiting attack surface to local filesystem only
//
//   - Input validation: All numeric status codes, operation IDs, and reference
//     formats are validated before processing
//
// # Basic Usage
//
//	opts := &parser.Options{
//		StrictValidation: true,
//		ResolveRefs:      true,
//	}
//
//	result, err := parser.ParseFile("openapi.yaml", opts)
//	if err != nil {
//		log.Fatalf("Parse failed: %v", err)
//	}
//
//	if !result.Valid {
//		for _, issue := range result.Issues {
//			fmt.Printf("%s: %s\n", issue.Severity, issue.Message)
//		}
//	}
//
// # Performance Notes
//
// When ResolveRefs is enabled, the parser performs additional marshaling/unmarshaling
// to resolve external references, which may impact performance on large documents.
// For read-only validation without reference resolution, set ResolveRefs to false.
//
// The parser maintains internal maps (Extra fields) on all structs to preserve
// unknown fields during parsing, allowing for extension properties and forward
// compatibility. This trades some memory for flexibility.
package parser
