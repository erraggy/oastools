package schemautil

import (
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
//  3. Select canonical name (alphabetically first) for each equivalence group
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
	equivalenceGroups := d.buildEquivalenceGroups(schemas, hashGroups)

	// Phase 3: Select canonical names and build result
	result := d.buildResult(schemas, equivalenceGroups)

	return result, nil
}

// buildEquivalenceGroups verifies equivalence within hash groups and splits false positives.
func (d *SchemaDeduplicator) buildEquivalenceGroups(
	schemas map[string]*parser.Schema,
	hashGroups map[uint64][]string,
) [][]string {
	var equivalenceGroups [][]string

	for _, names := range hashGroups {
		if len(names) == 1 {
			// Single schema in group - no duplicates possible
			equivalenceGroups = append(equivalenceGroups, names)
			continue
		}

		// Verify equivalence and split into true equivalence groups
		subGroups := d.verifyEquivalence(schemas, names)
		equivalenceGroups = append(equivalenceGroups, subGroups...)
	}

	return equivalenceGroups
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
		// Sort names alphabetically to select canonical name deterministically
		sort.Strings(group)
		canonical := group[0]

		// Store canonical schema
		result.CanonicalSchemas[canonical] = schemas[canonical]
		result.EquivalenceGroups[canonical] = group

		// Record aliases (all names except canonical)
		for i := 1; i < len(group); i++ {
			alias := group[i]
			result.Aliases[alias] = canonical
			result.RemovedCount++
		}
	}

	return result
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
