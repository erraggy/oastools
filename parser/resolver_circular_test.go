package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireMapKey extracts a map[string]any value from parent[key], failing
// the test with a clear message if the key is missing or has the wrong type.
// This prevents bare type-assertion panics deep inside nested map navigation.
func requireMapKey(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()
	v, ok := parent[key]
	require.True(t, ok, "expected key %q to exist", key)
	m, ok := v.(map[string]any)
	require.True(t, ok, "expected %q to be map[string]any, got %T", key, v)
	return m
}

// assertCircularResolutionBehavior checks the common assertions shared by the
// YAML and JSON circular-ref tests: Node schema resolution, circular ref
// preservation one level deeper, and response schema inlining.
func assertCircularResolutionBehavior(t *testing.T, data map[string]any) {
	t.Helper()

	// --- Node schema assertions ---
	components := requireMapKey(t, data, "components")
	schemas := requireMapKey(t, components, "schemas")
	node := requireMapKey(t, schemas, "Node")

	// Non-circular: Node itself should have type: object resolved inline
	assert.Equal(t, "object", node["type"], "Node.type should be 'object' (non-circular, resolved)")

	// Non-circular: Node.properties.value should have type: string
	properties := requireMapKey(t, node, "properties")
	value := requireMapKey(t, properties, "value")
	assert.Equal(t, "string", value["type"], "Node.properties.value.type should be 'string' (non-circular, resolved)")

	// The resolver resolves the first encounter of $ref: "#/components/schemas/Node"
	// (in Node.properties.next) by inlining the Node content. The circular detection
	// kicks in one level deeper: Node.properties.next.properties.next still has $ref
	// because at that point "#/components/schemas/Node" is in the resolving stack.
	next := requireMapKey(t, properties, "next")
	assert.Equal(t, "object", next["type"],
		"Node.properties.next should be resolved to Node content (type: object)")

	nextProps := requireMapKey(t, next, "properties")
	nextNext := requireMapKey(t, nextProps, "next")
	circularRef, hasRef := nextNext["$ref"]
	assert.True(t, hasRef,
		"Node.properties.next.properties.next should still have $ref (circular, left unresolved)")
	assert.Equal(t, "#/components/schemas/Node", circularRef,
		"circular $ref should point to #/components/schemas/Node")

	// --- Response schema assertions ---
	// paths./test.get.responses.200.content.application/json.schema (8 levels deep)
	paths := requireMapKey(t, data, "paths")
	testPath := requireMapKey(t, paths, "/test")
	get := requireMapKey(t, testPath, "get")
	responses := requireMapKey(t, get, "responses")
	resp200 := requireMapKey(t, responses, "200")
	content := requireMapKey(t, resp200, "content")
	appJSON := requireMapKey(t, content, "application/json")
	responseSchema := requireMapKey(t, appJSON, "schema")

	// The $ref to Node should have been inlined -- the response schema should
	// contain the resolved content (type: object), not a bare $ref.
	assert.Equal(t, "object", responseSchema["type"],
		"response schema should have type 'object' from resolved Node (not a bare $ref)")
	_, hasResponseRef := responseSchema["$ref"]
	assert.False(t, hasResponseRef,
		"response schema should not have a $ref key -- the non-circular ref to Node should be fully resolved")
}

// TestResolveRefs_CircularRefsPreservesNonCircularResolution verifies that when a spec
// contains circular references, non-circular references are still fully resolved.
// This is a regression test for the bug where hasCircularRefs caused the parser to
// skip re-marshaling, discarding all resolution work.
func TestResolveRefs_CircularRefsPreservesNonCircularResolution(t *testing.T) {
	p := New()
	p.ResolveRefs = true

	result, err := p.Parse(filepath.Join("..", "testdata", "circular-schema.yaml"))
	require.NoError(t, err, "Parse should succeed for circular-schema.yaml")

	data := result.Data
	require.NotNil(t, data, "result.Data should not be nil")

	assertCircularResolutionBehavior(t, data)
}

// TestResolveRefs_CircularRefsWarningMessage verifies that the parser emits
// the expected warning string when circular references are detected.
func TestResolveRefs_CircularRefsWarningMessage(t *testing.T) {
	p := New()
	p.ResolveRefs = true

	result, err := p.Parse(filepath.Join("..", "testdata", "circular-schema.yaml"))
	require.NoError(t, err, "Parse should succeed")

	expectedWarning := "Circular references detected. Non-circular references resolved normally. Circular references remain as $ref pointers."
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, expectedWarning) {
			found = true
			break
		}
	}
	assert.True(t, found,
		"expected warning containing %q, got warnings: %v", expectedWarning, result.Warnings)
}

// TestResolveRefs_CircularRefsJSONFastPath verifies that circular ref resolution
// works correctly when the input is JSON (triggering the JSON fast path in the parser).
func TestResolveRefs_CircularRefsJSONFastPath(t *testing.T) {
	// Build the same circular schema as circular-schema.yaml but in JSON format.
	circularJSON := map[string]any{
		"openapi": "3.0.0",
		"info": map[string]any{
			"title":   "Circular Schema API",
			"version": "1.0.0",
		},
		"paths": map[string]any{
			"/test": map[string]any{
				"get": map[string]any{
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Success",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"$ref": "#/components/schemas/Node",
									},
								},
							},
						},
					},
				},
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"Node": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"value": map[string]any{
							"type": "string",
						},
						"next": map[string]any{
							"$ref": "#/components/schemas/Node",
						},
					},
				},
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(circularJSON, "", "  ")
	require.NoError(t, err, "failed to marshal JSON fixture")

	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "circular-schema.json")
	require.NoError(t, os.WriteFile(jsonFile, jsonBytes, 0644), "failed to write JSON fixture")

	p := New()
	p.ResolveRefs = true

	result, err := p.Parse(jsonFile)
	require.NoError(t, err, "Parse should succeed for JSON circular schema")

	data := result.Data
	require.NotNil(t, data, "result.Data should not be nil")

	assertCircularResolutionBehavior(t, data)

	// Should also have the circular reference warning
	hasCircularWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "Circular references detected") {
			hasCircularWarning = true
			break
		}
	}
	assert.True(t, hasCircularWarning, "expected circular reference warning for JSON fast path")
}

// TestResolveRefs_NonCircularRefsUnaffected is a regression guard ensuring that
// non-circular specs continue to work correctly with ResolveRefs enabled, and
// that no spurious circular reference warnings are emitted.
func TestResolveRefs_NonCircularRefsUnaffected(t *testing.T) {
	p := New()
	p.ResolveRefs = true

	result, err := p.Parse(filepath.Join("..", "testdata", "petstore-v2.yaml"))
	require.NoError(t, err, "Parse should succeed for petstore-v2.yaml")

	// There should be no circular reference warnings
	for _, w := range result.Warnings {
		assert.False(t, strings.Contains(w, "Circular references detected"),
			"non-circular spec should not produce a circular reference warning, got: %s", w)
	}

	// Verify a known $ref is fully resolved:
	// paths./pets/{petId}.get.responses.200.content.application/json.schema
	// references #/components/schemas/Pet which should be resolved inline.
	data := result.Data
	require.NotNil(t, data, "result.Data should not be nil")

	paths := requireMapKey(t, data, "paths")
	petIdPath := requireMapKey(t, paths, "/pets/{petId}")
	get := requireMapKey(t, petIdPath, "get")
	responses := requireMapKey(t, get, "responses")
	resp200 := requireMapKey(t, responses, "200")
	content := requireMapKey(t, resp200, "content")
	appJSON := requireMapKey(t, content, "application/json")
	schema := requireMapKey(t, appJSON, "schema")

	// The $ref to Pet should have been fully resolved
	_, hasRef := schema["$ref"]
	assert.False(t, hasRef, "Pet schema ref should be fully resolved (no $ref key)")
	assert.Equal(t, "object", schema["type"], "resolved Pet schema should have type 'object'")

	// Verify that specific properties from the Pet schema are present
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "resolved Pet schema should have properties")
	_, hasName := properties["name"]
	assert.True(t, hasName, "resolved Pet schema should have 'name' property")
	_, hasID := properties["id"]
	assert.True(t, hasID, "resolved Pet schema should have 'id' property")
}
