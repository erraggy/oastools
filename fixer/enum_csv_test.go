package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsCSVEnumCandidate(t *testing.T) {
	tests := []struct {
		name     string
		schema   *parser.Schema
		expected bool
	}{
		{
			name:     "integer type with CSV string",
			schema:   &parser.Schema{Type: "integer", Enum: []any{"1,2,3"}},
			expected: true,
		},
		{
			name:     "number type with CSV string",
			schema:   &parser.Schema{Type: "number", Enum: []any{"1.5,2.5"}},
			expected: true,
		},
		{
			name:     "string type - not candidate",
			schema:   &parser.Schema{Type: "string", Enum: []any{"a,b,c"}},
			expected: false,
		},
		{
			name:     "integer with proper array - not candidate",
			schema:   &parser.Schema{Type: "integer", Enum: []any{int64(1), int64(2), int64(3)}},
			expected: false,
		},
		{
			name:     "nil schema",
			schema:   nil,
			expected: false,
		},
		{
			name:     "empty enum",
			schema:   &parser.Schema{Type: "integer", Enum: []any{}},
			expected: false,
		},
		{
			name:     "single string without comma",
			schema:   &parser.Schema{Type: "integer", Enum: []any{"42"}},
			expected: false,
		},
		{
			name:     "OAS 3.1 type array with integer",
			schema:   &parser.Schema{Type: []any{"integer", "null"}, Enum: []any{"1,2,3"}},
			expected: true,
		},
		{
			name:     "OAS 3.1 type array with number",
			schema:   &parser.Schema{Type: []any{"number"}, Enum: []any{"1.5,2.5"}},
			expected: true,
		},
		{
			name:     "OAS 3.1 type array with string - not candidate",
			schema:   &parser.Schema{Type: []any{"string", "null"}, Enum: []any{"a,b,c"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCSVEnumCandidate(tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpandCSVEnumValues(t *testing.T) {
	tests := []struct {
		name         string
		schema       *parser.Schema
		expectedEnum []any
		hadExpansion bool
	}{
		{
			name:         "basic CSV expansion - integer",
			schema:       &parser.Schema{Type: "integer", Enum: []any{"1,2,3"}},
			expectedEnum: []any{int64(1), int64(2), int64(3)},
			hadExpansion: true,
		},
		{
			name:         "whitespace handling",
			schema:       &parser.Schema{Type: "integer", Enum: []any{"1, 2, 3"}},
			expectedEnum: []any{int64(1), int64(2), int64(3)},
			hadExpansion: true,
		},
		{
			name:         "empty parts skipped",
			schema:       &parser.Schema{Type: "integer", Enum: []any{"1,,3"}},
			expectedEnum: []any{int64(1), int64(3)},
			hadExpansion: true,
		},
		{
			name:         "invalid parts skipped",
			schema:       &parser.Schema{Type: "integer", Enum: []any{"1,abc,3"}},
			expectedEnum: []any{int64(1), int64(3)},
			hadExpansion: true,
		},
		{
			name:         "number type float values",
			schema:       &parser.Schema{Type: "number", Enum: []any{"1.5,2.5,3.5"}},
			expectedEnum: []any{1.5, 2.5, 3.5},
			hadExpansion: true,
		},
		{
			name:         "mixed array - numeric kept",
			schema:       &parser.Schema{Type: "integer", Enum: []any{"1,2", int64(3)}},
			expectedEnum: []any{int64(1), int64(2), int64(3)},
			hadExpansion: true,
		},
		{
			name:         "single value no comma - kept as string",
			schema:       &parser.Schema{Type: "integer", Enum: []any{"42"}},
			expectedEnum: []any{"42"},
			hadExpansion: false,
		},
		{
			name:         "nil schema",
			schema:       nil,
			expectedEnum: nil,
			hadExpansion: false,
		},
		{
			name:         "empty enum",
			schema:       &parser.Schema{Type: "integer", Enum: []any{}},
			expectedEnum: []any{},
			hadExpansion: false,
		},
		{
			name:         "string type - no expansion",
			schema:       &parser.Schema{Type: "string", Enum: []any{"a,b,c"}},
			expectedEnum: []any{"a,b,c"},
			hadExpansion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expanded, hadExpansion := expandCSVEnumValues(tt.schema)
			assert.Equal(t, tt.hadExpansion, hadExpansion)
			assert.Equal(t, tt.expectedEnum, expanded)
		})
	}
}

func TestParseNumericValue(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		schemaType string
		expected   any
		wantErr    bool
	}{
		{"integer valid", "42", "integer", int64(42), false},
		{"integer negative", "-10", "integer", int64(-10), false},
		{"integer invalid", "abc", "integer", nil, true},
		{"number valid", "3.14", "number", 3.14, false},
		{"number integer input", "42", "number", 42.0, false},
		{"number invalid", "xyz", "number", nil, true},
		{"unsupported type", "42", "string", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseNumericValue(tt.input, tt.schemaType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetSchemaType(t *testing.T) {
	tests := []struct {
		name     string
		schema   *parser.Schema
		expected string
	}{
		{
			name:     "string type",
			schema:   &parser.Schema{Type: "string"},
			expected: "string",
		},
		{
			name:     "integer type",
			schema:   &parser.Schema{Type: "integer"},
			expected: "integer",
		},
		{
			name:     "number type",
			schema:   &parser.Schema{Type: "number"},
			expected: "number",
		},
		{
			name:     "nil type",
			schema:   &parser.Schema{Type: nil},
			expected: "",
		},
		{
			name:     "type array with integer",
			schema:   &parser.Schema{Type: []any{"integer", "null"}},
			expected: "integer",
		},
		{
			name:     "type array with number",
			schema:   &parser.Schema{Type: []any{"null", "number"}},
			expected: "number",
		},
		{
			name:     "type array with only null",
			schema:   &parser.Schema{Type: []any{"null"}},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSchemaType(tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

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
	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}

	result := &FixResult{Fixes: make([]Fix, 0)}
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

	f.fixSchemaCSVEnums(schema, "schemas.Nested", result)

	require.Len(t, result.Fixes, 3)

	// Check all nested schemas were fixed
	assert.Equal(t, []any{int64(10), int64(20), int64(30)}, schema.AllOf[0].Enum)
	assert.Equal(t, []any{1.0, 2.0}, schema.Items.(*parser.Schema).Enum)
	assert.Equal(t, []any{int64(100), int64(200)}, schema.AdditionalProperties.(*parser.Schema).Enum)
}
