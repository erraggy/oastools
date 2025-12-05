// Package generator provides Go code generation from OpenAPI Specification documents.
//
// The generator creates idiomatic Go code for API clients and server stubs from
// OAS 2.0 and OAS 3.x specifications. Generated code emphasizes type safety,
// proper error handling, and clean interfaces.
//
// # Quick Start
//
// Generate a client using functional options:
//
//	result, err := generator.GenerateWithOptions(
//		generator.WithFilePath("openapi.yaml"),
//		generator.WithPackageName("petstore"),
//		generator.WithClient(true),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	if err := result.WriteFiles("./generated"); err != nil {
//		log.Fatal(err)
//	}
//
// Or use a reusable Generator instance:
//
//	g := generator.New()
//	g.PackageName = "petstore"
//	g.GenerateClient = true
//	g.GenerateServer = true
//	result, _ := g.Generate("openapi.yaml")
//	result.WriteFiles("./generated")
//
// # Generation Modes
//
// The generator supports three modes:
//   - Client: HTTP client with methods for each operation
//   - Server: Interface definitions and request/response types
//   - Types: Schema-only generation (models)
//
// # Type Mapping
//
// OpenAPI types are mapped to Go types as follows:
//   - string → string (with format handling: date-time→time.Time, uuid→string, etc.)
//   - integer → int64 (int32 for format: int32)
//   - number → float64 (float32 for format: float)
//   - boolean → bool
//   - array → []T
//   - object → struct or map[string]T
//
// Optional fields use pointers, and nullable fields in OAS 3.1+ are handled
// with pointer types or generic Option[T] types (configurable).
//
// # Generated Files
//
// The generator produces the following files:
//   - types.go: Model structs from components/schemas
//   - client.go: HTTP client (when GenerateClient is true)
//   - server.go: Server interface (when GenerateServer is true)
//   - helpers.go: Shared utilities
//
// See the exported GenerateResult and GenerateIssue types for complete details.
package generator
