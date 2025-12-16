package parser

import (
	"bytes"
	"os"
	"testing"
)

// Benchmark Design Notes:
//
// File I/O Variance: Benchmarks involving file reads (BenchmarkParse, BenchmarkParseWithOptions
// with file paths) can vary significantly (+/- 50%) depending on filesystem caching, system load,
// and disk performance. These benchmarks measure end-to-end performance but are NOT reliable
// for detecting code-level performance regressions.
//
// For accurate performance comparison, use these I/O-free benchmarks:
//   - BenchmarkParseCore - Pre-loads all sizes, benchmarks core parsing (RECOMMENDED for CI)
//   - BenchmarkParseBytes - Pre-loads file data, benchmarks parsing (subset of sizes)
//   - BenchmarkDeepCopy - Benchmarks only document copying
//
// Note on b.Fatalf usage: Using b.Fatalf for errors in benchmark setup or execution is acceptable.
// These operations should never fail with valid test fixtures. If they fail, it indicates a bug.

// Benchmark fixtures
const (
	smallOAS3Path  = "../testdata/bench/small-oas3.yaml"
	mediumOAS3Path = "../testdata/bench/medium-oas3.yaml"
	largeOAS3Path  = "../testdata/bench/large-oas3.yaml"
	smallOAS2Path  = "../testdata/bench/small-oas2.yaml"
	mediumOAS2Path = "../testdata/bench/medium-oas2.yaml"
)

// BenchmarkParse benchmarks parsing OAS documents of various sizes
func BenchmarkParse(b *testing.B) {
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
			p := New()
			p.ResolveRefs = false
			p.ValidateStructure = true

			b.ReportAllocs()
			for b.Loop() {
				_, err := p.Parse(tt.path)
				if err != nil {
					b.Fatalf("Failed to parse: %v", err)
				}
			}
		})
	}
}

// BenchmarkParseNoValidation benchmarks parsing without structure validation
func BenchmarkParseNoValidation(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"SmallOAS3", smallOAS3Path},
		{"MediumOAS3", mediumOAS3Path},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			p := New()
			p.ResolveRefs = false
			p.ValidateStructure = false

			b.ReportAllocs()
			for b.Loop() {
				_, err := p.Parse(tt.path)
				if err != nil {
					b.Fatalf("Failed to parse: %v", err)
				}
			}
		})
	}
}

// BenchmarkParseBytes benchmarks parsing from byte slices
func BenchmarkParseBytes(b *testing.B) {
	tests := []struct {
		name string
		path string
	}{
		{"SmallOAS3", smallOAS3Path},
		{"MediumOAS3", mediumOAS3Path},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			data, err := os.ReadFile(tt.path)
			if err != nil {
				b.Fatalf("Failed to read file: %v", err)
			}

			p := New()
			p.ResolveRefs = false
			p.ValidateStructure = true

			b.ReportAllocs()
			for b.Loop() {
				_, err := p.ParseBytes(data)
				if err != nil {
					b.Fatalf("Failed to parse: %v", err)
				}
			}
		})
	}
}

// BenchmarkParseCore benchmarks core parsing performance without file I/O.
// This is the RECOMMENDED benchmark for detecting performance regressions
// as it eliminates filesystem variance and measures only parsing logic.
func BenchmarkParseCore(b *testing.B) {
	// Pre-load all test files in setup
	smallOAS3Data, err := os.ReadFile(smallOAS3Path)
	if err != nil {
		b.Fatalf("Failed to read small OAS3 file: %v", err)
	}
	mediumOAS3Data, err := os.ReadFile(mediumOAS3Path)
	if err != nil {
		b.Fatalf("Failed to read medium OAS3 file: %v", err)
	}
	largeOAS3Data, err := os.ReadFile(largeOAS3Path)
	if err != nil {
		b.Fatalf("Failed to read large OAS3 file: %v", err)
	}
	smallOAS2Data, err := os.ReadFile(smallOAS2Path)
	if err != nil {
		b.Fatalf("Failed to read small OAS2 file: %v", err)
	}
	mediumOAS2Data, err := os.ReadFile(mediumOAS2Path)
	if err != nil {
		b.Fatalf("Failed to read medium OAS2 file: %v", err)
	}

	tests := []struct {
		name string
		data []byte
	}{
		{"SmallOAS3", smallOAS3Data},
		{"MediumOAS3", mediumOAS3Data},
		{"LargeOAS3", largeOAS3Data},
		{"SmallOAS2", smallOAS2Data},
		{"MediumOAS2", mediumOAS2Data},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			p := New()
			p.ResolveRefs = false
			p.ValidateStructure = true

			b.ReportAllocs()
			for b.Loop() {
				_, err := p.ParseBytes(tt.data)
				if err != nil {
					b.Fatalf("Failed to parse: %v", err)
				}
			}
		})
	}
}

// BenchmarkParseWithOptions benchmarks the functional options API
func BenchmarkParseWithOptions(b *testing.B) {
	b.Run("FilePath/SmallOAS3", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := ParseWithOptions(
				WithFilePath(smallOAS3Path),
				WithValidateStructure(true),
			)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	})

	b.Run("Bytes/SmallOAS3", func(b *testing.B) {
		data, err := os.ReadFile(smallOAS3Path)
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		for b.Loop() {
			_, err := ParseWithOptions(
				WithBytes(data),
				WithValidateStructure(true),
			)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	})

	b.Run("ResolveRefs/SmallOAS3", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := ParseWithOptions(
				WithFilePath(smallOAS3Path),
				WithResolveRefs(true),
			)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	})
}

// BenchmarkParseReader benchmarks ParseReader method I/O performance
func BenchmarkParseReader(b *testing.B) {
	b.Run("MediumOAS3", func(b *testing.B) {
		data, err := os.ReadFile(mediumOAS3Path)
		if err != nil {
			b.Fatal(err)
		}

		p := New()
		p.ResolveRefs = false
		p.ValidateStructure = true

		b.ReportAllocs()
		for b.Loop() {
			reader := bytes.NewReader(data)
			_, err := p.ParseReader(reader)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	})
}

// BenchmarkParseResultCopy benchmarks ParseResult.Copy() deep copy performance
func BenchmarkParseResultCopy(b *testing.B) {
	b.Run("SmallOAS3", func(b *testing.B) {
		parseResult, err := ParseWithOptions(
			WithFilePath(smallOAS3Path),
			WithValidateStructure(true),
		)
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		for b.Loop() {
			copied := parseResult.Copy()
			if copied == nil {
				b.Fatal("Copy returned nil")
			}
		}
	})
}

// BenchmarkParseResolveRefs benchmarks Parse with reference resolution enabled
func BenchmarkParseResolveRefs(b *testing.B) {
	b.Run("MediumOAS3", func(b *testing.B) {
		p := New()
		p.ResolveRefs = true
		p.ValidateStructure = true

		b.ReportAllocs()
		for b.Loop() {
			_, err := p.Parse(mediumOAS3Path)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	})
}

// BenchmarkFormatBytes benchmarks FormatBytes utility function
func BenchmarkFormatBytes(b *testing.B) {
	testCases := []int64{
		512,              // 512 B
		1024,             // 1 KB
		1024 * 1024,      // 1 MB
		1024 * 1024 * 10, // 10 MB
	}

	b.ReportAllocs()
	for b.Loop() {
		for _, size := range testCases {
			_ = FormatBytes(size)
		}
	}
}

// BenchmarkDeepCopy benchmarks DeepCopy methods for OAS documents
func BenchmarkDeepCopy(b *testing.B) {
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
			p := New()
			p.ResolveRefs = false
			p.ValidateStructure = false

			result, err := p.Parse(tt.path)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			b.ReportAllocs()
			b.ResetTimer()

			if doc, ok := result.Document.(*OAS3Document); ok {
				for b.Loop() {
					_ = doc.DeepCopy()
				}
			} else if doc, ok := result.Document.(*OAS2Document); ok {
				for b.Loop() {
					_ = doc.DeepCopy()
				}
			}
		})
	}
}

// BenchmarkSourceMapOverhead benchmarks the overhead of source map generation.
// This demonstrates that:
// - Default parsing (no source map) has zero overhead
// - Source map generation adds measurable but acceptable overhead
func BenchmarkSourceMapOverhead(b *testing.B) {
	sizes := []struct {
		name string
		path string
	}{
		{"Small", smallOAS3Path},
		{"Medium", mediumOAS3Path},
		{"Large", largeOAS3Path},
	}

	for _, size := range sizes {
		// Baseline: Default parsing (no source map) - what users get by default
		b.Run(size.name+"/Default", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, err := ParseWithOptions(
					WithFilePath(size.path),
					WithValidateStructure(true),
					// No WithSourceMap - testing default behavior
				)
				if err != nil {
					b.Fatalf("Failed to parse: %v", err)
				}
			}
		})

		// Explicit disabled: Should be identical to default
		b.Run(size.name+"/SourceMapDisabled", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, err := ParseWithOptions(
					WithFilePath(size.path),
					WithValidateStructure(true),
					WithSourceMap(false), // Explicitly disabled
				)
				if err != nil {
					b.Fatalf("Failed to parse: %v", err)
				}
			}
		})

		// Source map enabled: Shows the overhead when feature is used
		b.Run(size.name+"/SourceMapEnabled", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, err := ParseWithOptions(
					WithFilePath(size.path),
					WithValidateStructure(true),
					WithSourceMap(true), // Enabled - expect overhead
				)
				if err != nil {
					b.Fatalf("Failed to parse: %v", err)
				}
			}
		})
	}
}
