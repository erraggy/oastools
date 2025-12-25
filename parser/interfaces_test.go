package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentAccessor_OAS2(t *testing.T) {
	result, err := ParseWithOptions(WithFilePath("../testdata/petstore-2.0.yaml"))
	require.NoError(t, err, "failed to parse OAS 2.0 file")

	accessor := result.AsAccessor()
	require.NotNil(t, accessor, "AsAccessor should not return nil for valid OAS 2.0")

	t.Run("GetInfo", func(t *testing.T) {
		info := accessor.GetInfo()
		require.NotNil(t, info)
		assert.Equal(t, "Petstore API", info.Title)
		assert.Equal(t, "1.0.0", info.Version)
	})

	t.Run("GetPaths", func(t *testing.T) {
		paths := accessor.GetPaths()
		require.NotNil(t, paths)
		assert.Contains(t, paths, "/pets")
		assert.Contains(t, paths, "/pets/{petId}")
	})

	t.Run("GetTags", func(t *testing.T) {
		tags := accessor.GetTags()
		require.NotNil(t, tags)
		assert.NotEmpty(t, tags)
	})

	t.Run("GetSecurity", func(t *testing.T) {
		security := accessor.GetSecurity()
		// OAS 2.0 petstore may or may not have global security
		// Just verify it doesn't panic
		_ = security
	})

	t.Run("GetExternalDocs", func(t *testing.T) {
		docs := accessor.GetExternalDocs()
		// May be nil, just verify it doesn't panic
		_ = docs
	})

	t.Run("GetSchemas", func(t *testing.T) {
		schemas := accessor.GetSchemas()
		require.NotNil(t, schemas, "OAS 2.0 petstore should have definitions")
		assert.Contains(t, schemas, "Pet")
		assert.Contains(t, schemas, "Error")
	})

	t.Run("GetSecuritySchemes", func(t *testing.T) {
		schemes := accessor.GetSecuritySchemes()
		// May be nil or non-nil depending on the spec
		_ = schemes
	})

	t.Run("GetParameters", func(t *testing.T) {
		params := accessor.GetParameters()
		// May be nil or non-nil depending on the spec
		_ = params
	})

	t.Run("GetResponses", func(t *testing.T) {
		responses := accessor.GetResponses()
		// May be nil or non-nil depending on the spec
		_ = responses
	})

	t.Run("GetVersion", func(t *testing.T) {
		version := accessor.GetVersion()
		assert.Equal(t, OASVersion20, version)
	})

	t.Run("GetVersionString", func(t *testing.T) {
		versionStr := accessor.GetVersionString()
		assert.Equal(t, "2.0", versionStr)
	})

	t.Run("SchemaRefPrefix", func(t *testing.T) {
		prefix := accessor.SchemaRefPrefix()
		assert.Equal(t, "#/definitions/", prefix)
	})
}

func TestDocumentAccessor_OAS3(t *testing.T) {
	result, err := ParseWithOptions(WithFilePath("../testdata/petstore-3.0.yaml"))
	require.NoError(t, err, "failed to parse OAS 3.0 file")

	accessor := result.AsAccessor()
	require.NotNil(t, accessor, "AsAccessor should not return nil for valid OAS 3.0")

	t.Run("GetInfo", func(t *testing.T) {
		info := accessor.GetInfo()
		require.NotNil(t, info)
		assert.Equal(t, "Petstore API", info.Title)
	})

	t.Run("GetPaths", func(t *testing.T) {
		paths := accessor.GetPaths()
		require.NotNil(t, paths)
		assert.Contains(t, paths, "/pets")
		assert.Contains(t, paths, "/pets/{petId}")
	})

	t.Run("GetTags", func(t *testing.T) {
		tags := accessor.GetTags()
		require.NotNil(t, tags)
		assert.NotEmpty(t, tags)
	})

	t.Run("GetSchemas", func(t *testing.T) {
		schemas := accessor.GetSchemas()
		require.NotNil(t, schemas, "OAS 3.0 petstore should have component schemas")
		assert.Contains(t, schemas, "Pet")
		assert.Contains(t, schemas, "Error")
	})

	t.Run("GetVersion", func(t *testing.T) {
		version := accessor.GetVersion()
		assert.True(t, version == OASVersion300 || version == OASVersion301 ||
			version == OASVersion302 || version == OASVersion303 || version == OASVersion304)
	})

	t.Run("GetVersionString", func(t *testing.T) {
		versionStr := accessor.GetVersionString()
		assert.Contains(t, versionStr, "3.0")
	})

	t.Run("SchemaRefPrefix", func(t *testing.T) {
		prefix := accessor.SchemaRefPrefix()
		assert.Equal(t, "#/components/schemas/", prefix)
	})
}

func TestDocumentAccessor_OAS31(t *testing.T) {
	result, err := ParseWithOptions(WithFilePath("../testdata/petstore-3.1.yaml"))
	require.NoError(t, err, "failed to parse OAS 3.1 file")

	accessor := result.AsAccessor()
	require.NotNil(t, accessor, "AsAccessor should not return nil for valid OAS 3.1")

	t.Run("GetVersion", func(t *testing.T) {
		version := accessor.GetVersion()
		assert.True(t, version == OASVersion310 || version == OASVersion311 || version == OASVersion312)
	})

	t.Run("SchemaRefPrefix", func(t *testing.T) {
		prefix := accessor.SchemaRefPrefix()
		assert.Equal(t, "#/components/schemas/", prefix)
	})
}

func TestDocumentAccessor_NilComponents(t *testing.T) {
	// Test that OAS3 accessors handle nil Components gracefully
	doc := &OAS3Document{
		OpenAPI:    "3.0.0",
		OASVersion: OASVersion300,
		Info:       &Info{Title: "Test", Version: "1.0"},
		Paths:      make(Paths),
		Components: nil, // Explicitly nil
	}

	t.Run("GetSchemas_NilComponents", func(t *testing.T) {
		schemas := doc.GetSchemas()
		assert.Nil(t, schemas)
	})

	t.Run("GetSecuritySchemes_NilComponents", func(t *testing.T) {
		schemes := doc.GetSecuritySchemes()
		assert.Nil(t, schemes)
	})

	t.Run("GetParameters_NilComponents", func(t *testing.T) {
		params := doc.GetParameters()
		assert.Nil(t, params)
	})

	t.Run("GetResponses_NilComponents", func(t *testing.T) {
		responses := doc.GetResponses()
		assert.Nil(t, responses)
	})
}

func TestAsAccessor_NilParseResult(t *testing.T) {
	var result *ParseResult
	accessor := result.AsAccessor()
	assert.Nil(t, accessor, "AsAccessor should return nil for nil ParseResult")
}

func TestAsAccessor_UnknownDocumentType(t *testing.T) {
	result := &ParseResult{
		Document: "not a valid document type",
	}
	accessor := result.AsAccessor()
	assert.Nil(t, accessor, "AsAccessor should return nil for unknown document type")
}

func TestAsAccessor_NilDocument(t *testing.T) {
	result := &ParseResult{
		Document: nil,
	}
	accessor := result.AsAccessor()
	assert.Nil(t, accessor, "AsAccessor should return nil when Document is nil")
}

func TestDocumentAccessor_VersionAgnosticIteration(t *testing.T) {
	// Test that we can use the same code path for both OAS 2.0 and 3.x
	testCases := []struct {
		name     string
		filePath string
	}{
		{"OAS2", "../testdata/petstore-2.0.yaml"},
		{"OAS3", "../testdata/petstore-3.0.yaml"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseWithOptions(WithFilePath(tc.filePath))
			require.NoError(t, err)

			accessor := result.AsAccessor()
			require.NotNil(t, accessor)

			// Count paths - works the same for both versions
			pathCount := 0
			for range accessor.GetPaths() {
				pathCount++
			}
			assert.Greater(t, pathCount, 0, "should have at least one path")

			// Count schemas - works the same for both versions
			schemaCount := 0
			for range accessor.GetSchemas() {
				schemaCount++
			}
			assert.Greater(t, schemaCount, 0, "should have at least one schema")

			// Version info accessible uniformly
			assert.True(t, accessor.GetVersion().IsValid())
			assert.NotEmpty(t, accessor.GetVersionString())
			assert.NotEmpty(t, accessor.SchemaRefPrefix())
		})
	}
}

func TestOAS2Document_DirectAccessorMethods(t *testing.T) {
	doc := &OAS2Document{
		Swagger:             "2.0",
		OASVersion:          OASVersion20,
		Info:                &Info{Title: "Test API", Version: "1.0"},
		Paths:               Paths{"/test": &PathItem{}},
		Definitions:         map[string]*Schema{"TestSchema": {}},
		SecurityDefinitions: map[string]*SecurityScheme{"apiKey": {}},
		Parameters:          map[string]*Parameter{"testParam": {}},
		Responses:           map[string]*Response{"200": {}},
		Security:            []SecurityRequirement{{"apiKey": {}}},
		Tags:                []*Tag{{Name: "test"}},
		ExternalDocs:        &ExternalDocs{URL: "https://example.com"},
	}

	assert.Equal(t, doc.Info, doc.GetInfo())
	assert.Equal(t, doc.Paths, doc.GetPaths())
	assert.Equal(t, doc.Tags, doc.GetTags())
	assert.Equal(t, doc.Security, doc.GetSecurity())
	assert.Equal(t, doc.ExternalDocs, doc.GetExternalDocs())
	assert.Equal(t, doc.Definitions, doc.GetSchemas())
	assert.Equal(t, doc.SecurityDefinitions, doc.GetSecuritySchemes())
	assert.Equal(t, doc.Parameters, doc.GetParameters())
	assert.Equal(t, doc.Responses, doc.GetResponses())
	assert.Equal(t, OASVersion20, doc.GetVersion())
	assert.Equal(t, "2.0", doc.GetVersionString())
	assert.Equal(t, "#/definitions/", doc.SchemaRefPrefix())
}

func TestOAS3Document_DirectAccessorMethods(t *testing.T) {
	doc := &OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: OASVersion303,
		Info:       &Info{Title: "Test API", Version: "1.0"},
		Paths:      Paths{"/test": &PathItem{}},
		Components: &Components{
			Schemas:         map[string]*Schema{"TestSchema": {}},
			SecuritySchemes: map[string]*SecurityScheme{"oauth2": {}},
			Parameters:      map[string]*Parameter{"testParam": {}},
			Responses:       map[string]*Response{"200": {}},
		},
		Security:     []SecurityRequirement{{"oauth2": {}}},
		Tags:         []*Tag{{Name: "test"}},
		ExternalDocs: &ExternalDocs{URL: "https://example.com"},
	}

	assert.Equal(t, doc.Info, doc.GetInfo())
	assert.Equal(t, doc.Paths, doc.GetPaths())
	assert.Equal(t, doc.Tags, doc.GetTags())
	assert.Equal(t, doc.Security, doc.GetSecurity())
	assert.Equal(t, doc.ExternalDocs, doc.GetExternalDocs())
	assert.Equal(t, doc.Components.Schemas, doc.GetSchemas())
	assert.Equal(t, doc.Components.SecuritySchemes, doc.GetSecuritySchemes())
	assert.Equal(t, doc.Components.Parameters, doc.GetParameters())
	assert.Equal(t, doc.Components.Responses, doc.GetResponses())
	assert.Equal(t, OASVersion303, doc.GetVersion())
	assert.Equal(t, "3.0.3", doc.GetVersionString())
	assert.Equal(t, "#/components/schemas/", doc.SchemaRefPrefix())
}
