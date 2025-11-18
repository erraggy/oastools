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

// BenchmarkValidateSmallOAS3 benchmarks validating a small OAS 3.0 document
func BenchmarkValidateSmallOAS3(b *testing.B) {
	v := New()
	v.IncludeWarnings = true
	v.StrictMode = false

	for b.Loop() {
		_, err := v.Validate(smallOAS3Path)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkValidateMediumOAS3 benchmarks validating a medium OAS 3.0 document
func BenchmarkValidateMediumOAS3(b *testing.B) {
	v := New()
	v.IncludeWarnings = true
	v.StrictMode = false

	for b.Loop() {
		_, err := v.Validate(mediumOAS3Path)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkValidateLargeOAS3 benchmarks validating a large OAS 3.0 document
func BenchmarkValidateLargeOAS3(b *testing.B) {
	v := New()
	v.IncludeWarnings = true
	v.StrictMode = false

	for b.Loop() {
		_, err := v.Validate(largeOAS3Path)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkValidateSmallOAS2 benchmarks validating a small OAS 2.0 document
func BenchmarkValidateSmallOAS2(b *testing.B) {
	v := New()
	v.IncludeWarnings = true
	v.StrictMode = false

	for b.Loop() {
		_, err := v.Validate(smallOAS2Path)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkValidateMediumOAS2 benchmarks validating a medium OAS 2.0 document
func BenchmarkValidateMediumOAS2(b *testing.B) {
	v := New()
	v.IncludeWarnings = true
	v.StrictMode = false

	for b.Loop() {
		_, err := v.Validate(mediumOAS2Path)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkValidateSmallOAS3NoWarnings benchmarks validation without warnings
func BenchmarkValidateSmallOAS3NoWarnings(b *testing.B) {
	v := New()
	v.IncludeWarnings = false
	v.StrictMode = false

	for b.Loop() {
		_, err := v.Validate(smallOAS3Path)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkValidateMediumOAS3NoWarnings benchmarks validation without warnings
func BenchmarkValidateMediumOAS3NoWarnings(b *testing.B) {
	v := New()
	v.IncludeWarnings = false
	v.StrictMode = false

	for b.Loop() {
		_, err := v.Validate(mediumOAS3Path)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkValidateParsedSmallOAS3 benchmarks validating an already-parsed document
func BenchmarkValidateParsedSmallOAS3(b *testing.B) {
	// Parse once
	parseResult, err := parser.Parse(smallOAS3Path, false, false)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	v := New()
	v.IncludeWarnings = true
	v.StrictMode = false

	for b.Loop() {
		_, err := v.ValidateParsed(*parseResult)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkValidateParsedMediumOAS3 benchmarks validating an already-parsed medium doc
func BenchmarkValidateParsedMediumOAS3(b *testing.B) {
	// Parse once
	parseResult, err := parser.Parse(mediumOAS3Path, false, false)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	v := New()
	v.IncludeWarnings = true
	v.StrictMode = false

	for b.Loop() {
		_, err := v.ValidateParsed(*parseResult)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkValidateParsedLargeOAS3 benchmarks validating an already-parsed large doc
func BenchmarkValidateParsedLargeOAS3(b *testing.B) {
	// Parse once
	parseResult, err := parser.Parse(largeOAS3Path, false, false)
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}

	v := New()
	v.IncludeWarnings = true
	v.StrictMode = false

	for b.Loop() {
		_, err := v.ValidateParsed(*parseResult)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkConvenienceValidateSmallOAS3 benchmarks the convenience function
func BenchmarkConvenienceValidateSmallOAS3(b *testing.B) {
	for b.Loop() {
		_, err := Validate(smallOAS3Path, true, false)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkConvenienceValidateMediumOAS3 benchmarks the convenience function with medium doc
func BenchmarkConvenienceValidateMediumOAS3(b *testing.B) {
	for b.Loop() {
		_, err := Validate(mediumOAS3Path, true, false)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkValidateStrictModeSmallOAS3 benchmarks strict mode validation
func BenchmarkValidateStrictModeSmallOAS3(b *testing.B) {
	v := New()
	v.IncludeWarnings = true
	v.StrictMode = true

	for b.Loop() {
		_, err := v.Validate(smallOAS3Path)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkValidateStrictModeMediumOAS3 benchmarks strict mode validation on medium doc
func BenchmarkValidateStrictModeMediumOAS3(b *testing.B) {
	v := New()
	v.IncludeWarnings = true
	v.StrictMode = true

	for b.Loop() {
		_, err := v.Validate(mediumOAS3Path)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}
