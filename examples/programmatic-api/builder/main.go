// Builder example demonstrating the programmatic API for constructing OpenAPI specs.
//
// This example shows how to:
//   - Create an OpenAPI spec from scratch using the fluent builder API
//   - Define operations with parameters, request bodies, and responses
//   - Use Go struct tags for automatic schema generation
//   - Configure security schemes
//   - Build and serialize the final specification
//   - Create a runnable HTTP server with ServerBuilder
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/erraggy/oastools/builder"
	"github.com/erraggy/oastools/parser"
)

// Book represents a book in our catalog.
// Note the `oas` struct tags for schema customization.
type Book struct {
	ID          int64     `json:"id" oas:"description=Unique book identifier,readOnly=true"`
	Title       string    `json:"title" oas:"description=Book title,minLength=1,maxLength=200"`
	Author      string    `json:"author" oas:"description=Author name,minLength=1"`
	ISBN        string    `json:"isbn" oas:"description=ISBN-13 identifier,pattern=^\\d{13}$"`
	Genre       string    `json:"genre" oas:"description=Book genre,enum=fiction|non-fiction|sci-fi|fantasy|mystery"`
	PublishedAt time.Time `json:"published_at" oas:"description=Publication date"`
	InStock     bool      `json:"in_stock" oas:"description=Stock availability"`
}

// CreateBookRequest is the request body for creating a book.
type CreateBookRequest struct {
	Title  string `json:"title" oas:"description=Book title,minLength=1,maxLength=200"`
	Author string `json:"author" oas:"description=Author name,minLength=1"`
	ISBN   string `json:"isbn" oas:"description=ISBN-13 identifier,pattern=^\\d{13}$"`
	Genre  string `json:"genre" oas:"description=Book genre,enum=fiction|non-fiction|sci-fi|fantasy|mystery"`
}

// UpdateBookRequest is the request body for updating a book.
type UpdateBookRequest struct {
	Title   *string `json:"title,omitempty" oas:"description=Book title,minLength=1,maxLength=200"`
	Author  *string `json:"author,omitempty" oas:"description=Author name,minLength=1"`
	InStock *bool   `json:"in_stock,omitempty" oas:"description=Stock availability"`
}

// Error represents an API error response.
type Error struct {
	Code    string `json:"code" oas:"description=Error code"`
	Message string `json:"message" oas:"description=Human-readable error message"`
}

func main() {
	fmt.Println("Builder Workflow")
	fmt.Println("================")
	fmt.Println()

	// Step 1: Create the builder with OAS 3.2.0 version
	fmt.Println("[1/6] Creating OpenAPI 3.2.0 spec builder...")
	spec := builder.New(parser.OASVersion320).
		SetTitle("Book Store API").
		SetVersion("1.0.0").
		SetDescription("A sample API demonstrating the oastools builder package")

	fmt.Println("      Base spec created")

	// Step 2: Add servers
	fmt.Println()
	fmt.Println("[2/6] Adding servers...")
	spec.AddServer("https://api.bookstore.example.com/v1",
		builder.WithServerDescription("Production server"),
	).AddServer("https://staging.bookstore.example.com/v1",
		builder.WithServerDescription("Staging server"),
	)
	fmt.Println("      Added 2 servers (production + staging)")

	// Step 3: Add tags for API organization
	fmt.Println()
	fmt.Println("[3/6] Adding tags...")
	spec.AddTag("books", builder.WithTagDescription("Operations for managing books"))
	fmt.Println("      Added 'books' tag")

	// Step 4: Configure security
	fmt.Println()
	fmt.Println("[4/6] Configuring security...")
	spec.AddAPIKeySecurityScheme(
		"api_key",
		"header",
		"X-API-Key",
		"API key for authentication",
	).SetSecurity(builder.SecurityRequirement("api_key"))
	fmt.Println("      Added API key security scheme (header: X-API-Key)")

	// Step 5: Add operations
	fmt.Println()
	fmt.Println("[5/6] Adding operations...")

	// GET /books - List all books
	spec.AddOperation(http.MethodGet, "/books",
		builder.WithOperationID("listBooks"),
		builder.WithSummary("List all books"),
		builder.WithTags("books"),
		builder.WithQueryParam("genre", "",
			builder.WithParamDescription("Filter by genre"),
			builder.WithParamEnum("fiction", "non-fiction", "sci-fi", "fantasy", "mystery"),
		),
		builder.WithQueryParam("limit", int32(0),
			builder.WithParamDescription("Maximum number of books to return"),
			builder.WithParamMinimum(1),
			builder.WithParamMaximum(100),
			builder.WithParamDefault(20),
		),
		builder.WithQueryParam("offset", int32(0),
			builder.WithParamDescription("Number of books to skip"),
			builder.WithParamMinimum(0),
			builder.WithParamDefault(0),
		),
		builder.WithResponse(http.StatusOK, []Book{},
			builder.WithResponseDescription("List of books"),
		),
		builder.WithResponse(http.StatusInternalServerError, Error{},
			builder.WithResponseDescription("Unexpected error"),
		),
	)
	fmt.Println("      ✓ GET /books (listBooks)")

	// POST /books - Create a book
	spec.AddOperation(http.MethodPost, "/books",
		builder.WithOperationID("createBook"),
		builder.WithSummary("Create a new book"),
		builder.WithTags("books"),
		builder.WithRequestBody("application/json", CreateBookRequest{},
			builder.WithRequired(true),
			builder.WithRequestDescription("Book to create"),
		),
		builder.WithResponse(http.StatusCreated, Book{},
			builder.WithResponseDescription("Book created successfully"),
		),
		builder.WithResponse(http.StatusBadRequest, Error{},
			builder.WithResponseDescription("Invalid request body"),
		),
	)
	fmt.Println("      ✓ POST /books (createBook)")

	// GET /books/{bookId} - Get a specific book
	spec.AddOperation(http.MethodGet, "/books/{bookId}",
		builder.WithOperationID("getBook"),
		builder.WithSummary("Get a book by ID"),
		builder.WithTags("books"),
		builder.WithPathParam("bookId", int64(0),
			builder.WithParamDescription("The ID of the book to retrieve"),
		),
		builder.WithResponse(http.StatusOK, Book{},
			builder.WithResponseDescription("Book details"),
		),
		builder.WithResponse(http.StatusNotFound, Error{},
			builder.WithResponseDescription("Book not found"),
		),
	)
	fmt.Println("      ✓ GET /books/{bookId} (getBook)")

	// PUT /books/{bookId} - Update a book
	spec.AddOperation(http.MethodPut, "/books/{bookId}",
		builder.WithOperationID("updateBook"),
		builder.WithSummary("Update a book"),
		builder.WithTags("books"),
		builder.WithPathParam("bookId", int64(0),
			builder.WithParamDescription("The ID of the book to update"),
		),
		builder.WithRequestBody("application/json", UpdateBookRequest{},
			builder.WithRequired(true),
			builder.WithRequestDescription("Fields to update"),
		),
		builder.WithResponse(http.StatusOK, Book{},
			builder.WithResponseDescription("Book updated successfully"),
		),
		builder.WithResponse(http.StatusNotFound, Error{},
			builder.WithResponseDescription("Book not found"),
		),
	)
	fmt.Println("      ✓ PUT /books/{bookId} (updateBook)")

	// DELETE /books/{bookId} - Delete a book
	spec.AddOperation(http.MethodDelete, "/books/{bookId}",
		builder.WithOperationID("deleteBook"),
		builder.WithSummary("Delete a book"),
		builder.WithTags("books"),
		builder.WithPathParam("bookId", int64(0),
			builder.WithParamDescription("The ID of the book to delete"),
		),
		builder.WithResponse(http.StatusNoContent, nil,
			builder.WithResponseDescription("Book deleted successfully"),
		),
		builder.WithResponse(http.StatusNotFound, Error{},
			builder.WithResponseDescription("Book not found"),
		),
	)
	fmt.Println("      ✓ DELETE /books/{bookId} (deleteBook)")

	// Step 6: Build and display results
	fmt.Println()
	fmt.Println("[6/6] Building specification...")

	doc, err := spec.BuildOAS3()
	if err != nil {
		log.Fatalf("Build error: %v", err)
	}

	fmt.Println("      Build successful!")

	// Display summary
	fmt.Println()
	fmt.Println("--- Specification Summary ---")
	fmt.Printf("OpenAPI Version: %s\n", doc.OpenAPI)
	fmt.Printf("Title: %s\n", doc.Info.Title)
	fmt.Printf("Version: %s\n", doc.Info.Version)
	fmt.Printf("Servers: %d\n", len(doc.Servers))
	fmt.Printf("Tags: %d\n", len(doc.Tags))
	fmt.Printf("Paths: %d\n", len(doc.Paths))
	fmt.Printf("Schemas: %d\n", len(doc.Components.Schemas))
	fmt.Printf("Security Schemes: %d\n", len(doc.Components.SecuritySchemes))

	// Count total operations
	var opCount int
	for _, pathItem := range doc.Paths {
		if pathItem.Get != nil {
			opCount++
		}
		if pathItem.Post != nil {
			opCount++
		}
		if pathItem.Put != nil {
			opCount++
		}
		if pathItem.Delete != nil {
			opCount++
		}
	}
	fmt.Printf("Operations: %d\n", opCount)

	// List generated schemas
	fmt.Println()
	fmt.Println("Generated Schemas:")
	for name := range doc.Components.Schemas {
		fmt.Printf("  - %s\n", name)
	}

	// Show serialization preview
	fmt.Println()
	fmt.Println("Paths defined:")
	for path := range doc.Paths {
		fmt.Printf("  - %s\n", path)
	}

	// Bonus: ServerBuilder demo
	fmt.Println()
	fmt.Println("=================================")
	fmt.Println()
	fmt.Println("[Bonus] ServerBuilder - Runnable HTTP Server")
	fmt.Println()
	demoServerBuilder()

	fmt.Println()
	fmt.Println("---")
	fmt.Println("Builder example complete")
}

// demoServerBuilder shows how to create a runnable HTTP server using ServerBuilder.
// ServerBuilder extends Builder to add handler registration and server building.
func demoServerBuilder() {
	fmt.Println("[1/3] Creating ServerBuilder...")

	// ServerBuilder extends Builder with server capabilities
	srv := builder.NewServerBuilder(parser.OASVersion320, builder.WithoutValidation()).
		SetTitle("Quick API").
		SetVersion("1.0.0")

	fmt.Println("      ServerBuilder created")

	// Add operations (same API as Builder)
	fmt.Println()
	fmt.Println("[2/3] Adding operations and handlers...")

	type StatusResponse struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}

	type MessageRequest struct {
		Text string `json:"text"`
	}

	type MessageResponse struct {
		ID   int    `json:"id"`
		Text string `json:"text"`
	}

	// GET /status - health check
	srv.AddOperation(http.MethodGet, "/status",
		builder.WithOperationID("getStatus"),
		builder.WithResponse(http.StatusOK, StatusResponse{}),
	)

	// Register handler for the operation
	srv.Handle(http.MethodGet, "/status", func(_ context.Context, _ *builder.Request) builder.Response {
		return builder.JSON(http.StatusOK, StatusResponse{
			Status:  "ok",
			Version: "1.0.0",
		})
	})
	fmt.Println("      ✓ GET /status with handler")

	// POST /messages - create message
	srv.AddOperation(http.MethodPost, "/messages",
		builder.WithOperationID("createMessage"),
		builder.WithRequestBody("application/json", MessageRequest{},
			builder.WithRequired(true),
		),
		builder.WithResponse(http.StatusCreated, MessageResponse{}),
	)

	srv.Handle(http.MethodPost, "/messages", func(_ context.Context, req *builder.Request) builder.Response {
		// req.Body contains the parsed request
		return builder.JSON(http.StatusCreated, MessageResponse{
			ID:   1,
			Text: "Message received",
		})
	})
	fmt.Println("      ✓ POST /messages with handler")

	// Build the server
	fmt.Println()
	fmt.Println("[3/3] Building server...")

	result, err := srv.BuildServer()
	if err != nil {
		log.Fatalf("Server build error: %v", err)
	}

	fmt.Println("      Server built successfully!")

	// Display results
	fmt.Println()
	fmt.Println("--- Server Summary ---")
	fmt.Printf("Handler Type: %T\n", result.Handler)
	fmt.Printf("Has Spec: %v\n", result.Spec != nil)
	fmt.Printf("Has Validator: %v\n", result.Validator != nil)

	// Show that we can test the handler
	fmt.Println()
	fmt.Println("Testing with ServerTest helper:")
	test := builder.NewServerTest(result)

	var status StatusResponse
	rec, err := test.GetJSON("/status", &status)
	if err != nil {
		log.Fatalf("Test error: %v", err)
	}

	fmt.Printf("  GET /status → %d\n", rec.Code)
	fmt.Printf("  Response: {status: %q, version: %q}\n", status.Status, status.Version)

	fmt.Println()
	fmt.Println("Server is ready to run with http.ListenAndServe(\":8080\", result.Handler)")
}
