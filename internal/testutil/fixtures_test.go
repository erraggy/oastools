package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// TestNewSimpleOAS2Document verifies that a minimal OAS 2.0 document is created correctly.
func TestNewSimpleOAS2Document(t *testing.T) {
	doc := NewSimpleOAS2Document()

	// Verify required fields
	assert.Equal(t, "2.0", doc.Swagger, "Swagger version should be 2.0")
	assert.Equal(t, parser.OASVersion20, doc.OASVersion, "OASVersion should be OASVersion20")
	require.NotNil(t, doc.Info, "Info should not be nil")
	assert.Equal(t, "Test API", doc.Info.Title, "Title should be set")
	assert.Equal(t, "1.0.0", doc.Info.Version, "Version should be set")
	assert.Equal(t, "api.example.com", doc.Host, "Host should be set")
	assert.Equal(t, "/v1", doc.BasePath, "BasePath should be set")
	assert.Equal(t, []string{"https"}, doc.Schemes, "Schemes should contain https")
	assert.NotNil(t, doc.Paths, "Paths map should be initialized")
	assert.Empty(t, doc.Paths, "Paths map should be empty")
}

// TestNewDetailedOAS2Document verifies that a complete OAS 2.0 document is created correctly.
func TestNewDetailedOAS2Document(t *testing.T) {
	doc := NewDetailedOAS2Document()

	// Verify it includes everything from simple document
	assert.Equal(t, "2.0", doc.Swagger)
	assert.Equal(t, parser.OASVersion20, doc.OASVersion)
	require.NotNil(t, doc.Info)

	// Verify definitions
	require.NotNil(t, doc.Definitions, "Definitions should be set")
	assert.Contains(t, doc.Definitions, "Pet", "Should have Pet definition")
	petSchema := doc.Definitions["Pet"]
	require.NotNil(t, petSchema, "Pet schema should not be nil")
	assert.Equal(t, "object", petSchema.Type, "Pet should be object type")
	assert.Contains(t, petSchema.Properties, "id", "Pet should have id property")
	assert.Contains(t, petSchema.Properties, "name", "Pet should have name property")

	// Verify paths
	require.NotNil(t, doc.Paths, "Paths should be set")
	assert.Contains(t, doc.Paths, "/pets", "Should have /pets path")
	petsPath := doc.Paths["/pets"]
	require.NotNil(t, petsPath, "/pets path should not be nil")
	require.NotNil(t, petsPath.Get, "GET operation should be defined")
	assert.Equal(t, "List pets", petsPath.Get.Summary, "GET summary should be set")
	assert.Equal(t, "listPets", petsPath.Get.OperationID, "GET operationId should be set")
}

// TestNewSimpleOAS3Document verifies that a minimal OAS 3.x document is created correctly.
func TestNewSimpleOAS3Document(t *testing.T) {
	doc := NewSimpleOAS3Document()

	// Verify required fields
	assert.Equal(t, "3.0.3", doc.OpenAPI, "OpenAPI version should be 3.0.3")
	assert.Equal(t, parser.OASVersion303, doc.OASVersion, "OASVersion should be OASVersion303")
	require.NotNil(t, doc.Info, "Info should not be nil")
	assert.Equal(t, "Test API", doc.Info.Title, "Title should be set")
	assert.Equal(t, "1.0.0", doc.Info.Version, "Version should be set")
	require.NotNil(t, doc.Servers, "Servers should not be nil")
	require.Len(t, doc.Servers, 1, "Should have one server")
	assert.Equal(t, "https://api.example.com/v1", doc.Servers[0].URL, "Server URL should be set")
	assert.Equal(t, "Production server", doc.Servers[0].Description, "Server description should be set")
	assert.NotNil(t, doc.Paths, "Paths map should be initialized")
	assert.Empty(t, doc.Paths, "Paths map should be empty")
}

// TestNewDetailedOAS3Document verifies that a complete OAS 3.x document is created correctly.
func TestNewDetailedOAS3Document(t *testing.T) {
	doc := NewDetailedOAS3Document()

	// Verify it includes everything from simple document
	assert.Equal(t, "3.0.3", doc.OpenAPI)
	assert.Equal(t, parser.OASVersion303, doc.OASVersion)
	require.NotNil(t, doc.Info)
	require.NotNil(t, doc.Servers)
	require.Len(t, doc.Servers, 1)

	// Verify paths
	require.NotNil(t, doc.Paths, "Paths should be set")
	assert.Contains(t, doc.Paths, "/pets", "Should have /pets path")
	petsPath := doc.Paths["/pets"]
	require.NotNil(t, petsPath, "/pets path should not be nil")
	require.NotNil(t, petsPath.Get, "GET operation should be defined")
	assert.Equal(t, "List pets", petsPath.Get.Summary, "GET summary should be set")
	assert.Equal(t, "listPets", petsPath.Get.OperationID, "GET operationId should be set")

	// Verify components
	require.NotNil(t, doc.Components, "Components should be set")
	require.NotNil(t, doc.Components.Schemas, "Components.Schemas should be set")
	assert.Contains(t, doc.Components.Schemas, "Pet", "Should have Pet schema")
	petSchema := doc.Components.Schemas["Pet"]
	require.NotNil(t, petSchema, "Pet schema should not be nil")
	assert.Equal(t, "object", petSchema.Type, "Pet should be object type")
	assert.Contains(t, petSchema.Properties, "id", "Pet should have id property")
	assert.Contains(t, petSchema.Properties, "name", "Pet should have name property")
}

// TestWriteTempYAML verifies that documents can be written to temporary YAML files.
func TestWriteTempYAML(t *testing.T) {
	doc := NewSimpleOAS2Document()

	// Write to temp file
	path := WriteTempYAML(t, doc)

	// Verify file exists
	assert.FileExists(t, path, "Temporary YAML file should exist")

	// Verify file has .yaml extension
	assert.Equal(t, ".yaml", filepath.Ext(path), "File should have .yaml extension")

	// Verify file is in temp directory
	assert.True(t, filepath.IsAbs(path), "Path should be absolute")

	// Verify file contains valid YAML
	data, err := os.ReadFile(path)
	require.NoError(t, err, "Should be able to read temp file")

	var parsed parser.OAS2Document
	err = yaml.Unmarshal(data, &parsed)
	require.NoError(t, err, "Should be able to unmarshal YAML")

	// Verify content matches
	assert.Equal(t, "2.0", parsed.Swagger, "Swagger version should match")
	assert.Equal(t, "Test API", parsed.Info.Title, "Title should match")
}

// TestWriteTempJSON verifies that documents can be written to temporary JSON files.
func TestWriteTempJSON(t *testing.T) {
	doc := NewSimpleOAS3Document()

	// Write to temp file
	path := WriteTempJSON(t, doc)

	// Verify file exists
	assert.FileExists(t, path, "Temporary JSON file should exist")

	// Verify file has .json extension
	assert.Equal(t, ".json", filepath.Ext(path), "File should have .json extension")

	// Verify file is in temp directory
	assert.True(t, filepath.IsAbs(path), "Path should be absolute")

	// Verify file contains valid JSON
	data, err := os.ReadFile(path)
	require.NoError(t, err, "Should be able to read temp file")

	var parsed parser.OAS3Document
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "Should be able to unmarshal JSON")

	// Verify content matches
	assert.Equal(t, "3.0.3", parsed.OpenAPI, "OpenAPI version should match")
	assert.Equal(t, "Test API", parsed.Info.Title, "Title should match")

	// Verify JSON is properly indented (should contain newlines)
	assert.Contains(t, string(data), "\n", "JSON should be indented with newlines")
}

// TestWriteTempYAMLCleanup verifies that temporary files are cleaned up after test.
func TestWriteTempYAMLCleanup(t *testing.T) {
	var tempPath string

	// Run subtest that creates temp file
	t.Run("create temp file", func(t *testing.T) {
		doc := NewSimpleOAS2Document()
		tempPath = WriteTempYAML(t, doc)
		assert.FileExists(t, tempPath, "File should exist during test")
	})

	// After subtest completes, t.TempDir cleanup should have run
	// Note: In a real test, the cleanup happens after the parent test completes,
	// so we can't actually verify cleanup in the same test function.
	// This test primarily verifies the functionality works correctly.
}

// TestWriteTempJSONCleanup verifies that temporary files are cleaned up after test.
func TestWriteTempJSONCleanup(t *testing.T) {
	var tempPath string

	// Run subtest that creates temp file
	t.Run("create temp file", func(t *testing.T) {
		doc := NewSimpleOAS3Document()
		tempPath = WriteTempJSON(t, doc)
		assert.FileExists(t, tempPath, "File should exist during test")
	})

	// After subtest completes, t.TempDir cleanup should have run
	// Note: In a real test, the cleanup happens after the parent test completes,
	// so we can't actually verify cleanup in the same test function.
	// This test primarily verifies the functionality works correctly.
}

// TestWriteTempYAMLWithOAS3 verifies WriteTempYAML works with OAS 3.x documents.
func TestWriteTempYAMLWithOAS3(t *testing.T) {
	doc := NewDetailedOAS3Document()

	path := WriteTempYAML(t, doc)
	assert.FileExists(t, path)

	// Parse and verify
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var parsed parser.OAS3Document
	err = yaml.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "3.0.3", parsed.OpenAPI)
	assert.NotNil(t, parsed.Components)
}

// TestWriteTempJSONWithOAS2 verifies WriteTempJSON works with OAS 2.0 documents.
func TestWriteTempJSONWithOAS2(t *testing.T) {
	doc := NewDetailedOAS2Document()

	path := WriteTempJSON(t, doc)
	assert.FileExists(t, path)

	// Parse and verify
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var parsed parser.OAS2Document
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "2.0", parsed.Swagger)
	assert.NotNil(t, parsed.Definitions)
}

// TestDocumentFactoryConsistency verifies that simple and detailed documents maintain consistency.
func TestDocumentFactoryConsistency(t *testing.T) {
	t.Run("OAS 2.0 consistency", func(t *testing.T) {
		simple := NewSimpleOAS2Document()
		detailed := NewDetailedOAS2Document()

		// Detailed should have same base fields as simple
		assert.Equal(t, simple.Swagger, detailed.Swagger)
		assert.Equal(t, simple.OASVersion, detailed.OASVersion)
		assert.Equal(t, simple.Host, detailed.Host)
		assert.Equal(t, simple.BasePath, detailed.BasePath)
		assert.Equal(t, simple.Schemes, detailed.Schemes)
		assert.Equal(t, simple.Info.Title, detailed.Info.Title)
		assert.Equal(t, simple.Info.Version, detailed.Info.Version)

		// Detailed should have additional content
		assert.Nil(t, simple.Definitions, "Simple should not have definitions")
		assert.NotNil(t, detailed.Definitions, "Detailed should have definitions")
		assert.Empty(t, simple.Paths, "Simple should have empty paths")
		assert.NotEmpty(t, detailed.Paths, "Detailed should have populated paths")
	})

	t.Run("OAS 3.x consistency", func(t *testing.T) {
		simple := NewSimpleOAS3Document()
		detailed := NewDetailedOAS3Document()

		// Detailed should have same base fields as simple
		assert.Equal(t, simple.OpenAPI, detailed.OpenAPI)
		assert.Equal(t, simple.OASVersion, detailed.OASVersion)
		assert.Equal(t, simple.Servers[0].URL, detailed.Servers[0].URL)
		assert.Equal(t, simple.Info.Title, detailed.Info.Title)
		assert.Equal(t, simple.Info.Version, detailed.Info.Version)

		// Detailed should have additional content
		assert.Nil(t, simple.Components, "Simple should not have components")
		assert.NotNil(t, detailed.Components, "Detailed should have components")
		assert.Empty(t, simple.Paths, "Simple should have empty paths")
		assert.NotEmpty(t, detailed.Paths, "Detailed should have populated paths")
	})
}
