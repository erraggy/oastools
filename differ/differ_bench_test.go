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

const (
	petstoreV1Path = "../testdata/petstore-v1.yaml"
	petstoreV2Path = "../testdata/petstore-v2.yaml"
)

// BenchmarkDiff benchmarks the Differ.Diff method
func BenchmarkDiff(b *testing.B) {
	b.Run("FilePath", func(b *testing.B) {
		d := New()
		d.Mode = ModeSimple

		b.ReportAllocs()
		for b.Loop() {
			_, err := d.Diff(petstoreV1Path, petstoreV2Path)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Parsed", func(b *testing.B) {
		d := New()
		d.Mode = ModeSimple

		source, err := parser.ParseWithOptions(
			parser.WithFilePath(petstoreV1Path),
			parser.WithValidateStructure(true),
		)
		if err != nil {
			b.Fatal(err)
		}
		target, err := parser.ParseWithOptions(
			parser.WithFilePath(petstoreV2Path),
			parser.WithValidateStructure(true),
		)
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		for b.Loop() {
			_, err := d.DiffParsed(*source, *target)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkDiffMode benchmarks different diff modes
func BenchmarkDiffMode(b *testing.B) {
	source, err := parser.ParseWithOptions(
		parser.WithFilePath(petstoreV1Path),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}
	target, err := parser.ParseWithOptions(
		parser.WithFilePath(petstoreV2Path),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Simple", func(b *testing.B) {
		d := New()
		d.Mode = ModeSimple

		b.ReportAllocs()
		for b.Loop() {
			_, err := d.DiffParsed(*source, *target)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Breaking", func(b *testing.B) {
		d := New()
		d.Mode = ModeBreaking

		b.ReportAllocs()
		for b.Loop() {
			_, err := d.DiffParsed(*source, *target)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkDiffIncludeInfo benchmarks with and without info changes
func BenchmarkDiffIncludeInfo(b *testing.B) {
	source, err := parser.ParseWithOptions(
		parser.WithFilePath(petstoreV1Path),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}
	target, err := parser.ParseWithOptions(
		parser.WithFilePath(petstoreV2Path),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("WithInfo", func(b *testing.B) {
		d := New()
		d.Mode = ModeBreaking
		d.IncludeInfo = true

		b.ReportAllocs()
		for b.Loop() {
			_, err := d.DiffParsed(*source, *target)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithoutInfo", func(b *testing.B) {
		d := New()
		d.Mode = ModeBreaking
		d.IncludeInfo = false

		b.ReportAllocs()
		for b.Loop() {
			_, err := d.DiffParsed(*source, *target)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkDiffIdentical benchmarks the fast path for identical specs
func BenchmarkDiffIdentical(b *testing.B) {
	source, err := parser.ParseWithOptions(
		parser.WithFilePath(petstoreV1Path),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		b.Fatal(err)
	}

	d := New()
	d.Mode = ModeSimple

	b.ReportAllocs()
	for b.Loop() {
		_, err := d.DiffParsed(*source, *source)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDiffWithOptions benchmarks the functional options API
func BenchmarkDiffWithOptions(b *testing.B) {
	b.Run("FilePath", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := DiffWithOptions(
				WithSourceFilePath(petstoreV1Path),
				WithTargetFilePath(petstoreV2Path),
			)
			if err != nil {
				b.Fatalf("Failed to diff: %v", err)
			}
		}
	})
}

// BenchmarkChangeString benchmarks Change.String() formatting performance
func BenchmarkChangeString(b *testing.B) {
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

	b.ReportAllocs()
	for b.Loop() {
		for _, change := range changes {
			_ = change.String()
		}
	}
}
