package overlay

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

// Note on b.Fatalf usage in benchmarks:
// Using b.Fatalf for errors in benchmark setup or execution is an acceptable pattern.
// These operations (apply, parse) should never fail with valid test fixtures.
// If they do fail, it indicates a bug that should halt the benchmark immediately.

// Benchmark fixtures
const (
	petstoreBasePath    = "../testdata/overlay/fixtures/petstore-base.yaml"
	petstoreOverlayPath = "../testdata/overlay/fixtures/petstore-overlay.yaml"
	smallOAS3Path       = "../testdata/bench/small-oas3.yaml"
	mediumOAS3Path      = "../testdata/bench/medium-oas3.yaml"
	largeOAS3Path       = "../testdata/bench/large-oas3.yaml"
)

// BenchmarkApply benchmarks applying overlays to documents of various sizes
func BenchmarkApply(b *testing.B) {
	// Create a test overlay with common operations
	overlay := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Benchmark", Version: "1.0.0"},
		Actions: []Action{
			{Target: "$.info", Update: map[string]any{"x-benchmark": true}},
			{Target: "$.paths.*", Update: map[string]any{"x-tested": true}},
			{Target: "$..deprecated", Update: true},
		},
	}

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
			a := NewApplier()

			b.ReportAllocs()
			for b.Loop() {
				_, err := a.Apply(tt.path, petstoreOverlayPath)
				if err != nil {
					b.Fatalf("Failed to apply: %v", err)
				}
			}
		})
	}

	// Test with the in-memory overlay
	for _, tt := range tests {
		b.Run(tt.name+"/InMemoryOverlay", func(b *testing.B) {
			spec, err := parser.ParseWithOptions(parser.WithFilePath(tt.path))
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			a := NewApplier()

			b.ReportAllocs()
			for b.Loop() {
				_, err := a.ApplyParsed(spec, overlay)
				if err != nil {
					b.Fatalf("Failed to apply: %v", err)
				}
			}
		})
	}
}

// BenchmarkApplyParsed benchmarks applying overlays to already-parsed documents
func BenchmarkApplyParsed(b *testing.B) {
	overlay := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Benchmark", Version: "1.0.0"},
		Actions: []Action{
			{Target: "$.info", Update: map[string]any{"title": "Updated"}},
			{Target: "$.paths.*", Update: map[string]any{"x-tested": true}},
		},
	}

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
			spec, err := parser.ParseWithOptions(parser.WithFilePath(tt.path))
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			a := NewApplier()

			b.ReportAllocs()
			for b.Loop() {
				_, err := a.ApplyParsed(spec, overlay)
				if err != nil {
					b.Fatalf("Failed to apply: %v", err)
				}
			}
		})
	}
}

// BenchmarkApplyWithOptions benchmarks the functional options API
func BenchmarkApplyWithOptions(b *testing.B) {
	b.Run("FilePaths", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := ApplyWithOptions(
				WithSpecFilePath(petstoreBasePath),
				WithOverlayFilePath(petstoreOverlayPath),
			)
			if err != nil {
				b.Fatalf("Failed to apply: %v", err)
			}
		}
	})

	b.Run("Parsed", func(b *testing.B) {
		spec, err := parser.ParseWithOptions(parser.WithFilePath(petstoreBasePath))
		if err != nil {
			b.Fatal(err)
		}
		overlay, err := ParseOverlayFile(petstoreOverlayPath)
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		for b.Loop() {
			_, err := ApplyWithOptions(
				WithSpecParsed(*spec),
				WithOverlayParsed(overlay),
			)
			if err != nil {
				b.Fatalf("Failed to apply: %v", err)
			}
		}
	})
}

// BenchmarkDryRun benchmarks the dry-run preview functionality
func BenchmarkDryRun(b *testing.B) {
	overlay := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Benchmark", Version: "1.0.0"},
		Actions: []Action{
			{Target: "$.info", Update: map[string]any{"title": "Updated"}},
			{Target: "$.paths.*", Update: map[string]any{"x-tested": true}},
			{Target: "$..deprecated", Remove: true},
		},
	}

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
			spec, err := parser.ParseWithOptions(parser.WithFilePath(tt.path))
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			a := NewApplier()

			b.ReportAllocs()
			for b.Loop() {
				_, err := a.DryRun(spec, overlay)
				if err != nil {
					b.Fatalf("Failed to dry-run: %v", err)
				}
			}
		})
	}
}

// BenchmarkRecursiveDescent benchmarks recursive descent JSONPath operations
func BenchmarkRecursiveDescent(b *testing.B) {
	overlay := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Benchmark", Version: "1.0.0"},
		Actions: []Action{
			{Target: "$..description", Update: "Updated description"},
			{Target: "$..summary", Update: "Updated summary"},
		},
	}

	b.Run("LargeOAS3", func(b *testing.B) {
		spec, err := parser.ParseWithOptions(parser.WithFilePath(largeOAS3Path))
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}

		a := NewApplier()

		b.ReportAllocs()
		for b.Loop() {
			_, err := a.ApplyParsed(spec, overlay)
			if err != nil {
				b.Fatalf("Failed to apply: %v", err)
			}
		}
	})
}

// BenchmarkCompoundFilters benchmarks compound filter expressions
func BenchmarkCompoundFilters(b *testing.B) {
	overlay := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Benchmark", Version: "1.0.0"},
		Actions: []Action{
			{Target: "$.paths.*[?@.deprecated==true && @.summary!='']", Update: map[string]any{"x-filtered": true}},
		},
	}

	b.Run("LargeOAS3", func(b *testing.B) {
		spec, err := parser.ParseWithOptions(parser.WithFilePath(largeOAS3Path))
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}

		a := NewApplier()

		b.ReportAllocs()
		for b.Loop() {
			_, err := a.ApplyParsed(spec, overlay)
			if err != nil {
				b.Fatalf("Failed to apply: %v", err)
			}
		}
	})
}

// BenchmarkValidate benchmarks overlay validation
func BenchmarkValidate(b *testing.B) {
	overlay := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Benchmark", Version: "1.0.0"},
		Actions: []Action{
			{Target: "$.info", Update: map[string]any{"title": "Updated"}},
			{Target: "$.paths.*", Update: map[string]any{"x-tested": true}},
			{Target: "$.components.schemas.*", Update: map[string]any{"x-validated": true}},
		},
	}

	b.ReportAllocs()
	for b.Loop() {
		errs := Validate(overlay)
		if len(errs) > 0 {
			b.Fatalf("Validation failed: %v", errs[0])
		}
	}
}

// BenchmarkParseOverlay benchmarks overlay parsing
func BenchmarkParseOverlay(b *testing.B) {
	b.Run("FromFile", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := ParseOverlayFile(petstoreOverlayPath)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	})
}
