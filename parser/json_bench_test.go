package parser

import (
	"encoding/json"
	"testing"
)

// Note on b.Fatalf usage in benchmarks:
// Using b.Fatalf for errors in benchmark setup or execution is an acceptable pattern.
// These operations (marshal, unmarshal, parse) should never fail with valid test fixtures.
// If they do fail, it indicates a bug that should halt the benchmark immediately.

// BenchmarkMarshalInfo benchmarks marshaling Info with various Extra field counts
func BenchmarkMarshalInfo(b *testing.B) {
	tests := []struct {
		name  string
		extra map[string]any
	}{
		{"NoExtra", nil},
		{"Extra1", map[string]any{
			"x-api-id": "test-001",
		}},
		{"Extra5", map[string]any{
			"x-api-id":      "test-001",
			"x-audience":    "internal",
			"x-team":        "platform",
			"x-environment": "production",
			"x-region":      "us-east-1",
		}},
		{"Extra10", map[string]any{
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
		}},
		{"Extra20", map[string]any{
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
		}},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			info := &Info{
				Title:       "Test API",
				Version:     "1.0.0",
				Description: "A test API for benchmarking",
				Extra:       tt.extra,
			}

			b.ReportAllocs()
			for b.Loop() {
				_, err := json.Marshal(info)
				if err != nil {
					b.Fatalf("Failed to marshal: %v", err)
				}
			}
		})
	}
}

// BenchmarkMarshalContact benchmarks marshaling Contact with and without Extra fields
func BenchmarkMarshalContact(b *testing.B) {
	tests := []struct {
		name  string
		extra map[string]any
	}{
		{"NoExtra", nil},
		{"WithExtra", map[string]any{
			"x-team-id":   "platform-001",
			"x-slack":     "#api-support",
			"x-on-call":   true,
			"x-timezone":  "UTC",
			"x-languages": []string{"en", "es", "fr"},
		}},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			contact := &Contact{
				Name:  "API Support",
				Email: "support@example.com",
				URL:   "https://example.com/support",
				Extra: tt.extra,
			}

			b.ReportAllocs()
			for b.Loop() {
				_, err := json.Marshal(contact)
				if err != nil {
					b.Fatalf("Failed to marshal: %v", err)
				}
			}
		})
	}
}

// BenchmarkMarshalServer benchmarks marshaling Server with and without Extra fields
func BenchmarkMarshalServer(b *testing.B) {
	tests := []struct {
		name  string
		extra map[string]any
	}{
		{"NoExtra", nil},
		{"WithExtra", map[string]any{
			"x-environment":    "production",
			"x-region":         "us-east-1",
			"x-load-balancer":  "alb-prod-001",
			"x-health-check":   "/health",
			"x-rate-limit":     10000,
			"x-cache-enabled":  true,
			"x-timeout-ms":     5000,
			"x-retry-attempts": 3,
		}},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			server := &Server{
				URL:         "https://api.example.com/v1",
				Description: "Production server",
				Extra:       tt.extra,
			}

			b.ReportAllocs()
			for b.Loop() {
				_, err := json.Marshal(server)
				if err != nil {
					b.Fatalf("Failed to marshal: %v", err)
				}
			}
		})
	}
}

// BenchmarkMarshalOAS3Document benchmarks marshaling OAS3 documents of various sizes
func BenchmarkMarshalOAS3Document(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"Small", smallOAS3Path},
		{"Medium", mediumOAS3Path},
		{"Large", largeOAS3Path},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			result, err := ParseWithOptions(WithFilePath(tt.path))
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			doc, ok := result.Document.(*OAS3Document)
			if !ok {
				b.Fatalf("Expected *OAS3Document, got %T", result.Document)
			}

			b.ReportAllocs()
			for b.Loop() {
				_, err := json.Marshal(doc)
				if err != nil {
					b.Fatalf("Failed to marshal: %v", err)
				}
			}
		})
	}
}

// BenchmarkMarshalOAS2Document benchmarks marshaling OAS2 documents of various sizes
func BenchmarkMarshalOAS2Document(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"Small", smallOAS2Path},
		{"Medium", mediumOAS2Path},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			result, err := ParseWithOptions(WithFilePath(tt.path))
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			doc, ok := result.Document.(*OAS2Document)
			if !ok {
				b.Fatalf("Expected *OAS2Document, got %T", result.Document)
			}

			b.ReportAllocs()
			for b.Loop() {
				_, err := json.Marshal(doc)
				if err != nil {
					b.Fatalf("Failed to marshal: %v", err)
				}
			}
		})
	}
}

// BenchmarkUnmarshalInfo benchmarks unmarshaling Info with and without Extra fields
func BenchmarkUnmarshalInfo(b *testing.B) {
	tests := []struct {
		name string
		data []byte
	}{
		{"NoExtra", []byte(`{"title":"Test API","version":"1.0.0","description":"A test API"}`)},
		{"WithExtra", []byte(`{
			"title":"Test API",
			"version":"1.0.0",
			"description":"A test API",
			"x-api-id":"test-001",
			"x-audience":"internal",
			"x-team":"platform",
			"x-environment":"production"
		}`)},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var info Info
				err := json.Unmarshal(tt.data, &info)
				if err != nil {
					b.Fatalf("Failed to unmarshal: %v", err)
				}
			}
		})
	}
}
