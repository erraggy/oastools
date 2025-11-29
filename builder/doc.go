// Package builder provides programmatic construction of OpenAPI Specification documents.
//
// The builder package enables users to construct OAS documents in Go using a fluent API
// with automatic reflection-based schema generation. Go types are passed directly to the
// API, and the builder automatically generates OpenAPI-compatible JSON schemas in the
// components.schemas section.
//
// # Quick Start
//
// Build a simple API specification:
//
//	spec := builder.New(parser.OASVersion320).
//		SetTitle("My API").
//		SetVersion("1.0.0")
//
//	spec.AddOperation(http.MethodGet, "/users",
//		builder.WithOperationID("listUsers"),
//		builder.WithResponse(http.StatusOK, []User{}),
//	)
//
//	doc, err := spec.Build()
//	if err != nil {
//		log.Fatal(err)
//	}
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
// Schemas are named using the Go convention of "package.TypeName" (e.g., "myapp.User").
// This ensures uniqueness and matches Go developers' expectations for type identification.
// Anonymous types are named "AnonymousType".
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
