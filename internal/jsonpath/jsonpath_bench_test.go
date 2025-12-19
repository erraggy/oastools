package jsonpath

import (
	"testing"
)

// BenchmarkParse benchmarks JSONPath parsing
func BenchmarkParse(b *testing.B) {
	paths := []struct {
		name string
		expr string
	}{
		{"Simple", "$.info.title"},
		{"Bracket", "$.paths['/users'].get"},
		{"Wildcard", "$.paths.*.get.responses"},
		{"Filter", "$.paths.*[?@.deprecated==true]"},
		{"CompoundFilter", "$.paths.*[?@.deprecated==true && @.summary!='']"},
		{"RecursiveDescent", "$..description"},
		{"Complex", "$.paths.*[?@.x-internal==true].*.responses.*.content"},
	}

	for _, tt := range paths {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, err := Parse(tt.expr)
				if err != nil {
					b.Fatalf("Failed to parse: %v", err)
				}
			}
		})
	}
}

// BenchmarkGet benchmarks JSONPath Get operations
func BenchmarkGet(b *testing.B) {
	doc := createBenchmarkDoc()

	paths := []struct {
		name string
		expr string
	}{
		{"Simple", "$.info.title"},
		{"Nested", "$.paths['/users'].get.summary"},
		{"Wildcard", "$.paths.*"},
		{"DeepWildcard", "$.paths.*.*.responses"},
		{"Filter", "$.paths.*[?@.deprecated==true]"},
		{"RecursiveDescent", "$..summary"},
	}

	for _, tt := range paths {
		b.Run(tt.name, func(b *testing.B) {
			path, err := Parse(tt.expr)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			b.ReportAllocs()
			for b.Loop() {
				_ = path.Get(doc)
			}
		})
	}
}

// BenchmarkModify benchmarks JSONPath Modify operations
func BenchmarkModify(b *testing.B) {
	paths := []struct {
		name string
		expr string
	}{
		{"Simple", "$.info.title"},
		{"Wildcard", "$.paths.*"},
		{"Filter", "$.paths.*[?@.deprecated==true]"},
	}

	for _, tt := range paths {
		b.Run(tt.name, func(b *testing.B) {
			path, err := Parse(tt.expr)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			b.ReportAllocs()
			for b.Loop() {
				// Create fresh doc each iteration
				doc := createBenchmarkDoc()
				err := path.Modify(doc, func(v any) any {
					if m, ok := v.(map[string]any); ok {
						m["x-modified"] = true
						return m
					}
					return v
				})
				if err != nil {
					b.Fatalf("Failed to modify: %v", err)
				}
			}
		})
	}
}

// BenchmarkRecursiveDescentGet benchmarks recursive descent operations
func BenchmarkRecursiveDescentGet(b *testing.B) {
	doc := createLargeBenchmarkDoc()

	paths := []struct {
		name string
		expr string
	}{
		{"AllSummary", "$..summary"},
		{"AllDescription", "$..description"},
		{"AllDeprecated", "$..deprecated"},
		{"WildcardDescendant", "$.paths..*"},
	}

	for _, tt := range paths {
		b.Run(tt.name, func(b *testing.B) {
			path, err := Parse(tt.expr)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			b.ReportAllocs()
			for b.Loop() {
				_ = path.Get(doc)
			}
		})
	}
}

// BenchmarkCompoundFilter benchmarks compound filter evaluation
func BenchmarkCompoundFilter(b *testing.B) {
	doc := createBenchmarkDoc()

	paths := []struct {
		name string
		expr string
	}{
		{"SimpleFilter", "$.paths.*[?@.deprecated==true]"},
		{"AndFilter", "$.paths.*[?@.deprecated==true && @.x-internal==false]"},
		{"OrFilter", "$.paths.*[?@.deprecated==true || @.x-internal==true]"},
		{"ChainedAnd", "$.paths.*[?@.deprecated==true && @.x-internal==false && @.x-public==true]"},
	}

	for _, tt := range paths {
		b.Run(tt.name, func(b *testing.B) {
			path, err := Parse(tt.expr)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}

			b.ReportAllocs()
			for b.Loop() {
				_ = path.Get(doc)
			}
		})
	}
}

// createBenchmarkDoc creates a document for benchmarking
func createBenchmarkDoc() map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Benchmark API",
			"version":     "1.0.0",
			"description": "API for benchmarking",
		},
		"paths": map[string]any{
			"/users": map[string]any{
				"deprecated": true,
				"x-internal": false,
				"x-public":   true,
				"get": map[string]any{
					"summary":     "List users",
					"description": "Returns a list of users",
					"responses": map[string]any{
						"200": map[string]any{"description": "OK"},
					},
				},
				"post": map[string]any{
					"summary":     "Create user",
					"description": "Creates a new user",
					"responses": map[string]any{
						"201": map[string]any{"description": "Created"},
					},
				},
			},
			"/admin": map[string]any{
				"deprecated": false,
				"x-internal": true,
				"x-public":   false,
				"get": map[string]any{
					"summary":     "Admin panel",
					"description": "Access admin panel",
					"responses": map[string]any{
						"200": map[string]any{"description": "OK"},
					},
				},
			},
			"/products": map[string]any{
				"deprecated": false,
				"x-internal": false,
				"x-public":   true,
				"get": map[string]any{
					"summary":     "List products",
					"description": "Returns products",
					"responses": map[string]any{
						"200": map[string]any{"description": "OK"},
					},
				},
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"User":    map[string]any{"type": "object", "description": "User schema"},
				"Product": map[string]any{"type": "object", "description": "Product schema"},
			},
		},
	}
}

// createLargeBenchmarkDoc creates a larger document for benchmarking recursive operations
func createLargeBenchmarkDoc() map[string]any {
	paths := make(map[string]any)
	for i := range 50 {
		pathName := "/resource" + string('A'+rune(i%26)) + string('0'+rune(i/26))
		paths[pathName] = map[string]any{
			"deprecated": i%5 == 0,
			"get": map[string]any{
				"summary":     "Get resource",
				"description": "Returns the resource",
				"deprecated":  i%3 == 0,
				"responses": map[string]any{
					"200": map[string]any{"description": "OK", "summary": "Success response"},
					"404": map[string]any{"description": "Not found", "summary": "Error response"},
				},
			},
			"post": map[string]any{
				"summary":     "Create resource",
				"description": "Creates a new resource",
				"deprecated":  i%7 == 0,
				"responses": map[string]any{
					"201": map[string]any{"description": "Created", "summary": "Created response"},
				},
			},
		}
	}

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Large Benchmark API",
			"version":     "1.0.0",
			"description": "Large API for benchmarking",
		},
		"paths": paths,
	}
}
