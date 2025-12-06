package converter

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

// Note on b.Fatalf usage in benchmarks:
// Using b.Fatalf for errors in benchmark setup or execution is an acceptable pattern.
// These operations (convert, parse) should never fail with valid test fixtures.
// If they do fail, it indicates a bug that should halt the benchmark immediately.

// Benchmark fixtures
const (
	smallOAS3Path  = "../testdata/bench/small-oas3.yaml"
	mediumOAS3Path = "../testdata/bench/medium-oas3.yaml"
	smallOAS2Path  = "../testdata/bench/small-oas2.yaml"
	mediumOAS2Path = "../testdata/bench/medium-oas2.yaml"
)

// BenchmarkConvertOAS2ToOAS3 benchmarks converting OAS 2.0 to 3.0.3
func BenchmarkConvertOAS2ToOAS3(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"Small", smallOAS2Path},
		{"Medium", mediumOAS2Path},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			c := New()
			c.StrictMode = false
			c.IncludeInfo = true

			b.ReportAllocs()
			for b.Loop() {
				_, err := c.Convert(tt.path, "3.0.3")
				if err != nil {
					b.Fatalf("Failed to convert: %v", err)
				}
			}
		})
	}
}

// BenchmarkConvertOAS3ToOAS2 benchmarks converting OAS 3.0 to 2.0
func BenchmarkConvertOAS3ToOAS2(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"Small", smallOAS3Path},
		{"Medium", mediumOAS3Path},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			c := New()
			c.StrictMode = false
			c.IncludeInfo = true

			b.ReportAllocs()
			for b.Loop() {
				_, err := c.Convert(tt.path, "2.0")
				if err != nil {
					b.Fatalf("Failed to convert: %v", err)
				}
			}
		})
	}
}

// BenchmarkConvertParsedOAS2ToOAS3 benchmarks converting already-parsed OAS 2.0 to 3.0.3
func BenchmarkConvertParsedOAS2ToOAS3(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"Small", smallOAS2Path},
		{"Medium", mediumOAS2Path},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			parseResult, err := parser.ParseWithOptions(
				parser.WithFilePath(tt.path),
			)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			c := New()
			c.StrictMode = false
			c.IncludeInfo = true

			b.ReportAllocs()
			for b.Loop() {
				_, err := c.ConvertParsed(*parseResult, "3.0.3")
				if err != nil {
					b.Fatalf("Failed to convert: %v", err)
				}
			}
		})
	}
}

// BenchmarkConvertParsedOAS3ToOAS2 benchmarks converting already-parsed OAS 3.0 to 2.0
func BenchmarkConvertParsedOAS3ToOAS2(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"Small", smallOAS3Path},
		{"Medium", mediumOAS3Path},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			parseResult, err := parser.ParseWithOptions(
				parser.WithFilePath(tt.path),
			)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			c := New()
			c.StrictMode = false
			c.IncludeInfo = true

			b.ReportAllocs()
			for b.Loop() {
				_, err := c.ConvertParsed(*parseResult, "2.0")
				if err != nil {
					b.Fatalf("Failed to convert: %v", err)
				}
			}
		})
	}
}

// BenchmarkConvertNoInfo benchmarks conversion without info messages
func BenchmarkConvertNoInfo(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"Small", smallOAS2Path},
		{"Medium", mediumOAS2Path},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			c := New()
			c.StrictMode = false
			c.IncludeInfo = false

			b.ReportAllocs()
			for b.Loop() {
				_, err := c.Convert(tt.path, "3.0.3")
				if err != nil {
					b.Fatalf("Failed to convert: %v", err)
				}
			}
		})
	}
}

// BenchmarkConvertWithOptions benchmarks the functional options API
func BenchmarkConvertWithOptions(b *testing.B) {
	b.Run("FilePath/Small", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := ConvertWithOptions(
				WithFilePath(smallOAS2Path),
				WithTargetVersion("3.0.3"),
			)
			if err != nil {
				b.Fatalf("Failed to convert: %v", err)
			}
		}
	})

	b.Run("Parsed/Small", func(b *testing.B) {
		parseResult, err := parser.ParseWithOptions(
			parser.WithFilePath(smallOAS2Path),
		)
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		for b.Loop() {
			_, err := ConvertWithOptions(
				WithParsed(*parseResult),
				WithTargetVersion("3.0.3"),
			)
			if err != nil {
				b.Fatalf("Failed to convert: %v", err)
			}
		}
	})
}

// BenchmarkConvertOAS30ToOAS31 benchmarks OAS 3.0 to OAS 3.1 version update
func BenchmarkConvertOAS30ToOAS31(b *testing.B) {
	b.Run("Small", func(b *testing.B) {
		c := New()
		c.StrictMode = false
		c.IncludeInfo = true

		b.ReportAllocs()
		for b.Loop() {
			_, err := c.Convert(smallOAS3Path, "3.1.0")
			if err != nil {
				b.Fatalf("Failed to convert: %v", err)
			}
		}
	})
}
