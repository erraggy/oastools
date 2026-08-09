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
func BenchmarkJoinRenames(b *testing.B) {
	docs := make([]parser.ParseResult, 5)
	for i := range docs {
		docs[i] = renameBenchDoc(fmt.Sprintf("doc%d", i), renameBenchSchemaCount)
	}

	run := func(b *testing.B, strategy CollisionStrategy, count int) {
		config := DefaultConfig()
		config.PathStrategy = StrategyAcceptLeft
		config.SchemaStrategy = strategy
		config.RenameTemplate = "{{.Name}}.{{.Source}}"
		j := New(config)

		b.ReportAllocs()
		for b.Loop() {
			if _, err := j.JoinParsed(docs[:count]); err != nil {
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

// dedupeBenchSchemaCount is how many schemas every document in
// BenchmarkJoinDeduplicateOrRename spells identically.
const dedupeBenchSchemaCount = 5

// dedupeBenchDoc builds one of many documents that mostly agree: the same
// schemas under the same names, referenced from a path of its own. When
// diverge is set, one of those schemas carries an extra property, which is the
// one collision in the whole join that a rename has to survive.
//
// The schemas reference nothing, so the collapse settles every one of them in
// its single pass and the two configurations under test produce the same
// document. See collapseDeferredRenames for what a reference between them
// would cost.
func dedupeBenchDoc(name string, diverge bool) parser.ParseResult {
	schemas := make(map[string]*parser.Schema, dedupeBenchSchemaCount)
	for i := range dedupeBenchSchemaCount {
		properties := map[string]*parser.Schema{
			"id":   {Type: "string"},
			"name": {Type: "string"},
		}
		if diverge && i == 0 {
			properties["only"+name] = &parser.Schema{Type: "string"}
		}
		schemas[fmt.Sprintf("Shared%d", i)] = &parser.Schema{Type: "object", Properties: properties}
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
								"application/json": {Schema: &parser.Schema{Ref: "#/components/schemas/Shared0"}},
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

// BenchmarkJoinDeduplicateOrRename compares the configurations available for
// joining many documents that mostly agree (#487).
//
// RenameRightThenDedup is what the strategy replaces: it renames a shared
// schema once per document, rewrites every reference to those names, collapses
// the aliases back and rewrites again. DeduplicateOrRename decides once,
// between the point where every rename is known and the point where any of
// them is applied, so the schemas headed for a collapse are dropped rather
// than copied. Both produce the same document.
func BenchmarkJoinDeduplicateOrRename(b *testing.B) {
	docs := make([]parser.ParseResult, 100)
	for i := range docs {
		// Diverging at 9 puts the one collision a rename has to survive inside
		// both the 10 and the 100 document slice.
		docs[i] = dedupeBenchDoc(fmt.Sprintf("doc%d", i), i == 9)
	}

	run := func(b *testing.B, config JoinerConfig, count int) {
		config.PathStrategy = StrategyAcceptLeft
		config.RenameTemplate = "{{.Name}}.{{.Source}}"
		j := New(config)

		b.ReportAllocs()
		for b.Loop() {
			if _, err := j.JoinParsed(docs[:count]); err != nil {
				b.Fatalf("Failed to join: %v", err)
			}
		}
	}

	renameThenDedup := DefaultConfig()
	renameThenDedup.SchemaStrategy = StrategyRenameRight
	renameThenDedup.SemanticDeduplication = true

	decideOnce := DefaultConfig()
	decideOnce.SchemaStrategy = StrategyDeduplicateOrRename

	for _, count := range []int{10, 100} {
		b.Run(fmt.Sprintf("RenameRightThenDedup/%dDocs", count), func(b *testing.B) {
			run(b, renameThenDedup, count)
		})
		b.Run(fmt.Sprintf("DeduplicateOrRename/%dDocs", count), func(b *testing.B) {
			run(b, decideOnce, count)
		})
	}
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
