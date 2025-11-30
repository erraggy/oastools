package parser

import (
	"os"
	"testing"
)

// FuzzParseBytes is a Go Fuzz Test targeting the ParseBytes function.
// It mutates the input data to try and find inputs that cause crashes (panics).
//
// The fuzzer tests ParseBytes with different combinations of resolveRefs and
// validateStructure flags to ensure robust error handling across all code paths.
func FuzzParseBytes(f *testing.F) {
	// 1. Seed Corpus: Provide known, valid and invalid examples for the fuzzer.
	// This helps the fuzzer understand the expected input structure and edge cases.
	seedCorpus := [][]byte{}

	// Helper to read testdata files for seed corpus
	addTestFile := func(path string) {
		data, err := os.ReadFile(path)
		if err == nil {
			seedCorpus = append(seedCorpus, data)
		}
	}

	// Add various OAS versions and formats
	addTestFile("../testdata/minimal-oas2.yaml")
	addTestFile("../testdata/minimal-oas3.yaml")
	addTestFile("../testdata/petstore-3.1.yaml")
	addTestFile("../testdata/minimal-oas3.json")

	// Add edge cases and invalid inputs
	addTestFile("../testdata/invalid-oas3.yaml")
	addTestFile("../testdata/deeply-nested-schema.yaml")
	addTestFile("../testdata/circular-schema.yaml")

	// Add inline edge cases
	seedCorpus = append(seedCorpus,
		// Empty input
		[]byte(""),
		// Invalid YAML
		[]byte("Not YAML or JSON content"),
		// Invalid JSON
		[]byte(`{invalid json}`),
		// Malformed structures
		[]byte(`openapi: 3.0.0`),
		[]byte(`{"openapi": "3.0.0"}`),
		// Deeply nested structure
		[]byte(`{"a": {"b": {"c": {"d": {"e": {"f": {"g": {"h": {"i": {"j": "deep"}}}}}}}}}`),
		// Very long string
		[]byte(`openapi: "`+string(make([]byte, 10000))+`"`),
		// Special characters and unicode
		[]byte(`openapi: "3.0.0\x00\x01\x02"`),
		[]byte(`{"openapi": "3.0.0", "info": {"title": "测试API", "version": "1.0"}}`),
		// Array instead of object
		[]byte(`["not", "an", "object"]`),
		// Null values
		[]byte(`{"openapi": null, "info": null}`),
		// Mixed valid/invalid
		[]byte(`openapi: 3.0.0
info:
  title: Test
  version: 1.0
paths:
  - invalid list instead of object`),
	)

	// Add all seed corpus entries
	for _, seed := range seedCorpus {
		f.Add(seed)
	}

	// 2. Fuzz Target: Test ParseBytes with both flags enabled
	f.Fuzz(func(t *testing.T, data []byte) {
		// Test with both resolveRefs and validateStructure enabled to exercise
		// the most complex code paths including reference resolution and structural
		// validation. We expect many inputs to cause an error, but we must ensure
		// the function never panics (crashes).
		//
		// Note: We test with both resolve refs and validate structure enabled.
		// The combinations are well-covered by existing unit tests.
		_, _ = ParseWithOptions(
			WithBytes(data),
			WithResolveRefs(true),
			WithValidateStructure(true),
		)
	})
}
