// HTTP Validation example demonstrating the httpvalidator package.
//
// This example shows how to:
//   - Create an HTTP validator from an OpenAPI spec
//   - Validate request parameters and bodies
//   - Extract path parameters from matched routes
//   - Validate response data against schema
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/erraggy/oastools/httpvalidator"
	"github.com/erraggy/oastools/parser"
)

// sensitiveHeaders lists header names that may contain credentials.
// Values from these headers should never be logged.
var sensitiveHeaders = map[string]bool{
	"authorization":   true,
	"x-api-key":       true,
	"x-auth-token":    true,
	"cookie":          true,
	"set-cookie":      true,
	"x-csrf-token":    true,
	"x-access-token":  true,
	"x-refresh-token": true,
}

// valuePattern matches quoted values in error messages like: value 'secret' or value "secret"
var valuePattern = regexp.MustCompile(`(value\s+)'[^']*'`)

func main() {
	specPath := findSpecPath("specs/api.yaml")

	fmt.Println("HTTP Validation Workflow")
	fmt.Println("========================")
	fmt.Println()

	// Step 1: Parse the spec and create validator
	fmt.Println("[1/6] Creating HTTP validator...")
	parsed, err := parser.ParseWithOptions(
		parser.WithFilePath(specPath),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	v, err := httpvalidator.New(parsed)
	if err != nil {
		log.Fatalf("Validator error: %v", err)
	}

	v.StrictMode = false // Allow unknown headers
	fmt.Printf("      Validator created, strict mode: %t\n", v.StrictMode)

	// Step 2: Valid GET request
	fmt.Println()
	fmt.Println("[2/6] Validating GET /todos?status=pending&limit=10...")
	req := httptest.NewRequest("GET", "/todos?status=pending&limit=10", nil)
	result, err := v.ValidateRequest(req)
	if err != nil {
		log.Fatalf("Validation error: %v", err)
	}
	printRequestResult(result)

	// Step 3: Invalid GET request (bad enum value)
	fmt.Println()
	fmt.Println("[3/6] Validating GET /todos?status=invalid...")
	req = httptest.NewRequest("GET", "/todos?status=invalid", nil)
	result, err = v.ValidateRequest(req)
	if err != nil {
		log.Fatalf("Validation error: %v", err)
	}
	printRequestResult(result)

	// Step 4: Valid POST request
	fmt.Println()
	fmt.Println("[4/6] Validating POST /todos with valid body...")
	body := `{"title": "Write documentation", "description": "Update README"}`
	req = httptest.NewRequest("POST", "/todos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	result, err = v.ValidateRequest(req)
	if err != nil {
		log.Fatalf("Validation error: %v", err)
	}
	printRequestResult(result)

	// Step 5: Invalid POST request (missing required field)
	fmt.Println()
	fmt.Println("[5/6] Validating POST /todos with invalid body...")
	body = `{"description": "Missing title field"}`
	req = httptest.NewRequest("POST", "/todos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	result, err = v.ValidateRequest(req)
	if err != nil {
		log.Fatalf("Validation error: %v", err)
	}
	printRequestResult(result)

	// Step 6: Path parameter extraction
	fmt.Println()
	fmt.Println("[6/6] Path parameter extraction...")
	req = httptest.NewRequest("GET", "/todos/42", nil)
	result, err = v.ValidateRequest(req)
	if err != nil {
		log.Fatalf("Validation error: %v", err)
	}
	fmt.Printf("      Matched Path: %s\n", result.MatchedPath)
	fmt.Printf("      todoId: %v\n", result.PathParams["todoId"])
	fmt.Printf("      Valid: %t\n", result.Valid)

	// Bonus: Response validation
	fmt.Println()
	fmt.Println("[Bonus] Response validation...")
	req = httptest.NewRequest("GET", "/todos/1", nil)
	responseBody := []byte(`{"id": 1, "title": "Test", "completed": false}`)
	respResult, err := v.ValidateResponseData(req, 200,
		http.Header{"Content-Type": []string{"application/json"}},
		responseBody)
	if err != nil {
		log.Fatalf("Response validation error: %v", err)
	}
	fmt.Printf("      Response Valid: %t\n", respResult.Valid)
	fmt.Printf("      Status Code: %d\n", respResult.StatusCode)

	fmt.Println()
	fmt.Println("---")
	fmt.Println("HTTP Validation examples complete")
}

// printRequestResult displays validation results with sanitized error messages.
// This demonstrates best practices for logging validation errors without
// exposing sensitive header values or request body content.
func printRequestResult(r *httpvalidator.RequestValidationResult) {
	fmt.Printf("      Valid: %t\n", r.Valid)
	if r.MatchedPath != "" {
		fmt.Printf("      Matched Path: %s\n", r.MatchedPath)
	}
	if len(r.Errors) > 0 {
		fmt.Printf("      Errors (%d):\n", len(r.Errors))
		for _, e := range r.Errors {
			path := e.Path
			if path == "" {
				path = "(request)"
			}
			// Always sanitize error messages before logging to avoid
			// exposing sensitive header values or request body content.
			safeMessage := sanitizeErrorMessage(e.Path, e.Message)
			fmt.Printf("        - [%s] %s\n", path, safeMessage)
		}
	}
}

// sanitizeErrorMessage redacts potentially sensitive values from validation
// error messages. This is critical for production logging where validation
// errors for headers like Authorization could expose credentials.
//
// Example transformations:
//
//	"header 'Authorization' value 'Bearer sk-live-xxx' invalid" → "header 'Authorization' value '[REDACTED]' invalid"
//	"query 'status' value 'pending' not in enum" → "query 'status' value 'pending' not in enum" (safe, not sensitive)
func sanitizeErrorMessage(path, message string) string {
	// Check if this error relates to a sensitive header
	pathLower := strings.ToLower(path)
	for header := range sensitiveHeaders {
		if strings.Contains(pathLower, header) {
			// Redact any quoted values in the message
			return valuePattern.ReplaceAllString(message, "${1}'[REDACTED]'")
		}
	}

	// For non-sensitive paths (query params, body fields), keep the full message
	// as these values are typically not credentials and aid debugging
	return message
}

// findSpecPath locates a file relative to the source file location.
func findSpecPath(relativePath string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Unable to get current file path")
	}
	return filepath.Join(filepath.Dir(filename), relativePath)
}
