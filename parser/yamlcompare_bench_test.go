package parser

// YAML Library Comparison Benchmarks
//
// This file benchmarks go.yaml.in/yaml/v4 (the current parser dependency)
// against github.com/goccy/go-yaml across several key operations:
//
//   - Unmarshal:       Raw YAML bytes -> map[string]any
//   - Marshal:         map[string]any -> YAML bytes
//   - RoundTrip:       Unmarshal then Marshal (full cycle)
//   - NodeParse:       Parse YAML into AST/Node representation
//   - UnmarshalStruct: Unmarshal into a typed Go struct
//   - Decoder:         Streaming decoder (bytes.NewReader-based)
//
// Each category is tested across Small, Medium, and Large OAS fixtures
// to show how performance scales with document size.
//
// Run with:
//   go test -bench=BenchmarkYAMLCompare -benchmem ./parser/

import (
	"bytes"
	"os"
	"testing"

	goccyyaml "github.com/goccy/go-yaml"
	goccyparser "github.com/goccy/go-yaml/parser"
	yamlv4 "go.yaml.in/yaml/v4"
)

// yamlBenchSize pairs a human-readable name with a fixture path.
type yamlBenchSize struct {
	name string
	path string
}

// yamlBenchSizes defines the fixture sizes used across all YAML comparison benchmarks.
// The path constants are defined in parser_bench_test.go.
var yamlBenchSizes = []yamlBenchSize{
	{name: "Small", path: smallOAS3Path},
	{name: "Medium", path: mediumOAS3Path},
	{name: "Large", path: largeOAS3Path},
}

// benchOAS3Doc is a minimal typed struct that mirrors the top-level structure
// of an OAS 3.x document. Used by BenchmarkYAMLCompare_UnmarshalStruct to
// measure typed deserialization performance.
type benchOAS3Doc struct {
	OpenAPI string         `yaml:"openapi"`
	Info    benchOAS3Info  `yaml:"info"`
	Paths   map[string]any `yaml:"paths"`
}

type benchOAS3Info struct {
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

// loadBenchFixture reads a fixture file and fatals on error.
// Intended for benchmark setup only.
func loadBenchFixture(b *testing.B, path string) []byte {
	b.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("Failed to load fixture %s: %v", path, err)
	}
	return data
}

// BenchmarkYAMLCompare_Unmarshal compares unmarshaling raw YAML bytes into map[string]any.
func BenchmarkYAMLCompare_Unmarshal(b *testing.B) {
	for _, size := range yamlBenchSizes {
		data := loadBenchFixture(b, size.path)

		b.Run(size.name+"/yamlv4", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var out map[string]any
				if err := yamlv4.Unmarshal(data, &out); err != nil {
					b.Fatalf("yamlv4 Unmarshal failed: %v", err)
				}
			}
		})

		b.Run(size.name+"/goccy", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var out map[string]any
				if err := goccyyaml.Unmarshal(data, &out); err != nil {
					b.Fatalf("goccy Unmarshal failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkYAMLCompare_Marshal compares marshaling a pre-parsed map[string]any back to YAML bytes.
// Each library marshals data it previously unmarshaled to ensure type compatibility.
func BenchmarkYAMLCompare_Marshal(b *testing.B) {
	for _, size := range yamlBenchSizes {
		data := loadBenchFixture(b, size.path)

		// Pre-parse with yamlv4
		var v4Data map[string]any
		if err := yamlv4.Unmarshal(data, &v4Data); err != nil {
			b.Fatalf("yamlv4 pre-parse failed for %s: %v", size.name, err)
		}

		// Pre-parse with goccy
		var goccyData map[string]any
		if err := goccyyaml.Unmarshal(data, &goccyData); err != nil {
			b.Fatalf("goccy pre-parse failed for %s: %v", size.name, err)
		}

		b.Run(size.name+"/yamlv4", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := yamlv4.Marshal(v4Data); err != nil {
					b.Fatalf("yamlv4 Marshal failed: %v", err)
				}
			}
		})

		b.Run(size.name+"/goccy", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := goccyyaml.Marshal(goccyData); err != nil {
					b.Fatalf("goccy Marshal failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkYAMLCompare_RoundTrip compares a full unmarshal-then-marshal cycle.
func BenchmarkYAMLCompare_RoundTrip(b *testing.B) {
	for _, size := range yamlBenchSizes {
		data := loadBenchFixture(b, size.path)

		b.Run(size.name+"/yamlv4", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var out map[string]any
				if err := yamlv4.Unmarshal(data, &out); err != nil {
					b.Fatalf("yamlv4 Unmarshal failed: %v", err)
				}
				if _, err := yamlv4.Marshal(out); err != nil {
					b.Fatalf("yamlv4 Marshal failed: %v", err)
				}
			}
		})

		b.Run(size.name+"/goccy", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var out map[string]any
				if err := goccyyaml.Unmarshal(data, &out); err != nil {
					b.Fatalf("goccy Unmarshal failed: %v", err)
				}
				if _, err := goccyyaml.Marshal(out); err != nil {
					b.Fatalf("goccy Marshal failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkYAMLCompare_NodeParse compares parsing YAML into AST/Node representation.
//   - yamlv4: Unmarshals into a yaml.Node tree
//   - goccy:  Uses goccyparser.ParseBytes to produce an *ast.File
func BenchmarkYAMLCompare_NodeParse(b *testing.B) {
	for _, size := range yamlBenchSizes {
		data := loadBenchFixture(b, size.path)

		b.Run(size.name+"/yamlv4", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var node yamlv4.Node
				if err := yamlv4.Unmarshal(data, &node); err != nil {
					b.Fatalf("yamlv4 Node unmarshal failed: %v", err)
				}
			}
		})

		b.Run(size.name+"/goccy", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := goccyparser.ParseBytes(data, 0); err != nil {
					b.Fatalf("goccy ParseBytes failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkYAMLCompare_UnmarshalStruct compares unmarshaling into a typed struct.
// Uses benchOAS3Doc which mirrors the top-level OAS 3.x structure.
func BenchmarkYAMLCompare_UnmarshalStruct(b *testing.B) {
	for _, size := range yamlBenchSizes {
		data := loadBenchFixture(b, size.path)

		b.Run(size.name+"/yamlv4", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var doc benchOAS3Doc
				if err := yamlv4.Unmarshal(data, &doc); err != nil {
					b.Fatalf("yamlv4 struct unmarshal failed: %v", err)
				}
			}
		})

		b.Run(size.name+"/goccy", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var doc benchOAS3Doc
				if err := goccyyaml.Unmarshal(data, &doc); err != nil {
					b.Fatalf("goccy struct unmarshal failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkYAMLCompare_Decoder compares streaming decoder performance.
// Both libraries provide a Decoder that reads from an io.Reader.
func BenchmarkYAMLCompare_Decoder(b *testing.B) {
	for _, size := range yamlBenchSizes {
		data := loadBenchFixture(b, size.path)

		b.Run(size.name+"/yamlv4", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var out map[string]any
				if err := yamlv4.NewDecoder(bytes.NewReader(data)).Decode(&out); err != nil {
					b.Fatalf("yamlv4 Decoder failed: %v", err)
				}
			}
		})

		b.Run(size.name+"/goccy", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var out map[string]any
				if err := goccyyaml.NewDecoder(bytes.NewReader(data)).Decode(&out); err != nil {
					b.Fatalf("goccy Decoder failed: %v", err)
				}
			}
		})
	}
}
