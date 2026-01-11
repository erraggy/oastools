package parser

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// assertKeyOrder verifies that keys appear in the expected order within the output string.
func assertKeyOrder(t *testing.T, output string, keys []string, format string) {
	t.Helper()
	if len(keys) < 2 {
		return
	}
	for i := 0; i < len(keys)-1; i++ {
		idx1 := strings.Index(output, keys[i])
		idx2 := strings.Index(output, keys[i+1])
		assert.True(t, idx1 < idx2, "%s: expected %q before %q", format, keys[i], keys[i+1])
	}
}

func TestMarshalOrderedJSON(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantErr        bool
		checkOrder     []string // keys in expected order (use index comparison)
		checkPathOrder []string // paths in expected order
	}{
		{
			name: "preserves extension field order",
			input: `{
				"openapi": "3.1.0",
				"info": {
					"title": "Order Test",
					"version": "1.0.0",
					"x-zebra": "should come first",
					"x-alpha": "should come second"
				},
				"paths": {}
			}`,
			checkOrder: []string{"x-zebra", "x-alpha"},
		},
		{
			name: "preserves path order",
			input: `{
				"openapi": "3.1.0",
				"info": {"title": "Test", "version": "1.0.0"},
				"paths": {
					"/zebra": {"summary": "Z endpoint"},
					"/alpha": {"summary": "A endpoint"},
					"/middle": {"summary": "M endpoint"}
				}
			}`,
			checkPathOrder: []string{"/zebra", "/alpha", "/middle"},
		},
		{
			name: "handles nested objects",
			input: `{
				"openapi": "3.1.0",
				"info": {"title": "Test", "version": "1.0.0"},
				"paths": {
					"/users": {
						"get": {
							"responses": {
								"200": {"description": "OK"},
								"404": {"description": "Not Found"}
							}
						}
					}
				}
			}`,
		},
		{
			name: "handles arrays",
			input: `{
				"openapi": "3.1.0",
				"info": {"title": "Test", "version": "1.0.0"},
				"servers": [
					{"url": "https://api.example.com"},
					{"url": "https://staging.example.com"}
				],
				"paths": {}
			}`,
		},
		{
			name: "handles empty objects",
			input: `{
				"openapi": "3.1.0",
				"info": {"title": "Test", "version": "1.0.0"},
				"paths": {},
				"components": {}
			}`,
		},
		{
			name: "handles empty arrays",
			input: `{
				"openapi": "3.1.0",
				"info": {"title": "Test", "version": "1.0.0"},
				"tags": [],
				"paths": {}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseWithOptions(
				WithBytes([]byte(tt.input)),
				WithPreserveOrder(true),
			)
			require.NoError(t, err)
			assert.True(t, result.HasPreservedOrder())

			output, err := result.MarshalOrderedJSON()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify valid JSON
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(output, &decoded))

			outputStr := string(output)
			assertKeyOrder(t, outputStr, tt.checkOrder, "key order")
			assertKeyOrder(t, outputStr, tt.checkPathOrder, "path order")
		})
	}
}

func TestMarshalOrderedYAML(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantErr        bool
		checkOrder     []string // keys in expected order
		checkPathOrder []string // paths in expected order
	}{
		{
			name: "preserves extension field order",
			input: `openapi: "3.1.0"
info:
  title: Order Test
  version: "1.0.0"
  x-zebra: first
  x-alpha: second
paths: {}
`,
			checkOrder: []string{"x-zebra", "x-alpha"},
		},
		{
			name: "preserves path order",
			input: `openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"
paths:
  /zebra:
    summary: Z endpoint
  /alpha:
    summary: A endpoint
`,
			checkPathOrder: []string{"/zebra", "/alpha"},
		},
		{
			name: "handles nested structures",
			input: `openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`,
		},
		{
			name: "handles arrays",
			input: `openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"
servers:
  - url: https://api.example.com
  - url: https://staging.example.com
paths: {}
`,
		},
		{
			name: "handles empty maps",
			input: `openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"
paths: {}
components: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseWithOptions(
				WithBytes([]byte(tt.input)),
				WithPreserveOrder(true),
			)
			require.NoError(t, err)

			output, err := result.MarshalOrderedYAML()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify valid YAML
			var decoded map[string]any
			require.NoError(t, yaml.Unmarshal(output, &decoded))

			outputStr := string(output)
			assertKeyOrder(t, outputStr, tt.checkOrder, "key order")
			assertKeyOrder(t, outputStr, tt.checkPathOrder, "path order")
		})
	}
}

func TestMarshalOrderedJSONIndent(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		prefix      string
		indent      string
		checkDecode bool // Some prefix options don't produce valid JSON
	}{
		{
			name:        "standard two-space indent",
			input:       `{"openapi": "3.1.0", "info": {"title": "Test", "version": "1.0.0"}, "paths": {}}`,
			prefix:      "",
			indent:      "  ",
			checkDecode: true,
		},
		{
			name:        "tab indent",
			input:       `{"openapi": "3.1.0", "info": {"title": "Test", "version": "1.0.0"}, "paths": {}}`,
			prefix:      "",
			indent:      "\t",
			checkDecode: true,
		},
		{
			name:        "with prefix (decorative, not valid JSON)",
			input:       `{"openapi": "3.1.0", "info": {"title": "Test", "version": "1.0.0"}, "paths": {}}`,
			prefix:      "// ",
			indent:      "  ",
			checkDecode: false, // prefix makes it invalid JSON
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseWithOptions(
				WithBytes([]byte(tt.input)),
				WithPreserveOrder(true),
			)
			require.NoError(t, err)

			output, err := result.MarshalOrderedJSONIndent(tt.prefix, tt.indent)
			require.NoError(t, err)

			// Verify it's properly indented (contains newlines)
			assert.Contains(t, string(output), "\n")

			// Verify it's valid JSON (if expected)
			if tt.checkDecode {
				var decoded map[string]any
				require.NoError(t, json.Unmarshal(output, &decoded))
			}
		})
	}
}

func TestMarshalOrderedFallback(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "JSON without order preservation",
			input: `{"openapi": "3.1.0", "info": {"title": "Test", "version": "1.0.0"}, "paths": {}}`,
		},
		{
			name: "YAML without order preservation",
			input: `openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"
paths: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse WITHOUT order preservation
			result, err := ParseWithOptions(
				WithBytes([]byte(tt.input)),
			)
			require.NoError(t, err)
			assert.False(t, result.HasPreservedOrder())

			// MarshalOrderedJSON should fall back to standard marshal
			jsonOutput, err := result.MarshalOrderedJSON()
			require.NoError(t, err)

			var jsonDecoded map[string]any
			require.NoError(t, json.Unmarshal(jsonOutput, &jsonDecoded))

			// MarshalOrderedYAML should fall back to standard marshal
			yamlOutput, err := result.MarshalOrderedYAML()
			require.NoError(t, err)

			var yamlDecoded map[string]any
			require.NoError(t, yaml.Unmarshal(yamlOutput, &yamlDecoded))
		})
	}
}

func TestOrderedMarshalRoundtrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "JSON roundtrip",
			input: `{
  "openapi": "3.1.0",
  "info": {
    "title": "Identity Test",
    "version": "1.0.0"
  },
  "paths": {
    "/users": {
      "get": {
        "responses": {
          "200": {"description": "OK"}
        }
      }
    }
  }
}`,
		},
		{
			name: "complex document",
			input: `{
  "openapi": "3.1.0",
  "info": {
    "title": "Data Integrity Test",
    "version": "2.0.0",
    "description": "Testing data preservation",
    "x-custom": {"nested": "value"}
  },
  "paths": {
    "/test": {
      "get": {
        "summary": "Test endpoint",
        "operationId": "testOp",
        "responses": {
          "200": {"description": "Success"},
          "404": {"description": "Not found"}
        }
      }
    }
  },
  "components": {
    "schemas": {
      "TestSchema": {
        "type": "object",
        "properties": {
          "id": {"type": "string"},
          "name": {"type": "string"}
        }
      }
    }
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseWithOptions(
				WithBytes([]byte(tt.input)),
				WithPreserveOrder(true),
			)
			require.NoError(t, err)

			output, err := result.MarshalOrderedJSONIndent("", "  ")
			require.NoError(t, err)

			// Parse both input and output for semantic comparison
			var inputData, outputData map[string]any
			require.NoError(t, json.Unmarshal([]byte(tt.input), &inputData))
			require.NoError(t, json.Unmarshal(output, &outputData))

			assert.Equal(t, inputData, outputData, "roundtrip should preserve all data")
		})
	}
}

func TestOrderedMarshalHashStability(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		iterations int
	}{
		{
			name: "JSON stability",
			input: `{
				"openapi": "3.1.0",
				"info": {"title": "Hash Test", "version": "1.0.0"},
				"paths": {"/a": {}, "/b": {}, "/c": {}}
			}`,
			iterations: 10,
		},
		{
			name: "YAML stability",
			input: `openapi: "3.1.0"
info:
  title: Hash Test
  version: "1.0.0"
paths:
  /a: {}
  /b: {}
  /c: {}
`,
			iterations: 10,
		},
		{
			name: "complex document stability",
			input: `{
				"openapi": "3.1.0",
				"info": {
					"title": "Complex Test",
					"version": "1.0.0",
					"x-custom-a": "a",
					"x-custom-b": "b"
				},
				"paths": {
					"/users": {"get": {"responses": {"200": {"description": "OK"}}}},
					"/orders": {"post": {"responses": {"201": {"description": "Created"}}}}
				},
				"components": {
					"schemas": {
						"User": {"type": "object"},
						"Order": {"type": "object"}
					}
				}
			}`,
			iterations: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputs := make([][]byte, 0, tt.iterations)

			for i := range tt.iterations {
				result, err := ParseWithOptions(
					WithBytes([]byte(tt.input)),
					WithPreserveOrder(true),
				)
				require.NoError(t, err, "parse iteration %d failed", i)

				output, err := result.MarshalOrderedJSON()
				require.NoError(t, err, "marshal iteration %d failed", i)
				outputs = append(outputs, output)
			}

			// All outputs should be byte-for-byte identical
			reference := outputs[0]
			for i := 1; i < len(outputs); i++ {
				assert.True(t, bytes.Equal(reference, outputs[i]),
					"iteration %d differs from reference", i)
			}
		})
	}
}

func TestOrderedMarshalEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "null values",
			input: `{
				"openapi": "3.1.0",
				"info": {"title": "Test", "version": "1.0.0", "description": null},
				"paths": {}
			}`,
		},
		{
			name: "boolean values",
			input: `{
				"openapi": "3.1.0",
				"info": {"title": "Test", "version": "1.0.0"},
				"paths": {},
				"x-deprecated": true,
				"x-active": false
			}`,
		},
		{
			name: "numeric values",
			input: `{
				"openapi": "3.1.0",
				"info": {"title": "Test", "version": "1.0.0"},
				"paths": {},
				"x-count": 42,
				"x-rate": 3.14
			}`,
		},
		{
			name: "deeply nested",
			input: `{
				"openapi": "3.1.0",
				"info": {"title": "Test", "version": "1.0.0"},
				"paths": {
					"/deep": {
						"get": {
							"responses": {
								"200": {
									"content": {
										"application/json": {
											"schema": {
												"properties": {
													"level1": {
														"properties": {
															"level2": {"type": "string"}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}`,
		},
		{
			name: "empty string values",
			input: `{
				"openapi": "3.1.0",
				"info": {"title": "", "version": "1.0.0"},
				"paths": {}
			}`,
		},
		{
			name: "special characters in keys",
			input: `{
				"openapi": "3.1.0",
				"info": {"title": "Test", "version": "1.0.0"},
				"paths": {
					"/users/{user-id}/posts/{post_id}": {}
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseWithOptions(
				WithBytes([]byte(tt.input)),
				WithPreserveOrder(true),
			)
			require.NoError(t, err)

			jsonOutput, err := result.MarshalOrderedJSON()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify valid JSON
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(jsonOutput, &decoded))

			// Verify YAML also works
			yamlOutput, err := result.MarshalOrderedYAML()
			require.NoError(t, err)

			var yamlDecoded map[string]any
			require.NoError(t, yaml.Unmarshal(yamlOutput, &yamlDecoded))
		})
	}
}

func TestWithPreserveOrderOption(t *testing.T) {
	tests := []struct {
		name          string
		preserveOrder bool
		wantPreserved bool
	}{
		{
			name:          "enabled",
			preserveOrder: true,
			wantPreserved: true,
		},
		{
			name:          "disabled",
			preserveOrder: false,
			wantPreserved: false,
		},
	}

	input := `{"openapi": "3.1.0", "info": {"title": "Test", "version": "1.0.0"}, "paths": {}}`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseWithOptions(
				WithBytes([]byte(input)),
				WithPreserveOrder(tt.preserveOrder),
			)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPreserved, result.HasPreservedOrder())
		})
	}
}

func TestOrderedMarshalWithSourceMap(t *testing.T) {
	input := `openapi: "3.1.0"
info:
  title: Combined Test
  version: "1.0.0"
paths:
  /test: {}
`

	result, err := ParseWithOptions(
		WithBytes([]byte(input)),
		WithPreserveOrder(true),
		WithSourceMap(true),
	)
	require.NoError(t, err)

	// Both features should work
	assert.True(t, result.HasPreservedOrder())
	assert.NotNil(t, result.SourceMap)

	// Marshal should still work
	output, err := result.MarshalOrderedJSON()
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(output, &decoded))
}

func TestOrderedMarshalCrossFormat(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		inputIsYAML    bool
		checkOrder     []string
		checkPathOrder []string
	}{
		{
			name: "YAML to JSON preserves order",
			input: `openapi: "3.1.0"
info:
  title: YAML to JSON
  version: "1.0.0"
  x-last: should be last
  x-first: should be first
paths:
  /second:
    summary: Second path
  /first:
    summary: First path
`,
			inputIsYAML:    true,
			checkOrder:     []string{"x-last", "x-first"},
			checkPathOrder: []string{"/second", "/first"},
		},
		{
			name: "JSON to YAML preserves order",
			input: `{
				"openapi": "3.1.0",
				"info": {
					"title": "JSON to YAML",
					"version": "1.0.0",
					"x-last": "last",
					"x-first": "first"
				},
				"paths": {
					"/second": {},
					"/first": {}
				}
			}`,
			inputIsYAML:    false,
			checkOrder:     []string{"x-last", "x-first"},
			checkPathOrder: []string{"/second", "/first"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseWithOptions(
				WithBytes([]byte(tt.input)),
				WithPreserveOrder(true),
			)
			require.NoError(t, err)

			// Marshal to JSON
			jsonOutput, err := result.MarshalOrderedJSON()
			require.NoError(t, err)

			var jsonDecoded map[string]any
			require.NoError(t, json.Unmarshal(jsonOutput, &jsonDecoded))

			// Marshal to YAML
			yamlOutput, err := result.MarshalOrderedYAML()
			require.NoError(t, err)

			var yamlDecoded map[string]any
			require.NoError(t, yaml.Unmarshal(yamlOutput, &yamlDecoded))

			// Check order in JSON output
			jsonStr := string(jsonOutput)
			assertKeyOrder(t, jsonStr, tt.checkOrder, "JSON key order")
			assertKeyOrder(t, jsonStr, tt.checkPathOrder, "JSON path order")

			// Check order in YAML output
			yamlStr := string(yamlOutput)
			assertKeyOrder(t, yamlStr, tt.checkOrder, "YAML key order")
		})
	}
}

func TestOrderedMarshalOAS2(t *testing.T) {
	input := `{
		"swagger": "2.0",
		"info": {"title": "OAS2 Test", "version": "1.0.0"},
		"paths": {
			"/zebra": {"get": {"responses": {"200": {"description": "OK"}}}},
			"/alpha": {"get": {"responses": {"200": {"description": "OK"}}}}
		}
	}`

	result, err := ParseWithOptions(
		WithBytes([]byte(input)),
		WithPreserveOrder(true),
	)
	require.NoError(t, err)

	output, err := result.MarshalOrderedJSON()
	require.NoError(t, err)

	outputStr := string(output)
	zebraIdx := strings.Index(outputStr, "/zebra")
	alphaIdx := strings.Index(outputStr, "/alpha")
	assert.True(t, zebraIdx < alphaIdx, "OAS2 path order not preserved")
}

func TestMarshalOrderedYAMLRoundtrip(t *testing.T) {
	input := `openapi: "3.1.0"
info:
  title: YAML Roundtrip
  version: "1.0.0"
paths:
  /second:
    summary: Second
  /first:
    summary: First
`

	result, err := ParseWithOptions(
		WithBytes([]byte(input)),
		WithPreserveOrder(true),
	)
	require.NoError(t, err)

	output, err := result.MarshalOrderedYAML()
	require.NoError(t, err)

	// Parse both for semantic comparison
	var inputData, outputData map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(input), &inputData))
	require.NoError(t, yaml.Unmarshal(output, &outputData))

	assert.Equal(t, inputData, outputData, "YAML roundtrip should preserve all data")

	// Order check
	outputStr := string(output)
	secondIdx := strings.Index(outputStr, "/second")
	firstIdx := strings.Index(outputStr, "/first")
	assert.True(t, secondIdx < firstIdx, "YAML order not preserved")
}

// TestMergeKeyOrder tests the mergeKeyOrder helper function.
func TestMergeKeyOrder(t *testing.T) {
	tests := []struct {
		name       string
		sourceKeys []string
		dataKeys   []string
		want       []string
	}{
		{
			name:       "same keys",
			sourceKeys: []string{"a", "b", "c"},
			dataKeys:   []string{"a", "b", "c"},
			want:       []string{"a", "b", "c"},
		},
		{
			name:       "extra keys in data sorted",
			sourceKeys: []string{"b", "a"},
			dataKeys:   []string{"a", "b", "z", "m"},
			want:       []string{"b", "a", "m", "z"},
		},
		{
			name:       "empty source keys",
			sourceKeys: []string{},
			dataKeys:   []string{"c", "a", "b"},
			want:       []string{"a", "b", "c"},
		},
		{
			name:       "empty data keys",
			sourceKeys: []string{"a", "b", "c"},
			dataKeys:   []string{},
			want:       []string{"a", "b", "c"},
		},
		{
			name:       "both empty",
			sourceKeys: []string{},
			dataKeys:   []string{},
			want:       []string{},
		},
		{
			name:       "nil source keys",
			sourceKeys: nil,
			dataKeys:   []string{"b", "a"},
			want:       []string{"a", "b"},
		},
		{
			name:       "nil data keys",
			sourceKeys: []string{"z", "a"},
			dataKeys:   nil,
			want:       []string{"z", "a"},
		},
		{
			name:       "keys removed from data",
			sourceKeys: []string{"a", "b", "c"},
			dataKeys:   []string{"a", "c"},
			want:       []string{"a", "b", "c"}, // b still in order but won't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeKeyOrder(tt.sourceKeys, tt.dataKeys)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBuildNodeIndex tests the buildNodeIndex helper function.
func TestBuildNodeIndex(t *testing.T) {
	tests := []struct {
		name     string
		node     *yaml.Node
		wantKeys []string
		wantNil  bool
	}{
		{
			name:    "nil node",
			node:    nil,
			wantNil: true,
		},
		{
			name: "non-mapping node",
			node: &yaml.Node{
				Kind: yaml.SequenceNode,
			},
			wantNil: true,
		},
		{
			name: "empty mapping",
			node: &yaml.Node{
				Kind:    yaml.MappingNode,
				Content: []*yaml.Node{},
			},
			wantKeys: []string{},
		},
		{
			name: "mapping with keys",
			node: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "key1"},
					{Kind: yaml.ScalarNode, Value: "value1"},
					{Kind: yaml.ScalarNode, Value: "key2"},
					{Kind: yaml.ScalarNode, Value: "value2"},
				},
			},
			wantKeys: []string{"key1", "key2"},
		},
		{
			name: "mapping with odd content count",
			node: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "key1"},
					{Kind: yaml.ScalarNode, Value: "value1"},
					{Kind: yaml.ScalarNode, Value: "key2"},
					// Missing value
				},
			},
			wantKeys: []string{"key1"}, // Only complete pairs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := buildNodeIndex(tt.node)

			if tt.wantNil {
				assert.Nil(t, idx)
				return
			}

			require.NotNil(t, idx)
			assert.Len(t, idx, len(tt.wantKeys))
			for _, key := range tt.wantKeys {
				assert.Contains(t, idx, key)
			}
		})
	}
}

// TestValueToNode tests the valueToNode helper function.
func TestValueToNode(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		wantKind  yaml.Kind
		wantTag   string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "nil value",
			value:     nil,
			wantKind:  yaml.ScalarNode,
			wantTag:   "!!null",
			wantValue: "null",
		},
		{
			name:      "bool true",
			value:     true,
			wantKind:  yaml.ScalarNode,
			wantTag:   "!!bool",
			wantValue: "true",
		},
		{
			name:      "bool false",
			value:     false,
			wantKind:  yaml.ScalarNode,
			wantTag:   "!!bool",
			wantValue: "false",
		},
		{
			name:      "int",
			value:     42,
			wantKind:  yaml.ScalarNode,
			wantTag:   "!!int",
			wantValue: "42",
		},
		{
			name:      "int64",
			value:     int64(9223372036854775807),
			wantKind:  yaml.ScalarNode,
			wantTag:   "!!int",
			wantValue: "9223372036854775807",
		},
		{
			name:      "float64",
			value:     3.14159,
			wantKind:  yaml.ScalarNode,
			wantTag:   "!!float",
			wantValue: "3.14159",
		},
		{
			name:      "string",
			value:     "hello",
			wantKind:  yaml.ScalarNode,
			wantTag:   "!!str",
			wantValue: "hello",
		},
		{
			name:      "empty string",
			value:     "",
			wantKind:  yaml.ScalarNode,
			wantTag:   "!!str",
			wantValue: "",
		},
		{
			name:     "empty slice",
			value:    []any{},
			wantKind: yaml.SequenceNode,
		},
		{
			name:     "slice with values",
			value:    []any{"a", "b"},
			wantKind: yaml.SequenceNode,
		},
		{
			name:     "empty map",
			value:    map[string]any{},
			wantKind: yaml.MappingNode,
		},
		{
			name:     "map with values",
			value:    map[string]any{"key": "value"},
			wantKind: yaml.MappingNode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := valueToNode(tt.value)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, node)

			assert.Equal(t, tt.wantKind, node.Kind)

			if tt.wantTag != "" {
				assert.Equal(t, tt.wantTag, node.Tag)
			}

			if tt.wantValue != "" {
				assert.Equal(t, tt.wantValue, node.Value)
			}
		})
	}
}

// TestValueToNodeUnknownType tests valueToNode with types that fall through to JSON marshal.
func TestValueToNodeUnknownType(t *testing.T) {
	// Custom struct should be marshaled via JSON path
	type CustomStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	node, err := valueToNode(CustomStruct{Name: "test", Value: 42})
	require.NoError(t, err)
	require.NotNil(t, node)

	// Should result in a mapping node after JSON roundtrip
	assert.Equal(t, yaml.MappingNode, node.Kind)
}

// TestBuildOrderedNode tests the buildOrderedNode function with various inputs.
func TestBuildOrderedNode(t *testing.T) {
	tests := []struct {
		name       string
		sourceNode *yaml.Node
		data       any
		wantKind   yaml.Kind
		wantErr    bool
	}{
		{
			name:       "nil source node",
			sourceNode: nil,
			data:       map[string]any{"key": "value"},
			wantKind:   yaml.MappingNode,
		},
		{
			name: "document node",
			sourceNode: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Content: []*yaml.Node{
							{Kind: yaml.ScalarNode, Value: "key"},
							{Kind: yaml.ScalarNode, Value: "value"},
						},
					},
				},
			},
			data:     map[string]any{"key": "value"},
			wantKind: yaml.DocumentNode,
		},
		{
			name: "empty document node",
			sourceNode: &yaml.Node{
				Kind:    yaml.DocumentNode,
				Content: []*yaml.Node{},
			},
			data:     "scalar",
			wantKind: yaml.ScalarNode,
		},
		{
			name: "mapping node with type mismatch",
			sourceNode: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "key"},
					{Kind: yaml.ScalarNode, Value: "value"},
				},
			},
			data:     "not a map", // type mismatch
			wantKind: yaml.ScalarNode,
		},
		{
			name: "sequence node with type mismatch",
			sourceNode: &yaml.Node{
				Kind:    yaml.SequenceNode,
				Content: []*yaml.Node{},
			},
			data:     "not a slice", // type mismatch
			wantKind: yaml.ScalarNode,
		},
		{
			name: "sequence node",
			sourceNode: &yaml.Node{
				Kind: yaml.SequenceNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "item1"},
				},
			},
			data:     []any{"item1", "item2"},
			wantKind: yaml.SequenceNode,
		},
		{
			name: "scalar node",
			sourceNode: &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "original",
			},
			data:     "new value",
			wantKind: yaml.ScalarNode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := buildOrderedNode(tt.sourceNode, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, node)
			assert.Equal(t, tt.wantKind, node.Kind)
		})
	}
}

// TestMarshalNodeAsJSONTypeMismatch tests marshalNodeAsJSON when data doesn't match node structure.
func TestMarshalNodeAsJSONTypeMismatch(t *testing.T) {
	tests := []struct {
		name string
		node *yaml.Node
		data any
	}{
		{
			name: "mapping node with non-map data",
			node: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "key"},
					{Kind: yaml.ScalarNode, Value: "value"},
				},
			},
			data: "string data",
		},
		{
			name: "sequence node with non-slice data",
			node: &yaml.Node{
				Kind:    yaml.SequenceNode,
				Content: []*yaml.Node{},
			},
			data: map[string]any{"key": "value"},
		},
		{
			name: "nil node",
			node: nil,
			data: map[string]any{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := marshalNodeAsJSON(&buf, tt.node, tt.data)
			require.NoError(t, err)

			// Should fall back to standard JSON marshal
			output := buf.Bytes()
			require.NotEmpty(t, output)

			// Verify it's valid JSON
			var decoded any
			require.NoError(t, json.Unmarshal(output, &decoded))
		})
	}
}

// TestExtractKeyOrder tests the extractKeyOrder helper function.
func TestExtractKeyOrder(t *testing.T) {
	tests := []struct {
		name string
		node *yaml.Node
		want []string
	}{
		{
			name: "non-mapping node",
			node: &yaml.Node{
				Kind: yaml.SequenceNode,
			},
			want: nil,
		},
		{
			name: "empty mapping",
			node: &yaml.Node{
				Kind:    yaml.MappingNode,
				Content: []*yaml.Node{},
			},
			want: []string{},
		},
		{
			name: "mapping with keys",
			node: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "first"},
					{Kind: yaml.ScalarNode, Value: "value1"},
					{Kind: yaml.ScalarNode, Value: "second"},
					{Kind: yaml.ScalarNode, Value: "value2"},
				},
			},
			want: []string{"first", "second"},
		},
		{
			name: "mapping with non-scalar key",
			node: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.MappingNode}, // non-scalar key
					{Kind: yaml.ScalarNode, Value: "value"},
					{Kind: yaml.ScalarNode, Value: "scalar-key"},
					{Kind: yaml.ScalarNode, Value: "value2"},
				},
			},
			want: []string{"scalar-key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractKeyOrder(tt.node)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestMarshalDeterminism tests that marshaling is deterministic.
// This is consolidated from roundtrip_determinism_test.go.
func TestMarshalDeterminism(t *testing.T) {
	doc := &OAS3Document{
		OpenAPI: "3.1.0",
		Info: &Info{
			Title:       "Determinism Test API",
			Description: "Testing roundtrip determinism",
			Version:     "1.0.0",
			Extra: map[string]any{
				"x-custom-a": "value-a",
				"x-custom-b": "value-b",
				"x-custom-c": "value-c",
			},
		},
		Paths: Paths{
			"/users":    {Summary: "Users"},
			"/orders":   {Summary: "Orders"},
			"/products": {Summary: "Products"},
		},
		Components: &Components{
			Schemas: map[string]*Schema{
				"User":    {Type: "object"},
				"Order":   {Type: "object"},
				"Product": {Type: "object"},
			},
		},
	}

	const iterations = 50
	outputs := make([][]byte, iterations)

	for i := range iterations {
		data, err := json.Marshal(doc)
		require.NoError(t, err, "marshal iteration %d failed", i)
		outputs[i] = data
	}

	reference := outputs[0]
	for i := 1; i < iterations; i++ {
		assert.True(t, bytes.Equal(reference, outputs[i]),
			"marshal is non-deterministic at iteration %d", i)
	}
}

// TestRoundtripDeterminism tests parse->marshal roundtrip determinism.
func TestRoundtripDeterminism(t *testing.T) {
	inputJSON := `{
		"openapi": "3.1.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0",
			"x-ext-a": "a",
			"x-ext-b": "b",
			"x-ext-c": "c"
		},
		"paths": {
			"/a": {"get": {"responses": {"200": {"description": "OK"}}}},
			"/b": {"get": {"responses": {"200": {"description": "OK"}}}},
			"/c": {"get": {"responses": {"200": {"description": "OK"}}}}
		},
		"components": {
			"schemas": {
				"A": {"type": "object"},
				"B": {"type": "object"},
				"C": {"type": "object"}
			}
		}
	}`

	const iterations = 50
	outputs := make([][]byte, iterations)

	for i := range iterations {
		p := New()
		result, err := p.ParseBytes([]byte(inputJSON))
		require.NoError(t, err, "parse iteration %d failed", i)

		doc := result.Document.(*OAS3Document)
		data, err := json.Marshal(doc)
		require.NoError(t, err, "marshal iteration %d failed", i)
		outputs[i] = data
	}

	reference := outputs[0]
	for i := 1; i < iterations; i++ {
		assert.True(t, bytes.Equal(reference, outputs[i]),
			"roundtrip is non-deterministic at iteration %d", i)
	}
}
