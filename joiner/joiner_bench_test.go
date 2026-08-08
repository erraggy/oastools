package joiner

import (
	"fmt"
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
//   - BenchmarkJoinParsed - Pre-parses documents, benchmarks only joining logic (RECOMMENDED for CI)
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

// BenchmarkJoinParsed benchmarks joining already-parsed documents.
// This is the RECOMMENDED benchmark for detecting performance regressions
// as it eliminates filesystem variance and measures only joining logic.
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
	// FiveDocs needs five distinct documents: the same document at two positions
	// is deep copied (#481), which is not what this measures.
	doc4, err := parser.ParseWithOptions(parser.WithFilePath(joinBaseOAS3Path))
	if err != nil {
		b.Fatalf("Failed to parse doc4: %v", err)
	}
	doc5, err := parser.ParseWithOptions(parser.WithFilePath(joinExt1OAS3Path))
	if err != nil {
		b.Fatalf("Failed to parse doc5: %v", err)
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
			_, err := j.JoinParsed([]parser.ParseResult{*doc1, *doc2, *doc3, *doc4, *doc5})
			if err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	})
}

// renameBenchSchemaCount is the number of schemas each renameBenchDoc defines.
const renameBenchSchemaCount = 20

// renameBenchDoc builds an OAS 3 document that shares every schema name with the
// others this function returns, so joining them collides on all of them.
func renameBenchDoc(name string, schemaCount int) parser.ParseResult {
	schemas := make(map[string]*parser.Schema, schemaCount)
	for i := range schemaCount {
		schemas[fmt.Sprintf("Model%d", i)] = &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"id": {Type: "string"},
				// So every rename has a reference to find and rewrite.
				"next": {Ref: fmt.Sprintf("#/components/schemas/Model%d", (i+1)%schemaCount)},
				// A property named after the document, so the schemas genuinely differ.
				"from" + name: {Type: "string"},
			},
		}
	}

	return parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: name, Version: "1.0.0"},
			Paths: parser.Paths{
				"/" + name: &parser.PathItem{Get: &parser.Operation{
					Responses: &parser.Responses{Codes: map[string]*parser.Response{
						"200": {
							Description: "ok",
							Content: map[string]*parser.MediaType{
								"application/json": {Schema: &parser.Schema{Ref: "#/components/schemas/Model0"}},
							},
						},
					}},
				}},
			},
			Components: &parser.Components{Schemas: schemas},
			OASVersion: parser.OASVersion303,
		},
		Version:      "3.0.3",
		OASVersion:   parser.OASVersion303,
		SourcePath:   name,
		SourceFormat: parser.SourceFormatJSON,
	}
}

// BenchmarkJoinRenames benchmarks joins that rename. The other benchmarks use
// accept-left over documents with disjoint schema names and never rename.
//
// Fixtures are rebuilt each iteration, outside the timed section, because
// joining rewrites the references of the documents it was handed (#480).
func BenchmarkJoinRenames(b *testing.B) {
	run := func(b *testing.B, strategy CollisionStrategy, count int) {
		config := DefaultConfig()
		config.PathStrategy = StrategyAcceptLeft
		config.SchemaStrategy = strategy
		config.RenameTemplate = "{{.Name}}.{{.Source}}"
		j := New(config)
		docs := make([]parser.ParseResult, count)

		b.ReportAllocs()
		for b.Loop() {
			b.StopTimer()
			for i := range docs {
				docs[i] = renameBenchDoc(fmt.Sprintf("doc%d", i), renameBenchSchemaCount)
			}
			b.StartTimer()

			if _, err := j.JoinParsed(docs); err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	}

	b.Run("RenameRightTwoDocs", func(b *testing.B) { run(b, StrategyRenameRight, 2) })
	b.Run("RenameRightThreeDocs", func(b *testing.B) { run(b, StrategyRenameRight, 3) })
	b.Run("RenameRightFiveDocs", func(b *testing.B) { run(b, StrategyRenameRight, 5) })
	// rename-left also tracks which document contributed each merged schema.
	b.Run("RenameLeftFiveDocs", func(b *testing.B) { run(b, StrategyRenameLeft, 5) })
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
