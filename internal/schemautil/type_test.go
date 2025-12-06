package schemautil

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

func TestGetSchemaTypes(t *testing.T) {
	tests := []struct {
		name     string
		schema   *parser.Schema
		expected []string
	}{
		{
			name:     "nil schema",
			schema:   nil,
			expected: nil,
		},
		{
			name:     "empty type",
			schema:   &parser.Schema{Type: ""},
			expected: nil,
		},
		{
			name:     "string type",
			schema:   &parser.Schema{Type: "string"},
			expected: []string{"string"},
		},
		{
			name:     "integer type",
			schema:   &parser.Schema{Type: "integer"},
			expected: []string{"integer"},
		},
		{
			name:     "array of any (OAS 3.1 style)",
			schema:   &parser.Schema{Type: []any{"string", "null"}},
			expected: []string{"string", "null"},
		},
		{
			name:     "array of strings",
			schema:   &parser.Schema{Type: []string{"string", "null"}},
			expected: []string{"string", "null"},
		},
		{
			name:     "array with non-string values filtered",
			schema:   &parser.Schema{Type: []any{"string", 123, "null"}},
			expected: []string{"string", "null"},
		},
		{
			name:     "unsupported type returns nil",
			schema:   &parser.Schema{Type: 123},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSchemaTypes(tt.schema)
			if len(result) != len(tt.expected) {
				t.Errorf("GetSchemaTypes() = %v, want %v", result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("GetSchemaTypes()[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestGetPrimaryType(t *testing.T) {
	tests := []struct {
		name     string
		schema   *parser.Schema
		expected string
	}{
		{
			name:     "nil schema",
			schema:   nil,
			expected: "",
		},
		{
			name:     "single string type",
			schema:   &parser.Schema{Type: "string"},
			expected: "string",
		},
		{
			name:     "array with null first",
			schema:   &parser.Schema{Type: []any{"null", "string"}},
			expected: "string",
		},
		{
			name:     "array with string first",
			schema:   &parser.Schema{Type: []any{"string", "null"}},
			expected: "string",
		},
		{
			name:     "only null type",
			schema:   &parser.Schema{Type: []any{"null"}},
			expected: "null",
		},
		{
			name:     "multiple non-null types",
			schema:   &parser.Schema{Type: []any{"string", "integer"}},
			expected: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrimaryType(tt.schema)
			if result != tt.expected {
				t.Errorf("GetPrimaryType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsNullable(t *testing.T) {
	tests := []struct {
		name     string
		schema   *parser.Schema
		expected bool
	}{
		{
			name:     "nil schema",
			schema:   nil,
			expected: false,
		},
		{
			name:     "string type not nullable",
			schema:   &parser.Schema{Type: "string"},
			expected: false,
		},
		{
			name:     "array with null",
			schema:   &parser.Schema{Type: []any{"string", "null"}},
			expected: true,
		},
		{
			name:     "array without null",
			schema:   &parser.Schema{Type: []any{"string", "integer"}},
			expected: false,
		},
		{
			name:     "only null",
			schema:   &parser.Schema{Type: "null"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNullable(tt.schema)
			if result != tt.expected {
				t.Errorf("IsNullable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasType(t *testing.T) {
	tests := []struct {
		name       string
		schema     *parser.Schema
		targetType string
		expected   bool
	}{
		{
			name:       "nil schema",
			schema:     nil,
			targetType: "string",
			expected:   false,
		},
		{
			name:       "matching string type",
			schema:     &parser.Schema{Type: "string"},
			targetType: "string",
			expected:   true,
		},
		{
			name:       "non-matching string type",
			schema:     &parser.Schema{Type: "integer"},
			targetType: "string",
			expected:   false,
		},
		{
			name:       "matching in array",
			schema:     &parser.Schema{Type: []any{"string", "null"}},
			targetType: "null",
			expected:   true,
		},
		{
			name:       "not in array",
			schema:     &parser.Schema{Type: []any{"string", "integer"}},
			targetType: "boolean",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasType(tt.schema, tt.targetType)
			if result != tt.expected {
				t.Errorf("HasType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsSingleType(t *testing.T) {
	tests := []struct {
		name     string
		schema   *parser.Schema
		expected bool
	}{
		{
			name:     "nil schema",
			schema:   nil,
			expected: false,
		},
		{
			name:     "single string type",
			schema:   &parser.Schema{Type: "string"},
			expected: true,
		},
		{
			name:     "string with null (nullable)",
			schema:   &parser.Schema{Type: []any{"string", "null"}},
			expected: true,
		},
		{
			name:     "multiple non-null types",
			schema:   &parser.Schema{Type: []any{"string", "integer"}},
			expected: false,
		},
		{
			name:     "only null",
			schema:   &parser.Schema{Type: []any{"null"}},
			expected: false,
		},
		{
			name:     "three types with null",
			schema:   &parser.Schema{Type: []any{"string", "integer", "null"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSingleType(tt.schema)
			if result != tt.expected {
				t.Errorf("IsSingleType() = %v, want %v", result, tt.expected)
			}
		})
	}
}
