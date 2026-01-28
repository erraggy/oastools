package joiner

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

// BenchmarkCompareSchemas benchmarks schema comparison with various schema sizes.
func BenchmarkCompareSchemas(b *testing.B) {
	b.Run("SimpleSchemas", func(b *testing.B) {
		left := &parser.Schema{Type: "string", Format: "email"}
		right := &parser.Schema{Type: "string", Format: "email"}

		b.ReportAllocs()
		for b.Loop() {
			CompareSchemas(left, right, EquivalenceModeDeep)
		}
	})

	b.Run("ObjectWithProperties", func(b *testing.B) {
		left := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"name":  {Type: "string"},
				"email": {Type: "string", Format: "email"},
				"age":   {Type: "integer"},
			},
			Required: []string{"name", "email"},
		}
		right := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"name":  {Type: "string"},
				"email": {Type: "string", Format: "email"},
				"age":   {Type: "integer"},
			},
			Required: []string{"name", "email"},
		}

		b.ReportAllocs()
		for b.Loop() {
			CompareSchemas(left, right, EquivalenceModeDeep)
		}
	})

	b.Run("NestedSchemas", func(b *testing.B) {
		left := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"user": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
						"address": {
							Type: "object",
							Properties: map[string]*parser.Schema{
								"street": {Type: "string"},
								"city":   {Type: "string"},
								"zip":    {Type: "string"},
							},
						},
					},
				},
			},
		}
		right := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"user": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
						"address": {
							Type: "object",
							Properties: map[string]*parser.Schema{
								"street": {Type: "string"},
								"city":   {Type: "string"},
								"zip":    {Type: "string"},
							},
						},
					},
				},
			},
		}

		b.ReportAllocs()
		for b.Loop() {
			CompareSchemas(left, right, EquivalenceModeDeep)
		}
	})

	b.Run("AllOfComposition", func(b *testing.B) {
		left := &parser.Schema{
			AllOf: []*parser.Schema{
				{Type: "object", Properties: map[string]*parser.Schema{"id": {Type: "integer"}}},
				{Type: "object", Properties: map[string]*parser.Schema{"name": {Type: "string"}}},
				{Type: "object", Properties: map[string]*parser.Schema{"email": {Type: "string"}}},
			},
		}
		right := &parser.Schema{
			AllOf: []*parser.Schema{
				{Type: "object", Properties: map[string]*parser.Schema{"id": {Type: "integer"}}},
				{Type: "object", Properties: map[string]*parser.Schema{"name": {Type: "string"}}},
				{Type: "object", Properties: map[string]*parser.Schema{"email": {Type: "string"}}},
			},
		}

		b.ReportAllocs()
		for b.Loop() {
			CompareSchemas(left, right, EquivalenceModeDeep)
		}
	})

	b.Run("DeeplyNested", func(b *testing.B) {
		// Create a deeply nested schema (5 levels)
		createNestedSchema := func() *parser.Schema {
			level5 := &parser.Schema{Type: "string"}
			level4 := &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{"level5": level5}}
			level3 := &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{"level4": level4}}
			level2 := &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{"level3": level3}}
			level1 := &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{"level2": level2}}
			return &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{"level1": level1}}
		}
		left := createNestedSchema()
		right := createNestedSchema()

		b.ReportAllocs()
		for b.Loop() {
			CompareSchemas(left, right, EquivalenceModeDeep)
		}
	})

	b.Run("ShallowMode", func(b *testing.B) {
		left := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"name":  {Type: "string"},
				"email": {Type: "string"},
				"age":   {Type: "integer"},
			},
		}
		right := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"name":  {Type: "string"},
				"email": {Type: "string"},
				"age":   {Type: "integer"},
			},
		}

		b.ReportAllocs()
		for b.Loop() {
			CompareSchemas(left, right, EquivalenceModeShallow)
		}
	})
}

// BenchmarkComparePath benchmarks the path builder vs string concatenation.
func BenchmarkComparePath(b *testing.B) {
	b.Run("StringConcat", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			path := ""
			path = pathJoin(path, "properties")
			path = pathJoin(path, "user")
			path = pathJoin(path, "address")
			path = pathJoin(path, "street")
			_ = path
		}
	})

	b.Run("ComparePathSlice", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			path := &comparePath{segments: make([]string, 0, 8)}
			path.push("properties")
			path.push("user")
			path.push("address")
			path.push("street")
			_ = path.String()
			path.pop()
			path.pop()
			path.pop()
			path.pop()
		}
	})

	b.Run("ComparePathSliceDeep", func(b *testing.B) {
		// Simulate deep recursion pattern
		b.ReportAllocs()
		for b.Loop() {
			path := &comparePath{segments: make([]string, 0, 8)}
			// Simulate going 5 levels deep and back up
			for i := range 5 {
				path.push("level")
				if i == 4 {
					_ = path.String() // Only materialize at deepest level
				}
			}
			for range 5 {
				path.pop()
			}
		}
	})
}

// BenchmarkCompareSchemasDifferences benchmarks comparison when schemas differ.
func BenchmarkCompareSchemasDifferences(b *testing.B) {
	b.Run("SingleDifference", func(b *testing.B) {
		left := &parser.Schema{Type: "string"}
		right := &parser.Schema{Type: "integer"}

		b.ReportAllocs()
		for b.Loop() {
			CompareSchemas(left, right, EquivalenceModeDeep)
		}
	})

	b.Run("MultipleDifferences", func(b *testing.B) {
		left := &parser.Schema{
			Type:   "string",
			Format: "email",
		}
		right := &parser.Schema{
			Type:   "integer",
			Format: "int64",
		}

		b.ReportAllocs()
		for b.Loop() {
			CompareSchemas(left, right, EquivalenceModeDeep)
		}
	})

	b.Run("NestedDifference", func(b *testing.B) {
		left := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"user": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
					},
				},
			},
		}
		right := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"user": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "integer"},
					},
				},
			},
		}

		b.ReportAllocs()
		for b.Loop() {
			CompareSchemas(left, right, EquivalenceModeDeep)
		}
	})
}
