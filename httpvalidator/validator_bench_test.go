package httpvalidator

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/erraggy/oastools/parser"
)

// Note on b.Fatalf usage in benchmarks:
// Using b.Fatalf for errors in benchmark setup or execution is an acceptable pattern.
// These operations should never fail with valid test fixtures.
// If they do fail, it indicates a bug that should halt the benchmark immediately.

// Benchmark fixtures
const (
	smallOAS3Path  = "../testdata/bench/small-oas3.yaml"
	mediumOAS3Path = "../testdata/bench/medium-oas3.yaml"
	largeOAS3Path  = "../testdata/bench/large-oas3.yaml"
)

// BenchmarkValidateRequest benchmarks HTTP request validation
func BenchmarkValidateRequest(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"SmallOAS3", smallOAS3Path},
		{"MediumOAS3", mediumOAS3Path},
		{"LargeOAS3", largeOAS3Path},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// Parse spec once
			parsed, err := parser.ParseWithOptions(
				parser.WithFilePath(tt.path),
			)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			// Create validator
			v, err := New(parsed)
			if err != nil {
				b.Fatalf("Failed to create validator: %v", err)
			}

			// Create sample request (path exists in all test specs)
			req, _ := http.NewRequest("GET", "/api/users", nil)

			for b.Loop() {
				_, err := v.ValidateRequest(req)
				if err != nil {
					b.Fatalf("Validation failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkValidateRequestWithParams benchmarks request validation with parameters
func BenchmarkValidateRequestWithParams(b *testing.B) {
	// Parse spec once
	parsed, err := parser.ParseWithOptions(
		parser.WithFilePath(smallOAS3Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	// Create validator
	v, err := New(parsed)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create request with query parameters
	req, _ := http.NewRequest("GET", "/api/users?page=1&limit=10&sort=name", nil)

	for b.Loop() {
		_, err := v.ValidateRequest(req)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

// BenchmarkValidateRequestWithBody benchmarks request validation with JSON body
func BenchmarkValidateRequestWithBody(b *testing.B) {
	// Parse spec once
	parsed, err := parser.ParseWithOptions(
		parser.WithFilePath(smallOAS3Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	// Create validator
	v, err := New(parsed)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create request with JSON body
	body := []byte(`{"name":"John Doe","email":"john@example.com"}`)

	for b.Loop() {
		req, _ := http.NewRequest("POST", "/api/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		_, err := v.ValidateRequest(req)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

// BenchmarkValidateResponse benchmarks HTTP response validation
func BenchmarkValidateResponse(b *testing.B) {
	// Parse spec once
	parsed, err := parser.ParseWithOptions(
		parser.WithFilePath(smallOAS3Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	// Create validator
	v, err := New(parsed)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create sample request and response
	req, _ := http.NewRequest("GET", "/api/users", nil)
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	body := []byte(`{"users":[{"id":"1","name":"John"}]}`)

	for b.Loop() {
		_, err := v.ValidateResponseData(req, 200, headers, body)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

// BenchmarkValidateStrictMode benchmarks strict mode validation
func BenchmarkValidateStrictMode(b *testing.B) {
	// Parse spec once
	parsed, err := parser.ParseWithOptions(
		parser.WithFilePath(smallOAS3Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	// Create validator with strict mode
	v, err := New(parsed)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}
	v.StrictMode = true

	// Create request
	req, _ := http.NewRequest("GET", "/api/users?page=1", nil)

	for b.Loop() {
		_, err := v.ValidateRequest(req)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

// BenchmarkValidateRequestWithOptions benchmarks the functional options API
func BenchmarkValidateRequestWithOptions(b *testing.B) {
	b.Run("FilePath/SmallOAS3", func(b *testing.B) {
		req, _ := http.NewRequest("GET", "/api/users", nil)

		for b.Loop() {
			_, err := ValidateRequestWithOptions(
				req,
				WithFilePath(smallOAS3Path),
				WithStrictMode(false),
			)
			if err != nil {
				b.Fatalf("Failed to validate: %v", err)
			}
		}
	})

	b.Run("Parsed/SmallOAS3", func(b *testing.B) {
		// Parse once
		parsed, err := parser.ParseWithOptions(
			parser.WithFilePath(smallOAS3Path),
		)
		if err != nil {
			b.Fatal(err)
		}

		req, _ := http.NewRequest("GET", "/api/users", nil)

		for b.Loop() {
			_, err := ValidateRequestWithOptions(
				req,
				WithParsed(parsed),
			)
			if err != nil {
				b.Fatalf("Failed to validate: %v", err)
			}
		}
	})
}

// BenchmarkPathMatching benchmarks path template matching
func BenchmarkPathMatching(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"SmallOAS3", smallOAS3Path},
		{"MediumOAS3", mediumOAS3Path},
		{"LargeOAS3", largeOAS3Path},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// Parse spec once
			parsed, err := parser.ParseWithOptions(
				parser.WithFilePath(tt.path),
			)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			// Create validator (initializes path matchers)
			v, err := New(parsed)
			if err != nil {
				b.Fatalf("Failed to create validator: %v", err)
			}

			// Create request
			req, _ := http.NewRequest("GET", "/api/users/123", nil)

			for b.Loop() {
				// Just test path matching performance (internal operation)
				_, err := v.ValidateRequest(req)
				if err != nil {
					b.Fatalf("Validation failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkParameterDeserialization benchmarks parameter deserialization performance
func BenchmarkParameterDeserialization(b *testing.B) {
	// Parse spec once
	parsed, err := parser.ParseWithOptions(
		parser.WithFilePath(smallOAS3Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	// Create validator
	v, err := New(parsed)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Request with multiple parameters of different types
	req, _ := http.NewRequest("GET", "/api/users?page=1&limit=10&tags=foo,bar,baz&sort=name&active=true", nil)

	for b.Loop() {
		_, err := v.ValidateRequest(req)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

// BenchmarkSchemaValidation benchmarks JSON schema validation performance
func BenchmarkSchemaValidation(b *testing.B) {
	// Parse spec once
	parsed, err := parser.ParseWithOptions(
		parser.WithFilePath(smallOAS3Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	// Create validator
	v, err := New(parsed)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Complex JSON body
	body := []byte(`{
		"name": "John Doe",
		"email": "john@example.com",
		"age": 30,
		"tags": ["developer", "golang"],
		"address": {
			"street": "123 Main St",
			"city": "Springfield",
			"zip": "12345"
		}
	}`)

	for b.Loop() {
		req, _ := http.NewRequest("POST", "/api/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		_, err := v.ValidateRequest(req)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}
