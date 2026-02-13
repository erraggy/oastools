package schemautil

import (
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

	assert.Len(t, group, 3)

	// First should be canonical (Address, alphabetically first)
	assert.Equal(t, "Address", group[0])
}
