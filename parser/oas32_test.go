package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOAS32SelfField tests that the $self field is properly parsed in OAS 3.2+ documents
func TestOAS32SelfField(t *testing.T) {
	spec := `
openapi: "3.2.0"
$self: "https://example.com/api/openapi.yaml"
info:
  title: Test API
  version: "1.0.0"
paths: {}
`
	result, err := ParseWithOptions(
		WithBytes([]byte(spec)),
		WithValidateStructure(true),
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.OAS3Document()
	require.True(t, ok)
	assert.Equal(t, "https://example.com/api/openapi.yaml", doc.Self)
}

// TestOAS32MediaTypesInComponents tests that mediaTypes can be defined in components
func TestOAS32MediaTypesInComponents(t *testing.T) {
	spec := `
openapi: "3.2.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              $ref: '#/components/mediaTypes/JsonResponse'
components:
  mediaTypes:
    JsonResponse:
      schema:
        type: object
        properties:
          message:
            type: string
`
	result, err := ParseWithOptions(
		WithBytes([]byte(spec)),
		WithValidateStructure(true),
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.OAS3Document()
	require.True(t, ok)
	require.NotNil(t, doc.Components)
	assert.Len(t, doc.Components.MediaTypes, 1)
	assert.NotNil(t, doc.Components.MediaTypes["JsonResponse"])
	assert.NotNil(t, doc.Components.MediaTypes["JsonResponse"].Schema)
}

// TestOAS32AdditionalOperations tests that custom HTTP methods can be defined via additionalOperations
func TestOAS32AdditionalOperations(t *testing.T) {
	spec := `
openapi: "3.2.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /resource:
    additionalOperations:
      PURGE:
        summary: Purge the resource cache
        operationId: purgeResource
        responses:
          "204":
            description: Cache purged
      LINK:
        summary: Create a link to another resource
        operationId: linkResource
        responses:
          "200":
            description: Link created
`
	result, err := ParseWithOptions(
		WithBytes([]byte(spec)),
		WithValidateStructure(true),
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.OAS3Document()
	require.True(t, ok)
	require.NotNil(t, doc.Paths)

	pathItem := doc.Paths["/resource"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.AdditionalOperations)
	assert.Len(t, pathItem.AdditionalOperations, 2)

	purgeOp := pathItem.AdditionalOperations["PURGE"]
	require.NotNil(t, purgeOp)
	assert.Equal(t, "purgeResource", purgeOp.OperationID)

	linkOp := pathItem.AdditionalOperations["LINK"]
	require.NotNil(t, linkOp)
	assert.Equal(t, "linkResource", linkOp.OperationID)
}

// TestOAS32GetOperationsIncludesAdditionalOperations tests that GetOperations includes custom methods
func TestOAS32GetOperationsIncludesAdditionalOperations(t *testing.T) {
	pathItem := &PathItem{
		Get: &Operation{OperationID: "getResource"},
		AdditionalOperations: map[string]*Operation{
			"PURGE": {OperationID: "purgeResource"},
			"LINK":  {OperationID: "linkResource"},
		},
	}

	// OAS 3.2+ should include additionalOperations
	ops := GetOperations(pathItem, OASVersion320)
	assert.NotNil(t, ops["get"])
	assert.NotNil(t, ops["PURGE"])
	assert.NotNil(t, ops["LINK"])
	assert.Equal(t, "purgeResource", ops["PURGE"].OperationID)

	// OAS 3.1 should NOT include additionalOperations
	ops31 := GetOperations(pathItem, OASVersion310)
	assert.NotNil(t, ops31["get"])
	assert.Nil(t, ops31["PURGE"])
	assert.Nil(t, ops31["LINK"])
}

// TestOAS32QueryMethod tests that the QUERY method is supported in OAS 3.2+
func TestOAS32QueryMethod(t *testing.T) {
	spec := `
openapi: "3.2.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /search:
    query:
      summary: Query resources
      operationId: queryResources
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        "200":
          description: Query results
`
	result, err := ParseWithOptions(
		WithBytes([]byte(spec)),
		WithValidateStructure(true),
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.OAS3Document()
	require.True(t, ok)

	pathItem := doc.Paths["/search"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Query)
	assert.Equal(t, "queryResources", pathItem.Query.OperationID)

	// Verify GetOperations includes query for OAS 3.2+
	ops := GetOperations(pathItem, OASVersion320)
	assert.NotNil(t, ops["query"])
}

// TestOAS32DeepCopy tests that new OAS 3.2 fields are properly deep copied
func TestOAS32DeepCopy(t *testing.T) {
	original := &OAS3Document{
		OpenAPI: "3.2.0",
		Self:    "https://example.com/api.yaml",
		Components: &Components{
			MediaTypes: map[string]*MediaType{
				"JsonResponse": {
					Schema: &Schema{
						Type: "object",
					},
				},
			},
		},
		Paths: Paths{
			"/test": &PathItem{
				AdditionalOperations: map[string]*Operation{
					"PURGE": {OperationID: "purgeResource"},
				},
			},
		},
	}

	copied := original.DeepCopy()

	// Verify the copy is independent
	assert.Equal(t, original.Self, copied.Self)
	assert.Equal(t, original.Components.MediaTypes["JsonResponse"].Schema.Type, copied.Components.MediaTypes["JsonResponse"].Schema.Type)

	// Modify original and verify copy is unaffected
	original.Self = "modified"
	original.Components.MediaTypes["JsonResponse"].Schema.Type = "string"
	original.Paths["/test"].AdditionalOperations["PURGE"].OperationID = "modified"

	assert.Equal(t, "https://example.com/api.yaml", copied.Self)
	assert.Equal(t, "object", copied.Components.MediaTypes["JsonResponse"].Schema.Type)
	assert.Equal(t, "purgeResource", copied.Paths["/test"].AdditionalOperations["PURGE"].OperationID)
}

// TestOAS32JSONMarshaling tests JSON round-trip for OAS 3.2 documents
func TestOAS32JSONMarshaling(t *testing.T) {
	spec := `{
  "openapi": "3.2.0",
  "$self": "https://example.com/api.yaml",
  "info": {
    "title": "Test API",
    "version": "1.0.0"
  },
  "paths": {
    "/test": {
      "additionalOperations": {
        "PURGE": {
          "operationId": "purgeResource",
          "responses": {
            "204": {
              "description": "Purged"
            }
          }
        }
      }
    }
  },
  "components": {
    "mediaTypes": {
      "JsonResponse": {
        "schema": {
          "type": "object"
        }
      }
    }
  }
}`
	result, err := ParseWithOptions(
		WithBytes([]byte(spec)),
		WithValidateStructure(true),
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, SourceFormatJSON, result.SourceFormat)

	doc, ok := result.OAS3Document()
	require.True(t, ok)
	assert.Equal(t, "https://example.com/api.yaml", doc.Self)
	assert.NotNil(t, doc.Components.MediaTypes["JsonResponse"])
	assert.NotNil(t, doc.Paths["/test"].AdditionalOperations["PURGE"])
}

// TestOAS32VersionDetection tests that OAS 3.2.0 is properly detected
func TestOAS32VersionDetection(t *testing.T) {
	tests := []struct {
		version  string
		expected OASVersion
	}{
		{"3.2.0", OASVersion320},
		{"3.2.1", OASVersion320}, // Future 3.2.x maps to 3.2.0
		{"3.2.5", OASVersion320}, // Future 3.2.x maps to 3.2.0
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			spec := strings.ReplaceAll(`
openapi: "VERSION"
info:
  title: Test
  version: "1.0"
paths: {}
`, "VERSION", tt.version)

			result, err := ParseWithOptions(
				WithBytes([]byte(spec)),
				WithValidateStructure(true),
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.OASVersion)
		})
	}
}
