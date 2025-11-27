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

// BenchmarkConvertOAS2ToOAS3Small benchmarks converting small OAS 2.0 to 3.0.3
func BenchmarkConvertOAS2ToOAS3Small(b *testing.B) {
	c := New()
	c.StrictMode = false
	c.IncludeInfo = true

	for b.Loop() {
		_, err := c.Convert(smallOAS2Path, "3.0.3")
		if err != nil {
			b.Fatalf("Failed to convert: %v", err)
		}
	}
}

// BenchmarkConvertOAS2ToOAS3Medium benchmarks converting medium OAS 2.0 to 3.0.3
func BenchmarkConvertOAS2ToOAS3Medium(b *testing.B) {
	c := New()
	c.StrictMode = false
	c.IncludeInfo = true

	for b.Loop() {
		_, err := c.Convert(mediumOAS2Path, "3.0.3")
		if err != nil {
			b.Fatalf("Failed to convert: %v", err)
		}
	}
}

// BenchmarkConvertOAS3ToOAS2Small benchmarks converting small OAS 3.0 to 2.0
func BenchmarkConvertOAS3ToOAS2Small(b *testing.B) {
	c := New()
	c.StrictMode = false
	c.IncludeInfo = true

	for b.Loop() {
		_, err := c.Convert(smallOAS3Path, "2.0")
		if err != nil {
			b.Fatalf("Failed to convert: %v", err)
		}
	}
}

// BenchmarkConvertOAS3ToOAS2Medium benchmarks converting medium OAS 3.0 to 2.0
func BenchmarkConvertOAS3ToOAS2Medium(b *testing.B) {
	c := New()
	c.StrictMode = false
	c.IncludeInfo = true

	for b.Loop() {
		_, err := c.Convert(mediumOAS3Path, "2.0")
		if err != nil {
			b.Fatalf("Failed to convert: %v", err)
		}
	}
}

// BenchmarkConvertParsedOAS2ToOAS3Small benchmarks converting already-parsed OAS 2.0 to 3.0.3
func BenchmarkConvertParsedOAS2ToOAS3Small(b *testing.B) {
	// Parse once
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath(smallOAS2Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	c := New()
	c.StrictMode = false
	c.IncludeInfo = true

	for b.Loop() {
		_, err := c.ConvertParsed(*parseResult, "3.0.3")
		if err != nil {
			b.Fatalf("Failed to convert: %v", err)
		}
	}
}

// BenchmarkConvertParsedOAS2ToOAS3Medium benchmarks converting already-parsed medium OAS 2.0
func BenchmarkConvertParsedOAS2ToOAS3Medium(b *testing.B) {
	// Parse once
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath(mediumOAS2Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	c := New()
	c.StrictMode = false
	c.IncludeInfo = true

	for b.Loop() {
		_, err := c.ConvertParsed(*parseResult, "3.0.3")
		if err != nil {
			b.Fatalf("Failed to convert: %v", err)
		}
	}
}

// BenchmarkConvertParsedOAS3ToOAS2Small benchmarks converting already-parsed OAS 3.0 to 2.0
func BenchmarkConvertParsedOAS3ToOAS2Small(b *testing.B) {
	// Parse once
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath(smallOAS3Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	c := New()
	c.StrictMode = false
	c.IncludeInfo = true

	for b.Loop() {
		_, err := c.ConvertParsed(*parseResult, "2.0")
		if err != nil {
			b.Fatalf("Failed to convert: %v", err)
		}
	}
}

// BenchmarkConvertParsedOAS3ToOAS2Medium benchmarks converting already-parsed medium OAS 3.0
func BenchmarkConvertParsedOAS3ToOAS2Medium(b *testing.B) {
	// Parse once
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath(mediumOAS3Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	c := New()
	c.StrictMode = false
	c.IncludeInfo = true

	for b.Loop() {
		_, err := c.ConvertParsed(*parseResult, "2.0")
		if err != nil {
			b.Fatalf("Failed to convert: %v", err)
		}
	}
}

// BenchmarkConvenienceConvertOAS2ToOAS3Small benchmarks the convenience function
func BenchmarkConvenienceConvertOAS2ToOAS3Small(b *testing.B) {
	for b.Loop() {
		_, err := Convert(smallOAS2Path, "3.0.3")
		if err != nil {
			b.Fatalf("Failed to convert: %v", err)
		}
	}
}

// BenchmarkConvenienceConvertOAS2ToOAS3Medium benchmarks the convenience function with medium doc
func BenchmarkConvenienceConvertOAS2ToOAS3Medium(b *testing.B) {
	for b.Loop() {
		_, err := Convert(mediumOAS2Path, "3.0.3")
		if err != nil {
			b.Fatalf("Failed to convert: %v", err)
		}
	}
}

// BenchmarkConvertNoInfoOAS2ToOAS3Small benchmarks conversion without info messages
func BenchmarkConvertNoInfoOAS2ToOAS3Small(b *testing.B) {
	c := New()
	c.StrictMode = false
	c.IncludeInfo = false

	for b.Loop() {
		_, err := c.Convert(smallOAS2Path, "3.0.3")
		if err != nil {
			b.Fatalf("Failed to convert: %v", err)
		}
	}
}

// BenchmarkConvertNoInfoOAS2ToOAS3Medium benchmarks conversion without info on medium doc
func BenchmarkConvertNoInfoOAS2ToOAS3Medium(b *testing.B) {
	c := New()
	c.StrictMode = false
	c.IncludeInfo = false

	for b.Loop() {
		_, err := c.Convert(mediumOAS2Path, "3.0.3")
		if err != nil {
			b.Fatalf("Failed to convert: %v", err)
		}
	}
}
