package joiner

import (
	"os"
	"testing"

	"github.com/erraggy/oastools/parser"
)

// Note on b.Fatalf usage in benchmarks:
// Using b.Fatalf for errors in benchmark setup or execution is an acceptable pattern.
// These operations (join, parse) should never fail with valid test fixtures.
// If they do fail, it indicates a bug that should halt the benchmark immediately.

// Benchmark fixtures
const (
	joinBaseOAS3Path = "../testdata/bench/join-base-oas3.yaml"
	joinExt1OAS3Path = "../testdata/bench/join-ext1-oas3.yaml"
	joinExt2OAS3Path = "../testdata/bench/join-ext2-oas3.yaml"
	mediumOAS3Path   = "../testdata/bench/medium-oas3.yaml"
)

// BenchmarkJoinTwoSmallDocs benchmarks joining 2 small documents
func BenchmarkJoinTwoSmallDocs(b *testing.B) {
	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft

	j := New(config)

	for b.Loop() {
		_, err := j.Join([]string{joinBaseOAS3Path, joinExt1OAS3Path})
		if err != nil {
			b.Fatalf("Failed to join: %v", err)
		}
	}
}

// BenchmarkJoinThreeSmallDocs benchmarks joining 3 small documents
func BenchmarkJoinThreeSmallDocs(b *testing.B) {
	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft

	j := New(config)

	for b.Loop() {
		_, err := j.Join([]string{joinBaseOAS3Path, joinExt1OAS3Path, joinExt2OAS3Path})
		if err != nil {
			b.Fatalf("Failed to join: %v", err)
		}
	}
}

// BenchmarkJoinParsedTwoSmallDocs benchmarks joining already-parsed documents
func BenchmarkJoinParsedTwoSmallDocs(b *testing.B) {
	// Parse once
	doc1, err := parser.ParseWithOptions(
		parser.WithFilePath(joinBaseOAS3Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse doc1: %v", err)
	}
	doc2, err := parser.ParseWithOptions(
		parser.WithFilePath(joinExt1OAS3Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse doc2: %v", err)
	}

	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft

	j := New(config)

	for b.Loop() {
		_, err := j.JoinParsed([]parser.ParseResult{*doc1, *doc2})
		if err != nil {
			b.Fatalf("Failed to join: %v", err)
		}
	}
}

// BenchmarkJoinParsedThreeSmallDocs benchmarks joining 3 already-parsed documents
func BenchmarkJoinParsedThreeSmallDocs(b *testing.B) {
	// Parse once
	doc1, err := parser.ParseWithOptions(
		parser.WithFilePath(joinBaseOAS3Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse doc1: %v", err)
	}
	doc2, err := parser.ParseWithOptions(
		parser.WithFilePath(joinExt1OAS3Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse doc2: %v", err)
	}
	doc3, err := parser.ParseWithOptions(
		parser.WithFilePath(joinExt2OAS3Path),
	)
	if err != nil {
		b.Fatalf("Failed to parse doc3: %v", err)
	}

	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft

	j := New(config)

	for b.Loop() {
		_, err := j.JoinParsed([]parser.ParseResult{*doc1, *doc2, *doc3})
		if err != nil {
			b.Fatalf("Failed to join: %v", err)
		}
	}
}

// BenchmarkJoinStrategyAcceptRight benchmarks with StrategyAcceptRight
func BenchmarkJoinStrategyAcceptRight(b *testing.B) {
	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptRight
	config.SchemaStrategy = StrategyAcceptRight

	j := New(config)

	for b.Loop() {
		_, err := j.Join([]string{joinBaseOAS3Path, joinExt1OAS3Path})
		if err != nil {
			b.Fatalf("Failed to join: %v", err)
		}
	}
}

// BenchmarkJoinWithArrayMerge benchmarks joining with array merging enabled
func BenchmarkJoinWithArrayMerge(b *testing.B) {
	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft
	config.MergeArrays = true

	j := New(config)

	for b.Loop() {
		_, err := j.Join([]string{joinBaseOAS3Path, joinExt1OAS3Path})
		if err != nil {
			b.Fatalf("Failed to join: %v", err)
		}
	}
}

// BenchmarkJoinWithDeduplicateTags benchmarks joining with tag deduplication
func BenchmarkJoinWithDeduplicateTags(b *testing.B) {
	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft
	config.DeduplicateTags = true

	j := New(config)

	for b.Loop() {
		_, err := j.Join([]string{joinBaseOAS3Path, joinExt1OAS3Path})
		if err != nil {
			b.Fatalf("Failed to join: %v", err)
		}
	}
}

// BenchmarkJoinWithOptionsTwoSmallDocs benchmarks JoinWithOptions convenience API
func BenchmarkJoinWithOptionsTwoSmallDocs(b *testing.B) {
	config := DefaultConfig()

	for b.Loop() {
		_, err := JoinWithOptions(
			WithFilePaths(joinBaseOAS3Path, joinExt1OAS3Path),
			WithConfig(config),
		)
		if err != nil {
			b.Fatalf("Failed to join: %v", err)
		}
	}
}

// BenchmarkJoinWithOptionsParsedTwoSmallDocs benchmarks JoinWithOptions with pre-parsed documents
func BenchmarkJoinWithOptionsParsedTwoSmallDocs(b *testing.B) {
	// Parse once
	doc1, err := parser.ParseWithOptions(
		parser.WithFilePath(joinBaseOAS3Path),
	)
	if err != nil {
		b.Fatal(err)
	}
	doc2, err := parser.ParseWithOptions(
		parser.WithFilePath(joinExt1OAS3Path),
	)
	if err != nil {
		b.Fatal(err)
	}

	config := DefaultConfig()

	for b.Loop() {
		_, err := JoinWithOptions(
			WithParsed(*doc1, *doc2),
			WithConfig(config),
		)
		if err != nil {
			b.Fatalf("Failed to join: %v", err)
		}
	}
}

// BenchmarkJoinWriteResult benchmarks WriteResult I/O performance
func BenchmarkJoinWriteResult(b *testing.B) {
	// Join once
	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft

	j := New(config)
	result, err := j.Join([]string{joinBaseOAS3Path, joinExt1OAS3Path})
	if err != nil {
		b.Fatal(err)
	}

	// Use temp file for writing
	tmpfile, err := os.CreateTemp("", "bench-join-*.yaml")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()
	if err := tmpfile.Close(); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		err := j.WriteResult(result, tmpfile.Name())
		if err != nil {
			b.Fatalf("Failed to write: %v", err)
		}
	}
}

// BenchmarkJoinFiveSmallDocs benchmarks joining 5 documents
func BenchmarkJoinFiveSmallDocs(b *testing.B) {
	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft

	j := New(config)

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
}

// BenchmarkDefaultConfig benchmarks DefaultConfig construction
func BenchmarkDefaultConfig(b *testing.B) {
	for b.Loop() {
		_ = DefaultConfig()
	}
}

// BenchmarkIsValidStrategy benchmarks IsValidStrategy validation
func BenchmarkIsValidStrategy(b *testing.B) {
	strategies := []string{
		string(StrategyAcceptLeft),
		string(StrategyAcceptRight),
		string(StrategyFailOnCollision),
		"invalid-strategy",
	}

	for b.Loop() {
		for _, strategy := range strategies {
			_ = IsValidStrategy(strategy)
		}
	}
}

// BenchmarkValidStrategies benchmarks ValidStrategies list generation
func BenchmarkValidStrategies(b *testing.B) {
	for b.Loop() {
		_ = ValidStrategies()
	}
}
