// prune_2020_12_test.go tests JSON Schema Draft 2020-12 field handling in schema pruning.
package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
)

// TestCollectSchemaRefs_UnevaluatedProperties tests that refs in unevaluatedProperties are collected
func TestCollectSchemaRefs_UnevaluatedProperties(t *testing.T) {
	t.Run("unevaluatedProperties as schema with ref", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "object",
			UnevaluatedProperties: &parser.Schema{
				Ref: "#/components/schemas/AdditionalData",
			},
		}

		refs := collectSchemaRefs(schema, "#/components/schemas/")
		assert.Contains(t, refs, "AdditionalData")
	})

	t.Run("unevaluatedProperties as bool", func(t *testing.T) {
		schema := &parser.Schema{
			Type:                  "object",
			UnevaluatedProperties: false,
		}

		refs := collectSchemaRefs(schema, "#/components/schemas/")
		assert.Empty(t, refs)
	})

	t.Run("unevaluatedProperties as map with ref", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "object",
			UnevaluatedProperties: map[string]any{
				"$ref": "#/components/schemas/DynamicData",
			},
		}

		refs := collectSchemaRefs(schema, "#/components/schemas/")
		assert.Contains(t, refs, "DynamicData")
	})
}

// TestCollectSchemaRefs_UnevaluatedItems tests that refs in unevaluatedItems are collected
func TestCollectSchemaRefs_UnevaluatedItems(t *testing.T) {
	t.Run("unevaluatedItems as schema with ref", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "array",
			UnevaluatedItems: &parser.Schema{
				Ref: "#/components/schemas/ArrayItem",
			},
		}

		refs := collectSchemaRefs(schema, "#/components/schemas/")
		assert.Contains(t, refs, "ArrayItem")
	})

	t.Run("unevaluatedItems as bool", func(t *testing.T) {
		schema := &parser.Schema{
			Type:             "array",
			UnevaluatedItems: true,
		}

		refs := collectSchemaRefs(schema, "#/components/schemas/")
		assert.Empty(t, refs)
	})

	t.Run("unevaluatedItems as map with ref", func(t *testing.T) {
		schema := &parser.Schema{
			Type: "array",
			UnevaluatedItems: map[string]any{
				"$ref": "#/components/schemas/UnknownItem",
			},
		}

		refs := collectSchemaRefs(schema, "#/components/schemas/")
		assert.Contains(t, refs, "UnknownItem")
	})
}

// TestCollectSchemaRefs_ContentSchema tests that refs in contentSchema are collected
func TestCollectSchemaRefs_ContentSchema(t *testing.T) {
	schema := &parser.Schema{
		Type:             "string",
		ContentEncoding:  "base64",
		ContentMediaType: "application/json",
		ContentSchema: &parser.Schema{
			Ref: "#/components/schemas/EncodedPayload",
		},
	}

	refs := collectSchemaRefs(schema, "#/components/schemas/")
	assert.Contains(t, refs, "EncodedPayload")
}

// TestIsPathItemEmpty_AdditionalOperations tests that paths with custom HTTP methods are not empty
func TestIsPathItemEmpty_AdditionalOperations(t *testing.T) {
	t.Run("path with additionalOperations is not empty", func(t *testing.T) {
		pathItem := &parser.PathItem{
			AdditionalOperations: map[string]*parser.Operation{
				"CUSTOM": {OperationID: "customOperation"},
			},
		}

		// OAS 3.2+ supports additionalOperations
		isEmpty := isPathItemEmpty(pathItem, parser.OASVersion320)
		assert.False(t, isEmpty, "path with additionalOperations should not be empty")
	})

	t.Run("path with additionalOperations ignored in older versions", func(t *testing.T) {
		pathItem := &parser.PathItem{
			AdditionalOperations: map[string]*parser.Operation{
				"CUSTOM": {OperationID: "customOperation"},
			},
		}

		// OAS 3.1 doesn't support additionalOperations
		isEmpty := isPathItemEmpty(pathItem, parser.OASVersion310)
		assert.True(t, isEmpty, "additionalOperations should be ignored in OAS 3.1")
	})

	t.Run("path with query method is not empty in OAS 3.2", func(t *testing.T) {
		pathItem := &parser.PathItem{
			Query: &parser.Operation{OperationID: "queryOperation"},
		}

		isEmpty := isPathItemEmpty(pathItem, parser.OASVersion320)
		assert.False(t, isEmpty, "path with Query should not be empty in OAS 3.2")
	})

	t.Run("empty path is empty", func(t *testing.T) {
		pathItem := &parser.PathItem{}

		isEmpty := isPathItemEmpty(pathItem, parser.OASVersion320)
		assert.True(t, isEmpty, "empty path should be empty")
	})
}

// TestCollectSchemaRefs_PrefixItems tests that refs in prefixItems are collected
func TestCollectSchemaRefs_PrefixItems(t *testing.T) {
	schema := &parser.Schema{
		Type: "array",
		PrefixItems: []*parser.Schema{
			{Ref: "#/components/schemas/FirstItem"},
			{Ref: "#/components/schemas/SecondItem"},
		},
	}

	refs := collectSchemaRefs(schema, "#/components/schemas/")
	assert.Contains(t, refs, "FirstItem")
	assert.Contains(t, refs, "SecondItem")
}

// TestCollectSchemaRefs_Contains tests that refs in contains are collected
func TestCollectSchemaRefs_Contains(t *testing.T) {
	schema := &parser.Schema{
		Type: "array",
		Contains: &parser.Schema{
			Ref: "#/components/schemas/RequiredItem",
		},
	}

	refs := collectSchemaRefs(schema, "#/components/schemas/")
	assert.Contains(t, refs, "RequiredItem")
}

// TestCollectSchemaRefs_PropertyNames tests that refs in propertyNames are collected
func TestCollectSchemaRefs_PropertyNames(t *testing.T) {
	schema := &parser.Schema{
		Type: "object",
		PropertyNames: &parser.Schema{
			Ref: "#/components/schemas/PropertyNameSchema",
		},
	}

	refs := collectSchemaRefs(schema, "#/components/schemas/")
	assert.Contains(t, refs, "PropertyNameSchema")
}

// TestCollectSchemaRefs_DependentSchemas tests that refs in dependentSchemas are collected
func TestCollectSchemaRefs_DependentSchemas(t *testing.T) {
	schema := &parser.Schema{
		Type: "object",
		DependentSchemas: map[string]*parser.Schema{
			"name": {Ref: "#/components/schemas/NameDependency"},
		},
	}

	refs := collectSchemaRefs(schema, "#/components/schemas/")
	assert.Contains(t, refs, "NameDependency")
}

// TestCollectSchemaRefs_ConditionalSchemas tests that refs in if/then/else are collected
func TestCollectSchemaRefs_ConditionalSchemas(t *testing.T) {
	schema := &parser.Schema{
		Type: "object",
		If:   &parser.Schema{Ref: "#/components/schemas/ConditionSchema"},
		Then: &parser.Schema{Ref: "#/components/schemas/ThenSchema"},
		Else: &parser.Schema{Ref: "#/components/schemas/ElseSchema"},
	}

	refs := collectSchemaRefs(schema, "#/components/schemas/")
	assert.Contains(t, refs, "ConditionSchema")
	assert.Contains(t, refs, "ThenSchema")
	assert.Contains(t, refs, "ElseSchema")
}

// TestCollectSchemaRefs_Defs tests that refs in $defs are collected
func TestCollectSchemaRefs_Defs(t *testing.T) {
	schema := &parser.Schema{
		Type: "object",
		Defs: map[string]*parser.Schema{
			"LocalDef": {Ref: "#/components/schemas/ExternalRef"},
		},
	}

	refs := collectSchemaRefs(schema, "#/components/schemas/")
	assert.Contains(t, refs, "ExternalRef")
}
