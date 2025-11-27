package differ

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

// Note on b.Fatalf usage in benchmarks:
// Using b.Fatalf for errors in benchmark setup or execution is an acceptable pattern.
// These operations (diff, parse) should never fail with valid test fixtures.
// If they do fail, it indicates a bug that should halt the benchmark immediately.

// Benchmark differ convenience functions
func BenchmarkDiffConvenience(b *testing.B) {
	for b.Loop() {
		_, err := Diff("../testdata/petstore-v1.yaml", "../testdata/petstore-v2.yaml")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDiffParsedConvenience(b *testing.B) {
	// Parse once, then benchmark diff many times
	source, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v1.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}
	target, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v2.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, err := DiffParsed(*source, *target)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark struct-based API
func BenchmarkDifferDiff(b *testing.B) {
	d := New()
	d.Mode = ModeSimple

	for b.Loop() {
		_, err := d.Diff("../testdata/petstore-v1.yaml", "../testdata/petstore-v2.yaml")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDifferDiffParsed(b *testing.B) {
	d := New()
	d.Mode = ModeSimple

	// Parse once
	source, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v1.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}
	target, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v2.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, err := d.DiffParsed(*source, *target)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark different modes
func BenchmarkDifferSimpleMode(b *testing.B) {
	d := New()
	d.Mode = ModeSimple

	source, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v1.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}
	target, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v2.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, err := d.DiffParsed(*source, *target)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDifferBreakingMode(b *testing.B) {
	d := New()
	d.Mode = ModeBreaking

	source, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v1.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}
	target, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v2.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, err := d.DiffParsed(*source, *target)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark with/without info changes
func BenchmarkDifferWithInfo(b *testing.B) {
	d := New()
	d.Mode = ModeBreaking
	d.IncludeInfo = true

	source, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v1.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}
	target, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v2.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, err := d.DiffParsed(*source, *target)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDifferWithoutInfo(b *testing.B) {
	d := New()
	d.Mode = ModeBreaking
	d.IncludeInfo = false

	source, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v1.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}
	target, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v2.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, err := d.DiffParsed(*source, *target)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark identical specs (fast path)
func BenchmarkDifferIdenticalSpecs(b *testing.B) {
	d := New()
	d.Mode = ModeSimple

	source, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v1.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, err := d.DiffParsed(*source, *source)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark parse-once pattern efficiency
func BenchmarkParseOnceDiffMany(b *testing.B) {
	// Parse once
	source, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v1.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}
	target, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v2.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}

	d := New()
	d.Mode = ModeBreaking

	for b.Loop() {
		// Diff many times without re-parsing
		_, err := d.DiffParsed(*source, *target)
		if err != nil {
			b.Fatal(err)
		}
	}
}
