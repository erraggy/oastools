package builder_test

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/erraggy/oastools/builder"
	"github.com/erraggy/oastools/parser"
)

// Pet represents a pet in the store.
type Pet struct {
	ID        int64     `json:"id" oas:"description=Unique pet identifier"`
	Name      string    `json:"name" oas:"minLength=1,description=Pet name"`
	Tag       string    `json:"tag,omitempty" oas:"description=Optional tag"`
	CreatedAt time.Time `json:"created_at" oas:"readOnly=true"`
}

// Error represents an API error.
type Error struct {
	Code    int32  `json:"code" oas:"description=Error code"`
	Message string `json:"message" oas:"description=Error message"`
}

// Example demonstrates basic builder usage.
func Example() {
	spec := builder.New(parser.OASVersion320).
		SetTitle("Pet Store API").
		SetVersion("1.0.0")

	spec.AddOperation(http.MethodGet, "/pets",
		builder.WithOperationID("listPets"),
		builder.WithResponse(http.StatusOK, []Pet{}),
	)

	// Use BuildOAS3() for type-safe access - no type assertion needed
	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("OpenAPI: %s\n", doc.OpenAPI)
	fmt.Printf("Title: %s\n", doc.Info.Title)
	fmt.Printf("Paths: %d\n", len(doc.Paths))
	// Output:
	// OpenAPI: 3.2.0
	// Title: Pet Store API
	// Paths: 1
}

// Example_withServer demonstrates adding servers.
func Example_withServer() {
	spec := builder.New(parser.OASVersion320).
		SetTitle("My API").
		SetVersion("1.0.0").
		AddServer("https://api.example.com/v1",
			builder.WithServerDescription("Production server"),
		).
		AddServer("https://staging.example.com/v1",
			builder.WithServerDescription("Staging server"),
		)

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Servers: %d\n", len(doc.Servers))
	fmt.Printf("First server: %s\n", doc.Servers[0].URL)
	// Output:
	// Servers: 2
	// First server: https://api.example.com/v1
}

// Example_withRequestBody demonstrates adding request bodies.
func Example_withRequestBody() {
	type CreatePetRequest struct {
		Name string `json:"name" oas:"minLength=1"`
		Tag  string `json:"tag,omitempty"`
	}

	spec := builder.New(parser.OASVersion320).
		SetTitle("Pet Store API").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/pets",
			builder.WithOperationID("createPet"),
			builder.WithRequestBody("application/json", CreatePetRequest{},
				builder.WithRequired(true),
				builder.WithRequestDescription("Pet to create"),
			),
			builder.WithResponse(http.StatusCreated, Pet{}),
		)

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	hasRequestBody := doc.Paths["/pets"].Post.RequestBody != nil
	fmt.Printf("Has request body: %v\n", hasRequestBody)
	// Output:
	// Has request body: true
}

// Example_withParameters demonstrates adding parameters.
func Example_withParameters() {
	spec := builder.New(parser.OASVersion320).
		SetTitle("Pet Store API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/pets/{petId}",
			builder.WithOperationID("getPet"),
			builder.WithPathParam("petId", int64(0),
				builder.WithParamDescription("The ID of the pet"),
			),
			builder.WithQueryParam("include", "",
				builder.WithParamDescription("Include related resources"),
			),
			builder.WithResponse(http.StatusOK, Pet{}),
		)

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	paramCount := len(doc.Paths["/pets/{petId}"].Get.Parameters)
	fmt.Printf("Parameters: %d\n", paramCount)
	// Output:
	// Parameters: 2
}

// Example_withSecurity demonstrates security configuration.
func Example_withSecurity() {
	spec := builder.New(parser.OASVersion320).
		SetTitle("Secure API").
		SetVersion("1.0.0").
		AddAPIKeySecurityScheme("api_key", "header", "X-API-Key", "API key authentication").
		AddHTTPSecurityScheme("bearer_auth", "bearer", "JWT", "Bearer token authentication").
		SetSecurity(
			builder.SecurityRequirement("api_key"),
			builder.SecurityRequirement("bearer_auth"),
		).
		AddOperation(http.MethodGet, "/secure",
			builder.WithOperationID("secureEndpoint"),
			builder.WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	schemeCount := len(doc.Components.SecuritySchemes)
	securityCount := len(doc.Security)
	fmt.Printf("Security schemes: %d\n", schemeCount)
	fmt.Printf("Global security requirements: %d\n", securityCount)
	// Output:
	// Security schemes: 2
	// Global security requirements: 2
}

// Example_completeAPI demonstrates a complete API specification.
func Example_completeAPI() {
	spec := builder.New(parser.OASVersion320).
		SetTitle("Pet Store API").
		SetVersion("1.0.0").
		SetDescription("A sample Pet Store API demonstrating the builder package").
		AddServer("https://api.petstore.example.com/v1",
			builder.WithServerDescription("Production server"),
		).
		AddTag("pets", builder.WithTagDescription("Operations about pets")).
		AddAPIKeySecurityScheme("api_key", "header", "X-API-Key", "API key").
		SetSecurity(builder.SecurityRequirement("api_key"))

	// List pets
	spec.AddOperation(http.MethodGet, "/pets",
		builder.WithOperationID("listPets"),
		builder.WithSummary("List all pets"),
		builder.WithTags("pets"),
		builder.WithQueryParam("limit", int32(0),
			builder.WithParamDescription("Maximum number of pets to return"),
		),
		builder.WithResponse(http.StatusOK, []Pet{},
			builder.WithResponseDescription("A list of pets"),
		),
		builder.WithResponse(http.StatusInternalServerError, Error{},
			builder.WithResponseDescription("Unexpected error"),
		),
	)

	// Get pet by ID
	spec.AddOperation(http.MethodGet, "/pets/{petId}",
		builder.WithOperationID("getPet"),
		builder.WithSummary("Get a pet by ID"),
		builder.WithTags("pets"),
		builder.WithPathParam("petId", int64(0),
			builder.WithParamDescription("The ID of the pet to retrieve"),
		),
		builder.WithResponse(http.StatusOK, Pet{},
			builder.WithResponseDescription("The requested pet"),
		),
		builder.WithResponse(http.StatusNotFound, Error{},
			builder.WithResponseDescription("Pet not found"),
		),
	)

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Title: %s\n", doc.Info.Title)
	fmt.Printf("Paths: %d\n", len(doc.Paths))
	fmt.Printf("Tags: %d\n", len(doc.Tags))
	fmt.Printf("Schemas: %d\n", len(doc.Components.Schemas))
	// Output:
	// Title: Pet Store API
	// Paths: 2
	// Tags: 1
	// Schemas: 2
}

// Example_schemaGeneration demonstrates automatic schema generation.
func Example_schemaGeneration() {
	type Address struct {
		Street  string `json:"street"`
		City    string `json:"city"`
		Country string `json:"country"`
	}

	type Customer struct {
		ID      int64   `json:"id"`
		Name    string  `json:"name"`
		Email   string  `json:"email" oas:"format=email"`
		Address Address `json:"address"`
	}

	spec := builder.New(parser.OASVersion320).
		SetTitle("Customer API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/customers/{id}",
			builder.WithOperationID("getCustomer"),
			builder.WithPathParam("id", int64(0)),
			builder.WithResponse(http.StatusOK, Customer{}),
		)

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	// Both Customer and Address schemas are auto-generated with package-qualified names
	_, hasCustomer := doc.Components.Schemas["builder_test.Customer"]
	_, hasAddress := doc.Components.Schemas["builder_test.Address"]
	fmt.Printf("Has Customer schema: %v\n", hasCustomer)
	fmt.Printf("Has Address schema: %v\n", hasAddress)
	// Output:
	// Has Customer schema: true
	// Has Address schema: true
}

// Example_fromDocument demonstrates modifying an existing document.
func Example_fromDocument() {
	// Create an existing document (in real code, this would be parsed from a file)
	existingDoc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &parser.Info{
			Title:   "Existing API",
			Version: "1.0.0",
		},
		Paths: parser.Paths{
			"/existing": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "existingOperation",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "Existing response",
							},
						},
					},
				},
			},
		},
	}

	// Create builder from existing document and add new operations
	spec := builder.FromDocument(existingDoc)

	type HealthResponse struct {
		Status string `json:"status"`
	}

	spec.AddOperation(http.MethodGet, "/health",
		builder.WithOperationID("healthCheck"),
		builder.WithResponse(http.StatusOK, HealthResponse{}),
	)

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Paths: %d\n", len(doc.Paths))
	fmt.Printf("Has /existing: %v\n", doc.Paths["/existing"] != nil)
	fmt.Printf("Has /health: %v\n", doc.Paths["/health"] != nil)
	// Output:
	// Paths: 2
	// Has /existing: true
	// Has /health: true
}

// Example_withParameterConstraints demonstrates adding parameter constraints.
func Example_withParameterConstraints() {
	spec := builder.New(parser.OASVersion320).
		SetTitle("Pet Store API").
		SetVersion("1.0.0")

	// Add operation with constrained parameters
	spec.AddOperation(http.MethodGet, "/pets",
		builder.WithOperationID("listPets"),
		// Numeric constraint: limit must be between 1 and 100, default 20
		builder.WithQueryParam("limit", int32(0),
			builder.WithParamDescription("Maximum number of pets to return"),
			builder.WithParamMinimum(1),
			builder.WithParamMaximum(100),
			builder.WithParamDefault(20),
		),
		// Enum constraint: status must be one of the allowed values
		builder.WithQueryParam("status", "",
			builder.WithParamDescription("Filter by status"),
			builder.WithParamEnum("available", "pending", "sold"),
			builder.WithParamDefault("available"),
		),
		// String constraint: name must match pattern and length
		builder.WithQueryParam("name", "",
			builder.WithParamDescription("Filter by name"),
			builder.WithParamMinLength(1),
			builder.WithParamMaxLength(50),
			builder.WithParamPattern("^[a-zA-Z]+$"),
		),
		builder.WithResponse(http.StatusOK, []Pet{}),
	)

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	params := doc.Paths["/pets"].Get.Parameters
	fmt.Printf("Parameters: %d\n", len(params))
	fmt.Printf("limit min: %.0f\n", *params[0].Schema.Minimum)
	fmt.Printf("limit max: %.0f\n", *params[0].Schema.Maximum)
	fmt.Printf("status enum count: %d\n", len(params[1].Schema.Enum))
	fmt.Printf("name pattern: %s\n", params[2].Schema.Pattern)
	// Output:
	// Parameters: 3
	// limit min: 1
	// limit max: 100
	// status enum count: 3
	// name pattern: ^[a-zA-Z]+$
}

// Example_withParamTypeFormatOverride demonstrates explicit type and format overrides.
// Use WithParamType and WithParamFormat when the Go type doesn't map directly to the
// desired OpenAPI type/format, such as using a string for UUID identifiers or
// representing binary data as base64.
func Example_withParamTypeFormatOverride() {
	spec := builder.New(parser.OASVersion320).
		SetTitle("ID API").
		SetVersion("1.0.0")

	spec.AddOperation(http.MethodGet, "/users/{user_id}",
		builder.WithOperationID("getUser"),
		// String with UUID format (inferred type is string, explicit format)
		builder.WithPathParam("user_id", "",
			builder.WithParamFormat("uuid"),
			builder.WithParamDescription("User UUID identifier"),
		),
		// Override type to integer with int64 format
		builder.WithQueryParam("version", 0,
			builder.WithParamType("integer"),
			builder.WithParamFormat("int64"),
			builder.WithParamDescription("API version number"),
		),
		builder.WithResponse(http.StatusOK, struct {
			ID string `json:"id"`
		}{}),
	)

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	params := doc.Paths["/users/{user_id}"].Get.Parameters
	fmt.Printf("user_id format: %s\n", params[0].Schema.Format)
	fmt.Printf("version type: %s\n", params[1].Schema.Type)
	fmt.Printf("version format: %s\n", params[1].Schema.Format)
	// Output:
	// user_id format: uuid
	// version type: integer
	// version format: int64
}

// Example_withFormParameters demonstrates using form parameters.
// Form parameters work differently in OAS 2.0 vs 3.x:
//   - OAS 2.0: parameters with in="formData"
//   - OAS 3.x: request body with application/x-www-form-urlencoded
func Example_withFormParameters() {
	// OAS 3.x example - form parameters become request body
	spec := builder.New(parser.OASVersion320).
		SetTitle("Login API").
		SetVersion("1.0.0")

	type LoginResponse struct {
		Token     string `json:"token"`
		ExpiresIn int32  `json:"expires_in"`
	}

	spec.AddOperation(http.MethodPost, "/login",
		builder.WithOperationID("login"),
		builder.WithSummary("User login"),
		// Form parameters are automatically converted to request body schema
		builder.WithFormParam("username", "",
			builder.WithParamDescription("User's username"),
			builder.WithParamRequired(true),
			builder.WithParamMinLength(3),
			builder.WithParamMaxLength(20),
		),
		builder.WithFormParam("password", "",
			builder.WithParamDescription("User's password"),
			builder.WithParamRequired(true),
			builder.WithParamMinLength(8),
		),
		builder.WithFormParam("remember_me", false,
			builder.WithParamDescription("Remember login session"),
			builder.WithParamDefault(false),
		),
		builder.WithResponse(http.StatusOK, LoginResponse{},
			builder.WithResponseDescription("Successful login"),
		),
		builder.WithResponse(http.StatusUnauthorized, Error{},
			builder.WithResponseDescription("Invalid credentials"),
		),
	)

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	// Form parameters are in the request body
	rb := doc.Paths["/login"].Post.RequestBody
	mediaType := rb.Content["application/x-www-form-urlencoded"]
	fmt.Printf("Request body content type: application/x-www-form-urlencoded\n")
	fmt.Printf("Form fields: %d\n", len(mediaType.Schema.Properties))
	fmt.Printf("Required fields: %d\n", len(mediaType.Schema.Required))
	fmt.Printf("Has username: %v\n", mediaType.Schema.Properties["username"] != nil)
	fmt.Printf("Has password: %v\n", mediaType.Schema.Properties["password"] != nil)
	fmt.Printf("Has remember_me: %v\n", mediaType.Schema.Properties["remember_me"] != nil)
	// Output:
	// Request body content type: application/x-www-form-urlencoded
	// Form fields: 3
	// Required fields: 2
	// Has username: true
	// Has password: true
	// Has remember_me: true
}

// Example_withFileUpload demonstrates file upload support using WithFileParam.
func Example_withFileUpload() {
	// OAS 3.x file upload with multipart/form-data
	spec := builder.New(parser.OASVersion320).
		SetTitle("File Upload API").
		SetVersion("1.0.0")

	spec.AddOperation(http.MethodPost, "/upload",
		builder.WithOperationID("uploadFile"),
		builder.WithFileParam("file",
			builder.WithParamRequired(true),
			builder.WithParamDescription("File to upload"),
		),
		builder.WithFormParam("description", "",
			builder.WithParamDescription("File description"),
		),
		builder.WithResponse(http.StatusOK, struct {
			Success bool   `json:"success"`
			FileID  string `json:"file_id"`
		}{}),
	)

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	rb := doc.Paths["/upload"].Post.RequestBody
	schema := rb.Content["multipart/form-data"].Schema
	fmt.Printf("Has file property: %v\n", schema.Properties["file"] != nil)
	fmt.Printf("File type: %s\n", schema.Properties["file"].Type)
	fmt.Printf("File format: %s\n", schema.Properties["file"].Format)
	fmt.Printf("Has description: %v\n", schema.Properties["description"] != nil)
	fmt.Printf("Required: %v\n", schema.Required)
	// Output:
	// Has file property: true
	// File type: string
	// File format: binary
	// Has description: true
	// Required: [file]
}

// Example_withRawSchema demonstrates using raw schemas for binary data.
func Example_withRawSchema() {
	spec := builder.New(parser.OASVersion320).
		SetTitle("File Download API").
		SetVersion("1.0.0")

	// Binary file download response
	binarySchema := &parser.Schema{
		Type:   "string",
		Format: "binary",
	}

	spec.AddOperation(http.MethodGet, "/download/{id}",
		builder.WithOperationID("downloadFile"),
		builder.WithPathParam("id", int64(0),
			builder.WithParamDescription("File ID"),
		),
		builder.WithResponseRawSchema(http.StatusOK, "application/octet-stream", binarySchema,
			builder.WithResponseDescription("Binary file content"),
			builder.WithResponseHeader("Content-Disposition", &parser.Header{
				Description: "Suggested filename",
				Schema:      &parser.Schema{Type: "string"},
			}),
		),
	)

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	resp := doc.Paths["/download/{id}"].Get.Responses.Codes["200"]
	mediaType := resp.Content["application/octet-stream"]
	fmt.Printf("Response content type: application/octet-stream\n")
	fmt.Printf("Schema type: %s\n", mediaType.Schema.Type)
	fmt.Printf("Schema format: %s\n", mediaType.Schema.Format)
	fmt.Printf("Has Content-Disposition header: %v\n", resp.Headers["Content-Disposition"] != nil)
	// Output:
	// Response content type: application/octet-stream
	// Schema type: string
	// Schema format: binary
	// Has Content-Disposition header: true
}

// Example_withComplexRawSchema demonstrates using raw schemas for complex multipart uploads.
func Example_withComplexRawSchema() {
	spec := builder.New(parser.OASVersion320).
		SetTitle("Complex Upload API").
		SetVersion("1.0.0")

	// Complex multipart schema with file and metadata
	uploadSchema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"file": {
				Type:        "string",
				Format:      "binary",
				Description: "The file data",
			},
			"metadata": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"filename": {
						Type:        "string",
						Description: "Original filename",
					},
					"tags": {
						Type:        "array",
						Items:       &parser.Schema{Type: "string"},
						Description: "File tags",
					},
				},
			},
		},
		Required: []string{"file"},
	}

	spec.AddOperation(http.MethodPost, "/upload-with-metadata",
		builder.WithOperationID("uploadWithMetadata"),
		builder.WithRequestBodyRawSchema("multipart/form-data", uploadSchema,
			builder.WithRequired(true),
			builder.WithRequestDescription("Upload file with metadata"),
		),
		builder.WithResponse(http.StatusCreated, struct {
			ID string `json:"id"`
		}{}),
	)

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatal(err)
	}

	rb := doc.Paths["/upload-with-metadata"].Post.RequestBody
	schema := rb.Content["multipart/form-data"].Schema
	fmt.Printf("Request body required: %v\n", rb.Required)
	fmt.Printf("Has file property: %v\n", schema.Properties["file"] != nil)
	fmt.Printf("Has metadata property: %v\n", schema.Properties["metadata"] != nil)
	fmt.Printf("File format: %s\n", schema.Properties["file"].Format)
	fmt.Printf("Required fields: %v\n", schema.Required)
	// Output:
	// Request body required: true
	// Has file property: true
	// Has metadata property: true
	// File format: binary
	// Required fields: [file]
}

// Example_schemaNamingPascalCase demonstrates PascalCase schema naming strategy.
// With SchemaNamingPascalCase, "package.TypeName" becomes "PackageTypeName".
func Example_schemaNamingPascalCase() {
	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	spec := builder.New(parser.OASVersion320,
		builder.WithSchemaNaming(builder.SchemaNamingPascalCase),
	).
		SetTitle("Example API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/users",
			builder.WithResponse(http.StatusOK, User{}),
		)

	doc, err := spec.BuildOAS3()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Print schema names
	for name := range doc.Components.Schemas {
		fmt.Println("Schema:", name)
	}
	// Output:
	// Schema: BuilderTestUser
}

// Example_schemaNamingTemplate demonstrates custom template-based schema naming.
// Templates use Go text/template syntax with helper functions like pascal, camel, etc.
func Example_schemaNamingTemplate() {
	type Product struct {
		ID    int     `json:"id"`
		Price float64 `json:"price"`
	}

	// Custom template: prefix with "API" and use pascal case
	spec := builder.New(parser.OASVersion320,
		builder.WithSchemaNameTemplate(`API{{pascal .Type}}`),
	).
		SetTitle("Example API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/products",
			builder.WithResponse(http.StatusOK, Product{}),
		)

	doc, err := spec.BuildOAS3()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Print schema names
	for name := range doc.Components.Schemas {
		fmt.Println("Schema:", name)
	}
	// Output:
	// Schema: APIProduct
}

// Example_schemaNamingCustomFunc demonstrates custom function-based schema naming.
// Use WithSchemaNameFunc for maximum flexibility when you need programmatic control
// over schema names based on type metadata.
func Example_schemaNamingCustomFunc() {
	type Order struct {
		ID     int64   `json:"id"`
		Total  float64 `json:"total"`
		Status string  `json:"status"`
	}

	// Custom naming function that prefixes schemas with API version
	// and converts the type name to uppercase
	apiVersion := "V2"
	customNamer := func(ctx builder.SchemaNameContext) string {
		// Use the type name, converting to uppercase for emphasis
		return apiVersion + "_" + ctx.Type
	}

	spec := builder.New(parser.OASVersion320,
		builder.WithSchemaNameFunc(customNamer),
	).
		SetTitle("Order API").
		SetVersion("2.0.0").
		AddOperation(http.MethodGet, "/orders",
			builder.WithResponse(http.StatusOK, Order{}),
		)

	doc, err := spec.BuildOAS3()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Print schema names
	for name := range doc.Components.Schemas {
		fmt.Println("Schema:", name)
	}
	// Output:
	// Schema: V2_Order
}

// Example_genericNamingConfig demonstrates fine-grained generic type naming configuration.
// Use WithGenericNamingConfig for full control over how generic type parameters are formatted.
func Example_genericNamingConfig() {
	// Configure generic naming with custom settings.
	// This example uses GenericNamingOf strategy with "And" as the separator
	// between multiple type parameters.
	//
	// For generic types like Response[User], this would produce "ResponseOfUser".
	// For types like Map[string,int], this would produce "MapOfStringAndOfInt".
	spec := builder.New(parser.OASVersion320,
		builder.WithGenericNamingConfig(builder.GenericNamingConfig{
			Strategy:        builder.GenericNamingOf,
			ParamSeparator:  "And",
			ApplyBaseCasing: true,
		}),
	).
		SetTitle("Example API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/pets",
			builder.WithResponse(http.StatusOK, []Pet{}),
		)

	doc, err := spec.BuildOAS3()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Title: %s\n", doc.Info.Title)
	fmt.Printf("Schemas: %d\n", len(doc.Components.Schemas))
	// Output:
	// Title: Example API
	// Schemas: 1
}

// Example_semanticDeduplication demonstrates automatic consolidation of identical schemas.
// When multiple Go types generate structurally identical schemas, enabling semantic
// deduplication identifies these duplicates and consolidates them to a single canonical schema.
func Example_semanticDeduplication() {
	// Define types that are structurally identical but have different names
	type UserID struct {
		Value int64 `json:"value"`
	}
	type CustomerID struct {
		Value int64 `json:"value"`
	}
	type OrderID struct {
		Value int64 `json:"value"`
	}

	// Build specification with semantic deduplication enabled
	spec := builder.New(parser.OASVersion320,
		builder.WithSemanticDeduplication(true),
	).
		SetTitle("ID API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/users/{id}",
			builder.WithResponse(http.StatusOK, UserID{}),
		).
		AddOperation(http.MethodGet, "/customers/{id}",
			builder.WithResponse(http.StatusOK, CustomerID{}),
		).
		AddOperation(http.MethodGet, "/orders/{id}",
			builder.WithResponse(http.StatusOK, OrderID{}),
		)

	doc, err := spec.BuildOAS3()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Without deduplication: 3 schemas (UserID, CustomerID, OrderID)
	// With deduplication: 1 schema (the alphabetically first canonical name)
	fmt.Printf("Title: %s\n", doc.Info.Title)
	fmt.Printf("Schemas: %d\n", len(doc.Components.Schemas))
	fmt.Printf("Operations: %d\n", len(doc.Paths))
	// Output:
	// Title: ID API
	// Schemas: 1
	// Operations: 3
}
