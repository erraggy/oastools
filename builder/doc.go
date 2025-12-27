// Package builder provides programmatic construction of OpenAPI Specification documents.
//
// The builder package enables users to construct OAS documents in Go using a fluent API
// with automatic reflection-based schema generation. Go types are passed directly to the
// API, and the builder automatically generates OpenAPI-compatible JSON schemas.
//
// # Supported Versions
//
// The builder supports both OAS 2.0 (Swagger) and OAS 3.x (3.0.0 through 3.2.0).
// Use New() with the desired OAS version and the corresponding Build method:
//
//   - New(parser.OASVersion20) + BuildOAS2() → *parser.OAS2Document with schemas in "definitions"
//   - New(parser.OASVersion3xx) + BuildOAS3() → *parser.OAS3Document with schemas in "components/schemas"
//
// The $ref paths are automatically adjusted based on the OAS version:
//   - OAS 2.0: "#/definitions/", "#/parameters/", "#/responses/"
//   - OAS 3.x: "#/components/schemas/", "#/components/parameters/", "#/components/responses/"
//
// # Validation
//
// The builder does not perform OAS specification validation. Use the validator package
// to validate built documents:
//
//	result, _ := spec.BuildResult()
//	valResult, _ := validator.ValidateWithOptions(validator.WithParsed(*result))
//
// # Quick Start
//
// Build an OAS 3.x API specification:
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
//	doc, err := spec.BuildOAS3()
//	if err != nil {
//		log.Fatal(err)
//	}
//	// doc is *parser.OAS3Document - no type assertion needed
//
// Build an OAS 2.0 (Swagger) API specification:
//
//	spec := builder.New(parser.OASVersion20).
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
// # Modifying Existing Documents
//
// Use FromDocument or FromOAS2Document to create a builder from an existing document:
//
//	// For OAS 3.x documents
//	b := builder.FromDocument(existingOAS3Doc)
//	b.AddOperation(http.MethodPost, "/users", ...)
//	newDoc, _ := b.BuildOAS3()
//
//	// For OAS 2.0 documents
//	b := builder.FromOAS2Document(existingSwaggerDoc)
//	b.AddOperation(http.MethodPost, "/users", ...)
//	newDoc, _ := b.BuildOAS2()
//
// # Webhooks (OAS 3.1+)
//
// For OAS 3.1+ specifications, webhooks can be added using AddWebhook:
//
//	spec := builder.New(parser.OASVersion310).
//		SetTitle("Webhook API").
//		SetVersion("1.0.0").
//		AddWebhook("userCreated", http.MethodPost,
//			builder.WithRequestBody("application/json", UserEvent{}),
//			builder.WithResponse(http.StatusOK, struct{}{}),
//		)
//
// # External Documentation
//
// Add document-level external documentation using SetExternalDocs:
//
//	spec := builder.New(parser.OASVersion320).
//		SetTitle("My API").
//		SetVersion("1.0.0").
//		SetExternalDocs(&parser.ExternalDocs{
//			URL:         "https://docs.example.com",
//			Description: "API documentation",
//		})
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
// # Extensible Schema Naming
//
// The default schema naming uses "package.TypeName" format. For custom naming,
// use options when creating a Builder:
//
// Built-in strategies:
//
//	// PascalCase: ModelsUser
//	spec := builder.New(parser.OASVersion320,
//	    builder.WithSchemaNaming(builder.SchemaNamingPascalCase),
//	)
//
//	// Type only (no package prefix): User
//	spec := builder.New(parser.OASVersion320,
//	    builder.WithSchemaNaming(builder.SchemaNamingTypeOnly),
//	)
//
// Available strategies:
//   - SchemaNamingDefault: "package.TypeName" (e.g., models.User)
//   - SchemaNamingPascalCase: "PackageTypeName" (e.g., ModelsUser)
//   - SchemaNamingCamelCase: "packageTypeName" (e.g., modelsUser)
//   - SchemaNamingSnakeCase: "package_type_name" (e.g., models_user)
//   - SchemaNamingKebabCase: "package-type-name" (e.g., models-user)
//   - SchemaNamingTypeOnly: "TypeName" (e.g., User) - may cause conflicts
//   - SchemaNamingFullPath: "full_path_TypeName" (e.g., github.com_org_models_User)
//
// Custom templates using Go text/template:
//
//	// Custom format: models+User
//	spec := builder.New(parser.OASVersion320,
//	    builder.WithSchemaNameTemplate(`{{.Package}}+{{.Type}}`),
//	)
//
// Available template functions: pascal, camel, snake, kebab, upper, lower,
// title, sanitize, trimPrefix, trimSuffix, replace, join.
//
// Custom naming function for maximum flexibility:
//
//	spec := builder.New(parser.OASVersion320,
//	    builder.WithSchemaNameFunc(func(ctx builder.SchemaNameContext) string {
//	        return strings.ToUpper(ctx.Type)
//	    }),
//	)
//
// Note: RegisterTypeAs always takes precedence over any naming strategy.
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
// Generic type naming strategies control how generic types are formatted:
//
//	// "Of" separator: Response[User] → ResponseOfUser
//	spec := builder.New(parser.OASVersion320,
//	    builder.WithGenericNaming(builder.GenericNamingOf),
//	)
//
// Available generic strategies:
//   - GenericNamingUnderscore: "Response_User_" (default)
//   - GenericNamingOf: "ResponseOfUser"
//   - GenericNamingFor: "ResponseForUser"
//   - GenericNamingAngleBrackets: "Response<User>" (URI-encoded in $ref)
//   - GenericNamingFlattened: "ResponseUser"
//
// For fine-grained control over generic naming:
//
//	spec := builder.New(parser.OASVersion320,
//	    builder.WithGenericNamingConfig(builder.GenericNamingConfig{
//	        Strategy:        builder.GenericNamingOf,
//	        ParamSeparator:  "And",
//	        ApplyBaseCasing: true,
//	    }),
//	)
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
// # Operation Responses
//
// Note: OpenAPI requires at least one response per operation. If no responses
// are defined, the resulting spec will fail OAS validation. Always use
// WithResponse() or WithDefaultResponse() to add responses to operations.
//
// # Parameter Constraints
//
// Add validation constraints to parameters using WithParam* options:
//
//	spec.AddOperation(http.MethodGet, "/pets",
//		builder.WithQueryParam("limit", int32(0),
//			builder.WithParamDescription("Maximum number of pets to return"),
//			builder.WithParamMinimum(1),
//			builder.WithParamMaximum(100),
//			builder.WithParamDefault(20),
//		),
//		builder.WithQueryParam("status", string(""),
//			builder.WithParamEnum("available", "pending", "sold"),
//		),
//		builder.WithQueryParam("name", string(""),
//			builder.WithParamMinLength(1),
//			builder.WithParamMaxLength(50),
//			builder.WithParamPattern("^[a-zA-Z]+$"),
//		),
//	)
//
// Supported parameter constraint options:
//   - WithParamMinimum(min float64) - Minimum value for numeric parameters
//   - WithParamMaximum(max float64) - Maximum value for numeric parameters
//   - WithParamExclusiveMinimum(exclusive bool) - Whether minimum is exclusive
//   - WithParamExclusiveMaximum(exclusive bool) - Whether maximum is exclusive
//   - WithParamMultipleOf(value float64) - Value must be a multiple of this (must be > 0)
//   - WithParamMinLength(min int) - Minimum length for string parameters (must be >= 0)
//   - WithParamMaxLength(max int) - Maximum length for string parameters (must be >= 0)
//   - WithParamPattern(pattern string) - Regex pattern for string parameters (validated at build time)
//   - WithParamMinItems(min int) - Minimum items for array parameters (must be >= 0)
//   - WithParamMaxItems(max int) - Maximum items for array parameters (must be >= 0)
//   - WithParamUniqueItems(unique bool) - Whether array items must be unique
//   - WithParamEnum(values ...any) - Allowed enumeration values
//   - WithParamDefault(value any) - Default value for the parameter
//
// # Explicit Type and Format Overrides
//
// Override the inferred OpenAPI type or format when the Go type doesn't map directly
// to your desired schema:
//
//	spec.AddOperation(http.MethodGet, "/users/{user_id}",
//		builder.WithPathParam("user_id", "",
//			builder.WithParamFormat("uuid"),
//		),
//		builder.WithQueryParam("version", 0,
//			builder.WithParamType("integer"),
//			builder.WithParamFormat("int64"),
//		),
//	)
//
// Available type/format override options:
//   - WithParamType(typeName string) - Override inferred type (string, integer, number, boolean, array, object)
//   - WithParamFormat(format string) - Override inferred format (uuid, email, date-time, byte, binary, etc.)
//   - WithParamSchema(schema *parser.Schema) - Full schema override (highest precedence)
//
// Precedence rules:
//   - WithParamSchema takes highest precedence (complete replacement)
//   - WithParamType replaces the inferred type
//   - WithParamFormat replaces the inferred format
//
// Constraint validation is performed when building the document. The following rules are enforced:
//   - minimum must be <= maximum (if both are set)
//   - minLength must be <= maxLength (if both are set)
//   - minItems must be <= maxItems (if both are set)
//   - minLength, maxLength, minItems, maxItems must be non-negative
//   - multipleOf must be greater than 0
//   - pattern must be a valid regex
//
// # Form Parameters
//
// Form parameters are handled differently across OAS versions. The builder provides a unified
// WithFormParam method that automatically adapts to the target OAS version:
//
//	spec.AddOperation(http.MethodPost, "/login",
//		builder.WithFormParam("username", string(""),
//			builder.WithParamRequired(true),
//			builder.WithParamMinLength(3),
//		),
//		builder.WithFormParam("password", string(""),
//			builder.WithParamRequired(true),
//			builder.WithParamMinLength(8),
//		),
//		builder.WithResponse(http.StatusOK, LoginResponse{}),
//	)
//
// OAS 2.0: Form parameters are created as parameters with in="formData":
//
//	parameters:
//	  - name: username
//	    in: formData
//	    type: string
//	    required: true
//	    minLength: 3
//
// OAS 3.x: Form parameters are added to the request body with application/x-www-form-urlencoded:
//
//	requestBody:
//	  content:
//	    application/x-www-form-urlencoded:
//	      schema:
//	        type: object
//	        properties:
//	          username:
//	            type: string
//	            minLength: 3
//	        required:
//	          - username
//
// All parameter constraints and options work with form parameters. If a request body already
// exists (e.g., for multipart/form-data), form parameters are merged into it with the
// application/x-www-form-urlencoded content type.
//
// # OAS Version Differences for Constraints
//
// The builder handles constraint placement automatically based on the OAS version:
//
// OAS 3.x (3.0.0+): Constraints are applied to the parameter's Schema field.
// The OAS 3.x specification separates the parameter metadata (name, location, required)
// from the value schema (type, format, constraints). This is the modern approach:
//
//	parameters:
//	  - name: limit
//	    in: query
//	    schema:
//	      type: integer
//	      minimum: 1        # Constraint on schema
//	      maximum: 100      # Constraint on schema
//
// OAS 2.0 (Swagger): Constraints are applied directly to the Parameter object.
// In OAS 2.0, non-body parameters have type and constraints as top-level fields:
//
//	parameters:
//	  - name: limit
//	    in: query
//	    type: integer
//	    minimum: 1          # Constraint on parameter
//	    maximum: 100        # Constraint on parameter
//
// The builder abstracts this difference, allowing you to use the same WithParam*
// options regardless of target OAS version.
//
// # Custom Content Types
//
// Use WithResponseContentType to specify content types other than the default "application/json":
//
//	spec.AddOperation(http.MethodGet, "/users",
//		builder.WithResponse(http.StatusOK, []User{},
//			builder.WithResponseContentType("application/xml"),
//		),
//	)
//
// For multiple content types, use WithRequestBodyContentTypes or WithResponseContentTypes:
//
//	spec.AddOperation(http.MethodPost, "/users",
//		builder.WithRequestBodyContentTypes(
//			[]string{"application/json", "application/xml"},
//			User{},
//		),
//		builder.WithResponseContentTypes(http.StatusOK,
//			[]string{"application/json", "application/xml"},
//			User{},
//		),
//	)
//
// # Vendor Extensions
//
// Add vendor extensions (x-* fields) to operations, parameters, responses, and request bodies:
//
//	spec.AddOperation(http.MethodGet, "/users",
//		builder.WithOperationExtension("x-rate-limit", 100),
//		builder.WithQueryParam("limit", int32(0),
//			builder.WithParamExtension("x-ui-widget", "slider"),
//		),
//		builder.WithResponse(http.StatusOK, []User{},
//			builder.WithResponseExtension("x-cache-ttl", 3600),
//		),
//	)
//
//	spec.AddOperation(http.MethodPost, "/users",
//		builder.WithRequestBody("application/json", User{},
//			builder.WithRequestBodyExtension("x-codegen-request-body-name", "user"),
//		),
//	)
//
// # OAS 2.0 Specific Options
//
// For OAS 2.0 specifications, additional options are available:
//
//	spec.AddOperation(http.MethodPost, "/users",
//		builder.WithConsumes("application/json", "application/xml"),
//		builder.WithProduces("application/json"),
//		builder.WithQueryParam("tags", []string{},
//			builder.WithParamCollectionFormat("csv"),  // csv, ssv, tsv, pipes, multi
//			builder.WithParamAllowEmptyValue(true),
//		),
//	)
//
// See the examples in example_test.go for more patterns.
//
// # Semantic Schema Deduplication
//
// The builder can automatically identify and consolidate structurally identical schemas,
// reducing document size when multiple types converge to the same structure.
//
// Enable via option:
//
//	spec := builder.New(parser.OASVersion320,
//	    builder.WithSemanticDeduplication(true),
//	)
//
//	// Add schemas that happen to be structurally identical
//	spec.AddOperation(http.MethodGet, "/addresses",
//	    builder.WithResponse(http.StatusOK, []Address{}),
//	)
//	spec.AddOperation(http.MethodGet, "/locations",
//	    builder.WithResponse(http.StatusOK, []Location{}), // Same structure as Address
//	)
//
//	doc, _ := spec.BuildOAS3()
//	// Only one schema exists; all $refs point to the canonical (alphabetically first) name
//
// Or call manually before building:
//
//	spec.DeduplicateSchemas()
//	doc, _ := spec.BuildOAS3()
//
// Deduplication compares schemas structurally, ignoring metadata fields (title, description,
// example, deprecated). When duplicates are found, the alphabetically first name becomes
// canonical, and all references are automatically rewritten.
//
// # Server Builder
//
// The ServerBuilder extends Builder to produce runnable HTTP servers directly from the fluent API.
// This enables a "code-first" development workflow where developers define API operations and their
// implementations in a single fluent API.
//
// Create a server builder:
//
//	srv := builder.NewServerBuilder(parser.OASVersion320).
//		SetTitle("Pet Store API").
//		SetVersion("1.0.0")
//
//	srv.AddOperation(http.MethodGet, "/pets",
//		builder.WithOperationID("listPets"),
//		builder.WithResponse(http.StatusOK, []Pet{}),
//	).Handle("listPets", func(ctx context.Context, req *builder.Request) builder.Response {
//		return builder.JSON(http.StatusOK, pets)
//	})
//
//	result, err := srv.BuildServer()
//	if err != nil {
//		log.Fatal(err)
//	}
//	http.ListenAndServe(":8080", result.Handler)
//
// The ServerBuilder provides:
//   - Automatic request validation using the httpvalidator package
//   - Type-safe response construction with response helpers (JSON, NoContent, Error, Redirect, Stream)
//   - Middleware support via the Use method
//   - Flexible routing using the stdlib net/http router
//   - Recovery middleware for panic handling
//   - Request logging middleware
//
// # Handler Registration
//
// Register handlers by operation ID using Handle or HandleFunc:
//
//	// Using the typed handler signature
//	srv.Handle("listPets", func(ctx context.Context, req *builder.Request) builder.Response {
//		// req.PathParams, req.QueryParams contain validated parameters
//		// req.Body contains the unmarshaled request body
//		return builder.JSON(http.StatusOK, pets)
//	})
//
//	// Using standard http.HandlerFunc for simpler operations
//	srv.HandleFunc("healthCheck", func(w http.ResponseWriter, r *http.Request) {
//		w.WriteHeader(http.StatusOK)
//		w.Write([]byte("OK"))
//	})
//
// Operations without registered handlers return 501 Not Implemented at runtime.
//
// # Response Helpers
//
// The builder provides convenient response constructors:
//
//	// JSON response
//	return builder.JSON(http.StatusOK, data)
//
//	// No content response
//	return builder.NoContent()
//
//	// Error response
//	return builder.Error(http.StatusNotFound, "not found")
//
//	// Redirect response
//	return builder.Redirect(http.StatusMovedPermanently, "/new-location")
//
//	// Streaming response
//	return builder.Stream(http.StatusOK, "application/octet-stream", reader)
//
//	// Fluent response builder
//	return builder.NewResponse(http.StatusOK).
//		Header("X-Custom", "value").
//		JSON(data)
//
// # Server Configuration
//
// Configure the server builder with options:
//
//	srv := builder.NewServerBuilder(parser.OASVersion320,
//		builder.WithoutValidation(),                    // Disable request validation
//		builder.WithRecovery(),                         // Enable panic recovery
//		builder.WithRequestLogging(loggingFunc),        // Enable request logging
//		builder.WithErrorHandler(customErrorHandler),   // Custom error handler
//		builder.WithNotFoundHandler(custom404Handler),  // Custom 404 handler
//	)
//
// # Middleware
//
// Add middleware to the server. Middleware is applied in order: first added = outermost.
//
//	srv.Use(corsMiddleware, authMiddleware, loggingMiddleware)
//
// # Testing
//
// The package provides testing utilities:
//
//	result := srv.MustBuildServer()
//	test := builder.NewServerTest(result)
//
//	// Execute requests
//	rec := test.Execute(builder.NewTestRequest(http.MethodGet, "/pets"))
//
//	// JSON helpers
//	var pets []Pet
//	rec, err := test.GetJSON("/pets", &pets)
//	rec, err := test.PostJSON("/pets", newPet, &created)
//
//	// Stub handlers for testing
//	srv.Handle("listPets", builder.StubHandler(builder.JSON(http.StatusOK, mockPets)))
//
// # Related Packages
//
// The builder integrates with other oastools packages:
//   - [github.com/erraggy/oastools/parser] - Builder generates parser-compatible documents
//   - [github.com/erraggy/oastools/validator] - Validate built specifications
//   - [github.com/erraggy/oastools/fixer] - Fix common validation errors in built specifications
//   - [github.com/erraggy/oastools/converter] - Convert built specs between OAS versions
//   - [github.com/erraggy/oastools/joiner] - Join built specs with existing documents
//   - [github.com/erraggy/oastools/differ] - Compare built specs with other specifications
//   - [github.com/erraggy/oastools/generator] - Generate code from built specifications
//   - [github.com/erraggy/oastools/httpvalidator] - Runtime request/response validation
package builder
