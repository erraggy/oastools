package schemautil

import "github.com/erraggy/oastools/parser"

// DeduplicationConfig configures semantic schema deduplication behavior.
type DeduplicationConfig struct {
	// EquivalenceMode controls comparison depth ("deep" recommended).
	// Uses joiner.EquivalenceMode values: "none", "shallow", "deep".
	EquivalenceMode string
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
