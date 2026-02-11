package schemautil

import (
	"testing"

	"github.com/erraggy/oastools/internal/testutil"
	"github.com/erraggy/oastools/parser"
)

// BenchmarkSchemaHasher_Hash benchmarks the hashing of various schema types.
func BenchmarkSchemaHasher_Hash(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		schema := &parser.Schema{
			Type: "string",
		}
		h := NewSchemaHasher()

		for b.Loop() {
			_ = h.Hash(schema)
		}
	})

	b.Run("Object", func(b *testing.B) {
		schema := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"id":   {Type: "integer"},
				"name": {Type: "string"},
			},
			Required: []string{"id", "name"},
		}
		h := NewSchemaHasher()

		for b.Loop() {
			_ = h.Hash(schema)
		}
	})

	b.Run("ComplexObject", func(b *testing.B) {
		schema := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"id":       {Type: "integer", Format: "int64"},
				"name":     {Type: "string", MinLength: testutil.Ptr(1), MaxLength: testutil.Ptr(100)},
				"email":    {Type: "string", Format: "email", Pattern: "^[a-z]+@[a-z]+\\.[a-z]+$"},
				"age":      {Type: "integer", Minimum: testutil.Ptr(0.0), Maximum: testutil.Ptr(150.0)},
				"active":   {Type: "boolean"},
				"tags":     {Type: "array", Items: &parser.Schema{Type: "string"}},
				"metadata": {Type: "object", AdditionalProperties: &parser.Schema{Type: "string"}},
			},
			Required: []string{"id", "name", "email"},
		}
		h := NewSchemaHasher()

		for b.Loop() {
			_ = h.Hash(schema)
		}
	})

	b.Run("NestedObject", func(b *testing.B) {
		schema := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"user": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"profile": {
							Type: "object",
							Properties: map[string]*parser.Schema{
								"name":    {Type: "string"},
								"avatar":  {Type: "string", Format: "uri"},
								"bio":     {Type: "string"},
								"website": {Type: "string", Format: "uri"},
							},
						},
						"settings": {
							Type: "object",
							Properties: map[string]*parser.Schema{
								"theme":         {Type: "string", Enum: []any{"light", "dark"}},
								"notifications": {Type: "boolean"},
								"language":      {Type: "string"},
							},
						},
					},
				},
			},
		}
		h := NewSchemaHasher()

		for b.Loop() {
			_ = h.Hash(schema)
		}
	})

	b.Run("Composition", func(b *testing.B) {
		schema := &parser.Schema{
			AllOf: []*parser.Schema{
				{Ref: "#/components/schemas/Base"},
				{
					Type: "object",
					Properties: map[string]*parser.Schema{
						"extended": {Type: "string"},
					},
				},
			},
		}
		h := NewSchemaHasher()

		for b.Loop() {
			_ = h.Hash(schema)
		}
	})
}

// BenchmarkSchemaHasher_GroupByHash benchmarks grouping schemas by hash.
func BenchmarkSchemaHasher_GroupByHash(b *testing.B) {
	b.Run("10Schemas", func(b *testing.B) {
		schemas := generateSchemas(10)
		h := NewSchemaHasher()

		for b.Loop() {
			_ = h.GroupByHash(schemas)
		}
	})

	b.Run("100Schemas", func(b *testing.B) {
		schemas := generateSchemas(100)
		h := NewSchemaHasher()

		for b.Loop() {
			_ = h.GroupByHash(schemas)
		}
	})

	b.Run("1000Schemas", func(b *testing.B) {
		schemas := generateSchemas(1000)
		h := NewSchemaHasher()

		for b.Loop() {
			_ = h.GroupByHash(schemas)
		}
	})
}

// generateSchemas creates n schemas with varying types for benchmarking.
func generateSchemas(n int) map[string]*parser.Schema {
	types := []string{"string", "integer", "boolean", "number"}
	schemas := make(map[string]*parser.Schema, n)

	for i := range n {
		schemaType := types[i%len(types)]
		name := string(rune('A'+i%26)) + string(rune('0'+i/26))

		schemas[name] = &parser.Schema{
			Type: schemaType,
			Properties: map[string]*parser.Schema{
				"field": {Type: "string"},
			},
		}
	}

	return schemas
}
