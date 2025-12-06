package validator

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

// Note on b.Fatalf usage in benchmarks:
// Using b.Fatalf for errors in benchmark setup or execution is an acceptable pattern.
// These operations (validate, parse) should never fail with valid test fixtures.
// If they do fail, it indicates a bug that should halt the benchmark immediately.

// Benchmark fixtures
const (
	smallOAS3Path  = "../testdata/bench/small-oas3.yaml"
	mediumOAS3Path = "../testdata/bench/medium-oas3.yaml"
	largeOAS3Path  = "../testdata/bench/large-oas3.yaml"
	smallOAS2Path  = "../testdata/bench/small-oas2.yaml"
	mediumOAS2Path = "../testdata/bench/medium-oas2.yaml"
)

// BenchmarkValidate benchmarks validating OAS documents of various sizes
func BenchmarkValidate(b *testing.B) {
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
			v := New()
			v.IncludeWarnings = true
			v.StrictMode = false

			b.ReportAllocs()
			for b.Loop() {
				_, err := v.Validate(tt.path)
				if err != nil {
					b.Fatalf("Failed to validate: %v", err)
				}
			}
		})
	}
}

// BenchmarkValidateNoWarnings benchmarks validation without collecting warnings
func BenchmarkValidateNoWarnings(b *testing.B) {
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
			v := New()
			v.IncludeWarnings = false
			v.StrictMode = false

			b.ReportAllocs()
			for b.Loop() {
				_, err := v.Validate(tt.path)
				if err != nil {
					b.Fatalf("Failed to validate: %v", err)
				}
			}
		})
	}
}

// BenchmarkValidateParsed benchmarks validating already-parsed documents
func BenchmarkValidateParsed(b *testing.B) {
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

			v := New()
			v.IncludeWarnings = true
			v.StrictMode = false

			b.ReportAllocs()
			for b.Loop() {
				_, err := v.ValidateParsed(*parseResult)
				if err != nil {
					b.Fatalf("Failed to validate: %v", err)
				}
			}
		})
	}
}

// BenchmarkValidateStrictMode benchmarks strict mode validation
func BenchmarkValidateStrictMode(b *testing.B) {
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
			v := New()
			v.IncludeWarnings = true
			v.StrictMode = true

			b.ReportAllocs()
			for b.Loop() {
				_, err := v.Validate(tt.path)
				if err != nil {
					b.Fatalf("Failed to validate: %v", err)
				}
			}
		})
	}
}

// BenchmarkValidateWithOptions benchmarks the functional options API
func BenchmarkValidateWithOptions(b *testing.B) {
	b.Run("FilePath/SmallOAS3", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := ValidateWithOptions(
				WithFilePath(smallOAS3Path),
				WithIncludeWarnings(true),
			)
			if err != nil {
				b.Fatalf("Failed to validate: %v", err)
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
			_, err := ValidateWithOptions(
				WithParsed(*parseResult),
			)
			if err != nil {
				b.Fatalf("Failed to validate: %v", err)
			}
		}
	})
}
