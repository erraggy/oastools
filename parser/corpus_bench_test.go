//go:build corpus

// Corpus benchmarks require the corpus build tag to run.
// Run with: go test -tags=corpus -bench=BenchmarkCorpus ./parser/...
// Or use: make bench-corpus

package parser

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
)

// BenchmarkCorpus_LargeSpecs benchmarks parsing of large (>5MB) corpus specifications.
// This benchmark is excluded by default to prevent memory exhaustion.
// Run with: go test -tags=corpus -bench=BenchmarkCorpus_LargeSpecs ./parser/...
func BenchmarkCorpus_LargeSpecs(b *testing.B) {
	if os.Getenv("SKIP_LARGE_TESTS") == "1" {
		b.Skip("Skipping large spec benchmarks (SKIP_LARGE_TESTS=1)")
	}

	for _, spec := range corpusutil.GetLargeSpecs() {
		if !spec.IsAvailable() {
			continue
		}

		b.Run(spec.Name, func(b *testing.B) {
			p := New()
			p.ResolveRefs = false
			p.ValidateStructure = true

			for b.Loop() {
				_, err := p.Parse(spec.GetLocalPath())
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCorpus_MarshalJSON benchmarks JSON marshaling of parsed corpus specifications.
// This measures the performance benefit of the marshal buffer pool.
// Run with: go test -tags=corpus -bench=BenchmarkCorpus_MarshalJSON ./parser/...
func BenchmarkCorpus_MarshalJSON(b *testing.B) {
	if os.Getenv("SKIP_LARGE_TESTS") == "1" {
		b.Skip("Skipping large spec benchmarks (SKIP_LARGE_TESTS=1)")
	}

	// Include all parseable specs (both regular and large) to test pool benefits at scale
	for _, spec := range corpusutil.GetParseableSpecs(true) {
		if !spec.IsAvailable() {
			continue
		}

		b.Run(spec.Name, func(b *testing.B) {
			// Parse once during setup (outside benchmark loop)
			p := New()
			p.ResolveRefs = false
			p.ValidateStructure = true

			result, err := p.Parse(spec.GetLocalPath())
			if err != nil {
				b.Fatalf("Failed to parse %s: %v", spec.Name, err)
			}

			// Get the appropriate marshaler based on document type
			var marshaler json.Marshaler
			switch doc := result.Document.(type) {
			case *OAS2Document:
				marshaler = doc
			case *OAS3Document:
				marshaler = doc
			default:
				b.Fatalf("Unexpected document type: %T", result.Document)
			}

			// Report bytes per operation for context
			data, _ := marshaler.MarshalJSON()
			b.SetBytes(int64(len(data)))

			b.ResetTimer()
			for b.Loop() {
				_, err := marshaler.MarshalJSON()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
