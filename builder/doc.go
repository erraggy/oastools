// Package builder provides programmatic construction of OpenAPI Specification documents.
//
// The builder package enables users to construct OAS documents in Go using a fluent API
// with automatic reflection-based schema generation. Go types are passed directly to the
// API, and the builder automatically generates OpenAPI-compatible JSON schemas.
//
// # Supported Versions
//
// The builder supports both OAS 2.0 (Swagger) and OAS 3.x (3.0.0 through 3.2.0).
// Use the appropriate factory function for type-safe document building:
//
//   - NewOAS2() + BuildOAS2() → *parser.OAS2Document with schemas in "definitions"
//   - NewOAS3(version) + BuildOAS3() → *parser.OAS3Document with schemas in "components/schemas"
//
// The $ref paths are automatically adjusted based on the builder type:
//   - OAS 2.0: "#/definitions/", "#/parameters/", "#/responses/"
//   - OAS 3.x: "#/components/schemas/", "#/components/parameters/", "#/components/responses/"
//
// # Quick Start
//
// Build an OAS 3.x API specification:
//
//	spec := builder.NewOAS3(parser.OASVersion320).
//		SetTitle("My API").
//		SetVersion("1.0.0")
//
//	spec.AddOperation(http.MethodGet, "/users",
//		builder.WithOperationID("listUsers"),
//		builder.WithResponse(http.StatusOK, []User{}),
//	)
//
//	doc, err := spec.BuildOAS3()
//	if err != nil {
//		log.Fatal(err)
//	}
//	// doc is *parser.OAS3Document - no type assertion needed
//
// Build an OAS 2.0 (Swagger) API specification:
//
//	spec := builder.NewOAS2().
//		SetTitle("My API").
//		SetVersion("1.0.0")
//
//	spec.AddOperation(http.MethodGet, "/users",
//		builder.WithOperationID("listUsers"),
//		builder.WithResponse(http.StatusOK, []User{}),
//	)
//
//	doc, err := spec.BuildOAS2()
//	if err != nil {
//		log.Fatal(err)
//	}
//	// doc is *parser.OAS2Document - no type assertion needed
//
// # Reflection-Based Schema Generation
//
// The core feature is automatic schema generation from Go types via reflection.
// When you pass a Go type to WithResponse, WithRequestBody, or parameter options,
// the builder inspects the type structure and generates an OpenAPI-compatible schema.
//
// Type mappings:
//   - string → string
//   - int, int32 → integer (format: int32)
//   - int64 → integer (format: int64)
//   - float32 → number (format: float)
//   - float64 → number (format: double)
//   - bool → boolean
//   - []T → array (items from T)
//   - map[string]T → object (additionalProperties from T)
//   - struct → object (properties from fields)
//   - *T → schema of T (nullable)
//   - time.Time → string (format: date-time)
//
// # Schema Naming
//
// Schemas are named using the Go convention of "package.TypeName" (e.g., "models.User").
// This ensures uniqueness and matches Go developers' expectations for type identification.
// If multiple packages have the same base name (e.g., github.com/foo/models and
// github.com/bar/models), the full package path is used to disambiguate
// (e.g., "github.com_foo_models.User").
// Anonymous types are named "AnonymousType".
//
// # Generic Types
//
// Go 1.18+ generic types are fully supported. The type parameters are included in the
// schema name but sanitized for URI safety by replacing brackets with underscores:
//
//	Response[User] → "builder.Response_User"
//	Map[string,int] → "builder.Map_string_int"
//	Response[List[User]] → "builder.Response_List_User"
//
// This ensures $ref URIs are valid and compatible with all OpenAPI tools, which may
// not handle square brackets properly in schema references.
//
// # Struct Tags
//
// Customize schema generation with struct tags:
//
//	type User struct {
//		ID    int64  `json:"id" oas:"description=Unique identifier"`
//		Name  string `json:"name" oas:"minLength=1,maxLength=100"`
//		Email string `json:"email" oas:"format=email"`
//		Role  string `json:"role" oas:"enum=admin|user|guest"`
//	}
//
// Supported oas tag options:
//   - description=<text> - Field description
//   - format=<format> - Override format (email, uri, uuid, date, date-time, etc.)
//   - enum=<val1>|<val2>|... - Enumeration values
//   - minimum=<n>, maximum=<n> - Numeric constraints
//   - minLength=<n>, maxLength=<n> - String length constraints
//   - pattern=<regex> - String pattern
//   - minItems=<n>, maxItems=<n> - Array constraints
//   - readOnly=true, writeOnly=true - Access modifiers
//   - nullable=true - Explicitly nullable
//   - deprecated=true - Mark as deprecated
//
// # Required Fields
//
// Required fields are determined by:
//   - Non-pointer fields without omitempty are required
//   - Fields with oas:"required=true" are explicitly required
//   - Fields with oas:"required=false" are explicitly optional
//
// # Integration with Other Packages
//
// The builder integrates with the validator package:
//
//	spec := builder.New(parser.OASVersion320).
//		SetTitle("My API").
//		SetVersion("1.0.0")
//	// ... add operations ...
//
//	result, _ := spec.BuildResult()
//	valResult, _ := validator.ValidateWithOptions(
//		validator.WithParsed(*result),
//	)
//
// See the examples in example_test.go for more patterns.
package builder
