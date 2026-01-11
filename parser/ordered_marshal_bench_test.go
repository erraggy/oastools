package parser

import (
	"os"
	"testing"
)

// Benchmark Design Notes:
//
// These benchmarks measure the performance of order-preserving marshaling.
// All benchmarks pre-parse documents OUTSIDE the benchmark loop to measure
// only the marshaling operation, not parsing overhead.
//
// Note on b.Fatalf usage: Using b.Fatalf for errors in benchmark setup or execution is acceptable.
// These operations should never fail with valid test fixtures. If they fail, it indicates a bug.

// BenchmarkMarshalOrderedJSON benchmarks order-preserving JSON marshaling
// for various document sizes.
func BenchmarkMarshalOrderedJSON(b *testing.B) {
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
			// Pre-parse document with order preservation
			result, err := ParseWithOptions(
				WithFilePath(tt.path),
				WithPreserveOrder(true),
			)
			if err != nil {
				b.Fatalf("Failed to parse %s: %v", tt.path, err)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, err := result.MarshalOrderedJSON()
				if err != nil {
					b.Fatalf("Failed to marshal: %v", err)
				}
			}
		})
	}
}

// BenchmarkMarshalOrderedYAML benchmarks order-preserving YAML marshaling
// for various document sizes.
func BenchmarkMarshalOrderedYAML(b *testing.B) {
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
			// Pre-parse document with order preservation
			result, err := ParseWithOptions(
				WithFilePath(tt.path),
				WithPreserveOrder(true),
			)
			if err != nil {
				b.Fatalf("Failed to parse %s: %v", tt.path, err)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, err := result.MarshalOrderedYAML()
				if err != nil {
					b.Fatalf("Failed to marshal: %v", err)
				}
			}
		})
	}
}

// BenchmarkMarshalOrderedJSONIndent benchmarks indented JSON marshaling
// with order preservation.
func BenchmarkMarshalOrderedJSONIndent(b *testing.B) {
	// Pre-parse medium document with order preservation
	result, err := ParseWithOptions(
		WithFilePath(mediumOAS3Path),
		WithPreserveOrder(true),
	)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := result.MarshalOrderedJSONIndent("", "  ")
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

// BenchmarkMarshalOrderedJSON_vs_Standard compares order-preserving
// marshaling against standard JSON marshaling to measure the overhead
// of preserving field order.
func BenchmarkMarshalOrderedJSON_vs_Standard(b *testing.B) {
	// Pre-load data for both parsing scenarios
	data, err := os.ReadFile(mediumOAS3Path)
	if err != nil {
		b.Fatalf("Failed to read file: %v", err)
	}

	// Pre-parse with order preservation enabled
	orderedResult, err := ParseWithOptions(
		WithBytes(data),
		WithPreserveOrder(true),
	)
	if err != nil {
		b.Fatalf("Failed to parse with order: %v", err)
	}

	// Pre-parse without order preservation (fallback to standard marshal)
	standardResult, err := ParseWithOptions(
		WithBytes(data),
		WithPreserveOrder(false),
	)
	if err != nil {
		b.Fatalf("Failed to parse without order: %v", err)
	}

	b.Run("OrderPreserving", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := orderedResult.MarshalOrderedJSON()
			if err != nil {
				b.Fatalf("Failed to marshal: %v", err)
			}
		}
	})

	b.Run("Standard", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			// When PreserveOrder is false, MarshalOrderedJSON falls back to standard
			_, err := standardResult.MarshalOrderedJSON()
			if err != nil {
				b.Fatalf("Failed to marshal: %v", err)
			}
		}
	})
}

// BenchmarkPreserveOrderOverhead measures the parsing overhead of
// enabling PreserveOrder during parsing.
func BenchmarkPreserveOrderOverhead(b *testing.B) {
	// Pre-load file data to eliminate I/O variance
	data, err := os.ReadFile(mediumOAS3Path)
	if err != nil {
		b.Fatalf("Failed to read file: %v", err)
	}

	b.Run("WithPreserveOrder", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := ParseWithOptions(
				WithBytes(data),
				WithPreserveOrder(true),
			)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	})

	b.Run("WithoutPreserveOrder", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := ParseWithOptions(
				WithBytes(data),
				WithPreserveOrder(false),
			)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	})
}
