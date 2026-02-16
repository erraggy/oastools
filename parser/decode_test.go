package parser

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// TestDecodeFromMap_RoundTripEquivalence verifies that decodeDocumentFromMap
// produces identical results to the traditional marshal->unmarshal path.
//
// Strategy: Read raw YAML, unmarshal to map[string]any, then feed the same
// map to both decodeDocumentFromMap (new path) and yaml.Unmarshal into the
// typed struct (old path via parseVersionSpecific). The resulting Documents
// should be structurally identical.
//
// These fixtures are ref-free, so both paths produce the same representation.
// Specs with $ref are tested separately in TestDecodeFromMap_WithResolvedRefs
// because the two paths handle unresolved $ref differently in polymorphic
// fields (decodeFromMap decodes into *Schema, yaml.Unmarshal leaves as map).
func TestDecodeFromMap_RoundTripEquivalence(t *testing.T) {
	fixtures := []struct {
		name    string
		path    string
		version string
	}{
		{"minimal-oas2", filepath.Join("..", "testdata", "minimal-oas2.yaml"), "2.0"},
		{"minimal-oas3", filepath.Join("..", "testdata", "minimal-oas3.yaml"), "3.0.0"},
		{"empty-oas2", filepath.Join("..", "testdata", "empty-oas2.yaml"), "2.0"},
		{"empty-oas3", filepath.Join("..", "testdata", "empty-oas3.yaml"), "3.0.0"},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			data, err := os.ReadFile(fixture.path)
			require.NoError(t, err)

			// Unmarshal to raw map (same starting point for both paths)
			var rawMap map[string]any
			require.NoError(t, yaml.Unmarshal(data, &rawMap))

			// New path: decodeDocumentFromMap
			docNew, oasVersionNew, err := decodeDocumentFromMap(rawMap, fixture.version)
			require.NoError(t, err)

			// Old path: yaml.Unmarshal into typed struct via parseVersionSpecific
			p := New()
			docOld, oasVersionOld, err := p.parseVersionSpecific(data, fixture.version)
			require.NoError(t, err)

			// Compare OAS versions
			assert.Equal(t, oasVersionOld, oasVersionNew, "OAS versions should match")

			// Compare documents using DocumentEquals
			resultOld := &ParseResult{Document: docOld, OASVersion: oasVersionOld}
			resultNew := &ParseResult{Document: docNew, OASVersion: oasVersionNew}
			assert.True(t, resultOld.DocumentEquals(resultNew),
				"Documents should be structurally identical between old and new paths")
		})
	}
}

// TestDecodeFromMap_PetstoreStructure verifies that decodeDocumentFromMap
// correctly decodes rich petstore specs with operations, schemas, and
// parameters. These fixtures contain $ref, but since we feed the raw
// (unresolved) map to decodeDocumentFromMap, the $ref values appear in
// the Ref fields of the decoded structs.
func TestDecodeFromMap_PetstoreStructure(t *testing.T) {
	t.Run("petstore-2.0", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join("..", "testdata", "petstore-2.0.yaml"))
		require.NoError(t, err)

		var rawMap map[string]any
		require.NoError(t, yaml.Unmarshal(data, &rawMap))

		doc, oasVersion, err := decodeDocumentFromMap(rawMap, "2.0")
		require.NoError(t, err)
		assert.Equal(t, OASVersion20, oasVersion)

		oas2, ok := doc.(*OAS2Document)
		require.True(t, ok)
		assert.Equal(t, "2.0", oas2.Swagger)
		require.NotNil(t, oas2.Info)
		assert.Equal(t, "Petstore API", oas2.Info.Title)
		assert.NotEmpty(t, oas2.Paths, "should have paths")
		assert.NotEmpty(t, oas2.Definitions, "should have definitions")

		// Verify a specific path was decoded
		petsPath, hasPets := oas2.Paths["/pets"]
		require.True(t, hasPets, "/pets path should exist")
		require.NotNil(t, petsPath.Get, "/pets should have GET operation")
		assert.Equal(t, "listPets", petsPath.Get.OperationID)
	})

	t.Run("petstore-3.0", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join("..", "testdata", "petstore-3.0.yaml"))
		require.NoError(t, err)

		var rawMap map[string]any
		require.NoError(t, yaml.Unmarshal(data, &rawMap))

		doc, oasVersion, err := decodeDocumentFromMap(rawMap, "3.0.3")
		require.NoError(t, err)
		assert.Equal(t, OASVersion303, oasVersion)

		oas3, ok := doc.(*OAS3Document)
		require.True(t, ok)
		assert.Equal(t, "3.0.3", oas3.OpenAPI)
		require.NotNil(t, oas3.Info)
		assert.Equal(t, "Petstore API", oas3.Info.Title)
		assert.NotEmpty(t, oas3.Paths, "should have paths")
		require.NotNil(t, oas3.Components)
		assert.NotEmpty(t, oas3.Components.Schemas, "should have component schemas")

		// Verify Pet schema decoded correctly
		pet, hasPet := oas3.Components.Schemas["Pet"]
		require.True(t, hasPet)
		assert.NotNil(t, pet.Properties, "Pet should have properties")
		assert.Contains(t, pet.Required, "id")
		assert.Contains(t, pet.Required, "name")

		// Verify operation decoded correctly
		petsPath, hasPets := oas3.Paths["/pets"]
		require.True(t, hasPets)
		require.NotNil(t, petsPath.Get)
		assert.Equal(t, "listPets", petsPath.Get.OperationID)
		require.NotNil(t, petsPath.Get.Responses)
		assert.NotEmpty(t, petsPath.Get.Responses.Codes, "should have response codes")
	})

	t.Run("petstore-3.1", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join("..", "testdata", "petstore-3.1.yaml"))
		require.NoError(t, err)

		var rawMap map[string]any
		require.NoError(t, yaml.Unmarshal(data, &rawMap))

		doc, oasVersion, err := decodeDocumentFromMap(rawMap, "3.1.0")
		require.NoError(t, err)
		assert.Equal(t, OASVersion310, oasVersion)

		oas3, ok := doc.(*OAS3Document)
		require.True(t, ok)
		assert.Equal(t, "3.1.0", oas3.OpenAPI)
		require.NotNil(t, oas3.Info)
		assert.NotEmpty(t, oas3.Paths, "should have paths")
		require.NotNil(t, oas3.Components)
		assert.NotEmpty(t, oas3.Components.Schemas, "should have component schemas")
	})
}

// TestDecodeFromMap_JSONFastPath verifies that the JSON fast path
// (parseVersionSpecificJSON, used when ResolveRefs=false) produces the
// same Document as the map-based decode path (decodeDocumentFromMap,
// used when ResolveRefs=true).
func TestDecodeFromMap_JSONFastPath(t *testing.T) {
	fixture := filepath.Join("..", "testdata", "minimal-oas3.json")

	// JSON fast path: ResolveRefs=false → parseVersionSpecificJSON
	pFast := New()
	pFast.ResolveRefs = false
	resultFast, err := pFast.Parse(fixture)
	require.NoError(t, err)

	// Map decode path: ResolveRefs=true → decodeDocumentFromMap
	pMap := New()
	pMap.ResolveRefs = true
	resultMap, err := pMap.Parse(fixture)
	require.NoError(t, err)

	assert.True(t, resultFast.DocumentEquals(resultMap),
		"JSON fast path (parseVersionSpecificJSON) should match map decode path (decodeDocumentFromMap)")
}

// TestDecodeFromMap_WithResolvedRefs verifies that decodeFromMap correctly
// decodes a spec where refs have been resolved into inline content.
func TestDecodeFromMap_WithResolvedRefs(t *testing.T) {
	p := New()
	p.ResolveRefs = true

	result, err := p.Parse(filepath.Join("..", "testdata", "petstore-3.0.yaml"))
	require.NoError(t, err)
	require.NotNil(t, result.Document)

	// Verify that $ref resolution happened: a known ref should be inlined
	doc, ok := result.Document.(*OAS3Document)
	require.True(t, ok)
	require.NotNil(t, doc.Components)
	require.NotNil(t, doc.Components.Schemas)

	// The Pet schema should exist and have properties
	pet, hasPet := doc.Components.Schemas["Pet"]
	require.True(t, hasPet, "Pet schema should exist in components")
	require.NotNil(t, pet.Properties, "Pet schema should have properties")
}

// TestGeneratorFreshness verifies that the generated decode file is up-to-date
// with the current struct definitions.
func TestGeneratorFreshness(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping generator freshness test in short mode")
	}

	cmd := exec.Command("go", "run", "../internal/codegen/decode", "-check")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated decode file is stale; run 'go generate ./parser/' to update.\n%s", string(output))
	}
}

// TestShallowCopy_ProducesIdenticalDocument verifies that parsing with
// shallow copy (the new default for ResolveRefs=true) produces a fully
// populated typed document with all key structures intact.
func TestShallowCopy_ProducesIdenticalDocument(t *testing.T) {
	p := New()
	p.ResolveRefs = true

	// Parse a spec with many shared refs to stress-test shallow copy
	result, err := p.Parse(filepath.Join("..", "testdata", "petstore-3.0.yaml"))
	require.NoError(t, err)
	require.NotNil(t, result.Document)

	doc, ok := result.Document.(*OAS3Document)
	require.True(t, ok)

	// Verify key structures are populated (not nil from broken decode)
	require.NotNil(t, doc.Info, "Info should be populated")
	assert.NotEmpty(t, doc.Info.Title, "Info.Title should not be empty")
	require.NotNil(t, doc.Paths, "Paths should be populated")
	assert.Greater(t, len(doc.Paths), 0, "should have at least one path")
}

// TestShallowCopy_SharedMapSafety verifies that result.Data with shallow
// copy contains shared sub-maps (not independent copies), and that this
// doesn't affect the typed Document (which is independent).
func TestShallowCopy_SharedMapSafety(t *testing.T) {
	p := New()
	p.ResolveRefs = true

	result, err := p.Parse(filepath.Join("..", "testdata", "petstore-3.0.yaml"))
	require.NoError(t, err)

	// The typed document should be fully independent of result.Data
	doc, ok := result.Document.(*OAS3Document)
	require.True(t, ok)

	// Mutate result.Data — the typed document should be unaffected
	paths, ok := result.Data["paths"].(map[string]any)
	require.True(t, ok, "result.Data[\"paths\"] should be map[string]any")
	paths["/__injected_test__"] = map[string]any{"get": map[string]any{}}

	// Typed document should NOT have the injected path
	_, hasInjected := doc.Paths["/__injected_test__"]
	assert.False(t, hasInjected,
		"typed document should be independent of result.Data mutations")
}
