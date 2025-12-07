package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

// Note on b.Fatalf usage in benchmarks:
// Using b.Fatalf for errors in benchmark setup or execution is an acceptable pattern.
// These operations (fix, parse) should never fail with valid test fixtures.
// If they do fail, it indicates a bug that should halt the benchmark immediately.

// Benchmark fixtures - use consistent naming with other packages
const (
	smallOAS3Path  = "../testdata/bench/small-oas3.yaml"
	mediumOAS3Path = "../testdata/bench/medium-oas3.yaml"
	largeOAS3Path  = "../testdata/bench/large-oas3.yaml"
	smallOAS2Path  = "../testdata/bench/small-oas2.yaml"
	mediumOAS2Path = "../testdata/bench/medium-oas2.yaml"
)

// BenchmarkFixDocuments benchmarks fixing OAS documents of various sizes
func BenchmarkFixDocuments(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"SmallOAS3", smallOAS3Path},
		{"MediumOAS3", mediumOAS3Path},
		{"LargeOAS3", largeOAS3Path},
		{"SmallOAS2", smallOAS2Path},
		{"MediumOAS2", mediumOAS2Path},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			f := New()
			f.InferTypes = false

			b.ReportAllocs()
			for b.Loop() {
				_, err := f.Fix(tt.path)
				if err != nil {
					b.Fatalf("Failed to fix: %v", err)
				}
			}
		})
	}
}

// BenchmarkFixWithInferTypes benchmarks fixing with type inference enabled
func BenchmarkFixWithInferTypes(b *testing.B) {
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
			f := New()
			f.InferTypes = true

			b.ReportAllocs()
			for b.Loop() {
				_, err := f.Fix(tt.path)
				if err != nil {
					b.Fatalf("Failed to fix: %v", err)
				}
			}
		})
	}
}

// BenchmarkFixParsed benchmarks fixing already-parsed documents
func BenchmarkFixParsed(b *testing.B) {
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
			parseResult, err := parser.ParseWithOptions(
				parser.WithFilePath(tt.path),
			)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			f := New()
			f.InferTypes = false

			b.ReportAllocs()
			for b.Loop() {
				_, err := f.FixParsed(*parseResult)
				if err != nil {
					b.Fatalf("Failed to fix: %v", err)
				}
			}
		})
	}
}

// BenchmarkFixWithOptions benchmarks the functional options API
func BenchmarkFixWithOptions(b *testing.B) {
	b.Run("FilePath/SmallOAS3", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := FixWithOptions(
				WithFilePath(smallOAS3Path),
				WithInferTypes(false),
			)
			if err != nil {
				b.Fatalf("Failed to fix: %v", err)
			}
		}
	})

	b.Run("Parsed/SmallOAS3", func(b *testing.B) {
		parseResult, err := parser.ParseWithOptions(
			parser.WithFilePath(smallOAS3Path),
		)
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		for b.Loop() {
			_, err := FixWithOptions(
				WithParsed(*parseResult),
			)
			if err != nil {
				b.Fatalf("Failed to fix: %v", err)
			}
		}
	})
}
