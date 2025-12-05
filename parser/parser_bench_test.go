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

// BenchmarkParseSmallOAS3 benchmarks parsing a small OAS 3.0 document (~60 lines)
func BenchmarkParseSmallOAS3(b *testing.B) {
	p := New()
	p.ResolveRefs = false
	p.ValidateStructure = true

	for b.Loop() {
		_, err := p.Parse(smallOAS3Path)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkParseMediumOAS3 benchmarks parsing a medium OAS 3.0 document (~570 lines)
func BenchmarkParseMediumOAS3(b *testing.B) {
	p := New()
	p.ResolveRefs = false
	p.ValidateStructure = true

	for b.Loop() {
		_, err := p.Parse(mediumOAS3Path)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkParseLargeOAS3 benchmarks parsing a large OAS 3.0 document (~6000 lines)
func BenchmarkParseLargeOAS3(b *testing.B) {
	p := New()
	p.ResolveRefs = false
	p.ValidateStructure = true

	for b.Loop() {
		_, err := p.Parse(largeOAS3Path)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkParseSmallOAS2 benchmarks parsing a small OAS 2.0 document (~60 lines)
func BenchmarkParseSmallOAS2(b *testing.B) {
	p := New()
	p.ResolveRefs = false
	p.ValidateStructure = true

	for b.Loop() {
		_, err := p.Parse(smallOAS2Path)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkParseMediumOAS2 benchmarks parsing a medium OAS 2.0 document (~530 lines)
func BenchmarkParseMediumOAS2(b *testing.B) {
	p := New()
	p.ResolveRefs = false
	p.ValidateStructure = true

	for b.Loop() {
		_, err := p.Parse(mediumOAS2Path)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkParseSmallOAS3NoValidation benchmarks parsing without validation
func BenchmarkParseSmallOAS3NoValidation(b *testing.B) {
	p := New()
	p.ResolveRefs = false
	p.ValidateStructure = false

	for b.Loop() {
		_, err := p.Parse(smallOAS3Path)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkParseMediumOAS3NoValidation benchmarks parsing without validation
func BenchmarkParseMediumOAS3NoValidation(b *testing.B) {
	p := New()
	p.ResolveRefs = false
	p.ValidateStructure = false

	for b.Loop() {
		_, err := p.Parse(mediumOAS3Path)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkParseBytes benchmarks parsing from byte slice
func BenchmarkParseBytesSmallOAS3(b *testing.B) {
	data, err := os.ReadFile(smallOAS3Path)
	if err != nil {
		b.Fatalf("Failed to read file: %v", err)
	}

	p := New()
	p.ResolveRefs = false
	p.ValidateStructure = true

	for b.Loop() {
		_, err := p.ParseBytes(data)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkParseBytesMediumOAS3 benchmarks parsing medium doc from bytes
func BenchmarkParseBytesMediumOAS3(b *testing.B) {
	data, err := os.ReadFile(mediumOAS3Path)
	if err != nil {
		b.Fatalf("Failed to read file: %v", err)
	}

	p := New()
	p.ResolveRefs = false
	p.ValidateStructure = true

	for b.Loop() {
		_, err := p.ParseBytes(data)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkParseWithOptionsSmallOAS3 benchmarks ParseWithOptions convenience API with file path
func BenchmarkParseWithOptionsSmallOAS3(b *testing.B) {
	for b.Loop() {
		_, err := ParseWithOptions(
			WithFilePath(smallOAS3Path),
			WithValidateStructure(true),
		)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkParseWithOptionsReaderSmallOAS3 benchmarks ParseWithOptions convenience API with Reader
func BenchmarkParseWithOptionsReaderSmallOAS3(b *testing.B) {
	data, err := os.ReadFile(smallOAS3Path)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, err := ParseWithOptions(
			WithBytes(data),
			WithValidateStructure(true),
		)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkParseWithOptionsResolveRefsSmallOAS3 benchmarks ParseWithOptions with reference resolution
func BenchmarkParseWithOptionsResolveRefsSmallOAS3(b *testing.B) {
	for b.Loop() {
		_, err := ParseWithOptions(
			WithFilePath(smallOAS3Path),
			WithResolveRefs(true),
		)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkParseReaderMediumOAS3 benchmarks ParseReader method I/O performance
func BenchmarkParseReaderMediumOAS3(b *testing.B) {
	data, err := os.ReadFile(mediumOAS3Path)
	if err != nil {
		b.Fatal(err)
	}

	p := New()
	p.ResolveRefs = false
	p.ValidateStructure = true

	for b.Loop() {
		reader := bytes.NewReader(data)
		_, err := p.ParseReader(reader)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkParseResultCopySmall benchmarks ParseResult.Copy() deep copy performance
func BenchmarkParseResultCopySmall(b *testing.B) {
	// Parse once
	parseResult, err := ParseWithOptions(
		WithFilePath(smallOAS3Path),
		WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		copy := parseResult.Copy()
		if copy == nil {
			b.Fatal("Copy returned nil")
		}
	}
}

// BenchmarkParseResolveRefsMediumOAS3 benchmarks Parse with ResolveRefs enabled
func BenchmarkParseResolveRefsMediumOAS3(b *testing.B) {
	p := New()
	p.ResolveRefs = true
	p.ValidateStructure = true

	for b.Loop() {
		_, err := p.Parse(mediumOAS3Path)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkFormatBytes benchmarks FormatBytes utility function
func BenchmarkFormatBytes(b *testing.B) {
	testCases := []int64{
		512,              // 512 B
		1024,             // 1 KB
		1024 * 1024,      // 1 MB
		1024 * 1024 * 10, // 10 MB
	}

	for b.Loop() {
		for _, size := range testCases {
			_ = FormatBytes(size)
		}
	}
}
