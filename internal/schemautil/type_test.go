package schemautil

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, tt.expected, result)
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
			assert.Equal(t, tt.expected, result)
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
			assert.Equal(t, tt.expected, result)
		})
	}
}
