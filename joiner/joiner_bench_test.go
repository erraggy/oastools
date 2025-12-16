package joiner

import (
	"os"
	"testing"

	"github.com/erraggy/oastools/parser"
)

// Benchmark Design Notes:
//
// File I/O Variance: Benchmarks involving file reads (BenchmarkJoin, BenchmarkJoinStrategy,
// BenchmarkJoinOptions, BenchmarkJoinWithOptions/FilePaths) can vary significantly (+/- 50%)
// depending on filesystem caching, system load, and disk performance. These benchmarks measure
// end-to-end performance but are NOT reliable for detecting code-level performance regressions.
//
// For accurate performance comparison, use these I/O-free benchmarks:
//   - BenchmarkJoinParsed - Pre-parses documents, benchmarks only joining logic (RECOMMENDED)
//   - BenchmarkJoinCore - Pre-loads all files, benchmarks core joining (recommended for CI)
//   - BenchmarkJoinWithOptions/Parsed - Functional options API without I/O
//   - BenchmarkJoinHelpers - Benchmarks helper functions only
//
// Note on b.Fatalf usage: Using b.Fatalf for errors in benchmark setup or execution is acceptable.
// These operations should never fail with valid test fixtures. If they fail, it indicates a bug.

// Benchmark fixtures
const (
	joinBaseOAS3Path = "../testdata/bench/join-base-oas3.yaml"
	joinExt1OAS3Path = "../testdata/bench/join-ext1-oas3.yaml"
	joinExt2OAS3Path = "../testdata/bench/join-ext2-oas3.yaml"
)

// BenchmarkJoin benchmarks joining documents from file paths
func BenchmarkJoin(b *testing.B) {
	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft
	j := New(config)

	b.Run("TwoDocs", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := j.Join([]string{joinBaseOAS3Path, joinExt1OAS3Path})
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})

	b.Run("ThreeDocs", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := j.Join([]string{joinBaseOAS3Path, joinExt1OAS3Path, joinExt2OAS3Path})
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})

	b.Run("FiveDocs", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := j.Join([]string{
				joinBaseOAS3Path,
				joinExt1OAS3Path,
				joinExt2OAS3Path,
				joinBaseOAS3Path,
				joinExt1OAS3Path,
			})
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})
}

// BenchmarkJoinParsed benchmarks joining already-parsed documents
func BenchmarkJoinParsed(b *testing.B) {
	doc1, err := parser.ParseWithOptions(parser.WithFilePath(joinBaseOAS3Path))
	if err != nil {
		b.Fatalf("Failed to parse doc1: %v", err)
	}
	doc2, err := parser.ParseWithOptions(parser.WithFilePath(joinExt1OAS3Path))
	if err != nil {
		b.Fatalf("Failed to parse doc2: %v", err)
	}
	doc3, err := parser.ParseWithOptions(parser.WithFilePath(joinExt2OAS3Path))
	if err != nil {
		b.Fatalf("Failed to parse doc3: %v", err)
	}

	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft
	j := New(config)

	b.Run("TwoDocs", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := j.JoinParsed([]parser.ParseResult{*doc1, *doc2})
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})

	b.Run("ThreeDocs", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := j.JoinParsed([]parser.ParseResult{*doc1, *doc2, *doc3})
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})
}

// BenchmarkJoinCore benchmarks core joining performance without file I/O.
// This is the RECOMMENDED benchmark for detecting performance regressions
// as it eliminates filesystem variance and measures only joining logic.
func BenchmarkJoinCore(b *testing.B) {
	// Pre-parse all documents in setup
	doc1, err := parser.ParseWithOptions(parser.WithFilePath(joinBaseOAS3Path))
	if err != nil {
		b.Fatalf("Failed to parse doc1: %v", err)
	}
	doc2, err := parser.ParseWithOptions(parser.WithFilePath(joinExt1OAS3Path))
	if err != nil {
		b.Fatalf("Failed to parse doc2: %v", err)
	}
	doc3, err := parser.ParseWithOptions(parser.WithFilePath(joinExt2OAS3Path))
	if err != nil {
		b.Fatalf("Failed to parse doc3: %v", err)
	}

	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft
	j := New(config)

	b.Run("TwoDocs", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := j.JoinParsed([]parser.ParseResult{*doc1, *doc2})
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})

	b.Run("ThreeDocs", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := j.JoinParsed([]parser.ParseResult{*doc1, *doc2, *doc3})
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})

	b.Run("FiveDocs", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := j.JoinParsed([]parser.ParseResult{*doc1, *doc2, *doc3, *doc1, *doc2})
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})
}

// BenchmarkJoinStrategy benchmarks different merge strategies
func BenchmarkJoinStrategy(b *testing.B) {
	b.Run("AcceptLeft", func(b *testing.B) {
		config := DefaultConfig()
		config.PathStrategy = StrategyAcceptLeft
		config.SchemaStrategy = StrategyAcceptLeft
		j := New(config)

		b.ReportAllocs()
		for b.Loop() {
			_, err := j.Join([]string{joinBaseOAS3Path, joinExt1OAS3Path})
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})

	b.Run("AcceptRight", func(b *testing.B) {
		config := DefaultConfig()
		config.PathStrategy = StrategyAcceptRight
		config.SchemaStrategy = StrategyAcceptRight
		j := New(config)

		b.ReportAllocs()
		for b.Loop() {
			_, err := j.Join([]string{joinBaseOAS3Path, joinExt1OAS3Path})
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})
}

// BenchmarkJoinOptions benchmarks join with various options
func BenchmarkJoinOptions(b *testing.B) {
	b.Run("MergeArrays", func(b *testing.B) {
		config := DefaultConfig()
		config.PathStrategy = StrategyAcceptLeft
		config.SchemaStrategy = StrategyAcceptLeft
		config.MergeArrays = true
		j := New(config)

		b.ReportAllocs()
		for b.Loop() {
			_, err := j.Join([]string{joinBaseOAS3Path, joinExt1OAS3Path})
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})

	b.Run("DeduplicateTags", func(b *testing.B) {
		config := DefaultConfig()
		config.PathStrategy = StrategyAcceptLeft
		config.SchemaStrategy = StrategyAcceptLeft
		config.DeduplicateTags = true
		j := New(config)

		b.ReportAllocs()
		for b.Loop() {
			_, err := j.Join([]string{joinBaseOAS3Path, joinExt1OAS3Path})
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})
}

// BenchmarkJoinWithOptions benchmarks the functional options API
func BenchmarkJoinWithOptions(b *testing.B) {
	config := DefaultConfig()

	b.Run("FilePaths", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := JoinWithOptions(
				WithFilePaths(joinBaseOAS3Path, joinExt1OAS3Path),
				WithConfig(config),
			)
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})

	b.Run("Parsed", func(b *testing.B) {
		doc1, err := parser.ParseWithOptions(parser.WithFilePath(joinBaseOAS3Path))
		if err != nil {
			b.Fatal(err)
		}
		doc2, err := parser.ParseWithOptions(parser.WithFilePath(joinExt1OAS3Path))
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		for b.Loop() {
			_, err := JoinWithOptions(
				WithParsed(*doc1, *doc2),
				WithConfig(config),
			)
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})
}

// BenchmarkJoinWriteResult benchmarks WriteResult I/O performance
func BenchmarkJoinWriteResult(b *testing.B) {
	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft
	j := New(config)

	result, err := j.Join([]string{joinBaseOAS3Path, joinExt1OAS3Path})
	if err != nil {
		b.Fatal(err)
	}

	tmpfile, err := os.CreateTemp("", "bench-join-*.yaml")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	if err := tmpfile.Close(); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		err := j.WriteResult(result, tmpfile.Name())
		if err != nil {
			b.Fatalf("Failed to write: %v", err)
		}
	}
}

// BenchmarkJoinHelpers benchmarks helper functions
func BenchmarkJoinHelpers(b *testing.B) {
	b.Run("DefaultConfig", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = DefaultConfig()
		}
	})

	b.Run("IsValidStrategy", func(b *testing.B) {
		strategies := []string{
			string(StrategyAcceptLeft),
			string(StrategyAcceptRight),
			string(StrategyFailOnCollision),
			"invalid-strategy",
		}

		b.ReportAllocs()
		for b.Loop() {
			for _, strategy := range strategies {
				_ = IsValidStrategy(strategy)
			}
		}
	})

	b.Run("ValidStrategies", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = ValidStrategies()
		}
	})
}
