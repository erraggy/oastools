//go:build corpus

// Corpus benchmarks require the corpus build tag to run.
// Run with: go test -tags=corpus -bench=BenchmarkCorpus ./fixer/...
// Or use: make bench-corpus

package fixer

import (
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
	"github.com/erraggy/oastools/parser"
)

// BenchmarkCorpus_Fix benchmarks fixing of corpus specifications.
// This benchmark is excluded by default to prevent memory exhaustion.
// Run with: go test -tags=corpus -bench=BenchmarkCorpus_Fix ./fixer/...
func BenchmarkCorpus_Fix(b *testing.B) {
	spec := corpusutil.GetByName("Asana")
	if spec == nil {
		b.Skip("Asana spec not found")
	}
	if !spec.IsAvailable() {
		b.Skipf("Corpus file %s not cached", spec.Filename)
	}

	p := parser.New()
	parseResult, err := p.Parse(spec.GetLocalPath())
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		f := New()
		_, err := f.FixParsed(*parseResult)
		if err != nil {
			b.Fatal(err)
		}
	}
}
