package differ

import (
	"testing"

	"github.com/erraggy/oastools/internal/severity"
	"github.com/erraggy/oastools/parser"
)

// Note on b.Fatalf usage in benchmarks:
// Using b.Fatalf for errors in benchmark setup or execution is an acceptable pattern.
// These operations (diff, parse) should never fail with valid test fixtures.
// If they do fail, it indicates a bug that should halt the benchmark immediately.

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

// BenchmarkDiffWithOptionsSmallOAS3 benchmarks DiffWithOptions convenience API
func BenchmarkDiffWithOptionsSmallOAS3(b *testing.B) {
	for b.Loop() {
		_, err := DiffWithOptions(
			WithSourceFilePath("../testdata/petstore-v1.yaml"),
			WithTargetFilePath("../testdata/petstore-v2.yaml"),
		)
		if err != nil {
			b.Fatalf("Failed to diff: %v", err)
		}
	}
}

// BenchmarkChangeString benchmarks Change.String() formatting performance
func BenchmarkChangeString(b *testing.B) {
	// Create sample changes
	changes := []Change{
		{
			Type:     ChangeTypeModified,
			Path:     "/paths/~1pet~1{petId}/get/summary",
			Message:  "Modified operation summary",
			Severity: severity.SeverityInfo,
		},
		{
			Type:     ChangeTypeAdded,
			Path:     "/paths/~1pet~1findByStatus",
			Message:  "Added new endpoint",
			Severity: severity.SeverityInfo,
		},
		{
			Type:     ChangeTypeRemoved,
			Path:     "/paths/~1user~1logout/post",
			Message:  "Removed endpoint",
			Severity: severity.SeverityError,
		},
	}

	for b.Loop() {
		for _, change := range changes {
			_ = change.String()
		}
	}
}
