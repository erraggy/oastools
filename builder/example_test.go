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
			builder.WithQueryParam("include", string(""),
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
