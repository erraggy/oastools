package parser

import (
	"bytes"
	"os"
	"testing"
)

// Note on b.Fatalf usage in benchmarks:
// Using b.Fatalf for errors in benchmark setup or execution is an acceptable pattern.
// These operations (parse, unmarshal, etc.) should never fail with valid test fixtures.
// If they do fail, it indicates a bug that should halt the benchmark immediately.

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
			copy := parseResult.Copy()
			if copy == nil {
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
