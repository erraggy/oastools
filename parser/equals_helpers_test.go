package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqualFloat64Ptr(t *testing.T) {
	tests := []struct {
		name string
		a    *float64
		b    *float64
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b non-nil",
			a:    nil,
			b:    ptr(3.14),
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    ptr(3.14),
			b:    nil,
			want: false,
		},
		{
			name: "both same value",
			a:    ptr(3.14),
			b:    ptr(3.14),
			want: true,
		},
		{
			name: "both different values",
			a:    ptr(3.14),
			b:    ptr(2.71),
			want: false,
		},
		{
			name: "both zero",
			a:    ptr(0.0),
			b:    ptr(0.0),
			want: true,
		},
		{
			name: "negative values equal",
			a:    ptr(-1.5),
			b:    ptr(-1.5),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalFloat64Ptr(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualIntPtr(t *testing.T) {
	tests := []struct {
		name string
		a    *int
		b    *int
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b non-nil",
			a:    nil,
			b:    intPtr(42),
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    intPtr(42),
			b:    nil,
			want: false,
		},
		{
			name: "both same value",
			a:    intPtr(42),
			b:    intPtr(42),
			want: true,
		},
		{
			name: "both different values",
			a:    intPtr(42),
			b:    intPtr(100),
			want: false,
		},
		{
			name: "both zero",
			a:    intPtr(0),
			b:    intPtr(0),
			want: true,
		},
		{
			name: "negative values equal",
			a:    intPtr(-5),
			b:    intPtr(-5),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalIntPtr(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualBoolPtr(t *testing.T) {
	tests := []struct {
		name string
		a    *bool
		b    *bool
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b non-nil true",
			a:    nil,
			b:    boolPtr(true),
			want: false,
		},
		{
			name: "a nil, b non-nil false",
			a:    nil,
			b:    boolPtr(false),
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    boolPtr(true),
			b:    nil,
			want: false,
		},
		{
			name: "both true",
			a:    boolPtr(true),
			b:    boolPtr(true),
			want: true,
		},
		{
			name: "both false",
			a:    boolPtr(false),
			b:    boolPtr(false),
			want: true,
		},
		{
			name: "true vs false",
			a:    boolPtr(true),
			b:    boolPtr(false),
			want: false,
		},
		{
			name: "false vs true",
			a:    boolPtr(false),
			b:    boolPtr(true),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalBoolPtr(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualStringSlice(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "both empty",
			a:    []string{},
			b:    []string{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    []string{},
			want: true,
		},
		{
			name: "empty vs nil",
			a:    []string{},
			b:    nil,
			want: true,
		},
		{
			name: "same elements same order",
			a:    []string{"a", "b", "c"},
			b:    []string{"a", "b", "c"},
			want: true,
		},
		{
			name: "same elements different order",
			a:    []string{"a", "b", "c"},
			b:    []string{"c", "b", "a"},
			want: false,
		},
		{
			name: "different lengths - a longer",
			a:    []string{"a", "b", "c"},
			b:    []string{"a", "b"},
			want: false,
		},
		{
			name: "different lengths - b longer",
			a:    []string{"a", "b"},
			b:    []string{"a", "b", "c"},
			want: false,
		},
		{
			name: "different elements",
			a:    []string{"a", "b"},
			b:    []string{"x", "y"},
			want: false,
		},
		{
			name: "single element equal",
			a:    []string{"single"},
			b:    []string{"single"},
			want: true,
		},
		{
			name: "single element different",
			a:    []string{"single"},
			b:    []string{"different"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalStringSlice(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualAnySlice(t *testing.T) {
	tests := []struct {
		name string
		a    []any
		b    []any
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "both empty",
			a:    []any{},
			b:    []any{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    []any{},
			want: true,
		},
		{
			name: "empty vs nil",
			a:    []any{},
			b:    nil,
			want: true,
		},
		{
			name: "same primitive elements",
			a:    []any{"a", 1, true, 3.14},
			b:    []any{"a", 1, true, 3.14},
			want: true,
		},
		{
			name: "same nested maps",
			a:    []any{map[string]any{"key": "value"}},
			b:    []any{map[string]any{"key": "value"}},
			want: true,
		},
		{
			name: "same nested slices",
			a:    []any{[]any{1, 2, 3}},
			b:    []any{[]any{1, 2, 3}},
			want: true,
		},
		{
			name: "different elements",
			a:    []any{"a", "b"},
			b:    []any{"x", "y"},
			want: false,
		},
		{
			name: "different lengths",
			a:    []any{1, 2, 3},
			b:    []any{1, 2},
			want: false,
		},
		{
			name: "different nested maps",
			a:    []any{map[string]any{"key": "value1"}},
			b:    []any{map[string]any{"key": "value2"}},
			want: false,
		},
		{
			name: "mixed types same values",
			a:    []any{"string", 42, true, nil},
			b:    []any{"string", 42, true, nil},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalAnySlice(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualMapStringAny(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]any
		b    map[string]any
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "both empty",
			a:    map[string]any{},
			b:    map[string]any{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    map[string]any{},
			want: true,
		},
		{
			name: "empty vs nil",
			a:    map[string]any{},
			b:    nil,
			want: true,
		},
		{
			name: "same keys and values",
			a:    map[string]any{"key1": "value1", "key2": 42},
			b:    map[string]any{"key1": "value1", "key2": 42},
			want: true,
		},
		{
			name: "different keys",
			a:    map[string]any{"key1": "value1"},
			b:    map[string]any{"key2": "value1"},
			want: false,
		},
		{
			name: "same keys different values",
			a:    map[string]any{"key": "value1"},
			b:    map[string]any{"key": "value2"},
			want: false,
		},
		{
			name: "nested maps equal",
			a:    map[string]any{"nested": map[string]any{"inner": "value"}},
			b:    map[string]any{"nested": map[string]any{"inner": "value"}},
			want: true,
		},
		{
			name: "nested maps different",
			a:    map[string]any{"nested": map[string]any{"inner": "value1"}},
			b:    map[string]any{"nested": map[string]any{"inner": "value2"}},
			want: false,
		},
		{
			name: "a has extra key",
			a:    map[string]any{"key1": "value1", "key2": "value2"},
			b:    map[string]any{"key1": "value1"},
			want: false,
		},
		{
			name: "b has extra key",
			a:    map[string]any{"key1": "value1"},
			b:    map[string]any{"key1": "value1", "key2": "value2"},
			want: false,
		},
		{
			name: "deeply nested structures equal",
			a: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": "deep value",
					},
				},
			},
			b: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": "deep value",
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalMapStringAny(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualMapStringBool(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]bool
		b    map[string]bool
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "both empty",
			a:    map[string]bool{},
			b:    map[string]bool{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    map[string]bool{},
			want: true,
		},
		{
			name: "same entries",
			a:    map[string]bool{"enabled": true, "disabled": false},
			b:    map[string]bool{"enabled": true, "disabled": false},
			want: true,
		},
		{
			name: "different values",
			a:    map[string]bool{"enabled": true},
			b:    map[string]bool{"enabled": false},
			want: false,
		},
		{
			name: "different keys",
			a:    map[string]bool{"key1": true},
			b:    map[string]bool{"key2": true},
			want: false,
		},
		{
			name: "a has extra key",
			a:    map[string]bool{"key1": true, "key2": false},
			b:    map[string]bool{"key1": true},
			want: false,
		},
		{
			name: "b has extra key",
			a:    map[string]bool{"key1": true},
			b:    map[string]bool{"key1": true, "key2": false},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalMapStringBool(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualMapStringStringSlice(t *testing.T) {
	tests := []struct {
		name string
		a    map[string][]string
		b    map[string][]string
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "both empty",
			a:    map[string][]string{},
			b:    map[string][]string{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    map[string][]string{},
			want: true,
		},
		{
			name: "same entries",
			a:    map[string][]string{"key1": {"a", "b"}, "key2": {"c", "d"}},
			b:    map[string][]string{"key1": {"a", "b"}, "key2": {"c", "d"}},
			want: true,
		},
		{
			name: "different slice values",
			a:    map[string][]string{"key1": {"a", "b"}},
			b:    map[string][]string{"key1": {"a", "c"}},
			want: false,
		},
		{
			name: "different slice order",
			a:    map[string][]string{"key1": {"a", "b"}},
			b:    map[string][]string{"key1": {"b", "a"}},
			want: false,
		},
		{
			name: "different keys",
			a:    map[string][]string{"key1": {"a", "b"}},
			b:    map[string][]string{"key2": {"a", "b"}},
			want: false,
		},
		{
			name: "a has extra key",
			a:    map[string][]string{"key1": {"a"}, "key2": {"b"}},
			b:    map[string][]string{"key1": {"a"}},
			want: false,
		},
		{
			name: "empty slice vs nil slice in map",
			a:    map[string][]string{"key1": {}},
			b:    map[string][]string{"key1": nil},
			want: true,
		},
		{
			name: "different slice lengths",
			a:    map[string][]string{"key1": {"a", "b", "c"}},
			b:    map[string][]string{"key1": {"a", "b"}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalMapStringStringSlice(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualSchemaType(t *testing.T) {
	tests := []struct {
		name string
		a    any
		b    any
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b non-nil string",
			a:    nil,
			b:    "object",
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    "object",
			b:    nil,
			want: false,
		},
		{
			name: "both same string",
			a:    "object",
			b:    "object",
			want: true,
		},
		{
			name: "different strings",
			a:    "object",
			b:    "string",
			want: false,
		},
		{
			name: "both same []string",
			a:    []string{"string", "null"},
			b:    []string{"string", "null"},
			want: true,
		},
		{
			name: "different []string",
			a:    []string{"string", "null"},
			b:    []string{"integer", "null"},
			want: false,
		},
		{
			name: "[]string different order",
			a:    []string{"string", "null"},
			b:    []string{"null", "string"},
			want: false,
		},
		{
			name: "string vs []string with same content - type mismatch",
			a:    "string",
			b:    []string{"string"},
			want: false,
		},
		{
			name: "[]string vs string - type mismatch",
			a:    []string{"string"},
			b:    "string",
			want: false,
		},
		{
			name: "both same []any",
			a:    []any{"string", "null"},
			b:    []any{"string", "null"},
			want: true,
		},
		{
			name: "different []any",
			a:    []any{"string", "null"},
			b:    []any{"integer", "null"},
			want: false,
		},
		{
			name: "[]string vs []any - type mismatch",
			a:    []string{"string", "null"},
			b:    []any{"string", "null"},
			want: false,
		},
		{
			name: "empty string vs nil",
			a:    "",
			b:    nil,
			want: false,
		},
		{
			name: "nil vs empty string",
			a:    nil,
			b:    "",
			want: false,
		},
		{
			name: "both empty string",
			a:    "",
			b:    "",
			want: true,
		},
		// reflect.DeepEqual fallback tests for unknown types
		{
			name: "unknown type - same struct values via reflect.DeepEqual",
			a:    struct{ X int }{X: 42},
			b:    struct{ X int }{X: 42},
			want: true,
		},
		{
			name: "unknown type - different struct values via reflect.DeepEqual",
			a:    struct{ X int }{X: 42},
			b:    struct{ X int }{X: 100},
			want: false,
		},
		{
			name: "unknown type - different struct types via reflect.DeepEqual",
			a:    struct{ X int }{X: 42},
			b:    struct{ Y int }{Y: 42},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalSchemaType(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualSchemaOrBool(t *testing.T) {
	// Create test schemas
	schema1 := &Schema{Type: "string"}
	schema2 := &Schema{Type: "string"}
	schema3 := &Schema{Type: "integer"}

	tests := []struct {
		name string
		a    any
		b    any
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b bool true",
			a:    nil,
			b:    true,
			want: false,
		},
		{
			name: "a bool, b nil",
			a:    true,
			b:    nil,
			want: false,
		},
		{
			name: "both bool true",
			a:    true,
			b:    true,
			want: true,
		},
		{
			name: "both bool false",
			a:    false,
			b:    false,
			want: true,
		},
		{
			name: "bool true vs false",
			a:    true,
			b:    false,
			want: false,
		},
		{
			name: "both same *Schema",
			a:    schema1,
			b:    schema2,
			want: true,
		},
		{
			name: "different *Schema",
			a:    schema1,
			b:    schema3,
			want: false,
		},
		{
			name: "bool vs *Schema - type mismatch",
			a:    true,
			b:    schema1,
			want: false,
		},
		{
			name: "*Schema vs bool - type mismatch",
			a:    schema1,
			b:    true,
			want: false,
		},
		{
			name: "a nil, b *Schema",
			a:    nil,
			b:    schema1,
			want: false,
		},
		{
			name: "*Schema vs nil",
			a:    schema1,
			b:    nil,
			want: false,
		},
		{
			name: "both nil *Schema typed",
			a:    (*Schema)(nil),
			b:    (*Schema)(nil),
			want: true,
		},
		{
			name: "nil *Schema vs non-nil *Schema",
			a:    (*Schema)(nil),
			b:    schema1,
			want: false,
		},
		// reflect.DeepEqual fallback tests for unknown types
		{
			name: "unknown type - same struct values via reflect.DeepEqual",
			a:    struct{ X int }{X: 42},
			b:    struct{ X int }{X: 42},
			want: true,
		},
		{
			name: "unknown type - different struct values via reflect.DeepEqual",
			a:    struct{ X int }{X: 42},
			b:    struct{ X int }{X: 100},
			want: false,
		},
		{
			name: "unknown type - different struct types via reflect.DeepEqual",
			a:    struct{ X int }{X: 42},
			b:    struct{ Y int }{Y: 42},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalSchemaOrBool(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualBoolOrNumber(t *testing.T) {
	tests := []struct {
		name string
		a    any
		b    any
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b bool",
			a:    nil,
			b:    true,
			want: false,
		},
		{
			name: "a bool, b nil",
			a:    true,
			b:    nil,
			want: false,
		},
		{
			name: "both bool true",
			a:    true,
			b:    true,
			want: true,
		},
		{
			name: "both bool false",
			a:    false,
			b:    false,
			want: true,
		},
		{
			name: "bool true vs false",
			a:    true,
			b:    false,
			want: false,
		},
		{
			name: "both same float64",
			a:    float64(3.14),
			b:    float64(3.14),
			want: true,
		},
		{
			name: "different float64",
			a:    float64(3.14),
			b:    float64(2.71),
			want: false,
		},
		{
			name: "both same int",
			a:    int(42),
			b:    int(42),
			want: true,
		},
		{
			name: "different int",
			a:    int(42),
			b:    int(100),
			want: false,
		},
		{
			name: "bool vs float64 - type mismatch",
			a:    true,
			b:    float64(1.0),
			want: false,
		},
		{
			name: "float64 vs bool - type mismatch",
			a:    float64(1.0),
			b:    true,
			want: false,
		},
		{
			name: "int vs float64 - type mismatch",
			a:    int(42),
			b:    float64(42.0),
			want: false,
		},
		{
			name: "float64 vs int - type mismatch",
			a:    float64(42.0),
			b:    int(42),
			want: false,
		},
		{
			name: "both same int64",
			a:    int64(42),
			b:    int64(42),
			want: true,
		},
		{
			name: "different int64",
			a:    int64(42),
			b:    int64(100),
			want: false,
		},
		{
			name: "int vs int64 - type mismatch",
			a:    int(42),
			b:    int64(42),
			want: false,
		},
		{
			name: "zero values - float64",
			a:    float64(0),
			b:    float64(0),
			want: true,
		},
		{
			name: "zero values - int",
			a:    int(0),
			b:    int(0),
			want: true,
		},
		{
			name: "negative float64 equal",
			a:    float64(-5.5),
			b:    float64(-5.5),
			want: true,
		},
		// reflect.DeepEqual fallback tests for unknown types
		{
			name: "unknown type - same struct values via reflect.DeepEqual",
			a:    struct{ X int }{X: 42},
			b:    struct{ X int }{X: 42},
			want: true,
		},
		{
			name: "unknown type - different struct values via reflect.DeepEqual",
			a:    struct{ X int }{X: 42},
			b:    struct{ X int }{X: 100},
			want: false,
		},
		{
			name: "unknown type - different struct types via reflect.DeepEqual",
			a:    struct{ X int }{X: 42},
			b:    struct{ Y int }{Y: 42},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalBoolOrNumber(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualJSONValue(t *testing.T) {
	tests := []struct {
		name string
		a    any
		b    any
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b non-nil",
			a:    nil,
			b:    "value",
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    "value",
			b:    nil,
			want: false,
		},
		// String tests
		{
			name: "same strings",
			a:    "hello",
			b:    "hello",
			want: true,
		},
		{
			name: "different strings",
			a:    "hello",
			b:    "world",
			want: false,
		},
		{
			name: "empty strings",
			a:    "",
			b:    "",
			want: true,
		},
		// Number tests
		{
			name: "same float64",
			a:    float64(3.14),
			b:    float64(3.14),
			want: true,
		},
		{
			name: "different float64",
			a:    float64(3.14),
			b:    float64(2.71),
			want: false,
		},
		{
			name: "same int",
			a:    int(42),
			b:    int(42),
			want: true,
		},
		{
			name: "different int",
			a:    int(42),
			b:    int(100),
			want: false,
		},
		{
			name: "same int64",
			a:    int64(42),
			b:    int64(42),
			want: true,
		},
		{
			name: "same float32",
			a:    float32(3.14),
			b:    float32(3.14),
			want: true,
		},
		{
			name: "same int32",
			a:    int32(42),
			b:    int32(42),
			want: true,
		},
		{
			name: "same int16",
			a:    int16(42),
			b:    int16(42),
			want: true,
		},
		{
			name: "same int8",
			a:    int8(42),
			b:    int8(42),
			want: true,
		},
		{
			name: "same uint",
			a:    uint(42),
			b:    uint(42),
			want: true,
		},
		{
			name: "same uint64",
			a:    uint64(42),
			b:    uint64(42),
			want: true,
		},
		{
			name: "same uint32",
			a:    uint32(42),
			b:    uint32(42),
			want: true,
		},
		{
			name: "same uint16",
			a:    uint16(42),
			b:    uint16(42),
			want: true,
		},
		{
			name: "same uint8",
			a:    uint8(42),
			b:    uint8(42),
			want: true,
		},
		// Bool tests
		{
			name: "same bools true",
			a:    true,
			b:    true,
			want: true,
		},
		{
			name: "same bools false",
			a:    false,
			b:    false,
			want: true,
		},
		{
			name: "different bools",
			a:    true,
			b:    false,
			want: false,
		},
		// Array tests
		{
			name: "same arrays",
			a:    []any{1, 2, 3},
			b:    []any{1, 2, 3},
			want: true,
		},
		{
			name: "different arrays",
			a:    []any{1, 2, 3},
			b:    []any{1, 2, 4},
			want: false,
		},
		{
			name: "different array lengths",
			a:    []any{1, 2, 3},
			b:    []any{1, 2},
			want: false,
		},
		{
			name: "empty arrays",
			a:    []any{},
			b:    []any{},
			want: true,
		},
		{
			name: "nested arrays equal",
			a:    []any{[]any{1, 2}, []any{3, 4}},
			b:    []any{[]any{1, 2}, []any{3, 4}},
			want: true,
		},
		{
			name: "nested arrays different",
			a:    []any{[]any{1, 2}, []any{3, 4}},
			b:    []any{[]any{1, 2}, []any{3, 5}},
			want: false,
		},
		// Object tests
		{
			name: "same objects",
			a:    map[string]any{"key": "value"},
			b:    map[string]any{"key": "value"},
			want: true,
		},
		{
			name: "different object values",
			a:    map[string]any{"key": "value1"},
			b:    map[string]any{"key": "value2"},
			want: false,
		},
		{
			name: "different object keys",
			a:    map[string]any{"key1": "value"},
			b:    map[string]any{"key2": "value"},
			want: false,
		},
		{
			name: "empty objects",
			a:    map[string]any{},
			b:    map[string]any{},
			want: true,
		},
		{
			name: "nested objects equal",
			a:    map[string]any{"outer": map[string]any{"inner": "value"}},
			b:    map[string]any{"outer": map[string]any{"inner": "value"}},
			want: true,
		},
		{
			name: "nested objects different",
			a:    map[string]any{"outer": map[string]any{"inner": "value1"}},
			b:    map[string]any{"outer": map[string]any{"inner": "value2"}},
			want: false,
		},
		// Mixed nested structures
		{
			name: "complex nested structure equal",
			a: map[string]any{
				"string": "value",
				"number": float64(42),
				"bool":   true,
				"array":  []any{1, 2, 3},
				"object": map[string]any{"nested": "value"},
			},
			b: map[string]any{
				"string": "value",
				"number": float64(42),
				"bool":   true,
				"array":  []any{1, 2, 3},
				"object": map[string]any{"nested": "value"},
			},
			want: true,
		},
		{
			name: "complex nested structure different",
			a: map[string]any{
				"string": "value",
				"number": float64(42),
			},
			b: map[string]any{
				"string": "value",
				"number": float64(43),
			},
			want: false,
		},
		// Type mismatches
		{
			name: "string vs int - type mismatch",
			a:    "42",
			b:    42,
			want: false,
		},
		{
			name: "int vs float64 - type mismatch",
			a:    int(42),
			b:    float64(42),
			want: false,
		},
		{
			name: "bool vs string - type mismatch",
			a:    true,
			b:    "true",
			want: false,
		},
		{
			name: "array vs object - type mismatch",
			a:    []any{1, 2, 3},
			b:    map[string]any{"0": 1, "1": 2, "2": 3},
			want: false,
		},
		// Different numeric types
		{
			name: "different float32",
			a:    float32(3.14),
			b:    float32(2.71),
			want: false,
		},
		{
			name: "different int64",
			a:    int64(42),
			b:    int64(100),
			want: false,
		},
		{
			name: "different uint",
			a:    uint(42),
			b:    uint(100),
			want: false,
		},
		// reflect.DeepEqual fallback tests for unknown types (custom structs in extensions)
		{
			name: "unknown type - same struct values via reflect.DeepEqual",
			a:    struct{ X int }{X: 42},
			b:    struct{ X int }{X: 42},
			want: true,
		},
		{
			name: "unknown type - different struct values via reflect.DeepEqual",
			a:    struct{ X int }{X: 42},
			b:    struct{ X int }{X: 100},
			want: false,
		},
		{
			name: "unknown type - different struct types via reflect.DeepEqual",
			a:    struct{ X int }{X: 42},
			b:    struct{ Y int }{Y: 42},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalJSONValue(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalMapStringString tests
// =============================================================================

func TestEqualMapStringString(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]string
		b    map[string]string
		want bool
	}{
		// Nil and empty handling
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "both empty",
			a:    map[string]string{},
			b:    map[string]string{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    map[string]string{},
			want: true,
		},
		{
			name: "empty vs nil",
			a:    map[string]string{},
			b:    nil,
			want: true,
		},
		// Same entries
		{
			name: "same single entry",
			a:    map[string]string{"key": "value"},
			b:    map[string]string{"key": "value"},
			want: true,
		},
		{
			name: "same multiple entries",
			a:    map[string]string{"key1": "value1", "key2": "value2"},
			b:    map[string]string{"key1": "value1", "key2": "value2"},
			want: true,
		},
		{
			name: "same entries - OAuth scopes",
			a: map[string]string{
				"read:users":  "Read user data",
				"write:users": "Modify user data",
				"admin":       "Full administrative access",
			},
			b: map[string]string{
				"read:users":  "Read user data",
				"write:users": "Modify user data",
				"admin":       "Full administrative access",
			},
			want: true,
		},
		// Different entries
		{
			name: "different values same key",
			a:    map[string]string{"key": "value1"},
			b:    map[string]string{"key": "value2"},
			want: false,
		},
		{
			name: "different keys same value",
			a:    map[string]string{"key1": "value"},
			b:    map[string]string{"key2": "value"},
			want: false,
		},
		{
			name: "a has extra key",
			a:    map[string]string{"key1": "value1", "key2": "value2"},
			b:    map[string]string{"key1": "value1"},
			want: false,
		},
		{
			name: "b has extra key",
			a:    map[string]string{"key1": "value1"},
			b:    map[string]string{"key1": "value1", "key2": "value2"},
			want: false,
		},
		// Edge cases
		{
			name: "empty string key",
			a:    map[string]string{"": "value"},
			b:    map[string]string{"": "value"},
			want: true,
		},
		{
			name: "empty string value",
			a:    map[string]string{"key": ""},
			b:    map[string]string{"key": ""},
			want: true,
		},
		{
			name: "empty string value vs non-empty",
			a:    map[string]string{"key": ""},
			b:    map[string]string{"key": "value"},
			want: false,
		},
		// Discriminator mapping use case
		{
			name: "discriminator mapping - same",
			a: map[string]string{
				"dog":  "#/components/schemas/Dog",
				"cat":  "#/components/schemas/Cat",
				"bird": "#/components/schemas/Bird",
			},
			b: map[string]string{
				"dog":  "#/components/schemas/Dog",
				"cat":  "#/components/schemas/Cat",
				"bird": "#/components/schemas/Bird",
			},
			want: true,
		},
		{
			name: "discriminator mapping - different ref",
			a: map[string]string{
				"dog": "#/components/schemas/Dog",
			},
			b: map[string]string{
				"dog": "#/components/schemas/Canine",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalMapStringString(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Note: ptr, intPtr, and boolPtr helper functions are defined in schema_test_helpers.go
