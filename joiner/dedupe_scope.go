package joiner

import (
	"sort"

	"github.com/erraggy/oastools/internal/schemautil"
)

// DeduplicationScope selects which names semantic deduplication may fold into
// another. It does not change which schemas compare equal: an equivalence
// group is built the same way under every scope, and the scope decides only
// what may be consolidated out of one.
type DeduplicationScope string

const (
	// DeduplicationScopeAll folds any equivalent name, whether a document
	// declared it or a collision rename generated it. This is the default.
	DeduplicationScopeAll DeduplicationScope = "all"

	// DeduplicationScopeGeneratedOnly folds only the names a collision rename
	// generated. A name a document declared keeps its own schema, whether or
	// not an equivalent name survives beside it:
	//
	//	{Common, Api_Common}      all: Common      generated-only: Common
	//	{Zebra, Api_Common}       all: Zebra       generated-only: Zebra
	//	{Inventory, Stock}        all: Inventory   generated-only: Inventory, Stock
	//
	// The survivor is chosen the same way under both scopes, so a group mixing
	// the two still folds its generated names into the declared one.
	DeduplicationScopeGeneratedOnly DeduplicationScope = "generated-only"
)

// ValidDeduplicationScopes returns all valid deduplication scope strings.
func ValidDeduplicationScopes() []string {
	return []string{
		string(DeduplicationScopeAll),
		string(DeduplicationScopeGeneratedOnly),
	}
}

// IsValidDeduplicationScope checks if a deduplication scope string is valid.
// The empty string is valid and means DeduplicationScopeAll, so a config that
// never set one behaves as it did before the scope existed.
func IsValidDeduplicationScope(scope string) bool {
	switch DeduplicationScope(scope) {
	case "", DeduplicationScopeAll, DeduplicationScopeGeneratedOnly:
		return true
	default:
		return false
	}
}

// foldableForScope returns the FoldableFunc a scope calls for, given the set of
// names a rename generated. It returns nil when every name may fold, which is
// what schemautil reads as "draw no distinction".
func foldableForScope(scope DeduplicationScope, generated map[string]bool) schemautil.FoldableFunc {
	// JoinerConfig is a struct a caller can fill in directly, so an
	// unrecognized value reaches here and reads as the default.
	resolved := scope
	if !IsValidDeduplicationScope(string(scope)) {
		resolved = DeduplicationScopeAll
	}
	if resolved != DeduplicationScopeGeneratedOnly {
		return nil
	}
	return func(name string) bool {
		return generated[name]
	}
}

// Consolidation is one group semantic deduplication folded, reported when
// JoinerConfig.DeduplicationReport is enabled.
type Consolidation struct {
	// Survivor is the name the group kept.
	Survivor string
	// SurvivorGenerated reports whether a collision rename generated Survivor.
	// A join that renames nothing produces no generated names at all, so this
	// is false throughout unless a collision forced a rename.
	SurvivorGenerated bool
	// Folded holds the names consolidated into Survivor, sorted, each carrying
	// whether a rename generated it and whether the joined document kept it.
	// It is never empty: a group that consolidated nothing is not reported.
	Folded []FoldedName
}

// FoldedName is one name a Consolidation consolidated into its survivor, where
// the name came from, and whether the joined document still carries it.
type FoldedName struct {
	// Name is the consolidated name as it stood in the joined document when
	// deduplication ran, which is after renames are applied. A generated name
	// was never in a source document, and a declared name that a collision
	// renamed reads here under the name the rename produced.
	Name string
	// Generated reports whether a collision rename produced this name rather
	// than a document declaring it. Under
	// DeduplicationScopeGeneratedOnly every folded name is generated.
	Generated bool
	// Pointer reports whether the joined document still carries this name, as a
	// $ref to Survivor, rather than having removed it. It is true for every
	// folded name under DeduplicationModePointer and false under
	// DeduplicationModeRemove, which is the difference between a name a
	// consumer can still refer to and one that is gone.
	Pointer bool
}

// recordConsolidations fills JoinResult.Consolidations from a deduplication
// result, when DeduplicationReport is enabled.
//
// It reads DeduplicationResult.EquivalenceGroups, which reports the surviving
// name first and the names folded into it after, so a group of one folded
// nothing and is not reported. Survivors are sorted, since ranging a map would
// order the report differently on every run.
func (j *Joiner) recordConsolidations(result *JoinResult, dedupe *schemautil.DeduplicationResult, mode DeduplicationMode) {
	if !j.config.DeduplicationReport || len(dedupe.Aliases) == 0 {
		return
	}

	survivors := make([]string, 0, len(dedupe.EquivalenceGroups))
	for survivor, names := range dedupe.EquivalenceGroups {
		if len(names) < 2 {
			continue
		}
		survivors = append(survivors, survivor)
	}
	sort.Strings(survivors)

	consolidations := make([]Consolidation, 0, len(survivors))
	for _, survivor := range survivors {
		names := dedupe.EquivalenceGroups[survivor]
		folded := make([]FoldedName, 0, len(names)-1)
		for _, name := range names[1:] {
			folded = append(folded, FoldedName{
				Name:      name,
				Generated: result.generated[name],
				Pointer:   mode == DeduplicationModePointer,
			})
		}
		consolidations = append(consolidations, Consolidation{
			Survivor:          survivor,
			SurvivorGenerated: result.generated[survivor],
			Folded:            folded,
		})
	}
	result.Consolidations = consolidations
}
