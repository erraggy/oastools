package parser

import (
	"testing"
)

// Benchmark Design Notes:
//
// These benchmarks measure the performance of equality comparison methods.
// They are designed to:
// 1. Test identical documents (best case - full traversal)
// 2. Test early exit scenarios (differences found early)
// 3. Compare performance across different document sizes
// 4. Compare OAS 2.0 vs OAS 3.x performance characteristics

// =============================================================================
// BenchmarkParseResultEquals - Full ParseResult equality benchmarks
// =============================================================================

// BenchmarkParseResultEquals benchmarks ParseResult.Equals() across different
// document sizes and OAS versions.
func BenchmarkParseResultEquals(b *testing.B) {
	b.Run("Identical/SmallOAS3", func(b *testing.B) {
		result, err := ParseWithOptions(WithFilePath(smallOAS3Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		other := result.Copy()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = result.Equals(other)
		}
	})

	b.Run("Identical/MediumOAS3", func(b *testing.B) {
		result, err := ParseWithOptions(WithFilePath(mediumOAS3Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		other := result.Copy()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = result.Equals(other)
		}
	})

	b.Run("Identical/LargeOAS3", func(b *testing.B) {
		result, err := ParseWithOptions(WithFilePath(largeOAS3Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		other := result.Copy()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = result.Equals(other)
		}
	})

	b.Run("Identical/SmallOAS2", func(b *testing.B) {
		result, err := ParseWithOptions(WithFilePath(smallOAS2Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		other := result.Copy()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = result.Equals(other)
		}
	})

	b.Run("Identical/MediumOAS2", func(b *testing.B) {
		result, err := ParseWithOptions(WithFilePath(mediumOAS2Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		other := result.Copy()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = result.Equals(other)
		}
	})
}

// =============================================================================
// BenchmarkEquals_EarlyExit - Tests early exit performance characteristics
// =============================================================================

// BenchmarkEquals_EarlyExit benchmarks how quickly Equals() exits when
// differences are found at various points in the comparison.
func BenchmarkEquals_EarlyExit(b *testing.B) {
	// DifferentVersion: Documents differ in Version field (early exit after enum check)
	b.Run("DifferentVersion", func(b *testing.B) {
		result1, err := ParseWithOptions(WithFilePath(mediumOAS3Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		result2 := result1.Copy()
		// Change the version string to trigger early exit
		result2.Version = "3.0.0"

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = result1.Equals(result2)
		}
	})

	// DifferentOASVersion: Documents differ in OASVersion enum (fastest possible exit)
	b.Run("DifferentOASVersion", func(b *testing.B) {
		result1, err := ParseWithOptions(WithFilePath(mediumOAS3Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		result2 := result1.Copy()
		// Change the OASVersion enum to trigger earliest exit
		result2.OASVersion = OASVersion310

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = result1.Equals(result2)
		}
	})

	// DifferentInfo: Documents differ in Info.Title (early exit during document comparison)
	b.Run("DifferentInfo", func(b *testing.B) {
		result1, err := ParseWithOptions(WithFilePath(mediumOAS3Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		result2 := result1.Copy()
		// Modify Info.Title in the copied document
		if doc, ok := result2.OAS3Document(); ok {
			doc.Info.Title = "Different Title"
		}

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = result1.Equals(result2)
		}
	})

	// DifferentLastPath: Documents differ only in last path (late exit, full traversal)
	b.Run("DifferentLastPath", func(b *testing.B) {
		result1, err := ParseWithOptions(WithFilePath(mediumOAS3Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		result2 := result1.Copy()
		// Add a new path to the copied document (will be compared last alphabetically with z-prefix)
		if doc, ok := result2.OAS3Document(); ok {
			if doc.Paths == nil {
				doc.Paths = make(Paths)
			}
			doc.Paths["/zzz-new-path"] = &PathItem{
				Get: &Operation{Summary: "New operation"},
			}
		}

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = result1.Equals(result2)
		}
	})
}

// =============================================================================
// BenchmarkSchemaEquals - Schema-level equality benchmarks
// =============================================================================

// BenchmarkSchemaEquals benchmarks Schema.Equals() for various schema complexities.
func BenchmarkSchemaEquals(b *testing.B) {
	// Simple: Schema with just Type and Format
	b.Run("Simple", func(b *testing.B) {
		schema1 := &Schema{
			Type:   "string",
			Format: "email",
		}
		schema2 := &Schema{
			Type:   "string",
			Format: "email",
		}

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = schema1.Equals(schema2)
		}
	})

	// WithProperties: Schema with 5 properties
	b.Run("WithProperties", func(b *testing.B) {
		schema1 := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"id":        {Type: "string", Format: "uuid"},
				"name":      {Type: "string"},
				"email":     {Type: "string", Format: "email"},
				"age":       {Type: "integer"},
				"createdAt": {Type: "string", Format: "date-time"},
			},
			Required: []string{"id", "name", "email"},
		}
		schema2 := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"id":        {Type: "string", Format: "uuid"},
				"name":      {Type: "string"},
				"email":     {Type: "string", Format: "email"},
				"age":       {Type: "integer"},
				"createdAt": {Type: "string", Format: "date-time"},
			},
			Required: []string{"id", "name", "email"},
		}

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = schema1.Equals(schema2)
		}
	})

	// Complex: Schema with AllOf, Properties, Required, etc.
	b.Run("Complex", func(b *testing.B) {
		baseSchema := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"id":   {Type: "string"},
				"type": {Type: "string"},
			},
		}
		schema1 := &Schema{
			AllOf: []*Schema{
				baseSchema,
				{
					Type: "object",
					Properties: map[string]*Schema{
						"name":        {Type: "string"},
						"description": {Type: "string"},
						"metadata": {
							Type: "object",
							AdditionalProperties: &Schema{
								Type: "string",
							},
						},
					},
					Required: []string{"name"},
				},
			},
			Title:       "ComplexSchema",
			Description: "A complex schema for testing",
		}
		schema2 := &Schema{
			AllOf: []*Schema{
				{
					Type: "object",
					Properties: map[string]*Schema{
						"id":   {Type: "string"},
						"type": {Type: "string"},
					},
				},
				{
					Type: "object",
					Properties: map[string]*Schema{
						"name":        {Type: "string"},
						"description": {Type: "string"},
						"metadata": {
							Type: "object",
							AdditionalProperties: &Schema{
								Type: "string",
							},
						},
					},
					Required: []string{"name"},
				},
			},
			Title:       "ComplexSchema",
			Description: "A complex schema for testing",
		}

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = schema1.Equals(schema2)
		}
	})

	// Nested: Schema with deeply nested structure
	b.Run("Nested", func(b *testing.B) {
		// Create a deeply nested schema (4 levels deep)
		level4 := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"value": {Type: "string"},
			},
		}
		level3 := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"nested": level4,
			},
		}
		level2 := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"nested": level3,
			},
		}
		schema1 := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"nested": level2,
			},
		}

		// Create an identical copy
		level4Copy := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"value": {Type: "string"},
			},
		}
		level3Copy := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"nested": level4Copy,
			},
		}
		level2Copy := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"nested": level3Copy,
			},
		}
		schema2 := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"nested": level2Copy,
			},
		}

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = schema1.Equals(schema2)
		}
	})
}

// =============================================================================
// BenchmarkDocumentEquals - Document-level equality benchmarks
// =============================================================================

// BenchmarkDocumentEquals benchmarks document equality at various sizes.
func BenchmarkDocumentEquals(b *testing.B) {
	b.Run("OAS3/Small", func(b *testing.B) {
		result, err := ParseWithOptions(WithFilePath(smallOAS3Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		doc1, ok := result.OAS3Document()
		if !ok {
			b.Fatal("expected OAS3 document")
		}
		doc2 := doc1.DeepCopy()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = doc1.Equals(doc2)
		}
	})

	b.Run("OAS3/Medium", func(b *testing.B) {
		result, err := ParseWithOptions(WithFilePath(mediumOAS3Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		doc1, ok := result.OAS3Document()
		if !ok {
			b.Fatal("expected OAS3 document")
		}
		doc2 := doc1.DeepCopy()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = doc1.Equals(doc2)
		}
	})

	b.Run("OAS3/Large", func(b *testing.B) {
		result, err := ParseWithOptions(WithFilePath(largeOAS3Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		doc1, ok := result.OAS3Document()
		if !ok {
			b.Fatal("expected OAS3 document")
		}
		doc2 := doc1.DeepCopy()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = doc1.Equals(doc2)
		}
	})

	b.Run("OAS2/Small", func(b *testing.B) {
		result, err := ParseWithOptions(WithFilePath(smallOAS2Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		doc1, ok := result.OAS2Document()
		if !ok {
			b.Fatal("expected OAS2 document")
		}
		doc2 := doc1.DeepCopy()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = doc1.Equals(doc2)
		}
	})

	b.Run("OAS2/Medium", func(b *testing.B) {
		result, err := ParseWithOptions(WithFilePath(mediumOAS2Path))
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
		doc1, ok := result.OAS2Document()
		if !ok {
			b.Fatal("expected OAS2 document")
		}
		doc2 := doc1.DeepCopy()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = doc1.Equals(doc2)
		}
	})
}
