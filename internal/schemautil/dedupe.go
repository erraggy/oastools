package schemautil

import "github.com/erraggy/oastools/parser"

// OutranksFunc reports whether name is a better canonical name than candidate.
// Deduplicate calls it to pick which name a group of equivalent schemas keeps.
//
// It exists because the deduplicator sees nothing but names and schemas. A
// caller that made some of those names up, as joiner does when a collision
// forces a rename, is the only one that can say so.
//
// Write it like a "less than": never say a name outranks itself, and if a
// outranks b and b outranks c, then a should outrank c. A function that breaks
// those rules still returns one of the group's names, just not a predictable
// one.
type OutranksFunc func(name, candidate string) bool

// DeduplicationConfig configures semantic schema deduplication behavior.
type DeduplicationConfig struct {
	// EquivalenceMode controls comparison depth ("deep" recommended).
	// Uses joiner.EquivalenceMode values: "none", "shallow", "deep".
	EquivalenceMode string

	// Outranks chooses the canonical name of an equivalence group. When nil,
	// the alphabetically first name wins.
	Outranks OutranksFunc
}

// DefaultDeduplicationConfig returns a DeduplicationConfig with sensible defaults.
func DefaultDeduplicationConfig() DeduplicationConfig {
	return DeduplicationConfig{
		EquivalenceMode: "deep",
	}
}

// DeduplicationResult contains the outcome of schema deduplication.
type DeduplicationResult struct {
	// CanonicalSchemas maps canonical names to their schema definitions.
	// Only canonical schemas are included; duplicates are removed.
	CanonicalSchemas map[string]*parser.Schema

	// Aliases maps alias schema names to their canonical name.
	// All references to alias names should be rewritten to canonical names.
	Aliases map[string]string

	// RemovedCount is the number of duplicate schemas that were removed.
	RemovedCount int

	// EquivalenceGroups maps canonical names to all equivalent schema names.
	// Includes the canonical name itself as the first element.
	EquivalenceGroups map[string][]string
}
