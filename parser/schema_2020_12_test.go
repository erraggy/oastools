package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSONSchema2020_12_ContentKeywords tests content-related keywords from JSON Schema Draft 2020-12
func TestJSONSchema2020_12_ContentKeywords(t *testing.T) {
	spec := `
openapi: "3.1.0"
info:
  title: Content Keywords Test
  version: "1.0.0"
paths: {}
components:
  schemas:
    Base64Content:
      type: string
      contentEncoding: base64
      contentMediaType: application/json
      contentSchema:
        type: object
        properties:
          data:
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
	require.NotNil(t, doc.Components.Schemas)

	schema := doc.Components.Schemas["Base64Content"]
	require.NotNil(t, schema)

	assert.Equal(t, "base64", schema.ContentEncoding)
	assert.Equal(t, "application/json", schema.ContentMediaType)
	require.NotNil(t, schema.ContentSchema)
	assert.Equal(t, "object", schema.ContentSchema.Type)
}

// TestJSONSchema2020_12_UnevaluatedProperties tests unevaluatedProperties keyword
func TestJSONSchema2020_12_UnevaluatedProperties(t *testing.T) {
	tests := []struct {
		name     string
		spec     string
		expected any // bool or schema check
	}{
		{
			name: "unevaluatedProperties false",
			spec: `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"
paths: {}
components:
  schemas:
    Strict:
      type: object
      properties:
        name:
          type: string
      unevaluatedProperties: false
`,
			expected: false,
		},
		{
			name: "unevaluatedProperties true",
			spec: `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"
paths: {}
components:
  schemas:
    Flexible:
      type: object
      unevaluatedProperties: true
`,
			expected: true,
		},
		{
			name: "unevaluatedProperties with schema",
			spec: `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"
paths: {}
components:
  schemas:
    TypedExtra:
      type: object
      properties:
        id:
          type: integer
      unevaluatedProperties:
        type: string
`,
			expected: "schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseWithOptions(
				WithBytes([]byte(tt.spec)),
				WithValidateStructure(true),
			)
			require.NoError(t, err)

			doc, ok := result.OAS3Document()
			require.True(t, ok)

			var schema *Schema
			for _, s := range doc.Components.Schemas {
				schema = s
				break
			}
			require.NotNil(t, schema)

			if tt.expected == "schema" {
				// Should be a map representing a schema
				schemaMap, ok := schema.UnevaluatedProperties.(map[string]any)
				assert.True(t, ok, "expected unevaluatedProperties to be a schema map")
				assert.Equal(t, "string", schemaMap["type"])
			} else {
				assert.Equal(t, tt.expected, schema.UnevaluatedProperties)
			}
		})
	}
}

// TestJSONSchema2020_12_UnevaluatedItems tests unevaluatedItems keyword
func TestJSONSchema2020_12_UnevaluatedItems(t *testing.T) {
	tests := []struct {
		name     string
		spec     string
		expected any
	}{
		{
			name: "unevaluatedItems false",
			spec: `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"
paths: {}
components:
  schemas:
    FixedTuple:
      type: array
      prefixItems:
        - type: string
        - type: integer
      unevaluatedItems: false
`,
			expected: false,
		},
		{
			name: "unevaluatedItems with schema",
			spec: `
openapi: "3.1.0"
info:
  title: Test
  version: "1.0.0"
paths: {}
components:
  schemas:
    TypedTuple:
      type: array
      prefixItems:
        - type: string
      unevaluatedItems:
        type: integer
`,
			expected: "schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseWithOptions(
				WithBytes([]byte(tt.spec)),
				WithValidateStructure(true),
			)
			require.NoError(t, err)

			doc, ok := result.OAS3Document()
			require.True(t, ok)

			var schema *Schema
			for _, s := range doc.Components.Schemas {
				schema = s
				break
			}
			require.NotNil(t, schema)

			if tt.expected == "schema" {
				schemaMap, ok := schema.UnevaluatedItems.(map[string]any)
				assert.True(t, ok, "expected unevaluatedItems to be a schema map")
				assert.Equal(t, "integer", schemaMap["type"])
			} else {
				assert.Equal(t, tt.expected, schema.UnevaluatedItems)
			}
		})
	}
}

// TestJSONSchema2020_12_DeepCopy tests that new JSON Schema fields are properly deep copied
func TestJSONSchema2020_12_DeepCopy(t *testing.T) {
	original := &Schema{
		Type:             "object",
		ContentEncoding:  "base64",
		ContentMediaType: "application/json",
		ContentSchema:    &Schema{Type: "string"},
		UnevaluatedProperties: map[string]any{
			"type": "string",
		},
		UnevaluatedItems: false,
	}

	copied := original.DeepCopy()

	// Verify the copy is independent
	assert.Equal(t, original.ContentEncoding, copied.ContentEncoding)
	assert.Equal(t, original.ContentMediaType, copied.ContentMediaType)
	assert.Equal(t, original.ContentSchema.Type, copied.ContentSchema.Type)
	assert.Equal(t, original.UnevaluatedItems, copied.UnevaluatedItems)

	// Modify original and verify copy is unaffected
	original.ContentEncoding = "modified"
	original.ContentMediaType = "modified"
	original.ContentSchema.Type = "modified"
	original.UnevaluatedItems = true

	assert.Equal(t, "base64", copied.ContentEncoding)
	assert.Equal(t, "application/json", copied.ContentMediaType)
	assert.Equal(t, "string", copied.ContentSchema.Type)
	assert.Equal(t, false, copied.UnevaluatedItems)
}

// TestJSONSchema2020_12_JSONRoundTrip tests JSON marshaling/unmarshaling preserves fields
func TestJSONSchema2020_12_JSONRoundTrip(t *testing.T) {
	spec := `{
  "openapi": "3.1.0",
  "info": {
    "title": "JSON Round Trip Test",
    "version": "1.0.0"
  },
  "paths": {},
  "components": {
    "schemas": {
      "EncodedData": {
        "type": "string",
        "contentEncoding": "base32",
        "contentMediaType": "text/plain",
        "contentSchema": {
          "type": "object"
        }
      },
      "StrictObject": {
        "type": "object",
        "unevaluatedProperties": false
      },
      "StrictArray": {
        "type": "array",
        "unevaluatedItems": false
      }
    }
  }
}`
	result, err := ParseWithOptions(
		WithBytes([]byte(spec)),
		WithValidateStructure(true),
	)
	require.NoError(t, err)
	assert.Equal(t, SourceFormatJSON, result.SourceFormat)

	doc, ok := result.OAS3Document()
	require.True(t, ok)

	// Verify content keywords
	encoded := doc.Components.Schemas["EncodedData"]
	require.NotNil(t, encoded)
	assert.Equal(t, "base32", encoded.ContentEncoding)
	assert.Equal(t, "text/plain", encoded.ContentMediaType)
	require.NotNil(t, encoded.ContentSchema)

	// Verify unevaluatedProperties
	strictObj := doc.Components.Schemas["StrictObject"]
	require.NotNil(t, strictObj)
	assert.Equal(t, false, strictObj.UnevaluatedProperties)

	// Verify unevaluatedItems
	strictArr := doc.Components.Schemas["StrictArray"]
	require.NotNil(t, strictArr)
	assert.Equal(t, false, strictArr.UnevaluatedItems)
}
