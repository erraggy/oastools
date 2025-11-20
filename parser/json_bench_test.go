package parser

import (
	"encoding/json"
	"testing"
)

// Note on b.Fatalf usage in benchmarks:
// Using b.Fatalf for errors in benchmark setup or execution is an acceptable pattern.
// These operations (marshal, unmarshal, parse) should never fail with valid test fixtures.
// If they do fail, it indicates a bug that should halt the benchmark immediately.

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
		Extra: map[string]any{
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

// BenchmarkMarshalInfoExtra1 benchmarks marshaling Info with 1 Extra field
func BenchmarkMarshalInfoExtra1(b *testing.B) {
	info := &Info{
		Title:       "Test API",
		Version:     "1.0.0",
		Description: "A test API for benchmarking",
		Extra: map[string]any{
			"x-api-id": "test-001",
		},
	}

	for b.Loop() {
		_, err := json.Marshal(info)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalInfoExtra5 benchmarks marshaling Info with 5 Extra fields
func BenchmarkMarshalInfoExtra5(b *testing.B) {
	info := &Info{
		Title:       "Test API",
		Version:     "1.0.0",
		Description: "A test API for benchmarking",
		Extra: map[string]any{
			"x-api-id":      "test-001",
			"x-audience":    "internal",
			"x-team":        "platform",
			"x-environment": "production",
			"x-region":      "us-east-1",
		},
	}

	for b.Loop() {
		_, err := json.Marshal(info)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalInfoExtra10 benchmarks marshaling Info with 10 Extra fields
func BenchmarkMarshalInfoExtra10(b *testing.B) {
	info := &Info{
		Title:       "Test API",
		Version:     "1.0.0",
		Description: "A test API for benchmarking",
		Extra: map[string]any{
			"x-api-id":       "test-001",
			"x-audience":     "internal",
			"x-team":         "platform",
			"x-environment":  "production",
			"x-region":       "us-east-1",
			"x-owner":        "platform-team",
			"x-cost-center":  "engineering",
			"x-project":      "api-gateway",
			"x-service-tier": "gold",
			"x-compliance":   "pci-dss",
		},
	}

	for b.Loop() {
		_, err := json.Marshal(info)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalInfoExtra20 benchmarks marshaling Info with 20 Extra fields
func BenchmarkMarshalInfoExtra20(b *testing.B) {
	info := &Info{
		Title:       "Test API",
		Version:     "1.0.0",
		Description: "A test API for benchmarking",
		Extra: map[string]any{
			"x-api-id":        "test-001",
			"x-audience":      "internal",
			"x-team":          "platform",
			"x-environment":   "production",
			"x-region":        "us-east-1",
			"x-owner":         "platform-team",
			"x-cost-center":   "engineering",
			"x-project":       "api-gateway",
			"x-service-tier":  "gold",
			"x-compliance":    "pci-dss",
			"x-department":    "engineering",
			"x-division":      "technology",
			"x-budget-code":   "ENG-2024-001",
			"x-manager":       "john.doe@example.com",
			"x-contact":       "api-team@example.com",
			"x-slack-channel": "#api-platform",
			"x-wiki":          "https://wiki.example.com/api",
			"x-oncall":        "api-platform-oncall",
			"x-sla":           "99.95",
			"x-support-level": "24x7",
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
		Extra: map[string]any{
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
		Extra: map[string]any{
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
