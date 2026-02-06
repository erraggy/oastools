package fixer

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenericSchemaFixerRefCorruption tests that package-qualified refs are not corrupted
// This is the integration test from issue #233
func TestGenericSchemaFixerRefCorruption(t *testing.T) {
	spec := []byte(`{
        "swagger": "2.0",
        "info": {"title": "Test", "version": "1.0.0"},
        "paths": {
            "/test": {
                "get": {
                    "operationId": "test",
                    "responses": {
                        "200": {
                            "description": "OK",
                            "schema": {"$ref": "#/definitions/Response[[]*common.Pet]"}
                        },
                        "403": {
                            "description": "Forbidden",
                            "schema": {"$ref": "#/definitions/common.Error"}
                        }
                    }
                }
            }
        },
        "definitions": {
            "Response[[]*common.Pet]": {
                "type": "object",
                "properties": {
                    "data": {"type": "array", "items": {"$ref": "#/definitions/common.Pet"}},
                    "meta": {"$ref": "#/definitions/common.MetaInfo"}
                }
            },
            "common.Pet": {"type": "object", "properties": {"id": {"type": "integer"}}},
            "common.Error": {"type": "object", "properties": {"code": {"type": "integer"}}},
            "common.MetaInfo": {
                "type": "object",
                "properties": {
                    "pagination": {"$ref": "#/definitions/common.Pagination"}
                }
            },
            "common.Pagination": {"type": "object", "properties": {"offset": {"type": "integer"}}}
        }
    }`)

	pr, err := parser.ParseWithOptions(parser.WithBytes(spec))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*pr),
		WithEnabledFixes(FixTypeRenamedGenericSchema),
		WithGenericNamingConfig(GenericNamingConfig{
			Strategy: GenericNamingOf,
		}),
	)
	require.NoError(t, err)

	doc := result.Document.(*parser.OAS2Document)

	// The generic schema should be renamed
	assert.NotContains(t, doc.Definitions, "Response[[]*common.Pet]",
		"generic schema should be renamed")

	// Package-qualified schemas should be UNCHANGED
	assert.Contains(t, doc.Definitions, "common.Pet",
		"non-generic schema should not be renamed")
	assert.Contains(t, doc.Definitions, "common.Error",
		"non-generic schema should not be renamed")
	assert.Contains(t, doc.Definitions, "common.MetaInfo",
		"non-generic schema should not be renamed")
	assert.Contains(t, doc.Definitions, "common.Pagination",
		"non-generic schema should not be renamed")

	// Critical: Refs to package-qualified schemas should be UNCHANGED
	metaInfo := doc.Definitions["common.MetaInfo"]
	require.NotNil(t, metaInfo)
	paginationRef := metaInfo.Properties["pagination"].Ref

	// THESE ARE THE BUG ASSERTIONS - refs should NOT be corrupted
	assert.NotEqual(t, "#/definitions/.common.Pagination", paginationRef,
		"ref should NOT have leading dot")
	assert.NotEqual(t, "#/definitions/*common.Pagination", paginationRef,
		"ref should NOT have asterisk prefix")
	assert.NotContains(t, paginationRef, "_0",
		"ref should NOT have _0 suffix mismatch")

	// Correct behavior - ref unchanged
	assert.Equal(t, "#/definitions/common.Pagination", paginationRef,
		"ref should be unchanged")

	// Verify the renamed schema has correct refs
	// Find the renamed schema (should be something like ResponseOfcommon.Pet)
	var renamedSchema *parser.Schema
	var renamedName string
	for name, schema := range doc.Definitions {
		if strings.HasPrefix(name, "Response") && name != "Response[[]*common.Pet]" {
			renamedSchema = schema
			renamedName = name
			break
		}
	}
	require.NotNil(t, renamedSchema, "should find renamed schema")
	t.Logf("Generic schema renamed to: %s", renamedName)

	// Check the data property ref
	if dataItems, ok := renamedSchema.Properties["data"].Items.(*parser.Schema); ok {
		assert.Equal(t, "#/definitions/common.Pet", dataItems.Ref,
			"data items ref should point to common.Pet")
	}

	// Check the meta property ref
	assert.Equal(t, "#/definitions/common.MetaInfo", renamedSchema.Properties["meta"].Ref,
		"meta ref should point to common.MetaInfo")
}

// TestFixInvalidSchemaNames_DiscriminatorMapping tests that discriminator mapping values
// are correctly rewritten when generic-named schemas are renamed. This exercises both
// full $ref paths and bare schema names in discriminator mappings, covering the
// discriminator branch of rewriteSchemaRefsRecursive and extractSchemaNameFromRefPath.
func TestFixInvalidSchemaNames_DiscriminatorMapping(t *testing.T) {
	spec := []byte(`openapi: "3.0.3"
info:
  title: Test
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/PetList[Dog]"
components:
  schemas:
    PetList[Dog]:
      type: object
      properties:
        items:
          type: array
          items:
            $ref: "#/components/schemas/Dog"
      discriminator:
        propertyName: petType
        mapping:
          dog: "#/components/schemas/Response[Dog]"
          cat: "Response[Cat]"
    Response[Dog]:
      type: object
      properties:
        name:
          type: string
    Response[Cat]:
      type: object
      properties:
        name:
          type: string
    Dog:
      type: object
      properties:
        breed:
          type: string
`)

	pr, err := parser.ParseWithOptions(parser.WithBytes(spec))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*pr),
		WithEnabledFixes(FixTypeRenamedGenericSchema),
		WithGenericNamingConfig(GenericNamingConfig{
			Strategy: GenericNamingOf,
		}),
	)
	require.NoError(t, err)

	doc := result.Document.(*parser.OAS3Document)

	// Generic schemas should be renamed
	assert.NotContains(t, doc.Components.Schemas, "PetList[Dog]",
		"generic schema PetList[Dog] should be renamed")
	assert.NotContains(t, doc.Components.Schemas, "Response[Dog]",
		"generic schema Response[Dog] should be renamed")
	assert.NotContains(t, doc.Components.Schemas, "Response[Cat]",
		"generic schema Response[Cat] should be renamed")

	// Non-generic schema should be unchanged
	assert.Contains(t, doc.Components.Schemas, "Dog",
		"non-generic schema Dog should not be renamed")

	// The renamed schemas should exist with "Of" naming
	assert.Contains(t, doc.Components.Schemas, "PetListOfDog",
		"PetList[Dog] should be renamed to PetListOfDog")
	assert.Contains(t, doc.Components.Schemas, "ResponseOfDog",
		"Response[Dog] should be renamed to ResponseOfDog")
	assert.Contains(t, doc.Components.Schemas, "ResponseOfCat",
		"Response[Cat] should be renamed to ResponseOfCat")

	// Verify discriminator mapping values were rewritten
	petListSchema := doc.Components.Schemas["PetListOfDog"]
	require.NotNil(t, petListSchema, "PetListOfDog schema should exist")
	require.NotNil(t, petListSchema.Discriminator, "discriminator should be preserved")
	require.NotNil(t, petListSchema.Discriminator.Mapping, "discriminator mapping should be preserved")

	// Full ref path mapping should be rewritten
	dogMapping, ok := petListSchema.Discriminator.Mapping["dog"]
	require.True(t, ok, "dog mapping key should exist")
	assert.Equal(t, "#/components/schemas/ResponseOfDog", dogMapping,
		"full ref discriminator mapping should be rewritten to new name")

	// Bare schema name mapping should also be rewritten
	catMapping, ok := petListSchema.Discriminator.Mapping["cat"]
	require.True(t, ok, "cat mapping key should exist")
	assert.Equal(t, "ResponseOfCat", catMapping,
		"bare name discriminator mapping should be rewritten to new name")
}

// TestGenericSchemaFixerRefCorruption_OAS3 tests that package-qualified refs are not corrupted in OAS 3.x
// This is the OAS 3.x version of the integration test from issue #233
func TestGenericSchemaFixerRefCorruption_OAS3(t *testing.T) {
	spec := []byte(`{
        "openapi": "3.0.3",
        "info": {"title": "Test", "version": "1.0.0"},
        "paths": {
            "/test": {
                "get": {
                    "operationId": "test",
                    "responses": {
                        "200": {
                            "description": "OK",
                            "content": {
                                "application/json": {
                                    "schema": {"$ref": "#/components/schemas/Response[[]*common.Pet]"}
                                }
                            }
                        },
                        "403": {
                            "description": "Forbidden",
                            "content": {
                                "application/json": {
                                    "schema": {"$ref": "#/components/schemas/common.Error"}
                                }
                            }
                        }
                    }
                }
            }
        },
        "components": {
            "schemas": {
                "Response[[]*common.Pet]": {
                    "type": "object",
                    "properties": {
                        "data": {"type": "array", "items": {"$ref": "#/components/schemas/common.Pet"}},
                        "meta": {"$ref": "#/components/schemas/common.MetaInfo"}
                    }
                },
                "common.Pet": {"type": "object", "properties": {"id": {"type": "integer"}}},
                "common.Error": {"type": "object", "properties": {"code": {"type": "integer"}}},
                "common.MetaInfo": {
                    "type": "object",
                    "properties": {
                        "pagination": {"$ref": "#/components/schemas/common.Pagination"}
                    }
                },
                "common.Pagination": {"type": "object", "properties": {"offset": {"type": "integer"}}}
            }
        }
    }`)

	pr, err := parser.ParseWithOptions(parser.WithBytes(spec))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*pr),
		WithEnabledFixes(FixTypeRenamedGenericSchema),
		WithGenericNamingConfig(GenericNamingConfig{
			Strategy: GenericNamingOf,
		}),
	)
	require.NoError(t, err)

	doc := result.Document.(*parser.OAS3Document)

	// The generic schema should be renamed
	assert.NotContains(t, doc.Components.Schemas, "Response[[]*common.Pet]",
		"generic schema should be renamed")

	// Package-qualified schemas should be UNCHANGED
	assert.Contains(t, doc.Components.Schemas, "common.Pet",
		"non-generic schema should not be renamed")
	assert.Contains(t, doc.Components.Schemas, "common.Error",
		"non-generic schema should not be renamed")
	assert.Contains(t, doc.Components.Schemas, "common.MetaInfo",
		"non-generic schema should not be renamed")
	assert.Contains(t, doc.Components.Schemas, "common.Pagination",
		"non-generic schema should not be renamed")

	// Critical: Refs to package-qualified schemas should be UNCHANGED
	metaInfo := doc.Components.Schemas["common.MetaInfo"]
	require.NotNil(t, metaInfo)
	paginationRef := metaInfo.Properties["pagination"].Ref

	// THESE ARE THE BUG ASSERTIONS - refs should NOT be corrupted
	assert.NotEqual(t, "#/components/schemas/.common.Pagination", paginationRef,
		"ref should NOT have leading dot")
	assert.NotEqual(t, "#/components/schemas/*common.Pagination", paginationRef,
		"ref should NOT have asterisk prefix")
	assert.NotContains(t, paginationRef, "_0",
		"ref should NOT have _0 suffix mismatch")

	// Correct behavior - ref unchanged
	assert.Equal(t, "#/components/schemas/common.Pagination", paginationRef,
		"ref should be unchanged")

	// Verify the renamed schema has correct refs
	// Find the renamed schema (should be something like ResponseOfCommonPet)
	var renamedSchema *parser.Schema
	var renamedName string
	for name, schema := range doc.Components.Schemas {
		if strings.HasPrefix(name, "Response") && name != "Response[[]*common.Pet]" {
			renamedSchema = schema
			renamedName = name
			break
		}
	}
	require.NotNil(t, renamedSchema, "should find renamed schema")
	t.Logf("Generic schema renamed to: %s", renamedName)

	// Check the data property ref
	if dataItems, ok := renamedSchema.Properties["data"].Items.(*parser.Schema); ok {
		assert.Equal(t, "#/components/schemas/common.Pet", dataItems.Ref,
			"data items ref should point to common.Pet")
	}

	// Check the meta property ref
	assert.Equal(t, "#/components/schemas/common.MetaInfo", renamedSchema.Properties["meta"].Ref,
		"meta ref should point to common.MetaInfo")
}
