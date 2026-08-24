package schemautil

import (
	"fmt"
	"slices"
	"sort"

	"github.com/erraggy/oastools/parser"
)

// CompareFunc compares two schemas for structural equivalence.
// Returns true if the schemas are semantically identical.
// This function type allows dependency injection to avoid import cycles.
type CompareFunc func(left, right *parser.Schema) bool

// SchemaDeduplicator identifies and consolidates semantically identical schemas.
type SchemaDeduplicator struct {
	config  DeduplicationConfig
	hasher  *SchemaHasher
	compare CompareFunc
}

// NewSchemaDeduplicator creates a new SchemaDeduplicator.
// The compare function is used to verify equivalence after hash grouping.
// If compare is nil, schemas are considered equivalent if they have the same hash
// (not recommended due to potential hash collisions).
func NewSchemaDeduplicator(config DeduplicationConfig, compare CompareFunc) *SchemaDeduplicator {
	return &SchemaDeduplicator{
		config:  config,
		hasher:  NewSchemaHasher(),
		compare: compare,
	}
}

// Deduplicate identifies semantically identical schemas and consolidates them.
// It returns a result containing only canonical schemas and a mapping of aliases.
//
// The algorithm:
//  1. Group schemas by structural hash (O(N))
//  2. Verify equivalence within each group using deep comparison
//  3. Select canonical name for each equivalence group, by
//     DeduplicationConfig.Outranks or alphabetically when it is nil
//  4. Build alias mapping and return only canonical schemas
func (d *SchemaDeduplicator) Deduplicate(schemas map[string]*parser.Schema) (*DeduplicationResult, error) {
	if len(schemas) < 2 {
		// Nothing to deduplicate
		result := &DeduplicationResult{
			CanonicalSchemas:  make(map[string]*parser.Schema, len(schemas)),
			Aliases:           make(map[string]string),
			RemovedCount:      0,
			EquivalenceGroups: make(map[string][]string),
		}
		for name, schema := range schemas {
			result.CanonicalSchemas[name] = schema
			result.EquivalenceGroups[name] = []string{name}
		}
		return result, nil
	}

	// Phase 1: Group schemas by hash
	hashGroups := d.hasher.GroupByHash(schemas)

	// Phase 2: Verify equivalence within each group and build equivalence groups
	equivalenceGroups, err := d.buildEquivalenceGroups(schemas, hashGroups)
	if err != nil {
		return nil, err
	}

	// Phase 3: Select canonical names and build result
	result := d.buildResult(schemas, equivalenceGroups)

	return result, nil
}

// buildEquivalenceGroups verifies equivalence within hash groups and splits false positives.
func (d *SchemaDeduplicator) buildEquivalenceGroups(
	schemas map[string]*parser.Schema,
	hashGroups map[uint64][]string,
) ([][]string, error) {
	var equivalenceGroups [][]string

	for _, names := range hashGroups {
		if len(names) == 1 {
			// Single schema in group - no duplicates possible
			equivalenceGroups = append(equivalenceGroups, names)
			continue
		}

		// Verify equivalence and split into true equivalence groups, then hand
		// each to the caller, which may hold some of its names apart.
		for _, group := range d.verifyEquivalence(schemas, names) {
			parts, err := d.split(group)
			if err != nil {
				return nil, err
			}
			equivalenceGroups = append(equivalenceGroups, parts...)
		}
	}

	return equivalenceGroups, nil
}

// verifyEquivalence uses deep comparison to split a hash group into true equivalence groups.
func (d *SchemaDeduplicator) verifyEquivalence(
	schemas map[string]*parser.Schema,
	names []string,
) [][]string {
	if d.compare == nil {
		// No comparison function - treat all hash-matching schemas as equivalent
		return [][]string{names}
	}

	// Use union-find-like grouping
	var groups [][]string

	for _, name := range names {
		schema := schemas[name]
		foundGroup := -1

		// Check if this schema matches any existing group
		for groupIdx, group := range groups {
			representative := schemas[group[0]]
			if d.compare(schema, representative) {
				foundGroup = groupIdx
				break
			}
		}

		if foundGroup >= 0 {
			// Add to existing group
			groups[foundGroup] = append(groups[foundGroup], name)
		} else {
			// Start a new group
			groups = append(groups, []string{name})
		}
	}

	return groups
}

// split applies the configured SplitFunc to one equivalence group, leaving the
// group whole when there is none.
func (d *SchemaDeduplicator) split(group []string) ([][]string, error) {
	if d.config.Split == nil || len(group) < 2 {
		return [][]string{group}, nil
	}
	// A SplitFunc receives a copy, so what it returns is checked against a
	// group it could not have edited. Handed the original, it could overwrite
	// an entry and return it, and the check would compare the result against
	// the names it just wrote rather than the ones the group had.
	parts := d.config.Split(slices.Clone(group))
	if err := checkPartition(group, parts); err != nil {
		return nil, err
	}
	return parts, nil
}

// checkPartition reports whether parts place every name in group exactly once.
//
// The result is taken at face value from here on, so a name left out is a
// schema dropped with no error, and a name in two parts is one schema written
// under two canonical names. Neither is recoverable later: reference rewriting
// reads the aliases this produces, so it cannot put back what never arrived.
func checkPartition(group []string, parts [][]string) error {
	placed := make(map[string]int, len(group))
	total := 0
	for _, part := range parts {
		if len(part) == 0 {
			return fmt.Errorf("schemautil: Split returned an empty part for group %v", group)
		}
		for _, name := range part {
			placed[name]++
			total++
		}
	}
	if total != len(group) {
		return fmt.Errorf("schemautil: Split returned %d names for a group of %d: %v",
			total, len(group), parts)
	}
	for _, name := range group {
		if placed[name] != 1 {
			return fmt.Errorf("schemautil: Split returned %q %d times for group %v",
				name, placed[name], group)
		}
	}
	return nil
}

// buildResult creates the final deduplication result.
func (d *SchemaDeduplicator) buildResult(
	schemas map[string]*parser.Schema,
	equivalenceGroups [][]string,
) *DeduplicationResult {
	result := &DeduplicationResult{
		CanonicalSchemas:  make(map[string]*parser.Schema),
		Aliases:           make(map[string]string),
		RemovedCount:      0,
		EquivalenceGroups: make(map[string][]string),
	}

	for _, group := range equivalenceGroups {
		// Sort names alphabetically so the canonical name does not depend on the
		// order the schemas were hashed in
		sort.Strings(group)
		canonical := d.canonicalName(group)

		// Store canonical schema
		result.CanonicalSchemas[canonical] = schemas[canonical]
		result.EquivalenceGroups[canonical] = canonicalFirst(group, canonical)

		// Record aliases (all names except canonical)
		for _, alias := range group {
			if alias == canonical {
				continue
			}
			result.Aliases[alias] = canonical
			result.RemovedCount++
		}
	}

	return result
}

// canonicalName picks the name an equivalence group keeps. group must be
// sorted, so the alphabetically first name is what an absent Outranks yields.
func (d *SchemaDeduplicator) canonicalName(group []string) string {
	canonical := group[0]
	if d.config.Outranks == nil {
		return canonical
	}
	for _, name := range group[1:] {
		if d.config.Outranks(name, canonical) {
			canonical = name
		}
	}
	return canonical
}

// canonicalFirst returns group with canonical moved to the front, which is
// where DeduplicationResult.EquivalenceGroups reports it. The rest stay in
// order, so the result is the sorted group whenever canonical is already first.
func canonicalFirst(group []string, canonical string) []string {
	ordered := make([]string, 0, len(group))
	ordered = append(ordered, canonical)
	for _, name := range group {
		if name != canonical {
			ordered = append(ordered, name)
		}
	}
	return ordered
}

// CanonicalName returns the canonical name for a schema name.
// If the name is not an alias, it returns the name unchanged.
func (r *DeduplicationResult) CanonicalName(name string) string {
	if canonical, ok := r.Aliases[name]; ok {
		return canonical
	}
	return name
}

// IsAlias returns true if the given name is an alias (not canonical).
func (r *DeduplicationResult) IsAlias(name string) bool {
	_, ok := r.Aliases[name]
	return ok
}

// IsCanonical returns true if the given name is a canonical schema name.
func (r *DeduplicationResult) IsCanonical(name string) bool {
	_, ok := r.CanonicalSchemas[name]
	return ok
}
