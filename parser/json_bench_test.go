package parser

import (
	"encoding/json"
	"testing"
)

// BenchmarkMarshalInfoNoExtra benchmarks marshaling Info without Extra fields
func BenchmarkMarshalInfoNoExtra(b *testing.B) {
	info := &Info{
		Title:       "Test API",
		Version:     "1.0.0",
		Description: "A test API for benchmarking",
	}

	for b.Loop() {
		_, err := json.Marshal(info)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalInfoWithExtra benchmarks marshaling Info with Extra fields
func BenchmarkMarshalInfoWithExtra(b *testing.B) {
	info := &Info{
		Title:       "Test API",
		Version:     "1.0.0",
		Description: "A test API for benchmarking",
		Extra: map[string]interface{}{
			"x-api-id":       "test-001",
			"x-audience":     "internal",
			"x-team":         "platform",
			"x-environment":  "production",
			"x-custom-field": "custom-value",
		},
	}

	for b.Loop() {
		_, err := json.Marshal(info)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalContactNoExtra benchmarks marshaling Contact without Extra fields
func BenchmarkMarshalContactNoExtra(b *testing.B) {
	contact := &Contact{
		Name:  "API Support",
		Email: "support@example.com",
		URL:   "https://example.com/support",
	}

	for b.Loop() {
		_, err := json.Marshal(contact)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalContactWithExtra benchmarks marshaling Contact with Extra fields
func BenchmarkMarshalContactWithExtra(b *testing.B) {
	contact := &Contact{
		Name:  "API Support",
		Email: "support@example.com",
		URL:   "https://example.com/support",
		Extra: map[string]interface{}{
			"x-team-id":   "platform-001",
			"x-slack":     "#api-support",
			"x-on-call":   true,
			"x-timezone":  "UTC",
			"x-languages": []string{"en", "es", "fr"},
		},
	}

	for b.Loop() {
		_, err := json.Marshal(contact)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalServerNoExtra benchmarks marshaling Server without Extra fields
func BenchmarkMarshalServerNoExtra(b *testing.B) {
	server := &Server{
		URL:         "https://api.example.com/v1",
		Description: "Production server",
	}

	for b.Loop() {
		_, err := json.Marshal(server)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalServerWithExtra benchmarks marshaling Server with Extra fields
func BenchmarkMarshalServerWithExtra(b *testing.B) {
	server := &Server{
		URL:         "https://api.example.com/v1",
		Description: "Production server",
		Extra: map[string]interface{}{
			"x-environment":    "production",
			"x-region":         "us-east-1",
			"x-load-balancer":  "alb-prod-001",
			"x-health-check":   "/health",
			"x-rate-limit":     10000,
			"x-cache-enabled":  true,
			"x-timeout-ms":     5000,
			"x-retry-attempts": 3,
		},
	}

	for b.Loop() {
		_, err := json.Marshal(server)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalOAS3DocumentSmall benchmarks marshaling a small OAS3 document
func BenchmarkMarshalOAS3DocumentSmall(b *testing.B) {
	// Parse a small document first
	result, err := Parse(smallOAS3Path, false, false)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	doc, ok := result.Document.(*OAS3Document)
	if !ok {
		b.Fatalf("Expected *OAS3Document, got %T", result.Document)
	}

	for b.Loop() {
		_, err := json.Marshal(doc)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalOAS3DocumentMedium benchmarks marshaling a medium OAS3 document
func BenchmarkMarshalOAS3DocumentMedium(b *testing.B) {
	// Parse a medium document first
	result, err := Parse(mediumOAS3Path, false, false)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	doc, ok := result.Document.(*OAS3Document)
	if !ok {
		b.Fatalf("Expected *OAS3Document, got %T", result.Document)
	}

	for b.Loop() {
		_, err := json.Marshal(doc)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalOAS3DocumentLarge benchmarks marshaling a large OAS3 document
func BenchmarkMarshalOAS3DocumentLarge(b *testing.B) {
	// Parse a large document first
	result, err := Parse(largeOAS3Path, false, false)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	doc, ok := result.Document.(*OAS3Document)
	if !ok {
		b.Fatalf("Expected *OAS3Document, got %T", result.Document)
	}

	for b.Loop() {
		_, err := json.Marshal(doc)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalOAS2DocumentSmall benchmarks marshaling a small OAS2 document
func BenchmarkMarshalOAS2DocumentSmall(b *testing.B) {
	// Parse a small document first
	result, err := Parse(smallOAS2Path, false, false)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	doc, ok := result.Document.(*OAS2Document)
	if !ok {
		b.Fatalf("Expected *OAS2Document, got %T", result.Document)
	}

	for b.Loop() {
		_, err := json.Marshal(doc)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalOAS2DocumentMedium benchmarks marshaling a medium OAS2 document
func BenchmarkMarshalOAS2DocumentMedium(b *testing.B) {
	// Parse a medium document first
	result, err := Parse(mediumOAS2Path, false, false)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	doc, ok := result.Document.(*OAS2Document)
	if !ok {
		b.Fatalf("Expected *OAS2Document, got %T", result.Document)
	}

	for b.Loop() {
		_, err := json.Marshal(doc)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkUnmarshalInfoNoExtra benchmarks unmarshaling Info without Extra fields
func BenchmarkUnmarshalInfoNoExtra(b *testing.B) {
	data := []byte(`{"title":"Test API","version":"1.0.0","description":"A test API"}`)

	for b.Loop() {
		var info Info
		err := json.Unmarshal(data, &info)
		if err != nil {
			b.Fatalf("Failed to unmarshal: %v", err)
		}
	}
}

// BenchmarkUnmarshalInfoWithExtra benchmarks unmarshaling Info with Extra fields
func BenchmarkUnmarshalInfoWithExtra(b *testing.B) {
	data := []byte(`{
		"title":"Test API",
		"version":"1.0.0",
		"description":"A test API",
		"x-api-id":"test-001",
		"x-audience":"internal",
		"x-team":"platform",
		"x-environment":"production"
	}`)

	for b.Loop() {
		var info Info
		err := json.Unmarshal(data, &info)
		if err != nil {
			b.Fatalf("Failed to unmarshal: %v", err)
		}
	}
}
