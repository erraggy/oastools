package schemautil

import (
	"slices"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// alwaysEqual is a compare function that always returns true
func alwaysEqual(_, _ *parser.Schema) bool {
	return true
}

// neverEqual is a compare function that always returns false
func neverEqual(_, _ *parser.Schema) bool {
	return false
}

// structuralEqual compares two schemas for structural equality (simplified for tests)
func structuralEqual(a, b *parser.Schema) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// Simple comparison for tests - just check type and format
	aType, _ := a.Type.(string)
	bType, _ := b.Type.(string)
	return aType == bType && a.Format == b.Format
}

func TestSchemaDeduplicator_Deduplicate_Empty(t *testing.T) {
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), alwaysEqual)

	result, err := deduper.Deduplicate(map[string]*parser.Schema{})
	require.NoError(t, err)

	assert.Empty(t, result.CanonicalSchemas)
	assert.Equal(t, 0, result.RemovedCount)
}

func TestSchemaDeduplicator_Deduplicate_Single(t *testing.T) {
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), alwaysEqual)

	schemas := map[string]*parser.Schema{
		"User": {Type: "object"},
	}

	result, err := deduper.Deduplicate(schemas)
	require.NoError(t, err)

	assert.Len(t, result.CanonicalSchemas, 1)
	assert.Contains(t, result.CanonicalSchemas, "User")
	assert.Equal(t, 0, result.RemovedCount)
}

func TestSchemaDeduplicator_Deduplicate_Duplicates(t *testing.T) {
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), structuralEqual)

	schemas := map[string]*parser.Schema{
		"Address":  {Type: "object"},
		"Location": {Type: "object"}, // Same as Address
		"User":     {Type: "object"}, // Same as Address
	}

	result, err := deduper.Deduplicate(schemas)
	require.NoError(t, err)

	// Should have 1 canonical schema (alphabetically first: Address)
	assert.Len(t, result.CanonicalSchemas, 1)
	assert.Contains(t, result.CanonicalSchemas, "Address")

	// Should have 2 aliases
	assert.Len(t, result.Aliases, 2)
	assert.Equal(t, "Address", result.Aliases["Location"])
	assert.Equal(t, "Address", result.Aliases["User"])

	assert.Equal(t, 2, result.RemovedCount)
}

func TestSchemaDeduplicator_Deduplicate_NoDuplicates(t *testing.T) {
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), structuralEqual)

	schemas := map[string]*parser.Schema{
		"User":    {Type: "object"},
		"Address": {Type: "string"},
		"Age":     {Type: "integer"},
	}

	result, err := deduper.Deduplicate(schemas)
	require.NoError(t, err)

	assert.Len(t, result.CanonicalSchemas, 3)
	assert.Empty(t, result.Aliases)
	assert.Equal(t, 0, result.RemovedCount)
}

func TestSchemaDeduplicator_Deduplicate_MultipleGroups(t *testing.T) {
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), structuralEqual)

	schemas := map[string]*parser.Schema{
		// Group 1: objects
		"Address":  {Type: "object"},
		"Location": {Type: "object"},
		// Group 2: strings
		"Name":  {Type: "string"},
		"Title": {Type: "string"},
		// Unique
		"Age": {Type: "integer"},
	}

	result, err := deduper.Deduplicate(schemas)
	require.NoError(t, err)

	// Should have 3 canonical schemas
	assert.Len(t, result.CanonicalSchemas, 3)

	// Check canonical names (alphabetically first in each group)
	assert.Contains(t, result.CanonicalSchemas, "Address")
	assert.Contains(t, result.CanonicalSchemas, "Name")
	assert.Contains(t, result.CanonicalSchemas, "Age")

	// Check aliases
	assert.Equal(t, "Address", result.Aliases["Location"])
	assert.Equal(t, "Name", result.Aliases["Title"])

	assert.Equal(t, 2, result.RemovedCount)
}

func TestSchemaDeduplicator_Deduplicate_AlphabeticCanonical(t *testing.T) {
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), alwaysEqual)

	schemas := map[string]*parser.Schema{
		"Zebra":  {Type: "object"},
		"Apple":  {Type: "object"},
		"Mango":  {Type: "object"},
		"Banana": {Type: "object"},
	}

	result, err := deduper.Deduplicate(schemas)
	require.NoError(t, err)

	// Apple should be canonical (alphabetically first)
	assert.Len(t, result.CanonicalSchemas, 1)
	assert.Contains(t, result.CanonicalSchemas, "Apple")

	// All others should be aliases to Apple
	for _, name := range []string{"Banana", "Mango", "Zebra"} {
		assert.Equal(t, "Apple", result.Aliases[name])
	}
}

func TestSchemaDeduplicator_Deduplicate_NilCompareFunc(t *testing.T) {
	// When compare func is nil, hash matching is enough
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), nil)

	schemas := map[string]*parser.Schema{
		"User":   {Type: "object"},
		"Person": {Type: "object"}, // Same hash as User
	}

	result, err := deduper.Deduplicate(schemas)
	require.NoError(t, err)

	// Should deduplicate based on hash alone
	assert.Len(t, result.CanonicalSchemas, 1)
}

func TestSchemaDeduplicator_Deduplicate_HashCollision(t *testing.T) {
	// Test that compare func correctly splits hash collisions
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), neverEqual)

	schemas := map[string]*parser.Schema{
		"User":   {Type: "object"},
		"Person": {Type: "object"}, // Same hash, but compare returns false
	}

	result, err := deduper.Deduplicate(schemas)
	require.NoError(t, err)

	// Should not deduplicate because compare returns false
	assert.Len(t, result.CanonicalSchemas, 2)
	assert.Empty(t, result.Aliases)
}

func TestDeduplicationResult_CanonicalName(t *testing.T) {
	result := &DeduplicationResult{
		CanonicalSchemas: map[string]*parser.Schema{
			"Address": {Type: "object"},
		},
		Aliases: map[string]string{
			"Location": "Address",
			"Place":    "Address",
		},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"alias returns canonical", "Location", "Address"},
		{"alias returns canonical 2", "Place", "Address"},
		{"canonical returns itself", "Address", "Address"},
		{"unknown returns itself", "Unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := result.CanonicalName(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDeduplicationResult_IsAlias(t *testing.T) {
	result := &DeduplicationResult{
		CanonicalSchemas: map[string]*parser.Schema{
			"Address": {Type: "object"},
		},
		Aliases: map[string]string{
			"Location": "Address",
		},
	}

	assert.True(t, result.IsAlias("Location"))
	assert.False(t, result.IsAlias("Address"))
	assert.False(t, result.IsAlias("Unknown"))
}

func TestDeduplicationResult_IsCanonical(t *testing.T) {
	result := &DeduplicationResult{
		CanonicalSchemas: map[string]*parser.Schema{
			"Address": {Type: "object"},
		},
		Aliases: map[string]string{
			"Location": "Address",
		},
	}

	assert.True(t, result.IsCanonical("Address"))
	assert.False(t, result.IsCanonical("Location"))
	assert.False(t, result.IsCanonical("Unknown"))
}

func TestDeduplicationResult_EquivalenceGroups(t *testing.T) {
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), alwaysEqual)

	schemas := map[string]*parser.Schema{
		"Address":  {Type: "object"},
		"Location": {Type: "object"},
		"Place":    {Type: "object"},
	}

	result, err := deduper.Deduplicate(schemas)
	require.NoError(t, err)

	// Check equivalence groups
	group, ok := result.EquivalenceGroups["Address"]
	require.True(t, ok, "Expected Address in equivalence groups")

	require.Len(t, group, 3)

	// First should be canonical (Address, alphabetically first)
	assert.Equal(t, "Address", group[0])
}

func TestSchemaDeduplicator_Outranks(t *testing.T) {
	schemas := map[string]*parser.Schema{
		"Address":  {Type: "object"},
		"Location": {Type: "object"},
		"Place":    {Type: "object"},
	}

	tests := []struct {
		name      string
		outranks  OutranksFunc
		canonical string
		aliases   []string
	}{
		{
			name:      "nil keeps the alphabetical tiebreak",
			outranks:  nil,
			canonical: "Address",
			aliases:   []string{"Location", "Place"},
		},
		{
			// The joiner's case: a name a rename generated loses to one a
			// document wrote, whatever the two sort like.
			name: "generated name loses to a declared one",
			outranks: func(name, candidate string) bool {
				generated := map[string]bool{"Address": true}
				if generated[name] != generated[candidate] {
					return !generated[name]
				}
				return name < candidate
			},
			canonical: "Location",
			aliases:   []string{"Address", "Place"},
		},
		{
			name: "every name outranking the last still settles on one",
			outranks: func(name, candidate string) bool {
				return name > candidate
			},
			canonical: "Place",
			aliases:   []string{"Address", "Location"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultDeduplicationConfig()
			config.Outranks = tt.outranks
			deduper := NewSchemaDeduplicator(config, alwaysEqual)

			result, err := deduper.Deduplicate(schemas)
			require.NoError(t, err)

			require.Len(t, result.CanonicalSchemas, 1)
			assert.Contains(t, result.CanonicalSchemas, tt.canonical)
			assert.Equal(t, len(tt.aliases), result.RemovedCount)
			for _, alias := range tt.aliases {
				assert.Equal(t, tt.canonical, result.CanonicalName(alias))
			}

			// The canonical name leads its group whether or not it sorts first.
			group := result.EquivalenceGroups[tt.canonical]
			require.Len(t, group, len(schemas))
			assert.Equal(t, tt.canonical, group[0])
			assert.Equal(t, tt.aliases, group[1:], "the rest stay sorted")
		})
	}
}

// splitHoldingApart builds a SplitFunc that keeps the named groups separate,
// standing in for the joiner's partition without its reference bookkeeping.
// Any name not listed joins a single remainder part, appended last.
func splitHoldingApart(apart ...[]string) SplitFunc {
	return func(group []string) [][]string {
		parts := make([][]string, 0, len(apart)+1)
		rest := make([]string, 0, len(group))
		for _, name := range group {
			part := -1
			for i, names := range apart {
				if slices.Contains(names, name) {
					part = i
					break
				}
			}
			if part < 0 {
				rest = append(rest, name)
				continue
			}
			for len(parts) <= part {
				parts = append(parts, nil)
			}
			parts[part] = append(parts[part], name)
		}
		if len(rest) > 0 {
			parts = append(parts, rest)
		}
		return parts
	}
}

func TestSchemaDeduplicator_Split(t *testing.T) {
	schemas := map[string]*parser.Schema{
		"Address":  {Type: "object"},
		"Location": {Type: "object"},
		"Place":    {Type: "object"},
	}

	tests := []struct {
		name   string
		split  SplitFunc
		groups [][]string
	}{
		{
			name:   "nil consolidates the group whole",
			split:  nil,
			groups: [][]string{{"Address", "Location", "Place"}},
		},
		{
			name:   "a name held apart keeps its own name",
			split:  splitHoldingApart([]string{"Location"}),
			groups: [][]string{{"Address", "Place"}, {"Location"}},
		},
		{
			name:   "two names held together still consolidate",
			split:  splitHoldingApart([]string{"Location", "Place"}),
			groups: [][]string{{"Address"}, {"Location", "Place"}},
		},
		{
			name: "every name held apart leaves every name standing",
			split: splitHoldingApart(
				[]string{"Address"}, []string{"Location"}, []string{"Place"}),
			groups: [][]string{{"Address"}, {"Location"}, {"Place"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultDeduplicationConfig()
			config.Split = tt.split
			deduper := NewSchemaDeduplicator(config, alwaysEqual)

			result, err := deduper.Deduplicate(schemas)
			require.NoError(t, err)

			require.Len(t, result.CanonicalSchemas, len(tt.groups))
			removed := 0
			for _, group := range tt.groups {
				canonical := group[0]
				require.Contains(t, result.CanonicalSchemas, canonical)
				assert.Equal(t, group, result.EquivalenceGroups[canonical])
				for _, alias := range group[1:] {
					assert.Equal(t, canonical, result.CanonicalName(alias))
					removed++
				}
			}
			assert.Equal(t, removed, result.RemovedCount)
		})
	}
}

// A group only one schema falls into is never handed to the SplitFunc, so a
// caller cannot be asked to partition something with nothing to partition.
func TestSchemaDeduplicator_SplitSkipsSingletons(t *testing.T) {
	schemas := map[string]*parser.Schema{
		"Address": {Type: "object"},
		"Count":   {Type: "integer"},
	}

	var asked [][]string
	config := DefaultDeduplicationConfig()
	config.Split = func(group []string) [][]string {
		asked = append(asked, group)
		return [][]string{group}
	}
	deduper := NewSchemaDeduplicator(config, func(left, right *parser.Schema) bool {
		return left.Type == right.Type
	})

	result, err := deduper.Deduplicate(schemas)
	require.NoError(t, err)

	assert.Empty(t, asked, "neither name shares a group with another")
	assert.Len(t, result.CanonicalSchemas, 2)
}

// A SplitFunc that loses, repeats, or empties a part would drop or duplicate a
// schema, and nothing downstream could tell. Deduplicate refuses instead.
func TestSchemaDeduplicator_SplitPartitionIsChecked(t *testing.T) {
	schemas := map[string]*parser.Schema{
		"Address":  {Type: "object"},
		"Location": {Type: "object"},
		"Place":    {Type: "object"},
	}

	tests := []struct {
		name  string
		split SplitFunc
		want  string
	}{
		{
			name:  "a name left out",
			split: func(group []string) [][]string { return [][]string{group[:len(group)-1]} },
			want:  "returned 2 names for a group of 3",
		},
		{
			name:  "a name in two parts",
			split: func(group []string) [][]string { return [][]string{group, {group[0]}} },
			want:  "returned 4 names for a group of 3",
		},
		{
			// By name rather than by position: a group arrives in the order
			// the schemas hashed in, which is a map range.
			name: "a name swapped for one that was not in the group",
			split: func(group []string) [][]string {
				swapped := make([]string, 0, len(group))
				for _, name := range group {
					if name == "Address" {
						swapped = append(swapped, "Unrelated")
						continue
					}
					swapped = append(swapped, name)
				}
				return [][]string{swapped}
			},
			want: `returned "Address" 0 times`,
		},
		{
			name:  "an empty part",
			split: func(group []string) [][]string { return [][]string{group, {}} },
			want:  "returned an empty part",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultDeduplicationConfig()
			config.Split = tt.split
			deduper := NewSchemaDeduplicator(config, alwaysEqual)

			result, err := deduper.Deduplicate(schemas)
			require.Error(t, err, "an unusable partition must not reach the result")
			assert.Contains(t, err.Error(), tt.want)
			assert.Nil(t, result)
		})
	}
}
