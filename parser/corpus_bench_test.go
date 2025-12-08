//go:build corpus

// Corpus benchmarks require the corpus build tag to run.
// Run with: go test -tags=corpus -bench=BenchmarkCorpus ./parser/...
// Or use: make bench-corpus

package parser

import (
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
