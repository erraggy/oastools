package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixSchemaCSVEnums(t *testing.T) {
	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}

	result := &FixResult{Fixes: make([]Fix, 0)}
	schema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"status": {
				Type: "integer",
				Enum: []any{"1,2,3"},
			},
		},
	}

	f.fixSchemaCSVEnums(schema, "definitions.Pet", result)

	require.Len(t, result.Fixes, 1)
	assert.Equal(t, FixTypeEnumCSVExpanded, result.Fixes[0].Type)
	assert.Equal(t, "definitions.Pet.properties.status", result.Fixes[0].Path)
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, schema.Properties["status"].Enum)
}

func TestFixSchemaCSVEnums_NestedSchemas(t *testing.T) {
	tests := []struct {
		name          string
		schema        *parser.Schema
		expectedFixes int
	}{
		{
			name: "allOf with CSV enum",
			schema: &parser.Schema{
				Type: "object",
				AllOf: []*parser.Schema{
					{
						Type: "integer",
						Enum: []any{"10,20,30"},
					},
				},
				Items: &parser.Schema{
					Type: "number",
					Enum: []any{"1.0,2.0"},
				},
				AdditionalProperties: &parser.Schema{
					Type: "integer",
					Enum: []any{"100,200"},
				},
			},
			expectedFixes: 3,
		},
		{
			name: "anyOf with CSV enum",
			schema: &parser.Schema{
				AnyOf: []*parser.Schema{
					{Type: "integer", Enum: []any{"1,2,3"}},
					{Type: "string"},
				},
			},
			expectedFixes: 1,
		},
		{
			name: "oneOf with CSV enum",
			schema: &parser.Schema{
				OneOf: []*parser.Schema{
					{Type: "string"},
					{Type: "integer", Enum: []any{"10,20,30"}},
				},
			},
			expectedFixes: 1,
		},
		{
			name: "not with CSV enum",
			schema: &parser.Schema{
				Not: &parser.Schema{
					Type: "integer",
					Enum: []any{"0,1"},
				},
			},
			expectedFixes: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New()
			f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}

			result := &FixResult{Fixes: make([]Fix, 0)}
			f.fixSchemaCSVEnums(tt.schema, "schemas.Nested", result)

			require.Len(t, result.Fixes, tt.expectedFixes)
		})
	}

	// Additional check for the allOf test to verify enum values
	t.Run("allOf values verified", func(t *testing.T) {
		f := New()
		f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}

		schema := &parser.Schema{
			Type: "object",
			AllOf: []*parser.Schema{
				{
					Type: "integer",
					Enum: []any{"10,20,30"},
				},
			},
			Items: &parser.Schema{
				Type: "number",
				Enum: []any{"1.0,2.0"},
			},
			AdditionalProperties: &parser.Schema{
				Type: "integer",
				Enum: []any{"100,200"},
			},
		}

		result := &FixResult{Fixes: make([]Fix, 0)}
		f.fixSchemaCSVEnums(schema, "schemas.Nested", result)

		// Check all nested schemas were fixed
		assert.Equal(t, []any{int64(10), int64(20), int64(30)}, schema.AllOf[0].Enum)
		assert.Equal(t, []any{1.0, 2.0}, schema.Items.(*parser.Schema).Enum)
		assert.Equal(t, []any{int64(100), int64(200)}, schema.AdditionalProperties.(*parser.Schema).Enum)
	})
}

func TestFixSchemaCSVEnums_SkippedPartsInDescription(t *testing.T) {
	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}

	result := &FixResult{Fixes: make([]Fix, 0)}
	schema := &parser.Schema{
		Type: "integer",
		Enum: []any{"1,abc,3"},
	}

	f.fixSchemaCSVEnums(schema, "definitions.Status", result)

	require.Len(t, result.Fixes, 1)
	fix := result.Fixes[0]
	assert.Contains(t, fix.Description, "skipped 1 invalid: abc")
	assert.Equal(t, []any{int64(1), int64(3)}, schema.Enum)
}

func TestFixSchemaCSVEnums_EmptyExpansionGuard(t *testing.T) {
	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}

	result := &FixResult{Fixes: make([]Fix, 0)}
	originalEnum := []any{"abc,def"}
	schema := &parser.Schema{
		Type: "integer",
		Enum: originalEnum,
	}

	f.fixSchemaCSVEnums(schema, "definitions.Status", result)

	// No fix should be applied when all parts are invalid
	assert.Len(t, result.Fixes, 0)
	// Schema enum should remain unchanged
	assert.Equal(t, originalEnum, schema.Enum)
}
