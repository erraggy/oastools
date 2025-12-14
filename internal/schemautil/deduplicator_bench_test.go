package schemautil

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

// BenchmarkSchemaDeduplicator_Deduplicate benchmarks the deduplication algorithm.
func BenchmarkSchemaDeduplicator_Deduplicate(b *testing.B) {
	// Simple comparison function for benchmarks
	compare := func(left, right *parser.Schema) bool {
		// Simple type comparison for benchmarking
		return left.Type == right.Type
	}

	b.Run("10Schemas_NoDupes", func(b *testing.B) {
		schemas := generateUniqueSchemas(10)
		config := DefaultDeduplicationConfig()
		d := NewSchemaDeduplicator(config, compare)

		for b.Loop() {
			_, _ = d.Deduplicate(schemas)
		}
	})

	b.Run("10Schemas_50%Dupes", func(b *testing.B) {
		schemas := generateSchemasWithDuplicates(10, 0.5)
		config := DefaultDeduplicationConfig()
		d := NewSchemaDeduplicator(config, compare)

		for b.Loop() {
			_, _ = d.Deduplicate(schemas)
		}
	})

	b.Run("100Schemas_NoDupes", func(b *testing.B) {
		schemas := generateUniqueSchemas(100)
		config := DefaultDeduplicationConfig()
		d := NewSchemaDeduplicator(config, compare)

		for b.Loop() {
			_, _ = d.Deduplicate(schemas)
		}
	})

	b.Run("100Schemas_50%Dupes", func(b *testing.B) {
		schemas := generateSchemasWithDuplicates(100, 0.5)
		config := DefaultDeduplicationConfig()
		d := NewSchemaDeduplicator(config, compare)

		for b.Loop() {
			_, _ = d.Deduplicate(schemas)
		}
	})

	b.Run("100Schemas_90%Dupes", func(b *testing.B) {
		schemas := generateSchemasWithDuplicates(100, 0.9)
		config := DefaultDeduplicationConfig()
		d := NewSchemaDeduplicator(config, compare)

		for b.Loop() {
			_, _ = d.Deduplicate(schemas)
		}
	})

	b.Run("1000Schemas_NoDupes", func(b *testing.B) {
		schemas := generateUniqueSchemas(1000)
		config := DefaultDeduplicationConfig()
		d := NewSchemaDeduplicator(config, compare)

		for b.Loop() {
			_, _ = d.Deduplicate(schemas)
		}
	})

	b.Run("1000Schemas_50%Dupes", func(b *testing.B) {
		schemas := generateSchemasWithDuplicates(1000, 0.5)
		config := DefaultDeduplicationConfig()
		d := NewSchemaDeduplicator(config, compare)

		for b.Loop() {
			_, _ = d.Deduplicate(schemas)
		}
	})
}

// generateUniqueSchemas creates n schemas that are all unique.
func generateUniqueSchemas(n int) map[string]*parser.Schema {
	schemas := make(map[string]*parser.Schema, n)

	for i := range n {
		name := schemaName(i)
		schemas[name] = &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"unique_field_" + name: {Type: "string"},
			},
		}
	}

	return schemas
}

// generateSchemasWithDuplicates creates n schemas where duplicateRatio are duplicates.
func generateSchemasWithDuplicates(n int, duplicateRatio float64) map[string]*parser.Schema {
	schemas := make(map[string]*parser.Schema, n)
	uniqueCount := int(float64(n) * (1 - duplicateRatio))
	if uniqueCount < 1 {
		uniqueCount = 1
	}

	for i := range n {
		name := schemaName(i)
		// Use modulo to create duplicates
		templateIdx := i % uniqueCount
		schemas[name] = &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"shared_field":                        {Type: "string"},
				"template_" + schemaName(templateIdx): {Type: "integer"},
			},
		}
	}

	return schemas
}

// schemaName generates a unique schema name from an index.
func schemaName(i int) string {
	result := ""
	i++ // Start from 1 to avoid empty string
	for i > 0 {
		i-- // Adjust for 0-indexed letters
		result = string(rune('A'+i%26)) + result
		i /= 26
	}
	return result
}
