package schemautil

import (
	"testing"

	"github.com/erraggy/oastools/parser"
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
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.CanonicalSchemas) != 0 {
		t.Errorf("Expected 0 canonical schemas, got %d", len(result.CanonicalSchemas))
	}
	if result.RemovedCount != 0 {
		t.Errorf("Expected 0 removed, got %d", result.RemovedCount)
	}
}

func TestSchemaDeduplicator_Deduplicate_Single(t *testing.T) {
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), alwaysEqual)

	schemas := map[string]*parser.Schema{
		"User": {Type: "object"},
	}

	result, err := deduper.Deduplicate(schemas)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.CanonicalSchemas) != 1 {
		t.Errorf("Expected 1 canonical schema, got %d", len(result.CanonicalSchemas))
	}
	if _, ok := result.CanonicalSchemas["User"]; !ok {
		t.Error("Expected User to be canonical")
	}
	if result.RemovedCount != 0 {
		t.Errorf("Expected 0 removed, got %d", result.RemovedCount)
	}
}

func TestSchemaDeduplicator_Deduplicate_Duplicates(t *testing.T) {
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), structuralEqual)

	schemas := map[string]*parser.Schema{
		"Address":  {Type: "object"},
		"Location": {Type: "object"}, // Same as Address
		"User":     {Type: "object"}, // Same as Address
	}

	result, err := deduper.Deduplicate(schemas)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have 1 canonical schema (alphabetically first: Address)
	if len(result.CanonicalSchemas) != 1 {
		t.Errorf("Expected 1 canonical schema, got %d", len(result.CanonicalSchemas))
	}
	if _, ok := result.CanonicalSchemas["Address"]; !ok {
		t.Error("Expected Address to be canonical (alphabetically first)")
	}

	// Should have 2 aliases
	if len(result.Aliases) != 2 {
		t.Errorf("Expected 2 aliases, got %d", len(result.Aliases))
	}
	if result.Aliases["Location"] != "Address" {
		t.Errorf("Expected Location -> Address, got %s", result.Aliases["Location"])
	}
	if result.Aliases["User"] != "Address" {
		t.Errorf("Expected User -> Address, got %s", result.Aliases["User"])
	}

	if result.RemovedCount != 2 {
		t.Errorf("Expected 2 removed, got %d", result.RemovedCount)
	}
}

func TestSchemaDeduplicator_Deduplicate_NoDuplicates(t *testing.T) {
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), structuralEqual)

	schemas := map[string]*parser.Schema{
		"User":    {Type: "object"},
		"Address": {Type: "string"},
		"Age":     {Type: "integer"},
	}

	result, err := deduper.Deduplicate(schemas)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.CanonicalSchemas) != 3 {
		t.Errorf("Expected 3 canonical schemas, got %d", len(result.CanonicalSchemas))
	}
	if len(result.Aliases) != 0 {
		t.Errorf("Expected 0 aliases, got %d", len(result.Aliases))
	}
	if result.RemovedCount != 0 {
		t.Errorf("Expected 0 removed, got %d", result.RemovedCount)
	}
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
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have 3 canonical schemas
	if len(result.CanonicalSchemas) != 3 {
		t.Errorf("Expected 3 canonical schemas, got %d", len(result.CanonicalSchemas))
	}

	// Check canonical names (alphabetically first in each group)
	if _, ok := result.CanonicalSchemas["Address"]; !ok {
		t.Error("Expected Address to be canonical")
	}
	if _, ok := result.CanonicalSchemas["Name"]; !ok {
		t.Error("Expected Name to be canonical")
	}
	if _, ok := result.CanonicalSchemas["Age"]; !ok {
		t.Error("Expected Age to be canonical")
	}

	// Check aliases
	if result.Aliases["Location"] != "Address" {
		t.Errorf("Expected Location -> Address, got %s", result.Aliases["Location"])
	}
	if result.Aliases["Title"] != "Name" {
		t.Errorf("Expected Title -> Name, got %s", result.Aliases["Title"])
	}

	if result.RemovedCount != 2 {
		t.Errorf("Expected 2 removed, got %d", result.RemovedCount)
	}
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
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Apple should be canonical (alphabetically first)
	if len(result.CanonicalSchemas) != 1 {
		t.Errorf("Expected 1 canonical schema, got %d", len(result.CanonicalSchemas))
	}
	if _, ok := result.CanonicalSchemas["Apple"]; !ok {
		t.Error("Expected Apple to be canonical (alphabetically first)")
	}

	// All others should be aliases to Apple
	for _, name := range []string{"Banana", "Mango", "Zebra"} {
		if result.Aliases[name] != "Apple" {
			t.Errorf("Expected %s -> Apple, got %s", name, result.Aliases[name])
		}
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
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should deduplicate based on hash alone
	if len(result.CanonicalSchemas) != 1 {
		t.Errorf("Expected 1 canonical schema, got %d", len(result.CanonicalSchemas))
	}
}

func TestSchemaDeduplicator_Deduplicate_HashCollision(t *testing.T) {
	// Test that compare func correctly splits hash collisions
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), neverEqual)

	schemas := map[string]*parser.Schema{
		"User":   {Type: "object"},
		"Person": {Type: "object"}, // Same hash, but compare returns false
	}

	result, err := deduper.Deduplicate(schemas)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should not deduplicate because compare returns false
	if len(result.CanonicalSchemas) != 2 {
		t.Errorf("Expected 2 canonical schemas (no dedup due to compare), got %d", len(result.CanonicalSchemas))
	}
	if len(result.Aliases) != 0 {
		t.Errorf("Expected 0 aliases, got %d", len(result.Aliases))
	}
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
			if got != tt.expected {
				t.Errorf("CanonicalName(%s) = %s, want %s", tt.input, got, tt.expected)
			}
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

	if !result.IsAlias("Location") {
		t.Error("Location should be an alias")
	}
	if result.IsAlias("Address") {
		t.Error("Address should not be an alias")
	}
	if result.IsAlias("Unknown") {
		t.Error("Unknown should not be an alias")
	}
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

	if !result.IsCanonical("Address") {
		t.Error("Address should be canonical")
	}
	if result.IsCanonical("Location") {
		t.Error("Location should not be canonical")
	}
	if result.IsCanonical("Unknown") {
		t.Error("Unknown should not be canonical")
	}
}

func TestDeduplicationResult_EquivalenceGroups(t *testing.T) {
	deduper := NewSchemaDeduplicator(DefaultDeduplicationConfig(), alwaysEqual)

	schemas := map[string]*parser.Schema{
		"Address":  {Type: "object"},
		"Location": {Type: "object"},
		"Place":    {Type: "object"},
	}

	result, err := deduper.Deduplicate(schemas)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check equivalence groups
	group, ok := result.EquivalenceGroups["Address"]
	if !ok {
		t.Fatal("Expected Address in equivalence groups")
	}

	if len(group) != 3 {
		t.Errorf("Expected 3 members in group, got %d", len(group))
	}

	// First should be canonical (Address, alphabetically first)
	if group[0] != "Address" {
		t.Errorf("Expected Address as first (canonical), got %s", group[0])
	}
}
