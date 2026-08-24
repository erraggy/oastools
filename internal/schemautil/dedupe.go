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

// SplitFunc decides how much of a group of equivalent schemas may actually be
// consolidated. Deduplicate calls it once per group, with the names whose
// schemas all compare equal, and consolidates each part it returns on its own:
//
//	group  {Address, Location, Place}
//	returns  {Address, Place}, {Location}   Address and Place collapse to one
//	                                        name, Location keeps its own
//
// Return group unchanged to consolidate it whole, which is what a nil
// SplitFunc does. Every name must come back exactly once: a dropped name loses
// its schema, and a repeated one duplicates it.
//
// It exists because equivalence is a question about shape and this one is not.
// Only a caller that can see how the names are referenced can say that two
// same-shaped schemas describe different things (#501).
type SplitFunc func(group []string) [][]string

// DeduplicationConfig configures semantic schema deduplication behavior.
type DeduplicationConfig struct {
	// EquivalenceMode controls comparison depth ("deep" recommended).
	// Uses joiner.EquivalenceMode values: "none", "shallow", "deep".
	EquivalenceMode string

	// Outranks chooses the canonical name of an equivalence group. When nil,
	// the alphabetically first name wins.
	Outranks OutranksFunc

	// Split holds names apart within a group of equivalent schemas. When nil,
	// each group is consolidated whole.
	Split SplitFunc
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
