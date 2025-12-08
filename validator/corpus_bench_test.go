//go:build corpus

// Corpus benchmarks require the corpus build tag to run.
// Run with: go test -tags=corpus -bench=BenchmarkCorpus ./validator/...
// Or use: make bench-corpus

package validator

import (
	"os"
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
)

// BenchmarkCorpus_Validate benchmarks validation of corpus specifications.
// This benchmark is excluded by default to prevent memory exhaustion.
// Run with: go test -tags=corpus -bench=BenchmarkCorpus_Validate ./validator/...
func BenchmarkCorpus_Validate(b *testing.B) {
	// Benchmark with Petstore (small, valid)
	spec := corpusutil.GetByName("Petstore")
	if spec == nil || !spec.IsAvailable() {
		b.Skip("Petstore spec not available")
	}

	b.Run("Petstore", func(b *testing.B) {
		v := New()
		for b.Loop() {
			_, err := v.Validate(spec.GetLocalPath())
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Benchmark with DigitalOcean (medium, valid)
	spec = corpusutil.GetByName("DigitalOcean")
	if spec == nil || !spec.IsAvailable() {
		return
	}

	b.Run("DigitalOcean", func(b *testing.B) {
		v := New()
		for b.Loop() {
			_, err := v.Validate(spec.GetLocalPath())
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkCorpus_LargeSpecs_Validate benchmarks validation of large (>5MB) corpus specs.
// This benchmark is excluded by default to prevent memory exhaustion.
// Run with: go test -tags=corpus -bench=BenchmarkCorpus_LargeSpecs_Validate ./validator/...
func BenchmarkCorpus_LargeSpecs_Validate(b *testing.B) {
	if os.Getenv("SKIP_LARGE_TESTS") == "1" {
		b.Skip("Skipping large spec benchmarks (SKIP_LARGE_TESTS=1)")
	}

	spec := corpusutil.GetByName("Stripe")
	if spec == nil || !spec.IsAvailable() {
		b.Skip("Stripe spec not available")
	}

	b.Run("Stripe", func(b *testing.B) {
		v := New()
		v.IncludeWarnings = false

		for b.Loop() {
			_, err := v.Validate(spec.GetLocalPath())
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
